package harness

import (
	"context"
	"fmt"
	"time"

	"github.com/chandrashekhartata/acgc/eval/datasets"
	"github.com/chandrashekhartata/acgc/internal/llm"
)

// BaselinePipeline sends the full untouched conversation history to the LLM,
// then appends the probe question as the final user turn. This is the
// "no-ACGC" comparison group — naive concatenation of every prior turn.
type BaselinePipeline struct {
	cfg    LLMConfig
	client *llm.Client
}

func NewBaselinePipeline(cfg LLMConfig) *BaselinePipeline {
	return &BaselinePipeline{cfg: cfg, client: cfg.build()}
}

func (p *BaselinePipeline) Kind() PipelineKind { return PipelineBaseline }

func (p *BaselinePipeline) Answer(ctx context.Context, history []datasets.Turn, probe datasets.Probe) (*ProbeResult, error) {
	messages := make([]llm.ChatMessage, 0, len(history)+2)
	messages = append(messages, llm.ChatMessage{
		Role:    "system",
		Content: "You are a helpful technical assistant. Answer concisely and accurately based on the conversation context.",
	})
	for _, t := range history {
		messages = append(messages, llm.ChatMessage{Role: t.Role, Content: t.Content})
	}
	messages = append(messages, llm.ChatMessage{Role: "user", Content: probe.Question})

	start := time.Now()
	result, err := p.client.Generate(ctx, messages, p.cfg.Temperature, p.cfg.MaxTokens)
	latency := time.Since(start)

	pr := &ProbeResult{
		ScenarioID: "", // filled by runner
		ProbeID:    probe.ID,
		Pipeline:   PipelineBaseline,
		Question:   probe.Question,
		LatencyMs:  latency.Milliseconds(),
	}
	if err != nil {
		pr.Error = err.Error()
		return pr, fmt.Errorf("baseline llm call: %w", err)
	}
	pr.Response = result.Content
	pr.PromptTokens = result.PromptTokens
	pr.OutputTokens = result.CompletionTokens
	return pr, nil
}
