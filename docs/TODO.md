# DoYou Music Analytics TODO

Active roadmap for the analytics service. Deployment/feature notes below reflect repository status reviewed on 2026-10-01; production rollout still requires operator verification.

## Remaining Steps (Immediate)

- [ ] **Deploy current release and verify production upgrade** — Current repository version is 0.10.0. Embedded migrations 001–007 are applied by the API at startup; test upgrade on a production-schema copy, then follow the operator rollout/rollback checklist. Do not replay migration 003 manually.
- [x] **Fix/validate heuristic feature formulas** — danceability uses beat regularity; acousticness uses spectral flatness; instrumentalness uses MFCC variation. Scores are deterministic heuristics, not trained ML predictions.
- [x] **Add emotion layer** — `track_emotions` table (migration 004); 8 dimensions computed deterministically in worker after each track; `GET /api/v1/tracks/:id/emotions` + bulk `GET /api/v1/emotions` + distribution endpoint
- [x] **Compute similarity graph** — `track_similarity` table (migration 005); `ComputeSimilarity` finds top-20 neighbours per track via cosine similarity in emotion space; worker recomputes every 30 min; `GET /api/v1/tracks/:id/similar?n=10` + `POST /api/v1/similarity/recompute` + `GET /api/v1/similarity/stats`
- [ ] **Redesign EmotionGraph as a hierarchical spatial map** — replace the compass-rose of floating orbs with emotion *regions/territories* on a 2D canvas; drilling into a region reveals progressively finer sub-clusters each layer adding one more feature dimension (e.g. melancholy → sadness → winter blues → rain); breadcrumb navigation to move between levels
- [ ] **AI taxonomy labeling + curation loop** — new "Curate" tab where a playing song gets an AI-proposed hierarchical label from a taxonomy tree; user accepts/adjusts/adds new nodes; all curations stored in DB; feedback retrains the emotion mapping weights over time
- [ ] Push package coverage toward 80%+ with priority on `internal/analyzer`, `internal/api`, `internal/db`
- [ ] **Playlist / Radio Show API** — Backend for the playlist builder in the scraper frontend EmotionGraph; playlists are user-ordered collections of track IDs with a name and optional notes
  - New table: `playlists (id, name, notes, track_ids JSONB ordered array, created_at, updated_at)`
  - Endpoints: `GET /api/v1/playlists`, `POST /api/v1/playlists`, `GET /api/v1/playlists/:id`, `PUT /api/v1/playlists/:id`, `DELETE /api/v1/playlists/:id`
- [ ] **Consolidate deployment** — create root-level `~/srv/docker-compose.prod.yml` using Compose `include:` to wire both stacks (one `docker compose up`, shared network, single watchtower); keep both repos independent
- [ ] Implement export endpoints (CSV/JSON) for analyzed tracks with practical filters
- [ ] Add safe bulk re-analysis flow for failed/outdated tracks (dry-run + progress)
- [ ] Improve worker observability (throughput/error metrics) and retry/backoff behavior
- [ ] Keep integration contracts stable for scraper dashboard consumption (`/stats`, `/stats/detailed`, `/tracks`)
- [ ] **Authentication (personal use → multi-user)**
  - Short term: token-based auth on the analytics API (currently unauthenticated behind nginx proxy)
  - Long term: per-user libraries — each user has their own track collection, analysis results, and dashboard
  - Design note: keep single-user mode as default; multi-user is an opt-in layer on top

## 5-Phase Roadmap

### Phase 1 — Stabilize & Ship
- [ ] Push 4 local unpushed commits (v0.12.0–v0.12.3) to origin → triggers CI + Watchtower
- [ ] Raise `internal/db` coverage from ~10% → 80%+ (mock-based tests for `ClaimPendingTrack`, cleanup, retry, missing-status paths)
- [ ] Maintain `internal/api` (91.5%) and `internal/analyzer` (100%) above 80%+ as code evolves
- [ ] Scraper frontend component test baseline: Vitest + RTL (currently 0% → 30%+)

### Phase 2 — Playlist / Radio Show API
- [ ] **Migration 008**: `playlists(id SERIAL PK, name TEXT NOT NULL, notes TEXT, track_ids JSONB ordered array, created_at TIMESTAMP, updated_at TIMESTAMP)`
- [ ] `internal/db/playlists.go`: `CreatePlaylist`, `GetPlaylist`, `ListPlaylists`, `UpdatePlaylist`, `DeletePlaylist`
- [ ] REST endpoints: `GET /api/v1/playlists`, `POST /api/v1/playlists`, `GET /api/v1/playlists/:id`, `PUT /api/v1/playlists/:id`, `DELETE /api/v1/playlists/:id`
- [ ] Mock tests for all 5 endpoints (table-driven, no live DB required)
- [ ] *Scraper frontend*: `PlaylistPanel` component in separate file; `EmotionGraph` only gets `onAddToPlaylist(node)` callback prop; click node → add to active playlist; ↑↓ reorder, × remove; named playlists (radio show title input); persist to `localStorage` first, sync to analytics API on save

