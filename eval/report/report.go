package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chandrashekhartata/acgc/eval/harness"
	"github.com/chandrashekhartata/acgc/eval/scoring"
)

// Bundle is everything the report writer needs to render the final output.
type Bundle struct {
	GeneratedAt time.Time              `json:"generated_at"`
	Model       string                 `json:"model"`
	Tokenizer   string                 `json:"tokenizer"`
	TokensSpent int                    `json:"tokens_spent"`
	Strategies  []harness.PipelineKind `json:"strategies"`
	Reference   harness.PipelineKind   `json:"reference_strategy"`

	StrategyAggregates []scoring.StrategyAggregate `json:"strategy_aggregates"`
	StrategyMetrics    []scoring.StrategyMetric    `json:"strategy_metrics"`

	// Pairs / Aggregate hold candidate-vs-reference verdicts (one candidate per
	// pair). Retained so the win/loss framing continues to work.
	Pairs     []scoring.PairResult `json:"pairs"`
	Aggregate scoring.Aggregate    `json:"aggregate"`

	// ResponsesByStrategy[strategy][scenario::probe] = probe result, for samples.
	ResponsesByStrategy map[harness.PipelineKind]map[string]harness.ProbeResult `json:"responses_by_strategy"`
}

// WriteAll writes both a JSON dump and a human-readable markdown report.
func WriteAll(dir string, b Bundle) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "results.json"), b); err != nil {
		return err
	}
	return writeMarkdown(filepath.Join(dir, "report.md"), b)
}

