package demo

import "testing"

func TestLoadScenario(t *testing.T) {
	sc := LoadScenario()
	if sc.ID == "" {
		t.Fatal("empty scenario id")
	}
	if sc.WarmUserSteps < 3 {
		t.Fatalf("warm user steps = %d, want >= 3", sc.WarmUserSteps)
	}
	if len(sc.Probe.ExpectedAny) == 0 {
		t.Fatal("probe expected_any empty")
	}
	// First 10 turns are decision block.
	if len(sc.Turns) < 14 {
		t.Fatalf("turns = %d, want curated slice", len(sc.Turns))
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
}
