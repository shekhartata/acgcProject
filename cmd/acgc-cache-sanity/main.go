package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/shekhartata/acgcProject/internal/config"
	"github.com/shekhartata/acgcProject/internal/llm"
)

func main() {
	cfg := config.Load()
	c := llm.NewClient(llm.Config{
		Provider: cfg.DefaultLLMProvider,
		BaseURL:  cfg.DefaultLLMBaseURL,
		APIKey:   cfg.DefaultLLMAPIKey,
		Model:    cfg.DefaultLLMModel,
	})
	ctx := context.Background()

	line := "Infrastructure decision record: primary datastore is CockroachDB for distributed ACID. "
	pad := strings.Repeat(line, 120) // ~2000+ tokens
	msgs := []llm.ChatMessage{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: pad},
		{Role: "user", Content: "What datastore did we pick? One word."},
	}

	fmt.Println("identical_requests_sanity (same bytes x5):")
	for i := 1; i <= 5; i++ {
		r, err := c.Generate(ctx, msgs, 0, 16)
		if err != nil {
			panic(err)
		}
		fmt.Printf("  call %d: prompt=%d cached=%d\n", i, r.PromptTokens, r.CachedPromptTokens)
	}
}
