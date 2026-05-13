package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIProvider hits the OpenAI-compatible /v1/embeddings endpoint.
// Reuses the same HTTP/auth conventions as internal/llm/client.go.
type OpenAIProvider struct {
	baseURL    string
	apiKey     string
	model      string
	dim        int
	httpClient *http.Client
}

func NewOpenAI(cfg Config) *OpenAIProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := cfg.Model
	if model == "" {
		model = "text-embedding-3-small"
	}
	dim := cfg.Dim
	if dim <= 0 {
		dim = 1536
	}
	return &OpenAIProvider{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		model:      model,
		dim:        dim,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *OpenAIProvider) Dim() int      { return p.dim }
func (p *OpenAIProvider) Model() string { return p.model }

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	// Some providers support a "dimensions" param for Matryoshka shrinking.
	// We don't set it for v1 — stick with the model's native dimension.
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("embed: empty text")
	}

	body, err := json.Marshal(embedRequest{Model: p.model, Input: text})
	if err != nil {
		return nil, fmt.Errorf("embed: marshal: %w", err)
	}

	// One retry on transient network/5xx errors. Keep it simple — full
	// exponential backoff lives in higher layers if we ever need it.
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		vec, retryable, err := p.doRequest(ctx, body)
		if err == nil {
			return vec, nil
		}
		lastErr = err
		if !retryable {
			return nil, err
		}
	}
	return nil, lastErr
}

func (p *OpenAIProvider) doRequest(ctx context.Context, body []byte) ([]float32, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("embed: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("embed: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, fmt.Errorf("embed: read body: %w", err)
	}

	if resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("embed: api %d: %s", resp.StatusCode, string(respBody))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("embed: api %d: %s", resp.StatusCode, string(respBody))
	}

	var out embedResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, false, fmt.Errorf("embed: unmarshal: %w", err)
	}
	if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
		return nil, false, fmt.Errorf("embed: empty response")
	}
	return out.Data[0].Embedding, false, nil
}
