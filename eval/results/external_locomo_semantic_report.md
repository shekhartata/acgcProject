# ACGC Context-Strategy Evaluation

**Generated:** 2026-07-04T14:36:14+05:30  
**Model:** `gpt-5`  
**Tokenizer:** `o200k_base`  
**Reference strategy:** `naive_full_history`  
**Strategies compared:** `naive_full_history`, `sliding_window`, `acgc`  
**Live tokens spent this run:** 0  

## Strategy comparison (side by side)

| Strategy | Probes | Avg Quality | Avg Prompt Tok | Avg Latency (ms) | Avg IPT | Tok Red% vs ref | Quality Δ vs ref | IPT Δ% vs ref |
|---|---|---|---|---|---|---|---|---|
| `naive_full_history (ref)` | 100 | 2.78 | 6734 | 6366 | 0.41 | 0.0% | +0.00 | +0.0% |
| `sliding_window` | 100 | 3.27 | 6761 | 6808 | 0.48 | -0.4% | +0.49 | +17.1% |
| `acgc` | 100 | 3.13 | 6123 | 11159 | 0.51 | 9.1% | +0.35 | +23.7% |

## Candidate vs reference (verdicts)

Reference: `naive_full_history`

- **Pairs evaluated:** 200
- **Avg quality (reference):** 2.78 / 5.0
- **Avg quality (candidate):** 3.20 / 5.0
- **Avg quality delta:** +0.42 (candidate - reference)
- **Avg token reduction:** 4.3%
- **Avg IPT (reference):** 0.41
- **Avg IPT (candidate):** 0.50
- **Avg IPT delta:** +20.4%
- **Quality regressions (>1.0 drop):** 34

### Verdict breakdown

- `ACGC_WIN` (better IPT, no quality loss): **130**
- `ACGC_WIN_STAR` (better IPT, but quality dropped): **0**
- `TIE`: **36**
- `ACGC_LOSS`: **34**
- `BASELINE_WIN` (reference strictly better): **0**

## Per-probe results

