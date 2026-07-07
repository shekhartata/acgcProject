package scoring

import "github.com/shekhartata/acgcProject/eval/harness"

// Verdict classifies a probe pair after scoring.
type Verdict string

const (
	VerdictACGCWin     Verdict = "ACGC_WIN"      // ACGC produced equal/better quality at lower tokens
	VerdictACGCWinStar Verdict = "ACGC_WIN_STAR" // IPT wins but raw quality regressed (motivates semantic search)
	VerdictTie         Verdict = "TIE"           // ~equal on both axes
	VerdictACGCLoss    Verdict = "ACGC_LOSS"     // quality dropped AND tokens didn't fall enough to compensate
	VerdictBaselineWin Verdict = "BASELINE_WIN"  // baseline strictly better
)

// PairResult is the fully-evaluated outcome for a single probe: a candidate
// strategy compared against the reference strategy. The *Baseline fields hold
// the reference-strategy values; the *ACGC fields hold the candidate values.
type PairResult struct {
	ScenarioID        string               `json:"scenario_id"`
	ProbeID           string               `json:"probe_id"`
	Strategy          harness.PipelineKind `json:"strategy"`           // candidate strategy
	Reference         harness.PipelineKind `json:"reference_strategy"` // reference strategy
	ScoreBaseline     float64              `json:"score_baseline"`
	ScoreACGC         float64              `json:"score_acgc"`
	TokensBaseline    int                  `json:"tokens_baseline"`
	TokensACGC        int                  `json:"tokens_acgc"`
	IPTBaseline       float64              `json:"ipt_baseline"`
	IPTACGC           float64              `json:"ipt_acgc"`
	IPTDelta          float64              `json:"ipt_delta"`     // ACGC - baseline (positive = ACGC better)
	IPTDeltaPct       float64              `json:"ipt_delta_pct"` // (ACGC - baseline) / baseline * 100
	QualityDelta      float64              `json:"quality_delta"` // ACGC - baseline (positive = ACGC better)
	TokenReductionPct float64              `json:"token_reduction_pct"`
	Verdict           Verdict              `json:"verdict"`
	ScoringMethod     string               `json:"scoring_method"` // "probe" or "judge"
	DetailBaseline    string               `json:"detail_baseline"`
	DetailACGC        string               `json:"detail_acgc"`
}

// Aggregate is the summary across all probes in the eval run.
type Aggregate struct {
	TotalPairs           int     `json:"total_pairs"`
	ACGCWins             int     `json:"acgc_wins"`
	ACGCWinsStar         int     `json:"acgc_wins_star"`
	Ties                 int     `json:"ties"`
	ACGCLosses           int     `json:"acgc_losses"`
	BaselineWins         int     `json:"baseline_wins"`
	AvgQualityBaseline   float64 `json:"avg_quality_baseline"`
	AvgQualityACGC       float64 `json:"avg_quality_acgc"`
	AvgQualityDelta      float64 `json:"avg_quality_delta"`
	AvgTokenReductionPct float64 `json:"avg_token_reduction_pct"`
	AvgIPTBaseline       float64 `json:"avg_ipt_baseline"`
	AvgIPTACGC           float64 `json:"avg_ipt_acgc"`
	AvgIPTDeltaPct       float64 `json:"avg_ipt_delta_pct"`
	RegressionCount      int     `json:"regression_count"` // pairs where quality dropped by > 1.0
}

// ComputeIPT applies the intelligence-per-token formula. We use score / (tokens/1000)
// so that a 1000-token prompt scoring 5/5 has IPT=5, and a 200-token prompt
// scoring 5/5 has IPT=25 — i.e. higher is better.
func ComputeIPT(score float64, tokens int) float64 {
	if tokens <= 0 {
		return 0
	}
	return score / (float64(tokens) / 1000.0)
}

