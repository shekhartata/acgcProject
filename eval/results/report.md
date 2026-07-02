# ACGC Context-Strategy Evaluation

**Generated:** 2026-07-01T15:33:39+05:30  
**Model:** `gpt-5`  
**Tokenizer:** `o200k_base`  
**Reference strategy:** `naive_full_history`  
**Strategies compared:** `naive_full_history`, `sliding_window`, `acgc`  
**Live tokens spent this run:** 13009  

## Strategy comparison (side by side)

| Strategy | Probes | Avg Quality | Avg Prompt Tok | Avg Latency (ms) | Avg IPT | Tok Red% vs ref | Quality Δ vs ref | IPT Δ% vs ref |
|---|---|---|---|---|---|---|---|---|
| `naive_full_history (ref)` | 12 | 5.00 | 2738 | 15757 | 4.59 | 0.0% | +0.00 | +0.0% |
| `sliding_window` | 12 | 3.25 | 2733 | 21492 | 4.25 | 0.2% | -1.75 | -7.6% |
| `acgc` | 12 | 5.00 | 2082 | 13124 | 5.05 | 24.0% | +0.00 | +10.1% |

## Candidate vs reference (verdicts)

Reference: `naive_full_history`

- **Pairs evaluated:** 24
- **Avg quality (reference):** 5.00 / 5.0
- **Avg quality (candidate):** 4.12 / 5.0
- **Avg quality delta:** -0.88 (candidate - reference)
- **Avg token reduction:** 7.7%
- **Avg IPT (reference):** 4.59
- **Avg IPT (candidate):** 4.65
- **Avg IPT delta:** +1.3%
- **Quality regressions (>1.0 drop):** 4

### Verdict breakdown

- `ACGC_WIN` (better IPT, no quality loss): **12**
- `ACGC_WIN_STAR` (better IPT, but quality dropped): **0**
- `TIE`: **8**
- `ACGC_LOSS`: **4**
- `BASELINE_WIN` (reference strictly better): **0**

## Per-probe results