| Scenario / Probe | Candidate | Method | Quality (ref / cand) | Tokens (ref / cand) | Token Red% | IPT (ref / cand) | IPT Δ% | Verdict |
|---|---|---|---|---|---|---|---|---|
| `locomo_conv-26` / `q102_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6600 / 6032 | 8.6% | 0.76 / 0.83 | +9.4% | ACGC_WIN |
| `locomo_conv-26` / `q102_single_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6600 / 6599 | 0.0% | 0.76 / 0.76 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q103_single_hop` | `acgc` | judge | 5.0 / 1.0 | 6599 / 6034 | 8.6% | 0.76 / 0.17 | -78.1% | ACGC_LOSS |
| `locomo_conv-26` / `q103_single_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6599 / 6598 | 0.0% | 0.76 / 0.15 | -80.0% | ACGC_LOSS |
| `locomo_conv-26` / `q11_temporal` | `acgc` | judge | 5.0 / 1.0 | 6600 / 6034 | 8.6% | 0.76 / 0.17 | -78.1% | ACGC_LOSS |
| `locomo_conv-26` / `q11_temporal` | `sliding_window` | judge | 5.0 / 2.0 | 6600 / 6599 | 0.0% | 0.76 / 0.30 | -60.0% | ACGC_LOSS |
| `locomo_conv-26` / `q132_single_hop` | `acgc` | judge | 1.0 / 1.0 | 6600 / 6032 | 8.6% | 0.15 / 0.17 | +9.4% | ACGC_WIN |
| `locomo_conv-26` / `q132_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6600 / 6599 | 0.0% | 0.15 / 0.76 | +400.1% | ACGC_WIN |
| `locomo_conv-26` / `q137_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6601 / 6033 | 8.6% | 0.30 / 0.83 | +173.5% | ACGC_WIN |
| `locomo_conv-26` / `q137_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6601 / 6600 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q145_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6597 / 6032 | 8.6% | 0.30 / 0.33 | +9.4% | ACGC_WIN |
| `locomo_conv-26` / `q145_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6597 / 6596 | 0.0% | 0.30 / 0.30 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q147_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6597 / 6035 | 8.5% | 0.30 / 0.83 | +173.3% | ACGC_WIN |
| `locomo_conv-26` / `q147_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6597 / 6596 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q148_single_hop` | `acgc` | judge | 1.0 / 5.0 | 6596 / 6023 | 8.7% | 0.15 / 0.83 | +447.6% | ACGC_WIN |
| `locomo_conv-26` / `q148_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6596 / 6595 | 0.0% | 0.15 / 0.76 | +400.1% | ACGC_WIN |
| `locomo_conv-26` / `q152_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6599 / 6033 | 8.6% | 0.30 / 0.33 | +9.4% | ACGC_WIN |
| `locomo_conv-26` / `q152_single_hop` | `sliding_window` | judge | 2.0 / 1.0 | 6599 / 6598 | 0.0% | 0.30 / 0.15 | -50.0% | TIE |
| `locomo_conv-26` / `q168_adversarial` | `acgc` | judge | 2.0 / 1.0 | 6600 / 6037 | 8.5% | 0.30 / 0.17 | -45.3% | TIE |
| `locomo_conv-26` / `q168_adversarial` | `sliding_window` | judge | 2.0 / 2.0 | 6600 / 6599 | 0.0% | 0.30 / 0.30 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q182_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6597 / 6025 | 8.7% | 0.76 / 0.83 | +9.5% | ACGC_WIN |
| `locomo_conv-26` / `q182_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6597 / 6596 | 0.0% | 0.76 / 0.30 | -60.0% | ACGC_LOSS |
| `locomo_conv-26` / `q43_open_domain` | `acgc` | judge | 5.0 / 5.0 | 6604 / 6030 | 8.7% | 0.76 / 0.83 | +9.5% | ACGC_WIN |
| `locomo_conv-26` / `q43_open_domain` | `sliding_window` | judge | 5.0 / 5.0 | 6604 / 6603 | 0.0% | 0.76 / 0.76 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q47_open_domain` | `acgc` | judge | 5.0 / 5.0 | 6599 / 6041 | 8.5% | 0.76 / 0.83 | +9.2% | ACGC_WIN |
| `locomo_conv-26` / `q47_open_domain` | `sliding_window` | judge | 5.0 / 5.0 | 6599 / 6598 | 0.0% | 0.76 / 0.76 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q48_multi_hop` | `acgc` | judge | 5.0 / 3.0 | 6598 / 6034 | 8.5% | 0.76 / 0.50 | -34.4% | ACGC_LOSS |
| `locomo_conv-26` / `q48_multi_hop` | `sliding_window` | judge | 5.0 / 3.0 | 6598 / 6597 | 0.0% | 0.76 / 0.45 | -40.0% | ACGC_LOSS |
| `locomo_conv-26` / `q4_multi_hop` | `acgc` | judge | 5.0 / 2.0 | 6593 / 6037 | 8.4% | 0.76 / 0.33 | -56.3% | ACGC_LOSS |
| `locomo_conv-26` / `q4_multi_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6593 / 6592 | 0.0% | 0.76 / 0.30 | -60.0% | ACGC_LOSS |
| `locomo_conv-26` / `q54_temporal` | `acgc` | judge | 2.0 / 5.0 | 6596 / 6035 | 8.5% | 0.30 / 0.83 | +173.2% | ACGC_WIN |
| `locomo_conv-26` / `q54_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6596 / 6595 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q55_temporal` | `acgc` | judge | 2.0 / 2.0 | 6597 / 6034 | 8.5% | 0.30 / 0.33 | +9.3% | ACGC_WIN |
| `locomo_conv-26` / `q55_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6597 / 6596 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q64_temporal` | `acgc` | judge | 2.0 / 2.0 | 6600 / 6036 | 8.5% | 0.30 / 0.33 | +9.3% | ACGC_WIN |
| `locomo_conv-26` / `q64_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6600 / 6599 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q73_temporal` | `acgc` | judge | 1.0 / 2.0 | 6597 / 6036 | 8.5% | 0.15 / 0.33 | +118.6% | ACGC_WIN |
| `locomo_conv-26` / `q73_temporal` | `sliding_window` | judge | 1.0 / 2.0 | 6597 / 6596 | 0.0% | 0.15 / 0.30 | +100.0% | ACGC_WIN |
| `locomo_conv-26` / `q82_open_domain` | `acgc` | judge | 2.0 / 5.0 | 6600 / 6034 | 8.6% | 0.30 / 0.83 | +173.5% | ACGC_WIN |
| `locomo_conv-26` / `q82_open_domain` | `sliding_window` | judge | 2.0 / 4.0 | 6600 / 6599 | 0.0% | 0.30 / 0.61 | +100.0% | ACGC_WIN |
| `locomo_conv-30` / `q100_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6713 / 6073 | 9.5% | 0.74 / 0.82 | +10.5% | ACGC_WIN |
| `locomo_conv-30` / `q100_adversarial` | `sliding_window` | judge | 5.0 / 5.0 | 6713 / 6800 | -1.3% | 0.74 / 0.74 | -1.3% | TIE |
| `locomo_conv-30` / `q16_temporal` | `acgc` | judge | 5.0 / 2.0 | 6714 / 6056 | 9.8% | 0.74 / 0.33 | -55.7% | ACGC_LOSS |
| `locomo_conv-30` / `q16_temporal` | `sliding_window` | judge | 5.0 / 2.0 | 6714 / 6801 | -1.3% | 0.74 / 0.29 | -60.5% | ACGC_LOSS |
| `locomo_conv-30` / `q18_multi_hop` | `acgc` | judge | 4.0 / 4.0 | 6717 / 6058 | 9.8% | 0.60 / 0.66 | +10.9% | ACGC_WIN |
| `locomo_conv-30` / `q18_multi_hop` | `sliding_window` | judge | 4.0 / 3.0 | 6717 / 6804 | -1.3% | 0.60 / 0.44 | -26.0% | TIE |
| `locomo_conv-30` / `q1_temporal` | `acgc` | judge | 5.0 / 2.0 | 6716 / 6059 | 9.8% | 0.74 / 0.33 | -55.7% | ACGC_LOSS |
| `locomo_conv-30` / `q1_temporal` | `sliding_window` | judge | 5.0 / 1.0 | 6716 / 6803 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q25_multi_hop` | `acgc` | judge | 2.0 / 3.0 | 6718 / 6069 | 9.7% | 0.30 / 0.49 | +66.0% | ACGC_WIN |
| `locomo_conv-30` / `q25_multi_hop` | `sliding_window` | judge | 2.0 / 3.0 | 6718 / 6805 | -1.3% | 0.30 / 0.44 | +48.1% | ACGC_WIN |
| `locomo_conv-30` / `q27_temporal` | `acgc` | judge | 2.0 / 2.0 | 6716 / 6032 | 10.2% | 0.30 / 0.33 | +11.3% | ACGC_WIN |
| `locomo_conv-30` / `q27_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6716 / 6803 | -1.3% | 0.30 / 0.73 | +146.8% | ACGC_WIN |
| `locomo_conv-30` / `q33_temporal` | `acgc` | judge | 1.0 / 1.0 | 6717 / 6056 | 9.8% | 0.15 / 0.17 | +10.9% | ACGC_WIN |
| `locomo_conv-30` / `q33_temporal` | `sliding_window` | judge | 1.0 / 5.0 | 6717 / 6804 | -1.3% | 0.15 / 0.73 | +393.6% | ACGC_WIN |
| `locomo_conv-30` / `q36_temporal` | `acgc` | judge | 2.0 / 1.0 | 6716 / 6062 | 9.7% | 0.30 / 0.16 | -44.6% | TIE |
| `locomo_conv-30` / `q36_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6716 / 6803 | -1.3% | 0.30 / 0.73 | +146.8% | ACGC_WIN |
| `locomo_conv-30` / `q47_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6719 / 6062 | 9.8% | 0.74 / 0.82 | +10.8% | ACGC_WIN |
| `locomo_conv-30` / `q47_single_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6719 / 6806 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q51_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6718 / 6067 | 9.7% | 0.74 / 0.82 | +10.7% | ACGC_WIN |
| `locomo_conv-30` / `q51_single_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6718 / 6805 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q55_single_hop` | `acgc` | judge | 5.0 / 3.0 | 6713 / 6051 | 9.9% | 0.74 / 0.50 | -33.4% | ACGC_LOSS |
| `locomo_conv-30` / `q55_single_hop` | `sliding_window` | judge | 5.0 / 4.0 | 6713 / 6800 | -1.3% | 0.74 / 0.59 | -21.0% | TIE |
| `locomo_conv-30` / `q56_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6713 / 6043 | 10.0% | 0.74 / 0.83 | +11.1% | ACGC_WIN |
| `locomo_conv-30` / `q56_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6713 / 6800 | -1.3% | 0.74 / 0.29 | -60.5% | ACGC_LOSS |
| `locomo_conv-30` / `q60_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6716 / 6046 | 10.0% | 0.74 / 0.83 | +11.1% | ACGC_WIN |
| `locomo_conv-30` / `q60_single_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6716 / 6803 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q62_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6715 / 6067 | 9.7% | 0.74 / 0.82 | +10.7% | ACGC_WIN |
| `locomo_conv-30` / `q62_single_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6715 / 6802 | -1.3% | 0.74 / 0.74 | -1.3% | TIE |
| `locomo_conv-30` / `q6_multi_hop` | `acgc` | judge | 5.0 / 5.0 | 6717 / 6057 | 9.8% | 0.74 / 0.83 | +10.9% | ACGC_WIN |
| `locomo_conv-30` / `q6_multi_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6717 / 6804 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q71_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6713 / 6050 | 9.9% | 0.74 / 0.83 | +11.0% | ACGC_WIN |
| `locomo_conv-30` / `q71_single_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6713 / 6800 | -1.3% | 0.74 / 0.74 | -1.3% | TIE |
| `locomo_conv-30` / `q87_adversarial` | `acgc` | judge | 1.0 / 0.0 | 6719 / 6059 | 9.8% | 0.15 / 0.00 | -100.0% | TIE |
| `locomo_conv-30` / `q87_adversarial` | `sliding_window` | judge | 1.0 / 5.0 | 6719 / 6806 | -1.3% | 0.15 / 0.73 | +393.6% | ACGC_WIN |
| `locomo_conv-30` / `q91_adversarial` | `acgc` | judge | 2.0 / 2.0 | 6718 / 6064 | 9.7% | 0.30 / 0.33 | +10.8% | ACGC_WIN |
| `locomo_conv-30` / `q91_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6718 / 6805 | -1.3% | 0.30 / 0.73 | +146.8% | ACGC_WIN |
| `locomo_conv-30` / `q94_adversarial` | `acgc` | judge | 1.0 / 1.0 | 6715 / 6060 | 9.8% | 0.15 / 0.17 | +10.8% | ACGC_WIN |
| `locomo_conv-30` / `q94_adversarial` | `sliding_window` | judge | 1.0 / 5.0 | 6715 / 6802 | -1.3% | 0.15 / 0.74 | +393.6% | ACGC_WIN |
| `locomo_conv-30` / `q97_adversarial` | `acgc` | judge | 5.0 / 2.0 | 6725 / 6055 | 10.0% | 0.74 / 0.33 | -55.6% | ACGC_LOSS |
| `locomo_conv-30` / `q97_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6725 / 6812 | -1.3% | 0.74 / 0.29 | -60.5% | ACGC_LOSS |
| `locomo_conv-41` / `q101_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6821 / 6227 | 8.7% | 0.29 / 0.32 | +9.5% | ACGC_WIN |
| `locomo_conv-41` / `q101_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6821 / 6830 | -0.1% | 0.29 / 0.29 | -0.1% | TIE |
| `locomo_conv-41` / `q10_multi_hop` | `acgc` | judge | 5.0 / 5.0 | 6827 / 6217 | 8.9% | 0.73 / 0.80 | +9.8% | ACGC_WIN |
| `locomo_conv-41` / `q10_multi_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6827 / 6836 | -0.1% | 0.73 / 0.15 | -80.0% | ACGC_LOSS |
| `locomo_conv-41` / `q11_temporal` | `acgc` | judge | 5.0 / 1.0 | 6814 / 6227 | 8.6% | 0.73 / 0.16 | -78.1% | ACGC_LOSS |
| `locomo_conv-41` / `q11_temporal` | `sliding_window` | judge | 5.0 / 2.0 | 6814 / 6823 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q131_single_hop` | `acgc` | judge | 1.0 / 0.0 | 6820 / 6217 | 8.8% | 0.15 / 0.00 | -100.0% | TIE |
| `locomo_conv-41` / `q131_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6820 / 6829 | -0.1% | 0.15 / 0.73 | +399.3% | ACGC_WIN |
| `locomo_conv-41` / `q140_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6822 / 6228 | 8.7% | 0.15 / 0.32 | +119.1% | ACGC_WIN |
| `locomo_conv-41` / `q140_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6822 / 6831 | -0.1% | 0.15 / 0.73 | +399.3% | ACGC_WIN |
| `locomo_conv-41` / `q158_adversarial` | `acgc` | judge | 2.0 / 5.0 | 6820 / 6225 | 8.7% | 0.29 / 0.80 | +173.9% | ACGC_WIN |
| `locomo_conv-41` / `q158_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6820 / 6829 | -0.1% | 0.29 / 0.73 | +149.7% | ACGC_WIN |
| `locomo_conv-41` / `q161_adversarial` | `acgc` | judge | 2.0 / 2.0 | 6823 / 6220 | 8.8% | 0.29 / 0.32 | +9.7% | ACGC_WIN |
| `locomo_conv-41` / `q161_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6823 / 6832 | -0.1% | 0.29 / 0.73 | +149.7% | ACGC_WIN |
| `locomo_conv-41` / `q162_adversarial` | `acgc` | judge | 5.0 / 2.0 | 6816 / 6231 | 8.6% | 0.73 / 0.32 | -56.2% | ACGC_LOSS |
| `locomo_conv-41` / `q162_adversarial` | `sliding_window` | judge | 5.0 / 4.0 | 6816 / 6825 | -0.1% | 0.73 / 0.59 | -20.1% | TIE |
| `locomo_conv-41` / `q167_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6822 / 6223 | 8.8% | 0.73 / 0.80 | +9.6% | ACGC_WIN |
| `locomo_conv-41` / `q167_adversarial` | `sliding_window` | judge | 5.0 / 5.0 | 6822 / 6831 | -0.1% | 0.73 / 0.73 | -0.1% | TIE |
| `locomo_conv-41` / `q179_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6822 / 6229 | 8.7% | 0.73 / 0.80 | +9.5% | ACGC_WIN |
| `locomo_conv-41` / `q179_adversarial` | `sliding_window` | judge | 5.0 / 4.0 | 6822 / 6831 | -0.1% | 0.73 / 0.59 | -20.1% | TIE |
| `locomo_conv-41` / `q183_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6824 / 6227 | 8.7% | 0.73 / 0.80 | +9.6% | ACGC_WIN |
| `locomo_conv-41` / `q183_adversarial` | `sliding_window` | judge | 5.0 / 1.0 | 6824 / 6833 | -0.1% | 0.73 / 0.15 | -80.0% | ACGC_LOSS |
| `locomo_conv-41` / `q38_multi_hop` | `acgc` | judge | 2.0 / 3.0 | 6817 / 6219 | 8.8% | 0.29 / 0.48 | +64.4% | ACGC_WIN |
| `locomo_conv-41` / `q38_multi_hop` | `sliding_window` | judge | 2.0 / 3.0 | 6817 / 6826 | -0.1% | 0.29 / 0.44 | +49.8% | ACGC_WIN |
| `locomo_conv-41` / `q42_open_domain` | `acgc` | judge | 2.0 / 2.0 | 6819 / 6215 | 8.9% | 0.29 / 0.32 | +9.7% | ACGC_WIN |
| `locomo_conv-41` / `q42_open_domain` | `sliding_window` | judge | 2.0 / 5.0 | 6819 / 6828 | -0.1% | 0.29 / 0.73 | +149.7% | ACGC_WIN |
| `locomo_conv-41` / `q43_multi_hop` | `acgc` | judge | 2.0 / 2.0 | 6816 / 6221 | 8.7% | 0.29 / 0.32 | +9.6% | ACGC_WIN |
| `locomo_conv-41` / `q43_multi_hop` | `sliding_window` | judge | 2.0 / 1.0 | 6816 / 6825 | -0.1% | 0.29 / 0.15 | -50.1% | TIE |
| `locomo_conv-41` / `q44_temporal` | `acgc` | judge | 1.0 / 5.0 | 6818 / 6226 | 8.7% | 0.15 / 0.80 | +447.5% | ACGC_WIN |
| `locomo_conv-41` / `q44_temporal` | `sliding_window` | judge | 1.0 / 5.0 | 6818 / 6827 | -0.1% | 0.15 / 0.73 | +399.3% | ACGC_WIN |
| `locomo_conv-41` / `q4_multi_hop` | `acgc` | judge | 5.0 / 2.0 | 6819 / 6230 | 8.6% | 0.73 / 0.32 | -56.2% | ACGC_LOSS |
| `locomo_conv-41` / `q4_multi_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6819 / 6828 | -0.1% | 0.73 / 0.15 | -80.0% | ACGC_LOSS |
| `locomo_conv-41` / `q57_temporal` | `acgc` | judge | 1.0 / 2.0 | 6820 / 6230 | 8.7% | 0.15 / 0.32 | +118.9% | ACGC_WIN |
| `locomo_conv-41` / `q57_temporal` | `sliding_window` | judge | 1.0 / 5.0 | 6820 / 6829 | -0.1% | 0.15 / 0.73 | +399.3% | ACGC_WIN |
| `locomo_conv-41` / `q70_single_hop` | `acgc` | judge | 3.0 / 2.0 | 6822 / 6227 | 8.7% | 0.44 / 0.32 | -27.0% | TIE |
| `locomo_conv-41` / `q70_single_hop` | `sliding_window` | judge | 3.0 / 2.0 | 6822 / 6831 | -0.1% | 0.44 / 0.29 | -33.4% | TIE |
| `locomo_conv-41` / `q78_single_hop` | `acgc` | judge | 5.0 / 2.0 | 6820 / 6231 | 8.6% | 0.73 / 0.32 | -56.2% | ACGC_LOSS |
| `locomo_conv-41` / `q78_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6820 / 6829 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q89_single_hop` | `acgc` | judge | 2.0 / 1.0 | 6817 / 6217 | 8.8% | 0.29 / 0.16 | -45.2% | TIE |
| `locomo_conv-41` / `q89_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6817 / 6826 | -0.1% | 0.29 / 0.29 | -0.1% | TIE |
| `locomo_conv-42` / `q158_single_hop` | `acgc` | judge | 1.0 / 5.0 | 6720 / 6055 | 9.9% | 0.15 / 0.83 | +454.9% | ACGC_WIN |
| `locomo_conv-42` / `q158_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6720 / 6690 | 0.4% | 0.15 / 0.75 | +402.2% | ACGC_WIN |
| `locomo_conv-42` / `q177_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6720 / 6065 | 9.7% | 0.30 / 0.82 | +177.0% | ACGC_WIN |
| `locomo_conv-42` / `q177_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6720 / 6690 | 0.4% | 0.30 / 0.75 | +151.1% | ACGC_WIN |
| `locomo_conv-42` / `q181_single_hop` | `acgc` | judge | 1.0 / 3.0 | 6706 / 6057 | 9.7% | 0.15 / 0.50 | +232.1% | ACGC_WIN |
| `locomo_conv-42` / `q181_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6706 / 6676 | 0.4% | 0.15 / 0.75 | +402.2% | ACGC_WIN |
| `locomo_conv-42` / `q185_single_hop` | `acgc` | judge | 1.0 / 4.0 | 6709 / 6055 | 9.7% | 0.15 / 0.66 | +343.2% | ACGC_WIN |
| `locomo_conv-42` / `q185_single_hop` | `sliding_window` | judge | 1.0 / 4.0 | 6709 / 6679 | 0.4% | 0.15 / 0.60 | +301.8% | ACGC_WIN |
| `locomo_conv-42` / `q187_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6710 / 6059 | 9.7% | 0.30 / 0.83 | +176.9% | ACGC_WIN |
| `locomo_conv-42` / `q187_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6710 / 6680 | 0.4% | 0.30 / 0.75 | +151.1% | ACGC_WIN |
| `locomo_conv-42` / `q199_single_hop` | `acgc` | judge | 2.0 / 4.0 | 6715 / 6074 | 9.5% | 0.30 / 0.66 | +121.1% | ACGC_WIN |
| `locomo_conv-42` / `q199_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6715 / 6685 | 0.4% | 0.30 / 0.75 | +151.1% | ACGC_WIN |
| `locomo_conv-42` / `q201_adversarial` | `acgc` | judge | 3.0 / 5.0 | 6704 / 6057 | 9.7% | 0.45 / 0.83 | +84.5% | ACGC_WIN |
| `locomo_conv-42` / `q201_adversarial` | `sliding_window` | judge | 3.0 / 5.0 | 6704 / 6674 | 0.4% | 0.45 / 0.75 | +67.4% | ACGC_WIN |
| `locomo_conv-42` / `q233_adversarial` | `acgc` | judge | 5.0 / 1.0 | 6711 / 6050 | 9.8% | 0.75 / 0.17 | -77.8% | ACGC_LOSS |
| `locomo_conv-42` / `q233_adversarial` | `sliding_window` | judge | 5.0 / 5.0 | 6711 / 6681 | 0.4% | 0.75 / 0.75 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q236_adversarial` | `acgc` | judge | 1.0 / 2.0 | 6722 / 6056 | 9.9% | 0.15 / 0.33 | +122.0% | ACGC_WIN |
| `locomo_conv-42` / `q236_adversarial` | `sliding_window` | judge | 1.0 / 1.0 | 6722 / 6692 | 0.4% | 0.15 / 0.15 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q245_adversarial` | `acgc` | judge | 1.0 / 1.0 | 6706 / 6058 | 9.7% | 0.15 / 0.17 | +10.7% | ACGC_WIN |
| `locomo_conv-42` / `q245_adversarial` | `sliding_window` | judge | 1.0 / 1.0 | 6706 / 6676 | 0.4% | 0.15 / 0.15 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q250_adversarial` | `acgc` | judge | 5.0 / 4.0 | 6715 / 6060 | 9.8% | 0.74 / 0.66 | -11.4% | TIE |
| `locomo_conv-42` / `q250_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6715 / 6685 | 0.4% | 0.74 / 0.30 | -59.8% | ACGC_LOSS |
| `locomo_conv-42` / `q39_temporal` | `acgc` | judge | 2.0 / 1.0 | 6706 / 6051 | 9.8% | 0.30 / 0.17 | -44.6% | TIE |
| `locomo_conv-42` / `q39_temporal` | `sliding_window` | judge | 2.0 / 2.0 | 6706 / 6676 | 0.4% | 0.30 / 0.30 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q43_multi_hop` | `acgc` | judge | 3.0 / 2.0 | 6705 / 6056 | 9.7% | 0.45 / 0.33 | -26.2% | TIE |
| `locomo_conv-42` / `q43_multi_hop` | `sliding_window` | judge | 3.0 / 1.0 | 6705 / 6675 | 0.4% | 0.45 / 0.15 | -66.5% | ACGC_LOSS |
| `locomo_conv-42` / `q5_open_domain` | `acgc` | judge | 0.0 / 2.0 | 6705 / 6054 | 9.7% | 0.00 / 0.33 | +0.0% | ACGC_WIN |
| `locomo_conv-42` / `q5_open_domain` | `sliding_window` | judge | 0.0 / 0.0 | 6705 / 6675 | 0.4% | 0.00 / 0.00 | +0.0% | TIE |
| `locomo_conv-42` / `q61_open_domain` | `acgc` | judge | 2.0 / 2.0 | 6702 / 6059 | 9.6% | 0.30 / 0.33 | +10.6% | ACGC_WIN |
| `locomo_conv-42` / `q61_open_domain` | `sliding_window` | judge | 2.0 / 2.0 | 6702 / 6672 | 0.4% | 0.30 / 0.30 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q66_temporal` | `acgc` | judge | 2.0 / 5.0 | 6712 / 6057 | 9.8% | 0.30 / 0.83 | +177.0% | ACGC_WIN |
| `locomo_conv-42` / `q66_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6712 / 6682 | 0.4% | 0.30 / 0.75 | +151.1% | ACGC_WIN |
| `locomo_conv-42` / `q71_multi_hop` | `acgc` | judge | 2.0 / 3.0 | 6705 / 6057 | 9.7% | 0.30 / 0.50 | +66.0% | ACGC_WIN |
| `locomo_conv-42` / `q71_multi_hop` | `sliding_window` | judge | 2.0 / 3.0 | 6705 / 6675 | 0.4% | 0.30 / 0.45 | +50.7% | ACGC_WIN |
| `locomo_conv-42` / `q73_temporal` | `acgc` | judge | 1.0 / 2.0 | 6713 / 6056 | 9.8% | 0.15 / 0.33 | +121.7% | ACGC_WIN |
| `locomo_conv-42` / `q73_temporal` | `sliding_window` | judge | 1.0 / 5.0 | 6713 / 6683 | 0.4% | 0.15 / 0.75 | +402.2% | ACGC_WIN |
| `locomo_conv-42` / `q80_multi_hop` | `acgc` | judge | 1.0 / 1.0 | 6704 / 6053 | 9.7% | 0.15 / 0.17 | +10.8% | ACGC_WIN |
| `locomo_conv-42` / `q80_multi_hop` | `sliding_window` | judge | 1.0 / 2.0 | 6704 / 6674 | 0.4% | 0.15 / 0.30 | +100.9% | ACGC_WIN |
| `locomo_conv-42` / `q91_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6704 / 6066 | 9.5% | 0.75 / 0.82 | +10.5% | ACGC_WIN |
| `locomo_conv-42` / `q91_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6704 / 6674 | 0.4% | 0.75 / 0.30 | -59.8% | ACGC_LOSS |
| `locomo_conv-43` / `q103_single_hop` | `acgc` | judge | 1.0 / 5.0 | 6826 / 6244 | 8.5% | 0.15 / 0.80 | +446.6% | ACGC_WIN |
| `locomo_conv-43` / `q103_single_hop` | `sliding_window` | judge | 1.0 / 1.0 | 6826 / 6893 | -1.0% | 0.15 / 0.15 | -1.0% | TIE |
| `locomo_conv-43` / `q114_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6821 / 6245 | 8.4% | 0.29 / 0.32 | +9.2% | ACGC_WIN |
| `locomo_conv-43` / `q114_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6821 / 6888 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q119_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6826 / 6236 | 8.6% | 0.29 / 0.80 | +173.7% | ACGC_WIN |
| `locomo_conv-43` / `q119_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6826 / 6893 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q129_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6827 / 6248 | 8.5% | 0.29 / 0.80 | +173.2% | ACGC_WIN |
| `locomo_conv-43` / `q129_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6827 / 6894 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q132_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6832 / 6245 | 8.6% | 0.29 / 0.80 | +173.5% | ACGC_WIN |
| `locomo_conv-43` / `q132_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6832 / 6899 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q133_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6824 / 6240 | 8.6% | 0.29 / 0.32 | +9.4% | ACGC_WIN |
| `locomo_conv-43` / `q133_single_hop` | `sliding_window` | judge | 2.0 / 1.0 | 6824 / 6891 | -1.0% | 0.29 / 0.15 | -50.5% | TIE |
| `locomo_conv-43` / `q136_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6824 / 6247 | 8.5% | 0.15 / 0.32 | +118.5% | ACGC_WIN |
| `locomo_conv-43` / `q136_single_hop` | `sliding_window` | judge | 1.0 / 2.0 | 6824 / 6891 | -1.0% | 0.15 / 0.29 | +98.1% | ACGC_WIN |
| `locomo_conv-43` / `q138_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6833 / 6237 | 8.7% | 0.29 / 0.32 | +9.6% | ACGC_WIN |
| `locomo_conv-43` / `q138_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6833 / 6900 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q142_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6833 / 6237 | 8.7% | 0.29 / 0.80 | +173.9% | ACGC_WIN |
| `locomo_conv-43` / `q142_single_hop` | `sliding_window` | judge | 2.0 / 1.0 | 6833 / 6900 | -1.0% | 0.29 / 0.14 | -50.5% | TIE |
| `locomo_conv-43` / `q155_single_hop` | `acgc` | judge | 3.0 / 5.0 | 6830 / 6239 | 8.7% | 0.44 / 0.80 | +82.5% | ACGC_WIN |
| `locomo_conv-43` / `q155_single_hop` | `sliding_window` | judge | 3.0 / 5.0 | 6830 / 6897 | -1.0% | 0.44 / 0.72 | +65.0% | ACGC_WIN |
| `locomo_conv-43` / `q163_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6824 / 6244 | 8.5% | 0.15 / 0.32 | +118.6% | ACGC_WIN |
| `locomo_conv-43` / `q163_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6824 / 6891 | -1.0% | 0.15 / 0.73 | +395.1% | ACGC_WIN |
| `locomo_conv-43` / `q177_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6828 / 6243 | 8.6% | 0.29 / 0.80 | +173.4% | ACGC_WIN |
| `locomo_conv-43` / `q177_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6828 / 6895 | -1.0% | 0.29 / 0.73 | +147.6% | ACGC_WIN |
| `locomo_conv-43` / `q180_adversarial` | `acgc` | judge | 2.0 / 5.0 | 6826 / 6251 | 8.4% | 0.29 / 0.80 | +173.0% | ACGC_WIN |
| `locomo_conv-43` / `q180_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6826 / 6893 | -1.0% | 0.29 / 0.73 | +147.6% | ACGC_WIN |
| `locomo_conv-43` / `q190_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6822 / 6241 | 8.5% | 0.73 / 0.80 | +9.3% | ACGC_WIN |
| `locomo_conv-43` / `q190_adversarial` | `sliding_window` | judge | 5.0 / 5.0 | 6822 / 6889 | -1.0% | 0.73 / 0.73 | -1.0% | TIE |
| `locomo_conv-43` / `q197_adversarial` | `acgc` | judge | 4.0 / 5.0 | 6829 / 6244 | 8.6% | 0.59 / 0.80 | +36.7% | ACGC_WIN |
| `locomo_conv-43` / `q197_adversarial` | `sliding_window` | judge | 4.0 / 5.0 | 6829 / 6896 | -1.0% | 0.59 / 0.73 | +23.8% | ACGC_WIN |
| `locomo_conv-43` / `q200_adversarial` | `acgc` | judge | 1.0 / 0.0 | 6827 / 6240 | 8.6% | 0.15 / 0.00 | -100.0% | TIE |
| `locomo_conv-43` / `q200_adversarial` | `sliding_window` | judge | 1.0 / 5.0 | 6827 / 6894 | -1.0% | 0.15 / 0.73 | +395.1% | ACGC_WIN |
| `locomo_conv-43` / `q221_adversarial` | `acgc` | judge | 2.0 / 1.0 | 6829 / 6242 | 8.6% | 0.29 / 0.16 | -45.3% | TIE |
| `locomo_conv-43` / `q221_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6829 / 6896 | -1.0% | 0.29 / 0.73 | +147.6% | ACGC_WIN |
| `locomo_conv-43` / `q43_multi_hop` | `acgc` | judge | 1.0 / 3.0 | 6819 / 6245 | 8.4% | 0.15 / 0.48 | +227.6% | ACGC_WIN |
| `locomo_conv-43` / `q43_multi_hop` | `sliding_window` | judge | 1.0 / 1.0 | 6819 / 6886 | -1.0% | 0.15 / 0.15 | -1.0% | TIE |
| `locomo_conv-43` / `q55_temporal` | `acgc` | judge | 1.0 / 2.0 | 6825 / 6237 | 8.6% | 0.15 / 0.32 | +118.9% | ACGC_WIN |
| `locomo_conv-43` / `q55_temporal` | `sliding_window` | judge | 1.0 / 1.0 | 6825 / 6892 | -1.0% | 0.15 / 0.15 | -1.0% | TIE |
| `locomo_conv-43` / `q63_multi_hop` | `acgc` | judge | 1.0 / 3.0 | 6823 / 6249 | 8.4% | 0.15 / 0.48 | +227.6% | ACGC_WIN |
| `locomo_conv-43` / `q63_multi_hop` | `sliding_window` | judge | 1.0 / 3.0 | 6823 / 6890 | -1.0% | 0.15 / 0.44 | +197.1% | ACGC_WIN |

