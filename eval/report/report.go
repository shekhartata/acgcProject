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
	GeneratedAt    time.Time                              `json:"generated_at"`
	Model          string                                 `json:"model"`
	TokensSpent    int                                    `json:"tokens_spent"`
	Pairs          []scoring.PairResult                   `json:"pairs"`
	Aggregate      scoring.Aggregate                      `json:"aggregate"`
	BaselineByPair map[string]harness.ProbeResult         `json:"baseline_responses"`
	ACGCByPair     map[string]harness.ProbeResult         `json:"acgc_responses"`
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

	sb.WriteString("# ACGC Quality & Intelligence-Per-Token Evaluation\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s  \n", b.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Model:** `%s`  \n", b.Model))
	sb.WriteString(fmt.Sprintf("**Live tokens spent this run:** %d  \n\n", b.TokensSpent))

	sb.WriteString("## Aggregate\n\n")
	writeAggregate(&sb, b.Aggregate)

	sb.WriteString("\n## Per-probe results\n\n")
	writeTable(&sb, b.Pairs)

	sb.WriteString("\n## Side-by-side response samples\n\n")
	writeSamples(&sb, b)

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func writeAggregate(sb *strings.Builder, a scoring.Aggregate) {
	sb.WriteString(fmt.Sprintf("- **Pairs evaluated:** %d\n", a.TotalPairs))
	sb.WriteString(fmt.Sprintf("- **Avg quality (baseline):** %.2f / 5.0\n", a.AvgQualityBaseline))
	sb.WriteString(fmt.Sprintf("- **Avg quality (ACGC):** %.2f / 5.0\n", a.AvgQualityACGC))
	sb.WriteString(fmt.Sprintf("- **Avg quality delta:** %+.2f (ACGC - baseline)\n", a.AvgQualityDelta))
	sb.WriteString(fmt.Sprintf("- **Avg token reduction:** %.1f%%\n", a.AvgTokenReductionPct))
	sb.WriteString(fmt.Sprintf("- **Avg IPT (baseline):** %.2f\n", a.AvgIPTBaseline))
	sb.WriteString(fmt.Sprintf("- **Avg IPT (ACGC):** %.2f\n", a.AvgIPTACGC))
	sb.WriteString(fmt.Sprintf("- **Avg IPT delta:** %+.1f%%\n", a.AvgIPTDeltaPct))
	sb.WriteString(fmt.Sprintf("- **Quality regressions (>1.0 drop):** %d\n\n", a.RegressionCount))

	sb.WriteString("### Verdict breakdown\n\n")
	sb.WriteString(fmt.Sprintf("- `ACGC_WIN` (better IPT, no quality loss): **%d**\n", a.ACGCWins))
	sb.WriteString(fmt.Sprintf("- `ACGC_WIN_STAR` (better IPT, but quality dropped — motivates semantic search): **%d**\n", a.ACGCWinsStar))
	sb.WriteString(fmt.Sprintf("- `TIE`: **%d**\n", a.Ties))
	sb.WriteString(fmt.Sprintf("- `ACGC_LOSS`: **%d**\n", a.ACGCLosses))
	sb.WriteString(fmt.Sprintf("- `BASELINE_WIN`: **%d**\n", a.BaselineWins))
}

func writeTable(sb *strings.Builder, pairs []scoring.PairResult) {
	sb.WriteString("| Scenario / Probe | Method | Quality (B / A) | Tokens (B / A) | Token Red% | IPT (B / A) | IPT Δ% | Verdict |\n")
	sb.WriteString("|---|---|---|---|---|---|---|---|\n")

	sorted := make([]scoring.PairResult, len(pairs))
	copy(sorted, pairs)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ScenarioID != sorted[j].ScenarioID {
			return sorted[i].ScenarioID < sorted[j].ScenarioID
		}
		return sorted[i].ProbeID < sorted[j].ProbeID
	})

	for _, p := range sorted {
		sb.WriteString(fmt.Sprintf("| `%s` / `%s` | %s | %.1f / %.1f | %d / %d | %.1f%% | %.2f / %.2f | %+.1f%% | %s |\n",
			p.ScenarioID, p.ProbeID,
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
	keys := make([]string, 0, len(b.BaselineByPair))
	for k := range b.BaselineByPair {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	max := 12
	if len(keys) < max {
		max = len(keys)
	}

	for i := 0; i < max; i++ {
		k := keys[i]
		base := b.BaselineByPair[k]
		acgc := b.ACGCByPair[k]

		sb.WriteString(fmt.Sprintf("### `%s`\n\n", k))
		sb.WriteString(fmt.Sprintf("**Question:** %s\n\n", base.Question))

		sb.WriteString(fmt.Sprintf("**Baseline** (%d prompt tokens, %d ms):\n\n", base.PromptTokens, base.LatencyMs))
		sb.WriteString("> " + indent(trunc(base.Response, 1200)) + "\n\n")

		sb.WriteString(fmt.Sprintf("**ACGC** (%d prompt tokens, %d ms):\n\n", acgc.PromptTokens, acgc.LatencyMs))
		sb.WriteString("> " + indent(trunc(acgc.Response, 1200)) + "\n\n")
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
