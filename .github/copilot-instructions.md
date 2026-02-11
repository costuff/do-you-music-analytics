# DoYou World Radio Scraper - Copilot Instructions

## Project Overview
This is a full-stack music scraper and management system for DoYou World Radio. It scrapes daily playlists from doyoutrackid.com, downloads tracks from streaming platforms, manages duplicates, and provides a REST API for automation.

## Tech Stack
- **Backend**: Go 1.23+ with Fiber v2 web framework
- **Scraping**: chromedp (headless Chrome) for web scraping
- **Downloading**: yt-dlp for audio extraction from SoundCloud/YouTube/Spotify/Apple Music
- **Storage**: Local filesystem + NAS backup at `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- **API**: REST API with async job tracking + WebSocket support
  - **Local Dev**: Port 3002 (avoids VS Code conflict on 3001)
  - **Docker**: Port 3000 (isolated container environment)
- **Frontend**: React 18 + TypeScript + Vite + Tailwind CSS (port 5173)
- **Logging**: Structured logging with zerolog
- **Config**: Centralized configuration via environment variables
- **Deployment**: Docker + Raspberry Pi/CasaOS compatible

## Project Structure (Go Standard Layout)
```
doyou-world-scraper/
├── cmd/
│   └── api/                # API server entry point
│       ├── main.go         # Application bootstrap with dependency injection
│       └── main_test.go    # (1 test)
├── internal/               # Private application code
│   ├── api/                # HTTP handlers and routes
│   │   ├── handlers.go     # HTTP request handlers (Handlers struct with methods)
│   │   └── handlers_test.go # Handler tests (18 tests)
│   ├── config/             # Configuration management
│   │   ├── config.go       # Config struct, Load(), Validate(), EnsureDirectories()
│   │   └── config_test.go  # (18 tests, 96.4% coverage)
│   ├── downloader/         # Track downloading logic
│   │   ├── downloader.go   # DownloadFromCSV, DownloadFromFolder
│   │   └── downloader_test.go # (79 tests)
│   ├── jobs/               # Job tracking and WebSocket
│   │   ├── jobs.go         # NewJob, GetJob, GetAllJobs, CancelJob, CleanupOldJobs
│   │   ├── websocket.go    # WebSocket hub for real-time updates
│   │   └── jobs_test.go    # (13 tests)
│   ├── scraper/            # Web scraping logic
│   │   ├── scraper.go      # ScrapeSingleDate, ScrapeMonth, ScrapeYear
│   │   └── scraper_test.go # (3 tests)
│   └── storage/            # File operations and utilities
│       ├── utilities.go    # RemoveDuplicates, MoveToNAS, ClearCSVs, ClearDownloads, GetTracks
│       └── utilities_test.go # (9 tests)
├── pkg/                    # Public/reusable packages
│   └── logger/             # Structured logging
│       ├── logger.go       # Zerolog wrapper with Init(), Info(), Error(), Debug(), etc.
│       └── logger_test.go  # (13 tests, 74.1% coverage)
├── frontend/               # React frontend dashboard
│   ├── src/
│   │   ├── components/     # React components (JobCard, ScrapeControls, etc.)
│   │   ├── hooks/          # Custom hooks (useWebSocket)
│   │   ├── services/       # API client (axios)
│   │   ├── types/          # TypeScript types
│   │   ├── App.tsx         # Main app component
│   │   └── main.tsx        # Entry point
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── README.md
├── data/                   # Data directory (gitignored)
│   ├── csv/                # Scraped CSV files (tracks-YYYY-MM-DD.csv)
│   └── downloads/          # Downloaded MP3 files (~10,000 files, 40GB)
├── api/                    # Legacy API code (kept for backward compatibility)
│   ├── *_test.go           # Legacy tests (88 tests)
│   └── README.md           # API documentation
├── main.go                 # CLI scraper (standalone, legacy)
├── download_soundcloud.go  # CLI downloader (standalone, legacy)
├── remove_duplicates.go    # CLI deduplicator (standalone, legacy)
├── match_downloads_to_csv.go # CLI matcher (standalone, legacy)
├── docs/                   # Documentation
│   ├── TODO.md             # Project roadmap
│   ├── DEVELOPMENT.md      # Development guide
│   ├── DOCKER.md           # Docker guide
│   └── PI-SETUP.md         # Raspberry Pi setup
├── api/Dockerfile          # API Docker image
├── frontend/Dockerfile     # Frontend Docker image
├── docker-compose.yml      # Docker orchestration
├── package.json            # Root package.json for dev workflow
├── go.mod                  # Go dependencies
└── README.md               # Project documentation
```

## Key Functionality

### 1. Scraping
- Scrapes track listings from `https://doyoutrackid.com/tracks/YYYY-MM-DD`
- Extracts: Title, Artist, Album, Label, ReleaseDate, Listen URL
- Saves to `csv/tracks-YYYY-MM-DD.csv`
- Supports single date, month, or full year scraping
- API: `POST /api/v1/scrape/date` `POST /api/v1/scrape/month` `POST /api/v1/scrape/year`

