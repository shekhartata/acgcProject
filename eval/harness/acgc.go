package harness

import (
	"context"
	"fmt"
	"time"

	"github.com/chandrashekhartata/acgc/eval/datasets"
	"github.com/chandrashekhartata/acgc/internal/compiler"
	"github.com/chandrashekhartata/acgc/internal/domain"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/llm"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/statetree"
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
}

func DefaultACGCConfig() ACGCConfig {
	return ACGCConfig{
		TokenBudget:           6000,
		MaxTreeDepth:          10,
		MaxChildrenPerNode:    50,
		LowRelevanceThreshold: 0.30,
		StaleAfterTurns:       15,
		MaxTokensPerNode:      2000,
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
	comp := compiler.NewCompiler(p.acgc.TokenBudget)
	collector := gc.NewGarbageCollector(gc.Policy{
		MaxPromptTokens:       p.acgc.TokenBudget,
		MaxTreeDepth:          p.acgc.MaxTreeDepth,
		MaxChildrenPerNode:    p.acgc.MaxChildrenPerNode,
		LowRelevanceThreshold: p.acgc.LowRelevanceThreshold,
		StaleAfterTurns:       p.acgc.StaleAfterTurns,
	}, sc, &gc.SimpleCompressor{})

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
		tree.AddNode(event)

		active := tree.GetActiveNodes()
		sc.ScoreAll(active, turnNum)

		activeTokens := 0
		for _, n := range active {
			activeTokens += n.TokenCount
		}
		if should, reason := collector.ShouldRun(tree, activeTokens); should {
			collector.Run(ctx, tree, reason)
		}
	}

	active := tree.GetActiveNodes()
	systemPrompt := "You are a helpful technical assistant. Answer concisely and accurately based on the provided context."
	compiled := comp.Compile(sessionID, taskID, probe.Question, active, systemPrompt)

	messages := []llm.ChatMessage{
		{Role: "system", Content: compiled.FinalPrompt},
		{Role: "user", Content: probe.Question},
	}

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
