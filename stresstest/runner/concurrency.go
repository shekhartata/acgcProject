package runner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chandrashekhartata/acgc/internal/compiler"
	"github.com/chandrashekhartata/acgc/internal/domain"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/statetree"
	"github.com/chandrashekhartata/acgc/stresstest/fixtures"
)

type ConcurrencyResult struct {
	TestName       string        `json:"test_name"`
	Passed         bool          `json:"passed"`
	Details        string        `json:"details"`
	Duration       time.Duration `json:"duration_ns"`
	OperationCount int           `json:"operation_count"`
}

// RunConcurrencyTests runs all concurrency stress tests.
func RunConcurrencyTests(convos map[string][]fixtures.Turn, cfg EngineConfig) []ConcurrencyResult {
	var results []ConcurrencyResult

	results = append(results, testParallelSessions(convos, cfg))
	results = append(results, testConcurrentReadWrite(cfg))
	results = append(results, testGCUnderContention(cfg))
	results = append(results, testConcurrentCompile(cfg))

	return results
}

// testParallelSessions: replay all conversations concurrently.
func testParallelSessions(convos map[string][]fixtures.Turn, cfg EngineConfig) ConcurrencyResult {
	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var sessions []SessionResult
	errCount := int32(0)

	for name, turns := range convos {
		wg.Add(1)
		go func(n string, t []fixtures.Turn) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt32(&errCount, 1)
				}
			}()
			result := ReplaySession(n, t, cfg)
			mu.Lock()
			sessions = append(sessions, result)
			mu.Unlock()
		}(name, turns)
	}

	wg.Wait()
	dur := time.Since(start)

	passed := errCount == 0 && len(sessions) == len(convos)
	details := fmt.Sprintf("%d/%d sessions completed successfully, %d panics",
		len(sessions), len(convos), errCount)

	return ConcurrencyResult{
		TestName:       "parallel_sessions",
		Passed:         passed,
		Details:        details,
		Duration:       dur,
		OperationCount: len(convos),
	}
}

// testConcurrentReadWrite: one goroutine writes events, multiple read compiled prompts concurrently.
func testConcurrentReadWrite(cfg EngineConfig) ConcurrencyResult {
	start := time.Now()
	sessionID := "concurrent_rw_test"
	taskID := "stress_task"

	tree := statetree.NewTree(sessionID, taskID)
	sc := scorer.NewScorer(cfg.StaleAfterTurns, cfg.MaxTokensPerNode)
	comp := compiler.NewCompiler(cfg.TokenBudget)

	totalOps := int32(0)
	errCount := int32(0)
	writerDone := make(chan struct{})

	// Writer goroutine: adds 40 events
	go func() {
		defer close(writerDone)
		for i := 0; i < 40; i++ {
			event := &domain.Event{
				EventID:    fmt.Sprintf("evt_rw_%d", i),
				SessionID:  sessionID,
				TaskID:     taskID,
				EventType:  domain.EventUserPrompt,
				Payload:    fmt.Sprintf("Concurrent test message %d about topic %d", i, i%5),
				TokenCount: 20 + (i * 3),
				CreatedAt:  time.Now(),
			}

			turn := tree.IncrementTurn()
			event.Sequence = turn
			tree.AddNode(event)

			activeNodes := tree.GetActiveNodes()
			sc.ScoreAll(activeNodes, turn)
			atomic.AddInt32(&totalOps, 1)

			time.Sleep(time.Millisecond)
		}
	}()

	// Reader goroutines: continuously compile prompts while writer is active
	var readWg sync.WaitGroup
	for r := 0; r < 5; r++ {
		readWg.Add(1)
		go func(readerID int) {
			defer readWg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt32(&errCount, 1)
				}
			}()
			for {
				select {
				case <-writerDone:
					return
				default:
					activeNodes := tree.GetActiveNodes()
					compiled := comp.Compile(sessionID, taskID,
						fmt.Sprintf("Reader %d query", readerID), activeNodes, "")
					if compiled == nil {
						atomic.AddInt32(&errCount, 1)
					}
					atomic.AddInt32(&totalOps, 1)
					time.Sleep(500 * time.Microsecond)
				}
			}
		}(r)
	}

	readWg.Wait()
	dur := time.Since(start)

	passed := errCount == 0
	details := fmt.Sprintf("%d total ops (1 writer + 5 readers), %d errors",
		totalOps, errCount)

	return ConcurrencyResult{
		TestName:       "concurrent_read_write",
		Passed:         passed,
		Details:        details,
		Duration:       dur,
		OperationCount: int(totalOps),
	}
}

