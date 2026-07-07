package compiler

import (
	"testing"

	"github.com/shekhartata/acgcProject/internal/domain"
)

func TestStabilizeRenderOrder_deterministic(t *testing.T) {
	nodes := []*domain.StateNode{
		{NodeID: "c", TurnNumber: 2, Summary: "third"},
		{NodeID: "a", TurnNumber: 1, Summary: "first"},
		{NodeID: "b", TurnNumber: 1, Summary: "second"},
	}
	got := StabilizeRenderOrder(nodes)
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	want := []string{"a", "b", "c"}
	for i, id := range want {
		if got[i].NodeID != id {
			t.Fatalf("index %d: got %q want %q", i, got[i].NodeID, id)
		}
	}
}

func TestCompile_cacheStableRender_sameTokens(t *testing.T) {
	nodes := testNodesForCacheRender()
	base := NewCompiler(6000).Compile("s", "t", "question?", nodes, "sys")
	stable := NewCompiler(6000).WithCacheStableRender(true).Compile("s", "t", "question?", nodes, "sys")

	if base.CompiledTokenCount != stable.CompiledTokenCount {
		t.Fatalf("token count: base=%d stable=%d", base.CompiledTokenCount, stable.CompiledTokenCount)
	}
	if len(base.ExcludedNodeRefs) != len(stable.ExcludedNodeRefs) {
		t.Fatalf("excluded count: base=%d stable=%d", len(base.ExcludedNodeRefs), len(stable.ExcludedNodeRefs))
	}
	exBase := map[string]bool{}
	for _, id := range base.ExcludedNodeRefs {
		exBase[id] = true
	}
	for _, id := range stable.ExcludedNodeRefs {
		if !exBase[id] {
			t.Fatalf("stable excluded %q not in base set", id)
		}
	}
}

func TestCompile_cacheStableRender_differentBytes(t *testing.T) {
	nodes := testNodesForCacheRender()
	base := NewCompiler(6000).Compile("s", "t", "question?", nodes, "sys")
	stable := NewCompiler(6000).WithCacheStableRender(true).Compile("s", "t", "question?", nodes, "sys")

	if base.FinalPrompt == stable.FinalPrompt {
		t.Fatal("expected FinalPrompt to differ when score order != turn order")
	}
	if !stable.CacheStableRender {
		t.Fatal("expected CacheStableRender=true on compiled prompt")
	}
}

func testNodesForCacheRender() []*domain.StateNode {
	return []*domain.StateNode{
		{
			NodeID: "n-old", NodeType: domain.NodeToolResult, TurnNumber: 1,
			Summary: "older tool output with more detail",
			TokenCount: 20,
			Scores:     domain.NodeScores{RetentionScore: 0.2},
		},
		{
			NodeID: "n-new", NodeType: domain.NodeToolResult, TurnNumber: 5,
			Summary: "newer tool output with more detail",
			TokenCount: 20,
			Scores:     domain.NodeScores{RetentionScore: 0.9},
		},
	}
}
