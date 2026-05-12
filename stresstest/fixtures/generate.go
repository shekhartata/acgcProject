package fixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Conversation struct {
	Name        string
	Description string
	Turns       []Turn
}

func GenerateAll(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	convos := []Conversation{
		linearDeepDive(),
		multiTopicPivot(),
		backtrackingSession(),
		toolHeavySession(),
		longSession(),
	}

	for _, c := range convos {
		path := filepath.Join(outputDir, c.Name+".jsonl")
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		enc := json.NewEncoder(f)
		for _, t := range c.Turns {
			if err := enc.Encode(t); err != nil {
				f.Close()
				return fmt.Errorf("encode turn: %w", err)
			}
		}
		f.Close()
	}
	return nil
}

func LoadConversation(path string) ([]Turn, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var turns []Turn
	dec := json.NewDecoder(f)
	for dec.More() {
		var t Turn
		if err := dec.Decode(&t); err != nil {
			return nil, err
		}
		turns = append(turns, t)
	}
	return turns, nil
}

// linearDeepDive: 35 turns on database schema design, going deeper each turn.
func linearDeepDive() Conversation {
	turns := []Turn{
		{Role: "user", Content: "I need to design a database schema for an e-commerce platform. We'll have users, products, orders, and reviews. What's the best approach?"},
		{Role: "assistant", Content: "For an e-commerce platform, I recommend starting with these core tables:\n\n1. **users** - id, email, name, password_hash, created_at\n2. **products** - id, name, description, price, stock_quantity, category_id\n3. **orders** - id, user_id, status, total_amount, created_at\n4. **order_items** - id, order_id, product_id, quantity, unit_price\n5. **reviews** - id, user_id, product_id, rating, comment, created_at\n\nKey relationships: users have many orders, orders have many order_items, each order_item references a product, and users can write reviews for products.\n\nShould I detail the indexes and constraints?"},
		{Role: "user", Content: "Yes, what indexes do I need for performance?"},
		{Role: "assistant", Content: "Here are the critical indexes:\n\n**users**: Unique index on email, index on created_at for admin queries.\n\n**products**: Index on category_id for browsing, composite index on (category_id, price) for filtered price sorting, full-text index on name and description for search.\n\n**orders**: Index on user_id for 'my orders' queries, composite index on (status, created_at) for admin order management, index on created_at for reporting.\n\n**order_items**: Composite index on (order_id, product_id) which is also a unique constraint, index on product_id for 'who bought this' queries.\n\n**reviews**: Composite index on (product_id, rating) for product review pages, index on user_id for 'my reviews'.\n\nAvoid over-indexing — each index slows writes. Start with these and add more based on actual query patterns."},
		{Role: "user", Content: "What about handling product variants like size and color?"},
		{Role: "assistant", Content: "There are two common approaches:\n\n**Approach A - Separate variants table (recommended)**:\n- Add a `product_variants` table: id, product_id, sku, size, color, price_modifier, stock_quantity\n- order_items references product_variant_id instead of product_id\n- Allows independent stock tracking per variant\n\n**Approach B - JSON attributes**:\n- Add a JSONB `attributes` column to products\n- More flexible but harder to query and index\n- Works better for unpredictable attribute sets\n\nI recommend Approach A for e-commerce because you need reliable stock tracking per variant. The variant table also makes it easy to have different prices per size/color.\n\nShould I update the full schema with variants?"},
		{Role: "user", Content: "Go with Approach A. Also, I need to handle product categories as a hierarchy - like Electronics > Phones > Smartphones."},
		{Role: "assistant", Content: "For hierarchical categories, I recommend the **Materialized Path** pattern:\n\n```sql\nCREATE TABLE categories (\n  id SERIAL PRIMARY KEY,\n  name VARCHAR(100) NOT NULL,\n  parent_id INT REFERENCES categories(id),\n  path VARCHAR(500) NOT NULL, -- e.g., '/1/5/12/'\n  depth INT NOT NULL DEFAULT 0\n);\n```\n\nThe `path` column stores the full ancestry path. This makes queries fast:\n- Find all children: `WHERE path LIKE '/1/5/%'`\n- Find all ancestors: split the path string\n- Find depth: just read the depth column\n\nAlternatives considered and rejected:\n- **Adjacency List** (just parent_id): Simple but recursive queries are slow\n- **Nested Sets**: Fast reads but very slow inserts/updates\n- **Closure Table**: Powerful but requires a separate table with O(n²) rows\n\nMaterialized Path is the best trade-off for e-commerce where categories change rarely but are queried constantly."},
		{Role: "user", Content: "What about shipping addresses? Users can have multiple."},
		{Role: "assistant", Content: "Add an `addresses` table:\n\n```sql\nCREATE TABLE addresses (\n  id SERIAL PRIMARY KEY,\n  user_id INT NOT NULL REFERENCES users(id),\n  label VARCHAR(50), -- 'Home', 'Office'\n  line1 VARCHAR(255) NOT NULL,\n  line2 VARCHAR(255),\n  city VARCHAR(100) NOT NULL,\n  state VARCHAR(100),\n  postal_code VARCHAR(20) NOT NULL,\n  country VARCHAR(2) NOT NULL, -- ISO country code\n  is_default BOOLEAN DEFAULT false,\n  created_at TIMESTAMP DEFAULT NOW()\n);\n```\n\nThen add `shipping_address_id` and `billing_address_id` to the orders table. Important: don't just reference the address — **snapshot the address data into the order** at checkout time. If the user later updates their address, historical orders should still show the original shipping address.\n\nPattern: orders table gets `shipping_address_snapshot JSONB` that captures the address at order time."},
		{Role: "user", Content: "Good point about snapshotting. Now, how should I handle payments and payment status?"},
		{Role: "assistant", Content: "Create a `payments` table separate from orders:\n\n```sql\nCREATE TABLE payments (\n  id SERIAL PRIMARY KEY,\n  order_id INT NOT NULL REFERENCES orders(id),\n  payment_method VARCHAR(50) NOT NULL, -- 'card', 'upi', 'cod'\n  payment_provider VARCHAR(50), -- 'stripe', 'razorpay'\n  provider_transaction_id VARCHAR(255),\n  amount DECIMAL(10,2) NOT NULL,\n  currency VARCHAR(3) DEFAULT 'INR',\n  status VARCHAR(20) NOT NULL, -- 'pending', 'completed', 'failed', 'refunded'\n  paid_at TIMESTAMP,\n  created_at TIMESTAMP DEFAULT NOW()\n);\n```\n\nKey decisions:\n- **One order can have multiple payments** (partial payments, split payments, refunds)\n- **Payment status is separate from order status** — an order can be 'confirmed' even if payment is 'pending' (COD)\n- **Never store raw card numbers** — only store provider transaction IDs\n- Add index on (order_id, status) for quick payment lookups\n\nOrder status flow: pending → confirmed → processing → shipped → delivered (or cancelled/returned at any point)."},
	}

	// Add more turns to reach 35
	extraTopics := []struct{ q, a string }{
		{"How do I handle inventory and prevent overselling?", "Use SELECT FOR UPDATE or advisory locks when processing orders. Decrement stock in a transaction: check quantity >= requested, decrement, create order_item. If concurrent requests try to buy the last item, only one succeeds. For high traffic, consider a separate inventory service with Redis-backed counters for real-time stock checks, with periodic sync back to the database."},
		{"What about wishlists and saved items?", "Create a `wishlists` table: id, user_id, created_at. And `wishlist_items`: id, wishlist_id, product_id, added_at. Allow multiple wishlists per user (gift lists, etc). Add a unique constraint on (wishlist_id, product_id) to prevent duplicates. Index on user_id for 'my wishlists' queries."},
		{"I need to support discount codes and promotions. What's the schema?", "Create a `promotions` table: id, code, type (percentage/fixed), value, min_order_amount, max_uses, uses_count, valid_from, valid_until, is_active. And a `promotion_uses` junction: id, promotion_id, user_id, order_id, used_at — to track who used what and enforce per-user limits. The order itself should store applied_promotion_id and discount_amount for audit."},
		{"How should I handle product images?", "Create a `product_images` table: id, product_id, variant_id (nullable), url, alt_text, sort_order, is_primary. Store images in S3/CloudFront, only store URLs in the database. The sort_order lets the frontend show images in a specific sequence. Nullable variant_id means the image can be for the main product or a specific variant."},
		{"What about search? Should I use Elasticsearch or database full-text search?", "For an MVP, PostgreSQL full-text search is sufficient. Create a tsvector column on products and a GIN index. For scale, migrate to Elasticsearch or Typesense later. The search service would index product name, description, category, and attributes. Keep the source of truth in PostgreSQL and sync to the search engine asynchronously."},
		{"Now let's talk about the notification system.", "Add a `notifications` table: id, user_id, type (order_status, promotion, review_reply), title, body, is_read, metadata JSONB, created_at. For real-time, use WebSockets or SSE. For email/SMS, queue notifications in a job system (like a notifications_queue table or Redis queue). Important: notification preferences table to let users opt out of specific types."},
		{"I need audit logging for admin actions. Best approach?", "Create an `audit_logs` table: id, admin_user_id, action (create/update/delete), entity_type (product/order/user), entity_id, old_values JSONB, new_values JSONB, ip_address, created_at. Use database triggers or application middleware to capture changes automatically. Partition by month for performance since this table grows fast. Add index on (entity_type, entity_id) for entity history queries."},
		{"What about order returns and refunds?", "Create a `returns` table: id, order_id, user_id, reason, status (requested/approved/rejected/completed), refund_amount, created_at. And `return_items`: return_id, order_item_id, quantity, condition. When a return is approved, create a new payment record with negative amount or status='refunded'. This keeps the financial trail clean. The original payment stays as-is; the refund is a separate payment record linked to the same order."},
		{"How do I handle multi-currency support?", "Store all prices in the smallest currency unit (cents/paise) as integers to avoid floating-point issues. Add a `currencies` table: code, name, symbol, exchange_rate, updated_at. Products store price in a base currency. At checkout, convert using the exchange rate and store both the base and display amounts in the order. Lock the exchange rate at order time so it doesn't change retroactively."},
		{"What's the migration strategy? Should I use Flyway or golang-migrate?", "For a Go project, golang-migrate is the natural choice. Create a `migrations/` folder with numbered SQL files: 001_create_users.up.sql, 001_create_users.down.sql, etc. Run migrations as part of your deployment pipeline. Key rules: never modify a deployed migration, always write both up and down, test down migrations in staging."},
		{"Let me revisit the users table. I need to support OAuth login alongside email/password.", "Add an `auth_providers` table: id, user_id, provider (google/github/apple), provider_user_id, access_token_encrypted, refresh_token_encrypted, created_at. One user can have multiple auth providers. The users table keeps password_hash nullable (OAuth users don't have a password). Add a unique constraint on (provider, provider_user_id) to prevent duplicate OAuth links."},
		{"How should I handle soft deletes vs hard deletes?", "Use soft deletes for business data (users, orders, products) — add a `deleted_at TIMESTAMP` column, set it instead of deleting. Use hard deletes for transient data (sessions, carts, temporary tokens). Create a partial index `WHERE deleted_at IS NULL` on all soft-deletable tables so queries automatically filter deleted records efficiently. Add a scheduled job to purge soft-deleted records older than the retention period."},
		{"What about rate limiting for the API?", "Implement rate limiting at the API gateway level, not in the database. Use a sliding window counter in Redis: key = user_id:endpoint, increment per request, expire after window. Database schema addition: `rate_limit_overrides` table for per-user custom limits (premium users get higher limits). Log rate-limited requests in an `api_logs` table for abuse detection."},
	}

	for _, et := range extraTopics {
		turns = append(turns, Turn{Role: "user", Content: et.q})
		turns = append(turns, Turn{Role: "assistant", Content: et.a})
	}

	return Conversation{
		Name:        "linear_deep_dive",
		Description: "35 turns progressively building an e-commerce database schema",
		Turns:       turns,
	}
}

