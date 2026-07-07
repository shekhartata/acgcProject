package scorer

import (
	"math"
	"strings"

	"github.com/shekhartata/acgcProject/internal/domain"
)

// Weights for the heuristic scoring formula.
type Weights struct {
	Recency         float64
	TypePriority    float64
	DependencyBoost float64
	UnresolvedBoost float64
	Semantic        float64
	RedundancyPen   float64
	ResolvedPen     float64
	StalePen        float64
	SizePen         float64
}

func DefaultWeights() Weights {
	return Weights{
		Recency:         0.25,
		TypePriority:    0.20,
		DependencyBoost: 0.15,
		UnresolvedBoost: 0.15,
		Semantic:        0.20,
		RedundancyPen:   0.10,
		ResolvedPen:     0.20,
		StalePen:        0.15,
		SizePen:         0.05,
	}
}

type Scorer struct {
	weights         Weights
	staleAfterTurns int
	maxTokensPerNode int
}

func NewScorer(staleAfterTurns, maxTokensPerNode int) *Scorer {
	return &Scorer{
		weights:          DefaultWeights(),
		staleAfterTurns:  staleAfterTurns,
		maxTokensPerNode: maxTokensPerNode,
	}
}

// SetSemanticWeight overrides the default semantic weight from config.
// 0 effectively disables the semantic term without disabling the other signals.
func (s *Scorer) SetSemanticWeight(w float64) {
	s.weights.Semantic = w
}

// ScoreAll computes retention scores for every node in the set.
// currentTurn is the latest turn number in the session.
// queryVec is the anchor query embedding for the semantic signal — when nil
// (or when a node has no embedding), the semantic component is treated as 0.
// This keeps existing callers safe and lets the heuristic-only stresstest
// keep working with no API key.
func (s *Scorer) ScoreAll(nodes []*domain.StateNode, currentTurn int64, queryVec []float32) {
	titleIndex := buildTitleIndex(nodes)

	for _, node := range nodes {
		node.Scores = s.scoreNode(node, currentTurn, nodes, titleIndex, queryVec)
	}
}

func (s *Scorer) scoreNode(
	node *domain.StateNode,
	currentTurn int64,
	allNodes []*domain.StateNode,
	titleIndex map[string]int,
	queryVec []float32,
) domain.NodeScores {
	scores := domain.NodeScores{}

	// 1. Recency: exponential decay based on turn distance
	turnDist := float64(currentTurn - node.TurnNumber)
	scores.Recency = math.Exp(-0.1 * turnDist)

	// 2. Type priority: some node types are inherently more important
	scores.TypePriority = typePriority(node.NodeType)

	// 3. Dependency weight: if other active nodes reference this node
	scores.DependencyWeight = dependencyWeight(node, allNodes)

	// 4. Unresolved boost: open issues should be retained
	if len(node.OpenIssues) > 0 && node.Status == domain.StatusActive {
		scores.UnresolvedBoost = 1.0
	}

	// 5. Redundancy penalty: duplicate titles suggest redundant content
	if count, ok := titleIndex[strings.ToLower(node.Title)]; ok && count > 1 {
		scores.Redundancy = math.Min(float64(count-1)*0.3, 1.0)
	}

	// 6. Resolved penalty
	if node.Status == domain.StatusResolved {
		scores.ResolvedPenalty = 1.0
	}

	// 7. Stale penalty
	if turnDist > float64(s.staleAfterTurns) {
		staleFactor := (turnDist - float64(s.staleAfterTurns)) / float64(s.staleAfterTurns)
		scores.StalePenalty = math.Min(staleFactor, 1.0)
	}

	// 8. Size penalty for oversized nodes
	if s.maxTokensPerNode > 0 && node.TokenCount > s.maxTokensPerNode {
		scores.SizePenalty = math.Min(float64(node.TokenCount)/float64(s.maxTokensPerNode)-1.0, 1.0)
	}

	// 9. Semantic relevance to the anchor query. Graceful when either side
	// is missing: dim mismatch or absent embeddings → signal stays at 0,
	// the heuristic 8 still decide retention.
	if queryVec != nil && len(node.Embedding) == len(queryVec) && len(queryVec) > 0 {
		scores.Semantic = CosineSim(queryVec, node.Embedding)
	}

	// Final weighted score
	scores.RetentionScore = clamp(
		s.weights.Recency*scores.Recency+
			s.weights.TypePriority*scores.TypePriority+
			s.weights.DependencyBoost*scores.DependencyWeight+
			s.weights.UnresolvedBoost*scores.UnresolvedBoost+
			s.weights.Semantic*scores.Semantic-
			s.weights.RedundancyPen*scores.Redundancy-
			s.weights.ResolvedPen*scores.ResolvedPenalty-
			s.weights.StalePen*scores.StalePenalty-
			s.weights.SizePen*scores.SizePenalty,
		0.0, 1.0,
	)

	return scores
}

// CosineSim computes cosine similarity between two equal-length vectors.
// Returns a value in [-1, 1], but for normalized text embeddings it is
// effectively [0, 1] in practice. Exposed for compiler re-blending.
func CosineSim(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		x := float64(a[i])
		y := float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func typePriority(nt domain.NodeType) float64 {
	switch nt {
	case domain.NodeGoal:
		return 1.0
	case domain.NodeConstraint:
		return 0.95
	case domain.NodeDecision:
		return 0.80
	case domain.NodeIssue:
		return 0.70
	case domain.NodeToolResult:
		return 0.50
	case domain.NodeAttempt:
		return 0.40
	case domain.NodeSummary, domain.NodeCompressedBranch:
		return 0.60
	case domain.NodeBackground:
		return 0.20
	default:
		return 0.30
	}
}

// dependencyWeight checks how many other active nodes list this node in their dependencies.
func dependencyWeight(node *domain.StateNode, allNodes []*domain.StateNode) float64 {
	count := 0
	for _, other := range allNodes {
		if other.NodeID == node.NodeID || other.Status != domain.StatusActive {
			continue
		}
		for _, dep := range other.Dependencies {
			if dep == node.NodeID {
				count++
				break
			}
		}
	}
	return math.Min(float64(count)*0.3, 1.0)
}

func buildTitleIndex(nodes []*domain.StateNode) map[string]int {
	idx := make(map[string]int, len(nodes))
	for _, n := range nodes {
		key := strings.ToLower(n.Title)
		idx[key]++
	}
	return idx
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
