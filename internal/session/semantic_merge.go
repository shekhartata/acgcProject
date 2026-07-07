package session

import (
	"sort"

	"github.com/shekhartata/acgcProject/internal/domain"
	"github.com/shekhartata/acgcProject/internal/statetree"
	"github.com/shekhartata/acgcProject/internal/vectorindex"
)

// MergeSemanticHits merges two hit lists, keeping the best cosine score per NodeID.
func MergeSemanticHits(hitsA, hitsArchive []vectorindex.Hit) []vectorindex.Hit {
	best := make(map[string]vectorindex.Hit)
	for _, h := range hitsA {
		if prev, ok := best[h.NodeID]; !ok || h.Score > prev.Score {
			best[h.NodeID] = h
		}
	}
	for _, h := range hitsArchive {
		if prev, ok := best[h.NodeID]; !ok || h.Score > prev.Score {
			best[h.NodeID] = h
		}
	}
	out := make([]vectorindex.Hit, 0, len(best))
	for _, h := range best {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// NodesForSemanticCompile joins active compilation nodes with archived nodes that semantic retrieval hit.
func NodesForSemanticCompile(tree *statetree.Tree, active []*domain.StateNode, merged []vectorindex.Hit) []*domain.StateNode {
	seen := make(map[string]bool, len(active)+len(merged))
	out := make([]*domain.StateNode, 0, len(active)+len(merged))
	for _, n := range active {
		if n == nil || seen[n.NodeID] {
			continue
		}
		seen[n.NodeID] = true
		out = append(out, n)
	}
	for _, h := range merged {
		if h.NodeID == "" || seen[h.NodeID] {
			continue
		}
		n, ok := tree.GetNode(h.NodeID)
		if !ok || n.Status != domain.StatusArchived {
			continue
		}
		seen[h.NodeID] = true
		out = append(out, n)
	}
	return out
}