// multiTopicPivot: 30 turns — first half on topic A, then a hard pivot to topic B.
func multiTopicPivot() Conversation {
	var turns []Turn

	// Topic A: Python web scraping (15 turns)
	topicA := []struct{ q, a string }{
		{"I want to build a web scraper in Python to collect product prices from multiple e-commerce sites. Where do I start?", "Start with the `requests` library for HTTP and `BeautifulSoup` for HTML parsing. For JavaScript-heavy sites, use `Playwright` or `Selenium`. Architecture: create a base Scraper class with site-specific subclasses. Store results in a SQLite database for simplicity. Use `time.sleep()` between requests to be polite."},
		{"How do I handle anti-scraping measures like CAPTCHAs and IP blocking?", "Several strategies: 1) Rotate user agents using a list of real browser UAs. 2) Use proxy rotation services like ScraperAPI or BrightData. 3) Add random delays between 2-10 seconds. 4) Use headless browsers with realistic behavior (mouse movements, scrolling). 5) For CAPTCHAs, use services like 2Captcha as last resort. Most importantly, respect robots.txt and rate limits."},
		{"Show me the base scraper class design.", "The base class should have: `__init__(self, base_url, delay)`, `fetch_page(url) -> BeautifulSoup`, `parse_product(soup) -> Product`, `run(urls) -> List[Product]`. Abstract methods: `parse_product` and `get_product_urls`. Concrete methods: `fetch_page` with retry logic, `save_results` to database. Use `@retry` decorator from tenacity for automatic retries on network errors."},
		{"What about scheduling the scraper to run daily?", "Use APScheduler for in-process scheduling or cron for system-level scheduling. For production: Celery with Redis broker for distributed task scheduling. Each scraper run should: 1) fetch current prices, 2) compare with last known prices, 3) store new prices with timestamp, 4) send alerts if price drops significantly. Store price history for trend analysis."},
		{"How do I store the scraped data efficiently?", "Use PostgreSQL with a time-series friendly schema: `products` table (id, url, name, site), `price_snapshots` table (id, product_id, price, currency, scraped_at). Partition price_snapshots by month for query performance. Add indexes on (product_id, scraped_at DESC) for latest price lookups. For analytics, create materialized views for daily/weekly price averages."},
		{"Can I parallelize the scraping? It's too slow sequentially.", "Use `asyncio` with `aiohttp` for async HTTP requests. Create a semaphore to limit concurrent requests per domain (respect their servers). Alternative: use `concurrent.futures.ThreadPoolExecutor` if you prefer threads. Architecture: producer-consumer pattern — one task discovers URLs, worker pool fetches and parses them. With async, you can scrape 50-100 pages concurrently while staying polite with per-domain rate limits."},
		{"How do I detect when a website changes its HTML structure?", "Implement structural validation: after parsing, check that expected fields are non-empty. If parsing yields null/empty for >10% of products, trigger an alert. Store a hash of the CSS selector paths used for parsing — if selectors stop matching, the site probably redesigned. Use a monitoring dashboard that shows parsing success rates per site per day. When a site breaks, only that site-specific parser needs updating."},
	}

	for _, t := range topicA {
		turns = append(turns, Turn{Role: "user", Content: t.q})
		turns = append(turns, Turn{Role: "assistant", Content: t.a})
	}

	// Hard pivot to Topic B: Kubernetes deployment
	turns = append(turns, Turn{Role: "user", Content: "Actually, let's completely switch topics. I need to deploy my Go microservices to Kubernetes. I'm new to K8s."})

	topicB := []struct{ q, a string }{
		{"What are the basic K8s concepts I need to understand?", "Core concepts: **Pod** = smallest deployable unit (one or more containers). **Deployment** = manages pod replicas and rolling updates. **Service** = stable network endpoint for pods. **ConfigMap/Secret** = configuration and sensitive data. **Namespace** = logical cluster partitioning. **Ingress** = HTTP routing from outside the cluster. Start by containerizing your Go services with Docker, then write Deployment and Service YAML manifests."},
		{"How do I structure my Kubernetes manifests?", "Use a `k8s/` directory with this structure: `k8s/base/` for common manifests, `k8s/overlays/dev/` and `k8s/overlays/prod/` for environment-specific overrides. Use Kustomize (built into kubectl) for overlay management. Each service gets: deployment.yaml, service.yaml, configmap.yaml. Shared resources: namespace.yaml, ingress.yaml, secrets.yaml (encrypted with SOPS or sealed-secrets)."},
		{"How should I handle database connections from K8s pods?", "Never run production databases inside K8s — use managed services (RDS, Cloud SQL, Atlas). Store connection strings in K8s Secrets. Use init containers to wait for database availability. Connection pooling: use PgBouncer as a sidecar container or a separate deployment. Set resource limits on pods to prevent one service from consuming all database connections."},
		{"What about health checks and readiness probes?", "Every Go service should expose: `/healthz` (liveness — is the process alive?) and `/readyz` (readiness — can it serve traffic?). Readiness should check database connectivity and downstream dependencies. Configure in deployment.yaml: livenessProbe with httpGet path=/healthz, readinessProbe with httpGet path=/readyz. Set initialDelaySeconds, periodSeconds, and failureThreshold appropriately."},
		{"How do I handle logging and monitoring in K8s?", "Logging: write to stdout/stderr (not files), use structured JSON logging. K8s collects stdout via container runtime. Aggregate with Fluentd/Fluent Bit → Elasticsearch/Loki. Monitoring: expose Prometheus metrics at `/metrics` endpoint. Deploy Prometheus + Grafana via Helm charts. Key metrics: request latency (p50/p95/p99), error rate, pod CPU/memory usage, database connection pool utilization."},
		{"What's the best way to handle secrets in K8s?", "Options ranked by security: 1) **External Secrets Operator** syncing from AWS Secrets Manager/Vault — best for production. 2) **Sealed Secrets** — encrypt secrets in git, controller decrypts in-cluster. 3) **SOPS** with age/KMS — encrypt YAML files, decrypt during deployment. 4) Plain K8s Secrets — base64 encoded, not encrypted at rest by default. Never commit plain secrets to git. Use RBAC to restrict secret access to specific namespaces and service accounts."},
		{"How do I set up CI/CD for K8s deployments?", "Pipeline: 1) Push to git → 2) CI runs tests → 3) Build Docker image → 4) Push to container registry → 5) Update K8s manifest with new image tag → 6) Apply to cluster. Use GitHub Actions or GitLab CI. For step 5-6, two approaches: **push-based** (CI runs kubectl apply) or **pull-based** (ArgoCD/Flux watches git and syncs). Pull-based (GitOps) is more secure — CI never needs cluster credentials."},
		{"How do I handle rolling updates without downtime?", "Kubernetes Deployments handle this natively. Set `strategy.type: RollingUpdate` with `maxUnavailable: 0` and `maxSurge: 1`. This ensures at least N pods are always running during updates. Key requirements: readiness probes must be configured (K8s won't send traffic until the new pod is ready), graceful shutdown handling in your Go code (catch SIGTERM, finish in-flight requests, close connections). Set `terminationGracePeriodSeconds` to match your longest expected request."},
	}

	for _, t := range topicB {
		turns = append(turns, Turn{Role: "user", Content: t.q})
		turns = append(turns, Turn{Role: "assistant", Content: t.a})
	}

	return Conversation{
		Name:        "multi_topic_pivot",
		Description: "30 turns — web scraping then hard pivot to Kubernetes",
		Turns:       turns,
	}
}

