package runner

import (
	"context"
	"hash/fnv"
	"math"
)

// MockEmbedder produces deterministic, content-derived vectors without any
// network call. Good enough to exercise the HNSW + scorer code paths under
// -race and to detect regressions in the semantic plumbing — NOT a substitute
// for a real embedding model for accuracy/quality measurement.
//
// Strategy: hash each token of the input into one of `dim` buckets, then
// L2-normalize. Same input → same vector; similar inputs (sharing tokens) →
// similar vectors. That's enough to make the semantic signal non-trivial
// during stress testing.
type MockEmbedder struct {
	dim int
}

func NewMockEmbedder(dim int) *MockEmbedder {
	if dim <= 0 {
		dim = 128
	}
	return &MockEmbedder{dim: dim}
}

func (m *MockEmbedder) Dim() int      { return m.dim }
func (m *MockEmbedder) Model() string { return "mock-bag-of-hashes" }

func (m *MockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, m.dim)
	// Word-level tokenization is good enough for the stress fixtures.
	start := -1
	for i := 0; i <= len(text); i++ {
		isSep := i == len(text) || text[i] == ' ' || text[i] == '\n' || text[i] == '\t' ||
			text[i] == '.' || text[i] == ',' || text[i] == ':' || text[i] == ';' ||
			text[i] == '!' || text[i] == '?'
		if !isSep {
			if start == -1 {
				start = i
			}
			continue
		}
		if start >= 0 && i > start {
			tok := text[start:i]
			h := fnv.New32a()
			_, _ = h.Write([]byte(tok))
			bucket := int(h.Sum32()) % m.dim
			if bucket < 0 {
				bucket += m.dim
			}
			vec[bucket] += 1.0
		}
		start = -1
	}

	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm == 0 {
		// All-zero would break cosine similarity. Push one bucket to 1.
		vec[0] = 1.0
		return vec, nil
	}
	inv := float32(1.0 / math.Sqrt(norm))
	for i := range vec {
		vec[i] *= inv
	}
	return vec, nil
}
