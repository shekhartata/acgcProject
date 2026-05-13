package runner

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chandrashekhartata/acgc/internal/compiler"
	"github.com/chandrashekhartata/acgc/internal/domain"
	"github.com/chandrashekhartata/acgc/internal/embedding"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/session"
	"github.com/chandrashekhartata/acgc/internal/statetree"
	"github.com/chandrashekhartata/acgc/internal/vectorindex"
	"github.com/chandrashekhartata/acgc/stresstest/fixtures"
)

type EngineConfig struct {
	TokenBudget           int
	MaxTreeDepth          int
	MaxChildrenPerNode    int
	LowRelevanceThreshold float64
	// DecisionSweepFloor: 0 disables soft floor for NodeDecision. Must sit
	// strictly below LowRelevanceThreshold or bare decisions become un-sweepable.
	DecisionSweepFloor float64
	// MaxActiveNodes: count-based GC trigger. 0 disables.
	MaxActiveNodes int
	// SweepHeadroomRatio: soft trigger at ratio × TokenBudget. 0 disables.
	SweepHeadroomRatio float64
	StaleAfterTurns    int
	MaxTokensPerNode   int

	// Optional. When Embedder is nil, semantic ops are skipped.
	Embedder             embedding.Provider
	SemanticWeight       float64
	TopKAtCompile        int
	ArchiveTopKAtCompile int
	EmbedDim             int
	HNSWM                int
	HNSWEFSearch         int
}

func DefaultConfig() EngineConfig {
	return EngineConfig{
		TokenBudget:           6000,
		MaxTreeDepth:          10,
		MaxChildrenPerNode:    50,
		LowRelevanceThreshold: 0.30,
		// Phase 2: 0.35 → 0.20; floor must sit below LowRelevanceThreshold.
		DecisionSweepFloor: 0.20,
		// Phase 2: dense-session triggers.
		MaxActiveNodes:       25,
		SweepHeadroomRatio:   0.60,
		StaleAfterTurns:      15,
		MaxTokensPerNode:     2000,
		SemanticWeight:       0.20,
		TopKAtCompile:        12,
		ArchiveTopKAtCompile: 12,
		EmbedDim:             128,
		HNSWM:                16,
		HNSWEFSearch:         50,
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
	if cfg.SemanticWeight > 0 {
		sc.SetSemanticWeight(cfg.SemanticWeight)
	}
	comp := compiler.NewCompiler(cfg.TokenBudget)
	gcPolicy := gc.Policy{
		MaxPromptTokens:       cfg.TokenBudget,
		MaxTreeDepth:          cfg.MaxTreeDepth,
		MaxChildrenPerNode:    cfg.MaxChildrenPerNode,
		LowRelevanceThreshold: cfg.LowRelevanceThreshold,
		DecisionSweepFloor:    cfg.DecisionSweepFloor,
		MaxActiveNodes:        cfg.MaxActiveNodes,
		SweepHeadroomRatio:    cfg.SweepHeadroomRatio,
		StaleAfterTurns:       cfg.StaleAfterTurns,
	}
	collector := gc.NewGarbageCollector(gcPolicy, sc, &gc.SimpleCompressor{})

	var (
		activeIdx         vectorindex.Index
		archIdx           vectorindex.Index
		lastUserEmbedding []float32
		semanticOn        = cfg.Embedder != nil
	)
	if semanticOn {
		activeIdx = vectorindex.NewHNSW(vectorindex.Config{
			Dim:      cfg.EmbedDim,
			M:        cfg.HNSWM,
			EFSearch: cfg.HNSWEFSearch,
		})
		archIdx = vectorindex.NewHNSW(vectorindex.Config{
			Dim:      cfg.EmbedDim,
			M:        cfg.HNSWM,
			EFSearch: cfg.HNSWEFSearch,
		})
	}

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
		node := tree.AddNode(event)

		if semanticOn && node != nil {
			vec, err := cfg.Embedder.Embed(context.Background(), turn.Content)
			if err != nil {
				log.Printf("stresstest: embed failed at turn %d: %v", turnNum, err)
			} else {
				node.Embedding = vec
				node.EmbedModel = cfg.Embedder.Model()
				if err := activeIdx.Insert(node.NodeID, vec); err != nil {
					log.Printf("stresstest: HNSW insert failed: %v", err)
				}
				if turn.Role == "user" {
					lastUserEmbedding = vec
				}
			}
		}

		activeNodes := tree.GetActiveNodes()
		sc.ScoreAll(activeNodes, turnNum, lastUserEmbedding)

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
			preIDs := make(map[string]bool, len(activeNodes))
			for _, n := range activeNodes {
				preIDs[n.NodeID] = true
			}
			gcResult := collector.Run(context.Background(), tree, reason, lastUserEmbedding)
			ts.GCTriggered = true
			ts.GCTokensFreed = gcResult.TokensFreed
			ts.GCNodesSwept = gcResult.NodesSwept
			result.GCRuns++
			result.TotalGCFreed += gcResult.TokensFreed

			if semanticOn && activeIdx != nil && archIdx != nil {
				postActive := tree.GetActiveNodes()
				postIDs := make(map[string]bool, len(postActive))
				for _, n := range postActive {
					postIDs[n.NodeID] = true
				}
				for id := range preIDs {
					if postIDs[id] {
						continue
					}
					if n, ok := tree.GetNode(id); ok && len(n.Embedding) > 0 {
						_ = archIdx.Insert(id, n.Embedding)
					}
					activeIdx.Delete(id)
				}
			}
		}

		// Compile prompt (the fast path)
		activeNodes = tree.GetActiveNodes()
		var compiled *domain.CompiledPrompt
		if semanticOn && turn.Role == "user" && turn.Content != "" {
			qVec, err := cfg.Embedder.Embed(context.Background(), turn.Content)
			if err != nil {
				compiled = comp.Compile(sessionID, taskID, turn.Content, activeNodes, "")
			} else {
				topK := cfg.TopKAtCompile
				if topK <= 0 {
					topK = 12
				}
				kz := cfg.ArchiveTopKAtCompile
				if kz <= 0 {
					kz = 12
				}
				hitsA, qErr := activeIdx.Query(qVec, topK)
				if qErr != nil {
					compiled = comp.Compile(sessionID, taskID, turn.Content, activeNodes, "")
				} else {
					hitsZ, zErr := archIdx.Query(qVec, kz)
					if zErr != nil {
						compiled = comp.Compile(sessionID, taskID, turn.Content, activeNodes, "")
					} else {
						merged := session.MergeSemanticHits(hitsA, hitsZ)
						nodes := session.NodesForSemanticCompile(tree, activeNodes, merged)
						compiled = comp.CompileWithSemantic(sessionID, taskID, turn.Content,
							nodes, "", cfg.SemanticWeight, merged)
					}
				}
			}
		} else {
			compiled = comp.Compile(sessionID, taskID, turn.Content, activeNodes, "")
		}

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
