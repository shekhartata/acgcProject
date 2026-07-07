package compiler

import (
	"sort"

	"github.com/shekhartata/acgcProject/internal/domain"
)

// StabilizeRenderOrder returns selected nodes sorted for cache-stable rendering.
// Selection is unchanged; only presentation order is deterministic (turn, then ID).
func StabilizeRenderOrder(nodes []*domain.StateNode) []*domain.StateNode {
	if len(nodes) <= 1 {
		return nodes
	}
	out := make([]*domain.StateNode, len(nodes))
	copy(out, nodes)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TurnNumber != out[j].TurnNumber {
			return out[i].TurnNumber < out[j].TurnNumber
		}
		return out[i].NodeID < out[j].NodeID
	})
	return out
}
