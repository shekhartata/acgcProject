package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	pb "github.com/shekhartata/acgcProject/api/proto"
	"github.com/shekhartata/acgcProject/internal/config"
	"github.com/shekhartata/acgcProject/internal/latencybench"
	"github.com/shekhartata/acgcProject/internal/llm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type benchSample struct {
	Iter           int               `json:"iter"`
	BaselineMs     int64             `json:"baseline_llm_ms"`
	Status         map[string]string `json:"status,omitempty"`
	RunRoundtripMs int64             `json:"acgc_run_round_trip_ms,omitempty"`
	ServerLatency  *pb.RunLatency    `json:"latency_breakdown,omitempty"`
	Err            string            `json:"error,omitempty"`
}

type pctSummary struct {
	P50 float64 `json:"p50_ms"`
	P95 float64 `json:"p95_ms"`
	P99 float64 `json:"p99_ms"`
}

func main() {
	log.SetFlags(0)

	grpcAddr := flag.String("grpc", "localhost:50051", "ACGC gRPC address (semantic-enabled server)")
	sessionFmt := flag.String("sessions", "latency-bench-%d", `sprintf SessionId pattern; must include %d when concurrency>1 to avoid shared sessions`)
	taskID := flag.String("task-id", "latency-bench", "task ID for CaptureEvent / Run")
	iterations := flag.Int("iterations", 30, "timed samples")
	concurrency := flag.Int("concurrency", 2, "max concurrent iterations")
	warmTurns := flag.Int("warm-turns", -1, "replay first N warm_pairs before timing (-1=all, 0=none)")
	fixturePath := flag.String("fixture", "", "JSON fixture path (default embedded)")
	discardN := flag.Int("discard-n", 0, "discard first N iterations by index from percentile summaries")
	outFmt := flag.String("output", "text", `report format: "text" or "json"`)
	enforceSemantic := flag.Bool("enforce-semantic", true, "fail when semantic_fallback=true (after discard-n)")
	captureDelay := flag.Duration("capture-delay", 8*time.Millisecond, "pause between CaptureEvent RPCs")
	settleDelay := flag.Duration("warm-settle-delay", 300*time.Millisecond, "pause after warm-up so workers drain")
	temperature := flag.Float64("temperature", 0.7, "LLM temperature for baseline + grpc llm_config")
	maxTokens := flag.Int("max-tokens", 1024, "LLM max_tokens for baseline + grpc llm_config")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `acgc-latencybench — naive baseline vs ACGC Run latency.

Assume the server runs semantic-on (%s=true) plus embed credentials.
Fine-grained curves need %s=true on the daemon.

USAGE:
  %s [flags]

FLAGS:
`, "`ACGC_SEMANTIC_ENABLED`", "`ACGC_LATENCY_BREAKDOWN`", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Example:
  %s -grpc localhost:50051 -iterations 40 -discard-n 5 \\
    -warm-settle-delay 400ms -sessions 'latency-session-%%d' -output json
`, os.Args[0])
	}

	flag.Parse()

	if *iterations <= 0 {
		log.Fatal("-iterations must be > 0")
	}
	if *concurrency <= 0 {
		log.Fatal("-concurrency must be > 0")
	}

	cfg := config.Load()
	fixture, err := latencybench.LoadFixture(*fixturePath)
	if err != nil {
		log.Fatalf("fixture: %v", err)
	}

	llmClient := llm.NewClient(llm.Config{
		Provider: cfg.DefaultLLMProvider,
		BaseURL:  cfg.DefaultLLMBaseURL,
		APIKey:   cfg.DefaultLLMAPIKey,
		Model:    cfg.DefaultLLMModel,
	})
	if cfg.DefaultLLMAPIKey == "" {
		log.Printf("WARNING: ACGC_LLM_API_KEY is empty — naive baseline / grpc llm_config need credentials.")
	}

	conn, err := grpc.NewClient(*grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc dial %s: %v", *grpcAddr, err)
	}
	defer conn.Close()

	pbCli := pb.NewACGCServiceClient(conn)
	pairsWarm := warmupPairs(fixture.WarmPairs, *warmTurns)

	samples := make([]benchSample, *iterations)
	sem := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup

	runOne := func(i int) benchSample {
		s := benchSample{Iter: i}
		sessionID, patErr := sprintSessionPattern(*sessionFmt, i)
		if patErr != "" {
			s.Err = patErr
			return s
		}

		wctx, cancelWarm := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancelWarm()

		if err := latencybench.WarmSessionGRPC(wctx, pbCli, sessionID, *taskID, pairsWarm, *captureDelay); err != nil {
			s.Err = fmt.Sprintf("warm replay: %v", err)
			return s
		}
		select {
		case <-wctx.Done():
			s.Err = wctx.Err().Error()
			return s
		case <-time.After(*settleDelay):
		}

		blCtx, cbl := context.WithTimeout(context.Background(), 300*time.Second)
		bms, _, _, _, berr := latencybench.NaiveLLMLatency(blCtx, llmClient, fixture, *temperature, *maxTokens)
		cbl()
		s.BaselineMs = bms
		if berr != nil {
			s.Status = map[string]string{"baseline": berr.Error()}
			s.Err = fmt.Sprintf("baseline: %v", berr)
			return s
		}

		req := &pb.RunRequest{
			SessionId:   sessionID,
			TaskId:      *taskID,
			UserMessage: fixture.Probe,
			TokenBudget: int32(cfg.DefaultTokenBudget),
			Policy:      "balanced",
			ConversationHistory: []*pb.Message{
				{Role: "system", Content: fixture.System},
			},
			LlmConfig: &pb.LLMConfig{
				Provider:    cfg.DefaultLLMProvider,
				Model:       cfg.DefaultLLMModel,
				ApiKey:      cfg.DefaultLLMAPIKey,
				BaseUrl:     cfg.DefaultLLMBaseURL,
				Temperature: float32(*temperature),
				MaxTokens:   int32(*maxTokens),
			},
		}

		gctx, cg := context.WithTimeout(context.Background(), 300*time.Second)
		resp, rt, gerr := latencybench.GRPCRunMeasured(gctx, pbCli, req)
		cg()

		s.RunRoundtripMs = rt.Milliseconds()
		if gerr != nil {
			if s.Status == nil {
				s.Status = make(map[string]string)
			}
			s.Status["grpc"] = gerr.Error()
			s.Err = gerr.Error()
			return s
		}
		s.ServerLatency = resp.GetLatencyBreakdown()
		return s
	}

	for i := 0; i < *iterations; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			samples[idx] = runOne(idx)
		}(i)
	}
	wg.Wait()

	baselineVals := valsAfterDiscard(samples, *discardN, func(bs benchSample) (float64, bool) {
		if bs.Err != "" {
			return 0, false
		}
		return float64(bs.BaselineMs), true
	})
	rtVals := valsAfterDiscard(samples, *discardN, func(bs benchSample) (float64, bool) {
		if bs.Err != "" || bs.RunRoundtripMs <= 0 {
			return 0, false
		}
		return float64(bs.RunRoundtripMs), true
	})
	srvLLMVals := valsAfterDiscard(samples, *discardN, func(bs benchSample) (float64, bool) {
		if bs.Err != "" || bs.ServerLatency == nil {
			return 0, false
		}
		return float64(bs.ServerLatency.GetLlmMs()), true
	})
	compileTotVals := valsAfterDiscard(samples, *discardN, func(bs benchSample) (float64, bool) {
		if bs.Err != "" || bs.ServerLatency == nil {
			return 0, false
		}
		return float64(bs.ServerLatency.GetCompileTotalMs()), true
	})

	blSum := summarize(baselineVals)
	rtSum := summarize(rtVals)
	srvSum := summarize(srvLLMVals)
	ctSum := summarize(compileTotVals)

	errCount := 0
	missingBreakdown := 0
	fallbackCtr := 0
	noEmbedCtr := 0
	for _, samp := range samples {
		if samp.Err != "" {
			errCount++
		}
		if samp.Err == "" && samp.ServerLatency == nil {
			missingBreakdown++
		}
		if samp.ServerLatency != nil && samp.ServerLatency.SemanticFallback {
			fallbackCtr++
		}
		if samp.ServerLatency != nil && samp.ServerLatency.GetCompileEmbedMs() == 0 && samp.Err == "" {
			noEmbedCtr++
		}
	}

	if missingBreakdown > 0 && *discardN < len(samples) {
		fmt.Fprintf(os.Stderr, "WARNING: %d successful iterations had empty latency_breakdown — set ACGC_LATENCY_BREAKDOWN=true on the server.\n", missingBreakdown)
	}

	okIterations := okSampleCount(samples, *discardN)
	if okIterations >= 4 && float64(noEmbedCtr) >= 0.5*float64(okIterations) {
		fmt.Fprintf(os.Stderr, "WARNING: compile_embed_ms was zero on >=50%% of successful runs — heuristic compile or timings misconfiguration.\n")
	}

	validAfterDiscard := filterValid(collectAfterDiscard(samples, *discardN))
	exitCode := 0
	if *enforceSemantic {
		for _, s := range validAfterDiscard {
			if sl := s.ServerLatency; sl != nil && sl.SemanticFallback {
				fmt.Fprintln(os.Stderr, "ERROR: semantic_fallback=true with -enforce-semantic (verify ACGC_SEMANTIC_ENABLED + embeddings).")
				exitCode = 3
				break
			}
		}
	}

	if errCount > 0 {
		fmt.Fprintf(os.Stderr, "NOTICE: %d iterations failed.\n", errCount)
	}

	report := struct {
		Settings struct {
			GRPC          string `json:"grpc"`
			Iterations    int    `json:"iterations"`
			DiscardN      int    `json:"discard_n"`
			SessionFmt    string `json:"sessions_pattern"`
			WarmPairsUsed int    `json:"warm_pairs_used"`
			Concurrency   int    `json:"concurrency"`
			Probe         string `json:"probe_excerpt,omitempty"`
		} `json:"settings"`
		Counts struct {
			Errors                  int `json:"errors"`
			MissingLatencyBreakdown int `json:"missing_latency_breakdown"`
			SemanticFallbackRuns    int `json:"semantic_fallback_observed"`
		} `json:"counts"`
		Baseline pctSummary `json:"baseline_llm_wall_ms"`
		ACGC     struct {
			RunRoundtrip pctSummary `json:"run_round_trip_ms"`
			ServerLLM    pctSummary `json:"server_reported_llm_ms"`
			CompileTotal pctSummary `json:"compile_total_ms"`
		} `json:"acgc"`
		CSVHistogram string        `json:"csv_histogram_summary"`
		Samples      []benchSample `json:"samples,omitempty"`
	}{}

	probeEx := fixture.Probe
	if len(probeEx) > 96 {
		probeEx = probeEx[:93] + "..."
	}
	report.Settings.GRPC = *grpcAddr
	report.Settings.Iterations = *iterations
	report.Settings.DiscardN = *discardN
	report.Settings.SessionFmt = *sessionFmt
	report.Settings.Concurrency = *concurrency
	report.Settings.WarmPairsUsed = len(pairsWarm)
	report.Settings.Probe = probeEx

	report.Counts.Errors = errCount
	report.Counts.MissingLatencyBreakdown = missingBreakdown
	report.Counts.SemanticFallbackRuns = fallbackCtr

	report.Baseline = blSum
	report.ACGC.RunRoundtrip = rtSum
	report.ACGC.ServerLLM = srvSum
	report.ACGC.CompileTotal = ctSum

	report.CSVHistogram = fmt.Sprintf(
		"series,p50_ms,p95_ms,p99_ms\nbaseline_llm,%.3f,%.3f,%.3f\nacgc_run_round_trip,%.3f,%.3f,%.3f\nacgc_server_llm,%.3f,%.3f,%.3f\nacgc_compile_total,%.3f,%.3f,%.3f\n",
		blSum.P50, blSum.P95, blSum.P99,
		rtSum.P50, rtSum.P95, rtSum.P99,
		srvSum.P50, srvSum.P95, srvSum.P99,
		ctSum.P50, ctSum.P95, ctSum.P99,
	)

	report.Samples = samples

	switch strings.ToLower(strings.TrimSpace(*outFmt)) {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encErr := enc.Encode(report); encErr != nil {
			log.Fatal(encErr)
		}
	case "text":
		fmt.Printf("GRPC=%s iterations=%d discard=%d warm_pairs=%d concurrency=%d\n\n", *grpcAddr, *iterations, *discardN, len(pairsWarm), *concurrency)
		fmt.Printf("BASELINE naive LLM wall:\n  p50=%.1f ms  p95=%.1f ms  p99=%.1f ms\n\n", blSum.P50, blSum.P95, blSum.P99)
		fmt.Printf("ACGC Run (client RTT):\n  p50=%.1f ms  p95=%.1f ms  p99=%.1f ms\n\n", rtSum.P50, rtSum.P95, rtSum.P99)
		fmt.Println("ACGC server latency_breakdown (needs ACGC_LATENCY_BREAKDOWN):")
		fmt.Printf("  llm_ms        p50=%.1f  p95=%.1f  p99=%.1f\n", srvSum.P50, srvSum.P95, srvSum.P99)
		fmt.Printf("  compile_total p50=%.1f  p95=%.1f  p99=%.1f\n\n", ctSum.P50, ctSum.P95, ctSum.P99)
		fmt.Println("CSV histogram:")
		fmt.Print(report.CSVHistogram)
	default:
		log.Fatalf("unknown -output %q", *outFmt)
	}

	os.Exit(exitCode)
}

func warmupPairs(all []latencybench.WarmPairTurn, warmTurns int) []latencybench.WarmPairTurn {
	switch {
	case len(all) == 0 || warmTurns == 0:
		return nil
	case warmTurns < 0:
		return all
	default:
		if warmTurns >= len(all) {
			return all
		}
		return append([]latencybench.WarmPairTurn(nil), all[:warmTurns]...)
	}
}

func sprintSessionPattern(pat string, i int) (string, string) {
	if !strings.Contains(pat, "%") {
		return pat, ""
	}
	if strings.Contains(pat, "%d") {
		return fmt.Sprintf(pat, i), ""
	}
	return "", fmt.Sprintf("sessions pattern %q must include %%d when it contains %%", pat)
}

func collectAfterDiscard(samples []benchSample, d int) []benchSample {
	if d < 0 {
		d = 0
	}
	if d >= len(samples) {
		return nil
	}
	return samples[d:]
}

func filterValid(in []benchSample) []benchSample {
	out := make([]benchSample, 0, len(in))
	for _, s := range in {
		if s.Err == "" && s.ServerLatency != nil {
			out = append(out, s)
		}
	}
	return out
}

func valsAfterDiscard(samples []benchSample, d int, pick func(benchSample) (float64, bool)) []float64 {
	var xs []float64
	for _, s := range collectAfterDiscard(samples, d) {
		if v, ok := pick(s); ok {
			xs = append(xs, v)
		}
	}
	return xs
}

func summarize(xs []float64) pctSummary {
	if len(xs) == 0 {
		return pctSummary{}
	}
	s := latencybench.SortCopy(xs)
	return pctSummary{
		P50: latencybench.QuantileLinear(s, 0.50),
		P95: latencybench.QuantileLinear(s, 0.95),
		P99: latencybench.QuantileLinear(s, 0.99),
	}
}

func okSampleCount(samples []benchSample, d int) int {
	n := 0
	for _, s := range collectAfterDiscard(samples, d) {
		if s.Err == "" && s.ServerLatency != nil {
			n++
		}
	}
	return n
}
