package harness

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/chandrashekhartata/acgc/internal/llm"
)

// GenerateWithRetry wraps client.Generate with retries for transient provider
// failures (HTTP 429/5xx, timeouts, connection resets). Long benchmark runs
// issue hundreds of sequential calls, and a single transient 500 should not
// void an entire scenario. Non-transient errors (4xx like context-length or
// auth) are returned immediately.
func GenerateWithRetry(ctx context.Context, client *llm.Client, messages []llm.ChatMessage, temperature float64, maxTokens int) (*llm.GenerateResult, error) {
	const attempts = 4
	backoff := 2 * time.Second

	var lastErr error
	for i := 0; i < attempts; i++ {
		result, err := client.Generate(ctx, messages, temperature, maxTokens)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isTransient(err) {
			return nil, err
		}
		if i < attempts-1 {
			log.Printf("  transient llm error (attempt %d/%d, retrying in %s): %v", i+1, attempts, backoff, err)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}
	}
	return nil, lastErr
}

func isTransient(err error) bool {
	msg := err.Error()
	for _, marker := range []string{
		"status 429", "status 500", "status 502", "status 503", "status 504",
		"timeout", "deadline exceeded", "connection reset", "EOF",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
