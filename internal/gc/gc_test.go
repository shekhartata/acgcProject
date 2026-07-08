package gc

import (
	"testing"

	"github.com/shekhartata/acgcProject/internal/scorer"
	"github.com/shekhartata/acgcProject/internal/statetree"
)

func TestShouldRun_budgetOverride(t *testing.T) {
	tree := statetree.NewTree("sess", "task")
	collector := NewGarbageCollector(Policy{
		MaxPromptTokens:    6000,
		SweepHeadroomRatio: 0.60,
	}, scorer.NewScorer(15, 500), &SimpleCompressor{})

	// Embedded policy: hard limit at 6000.
	if ok, _ := collector.ShouldRun(tree, 7000, 0); !ok {
		t.Fatal("expected token_pressure at 7000 with policy default")
	}

	// Override raises ceiling — below soft headroom (0.6 × 10000 = 6000).
	if ok, reason := collector.ShouldRun(tree, 5500, 10000); ok {
		t.Fatalf("expected no trigger with override budget, got %q", reason)
	}

	// Override lowers soft headroom: 4000 > 3000 (0.6 × 5000).
	if ok, reason := collector.ShouldRun(tree, 4000, 5000); !ok || reason != ReasonSoftHeadroom {
		t.Fatalf("expected soft_headroom with override, got ok=%v reason=%q", ok, reason)
	}
}
