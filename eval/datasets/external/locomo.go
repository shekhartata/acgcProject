package external

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/shekhartata/acgcProject/eval/datasets"
)

func init() { register(loCoMo{}) }

// loCoMo adapts the LoCoMo benchmark (https://github.com/snap-research/locomo):
// 10 very long two-person conversations (up to ~35 sessions) with ~2,000 QA
// pairs across five categories.
//
// Mapping: one conversation → one Scenario carrying all (or a sampled cap of)
// its QA pairs as probes at the end of the transcript. Grouping probes per
// conversation avoids replaying the same 35-session haystack once per
// question. Speaker A maps to the "user" role and speaker B to "assistant";
// every turn is prefixed with the speaker's name so identity attribution
// survives the role mapping.
type loCoMo struct{}

func (loCoMo) Name() string { return "locomo" }

// LoCoMo QA categories, per the published dataset.
var locomoCategories = map[int]string{
	1: "multi_hop",
	2: "temporal",
	3: "open_domain",
	4: "single_hop",
	5: "adversarial",
}

type locomoSample struct {
	SampleID     string          `json:"sample_id"`
	Conversation json.RawMessage `json:"conversation"`
	QA           []locomoQA      `json:"qa"`
}

type locomoQA struct {
	Question          string `json:"question"`
	Answer            any    `json:"answer"`
	AdversarialAnswer any    `json:"adversarial_answer"`
	Category          int    `json:"category"`
}

type locomoTurn struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
	// Image turns carry a caption; the image itself is not usable here.
	BlipCaption string `json:"blip_caption"`
}

func (loCoMo) Load(path string, opts Options) ([]datasets.Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("locomo: read %s: %w", path, err)
	}
	var samples []locomoSample
	if err := json.Unmarshal(raw, &samples); err != nil {
		return nil, fmt.Errorf("locomo: parse %s: %w", path, err)
	}

	var scenarios []datasets.Scenario
	for _, s := range samples {
		sc, err := locomoScenario(s, opts)
		if err != nil {
			return nil, err
		}
		if len(sc.Probes) > 0 {
			scenarios = append(scenarios, sc)
		}
	}
	return scenarios, nil
}

func locomoScenario(s locomoSample, opts Options) (datasets.Scenario, error) {
	turns, speakerA, err := locomoTurns(s.Conversation)
	if err != nil {
		return datasets.Scenario{}, fmt.Errorf("locomo sample %s: %w", s.SampleID, err)
	}

	// Filter QA by category name, then deterministically cap per conversation.
	filtered := make([]locomoQA, 0, len(s.QA))
	for _, qa := range s.QA {
		if typeAllowed(locomoCategories[qa.Category], opts.Types) {
			filtered = append(filtered, qa)
		}
	}

	var probes []datasets.Probe
	for _, i := range sampleIndices(len(filtered), opts.Sample, opts.Seed) {
		qa := filtered[i]
		category := locomoCategories[qa.Category]
		if category == "" {
			category = fmt.Sprintf("category_%d", qa.Category)
		}

		rubric := ""
		gold := answerString(qa.Answer)
		if qa.Category == 5 {
			// Adversarial: the premise is false / never mentioned. The
			// published gold is phrased as "not mentioned in the conversation".
			rubric = abstentionRubric()
			if gold == "" {
				gold = answerString(qa.AdversarialAnswer)
			}
		} else {
			rubric = judgeRubric(gold)
		}

		probes = append(probes, datasets.Probe{
			ID:          fmt.Sprintf("q%d_%s", i+1, category),
			ProbeAt:     len(turns),
			Question:    qa.Question,
			MatchType:   datasets.MatchJudge,
			JudgeRubric: rubric,
			Notes:       fmt.Sprintf("category=%s gold=%s", category, gold),
		})
	}

	return datasets.Scenario{
		ID:          "locomo_" + s.SampleID,
		Name:        "LoCoMo conversation " + s.SampleID,
		Category:    "locomo",
		Description: fmt.Sprintf("LoCoMo conversation %s: %d turns of two-person chat (speaker A=%s mapped to user), %d judge-scored probes.", s.SampleID, len(turns), speakerA, len(probes)),
		Turns:       turns,
		Probes:      probes,
	}, nil
}

// locomoTurns flattens the conversation object, whose sessions are stored as
// dynamic keys: session_1 (turn list), session_1_date_time (string), ...
// Returns the flattened turns and speaker A's name.
func locomoTurns(conv json.RawMessage) ([]datasets.Turn, string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(conv, &fields); err != nil {
		return nil, "", fmt.Errorf("parse conversation: %w", err)
	}

	speakerA := jsonString(fields["speaker_a"])

	// Collect session numbers present (keys like "session_3").
	var sessionNums []int
	for key := range fields {
		var n int
		if _, err := fmt.Sscanf(key, "session_%d", &n); err == nil && key == fmt.Sprintf("session_%d", n) {
			sessionNums = append(sessionNums, n)
		}
	}
	sort.Ints(sessionNums)

	var turns []datasets.Turn
	for _, n := range sessionNums {
		var sessionTurns []locomoTurn
		if err := json.Unmarshal(fields[fmt.Sprintf("session_%d", n)], &sessionTurns); err != nil {
			return nil, "", fmt.Errorf("parse session_%d: %w", n, err)
		}
		date := jsonString(fields[fmt.Sprintf("session_%d_date_time", n)])

		for ti, t := range sessionTurns {
			text := t.Text
			if text == "" && t.BlipCaption != "" {
				text = "[shared an image: " + t.BlipCaption + "]"
			}
			if text == "" {
				continue
			}
			content := t.Speaker + ": " + text
			if ti == 0 && date != "" {
				content = fmt.Sprintf("(New session on %s) %s", date, content)
			}
			role := "assistant"
			if t.Speaker == speakerA {
				role = "user"
			}
			turns = append(turns, datasets.Turn{Role: role, Content: content})
		}
	}
	return turns, speakerA, nil
}

func jsonString(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return strings.Trim(string(raw), `"`)
	}
	return s
}
