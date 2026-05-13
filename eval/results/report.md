# ACGC Quality & Intelligence-Per-Token Evaluation

**Generated:** 2026-05-13T11:25:04+05:30  
**Model:** `gpt-5`  
**Live tokens spent this run:** 27900  

## Aggregate

- **Pairs evaluated:** 8
- **Avg quality (baseline):** 3.44 / 5.0
- **Avg quality (ACGC):** 3.75 / 5.0
- **Avg quality delta:** +0.31 (ACGC - baseline)
- **Avg token reduction:** 10.9%
- **Avg IPT (baseline):** 4.10
- **Avg IPT (ACGC):** 5.59
- **Avg IPT delta:** +36.4%
- **Quality regressions (>1.0 drop):** 0

### Verdict breakdown

- `ACGC_WIN` (better IPT, no quality loss): **6**
- `ACGC_WIN_STAR` (better IPT, but quality dropped — motivates semantic search): **0**
- `TIE`: **2**
- `ACGC_LOSS`: **0**
- `BASELINE_WIN`: **0**

## Per-probe results

| Scenario / Probe | Method | Quality (B / A) | Tokens (B / A) | Token Red% | IPT (B / A) | IPT Δ% | Verdict |
|---|---|---|---|---|---|---|---|
| `constraint_adherence_1` / `p1` | judge | 0.0 / 0.0 | 864 / 761 | 11.9% | 0.00 / 0.00 | +0.0% | TIE |
| `contradiction_1` / `p1` | judge | 5.0 / 5.0 | 993 / 878 | 11.6% | 5.04 / 5.69 | +13.1% | ACGC_WIN |
| `long_range_recall_1` / `p1` | probe | 5.0 / 5.0 | 1125 / 978 | 13.1% | 4.44 / 5.11 | +15.0% | ACGC_WIN |
| `long_range_recall_1` / `p2` | probe | 5.0 / 5.0 | 1121 / 974 | 13.1% | 4.46 / 5.13 | +15.1% | ACGC_WIN |
| `long_range_recall_1` / `p3` | probe | 5.0 / 5.0 | 1125 / 978 | 13.1% | 4.44 / 5.11 | +15.0% | ACGC_WIN |
| `multi_hop_synth_1` / `p1` | judge | 0.0 / 0.0 | 1143 / 1000 | 12.5% | 0.00 / 0.00 | +0.0% | TIE |
| `recent_recall_1` / `p1` | probe | 2.5 / 5.0 | 305 / 298 | 2.3% | 8.20 / 16.78 | +104.7% | ACGC_WIN |
| `topic_switch_return_1` / `p1` | probe | 5.0 / 5.0 | 804 / 724 | 10.0% | 6.22 / 6.91 | +11.0% | ACGC_WIN |

## Side-by-side response samples

### `constraint_adherence_1::p1`

**Question:** Now write me a simple in-memory cache implementation for one of my services. Pick whatever language and supporting tools you think are best for this project — full freedom.

**Baseline** (864 prompt tokens, 49711 ms):

> 

**ACGC** (761 prompt tokens, 47757 ms):

> 

---

### `contradiction_1::p1`

**Question:** Where are we storing user sessions, and what mechanism are we using to identify them?

**Baseline** (993 prompt tokens, 8214 ms):

> - Storage: Server-side sessions in our database (sessions table: id, user_id, expires_at, last_seen_at, etc.).
> - Identification: An opaque, random session ID (not a JWT) stored in an HttpOnly, Secure, SameSite=Lax cookie (e.g., sid). The server reads the cookie and looks up the session in the DB.

**ACGC** (878 prompt tokens, 12060 ms):

> - Storage: Server-side sessions in the database (sessions table: id, user_id, expires_at, last_seen_at, ip_hash).
> - Identifier: An opaque, random session ID stored in an HttpOnly, Secure cookie (SameSite=Lax). The server looks up the session by this ID and can revoke/expire it server-side.
> 
> Note: We’re not using JWTs in localStorage; we switched to stateful, cookie-based sessions. CSRF protection is required.

---

### `long_range_recall_1::p1`

**Question:** Quick recap: which database did we agree on for the analytics dashboard?

**Baseline** (1125 prompt tokens, 4090 ms):

> TimescaleDB.

**ACGC** (978 prompt tokens, 6027 ms):

> TimescaleDB — chosen over InfluxDB so we can run relational queries alongside time-series data.

---

### `long_range_recall_1::p2`

**Question:** And which message bus did we pick for ingestion?

**Baseline** (1121 prompt tokens, 4887 ms):

> NATS JetStream — we chose it over Kafka for simpler ops with comparable throughput at our scale.

**ACGC** (974 prompt tokens, 3113 ms):

> NATS JetStream.

---

### `long_range_recall_1::p3`

**Question:** Remind me where we decided to deploy this — which cloud provider?

**Baseline** (1125 prompt tokens, 5375 ms):

> Hetzner Cloud — we chose it over AWS primarily for lower cost on IO‑heavy workloads.

**ACGC** (978 prompt tokens, 6601 ms):

> Hetzner Cloud. We chose it over AWS for lower cost and better IO performance for your workload.

---

### `multi_hop_synth_1::p1`

**Question:** Given everything we've discussed about this system, can you tell me realistically whether the budget is sufficient? Walk through the math.

**Baseline** (1143 prompt tokens, 52990 ms):

> 

**ACGC** (1000 prompt tokens, 123490 ms):

> 

---

### `recent_recall_1::p1`

**Question:** Quick reminder — should invoice numbers be sequential or random, and why?

**Baseline** (305 prompt tokens, 15076 ms):

> Sequential. Most tax/audit rules expect a unique, time-ordered invoice series per organization so documents are easy to track; random IDs break that audit trail. Use a per-org sequence/series; gaps from canceled/rolled-back invoices are usually acceptable if you keep the records (use a true gapless counter only if your jurisdiction mandates it).

**ACGC** (298 prompt tokens, 7822 ms):

> Sequential. Many jurisdictions require gap-free, sequential invoice numbers for tax/audit compliance. Use a dedicated PostgreSQL sequence per organization and assign the number in the same transaction that creates the invoice to avoid gaps under concurrency.

---

### `topic_switch_return_1::p1`

**Question:** Okay back to recipes — remind me of the exact scoring formula we agreed on for ranking recommendations.

**Baseline** (804 prompt tokens, 5195 ms):

> Final score = 0.6 × pantry_coverage + 0.3 × tfidf_similarity_to_liked_recipes + 0.1 × recency_boost
> 
> All three components should be normalized to 0–1 before weighting.

**ACGC** (724 prompt tokens, 15194 ms):

> Final score = 0.6 × pantry_coverage + 0.3 × tfidf_similarity_to_liked_recipes + 0.1 × recency_boost
> 
> - pantry_coverage: fraction (0–1) of a recipe’s ingredients the user already has
> - tfidf_similarity_to_liked_recipes: cosine similarity to the user’s liked recipes (e.g., mean or max)
> - recency_boost: normalized 0–1 freshness/trending factor

---