// backtrackingSession: explore approach A, reject it, try approach B.
func backtrackingSession() Conversation {
	topics := []struct{ q, a string }{
		{"I need to implement real-time notifications in my web app. Should I use WebSockets?", "WebSockets are a solid choice for real-time bidirectional communication. You'd maintain a persistent connection between client and server. In Go, use the `gorilla/websocket` package. Architecture: WebSocket server manages connections per user, broadcasts events to connected clients. For multiple server instances, you'll need a pub/sub system (Redis Pub/Sub) to fan out messages across servers."},
		{"Okay let me try WebSockets. How do I handle reconnection and connection drops?", "Client-side: implement exponential backoff reconnection (1s, 2s, 4s, 8s, max 30s). Use a heartbeat/ping-pong mechanism (send ping every 30s, expect pong within 10s, reconnect if not). Server-side: clean up stale connections with a goroutine that checks last ping time. Message queue: buffer messages during disconnection and replay on reconnect using a per-user message queue with sequence numbers."},
		{"Wait, this is getting complex. Managing WebSocket state across multiple server instances seems really hard. What about Server-Sent Events instead?", "Good instinct. SSE (Server-Sent Events) is simpler for your use case if you only need **server-to-client** push (which notifications are). Benefits over WebSockets: 1) Built on HTTP — works through all proxies and load balancers. 2) Automatic reconnection built into the browser API. 3) No special server infrastructure. 4) Simpler scaling — just an HTTP endpoint. Limitation: unidirectional (server → client only). But for notifications, that's all you need."},
		{"Hmm, but I might need bidirectional later for chat features. Let me reconsider...", "If chat is on the roadmap, you'll eventually need bidirectional anyway. But I'd recommend: **SSE for notifications now, WebSockets for chat later**. They can coexist. Don't over-engineer for a feature you might build. If you do go WebSockets for everything, consider using a managed service like Ably, Pusher, or AWS AppSync instead of building your own WebSocket infrastructure."},
		{"Actually, you know what, let me just use Firebase Cloud Messaging. It handles everything.", "FCM is great for mobile push notifications but has limitations for web: requires service workers, doesn't work in all browsers equally, messages can be delayed by up to several minutes (not truly real-time), you don't control the infrastructure. For web real-time notifications (show a bell icon with instant updates), SSE or WebSockets are still better. FCM is perfect as an additional channel for when the user closes the browser tab."},
		{"Okay I'm going back to SSE. Show me a Go implementation.", "Simple SSE endpoint in Go:\n\n```go\nfunc sseHandler(w http.ResponseWriter, r *http.Request) {\n    w.Header().Set(\"Content-Type\", \"text/event-stream\")\n    w.Header().Set(\"Cache-Control\", \"no-cache\")\n    w.Header().Set(\"Connection\", \"keep-alive\")\n    flusher := w.(http.Flusher)\n    \n    for {\n        select {\n        case msg := <-userChannel:\n            fmt.Fprintf(w, \"data: %s\\n\\n\", msg)\n            flusher.Flush()\n        case <-r.Context().Done():\n            return\n        }\n    }\n}\n```\n\nKey: use `http.Flusher` to push data immediately. Each connected user gets a channel. Use Redis Pub/Sub to distribute messages across multiple server instances."},
		{"How do I scale SSE connections across multiple servers?", "Use Redis Pub/Sub as the message bus. When a notification is created: publish to Redis channel `notifications:{user_id}`. Each SSE server subscribes to channels for its connected users. Architecture: Load balancer → SSE servers (sticky sessions not required because Redis handles fan-out) → Redis → notification producers. For 10K+ concurrent connections, use a dedicated SSE server pool separate from your API servers."},
		{"What about message ordering and deduplication?", "Add a monotonically increasing sequence number to each notification. Client tracks the last received sequence. On reconnect, client sends `Last-Event-ID` header (SSE standard) — server replays messages from that sequence. For deduplication, client ignores messages with sequence <= last seen. Server-side: store notifications in PostgreSQL with an auto-increment ID, query `WHERE id > last_event_id AND user_id = ?` on reconnect."},
		{"Great. Now I need to handle notification preferences — users should choose what they get notified about.", "Create a `notification_preferences` table: user_id, notification_type, channel (in_app/email/push), enabled. Default all to true. UI: settings page with toggles per notification type. Server-side: before sending any notification, check preferences. Cache preferences in Redis for fast lookups (invalidate on update). Notification types: order_updates, promotions, price_drops, review_replies, security_alerts (security alerts should always be on, not toggleable)."},
		{"One more thing — I need to batch digest emails instead of sending individual emails for every notification.", "Create a `notification_digest_queue` table: user_id, notification_type, payload JSONB, created_at, digest_sent_at. A cron job runs every hour (or daily, user-configurable): group unsent notifications by user, compile into a single digest email template, send via email service, mark as sent. Users can choose digest frequency in their preferences: instant, hourly, daily, weekly. For instant, skip the queue and send immediately."},
	}

	var turns []Turn
	for _, t := range topics {
		turns = append(turns, Turn{Role: "user", Content: t.q})
		turns = append(turns, Turn{Role: "assistant", Content: t.a})
	}

	return Conversation{
		Name:        "backtracking",
		Description: "20 turns exploring notification approaches — WebSockets, SSE, FCM, back to SSE",
		Turns:       turns,
	}
}

