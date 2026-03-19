# DoYou Music Analytics TODO

Active roadmap for the analytics service, synchronized with the scraper integration state as of 2026-03-16.

## Remaining Steps (Immediate)

- [ ] **Deploy v0.6.0 to production** — Watchtower will pull the new image; then apply migration 003 manually: `docker exec -it doyou-analytics-db psql -U doyou -d doyou_music -f /migrations/003_add_raw_features.up.sql`; then reset all `analysis_version = '1.0.0'` tracks to `pending` so they get re-analyzed with MFCCs + mode + accurate loudness_db
- [ ] **Fix/validate high-level feature formulas** — audit `scripts/analyze_audio.py`; valence/danceability/acousticness etc. are rough approximations; tune against tracks with known Spotify values; constants like `bpm−125` pivot are arbitrary
- [ ] **Add emotion layer** — new `track_emotions` table with 8 dimensions: `tension`, `melancholy`, `euphoria`, `aggression`, `serenity`, `nostalgia`, `sensuality`, `power` (each 0–1); computed deterministically from audio features via a mapping matrix; expose `GET /api/v1/tracks/:id/emotions`
- [ ] **Compute similarity graph** — `track_similarity` table `(track_a_id, track_b_id, score FLOAT)`; background Go job computes cosine similarity in emotion+feature space using k-nearest-neighbours (k=20); expose `GET /api/v1/tracks/:id/similar?n=10`
- [ ] Push package coverage toward 80%+ with priority on `internal/analyzer`, `internal/api`, `internal/db`
- [ ] **Consolidate deployment** — create root-level `~/srv/docker-compose.prod.yml` using Compose `include:` to wire both stacks (one `docker compose up`, shared network, single watchtower); keep both repos independent
- [ ] Implement export endpoints (CSV/JSON) for analyzed tracks with practical filters
- [ ] Add safe bulk re-analysis flow for failed/outdated tracks (dry-run + progress)
- [ ] Improve worker observability (throughput/error metrics) and retry/backoff behavior
- [ ] Keep integration contracts stable for scraper dashboard consumption (`/stats`, `/stats/detailed`, `/tracks`)
- [ ] **Authentication (personal use → multi-user)**
  - Short term: token-based auth on the analytics API (currently unauthenticated behind nginx proxy)
  - Long term: per-user libraries — each user has their own track collection, analysis results, and dashboard
  - Design note: keep single-user mode as default; multi-user is an opt-in layer on top

## Next Up (Ordered)

0. [ ] **Deploy v0.6.0 + re-queue existing tracks**
	- Watchtower will auto-pull the new image once pushed (check `docker ps` to confirm `doyou-analytics-worker` is on v0.6.0)
	- Apply migration 003 on Mac Mini:
	  ```
	  docker exec doyou-analytics-db psql -U doyou -d doyou_music \
	    -c "$(cat migrations/003_add_raw_features.up.sql)"
	  ```
	  (or copy the SQL file into the container first)
	- Re-queue old tracks for re-analysis:
	  ```sql
	  UPDATE tracks
	  SET analysis_status = 'pending', analyzed_at = NULL
	  WHERE id IN (
	      SELECT track_id FROM audio_features WHERE analysis_version = '1.0.0'
	  );
	  ```
	- Worker will automatically pick them up and write MFCCs, mode, loudness_db on next pass
1. [ ] **Fix and validate high-level feature formulas**
	- Audit `scripts/analyze_audio.py`; tune constants for energy, danceability, valence, acousticness, instrumentalness, speechiness, liveness, melodicness against tracks with known Spotify values
	- Specific issues to address:
	  - `danceability`: `1 - abs(bpm-125)/125` bottoms out at 0 for BPM ≥ 250; has no rhythm regularity component
	  - `valence`: purely brightness-based, ignores mode (major vs minor is the strongest valence signal)
	  - `acousticness`: inverse rolloff is a poor proxy; add harmonic-to-noise ratio or spectral flatness
	  - `instrumentalness`: ZCR alone is noisy; add MFCC variance (speech has high MFCC-1 std)
	  - After mode is now available: `valence += 0.3 if mode == 'major' else -0.3`
	- Add unit tests for output ranges [0,1] and edge cases (silence, pure tone, white noise)
2. [ ] **Emotion layer (new `track_emotions` table)**
	- Migration 004: `track_emotions (track_id PK FK, tension, melancholy, euphoria, aggression, serenity, nostalgia, sensuality, power DECIMAL(5,4), computed_at TIMESTAMP)`
	- Go struct `TrackEmotions` in `internal/db/emotions.go`
	- Worker step: after `SaveAudioFeatures` succeeds, call `ComputeAndSaveEmotions(ctx, trackID, features)` in same goroutine
	- Mapping matrix (starting point — tune after formula validation):
	  ```
	  tension    = 0.4*energy    + 0.3*(1-valence) + 0.2*bpm_norm  + 0.1*speechiness
	  melancholy = 0.5*(1-valence) + 0.3*(1-energy) + 0.2*(1-mode_is_major)
	  euphoria   = 0.4*valence   + 0.3*energy       + 0.2*danceability + 0.1*(1-speechiness)
	  aggression = 0.4*energy    + 0.3*(1-acousticness) + 0.2*bpm_norm + 0.1*(1-valence)
	  serenity   = 0.4*acousticness + 0.3*(1-energy) + 0.2*valence   + 0.1*(1-tension)
	  nostalgia  = 0.3*acousticness + 0.3*melodicness + 0.2*(1-valence) + 0.2*(1-danceability)
	  sensuality = 0.4*valence   + 0.3*(1-bpm_norm) + 0.2*melodicness + 0.1*liveness
	  power      = 0.5*energy    + 0.3*(1-acousticness) + 0.2*bpm_norm
	  ```
	  where `bpm_norm = min(1, bpm/200)` and `mode_is_major = 1 if mode == 'major' else 0`
	- API endpoints:
	  - `GET /api/v1/tracks/:id/emotions` → `TrackEmotions` JSON
	  - `GET /api/v1/emotions/distribution` → min/max/avg per dimension across all tracks
	- Future: manual user tagging/correction from UI
