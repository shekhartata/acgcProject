# ACGC — Agent Context Garbage Collector

A Go sidecar runtime that sits between an AI agent and its LLM, intercepting every interaction to build a structured context model. It scores relevance, prunes stale information, compresses resolved branches, and compiles only the most useful context into each LLM call — reducing token costs, lowering latency, and improving response quality.

---

## Table of Contents

- [Problem](#problem)
- [How It Works](#how-it-works)
  - [End-to-End Flow](#end-to-end-flow)
  - [State Tree](#state-tree)
  - [Relevance Scoring](#relevance-scoring)
  - [Garbage Collection](#garbage-collection)
  - [Prompt Compilation](#prompt-compilation)
  - [Context Rehydration](#context-rehydration)
- [Architecture](#architecture)
  - [Dual-Path Design](#dual-path-design)
  - [Concurrency Model](#concurrency-model)
  - [Storage Layer](#storage-layer)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Build and Run](#build-and-run)
  - [Environment Variables](#environment-variables)
- [Using ACGC](#using-acgc)
  - [Go SDK](#go-sdk)
  - [gRPC API](#grpc-api)
  - [Interactive Test Client](#interactive-test-client)
- [Stress Testing](#stress-testing)
- [Project Structure](#project-structure)
- [Current Status vs Roadmap](#current-status-vs-roadmap)

---

## Problem

LLM-powered agents accumulate context over long conversations — user messages, assistant responses, tool calls, tool results, errors, retries. Without management, this context grows unbounded:

- **Token bloat**: Sending the entire conversation history gets expensive fast. A 60-turn session can easily hit 10,000+ tokens per call.
- **Noise**: Old resolved issues, failed attempts, and redundant information dilute the signal. The LLM starts contradicting earlier decisions or forgetting constraints.
- **Latency**: More input tokens = slower responses, especially with reasoning models (GPT-5, o3) that scale processing time with input size.

ACGC solves this by treating context like memory in a running program — actively managing what stays, what gets compressed, and what gets archived.

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
│ 9. ASSEMBLE prompt:
│    system → user request → goals →
│    constraints → decisions →
│    compressed context → tool outputs →
│    open issues
│
│ 10. FORWARD to LLM
│
│ 11. RETURN response + stats
│     (original tokens, compiled tokens,
│      savings %, GC info)
└─────────┘
```

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

Nodes track: parent-child relationships, raw event references (for rehydration), token counts, creation/update times, and dependency links to other nodes.

**Node lifecycle:** `active` → `stale` → `archived` (or `active` → `compressed` when a branch is compacted).

### Relevance Scoring

Every active node receives a **retention score** (0.0 to 1.0) computed from 8 weighted signals:

| Signal | Weight | What It Measures |
|---|---|---|
| Recency | 0.25 | Exponential decay based on turns since creation. Recent = high. |
| Type Priority | 0.20 | Inherent importance of the node type (goals=1.0, background=0.2) |
| Dependency Weight | 0.15 | How many other active nodes depend on this one |
| Unresolved Boost | 0.15 | +1.0 if the node has open issues |
| Redundancy Penalty | -0.10 | Penalizes duplicate/similar nodes |
| Resolved Penalty | -0.20 | Penalizes nodes marked as resolved |
| Stale Penalty | -0.15 | Grows as node age exceeds `ACGC_STALE_TURNS` |
| Size Penalty | -0.05 | Penalizes excessively large nodes |

**Formula:**
```
RetentionScore = clamp(
    0.25×Recency + 0.20×TypePriority + 0.15×DependencyWeight + 0.15×UnresolvedBoost
  - 0.10×Redundancy - 0.20×ResolvedPenalty - 0.15×StalePenalty - 0.05×SizePenalty,
  0.0, 1.0
)
```

This is purely heuristic — no LLM calls, no embeddings. Fast enough to run on every turn (<1ms for 100 nodes).

### Garbage Collection

GC uses a **mark-sweep-compact** cycle, triggered automatically when any threshold is exceeded:

| Trigger | Default Threshold | What It Means |
|---|---|---|
| Token pressure | `6000` tokens | Total active node tokens exceed the budget |
| Tree depth | `10` levels | Tree is too deep (long dependency chains) |
| Tree width | `50` children | A single node has too many children |
| Low relevance | `0.30` avg score | Average retention score dropped too low |
| Resolved branch | Any resolved nodes | Nodes marked resolved can be cleaned up |
| Manual | N/A | Triggered via `TriggerGC` gRPC call |

**GC phases:**

1. **Mark**: Re-score all active nodes. Identify nodes with retention score below the threshold.
2. **Sweep**: Move low-scoring nodes to `archived` status. They're removed from the active prompt but their raw events remain in MongoDB.
3. **Compact**: If a parent has too many children or all children are resolved, compress the branch — multiple nodes are replaced by a single `compressed_branch` summary node.

**What GC preserves:** Goals, active constraints, nodes with open issues, nodes that other active nodes depend on. These get natural protection through higher type priority and boost signals.

**What GC removes:** Old tool outputs, resolved attempts, stale background context, redundant information.

### Prompt Compilation

The compiler assembles the final prompt within a strict token budget. It follows a priority order:

1. **System prompt** (always included)
2. **Current user message** (always included)
3. **Active goals** (always included — not budget-limited)
4. **Active constraints** (always included — not budget-limited)
5. **Key decisions** (budget-limited, sorted by retention score)
6. **Compressed context** (budget-limited)
7. **Tool outputs** (budget-limited)
8. **Open issues** (budget-limited)
9. **Additional context** (fills remaining budget)

Nodes within each bucket are sorted by retention score. The compiler greedily fills the budget from highest to lowest priority. Excluded nodes are tracked in the response (their IDs are returned for transparency).

**Token estimation:** Uses a `len(string) / 4` heuristic (~4 chars per token). This is fast but approximate — suitable for budget management without requiring a tokenizer dependency.

### Context Rehydration

When a compressed summary isn't sufficient (e.g., the user asks about a specific detail from an old conversation), ACGC can **rehydrate** — pull the original raw events from MongoDB's archive using the `raw_event_refs` stored on compressed branch nodes.

This means archiving is non-destructive. No data is ever deleted; it's just moved out of the active prompt.

---

## Architecture

### Dual-Path Design

ACGC separates reads (fast, synchronous) from writes (asynchronous, background):

- **Fast Path** (<50ms overhead): The gRPC `Run` call reads the in-memory state tree under a read lock, compiles the prompt, and forwards to the LLM. No database queries on the hot path.
- **Async Path**: Events are enqueued to a per-session buffered Go channel. A dedicated worker goroutine processes them: persists to MongoDB, updates the tree, scores, and triggers GC. This never blocks the gateway.

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
| `state_nodes` | Durable node records | Per-session lifecycle |
| `compressed_branches` | Compressed branch summaries with original refs | Per-session lifecycle |
| `snapshots` | Periodic full state tree snapshots for crash recovery | Rolling window |
| `gc_logs` | GC audit trail (trigger reason, nodes swept, tokens freed) | Indefinite |
| `session_metrics` | Per-session aggregated metrics | Indefinite |

MongoDB is only used for durability and analytics. The active state tree lives entirely in-memory for speed.

---

## Getting Started

### Prerequisites

- Go 1.25+
- MongoDB Atlas account (or local MongoDB via `docker compose up -d mongodb`)
- `protoc` + Go gRPC plugins (only for proto regeneration)

### Build and Run

```bash
# Copy and configure environment
cp .env.example .env
# Edit .env: set ACGC_MONGO_URI, optionally ACGC_LLM_API_KEY

# Build all binaries
make build

# Start the ACGC server
make run

# In another terminal, run the interactive test client
make testcli

# Run the stress test suite
make stresstest
```

### Environment Variables

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
| `ACGC_LOW_RELEVANCE` | `0.30` | GC trigger: minimum average retention score |
| `ACGC_STALE_TURNS` | `15` | Turns before staleness penalty kicks in |
| `ACGC_GC_INTERVAL` | `5` | Check GC every N turns |
| `ACGC_SESSION_BUFFER` | `100` | Per-session event channel buffer size |
| `ACGC_SESSION_IDLE_TIMEOUT` | `1800` | Seconds before idle session is cleaned up |
| `ACGC_SNAPSHOT_INTERVAL` | `60` | Seconds between state tree snapshots |

---

## Using ACGC

### Go SDK

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/chandrashekhartata/acgc/pkg/acgc"
)

func main() {
    runtime, err := acgc.NewContextRuntime(acgc.Config{
        ServerAddr:  "localhost:50051",
        SessionID:   "my-session",
        TaskID:      "my-task",
        TokenBudget: 6000,
        Policy:      acgc.PolicyBalanced,
        LLM: acgc.LLMConfig{
            Provider: "openai",
            Model:    "gpt-5",
            APIKey:   "sk-...",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer runtime.Close()

    result, err := runtime.Run(context.Background(), "Help me design a database schema")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Response)
    fmt.Printf("Tokens saved: %d (%.1f%% reduction)\n",
        result.TokensSaved, result.ReductionPercent)
}
```

### gRPC API

Any language can call ACGC via gRPC. The service definition is in `proto/acgc.proto`:

| RPC | Description |
|---|---|
| `Run` | Full intercept → compile → forward → capture cycle |
| `CaptureEvent` | Manually capture a single event into the state tree |
| `GetState` | Inspect the current state tree and node scores |
| `TriggerGC` | Manually trigger garbage collection |
| `GetMetrics` | Get token savings, GC stats, and session metrics |

Generate client stubs for any language:

```bash
# Go (already generated)
make proto

# Python
pip install grpcio-tools
python -m grpc_tools.protoc -I proto --python_out=. --grpc_python_out=. proto/acgc.proto

# TypeScript
npx grpc_tools_node_protoc --ts_out=. --grpc_out=. -I proto proto/acgc.proto
```

### Interactive Test Client

The `testcli` binary provides a REPL for manually testing ACGC with a real LLM:

```bash
make testcli
```

Type messages and see real-time token savings, GC triggers, and compiled prompt stats after each turn.

---

## Stress Testing

The stress test suite (`stresstest/`) validates ACGC's correctness and performance **without needing an LLM API key**. It runs the full ACGC pipeline in-process (tree → scorer → GC → compiler) against synthetic conversation fixtures.

```bash
# Run with race detector + verbose output
make stresstest

# Export results to JSON
make stresstest-export

# Custom options
./bin/stresstest -budget 4000 -v -export results.json -skip-concurrency
```

### What It Tests

**Token savings analysis** — Replays 5 synthetic conversations (175 total turns) and compares raw tokens (without ACGC) vs compiled tokens (with ACGC) at every turn:

| Session | Turns | Without ACGC | With ACGC | Reduction |
|---|---|---|---|---|
| long_session | 66 | 11,109 tokens | 491 tokens | 95.6% |
| linear_deep_dive | 38 | 2,674 tokens | 473 tokens | 82.3% |
| multi_topic_pivot | 31 | 1,710 tokens | 652 tokens | 61.9% |
| tool_heavy | 20 | 1,884 tokens | 857 tokens | 54.5% |
| backtracking | 20 | 1,322 tokens | 683 tokens | 48.3% |
| **Total** | **175** | **18,699** | **3,156** | **83.1%** |

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
│   ├── scorer/            # 8-signal heuristic retention scorer
│   ├── gc/                # Mark-sweep-compact GC + SimpleCompressor + LLMCompressor
│   ├── compiler/          # Token-budget-aware prompt compiler (priority-ordered assembly)
│   ├── llm/               # OpenAI-compatible HTTP client (GPT-5/o3 reasoning model support)
│   ├── session/           # Per-session goroutine manager (single-writer, channel-based)
│   └── gateway/           # gRPC server implementation
├── stresstest/
│   ├── fixtures/          # Synthetic conversation generator (5 scenarios, 175 turns)
│   ├── runner/            # Replay engine, coherency checker, concurrency tests, reporter
│   └── main.go            # Stress test CLI entry point
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
| State tree | Done | In-memory, node classification, parent-child relationships |
| Heuristic scorer | Done | 8 weighted signals, <1ms per 100 nodes |
| Garbage collector | Done | Mark-sweep-compact, 6 trigger types, policy-configurable |
| Simple compressor | Done | String-based branch compression (no LLM needed) |
| LLM compressor | Done | OpenAI-compatible, used when summarizer API key is configured |
| Prompt compiler | Done | Priority-ordered, budget-constrained assembly |
| MongoDB persistence | Done | 6 collections, bulk ops, snapshots, rehydration |
| Concurrency model | Done | Per-session goroutines, channels, RWMutex |
| Context rehydration | Done | Pull raw events from archive for compressed branches |
| Interactive test client | Done | REPL with real LLM integration |
| Stress test suite | Done | Token savings, coherency, concurrency (race-free) |
| LLM compatibility | Done | GPT-5 / o3 reasoning model support (dynamic parameter handling) |

### Planned (Post-MVP)

| Feature | Priority | Description |
|---|---|---|
| **Semantic scoring (Redis vector search)** | High | Replace/augment heuristic scoring with embedding-based similarity. Embeddings are stored durably in MongoDB alongside nodes, but the vector index lives in **Redis (RediSearch/Redis Stack)** for sub-millisecond in-memory similarity lookups on the hot path. Redis becomes the "in-memory layer" for embeddings — same pattern as the state tree (in-memory for speed, MongoDB for durability). MongoDB rebuilds the Redis index on startup or after a crash. This would dramatically improve coherency for topic-switching conversations. |
| **Redis Streams for event processing** | Medium | Replace per-session goroutines with Redis Streams for distributed event processing. Enables horizontal scaling — multiple ACGC instances can process events for different sessions. Also provides durable event queues (current channels lose events on crash). |
| **Policy engine** | Medium | Configurable GC policies per session/task. Aggressive (minimize tokens, accept lower coherency), conservative (preserve more context, higher token cost), balanced (current default). Policy hot-swapping during a session. |
| **Semantic deduplication** | Medium | Use embeddings to detect near-duplicate nodes (e.g., user asks the same question rephrased). Currently only detects exact title matches. |
| **Streaming support** | Medium | gRPC server-streaming for `Run` — stream LLM tokens back as they arrive instead of waiting for the full response. |
| **Multi-agent context sharing** | Low | Allow multiple agents to share a context tree (e.g., a coding agent and a review agent working on the same task). Requires conflict resolution for concurrent tree modifications. |
| **Admin dashboard** | Low | Web UI for inspecting state trees, viewing GC history, monitoring token savings across sessions, and manually triggering operations. |
| **Observability** | Low | Prometheus metrics endpoint, OpenTelemetry tracing for the full request lifecycle, structured JSON logging with correlation IDs. |
| **Context importance hints** | Low | Allow the agent to annotate events with importance hints ("this decision is critical", "this is temporary debug output") that the scorer uses as additional signals. |
| **Tiered storage** | Low | Hot tier (in-memory) → Warm tier (Redis/SSD) → Cold tier (MongoDB/S3). Currently only hot + cold. The warm tier would hold recently-archived nodes for faster rehydration. |
