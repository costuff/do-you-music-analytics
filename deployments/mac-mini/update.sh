#!/bin/bash
# Manually trigger Watchtower update check

set -e

echo "🔄 Triggering manual update check..."

# Restart watchtower to force immediate check
docker-compose -f docker-compose.prod.yml restart watchtower

echo "⏳ Waiting for Watchtower to check for updates..."
sleep 10

# Show Watchtower logs
echo ""
echo "📋 Watchtower logs (last 20 lines):"
docker-compose -f docker-compose.prod.yml logs --tail=20 watchtower

echo ""
echo "✅ Update check complete"
echo ""
echo "💡 View live updates:"
echo "   docker-compose -f docker-compose.prod.yml logs -f watchtower"
