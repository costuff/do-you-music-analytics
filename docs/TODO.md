# DoYou Music Analytics TODO

Active roadmap for the analytics service, synchronized with the scraper integration state as of 2026-03-16.

## Remaining Steps (Immediate)

- [ ] **Deploy v0.6.0 to production** — Watchtower will pull the new image; then apply migration 003 manually: `docker exec -it doyou-analytics-db psql -U doyou -d doyou_music -f /migrations/003_add_raw_features.up.sql`; then reset all `analysis_version = '1.0.0'` tracks to `pending` so they get re-analyzed with MFCCs + mode + accurate loudness_db
- [x] **Fix/validate high-level feature formulas** — danceability now uses beat-regularity (CoV of onset strength); valence incorporates mode shift; acousticness uses spectral flatness; instrumentalness uses MFCC-1 std
- [x] **Add emotion layer** — `track_emotions` table (migration 004); 8 dimensions computed deterministically in worker after each track; `GET /api/v1/tracks/:id/emotions` + bulk `GET /api/v1/emotions` + distribution endpoint
- [x] **Compute similarity graph** — `track_similarity` table (migration 005); `ComputeSimilarity` finds top-20 neighbours per track via cosine similarity in emotion space; worker recomputes every 30 min; `GET /api/v1/tracks/:id/similar?n=10` + `POST /api/v1/similarity/recompute` + `GET /api/v1/similarity/stats`
- [ ] **Redesign EmotionGraph as a hierarchical spatial map** — replace the compass-rose of floating orbs with emotion *regions/territories* on a 2D canvas; drilling into a region reveals progressively finer sub-clusters each layer adding one more feature dimension (e.g. melancholy → sadness → winter blues → rain); breadcrumb navigation to move between levels
- [ ] **AI taxonomy labeling + curation loop** — new "Curate" tab where a playing song gets an AI-proposed hierarchical label from a taxonomy tree; user accepts/adjusts/adds new nodes; all curations stored in DB; feedback retrains the emotion mapping weights over time
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

0. [ ] **Redesign EmotionGraph as a hierarchical spatial map** *(scraper frontend)*
	- **Problem with current layout**: 8 poles at compass positions + force-physics orbs is visually noisy; emotion poles don't feel like meaningful territories
	- **New layout concept**: 2D canvas where each emotion occupies a *region/zone*; tracks rendered as points inside their dominant-emotion region; the graph has gravity wells, not poles
	- **Hierarchy / drill-down UX**:
	  - Layer 0 (overview): 8 emotion zones visible as coloured territories (melancholy NW, serenity NE, euphoria SE, etc.)
	  - Layer 1 (click zone): sub-clusters based on the top-2 audio features in that zone (e.g. melancholy splits into acousticness × tempo quadrants → labels like "quiet sadness", "driving grief")
	  - Layer 2 (click sub-cluster): micro-clusters using 2 more features (e.g. spectral centroid + instrumentalness → "rainy instrumental", "winter blues vocals")
	  - Layer 3 (leaf): individual track nodes with title/artist tooltip
	- **Navigation**: breadcrumb trail at top ("All → Melancholy → Quiet Sadness"), back button, smooth zoom animation between layers
	- **Taxonomy labels**: at layers 1–2 use auto-generated feature-combination names; also allow user to rename any cluster (stored as `cluster_labels` in localStorage first, then DB later)
	- **Backend needed**: precompute 2D UMAP/t-SNE projection of emotion vectors server-side; store `(track_id, proj_x, proj_y)` in a new `track_projections` table (migration 006); expose `GET /api/v1/projections` for bulk fetch — avoids doing layout math in the browser for 10K tracks
	- Design question: fixed zone positions (hand-coded emotion→quadrant map) vs fully data-driven layout — start with fixed, move to data-driven once projection endpoint exists
