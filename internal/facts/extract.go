// Package facts provides deterministic extraction of verbatim needles from chat
// events into StateNode Facts/Decisions for GC protection and compiler output.
package facts

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/shekhartata/acgcProject/internal/domain"
)

const defaultMaxFacts = 16

// Known product-ish tokens we care about for eval/long-range probes (maintain manually).
var knownEntities = []string{
	"TimescaleDB",
	"JetStream",
	"InfluxDB",
	"PostgreSQL",
	"MongoDB",
	"Kafka",
	"Kubernetes",
}

var reADR = regexp.MustCompile(`(?i)\bADR[-_]?\d+\b`)

var reDecisionLine = regexp.MustCompile(`(?i)(?m)^\s*decision\s*\d+\s*[:-]\s*(.+)$`)

func appendUnique(dest []string, add []string, capN int) []string {
	if capN <= 0 {
		capN = defaultMaxFacts
	}
	seen := make(map[string]bool)
	var out []string
	for _, s := range dest {
		k := strings.TrimSpace(s)
		if k == "" || seen[strings.ToLower(k)] || len(out) >= capN {
			continue
		}
		seen[strings.ToLower(k)] = true
		out = append(out, k)
	}
	for _, s := range add {
		if len(out) >= capN {
			break
		}
		k := strings.TrimSpace(s)
		if k == "" || seen[strings.ToLower(k)] {
			continue
		}
		seen[strings.ToLower(k)] = true
		out = append(out, k)
	}
	return out
}

// ExtractFromEvent populates node.Facts and optionally node.Decisions from the raw event payload.
func ExtractFromEvent(node *domain.StateNode, event *domain.Event) {
	if node == nil || event == nil || strings.TrimSpace(event.Payload) == "" {
		return
	}

	var facts []string
	var decisions []string
	text := strings.TrimSpace(event.Payload)

	switch event.EventType {
	case domain.EventUserPrompt:
		facts = extractUserFacts(text)
	case domain.EventLLMResponse:
		decisions = extractAssistantDecisions(text)
		facts = extractLexiconRefs(text)
	default:
		facts = extractLexiconRefs(text)
	}

	node.Facts = appendUnique(node.Facts, facts, defaultMaxFacts)
	node.Decisions = appendUnique(node.Decisions, decisions, defaultMaxFacts)
}

func extractUserFacts(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "decision") && reDecisionLine.MatchString(line) {
			if m := reDecisionLine.FindStringSubmatch(line); len(m) > 1 {
				out = append(out, strings.TrimSpace(m[1]))
			}
		}
		out = append(out, extractLexiconRefs(line)...)
	}
	for _, m := range reADR.FindAllString(text, -1) {
		out = append(out, strings.TrimSpace(m))
	}
	return dedupeLimited(out, defaultMaxFacts)
}

func extractAssistantDecisions(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "noted") {
			rest := line
			colonIdx := strings.Index(rest, ":")
			if colonIdx >= 0 {
				rest = strings.TrimSpace(rest[colonIdx+1:])
			} else if len(low) >= 7 && strings.HasPrefix(low, "noted -") {
				rest = strings.TrimSpace(line[7:])
			}
			if rest != "" && !strings.EqualFold(strings.TrimSpace(rest), "noted") {
				out = append(out, rest)
			}
		}
	}
	if len(out) == 0 {
		if s := assistantFirstSentence(text); s != "" {
			out = append(out, s)
		}
	}
	return dedupeLimited(out, defaultMaxFacts)
}

func assistantFirstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			s := strings.TrimSpace(text[:i])
			if s != "" {
				return truncateRunes(s, 220)
			}
		}
	}
	for i := 0; i < len(text); i++ {
		if text[i] == '.' && (i == len(text)-1 || text[i+1] == ' ' || text[i+1] == '\n') {
			s := strings.TrimSpace(text[:i])
			if s != "" && len(s) > 30 {
				return truncateRunes(s, 220)
			}
		}
	}
	return truncateRunes(text, 220)
}

func truncateRunes(s string, maxR int) string {
	runes := []rune(s)
	if len(runes) <= maxR {
		return s
	}
	return string(runes[:maxR])
}

func extractLexiconRefs(line string) []string {
	var found []string
	for _, kw := range knownEntities {
		if strings.Contains(strings.ToLower(line), strings.ToLower(kw)) {
			found = append(found, kw)
		}
	}
	if matched, tok := standaloneToken(line, "NATS"); matched {
		found = append(found, tok)
	}
	if matched, tok := standaloneToken(line, "Hetzner"); matched {
		found = append(found, tok)
	}
	return dedupeLimited(found, defaultMaxFacts)
}

func standaloneToken(line, tok string) (bool, string) {
	lower := strings.ToLower(line)
	lt := strings.ToLower(tok)
	start := 0
	for {
		idx := strings.Index(lower[start:], lt)
		if idx < 0 {
			return false, ""
		}
		idx += start
		beforeOk := idx == 0 || !isAlphanumAscii(line[idx-1])
		end := idx + len(tok)
		afterOk := end >= len(line) || !isAlphanumAscii(line[end])
		if beforeOk && afterOk {
			return true, line[idx:end]
		}
		start = idx + 1
	}
}

func isAlphanumAscii(b byte) bool {
	r := rune(b)
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func dedupeLimited(in []string, capN int) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range in {
		k := strings.TrimSpace(s)
		if k == "" || seen[strings.ToLower(k)] {
			continue
		}
		seen[strings.ToLower(k)] = true
		out = append(out, k)
		if len(out) >= capN {
			break
		}
	}
	return out
}
