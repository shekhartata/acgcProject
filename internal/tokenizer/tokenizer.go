// Package tokenizer provides real, model-aware token counting for prompt
// budgeting and evaluation metrics. It replaces the historical len(s)/4
// character-length approximation, which is kept only as a defensive fallback.
package tokenizer

import "github.com/chandrashekhartata/acgc/internal/llm"

// TokenCounter counts tokens for raw text and for chat message slices.
//
// Implementations must be safe for concurrent use — the compiler, session
// manager, and eval harness may all share a single counter.
type TokenCounter interface {
	// Count returns the number of tokens in text.
	Count(text string) int
	// CountMessages returns the number of tokens for a chat request built from
	// messages, including the small per-message framing overhead used by
	// OpenAI-style chat models.
	CountMessages(messages []llm.ChatMessage) int
	// Name identifies the counter (encoding name, or "approx" for the fallback).
	Name() string
}

// approxCharsPerToken is the ratio used by the defensive fallback counter.
// Roughly matches English text under BPE encodings.
const approxCharsPerToken = 4

// FallbackCounter is the len(s)/approxCharsPerToken approximation. It is used
// only when a real tokenizer cannot be constructed (unknown model, offline
// vocab load failure). It requires no external data and never fails.
type FallbackCounter struct{}

func (FallbackCounter) Count(text string) int {
	return len(text) / approxCharsPerToken
}

func (FallbackCounter) CountMessages(messages []llm.ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += len(m.Role)/approxCharsPerToken + len(m.Content)/approxCharsPerToken + perMessageOverhead
	}
	return total + replyPrimingOverhead
}

func (FallbackCounter) Name() string { return "approx" }

// perMessageOverhead / replyPrimingOverhead approximate the fixed framing
// tokens OpenAI chat models add per message and per request. These match the
// commonly documented values for cl100k_base / o200k_base chat formats.
const (
	perMessageOverhead   = 3
	replyPrimingOverhead = 3
)

// defaultCounter is the process-wide counter used by call sites that keep the
// backward-compatible constructors (for example compiler.NewCompiler(budget)).
// It is initialized lazily to a model-agnostic real tokenizer with the
// approximate counter as a fallback.
var defaultCounter TokenCounter = New("")

// Default returns the process-wide token counter.
func Default() TokenCounter { return defaultCounter }

// SetDefault overrides the process-wide counter. Intended for wiring at
// startup (cmd/acgc, eval) once the configured model is known.
func SetDefault(c TokenCounter) {
	if c != nil {
		defaultCounter = c
	}
}
