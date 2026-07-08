package session

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shekhartata/acgcProject/internal/compiler"
	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/embedding"
	"github.com/shekhartata/acgcProject/internal/gc"
	"github.com/shekhartata/acgcProject/internal/scorer"
	"github.com/shekhartata/acgcProject/internal/statetree"
	"github.com/shekhartata/acgcProject/internal/store"
	"github.com/shekhartata/acgcProject/internal/tokenizer"
	"github.com/shekhartata/acgcProject/internal/vectorindex"
)

type SessionState struct {
	Tree         *statetree.Tree
	Metrics      *domain.SessionMetrics
	EventChan    chan *domain.Event
	LastActivity time.Time
	Cancel       context.CancelFunc

	// Semantic scoring per-session state. ActiveIndex serves live active-set
	// lookups; ArchiveIndex retains vectors for archived nodes swept off the
	// active index. Either is non-nil iff the manager has an embedder.
	ActiveIndex       vectorindex.Index
	ArchiveIndex      vectorindex.Index
	LastUserEmbedding []float32

	// TokenBudget is the effective compile/GC cap for this session. Seeded from
	// the server default; updated when Run sends token_budget > 0.
	TokenBudget int
}

type Manager struct {
	mu             sync.RWMutex
	sessions       map[string]*SessionState
	store          *store.MongoStore
	scorer         *scorer.Scorer
	gc             *gc.GarbageCollector
	defaultBudget  int
	tokenCounter   tokenizer.TokenCounter
	channelBuffer  int
	idleTimeout    time.Duration
	snapshotEvery  time.Duration

	embedder       embedding.Provider
	semanticWeight float64
	topKAtCompile  int
	archiveTopK    int
	hnswConfig     vectorindex.Config

	latencyBreakdown  bool
	cacheStableRender bool
}

type ManagerConfig struct {
	Store       *store.MongoStore
	Scorer      *scorer.Scorer
	GC          *gc.GarbageCollector
	TokenBudget int
	// TokenCounter is used for prompt token accounting. When nil, the
	// process-wide default (real model-aware tokenizer) is used.
	TokenCounter   tokenizer.TokenCounter
	ChannelBuffer  int
	IdleTimeoutS   int
	SnapshotEveryS int

	// Optional. When Embedder is nil, all semantic ops are skipped and the
	// system behaves identically to the pre-semantic version.
	Embedder             embedding.Provider
	SemanticWeight       float64
	TopKAtCompile        int
	ArchiveTopKAtCompile int
	HNSWConfig           vectorindex.Config

	LatencyBreakdown  bool
	CacheStableRender bool
}

func NewManager(cfg ManagerConfig) *Manager {
	topK := cfg.TopKAtCompile
	if topK <= 0 {
		topK = 12
	}
	archK := cfg.ArchiveTopKAtCompile
	if archK <= 0 {
		archK = 12
	}
	tc := cfg.TokenCounter
	if tc == nil {
		tc = tokenizer.Default()
	}
	return &Manager{
		sessions:         make(map[string]*SessionState),
		store:            cfg.Store,
		scorer:           cfg.Scorer,
		gc:                cfg.GC,
		defaultBudget:     cfg.TokenBudget,
		tokenCounter:      tc,
		channelBuffer:    cfg.ChannelBuffer,
		idleTimeout:      time.Duration(cfg.IdleTimeoutS) * time.Second,
		snapshotEvery:    time.Duration(cfg.SnapshotEveryS) * time.Second,
		embedder:         cfg.Embedder,
		semanticWeight:   cfg.SemanticWeight,
		topKAtCompile:    topK,
		archiveTopK:      archK,
		hnswConfig:       cfg.HNSWConfig,
		latencyBreakdown:  cfg.LatencyBreakdown,
		cacheStableRender: cfg.CacheStableRender,
	}
}

func (m *Manager) newCompiler(budget int) *compiler.Compiler {
	comp := compiler.NewCompilerWithCounter(budget, m.tokenCounter)
	if m.cacheStableRender {
		comp.WithCacheStableRender(true)
	}
	return comp
}

