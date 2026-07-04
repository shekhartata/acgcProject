// Package external adapts third-party long-term-memory benchmarks
// (LongMemEval, LoCoMo) into the harness's datasets.Scenario shape.
//
// Each adapter is a pure data loader: one benchmark instance (or
// conversation) becomes one Scenario, each gold question becomes a Probe at
// the end of the transcript. The runner, strategies, scoring, and report
// layers are untouched — they only ever see []datasets.Scenario.
//
// Benchmark data files are NOT vendored (size + licensing); fetch them with
// eval/datasets/external/fetch.sh.
package external

import (
	"fmt"
	"math/rand"

	"github.com/chandrashekhartata/acgc/eval/datasets"
)

// Options controls how much of a benchmark file is loaded.
type Options struct {
	// Sample caps how many instances are loaded (LongMemEval: instances;
	// LoCoMo: QA probes per conversation). 0 = all.
	Sample int
	// Seed drives deterministic subsampling so cache keys stay stable
	// across runs.
	Seed int64
	// Types optionally filters by benchmark question type / category
	// (e.g. "multi-session", "temporal-reasoning" for LongMemEval).
	Types []string
}

// Source is a benchmark adapter that loads scenarios from a data file.
type Source interface {
	// Name is the registry key ("longmemeval", "locomo").
	Name() string
	// Load parses the file at path and returns harness scenarios.
	Load(path string, opts Options) ([]datasets.Scenario, error)
}

var registry = map[string]Source{}

func register(s Source) { registry[s.Name()] = s }

// Lookup returns the adapter registered under name.
func Lookup(name string) (Source, bool) {
	s, ok := registry[name]
	return s, ok
}

// Names lists registered adapter names (for error messages).
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	return out
}

// sampleIndices returns up to n indices out of total, chosen by a seeded
// shuffle so the same (total, n, seed) always yields the same subset. The
// returned order preserves the original index order for readability.
func sampleIndices(total, n int, seed int64) []int {
	idx := make([]int, total)
	for i := range idx {
		idx[i] = i
	}
	if n <= 0 || n >= total {
		return idx
	}
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(total, func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })
	picked := idx[:n]
	// Restore chronological/file order within the picked subset.
	for i := 0; i < len(picked); i++ {
		for j := i + 1; j < len(picked); j++ {
			if picked[j] < picked[i] {
				picked[i], picked[j] = picked[j], picked[i]
			}
		}
	}
	return picked
}

// typeAllowed reports whether t passes the Options.Types filter.
func typeAllowed(t string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, f := range filter {
		if f == t {
			return true
		}
	}
	return false
}

// judgeRubric builds the standard rubric for a probe scored against a
// free-form gold answer.
func judgeRubric(goldAnswer string) string {
	return fmt.Sprintf(
		"The response is correct if and only if it conveys the same answer as this gold answer (paraphrase is fine, contradiction or omission is not): %q",
		goldAnswer)
}

// abstentionRubric is used for questions whose correct behavior is to say
// the information is not available in the conversation.
func abstentionRubric() string {
	return "The information needed to answer this question was never mentioned in the conversation. " +
		"The response is correct if and only if it states that the information is not available or it declines to answer. " +
		"Any confident substantive answer is incorrect."
}
