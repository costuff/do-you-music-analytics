# DoYou Music Analytics TODO

Active roadmap for the analytics service, synchronized with the scraper integration state as of 2026-03-16.

## Remaining Steps (Immediate)

- [ ] **Fix feature extraction algorithm** — audit `scripts/analyze_audio.py` high-level feature formulas (energy, danceability, valence, etc.); these are hand-crafted approximations from librosa low-level output and need validation/tuning
- [ ] Push package coverage toward 80%+ with priority on `internal/analyzer`, `internal/api`, `internal/db`
- [ ] Build music exploration visualization dashboard (scatter plots, histograms, interactive filters)
- [ ] **Consolidate deployment** — create root-level `~/srv/docker-compose.prod.yml` using Compose `include:` to wire both stacks (one `docker compose up`, shared network, single watchtower); keep both repos independent
- [ ] Implement export endpoints (CSV/JSON) for analyzed tracks with practical filters
- [ ] Add safe bulk re-analysis flow for failed/outdated tracks (dry-run + progress)
- [ ] Improve worker observability (throughput/error metrics) and retry/backoff behavior
- [ ] Keep integration contracts stable for scraper dashboard consumption (`/stats`, `/stats/detailed`, `/tracks`)

## Next Up (Ordered)

1. [ ] **Fix feature extraction algorithm (priority)**
	- Audit `scripts/analyze_audio.py` high-level feature formulas against real track output
	- Tune or replace hand-crafted librosa-derived formulas for energy, danceability, valence, acousticness, instrumentalness, speechiness, liveness, melodicness
	- Add unit tests for analyzer output ranges and edge cases
2. [ ] **Unit test coverage to 80%+**
	- `internal/analyzer`: failure-mode tests (python errors, invalid JSON, timeout paths)
	- `internal/api`: pagination/filter/error response tests
	- `internal/db`: claim/retry/missing-status integration scenarios
3. [ ] **Music exploration graph / visualization dashboard**
	- Scatter plot: Energy vs Danceability (per track, interactive)
	- BPM distribution histogram
	- Key/mode distribution (circle of fifths or bar chart)
	- Timeline: tracks analyzed over time
	- Filter by audio features (sliders or range inputs)
	- Recommend similar tracks based on feature proximity
4. [ ] **Consolidate deployment into root-level compose**
	- Create `~/srv/docker-compose.prod.yml` using `include:` directive
	- Move shared `doyou-network` and watchtower definitions to root file
	- Remove per-stack network/watchtower duplication
	- Single `docker compose -f ~/srv/docker-compose.prod.yml up -d` starts everything
5. [ ] **Export API**
	- Add CSV + JSON exports for analyzed tracks
	- Support filters (`status`, pagination, optional feature thresholds)
6. [ ] **Bulk re-analysis endpoint**
	- Trigger re-analysis for `failed` and optionally `completed` older than threshold
	- Include dry-run mode and progress reporting
7. [ ] **Worker observability and resilience**
	- Add clearer progress metrics in logs (claimed/processed/failed rates)
	- Tune retry/backoff for transient analyzer failures
8. [ ] **Dual-repo dev ergonomics**
	- Keep local workflow docs aligned with scraper integration
	- Maintain shared expectations for ports and env values

## Integration Contracts to Keep Stable

- [ ] Preserve response compatibility for:
  - `GET /api/v1/stats`
  - `GET /api/v1/stats/detailed`
  - `GET /api/v1/tracks`
- [ ] Re-validate CORS and connectivity against scraper frontend (`http://localhost:5173`) after API changes

## Documentation and Tooling

- [ ] Keep `docs/PREPUSH_CHECKLIST.md` and runtime docs synchronized
- [ ] Enforce linting with current `golangci-lint` configuration in CI
- [ ] Add short operator runbook for cleanup and scan cadence tuning

## Completed Recently

- [x] **Fixed `CleanupMissingTracks` bug (2026-03-16)** — nested `os.IsNotExist` AND logic silently skipped all missing tracks; replaced with `fileExists` flag that only requires the primary `fullPath` stat to succeed
- [x] **Set `CLEANUP_INTERVAL=10m` (2026-03-16)** — was `24h`; stale tracks now removed within 10 minutes of NAS deletion
- [x] **Fixed watchtower GHCR auth (2026-03-16)** — created static `~/.docker/watchtower-config.json` with inline credentials (osxkeychain inaccessible in containers)
- [x] **Fixed analytics proxy for remote users (2026-03-16)** — frontend bundle was hardcoding `localhost:8080`; added `/analytics-api/` nginx proxy block in scraper frontend, baked `VITE_ANALYTICS_API_URL=/analytics-api` via Dockerfile ARG
- [x] **Fixed DNS ambiguity 502 (2026-03-16)** — `proxy_pass http://api:3001` resolved to wrong container when frontend was on both Docker networks; changed to explicit `http://doyou-api:3001` and `http://doyou-analytics-api:8080/`
- [x] Removed duplicate `doyou-watchtower` from analytics compose (single watchtower instance in scraper stack)
- [x] Updated NAS path handling and scanner stability in production compose
- [x] Added explicit pre-push checklist for local validation before pipeline use
- [x] Established dual-repo local workflow guidance for analytics + scraper development
