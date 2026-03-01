# 🚀 Deployment Guide

## Overview

This guide covers deploying the DoYou Music Analytics service to your Mac mini (or any production environment) with automatic updates via GitHub Actions and Watchtower.

## Quick Start (TL;DR)

**On Mac mini (Watchtower already configured ✅):**
# 1. Clone and setup
git clone https://github.com/costuff/do-you-music-analytics.git
cd do-you-music-analytics
cp deployments/mac-mini/.env.example .env
nano .env  # Edit NAS_MUSIC_PATH and passwords

# 2. Start services (skip postgres if already running)
docker-compose -f docker-compose.prod.yml up -d worker api

# 3. Import MP3 files (CRITICAL - do this first!)
VALIDATE_NAS_ON_STARTUP=false go run ./cmd/import-tracks /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks

# 4. Verify
docker-compose -f docker-compose.prod.yml logs -f worker
```

**Auto-updates:** Just `git push` to main - GitHub Actions builds images, Watchtower pulls and restarts automatically!

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       Mac Mini Server                        │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │  PostgreSQL  │  │   Worker     │  │     API      │     │
│  │  (Database)  │◄─┤ (Analysis)   │◄─┤ (Monitoring) │     │
│  └──────────────┘  └──────┬───────┘  └──────────────┘     │
│                            │                                 │
│                            ▼                                 │
│                    ┌──────────────┐                         │
│                    │  NAS/Music   │                         │
│                    │  (9,875 MP3) │                         │
│                    └──────────────┘                         │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │           Watchtower (Auto-Updates)                  │  │
│  │  Polls GitHub Container Registry every 5 minutes     │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │ Push new images
                            │
                   ┌────────┴─────────┐
                   │ GitHub Actions   │
                   │ (CI/CD Pipeline) │
                   └──────────────────┘
                            ▲
                            │ git push
                            │
                   ┌────────┴─────────┐
                   │  Your Laptop     │
                   │  (Development)   │
                   └──────────────────┘
```

## Deployment Flow

1. **Developer pushes code** → GitHub
2. **GitHub Actions** builds Docker images, pushes to `ghcr.io`
3. **Watchtower** (already running on Mac mini) detects new images
4. **Watchtower** pulls and restarts containers automatically
5. **Zero downtime** - rolling restarts with health checks

---

## 🧪 Pre-Push Local Testing

**CRITICAL:** Always test Docker builds locally before pushing to `main` to prevent production issues.

### Why Test Locally?

- Catches race conditions and bugs before production
- Validates Docker image builds successfully
- Ensures all dependencies and paths work in containerized environment
- Prevents broken deployments that trigger unnecessary Watchtower updates

### Local Testing Workflow

**1. Build Both Docker Images:**

```bash
# Build API image
docker build -f deployments/docker/Dockerfile.api -t doyou-analytics-api:local .

# Build Worker image
docker build -f deployments/docker/Dockerfile.worker -t doyou-analytics-worker:local .

# Verify both images built successfully
docker images | grep doyou-analytics
```

**2. Run Services Locally with Docker Compose:**

```bash
# Start all services (postgres + api + worker)
docker-compose up -d

# Or start specific services
docker-compose up -d postgres  # Database first
docker-compose up -d api       # API server
docker-compose up -d worker    # Worker (processes tracks)

# Check logs
docker-compose logs -f worker
docker-compose logs -f api
```

**3. Verify Health Checks:**

```bash
# API health check
curl http://localhost:8080/health
# Expected: {"status":"healthy","database":"connected"}

# Check worker is processing
docker-compose logs worker | grep "Processing track"

# Check database connection
docker-compose exec postgres psql -U doyou -d doyou_music -c "SELECT COUNT(*) FROM tracks;"
```

**4. Test with Sample Tracks (Optional):**

