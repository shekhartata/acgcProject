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
	"github.com/chandrashekhartata/acgc/eval/harness"
	"github.com/chandrashekhartata/acgc/eval/report"
	"github.com/chandrashekhartata/acgc/eval/scoring"
	"github.com/chandrashekhartata/acgc/internal/config"
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
	)
	flag.Parse()

	cfg := config.Load()
	if cfg.DefaultLLMAPIKey == "" && !*cacheOnly {
		log.Fatal("ACGC_LLM_API_KEY is required (or use -cache-only to replay from cache)")
	}

	llmCfg := harness.LLMConfig{
		BaseURL:     cfg.DefaultLLMBaseURL,
		APIKey:      cfg.DefaultLLMAPIKey,
		Model:       cfg.DefaultLLMModel,
		Temperature: 0,
		// Bumped from 800 → 2500. GPT-5 / o1 / o3 are reasoning models that
		// consume part of MaxTokens for invisible reasoning. 800 was causing
		// empty responses (reasoning overflow). 2500 leaves room for both
		// reasoning and a substantive answer.
		MaxTokens: 2500,
	}

	acgcCfg := harness.DefaultACGCConfig()
	acgcCfg.TokenBudget = *tokenBudget

	cache, err := harness.NewCache(*cacheDir, llmCfg.Model)
	if err != nil {
		log.Fatalf("cache init: %v", err)
	}

	baseline := harness.NewBaselinePipeline(llmCfg)
	acgc := harness.NewACGCPipeline(llmCfg, acgcCfg)

	runner := harness.NewRunner(cache, baseline, acgc, harness.RunnerOptions{
		CacheOnly: *cacheOnly,
		BudgetCap: *budgetCap,
		Verbose:   *verbose,
	})

	scenarios := selectScenarios(*scenariosFlag)
	if len(scenarios) == 0 {
		log.Fatal("no scenarios selected")
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║          ACGC Quality & Intelligence-Per-Token Evaluation                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n  Model: %s\n", llmCfg.Model)
	fmt.Printf("  Scenarios: %d\n", len(scenarios))
	fmt.Printf("  Mode: %s\n", modeLabel(*cacheOnly, *useJudge))
	if *budgetCap > 0 {
		fmt.Printf("  Budget cap: %d tokens\n", *budgetCap)
	}
	fmt.Println()

	ctx := context.Background()

	allBaseline := make(map[string]harness.ProbeResult)
	allACGC := make(map[string]harness.ProbeResult)
	var pairs []scoring.PairResult

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

		baseRun, acgcRun, err := runner.RunScenario(ctx, sc)
		if err != nil {
			if errors.Is(err, harness.ErrBudgetExceeded) {
				fmt.Println("budget cap hit — stopping")
				break
			}
			fmt.Printf("error: %v\n", err)
			continue
		}

		for i, probe := range sc.Probes {
			baseResult := baseRun.ProbeResults[i]
			acgcResult := acgcRun.ProbeResults[i]
			key := sc.ID + "::" + probe.ID
			allBaseline[key] = baseResult
			allACGC[key] = acgcResult

			scoreB := scoring.ScoreProbe(probe, baseResult)
			scoreA := scoring.ScoreProbe(probe, acgcResult)

			if scoreB.Value < 0 || scoreA.Value < 0 {
				// open-ended probe — try judge if enabled
				if judge != nil {
					jb, ja, jerr := judge.JudgePair(ctx, probe, baseResult, acgcResult)
					if jerr != nil {
						log.Printf("    judge error for %s/%s: %v", sc.ID, probe.ID, jerr)
						continue
					}
					scoreB = jb
					scoreA = ja
				} else {
					// skip — no scoring method available
					continue
				}
			}

			pair := scoring.BuildPair(sc.ID, probe.ID, baseResult, acgcResult, scoreB, scoreA)
			pairs = append(pairs, pair)
		}

		fmt.Printf("done (%d probes)\n", len(sc.Probes))
	}

	if len(pairs) == 0 {
		fmt.Println("\n  No scored pairs produced. Did you forget -judge for open-ended scenarios?")
		os.Exit(1)
	}

	agg := scoring.AggregatePairs(pairs)

	bundle := report.Bundle{
		GeneratedAt:    time.Now(),
		Model:          llmCfg.Model,
		TokensSpent:    runner.TokensSpent(),
		Pairs:          pairs,
		Aggregate:      agg,
		BaselineByPair: allBaseline,
		ACGCByPair:     allACGC,
	}

	if err := report.WriteAll(*resultsDir, bundle); err != nil {
		log.Fatalf("write report: %v", err)
	}

	printSummary(agg, pairs, runner.TokensSpent())
	fmt.Printf("\n  Report written to: %s/report.md\n", *resultsDir)
	fmt.Printf("  Raw JSON: %s/results.json\n", *resultsDir)
}

func selectScenarios(filter string) []datasets.Scenario {
	all := datasets.All()
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

func printSummary(agg scoring.Aggregate, pairs []scoring.PairResult, tokensSpent int) {
	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("  ║                            SUMMARY                                   ║")
	fmt.Println("  ╚═══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Pairs evaluated:        %d\n", agg.TotalPairs)
	fmt.Printf("  Avg quality (baseline): %.2f / 5.0\n", agg.AvgQualityBaseline)
	fmt.Printf("  Avg quality (ACGC):     %.2f / 5.0\n", agg.AvgQualityACGC)
	fmt.Printf("  Quality delta:          %+.2f\n", agg.AvgQualityDelta)
	fmt.Printf("  Token reduction:        %.1f%% avg\n", agg.AvgTokenReductionPct)
	fmt.Printf("  IPT (baseline):         %.2f\n", agg.AvgIPTBaseline)
	fmt.Printf("  IPT (ACGC):             %.2f\n", agg.AvgIPTACGC)
	fmt.Printf("  IPT delta:              %+.1f%%\n", agg.AvgIPTDeltaPct)
	fmt.Println()
	fmt.Printf("  Verdicts:  WIN=%d  WIN*=%d  TIE=%d  LOSS=%d  BASELINE_WIN=%d\n",
		agg.ACGCWins, agg.ACGCWinsStar, agg.Ties, agg.ACGCLosses, agg.BaselineWins)
	fmt.Printf("  Regressions (>1.0 quality drop): %d\n", agg.RegressionCount)
	fmt.Printf("\n  Live tokens spent this run: %d\n", tokensSpent)
}