## Response samples

### `locomo_conv-26::q102_single_hop`

**Question:** Did Melanie make the black and white bowl in the photo?

**naive_full_history (ref)** (6600 prompt tokens, 3230 ms):

> Yes. Melanie said she made that black-and-white bowl in her pottery class.

**sliding_window** (6599 prompt tokens, 15784 ms):

> Yes—if you mean the pottery photo Melanie shared of her bowls. She said, “Check out my pottery creation in the pic,” so the black-and-white bowl in that photo would be one she made. If you’re referring to a different photo, let me know.

**acgc** (6032 prompt tokens, 6498 ms):

> Yes. Melanie said she made the bowl in her pottery class (yesterday).

---

### `locomo_conv-26::q103_single_hop`

**Question:** What kind of books does Caroline have in her library?

**naive_full_history (ref)** (6599 prompt tokens, 4241 ms):

> Children’s books—classics, stories from different cultures, and educational books.

**sliding_window** (6598 prompt tokens, 6816 ms):

> It’s not specified in the conversation. We don’t have details about the kinds of books in Caroline’s library.

**acgc** (6034 prompt tokens, 13525 ms):

> It’s not specified in the provided context. There are mentions of books generally (e.g., a book Caroline recommended and discussions about inspirational/mental health themes), but nothing details what’s in her library.

