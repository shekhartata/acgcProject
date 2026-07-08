package harness

import (
	"fmt"
	"log"
	"time"

	"github.com/shekhartata/acgcProject/internal/compiler"
	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/gc"
	"github.com/shekhartata/acgcProject/internal/scorer"
	"github.com/shekhartata/acgcProject/internal/session"
	"github.com/shekhartata/acgcProject/internal/statetree"
	"github.com/shekhartata/acgcProject/internal/tokenizer"
	"github.com/shekhartata/acgcProject/internal/vectorindex"
)

// acgcStrategy runs the full ACGC stack (statetree + scorer + GC + compiler,
// optionally the HNSW semantic layer) to build an optimized context prompt.
type acgcStrategy struct {
	cfg     ACGCConfig
	counter tokenizer.TokenCounter
}

// NewACGCStrategy returns the acgc context strategy.
func NewACGCStrategy(cfg ACGCConfig, counter tokenizer.TokenCounter) ContextStrategy {
	if counter == nil {
		counter = tokenizer.Default()
	}
	return &acgcStrategy{cfg: cfg, counter: counter}
}

func (s *acgcStrategy) Name() PipelineKind { return PipelineACGC }

func (s *acgcStrategy) BuildPrompt(in StrategyInput) (StrategyOutput, error) {
	ctx := in.Ctx
	sessionID := in.SessionID
	taskID := in.TaskID

	tree := statetree.NewTree(sessionID, taskID)
	sc := scorer.NewScorer(s.cfg.StaleAfterTurns, s.cfg.MaxTokensPerNode)
	if s.cfg.SemanticWeight > 0 {
		sc.SetSemanticWeight(s.cfg.SemanticWeight)
	}
	comp := compiler.NewCompilerWithCounter(in.TokenBudget, s.counter)
	if s.cfg.CacheStableRender {
		comp.WithCacheStableRender(true)
	}
	collector := gc.NewGarbageCollector(gc.Policy{
		MaxPromptTokens:       in.TokenBudget,
		MaxTreeDepth:          s.cfg.MaxTreeDepth,
		MaxChildrenPerNode:    s.cfg.MaxChildrenPerNode,
		LowRelevanceThreshold: s.cfg.LowRelevanceThreshold,
		DecisionSweepFloor:    s.cfg.DecisionSweepFloor,
		MaxActiveNodes:        s.cfg.MaxActiveNodes,
		SweepHeadroomRatio:    s.cfg.SweepHeadroomRatio,
		StaleAfterTurns:       s.cfg.StaleAfterTurns,
	}, sc, &gc.SimpleCompressor{})

	var (
		activeIdx  vectorindex.Index
		archIdx    vectorindex.Index
		semanticOn = s.cfg.Embedder != nil
	)
	if semanticOn {
		activeIdx = vectorindex.NewHNSW(s.cfg.HNSWConfig)
		archIdx = vectorindex.NewHNSW(s.cfg.HNSWConfig)
	}

	var lastUserEmbedding []float32
	for i, t := range in.History {
		eventType := domain.EventUserPrompt
		if t.Role == "assistant" {
			eventType = domain.EventLLMResponse
		}
		event := &domain.Event{
			EventID:    fmt.Sprintf("evt_%s_%d", sessionID, i),
			SessionID:  sessionID,
			TaskID:     taskID,
			EventType:  eventType,
			Payload:    t.Content,
			TokenCount: s.counter.Count(t.Content),
			CreatedAt:  time.Now(),
		}
		turnNum := tree.IncrementTurn()
		event.Sequence = turnNum
		node := tree.AddNode(event)

		if semanticOn && node != nil {
			text := embeddingPayload(node, event)
			if text != "" {
				vec, err := s.cfg.Embedder.Embed(ctx, text)
				if err != nil {
					log.Printf("eval/acgc: embed failed at turn %d: %v", turnNum, err)
				} else {
					node.Embedding = vec
					node.EmbedModel = s.cfg.Embedder.Model()
					if err := activeIdx.Insert(node.NodeID, vec); err != nil {
						log.Printf("eval/acgc: HNSW insert failed: %v", err)
					}
					if t.Role == "user" {
						lastUserEmbedding = vec
					}
				}
			}
		}

		active := tree.GetActiveNodes()
		sc.ScoreAll(active, turnNum, lastUserEmbedding)

		activeTokens := 0
		for _, n := range active {
			activeTokens += n.TokenCount
		}
		if should, reason := collector.ShouldRun(tree, activeTokens, 0); should {
			preIDs := make(map[string]bool, len(active))
			for _, n := range active {
				preIDs[n.NodeID] = true
			}
			collector.Run(ctx, tree, reason, lastUserEmbedding)
			if semanticOn && activeIdx != nil && archIdx != nil {
				postActive := tree.GetActiveNodes()
				postIDs := make(map[string]bool, len(postActive))
				for _, n := range postActive {
					postIDs[n.NodeID] = true
				}
				for id := range preIDs {
					if postIDs[id] {
						continue
					}
					if n, ok := tree.GetNode(id); ok && len(n.Embedding) > 0 {
						_ = archIdx.Insert(id, n.Embedding)
					}
					activeIdx.Delete(id)
				}
			}
		}
	}

	active := tree.GetActiveNodes()

	var compiled *domain.CompiledPrompt
	if semanticOn && in.Question != "" {
		compiled = s.compileSemantic(in, comp, tree, active, activeIdx, archIdx)
	} else {
		compiled = comp.Compile(sessionID, taskID, in.Question, active, in.SystemPrompt)
	}

	return StrategyOutput{
		FinalPrompt:        compiled.FinalPrompt,
		OriginalTokenCount: compiled.OriginalTokenCount,
		CompiledTokenCount: compiled.CompiledTokenCount,
		IncludedNodes:      len(active) - len(compiled.ExcludedNodeRefs),
		ExcludedNodes:      len(compiled.ExcludedNodeRefs),
	}, nil
}

