# ACGC Context-Strategy Evaluation

**Generated:** 2026-07-03T22:57:54+05:30  
**Model:** `gpt-5`  
**Tokenizer:** `o200k_base`  
**Reference strategy:** `naive_full_history`  
**Strategies compared:** `naive_full_history`, `sliding_window`, `acgc`  
**Live tokens spent this run:** 2961819  

## Strategy comparison (side by side)

| Strategy | Probes | Avg Quality | Avg Prompt Tok | Avg Latency (ms) | Avg IPT | Tok Red% vs ref | Quality Δ vs ref | IPT Δ% vs ref |
|---|---|---|---|---|---|---|---|---|
| `naive_full_history (ref)` | 124 | 2.94 | 6753 | 6292 | 0.44 | 0.0% | +0.00 | +0.0% |
| `sliding_window` | 124 | 3.21 | 6764 | 6860 | 0.47 | -0.2% | +0.27 | +8.8% |
| `acgc` | 124 | 2.97 | 6068 | 10296 | 0.49 | 10.1% | +0.02 | +12.4% |

## Candidate vs reference (verdicts)

Reference: `naive_full_history`

- **Pairs evaluated:** 248
- **Avg quality (reference):** 2.94 / 5.0
- **Avg quality (candidate):** 3.09 / 5.0
- **Avg quality delta:** +0.15 (candidate - reference)
- **Avg token reduction:** 5.0%
- **Avg IPT (reference):** 0.44
- **Avg IPT (candidate):** 0.48
- **Avg IPT delta:** +10.6%
- **Quality regressions (>1.0 drop):** 60

### Verdict breakdown

- `ACGC_WIN` (better IPT, no quality loss): **154**
- `ACGC_WIN_STAR` (better IPT, but quality dropped): **0**
- `TIE`: **34**
- `ACGC_LOSS`: **60**
- `BASELINE_WIN` (reference strictly better): **0**

## Per-probe results

