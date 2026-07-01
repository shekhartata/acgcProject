# ACGC Context-Strategy Evaluation

**Generated:** 2026-07-01T14:38:08+05:30  
**Model:** `gpt-5`  
**Tokenizer:** `o200k_base`  
**Reference strategy:** `naive_full_history`  
**Strategies compared:** `naive_full_history`, `sliding_window`, `acgc`  
**Live tokens spent this run:** 41393  

## Strategy comparison (side by side)

| Strategy | Probes | Avg Quality | Avg Prompt Tok | Avg Latency (ms) | Avg IPT | Tok Red% vs ref | Quality Δ vs ref | IPT Δ% vs ref |
|---|---|---|---|---|---|---|---|---|
| `naive_full_history (ref)` | 8 | 4.38 | 911 | 12499 | 5.93 | 0.0% | +0.00 | +0.0% |
| `sliding_window` | 8 | 3.62 | 911 | 11891 | 5.06 | 0.0% | -0.75 | -14.7% |
| `acgc` | 8 | 3.75 | 824 | 12238 | 5.59 | 9.6% | -0.62 | -5.8% |

## Candidate vs reference (verdicts)

Reference: `naive_full_history`

- **Pairs evaluated:** 16
- **Avg quality (reference):** 4.38 / 5.0
- **Avg quality (candidate):** 3.69 / 5.0
- **Avg quality delta:** -0.69 (candidate - reference)
- **Avg token reduction:** 4.5%
- **Avg IPT (reference):** 5.93
- **Avg IPT (candidate):** 5.33
- **Avg IPT delta:** -10.2%
- **Quality regressions (>1.0 drop):** 2

### Verdict breakdown

- `ACGC_WIN` (better IPT, no quality loss): **6**
- `ACGC_WIN_STAR` (better IPT, but quality dropped): **0**
- `TIE`: **8**
- `ACGC_LOSS`: **2**
- `BASELINE_WIN` (reference strictly better): **0**

## Per-probe results

| Scenario / Probe | Candidate | Method | Quality (ref / cand) | Tokens (ref / cand) | Token Red% | IPT (ref / cand) | IPT Δ% | Verdict |
|---|---|---|---|---|---|---|---|---|
| `constraint_adherence_1` / `p1` | `acgc` | judge | 5.0 / 0.0 | 842 / 761 | 9.6% | 5.94 / 0.00 | -100.0% | ACGC_LOSS |
| `constraint_adherence_1` / `p1` | `sliding_window` | judge | 5.0 / 0.0 | 842 / 842 | 0.0% | 5.94 / 0.00 | -100.0% | ACGC_LOSS |
| `contradiction_1` / `p1` | `acgc` | judge | 5.0 / 5.0 | 969 / 878 | 9.4% | 5.16 / 5.69 | +10.4% | ACGC_WIN |
| `contradiction_1` / `p1` | `sliding_window` | judge | 5.0 / 4.0 | 969 / 969 | 0.0% | 5.16 / 4.13 | -20.0% | TIE |
| `long_range_recall_1` / `p1` | `acgc` | probe | 5.0 / 5.0 | 1093 / 978 | 10.5% | 4.57 / 5.11 | +11.8% | ACGC_WIN |
| `long_range_recall_1` / `p1` | `sliding_window` | probe | 5.0 / 5.0 | 1093 / 1093 | 0.0% | 4.57 / 4.57 | +0.0% | TIE |
| `long_range_recall_1` / `p2` | `acgc` | probe | 5.0 / 5.0 | 1089 / 974 | 10.6% | 4.59 / 5.13 | +11.8% | ACGC_WIN |
| `long_range_recall_1` / `p2` | `sliding_window` | probe | 5.0 / 5.0 | 1089 / 1089 | 0.0% | 4.59 / 4.59 | +0.0% | TIE |
| `long_range_recall_1` / `p3` | `acgc` | probe | 5.0 / 5.0 | 1093 / 978 | 10.5% | 4.57 / 5.11 | +11.8% | ACGC_WIN |
| `long_range_recall_1` / `p3` | `sliding_window` | probe | 5.0 / 5.0 | 1093 / 1093 | 0.0% | 4.57 / 4.57 | +0.0% | TIE |
| `multi_hop_synth_1` / `p1` | `acgc` | judge | 0.0 / 0.0 | 1111 / 1000 | 10.0% | 0.00 / 0.00 | +0.0% | TIE |
| `multi_hop_synth_1` / `p1` | `sliding_window` | judge | 0.0 / 0.0 | 1111 / 1111 | 0.0% | 0.00 / 0.00 | +0.0% | TIE |
| `recent_recall_1` / `p1` | `acgc` | probe | 5.0 / 5.0 | 307 / 298 | 2.9% | 16.29 / 16.78 | +3.0% | ACGC_WIN |
| `recent_recall_1` / `p1` | `sliding_window` | probe | 5.0 / 5.0 | 307 / 307 | 0.0% | 16.29 / 16.29 | +0.0% | TIE |
| `topic_switch_return_1` / `p1` | `acgc` | probe | 5.0 / 5.0 | 787 / 724 | 8.0% | 6.35 / 6.91 | +8.7% | ACGC_WIN |
| `topic_switch_return_1` / `p1` | `sliding_window` | probe | 5.0 / 5.0 | 787 / 787 | 0.0% | 6.35 / 6.35 | +0.0% | TIE |

