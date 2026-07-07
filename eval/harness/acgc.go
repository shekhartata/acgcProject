package harness

import (
	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/embedding"
	"github.com/shekhartata/acgcProject/internal/vectorindex"
)

// ACGCConfig mirrors the runtime ACGC policy knobs. Defaults match the
// production server (cmd/acgc/main.go).
type ACGCConfig struct {
	TokenBudget           int
	MaxTreeDepth          int
	MaxChildrenPerNode    int
	LowRelevanceThreshold float64
	StaleAfterTurns       int
	MaxTokensPerNode      int
	// DecisionSweepFloor: GC soft floor for NodeDecision with empty Facts/Decisions (0 disables).
	// Must stay strictly below LowRelevanceThreshold or bare decisions become un-sweepable.
	DecisionSweepFloor float64
	// MaxActiveNodes: count-based GC trigger. 0 disables.
	MaxActiveNodes int
	// SweepHeadroomRatio: soft trigger at ratio × TokenBudget. 0 disables.
	SweepHeadroomRatio  float64
	ArchiveSemanticTopK int

	// Optional semantic scoring. When Embedder is nil, the eval runs in
	// pure-heuristic mode (matches v1).
	Embedder       embedding.Provider
	SemanticWeight float64
	TopKAtCompile  int
	HNSWConfig     vectorindex.Config
	// CacheStableRender reorders selected nodes into stable turn order for provider prefix caching.
	CacheStableRender bool
}

func DefaultACGCConfig() ACGCConfig {
	return ACGCConfig{
		TokenBudget:           6000,
		MaxTreeDepth:          10,
		MaxChildrenPerNode:    50,
		LowRelevanceThreshold: 0.30,
		StaleAfterTurns:       15,
		MaxTokensPerNode:      2000,
		// Phase 2: 0.35 → 0.20. Floor must sit below LowRelevanceThreshold,
		// otherwise bare NodeDecision nodes are never swept.
		DecisionSweepFloor: 0.20,
		// Phase 2: count + headroom triggers so GC actually fires on short
		// conversations that never approach TokenBudget.
		MaxActiveNodes:      25,
		SweepHeadroomRatio:  0.60,
		ArchiveSemanticTopK: 12,
	}
}

// embeddingPayload mirrors the helper in internal/session/manager.go.
// Kept in sync intentionally — the eval pipeline should embed the same text
// the runtime would, otherwise the eval doesn't reflect production behavior.
func embeddingPayload(node *domain.StateNode, event *domain.Event) string {
	if node != nil {
		if node.Summary != "" && node.Title != "" {
			return node.Title + ": " + node.Summary
		}
		if node.Summary != "" {
			return node.Summary
		}
		if node.Title != "" {
			return node.Title
		}
	}
	if event != nil {
		return event.Payload
	}
	return ""
}
