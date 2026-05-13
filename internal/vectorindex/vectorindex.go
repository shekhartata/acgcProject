// Package vectorindex wraps an in-memory HNSW implementation behind a small
// library-agnostic interface. Today this is github.com/coder/hnsw; swap-out
// would be confined to this file.
//
// Per-session usage: one Index per session, owned by the session worker for
// writes; concurrent reads are safe via the embedded RWMutex.
package vectorindex

import (
	"fmt"
	"math"
	"sync"

	"github.com/coder/hnsw"
)

// Hit is a single approximate-nearest-neighbor result.
//
// Score is cosine similarity in [0, 1]. Distance is the raw cosine distance
// returned by the underlying library (lower = more similar). We expose both
// so callers can pick whichever feels natural.
type Hit struct {
	NodeID   string
	Distance float32
	Score    float32
}

type Config struct {
	Dim      int
	M        int
	EFSearch int
}

func DefaultConfig() Config {
	return Config{Dim: 1536, M: 16, EFSearch: 50}
}

// Index is the small surface the rest of the codebase depends on.
type Index interface {
	Insert(id string, vec []float32) error
	Delete(id string) bool
	Query(vec []float32, k int) ([]Hit, error)
	Len() int
	RebuildFromVectors(items map[string][]float32) error
}

// HNSWIndex is the coder/hnsw-backed implementation.
//
// Implementation note: coder/hnsw@v0.6.1 has a Delete() that leaves the
// graph in an inconsistent state — subsequent Insert() panics with a nil
// deref inside layerNode.search. We work around this with a tombstone set:
// Delete just marks the ID, Query filters tombstones out, and we rebuild
// the underlying graph once the tombstone fraction crosses a threshold.
// This keeps lookups correct without ever exercising the library's buggy
// Delete path.
type HNSWIndex struct {
	cfg        Config
	mu         sync.RWMutex
	graph      *hnsw.Graph[string]
	vectors    map[string][]float32 // source-of-truth for rebuilds
	tombstones map[string]bool
}

func NewHNSW(cfg Config) *HNSWIndex {
	if cfg.M <= 0 {
		cfg.M = 16
	}
	if cfg.EFSearch <= 0 {
		cfg.EFSearch = 50
	}

	return &HNSWIndex{
		cfg:        cfg,
		graph:      newGraph(cfg),
		vectors:    make(map[string][]float32),
		tombstones: make(map[string]bool),
	}
}

func newGraph(cfg Config) *hnsw.Graph[string] {
	g := hnsw.NewGraph[string]()
	g.M = cfg.M
	g.EfSearch = cfg.EFSearch
	g.Distance = hnsw.CosineDistance
	return g
}

func (h *HNSWIndex) Insert(id string, vec []float32) error {
	if id == "" {
		return fmt.Errorf("vectorindex: empty id")
	}
	if len(vec) == 0 {
		return fmt.Errorf("vectorindex: empty vector for id=%s", id)
	}
	if h.cfg.Dim > 0 && len(vec) != h.cfg.Dim {
		return fmt.Errorf("vectorindex: dim mismatch for id=%s: got %d, want %d", id, len(vec), h.cfg.Dim)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	// Clear any stale tombstone (re-insert of a previously deleted id).
	delete(h.tombstones, id)
	h.vectors[id] = vec
	h.graph.Add(hnsw.MakeNode(id, vec))
	return nil
}

func (h *HNSWIndex) Delete(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.vectors[id]; !ok {
		return false
	}
	h.tombstones[id] = true

	// Rebuild eagerly when tombstones get heavy — keeps Query fan-out
	// from blowing up on long-running sessions.
	if len(h.tombstones)*5 > len(h.vectors) {
		h.compactLocked()
	}
	return true
}

func (h *HNSWIndex) Query(vec []float32, k int) ([]Hit, error) {
	if len(vec) == 0 {
		return nil, fmt.Errorf("vectorindex: empty query vector")
	}
	if k <= 0 {
		k = 10
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.graph.Len() == 0 {
		return nil, nil
	}
	if h.cfg.Dim > 0 && len(vec) != h.cfg.Dim {
		return nil, fmt.Errorf("vectorindex: query dim mismatch: got %d, want %d", len(vec), h.cfg.Dim)
	}

	// Over-fetch to compensate for tombstones we'll filter out.
	fetch := k + len(h.tombstones)
	if fetch > h.graph.Len() {
		fetch = h.graph.Len()
	}

	nodes := h.graph.Search(vec, fetch)
	hits := make([]Hit, 0, k)
	for _, n := range nodes {
		if h.tombstones[n.Key] {
			continue
		}
		d := cosineDistance(vec, n.Value)
		hits = append(hits, Hit{
			NodeID:   n.Key,
			Distance: d,
			Score:    clamp01(1.0 - d),
		})
		if len(hits) >= k {
			break
		}
	}
	return hits, nil
}

func (h *HNSWIndex) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.vectors) - len(h.tombstones)
}

// RebuildFromVectors replaces the index contents wholesale. Used on session
// rehydration: load embeddings from Mongo, rebuild graph from scratch.
func (h *HNSWIndex) RebuildFromVectors(items map[string][]float32) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.vectors = make(map[string][]float32, len(items))
	for id, vec := range items {
		if id == "" || len(vec) == 0 {
			continue
		}
		if h.cfg.Dim > 0 && len(vec) != h.cfg.Dim {
			continue
		}
		h.vectors[id] = vec
	}
	h.tombstones = make(map[string]bool)
	h.compactLocked()
	return nil
}

// compactLocked rebuilds the underlying graph from the vectors map,
// dropping any tombstoned entries. Caller must hold h.mu (write).
func (h *HNSWIndex) compactLocked() {
	g := newGraph(h.cfg)
	for id, vec := range h.vectors {
		if h.tombstones[id] {
			delete(h.vectors, id)
			continue
		}
		g.Add(hnsw.MakeNode(id, vec))
	}
	h.tombstones = make(map[string]bool)
	h.graph = g
}

// cosineDistance mirrors what coder/hnsw computes internally so we can
// surface a stable Score next to each Search result without re-querying
// the library for distances.
func cosineDistance(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 1.0
	}
	var dot, na, nb float64
	for i := range a {
		x := float64(a[i])
		y := float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	if na == 0 || nb == 0 {
		return 1.0
	}
	sim := dot / (math.Sqrt(na) * math.Sqrt(nb))
	return float32(1.0 - sim)
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
