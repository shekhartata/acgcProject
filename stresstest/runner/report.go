package runner

import (
	"fmt"
	"strings"
	"time"
)

func PrintSessionReport(r SessionResult) {
	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	fmt.Printf("  SESSION: %s\n", r.Name)
	fmt.Printf("%s\n", strings.Repeat("=", 80))
	fmt.Printf("  Turns: %d | Duration: %s\n", r.TotalTurns, r.Duration.Round(time.Millisecond))
	fmt.Printf("  Coherency Score: %.1f%%\n", r.CoherencyScore*100)
	fmt.Println()

	// Per-turn table
	fmt.Printf("  %-5s %-9s %8s %8s %8s %7s %6s %6s %5s\n",
		"Turn", "Role", "Raw", "Compiled", "Saved", "Red%", "Active", "Total", "GC")
	fmt.Printf("  %s\n", strings.Repeat("-", 75))

	for _, ts := range r.TurnStats {
		gcMark := ""
		if ts.GCTriggered {
			gcMark = fmt.Sprintf("✓(%d)", ts.GCTokensFreed)
		}
		fmt.Printf("  %-5d %-9s %8d %8d %8d %6.1f%% %6d %6d %s\n",
			ts.TurnNumber, ts.Role,
			ts.RawTokens, ts.CompiledTokens, ts.TokensSaved,
			ts.ReductionPct, ts.ActiveNodes, ts.TotalNodes, gcMark)
	}

	fmt.Println()
	fmt.Printf("  SUMMARY:\n")
	fmt.Printf("    Raw tokens (without ACGC):  %d\n", r.TotalRawTokens)
	fmt.Printf("    Compiled tokens (with ACGC): %d\n", r.TotalCompiled)
	fmt.Printf("    Tokens saved:                %d (%.1f%% reduction)\n", r.TotalSaved, r.AvgReductionPct)
	fmt.Printf("    Peak raw tokens:             %d\n", r.PeakRawTokens)
	fmt.Printf("    Peak compiled tokens:        %d\n", r.PeakCompiledTokens)
	fmt.Printf("    GC runs:                     %d (freed %d tokens total)\n", r.GCRuns, r.TotalGCFreed)
	fmt.Printf("    Final nodes:                 %d active / %d total\n", r.FinalActiveNodes, r.FinalTotalNodes)
}

func PrintSuiteReport(sr SuiteResult) {
	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	fmt.Printf("  CROSS-SESSION SUMMARY\n")
	fmt.Printf("%s\n", strings.Repeat("=", 80))

	fmt.Printf("\n  %-25s %6s %10s %10s %10s %7s %8s\n",
		"Session", "Turns", "Raw Tok", "Compiled", "Saved", "Red%", "Coherency")
	fmt.Printf("  %s\n", strings.Repeat("-", 78))

	for _, s := range sr.Sessions {
		fmt.Printf("  %-25s %6d %10d %10d %10d %6.1f%% %7.1f%%\n",
			truncReport(s.Name, 25), s.TotalTurns,
			s.TotalRawTokens, s.TotalCompiled, s.TotalSaved,
			s.AvgReductionPct, s.CoherencyScore*100)
	}

	fmt.Printf("  %s\n", strings.Repeat("-", 78))
	fmt.Printf("  %-25s %6s %10d %10d %10d %6.1f%% %7.1f%%\n",
		"TOTAL", "",
		sr.OverallRawTokens, sr.OverallCompiled, sr.OverallSaved,
		sr.OverallReduction, sr.AvgCoherency*100)

	fmt.Println()
	fmt.Printf("  Total sessions:    %d\n", sr.TotalSessions)
	fmt.Printf("  Total duration:    %s\n", sr.TotalDuration.Round(time.Millisecond))
	fmt.Printf("  Overall reduction: %.1f%%\n", sr.OverallReduction)
	fmt.Printf("  Avg coherency:     %.1f%%\n", sr.AvgCoherency*100)
	fmt.Println()
}

func PrintConcurrencyReport(results []ConcurrencyResult) {
	fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	fmt.Printf("  CONCURRENCY TEST RESULTS\n")
	fmt.Printf("%s\n", strings.Repeat("=", 80))

	allPassed := true
	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
			allPassed = false
		}
		fmt.Printf("\n  [%s] %s\n", status, r.TestName)
		fmt.Printf("    %s\n", r.Details)
		fmt.Printf("    Duration: %s | Operations: %d\n",
			r.Duration.Round(time.Millisecond), r.OperationCount)
	}

	fmt.Println()
	if allPassed {
		fmt.Println("  All concurrency tests PASSED")
	} else {
		fmt.Println("  Some concurrency tests FAILED")
	}
	fmt.Println()
}

func truncReport(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
