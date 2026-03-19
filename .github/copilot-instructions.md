# DoYou Music Analytics - Copilot Instructions

## Project Overview
This is an audio analysis and metadata extraction service for DoYou World Radio. It analyzes MP3 tracks from local storage, extracts 13 audio features using Python/librosa, stores results in PostgreSQL, and provides a REST API for querying analyzed tracks.

**Current Status (Mar 2026):**
- Running on Mac Mini alongside the scraper stack
- Tracks in DB vary — worker auto-imports from NAS on each scan, cleans up deleted files every 10 minutes
- Worker processes 6 tracks in parallel with librosa/Python
- Analytics API is proxied via the scraper frontend nginx at `/analytics-api/` (not exposed directly)

## 📚 Always Read These Docs First

Before making changes, read the relevant docs in `docs/`:

- **`docs/TODO.md`** — current priorities and roadmap
- **`docs/DUAL_REPO_DEVELOPMENT.md`** — how this repo and `doyou-world-scraper` work together locally and in production
- **`docs/PREPUSH_CHECKLIST.md`** — required checks before every push
- **`docs/DEPLOYMENT.md`** — production setup, watchtower, docker commands
- **`docs/TESTING.md`** — test patterns and coverage targets
- **`docs/API_USAGE.md`** — full API reference
- **`docs/DATABASE_USAGE.md`** — schema, queries, integration patterns
- **`docs/SCRAPER_INTEGRATION.md`** — how the scraper consumes the analytics API

## Tech Stack
- **Backend**: Go 1.25+ with standard library net/http + gorilla/mux router
- **Audio Analysis**: Python 3.11 + librosa v0.10+ + soundfile + ffmpeg
- **Database**: PostgreSQL 16 with pgx driver
- **Worker**: Continuous processor with 6 parallel workers (M2 optimized)
- **Storage**: `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks/` (13,542 MP3 files, ~65GB)
- **API**: REST API with JSON responses, CORS enabled
- **Deployment**: Docker Compose + GitHub Actions (GHCR) + Watchtower auto-updates
- **Logging**: Structured logging with zerolog
- **Config**: Centralized configuration via environment variables (.env file)

## Project Structure (Clean Architecture)
```
doyou-music-analytics/
├── cmd/                    # Application entry points
│   ├── api/                # REST API server
│   │   └── main.go         # HTTP server, routes, graceful shutdown
│   ├── worker/             # Track analyzer worker
│   │   └── main.go         # Batch processor with parallel workers
│   ├── migrate/            # Database migration tool
│   │   └── main.go         # Run SQL migrations
│   └── import-tracks/      # Track importer (legacy, not in repo)
├── internal/               # Private application code
│   ├── api/                # HTTP handlers
│   │   ├── handlers.go     # GET /health, /tracks, /search, /stats
│   │   └── handlers_test.go # (30 tests, 29.7% coverage)
│   ├── analyzer/           # Audio analysis engine
│   │   ├── analyzer.go     # Python subprocess integration (13 features)
│   │   └── analyzer_test.go # (1 test, 5.6% coverage)
│   ├── config/             # Configuration management
│   │   ├── config.go       # Load env vars, validate, set defaults
│   │   └── config_test.go  # (11 tests, 91.9% coverage)
│   ├── db/                 # Database layer
│   │   ├── db.go           # Connection pool, migrations
│   │   ├── tracks.go       # Track CRUD operations + atomic claiming
│   │   └── db_test.go      # (5 tests, 24.8% coverage)
│   └── storage/            # File operations
│       └── storage.go      # Scan NAS, verify MP3 files
├── pkg/                    # Public/reusable packages
│   └── logger/             # Structured logging
│       ├── logger.go       # Zerolog wrapper (Info, Error, Debug, Fatal)
│       └── logger_test.go  # (4 tests, 32.8% coverage)
├── scripts/                # Python analysis scripts
│   ├── analyze_audio.py    # librosa feature extraction (BPM, key, energy, etc.)
│   └── requirements.txt    # Python dependencies
├── migrations/             # SQL migrations
│   ├── 001_initial_schema.up.sql   # Create tables (tracks, audio_features)
│   └── 001_initial_schema.down.sql # Drop tables
├── deployments/            # Deployment configs
│   ├── docker/             # Dockerfiles
│   │   ├── Dockerfile.api      # Alpine-based API image
│   │   └── Dockerfile.worker   # Python 3.11-slim worker image
│   └── mac-mini/           # Mac mini environment config
│       └── .env.example    # Sample environment variables
├── docs/                   # Documentation
│   ├── ANALYTICS_ARCHITECTURE.md # System architecture and design
│   ├── DEPLOYMENT.md       # Deployment guide (includes local testing)
│   └── TODO.md             # Project roadmap
├── .github/                # GitHub configuration
│   ├── workflows/          # CI/CD pipelines
│   │   └── build.yml       # Test → Build → Push to GHCR
│   └── copilot-instructions.md # This file
├── docker-compose.yml      # Local development setup
├── docker-compose.prod.yml # Production setup with Watchtower
├── go.mod                  # Go dependencies
├── go.sum                  # Go dependency checksums
├── VERSION                 # Backend version (X.Y.Z format)
└── README.md               # Project documentation
```