| Scenario / Probe | Candidate | Method | Quality (ref / cand) | Tokens (ref / cand) | Token Red% | IPT (ref / cand) | IPT Δ% | Verdict |
|---|---|---|---|---|---|---|---|---|
| `constraint_adherence_1` / `p1` | `acgc` | judge | 5.0 / 5.0 | 842 / 761 | 9.6% | 5.94 / 6.57 | +10.6% | ACGC_WIN |
| `constraint_adherence_1` / `p1` | `sliding_window` | judge | 5.0 / 5.0 | 842 / 842 | 0.0% | 5.94 / 5.94 | +0.0% | TIE |
| `contradiction_1` / `p1` | `acgc` | judge | 5.0 / 5.0 | 969 / 878 | 9.4% | 5.16 / 5.69 | +10.4% | ACGC_WIN |
| `contradiction_1` / `p1` | `sliding_window` | judge | 5.0 / 4.0 | 969 / 969 | 0.0% | 5.16 / 4.13 | -20.0% | TIE |
| `deep_history_recall_1` / `p1` | `acgc` | probe | 5.0 / 5.0 | 6395 / 4578 | 28.4% | 0.78 / 1.09 | +39.7% | ACGC_WIN |
| `deep_history_recall_1` / `p1` | `sliding_window` | probe | 5.0 / 0.0 | 6395 / 6380 | 0.2% | 0.78 / 0.00 | -100.0% | ACGC_LOSS |
| `deep_history_recall_1` / `p2` | `acgc` | probe | 5.0 / 5.0 | 6390 / 4592 | 28.1% | 0.78 / 1.09 | +39.2% | ACGC_WIN |
| `deep_history_recall_1` / `p2` | `sliding_window` | probe | 5.0 / 0.0 | 6390 / 6375 | 0.2% | 0.78 / 0.00 | -100.0% | ACGC_LOSS |
| `deep_history_recall_1` / `p3` | `acgc` | probe | 5.0 / 5.0 | 6394 / 4625 | 27.7% | 0.78 / 1.08 | +38.2% | ACGC_WIN |
| `deep_history_recall_1` / `p3` | `sliding_window` | probe | 5.0 / 0.0 | 6394 / 6379 | 0.2% | 0.78 / 0.00 | -100.0% | ACGC_LOSS |
| `deep_history_recall_1` / `p4` | `acgc` | probe | 5.0 / 5.0 | 6391 / 4593 | 28.1% | 0.78 / 1.09 | +39.1% | ACGC_WIN |
| `deep_history_recall_1` / `p4` | `sliding_window` | probe | 5.0 / 0.0 | 6391 / 6376 | 0.2% | 0.78 / 0.00 | -100.0% | ACGC_LOSS |
| `long_range_recall_1` / `p1` | `acgc` | probe | 5.0 / 5.0 | 1093 / 978 | 10.5% | 4.57 / 5.11 | +11.8% | ACGC_WIN |
| `long_range_recall_1` / `p1` | `sliding_window` | probe | 5.0 / 5.0 | 1093 / 1093 | 0.0% | 4.57 / 4.57 | +0.0% | TIE |
| `long_range_recall_1` / `p2` | `acgc` | probe | 5.0 / 5.0 | 1089 / 974 | 10.6% | 4.59 / 5.13 | +11.8% | ACGC_WIN |
| `long_range_recall_1` / `p2` | `sliding_window` | probe | 5.0 / 5.0 | 1089 / 1089 | 0.0% | 4.59 / 4.59 | +0.0% | TIE |
| `long_range_recall_1` / `p3` | `acgc` | probe | 5.0 / 5.0 | 1093 / 978 | 10.5% | 4.57 / 5.11 | +11.8% | ACGC_WIN |
| `long_range_recall_1` / `p3` | `sliding_window` | probe | 5.0 / 5.0 | 1093 / 1093 | 0.0% | 4.57 / 4.57 | +0.0% | TIE |
| `multi_hop_synth_1` / `p1` | `acgc` | judge | 5.0 / 5.0 | 1111 / 1000 | 10.0% | 4.50 / 5.00 | +11.1% | ACGC_WIN |
| `multi_hop_synth_1` / `p1` | `sliding_window` | judge | 5.0 / 5.0 | 1111 / 1111 | 0.0% | 4.50 / 4.50 | +0.0% | TIE |
| `recent_recall_1` / `p1` | `acgc` | probe | 5.0 / 5.0 | 307 / 298 | 2.9% | 16.29 / 16.78 | +3.0% | ACGC_WIN |
| `recent_recall_1` / `p1` | `sliding_window` | probe | 5.0 / 5.0 | 307 / 307 | 0.0% | 16.29 / 16.29 | +0.0% | TIE |
| `topic_switch_return_1` / `p1` | `acgc` | probe | 5.0 / 5.0 | 787 / 724 | 8.0% | 6.35 / 6.91 | +8.7% | ACGC_WIN |
| `topic_switch_return_1` / `p1` | `sliding_window` | probe | 5.0 / 5.0 | 787 / 787 | 0.0% | 6.35 / 6.35 | +0.0% | TIE |

## Response samples

### `constraint_adherence_1::p1`

**Question:** Now write me a simple in-memory cache implementation for one of my services. Pick whatever language and supporting tools you think are best for this project — full freedom.

**naive_full_history (ref)** (842 prompt tokens, 48359 ms):

