// ACGC MongoDB initialization script
// Creates collections with indexes and TTL policies.

db = db.getSiblingDB("acgc");

// ---------------------------------------------------------------------------
// 1. events — append-only raw event archive
// ---------------------------------------------------------------------------
db.createCollection("events");

db.events.createIndex({ "session_id": 1, "sequence": 1 }, { name: "idx_events_session_seq" });
db.events.createIndex({ "session_id": 1, "event_type": 1 }, { name: "idx_events_session_type" });
db.events.createIndex({ "session_id": 1, "task_id": 1 },    { name: "idx_events_session_task" });
db.events.createIndex({ "created_at": 1 },                   { name: "idx_events_created" });
db.events.createIndex({ "event_id": 1 },                     { unique: true, name: "idx_events_id_unique" });

// ---------------------------------------------------------------------------
// 2. state_nodes — durable individual state tree node records
// ---------------------------------------------------------------------------
db.createCollection("state_nodes");

db.state_nodes.createIndex({ "session_id": 1, "node_id": 1 },  { unique: true, name: "idx_nodes_session_node" });
db.state_nodes.createIndex({ "session_id": 1, "status": 1 },   { name: "idx_nodes_session_status" });
db.state_nodes.createIndex({ "session_id": 1, "node_type": 1 }, { name: "idx_nodes_session_type" });
db.state_nodes.createIndex({ "session_id": 1, "parent_id": 1 }, { name: "idx_nodes_session_parent" });

// ---------------------------------------------------------------------------
// 3. compressed_branches — durably stored compressed branch summaries
// ---------------------------------------------------------------------------
db.createCollection("compressed_branches");

db.compressed_branches.createIndex({ "branch_id": 1 },            { unique: true, name: "idx_compressed_id_unique" });
db.compressed_branches.createIndex({ "session_id": 1 },           { name: "idx_compressed_session" });
db.compressed_branches.createIndex({ "session_id": 1, "task_id": 1 }, { name: "idx_compressed_session_task" });

// ---------------------------------------------------------------------------
// 4. snapshots — periodic full state tree snapshots for crash recovery
// ---------------------------------------------------------------------------
db.createCollection("snapshots");

db.snapshots.createIndex({ "session_id": 1, "snapshot_at": -1 }, { name: "idx_snapshots_session_time" });
db.snapshots.createIndex({ "snapshot_at": 1 }, { expireAfterSeconds: 604800, name: "idx_snapshots_ttl" });

// ---------------------------------------------------------------------------
// 5. gc_logs — garbage collection audit trail
// ---------------------------------------------------------------------------
db.createCollection("gc_logs");

db.gc_logs.createIndex({ "session_id": 1, "created_at": -1 }, { name: "idx_gc_session_time" });
db.gc_logs.createIndex({ "created_at": 1 }, { expireAfterSeconds: 2592000, name: "idx_gc_ttl" });

// ---------------------------------------------------------------------------
// 6. session_metrics — aggregated per-session metrics
// ---------------------------------------------------------------------------
db.createCollection("session_metrics");

db.session_metrics.createIndex({ "session_id": 1 }, { unique: true, name: "idx_metrics_session_unique" });
db.session_metrics.createIndex({ "session_started_at": -1 },     { name: "idx_metrics_started" });

print("ACGC MongoDB initialization complete.");
print("Collections: events, state_nodes, compressed_branches, snapshots, gc_logs, session_metrics");
