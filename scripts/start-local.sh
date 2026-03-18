#!/usr/bin/env bash
# Start local development environment for doyou-music-analytics
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$SCRIPT_DIR"

echo "==> Starting Postgres..."
docker compose up -d postgres

echo "==> Waiting for Postgres to be ready..."
MAX=30
COUNT=0
until docker compose exec -T postgres pg_isready -U doyou >/dev/null 2>&1; do
  COUNT=$((COUNT + 1))
  if [ "$COUNT" -ge "$MAX" ]; then
    echo "ERROR: Postgres did not become ready in time."
    docker compose ps
    exit 1
  fi
  echo "  waiting ($COUNT/$MAX)..."
  sleep 2
done
echo "==> Postgres is ready!"

echo ""
echo "==> Dev environment ready. Start services with:"
echo ""
echo "  Analytics API (port 8080):"
echo "    DATABASE_URL=postgresql://doyou:dev_password_change_me@localhost:5432/doyou_music \\"
echo "    NAS_MOUNT_PATH=/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks \\"
echo "    PORT=8080 CORS_ORIGINS=http://localhost:5173 \\"
echo "    VALIDATE_NAS_ON_STARTUP=false SHALLOW_SCAN=true \\"
echo "    go run ./cmd/api"
echo ""
echo "  Analytics Worker:"
echo "    DATABASE_URL=postgresql://doyou:dev_password_change_me@localhost:5432/doyou_music \\"
echo "    NAS_MOUNT_PATH=/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks \\"
echo "    PYTHON_PATH=\$(pwd)/.venv/bin/python \\"
echo "    VALIDATE_NAS_ON_STARTUP=false SHALLOW_SCAN=true \\"
echo "    MAX_WORKERS=2 ANALYSIS_TIMEOUT=90s SCAN_INTERVAL=6h ENABLE_CLEANUP=false \\"
echo "    go run ./cmd/worker"
echo ""
echo "  Scraper API (port 3002):"
echo "    cd ../doyou-world-scraper && PORT=3002 go run ./cmd/api"
echo ""
echo "  Scraper Frontend (port 5173):"
echo "    cd ../doyou-world-scraper/frontend && npm run dev"
