# Pre-Push Checklist

Use this checklist before every push to `main`.

## 1) Workspace and Config

- [ ] Confirm clean git state: `git status`
- [ ] Bump `VERSION` according to semantic versioning
- [ ] Validate `.env` values (`DATABASE_URL`, `NAS_MOUNT_PATH`, `PYTHON_PATH`, `API_PORT`)

## 2) Test and Build Gates

- [ ] Run tests: `go test ./...`
- [ ] Build API image: `docker build -f deployments/docker/Dockerfile.api -t doyou-analytics-api:local .`
- [ ] Build Worker image: `docker build -f deployments/docker/Dockerfile.worker -t doyou-analytics-worker:local .`

## 3) Local Runtime Verification

- [ ] Start stack: `docker compose up -d`
- [ ] API health check: `curl http://localhost:8080/health`
- [ ] Verify worker logs show normal processing: `docker compose logs --tail=100 worker`
- [ ] Verify DB connectivity from API and worker logs

## 4) Integration Verification

- [ ] Check stats endpoint: `curl http://localhost:8080/api/v1/stats`
- [ ] Check detailed stats endpoint: `curl http://localhost:8080/api/v1/stats/detailed`
- [ ] Confirm scraper dashboard can read analytics stats (`VITE_ANALYTICS_API_URL`)

## 5) Push Readiness

- [ ] Update docs/TODO entries for any workflow or behavior changes
- [ ] Commit with clear message
- [ ] Push and verify GitHub Actions succeeded

## 6) Post-Push Production Checks

- [ ] Confirm GHCR images were published
- [ ] Confirm Watchtower pulled and restarted containers
- [ ] Confirm `/health` and `/api/v1/stats` are healthy in production
