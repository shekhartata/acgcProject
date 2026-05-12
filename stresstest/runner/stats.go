package runner

import (
	"encoding/json"
	"os"
	"time"
)

type TurnStat struct {
	TurnNumber     int     `json:"turn_number"`
	Role           string  `json:"role"`
	RawTokens      int     `json:"raw_tokens"`
	CompiledTokens int     `json:"compiled_tokens"`
	TokensSaved    int     `json:"tokens_saved"`
	ReductionPct   float64 `json:"reduction_pct"`
	ActiveNodes    int     `json:"active_nodes"`
	TotalNodes     int     `json:"total_nodes"`
	ArchivedNodes  int     `json:"archived_nodes"`
	CompressedNodes int    `json:"compressed_nodes"`
	GCTriggered    bool    `json:"gc_triggered"`
	GCTokensFreed  int     `json:"gc_tokens_freed"`
	GCNodesSwept   int     `json:"gc_nodes_swept"`
}

type SessionResult struct {
	Name             string     `json:"name"`
	TotalTurns       int        `json:"total_turns"`
	TotalRawTokens   int        `json:"total_raw_tokens"`
	TotalCompiled    int        `json:"total_compiled_tokens"`
	TotalSaved       int        `json:"total_tokens_saved"`
	AvgReductionPct  float64    `json:"avg_reduction_pct"`
	PeakRawTokens    int        `json:"peak_raw_tokens"`
	PeakCompiledTokens int      `json:"peak_compiled_tokens"`
	GCRuns           int        `json:"gc_runs"`
	TotalGCFreed     int        `json:"total_gc_freed"`
	FinalActiveNodes int        `json:"final_active_nodes"`
	FinalTotalNodes  int        `json:"final_total_nodes"`
	CoherencyScore   float64    `json:"coherency_score"`
	Duration         time.Duration `json:"duration_ns"`
	TurnStats        []TurnStat `json:"turn_stats"`
}

type SuiteResult struct {
	Sessions         []SessionResult `json:"sessions"`
	TotalSessions    int             `json:"total_sessions"`
	OverallRawTokens int             `json:"overall_raw_tokens"`
	OverallCompiled  int             `json:"overall_compiled_tokens"`
	OverallSaved     int             `json:"overall_saved_tokens"`
	OverallReduction float64         `json:"overall_reduction_pct"`
	AvgCoherency     float64         `json:"avg_coherency_score"`
	TotalDuration    time.Duration   `json:"total_duration_ns"`
}

func (sr *SuiteResult) Compute() {
	sr.TotalSessions = len(sr.Sessions)
	for _, s := range sr.Sessions {
		sr.OverallRawTokens += s.TotalRawTokens
		sr.OverallCompiled += s.TotalCompiled
		sr.OverallSaved += s.TotalSaved
		sr.AvgCoherency += s.CoherencyScore
		sr.TotalDuration += s.Duration
	}
	if sr.OverallRawTokens > 0 {
		sr.OverallReduction = float64(sr.OverallSaved) / float64(sr.OverallRawTokens) * 100
	}
	if sr.TotalSessions > 0 {
		sr.AvgCoherency /= float64(sr.TotalSessions)
	}
}

func (sr *SuiteResult) ExportJSON(path string) error {
	data, err := json.MarshalIndent(sr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
