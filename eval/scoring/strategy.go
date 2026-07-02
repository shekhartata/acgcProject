package scoring

import "github.com/chandrashekhartata/acgc/eval/harness"

// StrategyMetric is a single probe's scored outcome for one strategy. This is
// the raw material for the side-by-side per-strategy comparison table.
type StrategyMetric struct {
	ScenarioID   string               `json:"scenario_id"`
	ProbeID      string               `json:"probe_id"`
	Strategy     harness.PipelineKind `json:"strategy"`
	Score        float64              `json:"score"`
	PromptTokens int                  `json:"prompt_tokens"`
	OutputTokens int                  `json:"output_tokens"`
	LatencyMs    int64                `json:"latency_ms"`
	IPT          float64              `json:"ipt"`
	Method       string               `json:"method"`
	Detail       string               `json:"detail"`
}

// NewStrategyMetric builds a StrategyMetric from a probe result + its score.
func NewStrategyMetric(scenarioID, probeID string, strategy harness.PipelineKind, res harness.ProbeResult, score Score) StrategyMetric {
	return StrategyMetric{
		ScenarioID:   scenarioID,
		ProbeID:      probeID,
		Strategy:     strategy,
		Score:        score.Value,
		PromptTokens: res.PromptTokens,
		OutputTokens: res.OutputTokens,
		LatencyMs:    res.LatencyMs,
		IPT:          ComputeIPT(score.Value, res.PromptTokens),
		Method:       score.Method,
		Detail:       score.Detail,
	}
}

// StrategyAggregate summarizes one strategy across all probes and compares it
// to the reference strategy.
type StrategyAggregate struct {
	Strategy        harness.PipelineKind `json:"strategy"`
	IsReference     bool                 `json:"is_reference"`
	Probes          int                  `json:"probes"`
	AvgQuality      float64              `json:"avg_quality"`
	AvgPromptTokens float64              `json:"avg_prompt_tokens"`
	AvgOutputTokens float64              `json:"avg_output_tokens"`
	AvgLatencyMs    float64              `json:"avg_latency_ms"`
	AvgIPT          float64              `json:"avg_ipt"`

	// Deltas vs the reference strategy (0 for the reference itself).
	TokenReductionPctVsRef float64 `json:"token_reduction_pct_vs_ref"`
	QualityDeltaVsRef      float64 `json:"quality_delta_vs_ref"`
	IPTDeltaPctVsRef       float64 `json:"ipt_delta_pct_vs_ref"`
}

// AggregateStrategies computes per-strategy averages and deltas versus the
// reference strategy. The returned slice preserves the given strategy order.
func AggregateStrategies(metrics []StrategyMetric, order []harness.PipelineKind, reference harness.PipelineKind) []StrategyAggregate {
	sums := make(map[harness.PipelineKind]*StrategyAggregate, len(order))
	for _, k := range order {
		sums[k] = &StrategyAggregate{Strategy: k, IsReference: k == reference}
	}

	for _, m := range metrics {
		agg, ok := sums[m.Strategy]
		if !ok {
			agg = &StrategyAggregate{Strategy: m.Strategy, IsReference: m.Strategy == reference}
			sums[m.Strategy] = agg
		}
		agg.Probes++
		agg.AvgQuality += m.Score
		agg.AvgPromptTokens += float64(m.PromptTokens)
		agg.AvgOutputTokens += float64(m.OutputTokens)
		agg.AvgLatencyMs += float64(m.LatencyMs)
		agg.AvgIPT += m.IPT
	}

	for _, agg := range sums {
		if agg.Probes == 0 {
			continue
		}
		n := float64(agg.Probes)
		agg.AvgQuality /= n
		agg.AvgPromptTokens /= n
		agg.AvgOutputTokens /= n
		agg.AvgLatencyMs /= n
		agg.AvgIPT /= n
	}

	ref := sums[reference]
	out := make([]StrategyAggregate, 0, len(order))
	for _, k := range order {
		agg := sums[k]
		if agg == nil {
			continue
		}
		if ref != nil && !agg.IsReference {
			agg.QualityDeltaVsRef = agg.AvgQuality - ref.AvgQuality
			if ref.AvgPromptTokens > 0 {
				agg.TokenReductionPctVsRef = (ref.AvgPromptTokens - agg.AvgPromptTokens) / ref.AvgPromptTokens * 100
			}
			if ref.AvgIPT > 0 {
				agg.IPTDeltaPctVsRef = (agg.AvgIPT - ref.AvgIPT) / ref.AvgIPT * 100
			}
		}
		out = append(out, *agg)
	}
	return out
}