// testGCUnderContention: triggers GC while reads/writes are happening.
func testGCUnderContention(cfg EngineConfig) ConcurrencyResult {
	start := time.Now()
	sessionID := "gc_contention_test"
	taskID := "stress_task"

	// Use aggressive GC policy to trigger it frequently
	aggressiveCfg := cfg
	aggressiveCfg.TokenBudget = 500
	aggressiveCfg.MaxChildrenPerNode = 5
	aggressiveCfg.LowRelevanceThreshold = 0.5

	tree := statetree.NewTree(sessionID, taskID)
	sc := scorer.NewScorer(aggressiveCfg.StaleAfterTurns, aggressiveCfg.MaxTokensPerNode)
	comp := compiler.NewCompiler(aggressiveCfg.TokenBudget)
	gcPolicy := gc.Policy{
		MaxPromptTokens:       aggressiveCfg.TokenBudget,
		MaxTreeDepth:          aggressiveCfg.MaxTreeDepth,
		MaxChildrenPerNode:    aggressiveCfg.MaxChildrenPerNode,
		LowRelevanceThreshold: aggressiveCfg.LowRelevanceThreshold,
		StaleAfterTurns:       aggressiveCfg.StaleAfterTurns,
	}
	collector := gc.NewGarbageCollector(gcPolicy, sc, &gc.SimpleCompressor{})

	totalOps := int32(0)
	errCount := int32(0)
	gcRuns := int32(0)

	var wg sync.WaitGroup

	// Writer + GC goroutine
	writerDone := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(writerDone)
		for i := 0; i < 50; i++ {
			event := &domain.Event{
				EventID:    fmt.Sprintf("evt_gc_%d", i),
				SessionID:  sessionID,
				TaskID:     taskID,
				EventType:  domain.EventUserPrompt,
				Payload:    fmt.Sprintf("GC contention test message %d with enough content to build up tokens for garbage collection trigger", i),
				TokenCount: 50 + (i * 5),
				CreatedAt:  time.Now(),
			}

			turn := tree.IncrementTurn()
			event.Sequence = turn
			tree.AddNode(event)

			activeNodes := tree.GetActiveNodes()
			sc.ScoreAll(activeNodes, turn)

			activeTokens := 0
			for _, n := range activeNodes {
				activeTokens += n.TokenCount
			}

			if shouldRun, reason := collector.ShouldRun(tree, activeTokens); shouldRun {
				collector.Run(context.Background(), tree, reason)
				atomic.AddInt32(&gcRuns, 1)
			}

			atomic.AddInt32(&totalOps, 1)
			time.Sleep(500 * time.Microsecond)
		}
	}()

	// Concurrent readers during GC
	for r := 0; r < 3; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt32(&errCount, 1)
				}
			}()
			for {
				select {
				case <-writerDone:
					return
				default:
					activeNodes := tree.GetActiveNodes()
					compiled := comp.Compile(sessionID, taskID,
						fmt.Sprintf("GC reader %d", readerID), activeNodes, "")
					if compiled == nil {
						atomic.AddInt32(&errCount, 1)
					}
					_ = tree.TurnCount()
					tree.Stats()
					atomic.AddInt32(&totalOps, 1)
					time.Sleep(200 * time.Microsecond)
				}
			}
		}(r)
	}

	wg.Wait()
	dur := time.Since(start)

	passed := errCount == 0
	details := fmt.Sprintf("%d total ops, %d GC runs, %d errors",
		totalOps, gcRuns, errCount)

	return ConcurrencyResult{
		TestName:       "gc_under_contention",
		Passed:         passed,
		Details:        details,
		Duration:       dur,
		OperationCount: int(totalOps),
	}
}

// testConcurrentCompile: multiple goroutines compile prompts from the same tree simultaneously.
func testConcurrentCompile(cfg EngineConfig) ConcurrencyResult {
	start := time.Now()
	sessionID := "concurrent_compile_test"
	taskID := "stress_task"

	tree := statetree.NewTree(sessionID, taskID)
	sc := scorer.NewScorer(cfg.StaleAfterTurns, cfg.MaxTokensPerNode)

	// Pre-populate tree with 30 nodes
	for i := 0; i < 30; i++ {
		event := &domain.Event{
			EventID:    fmt.Sprintf("evt_cc_%d", i),
			SessionID:  sessionID,
			TaskID:     taskID,
			EventType:  domain.EventUserPrompt,
			Payload:    fmt.Sprintf("Pre-populated message %d for concurrent compile testing", i),
			TokenCount: 25,
			CreatedAt:  time.Now(),
		}
		turn := tree.IncrementTurn()
		event.Sequence = turn
		tree.AddNode(event)
	}

	activeNodes := tree.GetActiveNodes()
	sc.ScoreAll(activeNodes, tree.TurnCount())

	totalOps := int32(0)
	errCount := int32(0)

	var wg sync.WaitGroup
	compilers := 10

	for c := 0; c < compilers; c++ {
		wg.Add(1)
		go func(compilerID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt32(&errCount, 1)
				}
			}()
			comp := compiler.NewCompiler(cfg.TokenBudget)
			for i := 0; i < 20; i++ {
				nodes := tree.GetActiveNodes()
				result := comp.Compile(sessionID, taskID,
					fmt.Sprintf("Compiler %d iteration %d", compilerID, i), nodes, "")
				if result == nil || result.FinalPrompt == "" {
					atomic.AddInt32(&errCount, 1)
				}
				atomic.AddInt32(&totalOps, 1)
			}
		}(c)
	}

	wg.Wait()
	dur := time.Since(start)

	passed := errCount == 0
	details := fmt.Sprintf("%d compilers × 20 iterations = %d total ops, %d errors",
		compilers, totalOps, errCount)

	return ConcurrencyResult{
		TestName:       "concurrent_compile",
		Passed:         passed,
		Details:        details,
		Duration:       dur,
		OperationCount: int(totalOps),
	}
}