---

### `locomo_conv-26::q11_temporal`

**Question:** How long has Caroline had her current group of friends for?

**naive_full_history (ref)** (6600 prompt tokens, 4153 ms):

> About four years, since she moved from her home country.

**sliding_window** (6599 prompt tokens, 4022 ms):

> It isn’t specified in the provided conversation.

**acgc** (6034 prompt tokens, 9556 ms):

> It isn’t specified in the provided conversation. Caroline mentions having a strong support network around her transition, but no exact timeframe is given.

---

### `locomo_conv-26::q132_single_hop`

**Question:** Who is Melanie a fan of in terms of modern music?

**naive_full_history (ref)** (6600 prompt tokens, 5510 ms):

> It isn’t mentioned in the provided conversation.

**sliding_window** (6599 prompt tokens, 2588 ms):

> Ed Sheeran.

**acgc** (6032 prompt tokens, 12579 ms):

> Matt Patterson.

---

### `locomo_conv-26::q137_single_hop`

**Question:** What does Melanie do to keep herself busy during her pottery break?

**naive_full_history (ref)** (6601 prompt tokens, 8990 ms):

> It isn’t mentioned in the conversation. Melanie doesn’t say what she does during a pottery break.

**sliding_window** (6600 prompt tokens, 2845 ms):

