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
	_, err := a.rt.CaptureEvent(ctx, eventType, content, nil)
	if err != nil {
		return err
	}
	// Give the session worker a moment to process the event channel.
	time.Sleep(40 * time.Millisecond)
	return nil
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
	time.Sleep(40 * time.Millisecond) // allow async ingest of Run's user/assistant events
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