// ClassifyVerdict applies the win/loss/tie rules. Thresholds:
//   - quality delta within ±0.5 = tie quality
//   - quality drop > 1.0 = regression (but may still win on IPT)
func ClassifyVerdict(qualityBaseline, qualityACGC, iptBaseline, iptACGC float64) Verdict {
	qDelta := qualityACGC - qualityBaseline

	switch {
	case qDelta >= -0.5 && iptACGC > iptBaseline:
		return VerdictACGCWin
	case qDelta < -0.5 && iptACGC > iptBaseline:
		return VerdictACGCWinStar // saved tokens but lost quality — motivates semantic search
	case qDelta > 0.5 && iptACGC < iptBaseline:
		return VerdictBaselineWin
	case qDelta < -1.0:
		return VerdictACGCLoss
	default:
		return VerdictTie
	}
}

// BuildPair produces a PairResult comparing a candidate strategy against the
// reference strategy for a single probe.
func BuildPair(
	scenarioID, probeID string,
	reference, candidate harness.PipelineKind,
	baseline harness.ProbeResult, acgc harness.ProbeResult,
	scoreBaseline, scoreACGC Score,
) PairResult {
	iptB := ComputeIPT(scoreBaseline.Value, baseline.PromptTokens)
	iptA := ComputeIPT(scoreACGC.Value, acgc.PromptTokens)

	pr := PairResult{
		ScenarioID:     scenarioID,
		ProbeID:        probeID,
		Strategy:       candidate,
		Reference:      reference,
		ScoreBaseline:  scoreBaseline.Value,
		ScoreACGC:      scoreACGC.Value,
		TokensBaseline: baseline.PromptTokens,
		TokensACGC:     acgc.PromptTokens,
		IPTBaseline:    iptB,
		IPTACGC:        iptA,
		IPTDelta:       iptA - iptB,
		QualityDelta:   scoreACGC.Value - scoreBaseline.Value,
		ScoringMethod:  scoreBaseline.Method,
		DetailBaseline: scoreBaseline.Detail,
		DetailACGC:     scoreACGC.Detail,
	}
	if iptB > 0 {
		pr.IPTDeltaPct = (iptA - iptB) / iptB * 100
	}
	if baseline.PromptTokens > 0 {
		pr.TokenReductionPct = float64(baseline.PromptTokens-acgc.PromptTokens) / float64(baseline.PromptTokens) * 100
	}
	pr.Verdict = ClassifyVerdict(scoreBaseline.Value, scoreACGC.Value, iptB, iptA)
	return pr
}

// AggregatePairs computes the run-level summary.
func AggregatePairs(pairs []PairResult) Aggregate {
	agg := Aggregate{TotalPairs: len(pairs)}
	if len(pairs) == 0 {
		return agg
	}
	for _, p := range pairs {
		switch p.Verdict {
		case VerdictACGCWin:
			agg.ACGCWins++
		case VerdictACGCWinStar:
			agg.ACGCWinsStar++
		case VerdictTie:
			agg.Ties++
		case VerdictACGCLoss:
			agg.ACGCLosses++
		case VerdictBaselineWin:
			agg.BaselineWins++
		}
		agg.AvgQualityBaseline += p.ScoreBaseline
		agg.AvgQualityACGC += p.ScoreACGC
		agg.AvgTokenReductionPct += p.TokenReductionPct
		agg.AvgIPTBaseline += p.IPTBaseline
		agg.AvgIPTACGC += p.IPTACGC
		if p.QualityDelta < -1.0 {
			agg.RegressionCount++
		}
	}
	n := float64(len(pairs))
	agg.AvgQualityBaseline /= n
	agg.AvgQualityACGC /= n
	agg.AvgQualityDelta = agg.AvgQualityACGC - agg.AvgQualityBaseline
	agg.AvgTokenReductionPct /= n
	agg.AvgIPTBaseline /= n
	agg.AvgIPTACGC /= n
	if agg.AvgIPTBaseline > 0 {
		agg.AvgIPTDeltaPct = (agg.AvgIPTACGC - agg.AvgIPTBaseline) / agg.AvgIPTBaseline * 100
	}
	return agg
}
