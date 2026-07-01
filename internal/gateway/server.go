package gateway

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/chandrashekhartata/acgc/api/proto"
	"github.com/chandrashekhartata/acgc/internal/domain"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/llm"
	"github.com/chandrashekhartata/acgc/internal/session"
	"github.com/chandrashekhartata/acgc/internal/tokenizer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedACGCServiceServer
	sessions   *session.Manager
	defaultLLM *llm.Client
}

func NewServer(sessions *session.Manager, defaultLLM *llm.Client) *Server {
	return &Server{
		sessions:   sessions,
		defaultLLM: defaultLLM,
	}
}

func (s *Server) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
	start := time.Now()

	sessionID := req.SessionId
	taskID := req.TaskId
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}
	if taskID == "" {
		taskID = "default"
	}

	// Ensure session exists
	s.sessions.GetOrCreate(ctx, sessionID, taskID)

	// Capture incoming user event (async — enqueue for background processing)
	userEvent := &domain.Event{
		EventID:    fmt.Sprintf("evt_%d_user", time.Now().UnixNano()),
		SessionID:  sessionID,
		TaskID:     taskID,
		EventType:  domain.EventUserPrompt,
		Payload:    req.UserMessage,
		TokenCount: tokenizer.Default().Count(req.UserMessage),
		CreatedAt:  time.Now(),
	}
	s.sessions.EnqueueEvent(sessionID, userEvent)

	// FAST PATH: compile prompt from current cached state
	systemPrompt := ""
	if len(req.ConversationHistory) > 0 {
		for _, m := range req.ConversationHistory {
			if m.Role == "system" {
				systemPrompt = m.Content
				break
			}
		}
	}

	tCompile := time.Now()
	compiled := s.sessions.CompilePrompt(sessionID, taskID, req.UserMessage, systemPrompt)
	compileWall := time.Since(tCompile)

	// Select LLM client
	llmClient := s.selectLLMClient(req.LlmConfig)

	// Wire format (Phase 2):
	//   [system, user(FinalPrompt = context body), user(CurrentUserMessage)]
	// FinalPrompt holds only the structured context; the user's question is
	// sent as its own user message so framing matches naive chat history and
	// no "## Current Request" wrapper is paid on the wire.
	messages := make([]llm.ChatMessage, 0, 3)
	if compiled.SystemPrompt != "" {
		messages = append(messages, llm.ChatMessage{Role: "system", Content: compiled.SystemPrompt})
	}
	if compiled.FinalPrompt != "" {
		messages = append(messages, llm.ChatMessage{Role: "user", Content: compiled.FinalPrompt})
	}
	messages = append(messages, llm.ChatMessage{Role: "user", Content: req.UserMessage})

	temperature := float64(0.7)
	maxTokens := 4096
	if req.LlmConfig != nil {
		if req.LlmConfig.Temperature > 0 {
			temperature = float64(req.LlmConfig.Temperature)
		}
		if req.LlmConfig.MaxTokens > 0 {
			maxTokens = int(req.LlmConfig.MaxTokens)
		}
	}

	tLLM := time.Now()
	result, err := llmClient.Generate(ctx, messages, temperature, maxTokens)
	llmDur := time.Since(tLLM)
	if err != nil {
		return nil, fmt.Errorf("llm generate: %w", err)
	}

	// Capture LLM response event (async)
	responseEvent := &domain.Event{
		EventID:    fmt.Sprintf("evt_%d_llm", time.Now().UnixNano()),
		SessionID:  sessionID,
		TaskID:     taskID,
		EventType:  domain.EventLLMResponse,
		Payload:    result.Content,
		TokenCount: result.CompletionTokens,
		Metadata: map[string]string{
			"prompt_tokens":     fmt.Sprintf("%d", result.PromptTokens),
			"completion_tokens": fmt.Sprintf("%d", result.CompletionTokens),
		},
		CreatedAt: time.Now(),
	}
	s.sessions.EnqueueEvent(sessionID, responseEvent)

	latencyMs := float64(time.Since(start).Milliseconds())
	log.Printf("gateway: run completed for %s in %.0fms (original: %d tokens, compiled: %d tokens)",
		sessionID, latencyMs, compiled.OriginalTokenCount, compiled.CompiledTokenCount)

	tokensSaved := compiled.OriginalTokenCount - compiled.CompiledTokenCount
	reductionPct := float64(0)
	if compiled.OriginalTokenCount > 0 {
		reductionPct = float64(tokensSaved) / float64(compiled.OriginalTokenCount) * 100
	}

	// Gather tree stats for the response
	_, activeNodes, compressedNodes, archivedNodes, _, _, _ := s.sessions.GetTreeStats(sessionID)

	resp := &pb.RunResponse{
		LlmResponse:      result.Content,
		CompiledPromptId: compiled.CompiledPromptID,
		Stats: &pb.PromptStats{
			OriginalTokenCount: int32(compiled.OriginalTokenCount),
			CompiledTokenCount: int32(compiled.CompiledTokenCount),
			TokensSaved:        int32(tokensSaved),
			ReductionPercent:   float32(reductionPct),
			ActiveNodes:        int32(activeNodes),
			CompressedNodes:    int32(compressedNodes),
			ArchivedNodes:      int32(archivedNodes),
		},
	}

	if bd := compiled.LatencyBreakdown; bd != nil {
		resp.LatencyBreakdown = &pb.RunLatency{
			CompileTotalMs:    int32(compileWall.Milliseconds()),
			CompileEmbedMs:    bd.CompileEmbedMs,
			CompileIndexMs:    bd.CompileIndexMs,
			CompileAssemblyMs: bd.CompileAssemblyMs,
			ComposeOverheadMs: bd.ComposeOverheadMs,
			LlmMs:             int32(llmDur.Milliseconds()),
			SemanticFallback:  bd.SemanticFallback,
		}
	}

	return resp, nil
}