### Phase 3 — Label Intelligence Pipeline
- [x] AI taxonomy labeling + curation loop (v0.12.x) — taxonomy tree, `track_labels`, curation endpoints, bulk auto-label — completed
- [ ] **Label profiles materialization**: `label_profiles(label_path TEXT PK, centroid JSONB, track_count INT, computed_at TIMESTAMP)`
- [ ] Background job: aggregate emotion vectors of all curated tracks per label path → per-label centroid; triggered on new curation batch saves
- [ ] New endpoint: `GET /api/v1/tracks/:id/suggest-label` → top-3 matching label paths + confidence score (cosine similarity to centroid)
- [ ] `internal/db/label_profiles.go` with upsert centroid logic
- [ ] *Scraper frontend*: replace rule-based `computeZoneSuggestions` in `CurateTab` with API-driven suggestions + confidence %; "Why?" tooltip showing top contributing features

### Phase 4 — 2D Projection for EmotionGraph Drill-down
- [ ] **Migration 006**: `track_projections(track_id INT REFERENCES tracks(id) PRIMARY KEY, proj_x FLOAT8, proj_y FLOAT8, zone TEXT, computed_at TIMESTAMP)`
- [ ] Python script `scripts/compute_projections.py`: UMAP on emotion vectors → 2D coordinates, batch-upsert into DB
- [ ] `POST /api/v1/projections/recompute` trigger + `GET /api/v1/projections?limit=10000` (streaming-friendly bulk fetch)
- [ ] Worker recomputes weekly (configurable via `PROJECTION_INTERVAL` env var)
- [ ] Tests: projection endpoint, recompute trigger, zone assignment roundtrip
- [ ] *Scraper frontend EmotionGraph*: replace physics simulation with scatter plot from precomputed `proj_x/proj_y`; 4-layer hierarchy (L0: 8 zones → L1: feature sub-clusters → L2: micro-clusters → L3: individual track nodes); breadcrumb nav ("All → Melancholy → Quiet Sadness"); smooth zoom animations between layers

### Phase 5 — Deployment Consolidation
- [ ] Create `~/srv/docker-compose.prod.yml` using Compose `include:` directive (references both per-repo compose files)
- [ ] Move shared `doyou-network` definition to root compose; keep per-repo composes working standalone for local dev
- [ ] Single `docker compose -f ~/srv/docker-compose.prod.yml up -d` starts entire stack
- [ ] Watchtower definition lives in root compose only (remove from per-repo prod files)
- [ ] Update `docs/DEPLOYMENT.md` with new single-command workflow

## Ongoing Backlog
- [ ] Export API: CSV + JSON exports for analyzed tracks with filters (`status`, pagination, feature thresholds)
- [ ] Bulk re-analysis endpoint: trigger re-analysis for `failed` and optionally stale `completed`; dry-run mode + progress reporting
- [ ] Worker observability: clearer progress metrics in logs (claimed/processed/failed rates); retry/backoff for transient analyzer failures
- [ ] Authentication: token-based auth on analytics API (currently unauthenticated behind nginx proxy); per-user libraries as opt-in later
- [ ] Show analytics version in scraper frontend footer (fetch `GET /analytics-api/api/v1/version` alongside scraper version)

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

## Scraper Frontend

- [ ] **Show analytics versions in footer** — the scraper frontend `Footer.tsx` currently shows backend + frontend versions from the scraper's own `/api/v1/version`. Extend it to also fetch `GET /analytics-api/api/v1/version` (proxied through nginx) and display the analytics API version alongside. Requires the analytics `/api/v1/version` endpoint to exist and return a `version` field (it does via the `VERSION` file baked into the image at build time).

## Completed Recently

- [x] **Similarity graph (v0.10.0, 2026-03-21)** — migration 005 adds `track_similarity` table; `ComputeSimilarity` loads all emotion vectors, computes pairwise cosine similarity, keeps top-20 neighbours per track and batch-upserts results; worker goroutine recomputes every 30 min; `GET /api/v1/tracks/:id/similar?n=10`, `POST /api/v1/similarity/recompute`, `GET /api/v1/similarity/stats` all wired up
- [x] **Fixed/validated high-level feature formulas (v0.9.x)** — danceability has beat-regularity component (onset CoV); valence incorporates mode shift; acousticness uses spectral flatness; instrumentalness uses MFCC-1 std
- [x] **Emotion layer (v0.9.0, 2026-03-21)** — migration 004; `ComputeEmotions` + `SaveEmotions` in worker; bulk `GET /api/v1/emotions`, per-track `GET /api/v1/tracks/:id/emotions`, `GET /api/v1/emotions/distribution`
- [x] **Auto-migration at API startup (v0.9.0)** — SQL files embedded via `//go:embed`; `RunMigrations` applies all pending `.up.sql` in sorted order at startup; eliminates manual migration steps

- [x] **Persisted raw audio features (v0.6.0, 2026-03-19)** — MFCCs (0–12 mean+std), spectral centroid/rolloff, ZCR, RMS energy were computed by librosa but silently discarded; migration 003 adds columns, `analyze_audio.py` now outputs them, `features.go` saves all 32 new fields
- [ ] **Key/mode detection** — historical Krumhansl-Kessler implementation is currently disabled; the analyzer emits null key/mode. Re-enable only with representative audio tests and document confidence/limitations.
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
