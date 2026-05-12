package datasets

// MatchType controls how a probe's expected answer is compared against the
// LLM response. Open-ended probes use TypeJudge and bypass deterministic
// scoring entirely.
type MatchType string

const (
	MatchRegex       MatchType = "regex"
	MatchExact       MatchType = "exact"
	MatchContainsAll MatchType = "contains_all"
	MatchContainsAny MatchType = "contains_any"
	MatchNumeric     MatchType = "numeric"
	MatchJudge       MatchType = "judge"
)

// Turn is a single message in a conversation. Role is "user" or "assistant".
type Turn struct {
	Role    string
	Content string
}

// Probe is a question injected at a specific turn index that has a known
// correct answer. The pipeline being evaluated sees the conversation up to
// turn ProbeAt, then receives Question and must produce a response that
// satisfies the matcher.
type Probe struct {
	ID            string
	ProbeAt       int       // turn index at which to inject the question (0-based)
	Question      string    // the question to ask
	MatchType     MatchType // how to score the response
	ExpectedAny   []string  // for regex/exact/contains_any/contains_all
	NumericValue  float64   // for numeric
	NumericTolPct float64   // tolerance % for numeric match (e.g., 0.05 = ±5%)
	JudgeRubric   string    // for MatchJudge: human-readable success criteria
	Notes         string    // why this probe exists / what failure mode it tests
}

// Scenario is a full evaluation case: a seed conversation plus a set of
// probe questions that test recall/coherence at specific points.
type Scenario struct {
	ID          string
	Name        string
	Category    string // recent_recall, long_range, constraint, topic_switch, contradiction, multi_hop
	Description string
	Turns       []Turn
	Probes      []Probe
}
