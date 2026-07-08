// acgc-cachebench measures provider prefix cache hits across sequential LLM
// calls with a growing ACGC session. Use -compare for OFF vs ON side-by-side.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/shekhartata/acgcProject/eval/datasets"
	"github.com/shekhartata/acgcProject/internal/compiler"
	"github.com/shekhartata/acgcProject/internal/config"
	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/embedding"
	"github.com/shekhartata/acgcProject/internal/gc"
	"github.com/shekhartata/acgcProject/internal/llm"
	"github.com/shekhartata/acgcProject/internal/scorer"
	"github.com/shekhartata/acgcProject/internal/session"
	"github.com/shekhartata/acgcProject/internal/statetree"
	"github.com/shekhartata/acgcProject/internal/tokenizer"
	"github.com/shekhartata/acgcProject/internal/vectorindex"
)

const systemPrompt = "You are a helpful technical assistant. Answer concisely and accurately based on the provided context."

func main() {
	log.SetFlags(0)

	scenarioID := flag.String("scenario", "deep_history_recall_1", "built-in eval scenario")
	turns := flag.Int("turns", 5, "sequential LLM calls as history grows")
	stableRender := flag.Bool("stable-render", false, "enable cache-stable render")
	compare := flag.Bool("compare", false, "run stable OFF then ON and print both tables")
	freeze := flag.Bool("freeze", false, "ingest full history once, then repeat identical probes (cache sanity)")
	tokenBudget := flag.Int("token-budget", 8000, "compiler token budget")
	maxTokens := flag.Int("max-tokens", 64, "max completion tokens per LLM call")
	sameQuestion := flag.String("question", "Which primary datastore did we commit to for this platform?", "probe question (repeated each turn)")
	flag.Parse()

	if *compare {
		if *freeze {
			fmt.Println("========== FREEZE: full history, identical probes ==========")
			fmt.Println("--- stable_render OFF ---")
			off := runFrozenBench(*scenarioID, *turns, false, *tokenBudget, *maxTokens, *sameQuestion)
			fmt.Println("--- stable_render ON ---")
			on := runFrozenBench(*scenarioID, *turns, true, *tokenBudget, *maxTokens, *sameQuestion)
			printCompareSummary(off, on)
			return
		}
		fmt.Println("========== GROW: history expands each turn ==========")
		fmt.Println("--- stable_render OFF ---")
		off := runIncrementalBench(*scenarioID, *turns, false, *tokenBudget, *maxTokens, *sameQuestion)
		fmt.Println("--- stable_render ON ---")
		on := runIncrementalBench(*scenarioID, *turns, true, *tokenBudget, *maxTokens, *sameQuestion)
		printCompareSummary(off, on)
		return
	}

	if *freeze {
		runFrozenBench(*scenarioID, *turns, *stableRender, *tokenBudget, *maxTokens, *sameQuestion)
		return
	}

	runIncrementalBench(*scenarioID, *turns, *stableRender, *tokenBudget, *maxTokens, *sameQuestion)
}

type benchResult struct {
	label          string
	rows           []rowResult
	totalCompiled  int
	totalCached    int
	turn2PlusCache int
}

type rowResult struct {
	turn         int
	historyTurns int
	compiled     int
	cached       int
	promptHash   string
	promptStable bool
}

