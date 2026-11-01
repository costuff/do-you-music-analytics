# DoYou Music Analytics - Architecture Plan

> **Historical document.** This is the original pre-implementation design plan,
> preserved from the `phase-6-comprehensive-tests` branch for context on why the
> service is built the way it is. It describes the system as *proposed*, not as
> built, so details such as schema, endpoints, and project layout have since
> changed. For current architecture see [ARCHITECTURE.md](ARCHITECTURE.md); for
> the live API see [API_USAGE.md](API_USAGE.md).

**Purpose**: Separate repository for CPU-intensive audio analysis of 9,875+ tracks  
**Rationale**: Raspberry Pi cannot handle batch librosa processing; run on powerful machine (desktop/server)  
**Repository**: `doyou-music-analytics` (to be created)

---

## Architecture Decision: Option C - Shared PostgreSQL (Recommended)

### Why PostgreSQL over Options A & B:

**vs Option A (Shared NFS):**
- ❌ Tight coupling via filesystem
- ❌ Race conditions on concurrent file access
- ❌ No query capabilities (can't filter by BPM, key, etc.)
- ✅ Simple setup

**vs Option B (REST API):**
- ❌ Network bandwidth for transferring 40GB of audio files
- ❌ Need to build full API contract
- ❌ Polling introduces latency
- ✅ Clean separation

**✅ Option C (PostgreSQL + Shared Storage):**
- ✅ Best of both worlds: DB for metadata + NFS for audio files
- ✅ Query capabilities (filter by features, date ranges)
- ✅ Atomic updates, transactions
- ✅ Easy to add Redis pub/sub later for real-time events
- ✅ Standard microservices pattern

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Shared PostgreSQL                        │
│  tracks table: id, file_path, title, artist, analyzed_at    │
│  features table: track_id, bpm, key, energy, danceability   │
└─────────────────────────────────────────────────────────────┘
         ↑                                          ↑
         │ Writes track metadata                   │ Writes audio features
         │ Reads analysis status                   │ Reads unanalyzed tracks
         │                                          │
┌────────┴──────────┐                    ┌─────────┴──────────┐
│  Scraper Service  │                    │ Analytics Service   │
│ (Raspberry Pi)    │                    │ (Powerful Machine)  │
│                   │                    │                     │
│ - Download tracks │                    │ - Batch analysis    │
│ - Save to NAS     │                    │ - librosa features  │
│ - Insert track    │                    │ - Update DB         │
│   metadata to DB  │                    │ - Progress tracking │
└───────────────────┘                    └─────────────────────┘
         │                                          │
         │                                          │
         └──────────────┬───────────────────────────┘
                        ↓
              ┌──────────────────┐
              │   Shared NAS     │
              │ /mnt/music/      │
              │                  │
              │ - MP3 files      │
              │ - 40GB storage   │
              │ - Both services  │
              │   read/write     │
              └──────────────────┘
```

---

## Database Schema

### `tracks` table (managed by both services)

```sql
CREATE TABLE tracks (
    id SERIAL PRIMARY KEY,
    file_path VARCHAR(512) NOT NULL UNIQUE,
    file_name VARCHAR(255) NOT NULL,
    title VARCHAR(255),
    artist VARCHAR(255),
    album VARCHAR(255),
    label VARCHAR(255),
    release_date DATE,
    file_size_bytes BIGINT,
    duration_seconds DECIMAL(10, 2),
    
    -- Timestamps
    downloaded_at TIMESTAMP DEFAULT NOW(),
    analyzed_at TIMESTAMP,
    
    -- Status tracking
    analysis_status VARCHAR(20) DEFAULT 'pending', -- pending, analyzing, completed, failed
    analysis_error TEXT,
    
    -- Indexes
    CONSTRAINT unique_file_path UNIQUE (file_path)
);

CREATE INDEX idx_tracks_analysis_status ON tracks(analysis_status);
CREATE INDEX idx_tracks_analyzed_at ON tracks(analyzed_at);
CREATE INDEX idx_tracks_artist ON tracks(artist);
CREATE INDEX idx_tracks_downloaded_at ON tracks(downloaded_at);
```

### `audio_features` table (managed by Analytics service)

```sql
CREATE TABLE audio_features (
    id SERIAL PRIMARY KEY,
    track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    
    -- Core features
    bpm DECIMAL(6, 2),
    key VARCHAR(10),           -- "C", "C#", "D", etc.
    mode VARCHAR(10),          -- "major", "minor"
    duration_seconds DECIMAL(10, 2),
    
    -- Spotify-like features (0.0-1.0 normalized)
    energy DECIMAL(5, 4),
    danceability DECIMAL(5, 4),
    valence DECIMAL(5, 4),
    acousticness DECIMAL(5, 4),
    instrumentalness DECIMAL(5, 4),
    speechiness DECIMAL(5, 4),
    liveness DECIMAL(5, 4),
    melodicness DECIMAL(5, 4),
    
    -- Technical features
    loudness_db DECIMAL(6, 2),
    tempo_confidence DECIMAL(5, 4),
    
    -- Analysis metadata
    analysis_version VARCHAR(20),
    analyzed_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT unique_track_features UNIQUE (track_id)
);

CREATE INDEX idx_features_bpm ON audio_features(bpm);
CREATE INDEX idx_features_energy ON audio_features(energy);
CREATE INDEX idx_features_danceability ON audio_features(danceability);
```

### `analysis_jobs` table (Analytics service)

```sql
CREATE TABLE analysis_jobs (
    id SERIAL PRIMARY KEY,
    status VARCHAR(20) DEFAULT 'pending', -- pending, running, completed, failed
    total_tracks INTEGER DEFAULT 0,
    analyzed_tracks INTEGER DEFAULT 0,
    failed_tracks INTEGER DEFAULT 0,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT
);
```

---

## Service Responsibilities

### Scraper Service (Raspberry Pi)
**What it does:**
- Scrape track listings from doyoutrackid.com
- Download MP3 files via yt-dlp
- Save files to NAS at `/mnt/music/downloads/`
- Insert track metadata into PostgreSQL `tracks` table
- Mark tracks as `analysis_status = 'pending'`
- Display analysis results in frontend (read from DB)

**What it DOESN'T do:**
- ❌ Batch audio analysis (too CPU-intensive)
- ❌ Run librosa (Python ML library)
- ❌ Process thousands of tracks

**Database Operations:**
```sql
-- After downloading a track
INSERT INTO tracks (file_path, file_name, title, artist, album, label, release_date, file_size_bytes)
VALUES ('/mnt/music/downloads/song.mp3', 'song.mp3', 'Song Title', 'Artist', 'Album', 'Label', '2025-01-15', 5242880)
ON CONFLICT (file_path) DO NOTHING;

-- Check analysis status for frontend display
SELECT t.*, af.bpm, af.key, af.energy, af.danceability
FROM tracks t
LEFT JOIN audio_features af ON t.id = af.track_id
WHERE t.analyzed_at IS NOT NULL
ORDER BY t.downloaded_at DESC;
```

---

### Analytics Service (Powerful Machine)
**What it does:**
- Mount NAS at `/mnt/music/` (read-only access to MP3s)
- Query PostgreSQL for unanalyzed tracks
- Run librosa analysis on each track
- Extract audio features (BPM, key, energy, etc.)
- Insert results into `audio_features` table
- Update track status to `analyzed_at = NOW()`
- Expose progress API for monitoring

**What it DOESN'T do:**
- ❌ Download or scrape tracks (Scraper's job)
- ❌ Manage track metadata (Scraper's job)

**Database Operations:**
```sql
-- Get pending tracks
SELECT id, file_path FROM tracks
WHERE analysis_status = 'pending' AND analyzed_at IS NULL
ORDER BY downloaded_at DESC
LIMIT 100;

-- Mark track as analyzing
UPDATE tracks SET analysis_status = 'analyzing' WHERE id = ?;

-- Insert analysis results
INSERT INTO audio_features (track_id, bpm, key, energy, danceability, valence, acousticness, ...)
VALUES (?, 128.5, 'Am', 0.85, 0.92, 0.65, 0.12, ...);

-- Mark track as analyzed
UPDATE tracks SET analyzed_at = NOW(), analysis_status = 'completed' WHERE id = ?;

-- Handle errors
UPDATE tracks SET analysis_status = 'failed', analysis_error = ? WHERE id = ?;
```

---

## Analytics Service Project Structure

```
doyou-music-analytics/
├── cmd/
│   ├── worker/           # Background worker for batch analysis
│   │   └── main.go
│   └── api/              # REST API for monitoring/control
│       └── main.go
├── internal/
│   ├── analyzer/         # Audio analysis logic
│   │   ├── analyzer.go   # Main analyzer with librosa integration
│   │   ├── features.go   # Feature extraction
│   │   └── batch.go      # Batch processing orchestration
│   ├── db/               # Database operations
│   │   ├── tracks.go     # Track CRUD
│   │   ├── features.go   # Features CRUD
│   │   └── jobs.go       # Job tracking
│   ├── storage/          # File system operations
│   │   ├── mounts.go     # NAS mount validation
│   │   └── audio.go      # Audio file utilities
│   ├── config/           # Configuration
│   │   └── config.go
│   └── api/              # HTTP API handlers
│       ├── handlers.go
│       └── routes.go
├── pkg/
│   └── logger/           # Structured logging
├── scripts/
│   ├── analyze_audio.py  # Python librosa wrapper (copy from scraper)
│   └── requirements.txt  # Python dependencies
├── migrations/           # Database migrations
│   ├── 001_create_tables.up.sql
│   └── 001_create_tables.down.sql
├── deployments/
│   └── docker/
│       ├── Dockerfile
│       └── docker-compose.yml
├── .env.example
├── go.mod
└── README.md
```

---

## API Endpoints (Analytics Service)

### Worker Control
- `POST /api/v1/analysis/start` - Start batch analysis job
- `POST /api/v1/analysis/stop` - Stop current job
- `GET /api/v1/analysis/status` - Get current job status
- `GET /api/v1/analysis/stats` - Overall stats (analyzed/pending/failed)

### Track Query
- `GET /api/v1/tracks?status=analyzed&limit=100` - List analyzed tracks
- `GET /api/v1/tracks/:id` - Get track with features
- `POST /api/v1/tracks/:id/reanalyze` - Re-run analysis for specific track

### Features Query
- `GET /api/v1/features/search?min_bpm=120&max_bpm=130` - Search by features
- `GET /api/v1/features/similar/:track_id` - Find similar tracks

### Visualization Data
- `GET /api/v1/viz/bpm-distribution` - Histogram data
- `GET /api/v1/viz/energy-vs-danceability` - Scatter plot data
- `GET /api/v1/viz/key-distribution` - Pie chart data

---

## Configuration

### Analytics Service `.env`

```bash
# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/doyou_music
DB_MAX_CONNS=20
DB_TIMEOUT=10s

# Storage
NAS_MOUNT_PATH=/mnt/music/downloads
VALIDATE_NAS_ON_STARTUP=true

# Analysis
BATCH_SIZE=10                    # Process N tracks in parallel
MAX_WORKERS=4                    # Number of parallel workers
PYTHON_PATH=/usr/bin/python3
ANALYSIS_SCRIPT=./scripts/analyze_audio.py
ANALYSIS_TIMEOUT=60s             # Per-track timeout

# API
API_PORT=4000
LOG_LEVEL=info
CORS_ORIGINS=http://localhost:5173,http://raspberrypi.local
```

### Scraper Service Updates (New DB Config)

```bash
# Existing config stays the same...

# NEW: Database connection
DATABASE_URL=postgresql://user:pass@db-host:5432/doyou_music
DB_ENABLED=true   # Enable DB integration (optional, defaults to file-based)
```

---

## Deployment Setup

### 1. PostgreSQL Setup (Docker)

```yaml
# docker-compose.yml (run on powerful machine or cloud)
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: doyou_music
      POSTGRES_USER: doyou
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    restart: unless-stopped

  analytics-worker:
    build: .
    environment:
      DATABASE_URL: postgresql://doyou:${DB_PASSWORD}@postgres:5432/doyou_music
      NAS_MOUNT_PATH: /mnt/music/downloads
    volumes:
      - /mnt/music:/mnt/music:ro  # Read-only NAS mount
    depends_on:
      - postgres
    restart: unless-stopped

  analytics-api:
    build: .
    command: /app/api
    environment:
      DATABASE_URL: postgresql://doyou:${DB_PASSWORD}@postgres:5432/doyou_music
    ports:
      - "4000:4000"
    depends_on:
      - postgres
    restart: unless-stopped

volumes:
  postgres_data:
```

### 2. NAS Mounting (Both Machines)

**On Raspberry Pi (Scraper):**
```bash
# fstab entry for persistent mount
//nas.local/Music /mnt/music cifs credentials=/root/.smbcreds,uid=1000,gid=1000 0 0
```

**On Powerful Machine (Analytics):**
```bash
# Same NAS, read-only
//nas.local/Music /mnt/music cifs credentials=/root/.smbcreds,uid=1000,gid=1000,ro 0 0
```

### 3. Database Migration

```bash
# Run on analytics service startup
go run cmd/migrate/main.go up

# Or use migrate CLI
migrate -database postgresql://... -path migrations up
```

---

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
- [ ] Create `doyou-music-analytics` repository
- [ ] Set up Go project structure
- [ ] Implement PostgreSQL schema + migrations
- [ ] Create database access layer (tracks, features)
- [ ] Copy Python analysis script from scraper
- [ ] Test NAS mounting and file access

### Phase 2: Core Analysis (Week 3-4)
- [ ] Implement batch analysis worker
- [ ] Parallel processing with configurable workers
- [ ] Error handling and retry logic
- [ ] Progress tracking and logging
- [ ] Single-track analysis for testing
- [ ] Write comprehensive tests

### Phase 3: API & Monitoring (Week 5-6)
- [ ] REST API for status/control
- [ ] Visualization data endpoints
- [ ] Prometheus metrics export
- [ ] Add Grafana dashboard
- [ ] WebSocket for real-time progress
- [ ] API documentation (Swagger)

### Phase 4: Integration (Week 7-8)
- [ ] Update scraper to write to PostgreSQL
- [ ] Backward compatibility (file-based mode)
- [ ] Update frontend to query analytics API
- [ ] Cross-service testing
- [ ] Performance optimization
- [ ] Documentation and deployment guide

### Phase 5: Advanced Features (Future)
- [ ] Redis pub/sub for events
- [ ] Music recommendation engine
- [ ] Playlist generation
- [ ] Genre classification (ML model)
- [ ] Waveform/spectrogram generation
- [ ] Duplicate detection by audio fingerprint

---

## Benefits of This Architecture

✅ **Scalability**: Analytics service can run on any powerful machine  
✅ **Separation of Concerns**: Scraper stays lightweight for Pi  
✅ **Query Performance**: PostgreSQL enables complex filters (BPM range, key, energy, etc.)  
✅ **Reliability**: DB transactions ensure data consistency  
✅ **Future-Ready**: Easy to add Redis, ML models, API integrations  
✅ **Standard Pattern**: Microservices with shared database (industry best practice)  
✅ **Fault Tolerance**: Services can restart independently  
✅ **Monitoring**: Centralized job tracking and progress  

---

## Migration Strategy

### Option 1: Fresh Start (Recommended)
- Leave existing JSON files as-is (10 analyzed tracks)
- Start analytics service with empty DB
- Slowly analyze all 9,875 tracks over time
- JSON files serve as backup/reference

### Option 2: Import Existing JSON
- Write import script to read existing JSON companion files
- Populate database with 10 analyzed tracks
- Continue with remaining 9,865 tracks

### Option 3: Hybrid
- Keep JSON files for local single-track analysis (Scraper)
- Use database only for batch results (Analytics)
- Frontend reads from both sources

**Recommendation**: Option 1 - clean slate, validate entire pipeline

---

## Next Steps

1. **Review this architecture** with team/stakeholders
2. **Choose PostgreSQL hosting**: Docker, cloud (AWS RDS, DigitalOcean Managed), or local
3. **Create GitHub repository**: `doyou-music-analytics`
4. **Set up project skeleton**: Go modules, folder structure
5. **Implement Phase 1**: Database schema, migrations, basic CRUD
6. **Test NAS access**: Verify both services can reach files
7. **Prototype worker**: Single-track analysis end-to-end

---

## Questions to Answer

- [ ] Where will PostgreSQL run? (same machine as analytics, cloud, or Raspberry Pi?)
- [ ] What's the NAS IP/hostname? (for docker-compose mount)
- [ ] Authentication: Do services need user management or just API keys?
- [ ] Monitoring: Self-hosted Grafana or cloud service (Datadog, etc.)?
- [ ] Timeline: When do you want this operational?
