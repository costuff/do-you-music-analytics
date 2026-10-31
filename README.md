# DoYou Music Analytics

Companion analytics service for [DoYou World Radio Scraper](https://github.com/costuff/doyou-world-scraper). It discovers MP3 files in a shared library, extracts audio features, persists processing state/results in PostgreSQL, and exposes REST APIs consumed by the scraper dashboard.

Collection size and throughput depend on the mounted library and deployment configuration; this README makes no scale, availability, or performance claims.

## Documentation

- [Architecture and design decisions](docs/ARCHITECTURE.md)
- [Development and validation guide](docs/DEVELOPMENT.md)
- [API reference](docs/API_USAGE.md)
- [Deployment guide](docs/DEPLOYMENT.md)
- [Database usage](docs/DATABASE_USAGE.md)
- [Dual-repository workflow](docs/DUAL_REPO_DEVELOPMENT.md)
- [Roadmap and next steps](docs/TODO.md)

## Architecture and data flow

The scraper owns playlist collection and downloading. This repository owns analytics discovery, per-track processing state, audio features, derived emotion values, similarity data, and the analytics API. The services communicate through a shared music-library mount and HTTP contracts; they do not share application code. Production Compose mounts the music library read-only in the analytics service.

The worker uses PostgreSQL transactions and `FOR UPDATE SKIP LOCKED` to coordinate concurrent claims. Python/librosa runs as a bounded Go-managed subprocess. Persisted attempts, retry eligibility timestamps, and analysis leases allow failed or abandoned work to be retried after process interruption. This is database-coordinated polling, not a broker-backed queue.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the Mermaid system diagram, failure behavior, and trade-offs.

## Implemented capabilities

- Scheduled MP3 library discovery and PostgreSQL-backed track status.
- Configurable concurrent Go worker pool and analyzer timeout.
- Python/librosa extraction, raw spectral/RMS/ZCR/MFCC fields, normalized heuristic audio features, deterministic emotion dimensions, and cosine-similarity results.
- Bounded exponential retries, persisted attempt counts, stale-claim recovery, and graceful interruption handling.
- Missing-file reconciliation and requeue when a previously missing path reappears.
- REST APIs for readiness, statistics, track browsing/search, emotions, and precomputed similar tracks.
- Request IDs, structured request logs, and Prometheus-format HTTP request metrics at `/metrics`.
- Independently built Docker API/worker images and GitHub Actions quality/build workflows.

Feature scores are heuristics, not Spotify-provided measurements or outputs from a trained ML model. Similarity is cosine similarity over deterministic emotion vectors, not embedding search. Musical key/mode fields are currently left unset by the analyzer because chroma estimation is disabled. See [Audio Analysis Scope and Limitations](docs/ARCHITECTURE.md#known-limitations).

## Quick start

### Requirements

- Go 1.25.7 (as specified in `go.mod`)
- Python 3.11 and dependencies from `scripts/requirements.txt`
- PostgreSQL 16+
- ffmpeg and a readable music directory for real audio processing

### Local setup

1. Install Go/Python dependencies and start a **development/test** PostgreSQL database.
2. Set `DATABASE_URL`, `NAS_MOUNT_PATH`, `PYTHON_PATH`, and `ANALYSIS_SCRIPT`; use an isolated database for tests.
3. Apply migrations with `go run ./cmd/migrate up`.
4. Start the API with `go run ./cmd/api` and the worker with `go run ./cmd/worker` in separate terminals.
5. Check `GET /health`, `GET /api/v1/stats`, and `GET /metrics` on the API address (default `http://localhost:8080`).

The API also applies embedded migrations at startup. Do not mount migration `.up.sql` and `.down.sql` files into Postgres `/docker-entrypoint-initdb.d`; the embedded API migration runner owns upgrades. For production deployment/secrets, follow [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md), not the local quick start.

## Configuration

See [.env.example](.env.example). Key settings include:

| Setting | Default | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | local development URL | PostgreSQL connection |
| `NAS_MOUNT_PATH` | `/mnt/music/downloads` | Readable music directory |
| `MAX_WORKERS` | `6` | Concurrent analysis workers (container deployment may override) |
| `SCAN_INTERVAL` | `30m` | Library discovery cadence |
| `ANALYSIS_TIMEOUT` | `300s` | Maximum duration per analyzer subprocess |
| `MAX_ANALYSIS_ATTEMPTS` | `5` | Attempts before a track remains failed |
| `RETRY_BASE_DELAY` | `30s` | Exponential retry delay base, capped at 64x |
| `ANALYSIS_LEASE_TIMEOUT` | `10m` | Stale claim recovery; must exceed analysis timeout |
| `ENABLE_CLEANUP` | `true` | Reconcile tracks whose files disappeared |
| `CLEANUP_ACTION` | `mark` | `mark` preserves DB history; `delete` removes rows |
| `CLEANUP_INTERVAL` | `24h` | Cleanup cadence (production Compose may override, e.g. `10m`) |
| `CORS_ORIGINS` | `http://localhost:5173` | Comma-separated browser origin allowlist |

CORS is not authentication. Restrict network/tunnel access appropriately; API user authentication is not implemented.

## API surface

- `GET /health` — database readiness and service version.
- `GET /metrics` — Prometheus text-format request counters, cumulative duration, and in-flight requests.
- `GET /api/v1/stats` and `/api/v1/stats/detailed` — processing counts and duration/rate summaries.
- `GET /api/v1/tracks`, `/api/v1/tracks/{id}`, `/api/v1/tracks/search`, `/api/v1/tracks/failed` — paginated/list/detail/search endpoints.
- `GET /api/v1/emotions`, `/api/v1/emotions/distribution`, `/api/v1/tracks/{id}/emotions` — derived emotion data.
- `GET /api/v1/tracks/{id}/similar?n=10`, `POST /api/v1/similarity/recompute`, `GET /api/v1/similarity/stats` — precomputed similarity endpoints.

The detailed request/response contract and examples are in [docs/API_USAGE.md](docs/API_USAGE.md).

## Testing, coverage, and build validation

```sh
# Formatting, static analysis, tests, and build
# gofmt -l should print no modified Go source files
gofmt -l $(git diff --name-only -- '*.go')
go vet ./...
go test -count=1 ./...
go build ./...

# Audio feature tests (install requirements first)
python -m unittest scripts.test_formulas

# Local image builds
docker build -f deployments/docker/Dockerfile.api -t doyou-analytics-api:local .
docker build -f deployments/docker/Dockerfile.worker -t doyou-analytics-worker:local .
```

DB integration tests require a **fresh isolated PostgreSQL database** with migrations applied; they create/delete test rows and run missing-file cleanup. Never point them at a production database. To include them in a coverage run:

```sh
go run ./cmd/migrate up
DATABASE_URL=postgresql://user:password@localhost:5432/doyou_test go test -count=1 -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Verified local baseline (2026-10-01)

- All Go packages passed `go test -count=1 ./...`, including DB integration tests against a fresh disposable PostgreSQL 16 instance with all seven migrations applied; `go vet ./...`, formatting, and `go build ./...` passed.
- Full-module Go statement coverage with DB tests enabled: **36.3%**. Core `internal/analyzer`, `internal/api`, `internal/config`, and `internal/db` packages together: **56.6%**; per package: analyzer **100.0%**, API **96.9%**, config **85.4%**, DB **32.5%**.
- Python audio formula/synthetic-audio suite: **31 tests passed** using dependencies in the worker image.
- Scraper companion frontend: **84 tests passed**, ESLint and production build passed. Scraper backend package coverage: **75.3%** total, including its command entrypoint at 0%; see the scraper README for package-level detail.
- Both production Compose files validated. All four API/worker/frontend images built locally. A disposable analytics API/database smoke test verified migration-on-startup, readiness, stats, metrics, version output, and subresource routing.

These are local verification results, not evidence of a production release, NAS-wide processing run, user load, or uptime. Docker images were not pushed and production services were not changed.

## Next steps

1. Improve analytics DB coverage with isolated per-test fixtures; test transactionality across feature/emotion persistence and retry/recovery edge cases.
2. Add content-hash/mtime identity so replaced files are detected and safely reanalyzed.
3. Export worker throughput, retries, stale-claim recovery, and queue-wait metrics; build operational dashboards/alerts.
4. Improve similarity-refresh consistency and benchmark the current quadratic algorithm before adopting vector search.
5. Add bounded export/reanalysis APIs and contract tests for pagination, errors, and scraper compatibility.
6. Add API authentication or enforce deployment access restrictions before broader exposure.
7. Validate upgrade/rollback behavior on a copy of the actual deployment schema and coordinate a deliberate production rollout.

The detailed ordered backlog lives in [docs/TODO.md](docs/TODO.md).
