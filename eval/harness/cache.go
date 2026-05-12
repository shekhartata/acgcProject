package harness

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Cache stores LLM responses on disk keyed by (scenario, probe, pipeline).
// Re-running the eval reuses cached responses unless the cache is cleared.
// File format is JSONL — one ProbeResult per line — for easy human inspection.
type Cache struct {
	dir     string
	mu      sync.Mutex
	entries map[string]*ProbeResult
	model   string
}

// NewCache opens (or creates) a cache directory. The model name is part of
// the cache key so swapping models invalidates the cache automatically.
func NewCache(dir, model string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	c := &Cache{
		dir:     dir,
		entries: make(map[string]*ProbeResult),
		model:   model,
	}
	if err := c.load(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Cache) load() error {
	path := c.filepath()
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open cache file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024) // allow long lines
	for scanner.Scan() {
		var r ProbeResult
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		c.entries[key(r.ScenarioID, r.ProbeID, r.Pipeline)] = &r
	}
	return scanner.Err()
}

func (c *Cache) filepath() string {
	safeModel := sanitize(c.model)
	return filepath.Join(c.dir, "responses_"+safeModel+".jsonl")
}

// Get returns a cached response if present. The returned ProbeResult has
// Cached=true set so the runner can avoid double-counting tokens spent.
func (c *Cache) Get(scenarioID, probeID string, pipeline PipelineKind) (*ProbeResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r, ok := c.entries[key(scenarioID, probeID, pipeline)]
	if !ok {
		return nil, false
	}
	cp := *r
	cp.Cached = true
	return &cp, true
}

// Put writes a fresh response to the in-memory map and appends it to disk.
func (c *Cache) Put(r *ProbeResult) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key(r.ScenarioID, r.ProbeID, r.Pipeline)] = r

	f, err := os.OpenFile(c.filepath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open cache for write: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(r)
}

// Has is a cheap presence check.
func (c *Cache) Has(scenarioID, probeID string, pipeline PipelineKind) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.entries[key(scenarioID, probeID, pipeline)]
	return ok
}

func key(scenarioID, probeID string, p PipelineKind) string {
	return scenarioID + "::" + probeID + "::" + string(p)
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	return string(out)
}