## Key Functionality

### 1. Track Importing
- Scans NAS directory for MP3 files
- Imports track metadata to PostgreSQL
- Status tracking: `pending`, `analyzing`, `completed`, `failed`
- Skip already imported tracks

### 2. Audio Analysis (13 Features)
- **Tempo/Rhythm**: BPM (Beats Per Minute), Duration
- **Tonality**: Key, Mode (Major/Minor)
- **Energy/Mood**: Energy, Danceability, Valence, Acousticness
- **Vocals**: Instrumentalness, Speechiness, Liveness
- **Melodic**: Melodicness
- **Dynamics**: Loudness (dB)
- Powered by: Python 3.11 + librosa + ffmpeg

### 3. Parallel Processing
- 4 concurrent workers analyzing tracks simultaneously
- Atomic track claiming prevents race conditions
- Automatic retry on failure
- Progress logging with track details

### 4. Automatic Cleanup
- Detects files deleted from NAS
- Two modes: "mark" (set status='missing') or "delete" (remove from DB)
- Configurable interval via `CLEANUP_INTERVAL` (default: 10m — production uses `delete` action)
- Keeps statistics accurate; tracks removed from NAS disappear from DB within 10 minutes

### 5. REST API
- Query analyzed tracks with audio features
- Search tracks by title, artist, album
- Get statistics (total tracks, analyzed, pending, failed)
- Health checks for monitoring

## API Endpoints
- `GET /health` - Health check (API + database connection)
- `GET /api/v1/stats` - Get analysis statistics (track counts by status)
- `GET /api/v1/stats/detailed` - Get detailed statistics (includes processing metrics)
- `GET /api/v1/tracks` - List all tracks (paginated, default 50 per page, supports `?limit=100&offset=0&status=completed`)
- `GET /api/v1/tracks/:id` - Get single track with audio features
- `GET /api/v1/tracks/search?q=query` - Search tracks by title/artist/album
- `GET /api/v1/tracks/failed?limit=50` - Get failed tracks with error messages

**Stats Response (`/api/v1/stats`):**
```json
{
  "total": 9875,
  "pending": 8234,
  "analyzing": 4,
  "completed": 1620,
  "failed": 17
}
```

**Detailed Stats Response (`/api/v1/stats/detailed`):**
```json
{
  "total": 9875,
  "pending": 8234,
  "analyzing": 4,
  "completed": 1620,
  "failed": 17,
  "avg_processing_time_sec": "2.34",
  "tracks_per_hour": "256.78",
  "completion_percentage": "16.40"
}
```

**Track Response Format:**
```json
{
  "id": 123,
  "title": "Track Name",
  "artist": "Artist Name",
  "album": "Album Name",
  "file_path": "/mnt/music/downloads/track.mp3",
  "status": "completed",
  "audio_features": {
    "bpm": 161.5,
    "key": "D#",
    "mode": "Minor",
    "duration": 301.5,
    "energy": 0.3989,
    "danceability": 0.6012,
    "valence": 0.1546,
    "acousticness": 0.0452,
    "instrumentalness": 0.987,
    "speechiness": 0.0234,
    "liveness": 0.1123,
    "melodicness": 0.456,
    "loudness": -8.5
  }
}
```