func writeJSON(path string, b Bundle) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeMarkdown(path string, b Bundle) error {
	var sb strings.Builder

	sb.WriteString("# ACGC Context-Strategy Evaluation\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s  \n", b.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Model:** `%s`  \n", b.Model))
	sb.WriteString(fmt.Sprintf("**Tokenizer:** `%s`  \n", b.Tokenizer))
	sb.WriteString(fmt.Sprintf("**Reference strategy:** `%s`  \n", b.Reference))
	sb.WriteString(fmt.Sprintf("**Strategies compared:** %s  \n", joinKinds(b.Strategies)))
	sb.WriteString(fmt.Sprintf("**Live tokens spent this run:** %d  \n\n", b.TokensSpent))

	sb.WriteString("## Strategy comparison (side by side)\n\n")
	writeStrategyTable(&sb, b.StrategyAggregates)

	sb.WriteString("\n## Candidate vs reference (verdicts)\n\n")
	writeAggregate(&sb, b.Aggregate, b.Reference)

	sb.WriteString("\n## Per-probe results\n\n")
	writeTable(&sb, b.Pairs)

	sb.WriteString("\n## Response samples\n\n")
	writeSamples(&sb, b)

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func joinKinds(kinds []harness.PipelineKind) string {
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = "`" + string(k) + "`"
	}
	return strings.Join(parts, ", ")
}

func writeStrategyTable(sb *strings.Builder, aggs []scoring.StrategyAggregate) {
	sb.WriteString("| Strategy | Probes | Avg Quality | Avg Prompt Tok | Avg Latency (ms) | Avg IPT | Tok Red% vs ref | Quality Δ vs ref | IPT Δ% vs ref |\n")
	sb.WriteString("|---|---|---|---|---|---|---|---|---|\n")
	for _, a := range aggs {
		name := string(a.Strategy)
		if a.IsReference {
			name += " (ref)"
		}
		sb.WriteString(fmt.Sprintf("| `%s` | %d | %.2f | %.0f | %.0f | %.2f | %.1f%% | %+.2f | %+.1f%% |\n",
			name, a.Probes, a.AvgQuality, a.AvgPromptTokens, a.AvgLatencyMs, a.AvgIPT,
			a.TokenReductionPctVsRef, a.QualityDeltaVsRef, a.IPTDeltaPctVsRef))
	}
}

func writeAggregate(sb *strings.Builder, a scoring.Aggregate, ref harness.PipelineKind) {
	sb.WriteString(fmt.Sprintf("Reference: `%s`\n\n", ref))
	sb.WriteString(fmt.Sprintf("- **Pairs evaluated:** %d\n", a.TotalPairs))
	sb.WriteString(fmt.Sprintf("- **Avg quality (reference):** %.2f / 5.0\n", a.AvgQualityBaseline))
	sb.WriteString(fmt.Sprintf("- **Avg quality (candidate):** %.2f / 5.0\n", a.AvgQualityACGC))
	sb.WriteString(fmt.Sprintf("- **Avg quality delta:** %+.2f (candidate - reference)\n", a.AvgQualityDelta))
	sb.WriteString(fmt.Sprintf("- **Avg token reduction:** %.1f%%\n", a.AvgTokenReductionPct))
	sb.WriteString(fmt.Sprintf("- **Avg IPT (reference):** %.2f\n", a.AvgIPTBaseline))
	sb.WriteString(fmt.Sprintf("- **Avg IPT (candidate):** %.2f\n", a.AvgIPTACGC))
	sb.WriteString(fmt.Sprintf("- **Avg IPT delta:** %+.1f%%\n", a.AvgIPTDeltaPct))
	sb.WriteString(fmt.Sprintf("- **Quality regressions (>1.0 drop):** %d\n\n", a.RegressionCount))

	sb.WriteString("### Verdict breakdown\n\n")
	sb.WriteString(fmt.Sprintf("- `ACGC_WIN` (better IPT, no quality loss): **%d**\n", a.ACGCWins))
	sb.WriteString(fmt.Sprintf("- `ACGC_WIN_STAR` (better IPT, but quality dropped): **%d**\n", a.ACGCWinsStar))
	sb.WriteString(fmt.Sprintf("- `TIE`: **%d**\n", a.Ties))
	sb.WriteString(fmt.Sprintf("- `ACGC_LOSS`: **%d**\n", a.ACGCLosses))
	sb.WriteString(fmt.Sprintf("- `BASELINE_WIN` (reference strictly better): **%d**\n", a.BaselineWins))
}

func writeTable(sb *strings.Builder, pairs []scoring.PairResult) {
	sb.WriteString("| Scenario / Probe | Candidate | Method | Quality (ref / cand) | Tokens (ref / cand) | Token Red% | IPT (ref / cand) | IPT Δ% | Verdict |\n")
	sb.WriteString("|---|---|---|---|---|---|---|---|---|\n")

	sorted := make([]scoring.PairResult, len(pairs))
	copy(sorted, pairs)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ScenarioID != sorted[j].ScenarioID {
			return sorted[i].ScenarioID < sorted[j].ScenarioID
		}
		if sorted[i].ProbeID != sorted[j].ProbeID {
			return sorted[i].ProbeID < sorted[j].ProbeID
		}
		return sorted[i].Strategy < sorted[j].Strategy
	})

	for _, p := range sorted {
		sb.WriteString(fmt.Sprintf("| `%s` / `%s` | `%s` | %s | %.1f / %.1f | %d / %d | %.1f%% | %.2f / %.2f | %+.1f%% | %s |\n",
			p.ScenarioID, p.ProbeID,
			p.Strategy,
			p.ScoringMethod,
			p.ScoreBaseline, p.ScoreACGC,
			p.TokensBaseline, p.TokensACGC,
			p.TokenReductionPct,
			p.IPTBaseline, p.IPTACGC,
			p.IPTDeltaPct,
			p.Verdict,
		))
	}
}

func writeSamples(sb *strings.Builder, b Bundle) {
	// Collect the set of pair keys from the reference strategy responses.
	refResponses := b.ResponsesByStrategy[b.Reference]
	keys := make([]string, 0, len(refResponses))
	for k := range refResponses {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	max := 12
	if len(keys) < max {
		max = len(keys)
	}

	for i := 0; i < max; i++ {
		k := keys[i]
		ref := refResponses[k]

		sb.WriteString(fmt.Sprintf("### `%s`\n\n", k))
		sb.WriteString(fmt.Sprintf("**Question:** %s\n\n", ref.Question))

		for _, strat := range b.Strategies {
			res, ok := b.ResponsesByStrategy[strat][k]
			if !ok {
				continue
			}
			label := string(strat)
			if strat == b.Reference {
				label += " (ref)"
			}
			sb.WriteString(fmt.Sprintf("**%s** (%d prompt tokens, %d ms):\n\n", label, res.PromptTokens, res.LatencyMs))
			sb.WriteString("> " + indent(trunc(res.Response, 1200)) + "\n\n")
		}
		sb.WriteString("---\n\n")
	}
}

func indent(s string) string {
	return strings.ReplaceAll(s, "\n", "\n> ")
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[truncated]"
}