| Scenario / Probe | Candidate | Method | Quality (ref / cand) | Tokens (ref / cand) | Token Red% | IPT (ref / cand) | IPT Δ% | Verdict |
|---|---|---|---|---|---|---|---|---|
| `locomo_conv-26` / `q102_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6600 / 6024 | 8.7% | 0.76 / 0.83 | +9.6% | ACGC_WIN |
| `locomo_conv-26` / `q102_single_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6600 / 6599 | 0.0% | 0.76 / 0.76 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q103_single_hop` | `acgc` | judge | 5.0 / 2.0 | 6599 / 6026 | 8.7% | 0.76 / 0.33 | -56.2% | ACGC_LOSS |
| `locomo_conv-26` / `q103_single_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6599 / 6598 | 0.0% | 0.76 / 0.15 | -80.0% | ACGC_LOSS |
| `locomo_conv-26` / `q11_temporal` | `acgc` | judge | 5.0 / 1.0 | 6600 / 6028 | 8.7% | 0.76 / 0.17 | -78.1% | ACGC_LOSS |
| `locomo_conv-26` / `q11_temporal` | `sliding_window` | judge | 5.0 / 0.0 | 6600 / 6599 | 0.0% | 0.76 / 0.00 | -100.0% | ACGC_LOSS |
| `locomo_conv-26` / `q132_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6600 / 6019 | 8.8% | 0.30 / 0.83 | +174.1% | ACGC_WIN |
| `locomo_conv-26` / `q132_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6600 / 6599 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q137_single_hop` | `acgc` | judge | 1.0 / 5.0 | 6601 / 6029 | 8.7% | 0.15 / 0.83 | +447.4% | ACGC_WIN |
| `locomo_conv-26` / `q137_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6601 / 6600 | 0.0% | 0.15 / 0.76 | +400.1% | ACGC_WIN |
| `locomo_conv-26` / `q145_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6597 / 6027 | 8.6% | 0.15 / 0.33 | +118.9% | ACGC_WIN |
| `locomo_conv-26` / `q145_single_hop` | `sliding_window` | judge | 1.0 / 2.0 | 6597 / 6596 | 0.0% | 0.15 / 0.30 | +100.0% | ACGC_WIN |
| `locomo_conv-26` / `q147_single_hop` | `acgc` | judge | 1.0 / 5.0 | 6597 / 6020 | 8.7% | 0.15 / 0.83 | +447.9% | ACGC_WIN |
| `locomo_conv-26` / `q147_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6597 / 6596 | 0.0% | 0.15 / 0.76 | +400.1% | ACGC_WIN |
| `locomo_conv-26` / `q148_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6596 / 6021 | 8.7% | 0.30 / 0.83 | +173.9% | ACGC_WIN |
| `locomo_conv-26` / `q148_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6596 / 6595 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q152_single_hop` | `acgc` | judge | 2.0 / 1.0 | 6599 / 6021 | 8.8% | 0.30 / 0.17 | -45.2% | TIE |
| `locomo_conv-26` / `q152_single_hop` | `sliding_window` | judge | 2.0 / 1.0 | 6599 / 6598 | 0.0% | 0.30 / 0.15 | -50.0% | TIE |
| `locomo_conv-26` / `q168_adversarial` | `acgc` | judge | 2.0 / 2.0 | 6600 / 6014 | 8.9% | 0.30 / 0.33 | +9.7% | ACGC_WIN |
| `locomo_conv-26` / `q168_adversarial` | `sliding_window` | judge | 2.0 / 2.0 | 6600 / 6599 | 0.0% | 0.30 / 0.30 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q182_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6597 / 6027 | 8.6% | 0.76 / 0.83 | +9.5% | ACGC_WIN |
| `locomo_conv-26` / `q182_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6597 / 6596 | 0.0% | 0.76 / 0.30 | -60.0% | ACGC_LOSS |
| `locomo_conv-26` / `q43_open_domain` | `acgc` | judge | 5.0 / 5.0 | 6604 / 6026 | 8.8% | 0.76 / 0.83 | +9.6% | ACGC_WIN |
| `locomo_conv-26` / `q43_open_domain` | `sliding_window` | judge | 5.0 / 5.0 | 6604 / 6603 | 0.0% | 0.76 / 0.76 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q47_open_domain` | `acgc` | judge | 5.0 / 5.0 | 6599 / 6018 | 8.8% | 0.76 / 0.83 | +9.7% | ACGC_WIN |
| `locomo_conv-26` / `q47_open_domain` | `sliding_window` | judge | 5.0 / 5.0 | 6599 / 6598 | 0.0% | 0.76 / 0.76 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q48_multi_hop` | `acgc` | judge | 5.0 / 0.0 | 6598 / 6022 | 8.7% | 0.76 / 0.00 | -100.0% | ACGC_LOSS |
| `locomo_conv-26` / `q48_multi_hop` | `sliding_window` | judge | 5.0 / 3.0 | 6598 / 6597 | 0.0% | 0.76 / 0.45 | -40.0% | ACGC_LOSS |
| `locomo_conv-26` / `q4_multi_hop` | `acgc` | judge | 5.0 / 2.0 | 6593 / 6022 | 8.7% | 0.76 / 0.33 | -56.2% | ACGC_LOSS |
| `locomo_conv-26` / `q4_multi_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6593 / 6592 | 0.0% | 0.76 / 0.30 | -60.0% | ACGC_LOSS |
| `locomo_conv-26` / `q54_temporal` | `acgc` | judge | 2.0 / 2.0 | 6596 / 6019 | 8.7% | 0.30 / 0.33 | +9.6% | ACGC_WIN |
| `locomo_conv-26` / `q54_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6596 / 6595 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q55_temporal` | `acgc` | judge | 2.0 / 2.0 | 6597 / 6019 | 8.8% | 0.30 / 0.33 | +9.6% | ACGC_WIN |
| `locomo_conv-26` / `q55_temporal` | `sliding_window` | judge | 2.0 / 4.0 | 6597 / 6596 | 0.0% | 0.30 / 0.61 | +100.0% | ACGC_WIN |
| `locomo_conv-26` / `q64_temporal` | `acgc` | judge | 2.0 / 2.0 | 6600 / 6022 | 8.8% | 0.30 / 0.33 | +9.6% | ACGC_WIN |
| `locomo_conv-26` / `q64_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6600 / 6599 | 0.0% | 0.30 / 0.76 | +150.0% | ACGC_WIN |
| `locomo_conv-26` / `q73_temporal` | `acgc` | judge | 1.0 / 2.0 | 6597 / 6019 | 8.8% | 0.15 / 0.33 | +119.2% | ACGC_WIN |
| `locomo_conv-26` / `q73_temporal` | `sliding_window` | judge | 1.0 / 1.0 | 6597 / 6596 | 0.0% | 0.15 / 0.15 | +0.0% | ACGC_WIN |
| `locomo_conv-26` / `q82_open_domain` | `acgc` | judge | 3.0 / 4.0 | 6600 / 6029 | 8.7% | 0.45 / 0.66 | +46.0% | ACGC_WIN |
| `locomo_conv-26` / `q82_open_domain` | `sliding_window` | judge | 3.0 / 2.0 | 6600 / 6599 | 0.0% | 0.45 / 0.30 | -33.3% | TIE |
| `locomo_conv-30` / `q100_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6713 / 5616 | 16.3% | 0.74 / 0.89 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q100_adversarial` | `sliding_window` | judge | 5.0 / 5.0 | 6713 / 6800 | -1.3% | 0.74 / 0.74 | -1.3% | TIE |
| `locomo_conv-30` / `q16_temporal` | `acgc` | judge | 5.0 / 1.0 | 6714 / 5617 | 16.3% | 0.74 / 0.18 | -76.1% | ACGC_LOSS |
| `locomo_conv-30` / `q16_temporal` | `sliding_window` | judge | 5.0 / 2.0 | 6714 / 6801 | -1.3% | 0.74 / 0.29 | -60.5% | ACGC_LOSS |
| `locomo_conv-30` / `q18_multi_hop` | `acgc` | judge | 4.0 / 3.0 | 6717 / 5620 | 16.3% | 0.60 / 0.53 | -10.4% | TIE |
| `locomo_conv-30` / `q18_multi_hop` | `sliding_window` | judge | 4.0 / 3.0 | 6717 / 6804 | -1.3% | 0.60 / 0.44 | -26.0% | TIE |
| `locomo_conv-30` / `q1_temporal` | `acgc` | judge | 5.0 / 2.0 | 6716 / 5619 | 16.3% | 0.74 / 0.36 | -52.2% | ACGC_LOSS |
| `locomo_conv-30` / `q1_temporal` | `sliding_window` | judge | 5.0 / 1.0 | 6716 / 6803 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q25_multi_hop` | `acgc` | judge | 2.0 / 3.0 | 6718 / 5621 | 16.3% | 0.30 / 0.53 | +79.3% | ACGC_WIN |
| `locomo_conv-30` / `q25_multi_hop` | `sliding_window` | judge | 2.0 / 3.0 | 6718 / 6805 | -1.3% | 0.30 / 0.44 | +48.1% | ACGC_WIN |
| `locomo_conv-30` / `q27_temporal` | `acgc` | judge | 1.0 / 2.0 | 6716 / 5619 | 16.3% | 0.15 / 0.36 | +139.0% | ACGC_WIN |
| `locomo_conv-30` / `q27_temporal` | `sliding_window` | judge | 1.0 / 5.0 | 6716 / 6803 | -1.3% | 0.15 / 0.73 | +393.6% | ACGC_WIN |
| `locomo_conv-30` / `q33_temporal` | `acgc` | judge | 2.0 / 2.0 | 6717 / 5620 | 16.3% | 0.30 / 0.36 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q33_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6717 / 6804 | -1.3% | 0.30 / 0.73 | +146.8% | ACGC_WIN |
| `locomo_conv-30` / `q36_temporal` | `acgc` | judge | 1.0 / 2.0 | 6716 / 5619 | 16.3% | 0.15 / 0.36 | +139.0% | ACGC_WIN |
| `locomo_conv-30` / `q36_temporal` | `sliding_window` | judge | 1.0 / 5.0 | 6716 / 6803 | -1.3% | 0.15 / 0.73 | +393.6% | ACGC_WIN |
| `locomo_conv-30` / `q47_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6719 / 5622 | 16.3% | 0.74 / 0.89 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q47_single_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6719 / 6806 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q51_single_hop` | `acgc` | judge | 5.0 / 1.0 | 6718 / 5621 | 16.3% | 0.74 / 0.18 | -76.1% | ACGC_LOSS |
| `locomo_conv-30` / `q51_single_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6718 / 6805 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q55_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6713 / 5616 | 16.3% | 0.74 / 0.89 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q55_single_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6713 / 6800 | -1.3% | 0.74 / 0.74 | -1.3% | TIE |
| `locomo_conv-30` / `q56_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6713 / 5616 | 16.3% | 0.74 / 0.89 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q56_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6713 / 6800 | -1.3% | 0.74 / 0.29 | -60.5% | ACGC_LOSS |
| `locomo_conv-30` / `q60_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6716 / 5619 | 16.3% | 0.74 / 0.89 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q60_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6716 / 6803 | -1.3% | 0.74 / 0.29 | -60.5% | ACGC_LOSS |
| `locomo_conv-30` / `q62_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6715 / 5618 | 16.3% | 0.74 / 0.89 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q62_single_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6715 / 6802 | -1.3% | 0.74 / 0.74 | -1.3% | TIE |
| `locomo_conv-30` / `q6_multi_hop` | `acgc` | judge | 5.0 / 2.0 | 6717 / 5620 | 16.3% | 0.74 / 0.36 | -52.2% | ACGC_LOSS |
| `locomo_conv-30` / `q6_multi_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6717 / 6804 | -1.3% | 0.74 / 0.15 | -80.3% | ACGC_LOSS |
| `locomo_conv-30` / `q71_single_hop` | `acgc` | judge | 5.0 / 5.0 | 6713 / 5616 | 16.3% | 0.74 / 0.89 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q71_single_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6713 / 6800 | -1.3% | 0.74 / 0.74 | -1.3% | TIE |
| `locomo_conv-30` / `q87_adversarial` | `acgc` | judge | 1.0 / 1.0 | 6719 / 5622 | 16.3% | 0.15 / 0.18 | +19.5% | ACGC_WIN |
| `locomo_conv-30` / `q87_adversarial` | `sliding_window` | judge | 1.0 / 5.0 | 6719 / 6806 | -1.3% | 0.15 / 0.73 | +393.6% | ACGC_WIN |
| `locomo_conv-30` / `q91_adversarial` | `acgc` | judge | 0.0 / 1.0 | 6718 / 5621 | 16.3% | 0.00 / 0.18 | +0.0% | ACGC_WIN |
| `locomo_conv-30` / `q91_adversarial` | `sliding_window` | judge | 0.0 / 5.0 | 6718 / 6805 | -1.3% | 0.00 / 0.73 | +0.0% | ACGC_WIN |
| `locomo_conv-30` / `q94_adversarial` | `acgc` | judge | 2.0 / 5.0 | 6715 / 5618 | 16.3% | 0.30 / 0.89 | +198.8% | ACGC_WIN |
| `locomo_conv-30` / `q94_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6715 / 6802 | -1.3% | 0.30 / 0.74 | +146.8% | ACGC_WIN |
| `locomo_conv-30` / `q97_adversarial` | `acgc` | judge | 5.0 / 2.0 | 6725 / 5628 | 16.3% | 0.74 / 0.36 | -52.2% | ACGC_LOSS |
| `locomo_conv-30` / `q97_adversarial` | `sliding_window` | judge | 5.0 / 4.0 | 6725 / 6812 | -1.3% | 0.74 / 0.59 | -21.0% | TIE |
| `locomo_conv-41` / `q101_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6821 / 6234 | 8.6% | 0.29 / 0.32 | +9.4% | ACGC_WIN |
| `locomo_conv-41` / `q101_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6821 / 6830 | -0.1% | 0.29 / 0.29 | -0.1% | TIE |
| `locomo_conv-41` / `q10_multi_hop` | `acgc` | judge | 5.0 / 2.0 | 6827 / 6224 | 8.8% | 0.73 / 0.32 | -56.1% | ACGC_LOSS |
| `locomo_conv-41` / `q10_multi_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6827 / 6836 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q11_temporal` | `acgc` | judge | 5.0 / 2.0 | 6814 / 6233 | 8.5% | 0.73 / 0.32 | -56.3% | ACGC_LOSS |
| `locomo_conv-41` / `q11_temporal` | `sliding_window` | judge | 5.0 / 2.0 | 6814 / 6823 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q131_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6820 / 6233 | 8.6% | 0.15 / 0.32 | +118.8% | ACGC_WIN |
| `locomo_conv-41` / `q131_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6820 / 6829 | -0.1% | 0.15 / 0.73 | +399.3% | ACGC_WIN |
| `locomo_conv-41` / `q140_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6822 / 6229 | 8.7% | 0.15 / 0.32 | +119.0% | ACGC_WIN |
| `locomo_conv-41` / `q140_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6822 / 6831 | -0.1% | 0.15 / 0.73 | +399.3% | ACGC_WIN |
| `locomo_conv-41` / `q158_adversarial` | `acgc` | judge | 2.0 / 5.0 | 6820 / 6232 | 8.6% | 0.29 / 0.80 | +173.6% | ACGC_WIN |
| `locomo_conv-41` / `q158_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6820 / 6829 | -0.1% | 0.29 / 0.73 | +149.7% | ACGC_WIN |
| `locomo_conv-41` / `q161_adversarial` | `acgc` | judge | 2.0 / 1.0 | 6823 / 6232 | 8.7% | 0.29 / 0.16 | -45.3% | TIE |
| `locomo_conv-41` / `q161_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6823 / 6832 | -0.1% | 0.29 / 0.73 | +149.7% | ACGC_WIN |
| `locomo_conv-41` / `q162_adversarial` | `acgc` | judge | 5.0 / 3.0 | 6816 / 6232 | 8.6% | 0.73 / 0.48 | -34.4% | ACGC_LOSS |
| `locomo_conv-41` / `q162_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6816 / 6825 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q167_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6822 / 6233 | 8.6% | 0.73 / 0.80 | +9.4% | ACGC_WIN |
| `locomo_conv-41` / `q167_adversarial` | `sliding_window` | judge | 5.0 / 5.0 | 6822 / 6831 | -0.1% | 0.73 / 0.73 | -0.1% | TIE |
| `locomo_conv-41` / `q179_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6822 / 6228 | 8.7% | 0.73 / 0.80 | +9.5% | ACGC_WIN |
| `locomo_conv-41` / `q179_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6822 / 6831 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q183_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6824 / 6227 | 8.7% | 0.73 / 0.80 | +9.6% | ACGC_WIN |
| `locomo_conv-41` / `q183_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6824 / 6833 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q38_multi_hop` | `acgc` | judge | 1.0 / 0.0 | 6817 / 6232 | 8.6% | 0.15 / 0.00 | -100.0% | TIE |
| `locomo_conv-41` / `q38_multi_hop` | `sliding_window` | judge | 1.0 / 3.0 | 6817 / 6826 | -0.1% | 0.15 / 0.44 | +199.6% | ACGC_WIN |
| `locomo_conv-41` / `q42_open_domain` | `acgc` | judge | 2.0 / 2.0 | 6819 / 6229 | 8.7% | 0.29 / 0.32 | +9.5% | ACGC_WIN |
| `locomo_conv-41` / `q42_open_domain` | `sliding_window` | judge | 2.0 / 5.0 | 6819 / 6828 | -0.1% | 0.29 / 0.73 | +149.7% | ACGC_WIN |
| `locomo_conv-41` / `q43_multi_hop` | `acgc` | judge | 2.0 / 2.0 | 6816 / 6229 | 8.6% | 0.29 / 0.32 | +9.4% | ACGC_WIN |
| `locomo_conv-41` / `q43_multi_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6816 / 6825 | -0.1% | 0.29 / 0.29 | -0.1% | TIE |
| `locomo_conv-41` / `q44_temporal` | `acgc` | judge | 2.0 / 2.0 | 6818 / 6237 | 8.5% | 0.29 / 0.32 | +9.3% | ACGC_WIN |
| `locomo_conv-41` / `q44_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6818 / 6827 | -0.1% | 0.29 / 0.73 | +149.7% | ACGC_WIN |
| `locomo_conv-41` / `q4_multi_hop` | `acgc` | judge | 5.0 / 5.0 | 6819 / 6230 | 8.6% | 0.73 / 0.80 | +9.5% | ACGC_WIN |
| `locomo_conv-41` / `q4_multi_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6819 / 6828 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q57_temporal` | `acgc` | judge | 2.0 / 2.0 | 6820 / 6233 | 8.6% | 0.29 / 0.32 | +9.4% | ACGC_WIN |
| `locomo_conv-41` / `q57_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6820 / 6829 | -0.1% | 0.29 / 0.73 | +149.7% | ACGC_WIN |
| `locomo_conv-41` / `q70_single_hop` | `acgc` | judge | 3.0 / 1.0 | 6822 / 6234 | 8.6% | 0.44 / 0.16 | -63.5% | ACGC_LOSS |
| `locomo_conv-41` / `q70_single_hop` | `sliding_window` | judge | 3.0 / 1.0 | 6822 / 6831 | -0.1% | 0.44 / 0.15 | -66.7% | ACGC_LOSS |
| `locomo_conv-41` / `q78_single_hop` | `acgc` | judge | 5.0 / 2.0 | 6820 / 6225 | 8.7% | 0.73 / 0.32 | -56.2% | ACGC_LOSS |
| `locomo_conv-41` / `q78_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6820 / 6829 | -0.1% | 0.73 / 0.29 | -60.1% | ACGC_LOSS |
| `locomo_conv-41` / `q89_single_hop` | `acgc` | judge | 2.0 / 1.0 | 6817 / 6235 | 8.5% | 0.29 / 0.16 | -45.3% | TIE |
| `locomo_conv-41` / `q89_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6817 / 6826 | -0.1% | 0.29 / 0.29 | -0.1% | TIE |
| `locomo_conv-42` / `q158_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6720 / 6047 | 10.0% | 0.30 / 0.83 | +177.8% | ACGC_WIN |
| `locomo_conv-42` / `q158_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6720 / 6690 | 0.4% | 0.30 / 0.75 | +151.1% | ACGC_WIN |
| `locomo_conv-42` / `q177_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6720 / 6045 | 10.0% | 0.30 / 0.83 | +177.9% | ACGC_WIN |
| `locomo_conv-42` / `q177_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6720 / 6690 | 0.4% | 0.30 / 0.75 | +151.1% | ACGC_WIN |
| `locomo_conv-42` / `q181_single_hop` | `acgc` | judge | 1.0 / 5.0 | 6706 / 6042 | 9.9% | 0.15 / 0.83 | +454.9% | ACGC_WIN |
| `locomo_conv-42` / `q181_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6706 / 6676 | 0.4% | 0.15 / 0.75 | +402.2% | ACGC_WIN |
| `locomo_conv-42` / `q185_single_hop` | `acgc` | judge | 1.0 / 3.0 | 6709 / 6045 | 9.9% | 0.15 / 0.50 | +233.0% | ACGC_WIN |
| `locomo_conv-42` / `q185_single_hop` | `sliding_window` | judge | 1.0 / 3.0 | 6709 / 6679 | 0.4% | 0.15 / 0.45 | +201.3% | ACGC_WIN |
| `locomo_conv-42` / `q187_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6710 / 6040 | 10.0% | 0.30 / 0.83 | +177.7% | ACGC_WIN |
| `locomo_conv-42` / `q187_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6710 / 6680 | 0.4% | 0.30 / 0.75 | +151.1% | ACGC_WIN |
| `locomo_conv-42` / `q199_single_hop` | `acgc` | judge | 1.0 / 5.0 | 6715 / 6045 | 10.0% | 0.15 / 0.83 | +455.4% | ACGC_WIN |
| `locomo_conv-42` / `q199_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6715 / 6685 | 0.4% | 0.15 / 0.75 | +402.2% | ACGC_WIN |
| `locomo_conv-42` / `q201_adversarial` | `acgc` | judge | 4.0 / 5.0 | 6704 / 6039 | 9.9% | 0.60 / 0.83 | +38.8% | ACGC_WIN |
| `locomo_conv-42` / `q201_adversarial` | `sliding_window` | judge | 4.0 / 5.0 | 6704 / 6674 | 0.4% | 0.60 / 0.75 | +25.6% | ACGC_WIN |
| `locomo_conv-42` / `q233_adversarial` | `acgc` | judge | 5.0 / 2.0 | 6711 / 6044 | 9.9% | 0.75 / 0.33 | -55.6% | ACGC_LOSS |
| `locomo_conv-42` / `q233_adversarial` | `sliding_window` | judge | 5.0 / 5.0 | 6711 / 6681 | 0.4% | 0.75 / 0.75 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q236_adversarial` | `acgc` | judge | 2.0 / 1.0 | 6722 / 6041 | 10.1% | 0.30 / 0.17 | -44.4% | TIE |
| `locomo_conv-42` / `q236_adversarial` | `sliding_window` | judge | 2.0 / 2.0 | 6722 / 6692 | 0.4% | 0.30 / 0.30 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q245_adversarial` | `acgc` | judge | 1.0 / 1.0 | 6706 / 6042 | 9.9% | 0.15 / 0.17 | +11.0% | ACGC_WIN |
| `locomo_conv-42` / `q245_adversarial` | `sliding_window` | judge | 1.0 / 1.0 | 6706 / 6676 | 0.4% | 0.15 / 0.15 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q250_adversarial` | `acgc` | judge | 5.0 / 2.0 | 6715 / 6045 | 10.0% | 0.74 / 0.33 | -55.6% | ACGC_LOSS |
| `locomo_conv-42` / `q250_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6715 / 6685 | 0.4% | 0.74 / 0.30 | -59.8% | ACGC_LOSS |
| `locomo_conv-42` / `q39_temporal` | `acgc` | judge | 2.0 / 2.0 | 6706 / 6043 | 9.9% | 0.30 / 0.33 | +11.0% | ACGC_WIN |
| `locomo_conv-42` / `q39_temporal` | `sliding_window` | judge | 2.0 / 2.0 | 6706 / 6676 | 0.4% | 0.30 / 0.30 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q43_multi_hop` | `acgc` | judge | 3.0 / 1.0 | 6705 / 6042 | 9.9% | 0.45 / 0.17 | -63.0% | ACGC_LOSS |
| `locomo_conv-42` / `q43_multi_hop` | `sliding_window` | judge | 3.0 / 1.0 | 6705 / 6675 | 0.4% | 0.45 / 0.15 | -66.5% | ACGC_LOSS |
| `locomo_conv-42` / `q5_open_domain` | `acgc` | judge | 2.0 / 2.0 | 6705 / 6041 | 9.9% | 0.30 / 0.33 | +11.0% | ACGC_WIN |
| `locomo_conv-42` / `q5_open_domain` | `sliding_window` | judge | 2.0 / 2.0 | 6705 / 6675 | 0.4% | 0.30 / 0.30 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q61_open_domain` | `acgc` | judge | 2.0 / 2.0 | 6702 / 6043 | 9.8% | 0.30 / 0.33 | +10.9% | ACGC_WIN |
| `locomo_conv-42` / `q61_open_domain` | `sliding_window` | judge | 2.0 / 2.0 | 6702 / 6672 | 0.4% | 0.30 / 0.30 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q66_temporal` | `acgc` | judge | 2.0 / 5.0 | 6712 / 6041 | 10.0% | 0.30 / 0.83 | +177.8% | ACGC_WIN |
| `locomo_conv-42` / `q66_temporal` | `sliding_window` | judge | 2.0 / 5.0 | 6712 / 6682 | 0.4% | 0.30 / 0.75 | +151.1% | ACGC_WIN |
| `locomo_conv-42` / `q71_multi_hop` | `acgc` | judge | 2.0 / 4.0 | 6705 / 6043 | 9.9% | 0.30 / 0.66 | +121.9% | ACGC_WIN |
| `locomo_conv-42` / `q71_multi_hop` | `sliding_window` | judge | 2.0 / 3.0 | 6705 / 6675 | 0.4% | 0.30 / 0.45 | +50.7% | ACGC_WIN |
| `locomo_conv-42` / `q73_temporal` | `acgc` | judge | 1.0 / 2.0 | 6713 / 6047 | 9.9% | 0.15 / 0.33 | +122.0% | ACGC_WIN |
| `locomo_conv-42` / `q73_temporal` | `sliding_window` | judge | 1.0 / 5.0 | 6713 / 6683 | 0.4% | 0.15 / 0.75 | +402.2% | ACGC_WIN |
| `locomo_conv-42` / `q80_multi_hop` | `acgc` | judge | 2.0 / 1.0 | 6704 / 6044 | 9.8% | 0.30 / 0.17 | -44.5% | TIE |
| `locomo_conv-42` / `q80_multi_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6704 / 6674 | 0.4% | 0.30 / 0.30 | +0.4% | ACGC_WIN |
| `locomo_conv-42` / `q91_single_hop` | `acgc` | judge | 5.0 / 1.0 | 6704 / 6047 | 9.8% | 0.75 / 0.17 | -77.8% | ACGC_LOSS |
| `locomo_conv-42` / `q91_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6704 / 6674 | 0.4% | 0.75 / 0.30 | -59.8% | ACGC_LOSS |
| `locomo_conv-43` / `q103_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6826 / 6222 | 8.8% | 0.29 / 0.32 | +9.7% | ACGC_WIN |
| `locomo_conv-43` / `q103_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6826 / 6893 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q114_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6821 / 6221 | 8.8% | 0.29 / 0.32 | +9.6% | ACGC_WIN |
| `locomo_conv-43` / `q114_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6821 / 6888 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q119_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6826 / 6219 | 8.9% | 0.15 / 0.32 | +119.5% | ACGC_WIN |
| `locomo_conv-43` / `q119_single_hop` | `sliding_window` | judge | 1.0 / 1.0 | 6826 / 6893 | -1.0% | 0.15 / 0.15 | -1.0% | TIE |
| `locomo_conv-43` / `q129_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6827 / 6222 | 8.9% | 0.29 / 0.32 | +9.7% | ACGC_WIN |
| `locomo_conv-43` / `q129_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6827 / 6894 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q132_single_hop` | `acgc` | judge | 2.0 / 1.0 | 6832 / 6221 | 8.9% | 0.29 / 0.16 | -45.1% | TIE |
| `locomo_conv-43` / `q132_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6832 / 6899 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q133_single_hop` | `acgc` | judge | 2.0 / 3.0 | 6824 / 6220 | 8.9% | 0.29 / 0.48 | +64.6% | ACGC_WIN |
| `locomo_conv-43` / `q133_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6824 / 6891 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q136_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6824 / 6220 | 8.9% | 0.15 / 0.32 | +119.4% | ACGC_WIN |
| `locomo_conv-43` / `q136_single_hop` | `sliding_window` | judge | 1.0 / 2.0 | 6824 / 6891 | -1.0% | 0.15 / 0.29 | +98.1% | ACGC_WIN |
| `locomo_conv-43` / `q138_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6833 / 6218 | 9.0% | 0.29 / 0.32 | +9.9% | ACGC_WIN |
| `locomo_conv-43` / `q138_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6833 / 6900 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q142_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6833 / 6223 | 8.9% | 0.29 / 0.80 | +174.5% | ACGC_WIN |
| `locomo_conv-43` / `q142_single_hop` | `sliding_window` | judge | 2.0 / 1.0 | 6833 / 6900 | -1.0% | 0.29 / 0.14 | -50.5% | TIE |
| `locomo_conv-43` / `q155_single_hop` | `acgc` | judge | 3.0 / 5.0 | 6830 / 6226 | 8.8% | 0.44 / 0.80 | +82.8% | ACGC_WIN |
| `locomo_conv-43` / `q155_single_hop` | `sliding_window` | judge | 3.0 / 5.0 | 6830 / 6897 | -1.0% | 0.44 / 0.72 | +65.0% | ACGC_WIN |
| `locomo_conv-43` / `q163_single_hop` | `acgc` | judge | 1.0 / 2.0 | 6824 / 6220 | 8.9% | 0.15 / 0.32 | +119.4% | ACGC_WIN |
| `locomo_conv-43` / `q163_single_hop` | `sliding_window` | judge | 1.0 / 5.0 | 6824 / 6891 | -1.0% | 0.15 / 0.73 | +395.1% | ACGC_WIN |
| `locomo_conv-43` / `q177_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6828 / 6226 | 8.8% | 0.29 / 0.80 | +174.2% | ACGC_WIN |
| `locomo_conv-43` / `q177_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6828 / 6895 | -1.0% | 0.29 / 0.73 | +147.6% | ACGC_WIN |
| `locomo_conv-43` / `q180_adversarial` | `acgc` | judge | 2.0 / 5.0 | 6826 / 6224 | 8.8% | 0.29 / 0.80 | +174.2% | ACGC_WIN |
| `locomo_conv-43` / `q180_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6826 / 6893 | -1.0% | 0.29 / 0.73 | +147.6% | ACGC_WIN |
| `locomo_conv-43` / `q190_adversarial` | `acgc` | judge | 5.0 / 5.0 | 6822 / 6223 | 8.8% | 0.73 / 0.80 | +9.6% | ACGC_WIN |
| `locomo_conv-43` / `q190_adversarial` | `sliding_window` | judge | 5.0 / 5.0 | 6822 / 6889 | -1.0% | 0.73 / 0.73 | -1.0% | TIE |
| `locomo_conv-43` / `q197_adversarial` | `acgc` | judge | 2.0 / 5.0 | 6829 / 6225 | 8.8% | 0.29 / 0.80 | +174.3% | ACGC_WIN |
| `locomo_conv-43` / `q197_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6829 / 6896 | -1.0% | 0.29 / 0.73 | +147.6% | ACGC_WIN |
| `locomo_conv-43` / `q200_adversarial` | `acgc` | judge | 1.0 / 5.0 | 6827 / 6219 | 8.9% | 0.15 / 0.80 | +448.9% | ACGC_WIN |
| `locomo_conv-43` / `q200_adversarial` | `sliding_window` | judge | 1.0 / 5.0 | 6827 / 6894 | -1.0% | 0.15 / 0.73 | +395.1% | ACGC_WIN |
| `locomo_conv-43` / `q221_adversarial` | `acgc` | judge | 1.0 / 1.0 | 6829 / 6227 | 8.8% | 0.15 / 0.16 | +9.7% | ACGC_WIN |
| `locomo_conv-43` / `q221_adversarial` | `sliding_window` | judge | 1.0 / 5.0 | 6829 / 6896 | -1.0% | 0.15 / 0.73 | +395.1% | ACGC_WIN |
| `locomo_conv-43` / `q43_multi_hop` | `acgc` | judge | 2.0 / 3.0 | 6819 / 6222 | 8.8% | 0.29 / 0.48 | +64.4% | ACGC_WIN |
| `locomo_conv-43` / `q43_multi_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6819 / 6886 | -1.0% | 0.29 / 0.29 | -1.0% | TIE |
| `locomo_conv-43` / `q55_temporal` | `acgc` | judge | 1.0 / 2.0 | 6825 / 6223 | 8.8% | 0.15 / 0.32 | +119.3% | ACGC_WIN |
| `locomo_conv-43` / `q55_temporal` | `sliding_window` | judge | 1.0 / 2.0 | 6825 / 6892 | -1.0% | 0.15 / 0.29 | +98.1% | ACGC_WIN |
| `locomo_conv-43` / `q63_multi_hop` | `acgc` | judge | 1.0 / 3.0 | 6823 / 6224 | 8.8% | 0.15 / 0.48 | +228.9% | ACGC_WIN |
| `locomo_conv-43` / `q63_multi_hop` | `sliding_window` | judge | 1.0 / 3.0 | 6823 / 6890 | -1.0% | 0.15 / 0.44 | +197.1% | ACGC_WIN |
| `locomo_conv-44` / `q10_multi_hop` | `acgc` | judge | 3.0 / 2.0 | 6820 / 6237 | 8.5% | 0.44 / 0.32 | -27.1% | TIE |
| `locomo_conv-44` / `q10_multi_hop` | `sliding_window` | judge | 3.0 / 3.0 | 6820 / 6759 | 0.9% | 0.44 / 0.44 | +0.9% | ACGC_WIN |
| `locomo_conv-44` / `q110_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6816 / 6238 | 8.5% | 0.29 / 0.80 | +173.2% | ACGC_WIN |
| `locomo_conv-44` / `q110_single_hop` | `sliding_window` | judge | 2.0 / 5.0 | 6816 / 6755 | 0.9% | 0.29 / 0.74 | +152.3% | ACGC_WIN |
| `locomo_conv-44` / `q126_adversarial` | `acgc` | judge | 2.0 / 5.0 | 6818 / 6238 | 8.5% | 0.29 / 0.80 | +173.2% | ACGC_WIN |
| `locomo_conv-44` / `q126_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6818 / 6757 | 0.9% | 0.29 / 0.74 | +152.3% | ACGC_WIN |
| `locomo_conv-44` / `q127_adversarial` | `acgc` | judge | 2.0 / 5.0 | 6815 / 6239 | 8.5% | 0.29 / 0.80 | +173.1% | ACGC_WIN |
| `locomo_conv-44` / `q127_adversarial` | `sliding_window` | judge | 2.0 / 5.0 | 6815 / 6754 | 0.9% | 0.29 / 0.74 | +152.3% | ACGC_WIN |
| `locomo_conv-44` / `q128_adversarial` | `acgc` | judge | 1.0 / 5.0 | 6815 / 6241 | 8.4% | 0.15 / 0.80 | +446.0% | ACGC_WIN |
| `locomo_conv-44` / `q128_adversarial` | `sliding_window` | judge | 1.0 / 5.0 | 6815 / 6754 | 0.9% | 0.15 / 0.74 | +404.5% | ACGC_WIN |
| `locomo_conv-44` / `q148_adversarial` | `acgc` | judge | 5.0 / 2.0 | 6811 / 6231 | 8.5% | 0.73 / 0.32 | -56.3% | ACGC_LOSS |
| `locomo_conv-44` / `q148_adversarial` | `sliding_window` | judge | 5.0 / 2.0 | 6811 / 6750 | 0.9% | 0.73 / 0.30 | -59.6% | ACGC_LOSS |
| `locomo_conv-44` / `q157_adversarial` | `acgc` | judge | 5.0 / 2.0 | 6819 / 6231 | 8.6% | 0.73 / 0.32 | -56.2% | ACGC_LOSS |
| `locomo_conv-44` / `q157_adversarial` | `sliding_window` | judge | 5.0 / 1.0 | 6819 / 6758 | 0.9% | 0.73 / 0.15 | -79.8% | ACGC_LOSS |
| `locomo_conv-44` / `q22_multi_hop` | `acgc` | judge | 3.0 / 2.0 | 6815 / 6232 | 8.6% | 0.44 / 0.32 | -27.1% | TIE |
| `locomo_conv-44` / `q22_multi_hop` | `sliding_window` | judge | 3.0 / 2.0 | 6815 / 6754 | 0.9% | 0.44 / 0.30 | -32.7% | TIE |
| `locomo_conv-44` / `q24_multi_hop` | `acgc` | judge | 5.0 / 3.0 | 6815 / 6240 | 8.4% | 0.73 / 0.48 | -34.5% | ACGC_LOSS |
| `locomo_conv-44` / `q24_multi_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6815 / 6754 | 0.9% | 0.73 / 0.30 | -59.6% | ACGC_LOSS |
| `locomo_conv-44` / `q2_temporal` | `acgc` | judge | 5.0 / 1.0 | 6815 / 6242 | 8.4% | 0.73 / 0.16 | -78.2% | ACGC_LOSS |
| `locomo_conv-44` / `q2_temporal` | `sliding_window` | judge | 5.0 / 1.0 | 6815 / 6754 | 0.9% | 0.73 / 0.15 | -79.8% | ACGC_LOSS |
| `locomo_conv-44` / `q41_multi_hop` | `acgc` | judge | 5.0 / 3.0 | 6819 / 6237 | 8.5% | 0.73 / 0.48 | -34.4% | ACGC_LOSS |
| `locomo_conv-44` / `q41_multi_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6819 / 6758 | 0.9% | 0.73 / 0.74 | +0.9% | ACGC_WIN |
| `locomo_conv-44` / `q43_multi_hop` | `acgc` | judge | 5.0 / 5.0 | 6813 / 6237 | 8.5% | 0.73 / 0.80 | +9.2% | ACGC_WIN |
| `locomo_conv-44` / `q43_multi_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6813 / 6752 | 0.9% | 0.73 / 0.74 | +0.9% | ACGC_WIN |
| `locomo_conv-44` / `q51_multi_hop` | `acgc` | judge | 1.0 / 5.0 | 6817 / 6239 | 8.5% | 0.15 / 0.80 | +446.3% | ACGC_WIN |
| `locomo_conv-44` / `q51_multi_hop` | `sliding_window` | judge | 1.0 / 1.0 | 6817 / 6756 | 0.9% | 0.15 / 0.15 | +0.9% | ACGC_WIN |
| `locomo_conv-44` / `q63_single_hop` | `acgc` | judge | 5.0 / 2.0 | 6817 / 6235 | 8.5% | 0.73 / 0.32 | -56.3% | ACGC_LOSS |
| `locomo_conv-44` / `q63_single_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6817 / 6756 | 0.9% | 0.73 / 0.74 | +0.9% | ACGC_WIN |
| `locomo_conv-44` / `q66_single_hop` | `acgc` | judge | 5.0 / 2.0 | 6815 / 6236 | 8.5% | 0.73 / 0.32 | -56.3% | ACGC_LOSS |
| `locomo_conv-44` / `q66_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6815 / 6754 | 0.9% | 0.73 / 0.30 | -59.6% | ACGC_LOSS |
| `locomo_conv-44` / `q68_single_hop` | `acgc` | judge | 5.0 / 2.0 | 6822 / 6236 | 8.6% | 0.73 / 0.32 | -56.2% | ACGC_LOSS |
| `locomo_conv-44` / `q68_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6822 / 6761 | 0.9% | 0.73 / 0.30 | -59.6% | ACGC_LOSS |
| `locomo_conv-44` / `q73_single_hop` | `acgc` | judge | 5.0 / 2.0 | 6815 / 6234 | 8.5% | 0.73 / 0.32 | -56.3% | ACGC_LOSS |
| `locomo_conv-44` / `q73_single_hop` | `sliding_window` | judge | 5.0 / 2.0 | 6815 / 6754 | 0.9% | 0.73 / 0.30 | -59.6% | ACGC_LOSS |
| `locomo_conv-44` / `q79_single_hop` | `acgc` | judge | 5.0 / 2.0 | 6830 / 6238 | 8.7% | 0.73 / 0.32 | -56.2% | ACGC_LOSS |
| `locomo_conv-44` / `q79_single_hop` | `sliding_window` | judge | 5.0 / 1.0 | 6830 / 6769 | 0.9% | 0.73 / 0.15 | -79.8% | ACGC_LOSS |
| `locomo_conv-44` / `q83_single_hop` | `acgc` | judge | 2.0 / 5.0 | 6813 / 6235 | 8.5% | 0.29 / 0.80 | +173.2% | ACGC_WIN |
| `locomo_conv-44` / `q83_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6813 / 6752 | 0.9% | 0.29 / 0.30 | +0.9% | ACGC_WIN |
| `locomo_conv-44` / `q87_single_hop` | `acgc` | judge | 2.0 / 2.0 | 6816 / 6235 | 8.5% | 0.29 / 0.32 | +9.3% | ACGC_WIN |
| `locomo_conv-44` / `q87_single_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6816 / 6755 | 0.9% | 0.29 / 0.30 | +0.9% | ACGC_WIN |
| `locomo_conv-47` / `q35_multi_hop` | `acgc` | judge | 2.0 / 3.0 | 6900 / 6241 | 9.6% | 0.29 / 0.48 | +65.8% | ACGC_WIN |
| `locomo_conv-47` / `q35_multi_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6900 / 6895 | 0.1% | 0.29 / 0.29 | +0.1% | ACGC_WIN |
| `locomo_conv-47` / `q3_multi_hop` | `acgc` | judge | 2.0 / 2.0 | 6902 / 6245 | 9.5% | 0.29 / 0.32 | +10.5% | ACGC_WIN |
| `locomo_conv-47` / `q3_multi_hop` | `sliding_window` | judge | 2.0 / 2.0 | 6902 / 6897 | 0.1% | 0.29 / 0.29 | +0.1% | ACGC_WIN |
| `locomo_conv-47` / `q41_multi_hop` | `acgc` | judge | 3.0 / 0.0 | 6899 / 6248 | 9.4% | 0.43 / 0.00 | -100.0% | ACGC_LOSS |
| `locomo_conv-47` / `q41_multi_hop` | `sliding_window` | judge | 3.0 / 2.0 | 6899 / 6894 | 0.1% | 0.43 / 0.29 | -33.3% | TIE |
| `locomo_conv-47` / `q4_multi_hop` | `acgc` | judge | 5.0 / 5.0 | 6897 / 6241 | 9.5% | 0.72 / 0.80 | +10.5% | ACGC_WIN |
| `locomo_conv-47` / `q4_multi_hop` | `sliding_window` | judge | 5.0 / 5.0 | 6897 / 6892 | 0.1% | 0.72 / 0.73 | +0.1% | ACGC_WIN |

