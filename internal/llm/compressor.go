package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chandrashekhartata/acgc/internal/domain"
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

	var allDecisions, allIssues, allEventRefs []string
	for _, n := range nodes {
		allDecisions = append(allDecisions, n.Decisions...)
		allIssues = append(allIssues, n.OpenIssues...)
		allEventRefs = append(allEventRefs, n.RawEventRefs...)
	}

	return &domain.StateNode{
		NodeID:       fmt.Sprintf("compressed_%d", time.Now().UnixNano()),
		NodeType:     domain.NodeCompressedBranch,
		Status:       domain.StatusCompressed,
		Title:        fmt.Sprintf("Compressed %d nodes", len(nodes)),
		Summary:      result.Content,
		Decisions:    allDecisions,
		OpenIssues:   allIssues,
		RawEventRefs: allEventRefs,
		TokenCount:   result.PromptTokens + result.CompletionTokens,
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
Be factual and terse. Output only the compressed summary.`

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
