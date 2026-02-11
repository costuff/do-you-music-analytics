# DoYou Music Analytics

**CPU-intensive audio analysis service for DoYou World Radio track collection**

Companion service to [doyou-world-scraper](https://github.com/costuff/doyou-world-scraper) for batch processing audio analysis of 9,875+ tracks using librosa.

---

## Architecture

This service is part of a microservices architecture:

- **Scraper Service** (Raspberry Pi): Downloads tracks, saves to NAS, writes metadata to PostgreSQL
- **Analytics Service** (This repo, Desktop/Server): Reads unanalyzed tracks from DB, runs librosa analysis, writes audio features back to DB
- **Shared PostgreSQL**: Central state management
- **Shared NAS**: Both services access MP3 files

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed design.

---

## Features

- ✅ Batch audio analysis with configurable parallelism
- ✅ PostgreSQL for state management and query capabilities
- ✅ Spotify-like audio features (BPM, key, energy, danceability, valence, etc.)
- ✅ REST API for monitoring and control
- ✅ Progress tracking and error handling
- ✅ Docker deployment ready

---

## Quick Start

### Prerequisites

- Go 1.23+
- Python 3.10+ with librosa
- PostgreSQL 16+
- NAS mount at `/mnt/music/downloads` (or configure path)

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
go run cmd/migrate/main.go up
```

### 4. Start Worker

```bash
go run cmd/worker/main.go
```

### 5. Start API (Optional)

```bash
go run cmd/api/main.go
```

---

## Configuration

See [.env.example](.env.example) for all configuration options.

**Key settings:**

- `DATABASE_URL`: PostgreSQL connection string
- `NAS_MOUNT_PATH`: Path to NAS directory with MP3 files
- `BATCH_SIZE`: Number of tracks to analyze in parallel
- `MAX_WORKERS`: Number of parallel worker goroutines

---

## API Endpoints

### Worker Control
- `POST /api/v1/analysis/start` - Start batch analysis
- `POST /api/v1/analysis/stop` - Stop current job
- `GET /api/v1/analysis/status` - Get job status

### Tracks
- `GET /api/v1/tracks?status=analyzed` - List tracks
- `GET /api/v1/tracks/:id` - Get track with features

### Visualization Data
- `GET /api/v1/viz/bpm-distribution` - BPM histogram
- `GET /api/v1/viz/energy-vs-danceability` - Scatter plot

See [API.md](docs/API.md) for full documentation.

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
go test ./...
```

### Database Migrations

```bash
# Create new migration
go run cmd/migrate/main.go create migration_name

# Apply migrations
go run cmd/migrate/main.go up

# Rollback
go run cmd/migrate/main.go down
```

---

## Deployment

### Docker Compose (Recommended)

```bash
cd deployments/docker
docker-compose up -d
```

This starts:
- PostgreSQL database
- Analytics worker
- Analytics API
- (Optional) Grafana for monitoring

### Manual Deployment

See [DEPLOYMENT.md](docs/DEPLOYMENT.md) for production setup guide.

---

## Database Schema

### `tracks` table
- Track metadata (file_path, title, artist, etc.)
- Analysis status tracking
- Managed by both scraper and analytics services

### `audio_features` table
- Audio analysis results
- BPM, key, energy, danceability, valence, etc.
- Managed by analytics service only

### `analysis_jobs` table
- Batch job progress tracking
- Statistics (total, analyzed, failed)

See [migrations/](migrations/) for full schema.

---

## Audio Features

Extracted using librosa and custom algorithms:

- **BPM**: Beats per minute (tempo)
- **Key**: Musical key (C, C#, D, etc.)
- **Mode**: Major or minor
- **Energy**: Perceived intensity (0.0-1.0)
- **Danceability**: Rhythmic stability (0.0-1.0)
- **Valence**: Musical positivity (0.0-1.0)
- **Acousticness**: Acoustic vs electronic (0.0-1.0)
- **Instrumentalness**: Vocal presence (0.0-1.0)
- **Speechiness**: Spoken word presence (0.0-1.0)
- **Liveness**: Live audience presence (0.0-1.0)
- **Melodicness**: Melodic clarity (0.0-1.0)

---

## Monitoring

Metrics exposed at `/metrics` (Prometheus format):

- `analysis_tracks_total` - Total tracks analyzed
- `analysis_tracks_failed` - Failed analyses
- `analysis_duration_seconds` - Per-track analysis time
- `analysis_queue_size` - Pending tracks

Import [grafana-dashboard.json](deployments/grafana/dashboard.json) for pre-built dashboard.

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

## Related Projects

- [doyou-world-scraper](https://github.com/costuff/doyou-world-scraper) - Track scraper and downloader (Raspberry Pi)

---

## Version

Current version: **0.1.0** (Pre-release)

See [CHANGELOG.md](CHANGELOG.md) for version history.
