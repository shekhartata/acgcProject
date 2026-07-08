package session

import "testing"

func TestEffectiveBudget(t *testing.T) {
	m := &Manager{defaultBudget: 6000}
	state := &SessionState{TokenBudget: 6000}

	if got := m.effectiveBudget(state, 2000); got != 2000 {
		t.Fatalf("request budget = %d, want 2000", got)
	}
	if state.TokenBudget != 2000 {
		t.Fatalf("session TokenBudget = %d, want 2000 persisted", state.TokenBudget)
	}

	if got := m.effectiveBudget(state, 0); got != 2000 {
		t.Fatalf("session budget = %d, want 2000", got)
	}

	state.TokenBudget = 0
	if got := m.effectiveBudget(state, 0); got != 6000 {
		t.Fatalf("fallback budget = %d, want 6000", got)
	}
}

func TestEffectiveBudget_requestOverridesDefault(t *testing.T) {
	m := &Manager{defaultBudget: 6000}
	state := &SessionState{TokenBudget: 6000}

	if got := m.effectiveBudget(state, 8000); got != 8000 {
		t.Fatalf("override budget = %d, want 8000", got)
	}
}
