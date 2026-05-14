package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	GRPCPort string
	MongoURI string
	MongoDB  string

	// Main LLM (the "master" — used for actual agent reasoning)
	DefaultLLMProvider string
	DefaultLLMModel    string
	DefaultLLMBaseURL  string
	DefaultLLMAPIKey   string

	// Summarizer LLM (cheap model for branch compression)
	SummarizerProvider string
	SummarizerModel    string
	SummarizerBaseURL  string
	SummarizerAPIKey   string

	// GC policy defaults
	DefaultTokenBudget    int
	MaxTreeDepth          int
	MaxChildrenPerNode    int
	LowRelevanceThreshold float64
	// DecisionSweepFloor: GC sweep compares max(retention, floor) vs LowRelevance for NodeDecision
	// nodes that have empty Facts/Decisions. See gc.Policy.DecisionSweepFloor.
	// MUST stay strictly below LowRelevanceThreshold or bare decisions become un-sweepable.
	GCDecisionSweepFloor float64
	// MaxActiveNodes: count-based GC trigger. 0 disables.
	GCMaxActiveNodes int
	// GCSweepHeadroomRatio: soft GC trigger at ratio × DefaultTokenBudget. 0 disables.
	GCSweepHeadroomRatio float64
	StaleAfterTurns      int
	GCCheckInterval      int // check every N turns

	// Session management
	SessionChannelBuffer int
	SessionIdleTimeoutS  int
	SnapshotIntervalS    int

	// Semantic scoring (HNSW + embeddings). When SemanticEnabled is false,
	// no embedder is constructed and the system falls back to v1 heuristics.
	SemanticEnabled     bool
	SemanticWeight      float64
	HNSWTopKAtCompile   int
	ArchiveSemanticTopK int // top-K retrieval from ArchiveIndex merged at compile
	HNSWM               int
	HNSWEFSearch        int
	EmbedProvider       string
	EmbedBaseURL        string
	EmbedAPIKey         string
	EmbedModel          string
	EmbedDim            int

	// When true, RunResponse may include CompilePrompt + LLM latency breakdown fields.
	LatencyBreakdown bool
}

func Load() *Config {
	loadDotEnv()

	return &Config{
		GRPCPort: envOrDefault("ACGC_GRPC_PORT", "50051"),
		MongoURI: envOrDefault("ACGC_MONGO_URI", ""),
		MongoDB:  envOrDefault("ACGC_MONGO_DB", "acgc"),

		DefaultLLMProvider: envOrDefault("ACGC_LLM_PROVIDER", "openai"),
		DefaultLLMModel:    envOrDefault("ACGC_LLM_MODEL", "gpt-4o-mini"),
		DefaultLLMBaseURL:  envOrDefault("ACGC_LLM_BASE_URL", "https://api.openai.com/v1"),
		DefaultLLMAPIKey:   envOrDefault("ACGC_LLM_API_KEY", ""),

		SummarizerProvider: envOrDefault("ACGC_SUMMARIZER_PROVIDER", "openai"),
		SummarizerModel:    envOrDefault("ACGC_SUMMARIZER_MODEL", "gpt-4o-mini"),
		SummarizerBaseURL:  envOrDefault("ACGC_SUMMARIZER_BASE_URL", "https://api.openai.com/v1"),
		SummarizerAPIKey:   envOrDefault("ACGC_SUMMARIZER_API_KEY", ""),

		DefaultTokenBudget:    envOrDefaultInt("ACGC_TOKEN_BUDGET", 6000),
		MaxTreeDepth:          envOrDefaultInt("ACGC_MAX_TREE_DEPTH", 10),
		MaxChildrenPerNode:    envOrDefaultInt("ACGC_MAX_CHILDREN", 50),
		LowRelevanceThreshold: envOrDefaultFloat("ACGC_LOW_RELEVANCE", 0.30),
		// Phase 2: lowered from 0.35 → 0.20 so the floor sits strictly below
		// LowRelevanceThreshold (0.30). With the previous value the floor was
		// above the sweep threshold, which made bare NodeDecision nodes
		// permanently un-sweepable.
		GCDecisionSweepFloor: envOrDefaultFloat("ACGC_GC_DECISION_SWEEP_FLOOR", 0.20),
		// Phase 2: count and headroom triggers so GC actually fires on dense
		// short-token sessions that never approach DefaultTokenBudget.
		GCMaxActiveNodes:     envOrDefaultInt("ACGC_GC_MAX_ACTIVE_NODES", 25),
		GCSweepHeadroomRatio: envOrDefaultFloat("ACGC_GC_SWEEP_HEADROOM_RATIO", 0.60),
		StaleAfterTurns:      envOrDefaultInt("ACGC_STALE_TURNS", 15),
		GCCheckInterval:      envOrDefaultInt("ACGC_GC_INTERVAL", 5),

		SessionChannelBuffer: envOrDefaultInt("ACGC_SESSION_BUFFER", 100),
		SessionIdleTimeoutS:  envOrDefaultInt("ACGC_SESSION_IDLE_TIMEOUT", 1800),
		SnapshotIntervalS:    envOrDefaultInt("ACGC_SNAPSHOT_INTERVAL", 60),

		SemanticEnabled:     envOrDefaultBool("ACGC_SEMANTIC_ENABLED", false),
		SemanticWeight:      envOrDefaultFloat("ACGC_SEMANTIC_WEIGHT", 0.20),
		HNSWTopKAtCompile:   envOrDefaultInt("ACGC_HNSW_TOP_K_AT_COMPILE", 12),
		ArchiveSemanticTopK: envOrDefaultInt("ACGC_ARCHIVE_SEMANTIC_TOP_K", 12),
		HNSWM:               envOrDefaultInt("ACGC_HNSW_M", 16),
		HNSWEFSearch:        envOrDefaultInt("ACGC_HNSW_EF_SEARCH", 50),
		EmbedProvider:       envOrDefault("ACGC_EMBED_PROVIDER", "openai"),
		EmbedBaseURL:        envOrDefault("ACGC_EMBED_BASE_URL", "https://api.openai.com/v1"),
		// Default the embed key to the main LLM key — same OpenAI account
		// in 99% of setups, no need to duplicate.
		EmbedAPIKey: envOrDefault("ACGC_EMBED_API_KEY", os.Getenv("ACGC_LLM_API_KEY")),
		EmbedModel:  envOrDefault("ACGC_EMBED_MODEL", "text-embedding-3-small"),
		EmbedDim:    envOrDefaultInt("ACGC_EMBED_DIM", 1536),

		LatencyBreakdown: envOrDefaultBool("ACGC_LATENCY_BREAKDOWN", false),
	}
}

func envOrDefaultBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// loadDotEnv reads a .env file from the working directory if it exists.
// Keeps it simple — no external dependency needed.
func loadDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip surrounding quotes if present
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		// Don't override existing env vars
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
	log.Print("config: loaded .env file")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envOrDefaultFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