## Database Usage & Integration

### Connection Details
```bash
Host: localhost (or doyou-analytics-db from Docker network)
Port: 5432
Database: doyou_music
User: doyou
Password: REDACTED_ROTATED_CREDENTIAL
```

### Common Queries for Scraper Integration

**Get All Completed Analyses:**
```sql
SELECT t.file_name, t.artist, t.title, af.bpm, af.key, af.energy, af.danceability, af.valence
FROM tracks t
JOIN audio_features af ON t.id = af.track_id
WHERE t.analysis_status = 'completed'
ORDER BY t.analyzed_at DESC;
```

**Find High Energy Danceable Tracks:**
```sql
SELECT t.file_name, af.bpm, af.energy, af.danceability
FROM tracks t
JOIN audio_features af ON t.id = af.track_id
WHERE t.analysis_status = 'completed'
  AND af.energy > 0.7
  AND af.danceability > 0.8
ORDER BY af.danceability DESC;
```

**Check if Specific Track is Analyzed:**
```sql
SELECT analysis_status, analyzed_at 
FROM tracks 
WHERE file_path = '/mnt/music/downloads/doyou-world-radio-tracks/filename.mp3';
```

**Get Progress Stats:**
```sql
SELECT analysis_status, COUNT(*) as count
FROM tracks
GROUP BY analysis_status;
```

### Python Integration Example (for Scraper)
```python
import psycopg2
from psycopg2.extras import RealDictCursor

conn = psycopg2.connect(
    host="localhost",
    port=5432,
    database="doyou_music",
    user="doyou",
    password="REDACTED_ROTATED_CREDENTIAL"
)

# Get completed analyses
with conn.cursor(cursor_factory=RealDictCursor) as cur:
    cur.execute("""
        SELECT t.file_name, af.bpm, af.key, af.energy
        FROM tracks t
        JOIN audio_features af ON t.id = af.track_id
        WHERE t.analysis_status = 'completed'
        ORDER BY t.analyzed_at DESC LIMIT 100
    """)
    tracks = cur.fetchall()
    for track in tracks:
        print(f"{track['file_name']}: {track['bpm']} BPM, {track['key']} key")
```

**📖 Full Database Documentation:** See [docs/DATABASE_USAGE.md](../docs/DATABASE_USAGE.md) for:
- Complete schema details
- All audio feature descriptions (what each 0-1 value means)
- Advanced query examples
- Integration patterns
- Monitoring commands

## Development Guidelines

### Running Locally (Development)
```bash
# Start PostgreSQL
docker-compose up -d postgres

# Run API server
go run ./cmd/api

# Run Worker (in separate terminal)
go run ./cmd/worker

# Or run both with docker-compose
docker-compose up
```

### 🧪 Pre-Push Local Testing (CRITICAL)
**ALWAYS test Docker builds locally before pushing to prevent production issues!**