// effectiveBudget resolves the compile/GC token cap. requestBudget > 0 updates
// the session and wins; otherwise the session's stored budget or server default.
func (m *Manager) effectiveBudget(state *SessionState, requestBudget int) int {
	if requestBudget > 0 {
		state.TokenBudget = requestBudget
		return requestBudget
	}
	if state != nil && state.TokenBudget > 0 {
		return state.TokenBudget
	}
	return m.defaultBudget
}

func (m *Manager) semanticEnabled() bool {
	return m.embedder != nil
}

// GetOrCreate returns an existing session or creates a new one.
func (m *Manager) GetOrCreate(ctx context.Context, sessionID, taskID string) *SessionState {
	m.mu.RLock()
	if s, ok := m.sessions[sessionID]; ok {
		m.mu.RUnlock()
		s.LastActivity = time.Now()
		return s
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if s, ok := m.sessions[sessionID]; ok {
		s.LastActivity = time.Now()
		return s
	}

	// Try restoring from snapshot
	var tree *statetree.Tree
	snapshot, err := m.store.LoadLatestSnapshot(ctx, sessionID)
	if err != nil {
		log.Printf("session: failed to load snapshot for %s: %v", sessionID, err)
	}
	if len(snapshot) > 0 {
		tree = statetree.RestoreFromSnapshot(sessionID, taskID, snapshot)
		log.Printf("session: restored %s from snapshot (%d nodes)", sessionID, len(snapshot))
	} else {
		tree = statetree.NewTree(sessionID, taskID)
	}

	workerCtx, cancel := context.WithCancel(ctx)
	state := &SessionState{
		Tree: tree,
		Metrics: &domain.SessionMetrics{
			SessionID:        sessionID,
			SessionStartedAt: time.Now(),
		},
		EventChan:    make(chan *domain.Event, m.channelBuffer),
		LastActivity: time.Now(),
		Cancel:       cancel,
		TokenBudget:  m.defaultBudget,
	}

	// Build the per-session HNSW index if semantic scoring is enabled.
	// On rehydration, prime it with whatever embeddings Mongo already has.
	if m.semanticEnabled() {
		state.ActiveIndex = vectorindex.NewHNSW(m.hnswConfig)
		state.ArchiveIndex = vectorindex.NewHNSW(m.hnswConfig)
		if vecs, err := m.store.LoadNodeEmbeddings(ctx, sessionID); err != nil {
			log.Printf("session: failed to load active embeddings for %s: %v", sessionID, err)
		} else if len(vecs) > 0 {
			if err := state.ActiveIndex.RebuildFromVectors(vecs); err != nil {
				log.Printf("session: failed to rebuild active HNSW for %s: %v", sessionID, err)
			} else {
				log.Printf("session: rehydrated active HNSW for %s (%d vectors)", sessionID, len(vecs))
			}
		}
		if avecs, err := m.store.LoadArchivedNodeEmbeddings(ctx, sessionID); err != nil {
			log.Printf("session: failed to load archived embeddings for %s: %v", sessionID, err)
		} else if len(avecs) > 0 {
			if err := state.ArchiveIndex.RebuildFromVectors(avecs); err != nil {
				log.Printf("session: failed to rebuild archive HNSW for %s: %v", sessionID, err)
			} else {
				log.Printf("session: rehydrated archive HNSW for %s (%d vectors)", sessionID, len(avecs))
			}
		}
	}

	m.sessions[sessionID] = state

	go m.sessionWorker(workerCtx, sessionID, state)

	return state
}

// CompilePrompt reads the current state and builds an optimized prompt (sync, fast path).
//
// When semantic scoring is enabled, the imminent user message is embedded and
// queried against the per-session ActiveIndex and ArchiveIndex. The compiler
// re-blends scores in a single pass, boosting nodes semantically close to the
// current query — including archived nodes when they match.
func (m *Manager) CompilePrompt(sessionID, taskID, userMessage, systemPrompt string, requestBudget int) *domain.CompiledPrompt {
	if m.latencyBreakdown {
		return m.compilePromptMeasured(sessionID, taskID, userMessage, systemPrompt, requestBudget)
	}
	m.mu.Lock()
	state, ok := m.sessions[sessionID]
	var budget int
	if ok {
		budget = m.effectiveBudget(state, requestBudget)
	}
	m.mu.Unlock()

	if !ok {
		// Session-not-found fallback: keep system + probe on the side so
		// callers (gateway, eval) emit them as their own chat messages.
		// FinalPrompt is empty because there's no context to assemble — the
		// gateway will skip the body user message and just send the probe.
		return &domain.CompiledPrompt{
			CompiledPromptID:   fmt.Sprintf("cp_%d", time.Now().UnixNano()),
			SessionID:          sessionID,
			TaskID:             taskID,
			CurrentUserMessage: userMessage,
			SystemPrompt:       systemPrompt,
			FinalPrompt:        "",
			OriginalTokenCount: m.tokenCounter.Count(systemPrompt) + m.tokenCounter.Count(userMessage),
			CompiledTokenCount: m.tokenCounter.Count(systemPrompt) + m.tokenCounter.Count(userMessage),
			CreatedAt:          time.Now(),
		}
	}

	activeNodes := state.Tree.GetActiveNodes()
	comp := m.newCompiler(budget)

	// Heuristic-only fast path: no embedder, no index, or no user message.
	if !m.semanticEnabled() || state.ActiveIndex == nil || state.ArchiveIndex == nil || userMessage == "" {
		return comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
	}

	// Per-compile semantic re-blend. On any failure, fall back to the
	// heuristic path so the hot path always returns a prompt.
	embedCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queryVec, err := m.embedder.Embed(embedCtx, userMessage)
	if err != nil {
		log.Printf("session: compile-time embed failed for %s: %v", sessionID, err)
		return comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
	}

	ka := m.topKAtCompile
	kz := m.archiveTopK
	hitsA, err := state.ActiveIndex.Query(queryVec, ka)
	if err != nil {
		log.Printf("session: active HNSW query failed for %s: %v", sessionID, err)
		return comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
	}
	hitsZ, err := state.ArchiveIndex.Query(queryVec, kz)
	if err != nil {
		log.Printf("session: archive HNSW query failed for %s: %v", sessionID, err)
		return comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
	}
	merged := MergeSemanticHits(hitsA, hitsZ)
	compileNodes := NodesForSemanticCompile(state.Tree, activeNodes, merged)

	return comp.CompileWithSemantic(sessionID, taskID, userMessage, compileNodes,
		systemPrompt, m.semanticWeight, merged)
}

func (m *Manager) compilePromptMeasured(sessionID, taskID, userMessage, systemPrompt string, requestBudget int) *domain.CompiledPrompt {
	compileOuter := time.Now()

	m.mu.Lock()
	state, ok := m.sessions[sessionID]
	var budget int
	if ok {
		budget = m.effectiveBudget(state, requestBudget)
	}
	m.mu.Unlock()

	if !ok {
		return &domain.CompiledPrompt{
			CompiledPromptID:   fmt.Sprintf("cp_%d", time.Now().UnixNano()),
			SessionID:          sessionID,
			TaskID:             taskID,
			CurrentUserMessage: userMessage,
			SystemPrompt:       systemPrompt,
			FinalPrompt:        "",
			OriginalTokenCount: m.tokenCounter.Count(systemPrompt) + m.tokenCounter.Count(userMessage),
			CompiledTokenCount: m.tokenCounter.Count(systemPrompt) + m.tokenCounter.Count(userMessage),
			CreatedAt:          time.Now(),
		}
	}

	bd := &domain.CompileLatencyBreakdown{}

	activeNodes := state.Tree.GetActiveNodes()
	comp := m.newCompiler(budget)

	if !m.semanticEnabled() || state.ActiveIndex == nil || state.ArchiveIndex == nil || userMessage == "" {
		tAsm := time.Now()
		cp := comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
		bd.CompileAssemblyMs = int32(time.Since(tAsm).Milliseconds())
		bd.SemanticFallback = true
		bd.CompileTotalMs = int32(time.Since(compileOuter).Milliseconds())
		cp.LatencyBreakdown = bd
		return cp
	}

	embedCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tEmb := time.Now()
	queryVec, err := m.embedder.Embed(embedCtx, userMessage)
	bd.CompileEmbedMs = int32(time.Since(tEmb).Milliseconds())
	if err != nil {
		log.Printf("session: compile-time embed failed for %s: %v", sessionID, err)
		tAsm := time.Now()
		cp := comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
		bd.CompileAssemblyMs = int32(time.Since(tAsm).Milliseconds())
		bd.SemanticFallback = true
		bd.CompileTotalMs = int32(time.Since(compileOuter).Milliseconds())
		cp.LatencyBreakdown = bd
		return cp
	}

	ka := m.topKAtCompile
	kz := m.archiveTopK
	tIdx := time.Now()
	hitsA, err := state.ActiveIndex.Query(queryVec, ka)
	if err != nil {
		log.Printf("session: active HNSW query failed for %s: %v", sessionID, err)
		bd.CompileIndexMs = int32(time.Since(tIdx).Milliseconds())
		tAsm := time.Now()
		cp := comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
		bd.CompileAssemblyMs = int32(time.Since(tAsm).Milliseconds())
		bd.SemanticFallback = true
		bd.CompileTotalMs = int32(time.Since(compileOuter).Milliseconds())
		cp.LatencyBreakdown = bd
		return cp
	}
	hitsZ, err := state.ArchiveIndex.Query(queryVec, kz)
	bd.CompileIndexMs = int32(time.Since(tIdx).Milliseconds())
	if err != nil {
		log.Printf("session: archive HNSW query failed for %s: %v", sessionID, err)
		tAsm := time.Now()
		cp := comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
		bd.CompileAssemblyMs = int32(time.Since(tAsm).Milliseconds())
		bd.SemanticFallback = true
		bd.CompileTotalMs = int32(time.Since(compileOuter).Milliseconds())
		cp.LatencyBreakdown = bd
		return cp
	}
	merged := MergeSemanticHits(hitsA, hitsZ)
	compileNodes := NodesForSemanticCompile(state.Tree, activeNodes, merged)

	tAsm := time.Now()
	cp := comp.CompileWithSemantic(sessionID, taskID, userMessage, compileNodes,
		systemPrompt, m.semanticWeight, merged)
	bd.CompileAssemblyMs = int32(time.Since(tAsm).Milliseconds())
	bd.SemanticFallback = false
	bd.CompileTotalMs = int32(time.Since(compileOuter).Milliseconds())
	cp.LatencyBreakdown = bd
	return cp
}

// EnqueueEvent sends an event to the session's async worker (non-blocking).
func (m *Manager) EnqueueEvent(sessionID string, event *domain.Event) bool {
	m.mu.RLock()
	state, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case state.EventChan <- event:
		return true
	default:
		// Channel full — backpressure. Don't block the gateway.
		log.Printf("session: event channel full for %s, dropping event %s", sessionID, event.EventID)
		return false
	}
}