```bash
# Import a few test tracks (use small subset for quick testing)
go run ./cmd/import-tracks /path/to/test/tracks

# Watch worker process them
docker-compose logs -f worker

# Verify features were extracted
docker-compose exec postgres psql -U doyou -d doyou_music \
  -c "SELECT title, bpm, key, energy FROM tracks t JOIN audio_features af ON t.id = af.track_id LIMIT 5;"
```

**5. Pre-Push Checklist:**

- [ ] All unit tests pass: `go test ./...`
- [ ] Docker API image builds successfully
- [ ] Docker Worker image builds successfully
- [ ] `docker-compose up` runs without errors
- [ ] API health check returns 200
- [ ] Worker connects to database
- [ ] Worker can process tracks without race conditions
- [ ] No panic or error logs in worker/api
- [ ] Python/librosa dependencies working in worker image

**6. Clean Up After Testing:**

```bash
# Stop all services
docker-compose down

# Optional: Remove test images
docker rmi doyou-analytics-api:local doyou-analytics-worker:local

# Optional: Reset test database
docker-compose down -v  # Removes volumes (deletes data)
```

### Only Push After Validation ✅

```bash
# After all checks pass
git add .
git commit -m "feat: your feature description"
git push origin main  # Triggers CI/CD → Watchtower deployment
```

> **Remember:** Every push to `main` triggers GitHub Actions build and Watchtower auto-deploys to Mac mini. Test locally first!

For a concise operational checklist, see [PREPUSH_CHECKLIST.md](PREPUSH_CHECKLIST.md).

---

## Initial Setup on Mac Mini

> **Note:** Watchtower is already configured on your Mac mini! You only need to set up the analytics services.

### 1. Prerequisites

```bash
# Install Docker Desktop for Mac or Docker CLI
brew install --cask docker

# Verify installation
docker --version
docker-compose --version
```

### 2. Clone Repository

```bash
cd ~/projects  # or your preferred directory
git clone https://github.com/costuff/do-you-music-analytics.git
cd do-you-music-analytics
```

### 3. Configure Environment

Create `.env` file in project root:

```bash
# Copy template
cp deployments/mac-mini/.env.example .env

# Edit with your values
nano .env
```

**Required environment variables:**

```env
# Database
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_USER=doyou
POSTGRES_DB=doyou_music
POSTGRES_PORT=5432

# NAS Path (CRITICAL - adjust to your setup)
NAS_MUSIC_PATH=/Volumes/Expansion/ExtHDD-8TB/Music/doyou-world-radio-tracks

# Worker Configuration
BATCH_SIZE=10           # Process 10 tracks at a time
MAX_WORKERS=4           # 4 parallel analyzers
ANALYSIS_TIMEOUT=90s    # 90 seconds per track

# API
API_PORT=8080

# Logging
LOG_LEVEL=info

# Watchtower (optional notifications)
# WATCHTOWER_NOTIFICATIONS=slack
# WATCHTOWER_NOTIFICATION_URL=https://hooks.slack.com/...
```

### 4. Authenticate Docker with GitHub

To pull private images from GitHub Container Registry:

```bash
# Create GitHub Personal Access Token
# Go to: https://github.com/settings/tokens
# Scopes needed: read:packages

# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin

# Verify login
docker pull ghcr.io/costuff/do-you-music-analytics-worker:latest
```

### 5. Start Services

```bash
# Start analytics services (Watchtower already running)
docker-compose -f docker-compose.prod.yml up -d postgres worker api

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f worker
docker-compose -f docker-compose.prod.yml logs -f api
```

> **Note:** Watchtower is already running on your Mac mini - no need to start it separately.

### 6. Run Initial Migration

```bash
# Run database migrations
docker-compose -f docker-compose.prod.yml exec postgres psql -U doyou -d doyou_music -f /docker-entrypoint-initdb.d/001_initial_schema.up.sql
```

### 7. Import MP3 Files to Database

**CRITICAL FIRST STEP:** Before the worker can analyze tracks, you need to import them to the database.

