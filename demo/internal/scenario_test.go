package demo

import (
	"strings"
	"testing"
)

func TestLoadScenario(t *testing.T) {
	sc := LoadScenario()
	if sc.ID == "" {
		t.Fatal("empty scenario id")
	}
	if sc.SeedUntil < 10 {
		t.Fatalf("seed_until = %d, want >= 10", sc.SeedUntil)
	}
	if sc.WarmUserSteps < 3 {
		t.Fatalf("warm user steps = %d, want >= 3", sc.WarmUserSteps)
	}
	if len(sc.Probe.ExpectedAny) == 0 {
		t.Fatal("probe expected_any empty")
	}
	if len(sc.Turns) <= sc.SeedUntil {
		t.Fatalf("turns=%d seed_until=%d: need live warm turns after seed", len(sc.Turns), sc.SeedUntil)
	}
}

func TestHitNeedle(t *testing.T) {
	if !HitNeedle("We chose CockroachDB for inventory.", []string{"CockroachDB", "Cockroach"}) {
		t.Fatal("expected hit")
	}
	if HitNeedle("We used Postgres.", []string{"CockroachDB"}) {
		t.Fatal("expected miss")
	}
}

func TestTakeaway(t *testing.T) {
	if got := Takeaway(true, true); got == "" {
		t.Fatal("empty takeaway")
	}
	if got := Takeaway(false, true); got == "" {
		t.Fatal("empty acgc-win takeaway")
	}
	if got := Takeaway(true, false); !strings.Contains(got, "ACGC missed") {
		t.Fatalf("naive-only takeaway = %q", got)
	}
}
