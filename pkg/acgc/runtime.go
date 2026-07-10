package acgc

import (
	"context"
	"fmt"
	"time"

	pb "github.com/shekhartata/acgcProject/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ContextRuntime is the public SDK interface for ACGC.
// It connects to the ACGC gRPC server and provides a simple API
// for running LLM calls through the context garbage collector.
type ContextRuntime struct {
	client    pb.ACGCServiceClient
	conn      *grpc.ClientConn
	sessionID string
	taskID    string
	budget    int32
	llmCfg    *pb.LLMConfig
}

type Config struct {
	ServerAddr  string // e.g. "localhost:50051"
	SessionID   string
	TaskID      string
	TokenBudget int
	LLM         LLMConfig
}

type LLMConfig struct {
	Provider    string
	Model       string
	APIKey      string
	BaseURL     string
	Temperature float32
	MaxTokens   int
}

// NewContextRuntime creates a new ACGC runtime connected to the gRPC server.
func NewContextRuntime(cfg Config) (*ContextRuntime, error) {
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = "localhost:50051"
	}
	if cfg.SessionID == "" {
		cfg.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}
	if cfg.TaskID == "" {
		cfg.TaskID = "default"
	}
	if cfg.TokenBudget == 0 {
		cfg.TokenBudget = 6000
	}

	conn, err := grpc.NewClient(
		cfg.ServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("acgc: connect to %s: %w", cfg.ServerAddr, err)
	}

	var llmCfg *pb.LLMConfig
	if cfg.LLM.APIKey != "" {
		llmCfg = &pb.LLMConfig{
			Provider:    cfg.LLM.Provider,
			Model:       cfg.LLM.Model,
			ApiKey:      cfg.LLM.APIKey,
			BaseUrl:     cfg.LLM.BaseURL,
			Temperature: cfg.LLM.Temperature,
			MaxTokens:   int32(cfg.LLM.MaxTokens),
		}
	}

	return &ContextRuntime{
		client:    pb.NewACGCServiceClient(conn),
		conn:      conn,
		sessionID: cfg.SessionID,
		taskID:    cfg.TaskID,
		budget:    int32(cfg.TokenBudget),
		llmCfg:    llmCfg,
	}, nil
}

// RunResult contains the LLM response and context optimization stats.
type RunResult struct {
	Response         string
	OriginalTokens   int
	CompiledTokens   int
	TokensSaved      int
	ReductionPercent float32
	GCTriggered      bool
	GCReason         string
	ActiveNodes      int
	CompressedNodes  int
	ArchivedNodes    int
}

// Run sends a user message through ACGC and returns the optimized LLM response.
func (r *ContextRuntime) Run(ctx context.Context, userMessage string) (*RunResult, error) {
	resp, err := r.client.Run(ctx, &pb.RunRequest{
		SessionId:   r.sessionID,
		TaskId:      r.taskID,
		UserMessage: userMessage,
		TokenBudget: r.budget,
		LlmConfig:   r.llmCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("acgc run: %w", err)
	}

	result := &RunResult{
		Response: resp.LlmResponse,
	}
	if resp.Stats != nil {
		result.OriginalTokens = int(resp.Stats.OriginalTokenCount)
		result.CompiledTokens = int(resp.Stats.CompiledTokenCount)
		result.TokensSaved = int(resp.Stats.TokensSaved)
		result.ReductionPercent = resp.Stats.ReductionPercent
		result.GCTriggered = resp.Stats.GcTriggered
		result.GCReason = resp.Stats.GcReason
		result.ActiveNodes = int(resp.Stats.ActiveNodes)
		result.CompressedNodes = int(resp.Stats.CompressedNodes)
		result.ArchivedNodes = int(resp.Stats.ArchivedNodes)
	}
	return result, nil
}

// SessionID returns the stable session id used for all RPCs.
func (r *ContextRuntime) SessionID() string { return r.sessionID }

// GetState returns tree stats for the current session.
func (r *ContextRuntime) GetState(ctx context.Context) (*pb.GetStateResponse, error) {
	return r.client.GetState(ctx, &pb.GetStateRequest{SessionId: r.sessionID})
}

// CaptureEvent manually captures an event into the session context.
// Returns an error if the server rejects the event (e.g. session channel full).
func (r *ContextRuntime) CaptureEvent(ctx context.Context, eventType, payload string, metadata map[string]string) (string, error) {
	resp, err := r.client.CaptureEvent(ctx, &pb.CaptureEventRequest{
		SessionId: r.sessionID,
		TaskId:    r.taskID,
		EventType: eventType,
		Payload:   payload,
		Metadata:  metadata,
	})
	if err != nil {
		return "", fmt.Errorf("acgc capture: %w", err)
	}
	if resp != nil && !resp.GetAccepted() {
		return resp.GetEventId(), fmt.Errorf("acgc capture: event not accepted (session channel full or session missing)")
	}
	return resp.GetEventId(), nil
}

// TriggerGC manually triggers garbage collection for this session.
func (r *ContextRuntime) TriggerGC(ctx context.Context) error {
	_, err := r.client.TriggerGC(ctx, &pb.TriggerGCRequest{
		SessionId: r.sessionID,
		Force:     true,
	})
	return err
}

// Metrics returns current session metrics.
func (r *ContextRuntime) Metrics(ctx context.Context) (*pb.GetMetricsResponse, error) {
	return r.client.GetMetrics(ctx, &pb.GetMetricsRequest{
		SessionId: r.sessionID,
	})
}

// Close releases the gRPC connection.
func (r *ContextRuntime) Close() error {
	return r.conn.Close()
}