## Response samples

### `locomo_conv-26::q102_single_hop`

**Question:** Did Melanie make the black and white bowl in the photo?

**naive_full_history (ref)** (6600 prompt tokens, 3230 ms):

> Yes. Melanie said she made that black-and-white bowl in her pottery class.

**sliding_window** (6599 prompt tokens, 15784 ms):

> Yes—if you mean the pottery photo Melanie shared of her bowls. She said, “Check out my pottery creation in the pic,” so the black-and-white bowl in that photo would be one she made. If you’re referring to a different photo, let me know.

**acgc** (6024 prompt tokens, 5249 ms):

> Yes. Melanie said she made that bowl in her pottery class.

---

### `locomo_conv-26::q103_single_hop`

**Question:** What kind of books does Caroline have in her library?

**naive_full_history (ref)** (6599 prompt tokens, 4241 ms):

> Children’s books—classics, stories from different cultures, and educational books.

**sliding_window** (6598 prompt tokens, 6816 ms):

> It’s not specified in the conversation. We don’t have details about the kinds of books in Caroline’s library.

**acgc** (6026 prompt tokens, 5693 ms):

> It’s not specified in the provided context. We only see Melanie ask Caroline about her library, but there’s no recorded reply describing the books.

---

### `locomo_conv-26::q11_temporal`

