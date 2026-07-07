package harness

import (
	"context"

	"github.com/shekhartata/acgcProject/eval/datasets"
	"github.com/shekhartata/acgcProject/internal/llm"
)

// PipelineKind identifies which context-management strategy produced a result.
type PipelineKind string

const (
	// PipelineNaive includes all available historical context (chronological)
	// up to the token budget — the "no context management" reference.
	PipelineNaive PipelineKind = "naive_full_history"
	// PipelineSliding keeps only the most recent context up to the token budget.
	PipelineSliding PipelineKind = "sliding_window"
	// PipelineACGC is the full ACGC stack (statetree + scorer + GC + compiler).
	PipelineACGC PipelineKind = "acgc"

	// PipelineBaseline is retained as an alias of the naive reference so older
	// cache files and callers keep resolving. New code should use PipelineNaive.
	PipelineBaseline PipelineKind = PipelineNaive
)

// AllStrategyKinds lists the selectable strategies in report order.
func AllStrategyKinds() []PipelineKind {
	return []PipelineKind{PipelineNaive, PipelineSliding, PipelineACGC}
}

// ParseStrategyKind resolves a user-supplied strategy name (case-insensitive,
// with a few friendly aliases) to a PipelineKind.
func ParseStrategyKind(s string) (PipelineKind, bool) {
	switch s {
	case "naive", "naive_full_history", "baseline", "full", "full_history":
		return PipelineNaive, true
	case "sliding", "sliding_window", "window":
		return PipelineSliding, true
	case "acgc":
		return PipelineACGC, true
	default:
		return "", false
	}
}

// ProbeResult is the LLM's answer to a single probe question, including
// real token counts measured from the API response.
type ProbeResult struct {
	ScenarioID     string       `json:"scenario_id"`
	ProbeID        string       `json:"probe_id"`
	Pipeline       PipelineKind `json:"pipeline"`
	CacheKeySuffix string       `json:"cache_key_suffix,omitempty"`
	Question       string       `json:"question"`
	Response       string       `json:"response"`
	PromptTokens       int          `json:"prompt_tokens"`
	OutputTokens       int          `json:"output_tokens"`
	CachedPromptTokens int          `json:"cached_prompt_tokens,omitempty"`
	LatencyMs          int64        `json:"latency_ms"`
	Cached         bool         `json:"cached"`
	Error          string       `json:"error,omitempty"`

	// Context-assembly accounting from the strategy (pre-wire estimates using
	// the model-aware tokenizer). PromptTokens above remains the ground-truth
	// count reported by the API.
	OriginalTokenCount int `json:"original_token_count,omitempty"`
	CompiledTokenCount int `json:"compiled_token_count,omitempty"`
	IncludedNodes      int `json:"included_nodes,omitempty"`
	ExcludedNodes      int `json:"excluded_nodes,omitempty"`
}

// ScenarioRun is the full output of running a single scenario through a
// single pipeline. Contains the probe results plus any per-pipeline metadata
// useful for the report.
type ScenarioRun struct {
	ScenarioID      string        `json:"scenario_id"`
	Pipeline        PipelineKind  `json:"pipeline"`
	ProbeResults    []ProbeResult `json:"probe_results"`
	TotalPromptToks int           `json:"total_prompt_tokens"`
	TotalOutputToks int           `json:"total_output_tokens"`
	TotalLatencyMs  int64         `json:"total_latency_ms"`
}

// Pipeline is the contract both Baseline and ACGC implement. It answers a
// single probe question, given the conversation history that preceded it.
type Pipeline interface {
	Kind() PipelineKind
	Answer(ctx context.Context, history []datasets.Turn, probe datasets.Probe) (*ProbeResult, error)
}

// CachingPipeline optionally distinguishes cache entries beyond Kind() — e.g.
// acgc with semantic scoring uses a different suffix so heuristic-only cached
// responses are not replayed when -semantic is enabled.
type CachingPipeline interface {
	Pipeline
	CacheKeySuffix() string
}

// LLMConfig is the minimal config the eval needs to talk to a real LLM.
type LLMConfig struct {
	BaseURL     string
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
}

func (c LLMConfig) build() *llm.Client {
	return llm.NewClient(llm.Config{
		BaseURL: c.BaseURL,
		APIKey:  c.APIKey,
		Model:   c.Model,
	})
}
