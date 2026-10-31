# Analytics Service Architecture

## Scope and boundaries

`do-you-music-analytics` owns the analytics PostgreSQL schema and the audio-analysis lifecycle. `doyou-world-scraper` owns playlist collection and downloads. The services exchange operational data through HTTP APIs and a shared filesystem location; neither service imports the other’s application code. In production, the analytics worker mounts the music library read-only.

```mermaid
flowchart LR
    Source[DoYou playlist source] --> Scraper[Scraper API / chromedp]
    Scraper --> Jobs[Scraper in-process async jobs]
    Jobs --> Downloader[yt-dlp downloader]
    Downloader --> Library[(Shared music library)]
    Library --> Scanner[Analytics scheduled scanner]
    Scanner --> Tracks[(PostgreSQL tracks / analysis state)]
    Tracks --> Claims[Go worker pool\nSKIP LOCKED claims]
    Claims --> Analyzer[Python / librosa subprocess]
    Analyzer --> Features[(PostgreSQL audio features)]
    Features --> Derived[Emotion derivation and similarity refresh]
    Derived --> AnalyticsDB[(PostgreSQL emotions / similarity)]
    AnalyticsDB --> API[Analytics REST API]
    API --> Dashboard[Scraper React dashboard]
```

The scraper job registry is currently process-local and ephemeral; a process restart loses its job history. Analytics work state is persisted in PostgreSQL. Do not describe scraper jobs as durable or the scraper as a message-queue-backed system.

## Analytics processing lifecycle

1. **Discovery:** scheduled scanner walks the configured music mount and inserts new MP3 paths as `pending`.
2. **Claim:** workers atomically select due pending rows under a transaction with `FOR UPDATE SKIP LOCKED`, increment attempts, and set `analysis_started_at`.
3. **Extraction:** a worker invokes the Python analyzer with a per-track timeout. Python emits JSON; Go maps validated output into the feature model.
4. **Persistence:** audio features are upserted; deterministic emotion dimensions are derived and upserted; only then is the track marked completed.
5. **Recovery:** on worker startup, `analyzing` rows older than `ANALYSIS_LEASE_TIMEOUT` are requeued with backoff or marked failed at the configured attempt limit. Failures are retried with capped exponential delay; shutdown attempts to release in-flight claims immediately.
6. **Similarity:** periodic computation stores top-k cosine-similarity pairs based on emotion vectors. This is a useful prototype but currently has quadratic all-pairs behavior and is not a vector index.

File-path uniqueness and upserts provide retry-safe persistence for a given discovered path. File content hashes/mtime are not yet used for change detection; replacing a file at an already completed path does not automatically guarantee re-analysis.

## Reliability and operational behavior

- PostgreSQL row locks coordinate workers across processes; there is no external broker.
- Retry attempts and next eligible times persist in the `tracks` table.
- API startup applies ordered embedded migrations under a PostgreSQL advisory transaction lock. Legacy schemas created by the former init-directory approach are adopted when their expected objects exist.
- Production Compose no longer mounts migration scripts into Postgres init hooks; the API migration runner is authoritative. The worker waits for the API health check (which implies migrations have completed).
- API readiness checks PostgreSQL connectivity. `/metrics` exports request totals by method/status, cumulative request duration, and in-flight requests. `/health` itself contributes to metrics.
- Production database/API host ports bind to loopback; containers on the Docker network continue to communicate internally. CORS is an origin policy, not authentication. Restrict the frontend/tunnel with the deployment’s access-control policy; no application-user authentication is implemented.

## Design decisions and trade-offs

### PostgreSQL as the work coordinator

The library and feature data already live in PostgreSQL, and `SKIP LOCKED` gives safe multi-worker claims without adding broker infrastructure. This is appropriate for the current single-library pipeline. A dedicated broker would add operational cost without a demonstrated need. Claim leases and bounded retry metadata handle process interruption, but the queue is database polling rather than push delivery.

### Python subprocess boundary

Librosa and its scientific dependencies remain in Python, while Go provides service lifecycle, bounded worker concurrency, database persistence, and HTTP APIs. JSON over a timed subprocess boundary keeps the two runtimes independently testable and avoids reimplementing mature audio libraries. It does add process startup/serialization overhead and feature schemas must stay aligned.

### Similarity before vector search

Current similarity uses 8 deterministic emotion dimensions and a top-20 neighbor table. This is explainable and does not justify pgvector yet. The current all-pairs computation is quadratic; benchmark and replace it with a vector index or approximate-nearest-neighbor approach only if collection growth makes recomputation or query latency a measured bottleneck.

### Separate services, shared storage

Scraping/downloading and CPU-intensive analysis have different dependencies and resource profiles, so they remain independently deployable. The shared filesystem is the ingestion contract; PostgreSQL is owned by analytics, and scraper UI integration happens through HTTP. This avoids coupling service releases to shared Go packages.

## Known limitations

- Scraper job state is not durable; analytics job attempts are durable per track.
- Track identity is currently path-based rather than content-hash-based.
- Similarity recomputation is quadratic and stored pairs are not a learned embedding index.
- Key/mode fields are not currently computed by the Python implementation.
- API endpoints have no user authentication. Production network/tunnel access controls must be configured externally.
- Metrics are request-level only; worker throughput and retry counters are available in logs/database status, not yet exported as Prometheus metrics.
- Database-dependent integration tests and Docker startup were not exercised on every developer machine; see the testing/deployment guides.
