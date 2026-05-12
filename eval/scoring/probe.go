package scoring

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/chandrashekhartata/acgc/eval/datasets"
	"github.com/chandrashekhartata/acgc/eval/harness"
)

// abstainPhrases are phrases that indicate the model is declining to recall a
// fact rather than actually recalling it. They override any keyword match to
// score 0 — a response that says "I don't have that decision captured, but
// here are some options like Timescale..." should NOT be credited for recall
// just because "Timescale" appears in a generic suggestion list.
//
// All checks are case-insensitive. Keep this list conservative — it should
// only catch explicit abstentions, not legitimate hedging in a correct answer.
var abstainPhrases = []string{
	"i don't have that",
	"i don't have a ",
	"i don't have any",
	"i do not have that",
	"i do not have a ",
	"don't have that decision",
	"do not have that decision",
	"not in the context",
	"not in the provided context",
	"not in our context",
	"not in the conversation",
	"isn't specified",
	"is not specified",
	"wasn't specified",
	"was not specified",
	"not specified in",
	"not captured",
	"isn't captured",
	"is not captured",
	"not recorded",
	"no decision recorded",
	"no decision was made",
	"no decision has been made",
	"i don't see that",
	"i don't see a ",
	"i don't see any",
	"i don't recall",
	"i do not recall",
	"not mentioned",
	"wasn't mentioned",
	"was not mentioned",
	"can you confirm",
	"could you confirm",
	"did we choose",
	"did we agree",
	"if we still need to choose",
	"if we haven't decided",
	"we haven't decided",
	"haven't picked",
	"haven't agreed",
}

// quoteNormalizer maps various unicode "smart quote" codepoints back to
// plain ASCII so substring matching works against model output. GPT-4/5 and
// similar models emit U+2019 (curly apostrophe) by default, which silently
// bypasses naive substring checks.
var quoteNormalizer = strings.NewReplacer(
	"\u2018", "'", // left single quote
	"\u2019", "'", // right single quote (this is the big one)
	"\u201A", "'", // single low-9 quote
	"\u201B", "'", // single high-reversed-9 quote
	"\u201C", "\"", // left double quote
	"\u201D", "\"", // right double quote
	"\u201E", "\"", // double low-9 quote
)

// detectAbstain returns true and the matched phrase if the response looks
// like the model is declining to recall rather than answering.
func detectAbstain(response string) (bool, string) {
	low := strings.ToLower(quoteNormalizer.Replace(response))
	for _, phrase := range abstainPhrases {
		if strings.Contains(low, phrase) {
			return true, phrase
		}
	}
	return false, ""
}

// ScoreProbe evaluates a single probe result against its match rules.
// Returns a 0-5 score. Open-ended (MatchJudge) probes return -1, signaling
// the caller should use the LLM judge instead.
func ScoreProbe(probe datasets.Probe, result harness.ProbeResult) Score {
	s := Score{
		ScenarioID: result.ScenarioID,
		ProbeID:    result.ProbeID,
		Pipeline:   result.Pipeline,
		Method:     "probe",
	}

	if result.Error != "" {
		s.Value = 0
		s.Detail = "error: " + result.Error
		return s
	}

	switch probe.MatchType {
	case datasets.MatchRegex:
		s.Value, s.Detail = matchRegex(probe.ExpectedAny, result.Response)
	case datasets.MatchExact:
		s.Value, s.Detail = matchExact(probe.ExpectedAny, result.Response)
	case datasets.MatchContainsAll:
		s.Value, s.Detail = matchContainsAll(probe.ExpectedAny, result.Response)
	case datasets.MatchContainsAny:
		s.Value, s.Detail = matchContainsAny(probe.ExpectedAny, result.Response)
	case datasets.MatchNumeric:
		s.Value, s.Detail = matchNumeric(probe.NumericValue, probe.NumericTolPct, result.Response)
	case datasets.MatchJudge:
		s.Value = -1 // signals "use judge"
		s.Method = "judge"
		s.Detail = "open-ended — defer to judge"
		return s // judge handles abstain on its own
	default:
		s.Value = 0
		s.Detail = fmt.Sprintf("unknown match type: %s", probe.MatchType)
	}

	// Empty response is always a 0 — keyword matches against empty string
	// already return 0, but be explicit so the detail message is clearer.
	if strings.TrimSpace(result.Response) == "" {
		s.Value = 0
		s.Detail = "empty response"
		return s
	}

	// Abstain override: if the model declined to answer (e.g. "I don't have
	// that decision captured"), force score to 0 regardless of any keyword
	// match. This kills false positives where a recalled keyword appears in
	// a generic suggestion list inside a "I don't know" response.
	if s.Value > 0 {
		if abstain, phrase := detectAbstain(result.Response); abstain {
			s.Value = 0
			s.Detail = fmt.Sprintf("abstain override (matched %q) — was: %s", phrase, s.Detail)
		}
	}
	return s
}

// matchRegex passes any-of: full marks if any pattern matches.
func matchRegex(patterns []string, response string) (float64, string) {
	for _, pat := range patterns {
		re, err := regexp.Compile("(?i)" + pat)
		if err != nil {
			continue
		}
		if re.MatchString(response) {
			return 5.0, "regex match: " + pat
		}
	}
	return 0, "no regex pattern matched"
}

// matchExact looks for an exact substring (case-insensitive, trimmed).
func matchExact(needles []string, response string) (float64, string) {
	low := strings.ToLower(strings.TrimSpace(response))
	for _, n := range needles {
		if strings.Contains(low, strings.ToLower(strings.TrimSpace(n))) {
			return 5.0, "exact match: " + n
		}
	}
	return 0, "no exact match found"
}

// matchContainsAll requires every needle to appear (case-insensitive).
// Partial credit: score = (matched / total) * 5.
func matchContainsAll(needles []string, response string) (float64, string) {
	low := strings.ToLower(response)
	matched := 0
	missing := []string{}
	for _, n := range needles {
		if strings.Contains(low, strings.ToLower(n)) {
			matched++
		} else {
			missing = append(missing, n)
		}
	}
	if len(needles) == 0 {
		return 5.0, "no needles configured"
	}
	score := float64(matched) / float64(len(needles)) * 5.0
	if len(missing) == 0 {
		return score, fmt.Sprintf("all %d needles matched", matched)
	}
	return score, fmt.Sprintf("%d/%d needles matched; missing: %s",
		matched, len(needles), strings.Join(missing, ", "))
}

// matchContainsAny gives full marks for one match.
func matchContainsAny(needles []string, response string) (float64, string) {
	low := strings.ToLower(response)
	for _, n := range needles {
		if strings.Contains(low, strings.ToLower(n)) {
			return 5.0, "matched: " + n
		}
	}
	return 0, "none of " + strings.Join(needles, ", ") + " found in response"
}

// matchNumeric extracts the first number from the response and compares
// against expected with the configured tolerance %.
func matchNumeric(expected, tolerancePct float64, response string) (float64, string) {
	re := regexp.MustCompile(`[-+]?\d*\.?\d+`)
	match := re.FindString(response)
	if match == "" {
		return 0, "no number found in response"
	}
	got, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, "could not parse number: " + match
	}
	tol := math.Abs(expected * tolerancePct)
	if math.Abs(got-expected) <= tol {
		return 5.0, fmt.Sprintf("numeric match: %v ≈ %v", got, expected)
	}
	return 0, fmt.Sprintf("numeric mismatch: got %v, expected %v ± %.0f%%", got, expected, tolerancePct*100)
}