```bash
# Local development (Mac mini directly)
VALIDATE_NAS_ON_STARTUP=false go run ./cmd/import-tracks /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks

# Or using Docker (recommended for production)
docker-compose -f docker-compose.prod.yml run --rm worker \
  go run ./cmd/import-tracks /mnt/music/downloads

# Expected output:
# Track Importer v0.4.0
# Music Dir: /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks
# Connected to DB
# Scanning...
# Found 9875 MP3 files
# Importing...
# Done: 9875 imported, 0 skipped, 0 failed
```

**What it does:**
- Recursively scans the music directory for `.mp3` files
- Imports file paths to `tracks` table with `status='pending'`
- Skips duplicates automatically
- Uses 10 concurrent goroutines for fast import

**Time estimate:** ~30-60 seconds for 10,000 files

### 8. Verify Deployment

```bash
# Check PostgreSQL
docker-compose -f docker-compose.prod.yml exec postgres psql -U doyou -d doyou_music -c "SELECT version();"

# Check worker logs
docker-compose -f docker-compose.prod.yml logs worker | tail -20

# Check API health (when Phase 5 is complete)
curl http://localhost:8080/health
```

---

## Auto-Update Workflow

### How It Works

1. **Push code** to GitHub `main` branch
2. **GitHub Actions** automatically:
   - Reads `VERSION` file
   - Builds multi-arch Docker images (amd64 + arm64)
   - Tags with `latest` and version number
   - Pushes to `ghcr.io/costuff/do-you-music-analytics-{worker,api}`
3. **Watchtower** on Mac mini:
   - Polls GitHub Container Registry every 5 minutes
   - Detects new image versions
   - Pulls new images
   - Performs rolling restart (worker finishes current job first)
   - Cleans up old images

### Monitoring Updates

```bash
# Watch Watchtower logs (running separately on Mac mini)
docker logs -f watchtower

# Or if Watchtower is in the compose stack:
docker-compose -f docker-compose.prod.yml logs -f watchtower

# Check current image versions
docker-compose -f docker-compose.prod.yml ps

# View Watchtower activity
docker-compose -f docker-compose.prod.yml exec watchtower watchtower --run-once --debug
```

### Manual Update (if needed)

```bash
# Pull latest images
docker-compose -f docker-compose.prod.yml pull

# Restart services
docker-compose -f docker-compose.prod.yml up -d

# Clean up old images
docker image prune -a -f
```

---

## Version Management

**CRITICAL:** Always bump version before pushing:

```bash
# Edit VERSION file
echo "0.4.0" > VERSION

# Commit
git add VERSION
git commit -m "chore: bump version to 0.4.0"
git push origin main

# GitHub Actions will build and push 0.4.0
# Watchtower will auto-deploy to Mac mini
```

**Version bumping rules:**
- **PATCH** (0.3.0 → 0.3.1): Bug fixes, minor tweaks
- **MINOR** (0.3.1 → 0.4.0): New features, new endpoints
- **MAJOR** (0.4.0 → 1.0.0): Breaking changes

---

## Troubleshooting

### Worker Not Starting

```bash
# Check logs
docker-compose -f docker-compose.prod.yml logs worker

# Common issues:
# 1. NAS path not mounted
docker-compose -f docker-compose.prod.yml exec worker ls -la /mnt/music/downloads

# 2. Database not ready
docker-compose -f docker-compose.prod.yml exec postgres pg_isready -U doyou

# 3. Python dependencies missing
docker-compose -f docker-compose.prod.yml exec worker python3 -c "import librosa; print(librosa.__version__)"
```

### Watchtower Not Updating

> **Note:** Watchtower is running separately on your Mac mini, not in the docker-compose stack.

```bash
# Check Watchtower container status
docker ps | grep watchtower

# View Watchtower logs
docker logs watchtower --tail 50

# Force update check (if you have access to restart it)
docker restart watchtower

# View recent update activity
docker logs watchtower | grep -i "Found new" | tail -10
```

