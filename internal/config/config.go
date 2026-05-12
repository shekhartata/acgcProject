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
	StaleAfterTurns       int
	GCCheckInterval       int // check every N turns

	// Session management
	SessionChannelBuffer int
	SessionIdleTimeoutS  int
	SnapshotIntervalS    int
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
		StaleAfterTurns:       envOrDefaultInt("ACGC_STALE_TURNS", 15),
		GCCheckInterval:       envOrDefaultInt("ACGC_GC_INTERVAL", 5),

		SessionChannelBuffer: envOrDefaultInt("ACGC_SESSION_BUFFER", 100),
		SessionIdleTimeoutS:  envOrDefaultInt("ACGC_SESSION_IDLE_TIMEOUT", 1800),
		SnapshotIntervalS:    envOrDefaultInt("ACGC_SNAPSHOT_INTERVAL", 60),
	}
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
