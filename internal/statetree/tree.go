package statetree

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chandrashekhartata/acgc/internal/domain"
)

// Tree is the in-memory active state tree for a single session.
// It is always accessed by a single session worker goroutine for writes,
// and may be read concurrently by the gateway (via RWMutex).
type Tree struct {
	mu        sync.RWMutex
	sessionID string
	taskID    string
	rootID    string
	nodes     map[string]*domain.StateNode
	turnCount int64
}

func NewTree(sessionID, taskID string) *Tree {
	rootID := fmt.Sprintf("root_%s", sessionID)
	root := &domain.StateNode{
		NodeID:    rootID,
		SessionID: sessionID,
		TaskID:    taskID,
		NodeType:  domain.NodeGoal,
		Status:    domain.StatusActive,
		Title:     "Session Root",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	nodes := map[string]*domain.StateNode{rootID: root}

	return &Tree{
		sessionID: sessionID,
		taskID:    taskID,
		rootID:    rootID,
		nodes:     nodes,
		turnCount: 0,
	}
}

// RestoreFromSnapshot rebuilds the tree from a persisted snapshot.
func RestoreFromSnapshot(sessionID, taskID string, snapshot []*domain.StateNode) *Tree {
	t := &Tree{
		sessionID: sessionID,
		taskID:    taskID,
		nodes:     make(map[string]*domain.StateNode, len(snapshot)),
	}

	var maxTurn int64
	for _, n := range snapshot {
		t.nodes[n.NodeID] = n
		if n.ParentID == "" || n.NodeType == domain.NodeGoal {
			t.rootID = n.NodeID
		}
		if n.TurnNumber > maxTurn {
			maxTurn = n.TurnNumber
		}
	}
	t.turnCount = maxTurn
	return t
}

func (t *Tree) SessionID() string {
	return t.sessionID
}

func (t *Tree) IncrementTurn() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.turnCount++
	return t.turnCount
}

func (t *Tree) TurnCount() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.turnCount
}

// AddNode classifies an event and inserts a new node into the tree.
func (t *Tree) AddNode(event *domain.Event) *domain.StateNode {
	t.mu.Lock()
	defer t.mu.Unlock()

	nodeType := classifyEvent(event)
	parentID := t.findBestParent(nodeType)

	node := &domain.StateNode{
		NodeID:       fmt.Sprintf("node_%s_%d", t.sessionID, t.turnCount),
		ParentID:     parentID,
		SessionID:    t.sessionID,
		TaskID:       t.taskID,
		NodeType:     nodeType,
		Status:       domain.StatusActive,
		Title:        buildTitle(event),
		Summary:      truncate(event.Payload, 500),
		RawEventRefs: []string{event.EventID},
		TokenCount:   event.TokenCount,
		TurnNumber:   t.turnCount,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	t.nodes[node.NodeID] = node
	if parent, ok := t.nodes[parentID]; ok {
		parent.ChildIDs = append(parent.ChildIDs, node.NodeID)
		parent.UpdatedAt = time.Now()
	}

	return node
}

// GetActiveNodes returns all nodes with status active (read-safe for prompt compiler).
func (t *Tree) GetActiveNodes() []*domain.StateNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var active []*domain.StateNode
	for _, n := range t.nodes {
		if n.Status == domain.StatusActive {
			active = append(active, n)
		}
	}
	return active
}

// GetAllNodes returns every node for snapshotting.
func (t *Tree) GetAllNodes() []*domain.StateNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	all := make([]*domain.StateNode, 0, len(t.nodes))
	for _, n := range t.nodes {
		all = append(all, n)
	}
	return all
}

func (t *Tree) GetNode(nodeID string) (*domain.StateNode, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	n, ok := t.nodes[nodeID]
	return n, ok
}

// UpdateNodeStatus changes the status of a node (used by GC).
func (t *Tree) UpdateNodeStatus(nodeID string, status domain.NodeStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n, ok := t.nodes[nodeID]; ok {
		n.Status = status
		n.UpdatedAt = time.Now()
	}
}

// ReplaceWithCompressed replaces a set of child nodes with a single compressed branch node.
func (t *Tree) ReplaceWithCompressed(parentID string, childIDs []string, compressed *domain.StateNode) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, cid := range childIDs {
		if n, ok := t.nodes[cid]; ok {
			n.Status = domain.StatusArchived
		}
	}

	t.nodes[compressed.NodeID] = compressed

	if parent, ok := t.nodes[parentID]; ok {
		remaining := make([]string, 0)
		childSet := make(map[string]bool, len(childIDs))
		for _, id := range childIDs {
			childSet[id] = true
		}
		for _, id := range parent.ChildIDs {
			if !childSet[id] {
				remaining = append(remaining, id)
			}
		}
		remaining = append(remaining, compressed.NodeID)
		parent.ChildIDs = remaining
		parent.UpdatedAt = time.Now()
	}
}

// Stats returns aggregate statistics about the tree.
func (t *Tree) Stats() (total, active, compressed, archived, maxDepth, maxWidth int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, n := range t.nodes {
		total++
		switch n.Status {
		case domain.StatusActive:
			active++
		case domain.StatusCompressed:
			compressed++
		case domain.StatusArchived:
			archived++
		}
		w := len(n.ChildIDs)
		if w > maxWidth {
			maxWidth = w
		}
	}
	maxDepth = t.depthFrom(t.rootID, 0)
	return
}

func (t *Tree) depthFrom(nodeID string, current int) int {
	node, ok := t.nodes[nodeID]
	if !ok {
		return current
	}
	max := current
	for _, cid := range node.ChildIDs {
		d := t.depthFrom(cid, current+1)
		if d > max {
			max = d
		}
	}
	return max
}

func (t *Tree) findBestParent(nodeType domain.NodeType) string {
	// Goals and constraints attach to root.
	if nodeType == domain.NodeGoal || nodeType == domain.NodeConstraint {
		return t.rootID
	}
	// Everything else attaches to the most recent active non-leaf or root.
	var best *domain.StateNode
	for _, n := range t.nodes {
		if n.Status != domain.StatusActive {
			continue
		}
		if best == nil || n.TurnNumber > best.TurnNumber {
			best = n
		}
	}
	if best != nil {
		return best.NodeID
	}
	return t.rootID
}

func classifyEvent(event *domain.Event) domain.NodeType {
	switch event.EventType {
	case domain.EventUserPrompt:
		lower := strings.ToLower(event.Payload)
		if containsAny(lower, []string{"must", "always", "never", "require", "constraint"}) {
			return domain.NodeConstraint
		}
		return domain.NodeGoal
	case domain.EventToolCall:
		return domain.NodeAttempt
	case domain.EventToolResult:
		return domain.NodeToolResult
	case domain.EventLLMResponse:
		return domain.NodeDecision
	case domain.EventError, domain.EventRetry:
		return domain.NodeIssue
	default:
		return domain.NodeBackground
	}
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func buildTitle(event *domain.Event) string {
	prefix := string(event.EventType)
	payload := truncate(event.Payload, 80)
	return fmt.Sprintf("[%s] %s", prefix, payload)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
