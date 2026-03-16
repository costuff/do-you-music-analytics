# DoYou Music Analytics

**Automated audio analysis service for DoYou World Radio track collection**

Companion service to [doyou-world-scraper](https://github.com/costuff/doyou-world-scraper) for continuous audio analysis of 9,875+ tracks using librosa.

## Documentation

- [Deployment Guide](docs/DEPLOYMENT.md)
- [Dual-Repo Development](docs/DUAL_REPO_DEVELOPMENT.md)
- [Testing Guide](docs/TESTING.md)
- [Scraper Integration](docs/SCRAPER_INTEGRATION.md)
- [Database Usage](docs/DATABASE_USAGE.md)
- [Pre-Push Checklist](docs/PREPUSH_CHECKLIST.md)
- [Roadmap / TODO](docs/TODO.md)

---

## Overview

This service automatically:
- 🔍 **Scans NAS** for new MP3 files every 30 minutes
- 📊 **Analyzes audio** with Python librosa (13 features: BPM, key, energy, etc.)
- 💾 **Stores results** in PostgreSQL for querying
- 🔄 **Runs continuously** with 6 parallel workers optimized for M2 Mac
- 📡 **Exposes REST API** for statistics and track queries

## Architecture

This service is part of a microservices architecture:

- **Scraper Service** (Mac mini/Raspberry Pi): Downloads tracks, saves to NAS
- **Analytics Service** (This repo, Mac mini): Scans NAS, analyzes tracks, stores features
- **Shared PostgreSQL**: Central database for tracks and audio features
- **Shared NAS**: Both services access `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks/`

See [docs/ANALYTICS_ARCHITECTURE.md](docs/ANALYTICS_ARCHITECTURE.md) for detailed design.

---

## Features

- ✅ **Automatic NAS scanning** - Detects new tracks every 30 minutes
- ✅ **Continuous processing** - 6 parallel workers, never stops
- ✅ **Missing file cleanup** - Automatically detects and handles deleted files (enabled by default, mark mode)
- ✅ **13 audio features** - BPM, key, energy, danceability, valence, acousticness, instrumentalness, speechiness, liveness, melodicness, loudness, duration, mode
- ✅ **REST API** - Query statistics and analyzed tracks
- ✅ **PostgreSQL storage** - Atomic operations, no race conditions
- ✅ **Docker deployment** - GitHub Actions → GHCR → Watchtower auto-updates
- ✅ **Comprehensive tests** - 99 tests covering JSON, scanner, paths, Docker, cleanup

---

## Quick Start

### Prerequisites

- Go 1.25+
- Python 3.11+ with librosa 0.10+
- PostgreSQL 16+
- NAS mount (Mac: `/Volumes/ExtHDD-8TB/Music/...` or Docker: `/mnt/music/downloads`)
- ffmpeg for audio decoding

### 1. Install Dependencies

```bash
# Go dependencies
go mod download

# Python dependencies
pip install -r scripts/requirements.txt
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env with your configuration
```

### 3. Run Database Migrations

```bash
go run ./cmd/migrate up
```

### 4. Start Worker (Runs Continuously)

```bash
go run ./cmd/worker
# Worker will:
# 1. Scan NAS for new tracks immediately
# 2. Start analyzing pending tracks with 6 parallel workers
# 3. Re-scan every 30 minutes for new tracks
# 4. Run until stopped with Ctrl+C
```

### 5. Start API (Optional, for monitoring)

```bash
go run ./cmd/api
# API runs on http://localhost:8080
```

---

## Configuration

See [.env.example](.env.example) for all configuration options.

**Key settings:**

- `DATABASE_URL`: PostgreSQL connection string
- `NAS_MOUNT_PATH`: Path to NAS directory with MP3 files
  - Mac: `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks`
  - Docker: `/mnt/music/downloads`
- `MAX_WORKERS`: Number of parallel workers (default: 6, optimal for M2 Mac)
- `SCAN_INTERVAL`: Automatic scan frequency (default: 30m)
- `ENABLE_CLEANUP`: Enable automatic cleanup of missing files (default: false)
- `CLEANUP_ACTION`: How to handle missing files - "mark" or "delete" (default: mark)
- `CLEANUP_INTERVAL`: Cleanup frequency (default: 10m)
- `PYTHON_PATH`: Python 3.11+ executable path
- `ANALYSIS_SCRIPT`: Path to analyze_audio.py

---

## API Endpoints

### Statistics
- `GET /health` - Health check
- `GET /api/v1/stats` - Track counts by status (total, pending, analyzing, completed, failed)
- `GET /api/v1/stats/detailed` - Stats + processing metrics (avg time, tracks/hour, completion %)

### Tracks
- `GET /api/v1/tracks?limit=50&offset=0&status=completed` - List tracks (paginated)
- `GET /api/v1/tracks/:id` - Get single track with audio features
- `GET /api/v1/tracks/search?q=artist+name` - Search by title/artist/album
- `GET /api/v1/tracks/failed?limit=50` - Get failed tracks with error messages

See [docs/API_USAGE.md](docs/API_USAGE.md) for detailed usage examples and Node.js integration code.

---

## Automatic Cleanup

The worker can automatically detect and handle files that no longer exist on the filesystem (e.g., deleted corrupted files).

**Configuration:**
```bash
ENABLE_CLEANUP=true           # Enable cleanup (default: true)
CLEANUP_ACTION=mark           # "mark" or "delete" (default: mark)
CLEANUP_INTERVAL=10m          # How often to check (default: 10m)
```

**Cleanup Actions:**
- **`mark`** (recommended, default): Sets `analysis_status='missing'` and adds error message. Preserves history for reference.
- **`delete`**: Permanently removes track records from database. Use with caution.

**Default Behavior:**
- Enabled by default with "mark" action (safe, preserves history)
- First cleanup runs 5 minutes after worker starts
- Subsequent cleanups run on the configured interval
- Logs all cleanup operations (marked/deleted counts)

This ensures statistics remain accurate when files are manually deleted from the NAS.

---

## Development

### Project Structure

```
doyou-music-analytics/
├── cmd/
│   ├── worker/           # Background worker
│   ├── api/              # REST API server
│   └── migrate/          # Database migrations
├── internal/
│   ├── analyzer/         # Audio analysis logic
│   ├── db/               # Database operations
│   ├── storage/          # File system operations
│   ├── config/           # Configuration
│   └── api/              # HTTP handlers
├── pkg/
│   └── logger/           # Structured logging
├── scripts/
│   ├── analyze_audio.py  # Python librosa wrapper
│   └── requirements.txt
├── migrations/           # SQL migrations
├── deployments/
│   └── docker/
└── docs/
```

### Running Tests

```bash
# Run all tests (99 tests)
go test ./...

# Run with coverage
go test ./... -cover

# Run specific package
go test ./internal/db -v

# Count tests
go test ./... -v 2>&1 | grep "^=== RUN" | wc -l
```

### Database Migrations

```bash
# Apply migrations
go run ./cmd/migrate up

# Check status
go run ./cmd/migrate status

# Rollback
go run ./cmd/migrate down
```

---

## Deployment

### Production (Mac mini with Watchtower)

```bash
# Use production compose file with auto-updates
docker-compose -f docker-compose.prod.yml up -d

# Watchtower polls GitHub Container Registry every 5 minutes
# Any push to main triggers: GitHub Actions → Build → Push → Auto-deploy
```

### Local Development

```bash
# Test Docker builds locally before pushing
docker build -f deployments/docker/Dockerfile.api -t doyou-analytics-api:local .
docker build -f deployments/docker/Dockerfile.worker -t doyou-analytics-worker:local .
docker-compose up
```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for complete production setup guide with pre-push testing checklist.

---

## Database Schema

### `tracks` table
- Track metadata (file_path, file_name, artist, title, album, etc.)
- Analysis status: `pending`, `analyzing`, `completed`, `failed`
- Timestamps: `downloaded_at`, `analyzed_at`
- Error tracking: `analysis_error` field

### `audio_features` table
- 13 audio features with `track_id` foreign key
- All feature fields are nullable (*float64, *string) for NULL support
- JSON tags for snake_case API responses
- Analysis version tracking

See [docs/DATABASE_USAGE.md](docs/DATABASE_USAGE.md) for schema details, SQL examples, and integration patterns.

---

## Audio Features (13 Total)

Extracted using librosa v0.10+ and custom algorithms:

**Tempo/Rhythm:**
- **BPM**: Beats per minute (float, typically 60-200)
- **Duration**: Track length in seconds

**Tonality:**
- **Key**: Musical key (C, C#, D, D#, E, F, F#, G, G#, A, A#, B)
- **Mode**: Major or Minor

**Energy/Mood:**
- **Energy**: Perceived intensity (0.0-1.0)
- **Danceability**: Rhythmic stability (0.0-1.0)
- **Valence**: Musical positivity (0.0-1.0)
- **Acousticness**: Acoustic vs electronic (0.0-1.0)

**Vocals:**
- **Instrumentalness**: Higher = less vocals (0.0-1.0)
- **Speechiness**: Spoken word presence (0.0-1.0)
- **Liveness**: Live audience presence (0.0-1.0)

**Melodic:**
- **Melodicness**: Melodic clarity (0.0-1.0)

**Dynamics:**
- **Loudness**: Track loudness in dB (typically -20 to 0)

---

## Monitoring

### API Statistics
```bash
# Get current stats
curl http://localhost:8080/api/v1/stats

# Get detailed stats with processing metrics
curl http://localhost:8080/api/v1/stats/detailed
```

### Docker Logs
```bash
# Watch worker logs
docker-compose logs -f worker

# Check last 100 lines
docker-compose logs --tail=100 worker
```

### Database Queries
```sql
-- Check progress
SELECT analysis_status, COUNT(*) FROM tracks GROUP BY analysis_status;

-- Get recently analyzed tracks
SELECT file_name, analyzed_at FROM tracks 
WHERE analysis_status = 'completed' 
ORDER BY analyzed_at DESC LIMIT 10;
```

---

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing`)
5. Open Pull Request

---

## License

MIT License - see [LICENSE](LICENSE) for details

---

## Documentation

- [docs/API_USAGE.md](docs/API_USAGE.md) - API integration guide with Node.js examples
- [docs/DATABASE_USAGE.md](docs/DATABASE_USAGE.md) - Database schema and query examples
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) - Production deployment guide
- [docs/ANALYTICS_ARCHITECTURE.md](docs/ANALYTICS_ARCHITECTURE.md) - System architecture details
- [docs/TESTING.md](docs/TESTING.md) - Testing guide and TDD workflow

## Related Projects

- [doyou-world-scraper](https://github.com/costuff/doyou-world-scraper) - Track scraper and downloader

---

## Version

Current version: **0.6.0**

See [VERSION](VERSION) file and git tags for version history.

**Version Management:** Always increment VERSION file before committing (PATCH for bugs, MINOR for features, MAJOR for breaking changes).
