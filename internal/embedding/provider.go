// Package embedding provides text-to-vector providers for semantic scoring.
//
// The Provider interface is intentionally small so we can swap providers
// (OpenAI now, a local ONNX model later) without touching call sites.
package embedding

import "context"

type Provider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dim() int
	Model() string
}

type Config struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Dim      int
}
