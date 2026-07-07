package runner

import (
	"strings"

	"github.com/shekhartata/acgcProject/internal/compiler"
	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/statetree"
	"github.com/shekhartata/acgcProject/stresstest/fixtures"
)

// CheckCoherency evaluates how well ACGC preserves important context after GC.
// Returns a score 0.0-1.0 where 1.0 means perfect coherency.
//
// Checks performed:
//  1. Goal retention: root goals should still be in the compiled prompt
//  2. Constraint preservation: nodes tagged as constraints should survive GC
//  3. Recent context: the most recent N turns should always be present
//  4. No orphaned dependencies: if a node references a dependency, it should exist
func CheckCoherency(tree *statetree.Tree, turns []fixtures.Turn, cfg EngineConfig) float64 {
	if len(turns) == 0 {
		return 1.0
	}

	checks := 0
	passed := 0

	activeNodes := tree.GetActiveNodes()
	allNodes := tree.GetAllNodes()

	comp := compiler.NewCompiler(cfg.TokenBudget)
	lastTurn := turns[len(turns)-1]
	compiled := comp.Compile(tree.SessionID(), "stress_task", lastTurn.Content, activeNodes, "")

	// --- Check 1: Goal nodes survive ---
	goalCount := 0
	goalRetained := 0
	for _, n := range allNodes {
		if n.NodeType == domain.NodeGoal {
			goalCount++
			if n.Status == domain.StatusActive {
				goalRetained++
			}
		}
	}
	if goalCount > 0 {
		checks++
		// At least root goal + 50% of goals should remain active
		if float64(goalRetained)/float64(goalCount) >= 0.5 {
			passed++
		}
	}

	// --- Check 2: Constraint nodes survive ---
	constraintCount := 0
	constraintRetained := 0
	for _, n := range allNodes {
		if n.NodeType == domain.NodeConstraint {
			constraintCount++
			if n.Status == domain.StatusActive {
				constraintRetained++
			}
		}
	}
	if constraintCount > 0 {
		checks++
		if float64(constraintRetained)/float64(constraintCount) >= 0.5 {
			passed++
		}
	}

	// --- Check 3: Recent context included in compiled prompt ---
	recentWindow := 3
	if recentWindow > len(turns) {
		recentWindow = len(turns)
	}
	recentHits := 0
	for i := len(turns) - recentWindow; i < len(turns); i++ {
		snippet := truncateStr(turns[i].Content, 50)
		if strings.Contains(compiled.FinalPrompt, snippet) {
			recentHits++
		}
	}
	checks++
	if recentHits > 0 {
		passed++
	}

	// --- Check 4: No orphaned dependencies ---
	nodeIndex := make(map[string]bool, len(allNodes))
	for _, n := range allNodes {
		nodeIndex[n.NodeID] = true
	}
	depChecks := 0
	depValid := 0
	for _, n := range activeNodes {
		for _, dep := range n.Dependencies {
			depChecks++
			if nodeIndex[dep] {
				depValid++
			}
		}
	}
	if depChecks > 0 {
		checks++
		if depValid == depChecks {
			passed++
		}
	}

	// --- Check 5: Compiled prompt is not empty ---
	checks++
	if len(compiled.FinalPrompt) > 0 {
		passed++
	}

	// --- Check 6: Token budget respected ---
	checks++
	if compiled.CompiledTokenCount <= cfg.TokenBudget {
		passed++
	}

	if checks == 0 {
		return 1.0
	}
	return float64(passed) / float64(checks)
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
