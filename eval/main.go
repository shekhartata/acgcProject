package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chandrashekhartata/acgc/eval/datasets"
	"github.com/chandrashekhartata/acgc/eval/datasets/external"
	"github.com/chandrashekhartata/acgc/eval/harness"
	"github.com/chandrashekhartata/acgc/eval/report"
	"github.com/chandrashekhartata/acgc/eval/scoring"
	"github.com/chandrashekhartata/acgc/internal/config"
	"github.com/chandrashekhartata/acgc/internal/embedding"
	"github.com/chandrashekhartata/acgc/internal/tokenizer"
	"github.com/chandrashekhartata/acgc/internal/vectorindex"
)

func main() {
	var (
		scenariosFlag = flag.String("scenarios", "", "comma-separated scenario IDs to run (empty = all)")
		budgetCap     = flag.Int("budget-cap", 0, "stop after N tokens spent on live API calls (0 = no cap)")
		useJudge      = flag.Bool("judge", false, "use LLM-as-judge for open-ended probes (otherwise they're skipped)")
		cacheOnly     = flag.Bool("cache-only", false, "do not make API calls; replay cached responses only")
		cacheDir      = flag.String("cache-dir", "eval/cache", "directory holding cached responses")
		resultsDir    = flag.String("results-dir", "eval/results", "directory to write the report into")
		judgeModel    = flag.String("judge-model", "", "model for the LLM judge (defaults to main model)")
		tokenBudget   = flag.Int("acgc-budget", 6000, "ACGC token budget for the compiler")
		verbose       = flag.Bool("v", false, "verbose per-probe logging")
		semantic      = flag.Bool("semantic", false, "enable HNSW semantic scoring in the ACGC pipeline (requires embedder API key)")
		strategiesArg = flag.String("strategies", "naive_full_history,acgc",
			"comma-separated context strategies to compare (naive_full_history, sliding_window, acgc); the first is the reference")
		maxTokens = flag.Int("max-tokens", 6000,
			"max completion tokens for probe answers (reasoning models spend part of this on hidden reasoning)")
		externalArg = flag.String("external", "",
			"external benchmark sources as name=path pairs, comma-separated (e.g. \"longmemeval=eval/datasets/external/data/longmemeval_s.json,locomo=.../locomo10.json\")")
		externalSample = flag.Int("external-sample", 20,
			"max instances per external source (LongMemEval) or probes per conversation (LoCoMo); 0 = all")
		externalSeed = flag.Int64("external-seed", 42,
			"seed for deterministic external-benchmark subsampling (keeps cache keys stable)")
		externalTypes = flag.String("external-types", "",
			"comma-separated question-type filter for external sources (e.g. \"multi-session,temporal-reasoning\" or \"single_hop,adversarial\")")
	)
	flag.Parse()

	cfg := config.Load()
	if cfg.DefaultLLMAPIKey == "" && !*cacheOnly {
		log.Fatal("ACGC_LLM_API_KEY is required (or use -cache-only to replay from cache)")
	}

	strategyKinds, reference, err := parseStrategies(*strategiesArg)
	if err != nil {
		log.Fatalf("strategies: %v", err)
	}

	llmCfg := harness.LLMConfig{
		BaseURL:     cfg.DefaultLLMBaseURL,
		APIKey:      cfg.DefaultLLMAPIKey,
		Model:       cfg.DefaultLLMModel,
		Temperature: 0,
		// Configurable via -max-tokens (default 6000). GPT-5 / o1 / o3 are
		// reasoning models that consume part of MaxTokens for invisible
		// reasoning, so too small a cap yields empty responses (reasoning
		// overflow). The default leaves ample room for both reasoning and a
		// substantive answer.
		MaxTokens: *maxTokens,
	}

	acgcCfg := harness.DefaultACGCConfig()
	acgcCfg.TokenBudget = *tokenBudget

	// Optionally enable the HNSW semantic layer in the ACGC pipeline.
	// Defaults to ACGC_LLM_API_KEY for the embedder when no dedicated key.
	if *semantic && !*cacheOnly {
		embedKey := cfg.EmbedAPIKey
		if embedKey == "" {
			embedKey = cfg.DefaultLLMAPIKey
		}
		if embedKey == "" {
			log.Fatal("semantic: -semantic set but no API key available for the embedder")
		}
		acgcCfg.Embedder = embedding.NewOpenAI(embedding.Config{
			BaseURL: cfg.EmbedBaseURL,
			APIKey:  embedKey,
			Model:   cfg.EmbedModel,
			Dim:     cfg.EmbedDim,
		})
		acgcCfg.SemanticWeight = cfg.SemanticWeight
		if acgcCfg.SemanticWeight <= 0 {
			acgcCfg.SemanticWeight = 0.20
		}
		acgcCfg.TopKAtCompile = cfg.HNSWTopKAtCompile
		if acgcCfg.TopKAtCompile <= 0 {
			acgcCfg.TopKAtCompile = 12
		}
		acgcCfg.HNSWConfig = vectorindex.Config{
			Dim:      cfg.EmbedDim,
			M:        cfg.HNSWM,
			EFSearch: cfg.HNSWEFSearch,
		}
	}

	// Use a real, model-aware tokenizer for all prompt/token accounting.
	counter := tokenizer.New(llmCfg.Model)
	tokenizer.SetDefault(counter)

	cache, err := harness.NewCache(*cacheDir, llmCfg.Model)
	if err != nil {
		log.Fatalf("cache init: %v", err)
	}

	pipelines := buildPipelines(strategyKinds, llmCfg, acgcCfg, counter)

	runner := harness.NewRunner(cache, pipelines, harness.RunnerOptions{
		CacheOnly: *cacheOnly,
		BudgetCap: *budgetCap,
		Verbose:   *verbose,
	})

	allScenarios := datasets.All()
	// External runs write to prefixed files (external_<sources>_report.md)
	// so they never clobber the built-in scenario report.
	reportPrefix := ""
	if *externalArg != "" {
		// External probes are judge-scored (free-form gold answers), so a
		// live run without -judge would skip every probe. Cache-only parse
		// validation is still allowed.
		if !*useJudge && !*cacheOnly {
			log.Fatal("-external requires -judge (external benchmark probes are judge-scored)")
		}
		ext, sourceNames, err := loadExternalScenarios(*externalArg, external.Options{
			Sample: *externalSample,
			Seed:   *externalSeed,
			Types:  splitNonEmpty(*externalTypes),
		})
		if err != nil {
			log.Fatalf("external benchmarks: %v", err)
		}
		// An external run evaluates ONLY the external scenarios; the
		// built-in scenario report (report.md/results.json) is untouched.
		allScenarios = ext
		reportPrefix = "external_" + strings.Join(sourceNames, "_")
		if acgcCfg.Embedder != nil {
			reportPrefix += "_semantic"
		}
	}

	scenarios := selectScenarios(*scenariosFlag, allScenarios)
	if len(scenarios) == 0 {
		log.Fatal("no scenarios selected")
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║          ACGC Quality & Intelligence-Per-Token Evaluation                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n  Model: %s\n", llmCfg.Model)
	fmt.Printf("  Tokenizer: %s\n", counter.Name())
	fmt.Printf("  Strategies: %s (reference: %s)\n", strategyList(strategyKinds), reference)
	fmt.Printf("  Budget: %d ctx tokens | answer cap: %d tokens\n", acgcCfg.TokenBudget, llmCfg.MaxTokens)
	fmt.Printf("  Scenarios: %d\n", len(scenarios))
	fmt.Printf("  Mode: %s\n", modeLabel(*cacheOnly, *useJudge))
	if acgcCfg.Embedder != nil {
		fmt.Printf("  Semantic: enabled (%s, dim=%d, w=%.2f, topK=%d)\n",
			acgcCfg.Embedder.Model(), cfg.EmbedDim, acgcCfg.SemanticWeight, acgcCfg.TopKAtCompile)
	} else {
		fmt.Println("  Semantic: disabled (heuristic-only)")
	}
	if *budgetCap > 0 {
		fmt.Printf("  Budget cap: %d tokens\n", *budgetCap)
	}
	fmt.Println()

	ctx := context.Background()

	responses := make(map[harness.PipelineKind]map[string]harness.ProbeResult)
	for _, k := range strategyKinds {
		responses[k] = make(map[string]harness.ProbeResult)
	}
	var pairs []scoring.PairResult
	var metrics []scoring.StrategyMetric

	var judge *scoring.JudgeClient
	if *useJudge {
		model := *judgeModel
		if model == "" {
			model = llmCfg.Model
		}
		judge = scoring.NewJudgeClient(llmCfg.BaseURL, llmCfg.APIKey, model, time.Now().UnixNano())
	}

	for _, sc := range scenarios {
		fmt.Printf("  ▸ %s ... ", sc.ID)

		runs, err := runner.RunScenario(ctx, sc)
		if err != nil {
			if errors.Is(err, harness.ErrBudgetExceeded) {
				fmt.Println("budget cap hit — stopping")
				break
			}
			fmt.Printf("error: %v\n", err)
			continue
		}

		for i, probe := range sc.Probes {
			key := sc.ID + "::" + probe.ID

			// Collect each strategy's probe result.
			results := make(map[harness.PipelineKind]harness.ProbeResult, len(strategyKinds))
			for _, k := range strategyKinds {
				res := runs[k].ProbeResults[i]
				results[k] = res
				responses[k][key] = res
			}

			// Score every strategy. For open-ended (judge) probes, score the
			// reference and each candidate via the blinded pairwise judge.
			scores, ok := scoreAllStrategies(ctx, probe, strategyKinds, reference, results, judge)
			if !ok {
				continue // open-ended probe with no judge available
			}

			for _, k := range strategyKinds {
				metrics = append(metrics, scoring.NewStrategyMetric(sc.ID, probe.ID, k, results[k], scores[k]))
			}

			refResult := results[reference]
			refScore := scores[reference]
			for _, k := range strategyKinds {
				if k == reference {
					continue
				}
				pair := scoring.BuildPair(sc.ID, probe.ID, reference, k, refResult, results[k], refScore, scores[k])
				pairs = append(pairs, pair)
			}
		}

		fmt.Printf("done (%d probes)\n", len(sc.Probes))
	}

	if len(metrics) == 0 {
		fmt.Println("\n  No scored results produced. Did you forget -judge for open-ended scenarios?")
		os.Exit(1)
	}

	strategyAggs := scoring.AggregateStrategies(metrics, strategyKinds, reference)
	agg := scoring.AggregatePairs(pairs)

	bundle := report.Bundle{
		GeneratedAt:         time.Now(),
		Model:               llmCfg.Model,
		Tokenizer:           counter.Name(),
		TokensSpent:         runner.TokensSpent(),
		Strategies:          strategyKinds,
		Reference:           reference,
		StrategyAggregates:  strategyAggs,
		StrategyMetrics:     metrics,
		Pairs:               pairs,
		Aggregate:           agg,
		ResponsesByStrategy: responses,
	}

	if err := report.WriteAll(*resultsDir, reportPrefix, bundle); err != nil {
		log.Fatalf("write report: %v", err)
	}

	printSummary(strategyAggs, agg, runner.TokensSpent())
	fmt.Printf("\n  Report written to: %s/%s\n", *resultsDir, report.FileName(reportPrefix, "report.md"))
	fmt.Printf("  Raw JSON: %s/%s\n", *resultsDir, report.FileName(reportPrefix, "results.json"))
}

// scoreAllStrategies scores every strategy for a probe. Deterministic probes
// are scored independently; open-ended (judge) probes score the reference and
// each candidate through the blinded pairwise judge. Returns ok=false when the
// probe is open-ended but no judge is configured.
func scoreAllStrategies(
	ctx context.Context,
	probe datasets.Probe,
	strategyKinds []harness.PipelineKind,
	reference harness.PipelineKind,
	results map[harness.PipelineKind]harness.ProbeResult,
	judge *scoring.JudgeClient,
) (map[harness.PipelineKind]scoring.Score, bool) {
	scores := make(map[harness.PipelineKind]scoring.Score, len(strategyKinds))

	// Deterministic path: probe scoring returns >= 0.
	deterministic := scoring.ScoreProbe(probe, results[reference])
	if deterministic.Value >= 0 {
		for _, k := range strategyKinds {
			scores[k] = scoring.ScoreProbe(probe, results[k])
		}
		return scores, true
	}

	// Open-ended path.
	if judge == nil {
		return nil, false
	}
	refResult := results[reference]
	refScoreSet := false
	for _, k := range strategyKinds {
		if k == reference {
			continue
		}
		jb, jc, jerr := judge.JudgePair(ctx, probe, refResult, results[k])
		if jerr != nil {
			log.Printf("    judge error for probe %s (%s): %v", probe.ID, k, jerr)
			continue
		}
		scores[k] = jc
		if !refScoreSet {
			scores[reference] = jb
			refScoreSet = true
		}
	}
	if !refScoreSet {
		return nil, false
	}
	return scores, true
}

// parseStrategies resolves the -strategies flag into an ordered, de-duplicated
// list of pipeline kinds. The first entry is the reference strategy.
func parseStrategies(arg string) ([]harness.PipelineKind, harness.PipelineKind, error) {
	seen := make(map[harness.PipelineKind]bool)
	var kinds []harness.PipelineKind
	for _, raw := range strings.Split(arg, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		kind, ok := harness.ParseStrategyKind(name)
		if !ok {
			return nil, "", fmt.Errorf("unknown strategy %q (valid: naive_full_history, sliding_window, acgc)", name)
		}
		if seen[kind] {
			continue
		}
		seen[kind] = true
		kinds = append(kinds, kind)
	}
	if len(kinds) == 0 {
		return nil, "", fmt.Errorf("no strategies selected")
	}
	if len(kinds) < 2 {
		fmt.Println("  note: only one strategy selected — comparison table will have no candidate deltas")
	}
	return kinds, kinds[0], nil
}

// buildPipelines constructs a runnable pipeline per selected strategy.
func buildPipelines(kinds []harness.PipelineKind, llmCfg harness.LLMConfig, acgcCfg harness.ACGCConfig, counter tokenizer.TokenCounter) []harness.Pipeline {
	pipelines := make([]harness.Pipeline, 0, len(kinds))
	for _, k := range kinds {
		var strat harness.ContextStrategy
		switch k {
		case harness.PipelineNaive:
			strat = harness.NewNaiveStrategy()
		case harness.PipelineSliding:
			strat = harness.NewSlidingStrategy()
		case harness.PipelineACGC:
			strat = harness.NewACGCStrategy(acgcCfg, counter)
		default:
			continue
		}
		pipelines = append(pipelines, harness.NewStrategyPipeline(strat, llmCfg, acgcCfg, counter))
		if k == harness.PipelineACGC && acgcCfg.Embedder != nil {
			if sp, ok := pipelines[len(pipelines)-1].(*harness.StrategyPipeline); ok {
				sp.SetCacheKeySuffix("semantic")
			}
		}
	}
	return pipelines
}

func strategyList(kinds []harness.PipelineKind) string {
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = string(k)
	}
	return strings.Join(parts, ", ")
}

