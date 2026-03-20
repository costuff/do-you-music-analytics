## Local development helpers
## Usage: make dev-api | make dev-worker | make db | make stop

# Load .env.local if it exists
ifneq (,$(wildcard .env.local))
  include .env.local
  export
endif

.PHONY: db dev-api dev-worker stop migrate logs test

## Start Postgres (if not already running)
db:
	docker compose up -d postgres
	@echo "Postgres ready at localhost:5432"

## Run the API locally (reads .env.local automatically)
dev-api: db
	go run ./cmd/api

## Run the worker locally
dev-worker: db
	go run ./cmd/worker

## Apply all pending migrations
migrate: db
	go run ./cmd/migrate

## Stop Postgres
stop:
	docker compose down

## Tail Postgres logs
logs:
	docker compose logs -f postgres

## Run Go tests (no DB required for pure unit tests)
test:
	go test ./... -count=1

## Run Python formula tests
test-py:
	python3 -m unittest discover -s scripts -p "test_*.py" -v