## Response samples

### `constraint_adherence_1::p1`

**Question:** Now write me a simple in-memory cache implementation for one of my services. Pick whatever language and supporting tools you think are best for this project — full freedom.

**naive_full_history (ref)** (842 prompt tokens, 23304 ms):

> Using Go per your constraints. Here’s a small, concurrency-safe in-memory cache with optional per-item TTL and a background janitor to evict expired entries. No external dependencies.
> 
> package cache
> 
> import (
> 	"sync"
> 	"time"
> )
> 
> type Cache struct {
> 	mu           sync.RWMutex
> 	items        map[string]entry
> 	defaultTTL   time.Duration
> 	stopCh       chan struct{}
> 	wg           sync.WaitGroup
> 	onEvict      func(key string, val any)
> 	cleanupEvery time.Duration
> }
> 
> type entry struct {
> 	value     any
> 	expiresAt time.Time // zero means no expiration
> }
> 
> // New creates a cache.
> // - defaultTTL <= 0 means entries never expire by default.
> // - cleanupEvery <= 0 disables background cleanup (expired entries are evicted lazily on Get).
> // - onEvict (optional) is called after an entry is removed due to expiration (not on explicit Delete).
> func New(defaultTTL, cleanupEvery time.Duration, onEvict func(string, any)) *Cache {
> 	c := &Cache{
> 		items:        make(map[string]entry),
> 		defaultTTL:   defaultTTL,
> 		stopCh:       make(chan struct{}),
> 		onEvict:      onEvict,
> 		cleanupEvery: cleanupEvery,
> 	}
> 	if cleanupEvery > 0 {
> 		c.wg.Add(1)
> 		go c.janitor()
> 	}
> 	return c
> }
> 
> // Set stores a value with an opt...[truncated]

**sliding_window** (842 prompt tokens, 27364 ms):

> 

**acgc** (761 prompt tokens, 34440 ms):

> 

---

### `contradiction_1::p1`

**Question:** Where are we storing user sessions, and what mechanism are we using to identify them?

**naive_full_history (ref)** (969 prompt tokens, 4848 ms):

> - Storage: Server-side in our database (sessions table).
> - Identification: An opaque, random session ID (not a JWT) sent in an HttpOnly, Secure cookie (SameSite=Lax).

**sliding_window** (969 prompt tokens, 4261 ms):

> - Storage: Server-side sessions in the database (sessions table).
> - Identification: An opaque session ID (not a JWT) stored in an HttpOnly cookie sent with each request.

**acgc** (878 prompt tokens, 10011 ms):