**Question:** How long has Caroline had her current group of friends for?

**naive_full_history (ref)** (6600 prompt tokens, 4153 ms):

> About four years, since she moved from her home country.

**sliding_window** (6599 prompt tokens, 4022 ms):

> It isn’t specified in the provided conversation.

**acgc** (6028 prompt tokens, 6215 ms):

> It isn’t specified in the provided context.

---

### `locomo_conv-26::q132_single_hop`

**Question:** Who is Melanie a fan of in terms of modern music?

**naive_full_history (ref)** (6600 prompt tokens, 5510 ms):

> It isn’t mentioned in the provided conversation.

**sliding_window** (6599 prompt tokens, 2588 ms):

> Ed Sheeran.

**acgc** (6019 prompt tokens, 2487 ms):

> Ed Sheeran.

---

### `locomo_conv-26::q137_single_hop`

**Question:** What does Melanie do to keep herself busy during her pottery break?

**naive_full_history (ref)** (6601 prompt tokens, 8990 ms):

> It isn’t mentioned in the conversation. Melanie doesn’t say what she does during a pottery break.

**sliding_window** (6600 prompt tokens, 2845 ms):

> She’s been reading the book you recommended and painting.

**acgc** (6029 prompt tokens, 3867 ms):

> She kept busy by reading the book you recommended and painting.

