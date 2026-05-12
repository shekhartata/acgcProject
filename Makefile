PROTO_DIR := proto
API_DIR := api/proto
BINARY := acgc

.PHONY: proto build run test clean tidy lint mongo mongo-down mongo-logs stresstest

# --- Build ---

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		-I $(PROTO_DIR) $(PROTO_DIR)/acgc.proto
	mv acgc.pb.go acgc_grpc.pb.go $(API_DIR)/ 2>/dev/null || true

build:
	go build -o bin/$(BINARY) ./cmd/acgc
	go build -o bin/testcli ./cmd/testcli
	go build -o bin/stresstest ./stresstest

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

# --- Full Stack ---

up: build
	./bin/$(BINARY)

down: mongo-down
