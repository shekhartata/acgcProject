package tokenizer

import (
	"log"
	"strings"
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"
	tiktokenloader "github.com/pkoukk/tiktoken-go-loader"

	"github.com/shekhartata/acgcProject/internal/llm"
)

// offlineOnce installs the embedded (offline) BPE vocabulary loader exactly
// once so token counting never depends on network access at runtime.
var offlineOnce sync.Once

func useOfflineLoader() {
	offlineOnce.Do(func() {
		tiktoken.SetBpeLoader(tiktokenloader.NewOfflineLoader())
	})
}

// tiktokenCounter is a real BPE-based token counter backed by tiktoken-go.
type tiktokenCounter struct {
	enc      *tiktoken.Tiktoken
	encoding string
}

// New returns a TokenCounter for the given model. It resolves the correct BPE
// encoding for the model (falling back to a sensible default for unknown
// models) and, if the tokenizer cannot be built at all, returns the
// approximate FallbackCounter so callers always get a usable counter.
func New(model string) TokenCounter {
	useOfflineLoader()

	if enc, err := tiktoken.EncodingForModel(model); err == nil {
		return &tiktokenCounter{enc: enc, encoding: encodingForModel(model)}
	}

	encName := encodingForModel(model)
	enc, err := tiktoken.GetEncoding(encName)
	if err != nil {
		log.Printf("tokenizer: failed to load encoding %q for model %q (%v) — using approximate fallback", encName, model, err)
		return FallbackCounter{}
	}
	return &tiktokenCounter{enc: enc, encoding: encName}
}

// encodingForModel maps a model name to a BPE encoding. Newer OpenAI models
// (gpt-4o, gpt-5, o-series) use o200k_base; older ones use cl100k_base.
func encodingForModel(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case m == "":
		return "o200k_base"
	case strings.HasPrefix(m, "gpt-4o"),
		strings.HasPrefix(m, "gpt-5"),
		strings.HasPrefix(m, "o1"),
		strings.HasPrefix(m, "o3"),
		strings.HasPrefix(m, "o4"):
		return "o200k_base"
	default:
		return "cl100k_base"
	}
}

func (t *tiktokenCounter) Count(text string) int {
	if text == "" {
		return 0
	}
	return len(t.enc.Encode(text, nil, nil))
}

func (t *tiktokenCounter) CountMessages(messages []llm.ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += t.Count(m.Role) + t.Count(m.Content) + perMessageOverhead
	}
	return total + replyPrimingOverhead
}

func (t *tiktokenCounter) Name() string { return t.encoding }
