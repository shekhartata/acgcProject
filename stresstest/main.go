package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/chandrashekhartata/acgc/stresstest/fixtures"
	"github.com/chandrashekhartata/acgc/stresstest/runner"
)

func main() {
	var (
		dataDir       = flag.String("data", "", "directory containing .jsonl fixture files (auto-generates if empty)")
		tokenBudget   = flag.Int("budget", 6000, "ACGC token budget")
		exportJSON    = flag.String("export", "", "export results to JSON file")
		skipConc      = flag.Bool("skip-concurrency", false, "skip concurrency tests")
		verbose       = flag.Bool("v", false, "print per-turn stats for each session")
	)
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    ACGC Stress Test Suite                                   ║")
	fmt.Println("║                    Agent Context Garbage Collector                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Load or generate fixtures
	var dir string
	if *dataDir != "" {
		dir = *dataDir
	} else {
		dir = filepath.Join(os.TempDir(), "acgc_stresstest_data")
		fmt.Printf("  Generating test fixtures in %s ...\n", dir)
		if err := fixtures.GenerateAll(dir); err != nil {
			log.Fatalf("  Failed to generate fixtures: %v", err)
		}
		fmt.Println("  Generated 5 conversation fixtures.")
	}

	// Load all .jsonl files
	convos := make(map[string][]fixtures.Turn)
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("  Failed to read data dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		turns, err := fixtures.LoadConversation(path)
		if err != nil {
			log.Printf("  Warning: failed to load %s: %v", e.Name(), err)
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".jsonl")
		convos[name] = turns
	}

	if len(convos) == 0 {
		log.Fatal("  No conversation fixtures found!")
	}

	fmt.Printf("\n  Loaded %d conversations:\n", len(convos))
	for name, turns := range convos {
		fmt.Printf("    - %-25s (%d turns)\n", name, len(turns))
	}

	cfg := runner.DefaultConfig()
	cfg.TokenBudget = *tokenBudget

	// Step 2: Replay each session
	fmt.Println("\n  Running token savings analysis...")
	suiteResult := runner.SuiteResult{}

	for name, turns := range convos {
		fmt.Printf("  ▸ Replaying %-25s ...", name)
		result := runner.ReplaySession(name, turns, cfg)
		suiteResult.Sessions = append(suiteResult.Sessions, result)
		fmt.Printf(" done (%d turns, %.1f%% reduction)\n",
			result.TotalTurns, result.AvgReductionPct)

		if *verbose {
			runner.PrintSessionReport(result)
		}
	}

	suiteResult.Compute()
	runner.PrintSuiteReport(suiteResult)

	// Step 3: Concurrency tests
	if !*skipConc {
		fmt.Println("  Running concurrency stress tests...")
		concResults := runner.RunConcurrencyTests(convos, cfg)
		runner.PrintConcurrencyReport(concResults)
	}

	// Step 4: Export
	if *exportJSON != "" {
		if err := suiteResult.ExportJSON(*exportJSON); err != nil {
			log.Printf("  Failed to export JSON: %v", err)
		} else {
			fmt.Printf("  Results exported to %s\n", *exportJSON)
		}
	}

	fmt.Println("  Stress test complete.")
}
