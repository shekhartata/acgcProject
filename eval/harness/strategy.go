package harness

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shekhartata/acgcProject/eval/datasets"
	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/llm"
	"github.com/shekhartata/acgcProject/internal/tokenizer"
)

// StrategyInput is the context a ContextStrategy needs to turn a conversation
// history + probe into a final context prompt. Every strategy receives the
// same input so their outputs are directly comparable.
type StrategyInput struct {
	Ctx          context.Context
	SessionID    string
	TaskID       string
	SystemPrompt string
	// History is the conversation up to (but excluding) the probe turn.
	History []datasets.Turn
	// Nodes is History pre-converted into chronological state nodes (with real
	// token counts). Simple strategies (naive, sliding) select from this list;
	// the ACGC strategy runs its own full replay from History instead.
	Nodes []*domain.StateNode
	// Question is the probe question being asked.
	Question string
	// TokenBudget caps how much context the strategy may include.
	TokenBudget int
	// TokenCounter is the shared, model-aware token counter.
	TokenCounter tokenizer.TokenCounter
}

// StrategyOutput is the assembled context for a probe plus the token accounting
// used for side-by-side comparison.
type StrategyOutput struct {
	// FinalPrompt is the structured context body sent as its own user message.
	FinalPrompt string
	// OriginalTokenCount is the token cost of the full available context
	// (system + all history nodes + question) before any budgeting.
	OriginalTokenCount int
	// CompiledTokenCount is the token cost actually sent on the wire
	// (system + FinalPrompt + question).
	CompiledTokenCount int
	// IncludedNodes / ExcludedNodes count how many history nodes made the cut.
	IncludedNodes int
	ExcludedNodes int
}

// ContextStrategy builds the final context prompt for a probe from a shared
// StrategyInput. It is purely about context assembly — the LLM call, caching,
// and scoring are handled uniformly by the runner for every strategy.
type ContextStrategy interface {
	Name() PipelineKind
	BuildPrompt(in StrategyInput) (StrategyOutput, error)
}

// evalSystemPrompt is shared by every strategy so the only variable across
// strategies is the context body — an apples-to-apples comparison.
const evalSystemPrompt = "You are a helpful technical assistant. Answer concisely and accurately based on the provided context."

// buildHistoryNodes converts a conversation history into chronological state
// nodes with real token counts. Used by the naive and sliding strategies as
// their "available context" without running the full ACGC stack.
func buildHistoryNodes(history []datasets.Turn, counter tokenizer.TokenCounter) []*domain.StateNode {
	nodes := make([]*domain.StateNode, 0, len(history))
	for i, t := range history {
		nodeType := domain.NodeGoal
		if t.Role == "assistant" {
			nodeType = domain.NodeDecision
		}
		nodes = append(nodes, &domain.StateNode{
			NodeID:     fmt.Sprintf("turn_%d", i),
			NodeType:   nodeType,
			Status:     domain.StatusActive,
			Summary:    t.Content,
			TokenCount: counter.Count(t.Content),
			TurnNumber: int64(i),
			CreatedAt:  time.Now(),
		})
	}
	return nodes
}

// selectChronological returns nodes in original order until the budget is hit.
func selectChronological(nodes []*domain.StateNode, budget int) (included []*domain.StateNode, excluded int, used int) {
	for _, n := range nodes {
		cost := n.TokenCount
		if used+cost > budget {
			excluded++
			continue
		}
		included = append(included, n)
		used += cost
	}
	return included, excluded, used
}

// formatNodesSection renders a list of nodes as a plain context block.
func formatNodesSection(nodes []*domain.StateNode) string {
	var b strings.Builder
	b.WriteString("## Conversation Context\n")
	for _, n := range nodes {
		content := n.Summary
		if content == "" {
			content = n.Title
		}
		role := "user"
		if n.NodeType == domain.NodeDecision {
			role = "assistant"
		}
		b.WriteString(fmt.Sprintf("- [%s] %s\n", role, content))
	}
	return b.String()
}

// assembleMessages builds the wire-format chat request shared by all
// strategies: [system, user(context body), user(probe question)].
func assembleMessages(systemPrompt, finalPrompt, question string) []llm.ChatMessage {
	messages := make([]llm.ChatMessage, 0, 3)
	if systemPrompt != "" {
		messages = append(messages, llm.ChatMessage{Role: "system", Content: systemPrompt})
	}
	if finalPrompt != "" {
		messages = append(messages, llm.ChatMessage{Role: "user", Content: finalPrompt})
	}
	messages = append(messages, llm.ChatMessage{Role: "user", Content: question})
	return messages
}
