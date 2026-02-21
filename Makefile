BIN_DIR := bin
BIN := $(BIN_DIR)/lorecraft
GO := go

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all build clean fmt vet test tidy db-up db-down db-logs

all: build

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/lorecraft

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

db-up:
	docker compose -f compose.yaml up -d

db-down:
	docker compose -f compose.yaml down

db-logs:
	docker compose -f compose.yaml logs -f postgres