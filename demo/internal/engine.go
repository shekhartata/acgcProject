package demo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shekhartata/acgcProject/internal/llm"
	"github.com/shekhartata/acgcProject/internal/tokenizer"
	"github.com/shekhartata/acgcProject/pkg/acgc"
)

// EngineConfig wires LLM + ACGC sidecar for the demo.
type EngineConfig struct {
	ACGCAddr     string
	TokenBudget  int
	LLMProvider  string
	LLMBaseURL   string
	LLMAPIKey    string
	LLMModel     string
	MaxTokens    int
}

// Engine holds in-memory demo sessions.
type Engine struct {
	cfg      EngineConfig
	scenario Scenario
	mu       sync.Mutex
	sessions map[string]*Session
}

// Session is one dual-pane demo run.
type Session struct {
	ID           string
	Budget       int
	Model        string
	Scenario     Scenario
	cursor       int // index into Scenario.Turns
	userStepsDone int
	seeded       bool
	naive        *naivePane
	acgc         *acgcPane
	probed       bool
}

// NewEngine creates a demo engine.
func NewEngine(cfg EngineConfig) *Engine {
	if cfg.TokenBudget <= 0 {
		cfg.TokenBudget = 6000
	}
	if cfg.ACGCAddr == "" {
		cfg.ACGCAddr = "localhost:50051"
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 512
	}
	return &Engine{
		cfg:      cfg,
		scenario: LoadScenario(),
		sessions: make(map[string]*Session),
	}
}

