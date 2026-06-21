# SeatRush developer shortcuts.
# DB URL is read from .env at runtime by the app, but migrations use the CLI,
# so we repeat it here for the migrate target.

DB_URL ?= postgres://seatrush:seatrush@localhost:5433/seatrush?sslmode=disable
MIGRATIONS := migrations

.PHONY: up down migrate migrate-down seed run test build tidy

## up: start postgres + redis
up:
	docker compose up -d

## down: stop containers
down:
	docker compose down

## migrate: apply all up migrations
migrate:
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" up

## migrate-down: roll back one migration
migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DB_URL)" down 1

## seed: insert admin + mock venues (idempotent)
seed:
	go run ./cmd/seed

## run: start the API server
run:
	go run ./cmd/api

## test: run unit/integration tests (Redis must be up)
test:
	go test ./... -count=1

## build: compile both binaries into ./bin
build:
	go build -o bin/api ./cmd/api
	go build -o bin/seed ./cmd/seed

## tidy: sync go.mod
tidy:
	go mod tidy
