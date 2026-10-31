# Development Guide

## Prerequisites

- Go version specified by `go.mod` (currently Go 1.25.7)
- Python 3.11 and packages in `scripts/requirements.txt`
- PostgreSQL 16 for integration tests and end-to-end local runs
- A test music directory containing a small number of MP3 files for worker validation

## Local setup

1. Copy `.env.example` to `.env`, set a local PostgreSQL URL and music mount path, and install Go/Python dependencies.
2. Start PostgreSQL, then run `go run ./cmd/migrate up`.
3. In separate terminals, start `go run ./cmd/api` and `go run ./cmd/worker`.
4. Confirm `/health`, `/api/v1/stats`, and `/metrics` respond. Worker scans immediately and on `SCAN_INTERVAL`.

The API applies embedded pending migrations during startup; the migrate command remains useful for explicit schema administration. Do not initialize production schemas by mounting all migration files under `/docker-entrypoint-initdb.d`.

## Quality checks

From the repository root:

- `gofmt -w` on changed Go files; CI checks that tracked Go files are formatted.
- `go vet ./...`
- `go test -count=1 ./...`
- `python -m pip install -r scripts/requirements.txt`
- `python -m unittest scripts.test_formulas`
- `docker compose -f docker-compose.prod.yml config` (requires deployment variables, including `POSTGRES_PASSWORD`)
- Build worker/API images using the Dockerfiles under `deployments/docker/`.

Database integration tests are opt-in through `DATABASE_URL`; run them only against an isolated test database because some tests insert/delete rows. Python integration tests may require audio codecs and additional CPU time.

## Retry and recovery behavior

Defaults are five total attempts, 30 seconds base retry delay, and a ten-minute stale-analysis lease. Backoff doubles per failure and is capped at 64 times the base. A shutdown cancellation returns an active claim to `pending`; a hard process/container kill leaves the claim in `analyzing` until the lease expires and the next worker startup recovers it. Configure the lease longer than the normal worst-case analysis duration to avoid reclaiming live work.

A file at an already-known path is treated as the same track. Content hash/mtime change detection and automated re-analysis are not yet implemented.

## Deployment safety

Production Compose requires a caller-supplied `POSTGRES_PASSWORD`, binds database/API published ports to localhost, mounts the music library read-only in analytics containers, and relies on the scraper frontend/Docker network for integration. Keep public access behind the configured tunnel/reverse proxy and its access controls. CORS allowlists browser origins; it does not authenticate callers.