// Start creates a new dual-pane session.
func (e *Engine) Start(ctx context.Context, req StartRequest) (*StartResponse, error) {
	budget := req.TokenBudget
	if budget <= 0 {
		budget = e.cfg.TokenBudget
	}
	addr := req.ACGCAddr
	if addr == "" {
		addr = e.cfg.ACGCAddr
	}

	counter := tokenizer.New(e.cfg.LLMModel)
	llmClient := llm.NewClient(llm.Config{
		Provider: e.cfg.LLMProvider,
		BaseURL:  e.cfg.LLMBaseURL,
		APIKey:   e.cfg.LLMAPIKey,
		Model:    e.cfg.LLMModel,
	})

	sessionID := fmt.Sprintf("demo_%d", time.Now().UnixNano())
	rt, err := acgc.NewContextRuntime(acgc.Config{
		ServerAddr:  addr,
		SessionID:   sessionID,
		TaskID:      "marketing-demo",
		TokenBudget: budget,
		LLM: acgc.LLMConfig{
			Provider:    e.cfg.LLMProvider,
			Model:       e.cfg.LLMModel,
			APIKey:      e.cfg.LLMAPIKey,
			BaseURL:     e.cfg.LLMBaseURL,
			Temperature: 0.3,
			MaxTokens:   e.cfg.MaxTokens,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("connect to ACGC at %s: %w", addr, err)
	}

	sc := e.scenario
	sess := &Session{
		ID:       sessionID,
		Budget:   budget,
		Model:    e.cfg.LLMModel,
		Scenario: sc,
		naive:    newNaivePane(llmClient, counter, systemPrompt, budget),
		acgc:     newACGCPane(rt, budget, systemPrompt),
	}

	if err := sess.ensureSeeded(ctx); err != nil {
		rt.Close()
		return nil, fmt.Errorf("seed ACGC session (is ./bin/acgc running?): %w", err)
	}

	e.mu.Lock()
	e.sessions[sessionID] = sess
	e.mu.Unlock()

	return &StartResponse{
		SessionID:       sessionID,
		TurnsPlanned:    sc.WarmUserSteps,
		WarmUserSteps:   sc.WarmUserSteps,
		Budget:          budget,
		Model:           e.cfg.LLMModel,
		Subtitle:        "Same conversation. Same budget. Different context strategy.",
		ScenarioID:      sc.ID,
		ScenarioName:    sc.Name,
		NaiveTranscript: append([]ChatLine{}, sess.naive.transcript...),
		ACGCTranscript:  append([]ChatLine{}, sess.acgc.transcript...),
	}, nil
}

func (e *Engine) get(id string) (*Session, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s, ok := e.sessions[id]
	if !ok {
		return nil, fmt.Errorf("unknown session %q", id)
	}
	return s, nil
}

// Next advances one scripted user turn through both panes.
func (e *Engine) Next(ctx context.Context, req NextRequest) (*NextResponse, error) {
	s, err := e.get(req.SessionID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureSeeded(ctx); err != nil {
		return nil, err
	}

	userMsg, ok := s.nextUserMessage()
	if !ok {
		return &NextResponse{
			TurnIndex:     s.userStepsDone,
			Done:          true,
			WarmRemaining: 0,
			Naive:         PaneTurn{Transcript: append([]ChatLine{}, s.naive.transcript...)},
			ACGC:          PaneTurn{Transcript: append([]ChatLine{}, s.acgc.transcript...)},
		}, nil
	}

	var (
		naiveAsst string
		acgcAsst  string
		nStats    NaiveStats
		aStats    ACGCStats
		wg        sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		naiveAsst, nStats = s.naive.run(ctx, userMsg)
	}()
	go func() {
		defer wg.Done()
		acgcAsst, aStats = s.acgc.run(ctx, userMsg)
	}()
	wg.Wait()

	s.userStepsDone++
	remaining := s.Scenario.WarmUserSteps - s.userStepsDone
	if remaining < 0 {
		remaining = 0
	}

	return &NextResponse{
		TurnIndex:     s.userStepsDone,
		UserMessage:   userMsg,
		Done:          remaining == 0,
		WarmRemaining: remaining,
		Naive: PaneTurn{
			Assistant:  naiveAsst,
			Transcript: append([]ChatLine{}, s.naive.transcript...),
			NaiveStats: &nStats,
		},
		ACGC: PaneTurn{
			Assistant: acgcAsst,
			Transcript: append([]ChatLine{}, s.acgc.transcript...),
			ACGCStats:  &aStats,
		},
	}, nil
}

// Probe asks the recall question on both panes.
func (e *Engine) Probe(ctx context.Context, req ProbeRequest) (*ProbeResponse, error) {
	s, err := e.get(req.SessionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSeeded(ctx); err != nil {
		return nil, err
	}
	if s.userStepsDone < s.Scenario.WarmUserSteps {
		return nil, fmt.Errorf("not_ready: finish warm turns first (%d/%d)", s.userStepsDone, s.Scenario.WarmUserSteps)
	}

	q := s.Scenario.Probe.Question
	var (
		naiveAsst string
		acgcAsst  string
		nStats    NaiveStats
		aStats    ACGCStats
		wg        sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		naiveAsst, nStats = s.naive.run(ctx, q)
	}()
	go func() {
		defer wg.Done()
		acgcAsst, aStats = s.acgc.run(ctx, q)
	}()
	wg.Wait()
	s.probed = true

	nHit := HitNeedle(naiveAsst, s.Scenario.Probe.ExpectedAny)
	aHit := HitNeedle(acgcAsst, s.Scenario.Probe.ExpectedAny)

	return &ProbeResponse{
		Question: q,
		Naive: ProbePane{
			Answer:    naiveAsst,
			HitNeedle: nHit,
			Error:     nStats.Error,
			Stats:     nStats,
		},
		ACGC: ProbePane{
			Answer:    acgcAsst,
			HitNeedle: aHit,
			Error:     aStats.Error,
			Stats:     aStats,
		},
		Takeaway: Takeaway(nHit, aHit),
	}, nil
}

// Reset removes a session.
func (e *Engine) Reset(req ResetRequest) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if s, ok := e.sessions[req.SessionID]; ok {
		s.acgc.close()
		delete(e.sessions, req.SessionID)
	}
	return nil
}

// ensureSeeded ingests kickoff + decision Q/A into both panes without live LLM calls.
func (s *Session) ensureSeeded(ctx context.Context) error {
	if s.seeded {
		return nil
	}
	// Seed until just before the first filler user turn (after decision block).
	// Decision block is first 10 turns; seed all of them.
	const seedUntil = 10
	limit := seedUntil
	if limit > len(s.Scenario.Turns) {
		limit = len(s.Scenario.Turns)
	}
	for i := 0; i < limit; i++ {
		t := s.Scenario.Turns[i]
		s.naive.seed(t.Role, t.Content)
		if err := s.acgc.seed(ctx, t.Role, t.Content); err != nil {
			return fmt.Errorf("acgc seed turn %d: %w", i, err)
		}
	}
	s.cursor = limit
	s.seeded = true
	// Count seeded user turns toward warm progress display? No — warm steps
	// are live Next() calls on remaining user turns only.
	return nil
}

func (s *Session) nextUserMessage() (string, bool) {
	for s.cursor < len(s.Scenario.Turns) {
		t := s.Scenario.Turns[s.cursor]
		s.cursor++
		if t.Role == "user" {
			// Skip scripted assistant replies after this user — we generate live.
			// But if the next turn is a scripted assistant (filler Q/A), we still
			// only send the user message live; we do NOT ingest the scripted
			// assistant (live reply replaces it). Advance past a following
			// assistant script line so we don't treat it as a user step later.
			if s.cursor < len(s.Scenario.Turns) && s.Scenario.Turns[s.cursor].Role == "assistant" {
				s.cursor++
			}
			return t.Content, true
		}
		// Unexpected non-user at cursor after seed: ingest into both as seed-like.
		s.naive.seed(t.Role, t.Content)
		_ = s.acgc.seed(context.Background(), t.Role, t.Content)
	}
	return "", false
}
