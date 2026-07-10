package demo

// ChatLine is one message shown in a pane transcript.
type ChatLine struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NaiveStats are per-turn metrics for the full-history pane.
type NaiveStats struct {
	PromptTokens           int    `json:"prompt_tokens"`
	CumulativePromptTokens int    `json:"cumulative_prompt_tokens"`
	CompletionTokens       int    `json:"completion_tokens"`
	HistoryMsgs            int    `json:"history_msgs"`
	PromptPreview          string `json:"prompt_preview"`
	Error                  string `json:"error,omitempty"`
}

// ACGCStats are per-turn metrics from the sidecar Run response.
type ACGCStats struct {
	PromptTokens     int     `json:"prompt_tokens"`
	OriginalTokens   int     `json:"original_tokens"`
	ReductionPct     float32 `json:"reduction_pct"`
	ActiveNodes      int     `json:"active_nodes"`
	CompressedNodes  int     `json:"compressed_nodes"`
	ArchivedNodes    int     `json:"archived_nodes"`
	GCTriggered      bool    `json:"gc_triggered"`
	GCReason         string  `json:"gc_reason,omitempty"`
	TokenBudget      int     `json:"token_budget"`
	PromptPreview    string  `json:"prompt_preview"`
	Error            string  `json:"error,omitempty"`
}

// StartRequest configures a new demo session.
type StartRequest struct {
	TokenBudget int    `json:"token_budget"`
	ACGCAddr    string `json:"acgc_addr"`
}

// StartResponse is returned by POST /api/demo/start.
type StartResponse struct {
	SessionID      string     `json:"session_id"`
	TurnsPlanned   int        `json:"turns_planned"`
	WarmUserSteps  int        `json:"warm_user_steps"`
	Budget         int        `json:"budget"`
	Model          string     `json:"model"`
	Subtitle       string     `json:"subtitle"`
	ScenarioID     string     `json:"scenario_id"`
	ScenarioName   string     `json:"scenario_name"`
	NaiveTranscript []ChatLine `json:"naive_transcript"`
	ACGCTranscript  []ChatLine `json:"acgc_transcript"`
}

// NextRequest advances one scripted user step.
type NextRequest struct {
	SessionID string `json:"session_id"`
}

// NextResponse is returned by POST /api/demo/next.
type NextResponse struct {
	TurnIndex     int        `json:"turn_index"`
	UserMessage   string     `json:"user_message"`
	Done          bool       `json:"done"`
	WarmRemaining int        `json:"warm_remaining"`
	Naive         PaneTurn   `json:"naive"`
	ACGC          PaneTurn   `json:"acgc"`
}

// PaneTurn is one pane's result for a demo step.
type PaneTurn struct {
	Assistant string     `json:"assistant"`
	Transcript []ChatLine `json:"transcript"`
	NaiveStats *NaiveStats `json:"naive_stats,omitempty"`
	ACGCStats  *ACGCStats  `json:"acgc_stats,omitempty"`
}

// ProbeRequest runs the recall probe on both panes.
type ProbeRequest struct {
	SessionID string `json:"session_id"`
}

// ProbePane is one pane's probe outcome.
type ProbePane struct {
	Answer    string `json:"answer"`
	HitNeedle bool   `json:"hit_needle"`
	Error     string `json:"error,omitempty"`
	Stats     any    `json:"stats,omitempty"`
}

// ProbeResponse is returned by POST /api/demo/probe.
type ProbeResponse struct {
	Question string    `json:"question"`
	Naive    ProbePane `json:"naive"`
	ACGC     ProbePane `json:"acgc"`
	Takeaway string    `json:"takeaway"`
}

// ResetRequest clears a demo session.
type ResetRequest struct {
	SessionID string `json:"session_id"`
}
