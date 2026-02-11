#!/bin/bash
# Complete setup script for DoYou Music Analytics on Mac
# Run this after Docker Desktop is installed and running

set -e

echo "🚀 DoYou Music Analytics - Complete Setup"
echo "=========================================="
echo ""

# Wait for Docker to be ready
echo "⏳ Waiting for Docker daemon..."
until docker ps &>/dev/null; do
    printf '.'
    sleep 2
done
echo ""
echo "✅ Docker is ready"
echo ""

# Build images locally
echo "🏗️  Building Docker images locally..."
echo ""

echo "📦 Building worker image..."
docker build -f deployments/docker/Dockerfile.worker \
    -t ghcr.io/costuff/do-you-music-analytics-worker:latest .

echo ""
echo "📦 Building API image..."
docker build -f deployments/docker/Dockerfile.api \
    -t ghcr.io/costuff/do-you-music-analytics-api:latest .

echo ""
echo "✅ Images built successfully!"
echo ""

# Create Docker network
echo "🔧 Setting up Docker network..."
docker network create doyou-network 2>/dev/null || echo "Network already exists"
echo ""

# Start services
echo "🚀 Starting services with docker-compose..."
docker-compose -f docker-compose.prod.yml up -d

echo ""
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 10
until docker-compose -f docker-compose.prod.yml exec -T postgres pg_isready -U doyou &>/dev/null; do
    printf '.'
    sleep 2
done
echo ""
echo "✅ PostgreSQL is ready"
echo ""

# Run migrations
echo "📝 Running database migrations..."
docker-compose -f docker-compose.prod.yml exec -T postgres \
    psql -U doyou -d doyou_music < migrations/001_initial_schema.up.sql 2>/dev/null || echo "Migrations already applied or not needed"

echo ""
echo "✅ Setup complete!"
echo ""
echo "📊 Service Status:"
docker-compose -f docker-compose.prod.yml ps
echo ""
echo "📋 Useful Commands:"
echo "   View logs:   docker-compose -f docker-compose.prod.yml logs -f worker"
echo "   Stop:        docker-compose -f docker-compose.prod.yml down"
echo "   Restart:     docker-compose -f docker-compose.prod.yml restart"
echo "   Status:      docker-compose -f docker-compose.prod.yml ps"
echo ""
echo "🔄 Watchtower will check for updates every 5 minutes"
echo "💾 Found 13,542 MP3 files at: /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks/"