---

### `locomo_conv-26::q145_single_hop`

**Question:** How did Melanie's son handle the accident?

**naive_full_history (ref)** (6597 prompt tokens, 4238 ms):

> I don’t see any mention of an accident (or how Melanie’s son handled one) in the provided conversation. Could you share more details or when this happened?

**sliding_window** (6596 prompt tokens, 4286 ms):

> He was okay after the accident.

**acgc** (6027 prompt tokens, 6250 ms):

> He was okay—no serious injuries. The accident scared everyone, but he handled it fine, and his siblings were reassured he’d be alright.

---

### `locomo_conv-26::q147_single_hop`

**Question:** How did Melanie's children handle the accident?

**naive_full_history (ref)** (6597 prompt tokens, 5238 ms):

> I don’t see any mention of an accident in the provided conversation, so there’s no info on how Melanie’s children handled it. Can you share more details about the accident or when it happened?

**sliding_window** (6596 prompt tokens, 3165 ms):

> They were scared at first, but Melanie reassured them their brother would be OK, and they handled it well—she said they’re tough kids.

**acgc** (6020 prompt tokens, 4521 ms):

> They were scared at first, but after Melanie reassured them and explained their brother would be okay, they handled it well—she said they’re tough kids.

---

### `locomo_conv-26::q148_single_hop`