> You asked for full freedom, but per your earlier constraints I’ll implement this in Go (no Python) and avoid Redis. This is a simple, thread-safe, in-memory cache with optional per-item TTL and a background janitor to purge expired entries.
> 
> File: internal/cache/cache.go
> package cache
> 
> import (
> 	"sync"
> 	"time"
> )
> 
> // Cache is a generic, in-memory, concurrent-safe cache with optional TTL per entry.
> // Zero or negative TTL means "no expiration".
> type Cache[K comparable, V any] struct {
> 	mu            sync.RWMutex
> 	data          map[K]entry[V]
> 	defaultTTL    time.Duration
> 	purgeInterval time.Duration
> 
> 	stopCh chan struct{}
> 	wg     sync.WaitGroup
> }
> 
> type entry[V any] struct {
> 	v   V
> 	exp time.Time // zero time means no expiration
> }
> 
> // New creates a new Cache.
> // defaultTTL: if > 0, used when Set is called with ttl <= 0.
> // purgeInterval: how often expired entries are removed in the background; if <= 0, no janitor runs.
> func New[K comparable, V any](defaultTTL, purgeInterval time.Duration) *Cache[K, V] {
> 	c := &Cache[K, V]{
> 		data:          make(map[K]entry[V]),
> 		defaultTTL:    defaultTTL,
> 		purgeInterval: purgeInterval,
> 		stopCh:        make(chan struct{}),
> 	}
> 	if purgeInterval > 0 ...[truncated]

**sliding_window** (842 prompt tokens, 60135 ms):

> Given your constraints, here’s a small, thread-safe, in-memory TTL cache in Go (no external services, no Redis). It supports per-entry or default TTL, periodic cleanup, and graceful shutdown.
> 
> File: internal/cache/cache.go
> ```go
> package cache
> 
> import (
> 	"sync"
> 	"time"
> )
> 
> // Cache is a generic, thread-safe in-memory TTL cache.
> // - Keys must be comparable.
> // - Values can be any type.
> // - defaultTTL <= 0 means entries do not expire by default.
> // - cleanupInterval <= 0 disables the background janitor (expired items are removed on Get/Set).
> type Cache[K comparable, V any] struct {
> 	mu              sync.RWMutex
> 	items           map[K]item[V]
> 	defaultTTL      time.Duration
> 	cleanupInterval time.Duration
> 
> 	stopCh    chan struct{}
> 	stopped   chan struct{}
> 	closeOnce sync.Once
> }
> 
> type item[V any] struct {
> 	value     V
> 	expiresAt time.Time // zero time means no expiration
> }
> 
> // New creates a new cache with a default TTL and optional cleanup interval.
> // If cleanupInterval > 0, a goroutine periodically removes expired items.
> // Call Close() to stop the janitor when your service shuts down.
> func New[K comparable, V any](defaultTTL, cleanupInterval time.Duration) *Cache[K, V] {
> 	c := &Cach...[truncated]

**acgc** (761 prompt tokens, 40785 ms):

> Below is a small, production-friendly in-memory cache in Go. It’s thread-safe, supports TTL per entry, optional max capacity with LRU eviction, and an optional background janitor for removing expired items. No external dependencies.
> 
> File: cache.go
> 
> package cache
> 
> import (
> 	"container/list"
> 	"sync"
> 	"time"
> )
> 
> type CacheOptions struct {
> 	// Capacity is the max number of items to keep.
> 	// 0 or negative means unbounded.
> 	Capacity int
> 
> 	// DefaultTTL applies when Set is called without an explicit TTL.
> 	// Zero means entries do not expire by default.
> 	DefaultTTL time.Duration
> 
> 	// CleanupInterval runs a background janitor that removes expired items.
> 	// Zero disables the janitor.
> 	CleanupInterval time.Duration
> }
> 
> type entry struct {
> 	key   string
> 	value any
> 	exp   time.Time // zero time means no expiration
> }
> 
> type Cache struct {
> 	mu   sync.Mutex
> 	ll   *list.List                   // most-recently-used at front
> 	data map[string]*list.Element     // key -> *list.Element where Element.Value is *entry
> 
> 	opts     CacheOptions
> 	stopCh   chan struct{}
> 	stopped  bool
> 	wg       sync.WaitGroup
> 	nowFunc  func() time.Time // for testing; defaults to time.Now
> }
> 
> // New creates a new Cache with th...[truncated]

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

### `deep_history_recall_1::p1`

**Question:** Way back at kickoff — which primary datastore did we commit to for this platform?

**naive_full_history (ref)** (6395 prompt tokens, 3100 ms):

> CockroachDB.

**sliding_window** (6380 prompt tokens, 29106 ms):

> I don’t see that decision captured in this thread, so I can’t say for sure. It should be in the kickoff ADR or infra code. Quick ways to confirm:
> 
> - Check docs: docs/adr/0001-primary-datastore.md (or kickoff notes/Confluence “Architecture — Datastore”).
> - Infra repo: look for aws_rds_cluster/aws_db_instance (Postgres/MySQL), aws_dynamodb_table, google_sql_database, etc.
> - App config: env/Helm values for DATABASE_URL, JDBC URLs, or PRIMARY_DATASTORE.
> - Code search:
>   - rg -nEi 'postgres|jdbc:postgresql|psycopg2|pgx|mysql|jdbc:mysql|dynamodb|MongoClient|spanner|cockroach' .
> 
> If you can share the ADR title or a snippet of infra/config, I’ll confirm and document it in the runbook.

**acgc** (4578 prompt tokens, 3997 ms):

> CockroachDB — for distributed SQL with strong consistency across regions.

---

### `deep_history_recall_1::p2`

**Question:** And which message broker did we choose for order events?

**naive_full_history (ref)** (6390 prompt tokens, 4327 ms):

> RabbitMQ — we chose it over Kafka for per-message routing and simpler ops at our scale.

**sliding_window** (6375 prompt tokens, 23716 ms):

> I don’t see that decision in the provided context. It should be documented in your repo (ADR/decisions), IaC (Terraform/Helm), or service config.
> 
> Quick ways to confirm:
> - Search code/config for “kafka”, “rabbit”, “sns/sqs”, “nats”, or env vars like ORDER_EVENTS_BROKER/KAFKA_BROKERS.
> - Check Terraform for aws_msk_cluster, aws_sqs_queue/sns_topic, or GCP Pub/Sub resources.
> - Look at the order-service Helm values or runtime env in prod/staging.
> 
> If you share the repo/path or environment, I can pinpoint it.

**acgc** (4592 prompt tokens, 4373 ms):

> RabbitMQ — chosen over Kafka for modest throughput, per-message routing, and simpler ops.

---

### `deep_history_recall_1::p3`

**Question:** Remind me which cloud provider and region we settled on, and why.

**naive_full_history (ref)** (6394 prompt tokens, 6846 ms):

> Google Cloud (GCP), europe-west4. We chose it to keep all customer data in the EU for residency compliance, with the bonus of GCP’s managed networking/IAM and low latency for EU users.

**sliding_window** (6379 prompt tokens, 27731 ms):

> I don’t have a record of the chosen cloud provider/region in this thread, so I can’t state it confidently.
> 
> Quick places to check:
> - ADRs (e.g., “Cloud platform and region”)
> - infra/terraform: provider blocks and region variables
> - CI/CD env vars (e.g., AWS_REGION/GCP_PROJECT/AZURE_LOCATION)
> - Container registry hosts (ECR vs GCR vs ACR) and bucket names
> - Kubernetes context/cluster naming conventions
> 
> If you want a recommendation rather than the prior decision, say so and I’ll propose a provider/region pair with rationale (latency to our DCs/carriers, service availability, egress/cost, data residency, and DR pairing).

**acgc** (4625 prompt tokens, 6252 ms):

> Google Cloud (GCP), region europe-west4. We chose it to meet EU customer data‑residency requirements by keeping data within the EU and simplifying compliance.

---

### `deep_history_recall_1::p4`

**Question:** What barcode standard did we standardize on for physical labels?

**naive_full_history (ref)** (6391 prompt tokens, 3088 ms):

> GS1 DataMatrix (2D).

**sliding_window** (6376 prompt tokens, 13705 ms):

> I don’t see that detail in the shared context. Likely places to confirm:
> - Label-print service repo: README/config (e.g., BARCODE_STANDARD/BARCODE_SYMBOLOGY) and ZPL/EPL templates
> - ADRs/architecture-decisions (look for “labeling” or “barcode”)
> - Warehouse ops/SOP docs or printer profiles
> 
> Tip: if you use ZPL, search templates for:
> - ^BC → Code 128 (often GS1-128 when AIs are used)
> - ^B3 → Code 39
> - ^BQ → QR
> - ^BX → Data Matrix
> 
> If it’s not decided yet, my default recommendation:
> - GS1-128 (Code 128 with Application Identifiers) for shipping/parcel labels (broad carrier compatibility, good density)
> - Data Matrix for small item labels (very compact, strong error correction)
> 
> Want me to check a specific repo/path or make a short ADR with the rationale?

**acgc** (4593 prompt tokens, 2667 ms):

> GS1 DataMatrix (2D) barcode.

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

**naive_full_history (ref)** (1111 prompt tokens, 91785 ms):

> Short answer: $4k/month is only feasible if you avoid paying internet egress, rely heavily on spot GPUs, keep storage retention short, and accept queuing (not real‑time turnaround at peak). If you must serve video to the internet from your cloud, the budget will be blown by egress alone.
> 
> Assumptions for a realistic baseline
> - Cloud: AWS us-east-1 pricing (similar magnitude on GCP/Azure).
> - GPU: g5.xlarge (A10G) on spot at ~$0.35/hr effective blended rate.
> - Encoding: 4K ABR ladder (2160p + downscales via NVENC). Approx 1 GPU-hour per input hour for H.264/HEVC at 30 fps.
> - Average upload: 2 GB ≈ ~20 minutes at ~15 Mbps (0.33 hours). If your content is longer/denser or 60 fps, adjust accordingly.
> - Retention: keep sources and outputs hot for 7 days, then delete/archive. Raw deleted after successful transcode.
> - Delivery: no public internet egress; deliver to customer-owned bucket in the same region (or they pull via private link/CDN they pay for).
> 
> Compute math
> - Capacity plan: 8× g5.xlarge spot.
>   - Cost: 8 GPUs × $0.35/hr × 24 × 30 ≈ $2,016/month.
>   - Throughput: 8 GPU-hrs per wall-clock hour → 8 × 24 × 30 ≈ 5,760 input hours/month.
>   - With 20‑minute average job...[truncated]

**sliding_window** (1111 prompt tokens, 76556 ms):

> Short answer: it can be enough, but only for a modest workload. With 4K support and a $4k all‑in cap, you’ll be constrained mainly by egress and (secondarily) GPU time. Here’s the back‑of‑the‑envelope math you can tune with your actual volumes.
> 
> Assumptions (reasonable defaults you can swap):
> - Storage (S3‑like): $0.023/GB‑month; request costs negligible vs data.
> - Egress to internet: $0.09/GB (first 10–50 TB tier).
> - GPU for 4K (NVENC on T4/A10G/L4): blended $0.30–$0.60/hr using spot + occasional on‑demand.
> - 4K bitrate: ~20 Mb/s → ~9 GB/hour of video.
> - Your average file is 2 GB → ~13 minutes of video.
> - Retention: originals 7 days, outputs 30 days. Single 4K output per job (no ladder).
> 
> Per‑job cost (2 GB output):
> - Compute: 13/60 hr × $0.30–$0.60 ≈ $0.065–$0.13
> - Storage: originals (7d) + outputs (30d) = 2 GB × (1 + 7/30) × $0.023 ≈ $0.051
> - Egress: 2 GB × $0.09 = $0.18
> - Total per job ≈ $0.30–$0.36 (excludes small control‑plane costs)
> 
> What the $4k/month buys under those assumptions:
> - Jobs/month: $4,000 ÷ ~$0.30–$0.36 ≈ 11k–13k completed jobs with one download each.
> - Or, think in data terms:
>   - Egress is the ceiling: $4,0...[truncated]

**acgc** (1000 prompt tokens, 70920 ms):

> Short answer: it can be enough, but only at relatively low monthly volume. With 4K required and 2GB average inputs, the main cost drivers will be egress and GPU hours. Past roughly 6k outputs/month, $4000/mo is unlikely to hold without shifting costs to the customer.
> 
> Assumptions (industry-typical, adjust if you have better numbers)
> - Avg input size: 2 GB → implies ~22–33 minutes source length at 8–12 Mbps (assume ~25 min average).
> - Output: 4K HEVC at 25–35 Mbps (sports). For 25 min: 4.7–6.6 GB per output.
> - GPU: NVENC on a single A10/T4-class GPU can do ~1x realtime for 4K30 single-pass; 4K60 or higher quality/two-pass can be ~2x slower.
> - GPU pricing (spot, aggressive use): ~$0.40/GPU-hour (varies by region/provider). On-demand is ~2–3x higher; try to avoid it.
> - Storage: object storage ~$0.023/GB-month. Assume 7-day retention for inputs+outputs.
> - Egress: $0.07–$0.09/GB to internet (CDN + cloud egress blended).
> 
> Per-job cost (variable)
> - GPU compute:
>   - 25 min at 1x realtime: 0.417 GPU-hours × $0.40 ≈ $0.17
>   - If 60 fps or higher quality/two-pass (~2x): 0.83 GPU-hours ≈ $0.33
>   - Practical range: $0.17–$0.70 depending on preset/fps/spot availability
> - Sto...[truncated]

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