func runIncrementalBench(scenarioID string, nTurns int, stableRender bool, tokenBudget, maxTokens int, question string) benchResult {
	cfg := config.Load()
	if cfg.DefaultLLMAPIKey == "" {
		log.Fatal("ACGC_LLM_API_KEY required")
	}

	sc := datasets.ByID(scenarioID)
	if sc == nil {
		log.Fatalf("unknown scenario %q", scenarioID)
	}

	counter := tokenizer.New(cfg.DefaultLLMModel)
	llmClient := llm.NewClient(llm.Config{
		Provider: cfg.DefaultLLMProvider,
		BaseURL:  cfg.DefaultLLMBaseURL,
		APIKey:   cfg.DefaultLLMAPIKey,
		Model:    cfg.DefaultLLMModel,
	})

	embedKey := cfg.EmbedAPIKey
	if embedKey == "" {
		embedKey = cfg.DefaultLLMAPIKey
	}
	embedder := embedding.NewOpenAI(embedding.Config{
		BaseURL: cfg.EmbedBaseURL,
		APIKey:  embedKey,
		Model:   cfg.EmbedModel,
	})

	label := "OFF"
	if stableRender {
		label = "ON"
	}
	fmt.Printf("mode=incremental scenario=%s stable_render=%s semantic=true token_budget=%d probe_turns=%d\n",
		scenarioID, label, tokenBudget, nTurns)

	ctx := context.Background()
	sessionID := "cache_bench"
	taskID := "cache_bench"

	tree := statetree.NewTree(sessionID, taskID)
	scr := scorer.NewScorer(15, 2000)
	scr.SetSemanticWeight(0.20)

	comp := compiler.NewCompilerWithCounter(tokenBudget, counter)
	if stableRender {
		comp.WithCacheStableRender(true)
	}

	collector := gc.NewGarbageCollector(gc.Policy{
		MaxPromptTokens:       tokenBudget,
		MaxTreeDepth:          10,
		MaxChildrenPerNode:    50,
		LowRelevanceThreshold: 0.30,
		DecisionSweepFloor:    0.20,
		MaxActiveNodes:        0,
		SweepHeadroomRatio:    0,
		StaleAfterTurns:       15,
	}, scr, &gc.SimpleCompressor{})

	hnswCfg := vectorindex.Config{
		Dim: cfg.EmbedDim, M: cfg.HNSWM, EFSearch: cfg.HNSWEFSearch,
	}
	activeIdx := vectorindex.NewHNSW(hnswCfg)
	archIdx := vectorindex.NewHNSW(hnswCfg)

	var lastUserEmbedding []float32
	chunk := len(sc.Turns) / nTurns
	if chunk < 1 {
		chunk = 1
	}

	fmt.Println("\nturn | hist_turns | compiled | cached | cache% | prompt_stable")
	fmt.Println("-----+------------+----------+--------+--------+----------------")

	var (
		res        benchResult
		prevPrompt string
		ingested   int
	)
	res.label = label

	for i := 0; i < nTurns; i++ {
		end := (i + 1) * chunk
		if end > len(sc.Turns) {
			end = len(sc.Turns)
		}

		for j := ingested; j < end; j++ {
			t := sc.Turns[j]
			eventType := domain.EventUserPrompt
			if t.Role == "assistant" {
				eventType = domain.EventLLMResponse
			}
			event := &domain.Event{
				EventID:    fmt.Sprintf("evt_%s_%d", sessionID, j),
				SessionID:  sessionID,
				TaskID:     taskID,
				EventType:  eventType,
				Payload:    t.Content,
				TokenCount: counter.Count(t.Content),
				CreatedAt:  time.Now(),
			}
			turnNum := tree.IncrementTurn()
			event.Sequence = turnNum
			node := tree.AddNode(event)

			if node != nil {
				text := embeddingPayload(node, event)
				if text != "" {
					vec, err := embedder.Embed(ctx, text)
					if err != nil {
						log.Printf("embed turn %d: %v", j, err)
					} else {
						node.Embedding = vec
						_ = activeIdx.Insert(node.NodeID, vec)
						if t.Role == "user" {
							lastUserEmbedding = vec
						}
					}
				}
			}

			active := tree.GetActiveNodes()
			scr.ScoreAll(active, turnNum, lastUserEmbedding)
			activeTokens := 0
			for _, n := range active {
				activeTokens += n.TokenCount
			}
			if should, reason := collector.ShouldRun(tree, activeTokens, 0); should {
				preIDs := make(map[string]bool, len(active))
				for _, n := range active {
					preIDs[n.NodeID] = true
				}
				collector.Run(ctx, tree, reason, lastUserEmbedding)
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
		ingested = end

		active := tree.GetActiveNodes()
		var compiled *domain.CompiledPrompt
		qVec, err := embedder.Embed(ctx, question)
		if err != nil {
			log.Printf("compile embed: %v", err)
			compiled = comp.Compile(sessionID, taskID, question, active, systemPrompt)
		} else {
			hitsA, _ := activeIdx.Query(qVec, 12)
			hitsZ, _ := archIdx.Query(qVec, 12)
			merged := session.MergeSemanticHits(hitsA, hitsZ)
			nodes := session.NodesForSemanticCompile(tree, active, merged)
			compiled = comp.CompileWithSemantic(sessionID, taskID, question, nodes, systemPrompt, 0.20, merged)
		}

		msgs := []llm.ChatMessage{{Role: "system", Content: systemPrompt}}
		if compiled.FinalPrompt != "" {
			msgs = append(msgs, llm.ChatMessage{Role: "user", Content: compiled.FinalPrompt})
		}
		msgs = append(msgs, llm.ChatMessage{Role: "user", Content: question})

		gen, err := llmClient.Generate(ctx, msgs, 0, maxTokens)
		if err != nil {
			log.Fatalf("llm turn %d: %v", i+1, err)
		}

		stable := i > 0 && compiled.FinalPrompt == prevPrompt
		pct := 0.0
		if compiled.CompiledTokenCount > 0 {
			pct = float64(gen.CachedPromptTokens) / float64(compiled.CompiledTokenCount) * 100
		}

		fmt.Printf("%4d | %10d | %8d | %6d | %5.1f%% | %v\n",
			i+1, end, compiled.CompiledTokenCount, gen.CachedPromptTokens, pct, stable)

		res.rows = append(res.rows, rowResult{
			turn: i + 1, historyTurns: end, compiled: compiled.CompiledTokenCount,
			cached: gen.CachedPromptTokens, promptHash: hashPrompt(compiled.FinalPrompt), promptStable: stable,
		})
		res.totalCompiled += compiled.CompiledTokenCount
		res.totalCached += gen.CachedPromptTokens
		if i > 0 {
			res.turn2PlusCache += gen.CachedPromptTokens
		}
		prevPrompt = compiled.FinalPrompt
	}

	fmt.Printf(" sum |            | %8d | %6d | %5.1f%% |\n",
		res.totalCompiled, res.totalCached, pct(res.totalCached, res.totalCompiled))
	return res
}

func runFrozenBench(scenarioID string, nTurns int, stableRender bool, tokenBudget, maxTokens int, question string) benchResult {
	sc := datasets.ByID(scenarioID)
	if sc == nil {
		log.Fatalf("unknown scenario %q", scenarioID)
	}

	cfg := config.Load()
	llmClient := llm.NewClient(llm.Config{
		Provider: cfg.DefaultLLMProvider,
		BaseURL:  cfg.DefaultLLMBaseURL,
		APIKey:   cfg.DefaultLLMAPIKey,
		Model:    cfg.DefaultLLMModel,
	})

	// Re-run full ingest to rebuild identical session state.
	ctx := context.Background()
	sess, compiled, msgs := buildSessionForProbe(ctx, *sc, stableRender, tokenBudget, question, len(sc.Turns))

	label := "OFF"
	if stableRender {
		label = "ON"
	}
	fmt.Printf("mode=frozen scenario=%s stable_render=%s compiled=%d repeating=%d\n",
		scenarioID, label, compiled.CompiledTokenCount, nTurns)

	fmt.Println("\nturn | hist_turns | compiled | cached | cache% | prompt_stable")
	fmt.Println("-----+------------+----------+--------+--------+----------------")

	var (
		out       benchResult
		prevPrompt string
	)
	out.label = label
	_ = sess

	for i := 0; i < nTurns; i++ {
		gen, err := llmClient.Generate(ctx, msgs, 0, maxTokens)
		if err != nil {
			log.Fatalf("llm turn %d: %v", i+1, err)
		}
		stable := i > 0 && compiled.FinalPrompt == prevPrompt
		cpct := 0.0
		if compiled.CompiledTokenCount > 0 {
			cpct = float64(gen.CachedPromptTokens) / float64(compiled.CompiledTokenCount) * 100
		}
		fmt.Printf("%4d | %10d | %8d | %6d | %5.1f%% | %v\n",
			i+1, len(sc.Turns), compiled.CompiledTokenCount, gen.CachedPromptTokens, cpct, stable)

		out.rows = append(out.rows, rowResult{
			turn: i + 1, historyTurns: len(sc.Turns), compiled: compiled.CompiledTokenCount,
			cached: gen.CachedPromptTokens, promptHash: hashPrompt(compiled.FinalPrompt), promptStable: stable,
		})
		out.totalCompiled += compiled.CompiledTokenCount
		out.totalCached += gen.CachedPromptTokens
		if i > 0 {
			out.turn2PlusCache += gen.CachedPromptTokens
		}
		prevPrompt = compiled.FinalPrompt
	}

	fmt.Printf(" sum |            | %8d | %6d | %5.1f%% |\n",
		out.totalCompiled, out.totalCached, pct(out.totalCached, out.totalCompiled))
	return out
}

func buildSessionForProbe(ctx context.Context, sc datasets.Scenario, stableRender bool, tokenBudget int, question string, ingestTurns int) (*benchSession, *domain.CompiledPrompt, []llm.ChatMessage) {
	cfg := config.Load()
	counter := tokenizer.New(cfg.DefaultLLMModel)
	embedKey := cfg.EmbedAPIKey
	if embedKey == "" {
		embedKey = cfg.DefaultLLMAPIKey
	}
	embedder := embedding.NewOpenAI(embedding.Config{
		BaseURL: cfg.EmbedBaseURL,
		APIKey:  embedKey,
		Model:   cfg.EmbedModel,
	})

	s := newBenchSession(stableRender, tokenBudget, counter, embedder, cfg)
	for j := 0; j < ingestTurns && j < len(sc.Turns); j++ {
		s.ingestTurn(ctx, sc.Turns[j], j)
	}
	compiled, msgs := s.compileAndMessages(ctx, question)
	return s, compiled, msgs
}

type benchSession struct {
	tree              *statetree.Tree
	scr               *scorer.Scorer
	comp              *compiler.Compiler
	collector         *gc.GarbageCollector
	activeIdx, archIdx vectorindex.Index
	embedder          embedding.Provider
	counter           tokenizer.TokenCounter
}

func newBenchSession(stableRender bool, tokenBudget int, counter tokenizer.TokenCounter, embedder embedding.Provider, cfg *config.Config) *benchSession {
	tree := statetree.NewTree("cache_bench", "cache_bench")
	scr := scorer.NewScorer(15, 2000)
	scr.SetSemanticWeight(0.20)
	comp := compiler.NewCompilerWithCounter(tokenBudget, counter)
	if stableRender {
		comp.WithCacheStableRender(true)
	}
	collector := gc.NewGarbageCollector(gc.Policy{
		MaxPromptTokens: tokenBudget, MaxTreeDepth: 10, MaxChildrenPerNode: 50,
		LowRelevanceThreshold: 0.30, DecisionSweepFloor: 0.20,
		MaxActiveNodes: 0, SweepHeadroomRatio: 0, StaleAfterTurns: 15,
	}, scr, &gc.SimpleCompressor{})
	hnswCfg := vectorindex.Config{Dim: cfg.EmbedDim, M: cfg.HNSWM, EFSearch: cfg.HNSWEFSearch}
	return &benchSession{
		tree: tree, scr: scr, comp: comp, collector: collector,
		activeIdx: vectorindex.NewHNSW(hnswCfg), archIdx: vectorindex.NewHNSW(hnswCfg),
		embedder: embedder, counter: counter,
	}
}

func (s *benchSession) ingestTurn(ctx context.Context, t datasets.Turn, j int) {
	eventType := domain.EventUserPrompt
	if t.Role == "assistant" {
		eventType = domain.EventLLMResponse
	}
	event := &domain.Event{
		EventID: fmt.Sprintf("evt_%d", j), SessionID: "cache_bench", TaskID: "cache_bench",
		EventType: eventType, Payload: t.Content, TokenCount: s.counter.Count(t.Content), CreatedAt: time.Now(),
	}
	turnNum := s.tree.IncrementTurn()
	event.Sequence = turnNum
	node := s.tree.AddNode(event)
	if node == nil {
		return
	}
	text := embeddingPayload(node, event)
	if text == "" {
		return
	}
	vec, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return
	}
	node.Embedding = vec
	_ = s.activeIdx.Insert(node.NodeID, vec)
	var lastUser []float32
	if t.Role == "user" {
		lastUser = vec
	}
	active := s.tree.GetActiveNodes()
	s.scr.ScoreAll(active, turnNum, lastUser)
	activeTokens := 0
	for _, n := range active {
		activeTokens += n.TokenCount
	}
	if should, reason := s.collector.ShouldRun(s.tree, activeTokens, 0); should {
		s.collector.Run(ctx, s.tree, reason, lastUser)
	}
}

func (s *benchSession) compileAndMessages(ctx context.Context, question string) (*domain.CompiledPrompt, []llm.ChatMessage) {
	active := s.tree.GetActiveNodes()
	qVec, err := s.embedder.Embed(ctx, question)
	var compiled *domain.CompiledPrompt
	if err != nil {
		compiled = s.comp.Compile("cache_bench", "cache_bench", question, active, systemPrompt)
	} else {
		hitsA, _ := s.activeIdx.Query(qVec, 12)
		hitsZ, _ := s.archIdx.Query(qVec, 12)
		merged := session.MergeSemanticHits(hitsA, hitsZ)
		nodes := session.NodesForSemanticCompile(s.tree, active, merged)
		compiled = s.comp.CompileWithSemantic("cache_bench", "cache_bench", question, nodes, systemPrompt, 0.20, merged)
	}
	msgs := []llm.ChatMessage{{Role: "system", Content: systemPrompt}}
	if compiled.FinalPrompt != "" {
		msgs = append(msgs, llm.ChatMessage{Role: "user", Content: compiled.FinalPrompt})
	}
	msgs = append(msgs, llm.ChatMessage{Role: "user", Content: question})
	return compiled, msgs
}

func printCompareSummary(off, on benchResult) {
	fmt.Println("\n========== COMPARISON (turns 2-5 cache hits) ==========")
	fmt.Printf("stable_render=OFF: turn2+ cached=%d  total cached=%d  total compiled=%d\n",
		off.turn2PlusCache, off.totalCached, off.totalCompiled)
	fmt.Printf("stable_render=ON:  turn2+ cached=%d  total cached=%d  total compiled=%d\n",
		on.turn2PlusCache, on.totalCached, on.totalCompiled)

	offStable := countStablePrompts(off)
	onStable := countStablePrompts(on)
	fmt.Printf("prompt byte-stable turn-to-turn: OFF=%d/%d  ON=%d/%d (turns 2-5)\n",
		offStable, len(off.rows)-1, onStable, len(on.rows)-1)

	delta := on.turn2PlusCache - off.turn2PlusCache
	fmt.Printf("cache delta (ON - OFF) turn2+: %+d tokens\n", delta)
}

func countStablePrompts(r benchResult) int {
	n := 0
	for _, row := range r.rows {
		if row.turn > 1 && row.promptStable {
			n++
		}
	}
	return n
}

func pct(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den) * 100
}

func hashPrompt(s string) string {
	if s == "" {
		return "empty"
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

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