**Question:** How did Melanie feel after the accident?

**naive_full_history (ref)** (6596 prompt tokens, 4040 ms):

> I don’t see any mention of an accident in the conversation, so there’s no info on how Melanie felt afterward. Could you share more context or where the accident is referenced?

**sliding_window** (6595 prompt tokens, 4931 ms):

> She was really scared and “freaked” during the accident, then relieved and grateful her son was okay—it reminded her to cherish her family.

**acgc** (6021 prompt tokens, 4416 ms):

> She was really scared and freaked out, but very relieved and grateful her son was okay. The scare reminded her that life is precious and made her appreciate her family even more.

---

### `locomo_conv-26::q152_single_hop`

**Question:** What did Melanie do after the road trip to relax?

**naive_full_history (ref)** (6599 prompt tokens, 12298 ms):

> I don’t see any road trip mentioned in the provided chat. The closest is a camping trip; after that, Melanie said she had a quiet weekend to unplug and spend time with her kids. If you meant a different trip, can you share more details?

**sliding_window** (6598 prompt tokens, 28123 ms):

> She went on a camping trip with her family.

**acgc** (6021 prompt tokens, 5297 ms):

> She went camping with her family.

---

### `locomo_conv-26::q168_adversarial`

**Question:** Did Caroline make the black and white bowl in the photo?

