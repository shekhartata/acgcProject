package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/shekhartata/acgcProject/pkg/acgc"
)

// acgcPane talks to the ACGC sidecar via the public SDK only.
type acgcPane struct {
	rt           *acgc.ContextRuntime
	budget       int
	transcript   []ChatLine
	systemPrompt string
	seededEvents int
}

func newACGCPane(rt *acgc.ContextRuntime, budget int, systemPrompt string) *acgcPane {
	return &acgcPane{rt: rt, budget: budget, systemPrompt: systemPrompt}
}

func (a *acgcPane) seed(ctx context.Context, role, content string) error {
	a.transcript = append(a.transcript, ChatLine{Role: role, Content: content})
	eventType := "user_prompt"
	if role == "assistant" {
		eventType = "llm_response"
	}
	if _, err := a.rt.CaptureEvent(ctx, eventType, content, nil); err != nil {
		return err
	}
	a.seededEvents++
	return nil
}

// waitForSeededNodes blocks until GetState reports at least root + seeded event nodes.
// Under semantic embed latency (or 429s) each event can take hundreds of ms, so we
// allow a long deadline and keep waiting while the node count is still climbing.
func (a *acgcPane) waitForSeededNodes(ctx context.Context) error {
	minTotal := a.seededEvents + 1 // root + one node per CaptureEvent
	deadline := time.Now().Add(90 * time.Second)
	stallLimit := 8 * time.Second
	var lastTotal int32
	lastProgress := time.Now()
	for time.Now().Before(deadline) {
		st, err := a.rt.GetState(ctx)
		if err != nil {
			return fmt.Errorf("get state while waiting for seed: %w", err)
		}
		if ts := st.GetTreeStats(); ts != nil {
			n := ts.GetTotalNodes()
			if n > lastTotal {
				lastTotal = n
				lastProgress = time.Now()
			}
			if int(lastTotal) >= minTotal {
				return nil
			}
			// Channel drained / worker stuck — don't burn the full deadline.
			if lastTotal > 0 && time.Since(lastProgress) > stallLimit {
				return fmt.Errorf("ACGC seed stalled at %d/%d nodes (no progress for %s); check server logs for dropped events or embed errors",
					lastTotal, minTotal, stallLimit)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for ACGC seed: have %d nodes, want >= %d (seeded %d events)",
		lastTotal, minTotal, a.seededEvents)
}

func (a *acgcPane) run(ctx context.Context, userMessage string) (assistant string, stats ACGCStats) {
	a.transcript = append(a.transcript, ChatLine{Role: "user", Content: userMessage})
	stats.TokenBudget = a.budget
	stats.PromptPreview = formatACGCPreview(a.systemPrompt, userMessage)

	result, err := a.rt.Run(ctx, userMessage)
	if err != nil {
		stats.Error = err.Error()
		return "", stats
	}

	assistant = result.Response
	stats.PromptTokens = result.CompiledTokens
	stats.OriginalTokens = result.OriginalTokens
	stats.ReductionPct = result.ReductionPercent
	stats.ActiveNodes = result.ActiveNodes
	stats.CompressedNodes = result.CompressedNodes
	stats.ArchivedNodes = result.ArchivedNodes
	stats.GCTriggered = result.GCTriggered
	stats.GCReason = result.GCReason
	stats.PromptPreview = formatACGCPreview(a.systemPrompt, userMessage) +
		fmt.Sprintf("\n\n---\n[server stats] original=%d compiled=%d saved=%d (%.1f%%) active=%d compressed=%d archived=%d",
			result.OriginalTokens, result.CompiledTokens, result.TokensSaved, result.ReductionPercent,
			result.ActiveNodes, result.CompressedNodes, result.ArchivedNodes)

	a.transcript = append(a.transcript, ChatLine{Role: "assistant", Content: assistant})
	return assistant, stats
}

func formatACGCPreview(system, question string) string {
	var b string
	if system != "" {
		b += "[system]\n" + system + "\n\n"
	}
	b += "[user: FinalPrompt]\n(compiled on ACGC server — structured goals/constraints/decisions/context within budget)\n\n"
	b += "[user: current]\n" + question
	return FormatPreview(b, 4000)
}

func (a *acgcPane) close() {
	if a.rt != nil {
		_ = a.rt.Close()
	}
}
