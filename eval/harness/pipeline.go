package harness

import (
	"context"
	"fmt"
	"time"

	"github.com/chandrashekhartata/acgc/eval/datasets"
	"github.com/chandrashekhartata/acgc/internal/llm"
	"github.com/chandrashekhartata/acgc/internal/tokenizer"
)

// StrategyPipeline adapts any ContextStrategy to the Pipeline contract: it
// builds the context via the strategy, then performs the shared LLM call and
// records real token/latency metrics. Every strategy goes through the exact
// same LLM path so results are directly comparable.
type StrategyPipeline struct {
	strategy    ContextStrategy
	cfg         LLMConfig
	acgc        ACGCConfig
	counter     tokenizer.TokenCounter
	client      *llm.Client
	tokenBudget int
}

// NewStrategyPipeline wires a ContextStrategy into a runnable pipeline.
func NewStrategyPipeline(strategy ContextStrategy, cfg LLMConfig, acgcCfg ACGCConfig, counter tokenizer.TokenCounter) *StrategyPipeline {
	if counter == nil {
		counter = tokenizer.Default()
	}
	return &StrategyPipeline{
		strategy:    strategy,
		cfg:         cfg,
		acgc:        acgcCfg,
		counter:     counter,
		client:      cfg.build(),
		tokenBudget: acgcCfg.TokenBudget,
	}
}

// Strategy exposes the underlying context strategy (for metadata/reporting).
func (p *StrategyPipeline) Strategy() ContextStrategy { return p.strategy }

func (p *StrategyPipeline) Kind() PipelineKind { return p.strategy.Name() }

func (p *StrategyPipeline) Answer(ctx context.Context, history []datasets.Turn, probe datasets.Probe) (*ProbeResult, error) {
	sessionID := fmt.Sprintf("eval_%d", time.Now().UnixNano())
	taskID := "eval_task"

	in := StrategyInput{
		Ctx:          ctx,
		SessionID:    sessionID,
		TaskID:       taskID,
		SystemPrompt: evalSystemPrompt,
		History:      history,
		Nodes:        buildHistoryNodes(history, p.counter),
		Question:     probe.Question,
		TokenBudget:  p.tokenBudget,
		TokenCounter: p.counter,
	}

	pr := &ProbeResult{
		ProbeID:  probe.ID,
		Pipeline: p.strategy.Name(),
		Question: probe.Question,
	}

	out, err := p.strategy.BuildPrompt(in)
	if err != nil {
		pr.Error = err.Error()
		return pr, fmt.Errorf("%s build prompt: %w", p.strategy.Name(), err)
	}
	pr.OriginalTokenCount = out.OriginalTokenCount
	pr.CompiledTokenCount = out.CompiledTokenCount
	pr.IncludedNodes = out.IncludedNodes
	pr.ExcludedNodes = out.ExcludedNodes

	messages := assembleMessages(in.SystemPrompt, out.FinalPrompt, probe.Question)

	start := time.Now()
	result, genErr := GenerateWithRetry(ctx, p.client, messages, p.cfg.Temperature, p.cfg.MaxTokens)
	pr.LatencyMs = time.Since(start).Milliseconds()

	if genErr != nil {
		pr.Error = genErr.Error()
		return pr, fmt.Errorf("%s llm call: %w", p.strategy.Name(), genErr)
	}
	pr.Response = result.Content
	pr.PromptTokens = result.PromptTokens
	pr.OutputTokens = result.CompletionTokens
	return pr, nil
}
