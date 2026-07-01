package datasets

// ---------------------------------------------------------------------------
// Scenario 7: Deep-history recall — designed to exceed the token budget.
//
// The existing scenarios all fit inside the default 6000-token compiler
// budget, so budget-driven truncation (naive keeps the oldest turns, sliding
// keeps the newest) never actually fires and the strategies never diverge.
//
// This scenario states four decisions up front, then buries them under ~85
// large filler Q/A pairs (~13-15k raw tokens, well past 2x the budget). At that
// size the three strategies diverge sharply:
//   - naive_full_history: fills the budget from the OLDEST turn, so it should
//     still contain the early decisions (tail filler is dropped).
//   - sliding_window: fills from the NEWEST turn, so the early decisions fall
//     off the window entirely — expect recall failures.
//   - acgc: retention-scores + compresses, keeping the decisions at a fraction
//     of the token cost.
// ---------------------------------------------------------------------------

// bigFillerPair returns a Q/A pair with a long, realistic-but-irrelevant answer
// (~180-220 tokens combined). Used to bulk conversations well past the token
// budget without needing an unreasonable number of turns.
func bigFillerPair(topic string, i int) []Turn {
	questions := []string{
		"Walk me through how you'd approach " + topic + " for a system at our scale.",
		"What are the main failure modes we should design around for " + topic + "?",
		"How would you structure the rollout plan for " + topic + " across environments?",
		"What operational metrics and alerts matter most for " + topic + "?",
		"How do we keep " + topic + " maintainable as the team grows?",
		"What are the cost trade-offs we should weigh with " + topic + "?",
		"How should we test " + topic + " so regressions get caught early?",
		"What's a sensible phased migration path for " + topic + "?",
	}
	answers := []string{
		"For " + topic + " at your scale, start by writing down the invariants you can never violate, then design the happy path around them. Keep the first version boring: a single well-understood component, synchronous where you can afford it, with explicit timeouts and bounded retries. Instrument every boundary from day one so you can see latency and error rates per dependency. Resist the urge to shard or cache until a profile proves you need to — most teams add complexity here that they never recoup. Once the baseline is stable, introduce back-pressure and graceful degradation so a slow dependency sheds load instead of collapsing the whole path. Document the decision and the rejected alternatives so the next engineer understands the why, not just the what.",
		"The failure modes for " + topic + " usually cluster into three buckets: resource exhaustion, partial failure, and correctness drift. Resource exhaustion is memory, file descriptors, and connection pools — set hard limits and fail fast rather than degrade silently. Partial failure is the nasty one: a downstream returns slowly or half-succeeds, so use idempotency keys, timeouts shorter than the caller's, and circuit breakers to contain blast radius. Correctness drift creeps in when retries duplicate work or when eventual consistency surprises a caller expecting read-your-writes. Make the consistency model explicit at the API boundary and add reconciliation jobs that detect and repair divergence. Alert on the leading indicators — queue depth, retry rate, saturation — not just on the outage itself.",
		"A safe rollout for " + topic + " goes environment by environment behind a feature flag. Land it dark in staging first with synthetic traffic that mirrors production shape, not just volume. Then enable it for internal users, then a 1% canary, watching error budget burn rate rather than raw error counts. Keep the old path runnable in parallel so rollback is a config flip, not a redeploy. Bake in a kill switch and rehearse using it before you need it. Only remove the legacy code once the new path has held a full peak cycle — usually a week including your busiest day — with no regressions. Write the rollback runbook before the rollout, not during the incident.",
		"For " + topic + " the metrics that matter are the golden signals: latency (p50/p95/p99, not averages), traffic, errors, and saturation. Add domain-specific counters for the operations that carry business meaning, and always expose a saturation metric so you can see how close you are to a limit before you hit it. Alert on symptoms users feel — elevated p99 or error rate — and use the resource metrics for diagnosis, not paging. Set SLOs and page on burn rate so a slow leak wakes someone before it becomes an outage. Keep dashboards sparse: a handful of high-signal graphs beats fifty that nobody reads during an incident.",
		"Keeping " + topic + " maintainable as the team grows is mostly about boundaries and conventions. Draw clear module ownership so changes stay local and code review has an obvious reviewer. Prefer explicit interfaces over shared mutable state, and keep the dependency graph acyclic so people can reason about a subsystem in isolation. Invest early in fast tests and a one-command local setup — friction here compounds across every new hire. Write down the non-obvious decisions as short ADRs so tribal knowledge survives turnover. Automate the boring stuff (formatting, linting, releases) so humans spend their attention on design, not ceremony.",
		"The cost trade-offs for " + topic + " come down to when you pay: build time, run time, or operational time. Managed services cost more per unit but save operational time and reduce the failure surface you own — usually worth it until scale makes the markup painful. Right-size before you optimize: most bills are dominated by idle headroom and over-provisioned instances, not by a clever algorithm. Use spot or preemptible capacity for anything interruptible, tier storage by access frequency, and put a budget alert on every account. Remember that engineering time is often the most expensive line item, so don't spend a week shaving dollars off a bill that a config change would fix.",
		"To test " + topic + " so regressions surface early, build a pyramid: many fast unit tests for the logic, a smaller set of integration tests at the real boundaries, and a thin layer of end-to-end smoke tests in CI. Make the tests deterministic — control time, randomness, and external calls behind seams you can stub. Add property-based tests for anything with tricky invariants, and golden-file tests for output formats that must stay stable. Run the fast tests on every commit and gate merges on them; run the slow suite on a schedule. When a bug escapes, add the failing case as a regression test before you fix it so it can never come back silently.",
		"A phased migration for " + topic + " starts with the strangler pattern: stand the new path up alongside the old one and route a trickle of traffic through it. Keep both writing to a shared source of truth during the transition, with a reconciliation job proving they agree. Migrate reads before writes where you can, since reads are easier to roll back. Move one cohort at a time — by tenant, region, or feature — and hold each cohort long enough to observe a full traffic cycle. Only decommission the legacy path after a bake period with clean metrics, and keep the data export around long enough that a late-discovered bug is recoverable.",
	}
	q := questions[i%len(questions)]
	an := answers[i%len(answers)]
	return []Turn{u(q), a(an)}
}

