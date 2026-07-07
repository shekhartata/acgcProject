package external

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shekhartata/acgcProject/eval/datasets"
)

func mustLoad(t *testing.T, name, path string, opts Options) []datasets.Scenario {
	t.Helper()
	src, ok := Lookup(name)
	if !ok {
		t.Fatalf("adapter %q not registered", name)
	}
	scenarios, err := src.Load(path, opts)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return scenarios
}

// --- LongMemEval ---

func TestLongMemEvalLoad(t *testing.T) {
	scenarios := mustLoad(t, "longmemeval", "testdata/longmemeval_sample.json", Options{})
	if len(scenarios) != 3 {
		t.Fatalf("want 3 scenarios, got %d", len(scenarios))
	}

	s := scenarios[0]
	if s.ID != "lme_lme_fixture_1" {
		t.Errorf("scenario ID = %q", s.ID)
	}
	if s.Category != "lme_multi-session" {
		t.Errorf("category = %q", s.Category)
	}
	if len(s.Turns) != 4 {
		t.Fatalf("want 4 flattened turns, got %d", len(s.Turns))
	}
	// First turn of each session carries the session date annotation.
	if !strings.HasPrefix(s.Turns[0].Content, "(New session on 2023/05/20") {
		t.Errorf("session 1 first turn missing date annotation: %q", s.Turns[0].Content)
	}
	if !strings.HasPrefix(s.Turns[2].Content, "(New session on 2023/05/25") {
		t.Errorf("session 2 first turn missing date annotation: %q", s.Turns[2].Content)
	}
	// Non-first turns must not be annotated.
	if strings.Contains(s.Turns[1].Content, "New session") {
		t.Errorf("non-first turn unexpectedly annotated: %q", s.Turns[1].Content)
	}

	if len(s.Probes) != 1 {
		t.Fatalf("want 1 probe, got %d", len(s.Probes))
	}
	p := s.Probes[0]
	if p.MatchType != datasets.MatchJudge {
		t.Errorf("match type = %q", p.MatchType)
	}
	if p.ProbeAt != len(s.Turns) {
		t.Errorf("probe at %d, want end of transcript %d", p.ProbeAt, len(s.Turns))
	}
	if !strings.Contains(p.Question, "(Current date: 2023/05/30") {
		t.Errorf("question missing current-date annotation: %q", p.Question)
	}
	if !strings.Contains(p.JudgeRubric, "a beagle") {
		t.Errorf("rubric missing gold answer: %q", p.JudgeRubric)
	}
}

func TestLongMemEvalAbstention(t *testing.T) {
	scenarios := mustLoad(t, "longmemeval", "testdata/longmemeval_sample.json", Options{})
	abs := scenarios[1]
	if !strings.HasSuffix(abs.ID, "_abs") {
		t.Fatalf("expected abstention fixture second, got %s", abs.ID)
	}
	rubric := abs.Probes[0].JudgeRubric
	if !strings.Contains(rubric, "declines to answer") {
		t.Errorf("abstention instance should use the abstention rubric, got: %q", rubric)
	}
}

func TestLongMemEvalNumericGoldAnswer(t *testing.T) {
	scenarios := mustLoad(t, "longmemeval", "testdata/longmemeval_sample.json", Options{})
	p := scenarios[2].Probes[0]
	if !strings.Contains(p.JudgeRubric, "5") {
		t.Errorf("numeric gold answer not normalized into rubric: %q", p.JudgeRubric)
	}
}

func TestLongMemEvalTypeFilter(t *testing.T) {
	scenarios := mustLoad(t, "longmemeval", "testdata/longmemeval_sample.json",
		Options{Types: []string{"temporal-reasoning"}})
	if len(scenarios) != 1 {
		t.Fatalf("want 1 scenario after type filter, got %d", len(scenarios))
	}
	if scenarios[0].Category != "lme_temporal-reasoning" {
		t.Errorf("filtered wrong type: %s", scenarios[0].Category)
	}
}

func TestLongMemEvalSampling(t *testing.T) {
	a := mustLoad(t, "longmemeval", "testdata/longmemeval_sample.json", Options{Sample: 2, Seed: 7})
	b := mustLoad(t, "longmemeval", "testdata/longmemeval_sample.json", Options{Sample: 2, Seed: 7})
	if len(a) != 2 {
		t.Fatalf("want 2 sampled scenarios, got %d", len(a))
	}
	var idsA, idsB []string
	for _, s := range a {
		idsA = append(idsA, s.ID)
	}
	for _, s := range b {
		idsB = append(idsB, s.ID)
	}
	if !reflect.DeepEqual(idsA, idsB) {
		t.Errorf("same seed gave different samples: %v vs %v", idsA, idsB)
	}
}

// --- LoCoMo ---