> She’s been reading the book you recommended and painting.

**acgc** (6033 prompt tokens, 4774 ms):

> She’s been reading the book you recommended and painting to keep busy.

---

### `locomo_conv-26::q145_single_hop`

**Question:** How did Melanie's son handle the accident?

**naive_full_history (ref)** (6597 prompt tokens, 4238 ms):

> I don’t see any mention of an accident (or how Melanie’s son handled one) in the provided conversation. Could you share more details or when this happened?

**sliding_window** (6596 prompt tokens, 4286 ms):

> He was okay after the accident.

**acgc** (6032 prompt tokens, 12185 ms):

> He was okay. The accident scared everyone, but her son ended up fine.

---

### `locomo_conv-26::q147_single_hop`

**Question:** How did Melanie's children handle the accident?

**naive_full_history (ref)** (6597 prompt tokens, 5238 ms):

> I don’t see any mention of an accident in the provided conversation, so there’s no info on how Melanie’s children handled it. Can you share more details about the accident or when it happened?

**sliding_window** (6596 prompt tokens, 3165 ms):

> They were scared at first, but Melanie reassured them their brother would be OK, and they handled it well—she said they’re tough kids.

**acgc** (6035 prompt tokens, 11865 ms):

> They were really scared—Melanie said “we were all freaked” when her son had the accident—but he was okay, and the kids later calmed down and enjoyed the rest of the trip (like the Grand Canyon).