> Stateful. We store sessions server-side in the database (sessions table: id, user_id, expires_at, last_seen_at, ip_hash). Clients are identified by an opaque session ID sent in an HttpOnly, SameSite=Lax cookie (not a JWT); the server looks up the session by that ID.

---

### `long_range_recall_1::p1`

**Question:** Quick recap: which database did we agree on for the analytics dashboard?

**naive_full_history (ref)** (1093 prompt tokens, 3227 ms):

> TimescaleDB.

**sliding_window** (1093 prompt tokens, 2865 ms):

> TimescaleDB.

**acgc** (978 prompt tokens, 2786 ms):

> TimescaleDB as the primary database.

---

### `long_range_recall_1::p2`

**Question:** And which message bus did we pick for ingestion?

**naive_full_history (ref)** (1089 prompt tokens, 3610 ms):

> NATS JetStream.

**sliding_window** (1089 prompt tokens, 2726 ms):

> NATS JetStream.

**acgc** (974 prompt tokens, 3057 ms):

> NATS JetStream.

---

### `long_range_recall_1::p3`

**Question:** Remind me where we decided to deploy this — which cloud provider?

**naive_full_history (ref)** (1093 prompt tokens, 4498 ms):

> Hetzner Cloud — we chose it over AWS due to cost, especially for an IO‑heavy workload.

**sliding_window** (1093 prompt tokens, 4718 ms):

> Hetzner Cloud — we chose it over AWS for lower cost on IO‑heavy workloads.

**acgc** (978 prompt tokens, 3726 ms):

> Hetzner Cloud — chosen over AWS for better cost/performance on IO‑heavy workloads.

---

### `multi_hop_synth_1::p1`

**Question:** Given everything we've discussed about this system, can you tell me realistically whether the budget is sufficient? Walk through the math.

**naive_full_history (ref)** (1111 prompt tokens, 45114 ms):

> 

**sliding_window** (1111 prompt tokens, 40814 ms):

> 

**acgc** (1000 prompt tokens, 34966 ms):

> 

---

### `recent_recall_1::p1`

**Question:** Quick reminder — should invoice numbers be sequential or random, and why?

**naive_full_history (ref)** (307 prompt tokens, 10533 ms):

> Sequential. Many jurisdictions require gap-free, auditable invoice numbering for tax compliance; random IDs break that and make audits/reconciliation harder.

**sliding_window** (307 prompt tokens, 8371 ms):

> Sequential. Many jurisdictions require chronological, gap-free invoice numbering for tax compliance and audits; random IDs make reconciliation and auditing harder.

**acgc** (298 prompt tokens, 3123 ms):

> Sequential. Many jurisdictions require gap-free, sequential invoice numbering for tax/audit compliance. Use a dedicated PostgreSQL sequence per organization and assign the number within the same transaction that creates the invoice to avoid gaps under concurrency.

---

### `topic_switch_return_1::p1`

**Question:** Okay back to recipes — remind me of the exact scoring formula we agreed on for ranking recommendations.

**naive_full_history (ref)** (787 prompt tokens, 4858 ms):

> Final score = 0.6 * pantry_coverage + 0.3 * tfidf_similarity_to_liked_recipes + 0.1 * recency_boost
> 
> Make sure each component is normalized to 0–1.

**sliding_window** (787 prompt tokens, 4009 ms):

> Final score = 0.6 × pantry_coverage + 0.3 × tfidf_similarity_to_liked_recipes + 0.1 × recency_boost
> 
> - pantry_coverage: fraction of recipe ingredients the user already has
> - tfidf_similarity_to_liked_recipes: cosine similarity to the user’s liked recipes (TF-IDF on ingredients)
> - recency_boost: small boost for seasonal/trending/fresh content
> 
> Normalize each term to 0–1 before weighting.

**acgc** (724 prompt tokens, 5797 ms):

> Final score = 0.6 × pantry_coverage + 0.3 × tfidf_similarity_to_liked_recipes + 0.1 × recency_boost
> 
> Assume each component is normalized to 0–1; weights sum to 1.

---

