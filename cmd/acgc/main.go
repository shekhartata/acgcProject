package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/chandrashekhartata/acgc/api/proto"
	"github.com/chandrashekhartata/acgc/internal/config"
	"github.com/chandrashekhartata/acgc/internal/gateway"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/llm"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/session"
	"github.com/chandrashekhartata/acgc/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// MongoDB
	if cfg.MongoURI == "" {
		log.Fatal("ACGC_MONGO_URI is required. Set it in .env or as an environment variable.\n" +
			"  Example: ACGC_MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/?retryWrites=true&w=majority")
	}
	mongoStore, err := store.NewMongoStore(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	defer mongoStore.Close(ctx)

	// Mask credentials in log output
	maskedURI := cfg.MongoURI
	if idx := len(maskedURI); idx > 30 {
		maskedURI = maskedURI[:30] + "..."
	}
	log.Printf("connected to MongoDB Atlas (db: %s)", cfg.MongoDB)

	// Scorer
	sc := scorer.NewScorer(cfg.StaleAfterTurns, 2000)

	// Compressor — use LLM compressor if API key is configured, else simple fallback
	var compressor gc.Compressor
	if cfg.SummarizerAPIKey != "" {
		summarizerClient := llm.NewClient(llm.Config{
			Provider: cfg.SummarizerProvider,
			BaseURL:  cfg.SummarizerBaseURL,
			APIKey:   cfg.SummarizerAPIKey,
			Model:    cfg.SummarizerModel,
		})
		compressor = llm.NewLLMCompressor(summarizerClient)
		log.Printf("branch compressor: LLM (%s/%s)", cfg.SummarizerProvider, cfg.SummarizerModel)
	} else {
		compressor = &gc.SimpleCompressor{}
		log.Print("branch compressor: simple (no LLM configured)")
	}

	// Garbage Collector
	gcPolicy := gc.Policy{
		MaxPromptTokens:       cfg.DefaultTokenBudget,
		MaxTreeDepth:          cfg.MaxTreeDepth,
		MaxChildrenPerNode:    cfg.MaxChildrenPerNode,
		LowRelevanceThreshold: cfg.LowRelevanceThreshold,
		StaleAfterTurns:       cfg.StaleAfterTurns,
	}
	collector := gc.NewGarbageCollector(gcPolicy, sc, compressor)

	// Session Manager
	sessMgr := session.NewManager(session.ManagerConfig{
		Store:          mongoStore,
		Scorer:         sc,
		GC:             collector,
		TokenBudget:    cfg.DefaultTokenBudget,
		ChannelBuffer:  cfg.SessionChannelBuffer,
		IdleTimeoutS:   cfg.SessionIdleTimeoutS,
		SnapshotEveryS: cfg.SnapshotIntervalS,
	})

	// Default LLM client (the "master" LLM — used when request doesn't specify its own)
	defaultLLM := llm.NewClient(llm.Config{
		Provider: cfg.DefaultLLMProvider,
		BaseURL:  cfg.DefaultLLMBaseURL,
		APIKey:   cfg.DefaultLLMAPIKey,
		Model:    cfg.DefaultLLMModel,
	})
	if cfg.DefaultLLMAPIKey != "" {
		log.Printf("main LLM: %s/%s (API key configured)", cfg.DefaultLLMProvider, cfg.DefaultLLMModel)
	} else {
		log.Print("main LLM: no API key — Run RPC requires llm_config.api_key per request")
	}

	// gRPC server
	grpcServer := grpc.NewServer()
	acgcServer := gateway.NewServer(sessMgr, defaultLLM)
	pb.RegisterACGCServiceServer(grpcServer, acgcServer)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down...", sig)
		grpcServer.GracefulStop()
		cancel()
	}()

	log.Printf("ACGC gRPC server listening on :%s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
