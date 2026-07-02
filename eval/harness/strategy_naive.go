package harness

// naiveStrategy includes all available historical context in chronological
// order until the token budget is exhausted. This is the "no context
// management" reference every other strategy is measured against.
type naiveStrategy struct{}

// NewNaiveStrategy returns the naive_full_history context strategy.
func NewNaiveStrategy() ContextStrategy { return naiveStrategy{} }

func (naiveStrategy) Name() PipelineKind { return PipelineNaive }

func (naiveStrategy) BuildPrompt(in StrategyInput) (StrategyOutput, error) {
	out := StrategyOutput{}

	original := in.TokenCounter.Count(in.SystemPrompt) + in.TokenCounter.Count(in.Question)
	for _, n := range in.Nodes {
		original += n.TokenCount
	}
	out.OriginalTokenCount = original

	included, excluded, _ := selectChronological(in.Nodes, in.TokenBudget)
	out.IncludedNodes = len(included)
	out.ExcludedNodes = excluded

	if len(included) > 0 {
		out.FinalPrompt = formatNodesSection(included)
	}
	out.CompiledTokenCount = in.TokenCounter.Count(in.SystemPrompt) +
		in.TokenCounter.Count(out.FinalPrompt) +
		in.TokenCounter.Count(in.Question)
	return out, nil
}