// sessionWorker is the single-writer goroutine for a session.
// It processes events in order, updates the tree, scores, and triggers GC.
func (m *Manager) sessionWorker(ctx context.Context, sessionID string, state *SessionState) {
	snapshotTicker := time.NewTicker(m.snapshotEvery)
	defer snapshotTicker.Stop()

	idleCheck := time.NewTicker(30 * time.Second)
	defer idleCheck.Stop()

	for {
		select {
		case <-ctx.Done():
			m.snapshotSession(sessionID, state)
			return

		case event := <-state.EventChan:
			m.processEvent(ctx, sessionID, state, event)

		case <-snapshotTicker.C:
			m.snapshotSession(sessionID, state)

		case <-idleCheck.C:
			if time.Since(state.LastActivity) > m.idleTimeout {
				log.Printf("session: %s idle, shutting down worker", sessionID)
				m.snapshotSession(sessionID, state)
				m.mu.Lock()
				delete(m.sessions, sessionID)
				m.mu.Unlock()
				state.Cancel()
				return
			}
		}
	}
}

func (m *Manager) processEvent(_ context.Context, sessionID string, state *SessionState, event *domain.Event) {
	// Use an independent context for all MongoDB operations.
	// The original gRPC request context may already be cancelled by the time
	// the async worker processes the event.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Persist raw event to MongoDB
	if err := m.store.AppendEvent(ctx, event); err != nil {
		log.Printf("session: failed to persist event %s: %v", event.EventID, err)
	}

	// 2. Update in-memory state tree
	turn := state.Tree.IncrementTurn()
	event.Sequence = turn
	node := state.Tree.AddNode(event)

	// 3. Embed the node payload (best-effort) and insert into the per-session
	// HNSW index. Done synchronously within the single-writer worker to
	// preserve ordering — embedding latency (50–150ms) is acceptable here
	// because this path is async w.r.t. the gateway.
	if m.semanticEnabled() && state.ActiveIndex != nil {
		text := payloadForEmbedding(node, event)
		if text != "" {
			vec, err := m.embedder.Embed(ctx, text)
			if err != nil {
				log.Printf("session: embed failed for node %s: %v", node.NodeID, err)
			} else {
				node.Embedding = vec
				node.EmbedModel = m.embedder.Model()
				if insErr := state.ActiveIndex.Insert(node.NodeID, vec); insErr != nil {
					log.Printf("session: HNSW insert failed for %s: %v", node.NodeID, insErr)
				}
				if event.EventType == domain.EventUserPrompt {
					state.LastUserEmbedding = vec
				}
			}
		}
	}

	// 4. Persist the (possibly embedding-bearing) node durably
	if err := m.store.UpsertNode(ctx, node); err != nil {
		log.Printf("session: failed to persist node %s: %v", node.NodeID, err)
	}

	// 5. Re-score active nodes against the current semantic anchor
	activeNodes := state.Tree.GetActiveNodes()
	m.scorer.ScoreAll(activeNodes, turn, state.LastUserEmbedding)

	// 6. Update metrics
	state.Metrics.TotalEvents++
	if event.EventType == domain.EventUserPrompt {
		state.Metrics.TotalTurns++
	}

	// 7. Check if GC should run
	estimatedTokens := 0
	for _, n := range activeNodes {
		estimatedTokens += n.TokenCount
	}
	if shouldRun, reason := m.gc.ShouldRun(state.Tree, estimatedTokens, state.TokenBudget); shouldRun {
		// Capture pre-GC active set so we can diff against post-GC to
		// figure out which IDs were swept/archived (and remove them from
		// the HNSW index).
		preIDs := make(map[string]bool, len(activeNodes))
		for _, n := range activeNodes {
			preIDs[n.NodeID] = true
		}

		gcStart := time.Now()
		result := m.gc.Run(ctx, state.Tree, reason, state.LastUserEmbedding)
		gcDuration := time.Since(gcStart)

		state.Metrics.GCRuns++
		state.Metrics.TotalTokensSaved += result.TokensFreed
		state.Metrics.BranchesCompressed += result.BranchesCompressed

		// Index hygiene: embeddings for nodes that dropped out of the active set move
		// to the archive index before removal from the active HNSW graph.
		if m.semanticEnabled() && state.ActiveIndex != nil && state.ArchiveIndex != nil {
			reconcileSemanticIndices(state, state.Tree, preIDs)
		}

		// Persist swept node status changes to MongoDB
		if result.NodesSwept > 0 || result.BranchesCompressed > 0 {
			allNodes := state.Tree.GetAllNodes()
			if err := m.store.UpsertNodes(ctx, allNodes); err != nil {
				log.Printf("session: failed to persist GC node changes: %v", err)
			}
		}

		// Persist compressed branches durably
		for _, cb := range result.CompressedBranchRecords {
			if err := m.store.SaveCompressedBranch(ctx, cb); err != nil {
				log.Printf("session: failed to persist compressed branch: %v", err)
			}
		}

		// Log GC with duration
		m.store.LogGC(ctx, &store.GCLog{
			SessionID:          sessionID,
			TriggerReason:      string(reason),
			NodesSwept:         result.NodesSwept,
			BranchesCompressed: result.BranchesCompressed,
			TokensFreed:        result.TokensFreed,
			DurationMs:         float64(gcDuration.Milliseconds()),
			CreatedAt:          time.Now(),
		})

		log.Printf("session: GC ran for %s — reason: %s, swept: %d, compressed: %d, freed: %d tokens, took: %dms",
			sessionID, reason, result.NodesSwept, result.BranchesCompressed, result.TokensFreed, gcDuration.Milliseconds())
	}

	// 7. Periodically persist session metrics (every 5 events to avoid write amplification)
	if state.Metrics.TotalEvents%5 == 0 {
		if err := m.store.UpsertSessionMetrics(ctx, state.Metrics); err != nil {
			log.Printf("session: failed to persist metrics for %s: %v", sessionID, err)
		}
	}
}

