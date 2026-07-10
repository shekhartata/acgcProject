package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	demo "github.com/shekhartata/acgcProject/demo/internal"
	"github.com/shekhartata/acgcProject/internal/config"
)

//go:embed static/*
var staticFS embed.FS

func main() {
	log.SetFlags(0)
	addr := flag.String("addr", ":8080", "HTTP listen address")
	acgcAddr := flag.String("acgc", "", "ACGC gRPC address (default localhost:50051 or ACGC_DEMO_ACGC_ADDR)")
	budget := flag.Int("budget", 0, "token budget override (default 1800 or ACGC_DEMO_TOKEN_BUDGET)")
	flag.Parse()

	cfg := config.Load()
	if cfg.DefaultLLMAPIKey == "" {
		log.Fatal("ACGC_LLM_API_KEY is required (naive pane + ACGC server LLM config)")
	}

	grpcAddr := *acgcAddr
	if grpcAddr == "" {
		grpcAddr = os.Getenv("ACGC_DEMO_ACGC_ADDR")
	}
	if grpcAddr == "" {
		grpcAddr = "localhost:50051"
	}

	tokBudget := *budget
	if tokBudget <= 0 {
		// Prefer a tight demo budget over the server default (6000) so the
		// newest-first naive pane actually drops early decisions.
		if v := os.Getenv("ACGC_DEMO_TOKEN_BUDGET"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				tokBudget = n
			}
		}
		if tokBudget <= 0 {
			tokBudget = 1800
		}
	}

	engine := demo.NewEngine(demo.EngineConfig{
		ACGCAddr:    grpcAddr,
		TokenBudget: tokBudget,
		LLMProvider: cfg.DefaultLLMProvider,
		LLMBaseURL:  cfg.DefaultLLMBaseURL,
		LLMAPIKey:   cfg.DefaultLLMAPIKey,
		LLMModel:    cfg.DefaultLLMModel,
		MaxTokens:   2048,
	})

	mux := http.NewServeMux()
	api := &demo.API{Engine: engine}
	api.Mount(mux)

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	log.Printf("ACGC marketing demo on http://localhost%s", *addr)
	log.Printf("  ACGC sidecar: %s", grpcAddr)
	log.Printf("  budget=%d model=%s", tokBudget, cfg.DefaultLLMModel)
	log.Printf("  Prerequisites: MongoDB + ./bin/acgc must be running (rebuild after server fixes)")
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
