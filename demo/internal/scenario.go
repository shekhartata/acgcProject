package demo

import (
	"fmt"
	"strings"

	"github.com/shekhartata/acgcProject/eval/datasets"
)

const systemPrompt = "You are a helpful technical assistant. Answer concisely and accurately based on the provided context. Remember architectural decisions stated earlier in the conversation."

// Scenario is the curated marketing demo script.
type Scenario struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Turns        []Turn   `json:"turns"`
	Probe        Probe    `json:"probe"`
	WarmUserSteps int     `json:"warm_user_steps"`
}

// Turn is a scripted message.
type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Probe is the recall question asked after warm turns.
type Probe struct {
	Question    string   `json:"question"`
	ExpectedAny []string `json:"expected_any"`
}

// LoadScenario returns the marketing demo slice derived from deep_history_recall_1.
// It keeps the four early decisions, a handful of large filler pairs, then probe p1.
func LoadScenario() Scenario {
	src := datasets.ByID("deep_history_recall_1")
	if src == nil {
		panic("deep_history_recall_1 missing from eval datasets")
	}

	// Kickoff + 4 decision Q/A pairs = first 10 turns (indices 0..9).
	const decisionEnd = 10
	if len(src.Turns) < decisionEnd {
		panic("deep_history_recall_1 too short")
	}

	turns := make([]Turn, 0, 24)
	for _, t := range src.Turns[:decisionEnd] {
		turns = append(turns, Turn{Role: t.Role, Content: t.Content})
	}

	// Append 5 large filler Q/A pairs (~10 turns) so naive history grows past a tight budget.
	fillerCount := 0
	for i := decisionEnd; i < len(src.Turns) && fillerCount < 10; i++ {
		turns = append(turns, Turn{Role: src.Turns[i].Role, Content: src.Turns[i].Content})
		fillerCount++
	}

	warmUsers := 0
	for i := decisionEnd; i < len(turns); i++ {
		if turns[i].Role == "user" {
			warmUsers++
		}
	}

	probe := Probe{
		Question:    "Way back at kickoff — which primary datastore did we commit to for this platform?",
		ExpectedAny: []string{"CockroachDB", "Cockroach"},
	}
	if len(src.Probes) > 0 {
		probe.Question = src.Probes[0].Question
		if len(src.Probes[0].ExpectedAny) > 0 {
			probe.ExpectedAny = append([]string{}, src.Probes[0].ExpectedAny...)
		}
	}

	return Scenario{
		ID:            "demo_deep_history_slice",
		Name:          "Deep-history recall (demo slice)",
		Description:   "Four early decisions, then filler turns that stress a shared token budget. Probe asks for CockroachDB.",
		Turns:         turns,
		Probe:         probe,
		WarmUserSteps: warmUsers,
	}
}

// HitNeedle reports whether answer contains any expected substring (case-insensitive).
func HitNeedle(answer string, expectedAny []string) bool {
	lower := strings.ToLower(answer)
	for _, e := range expectedAny {
		if e != "" && strings.Contains(lower, strings.ToLower(e)) {
			return true
		}
	}
	return false
}

// Takeaway returns the footer string after a probe.
func Takeaway(naiveHit, acgcHit bool) string {
	switch {
	case naiveHit && acgcHit:
		return "Both recalled — compare tokens."
	case acgcHit && !naiveHit:
		return "ACGC kept the early decision inside budget."
	case naiveHit && !acgcHit:
		return "Investigate — demo scenario/budget may need tuning."
	default:
		return "Neither pane hit the needle — check model/API and scenario seeding."
	}
}

// FormatPreview truncates a prompt preview for the UI.
func FormatPreview(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n… (%d more chars)", len(s)-max)
}
