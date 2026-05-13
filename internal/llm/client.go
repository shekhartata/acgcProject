package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a provider-agnostic LLM client using the OpenAI-compatible chat completions API.
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type Config struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		switch cfg.Provider {
		case "anthropic":
			baseURL = "https://api.anthropic.com/v1"
		default:
			baseURL = "https://api.openai.com/v1"
		}
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type GenerateResult struct {
	Content          string
	PromptTokens     int
	CompletionTokens int
	// FinishReason is the provider-reported termination reason, e.g. "stop",
	// "length", "content_filter". Useful when Content is empty (a reasoning
	// model that exhausted MaxTokens on hidden reasoning will report "length").
	FinishReason string
}

// isReasoningModel returns true for models that don't support temperature/top_p
// (o1, o3, gpt-5, and similar reasoning models).
func (c *Client) isReasoningModel() bool {
	m := strings.ToLower(c.model)
	for _, prefix := range []string{"o1", "o3", "gpt-5"} {
		if strings.HasPrefix(m, prefix) {
			return true
		}
	}
	return false
}

func (c *Client) Generate(ctx context.Context, messages []ChatMessage, temperature float64, maxTokens int) (*GenerateResult, error) {
	// Build request body dynamically — reasoning models (o1, o3, gpt-5)
	// don't support temperature/top_p, and newer models use
	// max_completion_tokens instead of max_tokens.
	body := map[string]any{
		"model":    c.model,
		"messages": messages,
	}

	if !c.isReasoningModel() && temperature > 0 {
		body["temperature"] = temperature
	}

	if maxTokens > 0 {
		body["max_completion_tokens"] = maxTokens
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("llm returned no choices")
	}

	return &GenerateResult{
		Content:          chatResp.Choices[0].Message.Content,
		PromptTokens:     chatResp.Usage.PromptTokens,
		CompletionTokens: chatResp.Usage.CompletionTokens,
		FinishReason:     chatResp.Choices[0].FinishReason,
	}, nil
}