### 2. Downloading
- Resolves `lis.tn` short URLs to playable platform URLs
- Priority order: SoundCloud > YouTube > YouTube Music > Spotify > Apple Music
- Downloads as MP3 with embedded metadata (title, artist, album, label, date)
- Skips already downloaded files
- Tracks failed downloads in `downloads/failed_songs.csv`
- API: `POST /api/v1/download/csv` `POST /api/v1/download/all`

### 3. Deduplication
- Scans both `downloads/` and NAS folders
- Uses MD5 hashing to detect duplicate MP3 files
- Removes duplicates automatically
- API: `POST /api/v1/util/deduplicate`

### 4. Matching
- Fuzzy matches downloaded MP3 files to CSV entries
- Uses word overlap scoring on title and artist
- Identifies which CSV has the most matches (likely source)
- Shows top 20 CSVs and latest date among top matches
- API: `POST /api/v1/util/match`

### 5. Job Tracking
- All long-running operations use async jobs with UUIDs
- Job statuses: `running`, `completed`, `failed`
- Progress tracking with percentage updates
- Jobs auto-cleanup after 24 hours
- API: `GET /api/v1/jobs` `GET /api/v1/jobs/:id`

## API Endpoints
- `GET /api/v1/health` - Health check
- `PEndpoint**: `ws://localhost:3000/api/v1/ws`
- **Message Format**: `{"type": "job_update", "data": {...job}}`
- **Auto-reconnect**: Frontend automatically reconnects on disconnect
- **Real-time Updates**: Job progress broadcasts to all connected clients
- **Ping/Pong**: 30-second heartbeat to keep connections alive

## Frontend Architecture
- **Framework**: React 18 + TypeScript + Vite
- **Styling**: Tailwind CSS with dark slate theme
- **State Management**: React hooks (useState, useEffect, useCallback)
- **API Client**: Axios with proxy to backend
- **WebSocket**: Custom useWebSocket hook for real-time job updates
- **Components**:
  - `ScrapeControls`: Date/month/year pickers with scrape buttons
  - `DownloadControls`: Buttons for downloads and utilities
  - `StatsPanel`: Real-time statistics dashboard (auto-refresh every 30s)
  - `JobsList`: Live job cards with progress bars
  - `JobCard`: Individual job display with status colors
  - `ConnectionStatus`: WebSocket connection indicator

## Data & Statistics
- **CSV Files**: Stored in `csv/` directory
- **Downloads**: 9,875 MP3 files (~40GB) in `downloads/` folder
- **NAS Backup**: 4.3GB at `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- **Match Rate**: ~68% of files match to latest scraped CSV

## Development Guidelines

### Running Both Servers
```bash
# Option 1: Single command (recommended)
npm run dev

# Option 2: Separate terminals
# Terminal 1:
go run ./cmd/api
# Terminal 2:
cd frontend && npm run dev