**naive_full_history (ref)** (6600 prompt tokens, 3619 ms):

> No. Melanie made the black and white bowl in her pottery class.

**sliding_window** (6599 prompt tokens, 5597 ms):

> No. The black and white bowl was made by Melanie, not Caroline—Caroline even said she hadn’t tried pottery yet.

**acgc** (6014 prompt tokens, 4885 ms):

> No. Melanie made the bowl (she said she made it in her pottery class).

---

### `locomo_conv-26::q182_adversarial`

**Question:** What did Melanie make for a local church?

**naive_full_history (ref)** (6597 prompt tokens, 4077 ms):

> It isn’t mentioned in the provided conversation that Melanie made anything for a local church. If you have a specific message or date in mind, please share it and I’ll check.

**sliding_window** (6596 prompt tokens, 4246 ms):

> She didn’t make anything for a local church—Caroline did. Caroline made a stained glass window for a local church.

**acgc** (6027 prompt tokens, 22886 ms):

> It isn’t specified in the provided conversation.

---

### `locomo_conv-26::q43_open_domain`

**Question:** Would Melanie be more interested in going to a national park or a theme park?

**naive_full_history (ref)** (6604 prompt tokens, 6093 ms):

> A national park. Melanie loves camping, hiking, and exploring nature with her family, so a national park fits her interests better than a theme park.

**sliding_window** (6603 prompt tokens, 3858 ms):

> A national park. Melanie loves nature—camping, hiking, park visits, and trips like the Grand Canyon—and often talks about how peaceful and refreshing the outdoors is.

**acgc** (6026 prompt tokens, 3422 ms):

> A national park. Melanie loves nature, camping, hiking, stargazing, and has great memories from places like the Grand Canyon—she consistently talks about how peaceful and refreshing the outdoors is.

---

