package demo

import (
	"fmt"
	"strings"

	"github.com/shekhartata/acgcProject/eval/datasets"
)

const systemPrompt = "You are a helpful technical assistant. Answer concisely and accurately based on the provided context. Remember architectural decisions stated earlier in the conversation."

// Scenario is the curated marketing demo script.
type Scenario struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Turns         []Turn `json:"turns"`
	Probe         Probe  `json:"probe"`
	SeedUntil     int    `json:"seed_until"`      // ingest Turns[0:SeedUntil] without live LLM
	WarmUserSteps int    `json:"warm_user_steps"` // live Next() user turns after seed
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
// Decisions + bulk filler are seeded (no live LLM); a few filler user turns run live
// so the UI shows activity, then the CockroachDB probe fires.
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

	// Seed enough large filler that newest-first naive at ~1800 tokens drops early decisions.
	// Keep this modest: each CaptureEvent is processed serially on the server (and embeds
	// add latency when ACGC_SEMANTIC_ENABLED=true), so huge seeds make Start feel hung.
	const fillerSeedTurns = 24 // 12 Q/A pairs
	const liveWarmPairs = 3    // 3 live user turns for the UI

	need := decisionEnd + fillerSeedTurns + liveWarmPairs*2
	if len(src.Turns) < need {
		panic(fmt.Sprintf("deep_history_recall_1 too short: have %d need %d", len(src.Turns), need))
	}

	turns := make([]Turn, 0, need)
	for _, t := range src.Turns[:need] {
		turns = append(turns, Turn{Role: t.Role, Content: t.Content})
	}

	seedUntil := decisionEnd + fillerSeedTurns
	warmUsers := 0
	for i := seedUntil; i < len(turns); i++ {
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
		Description:   "Four early decisions buried under seeded filler past a tight budget; naive uses a newest-first window. Probe asks for CockroachDB.",
		Turns:         turns,
		Probe:         probe,
		SeedUntil:     seedUntil,
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
		return "Both recalled — compare tokens (ACGC should use fewer)."
	case acgcHit && !naiveHit:
		return "ACGC kept the early decision inside budget; naive's newest-first window dropped it."
	case naiveHit && !acgcHit:
		return "Naive recalled but ACGC missed — check sidecar seeding (GetState node count) and that ./bin/acgc was rebuilt."
	default:
		return "Neither pane hit the needle — check model/API keys, MaxTokens, and scenario seeding."
	}
}

// FormatPreview truncates a prompt preview for the UI.
func FormatPreview(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n… (%d more chars)", len(s)-max)
}
