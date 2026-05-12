package session

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chandrashekhartata/acgc/internal/compiler"
	"github.com/chandrashekhartata/acgc/internal/domain"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/statetree"
	"github.com/chandrashekhartata/acgc/internal/store"
)

type SessionState struct {
	Tree          *statetree.Tree
	Metrics       *domain.SessionMetrics
	EventChan     chan *domain.Event
	LastActivity  time.Time
	Cancel        context.CancelFunc
}

type Manager struct {
	mu             sync.RWMutex
	sessions       map[string]*SessionState
	store          *store.MongoStore
	scorer         *scorer.Scorer
	gc             *gc.GarbageCollector
	compilerBudget int
	channelBuffer  int
	idleTimeout    time.Duration
	snapshotEvery  time.Duration
}

type ManagerConfig struct {
	Store          *store.MongoStore
	Scorer         *scorer.Scorer
	GC             *gc.GarbageCollector
	TokenBudget    int
	ChannelBuffer  int
	IdleTimeoutS   int
	SnapshotEveryS int
}

func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		sessions:       make(map[string]*SessionState),
		store:          cfg.Store,
		scorer:         cfg.Scorer,
		gc:             cfg.GC,
		compilerBudget: cfg.TokenBudget,
		channelBuffer:  cfg.ChannelBuffer,
		idleTimeout:    time.Duration(cfg.IdleTimeoutS) * time.Second,
		snapshotEvery:  time.Duration(cfg.SnapshotEveryS) * time.Second,
	}
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
	}

	m.sessions[sessionID] = state

	// Launch the single-writer worker for this session
	go m.sessionWorker(workerCtx, sessionID, state)

	return state
}

// CompilePrompt reads the current state and builds an optimized prompt (sync, fast path).
func (m *Manager) CompilePrompt(sessionID, taskID, userMessage, systemPrompt string) *domain.CompiledPrompt {
	m.mu.RLock()
	state, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		// No session state — pass through with just the user message
		return &domain.CompiledPrompt{
			CompiledPromptID:   fmt.Sprintf("cp_%d", time.Now().UnixNano()),
			SessionID:          sessionID,
			TaskID:             taskID,
			CurrentUserMessage: userMessage,
			FinalPrompt:        systemPrompt + "\n\n" + userMessage,
			OriginalTokenCount: estimateTokens(systemPrompt + userMessage),
			CompiledTokenCount: estimateTokens(systemPrompt + userMessage),
			CreatedAt:          time.Now(),
		}
	}

	activeNodes := state.Tree.GetActiveNodes()

	// Use pre-computed scores — don't re-score on the hot path
	comp := compiler.NewCompiler(m.compilerBudget)
	return comp.Compile(sessionID, taskID, userMessage, activeNodes, systemPrompt)
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

	// 3. Persist the new node durably to MongoDB
	if err := m.store.UpsertNode(ctx, node); err != nil {
		log.Printf("session: failed to persist node %s: %v", node.NodeID, err)
	}

	// 4. Re-score active nodes
	activeNodes := state.Tree.GetActiveNodes()
	m.scorer.ScoreAll(activeNodes, turn)

	// 5. Update metrics
	state.Metrics.TotalEvents++
	if event.EventType == domain.EventUserPrompt {
		state.Metrics.TotalTurns++
	}

	// 6. Check if GC should run
	estimatedTokens := 0
	for _, n := range activeNodes {
		estimatedTokens += n.TokenCount
	}
	if shouldRun, reason := m.gc.ShouldRun(state.Tree, estimatedTokens); shouldRun {
		gcStart := time.Now()
		result := m.gc.Run(ctx, state.Tree, reason)
		gcDuration := time.Since(gcStart)

		state.Metrics.GCRuns++
		state.Metrics.TotalTokensSaved += result.TokensFreed
		state.Metrics.BranchesCompressed += result.BranchesCompressed

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

func estimateTokens(s string) int {
	return len(s) / 4
}
