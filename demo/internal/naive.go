package demo

import (
	"context"
	"fmt"
	"strings"

	"github.com/shekhartata/acgcProject/internal/llm"
	"github.com/shekhartata/acgcProject/internal/tokenizer"
)

// naivePane holds growing full-history state and calls the LLM directly.
type naivePane struct {
	client       *llm.Client
	counter      tokenizer.TokenCounter
	systemPrompt string
	budget       int
	history      []llm.ChatMessage // user/assistant only (system separate)
	transcript   []ChatLine
	cumPrompt    int
}

func newNaivePane(client *llm.Client, counter tokenizer.TokenCounter, systemPrompt string, budget int) *naivePane {
	return &naivePane{
		client:       client,
		counter:      counter,
		systemPrompt: systemPrompt,
		budget:       budget,
	}
}

func (n *naivePane) seed(role, content string) {
	n.history = append(n.history, llm.ChatMessage{Role: role, Content: content})
	n.transcript = append(n.transcript, ChatLine{Role: role, Content: content})
}

func (n *naivePane) buildMessages(question string) (msgs []llm.ChatMessage, preview string, promptTokens int) {
	msgs = make([]llm.ChatMessage, 0, 3)
	if n.systemPrompt != "" {
		msgs = append(msgs, llm.ChatMessage{Role: "system", Content: n.systemPrompt})
	}

	// Chronological oldest-first fill within budget (eval naive_full_history).
	reserve := n.counter.Count(n.systemPrompt) + n.counter.Count(question)
	remaining := n.budget - reserve
	if remaining < 0 {
		remaining = 0
	}

	var body strings.Builder
	body.WriteString("## Conversation Context\n")
	used := 0
	included := 0
	for _, m := range n.history {
		cost := n.counter.Count(m.Content) + 8 // role/framing overhead approx
		if used+cost > remaining {
			continue
		}
		body.WriteString(fmt.Sprintf("- [%s] %s\n", m.Role, m.Content))
		used += cost
		included++
	}

	finalPrompt := ""
	if included > 0 {
		finalPrompt = body.String()
		msgs = append(msgs, llm.ChatMessage{Role: "user", Content: finalPrompt})
	}
	msgs = append(msgs, llm.ChatMessage{Role: "user", Content: question})

	promptTokens = n.counter.Count(n.systemPrompt) + n.counter.Count(finalPrompt) + n.counter.Count(question)
	preview = formatNaivePreview(n.systemPrompt, finalPrompt, question)
	return msgs, preview, promptTokens
}

func formatNaivePreview(system, contextBody, question string) string {
	var b strings.Builder
	if system != "" {
		b.WriteString("[system]\n")
		b.WriteString(system)
		b.WriteString("\n\n")
	}
	if contextBody != "" {
		b.WriteString("[user: context]\n")
		b.WriteString(contextBody)
		b.WriteString("\n\n")
	}
	b.WriteString("[user: current]\n")
	b.WriteString(question)
	return FormatPreview(b.String(), 4000)
}

func (n *naivePane) run(ctx context.Context, userMessage string) (assistant string, stats NaiveStats) {
	n.transcript = append(n.transcript, ChatLine{Role: "user", Content: userMessage})

	msgs, preview, estTokens := n.buildMessages(userMessage)
	stats.PromptPreview = preview
	stats.HistoryMsgs = len(n.history) + 1 // + current user (not yet in history)

	result, err := n.client.Generate(ctx, msgs, 0.3, 512)
	if err != nil {
		stats.Error = err.Error()
		stats.PromptTokens = estTokens
		n.cumPrompt += estTokens
		stats.CumulativePromptTokens = n.cumPrompt
		// Still record user turn so panes stay aligned.
		n.history = append(n.history, llm.ChatMessage{Role: "user", Content: userMessage})
		return "", stats
	}

	assistant = result.Content
	stats.PromptTokens = result.PromptTokens
	if stats.PromptTokens == 0 {
		stats.PromptTokens = estTokens
	}
	stats.CompletionTokens = result.CompletionTokens
	n.cumPrompt += stats.PromptTokens
	stats.CumulativePromptTokens = n.cumPrompt
	stats.HistoryMsgs = len(n.history) + 2

	n.history = append(n.history,
		llm.ChatMessage{Role: "user", Content: userMessage},
		llm.ChatMessage{Role: "assistant", Content: assistant},
	)
	n.transcript = append(n.transcript, ChatLine{Role: "assistant", Content: assistant})
	return assistant, stats
}
