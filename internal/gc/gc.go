package gc

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/chandrashekhartata/acgc/internal/domain"
	"github.com/chandrashekhartata/acgc/internal/facts"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/statetree"
	"github.com/chandrashekhartata/acgc/internal/store"
	"github.com/chandrashekhartata/acgc/internal/tokenizer"
)

type Policy struct {
	MaxPromptTokens       int
	MaxTreeDepth          int
	MaxChildrenPerNode    int
	LowRelevanceThreshold float64
	StaleAfterTurns       int
	// DecisionSweepFloor: when non-zero, sweep score for NodeDecision nodes (with
	// empty Facts/Decisions) is considered max(retention_score, DecisionSweepFloor)
	// for the low-relevance threshold check only — reduces accidental sweep of confirmations.
	//
	// IMPORTANT: keep DecisionSweepFloor STRICTLY BELOW LowRelevanceThreshold,
	// otherwise the floor turns into an absolute "never sweep" rule for bare
	// decisions and the GC effectively never reclaims filler assistant turns.
	DecisionSweepFloor float64
	// MaxActiveNodes: when > 0, trigger GC once the active set exceeds this
	// count. Catches dense sessions with many small turns that never approach
	// MaxPromptTokens but still bloat the compiled prompt.
	MaxActiveNodes int
	// SweepHeadroomRatio: when > 0, trigger GC once estimated active tokens
	// exceed MaxPromptTokens * SweepHeadroomRatio. A value of 1.0 disables the
	// soft trigger (only the hard MaxPromptTokens ceiling fires). Typical: 0.6.
	SweepHeadroomRatio float64
}

type TriggerReason string

const (
	ReasonTokenPressure  TriggerReason = "token_pressure"
	ReasonSoftHeadroom   TriggerReason = "soft_headroom"
	ReasonActiveCount    TriggerReason = "active_count"
	ReasonTreeDepth      TriggerReason = "tree_depth"
	ReasonTreeWidth      TriggerReason = "tree_width"
	ReasonLowRelevance   TriggerReason = "low_relevance"
	ReasonResolvedBranch TriggerReason = "resolved_branch"
	ReasonManual         TriggerReason = "manual"
)

type GCResult struct {
	Triggered               bool
	Reason                  TriggerReason
	NodesSwept              int
	BranchesCompressed      int
	TokensFreed             int
	CompressedBranchRecords []*store.CompressedBranch
}

type Compressor interface {
	Compress(ctx context.Context, nodes []*domain.StateNode) (*domain.StateNode, error)
}

type GarbageCollector struct {
	policy     Policy
	scorer     *scorer.Scorer
	compressor Compressor
}

func NewGarbageCollector(policy Policy, sc *scorer.Scorer, comp Compressor) *GarbageCollector {
	return &GarbageCollector{
		policy:     policy,
		scorer:     sc,
		compressor: comp,
	}
}

func (gc *GarbageCollector) shouldDeferSweepFacts(n *domain.StateNode) bool {
	return len(n.Facts) > 0 || len(n.Decisions) > 0
}

// ShouldRun checks all trigger conditions and returns the reason if GC should run.
//
// Trigger ordering (most specific first):
//  1. Hard token ceiling — prompt won't fit at all
//  2. Soft headroom — active tokens crossed SweepHeadroomRatio × budget
//  3. Active node count — dense session with many small turns
//  4. Tree depth / width — structural overflow
//  5. Low average relevance — session content has gone stale
//  6. Has resolved nodes — opportunistic cleanup
func (gc *GarbageCollector) ShouldRun(tree *statetree.Tree, estimatedTokens int) (bool, TriggerReason) {
	_, _, _, _, maxDepth, maxWidth := tree.Stats()

	if estimatedTokens > gc.policy.MaxPromptTokens {
		return true, ReasonTokenPressure
	}
	if gc.policy.SweepHeadroomRatio > 0 && gc.policy.SweepHeadroomRatio < 1.0 && gc.policy.MaxPromptTokens > 0 {
		softLimit := int(float64(gc.policy.MaxPromptTokens) * gc.policy.SweepHeadroomRatio)
		if estimatedTokens > softLimit {
			return true, ReasonSoftHeadroom
		}
	}
	activeNodes := tree.GetActiveNodes()
	if gc.policy.MaxActiveNodes > 0 && len(activeNodes) > gc.policy.MaxActiveNodes {
		return true, ReasonActiveCount
	}
	if maxDepth > gc.policy.MaxTreeDepth {
		return true, ReasonTreeDepth
	}
	if maxWidth > gc.policy.MaxChildrenPerNode {
		return true, ReasonTreeWidth
	}

	if len(activeNodes) > 0 {
		avgRelevance := 0.0
		hasResolved := false
		for _, n := range activeNodes {
			avgRelevance += n.Scores.RetentionScore
			if n.Status == domain.StatusResolved {
				hasResolved = true
			}
		}
		avgRelevance /= float64(len(activeNodes))

		if avgRelevance < gc.policy.LowRelevanceThreshold {
			return true, ReasonLowRelevance
		}
		if hasResolved {
			return true, ReasonResolvedBranch
		}
	}

	return false, ""
}

