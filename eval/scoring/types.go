package scoring

import "github.com/chandrashekhartata/acgc/eval/harness"

// Score is a normalized 0.0 - 5.0 quality score for a single probe response.
// 5.0 = perfect, 0.0 = wrong.
type Score struct {
	ScenarioID string               `json:"scenario_id"`
	ProbeID    string               `json:"probe_id"`
	Pipeline   harness.PipelineKind `json:"pipeline"`
	Value      float64              `json:"value"`
	Method     string               `json:"method"` // "probe" or "judge"
	Detail     string               `json:"detail"` // human-readable explanation
}

// ScoreSet groups scores by (scenario, probe) for paired comparison.
type ScoreSet struct {
	Scores []Score `json:"scores"`
}

// LookupPair finds matching baseline/acgc scores for the same probe.
func (s *ScoreSet) LookupPair(scenarioID, probeID string) (baseline, acgc *Score) {
	for i := range s.Scores {
		sc := &s.Scores[i]
		if sc.ScenarioID != scenarioID || sc.ProbeID != probeID {
			continue
		}
		switch sc.Pipeline {
		case harness.PipelineBaseline:
			baseline = sc
		case harness.PipelineACGC:
			acgc = sc
		}
	}
	return baseline, acgc
}