func TestLoCoMoLoad(t *testing.T) {
	scenarios := mustLoad(t, "locomo", "testdata/locomo_sample.json", Options{})
	if len(scenarios) != 1 {
		t.Fatalf("want 1 scenario (one per conversation), got %d", len(scenarios))
	}
	s := scenarios[0]
	if s.ID != "locomo_conv-1" {
		t.Errorf("scenario ID = %q", s.ID)
	}
	if len(s.Turns) != 6 {
		t.Fatalf("want 6 turns, got %d", len(s.Turns))
	}

	// Speaker A (Caroline) → user, speaker B (Melanie) → assistant, and
	// every turn is prefixed with the speaker name.
	if s.Turns[0].Role != "user" || !strings.Contains(s.Turns[0].Content, "Caroline:") {
		t.Errorf("speaker A turn wrong: role=%s content=%q", s.Turns[0].Role, s.Turns[0].Content)
	}
	if s.Turns[1].Role != "assistant" || !strings.Contains(s.Turns[1].Content, "Melanie:") {
		t.Errorf("speaker B turn wrong: role=%s content=%q", s.Turns[1].Role, s.Turns[1].Content)
	}

	// Session boundaries carry the date annotation.
	if !strings.HasPrefix(s.Turns[0].Content, "(New session on 1:00 pm on 8 May, 2023)") {
		t.Errorf("session 1 date annotation missing: %q", s.Turns[0].Content)
	}
	if !strings.HasPrefix(s.Turns[3].Content, "(New session on 4:00 pm on 25 May, 2023)") {
		t.Errorf("session 2 date annotation missing: %q", s.Turns[3].Content)
	}

	// Image-only turn is replaced with its caption.
	if !strings.Contains(s.Turns[4].Content, "[shared an image: a blue ceramic vase") {
		t.Errorf("image turn not converted to caption: %q", s.Turns[4].Content)
	}

	if len(s.Probes) != 4 {
		t.Fatalf("want 4 probes, got %d", len(s.Probes))
	}
	for _, p := range s.Probes {
		if p.MatchType != datasets.MatchJudge {
			t.Errorf("probe %s: match type = %q", p.ID, p.MatchType)
		}
		if p.ProbeAt != len(s.Turns) {
			t.Errorf("probe %s at %d, want %d", p.ID, p.ProbeAt, len(s.Turns))
		}
	}
}

func TestLoCoMoAdversarialRubric(t *testing.T) {
	scenarios := mustLoad(t, "locomo", "testdata/locomo_sample.json", Options{})
	var adversarial *datasets.Probe
	for i := range scenarios[0].Probes {
		if strings.Contains(scenarios[0].Probes[i].ID, "adversarial") {
			adversarial = &scenarios[0].Probes[i]
		}
	}
	if adversarial == nil {
		t.Fatal("no adversarial probe found")
	}
	if !strings.Contains(adversarial.JudgeRubric, "declines to answer") {
		t.Errorf("adversarial probe should use abstention rubric: %q", adversarial.JudgeRubric)
	}
	if !strings.Contains(adversarial.Notes, "painting class") {
		t.Errorf("adversarial gold should fall back to adversarial_answer: %q", adversarial.Notes)
	}
}

func TestLoCoMoCategoryFilter(t *testing.T) {
	scenarios := mustLoad(t, "locomo", "testdata/locomo_sample.json",
		Options{Types: []string{"temporal"}})
	if len(scenarios) != 1 || len(scenarios[0].Probes) != 1 {
		t.Fatalf("want 1 scenario with 1 temporal probe, got %+v", scenarios)
	}
	if !strings.Contains(scenarios[0].Probes[0].ID, "temporal") {
		t.Errorf("wrong probe kept: %s", scenarios[0].Probes[0].ID)
	}
}

func TestLoCoMoSamplingDeterminism(t *testing.T) {
	a := mustLoad(t, "locomo", "testdata/locomo_sample.json", Options{Sample: 2, Seed: 99})
	b := mustLoad(t, "locomo", "testdata/locomo_sample.json", Options{Sample: 2, Seed: 99})
	if len(a[0].Probes) != 2 {
		t.Fatalf("want 2 sampled probes, got %d", len(a[0].Probes))
	}
	var idsA, idsB []string
	for _, p := range a[0].Probes {
		idsA = append(idsA, p.ID)
	}
	for _, p := range b[0].Probes {
		idsB = append(idsB, p.ID)
	}
	if !reflect.DeepEqual(idsA, idsB) {
		t.Errorf("same seed gave different probe samples: %v vs %v", idsA, idsB)
	}
}

// --- sampler ---

func TestSampleIndices(t *testing.T) {
	// n >= total returns everything in order.
	if got := sampleIndices(3, 5, 1); !reflect.DeepEqual(got, []int{0, 1, 2}) {
		t.Errorf("oversized sample: %v", got)
	}
	// n == 0 means "all".
	if got := sampleIndices(3, 0, 1); !reflect.DeepEqual(got, []int{0, 1, 2}) {
		t.Errorf("zero sample: %v", got)
	}
	// Deterministic for a fixed seed, and returned in ascending order.
	a := sampleIndices(100, 10, 42)
	b := sampleIndices(100, 10, 42)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("same seed differed: %v vs %v", a, b)
	}
	for i := 1; i < len(a); i++ {
		if a[i] <= a[i-1] {
			t.Errorf("sample not in ascending order: %v", a)
		}
	}
	// Different seeds should (overwhelmingly) give different subsets.
	c := sampleIndices(100, 10, 43)
	if reflect.DeepEqual(a, c) {
		t.Errorf("different seeds gave identical samples: %v", a)
	}
}
