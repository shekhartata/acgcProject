package domain

import "time"

type EventType string

const (
	EventUserPrompt  EventType = "user_prompt"
	EventAgentPrompt EventType = "agent_prompt"
	EventLLMResponse EventType = "llm_response"
	EventToolCall    EventType = "tool_call"
	EventToolResult  EventType = "tool_result"
	EventError       EventType = "error"
	EventRetry       EventType = "retry"
	EventGCTrigger   EventType = "gc_trigger"
	EventBranchComp  EventType = "branch_compression"
	EventSnapshot    EventType = "state_snapshot"
	EventSystem      EventType = "system"
)

type Event struct {
	EventID   string            `bson:"event_id" json:"event_id"`
	SessionID string            `bson:"session_id" json:"session_id"`
	TaskID    string            `bson:"task_id" json:"task_id"`
	EventType EventType         `bson:"event_type" json:"event_type"`
	Sequence  int64             `bson:"sequence" json:"sequence"`
	Payload   string            `bson:"payload" json:"payload"`
	TokenCount int              `bson:"token_count" json:"token_count"`
	Metadata  map[string]string `bson:"metadata" json:"metadata"`
	CreatedAt time.Time         `bson:"created_at" json:"created_at"`
}