---

### `locomo_conv-26::q148_single_hop`

**Question:** How did Melanie feel after the accident?

**naive_full_history (ref)** (6596 prompt tokens, 4040 ms):

> I don’t see any mention of an accident in the conversation, so there’s no info on how Melanie felt afterward. Could you share more context or where the accident is referenced?

**sliding_window** (6595 prompt tokens, 4931 ms):

> She was really scared and “freaked” during the accident, then relieved and grateful her son was okay—it reminded her to cherish her family.

**acgc** (6023 prompt tokens, 5306 ms):

> She was really scared and freaked out, but relieved and thankful her son was okay. It also reminded her how precious life is and to cherish her family.

---

### `locomo_conv-26::q152_single_hop`

**Question:** What did Melanie do after the road trip to relax?

**naive_full_history (ref)** (6599 prompt tokens, 12298 ms):

> I don’t see any road trip mentioned in the provided chat. The closest is a camping trip; after that, Melanie said she had a quiet weekend to unplug and spend time with her kids. If you meant a different trip, can you share more details?

**sliding_window** (6598 prompt tokens, 28123 ms):

> She went on a camping trip with her family.

**acgc** (6033 prompt tokens, 8593 ms):

> She did pottery with her kids—made that plate—as a way to unwind after the road trip.