3. [ ] **Similarity graph computation**
	- Migration 005: `track_similarity (track_a_id INT, track_b_id INT, score DECIMAL(6,5), computed_at TIMESTAMP, PRIMARY KEY (track_a_id, track_b_id))`
	- Background Go job in worker (new goroutine, triggered after each analysis batch or via `POST /api/v1/similarity/recompute`):
	  1. Load all `track_emotions` rows into memory as float64 vectors
	  2. For each track, compute cosine similarity against all others (vectorized with BLAS or manual dot product)
	  3. Keep top-20 neighbours per track; upsert into `track_similarity`
	  4. For 10K tracks: ~50M comparisons, feasible in a few minutes in Go; shard by batch if needed
	- API: `GET /api/v1/tracks/:id/similar?n=10` → `[]TrackSimilarity{TrackID, Score, Track}`
	- Index: `CREATE INDEX ON track_similarity (track_a_id, score DESC)` for fast top-N lookup
4. [ ] **Unit test coverage to 80%+**
	- `internal/analyzer`: failure-mode tests (python errors, invalid JSON, timeout paths)
	- `internal/api`: pagination/filter/error response tests
	- `internal/db`: claim/retry/missing-status integration scenarios
5. [ ] **Consolidate deployment into root-level compose**
	- Create `~/srv/docker-compose.prod.yml` using `include:` directive
	- Move shared `doyou-network` and watchtower definitions to root file
	- Single `docker compose -f ~/srv/docker-compose.prod.yml up -d` starts everything
6. [ ] **Export API**
	- Add CSV + JSON exports for analyzed tracks
	- Support filters (`status`, pagination, optional feature thresholds)
7. [ ] **Bulk re-analysis endpoint**
	- Trigger re-analysis for `failed` and optionally `completed` older than threshold
	- Include dry-run mode and progress reporting
8. [ ] **Worker observability and resilience**
	- Add clearer progress metrics in logs (claimed/processed/failed rates)
	- Tune retry/backoff for transient analyzer failures
9. [ ] **Dual-repo dev ergonomics**
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

- [x] **Persisted raw audio features (v0.6.0, 2026-03-19)** — MFCCs (0–12 mean+std), spectral centroid/rolloff, ZCR, RMS energy were computed by librosa but silently discarded; migration 003 adds columns, `analyze_audio.py` now outputs them, `features.go` saves all 32 new fields
- [x] **Mode detection (v0.6.0, 2026-03-19)** — Added Krumhansl-Kessler tonal hierarchy profiles to `analyze_audio.py`; mode (major/minor) and mode_confidence now computed and saved; `audio_features.mode` column was always there but never written to
- [x] **Loudness dBFS (v0.6.0, 2026-03-19)** — `loudness_db` now computed from RMS mean (20·log₁₀); was in DB schema and Go struct but Python never emitted it

- [x] **Fixed `CleanupMissingTracks` bug (2026-03-16)** — nested `os.IsNotExist` AND logic silently skipped all missing tracks; replaced with `fileExists` flag that only requires the primary `fullPath` stat to succeed
- [x] **Set `CLEANUP_INTERVAL=10m` (2026-03-16)** — was `24h`; stale tracks now removed within 10 minutes of NAS deletion
- [x] **Fixed watchtower GHCR auth (2026-03-16)** — created static `~/.docker/watchtower-config.json` with inline credentials (osxkeychain inaccessible in containers)
- [x] **Fixed analytics proxy for remote users (2026-03-16)** — frontend bundle was hardcoding `localhost:8080`; added `/analytics-api/` nginx proxy block in scraper frontend, baked `VITE_ANALYTICS_API_URL=/analytics-api` via Dockerfile ARG
- [x] **Fixed DNS ambiguity 502 (2026-03-16)** — `proxy_pass http://api:3001` resolved to wrong container when frontend was on both Docker networks; changed to explicit `http://doyou-api:3001` and `http://doyou-analytics-api:8080/`
- [x] Removed duplicate `doyou-watchtower` from analytics compose (single watchtower instance in scraper stack)
- [x] Updated NAS path handling and scanner stability in production compose
- [x] Added explicit pre-push checklist for local validation before pipeline use
- [x] Established dual-repo local workflow guidance for analytics + scraper development