### Database Connection Issues

```bash
# Test connection from worker
docker-compose -f docker-compose.prod.yml exec worker env | grep DATABASE_URL

# Connect to database directly
docker-compose -f docker-compose.prod.yml exec postgres psql -U doyou -d doyou_music

# Check running queries
docker-compose -f docker-compose.prod.yml exec postgres psql -U doyou -d doyou_music -c "SELECT * FROM pg_stat_activity;"
```

### NAS Path Issues

```bash
# Verify NAS mounted on Mac mini
ls -la /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks

# Check if visible in container
docker-compose -f docker-compose.prod.yml exec worker ls -la /mnt/music/downloads

# Count MP3 files
docker-compose -f docker-compose.prod.yml exec worker find /mnt/music/downloads -name "*.mp3" | wc -l
```

---

## Maintenance

### Backup Database

```bash
# Dump database
docker-compose -f docker-compose.prod.yml exec -T postgres pg_dump -U doyou doyou_music > backup_$(date +%Y%m%d).sql

# Restore database
cat backup_20260211.sql | docker-compose -f docker-compose.prod.yml exec -T postgres psql -U doyou -d doyou_music
```

### View Analytics Progress

```bash
# Check track analysis stats
docker-compose -f docker-compose.prod.yml exec postgres psql -U doyou -d doyou_music -c "
SELECT 
  COUNT(*) FILTER (WHERE status = 'pending') as pending,
  COUNT(*) FILTER (WHERE status = 'completed') as completed,
  COUNT(*) FILTER (WHERE status = 'failed') as failed,
  COUNT(*) as total
FROM tracks;
"

# Expected after import:
#  pending | completed | failed | total
# ---------+-----------+--------+-------
#     9875 |         0 |      0 |  9875

# After worker runs for a while:
#  pending | completed | failed | total
# ---------+-----------+--------+-------
#     8234 |      1620 |     21 |  9875
```

### Stop All Services

```bash
# Stop gracefully (worker finishes current job)
docker-compose -f docker-compose.prod.yml stop

# Stop immediately
docker-compose -f docker-compose.prod.yml down

# Stop and remove volumes (WARNING: deletes database)
docker-compose -f docker-compose.prod.yml down -v
```

---

## Performance Tuning

### Adjust Worker Concurrency

Edit `.env`:

```env
MAX_WORKERS=8        # Increase parallel analyzers (2-8 recommended)
BATCH_SIZE=20        # Process more tracks per batch
ANALYSIS_TIMEOUT=120s  # Allow more time for complex tracks
```

Restart worker:

```bash
docker-compose -f docker-compose.prod.yml restart worker
```

### Monitor Resource Usage

```bash
# Watch container stats
docker stats doyou-analytics-worker doyou-analytics-db

# Mac mini system resources
top -l 1 | grep -E "^CPU|^PhysMem"
```

---

## Security

### Sensitive Data

- **Never commit `.env` file** to Git
- Store `POSTGRES_PASSWORD` securely
- Use GitHub Secrets for CI/CD credentials
- Rotate passwords periodically

### Network Security

- PostgreSQL only exposed on localhost
- API behind reverse proxy (Cloudflare Tunnel recommended)
- NAS mounted read-only in worker container

---

## Next Steps

1. **Phase 4**: Worker loop now auto-deploys on push
2. **Phase 5**: API server for monitoring progress
3. **Web UI**: React dashboard for track browsing (future)

---

## Support

**Logs location:**
```bash
# View all logs
docker-compose -f docker-compose.prod.yml logs

# Follow specific service
docker-compose -f docker-compose.prod.yml logs -f worker

# Save logs to file
docker-compose -f docker-compose.prod.yml logs > deployment.log
```

**Health checks:**
- Worker: `pgrep -x worker` inside container
- API: `http://localhost:8080/health`
- Database: `pg_isready -U doyou`
