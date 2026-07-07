package store

import (
	"context"
	"fmt"
	"time"

	"github.com/shekhartata/acgcProject/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoStore struct {
	client             *mongo.Client
	db                 *mongo.Database
	events             *mongo.Collection
	nodes              *mongo.Collection
	compressedBranches *mongo.Collection
	snapshots          *mongo.Collection
	gcLogs             *mongo.Collection
	sessionMetrics     *mongo.Collection
}

func NewMongoStore(ctx context.Context, uri, dbName string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	// Atlas TLS handshake can take longer than a local connection
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := client.Ping(ctx2, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	db := client.Database(dbName)
	s := &MongoStore{
		client:             client,
		db:                 db,
		events:             db.Collection("events"),
		nodes:              db.Collection("state_nodes"),
		compressedBranches: db.Collection("compressed_branches"),
		snapshots:          db.Collection("snapshots"),
		gcLogs:             db.Collection("gc_logs"),
		sessionMetrics:     db.Collection("session_metrics"),
	}

	if err := s.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("ensure indexes: %w", err)
	}

	return s, nil
}

func (s *MongoStore) ensureIndexes(ctx context.Context) error {
	// Events indexes
	if _, err := s.events.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "sequence", Value: 1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "event_type", Value: 1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "task_id", Value: 1}}},
		{Keys: bson.D{{Key: "event_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: 1}}},
	}); err != nil {
		return fmt.Errorf("events indexes: %w", err)
	}

	// State nodes indexes
	if _, err := s.nodes.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "node_id", Value: 1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "node_type", Value: 1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "parent_id", Value: 1}}},
	}); err != nil {
		return fmt.Errorf("nodes indexes: %w", err)
	}

	// Compressed branches indexes
	if _, err := s.compressedBranches.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "branch_id", Value: 1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "task_id", Value: 1}}},
	}); err != nil {
		return fmt.Errorf("compressed_branches indexes: %w", err)
	}

	// Snapshots indexes
	if _, err := s.snapshots.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "snapshot_at", Value: -1}}},
	}); err != nil {
		return fmt.Errorf("snapshots indexes: %w", err)
	}

	// GC logs indexes
	if _, err := s.gcLogs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "created_at", Value: -1}}},
	}); err != nil {
		return fmt.Errorf("gc_logs indexes: %w", err)
	}

	// Session metrics indexes
	if _, err := s.sessionMetrics.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "session_id", Value: 1}}},
	}); err != nil {
		return fmt.Errorf("session_metrics indexes: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Events — append-only raw archive
// ---------------------------------------------------------------------------

func (s *MongoStore) AppendEvent(ctx context.Context, event *domain.Event) error {
	_, err := s.events.InsertOne(ctx, event)
	return err
}

// AppendEvents inserts a batch of events in one round-trip.
func (s *MongoStore) AppendEvents(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}
	docs := make([]interface{}, len(events))
	for i, e := range events {
		docs[i] = e
	}
	_, err := s.events.InsertMany(ctx, docs)
	return err
}

func (s *MongoStore) GetEvents(ctx context.Context, sessionID string, fromSeq int64, limit int) ([]*domain.Event, error) {
	filter := bson.M{
		"session_id": sessionID,
		"sequence":   bson.M{"$gte": fromSeq},
	}
	opts := options.Find().SetSort(bson.D{{Key: "sequence", Value: 1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := s.events.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*domain.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *MongoStore) GetEventsByType(ctx context.Context, sessionID string, eventType domain.EventType) ([]*domain.Event, error) {
	filter := bson.M{
		"session_id": sessionID,
		"event_type": eventType,
	}
	cursor, err := s.events.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "sequence", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*domain.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// GetEventsByIDs fetches specific events by their event_ids — used by the rehydration engine
// to pull raw payloads when a compressed summary is insufficient.
func (s *MongoStore) GetEventsByIDs(ctx context.Context, eventIDs []string) ([]*domain.Event, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	filter := bson.M{"event_id": bson.M{"$in": eventIDs}}
	cursor, err := s.events.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "sequence", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*domain.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// GetRecentEvents fetches the last N events for a session, ordered newest-first.
func (s *MongoStore) GetRecentEvents(ctx context.Context, sessionID string, limit int) ([]*domain.Event, error) {
	filter := bson.M{"session_id": sessionID}
	opts := options.Find().
		SetSort(bson.D{{Key: "sequence", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := s.events.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*domain.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *MongoStore) CountEvents(ctx context.Context, sessionID string) (int64, error) {
	return s.events.CountDocuments(ctx, bson.M{"session_id": sessionID})
}

// GetEventCountByType returns how many events of each type exist for a session.
func (s *MongoStore) GetEventCountByType(ctx context.Context, sessionID string) (map[string]int64, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"session_id": sessionID}},
		bson.M{"$group": bson.M{
			"_id":   "$event_type",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := s.events.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]int64)
	for cursor.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		result[row.ID] = row.Count
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// State Nodes — durable node persistence (individual upserts)
// ---------------------------------------------------------------------------

// UpsertNode writes or updates a single state tree node durably.
func (s *MongoStore) UpsertNode(ctx context.Context, node *domain.StateNode) error {
	filter := bson.M{
		"session_id": node.SessionID,
		"node_id":    node.NodeID,
	}
	update := bson.M{"$set": node}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := s.nodes.UpdateOne(ctx, filter, update, opts)
	return err
}

// UpsertNodes writes or updates multiple nodes in one batch.
func (s *MongoStore) UpsertNodes(ctx context.Context, nodes []*domain.StateNode) error {
	if len(nodes) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, len(nodes))
	for i, node := range nodes {
		filter := bson.M{
			"session_id": node.SessionID,
			"node_id":    node.NodeID,
		}
		models[i] = mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(bson.M{"$set": node}).
			SetUpsert(true)
	}
	_, err := s.nodes.BulkWrite(ctx, models)
	return err
}

// GetNodesBySession returns all persisted nodes for a session, optionally filtered by status.
func (s *MongoStore) GetNodesBySession(ctx context.Context, sessionID string, statuses ...domain.NodeStatus) ([]*domain.StateNode, error) {
	filter := bson.M{"session_id": sessionID}
	if len(statuses) > 0 {
		statusStrs := make([]string, len(statuses))
		for i, st := range statuses {
			statusStrs[i] = string(st)
		}
		filter["status"] = bson.M{"$in": statusStrs}
	}

	cursor, err := s.nodes.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var nodes []*domain.StateNode
	if err := cursor.All(ctx, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNodesByParent returns all children of a given parent node.
func (s *MongoStore) GetNodesByParent(ctx context.Context, sessionID, parentID string) ([]*domain.StateNode, error) {
	filter := bson.M{
		"session_id": sessionID,
		"parent_id":  parentID,
	}
	cursor, err := s.nodes.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var nodes []*domain.StateNode
	if err := cursor.All(ctx, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// UpdateNodeStatus changes a node's status in the durable store.
func (s *MongoStore) UpdateNodeStatus(ctx context.Context, sessionID, nodeID string, status domain.NodeStatus) error {
	filter := bson.M{
		"session_id": sessionID,
		"node_id":    nodeID,
	}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	_, err := s.nodes.UpdateOne(ctx, filter, update)
	return err
}

// LoadNodeEmbeddings returns the embedding vector for every active node in
// the session that has one persisted. Used to rebuild the in-memory HNSW
// index when a session is rehydrated from a snapshot.
//
// Projected query — pulls only node_id + embedding to keep the payload small
// for sessions with thousands of nodes.
func (s *MongoStore) LoadNodeEmbeddings(ctx context.Context, sessionID string) (map[string][]float32, error) {
	filter := bson.M{
		"session_id": sessionID,
		"embedding":  bson.M{"$exists": true, "$ne": nil},
		"status":     bson.M{"$ne": string(domain.StatusArchived)},
	}
	opts := options.Find().SetProjection(bson.M{"node_id": 1, "embedding": 1})
	cursor, err := s.nodes.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	out := make(map[string][]float32)
	for cursor.Next(ctx) {
		var row struct {
			NodeID    string    `bson:"node_id"`
			Embedding []float32 `bson:"embedding"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		if row.NodeID == "" || len(row.Embedding) == 0 {
			continue
		}
		out[row.NodeID] = row.Embedding
	}
	return out, nil
}

// LoadArchivedNodeEmbeddings returns embeddings for archived nodes only (dual HNSW archive index rebuild).
func (s *MongoStore) LoadArchivedNodeEmbeddings(ctx context.Context, sessionID string) (map[string][]float32, error) {
	filter := bson.M{
		"session_id": sessionID,
		"embedding":  bson.M{"$exists": true, "$ne": nil},
		"status":     string(domain.StatusArchived),
	}
	opts := options.Find().SetProjection(bson.M{"node_id": 1, "embedding": 1})
	cursor, err := s.nodes.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	out := make(map[string][]float32)
	for cursor.Next(ctx) {
		var row struct {
			NodeID    string    `bson:"node_id"`
			Embedding []float32 `bson:"embedding"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		if row.NodeID == "" || len(row.Embedding) == 0 {
			continue
		}
		out[row.NodeID] = row.Embedding
	}
	return out, nil
}

// ArchiveNodes marks multiple nodes as archived in one batch.
func (s *MongoStore) ArchiveNodes(ctx context.Context, sessionID string, nodeIDs []string) error {
	if len(nodeIDs) == 0 {
		return nil
	}
	filter := bson.M{
		"session_id": sessionID,
		"node_id":    bson.M{"$in": nodeIDs},
	}
	update := bson.M{
		"$set": bson.M{
			"status":     domain.StatusArchived,
			"updated_at": time.Now(),
		},
	}
	_, err := s.nodes.UpdateMany(ctx, filter, update)
	return err
}

// ---------------------------------------------------------------------------
// Compressed Branches — durable compressed summaries
// ---------------------------------------------------------------------------

// CompressedBranch is the durable representation of a compressed branch.
type CompressedBranch struct {
	BranchID             string    `bson:"branch_id"`
	SessionID            string    `bson:"session_id"`
	TaskID               string    `bson:"task_id"`
	ParentNodeID         string    `bson:"parent_node_id"`
	OriginalNodeIDs      []string  `bson:"original_node_ids"`
	Summary              string    `bson:"summary"`
	KeyDecisions         []string  `bson:"key_decisions"`
	ExactFacts           []string  `bson:"exact_facts,omitempty"`
	OpenIssues           []string  `bson:"open_issues"`
	ImportantConstraints []string  `bson:"important_constraints"`
	RawEventRefs         []string  `bson:"raw_event_refs"`
	OriginalTokenCount   int       `bson:"original_token_count"`
	CompressedTokenCount int       `bson:"compressed_token_count"`
	CreatedAt            time.Time `bson:"created_at"`
}

func (s *MongoStore) SaveCompressedBranch(ctx context.Context, branch *CompressedBranch) error {
	_, err := s.compressedBranches.InsertOne(ctx, branch)
	return err
}

// GetCompressedBranches returns all compressed branches for a session.
func (s *MongoStore) GetCompressedBranches(ctx context.Context, sessionID string) ([]*CompressedBranch, error) {
	filter := bson.M{"session_id": sessionID}
	cursor, err := s.compressedBranches.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var branches []*CompressedBranch
	if err := cursor.All(ctx, &branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// GetCompressedBranch fetches a single compressed branch by ID — used by rehydration.
func (s *MongoStore) GetCompressedBranch(ctx context.Context, branchID string) (*CompressedBranch, error) {
	var branch CompressedBranch
	err := s.compressedBranches.FindOne(ctx, bson.M{"branch_id": branchID}).Decode(&branch)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &branch, nil
}

// ---------------------------------------------------------------------------
// Snapshots — periodic full state tree snapshots for crash recovery
// ---------------------------------------------------------------------------

type Snapshot struct {
	SessionID  string              `bson:"session_id"`
	Nodes      []*domain.StateNode `bson:"nodes"`
	SnapshotAt time.Time           `bson:"snapshot_at"`
}

func (s *MongoStore) SnapshotNodes(ctx context.Context, sessionID string, nodes []*domain.StateNode) error {
	if len(nodes) == 0 {
		return nil
	}
	snapshot := &Snapshot{
		SessionID:  sessionID,
		Nodes:      nodes,
		SnapshotAt: time.Now(),
	}
	_, err := s.snapshots.InsertOne(ctx, snapshot)
	return err
}

func (s *MongoStore) LoadLatestSnapshot(ctx context.Context, sessionID string) ([]*domain.StateNode, error) {
	filter := bson.M{"session_id": sessionID}
	opts := options.FindOne().SetSort(bson.D{{Key: "snapshot_at", Value: -1}})

	var result Snapshot
	err := s.snapshots.FindOne(ctx, filter, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return result.Nodes, nil
}

// ListSnapshots returns snapshot metadata (without full node data) for a session.
func (s *MongoStore) ListSnapshots(ctx context.Context, sessionID string, limit int) ([]time.Time, error) {
	filter := bson.M{"session_id": sessionID}
	opts := options.Find().
		SetSort(bson.D{{Key: "snapshot_at", Value: -1}}).
		SetProjection(bson.M{"snapshot_at": 1}).
		SetLimit(int64(limit))

	cursor, err := s.snapshots.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var timestamps []time.Time
	for cursor.Next(ctx) {
		var row struct {
			SnapshotAt time.Time `bson:"snapshot_at"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		timestamps = append(timestamps, row.SnapshotAt)
	}
	return timestamps, nil
}

// ---------------------------------------------------------------------------
// GC Logs — garbage collection audit trail
// ---------------------------------------------------------------------------

type GCLog struct {
	SessionID          string    `bson:"session_id"`
	TriggerReason      string    `bson:"trigger_reason"`
	NodesSwept         int       `bson:"nodes_swept"`
	BranchesCompressed int       `bson:"branches_compressed"`
	TokensFreed        int       `bson:"tokens_freed"`
	DurationMs         float64   `bson:"duration_ms"`
	CreatedAt          time.Time `bson:"created_at"`
}

func (s *MongoStore) LogGC(ctx context.Context, log *GCLog) error {
	_, err := s.gcLogs.InsertOne(ctx, log)
	return err
}

// GetGCHistory returns the last N GC events for a session.
func (s *MongoStore) GetGCHistory(ctx context.Context, sessionID string, limit int) ([]*GCLog, error) {
	filter := bson.M{"session_id": sessionID}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := s.gcLogs.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []*GCLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// GetGCSummary returns aggregate GC stats for a session.
func (s *MongoStore) GetGCSummary(ctx context.Context, sessionID string) (totalRuns int64, totalSwept int64, totalFreed int64, err error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"session_id": sessionID}},
		bson.M{"$group": bson.M{
			"_id":         nil,
			"total_runs":  bson.M{"$sum": 1},
			"total_swept": bson.M{"$sum": "$nodes_swept"},
			"total_freed": bson.M{"$sum": "$tokens_freed"},
		}},
	}

	cursor, err := s.gcLogs.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, 0, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var row struct {
			TotalRuns  int64 `bson:"total_runs"`
			TotalSwept int64 `bson:"total_swept"`
			TotalFreed int64 `bson:"total_freed"`
		}
		if err := cursor.Decode(&row); err != nil {
			return 0, 0, 0, err
		}
		return row.TotalRuns, row.TotalSwept, row.TotalFreed, nil
	}
	return 0, 0, 0, nil
}

// ---------------------------------------------------------------------------
// Session Metrics — aggregated per-session metrics
// ---------------------------------------------------------------------------

// UpsertSessionMetrics creates or updates the metrics document for a session.
func (s *MongoStore) UpsertSessionMetrics(ctx context.Context, metrics *domain.SessionMetrics) error {
	filter := bson.M{"session_id": metrics.SessionID}
	update := bson.M{
		"$set": bson.M{
			"total_events":        metrics.TotalEvents,
			"total_turns":         metrics.TotalTurns,
			"gc_runs":             metrics.GCRuns,
			"total_tokens_saved":  metrics.TotalTokensSaved,
			"avg_reduction_pct":   metrics.AvgReductionPct,
			"branches_compressed": metrics.BranchesCompressed,
			"rehydration_events":  metrics.RehydrationEvents,
			"avg_latency_ms":      metrics.AvgLatencyMs,
			"session_started_at":  metrics.SessionStartedAt,
			"last_updated_at":     time.Now(),
		},
	}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := s.sessionMetrics.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetSessionMetrics retrieves persisted metrics for a session.
func (s *MongoStore) GetSessionMetrics(ctx context.Context, sessionID string) (*domain.SessionMetrics, error) {
	var metrics domain.SessionMetrics
	err := s.sessionMetrics.FindOne(ctx, bson.M{"session_id": sessionID}).Decode(&metrics)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &metrics, nil
}

// ListSessions returns session IDs with their start times, ordered most-recent-first.
func (s *MongoStore) ListSessions(ctx context.Context, limit int) ([]*domain.SessionMetrics, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "session_started_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := s.sessionMetrics.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []*domain.SessionMetrics
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// ---------------------------------------------------------------------------
// Rehydration — pull archived data back into active context
// ---------------------------------------------------------------------------

// RehydrateFromArchive fetches raw events referenced by a compressed branch,
// used when the compressed summary is insufficient for the current task.
func (s *MongoStore) RehydrateFromArchive(ctx context.Context, branchID string) ([]*domain.Event, error) {
	branch, err := s.GetCompressedBranch(ctx, branchID)
	if err != nil {
		return nil, fmt.Errorf("load compressed branch: %w", err)
	}
	if branch == nil {
		return nil, nil
	}
	return s.GetEventsByIDs(ctx, branch.RawEventRefs)
}

// RehydrateNodeEvents fetches raw events referenced by a specific state node.
func (s *MongoStore) RehydrateNodeEvents(ctx context.Context, node *domain.StateNode) ([]*domain.Event, error) {
	if len(node.RawEventRefs) == 0 {
		return nil, nil
	}
	return s.GetEventsByIDs(ctx, node.RawEventRefs)
}

// ---------------------------------------------------------------------------
// Token Analytics — aggregate token usage queries
// ---------------------------------------------------------------------------

// GetTokenUsageBySession returns total tokens across all events for a session.
func (s *MongoStore) GetTokenUsageBySession(ctx context.Context, sessionID string) (int64, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"session_id": sessionID}},
		bson.M{"$group": bson.M{
			"_id":          nil,
			"total_tokens": bson.M{"$sum": "$token_count"},
		}},
	}

	cursor, err := s.events.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var row struct {
			TotalTokens int64 `bson:"total_tokens"`
		}
		if err := cursor.Decode(&row); err != nil {
			return 0, err
		}
		return row.TotalTokens, nil
	}
	return 0, nil
}

// GetCompressionStats returns aggregate compression savings for a session.
func (s *MongoStore) GetCompressionStats(ctx context.Context, sessionID string) (totalOriginal, totalCompressed int64, err error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"session_id": sessionID}},
		bson.M{"$group": bson.M{
			"_id":              nil,
			"total_original":   bson.M{"$sum": "$original_token_count"},
			"total_compressed": bson.M{"$sum": "$compressed_token_count"},
		}},
	}

	cursor, err := s.compressedBranches.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var row struct {
			TotalOriginal   int64 `bson:"total_original"`
			TotalCompressed int64 `bson:"total_compressed"`
		}
		if err := cursor.Decode(&row); err != nil {
			return 0, 0, err
		}
		return row.TotalOriginal, row.TotalCompressed, nil
	}
	return 0, 0, nil
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
