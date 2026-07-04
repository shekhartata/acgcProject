PROTO_DIR := proto
API_DIR := api/proto
BINARY := acgc

.PHONY: proto build run test clean tidy lint mongo mongo-down mongo-logs mongo-shell stresstest eval eval-cached eval-judge eval-strategies eval-semantic eval-semantic-judge stresstest-semantic latency-bench eval-fetch-external eval-longmemeval eval-locomo eval-longmemeval-semantic eval-locomo-semantic

# --- Build ---

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		-I $(PROTO_DIR) $(PROTO_DIR)/acgc.proto
	mv acgc.pb.go acgc_grpc.pb.go $(API_DIR)/ 2>/dev/null || true

build:
	go build -o bin/$(BINARY) ./cmd/acgc
	go build -o bin/testcli ./cmd/testcli
	go build -o bin/acgc-latencybench ./cmd/acgc-latencybench
	go build -o bin/stresstest ./stresstest
	go build -o bin/eval ./eval

run: build
	./bin/$(BINARY)

test:
	go test ./... -v

clean:
	rm -rf bin/
	go clean

tidy:
	go mod tidy

lint:
	go vet ./...

# --- MongoDB ---

mongo:
	docker compose up -d mongodb
	@echo "MongoDB running at mongodb://localhost:27017 (db: acgc)"

mongo-down:
	docker compose down

mongo-logs:
	docker compose logs -f mongodb

mongo-shell:
	docker exec -it acgc-mongo mongosh acgc

# --- Interactive Test ---

testcli: build
	./bin/testcli

# --- Stress Test ---

stresstest:
	go run -race ./stresstest/ -v

stresstest-export:
	go run ./stresstest/ -export stresstest/results.json

# --- Quality / Intelligence-per-token Eval ---

# Live run with probe-based scoring only (no judge). Requires ACGC_LLM_API_KEY.
eval:
	go run ./eval -v

# Live run with LLM-as-judge enabled for open-ended scenarios. Spends more tokens.
eval-judge:
	go run ./eval -v -judge

# Compare all three context strategies (naive_full_history is the reference).
# Uses the model-aware tokenizer for token accounting.
eval-strategies:
	go run ./eval -v -strategies "naive_full_history,sliding_window,acgc"

# Replay scored results from cache only — does NOT call the API. Free.
eval-cached:
	go run ./eval -cache-only

# Wipe cached responses (forces fresh API calls on next run).
eval-clean:
	rm -rf eval/cache eval/results

# Live run with HNSW semantic scoring in the ACGC pipeline. Embedding calls
# add a small cost (text-embedding-3-small is ~$0.02/1M tokens, negligible).
eval-semantic:
	go run ./eval -v -semantic

# Same as above but with the LLM judge for open-ended probes.
eval-semantic-judge:
	go run ./eval -v -semantic -judge

# Stress test with a deterministic mock embedder (no API key) — exercises the
# semantic code paths under -race without spending any cents.
stresstest-semantic:
	go run -race ./stresstest/ -v -semantic

# --- External benchmarks (LongMemEval, LoCoMo) ---

EXTERNAL_DATA := eval/datasets/external/data

# Download benchmark data files (not vendored; ~40 MB for LongMemEval).
eval-fetch-external:
	./eval/datasets/external/fetch.sh

# LongMemEval: 20 sampled instances, judge-scored, all three strategies.
# Costs real tokens — each instance carries a large multi-session haystack.
eval-longmemeval:
	go run ./eval -v -judge \
		-strategies "naive_full_history,sliding_window,acgc" \
		-external "longmemeval=$(EXTERNAL_DATA)/longmemeval_s_cleaned.json" \
		-external-sample 20

# LoCoMo: all 10 conversations, 20 sampled probes each, judge-scored.
eval-locomo:
	go run ./eval -v -judge \
		-strategies "naive_full_history,sliding_window,acgc" \
		-external "locomo=$(EXTERNAL_DATA)/locomo10.json" \
		-external-sample 20

# Same as above with HNSW semantic scoring in the ACGC pipeline.
eval-longmemeval-semantic:
	go run ./eval -v -judge -semantic \
		-strategies "naive_full_history,sliding_window,acgc" \
		-external "longmemeval=$(EXTERNAL_DATA)/longmemeval_s_cleaned.json" \
		-external-sample 20

eval-locomo-semantic:
	go run ./eval -v -judge -semantic \
		-strategies "naive_full_history,sliding_window,acgc" \
		-external "locomo=$(EXTERNAL_DATA)/locomo10.json" \
		-external-sample 20

# --- Latency benchmarking (naive vs grpc Run) ---

latency-bench: build
	@echo "Built bin/acgc-latencybench. Example (server: semantic on + Mongo + optional ACGC_LATENCY_BREAKDOWN=true):"
	@echo "  ./bin/acgc-latencybench -grpc localhost:50051 -iterations 30 -discard-n 3 -warm-settle-delay 400ms"

# --- Full Stack ---

up: build
	./bin/$(BINARY)

down: mongo-down
