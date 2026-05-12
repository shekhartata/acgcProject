package harness

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/chandrashekhartata/acgc/eval/datasets"
)

// ErrBudgetExceeded is returned when the configured budget cap is hit.
var ErrBudgetExceeded = errors.New("token budget cap exceeded")

// RunnerOptions controls the eval execution. CacheOnly skips API calls
// entirely — anything not in the cache returns an error result. BudgetCap
// is a soft ceiling on total prompt tokens spent (0 means no cap).
type RunnerOptions struct {
	CacheOnly bool
	BudgetCap int
	Verbose   bool
}

// Runner executes scenarios through both pipelines, with caching, and
// returns the full per-pipeline results.
type Runner struct {
	cache    *Cache
	baseline Pipeline
	acgc     Pipeline
	opts     RunnerOptions

	tokensSpent int
}

func NewRunner(cache *Cache, baseline, acgc Pipeline, opts RunnerOptions) *Runner {
	return &Runner{cache: cache, baseline: baseline, acgc: acgc, opts: opts}
}

// TokensSpent reports tokens spent on live API calls (cached responses do
// not count). Used for honest cost accounting.
func (r *Runner) TokensSpent() int { return r.tokensSpent }

// RunScenario evaluates one scenario through both pipelines. Each probe is
// answered separately by both pipelines, sharing the same prior history.
func (r *Runner) RunScenario(ctx context.Context, sc datasets.Scenario) (baseline, acgc ScenarioRun, err error) {
	baseline.ScenarioID = sc.ID
	baseline.Pipeline = PipelineBaseline
	acgc.ScenarioID = sc.ID
	acgc.Pipeline = PipelineACGC

	for _, probe := range sc.Probes {
		// Slice history up to ProbeAt — probes are injected at known turn indices.
		history := sc.Turns
		if probe.ProbeAt > 0 && probe.ProbeAt < len(sc.Turns) {
			history = sc.Turns[:probe.ProbeAt]
		}

		baseResult, err := r.answer(ctx, r.baseline, sc.ID, history, probe)
		if err != nil {
			return baseline, acgc, fmt.Errorf("baseline probe %s: %w", probe.ID, err)
		}
		baseline.ProbeResults = append(baseline.ProbeResults, *baseResult)
		baseline.TotalPromptToks += baseResult.PromptTokens
		baseline.TotalOutputToks += baseResult.OutputTokens
		baseline.TotalLatencyMs += baseResult.LatencyMs

		acgcResult, err := r.answer(ctx, r.acgc, sc.ID, history, probe)
		if err != nil {
			return baseline, acgc, fmt.Errorf("acgc probe %s: %w", probe.ID, err)
		}
		acgc.ProbeResults = append(acgc.ProbeResults, *acgcResult)
		acgc.TotalPromptToks += acgcResult.PromptTokens
		acgc.TotalOutputToks += acgcResult.OutputTokens
		acgc.TotalLatencyMs += acgcResult.LatencyMs

		if r.opts.Verbose {
			log.Printf("  [%s/%s] baseline=%d tok, acgc=%d tok (saved %d, %.1f%%)",
				sc.ID, probe.ID,
				baseResult.PromptTokens, acgcResult.PromptTokens,
				baseResult.PromptTokens-acgcResult.PromptTokens,
				pct(baseResult.PromptTokens-acgcResult.PromptTokens, baseResult.PromptTokens))
		}
	}
	return baseline, acgc, nil
}

func (r *Runner) answer(ctx context.Context, p Pipeline, scenarioID string, history []datasets.Turn, probe datasets.Probe) (*ProbeResult, error) {
	if cached, ok := r.cache.Get(scenarioID, probe.ID, p.Kind()); ok {
		cached.ScenarioID = scenarioID
		return cached, nil
	}

	if r.opts.CacheOnly {
		return &ProbeResult{
			ScenarioID: scenarioID,
			ProbeID:    probe.ID,
			Pipeline:   p.Kind(),
			Question:   probe.Question,
			Error:      "cache miss (cache-only mode)",
		}, nil
	}

	if r.opts.BudgetCap > 0 && r.tokensSpent >= r.opts.BudgetCap {
		return nil, ErrBudgetExceeded
	}

	result, err := p.Answer(ctx, history, probe)
	if err != nil {
		return result, err
	}
	result.ScenarioID = scenarioID

	r.tokensSpent += result.PromptTokens + result.OutputTokens

	if err := r.cache.Put(result); err != nil {
		log.Printf("  cache write failed: %v", err)
	}
	return result, nil
}

func pct(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den) * 100
}