func (s *Server) CaptureEvent(ctx context.Context, req *pb.CaptureEventRequest) (*pb.CaptureEventResponse, error) {
	sessionID := req.SessionId
	taskID := req.TaskId

	s.sessions.GetOrCreate(ctx, sessionID, taskID)

	event := &domain.Event{
		EventID:    fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		SessionID:  sessionID,
		TaskID:     taskID,
		EventType:  domain.EventType(req.EventType),
		Payload:    req.Payload,
		TokenCount: tokenizer.Default().Count(req.Payload),
		Metadata:   req.Metadata,
		CreatedAt:  time.Now(),
	}

	accepted := s.sessions.EnqueueEvent(sessionID, event)
	return &pb.CaptureEventResponse{
		EventId:  event.EventID,
		Accepted: accepted,
	}, nil
}

func (s *Server) GetState(ctx context.Context, req *pb.GetStateRequest) (*pb.GetStateResponse, error) {
	sessionID := req.SessionId

	total, active, compressed, archived, maxDepth, maxWidth, ok := s.sessions.GetTreeStats(sessionID)
	if !ok {
		return &pb.GetStateResponse{SessionId: sessionID}, nil
	}

	resp := &pb.GetStateResponse{
		SessionId: sessionID,
		TreeStats: &pb.TreeStats{
			TotalNodes:      int32(total),
			ActiveNodes:     int32(active),
			CompressedNodes: int32(compressed),
			ArchivedNodes:   int32(archived),
			MaxDepth:        int32(maxDepth),
			MaxWidth:        int32(maxWidth),
		},
	}

	return resp, nil
}

func (s *Server) TriggerGC(ctx context.Context, req *pb.TriggerGCRequest) (*pb.TriggerGCResponse, error) {
	sessionID := req.SessionId

	state := s.sessions.GetOrCreate(ctx, sessionID, "default")
	_ = state

	// Enqueue a GC trigger event so the session worker handles it
	gcEvent := &domain.Event{
		EventID:   fmt.Sprintf("evt_%d_gc", time.Now().UnixNano()),
		SessionID: sessionID,
		EventType: domain.EventGCTrigger,
		Payload:   "manual_trigger",
		CreatedAt: time.Now(),
	}
	s.sessions.EnqueueEvent(sessionID, gcEvent)

	return &pb.TriggerGCResponse{
		Triggered: true,
		Reason:    string(gc.ReasonManual),
	}, nil
}

func (s *Server) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	metrics, ok := s.sessions.GetMetrics(req.SessionId)
	if !ok {
		return &pb.GetMetricsResponse{SessionId: req.SessionId}, nil
	}

	return &pb.GetMetricsResponse{
		SessionId:            metrics.SessionID,
		TotalEvents:          int32(metrics.TotalEvents),
		TotalTurns:           int32(metrics.TotalTurns),
		GcRuns:               int32(metrics.GCRuns),
		TotalTokensSaved:     int32(metrics.TotalTokensSaved),
		AvgReductionPercent:  float32(metrics.AvgReductionPct),
		BranchesCompressed:   int32(metrics.BranchesCompressed),
		RehydrationEvents:    int32(metrics.RehydrationEvents),
		AvgLatencyOverheadMs: float32(metrics.AvgLatencyMs),
		SessionStartedAt:     timestamppb.New(metrics.SessionStartedAt),
	}, nil
}

func (s *Server) selectLLMClient(cfg *pb.LLMConfig) *llm.Client {
	if cfg == nil || cfg.ApiKey == "" {
		return s.defaultLLM
	}
	return llm.NewClient(llm.Config{
		Provider: cfg.Provider,
		BaseURL:  cfg.BaseUrl,
		APIKey:   cfg.ApiKey,
		Model:    cfg.Model,
	})
}
