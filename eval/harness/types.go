package harness

import (
	"context"

	"github.com/chandrashekhartata/acgc/eval/datasets"
	"github.com/chandrashekhartata/acgc/internal/llm"
)

// PipelineKind identifies which prompt-assembly strategy was used.
type PipelineKind string

const (
	PipelineBaseline PipelineKind = "baseline"
	PipelineACGC     PipelineKind = "acgc"
)

// ProbeResult is the LLM's answer to a single probe question, including
// real token counts measured from the API response.
type ProbeResult struct {
	ScenarioID   string             `json:"scenario_id"`
	ProbeID      string             `json:"probe_id"`
	Pipeline     PipelineKind       `json:"pipeline"`
	Question     string             `json:"question"`
	Response     string             `json:"response"`
	PromptTokens int                `json:"prompt_tokens"`
	OutputTokens int                `json:"output_tokens"`
	LatencyMs    int64              `json:"latency_ms"`
	Cached       bool               `json:"cached"`
	Error        string             `json:"error,omitempty"`
}

// ScenarioRun is the full output of running a single scenario through a
// single pipeline. Contains the probe results plus any per-pipeline metadata
// useful for the report.
type ScenarioRun struct {
	ScenarioID      string         `json:"scenario_id"`
	Pipeline        PipelineKind   `json:"pipeline"`
	ProbeResults    []ProbeResult  `json:"probe_results"`
	TotalPromptToks int            `json:"total_prompt_tokens"`
	TotalOutputToks int            `json:"total_output_tokens"`
	TotalLatencyMs  int64          `json:"total_latency_ms"`
}

// Pipeline is the contract both Baseline and ACGC implement. It answers a
// single probe question, given the conversation history that preceded it.
type Pipeline interface {
	Kind() PipelineKind
	Answer(ctx context.Context, history []datasets.Turn, probe datasets.Probe) (*ProbeResult, error)
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
