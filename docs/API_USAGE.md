# API Usage Guide

## Querying Analysis Statistics

The DoYou Music Analytics API provides endpoints to check how many songs have been analyzed and get detailed processing metrics.

### Basic Statistics Endpoint

**Endpoint:** `GET /api/v1/stats`

Returns track counts by analysis status.

```bash
curl http://localhost:8080/api/v1/stats
```

**Response:**

Example shape only; counts and timings below are illustrative, not live project metrics.
```json
{
  "total": 9875,
  "pending": 8234,
  "analyzing": 4,
  "completed": 1620,
  "failed": 17
}
```

**Field Descriptions:**
- `total`: Total number of tracks imported from NAS
- `pending`: Tracks waiting to be analyzed
- `analyzing`: Tracks currently being processed (usually 0-4)
- `completed`: **Number of songs successfully analyzed** ✅
- `failed`: Tracks that failed analysis

### Detailed Statistics Endpoint

**Endpoint:** `GET /api/v1/stats/detailed`

Returns basic stats plus processing performance metrics.

```bash
curl http://localhost:8080/api/v1/stats/detailed
```

**Response:**
```json
{
  "total": 9875,
  "pending": 8234,
  "analyzing": 4,
  "completed": 1620,
  "failed": 17,
  "avg_processing_time_sec": "2.34",
  "tracks_per_hour": "256.78",
  "completion_percentage": "16.40"
}
```

**Additional Fields:**
- `avg_processing_time_sec`: Average time to analyze one track (seconds)
- `tracks_per_hour`: Processing throughput (tracks/hour)
- `completion_percentage`: Percentage of tracks analyzed (completed/total × 100)

## Using from Node.js (DoYou Scraper Project)

### Example 1: Simple Fetch

```javascript
async function getAnalyzedCount() {
  try {
    const response = await fetch('http://localhost:8080/api/v1/stats');
    const stats = await response.json();
    
    console.log(`Analyzed tracks: ${stats.completed} / ${stats.total}`);
    console.log(`Success rate: ${((stats.completed / stats.total) * 100).toFixed(2)}%`);
    
    return stats.completed;
  } catch (error) {
    console.error('Failed to fetch analytics stats:', error);
    return 0;
  }
}

// Use in your scraper
const analyzedCount = await getAnalyzedCount();
```

### Example 2: Display in Statistics Dashboard

```javascript
import fetch from 'node-fetch';

async function getProcessingStats() {
  const response = await fetch('http://localhost:8080/api/v1/stats/detailed');
  const stats = await response.json();
  
  return {
    totalTracks: stats.total,
    analyzed: stats.completed,
    pending: stats.pending,
    failed: stats.failed,
    avgProcessingTime: stats.avg_processing_time_sec,
    processingRate: stats.tracks_per_hour,
    completionPercentage: stats.completion_percentage
  };
}

// Display in your dashboard
const stats = await getProcessingStats();
console.log('╔════════════════════════════════════╗');
console.log('║   Music Analysis Progress          ║');
console.log('╠════════════════════════════════════╣');
console.log(`║ Total Tracks:      ${stats.totalTracks.toString().padStart(6)}       ║`);
console.log(`║ Analyzed:          ${stats.analyzed.toString().padStart(6)}       ║`);
console.log(`║ Pending:           ${stats.pending.toString().padStart(6)}       ║`);
console.log(`║ Failed:            ${stats.failed.toString().padStart(6)}       ║`);
console.log(`║ Progress:          ${stats.completionPercentage}%     ║`);
console.log(`║ Rate:              ${stats.processingRate} tracks/hr ║`);
console.log('╚════════════════════════════════════╝');
```

### Example 3: Periodically Check Progress

```javascript
async function monitorProgress() {
  setInterval(async () => {
    const response = await fetch('http://localhost:8080/api/v1/stats');
    const stats = await response.json();
    
    console.log(`[${new Date().toISOString()}] Analysis Progress: ${stats.completed}/${stats.total} (${((stats.completed / stats.total) * 100).toFixed(2)}%)`);
  }, 30000); // Check every 30 seconds
}

monitorProgress();
```

## Production URL

When running in production (Mac mini), replace `localhost:8080` with your production URL:

```javascript
const ANALYTICS_API_URL = process.env.ANALYTICS_API_URL || 'http://localhost:8080';

async function getStats() {
  const response = await fetch(`${ANALYTICS_API_URL}/api/v1/stats`);
  return response.json();
}
```

## Error Handling

Always handle potential errors when querying the API:

```javascript
async function getStatsWithRetry(maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch('http://localhost:8080/api/v1/stats', {
        timeout: 5000 // 5 second timeout
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      
      return await response.json();
    } catch (error) {
      console.error(`Attempt ${i + 1} failed:`, error.message);
      
      if (i === maxRetries - 1) {
        // Last attempt failed
        return {
          total: 0,
          pending: 0,
          analyzing: 0,
          completed: 0,
          failed: 0,
          error: true
        };
      }
      
      // Wait before retrying (exponential backoff)
      await new Promise(resolve => setTimeout(resolve, Math.pow(2, i) * 1000));
    }
  }
}
```

## CORS

The API allows browser origins listed in `CORS_ORIGINS` (comma-separated). CORS is not authentication; use network or tunnel access controls for private deployments. The analytics host-published API port binds to localhost by default.

```javascript
// In your React/Vue/etc. dashboard
useEffect(() => {
  async function loadStats() {
    const response = await fetch('http://your-mac-mini-ip:8080/api/v1/stats');
    const data = await response.json();
    setAnalyzedCount(data.completed);
  }
  
  loadStats();
  const interval = setInterval(loadStats, 60000); // Update every minute
  
  return () => clearInterval(interval);
}, []);
```

## Similarity

- `GET /api/v1/tracks/{id}/similar?n=10` returns up to 100 precomputed neighbors (default 10).
- `POST /api/v1/similarity/recompute` starts a recomputation; a request while one is running receives `409`.
- `GET /api/v1/similarity/stats` reports stored pair counts and recomputation status.

## Operations

- `GET /health` verifies PostgreSQL connectivity and returns `503` when it is unavailable.
- `GET /metrics` exposes Prometheus text-format request counters, cumulative duration, and in-flight request count.
- Responses include `X-Request-ID`; structured API request logs include the same correlation value.

## Testing the Endpoint

```bash
# Simple test
curl http://localhost:8080/api/v1/stats | jq

# Get just the completed count
curl -s http://localhost:8080/api/v1/stats | jq '.completed'

# Pretty print detailed stats
curl -s http://localhost:8080/api/v1/stats/detailed | jq
```
