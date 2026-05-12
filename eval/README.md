# ACGC Quality & Intelligence-Per-Token Evaluation

The `stresstest/` suite proves the GC works (tokens shrink, no races). This
suite proves the **answers stay correct** when the LLM sees ACGC's compiled
prompt instead of the full conversation history.

It runs each scenario twice — once with **full history** (baseline) and once
with **ACGC-compiled prompt** — and scores the responses to compute:

```
intelligence_per_token = quality_score / (prompt_tokens / 1000)
```

Higher IPT = more useful answer per token spent.

## How to run

```bash
# 1. Make sure ACGC_LLM_API_KEY is set in .env

# 2. First time — live run, probe scoring only (cheap, deterministic)
make eval

# 3. Replay scored results from cache (no API calls, free)
make eval-cached

# 4. With LLM-as-judge for open-ended scenarios (more tokens, broader coverage)
make eval-judge

# 5. Run a single scenario
go run ./eval -scenarios=long_range_recall_1 -v

# 6. Budget-capped run (stops after N tokens spent on live calls)
go run ./eval -budget-cap=20000

# 7. Force fresh API calls
make eval-clean && make eval
```

Output:
- `eval/results/report.md` — human-readable report with verdicts and side-by-side samples
- `eval/results/results.json` — machine-readable full results
- `eval/cache/responses_<model>.jsonl` — every LLM response, keyed for replay

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

- `ACGC_WIN` — same/better quality at fewer tokens (the goal)
- `ACGC_WIN_STAR` — better IPT but quality regressed (motivates semantic vector search)
- `TIE` — roughly equal on quality and IPT
- `ACGC_LOSS` — quality dropped more than 1.0 point
- `BASELINE_WIN` — baseline strictly better

## Cost notes

- Probe-only run, all 6 scenarios: roughly 8 probes × 2 pipelines × ~3000 tokens avg ≈ 50K tokens (single-digit dollars on GPT-5).
- With `-judge` enabled: add another ~8 judge calls @ ~2000 tokens ≈ 16K tokens extra.
- All API responses are cached, so iterating on scoring logic is free.
