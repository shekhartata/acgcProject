package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/chandrashekhartata/acgc/internal/compiler"
	"github.com/chandrashekhartata/acgc/internal/domain"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/statetree"
	"github.com/chandrashekhartata/acgc/stresstest/fixtures"
)

type EngineConfig struct {
	TokenBudget           int
	MaxTreeDepth          int
	MaxChildrenPerNode    int
	LowRelevanceThreshold float64
	StaleAfterTurns       int
	MaxTokensPerNode      int
}

func DefaultConfig() EngineConfig {
	return EngineConfig{
		TokenBudget:           6000,
		MaxTreeDepth:          10,
		MaxChildrenPerNode:    50,
		LowRelevanceThreshold: 0.30,
		StaleAfterTurns:       15,
		MaxTokensPerNode:      2000,
	}
}

func estimateTokens(s string) int {
	return len(s) / 4
}

// ReplaySession runs a single conversation through the in-process ACGC pipeline
// and returns per-turn stats showing raw vs compiled token counts.
func ReplaySession(name string, turns []fixtures.Turn, cfg EngineConfig) SessionResult {
	start := time.Now()

	sessionID := fmt.Sprintf("stress_%s_%d", name, time.Now().UnixNano())
	taskID := "stress_task"

	tree := statetree.NewTree(sessionID, taskID)
	sc := scorer.NewScorer(cfg.StaleAfterTurns, cfg.MaxTokensPerNode)
	comp := compiler.NewCompiler(cfg.TokenBudget)
	gcPolicy := gc.Policy{
		MaxPromptTokens:       cfg.TokenBudget,
		MaxTreeDepth:          cfg.MaxTreeDepth,
		MaxChildrenPerNode:    cfg.MaxChildrenPerNode,
		LowRelevanceThreshold: cfg.LowRelevanceThreshold,
		StaleAfterTurns:       cfg.StaleAfterTurns,
	}
	collector := gc.NewGarbageCollector(gcPolicy, sc, &gc.SimpleCompressor{})

	result := SessionResult{Name: name}

	// Track cumulative raw tokens (what you'd send without ACGC)
	cumulativeRawTokens := 0

	for i, turn := range turns {
		turnTokens := estimateTokens(turn.Content)
		cumulativeRawTokens += turnTokens

		eventType := domain.EventUserPrompt
		if turn.Role == "assistant" {
			eventType = domain.EventLLMResponse
		}

		event := &domain.Event{
			EventID:    fmt.Sprintf("evt_%s_%d", sessionID, i),
			SessionID:  sessionID,
			TaskID:     taskID,
			EventType:  eventType,
			Payload:    turn.Content,
			TokenCount: turnTokens,
			CreatedAt:  time.Now(),
		}

		// Simulate the session worker pipeline
		turnNum := tree.IncrementTurn()
		event.Sequence = turnNum
		tree.AddNode(event)

		activeNodes := tree.GetActiveNodes()
		sc.ScoreAll(activeNodes, turnNum)

		// Check and run GC
		ts := TurnStat{
			TurnNumber: i + 1,
			Role:       turn.Role,
			RawTokens:  cumulativeRawTokens,
		}

		estimatedActiveTokens := 0
		for _, n := range activeNodes {
			estimatedActiveTokens += n.TokenCount
		}

		if shouldRun, reason := collector.ShouldRun(tree, estimatedActiveTokens); shouldRun {
			gcResult := collector.Run(context.Background(), tree, reason)
			ts.GCTriggered = true
			ts.GCTokensFreed = gcResult.TokensFreed
			ts.GCNodesSwept = gcResult.NodesSwept
			result.GCRuns++
			result.TotalGCFreed += gcResult.TokensFreed
		}

		// Compile prompt (the fast path)
		activeNodes = tree.GetActiveNodes()
		compiled := comp.Compile(sessionID, taskID, turn.Content, activeNodes, "")

		ts.CompiledTokens = compiled.CompiledTokenCount
		ts.TokensSaved = ts.RawTokens - ts.CompiledTokens
		if ts.RawTokens > 0 {
			ts.ReductionPct = float64(ts.TokensSaved) / float64(ts.RawTokens) * 100
		}

		total, active, compressed, archived, _, _ := tree.Stats()
		ts.ActiveNodes = active
		ts.TotalNodes = total
		ts.ArchivedNodes = archived
		ts.CompressedNodes = compressed

		result.TurnStats = append(result.TurnStats, ts)

		// Track peaks
		if ts.RawTokens > result.PeakRawTokens {
			result.PeakRawTokens = ts.RawTokens
		}
		if ts.CompiledTokens > result.PeakCompiledTokens {
			result.PeakCompiledTokens = ts.CompiledTokens
		}
	}

	// Aggregate session result
	result.TotalTurns = len(turns)
	if len(result.TurnStats) > 0 {
		last := result.TurnStats[len(result.TurnStats)-1]
		result.TotalRawTokens = last.RawTokens
		result.TotalCompiled = last.CompiledTokens
		result.TotalSaved = last.TokensSaved
		if result.TotalRawTokens > 0 {
			result.AvgReductionPct = float64(result.TotalSaved) / float64(result.TotalRawTokens) * 100
		}
		result.FinalActiveNodes = last.ActiveNodes
		result.FinalTotalNodes = last.TotalNodes
	}

	result.Duration = time.Since(start)

	// Run coherency checks
	result.CoherencyScore = CheckCoherency(tree, turns, cfg)

	return result
}
