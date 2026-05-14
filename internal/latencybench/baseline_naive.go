package latencybench

import (
	"context"
	"time"

	"github.com/chandrashekhartata/acgc/internal/llm"
)

// NaiveLLMLatency runs the full naive transcript (mirrors eval/harness baseline):
// one system prompt, every warm exchange as user→assistant pairs, probe as final user message.
func NaiveLLMLatency(
	ctx context.Context,
	client *llm.Client,
	fix *Fixture,
	temperature float64,
	maxTokens int,
) (llmWallMs int64, promptTok, completionTok int, finish string, err error) {
	msgs := make([]llm.ChatMessage, 0, 3+len(fix.WarmPairs)*2)
	sysPrompt := fix.System
	if sysPrompt == "" {
		sysPrompt = "You are a helpful technical assistant. Answer concisely and accurately based on the conversation context."
	}
	msgs = append(msgs, llm.ChatMessage{
		Role:    "system",
		Content: sysPrompt,
	})
	for _, pair := range fix.WarmPairs {
		msgs = append(msgs, llm.ChatMessage{Role: "user", Content: pair.User})
		msgs = append(msgs, llm.ChatMessage{Role: "assistant", Content: pair.Assistant})
	}
	msgs = append(msgs, llm.ChatMessage{Role: "user", Content: fix.Probe})

	start := time.Now()
	res, err := client.Generate(ctx, msgs, temperature, maxTokens)
	llmWallMs = time.Since(start).Milliseconds()
	if err != nil {
		return llmWallMs, 0, 0, "", err
	}
	return llmWallMs, res.PromptTokens, res.CompletionTokens, res.FinishReason, nil
}
