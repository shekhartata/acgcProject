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
	"github.com/chandrashekhartata/acgc/internal/embedding"
	"github.com/chandrashekhartata/acgc/internal/gateway"
	"github.com/chandrashekhartata/acgc/internal/gc"
	"github.com/chandrashekhartata/acgc/internal/llm"
	"github.com/chandrashekhartata/acgc/internal/scorer"
	"github.com/chandrashekhartata/acgc/internal/session"
	"github.com/chandrashekhartata/acgc/internal/store"
	"github.com/chandrashekhartata/acgc/internal/tokenizer"
	"github.com/chandrashekhartata/acgc/internal/vectorindex"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()

	// Real, model-aware token counting for prompt budgeting and metrics.
	tokenCounter := tokenizer.New(cfg.DefaultLLMModel)
	tokenizer.SetDefault(tokenCounter)
	log.Printf("tokenizer: %s (model %q)", tokenCounter.Name(), cfg.DefaultLLMModel)

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
	if cfg.SemanticEnabled {
		sc.SetSemanticWeight(cfg.SemanticWeight)
	}

	// Embedding provider (semantic scoring). Constructed only when enabled
	// — keeps v1 behavior identical when the flag is off.
	var embedder embedding.Provider
	if cfg.SemanticEnabled {
		if cfg.EmbedAPIKey == "" {
			log.Print("semantic: ACGC_SEMANTIC_ENABLED=true but no embed API key — falling back to heuristic-only")
		} else {
			switch cfg.EmbedProvider {
			case "openai", "":
				embedder = embedding.NewOpenAI(embedding.Config{
					Provider: cfg.EmbedProvider,
					BaseURL:  cfg.EmbedBaseURL,
					APIKey:   cfg.EmbedAPIKey,
					Model:    cfg.EmbedModel,
					Dim:      cfg.EmbedDim,
				})
				log.Printf("semantic: enabled — embedder=%s/%s (dim=%d), weight=%.2f, topK=%d",
					cfg.EmbedProvider, cfg.EmbedModel, cfg.EmbedDim, cfg.SemanticWeight, cfg.HNSWTopKAtCompile)
			default:
				log.Printf("semantic: unknown provider %q — falling back to heuristic-only", cfg.EmbedProvider)
			}
		}
	} else {
		log.Print("semantic: disabled (ACGC_SEMANTIC_ENABLED=false)")
	}

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
		DecisionSweepFloor:    cfg.GCDecisionSweepFloor,
		MaxActiveNodes:        cfg.GCMaxActiveNodes,
		SweepHeadroomRatio:    cfg.GCSweepHeadroomRatio,
		StaleAfterTurns:       cfg.StaleAfterTurns,
	}
	collector := gc.NewGarbageCollector(gcPolicy, sc, compressor)

	// Session Manager
	sessMgr := session.NewManager(session.ManagerConfig{
		Store:                mongoStore,
		Scorer:               sc,
		GC:                   collector,
		TokenBudget:          cfg.DefaultTokenBudget,
		TokenCounter:         tokenCounter,
		ChannelBuffer:        cfg.SessionChannelBuffer,
		IdleTimeoutS:         cfg.SessionIdleTimeoutS,
		SnapshotEveryS:       cfg.SnapshotIntervalS,
		Embedder:             embedder,
		SemanticWeight:       cfg.SemanticWeight,
		TopKAtCompile:        cfg.HNSWTopKAtCompile,
		ArchiveTopKAtCompile: cfg.ArchiveSemanticTopK,
		HNSWConfig: vectorindex.Config{
			Dim:      cfg.EmbedDim,
			M:        cfg.HNSWM,
			EFSearch: cfg.HNSWEFSearch,
		},
		LatencyBreakdown: cfg.LatencyBreakdown,
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
