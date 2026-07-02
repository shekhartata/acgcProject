package tokenizer

import (
	"testing"

	"github.com/chandrashekhartata/acgc/internal/llm"
)

func TestFallbackCounter(t *testing.T) {
	var c FallbackCounter
	if got := c.Count("12345678"); got != 2 {
		t.Fatalf("Count(8 chars) = %d, want 2", got)
	}
	if got := c.Count(""); got != 0 {
		t.Fatalf("Count(empty) = %d, want 0", got)
	}
	if c.Name() != "approx" {
		t.Fatalf("Name() = %q, want approx", c.Name())
	}
}

func TestFallbackCountMessages(t *testing.T) {
	var c FallbackCounter
	msgs := []llm.ChatMessage{
		{Role: "system", Content: "abcd"},
		{Role: "user", Content: "efgh"},
	}
	// per message: len(role)/4 + len(content)/4 + 3, plus 3 priming.
	// system: 6/4=1 + 4/4=1 + 3 = 5 ; user: 4/4=1 + 4/4=1 + 3 = 5 ; +3 = 13
	if got := c.CountMessages(msgs); got != 13 {
		t.Fatalf("CountMessages = %d, want 13", got)
	}
}

func TestNewReturnsUsableCounter(t *testing.T) {
	for _, model := range []string{"", "gpt-4o-mini", "gpt-5", "gpt-4", "some-unknown-model"} {
		c := New(model)
		if c == nil {
			t.Fatalf("New(%q) returned nil", model)
		}
		if got := c.Count("hello world, this is a token counting test"); got <= 0 {
			t.Fatalf("New(%q).Count returned non-positive %d", model, got)
		}
		// CountMessages should be at least the sum of content tokens.
		msgs := []llm.ChatMessage{{Role: "user", Content: "hello world"}}
		if got := c.CountMessages(msgs); got <= 0 {
			t.Fatalf("New(%q).CountMessages returned non-positive %d", model, got)
		}
	}
}

func TestDefaultCounter(t *testing.T) {
	if Default() == nil {
		t.Fatal("Default() returned nil")
	}
	orig := Default()
	t.Cleanup(func() { SetDefault(orig) })

	SetDefault(FallbackCounter{})
	if Default().Name() != "approx" {
		t.Fatalf("after SetDefault, Name() = %q, want approx", Default().Name())
	}
	// nil should be ignored.
	SetDefault(nil)
	if Default().Name() != "approx" {
		t.Fatalf("SetDefault(nil) changed default counter")
	}
}
