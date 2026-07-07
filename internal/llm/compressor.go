package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/facts"
)

// LLMCompressor uses a cheap LLM to compress branches into concise summaries.
// It implements gc.Compressor.
type LLMCompressor struct {
	client *Client
}

func NewLLMCompressor(client *Client) *LLMCompressor {
	return &LLMCompressor{client: client}
}

func (c *LLMCompressor) Compress(ctx context.Context, nodes []*domain.StateNode) (*domain.StateNode, error) {
	prompt := buildCompressionPrompt(nodes)

	result, err := c.client.Generate(ctx, []ChatMessage{
		{Role: "system", Content: compressionSystemPrompt},
		{Role: "user", Content: prompt},
	}, 0.2, 500)
	if err != nil {
		return nil, fmt.Errorf("llm compress: %w", err)
	}

	mergedFacts, mergedDecisions := facts.MergeFromNodes(nodes, 0, 0)
	var allIssues, allEventRefs []string
	for _, n := range nodes {
		allIssues = append(allIssues, n.OpenIssues...)
		allEventRefs = append(allEventRefs, n.RawEventRefs...)
	}

	body, ents := facts.StripTrailingEntitiesLine(result.Content, 16)
	body = strings.TrimSpace(body)
	combinedFacts := facts.UnionFacts(mergedFacts, ents, 16)

	summaryOut := strings.TrimSpace(facts.VerifiedFactsPrefix(combinedFacts) + body)
	if summaryOut == "" {
		summaryOut = strings.TrimSpace(result.Content)
	}

	tok := result.PromptTokens + result.CompletionTokens
	if est := len(summaryOut) / 4; est > tok {
		tok = est
	}

	return &domain.StateNode{
		NodeID:       fmt.Sprintf("compressed_%d", time.Now().UnixNano()),
		NodeType:     domain.NodeCompressedBranch,
		Status:       domain.StatusCompressed,
		Title:        fmt.Sprintf("Compressed %d nodes", len(nodes)),
		Summary:      summaryOut,
		Facts:        combinedFacts,
		Decisions:    mergedDecisions,
		OpenIssues:   allIssues,
		RawEventRefs: allEventRefs,
		TokenCount:   tok,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

const compressionSystemPrompt = `You are a context compressor for an AI agent system.
Given a set of conversation nodes, produce a concise summary that preserves:
- Key decisions made
- Active constraints
- Unresolved issues
- Final conclusions
- Important dependencies

Remove: repetition, filler, dead-end explorations, temporary reasoning, duplicate information.
Be factual and terse.

Output format (required):
1) Several lines of compressed summary text (no preamble).
2) On the FINAL line exactly: ENTITIES: <comma-separated proper nouns and short verbatim noun phrases>`

func buildCompressionPrompt(nodes []*domain.StateNode) string {
	var b strings.Builder
	b.WriteString("Compress the following conversation nodes into a single concise summary:\n\n")
	for i, n := range nodes {
		b.WriteString(fmt.Sprintf("--- Node %d [%s] (%s) ---\n", i+1, n.NodeType, n.Status))
		if n.Title != "" {
			b.WriteString(fmt.Sprintf("Title: %s\n", n.Title))
		}
		if n.Summary != "" {
			b.WriteString(fmt.Sprintf("Content: %s\n", n.Summary))
		}
		if len(n.Facts) > 0 {
			b.WriteString(fmt.Sprintf("Facts: %s\n", strings.Join(n.Facts, "; ")))
		}
		if len(n.Decisions) > 0 {
			b.WriteString(fmt.Sprintf("Decisions: %s\n", strings.Join(n.Decisions, "; ")))
		}
		if len(n.OpenIssues) > 0 {
			b.WriteString(fmt.Sprintf("Open Issues: %s\n", strings.Join(n.OpenIssues, "; ")))
		}
		b.WriteString("\n")
	}
	return b.String()
}
