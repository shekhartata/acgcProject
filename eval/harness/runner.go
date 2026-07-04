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

// Runner executes scenarios through an ordered set of strategy pipelines,
// with caching, and returns the full per-strategy results.
type Runner struct {
	cache     *Cache
	pipelines []Pipeline
	opts      RunnerOptions

	tokensSpent int
}

// NewRunner builds a runner over an ordered list of pipelines. The order is
// preserved in reporting; conventionally the reference strategy comes first.
func NewRunner(cache *Cache, pipelines []Pipeline, opts RunnerOptions) *Runner {
	return &Runner{cache: cache, pipelines: pipelines, opts: opts}
}

// Pipelines returns the pipeline kinds in run order.
func (r *Runner) Pipelines() []PipelineKind {
	kinds := make([]PipelineKind, len(r.pipelines))
	for i, p := range r.pipelines {
		kinds[i] = p.Kind()
	}
	return kinds
}

// TokensSpent reports tokens spent on live API calls (cached responses do
// not count). Used for honest cost accounting.
func (r *Runner) TokensSpent() int { return r.tokensSpent }

// RunScenario evaluates one scenario through every configured strategy. Each
// probe is answered separately by every strategy, sharing the same prior
// history. Results are keyed by strategy kind.
func (r *Runner) RunScenario(ctx context.Context, sc datasets.Scenario) (map[PipelineKind]ScenarioRun, error) {
	runs := make(map[PipelineKind]ScenarioRun, len(r.pipelines))
	for _, p := range r.pipelines {
		runs[p.Kind()] = ScenarioRun{ScenarioID: sc.ID, Pipeline: p.Kind()}
	}

	for _, probe := range sc.Probes {
		// Slice history up to ProbeAt — probes are injected at known turn indices.
		history := sc.Turns
		if probe.ProbeAt > 0 && probe.ProbeAt < len(sc.Turns) {
			history = sc.Turns[:probe.ProbeAt]
		}

		for _, p := range r.pipelines {
			res, err := r.answer(ctx, p, sc.ID, history, probe)
			if err != nil {
				return runs, fmt.Errorf("%s probe %s: %w", p.Kind(), probe.ID, err)
			}
			run := runs[p.Kind()]
			run.ProbeResults = append(run.ProbeResults, *res)
			run.TotalPromptToks += res.PromptTokens
			run.TotalOutputToks += res.OutputTokens
			run.TotalLatencyMs += res.LatencyMs
			runs[p.Kind()] = run
		}

		if r.opts.Verbose {
			r.logProbe(sc.ID, probe.ID, runs)
		}
	}
	return runs, nil
}

func (r *Runner) logProbe(scenarioID, probeID string, runs map[PipelineKind]ScenarioRun) {
	for _, p := range r.pipelines {
		run := runs[p.Kind()]
		if len(run.ProbeResults) == 0 {
			continue
		}
		last := run.ProbeResults[len(run.ProbeResults)-1]
		log.Printf("  [%s/%s] %-18s prompt=%d tok, out=%d tok, %d ms",
			scenarioID, probeID, p.Kind(), last.PromptTokens, last.OutputTokens, last.LatencyMs)
	}
}

func (r *Runner) answer(ctx context.Context, p Pipeline, scenarioID string, history []datasets.Turn, probe datasets.Probe) (*ProbeResult, error) {
	cacheSuffix := ""
	if cp, ok := p.(CachingPipeline); ok {
		cacheSuffix = cp.CacheKeySuffix()
	}

	if cached, ok := r.cache.Get(scenarioID, probe.ID, p.Kind(), cacheSuffix); ok {
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
	result.CacheKeySuffix = cacheSuffix

	r.tokensSpent += result.PromptTokens + result.OutputTokens

	if err := r.cache.Put(result); err != nil {
		log.Printf("  cache write failed: %v", err)
	}
	return result, nil
}
