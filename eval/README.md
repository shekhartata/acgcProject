# ACGC Quality & Intelligence-Per-Token Evaluation

The `stresstest/` suite proves the GC works (tokens shrink, no races). This
suite proves the **answers stay correct** when the LLM sees a budget-limited
context instead of the full conversation history.

It runs each scenario through a configurable set of **context strategies** and
scores the responses to compute:

```
intelligence_per_token = quality_score / (prompt_tokens / 1000)
```

Higher IPT = more useful answer per token spent.

## Context strategies

All strategies share the same system prompt, token budget, LLM, scoring, and
cache — the only variable is how each one selects context for the probe. The
first strategy in the list is the **reference** every candidate is compared to.

| Strategy | Key | What it does |
|---|---|---|
| Naive full history | `naive_full_history` | Include all history (chronological) up to the token budget. The default reference — "no context management". |
| Sliding window | `sliding_window` | Keep only the most recent context, filling the budget newest-first. |
| ACGC | `acgc` | Full ACGC stack: statetree + scorer + GC + compiler (+ optional HNSW semantic re-blend). |

Select strategies with the `-strategies` flag (comma-separated, first = reference):

```bash
go run ./eval -strategies "naive_full_history,sliding_window,acgc" -v
# or
make eval-strategies
```

The default (`make eval`) compares `naive_full_history,acgc`.

## Accurate token counting

Token accounting uses a real, model-aware BPE tokenizer (`tiktoken-go`,
resolved from the configured model — e.g. `o200k_base` for GPT-4o/GPT-5),
not the old `len(s)/4` approximation. The tokenizer name is printed at the top
of every run and recorded in `results.json`. If the encoding cannot be loaded
for some reason, the harness falls back to the approximate counter automatically.

## How to run

```bash
# 1. Make sure ACGC_LLM_API_KEY is set in .env

# 2. First time — live run, probe scoring only (cheap, deterministic)
make eval

# 3. Replay scored results from cache (no API calls, free)
make eval-cached

# 4. With LLM-as-judge for open-ended scenarios (more tokens, broader coverage)
make eval-judge

# 5. Compare all three context strategies side by side
make eval-strategies

# 6. Run a single scenario
go run ./eval -scenarios=long_range_recall_1 -v

# 7. Budget-capped run (stops after N tokens spent on live calls)
go run ./eval -budget-cap=20000

# 8. Force fresh API calls
make eval-clean && make eval
```

Output:
- `eval/results/report.md` — human-readable report with a side-by-side strategy table, candidate-vs-reference verdicts, and per-strategy response samples
- `eval/results/results.json` — machine-readable full results (per-strategy metrics + pairwise verdicts + tokenizer used)
- `eval/cache/responses_<model>.jsonl` — every LLM response, keyed by `scenario::probe::strategy` for replay

## Scenarios

| ID | Category | What it tests |
|---|---|---|
| `recent_recall_1` | recent_recall | Sanity check — both should ace this |
| `long_range_recall_1` | long_range | 3 facts stated early, ~38 noise turns, then probes for each |
| `constraint_adherence_1` | constraint | Constraints stated upfront, then a request that tempts the model to violate them |
| `topic_switch_return_1` | topic_switch | Topic A, hard pivot to topic B, then callback to A's specific decision |
| `contradiction_1` | contradiction | Decision X then explicit reversal to Y — latest must win |
| `multi_hop_synth_1` | multi_hop | Three facts spread out; probe requires synthesizing all three |

## Verdicts

Verdicts describe a **candidate strategy vs the reference strategy** for a probe:

- `ACGC_WIN` — candidate has same/better quality at fewer tokens (the goal)
- `ACGC_WIN_STAR` — better IPT but quality regressed (motivates semantic vector search)
- `TIE` — roughly equal on quality and IPT
- `ACGC_LOSS` — candidate quality dropped more than 1.0 point
- `BASELINE_WIN` — reference strictly better

(The verdict labels keep the `ACGC_*` names for backward compatibility; they
apply to whichever candidate strategy is being compared to the reference.)

## Cost notes

- Cost scales with the number of strategies: each probe now issues one LLM call
  **per strategy**. Probe-only run, all 6 scenarios × 3 strategies ≈ 8 probes ×
  3 strategies × ~3000 tokens avg ≈ 70K tokens.
- With `-judge` enabled: add one judge call per candidate strategy per open-ended probe.
- All API responses are cached (keyed per strategy), so iterating on scoring logic is free.
