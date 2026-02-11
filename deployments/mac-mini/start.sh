#!/bin/bash
# Start DoYou Music Analytics services

set -e

echo "🚀 Starting DoYou Music Analytics..."

# Check if .env exists
if [ ! -f .env ]; then
    echo "❌ .env file not found. Run setup.sh first."
    exit 1
fi

# Start services
docker-compose -f docker-compose.prod.yml up -d

# Wait a moment for startup
sleep 3

# Show status
echo ""
echo "📊 Service Status:"
docker-compose -f docker-compose.prod.yml ps

echo ""
echo "✅ Services started!"
echo ""
echo "📋 View logs:"
echo "   All:     docker-compose -f docker-compose.prod.yml logs -f"
echo "   Worker:  docker-compose -f docker-compose.prod.yml logs -f worker"
echo "   API:     docker-compose -f docker-compose.prod.yml logs -f api"
