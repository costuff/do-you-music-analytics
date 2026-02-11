# 🚀 Deployment Guide

## Overview

This guide covers deploying the DoYou Music Analytics service to your Mac mini (or any production environment) with automatic updates via GitHub Actions and Watchtower.

## Quick Start (TL;DR)

**On Mac mini:**
```bash
# 1. Clone and setup
git clone https://github.com/costuff/do-you-music-analytics.git
cd do-you-music-analytics
cp deployments/mac-mini/.env.example .env
nano .env  # Edit NAS_MUSIC_PATH and passwords

# 2. Start services
docker-compose -f docker-compose.prod.yml up -d

# 3. Import MP3 files (CRITICAL - do this first!)
VALIDATE_NAS_ON_STARTUP=false go run ./cmd/import-tracks /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks

# 4. Verify
docker-compose -f docker-compose.prod.yml logs -f worker
```

**Auto-updates:** Just `git push` to main - GitHub Actions + Watchtower handle the rest!

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
3. **Watchtower** on Mac mini detects new images
4. **Watchtower** pulls and restarts containers automatically
5. **Zero downtime** - rolling restarts with health checks

---

## Initial Setup on Mac Mini

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
NAS_MUSIC_PATH=/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks

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
# Start all services
docker-compose -f docker-compose.prod.yml up -d

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f worker
docker-compose -f docker-compose.prod.yml logs -f api
docker-compose -f docker-compose.prod.yml logs -f watchtower
```

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
# Watch Watchtower logs
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

```bash
# Check authentication
docker-compose -f docker-compose.prod.yml exec watchtower cat /config.json

# Force update check
docker-compose -f docker-compose.prod.yml restart watchtower

# View debug logs
docker-compose -f docker-compose.prod.yml logs watchtower | grep -i update
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
