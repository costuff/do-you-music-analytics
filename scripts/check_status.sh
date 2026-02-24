#!/bin/bash
# Quick status checker for DoYou Music Analytics
# Run this on your Mac mini to see current state

echo "==================================================="
echo "📊 DoYou Music Analytics - Status Check"
echo "==================================================="
echo ""

# Check if containers are running
echo "🐳 Docker Container Status:"
docker ps --filter "name=doyou-analytics" --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
echo ""

# Check worker logs for cleanup status
echo "🧹 Cleanup Status (last 20 lines of worker logs):"
docker logs doyou-analytics-worker --tail 20 2>/dev/null | grep -E "(Cleanup|🧹|Missing)" || echo "No cleanup logs found yet"
echo ""

# Check database statistics
echo "📈 Analysis Progress:"
docker exec doyou-analytics-db psql -U doyou -d doyou_music -c "
SELECT 
    analysis_status, 
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
FROM tracks 
GROUP BY analysis_status 
ORDER BY 
    CASE analysis_status
        WHEN 'completed' THEN 1
        WHEN 'pending' THEN 2
        WHEN 'analyzing' THEN 3
        WHEN 'failed' THEN 4
        WHEN 'missing' THEN 5
        ELSE 6
    END;
" 2>/dev/null || echo "Can't connect to database"
echo ""

# Check recent analysis activity
echo "⚡ Recent Activity (last 10 tracks):"
docker exec doyou-analytics-db psql -U doyou -d doyou_music -c "
SELECT 
    file_name,
    analysis_status,
    analyzed_at,
    CASE 
        WHEN analysis_error IS NOT NULL THEN LEFT(analysis_error, 50) || '...'
        ELSE 'OK'
    END as status
FROM tracks 
WHERE analyzed_at IS NOT NULL 
ORDER BY analyzed_at DESC 
LIMIT 10;
" 2>/dev/null || echo "Can't fetch recent activity"
echo ""

# Show current version
echo "📦 Running Version:"
docker inspect doyou-analytics-worker 2>/dev/null | grep -A2 "org.opencontainers.image.version" || echo "Version info not available"
echo ""

echo "==================================================="
echo "To see live worker logs: docker logs -f doyou-analytics-worker"
echo "To see API status: curl http://localhost:8080/api/v1/stats"
echo "==================================================="
