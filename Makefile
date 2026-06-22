# ─────────────────────────────────────────────────────────────────────────────
# SeatRush — Makefile
# Run `make` or `make help` to list available targets.
# ─────────────────────────────────────────────────────────────────────────────

# Load variables from .env if present (so DB_URL etc. can be overridden there).
ifneq (,$(wildcard .env))
include .env
export
endif

# ---- Config (override on the command line, e.g. `make migrate DB_URL=...`) ----
# Prefer DATABASE_URL from .env so migrations hit the same DB as the app.
DB_URL      ?= $(or $(DATABASE_URL),postgres://seatrush:seatrush@localhost:5433/seatrush?sslmode=disable)
MIGRATIONS  := migrations
BIN_DIR     := bin
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -ldflags "-s -w -X main.version=$(VERSION)"

# Use bash and fail fast on errors in recipe pipelines.
SHELL := bash
.DEFAULT_GOAL := help

.PHONY: help up down logs ps migrate migrate-down migrate-create migrate-force \
        seed stop run build clean test test-cover race fmt vet tidy verify

# ---- Help ---------------------------------------------------------------------

help: ## Show this help
	@echo "SeatRush — available targets:"
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ---- Infrastructure (Docker) --------------------------------------------------

up: ## Start postgres + redis in the background
	docker compose up -d

down: ## Stop and remove containers
	docker compose down

logs: ## Tail container logs
	docker compose logs -f

ps: ## Show container status
	docker compose ps

# ---- Database migrations ------------------------------------------------------

migrate: ## Apply all up migrations
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" up

migrate-down: ## Roll back the most recent migration
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" down 1

migrate-create: ## Create a new migration pair: make migrate-create name=add_foo
	@test -n "$(name)" || (echo "usage: make migrate-create name=<name>"; exit 1)
	migrate create -ext sql -dir $(MIGRATIONS) -seq $(name)

migrate-force: ## Force the schema version (recovery): make migrate-force version=N
	@test -n "$(version)" || (echo "usage: make migrate-force version=<N>"; exit 1)
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" force $(version)

seed: ## Insert the admin account + mock venues (idempotent)
	go run ./cmd/seed

# ---- Build & run --------------------------------------------------------------

stop: ## Kill any process currently holding port 8080
	@lsof -nP -iTCP:8080 -sTCP:LISTEN 2>/dev/null | awk 'NR>1 {print $$2}' | xargs -r kill && echo "stopped" || echo "nothing on :8080"

run: ## Run the API server (kills any stale process on :8080 first)
	@$(MAKE) stop
	go run ./cmd/api

build: ## Compile both binaries into ./bin
	go build $(LDFLAGS) -o $(BIN_DIR)/api  ./cmd/api
	go build $(LDFLAGS) -o $(BIN_DIR)/seed ./cmd/seed

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

# ---- Quality ------------------------------------------------------------------

test: ## Run all tests (Redis must be up)
	go test ./... -count=1

test-cover: ## Run tests with a coverage report (coverage.out + html)
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html

race: ## Run tests with the race detector
	go test ./... -race -count=1

fmt: ## Format all Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

tidy: ## Sync go.mod / go.sum
	go mod tidy

verify: fmt vet test ## Format, vet, and test in one shot