// compileSemantic performs the compile-time semantic re-blend, falling back to
// the heuristic compile on any failure so a prompt is always produced.
func (s *acgcStrategy) compileSemantic(
	in StrategyInput,
	comp *compiler.Compiler,
	tree *statetree.Tree,
	active []*domain.StateNode,
	activeIdx, archIdx vectorindex.Index,
) *domain.CompiledPrompt {
	qVec, err := s.cfg.Embedder.Embed(in.Ctx, in.Question)
	if err != nil {
		log.Printf("eval/acgc: compile-time embed failed: %v", err)
		return comp.Compile(in.SessionID, in.TaskID, in.Question, active, in.SystemPrompt)
	}
	topK := s.cfg.TopKAtCompile
	if topK <= 0 {
		topK = 12
	}
	kz := s.cfg.ArchiveSemanticTopK
	if kz <= 0 {
		kz = 12
	}
	hitsA, qErr := activeIdx.Query(qVec, topK)
	if qErr != nil {
		log.Printf("eval/acgc: active HNSW query failed: %v", qErr)
		return comp.Compile(in.SessionID, in.TaskID, in.Question, active, in.SystemPrompt)
	}
	hitsZ, zErr := archIdx.Query(qVec, kz)
	if zErr != nil {
		log.Printf("eval/acgc: archive HNSW query failed: %v", zErr)
		return comp.Compile(in.SessionID, in.TaskID, in.Question, active, in.SystemPrompt)
	}
	w := s.cfg.SemanticWeight
	if w <= 0 {
		w = 0.20
	}
	merged := session.MergeSemanticHits(hitsA, hitsZ)
	nodes := session.NodesForSemanticCompile(tree, active, merged)
	return comp.CompileWithSemantic(in.SessionID, in.TaskID, in.Question, nodes, in.SystemPrompt, w, merged)
}
