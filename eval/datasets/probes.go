package datasets

// All returns every built-in scenario across all categories.
func All() []Scenario {
	return []Scenario{
		recentRecall(),
		longRangeRecall(),
		constraintAdherence(),
		topicSwitchAndReturn(),
		contradictionHandling(),
		multiHopSynthesis(),
	}
}

// ByID returns a single scenario by its ID (or nil).
func ByID(id string) *Scenario {
	for _, s := range All() {
		if s.ID == id {
			return &s
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers for building filler turns. Filler is realistic but irrelevant noise
// designed to push earlier facts past the ACGC staleness threshold.
// ---------------------------------------------------------------------------

func u(content string) Turn { return Turn{Role: "user", Content: content} }
func a(content string) Turn { return Turn{Role: "assistant", Content: content} }

// fillerPair returns a Q/A pair on a generic side topic. Used to bulk up
// conversations and trigger ACGC's GC.
func fillerPair(topic string, i int) []Turn {
	questions := []string{
		"Can you explain " + topic + " in more detail?",
		"What are best practices for " + topic + "?",
		"Are there common pitfalls with " + topic + "?",
		"How does " + topic + " compare to alternatives?",
		"What testing approach do you recommend for " + topic + "?",
		"What about scaling concerns with " + topic + "?",
		"How do I monitor " + topic + " in production?",
		"What are the security implications of " + topic + "?",
	}
	answers := []string{
		topic + " has several aspects worth considering. The key is to start simple and add complexity only when needed. Most teams over-engineer this early.",
		"Best practices for " + topic + ": follow established conventions, write tests, document decisions, and review with your team. The community has converged on reasonable defaults.",
		"Common pitfalls with " + topic + " include: premature optimization, ignoring error cases, and not measuring before optimizing. Most issues stem from skipping fundamentals.",
		topic + " compared to alternatives: each tool has trade-offs. The right choice depends on your team's expertise, scale requirements, and operational constraints.",
		"For testing " + topic + ", combine unit tests for logic, integration tests for boundaries, and smoke tests in CI. Aim for fast feedback loops.",
		"Scaling " + topic + " usually means: horizontal scaling first, profile before optimizing, cache aggressively, and consider read/write split for data layers.",
		"Monitoring " + topic + ": expose Prometheus metrics, structured logs, and tracing. Set SLOs on the critical paths and alert on burn rate, not raw thresholds.",
		"Security for " + topic + ": apply least privilege, validate all inputs at boundaries, audit dependencies regularly, and rotate secrets via your secret manager.",
	}
	q := questions[i%len(questions)]
	an := answers[i%len(answers)]
	return []Turn{u(q), a(an)}
}

// ---------------------------------------------------------------------------
// Scenario 1: Recent Recall (sanity check)
// ---------------------------------------------------------------------------

func recentRecall() Scenario {
	turns := []Turn{
		u("I'm building a SaaS billing system. What database should I use?"),
		a("For a SaaS billing system, PostgreSQL is the standard choice. It has strong ACID guarantees which are critical for financial data, mature support for transactions across multiple tables, and excellent decimal/numeric types that avoid floating-point errors when handling money. Stay away from MongoDB for billing — eventual consistency is wrong for invoices."),
		u("Great. What's the recommended way to store monetary amounts?"),
		a("Always store money as integers in the smallest unit of the currency (cents for USD, paise for INR). Never use floats — they cause rounding errors that compound over time. In PostgreSQL, use BIGINT for the amount column and a separate VARCHAR(3) column for the ISO currency code. Convert to display format only at the presentation layer."),
		u("Okay. What about handling invoice numbers — should they be sequential or random?"),
		a("Use sequential invoice numbers for compliance — many jurisdictions require gap-free sequential numbering for tax purposes. Implement this with a dedicated PostgreSQL sequence per organization (not the global SERIAL), and assign the number inside the same transaction that creates the invoice. This guarantees no gaps even under concurrent inserts."),
	}

	return Scenario{
		ID:          "recent_recall_1",
		Name:        "Recent recall — billing basics",
		Category:    "recent_recall",
		Description: "3 short turns, then probe about the most recent decision. Both pipelines should ace this.",
		Turns:       turns,
		Probes: []Probe{
			{
				ID:          "p1",
				ProbeAt:     len(turns),
				Question:    "Quick reminder — should invoice numbers be sequential or random, and why?",
				MatchType:   MatchContainsAll,
				ExpectedAny: []string{"sequential", "compliance"},
				Notes:       "Last decision in conversation. ACGC must preserve recent turns.",
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: Long-range Recall — the real test
// ---------------------------------------------------------------------------

func longRangeRecall() Scenario {
	turns := []Turn{
		u("I'm starting a new project — a real-time analytics dashboard for IoT sensors. I want you to remember three key decisions we'll make."),
		a("Sounds good. I'll help you make these decisions and remember them. What's the first?"),
		u("Decision 1: We'll use TimescaleDB as our primary database, not InfluxDB. The reason is we need relational queries alongside time-series data."),
		a("Noted: TimescaleDB is the chosen database. This is a solid pick — TimescaleDB gives you full PostgreSQL compatibility (joins, complex WHERE clauses, foreign keys) plus optimized time-series storage via hypertables. InfluxDB would be faster for pure time-series writes but you'd lose relational capabilities."),
		u("Decision 2: For the ingestion pipeline, we'll use NATS JetStream, not Kafka. Reasoning: simpler ops, similar throughput at our scale."),
		a("Noted: NATS JetStream for ingestion. Good choice for your scale. JetStream gives you durable streams without ZooKeeper, simpler clustering, and lower memory footprint than Kafka. It supports message replay and consumer groups so you keep most of Kafka's useful features."),
		u("Decision 3: We'll deploy on Hetzner Cloud, not AWS. Cost is the primary driver — we expect to be IO-heavy."),
		a("Noted: Hetzner Cloud as the deployment target. Smart for IO-heavy workloads — Hetzner's dedicated NVMe storage performs better than equivalent AWS instances at roughly 1/5th the cost. The trade-off is fewer managed services, but for your stack (self-hosted DB and message bus) that's not a major concern."),
	}

	for i := 0; i < 16; i++ {
		turns = append(turns, fillerPair("authentication", i)...)
	}

	return Scenario{
		ID:          "long_range_recall_1",
		Name:        "Long-range recall — three decisions",
		Category:    "long_range",
		Description: "Three explicit decisions early, then 16 noise turns, then probe each decision. The hardest test for ACGC's heuristic scorer.",
		Turns:       turns,
		Probes: []Probe{
			{
				ID:          "p1",
				ProbeAt:     len(turns),
				Question:    "Quick recap: which database did we agree on for the analytics dashboard?",
				MatchType:   MatchContainsAny,
				ExpectedAny: []string{"TimescaleDB", "Timescale"},
				Notes:       "Decision from turn 3. Tests retention across ~38 turns of noise.",
			},
			{
				ID:          "p2",
				ProbeAt:     len(turns),
				Question:    "And which message bus did we pick for ingestion?",
				MatchType:   MatchContainsAny,
				ExpectedAny: []string{"NATS", "JetStream"},
				Notes:       "Decision from turn 5. Same retention test, different fact.",
			},
			{
				ID:          "p3",
				ProbeAt:     len(turns),
				Question:    "Remind me where we decided to deploy this — which cloud provider?",
				MatchType:   MatchContainsAny,
				ExpectedAny: []string{"Hetzner"},
				Notes:       "Decision from turn 7. Probes the third long-range fact.",
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Scenario 3: Constraint Adherence
// ---------------------------------------------------------------------------

func constraintAdherence() Scenario {
	turns := []Turn{
		u("Important constraints for this project — I want you to remember these throughout: must use Go, never Python. Must use MongoDB, never PostgreSQL. Must avoid Redis — we have hard requirements on no Redis at all in this stack."),
		a("Understood. Constraints locked in: language is Go (no Python), database is MongoDB (no PostgreSQL), no Redis allowed in any component. I'll respect these throughout our session."),
		u("Now, what's a good way to structure a microservice project layout?"),
		a("For Go microservices: use a standard layout with cmd/ for binaries, internal/ for private packages, pkg/ for public SDKs (if any), and api/ for API contracts (proto, OpenAPI). Each service gets its own go.mod for independent versioning. Configuration via environment variables, structured logging via slog, graceful shutdown via signal handlers."),
	}

	for i := 0; i < 13; i++ {
		turns = append(turns, fillerPair("CI/CD", i)...)
	}

	return Scenario{
		ID:          "constraint_adherence_1",
		Name:        "Constraint adherence under pressure",
		Category:    "constraint",
		Description: "Constraints stated upfront, then heavy filler, then a request that tempts the model to suggest forbidden tech.",
		Turns:       turns,
		Probes: []Probe{
			{
				ID:          "p1",
				ProbeAt:     len(turns),
				Question:    "Now write me a simple in-memory cache implementation for one of my services. Pick whatever language and supporting tools you think are best for this project — full freedom.",
				MatchType:   MatchJudge,
				JudgeRubric: "The response should use Go (not Python, Java, etc.) and must NOT recommend Redis. Penalize heavily if Python or Redis is suggested. Award full marks only if the answer respects all three project constraints (Go, MongoDB, no Redis) — though MongoDB doesn't apply to a cache, the answer should at minimum not violate language and Redis constraints.",
				Notes:       "Tests whether ACGC retains the constraint nodes (they should get high type-priority protection).",
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Scenario 4: Topic Switch and Return
// ---------------------------------------------------------------------------

func topicSwitchAndReturn() Scenario {
	turns := []Turn{
		u("I'm building a recipe recommendation system. The core algorithm should suggest recipes based on a user's pantry contents."),
		a("A pantry-based recommender works well as a set-coverage problem: rank recipes by what % of their ingredients the user already has. For better results, weight by ingredient cost (avoid suggesting recipes that need expensive missing ingredients) and personal preference history."),
		u("I want to use TF-IDF on ingredient lists to find similar recipes — does that make sense?"),
		a("TF-IDF on ingredients is a reasonable baseline. Treat each recipe as a document, ingredients as terms. Rare ingredients (saffron) get high IDF, common ones (salt) get low IDF. Compute cosine similarity between recipes. Combine with the pantry match score as a weighted ensemble."),
		u("Got it. What scoring formula should I use for the final recommendation rank?"),
		a("Final score = 0.6 × pantry_coverage + 0.3 × tfidf_similarity_to_liked_recipes + 0.1 × recency_boost. The pantry coverage dominates because it's the most actionable signal. The TF-IDF term lets you generalize to similar recipes. The recency boost surfaces seasonal/trending items."),
	}

	turns = append(turns,
		u("Okay let me put recipes aside for a bit. Help me debug a separate issue — my Kubernetes deployment keeps crashing with OOMKilled."),
		a("OOMKilled means a container exceeded its memory limit. Steps to debug: (1) `kubectl describe pod` to confirm the reason, (2) check `kubectl top pod` history if metrics-server is installed, (3) review your container's memory usage pattern — Java/Node tend to use heap aggressively, (4) either raise the memory limit OR fix the leak. Set both requests and limits; mismatches cause scheduling issues."),
	)
	for i := 0; i < 8; i++ {
		turns = append(turns, fillerPair("Kubernetes", i)...)
	}

	return Scenario{
		ID:          "topic_switch_return_1",
		Name:        "Topic switch then return",
		Category:    "topic_switch",
		Description: "Recipe system discussion, hard pivot to Kubernetes for 9 turns, then a callback to the recipe scoring formula. Exposes heuristic-scorer weakness.",
		Turns:       turns,
		Probes: []Probe{
			{
				ID:          "p1",
				ProbeAt:     len(turns),
				Question:    "Okay back to recipes — remind me of the exact scoring formula we agreed on for ranking recommendations.",
				MatchType:   MatchContainsAll,
				ExpectedAny: []string{"0.6", "0.3", "0.1"},
				Notes:       "Specific decision (the weights) from a topic that ACGC's recency scoring will deprioritize after the K8s detour.",
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Scenario 5: Contradiction Handling
// ---------------------------------------------------------------------------

func contradictionHandling() Scenario {
	turns := []Turn{
		u("We need to pick a session storage strategy for the web app. Let's go with JWT tokens stored in localStorage."),
		a("JWT in localStorage is the simplest approach: stateless, no server-side session store needed, works well with SPAs. Caveat: vulnerable to XSS if you don't sanitize inputs religiously. Token lifetime should be short (~15 min) with a refresh token rotation."),
		u("Sounds good. What expiry should we use?"),
		a("Access tokens: 15 minutes. Refresh tokens: 7 days, rotated on every use. Store the refresh token in an HttpOnly cookie to mitigate XSS. The access token in localStorage is acceptable given the short lifetime — even if leaked, the window of exploitation is small."),
	}

	for i := 0; i < 6; i++ {
		turns = append(turns, fillerPair("frontend state management", i)...)
	}

	turns = append(turns,
		u("Actually, I've changed my mind on the auth approach. Forget localStorage — let's switch to httpOnly cookies with server-side sessions stored in our database. Security team raised concerns about XSS, so we're going stateful."),
		a("Got it — switching to server-side sessions in the database, accessed via httpOnly cookies. This is more secure: cookies are not accessible to JavaScript, eliminating XSS-based token theft. You'll need a sessions table (id, user_id, expires_at, last_seen_at, ip_hash). Issue a long-lived opaque session ID (not a JWT) and revoke server-side on logout. CSRF protection becomes mandatory now since cookies are sent automatically — use SameSite=Lax and a CSRF token for state-changing requests."),
	)

	for i := 0; i < 7; i++ {
		turns = append(turns, fillerPair("database indexing", i)...)
	}

	return Scenario{
		ID:          "contradiction_1",
		Name:        "Contradiction — auth approach reversed",
		Category:    "contradiction",
		Description: "Initial decision (JWT/localStorage) gets explicitly reversed mid-conversation. Probe asks for the current approach. Latest decision must win.",
		Turns:       turns,
		Probes: []Probe{
			{
				ID:          "p1",
				ProbeAt:     len(turns),
				Question:    "Where are we storing user sessions, and what mechanism are we using to identify them?",
				MatchType:   MatchJudge,
				JudgeRubric: "Correct answer must describe SERVER-SIDE sessions (database-stored) with HTTPONLY COOKIES carrying the session ID. WRONG if it says JWT in localStorage — that was the reversed decision. Award 5/5 only if the response correctly identifies the current state (post-reversal) and ideally mentions CSRF protection or SameSite. Award 0-1 if it gives the obsolete JWT/localStorage answer.",
				Notes:       "Critical test — the model must use the most recent decision, not the original one.",
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Scenario 6: Multi-hop Synthesis
// ---------------------------------------------------------------------------

func multiHopSynthesis() Scenario {
	turns := []Turn{
		u("Quick context for the system I'm building: it's a video transcoding service. The frontend uploads a file, we process it, and notify the user when done. Three facts you should remember:"),
		a("Got it — go ahead."),
		u("Fact A: peak load is 500 concurrent uploads, average file size is 2GB."),
		a("Noted: 500 concurrent × 2GB = 1TB of in-flight data at peak. That's a significant storage and bandwidth consideration. We'll need direct-to-storage uploads (presigned URLs) to keep the API servers out of the data path."),
	}

	for i := 0; i < 5; i++ {
		turns = append(turns, fillerPair("error handling", i)...)
	}

	turns = append(turns,
		u("Fact B: we have a hard budget cap of $4000/month total for compute and storage. This includes everything."),
		a("Noted: $4000/month all-in. That's tight for video transcoding — GPU instances are the main cost driver. We'll need to use spot instances aggressively, batch transcoding jobs to maximize GPU utilization, and tier storage (hot/cold) to control storage spend."),
	)

	for i := 0; i < 6; i++ {
		turns = append(turns, fillerPair("API rate limiting", i)...)
	}

	turns = append(turns,
		u("Fact C: we MUST support 4K output resolution because our main customer is a sports broadcaster."),
		a("Noted: 4K output is non-negotiable. This rules out cheaper CPU-only transcoding for the 4K pipeline — you need GPU acceleration (NVENC or similar). Expect 3-5x the cost per minute of output vs 1080p. Combined with the $4000 budget, this is going to be a tight balance."),
	)

	for i := 0; i < 5; i++ {
		turns = append(turns, fillerPair("logging strategy", i)...)
	}

	return Scenario{
		ID:          "multi_hop_synth_1",
		Name:        "Multi-hop synthesis — capacity planning",
		Category:    "multi_hop",
		Description: "Three facts (load, budget, 4K requirement) spread across the conversation. Probe asks for a synthesis that requires all three.",
		Turns:       turns,
		Probes: []Probe{
			{
				ID:          "p1",
				ProbeAt:     len(turns),
				Question:    "Given everything we've discussed about this system, can you tell me realistically whether the budget is sufficient? Walk through the math.",
				MatchType:   MatchJudge,
				JudgeRubric: "The answer must reference ALL THREE facts: (1) 500 concurrent uploads at 2GB each (or the 1TB in-flight number), (2) the $4000/month budget cap, and (3) the 4K output requirement / GPU cost implication. Full marks (5) only if all three are cited and the analysis acknowledges the tightness/tension between them. Score 3-4 if two facts cited. Score 0-2 if only one or none.",
				Notes:       "Tests whether ACGC retains spaced-out facts that must be synthesized together.",
			},
		},
	}
}