// toolHeavySession: lots of code snippets and tool-like outputs.
func toolHeavySession() Conversation {
	topics := []struct{ q, a string }{
		{"Run this SQL and tell me what's wrong: SELECT * FROM users WHERE email = 'test@example.com' AND deleted_at = NULL", "The bug: `deleted_at = NULL` should be `deleted_at IS NULL`. In SQL, `NULL` is not a value — it's the absence of a value. The `=` operator always returns `NULL` (not true/false) when comparing with NULL. Corrected query:\n```sql\nSELECT * FROM users WHERE email = 'test@example.com' AND deleted_at IS NULL;\n```"},
		{"I ran EXPLAIN ANALYZE on my slow query and got this: Seq Scan on orders, cost=0.00..45892.00, rows=1250000, actual time=0.012..312.456", "A sequential scan on 1.25M rows is the problem. The query is scanning the entire orders table. Solutions:\n1. Add an index on the column(s) in your WHERE clause\n2. If you're filtering by user_id: `CREATE INDEX idx_orders_user ON orders(user_id);`\n3. If filtering by date range: `CREATE INDEX idx_orders_date ON orders(created_at);`\n4. Run `ANALYZE orders;` to update table statistics\n5. Check if `work_mem` is too low — increase it for complex sort operations\n\nAfter adding the index, re-run EXPLAIN ANALYZE. You should see an Index Scan with dramatically lower cost."},
		{"Here's my Go code that's leaking goroutines:\n```go\nfunc processItems(items []Item) {\n    for _, item := range items {\n        go func() {\n            result := heavyComputation(item)\n            saveResult(result)\n        }()\n    }\n}\n```", "Two bugs:\n1. **Closure capture bug**: The goroutine captures `item` by reference, not value. All goroutines will use the last value of `item`. Fix: pass `item` as a parameter.\n2. **No goroutine lifecycle management**: Goroutines are fire-and-forget — no way to wait for completion or limit concurrency.\n\nFixed:\n```go\nfunc processItems(items []Item) {\n    var wg sync.WaitGroup\n    sem := make(chan struct{}, 10) // limit to 10 concurrent\n    for _, item := range items {\n        wg.Add(1)\n        go func(it Item) {\n            defer wg.Done()\n            sem <- struct{}{}\n            defer func() { <-sem }()\n            result := heavyComputation(it)\n            saveResult(result)\n        }(item)\n    }\n    wg.Wait()\n}\n```"},
		{"Now I have a race condition. The test shows: WARNING: DATA RACE at saveResult(). How do I fix it?", "The `saveResult()` function likely writes to a shared data structure without synchronization. Options:\n1. **Mutex**: Wrap the shared write in `sync.Mutex` Lock/Unlock\n2. **Channel**: Send results to a single consumer goroutine via a channel\n3. **sync.Map**: If saving to a map, use sync.Map instead of regular map\n\nChannel approach (recommended for your pattern):\n```go\nresults := make(chan Result, len(items))\n// ... goroutines send to results channel ...\ngo func() { wg.Wait(); close(results) }()\nfor r := range results {\n    saveResult(r) // single writer, no race\n}\n```"},
		{"I'm getting this error in production: 'too many open files'. My Go server has 500 concurrent connections.", "Your process is hitting the file descriptor limit. Each TCP connection = 1 file descriptor. Fixes:\n1. **Increase ulimit**: `ulimit -n 65535` (temporary) or edit `/etc/security/limits.conf` (permanent)\n2. **Check for connection leaks**: Ensure every `http.Response.Body` is closed with `defer resp.Body.Close()`\n3. **Set timeouts**: `http.Server{ReadTimeout: 10s, WriteTimeout: 30s, IdleTimeout: 120s}`\n4. **Connection pooling**: Set `MaxIdleConns`, `MaxIdleConnsPerHost` on your HTTP transport\n5. **Monitor**: `lsof -p <pid> | wc -l` to see current open FDs\n\nMost common cause: forgetting to close response bodies in HTTP clients."},
		{"My Docker image is 1.2GB. How do I reduce it?", "Use multi-stage builds:\n```dockerfile\n# Build stage\nFROM golang:1.22-alpine AS builder\nWORKDIR /app\nCOPY go.* ./\nRUN go mod download\nCOPY . .\nRUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /server ./cmd/server\n\n# Runtime stage\nFROM alpine:3.19\nRUN apk add --no-cache ca-certificates\nCOPY --from=builder /server /server\nEXPOSE 8080\nCMD [\"/server\"]\n```\nThis typically reduces a 1.2GB image to ~15-25MB. Key: `CGO_ENABLED=0` for static binary, `-ldflags='-s -w'` strips debug info, `alpine` base is ~5MB."},
		{"Getting a segfault in production: signal SIGSEGV: segmentation violation. Stack trace points to a nil pointer in my handler.", "Common causes of nil pointer in Go HTTP handlers:\n1. **Uninitialized dependency**: Your handler struct has a nil field (database client, logger, etc). Check initialization.\n2. **Nil response from database**: `db.QueryRow().Scan()` found no rows but you didn't check `sql.ErrNoRows`\n3. **Nil map**: Using a map that was declared but never initialized with `make()`\n4. **Interface nil check trap**: `var err error = (*MyError)(nil)` — this is non-nil even though the pointer is nil\n\nFix: Add nil checks before dereferencing. Use `recover()` middleware to catch panics gracefully:\n```go\nfunc recoveryMiddleware(next http.Handler) http.Handler {\n    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n        defer func() {\n            if err := recover(); err != nil {\n                log.Printf(\"panic: %v\\n%s\", err, debug.Stack())\n                http.Error(w, \"internal error\", 500)\n            }\n        }()\n        next.ServeHTTP(w, r)\n    })\n}\n```"},
		{"My API response time jumped from 50ms to 2 seconds. Nothing changed in my code. How do I debug?", "Systematic debugging approach:\n1. **Check database**: Run `SELECT * FROM pg_stat_activity WHERE state = 'active';` — look for long-running queries or lock waits\n2. **Check connections**: Is the connection pool exhausted? `pg_stat_activity` count vs your max_connections\n3. **Check external services**: Any downstream API calls timing out? Add timeout context to all external calls\n4. **Check resources**: `top`, `iostat`, `df -h` — CPU spike? Disk full? Swap thrashing?\n5. **Check recent deployments**: Even if 'nothing changed', check infra changes — DNS, TLS cert renewal, load balancer config\n6. **Add timing middleware**: Log time spent in each layer (handler, database, external calls) to identify the bottleneck\n\nMost common cause: database — missing index on a new query pattern or table growing past a threshold where an index becomes necessary."},
		{"Found it — it was a missing index. But now I need to add the index without downtime on a table with 50M rows.", "Use `CREATE INDEX CONCURRENTLY`:\n```sql\nCREATE INDEX CONCURRENTLY idx_orders_status_date ON orders(status, created_at);\n```\nThis builds the index without locking the table for writes. It takes longer but doesn't cause downtime. Caveats:\n1. Cannot run inside a transaction\n2. If it fails partway, it leaves an INVALID index — drop it and retry\n3. Requires extra disk space temporarily (2x index size)\n4. Can be slower on tables with heavy write traffic\n\nMonitor progress: `SELECT * FROM pg_stat_progress_create_index;`"},
		{"One more — my Go tests are flaky. They pass locally but fail in CI randomly.", "Common causes of flaky Go tests:\n1. **Time-dependent**: Tests use `time.Now()` or `time.Sleep()` — use a clock interface or `time.After` with generous buffers\n2. **Port conflicts**: Tests bind to hardcoded ports — use `:0` for random port assignment\n3. **Shared state**: Tests modify package-level variables — use `t.Parallel()` carefully and reset state in setup\n4. **Database state**: Tests depend on data from other tests — use test transactions that rollback\n5. **Race conditions**: Run `go test -race -count=10` locally to reproduce\n6. **Resource exhaustion**: CI has fewer resources — tests that barely pass locally fail under load\n\nQuick fix: Run `go test -count=100 -race ./...` locally until you reproduce the failure, then fix the root cause."},
	}

	var turns []Turn
	for _, t := range topics {
		turns = append(turns, Turn{Role: "user", Content: t.q})
		turns = append(turns, Turn{Role: "assistant", Content: t.a})
	}

	return Conversation{
		Name:        "tool_heavy",
		Description: "20 turns with code snippets, SQL, error outputs — simulating tool-heavy debugging",
		Turns:       turns,
	}
}

