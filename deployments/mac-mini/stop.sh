#!/bin/bash
# Stop DoYou Music Analytics services gracefully

set -e

echo "🛑 Stopping DoYou Music Analytics..."

# Stop services (graceful stop, gives containers 10s to finish)
docker-compose -f docker-compose.prod.yml stop

echo ""
echo "✅ Services stopped"
echo ""
echo "💡 To completely remove containers:"
echo "   docker-compose -f docker-compose.prod.yml down"
echo ""
echo "💡 To remove containers and data (WARNING: deletes database):"
echo "   docker-compose -f docker-compose.prod.yml down -v"