1. [ ] **AI taxonomy labeling + active-learning curation tab** *(scraper frontend + analytics backend)*
	- **Concept**: a "Curate" tab where you play a track and are shown an AI-proposed hierarchical label; you accept, adjust, or push the label deeper/shallower; the system learns from your corrections
	- **Taxonomy tree structure**: JSON tree of named nodes, root = top-level emotions, children = finer sub-genres/moods; stored in DB table `taxonomy_nodes (id, parent_id, name, description, created_at, source ENUM('auto','user'))` — migration 007
	- **Auto-labeling flow**:
	  1. Given a track's emotion vector, traverse the taxonomy tree; at each level pick the best-fitting child node using cosine similarity to that node's centroid
	  2. Stop when confidence drops below threshold or leaf reached
	  3. Return the path: `["melancholy", "sadness", "winter blues"]`
	- **Curation storage**: `track_labels (id, track_id, label_path JSONB, auto_label_path JSONB, curated_by TEXT, notes TEXT, created_at TIMESTAMP)` — migration 007 (same migration)
	- **Analytics API endpoints**:
	  - `GET /api/v1/taxonomy` → full taxonomy tree as nested JSON
	  - `POST /api/v1/taxonomy/nodes` → add a new taxonomy node (user-defined)
	  - `GET /api/v1/tracks/:id/label` → current auto + user label
	  - `POST /api/v1/tracks/:id/label` → save a user curation
	  - `GET /api/v1/labels?path=melancholy/sadness` → all tracks under a taxonomy path
	- **Training feedback loop**: after N new user curations, a background go routine adjusts the emotion mapping matrix weights via a simple gradient step (compare `auto_label_path` vs `label_path`, backprop through the deterministic formula); persist adjusted weights as a JSON config in DB (`model_config` table), reload on next startup
	- **UX in Curate tab**:
	  - Left panel: track player (embed or trigger yt-dlp URL) + metadata
	  - Right panel: taxonomy tree with highlighted auto-proposed path; user clicks nodes to change; text field to add new taxonomy node inline; "Accept" / "Skip" / "Re-label" buttons
	  - Progress counter: "X tracks curated, model updated Y times"
	- Start simple: localStorage curation queue + manual API save; add feedback loop once >50 curations exist
2. [ ] **Unit test coverage to 80%+**
	- `internal/analyzer`: failure-mode tests (python errors, invalid JSON, timeout paths)
	- `internal/api`: pagination/filter/error response tests
	- `internal/db`: claim/retry/missing-status integration scenarios
3. [ ] **Consolidate deployment into root-level compose**
	- Create `~/srv/docker-compose.prod.yml` using `include:` directive
	- Move shared `doyou-network` and watchtower definitions to root file
	- Single `docker compose -f ~/srv/docker-compose.prod.yml up -d` starts everything
4. [ ] **Export API**
	- Add CSV + JSON exports for analyzed tracks
	- Support filters (`status`, pagination, optional feature thresholds)
5. [ ] **Bulk re-analysis endpoint**
	- Trigger re-analysis for `failed` and optionally `completed` older than threshold
	- Include dry-run mode and progress reporting
6. [ ] **Worker observability and resilience**
	- Add clearer progress metrics in logs (claimed/processed/failed rates)
	- Tune retry/backoff for transient analyzer failures
7. [ ] **Dual-repo dev ergonomics**
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

## Scraper Frontend

- [ ] **Show analytics versions in footer** — the scraper frontend `Footer.tsx` currently shows backend + frontend versions from the scraper's own `/api/v1/version`. Extend it to also fetch `GET /analytics-api/api/v1/version` (proxied through nginx) and display the analytics API version alongside. Requires the analytics `/api/v1/version` endpoint to exist and return a `version` field (it does via the `VERSION` file baked into the image at build time).

## Completed Recently

- [x] **Similarity graph (v0.10.0, 2026-03-21)** — migration 005 adds `track_similarity` table; `ComputeSimilarity` loads all emotion vectors, computes pairwise cosine similarity, keeps top-20 neighbours per track and batch-upserts results; worker goroutine recomputes every 30 min; `GET /api/v1/tracks/:id/similar?n=10`, `POST /api/v1/similarity/recompute`, `GET /api/v1/similarity/stats` all wired up
- [x] **Fixed/validated high-level feature formulas (v0.9.x)** — danceability has beat-regularity component (onset CoV); valence incorporates mode shift; acousticness uses spectral flatness; instrumentalness uses MFCC-1 std
- [x] **Emotion layer (v0.9.0, 2026-03-21)** — migration 004; `ComputeEmotions` + `SaveEmotions` in worker; bulk `GET /api/v1/emotions`, per-track `GET /api/v1/tracks/:id/emotions`, `GET /api/v1/emotions/distribution`
- [x] **Auto-migration at API startup (v0.9.0)** — SQL files embedded via `//go:embed`; `RunMigrations` applies all pending `.up.sql` in sorted order at startup; eliminates manual migration steps

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
