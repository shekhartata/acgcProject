package harness

import "github.com/shekhartata/acgcProject/internal/domain"

// slidingStrategy keeps only the most recent context, filling the token budget
// newest-first. It models the common "last N turns" context window baseline.
type slidingStrategy struct{}

// NewSlidingStrategy returns the sliding_window context strategy.
func NewSlidingStrategy() ContextStrategy { return slidingStrategy{} }

func (slidingStrategy) Name() PipelineKind { return PipelineSliding }

func (slidingStrategy) BuildPrompt(in StrategyInput) (StrategyOutput, error) {
	out := StrategyOutput{}

	original := in.TokenCounter.Count(in.SystemPrompt) + in.TokenCounter.Count(in.Question)
	for _, n := range in.Nodes {
		original += n.TokenCount
	}
	out.OriginalTokenCount = original

	// Walk newest-first, keeping nodes until the budget is hit, then restore
	// chronological order so the rendered context reads naturally.
	var kept []*domain.StateNode
	used := 0
	excluded := 0
	for i := len(in.Nodes) - 1; i >= 0; i-- {
		n := in.Nodes[i]
		if used+n.TokenCount > in.TokenBudget {
			excluded++
			continue
		}
		kept = append(kept, n)
		used += n.TokenCount
	}
	// Reverse kept back into chronological order.
	for l, r := 0, len(kept)-1; l < r; l, r = l+1, r-1 {
		kept[l], kept[r] = kept[r], kept[l]
	}

	out.IncludedNodes = len(kept)
	out.ExcludedNodes = excluded
	if len(kept) > 0 {
		out.FinalPrompt = formatNodesSection(kept)
	}
	out.CompiledTokenCount = in.TokenCounter.Count(in.SystemPrompt) +
		in.TokenCounter.Count(out.FinalPrompt) +
		in.TokenCounter.Count(in.Question)
	return out, nil
}