See [docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md#🧪-pre-push-local-testing) for detailed workflow:

1. **Build both Docker images** (API + Worker)
2. **Run `docker-compose up`** to test locally
3. **Verify health checks** (API, database, worker logs)
4. **Test with sample tracks** (optional)
5. **Complete pre-push checklist** (tests, builds, health checks)
6. **Only push after validation passes** ✅

**Quick Commands:**
```bash
# Build images
docker build -f deployments/docker/Dockerfile.api -t doyou-analytics-api:local .
docker build -f deployments/docker/Dockerfile.worker -t doyou-analytics-worker:local .

# Test locally
docker-compose up -d
curl http://localhost:8080/health

# Check logs
docker-compose logs -f worker

# Clean up
docker-compose down
```

> **Remember:** Every push to `main` triggers GitHub Actions → Watchtower auto-deploys to Mac mini within 5-10 minutes!


### Testing Requirements
**CRITICAL**: This project follows Test-Driven Development (TDD). All features MUST have tests written first.

**Test-Driven Development Workflow (MANDATORY):**
1. **Write failing test first** - Before writing any production code
2. **Run test and verify it fails** - `go test -v ./...`
3. **Write minimal code** to make the test pass
4. **Run test and verify it passes**
5. **Refactor** if needed while keeping tests green
6. **Update test stats** in the documentation (test count and coverage %)
7. **Commit** with test and implementation together

**Test Coverage Goals:**
- Maintain **80%+ test coverage** for all Go code (currently at ~50%)
- Write tests for **every new function** - no exceptions
- Update existing tests when modifying functionality
- Run tests before every commit: `go test ./...`
- Never commit production code without corresponding tests

**Testing Standards:**
- Unit tests: Test individual functions in isolation
- Integration tests: Test API endpoints end-to-end
- Use table-driven tests for multiple scenarios
- Mock external dependencies (Python subprocess, file system when appropriate)
- Use `t.TempDir()` for file system tests
- Add `t.Skip()` for tests that require network/external resources (e.g., DATABASE_URL)
- Test both happy path and error cases
- Use meaningful test names: `TestFunctionName_Scenario_ExpectedBehavior`

**Test Files Location:**
- `internal/*_test.go` - Test files (handlers_test.go, analyzer_test.go, config_test.go, db_test.go, scanner_test.go)
- `pkg/*_test.go` - Package tests (logger_test.go)
- Run: `go test ./...`
- Coverage: `go test -cover ./...`
- Count tests: `go test -v ./... 2>&1 | grep -c "^=== RUN"`
  - **Current Stats**: 142 tests, ~70% average coverage
    - internal/api: 23 tests, 91.5% coverage
    - internal/analyzer: 10 tests, 100% coverage
    - internal/config: 11 tests, 91.9% coverage
    - internal/db: 54 tests, ~10% coverage (skipped without DATABASE_URL)
    - pkg/logger: 13 tests, 51.7% coverage

**When Adding Features:**
1. Write tests first (TDD approach preferred)
2. Implement feature
3. Ensure tests pass: `go test ./...`
4. Check coverage: `go test -cover ./...`
5. **Bump version number** in `VERSION` (PATCH for bugs, MINOR for features, MAJOR for breaking changes)
6. **Update test stats in this file** - Update lines above with new test count and coverage
7. Commit with meaningful message (include version in commit if bumped)

**Documentation Maintenance:**
- **ALWAYS update test coverage statistics** after adding/modifying tests
- Run `go test -cover ./...` to get coverage percentage
- Run `go test -v ./... 2>&1 | grep -c "^=== RUN"` to count total tests
- Update format: `# 99 tests passing, ~50% coverage`
- Keep API documentation synchronized with actual endpoints
- Update `docs/TODO.md` when completing tasks or adding new ones

**Remember:** Tests are not optional. They are part of the feature. A feature without tests is an incomplete feature.



## Dependencies

### Backend (Go)
```go
github.com/gorilla/mux          // HTTP router
github.com/jackc/pgx/v5         // PostgreSQL driver
github.com/rs/zerolog           // Structured logging
```

### Python (Audio Analysis)
```python
librosa>=0.10.0                 # Audio analysis
soundfile>=0.12.1               # Audio file I/O
numpy>=1.24.0                   # Numerical computing
```

### System Dependencies
- **Python 3.11+** - Required for librosa
- **ffmpeg** - Audio decoding backend
- **libsndfile1** - Sound file library
- **PostgreSQL 16** - Database

Install Go packages: `go get <package>`
Install Python packages: `pip install -r scripts/requirements.txt`

## Development Guidelines

### Code Style
- Use Go standard formatting (`gofmt`)
- Handle errors explicitly (return error values)
- Clean up resources with `defer` statements
- Use structured logging via pkg/logger package
- Use config package for all path references (no hardcoded paths)
- Never hardcode Python path - use PYTHON_PATH env var

### Common Patterns
```go
// Database pattern with atomic operations
func (db *Database) ClaimPendingTrack(ctx context.Context) (*Track, error) {
    tx, err := db.pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)
    
    // Use FOR UPDATE SKIP LOCKED for atomic claiming
    var track Track
    err = tx.QueryRow(ctx, query).Scan(&track.ID, &track.Title, ...)
    if err != nil {
        return nil, err
    }
    
    // Update status in same transaction
    _, err = tx.Exec(ctx, updateQuery, track.ID)
    if err != nil {
        return nil, err
    }
    
    return &track, tx.Commit(ctx)
}

// Handler pattern with error handling
func (h *Handlers) GetTracks(w http.ResponseWriter, r *http.Request) {
    tracks, err := h.db.GetTracks(r.Context())
    if err != nil {
        logger.Error("Failed to get tracks", "error", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tracks)
}

// Python subprocess for analysis
func (a *Analyzer) AnalyzeTrack(filePath string) (*AudioFeatures, error) {
    cmd := exec.Command(a.pythonPath, a.scriptPath, filePath)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("analysis failed: %w", err)
    }
    
    var features AudioFeatures
    if err := json.Unmarshal(output, &features); err != nil {
        return nil, fmt.Errorf("parse failed: %w", err)
    }
    
    return &features, nil
}
```

## Important Notes
- **NAS Path (machine-specific)**: Set `NAS_MUSIC_PATH` in `.env` to the host path on each machine — this is gitignored and differs per machine:
  - Mac Mini (production): `NAS_MUSIC_PATH=/Volumes/Expansion/ExtHDD-8TB/Music`
  - Local dev machine: `NAS_MUSIC_PATH=/wherever/your/nas/is/mounted`
- **Container NAS Path**: Always `/mnt/music/downloads` inside the container — `docker-compose.prod.yml` mounts `NAS_MUSIC_PATH` → `/mnt/music/downloads`
- **Without Docker (local Go binary)**: Set `NAS_MOUNT_PATH` directly to the music path, set `VALIDATE_NAS_ON_STARTUP=false`
- **Analytics proxy**: In production the analytics API is NOT exposed directly — nginx in the scraper frontend proxies `/analytics-api/` → `doyou-analytics-api:8080`. Both stacks share `doyou-network`.
- **Analysis Time**: ~2 seconds per track with parallel processing
- **Parallel Workers**: 4 workers process tracks simultaneously
- **Race Condition Prevention**: Uses `ClaimPendingTrack()` with `FOR UPDATE SKIP LOCKED`
- **Python Version**: Must be 3.11+ for librosa compatibility
- **Deployment**: Watchtower polls every 5 minutes, auto-deploys new images
- **CI/CD**: GitHub Actions runs tests before building Docker images

## Version Management (⚠️ CRITICAL - DO THIS BEFORE EVERY COMMIT)
**ALWAYS increment version when committing changes - no exceptions:**
- Version file: Update `VERSION` file in project root (format: `X.Y.Z`)
- Version format: `MAJOR.MINOR.PATCH` (e.g., `0.5.0` → `0.5.1` for bug fixes)
- **Increment rules (follow strictly):**
  - PATCH (0.5.0 → 0.5.1): Bug fixes, performance improvements, minor refactors
  - MINOR (0.5.1 → 0.6.0): New features, new endpoints, significant enhancements
  - MAJOR (0.6.0 → 1.0.0): Breaking API changes, major architecture changes
- **Why this matters:**
  - Docker images are tagged with version from VERSION file
  - Helps track which code is running in production
  - Watchtower detects new images based on tags
- **Before every commit:** Open `VERSION`, increment appropriately, save, then commit

## Next Steps (from TODO.md)
1. ✅ ~~Phase 1-5: Database, API, Worker with parallel processing~~ - Completed
2. **Phase 6: Search and filtering** - Add advanced query capabilities
3. **Phase 7: Export functionality** - CSV/JSON exports of analyzed tracks
4. **Phase 8: Batch re-analysis** - Re-analyze failed or outdated tracks
5. **Phase 9: Track metadata enrichment** - Genre, mood classification
6. **Phase 10: Frontend dashboard** - Browse and visualize audio features
