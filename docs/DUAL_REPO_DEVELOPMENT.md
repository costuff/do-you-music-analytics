# Dual-Repo Development (Analytics + Scraper)

Use this when developing `doyou-music-analytics` and `doyou-world-scraper` together locally.

## Topology

### Local (development)

- Analytics API: `http://localhost:8080`
- Analytics DB: `localhost:5432`
- Scraper API: `http://localhost:3002`
- Scraper frontend: `http://localhost:5173`
- Scraper frontend proxies analytics via: `/analytics-api` → `http://localhost:8080` (Vite proxy, automatic)

### Production (Mac Mini)

- Scraper API + Frontend: `https://do-you-world-scraper.costuff.work` (Cloudflare Tunnel)
- Analytics API proxied at: `https://do-you-world-scraper.costuff.work/analytics-api/`
- nginx in the scraper frontend container proxies `/analytics-api/` → `doyou-analytics-api:8080`
- Frontend container is on both Docker networks (`doyou-world-scraper_doyou-network` + `doyou-network`)
- Single Watchtower (scraper stack) monitors all containers on the machine

## Required Environment

In analytics `.env`:

```bash
DATABASE_URL=postgres://doyou:REDACTED_ROTATED_CREDENTIAL@localhost:5432/doyou_music?sslmode=disable
API_PORT=8080
CORS_ORIGINS=http://localhost:5173
NAS_MOUNT_PATH=/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks
```

**No `frontend/.env.local` needed.** The Vite dev server proxy in `scraper/vite.config.ts` automatically
forwards `/analytics-api/*` → `http://localhost:8080/*`. The production scraper Dockerfile bakes
`VITE_ANALYTICS_API_URL=/analytics-api` — no extra CI or env config needed.

## Startup Order

### 1) Start analytics database

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-music-analytics
docker compose up -d postgres
```

### 2) Start analytics services

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-music-analytics
PORT=8080 go run ./cmd/api
```

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-music-analytics
go run ./cmd/worker
```

### 3) Start scraper services

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-world-scraper
PORT=3002 go run ./cmd/api
```

```bash
cd /Users/skoutelis/Desktop/workspace/doyou-world-scraper/frontend
npm run dev
```

## Sanity Checks

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/stats/detailed
curl http://localhost:3002/api/v1/health
```

Open `http://localhost:5173` and verify analytics stats are visible.

## Daily Development Pattern

1. Make changes in one repo.
2. Run focused tests in that repo.
3. Re-run sanity checks above.
4. Before push, run each repo's pre-push checklist and Docker verification.

## Fast Analytics Mode for Daily Work

When you start from VS Code scraper tasks using `Start Full Stack (Fast Analytics)`, the worker runs with:

- `MAX_WORKERS=2`
- `ANALYSIS_TIMEOUT=90s`
- `SCAN_INTERVAL=6h`
- `ENABLE_CLEANUP=false`

This keeps CPU and disk load lower while you iterate on API/UI. Before release validation, switch back to production-like worker settings.

## Current Practical Workflow

Primary control point is scraper VS Code tasks (`.vscode/tasks.json`):

- `Start Full Stack (Fast Analytics)`
- `Restart Full Stack (Fast Analytics)`

Use `Restart ...` when logs/ports look inconsistent.

## Verify Analysis Activity

Worker can be healthy but idle. Confirm with:

```bash
curl http://localhost:8080/api/v1/stats
```

- `pending > 0` and/or `analyzing > 0` => active processing
- `pending = 0` and `analyzing = 0` => idle queue (expected)

To create work for the worker, import tracks into analytics DB from a path that actually contains MP3 files.