func selectScenarios(filter string, all []datasets.Scenario) []datasets.Scenario {
	if filter == "" {
		return all
	}
	wanted := make(map[string]bool)
	for _, id := range strings.Split(filter, ",") {
		wanted[strings.TrimSpace(id)] = true
	}
	var out []datasets.Scenario
	for _, s := range all {
		if wanted[s.ID] {
			out = append(out, s)
		}
	}
	return out
}

// loadExternalScenarios parses the -external flag ("name=path,name=path")
// and loads scenarios from each registered adapter. It also returns the
// source names in flag order, used to prefix the report files.
func loadExternalScenarios(arg string, opts external.Options) ([]datasets.Scenario, []string, error) {
	var out []datasets.Scenario
	var names []string
	for _, pair := range strings.Split(arg, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		name, path, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, nil, fmt.Errorf("bad -external entry %q (want name=path)", pair)
		}
		name = strings.TrimSpace(name)
		src, found := external.Lookup(name)
		if !found {
			return nil, nil, fmt.Errorf("unknown external source %q (available: %s)",
				name, strings.Join(external.Names(), ", "))
		}
		scenarios, err := src.Load(strings.TrimSpace(path), opts)
		if err != nil {
			return nil, nil, err
		}
		log.Printf("external: loaded %d scenarios from %s (%s)", len(scenarios), name, path)
		out = append(out, scenarios...)
		names = append(names, name)
	}
	return out, names, nil
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func modeLabel(cacheOnly, judge bool) string {
	var parts []string
	if cacheOnly {
		parts = append(parts, "cache-only (no API calls)")
	} else {
		parts = append(parts, "live")
	}
	if judge {
		parts = append(parts, "with LLM judge")
	} else {
		parts = append(parts, "probe scoring only")
	}
	return strings.Join(parts, ", ")
}