func deepHistoryRecall() Scenario {
	turns := []Turn{
		u("New project kickoff: a warehouse inventory and order-routing platform. I'll state four architectural decisions up front and I want you to hold onto them for the whole session."),
		a("Great — I'll remember all four. Go ahead and state them and I'll confirm each one."),
		u("Decision 1: our primary datastore will be CockroachDB, not Cassandra. We need distributed SQL with strong consistency for inventory counts across regions."),
		a("Confirmed: CockroachDB is the primary datastore. Good fit — it gives you distributed ACID transactions and SQL semantics, so inventory decrements stay correct across regions without the eventual-consistency footguns you'd hit with Cassandra."),
		u("Decision 2: order events flow through RabbitMQ, not Kafka. Our throughput is modest and we value per-message routing and simpler operations."),
		a("Confirmed: RabbitMQ for order events. Sensible at your scale — flexible topic/exchange routing, mature management tooling, and far less operational weight than a Kafka cluster when you don't need its log-replay throughput."),
		u("Decision 3: we deploy on Google Cloud in the europe-west4 region, not AWS. Data-residency requirements for EU customers drive this."),
		a("Confirmed: Google Cloud, europe-west4. That keeps customer data in the EU for residency compliance and gives you managed networking and IAM to lean on for the rest of the stack."),
		u("Decision 4: all physical labels use the GS1 DataMatrix barcode standard, not plain Code-128. We need 2D codes for dense SKU plus lot/expiry data."),
		a("Confirmed: GS1 DataMatrix for labels. Right call for warehousing — 2D DataMatrix packs GTIN, lot, and expiry into a small footprint and scans reliably at speed on conveyor lines, which linear Code-128 can't match."),
	}

	// Bury the decisions under ~55 large filler turns across rotating topics.
	fillerTopics := []string{
		"the picking-route optimizer",
		"the label-print service",
		"the reservation ledger",
		"the returns-intake workflow",
		"the carrier-integration layer",
		"the stock-reconciliation job",
		"the demand-forecasting model",
	}
	for i := 0; i < 85; i++ {
		topic := fillerTopics[i%len(fillerTopics)]
		turns = append(turns, bigFillerPair(topic, i)...)
	}

	return Scenario{
		ID:          "deep_history_recall_1",
		Name:        "Deep-history recall — beyond the token budget",
		Category:    "deep_history",
		Description: "Four decisions stated up front, then ~85 large filler Q/A pairs (~13-15k raw tokens, >2x the default budget). Forces naive/sliding/acgc to diverge on which context survives.",
		Turns:       turns,
		Probes: []Probe{
			{
				ID:          "p1",
				ProbeAt:     len(turns),
				Question:    "Way back at kickoff — which primary datastore did we commit to for this platform?",
				MatchType:   MatchContainsAny,
				ExpectedAny: []string{"CockroachDB", "Cockroach"},
				Notes:       "Decision 1, buried under >2x-budget of filler. Sliding window should drop it; naive/acgc should keep it.",
			},
			{
				ID:          "p2",
				ProbeAt:     len(turns),
				Question:    "And which message broker did we choose for order events?",
				MatchType:   MatchContainsAny,
				ExpectedAny: []string{"RabbitMQ", "Rabbit"},
				Notes:       "Decision 2, same deep-recall test with a different fact.",
			},
			{
				ID:          "p3",
				ProbeAt:     len(turns),
				Question:    "Remind me which cloud provider and region we settled on, and why.",
				MatchType:   MatchContainsAny,
				ExpectedAny: []string{"Google Cloud", "GCP", "europe-west4"},
				Notes:       "Decision 3. Data-residency rationale lives in the buried turn.",
			},
			{
				ID:          "p4",
				ProbeAt:     len(turns),
				Question:    "What barcode standard did we standardize on for physical labels?",
				MatchType:   MatchContainsAny,
				ExpectedAny: []string{"DataMatrix", "GS1"},
				Notes:       "Decision 4, the earliest-buried fact after the most filler.",
			},
		},
	}
}