func (m *Manager) snapshotSession(sessionID string, state *SessionState) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	nodes := state.Tree.GetAllNodes()

	// 1. Save full snapshot for crash recovery
	if err := m.store.SnapshotNodes(ctx, sessionID, nodes); err != nil {
		log.Printf("session: snapshot failed for %s: %v", sessionID, err)
	}

	// 2. Upsert all nodes durably (individual node records)
	if err := m.store.UpsertNodes(ctx, nodes); err != nil {
		log.Printf("session: node upsert failed for %s: %v", sessionID, err)
	}

	// 3. Persist session metrics
	if err := m.store.UpsertSessionMetrics(ctx, state.Metrics); err != nil {
		log.Printf("session: metrics persist failed for %s: %v", sessionID, err)
	}

	log.Printf("session: snapshot saved for %s (%d nodes)", sessionID, len(nodes))
}

// GetMetrics returns session metrics (read-safe).
func (m *Manager) GetMetrics(sessionID string) (*domain.SessionMetrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	return state.Metrics, true
}

// GetTreeStats returns tree statistics for a session.
func (m *Manager) GetTreeStats(sessionID string) (total, active, compressed, archived, maxDepth, maxWidth int, ok bool) {
	m.mu.RLock()
	state, exists := m.sessions[sessionID]
	m.mu.RUnlock()
	if !exists {
		return 0, 0, 0, 0, 0, 0, false
	}
	total, active, compressed, archived, maxDepth, maxWidth = state.Tree.Stats()
	ok = true
	return
}

func reconcileSemanticIndices(state *SessionState, tree *statetree.Tree, preActiveIDs map[string]bool) {
	if state == nil || state.ActiveIndex == nil || state.ArchiveIndex == nil {
		return
	}
	postActive := tree.GetActiveNodes()
	postIDs := make(map[string]bool, len(postActive))
	for _, n := range postActive {
		postIDs[n.NodeID] = true
	}
	for id := range preActiveIDs {
		if postIDs[id] {
			continue
		}
		if n, ok := tree.GetNode(id); ok && len(n.Embedding) > 0 {
			_ = state.ArchiveIndex.Insert(id, n.Embedding)
		}
		state.ActiveIndex.Delete(id)
	}
}

// payloadForEmbedding picks the best text to send to the embedding API for a
// given node + event. Prefers the cleaned-up node summary/title (low-noise),
// falling back to the raw event payload when the tree hasn't extracted a
// summary yet (early in a turn, or for tool/system events).
func payloadForEmbedding(node *domain.StateNode, event *domain.Event) string {
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