// Run executes the mark-sweep-compact cycle on the state tree.
// It returns the result including any compressed branch records that should be
// persisted durably by the caller (session manager).
//
// queryVec is the semantic anchor for re-scoring inside GC. The session
// worker should pass its cached LastUserEmbedding here. Nil is acceptable —
// the scorer will fall back to heuristic-only.
func (gc *GarbageCollector) Run(ctx context.Context, tree *statetree.Tree, reason TriggerReason, queryVec []float32) GCResult {
	result := GCResult{Triggered: true, Reason: reason}

	activeNodes := tree.GetActiveNodes()
	currentTurn := tree.TurnCount()

	gc.scorer.ScoreAll(activeNodes, currentTurn, queryVec)

	// MARK: identify nodes to sweep and branches to compact
	var toSweep []string
	parentChildren := make(map[string][]*domain.StateNode)

	for _, node := range activeNodes {
		if gc.shouldDeferSweepFacts(node) {
			continue
		}
		sweepScore := node.Scores.RetentionScore
		if gc.policy.DecisionSweepFloor > 0 && len(node.Facts) == 0 && len(node.Decisions) == 0 && node.NodeType == domain.NodeDecision {
			sweepScore = math.Max(sweepScore, gc.policy.DecisionSweepFloor)
		}
		if sweepScore < gc.policy.LowRelevanceThreshold {
			toSweep = append(toSweep, node.NodeID)
		}
		if node.ParentID != "" {
			parentChildren[node.ParentID] = append(parentChildren[node.ParentID], node)
		}
	}

	// SWEEP: move low-value nodes to archived
	for _, nodeID := range toSweep {
		tree.UpdateNodeStatus(nodeID, domain.StatusArchived)
		result.NodesSwept++
		if n, ok := tree.GetNode(nodeID); ok {
			result.TokensFreed += n.TokenCount
		}
	}

	// COMPACT: compress branches with too many children or fully resolved branches
	for parentID, children := range parentChildren {
		if len(children) < gc.policy.MaxChildrenPerNode && !allResolved(children) {
			continue
		}

		compressible := filterCompressible(children)
		if len(compressible) < 3 {
			continue
		}

		compressed, err := gc.compressor.Compress(ctx, compressible)
		if err != nil {
			log.Printf("gc: compress failed for parent %s: %v", parentID, err)
			continue
		}

		childIDs := make([]string, len(compressible))
		originalTokens := 0
		var allEventRefs []string
		var allIssues, allConstraints []string

		for i, c := range compressible {
			childIDs[i] = c.NodeID
			originalTokens += c.TokenCount
			allEventRefs = append(allEventRefs, c.RawEventRefs...)
			allIssues = append(allIssues, c.OpenIssues...)
			if c.NodeType == domain.NodeConstraint {
				allConstraints = append(allConstraints, c.Summary)
			}
		}

		compressed.SessionID = tree.SessionID()
		compressed.ParentID = parentID
		tree.ReplaceWithCompressed(parentID, childIDs, compressed)

		result.TokensFreed += originalTokens - compressed.TokenCount
		result.BranchesCompressed++

		// Build durable compressed branch record
		result.CompressedBranchRecords = append(result.CompressedBranchRecords, &store.CompressedBranch{
			BranchID:             compressed.NodeID,
			SessionID:            compressed.SessionID,
			TaskID:               compressed.TaskID,
			ParentNodeID:         parentID,
			OriginalNodeIDs:      childIDs,
			Summary:              compressed.Summary,
			KeyDecisions:         compressed.Decisions,
			ExactFacts:           compressed.Facts,
			OpenIssues:           allIssues,
			ImportantConstraints: allConstraints,
			RawEventRefs:         allEventRefs,
			OriginalTokenCount:   originalTokens,
			CompressedTokenCount: compressed.TokenCount,
			CreatedAt:            time.Now(),
		})
	}

	return result
}

// ForceRun runs GC regardless of policy triggers.
func (gc *GarbageCollector) ForceRun(ctx context.Context, tree *statetree.Tree, queryVec []float32) GCResult {
	return gc.Run(ctx, tree, ReasonManual, queryVec)
}

func allResolved(nodes []*domain.StateNode) bool {
	for _, n := range nodes {
		if n.Status == domain.StatusActive {
			return false
		}
	}
	return true
}

func filterCompressible(nodes []*domain.StateNode) []*domain.StateNode {
	var out []*domain.StateNode
	for _, n := range nodes {
		if n.NodeType == domain.NodeGoal || n.NodeType == domain.NodeConstraint {
			continue
		}
		out = append(out, n)
	}
	return out
}

// SimpleCompressor uses string concatenation to summarize branches without an LLM call.
type SimpleCompressor struct{}

func (c *SimpleCompressor) Compress(_ context.Context, nodes []*domain.StateNode) (*domain.StateNode, error) {
	mergedFacts, mergedDecisions := facts.MergeFromNodes(nodes, 0, 0)
	var issues []string
	var eventRefs []string
	var summaryParts []string

	for _, n := range nodes {
		issues = append(issues, n.OpenIssues...)
		eventRefs = append(eventRefs, n.RawEventRefs...)
		if n.Summary != "" {
			summaryParts = append(summaryParts, n.Summary)
		}
	}

	core := ""
	if len(summaryParts) > 3 {
		core = summaryParts[0] + " ... " + summaryParts[len(summaryParts)-1]
	} else {
		for i, s := range summaryParts {
			if i > 0 {
				core += " | "
			}
			core += s
		}
	}
	summaryBody := "Compressed branch: " + core
	summary := facts.VerifiedFactsPrefix(mergedFacts) + summaryBody

	return &domain.StateNode{
		NodeID:       fmt.Sprintf("compressed_%d", time.Now().UnixNano()),
		NodeType:     domain.NodeCompressedBranch,
		Status:       domain.StatusCompressed,
		Title:        fmt.Sprintf("Compressed %d nodes", len(nodes)),
		Summary:      summary,
		Facts:        mergedFacts,
		Decisions:    mergedDecisions,
		OpenIssues:   issues,
		RawEventRefs: eventRefs,
		TokenCount:   tokenizer.Default().Count(summary),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}
