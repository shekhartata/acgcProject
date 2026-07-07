package llm

import (
	"encoding/json"
	"testing"
)

func TestChatResponse_cachedTokens(t *testing.T) {
	body := []byte(`{
		"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],
		"usage":{
			"prompt_tokens":100,
			"completion_tokens":10,
			"total_tokens":110,
			"prompt_tokens_details":{"cached_tokens":80}
		}
	}`)
	var resp chatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Usage.PromptTokens != 100 {
		t.Fatalf("prompt_tokens = %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.PromptTokensDetails.CachedTokens != 80 {
		t.Fatalf("cached_tokens = %d", resp.Usage.PromptTokensDetails.CachedTokens)
	}
}

func TestChatResponse_noCachedTokensField(t *testing.T) {
	body := []byte(`{
		"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":50,"completion_tokens":5,"total_tokens":55}
	}`)
	var resp chatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Usage.PromptTokensDetails.CachedTokens != 0 {
		t.Fatalf("cached_tokens = %d, want 0", resp.Usage.PromptTokensDetails.CachedTokens)
	}
}

func TestChatResponse_anthropicProxyShape(t *testing.T) {
	body := []byte(`{
		"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":50,"completion_tokens":5,"total_tokens":55,"input_tokens":50}
	}`)
	var resp chatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Usage.PromptTokens != 50 {
		t.Fatalf("prompt_tokens = %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.PromptTokensDetails.CachedTokens != 0 {
		t.Fatalf("cached_tokens = %d, want 0", resp.Usage.PromptTokensDetails.CachedTokens)
	}
}
