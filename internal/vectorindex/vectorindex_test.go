package vectorindex

import (
	"fmt"
	"hash/fnv"
	"math"
	"testing"
)

// deterministic mock vec (same algo as stresstest.MockEmbedder, inlined to
// avoid importing the test target's siblings)
func mockVec(text string, dim int) []float32 {
	vec := make([]float32, dim)
	start := -1
	for i := 0; i <= len(text); i++ {
		isSep := i == len(text) || text[i] == ' ' || text[i] == '\n'
		if !isSep {
			if start == -1 {
				start = i
			}
			continue
		}
		if start >= 0 && i > start {
			h := fnv.New32a()
			_, _ = h.Write([]byte(text[start:i]))
			b := int(h.Sum32()) % dim
			if b < 0 {
				b += dim
			}
			vec[b] += 1.0
		}
		start = -1
	}
	var n float64
	for _, v := range vec {
		n += float64(v) * float64(v)
	}
	if n == 0 {
		vec[0] = 1
		return vec
	}
	inv := float32(1.0 / math.Sqrt(n))
	for i := range vec {
		vec[i] *= inv
	}
	return vec
}

func TestMockInsertChain(t *testing.T) {
	idx := NewHNSW(Config{Dim: 128, M: 16, EFSearch: 50})
	texts := []string{
		"hello world", "hello there", "completely different",
		"hello world again", "another sample text",
	}
	for i, txt := range texts {
		v := mockVec(txt, 128)
		if err := idx.Insert(fmt.Sprintf("n%d", i), v); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
		t.Logf("ok n%d, idx.Len=%d", i, idx.Len())
	}
}

func TestMockInsertManySimilar(t *testing.T) {
	idx := NewHNSW(Config{Dim: 128, M: 16, EFSearch: 50})
	// Simulate long_session: 66 turns with overlapping vocab.
	templates := []string{
		"can you help me with %d the project setup",
		"yes here is step %d in the deployment process",
		"what about edge case %d when the user is offline",
		"good question for case %d the system retries",
	}
	for i := 0; i < 66; i++ {
		txt := fmt.Sprintf(templates[i%len(templates)], i)
		v := mockVec(txt, 128)
		if err := idx.Insert(fmt.Sprintf("n%d", i), v); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	t.Logf("done, idx.Len=%d", idx.Len())
}

func TestMockInsertDeleteInterleaved(t *testing.T) {
	// Simulate ACGC's pattern: insert, occasional GC sweep that deletes
	// several IDs, more inserts. This is what crashes ReplaySession.
	idx := NewHNSW(Config{Dim: 128, M: 16, EFSearch: 50})
	templates := []string{
		"can you help me with %d the project setup",
		"yes here is step %d in the deployment process",
		"what about edge case %d when the user is offline",
		"good question for case %d the system retries",
	}
	for i := 0; i < 66; i++ {
		txt := fmt.Sprintf(templates[i%len(templates)], i)
		v := mockVec(txt, 128)
		if err := idx.Insert(fmt.Sprintf("n%d", i), v); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
		// Simulate a GC sweep every 10 turns: drop 3 oldest IDs.
		if i > 0 && i%10 == 0 {
			for d := i - 9; d < i-6; d++ {
				idx.Delete(fmt.Sprintf("n%d", d))
			}
			t.Logf("after gc at turn %d: idx.Len=%d", i, idx.Len())
		}
	}
}

func TestMockInsertEmptyAndShort(t *testing.T) {
	idx := NewHNSW(Config{Dim: 128, M: 16, EFSearch: 50})
	cases := []string{"", " ", "  ", "a", "ab", "ab cd", "ab cd ef"}
	for i, txt := range cases {
		v := mockVec(txt, 128)
		if err := idx.Insert(fmt.Sprintf("e%d", i), v); err != nil {
			t.Fatalf("insert %d (%q): %v", i, txt, err)
		}
		t.Logf("ok %d (%q), idx.Len=%d", i, txt, idx.Len())
	}
}
