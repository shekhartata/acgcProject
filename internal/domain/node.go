package domain

import "time"

type NodeType string

const (
	NodeGoal             NodeType = "goal"
	NodeConstraint       NodeType = "constraint"
	NodeDecision         NodeType = "decision"
	NodeAttempt          NodeType = "attempt"
	NodeToolResult       NodeType = "tool_result"
	NodeIssue            NodeType = "issue"
	NodeSummary          NodeType = "summary"
	NodeCompressedBranch NodeType = "compressed_branch"
	NodeBackground       NodeType = "background"
)

type NodeStatus string

const (
	StatusActive     NodeStatus = "active"
	StatusResolved   NodeStatus = "resolved"
	StatusStale      NodeStatus = "stale"
	StatusCompressed NodeStatus = "compressed"
	StatusArchived   NodeStatus = "archived"
)

type NodeScores struct {
	Recency         float64 `bson:"recency" json:"recency"`
	DependencyWeight float64 `bson:"dependency_weight" json:"dependency_weight"`
	Redundancy      float64 `bson:"redundancy" json:"redundancy"`
	TypePriority    float64 `bson:"type_priority" json:"type_priority"`
	UnresolvedBoost float64 `bson:"unresolved_boost" json:"unresolved_boost"`
	ResolvedPenalty float64 `bson:"resolved_penalty" json:"resolved_penalty"`
	StalePenalty    float64 `bson:"stale_penalty" json:"stale_penalty"`
	SizePenalty     float64 `bson:"size_penalty" json:"size_penalty"`
	RetentionScore  float64 `bson:"retention_score" json:"retention_score"`
}

type StateNode struct {
	NodeID       string     `bson:"node_id" json:"node_id"`
	ParentID     string     `bson:"parent_id" json:"parent_id"`
	SessionID    string     `bson:"session_id" json:"session_id"`
	TaskID       string     `bson:"task_id" json:"task_id"`
	NodeType     NodeType   `bson:"node_type" json:"node_type"`
	Status       NodeStatus `bson:"status" json:"status"`
	Title        string     `bson:"title" json:"title"`
	Summary      string     `bson:"summary" json:"summary"`
	Facts        []string   `bson:"facts" json:"facts"`
	Decisions    []string   `bson:"decisions" json:"decisions"`
	OpenIssues   []string   `bson:"open_issues" json:"open_issues"`
	Dependencies []string   `bson:"dependencies" json:"dependencies"`
	RawEventRefs []string   `bson:"raw_event_refs" json:"raw_event_refs"`
	ChildIDs     []string   `bson:"child_ids" json:"child_ids"`
	TokenCount   int        `bson:"token_count" json:"token_count"`
	TurnNumber   int64      `bson:"turn_number" json:"turn_number"`
	Scores       NodeScores `bson:"scores" json:"scores"`
	CreatedAt    time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `bson:"updated_at" json:"updated_at"`
}
