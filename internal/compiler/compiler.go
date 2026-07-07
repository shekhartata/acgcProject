package compiler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/tokenizer"
	"github.com/shekhartata/acgcProject/internal/vectorindex"
)

type Compiler struct {
	tokenBudget       int
	tokenCounter      tokenizer.TokenCounter
	cacheStableRender bool
}

// WithCacheStableRender enables deterministic chronological render order after
// budget selection. Score/semantic sort still controls inclusion.
func (c *Compiler) WithCacheStableRender(enabled bool) *Compiler {
	c.cacheStableRender = enabled
	return c
}

// NewCompiler builds a compiler with the given token budget. It uses the
// process-wide default token counter, preserving the original single-argument
// signature for existing callers (PRD backward-compatibility requirement).
func NewCompiler(tokenBudget int) *Compiler {
	return NewCompilerWithCounter(tokenBudget, tokenizer.Default())
}

// NewCompilerWithCounter builds a compiler with an explicit token counter.
// Prefer this when the caller knows the target model so token accounting is
// accurate. A nil counter falls back to the process-wide default.
func NewCompilerWithCounter(tokenBudget int, counter tokenizer.TokenCounter) *Compiler {
	if counter == nil {
		counter = tokenizer.Default()
	}
	return &Compiler{tokenBudget: tokenBudget, tokenCounter: counter}
}

func (c *Compiler) count(s string) int {
	if c.tokenCounter == nil {
		return len(s) / 4
	}
	return c.tokenCounter.Count(s)
}

// Compile builds an optimized prompt from the state tree nodes.
// Uses the pre-computed RetentionScore on each node for ranking.
func (c *Compiler) Compile(
	sessionID, taskID, userMessage string,
	nodes []*domain.StateNode,
	systemPrompt string,
) *domain.CompiledPrompt {
	return c.compile(sessionID, taskID, userMessage, nodes, systemPrompt, nil)
}

// CompileWithSemantic re-blends a fresh semantic signal at compile time using
// the imminent user query embedding. Nodes that show up in topK get their
// score boosted by semanticWeight * (hitScore - previous semantic component).
// Nodes outside topK are unaffected. This keeps the hot-path overhead bounded
// to a single linear pass + a small map lookup.
func (c *Compiler) CompileWithSemantic(
	sessionID, taskID, userMessage string,
	nodes []*domain.StateNode,
	systemPrompt string,
	semanticWeight float64,
	topK []vectorindex.Hit,
) *domain.CompiledPrompt {
	if len(topK) == 0 || semanticWeight <= 0 {
		return c.compile(sessionID, taskID, userMessage, nodes, systemPrompt, nil)
	}
	hitScores := make(map[string]float64, len(topK))
	for _, h := range topK {
		hitScores[h.NodeID] = float64(h.Score)
	}
	effective := make(map[string]float64, len(nodes))
	for _, n := range nodes {
		base := n.Scores.RetentionScore
		if hit, ok := hitScores[n.NodeID]; ok {
			// Replace whatever stale semantic contribution was baked in
			// with the fresh per-compile signal. Subtract old, add new.
			base += semanticWeight * (hit - n.Scores.Semantic)
		}
		effective[n.NodeID] = base
	}
	return c.compile(sessionID, taskID, userMessage, nodes, systemPrompt, effective)
}

