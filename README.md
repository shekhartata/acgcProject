# ACGC — Agent Context Garbage Collector

A Go sidecar runtime that sits between an AI agent and its LLM, intercepting every interaction to build a structured context model. It scores relevance (**heuristic + optional semantic / embeddings**), prunes stale information, compresses resolved branches, and compiles only the most useful context into each LLM call — optionally **pulling archived nodes back into the prompt** via dual **HNSW** indexes — reducing token costs, lowering latency, and improving recall on long-range and topic-switching sessions.

**How it plugs in** — your app stops sending the full conversation history to the LLM; ACGC takes that over:

```
┌──────────────┐   gRPC (Run)    ┌──────────────┐   HTTP    ┌──────────┐
│  Your Agent  │ ──────────────► │  ACGC Server │ ────────► │  LLM API │
│  Application │ ◄────────────── │  (sidecar)   │ ◄──────── │          │
└──────────────┘  response+stats └──────┬───────┘           └──────────┘
                                        │ async persistence
                                    ┌───▼──────┐
                                    │ MongoDB  │
                                    └──────────┘
```

---

## Table of Contents

- [Why ACGC (the problem)](#why-acgc-the-problem)
- [Quickstart — pick your path](#quickstart--pick-your-path)
- [Integrating ACGC into your application](#integrating-acgc-into-your-application)
  - [The sidecar model](#the-sidecar-model)
  - [Go SDK (multi-turn)](#go-sdk-multi-turn)
  - [Sessions, tasks, and the LLM key](#sessions-tasks-and-the-llm-key)
  - [Capturing tool results and other events](#capturing-tool-results-and-other-events)
  - [Other languages (gRPC)](#other-languages-grpc)
  - [Interactive test client](#interactive-test-client)
- [How It Works](#how-it-works)
  - [End-to-End Flow](#end-to-end-flow)
  - [State Tree](#state-tree)
  - [Relevance Scoring](#relevance-scoring)
  - [Semantic embeddings and HNSW (optional)](#semantic-embeddings-and-hnsw-optional)
  - [Facts and verbatim decisions](#facts-and-verbatim-decisions)
  - [Garbage Collection](#garbage-collection)
  - [Prompt Compilation](#prompt-compilation)
  - [Context Rehydration](#context-rehydration)
- [Architecture](#architecture)
  - [Dual-Path Design](#dual-path-design)
  - [Concurrency Model](#concurrency-model)
  - [Storage Layer](#storage-layer)
- [Configuration reference](#configuration-reference)
- [Benchmarks and Evaluation](#benchmarks-and-evaluation)
  - [Stress test suite (no API key)](#stress-test-suite-no-api-key)
  - [Latency benchmarking (`acgc-latencybench`)](#latency-benchmarking-acgc-latencybench)
  - [Quality evaluation (LLM harness)](#quality-evaluation-llm-harness)
  - [External benchmark evaluation](#external-benchmark-evaluation)
- [Project Structure](#project-structure)
- [Current Status vs Roadmap](#current-status-vs-roadmap)

---

## Why ACGC (the problem)

LLM-powered agents accumulate context over long conversations — user messages, assistant responses, tool calls, tool results, errors, retries. Without management, this context grows unbounded:

- **Token bloat**: Sending the entire conversation history gets expensive fast. A 60-turn session can easily hit 10,000+ tokens per call.
- **Noise**: Old resolved issues, failed attempts, and redundant information dilute the signal. The LLM starts contradicting earlier decisions or forgetting constraints.
- **Latency**: More input tokens = slower responses, especially with reasoning models (GPT-5, o3) that scale processing time with input size.

ACGC solves this by treating context like memory in a running program — actively managing what stays, what gets compressed, and what gets archived.

---

## Quickstart — pick your path

Different goals need different amounts of setup. Pick the row that matches yours:

| I want to… | Path | Needs MongoDB? | Needs LLM API key? | Time |
|---|---|---|---|---|
| Sanity-check the pipeline and see token savings | [1. Smoke test](#path-1--smoke-test-no-keys-no-database) | No | No | ~2 min |
| Chat with it live and watch GC/token stats per turn | [2. Live demo](#path-2--live-demo-server--test-client) | Yes | Yes | ~5 min |
| Wire it into my own agent | [3. Integrate](#path-3--integrate-into-your-app) | Yes | Yes | ~10 min |
| See quality/latency numbers vs naive baselines | [4. Benchmark](#path-4--benchmark-it) | Depends | Mostly yes | varies |

**Prerequisites:** Go 1.25+. MongoDB only for paths 2–3 (local via `docker compose up -d mongodb`, or Atlas). `protoc` + Go gRPC plugins only if you regenerate protos.

### Path 1 — Smoke test (no keys, no database)

Runs the full ACGC pipeline in-process (tree → scorer → GC → compiler) against synthetic conversations:

```bash
go run ./stresstest/
# Optional: exercise the semantic path with a mock embedder (still free)
go run ./stresstest/ -semantic
```

You'll see per-session token reduction (e.g. ~74% on the 66-turn `long_session` fixture) and coherency checks. Details: [Stress test suite](#stress-test-suite-no-api-key).

### Path 2 — Live demo (server + test client)

```bash
# 1. Configure: only two variables are required for this path
cp .env.example .env
#    Edit .env → set ACGC_MONGO_URI and ACGC_LLM_API_KEY

# 2. Start MongoDB (skip if using Atlas)
docker compose up -d mongodb

# 3. Build and start the ACGC server
make build && make run

# 4. In another terminal: interactive REPL through a real LLM
make testcli
```

Type messages and watch token savings, GC triggers, and compiled-prompt stats after each turn (`/state`, `/metrics`, `/gc`, `/quit`).

### Path 3 — Integrate into your app

**Prerequisites:** Path 2 steps 1–3 (MongoDB + server running on `:50051`).

**Integration in 3 steps:**

1. **One session per conversation** — pick a stable `SessionID` (e.g. `user-42-chat-2026-07-04`) and reuse it on every turn.
2. **Replace your LLM call** — send only the new user message; ACGC compiles context, calls the LLM, and returns the answer plus token stats.
3. **Optional: feed tool output** — `CaptureEvent(ctx, "tool_result", output, metadata)` so GC can prune stale tool noise.

```go
runtime, _ := acgc.NewContextRuntime(acgc.Config{
    ServerAddr:  "localhost:50051",
    SessionID:   "user-42-chat-2026-07-04", // reuse every turn
    TokenBudget: 6000,
})
result, _ := runtime.Run(ctx, userMessage) // replaces llmClient.Chat(fullHistory)
fmt.Println(result.Response)
```

**Before → after:** you stop sending full history on every call; ACGC owns memory, GC, compilation, and persistence.

Full guide: [Integrating ACGC into your application](#integrating-acgc-into-your-application).

### Path 4 — Benchmark it

Four harnesses answer four different questions — see [Benchmarks and Evaluation](#benchmarks-and-evaluation) for the orientation table, commands, and recorded numbers. You do **not** need to run any of them to use ACGC; they exist to justify it.

---

## Integrating ACGC into your application

| You send | ACGC handles |
|---|---|
| New user message (`Run`) | State tree, scoring, GC, compile, LLM call |
| Tool results / errors (`CaptureEvent`) | Classify, score, eventually archive |
| Stable `SessionID` per conversation | Persistence + rehydration after restart |

**Go** → use the SDK in `pkg/acgc` (example below). **Other languages** → call the gRPC `Run` RPC directly ([Other languages](#other-languages-grpc)).

### The sidecar model

ACGC is a separate process (default `localhost:50051`) that **owns the LLM call**. The integration change in your agent is:

**Before** — your app manages history and calls the LLM directly:

```go
history = append(history, userMsg)
resp := llmClient.Chat(ctx, history)   // full history every call, grows unbounded
```

**After** — your app sends only the new message to ACGC; ACGC maintains the context model, compiles a budgeted prompt, calls the LLM, and returns the response plus savings stats:

```go
result, _ := runtime.Run(ctx, userMsg) // ACGC handles history, compilation, and the LLM call
```

There is nothing else to keep in sync: ACGC persists raw events to MongoDB and rebuilds its in-memory state on restart ([Context Rehydration](#context-rehydration)).

### Go SDK (multi-turn)

A realistic chat loop — one runtime per conversation, `Run` on every user turn:

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"

    "github.com/chandrashekhartata/acgc/pkg/acgc"
)

func main() {
    runtime, err := acgc.NewContextRuntime(acgc.Config{
        ServerAddr:  "localhost:50051",
        SessionID:   "user-42-chat-2026-07-02", // one per conversation; reuse across turns
        TaskID:      "schema-design",           // logical task within the session
        TokenBudget: 6000,                      // compiled prompt budget
        Policy:      acgc.PolicyBalanced,       // or PolicyAggressive / PolicyConservative
        LLM: acgc.LLMConfig{                    // optional — omit to use the server's ACGC_LLM_* config
            Provider: "openai",
            Model:    "gpt-5",
            APIKey:   os.Getenv("OPENAI_API_KEY"),
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer runtime.Close()

    scanner := bufio.NewScanner(os.Stdin)
    for fmt.Print("you> "); scanner.Scan(); fmt.Print("you> ") {
        result, err := runtime.Run(context.Background(), scanner.Text())
        if err != nil {
            log.Printf("run failed: %v", err)
            continue
        }
        fmt.Println(result.Response)
        fmt.Printf("  [tokens: %d → %d, saved %.1f%%; GC: %v]\n",
            result.OriginalTokens, result.CompiledTokens,
            result.ReductionPercent, result.GCTriggered)
    }
}
```

Every `Run` returns the LLM response plus `OriginalTokens`, `CompiledTokens`, `TokensSaved`, `ReductionPercent`, and whether GC fired — so you can log savings per turn in production.

### Sessions, tasks, and the LLM key

- **`SessionID`** scopes the context model: one state tree, one worker goroutine, one pair of HNSW indexes per session. Use **one session per conversation** (per user chat, per agent job). Reusing the ID across turns is what gives ACGC its memory; a fresh ID starts a blank context. Idle sessions are cleaned up after `ACGC_SESSION_IDLE_TIMEOUT` (default 30 min) and can be rehydrated from MongoDB.
- **`TaskID`** labels a logical task inside a session (used for grouping/metrics); `"default"` is fine.
- **LLM config is optional per client.** If the SDK's `LLM.APIKey` is empty, the server uses its own `ACGC_LLM_*` env config. Pass per-request config when different callers need different models/keys; use server config when you want the key in exactly one place.

### Capturing tool results and other events

`Run` covers user turns. For agent loops that produce **tool calls, tool results, or errors**, feed them into the context model with `CaptureEvent` so the scorer and GC can see (and later prune) them:

```go
// After executing a tool in your agent loop:
eventID, err := runtime.CaptureEvent(ctx, "tool_result", toolOutput, map[string]string{
    "tool": "run_tests",
})
```

Supported event types (from `internal/domain`): `user_prompt`, `agent_prompt`, `llm_response`, `tool_call`, `tool_result`, `error`, `retry`, `system`. Tool results are classified as low-retention nodes — they get swept quickly once resolved, which is usually exactly what you want.

The SDK also exposes `TriggerGC(ctx)` (force a collection) and `Metrics(ctx)` (session-level savings and GC stats).

### Other languages (gRPC)

Any language can call ACGC via gRPC. The service definition is in `proto/acgc.proto`:

| RPC | Description |
|---|---|
| `Run` | Full intercept → compile → forward → capture cycle |
| `CaptureEvent` | Manually capture a single event into the state tree |
| `GetState` | Inspect the current state tree and node scores |
| `TriggerGC` | Manually trigger garbage collection |
| `GetMetrics` | Get token savings, GC stats, and session metrics |

Generate client stubs:

```bash
# Go (already generated)
make proto

# Python
pip install grpcio-tools
python -m grpc_tools.protoc -I proto --python_out=. --grpc_python_out=. proto/acgc.proto

# TypeScript
npx grpc_tools_node_protoc --ts_out=. --grpc_out=. -I proto proto/acgc.proto
```

The integration contract is the same as the SDK: call `Run(session_id, task_id, user_message, token_budget, policy, llm_config)` per user turn, keep `session_id` stable across the conversation.

### Interactive test client

The `testcli` binary provides a REPL for manually testing ACGC with a real LLM (server must be running):

```bash
make testcli
```

Type messages and see real-time token savings, GC triggers, and compiled prompt stats after each turn.

---

## How It Works

### End-to-End Flow

```
User sends message
       │
       ▼
┌─────────────────────────────────┐
│  1. INTERCEPT                   │
│  ACGC Gateway receives the      │
│  request via gRPC               │
└──────────┬──────────────────────┘
           │
     ┌─────┴─────┐
     │           │
     ▼           ▼
┌─────────┐ ┌───────────────────────────────────┐
│ FAST    │ │ ASYNC PATH (background goroutine) │
│ PATH    │ │                                   │
│ (sync)  │ │ 2. CAPTURE: persist raw event     │
│         │ │    to MongoDB archive              │
│         │ │                                   │
│         │ │ 3. CLASSIFY: determine node type   │
│         │ │    (goal/constraint/decision/      │
│         │ │     tool_result/issue/background)  │
│         │ │                                   │
│         │ │ 4. BUILD: insert node into the     │
│         │ │    in-memory state tree             │
│         │ │                                   │
│         │ │ 5. SCORE: compute retention scores │
│         │ │    for all active nodes             │
│         │ │                                   │
│         │ │ 6. GC CHECK: if token pressure,    │
│         │ │    tree depth, or staleness         │
│         │ │    thresholds are exceeded →        │
│         │ │    run mark-sweep-compact           │
│         │ │                                   │
│ 7. READ │ │                                   │
│  state  │ │                                   │
│  tree   │ └───────────────────────────────────┘
│         │
│ 8. COMPILE: select highest-scored
│    nodes within token budget
│
│ 9. ASSEMBLE messages for LLM:
│    system (once) → user(compiled context) → user(current query)
│
│ 10. FORWARD to LLM
│
│ 11. RETURN response + stats
│     (original tokens, compiled tokens,
│      savings %, GC info)
└─────────┘
```

**Optional semantic path (`ACGC_SEMANTIC_ENABLED=true`):** after each node is created, the worker can **compute an embedding** (OpenAI-compatible; default `text-embedding-3-small`) and insert it into a per-session **Active** HNSW graph. The **last user message** embedding anchors **cosine similarity** inside the retention scorer. On **GC sweep**, vectors for nodes that leave the active set move to a second **Archive** HNSW so they stay retrievable. On **compile**, the imminent user message is embedded again; top‑K hits from **Active ∪ Archive** are merged, **archived hits** are included in the compilation node set, and the compiler **re-blends** scores with a configurable **semantic weight** (see [Semantic embeddings and HNSW (optional)](#semantic-embeddings-and-hnsw-optional)).

### State Tree

Every interaction is classified into a typed node and inserted into a tree structure:

| Node Type | What It Represents | GC Protection |
|---|---|---|
| `goal` | What the user is trying to achieve | High — goals survive longest |
| `constraint` | Rules the solution must follow ("must use Go", "no Redis") | High — almost never swept |
| `decision` | A choice made during the conversation | Medium |
| `attempt` | A tool call or action taken | Low — swept after resolution |
| `tool_result` | Output from a tool/command | Low |
| `issue` | An error, bug, or unresolved problem | Medium — boosted while unresolved |
| `summary` | A summarized block of older context | Medium |
| `compressed_branch` | Multiple old nodes compressed into one | Medium |
| `background` | Miscellaneous context | Lowest |

Nodes track: parent-child relationships, raw event references (for rehydration), token counts, creation/update times, dependency links to other nodes, optional **`Facts` / `Decisions`** string slices (see [Facts and verbatim decisions](#facts-and-verbatim-decisions)), and optional **`Embedding`** (+ model id) when semantic mode is on.

**Node lifecycle:** `active` → `stale` → `archived` (or `active` → `compressed` when a branch is compacted).

### Relevance Scoring

Every active node receives a **retention score** (0.0 to 1.0) from a **weighted sum of signals** (see `internal/scorer`). Default weights match the table below.

| Signal | Default weight | What it measures |
|---|---|---|
| Recency | 0.25 | Exponential decay by turn distance; recent nodes score higher. |
| Type priority | 0.20 | Intrinsic importance by node type (goals highest, background lowest). |
| Dependency weight | 0.15 | Boost if other active nodes depend on this node. |
| Unresolved boost | 0.15 | Boost while the node carries open issues. |
| Semantic | **0.20** when enabled | Cosine similarity between the node's **embedding** and the anchor query vector (typically the **latest user message** embedding). Zero if semantic mode is off or vectors are missing. |
| Redundancy penalty | −0.10 | Duplicate titles / redundant nodes. |
| Resolved penalty | −0.20 | Resolved nodes penalized toward sweep. |
| Stale penalty | −0.15 | Grows when age exceeds `ACGC_STALE_TURNS`. |
| Size penalty | −0.05 | Oversized payloads vs `MaxTokensPerNode`. |

**Combined shape (conceptual):**

```
RetentionScore = clamp(
    w_recency×Recency + w_type×TypePriority + w_dep×Dependency + w_open×Unresolved
  + w_sem×Semantic
  − w_red×Redundancy − w_res×Resolved − w_stale×Stale − w_size×SizePenalty,
  0.0, 1.0)
```

With **`ACGC_SEMANTIC_ENABLED=false`**, no embedder is constructed, **`Semantic` stays 0**, and scoring is heuristic-only (fast, predictable, no embedding cost). With semantic mode **on**, the worker embeds node payloads **best-effort** and uses the cached **last user** embedding as the scorer anchor so freshness of the user intent shapes retention before compile-time retrieval runs again.

---

### Semantic embeddings and HNSW (optional)

When **`ACGC_SEMANTIC_ENABLED=true`** and embed credentials are available:

1. **Per-node embeddings** — After a node is written, the session worker derives text from title/summary/payload (`internal/session`), calls the embedder (**OpenAI-compatible** REST; defaults in `internal/config`), and stores **`Embedding`** on the node plus MongoDB (`internal/store`).
2. **Dual in-memory graphs** — Each session maintains two **`coder/hnsw`** graphs wrapped by `internal/vectorindex`: **`ActiveIndex`** (live active-set vectors) and **`ArchiveIndex`** (vectors for nodes that were **swept** off the active list). On GC, embeddings for removed active IDs are **inserted into Archive** and **deleted from Active** so archived content stays searchable (`reconcileSemanticIndices` / eval & stress mirrors).
3. **Compile-time retrieval** — `CompilePrompt` embeds the **current user message**, queries Active **top‑K** and Archive **top‑K** separately, **`MergeSemanticHits`** keeps the best score per node id, then **`NodesForSemanticCompile`** unions **active nodes** plus **matching archived** nodes. **`CompileWithSemantic`** adjusts the effective sort/budget ranking using **`semantic_weight × (hitScore − node's prior semantic contribution)`** so the imminent query boosts relevant ghosts without rewriting stored scores wholesale.
4. **Cold start** — Rehydrating a session rebuilds both graphs from Mongo via **`LoadNodeEmbeddings`** (active) and **`LoadArchivedNodeEmbeddings`** (archived-only).

**Defaults (env):** `ACGC_SEMANTIC_WEIGHT=0.20`, `ACGC_HNSW_TOP_K_AT_COMPILE=12`, `ACGC_ARCHIVE_SEMANTIC_TOP_K=12`, `ACGC_HNSW_M=16`, `ACGC_HNSW_EF_SEARCH=50`, `ACGC_EMBED_MODEL=text-embedding-3-small`, `ACGC_EMBED_DIM=1536`. See [Configuration reference](#configuration-reference).

The **stress** harness can exercise this path **without billing** embeddings using **`-semantic`** + a deterministic **`MockEmbedder`** (`make stresstest-semantic`).

---

### Facts and verbatim decisions

**`internal/facts`** performs **deterministic** extraction from user prompts and assistant replies (patterns + small lexicon): important tokens and short **decisions** land in **`node.Facts`** / **`node.Decisions`**. Compression paths merge children so **verbatim snippets** survive into **`compressed_branch`** summaries (LLM branch also asks for trailing **`ENTITIES:`** merged back into **`Facts`**).

**GC hybrid policy:** Nodes with **any** Facts or Decisions are **never swept solely for low relevance** (hard defer). Bare **`NodeDecision`** nodes use **`DecisionSweepFloor`**: sweep compares **`max(raw_score, floor)`** against **`LowRelevanceThreshold`** — the floor **must stay below** the relevance threshold so filler assistant turns remain reclaimable (`internal/gc`).

**Prompt rendering:** **`internal/compiler`** adds indented **`facts:`** / **`decisions:`** lines **only for** `compressed_branch` / `summary` nodes, where summaries might omit raw wording; ordinary active nodes expose content in their bullet **`Summary`** so we avoid duplicate token overhead.

---

### Garbage Collection

GC uses a **mark-sweep-compact** cycle whenever **any** automatic trigger fires (or `TriggerGC` supplies **manual**). **Estimated active-node token sum** feeds the pressure / headroom checks.

| Trigger | Default | What it means |
|---|---|---|
| **Token pressure** | `ACGC_TOKEN_BUDGET` (~6000) | Sum of active node **`TokenCount`** exceeds the compiled prompt budget. |
| **Soft headroom** | `ACGC_GC_SWEEP_HEADROOM_RATIO` (**0.60**) × budget | Estimated active tokens **>** 3600 — early GC so dense short-turn sessions compact before saturation. (**0** disables.) |
| **Too many active nodes** | **`ACGC_GC_MAX_ACTIVE_NODES` (25)** | Active node cardinality exceeded — complements token triggers on many small utterances. (**0** disables.) |
| **Tree depth** | `10` | Max lineage depth exceeds limit. |
| **Tree width** | `50` | A parent has too many children. |
| **Low average relevance** | `0.30` | Mean retention score across actives below threshold. |
| **Resolved branch** | (any) | At least one resolved node — opportunistic cleanup. |
| **Manual** | — | `TriggerGC` RPC. |

**GC phases:**

1. **Mark / score** — Re-score all active nodes (including **semantic** term when enabled). Candidates for sweep have effective score **below** `LowRelevanceThreshold` after policy tweaks.
2. **Sweep** — Low-scoring nodes become **`archived`**; raw events stay in MongoDB. **Never swept** on relevance alone if **`len(Facts) > 0` or `len(Decisions) > 0`**. Bare **`NodeDecision`** uses **`DecisionSweepFloor`**: sweep score = **`max(raw, floor)`** — the floor must remain **below** **`LowRelevanceThreshold`** so generic assistant chatter stays reclaimable (default floor **0.20** vs threshold **0.30**).
3. **Compact** — Wide or resolved branches compress into **`compressed_branch`** nodes via **SimpleCompressor** or **LLMCompressor** (`internal/gc`, `internal/llm`).

After sweep, **`internal/session`** reconciles semantic indexes: embeddings for archived IDs migrate **Active → Archive** HNSW.

**Natural protection:** Goals, constraints, factual/decision-bearing nodes (hard deferral), dependents, unresolved issues remain in the competition set unless policy explicitly archives them via other pathways.

---

### Prompt Compilation

The compiler (`internal/compiler`) builds **`FinalPrompt`**: Markdown sections (**`## Active Goals`**, **`## Constraints`**, **`## Key Decisions`**, **`## Prior Context`**, …) joined by `\n\n---\n\n`, **within `ACGC_TOKEN_BUDGET`**.

**Reservation & priority:** Estimated tokens reserve **system** + **imminent user message** first so the structured body fits what the API will actually send. **All buckets**, including goals and constraints, participate in **`selectWithinBudget`** in priority order (goals → constraints → decisions → compressed → tool outputs → issues → remainder) so nothing bypasses the cap.

**Semantic compile:** When `CompileWithSemantic` runs, archived nodes surfaced by **HNSW** are part of that same priority pass.

**Wire format (`internal/gateway`, `eval/harness`):** chat messages are **`[system, user(context = FinalPrompt), user(current message)]`** — the system string is **not** duplicated inside `FinalPrompt`; `CompiledTokenCount` accounts for **system + FinalPrompt + current user text** for apples-to-apples stats.

**Token counting:** a real, model-aware BPE tokenizer (`internal/tokenizer`, backed by `tiktoken-go`) is used for budgeting and stats. The encoding is resolved from the configured model (e.g. `o200k_base` for GPT-4o/GPT-5, `cl100k_base` for GPT-4). If the encoding can't be loaded it falls back to the historical `len(string) / 4` (~4 chars per token) approximation. `NewCompiler(budget)` remains supported (it uses the process-wide default counter); `NewCompilerWithCounter(budget, counter)` injects an explicit one.

---

### Context Rehydration

When a compressed summary isn't sufficient (e.g., the user asks about a specific detail from an old conversation), ACGC can **rehydrate** — pull the original raw events from MongoDB's archive using the `raw_event_refs` stored on compressed branch nodes.

This means archiving is non-destructive. No data is ever deleted; it's just moved out of the active prompt.

---

## Architecture

### Dual-Path Design

ACGC separates reads (fast, synchronous) from writes (asynchronous, background):

- **Fast Path** (<50ms overhead): The gRPC `Run` call reads the in-memory state tree under a read lock, compiles the prompt, and forwards to the LLM. No database queries on the hot path.
- **Async Path**: Events are enqueued to a per-session buffered Go channel. A dedicated worker goroutine persists to MongoDB, updates the tree, scores, optionally **embeds** and maintains **dual HNSW** indexes when semantic mode is on, and triggers GC. This never blocks the gateway.

### Concurrency Model

```
Session A ──→ [channel A] ──→ [Worker Goroutine A] ──→ writes to Tree A
Session B ──→ [channel B] ──→ [Worker Goroutine B] ──→ writes to Tree B
                                                        │
Gateway (gRPC handlers) ────── reads from Tree A/B ─────┘
                               (via RWMutex read lock)
```

- **Single-writer per session**: Each session has exactly one worker goroutine that processes events sequentially. This eliminates write-write races without complex locking.
- **Concurrent readers**: The gRPC gateway reads the state tree under `sync.RWMutex` read locks. Multiple `CompilePrompt` calls can execute in parallel without blocking each other or the writer.
- **Buffered channels**: Events are queued in a per-session buffered channel (default: 100). If the channel is full, events are dropped with a log warning (backpressure, not crash).
- **No Redis needed**: Go's goroutines + channels + mutexes replace what would typically require Redis Streams for async event processing.

### Storage Layer

**MongoDB Atlas** is the durable persistence layer. Six collections:

| Collection | Purpose | Retention |
|---|---|---|
| `events` | Raw event archive (append-only) | TTL-based (configurable) |
| `state_nodes` | Durable node records (including optional **embedding** vectors + model id for semantic mode) | Per-session lifecycle |
| `compressed_branches` | Compressed branch summaries with original refs | Per-session lifecycle |
| `snapshots` | Periodic full state tree snapshots for crash recovery | Rolling window |
| `gc_logs` | GC audit trail (trigger reason, nodes swept, tokens freed) | Indefinite |
| `session_metrics` | Per-session aggregated metrics | Indefinite |

MongoDB is only used for durability and analytics. The active state tree lives entirely in-memory for speed.

---

## Configuration reference

Copy `.env.example` to `.env` and set what your path needs. **Minimum for a live server:** `ACGC_MONGO_URI` + `ACGC_LLM_API_KEY`. Everything else has sensible defaults. For semantic mode add `ACGC_SEMANTIC_ENABLED=true` (the embed key falls back to the LLM key).

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `ACGC_GRPC_PORT` | `50051` | gRPC server port |
| `ACGC_MONGO_URI` | (required) | MongoDB connection string |
| `ACGC_MONGO_DB` | `acgc` | MongoDB database name |
| `ACGC_LLM_PROVIDER` | `openai` | Main LLM provider |
| `ACGC_LLM_MODEL` | `gpt-4o-mini` | Main LLM model for agent reasoning |
| `ACGC_LLM_API_KEY` | (empty) | API key for the main LLM |
| `ACGC_LLM_BASE_URL` | `https://api.openai.com/v1` | LLM API base URL |
| `ACGC_SUMMARIZER_PROVIDER` | `openai` | Summarizer LLM provider |
| `ACGC_SUMMARIZER_MODEL` | `gpt-4o-mini` | Model for LLM-based branch compression |
| `ACGC_SUMMARIZER_API_KEY` | (empty) | API key for summarizer (falls back to simple compression if empty) |
| `ACGC_TOKEN_BUDGET` | `6000` | Default token budget per compiled prompt |
| `ACGC_MAX_TREE_DEPTH` | `10` | GC trigger: max tree depth |
| `ACGC_MAX_CHILDREN` | `50` | GC trigger: max children per node |
| `ACGC_LOW_RELEVANCE` | `0.30` | GC trigger: sweep when average retention across actives falls below this |
| `ACGC_GC_DECISION_SWEEP_FLOOR` | `0.20` | Soft floor for bare `NodeDecision` sweep score; **must stay below** `ACGC_LOW_RELEVANCE` |
| `ACGC_GC_MAX_ACTIVE_NODES` | `25` | GC trigger: active node count cap (`0` = off) |
| `ACGC_GC_SWEEP_HEADROOM_RATIO` | `0.60` | GC trigger when estimated active tokens exceed ratio × budget (`0` = off) |
| `ACGC_STALE_TURNS` | `15` | Turns before staleness penalty kicks in |
| `ACGC_GC_INTERVAL` | `5` | Check GC every N turns |
| `ACGC_SESSION_BUFFER` | `100` | Per-session event channel buffer size |
| `ACGC_SESSION_IDLE_TIMEOUT` | `1800` | Seconds before idle session is cleaned up |
| `ACGC_SNAPSHOT_INTERVAL` | `60` | Seconds between state tree snapshots |
| `ACGC_SEMANTIC_ENABLED` | `false` | When `true`, construct embedder + dual HNSW per session |
| `ACGC_SEMANTIC_WEIGHT` | `0.20` | Weight on **semantic** signal in scorer; also used for compile-time re-blending |
| `ACGC_HNSW_TOP_K_AT_COMPILE` | `12` | Top-K retrieval from Active index at compile |
| `ACGC_ARCHIVE_SEMANTIC_TOP_K` | `12` | Top-K retrieval from Archive index at compile |
| `ACGC_HNSW_M` | `16` | HNSW graph degree (M) |
| `ACGC_HNSW_EF_SEARCH` | `50` | HNSW ef search width |
| `ACGC_EMBED_PROVIDER` | `openai` | Embedding provider id |
| `ACGC_EMBED_BASE_URL` | `https://api.openai.com/v1` | Embedding API base URL |
| `ACGC_EMBED_API_KEY` | (falls back to `ACGC_LLM_API_KEY`) | Key for embedding calls |
| `ACGC_EMBED_MODEL` | `text-embedding-3-small` | Embedding model name |
| `ACGC_EMBED_DIM` | `1536` | Vector dimension (must match model) |
| `ACGC_LATENCY_BREAKDOWN` | `false` | When `true`, `RunResponse` may include `latency_breakdown` (compile phases + `llm_ms`; minimal overhead when off) |

---

## Benchmarks and Evaluation

Four harnesses answer four different questions. None are required to *use* ACGC — they justify it:

| Question | Harness | Needs MongoDB | Needs API key | Cost |
|---|---|---|---|---|
| Does the pipeline work and save tokens? | [Stress test suite](#stress-test-suite-no-api-key) | No | No | Free |
| How much wall-clock does ACGC add per call? | [Latency bench](#latency-benchmarking-acgc-latencybench) | Yes (live server) | Yes | LLM calls |
| Does answer **quality** hold up on hand-written scenarios? | [Quality evaluation](#quality-evaluation-llm-harness) | No (in-process) | Yes (+ embeddings with `-semantic`) | LLM + judge calls |
| Does ACGC hold up on **published long-memory benchmarks**? | [External benchmark evaluation](#external-benchmark-evaluation) | No (in-process) | Yes (+ embeddings with `-semantic`) | LLM + judge calls (larger) |

### Stress test suite (no API key)

The stress test suite (`stresstest/`) validates ACGC's correctness and performance **without needing an LLM API key**. It runs the full ACGC pipeline in-process (tree → scorer → GC → compiler) against synthetic conversation fixtures.

```bash
# Run with race detector + verbose output (slower than a plain `go run`)
make stresstest

# Same pipeline with mock embeddings + dual HNSW (no API spend)
make stresstest-semantic

# Export results to JSON (tokenizer-backed numbers; see recorded run below)
make stresstest-export

# Custom options
./bin/stresstest -budget 6000 -v -semantic -export stresstest/results.json -skip-concurrency
```

#### What it tests

**Token savings analysis** — Replays 5 synthetic conversations (175 total turns) and compares, at each turn:

- **Raw tokens:** cumulative verbatim transcript, counted with the real BPE tokenizer (`internal/tokenizer`, default **`o200k_base`** via `tiktoken-go`) and summed over every prior turn — the naive “send full history” baseline.
- **Compiled tokens (`CompiledTokenCount`):** same tokenizer-backed count for the simulated next API call (`FinalPrompt` + current turn payload; system message omitted in the harness). Matches production accounting: structured context blob plus the imminent user or assistant utterance once.

Both paths share one counter in `stresstest/runner/engine.go` (`tokenizer.Default()` → compiler via `NewCompilerWithCounter`). The historical `len(string)/4` approximation is used only if tiktoken fails to load (same defensive fallback as the rest of the codebase).

Session-level reduction is **`(final_raw − final_compiled) / final_raw`**, evaluated on the **last turn**—this is exactly where naive history is largest versus a compressed active set plus one current message.

Recorded run (**2026-07-02**), default policy (heuristic-only, `-semantic` off), `go run ./stresstest/ -export stresstest/results.json`:

| Session | Turns | Final raw tokens | Final compiled | Saved | Reduction |
|---|---:|---:|---:|---:|---:|
| long_session | 66 | 8,831 | 2,276 | 6,555 | **74.2%** |
| linear_deep_dive | 38 | 2,383 | 1,948 | 435 | **18.3%** |
| tool_heavy | 20 | 1,894 | 1,593 | 301 | **15.9%** |
| multi_topic_pivot | 31 | 1,533 | 1,547 | −14 | **−0.9%** |
| backtracking | 20 | 1,128 | 1,136 | −8 | **−0.7%** |
| **All sessions** | **175** | **15,769** | **8,500** | **7,269** | **46.1%** |

**Scale takeaway:** **`long_session` is the throughput story** (~8.8k-token cumulative naive history vs ~2.3k-token compiled call)—where GC and compaction actually win. **`multi_topic_pivot`** and **`backtracking`** stay near parity or slightly negative: branching trees under the replay harness accumulate less linear raw history relative to Markdown section overhead (headers, separators) before compaction catches up fully.

With **`-semantic`** (mock deterministic embedder, no API cost), aggregate reduction on the same fixtures was ~**44.2%** overall (mock HNSW slightly shifts which nodes survive into the compilation set); `long_session` remained ~**74.1%** savings.

Artifacts: `stresstest/results.json` (export from the command above).

**Coherency checks** — After GC runs, verifies that important context survives:
- Goal nodes remain active
- Constraint nodes survive GC
- Recent messages appear in the compiled prompt
- No orphaned dependency references
- Compiled prompt is non-empty and within token budget

**Concurrency stress tests** (run with Go's `-race` detector):
- Parallel sessions: all 5 conversations replayed simultaneously
- Concurrent read/write: 1 writer + 5 readers on the same state tree
- GC under contention: GC running while readers query concurrently
- Concurrent compile: 10 compilers reading the same tree in parallel

### Latency benchmarking (`acgc-latencybench`)

The **`acgc-latencybench`** binary (`cmd/acgc-latencybench` + `internal/latencybench/`) measures **how long each path takes** under a repeatable fixture: it runs a **naive baseline** (one direct LLM call with the full scripted transcript) and **`Run`** over **gRPC** to a live **`./bin/acgc`** server.

#### Follow these steps

1. **Start MongoDB** (if local): `make mongo`.
2. **Configure the server** in `.env` or your shell:
   - **`ACGC_SEMANTIC_ENABLED=true`** + **`ACGC_EMBED_API_KEY`** (or rely on fallback to **`ACGC_LLM_API_KEY`**) when you want semantic compile paths exercised.
   - **`ACGC_LATENCY_BREAKDOWN=true`** when you want **`RunResponse.latency_breakdown`** filled (`compile_*` buckets + **`llm_ms`**). With it **off**, percentiles for server breakdown stay empty — **not** zero latency.
3. **Start the daemon**: `make build && ./bin/acgc`.
4. **Run the bench** (same machine env as eval for **`ACGC_LLM_*`**):

```bash
make latency-bench
./bin/acgc-latencybench -grpc localhost:50051 -iterations 30 -discard-n 5 \
  -warm-settle-delay 400ms -output json > latency_report.json
```

5. **Read the report**: JSON prints to **stdout** (redirect as above). Warnings (e.g. missing breakdown) go to **stderr**.

Use **`./bin/acgc-latencybench -h`** for flags (`-sessions`, `-warm-turns`, `-fixture`, `-concurrency`, `-enforce-semantic`, etc.).

#### What each number means

| Output bucket | What it actually measures |
|---------------|---------------------------|
| **`baseline_llm_ms`** | Client-side wall clock around **one** **`Generate`** call: scripted **system + all warm turns + probe** (no ACGC compile). |
| **`acgc_run_round_trip_ms`** | Client-side wall clock for the entire **`Run`** RPC = **compile + server LLM + tiny gRPC overhead**. **Includes LLM time**, not “non-LLM only.” |
| **`latency_breakdown.llm_ms`** | Server-side wall clock around **`Generate`** inside the gateway (**upstream model only**, after compile). |
| **`latency_breakdown.compile_total_ms`** | Server-side wall clock around **`CompilePrompt`** (embedding / HNSW / Markdown assembly roll-up). |
| **`latency_breakdown.compile_embed_ms`** | Time in **`Embed`** at compile time (often close to **`compile_total`** when index + assembly are sub‑millisecond). |

**Percentiles:**

| Stat | Plain-language meaning |
|------|-------------------------|
| **P50** | **Median** — half of samples are faster, half slower (“typical” run). |
| **P95** | **Tail** — 95% of samples finish within this time; captures noisy providers / contention. |
| **P99** | **Extreme tail** — rare slow runs (GC spikes, slow embeds, API stalls). |

The harness applies **`-discard-n N`** by dropping the **first N iterations by index** from percentile summaries only (burn-in); raw **`samples`** still list every iteration.

#### Major driver (what dominates latency)

Inside a successful **`Run`**, **`llm_ms` ≫ `compile_total_ms`** in typical setups: **upstream LLM generation is the largest component**. Compile adds **hundreds of ms to a few seconds** on semantic paths; **`run_round_trip` ≈ `compile_total_ms` + `llm_ms`** (+ small slack).

Baseline **`baseline_llm_ms`** vs server **`llm_ms`** are **not identical prompts** (verbatim transcript vs compiled **`FinalPrompt` + probe framing), so **longer `llm_ms` does not imply a timer bug** — it usually means **different prompt/output workload**.

#### Recorded example run (reference numbers)

Below is one **representative** bench captured **2026-05-14**: **`localhost:50051`**, **`iterations=30`**, **`discard_n=5`**, **`concurrency=2`**, default embedded fixture (**2 warm pairs**, probe about **`go.mod` `replace` directives**), **`ACGC_LATENCY_BREAKDOWN=true`**, model **`gpt-5`** via OpenAI-compatible API. **Two iterations** hit transient **`connection reset by peer`** errors and are **excluded** from the percentile aggregates below.

**End-to-end comparison** — naive vs full **`Run`** (**both include baseline-side LLM only on the baseline column**; ACGC column is **full RPC** including compile **and** server LLM):

| Metric | P50 (median) | P95 (tail) | P99 (extreme tail) |
|--------|-------------:|-----------:|-------------------:|
| Naive baseline LLM wall | 9.3 s | 15.0 s | 18.1 s |
| ACGC **`Run`** client RTT | 11.6 s | 19.2 s | 22.0 s |
| **Net Δ** (ACGC − baseline) | **+2.3 s** | **+4.2 s** | **+3.8 s** |

**Inside ACGC `Run`** (server **`latency_breakdown`** — same percentile basis):

| Component | P50 | P95 | P99 |
|-----------|----:|----:|----:|
| **`llm_ms`** (upstream completion) | 11.2 s | 18.8 s | 21.2 s |
| **`compile_total_ms`** | 0.65 s | 1.05 s | 2.39 s |

**Takeaway:** On this fixture, **`Run`** was **~2–4 s slower at median/tail** than sending the naive transcript once; **most of wall clock inside `Run` is still `llm_ms`**. Your numbers will vary with **model**, **network**, **`warm-settle-delay`**, and **`discard_n`**.

### Quality evaluation (LLM harness)

The **`eval/`** package runs end-to-end comparisons across a configurable set of **context strategies** selected with `-strategies` (comma-separated, first = reference):

- **`naive_full_history`** — all history up to the token budget (the default reference; "no context management").
- **`sliding_window`** — most-recent context only, filling the budget newest-first.
- **`acgc`** — full ACGC replay with GC/compiler and optional **semantic retrieval** (active + archive HNSW).

Every strategy shares the same system prompt, budget, LLM, tokenizer, scoring, and cache, so results are directly comparable. It records **prompt token counts** (via a real model-aware BPE tokenizer, `internal/tokenizer`), **latency**, **probe-based** factual checks (`MatchContains*` on expected needles), **LLM-as-judge** scores for open-ended probes, **intelligence-per-token** (IPT = quality ÷ prompt tokens), and aggregates candidate-vs-reference verdicts plus a side-by-side per-strategy table.

Requires **`ACGC_LLM_API_KEY`** (and embeddings when using `-semantic`; see `eval/README.md`). Reports land in **`eval/results/`** (`report.md`, `results.json`). Run all three strategies with `make eval-strategies` or `go run ./eval -strategies "naive_full_history,sliding_window,acgc" -v`.

#### How to reproduce

```bash
make eval-clean    # wipe eval/cache + eval/results (optional but recommended for fresh run)
make eval-semantic-judge
```

Equivalent: `go run ./eval -v -semantic -judge`

#### Recorded run (**2026-07-01**)

Configuration as executed: **`gpt-5`** for answer + judge generations (from **`ACGC_LLM_MODEL`** / env), embeddings via **`-semantic`** (`text-embedding-3-small`), semantic weight **0.20**, top-K **12**, archive semantic top-K **12**, LLM judge on, raised answer cap via **`-max-tokens`** (see below). **Three strategies compared** — `naive_full_history` (reference), `sliding_window`, `acgc` — across **7 scenarios / 12 probes each** (24 candidate-vs-reference pairs).

##### Aggregate summary (per strategy, 12 probes)

| Strategy | Avg quality (/5.0) | Avg prompt tokens | Avg IPT | Token reduction vs ref | Quality Δ vs ref |
|---|---:|---:|---:|---:|---:|
| `naive_full_history` (ref) | 5.00 | 2738 | 4.59 | 0.0% | +0.00 |
| `sliding_window` | 3.25 | 2733 | 4.25 | 0.2% | **−1.75** |
| `acgc` | **5.00** | **2082** | **5.05** | **24.0%** | **+0.00** |

Candidate-vs-reference verdicts across 24 pairs: **`ACGC_WIN` = 12, `TIE` = 8, `ACGC_LOSS` = 4** (all four losses are `sliding_window` on the deep-history scenario; **ACGC has zero regressions**).

Interpretation (harness semantics): **`ACGC_WIN`** = strictly better IPT on that pair **without** a quality regression relative to the reference; probes that tie on quality with no token win remain **`TIE`**. **`acgc` matches the reference's perfect quality at 24% fewer tokens and the best IPT, while `sliding_window` — a pure recency heuristic — saves no tokens and collapses in quality once history exceeds the budget.**

##### Two regimes

**Regime 1 — history fits the budget (the six small scenarios).** No strategy has to truncate, so `sliding_window` sees the same full history as the reference (**identical prompts → `TIE`, 0% savings**), while `acgc` still compresses for a free discount at equal quality:

| Scenario / probe | Scoring | Quality (ref / acgc) | Prompt tokens (ref / acgc) | Token savings | Verdict (acgc) |
|---|---|---:|---:|---:|---|
| `recent_recall_1` / `p1` | Probe | 5.0 / 5.0 | 307 / **298** | **2.9%** | **`ACGC_WIN`** |
| `long_range_recall_1` / `p1`–`p3` | Probe | 5.0 / 5.0 | ~1092 / ~**977** | ~**10.5%** each | **`ACGC_WIN`** (×3) |
| `constraint_adherence_1` / `p1` | Judge | 5.0 / 5.0 | 842 / **761** | **9.6%** | **`ACGC_WIN`** |
| `topic_switch_return_1` / `p1` | Probe | 5.0 / 5.0 | 787 / **724** | **8.0%** | **`ACGC_WIN`** |
| `contradiction_1` / `p1` | Judge | 5.0 / 5.0 | 969 / **878** | **9.4%** | **`ACGC_WIN`** |
| `multi_hop_synth_1` / `p1` | Judge | 5.0 / 5.0 | 1111 / **1000** | **10.0%** | **`ACGC_WIN`** |

**Regime 2 — history exceeds the budget (`deep_history_recall_1`, ~13.3k raw tokens, >2× the 6000-token budget).** Four decisions stated up front are buried under ~13k tokens of filler. Now the budget bites and the strategies diverge hard:

| Probe | Quality (ref / cand) | Prompt tokens (ref / cand) | Token savings | Verdict |
|---|---:|---:|---:|---|
| `deep_history_recall_1` / `p1`–`p4` · **acgc** | 5.0 / **5.0** | ~6392 / **~4597** | ~**28%** each | **`ACGC_WIN`** (×4) |
| `deep_history_recall_1` / `p1`–`p4` · **sliding_window** | 5.0 / **0.0** | ~6392 / ~6378 | ~0.2% | **`ACGC_LOSS`** (×4) |

- **`naive_full_history`** fills the budget oldest-first, so the early decisions stay in-window → recalls all four (5.0), but burns the full ~6,400-token budget.
- **`sliding_window`** fills newest-first, so the early decisions fall off the window entirely → **0.0 on every probe** at the same token cost — the silent "lost the old decision" failure mode.
- **`acgc`** retention-scores + compresses, keeping the four decisions at **~28% lower token cost** with full quality.

##### On the raised answer cap

Two reasoning-heavy probes (`constraint_adherence_1`, `multi_hop_synth_1`) previously exhausted the old hardcoded 2,500-token completion cap on hidden reasoning and returned **empty text scored 0**. With the new **`-max-tokens`** flag (default **6000**, raised to **10000** for this run to give the compressed-context `multi_hop` probe enough room), **all previously-empty probes now produce non-empty scored answers** — which is why the reference and `acgc` both reach a clean 5.00 aggregate.

Artifacts for this snapshot: regenerate with the command above (add `-max-tokens 10000` for the reasoning-heavy probes), or inspect **`eval/results/report.md`** + **`eval/results/results.json`**.

### External benchmark evaluation

The built-in quality harness uses small, hand-written scenarios. **External benchmarks** run the same three-strategy comparison against **published long-memory datasets** — real multi-session chat logs and QA probes — so the numbers reflect how ACGC behaves on workloads closer to production agents.

#### What it tests

We ask a simple question for each probe: **can ACGC answer a memory question as well as (or better than) sending the full chat history, while using fewer tokens?** An LLM judge scores every answer 0–5. All three strategies share the same model, budget, tokenizer, and judge; semantic retrieval is on (`-semantic`: embeddings + archive HNSW).

| Benchmark | What it simulates | Our recorded run |
|---|---|---|
| **[LongMemEval](https://github.com/xiaowu0162/LongMemEval)** | Many chat sessions over weeks; one question per instance (“what did I say about X last month?”) | 20 sampled instances |
| **[LoCoMo](https://github.com/snap-research/locomo)** | Two-speaker long dialogues; many QA probes per conversation (factual, temporal, multi-hop, adversarial) | 5 conversations, 100 probes |

Strategies compared (same as built-in eval):

- **`naive_full_history`** — reference; stuff as much history as fits the budget.
- **`sliding_window`** — keep only the most recent turns.
- **`acgc`** — full ACGC stack with GC, compiler, and semantic archive retrieval.

#### How to reproduce

```bash
make eval-fetch-external          # downloads datasets (gitignored under eval/datasets/external/data/)
make eval-longmemeval-semantic    # LongMemEval, 20 instances
make eval-locomo-semantic         # LoCoMo, 10 convs × 20 probes

# LoCoMo subset (5 conversations):
go run ./eval -v -judge -semantic \
  -strategies "naive_full_history,sliding_window,acgc" \
  -external "locomo=eval/datasets/external/data/locomo10.json" \
  -external-sample 20 \
  -scenarios "locomo_conv-26,locomo_conv-30,locomo_conv-41,locomo_conv-42,locomo_conv-43"
```

Reports land in **`eval/results/`** with an `external_<benchmark>_semantic_` prefix (e.g. `external_longmemeval_semantic_report.md`). Adapter details and flags: **`eval/README.md`**.

#### Recorded semantic runs (**2026-07-04**)

Configuration: **`gpt-5`** for answer + judge, **`text-embedding-3-small`** embeddings, semantic weight **0.20**, **6000-token** budget, three strategies.

**LongMemEval (20 instances)**

| Strategy | Avg quality (/5.0) | Avg prompt tokens | Token savings vs naive | Verdicts vs naive |
|---|---:|---:|---:|---|
| `naive_full_history` (ref) | 2.20 | 6235 | — | — |
| `sliding_window` | 2.30 | 6214 | ~0% | — |
| **`acgc`** | **3.00** | **2473** | **60.3%** | 30 WIN, 9 TIE, 0 LOSS |

**LoCoMo (5 conversations, 100 probes)** — `conv-26`, `30`, `41`, `42`, `43`

| Strategy | Avg quality (/5.0) | Avg prompt tokens | Token savings vs naive | Verdicts vs naive |
|---|---:|---:|---:|---|
| `naive_full_history` (ref) | 2.78 | 6734 | — | — |
| **`sliding_window`** | **3.27** | 6761 | ~0% | — |
| `acgc` | 3.13 | 6123 | 9.1% | 130 WIN, 36 TIE, 34 LOSS |

Artifacts: **`eval/results/external_longmemeval_semantic_report.md`**, **`eval/results/external_locomo_semantic_report.md`** (+ matching `..._results.json`).

#### Takeaways

- **LongMemEval is ACGC's home turf.** Histories are long and spread across many sessions — "most recent turns" (sliding window) usually miss the answer. Semantic ACGC pulls the right old content back from the archive and wins on **both** quality (+0.80 vs naive) and cost (~60% fewer tokens), with zero quality regressions.
- **LoCoMo is a different beast.** Dialogue is dense; most turns matter and the history already sits near the token budget (~6.7k naive vs 6k budget). **Recent context often is the answer**, so sliding window scores highest (3.27). ACGC still beats naive (+0.35 quality, ~9% savings) but not sliding. **Temporal questions** are the main weak spot — session-date annotations and older turns can be compressed away.
- **Rule of thumb:** use **ACGC + semantic** for agents with long, multi-session memory (support tickets, research assistants, personal memory). For **short, dense chats** already near budget, **sliding window** may win on quality until temporal handling improves — or run both and pick by workload.

---

## Project Structure

```
acgcProject/
├── cmd/
│   ├── acgc/              # Server entry point
│   └── testcli/           # Interactive REPL test client
├── api/proto/             # Generated gRPC Go code
├── proto/                 # Protobuf service definitions
├── pkg/acgc/              # Public Go SDK (ContextRuntime)
├── internal/
│   ├── config/            # Env-based configuration + .env loader
│   ├── domain/            # Core types: Event, StateNode, CompiledPrompt, SessionMetrics
│   ├── store/             # MongoDB persistence (6 collections, 25+ methods)
│   ├── statetree/         # In-memory state tree with RWMutex, node classification
│   ├── scorer/            # Heuristic + semantic (cosine) retention scoring
│   ├── gc/                # Mark-sweep-compact GC + SimpleCompressor + LLMCompressor
│   ├── compiler/          # Budget-aware Markdown assembly (`FinalPrompt`; system separate)
│   ├── facts/             # Deterministic Facts/Decisions extraction + merge helpers
│   ├── embedding/         # OpenAI-compatible embed provider interface
│   ├── vectorindex/       # In-memory HNSW wrapper (dual Active/Archive per session)
│   ├── llm/               # OpenAI-compatible HTTP client (GPT-5/o3 reasoning model support)
│   ├── tokenizer/         # Model-aware BPE token counting (tiktoken-go + fallback)
│   ├── session/           # Session worker + CompilePrompt semantic merge helpers
│   └── gateway/           # gRPC server implementation
├── stresstest/
│   ├── fixtures/          # Synthetic conversation generator (5 scenarios, 175 turns)
│   ├── runner/            # Replay engine, coherency checker, concurrency tests, reporter
│   └── main.go            # Stress test CLI entry point
├── eval/
│   ├── main.go            # Eval CLI
│   ├── datasets/          # Scripted scenarios + probes
│   ├── harness/           # Pluggable context strategies (naive/sliding/acgc) + caching
│   ├── scoring/           # Probe matching + LLM judge + metrics
│   └── report/            # Generates eval/results/report.md and results.json
├── mongo-init/            # MongoDB index + TTL setup script
├── Makefile               # Build, run, test, stress test targets
├── docker-compose.yml     # Local MongoDB (alternative to Atlas)
├── .env.example           # Environment variable template
└── go.mod
```

---

## Current Status vs Roadmap

### Shipped (MVP)

| Component | Status | Details |
|---|---|---|
| gRPC interface | Done | 5 RPCs, language-agnostic, proto definitions |
| Go SDK | Done | `pkg/acgc` with `ContextRuntime` |
| State tree | Done | In-memory, typed nodes, Facts/Decisions, optional embeddings |
| Heuristic + semantic scorer | Done | Eight signals plus **0.20× cosine similarity** when semantic mode is on; heuristic-only stays sub-millisecond for ~100 nodes (embedding HTTP calls add their own latency) |
| Dual HNSW (active + archive) | Done | Per-session graphs, GC reconciliation, Mongo rehydration |
| Facts pipeline | Done | `internal/facts` extraction, GC deferral, compressor + compiler hooks |
| Garbage collector | Done | Mark-sweep-compact + **soft headroom** + **max active nodes** + hybrid factual protection |
| Simple compressor | Done | String-based branch compression (no LLM needed) |
| LLM compressor | Done | OpenAI-compatible summaries + **`ENTITIES:`** → verbatim facts |
| Prompt compiler | Done | Budgeted Markdown sections + **dual user messages** (`FinalPrompt` + current turn) |
| MongoDB persistence | Done | 6 collections; node embeddings + archived embedding queries |
| Concurrency model | Done | Per-session goroutines, channels, RWMutex |
| Context rehydration | Done | Pull raw events from archive for compressed branches |
| Interactive test client | Done | REPL with real LLM integration |
| Stress test suite | Done | Token savings, coherency, concurrency (race-free) |
| Quality evaluation (`eval/`) | Done | Pluggable strategies (naive / sliding_window / acgc) with probe + judge scoring; semantic path optional |
| LLM compatibility | Done | GPT-5 / o3 reasoning model support (dynamic parameter handling) |

### Planned (Post-MVP)

| Feature | Priority | Description |
|---|---|---|
| **Shared vector tier (Redis / external ANN)** | High | Ship path already uses **in-process dual HNSW + Mongo embeddings**. Moving the graph to **Redis (RediSearch / Redis Stack)** or another shared ANN tier would accelerate cold/warm multi-instance deployments, reduce per-process memory for very large sessions, and centralize vector updates—**not** a prerequisite for semantic retrieval, which works today on a single node. |
| **Redis Streams for event processing** | Medium | Replace per-session goroutines with Redis Streams for distributed event processing. Enables horizontal scaling — multiple ACGC instances can process events for different sessions. Also provides durable event queues (current channels lose events on crash). |
| **Policy engine** | Medium | Configurable GC policies per session/task. Aggressive (minimize tokens, accept lower coherency), conservative (preserve more context, higher token cost), balanced (current default). Policy hot-swapping during a session. |
| **Semantic deduplication** | Medium | Use embeddings to detect near-duplicate nodes (e.g., user asks the same question rephrased). Currently only detects exact title matches. |
| **Streaming support** | Medium | gRPC server-streaming for `Run` — stream LLM tokens back as they arrive instead of waiting for the full response. |
| **Multi-agent context sharing** | Low | Allow multiple agents to share a context tree (e.g., a coding agent and a review agent working on the same task). Requires conflict resolution for concurrent tree modifications. |
| **Admin dashboard** | Low | Web UI for inspecting state trees, viewing GC history, monitoring token savings across sessions, and manually triggering operations. |
| **Observability** | Low | Prometheus metrics endpoint, OpenTelemetry tracing for the full request lifecycle, structured JSON logging with correlation IDs. |
| **Context importance hints** | Low | Allow the agent to annotate events with importance hints ("this decision is critical", "this is temporary debug output") that the scorer uses as additional signals. |
| **Tiered storage** | Low | Hot tier (in-memory) → Warm tier (Redis/SSD) → Cold tier (MongoDB/S3). Currently only hot + cold. The warm tier would hold recently-archived nodes for faster rehydration. |
