package domain

import "time"

type CompiledPrompt struct {
	CompiledPromptID    string   `json:"compiled_prompt_id"`
	SessionID           string   `json:"session_id"`
	TaskID              string   `json:"task_id"`
	CurrentUserMessage  string   `json:"current_user_message"`
	ActiveGoal          string   `json:"active_goal"`
	ActiveConstraints   []string `json:"active_constraints"`
	RelevantDecisions   []string `json:"relevant_decisions"`
	RelevantToolOutputs []string `json:"relevant_tool_outputs"`
	CompressedContext   []string `json:"compressed_context"`
	OpenIssues          []string `json:"open_issues"`
	ExcludedNodeRefs    []string `json:"excluded_node_refs"`
	FinalPrompt         string   `json:"final_prompt"`
	OriginalTokenCount  int      `json:"original_token_count"`
	CompiledTokenCount  int      `json:"compiled_token_count"`
	CreatedAt           time.Time `json:"created_at"`
}

type SessionMetrics struct {
	SessionID          string    `bson:"session_id" json:"session_id"`
	TotalEvents        int       `bson:"total_events" json:"total_events"`
	TotalTurns         int       `bson:"total_turns" json:"total_turns"`
	GCRuns             int       `bson:"gc_runs" json:"gc_runs"`
	TotalTokensSaved   int       `bson:"total_tokens_saved" json:"total_tokens_saved"`
	AvgReductionPct    float64   `bson:"avg_reduction_pct" json:"avg_reduction_pct"`
	BranchesCompressed int       `bson:"branches_compressed" json:"branches_compressed"`
	RehydrationEvents  int       `bson:"rehydration_events" json:"rehydration_events"`
	AvgLatencyMs       float64   `bson:"avg_latency_ms" json:"avg_latency_ms"`
	SessionStartedAt   time.Time `bson:"session_started_at" json:"session_started_at"`
}