// compile is the shared assembly path. When effective is nil, nodes are
// sorted by their pre-computed RetentionScore; when provided, the map
// overrides the sort key without mutating the node scores themselves.
//
// Phase 2 token-reduction notes:
//   - userMessage / "## Current Request" is NOT inlined into FinalPrompt.
//     Callers send the probe as its own user chat message (CurrentUserMessage)
//     so the wire format is [system, user(context), user(probe)] — same
//     framing as the baseline pipeline, no decoration overhead.
//   - goals/constraints are no longer "always include"; they go through the
//     same selectWithinBudget gate as everything else (still first in priority
//     so they win ties).
//   - facts:/decisions: sub-lines are emitted only for compressed/summary
//     nodes. Active nodes already carry that text inside Summary; duplicating
//     it bloats the prompt by ~7 tokens per node.
func (c *Compiler) compile(
	sessionID, taskID, userMessage string,
	nodes []*domain.StateNode,
	systemPrompt string,
	effective map[string]float64,
) *domain.CompiledPrompt {
	cp := &domain.CompiledPrompt{
		CompiledPromptID:   fmt.Sprintf("cp_%d", time.Now().UnixNano()),
		SessionID:          sessionID,
		TaskID:             taskID,
		CurrentUserMessage: userMessage,
		SystemPrompt:       systemPrompt,
		CreatedAt:          time.Now(),
		CacheStableRender:  c.cacheStableRender,
	}

	originalTokens := c.count(systemPrompt) + c.count(userMessage)
	for _, n := range nodes {
		originalTokens += n.TokenCount
	}
	cp.OriginalTokenCount = originalTokens

	var goals, constraints, decisions, toolOutputs, compressed, issues []*domain.StateNode
	var rest []*domain.StateNode

	for _, n := range nodes {
		switch n.NodeType {
		case domain.NodeGoal:
			goals = append(goals, n)
		case domain.NodeConstraint:
			constraints = append(constraints, n)
		case domain.NodeDecision:
			decisions = append(decisions, n)
		case domain.NodeToolResult:
			toolOutputs = append(toolOutputs, n)
		case domain.NodeCompressedBranch, domain.NodeSummary:
			compressed = append(compressed, n)
		case domain.NodeIssue:
			issues = append(issues, n)
		default:
			rest = append(rest, n)
		}
	}

	sortBy := func(ns []*domain.StateNode) {
		if effective == nil {
			sortByScore(ns)
			return
		}
		sortByEffective(ns, effective)
	}
	sortBy(goals)
	sortBy(constraints)
	sortBy(decisions)
	sortBy(toolOutputs)
	sortBy(compressed)
	sortBy(issues)
	sortBy(rest)

	// Budget bookkeeping: systemPrompt + userMessage will be sent as separate
	// chat messages; reserve their tokens so the body sections don't blow
	// past the API's prompt_tokens budget.
	var sections []string
	tokensUsed := 0
	if systemPrompt != "" {
		tokensUsed += c.count(systemPrompt)
	}
	if userMessage != "" {
		tokensUsed += c.count(userMessage)
	}

	// Record the top goal title for downstream consumers (telemetry, audit),
	// but the actual inclusion decision goes through the budget gate below.
	if len(goals) > 0 {
		cp.ActiveGoal = goals[0].Title
	}
	for _, cn := range constraints {
		cp.ActiveConstraints = append(cp.ActiveConstraints, cn.Summary)
	}

	// Single budget-respecting pass. Goals first (highest priority), then
	// constraints, then decisions, etc. Nothing bypasses selectWithinBudget.
	remaining := c.tokenBudget - tokensUsed
	priorityBuckets := [][]*domain.StateNode{goals, constraints, decisions, compressed, toolOutputs, issues, rest}
	bucketLabels := []string{"Active Goals", "Constraints", "Key Decisions", "Prior Context", "Tool Outputs", "Open Issues", "Additional Context"}

	for i, bucket := range priorityBuckets {
		if remaining <= 0 || len(bucket) == 0 {
			continue
		}

		selected, excluded, used := c.selectWithinBudget(bucket, remaining)
		if len(selected) > 0 {
			render := selected
			if c.cacheStableRender {
				render = StabilizeRenderOrder(selected)
			}
			text := formatSection(bucketLabels[i], render)
			sections = append(sections, text)
			tokensUsed += used
			remaining -= used

			switch bucketLabels[i] {
			case "Key Decisions":
				for _, s := range render {
					cp.RelevantDecisions = append(cp.RelevantDecisions, s.Summary)
				}
			case "Tool Outputs":
				for _, s := range render {
					cp.RelevantToolOutputs = append(cp.RelevantToolOutputs, s.Summary)
				}
			case "Prior Context":
				for _, s := range render {
					cp.CompressedContext = append(cp.CompressedContext, s.Summary)
				}
			case "Open Issues":
				for _, s := range render {
					cp.OpenIssues = append(cp.OpenIssues, s.Summary)
				}
			}

			for _, e := range excluded {
				cp.ExcludedNodeRefs = append(cp.ExcludedNodeRefs, e.NodeID)
			}
		}
	}

	cp.FinalPrompt = strings.Join(sections, "\n\n---\n\n")
	// CompiledTokenCount mirrors the wire: system + context body + user probe,
	// all as separate chat messages.
	cp.CompiledTokenCount = c.count(systemPrompt) + c.count(cp.FinalPrompt) + c.count(userMessage)

	return cp
}

func (c *Compiler) selectWithinBudget(nodes []*domain.StateNode, budget int) (selected, excluded []*domain.StateNode, tokensUsed int) {
	for _, n := range nodes {
		cost := n.TokenCount
		if cost == 0 {
			cost = c.count(n.Summary)
		}
		// Only compressed/summary nodes will render facts:/decisions: sub-lines
		// (active nodes already include that text in their Summary). Cost the
		// extra tokens here so the budget gate matches what formatSection emits.
		if emitsFactsLine(n) {
			cost += c.count(strings.Join(n.Facts, "; "))
			cost += c.count(strings.Join(n.Decisions, "; "))
		}
		if tokensUsed+cost <= budget {
			selected = append(selected, n)
			tokensUsed += cost
		} else {
			excluded = append(excluded, n)
		}
	}
	return
}

// emitsFactsLine reports whether formatSection will emit a `facts:` / `decisions:`
// sub-line for this node. Active nodes already carry that information verbatim
// in Summary, so duplicating it on the line below is pure bloat. We only
// surface the sub-lines on compressed/summary nodes where the original payload
// is gone and the Facts/Decisions slices are the only place those entities live.
func emitsFactsLine(n *domain.StateNode) bool {
	return n.NodeType == domain.NodeCompressedBranch || n.NodeType == domain.NodeSummary
}

func formatSection(label string, nodes []*domain.StateNode) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## %s\n", label))
	for _, n := range nodes {
		if n.Summary != "" {
			b.WriteString(fmt.Sprintf("- %s\n", n.Summary))
		} else if n.Title != "" {
			b.WriteString(fmt.Sprintf("- %s\n", n.Title))
		}
		if !emitsFactsLine(n) {
			continue
		}
		if len(n.Facts) > 0 {
			b.WriteString(fmt.Sprintf("  facts: %s\n", strings.Join(n.Facts, "; ")))
		}
		if len(n.Decisions) > 0 {
			b.WriteString(fmt.Sprintf("  decisions: %s\n", strings.Join(n.Decisions, "; ")))
		}
	}
	return b.String()
}

func sortByScore(nodes []*domain.StateNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Scores.RetentionScore > nodes[j].Scores.RetentionScore
	})
}

func sortByEffective(nodes []*domain.StateNode, effective map[string]float64) {
	sort.Slice(nodes, func(i, j int) bool {
		ai, aj := nodes[i].Scores.RetentionScore, nodes[j].Scores.RetentionScore
		if v, ok := effective[nodes[i].NodeID]; ok {
			ai = v
		}
		if v, ok := effective[nodes[j].NodeID]; ok {
			aj = v
		}
		return ai > aj
	})
}
