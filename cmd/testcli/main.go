package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	pb "github.com/shekhartata/acgcProject/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	banner = `
╔═══════════════════════════════════════════════════════════════╗
║           ACGC Interactive Test Client                        ║
║           Agent Context Garbage Collector                     ║
╠═══════════════════════════════════════════════════════════════╣
║  Commands:                                                    ║
║    (just type)  Send a message through ACGC → LLM → response ║
║    /state       Show current state tree stats                 ║
║    /metrics     Show session metrics & token savings           ║
║    /gc          Manually trigger garbage collection            ║
║    /help        Show this help                                ║
║    /quit        Exit                                          ║
╚═══════════════════════════════════════════════════════════════╝`

	divider = "───────────────────────────────────────────────────"
)

func main() {
	serverAddr := "localhost:50051"
	if v := os.Getenv("ACGC_SERVER"); v != "" {
		serverAddr = v
	}

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to ACGC server at %s: %v\n", serverAddr, err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewACGCServiceClient(conn)
	sessionID := fmt.Sprintf("test_%d", time.Now().Unix())

	fmt.Println(banner)
	fmt.Printf("\n  Server:     %s\n", serverAddr)
	fmt.Printf("  Session ID: %s\n\n", sessionID)

	// Cumulative tracking for the session
	var turnNumber int
	var totalOriginal, totalCompiled, totalSaved int

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch {
		case input == "/quit" || input == "/exit":
			fmt.Println("\nGoodbye.")
			return

		case input == "/help":
			fmt.Println(banner)

		case input == "/state":
			showState(client, sessionID)

		case input == "/metrics":
			showMetrics(client, sessionID, totalOriginal, totalCompiled, totalSaved)

		case input == "/gc":
			triggerGC(client, sessionID)

		default:
			turnNumber++
			orig, comp, saved := sendMessage(client, sessionID, input, turnNumber)
			totalOriginal += orig
			totalCompiled += comp
			totalSaved += saved
		}
	}
}

func sendMessage(client pb.ACGCServiceClient, sessionID, message string, turn int) (original, compiled, saved int) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	start := time.Now()

	resp, err := client.Run(ctx, &pb.RunRequest{
		SessionId:   sessionID,
		TaskId:      "interactive-test",
		UserMessage: message,
		TokenBudget: 6000,
		Policy:      "balanced",
		ConversationHistory: []*pb.Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant. Keep responses concise but informative.",
			},
		},
	})
	if err != nil {
		fmt.Printf("\n  ERROR: %v\n\n", err)
		return 0, 0, 0
	}

	elapsed := time.Since(start)

	// Show LLM response
	fmt.Printf("\n%s\n", divider)
	fmt.Printf("  LLM Response (turn %d, %dms):\n", turn, elapsed.Milliseconds())
	fmt.Printf("%s\n\n", divider)

	// Word-wrap the response for readability
	printWrapped(resp.LlmResponse, 70, "  ")

	// Show stats
	fmt.Printf("\n%s\n", divider)
	fmt.Printf("  ACGC Stats (turn %d):\n", turn)
	fmt.Printf("%s\n", divider)

	if resp.Stats != nil {
		original = int(resp.Stats.OriginalTokenCount)
		compiled = int(resp.Stats.CompiledTokenCount)
		saved = int(resp.Stats.TokensSaved)

		fmt.Printf("  Original tokens:  %d\n", original)
		fmt.Printf("  Compiled tokens:  %d\n", compiled)
		fmt.Printf("  Tokens saved:     %d", saved)
		if saved > 0 {
			fmt.Printf(" (%.1f%% reduction)", resp.Stats.ReductionPercent)
		}
		fmt.Println()

		fmt.Printf("  Active nodes:     %d\n", resp.Stats.ActiveNodes)
		if resp.Stats.CompressedNodes > 0 {
			fmt.Printf("  Compressed nodes: %d\n", resp.Stats.CompressedNodes)
		}
		if resp.Stats.ArchivedNodes > 0 {
			fmt.Printf("  Archived nodes:   %d\n", resp.Stats.ArchivedNodes)
		}
		if resp.Stats.GcTriggered {
			fmt.Printf("  GC triggered:     YES (%s)\n", resp.Stats.GcReason)
		}
	}
	fmt.Printf("%s\n\n", divider)

	return original, compiled, saved
}

func showState(client pb.ACGCServiceClient, sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetState(ctx, &pb.GetStateRequest{
		SessionId:      sessionID,
		IncludeScores:  true,
	})
	if err != nil {
		fmt.Printf("\n  ERROR: %v\n\n", err)
		return
	}

	fmt.Printf("\n%s\n", divider)
	fmt.Printf("  State Tree: %s\n", sessionID)
	fmt.Printf("%s\n", divider)

	if resp.TreeStats != nil {
		fmt.Printf("  Total nodes:      %d\n", resp.TreeStats.TotalNodes)
		fmt.Printf("  Active nodes:     %d\n", resp.TreeStats.ActiveNodes)
		fmt.Printf("  Compressed nodes: %d\n", resp.TreeStats.CompressedNodes)
		fmt.Printf("  Archived nodes:   %d\n", resp.TreeStats.ArchivedNodes)
		fmt.Printf("  Max depth:        %d\n", resp.TreeStats.MaxDepth)
		fmt.Printf("  Max width:        %d\n", resp.TreeStats.MaxWidth)
	} else {
		fmt.Println("  (no state tree yet)")
	}
	fmt.Printf("%s\n\n", divider)
}

func showMetrics(client pb.ACGCServiceClient, sessionID string, totalOrig, totalComp, totalSaved int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetMetrics(ctx, &pb.GetMetricsRequest{
		SessionId: sessionID,
	})
	if err != nil {
		fmt.Printf("\n  ERROR: %v\n\n", err)
		return
	}

	fmt.Printf("\n%s\n", divider)
	fmt.Printf("  Session Metrics: %s\n", sessionID)
	fmt.Printf("%s\n", divider)
	fmt.Printf("  Total events:         %d\n", resp.TotalEvents)
	fmt.Printf("  Total turns:          %d\n", resp.TotalTurns)
	fmt.Printf("  GC runs:              %d\n", resp.GcRuns)
	fmt.Printf("  Branches compressed:  %d\n", resp.BranchesCompressed)

	fmt.Printf("\n  Cumulative Token Stats (this session):\n")
	fmt.Printf("    Total original:     %d tokens\n", totalOrig)
	fmt.Printf("    Total compiled:     %d tokens\n", totalComp)
	fmt.Printf("    Total saved:        %d tokens\n", totalSaved)
	if totalOrig > 0 {
		pct := float64(totalSaved) / float64(totalOrig) * 100
		fmt.Printf("    Overall reduction:  %.1f%%\n", pct)
	}
	fmt.Printf("%s\n\n", divider)
}

func triggerGC(client pb.ACGCServiceClient, sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.TriggerGC(ctx, &pb.TriggerGCRequest{
		SessionId: sessionID,
		Force:     true,
	})
	if err != nil {
		fmt.Printf("\n  ERROR: %v\n\n", err)
		return
	}

	fmt.Printf("\n%s\n", divider)
	fmt.Printf("  Garbage Collection\n")
	fmt.Printf("%s\n", divider)
	fmt.Printf("  Triggered:  %v\n", resp.Triggered)
	fmt.Printf("  Reason:     %s\n", resp.Reason)
	if resp.NodesSwept > 0 {
		fmt.Printf("  Swept:      %d nodes\n", resp.NodesSwept)
	}
	if resp.BranchesCompressed > 0 {
		fmt.Printf("  Compressed: %d branches\n", resp.BranchesCompressed)
	}
	if resp.TokensFreed > 0 {
		fmt.Printf("  Freed:      %d tokens\n", resp.TokensFreed)
	}
	fmt.Printf("%s\n\n", divider)
}

func printWrapped(text string, width int, indent string) {
	words := strings.Fields(text)
	line := indent
	for _, word := range words {
		if len(line)+len(word)+1 > width+len(indent) {
			fmt.Println(line)
			line = indent + word
		} else {
			if line == indent {
				line += word
			} else {
				line += " " + word
			}
		}
	}
	if line != indent {
		fmt.Println(line)
	}
}
