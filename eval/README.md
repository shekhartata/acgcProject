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

### Key flags

| Flag | Default | Purpose |
|---|---|---|
| `-strategies` | `naive_full_history,acgc` | Comma-separated strategies to compare; the first is the reference. |
| `-acgc-budget` | `6000` | Context token budget shared by every strategy — how much history each one may include. |
| `-max-tokens` | `6000` | Max completion tokens for probe answers. Reasoning models (GPT-5/o1/o3) spend part of this on hidden reasoning, so too small a cap yields empty responses; raise it if answers come back blank. |

```bash
# Exercise budget-driven divergence: a dataset larger than the budget, roomy answer cap
go run ./eval -v -strategies "naive_full_history,sliding_window,acgc" \
  -scenarios deep_history_recall_1 -acgc-budget 6000 -max-tokens 6000
```

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
| `deep_history_recall_1` | deep_history | Four decisions up front, then ~85 large filler Q/A pairs (~13-15k raw tokens, >2x the default budget). Purpose-built to exceed the token budget so `naive_full_history` (keeps oldest), `sliding_window` (keeps newest, drops the decisions), and `acgc` (compresses, keeps the decisions cheaply) visibly diverge on tokens and recall. |

## External benchmarks (LongMemEval, LoCoMo)

Besides the built-in scenarios above, the harness can load two published
long-term-memory benchmarks through modular adapters in
`eval/datasets/external/`. Adapters are pure data loaders: each benchmark
instance (or conversation) becomes a regular `Scenario`, so the strategies,
scoring, caching, and reporting pipeline is completely unchanged.

| Source | Key | Mapping |
|---|---|---|
| [LongMemEval](https://github.com/xiaowu0162/LongMemEval) | `longmemeval` | One instance → one scenario: multi-session haystack flattened chronologically, session dates annotated inline, one judge-scored probe against the gold answer. Instances whose ID ends in `_abs` use an abstention rubric (correct = "not in the conversation"). |
| [LoCoMo](https://github.com/snap-research/locomo) | `locomo` | One conversation → one scenario: speaker A maps to `user`, speaker B to `assistant`, every turn prefixed with the speaker's name, image turns replaced by their captions. All (or a sampled cap of) QA pairs become judge-scored probes; category 5 (adversarial) uses the abstention rubric. |

Data files are **not vendored** (size + licensing). Fetch them once:

```bash
make eval-fetch-external          # downloads into eval/datasets/external/data/ (gitignored)
```

Then run:

```bash
make eval-longmemeval             # 20 sampled instances, judge-scored, 3 strategies
make eval-locomo                  # 10 conversations, 20 sampled probes each

# or hand-rolled:
go run ./eval -v -judge \
  -external "longmemeval=eval/datasets/external/data/longmemeval_s_cleaned.json" \
  -external-sample 10 -external-seed 42 \
  -external-types "multi-session,temporal-reasoning"
```

### External flags

| Flag | Default | Purpose |
|---|---|---|
| `-external` | (empty) | `name=path` pairs, comma-separated. Registered names: `longmemeval`, `locomo`. Requires `-judge` on live runs (all external probes are judge-scored). |
| `-external-sample` | `20` | Cap per source: instances for LongMemEval, probes per conversation for LoCoMo. `0` = all. |
| `-external-seed` | `42` | Seed for deterministic subsampling — the same seed always selects the same subset, keeping cache keys stable across runs. |
| `-external-types` | (empty) | Filter by question type (LongMemEval: `single-session-user`, `multi-session`, `temporal-reasoning`, `knowledge-update`, ...) or category (LoCoMo: `single_hop`, `multi_hop`, `temporal`, `open_domain`, `adversarial`). |

External runs evaluate **only** the external scenarios and write to their own
prefixed files, so the built-in report is never clobbered:

- `eval/results/external_longmemeval_report.md` / `..._results.json`
- `eval/results/external_locomo_report.md` / `..._results.json`
- (a mixed run writes `external_longmemeval_locomo_*`)

Cost warning: external instances are much larger than the built-in scenarios
(LongMemEval-S is ~115k history tokens per instance before budgeting). Start
with a small `-external-sample` and/or a `-budget-cap`, and remember every
response is cached for free replay.

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
