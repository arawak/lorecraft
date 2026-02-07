BIN_DIR := bin
BIN := $(BIN_DIR)/lorecraft
GO := go

.PHONY: all build clean fmt vet test tidy neo4j-up neo4j-down neo4j-logs

all: build

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	$(GO) build -o $(BIN) ./cmd/lorecraft

clean:
	rm -rf $(BIN_DIR)

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

neo4j-up:
	docker compose -f compose.yaml up -d

neo4j-down:
	docker compose -f compose.yaml down

neo4j-logs:
	docker compose -f compose.yaml logs -f neo4j