---

### `locomo_conv-26::q168_adversarial`

**Question:** Did Caroline make the black and white bowl in the photo?

**naive_full_history (ref)** (6600 prompt tokens, 3619 ms):

> No. Melanie made the black and white bowl in her pottery class.

**sliding_window** (6599 prompt tokens, 5597 ms):

> No. The black and white bowl was made by Melanie, not Caroline—Caroline even said she hadn’t tried pottery yet.

**acgc** (6037 prompt tokens, 4040 ms):

> No. Melanie made the black-and-white bowl in her pottery class, not Caroline.

---

### `locomo_conv-26::q182_adversarial`

**Question:** What did Melanie make for a local church?

**naive_full_history (ref)** (6597 prompt tokens, 4077 ms):

> It isn’t mentioned in the provided conversation that Melanie made anything for a local church. If you have a specific message or date in mind, please share it and I’ll check.

**sliding_window** (6596 prompt tokens, 4246 ms):

> She didn’t make anything for a local church—Caroline did. Caroline made a stained glass window for a local church.

**acgc** (6025 prompt tokens, 19204 ms):

> I don’t see any mention here of Melanie making something for a local church. Could you point me to the message or give more context?

---

### `locomo_conv-26::q43_open_domain`

**Question:** Would Melanie be more interested in going to a national park or a theme park?

**naive_full_history (ref)** (6604 prompt tokens, 6093 ms):

> A national park. Melanie loves camping, hiking, and exploring nature with her family, so a national park fits her interests better than a theme park.

**sliding_window** (6603 prompt tokens, 3858 ms):

> A national park. Melanie loves nature—camping, hiking, park visits, and trips like the Grand Canyon—and often talks about how peaceful and refreshing the outdoors is.

**acgc** (6030 prompt tokens, 6466 ms):

> A national park. Melanie repeatedly talks about loving camping, hiking, and being in nature—waking to birds, fresh air, and the peace and serenity it brings—so a national park fits her interests much more than a theme park.

---

