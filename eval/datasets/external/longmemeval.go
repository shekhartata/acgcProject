package external

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/chandrashekhartata/acgc/eval/datasets"
)

func init() { register(longMemEval{}) }

// longMemEval adapts the LongMemEval benchmark
// (https://github.com/xiaowu0162/LongMemEval): 500 instances per file, each
// with a multi-session chat haystack and one gold question.
//
// Mapping: one instance → one Scenario with a single judge-scored probe at
// the end of the flattened haystack. Session timestamps are annotated inline
// because temporal-reasoning and knowledge-update questions depend on them.
type longMemEval struct{}

func (longMemEval) Name() string { return "longmemeval" }

// lmeInstance mirrors the fields of longmemeval_{s,m,oracle}.json.
type lmeInstance struct {
	QuestionID       string      `json:"question_id"`
	QuestionType     string      `json:"question_type"`
	Question         string      `json:"question"`
	Answer           any         `json:"answer"` // string or number in the wild
	QuestionDate     string      `json:"question_date"`
	HaystackDates    []string    `json:"haystack_dates"`
	HaystackSessions [][]lmeTurn `json:"haystack_sessions"`
}

type lmeTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (longMemEval) Load(path string, opts Options) ([]datasets.Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("longmemeval: read %s: %w", path, err)
	}
	var instances []lmeInstance
	if err := json.Unmarshal(raw, &instances); err != nil {
		return nil, fmt.Errorf("longmemeval: parse %s: %w", path, err)
	}

	// Apply the type filter before sampling so -external-sample N means
	// "N instances of the kinds I asked for".
	filtered := make([]lmeInstance, 0, len(instances))
	for _, inst := range instances {
		if typeAllowed(inst.QuestionType, opts.Types) {
			filtered = append(filtered, inst)
		}
	}

	var scenarios []datasets.Scenario
	for _, i := range sampleIndices(len(filtered), opts.Sample, opts.Seed) {
		scenarios = append(scenarios, lmeScenario(filtered[i]))
	}
	return scenarios, nil
}

func lmeScenario(inst lmeInstance) datasets.Scenario {
	var turns []datasets.Turn
	for si, session := range inst.HaystackSessions {
		date := ""
		if si < len(inst.HaystackDates) {
			date = inst.HaystackDates[si]
		}
		for ti, t := range session {
			content := t.Content
			// Annotate the first turn of each session with its timestamp so
			// temporal questions are answerable from the transcript alone.
			if ti == 0 && date != "" {
				content = fmt.Sprintf("(New session on %s) %s", date, content)
			}
			role := t.Role
			if role != "user" && role != "assistant" {
				role = "user"
			}
			turns = append(turns, datasets.Turn{Role: role, Content: content})
		}
	}

	gold := answerString(inst.Answer)
	rubric := judgeRubric(gold)
	if strings.HasSuffix(inst.QuestionID, "_abs") {
		rubric = abstentionRubric()
	}

	question := inst.Question
	if inst.QuestionDate != "" {
		question = fmt.Sprintf("(Current date: %s) %s", inst.QuestionDate, question)
	}

	return datasets.Scenario{
		ID:          "lme_" + inst.QuestionID,
		Name:        "LongMemEval " + inst.QuestionType + " " + inst.QuestionID,
		Category:    "lme_" + inst.QuestionType,
		Description: fmt.Sprintf("LongMemEval instance %s (%s): %d sessions, judge-scored against the gold answer.", inst.QuestionID, inst.QuestionType, len(inst.HaystackSessions)),
		Turns:       turns,
		Probes: []datasets.Probe{{
			ID:          "q",
			ProbeAt:     len(turns),
			Question:    question,
			MatchType:   datasets.MatchJudge,
			JudgeRubric: rubric,
			Notes:       "gold: " + gold,
		}},
	}
}

// answerString normalizes the gold answer field, which is usually a string
// but occasionally a number in the published files.
func answerString(v any) string {
	switch a := v.(type) {
	case string:
		return a
	case nil:
		return ""
	default:
		b, err := json.Marshal(a)
		if err != nil {
			return fmt.Sprintf("%v", a)
		}
		return string(b)
	}
}