# Option 3: VS Code tasks
# Cmd+Shift+P → Tasks: Run Task → Start All Servers
```

### Testing Requirements
**CRITICAL**: This project follows Test-Driven Development (TDD). All features MUST have tests written first.

**Test-Driven Development Workflow (MANDATORY):**
1. **Write failing test first** - Before writing any production code
2. **Run test and verify it fails** - `cd api && go test -v ./...`
3. **Write minimal code** to make the test pass
4. **Run test and verify it passes**
5. **Refactor** if needed while keeping tests green
6. **Update test stats** in the documentation (test count and coverage %)
7. **Commit** with test and implementation together

**Test Coverage Goals:**
- Maintain **80%+ test coverage** for all Go code (currently at ~60%)
- Write tests for **every new function** - no exceptions
- Update existing tests when modifying functionality
- Run tests before every commit: `go test ./...`
- Never commit production code without corresponding tests
- **Test deployment scenarios**: Docker vs local, NAS path detection across environments

**Testing Standards:**
- Unit tests: Test individual functions in isolation
- Integration tests: Test API endpoints end-to-end
- **Deployment scenario tests**: Test Docker/local path detection (see handlers_nas_test.go)
- Use table-driven tests for multiple scenarios
- Mock external dependencies (network calls, file system when appropriate)
- Use `t.TempDir()` for file system tests
- Add `t.Skip()` for tests that require network/external resources
- Test both happy path and error cases
- Use meaningful test names: `TestFunctionName_Scenario_ExpectedBehavior`

**Test Files Location:**
- `internal/*_test.go` - New structure test files (handlers_test.go, jobs_test.go, handlers_nas_test.go, scraper_concurrency_test.go, handlers_nas_resolve_test.go, etc.)
- `api/*_test.go` - Legacy test files (still passing)
- Run: `go test ./internal/... ./api/...`
- Coverage: `go test -cover ./internal/... ./api/...`
- Count tests: `go test -v ./internal/... ./api/... 2>&1 | grep -c "^=== RUN"`
  - **Current Stats**: 376 test cases, ~63% average coverage
    - cmd/api: 1 test, 0.0% (main not testable)
    - internal/api: 51 tests (including analyzed tracks endpoint), 61.1%
  - internal/config: 18 tests, 96.4%
  - internal/downloader: 79 tests, 69.7%
  - internal/jobs: 26 tests (includes extended lifecycle, cancellation, concurrency tests), 58.0%
  - internal/scraper: 15 tests (including concurrency tests), 60.3%
  - internal/storage: 77 tests (includes extended utilities tests for filtering, sorting, caching), 60.5%
  - pkg/logger: 13 tests, 74.1%
  - **Phase 2 Chrome Fixes**: 9 tests for semaphore limits and job cancellation (all passing)
  - **NAS Path Resolution**: 10 new tests for auto-detection and error handling (all passing)
  - **Extended Test Suites**: 28 new tests for job lifecycle, storage utilities, and edge cases

**When Adding Features:**
1. Write tests first (TDD approach preferred)
2. Implement feature
3. Ensure tests pass
4. Check coverage: `go test -cover`
5. **Bump version numbers** in `VERSION` and `frontend/VERSION` (PATCH for bugs, MINOR for features, MAJOR for breaking changes)
6. **Update test stats in this file** - Update line 29 with: `go test -v ./... 2>&1 | grep -c "^=== RUN"` and `go test -cover`
7. Commit with meaningful message (include version in commit if bumped)

**Documentation Maintenance:**
- **ALWAYS update test coverage statistics** after adding/modifying tests
- Run `go test -cover ./internal/... ./api/...` to get coverage percentage
- Run `go test -v ./internal/... ./api/... 2>&1 | grep -c "^=== RUN"` to count total tests
- Update format: `# 239 tests passing (157 test cases), ~61.5% coverage`
- Keep API documentation in `api/README.md` synchronized with actual endpoints
- Update `docs/TODO.md` when completing tasks or adding new ones

**Remember:** Tests are not optional. They are part of the feature. A feature without tests is an incomplete feature.

**Documentation:**
- Keep `docs/TODO.md` updated with task status
- Update `docs/DEVELOPMENT.md` when changing dev workflow
- Update API documentation in `api/README.md` for new endpoints
- All API operations return 202 Accepted immediately with job ID
- Check job status with `GET /api/v1/jobs/:id` or via WebSocket
- CSV column 5 (index 5) contains the Listen URL by default
- NAS path: `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
- chromedp needs Chrome/Chromium installed
- yt-dlp must be available in PATH
- Frontend proxies `/api` requests to backend automatically

## Dependencies

### Backend (Go)
```go
github.com/gofiber/fiber/v2
github.com/gofiber/websocket/v2
github.com/chromedp/chromedp
github.com/chromedp/cdproto
github.com/google/uuid
github.com/rs/zerolog
golang.org/x/net/html
```

### Frontend (npm)
```json
react, react-dom, axios, date-fns
@types/react, @types/react-dom
vite, @vitejs/plugin-react
tailwindcss, autoprefixer, postcss
typescript, eslint
```

Install with: `go get <package>` or `npm install <package>`

## Development Guidelines

### Code Style
- Use Go standard formatting (`gofmt`)
- Handle errors explicitly (return error values)
- Use context with timeouts for chromedp operations
- Clean up resources with `defer` statements
- Add progress callbacks for long operations
- Use structured logging via pkg/logger package
- Use config package for all path references (no hardcoded paths)

### Common Patterns
```go
// Job pattern for async operations
job := jobs.NewJob("job_type", fiber.Map{"param": "value"})
go func() {
    result, err := SomeOperation(params)
    if err != nil {
        job.Fail(fmt.Sprintf("Operation failed: %v", err))
        return
    }
    job.Complete(fiber.Map{"result": result})
}()

// Handler pattern with dependency injection
type Handlers struct {
    cfg *config.Config
}

func (h *Handlers) HandleSomeAction(c *fiber.Ctx) error {
    csvPath := h.cfg.CSVDir  // Use config, not hardcoded paths
    logger.Info("Processing action", "path", csvPath)
    // ...
}
```

## Next Steps (from TODO.md)
1. ✅ ~~Add WebSocket support~~ - Completed
2. ✅ ~~Build React frontend~~ - Completed
3. ✅ ~~Refactor to Go standard layout~~ - Completed
4. **Music metadata extraction** using audio libraries (PRIORITY 1)
5. **Browse/search tracks UI** in frontend (PRIORITY 1)
6. **Docker deployment** updates for new structure
7. **Authentication** and user management

## Important Notes
- **Port Configuration**:
  - Local development (outside Docker): Port 3002 (avoids VS Code Helper conflict on 3001)
  - Docker development: Port 3000 (docker-compose.yml)
  - Docker production: Port 3001 (docker-compose.prod.yml, for Cloudflare tunnel)
  - Frontend Vite dev server: Port 5173
- All API operations return 202 Accepted immediately with job ID
- Check job status with `GET /api/v1/jobs/:id` or via WebSocket
- CSV column 5 (index 5) contains the Listen URL by default
- NAS path: Configurable via h.cfg.NASPath (default: empty for local, Docker auto-detects /nas)
- Set NAS_PATH environment variable for local NAS access (e.g., /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks)
- chromedp needs Chrome/Chromium installed
- yt-dlp must be available in PATH
- Frontend proxies `/api` requests to backend automatically
- Stats endpoint checks if NAS exists before scanning

## Version Management (⚠️ CRITICAL - DO THIS BEFORE EVERY COMMIT)
**ALWAYS increment versions when committing ANY changes - no exceptions:**
- Backend version: Update `VERSION` file in project root (format: `X.Y.Z`)
- Frontend version: Update `frontend/VERSION` file (format: `X.Y.Z`)
- **Keep both files synchronized** - always bump both together to the same version
- Version format: `MAJOR.MINOR.PATCH` (e.g., `1.0.1` → `1.0.2` for bug fixes)
- **Increment rules (follow strictly):**
  - PATCH (1.0.1 → 1.0.2): Bug fixes, performance improvements, minor refactors
  - MINOR (1.0.2 → 1.1.0): New features, new endpoints, significant enhancements (backwards compatible)
  - MAJOR (1.1.0 → 2.0.0): Breaking API changes, major architecture changes
- **Why this matters:**
  - Versions are displayed in the UI footer for deployment tracking
  - Watchtower uses image changes to trigger deployments
  - Version files are included in Docker images and read at runtime
  - Helps track which code is running in production
- **Before every commit:** Open both `VERSION` and `frontend/VERSION`, increment appropriately, save, then commit
