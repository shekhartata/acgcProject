package latencybench

import (
	"math"
	"sort"
)

// QuantileLinear returns an approximate percentile in [0,1] using linear
// interpolation between closest ranks (common p50/p95/p99 use).
func QuantileLinear(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := float64(len(sorted)-1) * p
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

func SortCopy(xs []float64) []float64 {
	out := make([]float64, len(xs))
	copy(out, xs)
	sort.Float64s(out)
	return out
}
