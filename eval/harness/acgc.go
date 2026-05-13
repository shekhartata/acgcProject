package harness

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chandrashekhartata/acgc/eval/datasets"
	"github.com/chandrashekhartata/acgc/internal/compiler"
	"github.com/chandrashekhartata/acgc/internal/domain"
	"github.com/chandrashekhartata/acgc/internal/embedding"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/llm"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/session"
	"github.com/chandrashekhartata/acgc/internal/statetree"
	"github.com/chandrashekhartata/acgc/internal/vectorindex"
)

// ACGCConfig mirrors the runtime ACGC policy knobs. Defaults match the
// production server (cmd/acgc/main.go).
type ACGCConfig struct {
	TokenBudget           int
	MaxTreeDepth          int
	MaxChildrenPerNode    int
	LowRelevanceThreshold float64
	StaleAfterTurns       int
	MaxTokensPerNode      int
	// DecisionSweepFloor: GC soft floor for NodeDecision with empty Facts/Decisions (0 disables).
	// Must stay strictly below LowRelevanceThreshold or bare decisions become un-sweepable.
	DecisionSweepFloor float64
	// MaxActiveNodes: count-based GC trigger. 0 disables.
	MaxActiveNodes int
	// SweepHeadroomRatio: soft trigger at ratio × TokenBudget. 0 disables.
	SweepHeadroomRatio  float64
	ArchiveSemanticTopK int

	// Optional semantic scoring. When Embedder is nil, the eval runs in
	// pure-heuristic mode (matches v1).
	Embedder       embedding.Provider
	SemanticWeight float64
	TopKAtCompile  int
	HNSWConfig     vectorindex.Config
}

func DefaultACGCConfig() ACGCConfig {
	return ACGCConfig{
		TokenBudget:           6000,
		MaxTreeDepth:          10,
		MaxChildrenPerNode:    50,
		LowRelevanceThreshold: 0.30,
		StaleAfterTurns:       15,
		MaxTokensPerNode:      2000,
		// Phase 2: 0.35 → 0.20. Floor must sit below LowRelevanceThreshold,
		// otherwise bare NodeDecision nodes are never swept.
		DecisionSweepFloor: 0.20,
		// Phase 2: count + headroom triggers so GC actually fires on short
		// conversations that never approach TokenBudget.
		MaxActiveNodes:      25,
		SweepHeadroomRatio:  0.60,
		ArchiveSemanticTopK: 12,
	}
}

// ACGCPipeline replays the conversation through the in-process ACGC stack
// (statetree + scorer + GC + compiler) and sends the compiled prompt to the
// LLM. This is what production ACGC does, minus MongoDB persistence.
type ACGCPipeline struct {
	cfg    LLMConfig
	acgc   ACGCConfig
	client *llm.Client
}

func NewACGCPipeline(cfg LLMConfig, acgcCfg ACGCConfig) *ACGCPipeline {
	return &ACGCPipeline{cfg: cfg, acgc: acgcCfg, client: cfg.build()}
}

func (p *ACGCPipeline) Kind() PipelineKind { return PipelineACGC }

