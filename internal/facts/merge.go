package facts

import (
	"strings"

	"github.com/shekhartata/acgcProject/internal/domain"
)

// MergeFromNodes merges all unique Facts then all unique Decisions from children (dedupe, cap).
func MergeFromNodes(nodes []*domain.StateNode, capFacts, capDec int) ([]string, []string) {
	if capFacts <= 0 {
		capFacts = defaultMaxFacts
	}
	if capDec <= 0 {
		capDec = defaultMaxFacts
	}
	var f, d []string
	for _, n := range nodes {
		if n == nil {
			continue
		}
		f = append(f, n.Facts...)
		d = append(d, n.Decisions...)
	}
	return dedupeLimited(f, capFacts), dedupeLimited(d, capDec)
}

// VerifiedFactsPrefix returns non-empty prefixed block for summaries.
func VerifiedFactsPrefix(facts []string) string {
	if len(facts) == 0 {
		return ""
	}
	return "Verified facts (verbatim): " + stringsJoinComma(facts) + "\n\n"
}

// UnionFacts merges facts from two sources with dedupe and cap.
func UnionFacts(a, b []string, capN int) []string {
	return appendUnique(a, b, capN)
}

// StripTrailingEntitiesLine removes a trailing "ENTITIES:" line from model output,
// returning the preceding body as summary text and comma-split entities (deduped/capped).
func StripTrailingEntitiesLine(content string, capEntities int) (body string, entities []string) {
	text := strings.TrimRight(content, "\n\r ")
	if text == "" {
		return "", nil
	}
	lastNL := strings.LastIndexByte(text, '\n')
	var lastLine, prefix string
	if lastNL >= 0 {
		lastLine = strings.TrimSpace(text[lastNL+1:])
		prefix = strings.TrimRight(text[:lastNL], "\n\r ")
	} else {
		lastLine = strings.TrimSpace(text)
		prefix = ""
	}
	ll := strings.TrimSpace(lastLine)
	if ll == "" {
		return strings.TrimRight(content, "\n\r "), nil
	}
	// Final line must be ENTITIES:<comma-separated> (case-insensitive keyword).
	colonIdx := strings.IndexByte(ll, ':')
	if colonIdx > 0 && strings.EqualFold(strings.TrimSpace(ll[:colonIdx]), "ENTITIES") {
		raw := strings.TrimSpace(ll[colonIdx+1:])
		body = strings.TrimSpace(prefix)
		for _, part := range strings.Split(raw, ",") {
			p := strings.TrimSpace(part)
			if p != "" {
				entities = append(entities, p)
			}
		}
		entities = dedupeLimited(entities, capEntities)
		return body, entities
	}
	return strings.TrimRight(content, "\n\r "), nil
}

func stringsJoinComma(slice []string) string {
	switch len(slice) {
	case 0:
		return ""
	case 1:
		return slice[0]
	default:
		var b strings.Builder
		for i, s := range slice {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(s)
		}
		return b.String()
	}
}
