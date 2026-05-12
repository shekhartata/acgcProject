# ACGC Quality & Intelligence-Per-Token Evaluation

**Generated:** 2026-05-12T13:35:38+05:30  
**Model:** `gpt-5`  
**Live tokens spent this run:** 0  

## Aggregate

- **Pairs evaluated:** 5
- **Avg quality (baseline):** 5.00 / 5.0
- **Avg quality (ACGC):** 1.00 / 5.0
- **Avg quality delta:** -4.00 (ACGC - baseline)
- **Avg token reduction:** 57.4%
- **Avg IPT (baseline):** 7.19
- **Avg IPT (ACGC):** 3.18
- **Avg IPT delta:** -55.7%
- **Quality regressions (>1.0 drop):** 4

### Verdict breakdown

- `ACGC_WIN` (better IPT, no quality loss): **0**
- `ACGC_WIN_STAR` (better IPT, but quality dropped — motivates semantic search): **0**
- `TIE`: **1**
- `ACGC_LOSS`: **4**
- `BASELINE_WIN`: **0**

## Per-probe results

| Scenario / Probe | Method | Quality (B / A) | Tokens (B / A) | Token Red% | IPT (B / A) | IPT Δ% | Verdict |
|---|---|---|---|---|---|---|---|
| `long_range_recall_1` / `p1` | probe | 5.0 / 0.0 | 1125 / 266 | 76.4% | 4.44 / 0.00 | -100.0% | ACGC_LOSS |
| `long_range_recall_1` / `p2` | probe | 5.0 / 0.0 | 1121 / 258 | 77.0% | 4.46 / 0.00 | -100.0% | ACGC_LOSS |
| `long_range_recall_1` / `p3` | probe | 5.0 / 0.0 | 1125 / 266 | 76.4% | 4.44 / 0.00 | -100.0% | ACGC_LOSS |
| `recent_recall_1` / `p1` | probe | 5.0 / 5.0 | 305 / 314 | -3.0% | 16.39 / 15.92 | -2.9% | TIE |
| `topic_switch_return_1` / `p1` | probe | 5.0 / 0.0 | 804 / 320 | 60.2% | 6.22 / 0.00 | -100.0% | ACGC_LOSS |

## Side-by-side response samples

### `constraint_adherence_1::p1`

**Question:** Now write me a simple in-memory cache implementation for one of my services. Pick whatever language and supporting tools you think are best for this project — full freedom.

**Baseline** (864 prompt tokens, 113646 ms):

> 

**ACGC** (314 prompt tokens, 174743 ms):

> 

---

### `contradiction_1::p1`

**Question:** Where are we storing user sessions, and what mechanism are we using to identify them?

**Baseline** (993 prompt tokens, 16537 ms):

> - Storage: Server-side sessions in our database (sessions table: id, user_id, expires_at, last_seen_at, etc.).
> - Identification: An opaque session ID (not a JWT) stored in an HttpOnly, Secure, SameSite=Lax cookie. The server reads the cookie, looks up the session by ID, and authenticates the user.

**ACGC** (292 prompt tokens, 43826 ms):

> I don’t have visibility into your deployed config here. In our standard setup:
> 
> - Storage: Server-side sessions are stored in Redis (key prefix sess:, sliding TTL).
> - Identification: Clients are identified by an opaque session ID carried in an HttpOnly, Secure cookie (name: sid). For API-only calls, we accept the same cookie; Authorization: Bearer is supported only for PATs or service tokens, not user sessions.
> 
> If you’re running the stateless/JWT mode instead, there’s no server-side session storage; the signed JWT in the cookie/header identifies the user, with refresh tokens kept in Redis for revocation. Please confirm which mode your environment uses.

---

### `long_range_recall_1::p1`

**Question:** Quick recap: which database did we agree on for the analytics dashboard?

**Baseline** (1125 prompt tokens, 8895 ms):

> TimescaleDB.

**ACGC** (266 prompt tokens, 30124 ms):

> I don’t have a record of that decision in the provided context. Can you confirm which database we settled on, or want me to restate a recommendation based on your scale and stack?

---

### `long_range_recall_1::p2`

**Question:** And which message bus did we pick for ingestion?

**Baseline** (1121 prompt tokens, 8710 ms):

> NATS JetStream (not Kafka).

**ACGC** (258 prompt tokens, 33610 ms):

> I don’t have that decision in the current context. Which project/environment are you referring to? If you can point me to the doc or shortlist (e.g., Kafka/RabbitMQ/NATS/Pulsar/Kinesis/Pub/Sub), I’ll confirm the exact choice.

---

### `long_range_recall_1::p3`

**Question:** Remind me where we decided to deploy this — which cloud provider?

**Baseline** (1125 prompt tokens, 7807 ms):

> Hetzner Cloud — we chose it over AWS primarily for lower cost on IO‑heavy workloads.

**ACGC** (266 prompt tokens, 14322 ms):

> I don’t see a cloud provider specified in the provided notes. Could you confirm which one we chose (AWS, GCP, Azure, etc.)? If it isn’t decided yet, I can help compare options based on our auth, scaling, and monitoring goals.

---

### `multi_hop_synth_1::p1`

**Question:** Given everything we've discussed about this system, can you tell me realistically whether the budget is sufficient? Walk through the math.

**Baseline** (1143 prompt tokens, 106619 ms):

> 

**ACGC** (258 prompt tokens, 130745 ms):

> 

---

### `recent_recall_1::p1`

**Question:** Quick reminder — should invoice numbers be sequential or random, and why?

**Baseline** (305 prompt tokens, 20009 ms):

> Sequential. Many jurisdictions require chronologically increasing, non-reused invoice numbers for audit/compliance. If you need an unguessable public reference, add a separate random/UUID field, but keep the official invoice number sequential (typically per organization).

**ACGC** (314 prompt tokens, 26658 ms):

> Sequential.
> 
> - Why: Many jurisdictions require gap-free, chronological invoice numbering for tax compliance and audits. It also improves traceability and reconciliation.
> - How: Use a dedicated PostgreSQL sequence per organization and assign the number within the same transaction that creates the invoice to ensure no gaps even under concurrency.

---

### `topic_switch_return_1::p1`

**Question:** Okay back to recipes — remind me of the exact scoring formula we agreed on for ranking recommendations.

**Baseline** (804 prompt tokens, 19370 ms):

> Final score = 0.6 × pantry_coverage + 0.3 × tfidf_similarity_to_liked_recipes + 0.1 × recency_boost
> 
> Assuming each component is normalized to [0,1].

**ACGC** (320 prompt tokens, 116774 ms):

> I don’t have our earlier notes in this thread. If you can’t paste them, here’s the structure we used most recently:
> 
> - Hard filters first: exclude anything that violates allergies/dietary rules.
> - Base relevance (all features scaled 0–1):
>   z_i = w_pref*pref_match_i + w_time*time_fit_i + w_nut*diet_fit_i + w_rate*rating_i + w_pop*popularity_i + w_rec*recency/seasonality_i − w_cost*cost_i − w_miss*missing_ingredients_i
> - Diversity re-rank (MMR-style) when building the list:
>   final(i) = α*z_i − (1−α)*max_{j in S} sim(i, j), selected greedily into S
> - Tie-breakers: higher rating, fewer missing ingredients, then shorter prep time.
> 
> If you want the exact weights and α we agreed on, tell me the values and I’ll confirm, or I can propose defaults.

---