// longSession: 60+ turns to stress test GC triggers.
func longSession() Conversation {
	var turns []Turn
	topics := []string{
		"authentication", "authorization", "rate limiting", "caching",
		"database optimization", "API design", "error handling", "logging",
		"monitoring", "deployment", "testing", "security",
		"performance", "scalability", "documentation", "code review",
	}

	questionTemplates := []string{
		"What are the best practices for %s in a Go microservice?",
		"How do I implement %s properly?",
		"What are common mistakes with %s?",
		"Can you show me a production-ready %s implementation?",
		"How do I test %s effectively?",
	}

	answerTemplate := func(topic, qType string) string {
		base := fmt.Sprintf("For %s in Go microservices, here are the key considerations:\n\n", topic)
		points := []string{
			fmt.Sprintf("**Architecture**: Design your %s layer as middleware that can be composed with other handlers. Keep it decoupled from business logic.", topic),
			fmt.Sprintf("**Configuration**: Make %s parameters configurable via environment variables. Use sensible defaults that work for development.", topic),
			fmt.Sprintf("**Error handling**: %s failures should be logged with structured context (request ID, user ID, timestamp) and return appropriate HTTP status codes.", topic),
			fmt.Sprintf("**Testing**: Write both unit tests (mock dependencies) and integration tests (use testcontainers) for your %s implementation.", topic),
			fmt.Sprintf("**Monitoring**: Expose %s metrics via Prometheus (counters for successes/failures, histograms for latency). Set up alerts for anomalies.", topic),
			fmt.Sprintf("**Documentation**: Document your %s approach in an ADR (Architecture Decision Record) explaining why you chose this implementation over alternatives.", topic),
		}
		var b strings.Builder
		b.WriteString(base)
		for i, p := range points {
			b.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, p))
		}
		b.WriteString(fmt.Sprintf("Common pitfall: Don't implement %s from scratch when battle-tested libraries exist. For Go, check the awesome-go list for vetted packages.\n\n", topic))
		b.WriteString(fmt.Sprintf("Production checklist for %s:\n- [ ] Unit tests with >80%% coverage\n- [ ] Integration test in CI\n- [ ] Metrics exposed\n- [ ] Alerts configured\n- [ ] Documented in README\n- [ ] Reviewed by team\n", topic))
		return b.String()
	}

	qIdx := 0
	for _, topic := range topics {
		for j := 0; j < 2; j++ {
			q := fmt.Sprintf(questionTemplates[qIdx%len(questionTemplates)], topic)
			a := answerTemplate(topic, q)
			turns = append(turns, Turn{Role: "user", Content: q})
			turns = append(turns, Turn{Role: "assistant", Content: a})
			qIdx++
		}
	}

	// Add a few wrap-up turns
	turns = append(turns, Turn{Role: "user", Content: "Can you summarize all the key decisions we've made across this entire conversation?"})
	turns = append(turns, Turn{Role: "assistant", Content: "Here's a summary of all architectural decisions made:\n\n1. Middleware-based architecture for cross-cutting concerns\n2. Environment-variable configuration with sensible defaults\n3. Structured logging with request context\n4. Prometheus metrics for all critical paths\n5. Testcontainers for integration testing\n6. ADR documentation for major decisions\n7. No NIH syndrome — use proven libraries\n8. CI/CD pipeline with test, build, deploy stages\n\nThese decisions form a coherent, production-ready Go microservice framework. The key theme: keep things simple, observable, and well-tested."})

	return Conversation{
		Name:        "long_session",
		Description: "34 turns covering 16 microservice topics — stress tests GC with volume",
		Turns:       turns,
	}
}
