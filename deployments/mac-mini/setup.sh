#!/bin/bash
# Mac Mini Initial Setup Script

set -e

echo "🚀 DoYou Music Analytics - Mac Mini Setup"
echo "=========================================="

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker Desktop for Mac first:"
    echo "   brew install --cask docker"
    exit 1
fi

echo "✅ Docker found: $(docker --version)"

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose not found. Please install it:"
    echo "   brew install docker-compose"
    exit 1
fi

echo "✅ docker-compose found: $(docker-compose --version)"

# Check if .env exists
if [ ! -f .env ]; then
    echo ""
    echo "⚠️  .env file not found!"
    echo ""
    read -p "Would you like to copy .env.example now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cp deployments/mac-mini/.env.example .env
        echo "✅ Created .env file from template"
        echo ""
        echo "⚠️  IMPORTANT: Edit .env file with your settings:"
        echo "   1. Set POSTGRES_PASSWORD to a secure password"
        echo "   2. Update NAS_MUSIC_PATH to your NAS mount point"
        echo "   3. Adjust MAX_WORKERS based on CPU cores"
        echo ""
        read -p "Press Enter after editing .env to continue..."
    else
        echo "❌ Setup cannot continue without .env file"
        exit 1
    fi
fi

# Source .env
source .env

# Verify NAS path exists
if [ ! -d "$NAS_MUSIC_PATH" ]; then
    echo "⚠️  WARNING: NAS path not found: $NAS_MUSIC_PATH"
    echo "   Please ensure NAS is mounted or update .env with correct path"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    MP3_COUNT=$(find "$NAS_MUSIC_PATH" -name "*.mp3" 2>/dev/null | wc -l | tr -d ' ')
    echo "✅ NAS path found with $MP3_COUNT MP3 files"
fi

# Check GitHub authentication
echo ""
echo "📦 Checking GitHub Container Registry authentication..."
if docker pull ghcr.io/costuff/do-you-music-analytics-worker:latest &> /dev/null; then
    echo "✅ GitHub Container Registry authenticated"
else
    echo "⚠️  Cannot pull images from GitHub Container Registry"
    echo ""
    echo "Please authenticate with:"
    echo "  1. Create GitHub PAT: https://github.com/settings/tokens"
    echo "     Scope needed: read:packages"
    echo "  2. Login:"
    echo "     echo \$GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Pull latest images
echo ""
echo "📥 Pulling latest Docker images..."
docker-compose -f docker-compose.prod.yml pull

# Create network if doesn't exist
echo ""
echo "🔧 Setting up Docker network..."
docker network create doyou-network 2>/dev/null || echo "Network already exists"

# Start services
echo ""
echo "🚀 Starting services..."
docker-compose -f docker-compose.prod.yml up -d

# Wait for PostgreSQL
echo ""
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 5
until docker-compose -f docker-compose.prod.yml exec -T postgres pg_isready -U $POSTGRES_USER &>/dev/null; do
    printf '.'
    sleep 1
done
echo ""
echo "✅ PostgreSQL is ready"

# Check if migrations needed
echo ""
echo "🗄️  Checking database schema..."
MIGRATION_COUNT=$(docker-compose -f docker-compose.prod.yml exec -T postgres psql -U $POSTGRES_USER -d $POSTGRES_DB -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ' || echo "0")

if [ "$MIGRATION_COUNT" -eq "0" ]; then
    echo "📝 Running initial migrations..."
    docker-compose -f docker-compose.prod.yml exec -T postgres psql -U $POSTGRES_USER -d $POSTGRES_DB < migrations/001_initial_schema.up.sql
    echo "✅ Migrations complete"
else
    echo "✅ Database schema already initialized ($MIGRATION_COUNT tables)"
fi

# Show status
echo ""
echo "📊 Service Status:"
docker-compose -f docker-compose.prod.yml ps

# Show logs
echo ""
echo "📋 Recent logs (last 10 lines):"
docker-compose -f docker-compose.prod.yml logs --tail=10

echo ""
echo "✅ Setup complete!"
echo ""
echo "📚 Useful commands:"
echo "   Start:   bash deployments/mac-mini/start.sh"
echo "   Stop:    bash deployments/mac-mini/stop.sh"
echo "   Logs:    docker-compose -f docker-compose.prod.yml logs -f worker"
echo "   Status:  docker-compose -f docker-compose.prod.yml ps"
echo ""
echo "🔄 Watchtower will automatically check for updates every 5 minutes"