func (p *ACGCPipeline) Answer(ctx context.Context, history []datasets.Turn, probe datasets.Probe) (*ProbeResult, error) {
	sessionID := fmt.Sprintf("eval_%d", time.Now().UnixNano())
	taskID := "eval_task"

	tree := statetree.NewTree(sessionID, taskID)
	sc := scorer.NewScorer(p.acgc.StaleAfterTurns, p.acgc.MaxTokensPerNode)
	if p.acgc.SemanticWeight > 0 {
		sc.SetSemanticWeight(p.acgc.SemanticWeight)
	}
	comp := compiler.NewCompiler(p.acgc.TokenBudget)
	collector := gc.NewGarbageCollector(gc.Policy{
		MaxPromptTokens:       p.acgc.TokenBudget,
		MaxTreeDepth:          p.acgc.MaxTreeDepth,
		MaxChildrenPerNode:    p.acgc.MaxChildrenPerNode,
		LowRelevanceThreshold: p.acgc.LowRelevanceThreshold,
		DecisionSweepFloor:    p.acgc.DecisionSweepFloor,
		MaxActiveNodes:        p.acgc.MaxActiveNodes,
		SweepHeadroomRatio:    p.acgc.SweepHeadroomRatio,
		StaleAfterTurns:       p.acgc.StaleAfterTurns,
	}, sc, &gc.SimpleCompressor{})

	var (
		activeIdx  vectorindex.Index
		archIdx    vectorindex.Index
		semanticOn = p.acgc.Embedder != nil
	)
	if semanticOn {
		activeIdx = vectorindex.NewHNSW(p.acgc.HNSWConfig)
		archIdx = vectorindex.NewHNSW(p.acgc.HNSWConfig)
	}

	var lastUserEmbedding []float32
	// Replay every turn through ACGC.
	for i, t := range history {
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
			TokenCount: estTokens(t.Content),
			CreatedAt:  time.Now(),
		}
		turnNum := tree.IncrementTurn()
		event.Sequence = turnNum
		node := tree.AddNode(event)

		if semanticOn && node != nil {
			text := embeddingPayload(node, event)
			if text != "" {
				vec, err := p.acgc.Embedder.Embed(ctx, text)
				if err != nil {
					log.Printf("eval/acgc: embed failed at turn %d: %v", turnNum, err)
				} else {
					node.Embedding = vec
					node.EmbedModel = p.acgc.Embedder.Model()
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
		if should, reason := collector.ShouldRun(tree, activeTokens); should {
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
	systemPrompt := "You are a helpful technical assistant. Answer concisely and accurately based on the provided context."

	var compiled *domain.CompiledPrompt
	if semanticOn && probe.Question != "" {
		qVec, err := p.acgc.Embedder.Embed(ctx, probe.Question)
		if err != nil {
			log.Printf("eval/acgc: compile-time embed failed: %v", err)
			compiled = comp.Compile(sessionID, taskID, probe.Question, active, systemPrompt)
		} else {
			topK := p.acgc.TopKAtCompile
			if topK <= 0 {
				topK = 12
			}
			kz := p.acgc.ArchiveSemanticTopK
			if kz <= 0 {
				kz = 12
			}
			hitsA, qErr := activeIdx.Query(qVec, topK)
			if qErr != nil {
				log.Printf("eval/acgc: active HNSW query failed: %v", qErr)
				compiled = comp.Compile(sessionID, taskID, probe.Question, active, systemPrompt)
			} else {
				hitsZ, zErr := archIdx.Query(qVec, kz)
				if zErr != nil {
					log.Printf("eval/acgc: archive HNSW query failed: %v", zErr)
					compiled = comp.Compile(sessionID, taskID, probe.Question, active, systemPrompt)
				} else {
					w := p.acgc.SemanticWeight
					if w <= 0 {
						w = 0.20
					}
					merged := session.MergeSemanticHits(hitsA, hitsZ)
					nodes := session.NodesForSemanticCompile(tree, active, merged)
					compiled = comp.CompileWithSemantic(sessionID, taskID, probe.Question, nodes, systemPrompt, w, merged)
				}
			}
		}
	} else {
		compiled = comp.Compile(sessionID, taskID, probe.Question, active, systemPrompt)
	}

	// Wire format (matches gateway.Server.Run, Phase 2):
	//   [system, user(FinalPrompt = context body), user(probe.Question)]
	// FinalPrompt no longer inlines the probe — it's the structured context.
	// The probe is sent as its own user message so the framing matches baseline
	// (which also sends the question as a separate user turn).
	messages := make([]llm.ChatMessage, 0, 3)
	if compiled.SystemPrompt != "" {
		messages = append(messages, llm.ChatMessage{Role: "system", Content: compiled.SystemPrompt})
	}
	if compiled.FinalPrompt != "" {
		messages = append(messages, llm.ChatMessage{Role: "user", Content: compiled.FinalPrompt})
	}
	messages = append(messages, llm.ChatMessage{Role: "user", Content: probe.Question})

	start := time.Now()
	result, err := p.client.Generate(ctx, messages, p.cfg.Temperature, p.cfg.MaxTokens)
	latency := time.Since(start)

	pr := &ProbeResult{
		ScenarioID: "",
		ProbeID:    probe.ID,
		Pipeline:   PipelineACGC,
		Question:   probe.Question,
		LatencyMs:  latency.Milliseconds(),
	}
	if err != nil {
		pr.Error = err.Error()
		return pr, fmt.Errorf("acgc llm call: %w", err)
	}
	pr.Response = result.Content
	pr.PromptTokens = result.PromptTokens
	pr.OutputTokens = result.CompletionTokens
	return pr, nil
}

func estTokens(s string) int { return len(s) / 4 }

// embeddingPayload mirrors the helper in internal/session/manager.go.
// Kept in sync intentionally — the eval pipeline should embed the same text
// the runtime would, otherwise the eval doesn't reflect production behavior.
func embeddingPayload(node *domain.StateNode, event *domain.Event) string {
	if node != nil {
		if node.Summary != "" && node.Title != "" {
			return node.Title + ": " + node.Summary
		}
		if node.Summary != "" {
			return node.Summary
		}
		if node.Title != "" {
			return node.Title
		}
	}
	if event != nil {
		return event.Payload
	}
	return ""
}