func printSummary(strategyAggs []scoring.StrategyAggregate, agg scoring.Aggregate, tokensSpent int) {
	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("  ║                            SUMMARY                                   ║")
	fmt.Println("  ╚═══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  %-20s %8s %10s %10s %8s %10s\n", "Strategy", "Quality", "PromptTok", "Latency", "IPT", "TokRed%")
	fmt.Printf("  %s\n", strings.Repeat("-", 72))
	for _, a := range strategyAggs {
		name := string(a.Strategy)
		if a.IsReference {
			name += " *"
		}
		fmt.Printf("  %-20s %8.2f %10.0f %9.0fms %8.2f %9.1f%%\n",
			name, a.AvgQuality, a.AvgPromptTokens, a.AvgLatencyMs, a.AvgIPT, a.TokenReductionPctVsRef)
	}
	fmt.Println("  (* = reference strategy; TokRed% is vs reference)")
	fmt.Println()
	fmt.Printf("  Candidate-vs-reference verdicts:\n")
	fmt.Printf("    WIN=%d  WIN*=%d  TIE=%d  LOSS=%d  REF_WIN=%d  (pairs=%d, regressions=%d)\n",
		agg.ACGCWins, agg.ACGCWinsStar, agg.Ties, agg.ACGCLosses, agg.BaselineWins,
		agg.TotalPairs, agg.RegressionCount)
	fmt.Printf("\n  Live tokens spent this run: %d\n", tokensSpent)
}
