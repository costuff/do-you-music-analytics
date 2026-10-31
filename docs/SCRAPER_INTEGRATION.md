# Analytics Service Integration Guide

**For DoYou World Scraper Integration**

This document provides everything you need to integrate the scraper with the analytics service.

---

## 📊 Service Status

**Current Version:** 0.6.0  
**Status:** ✅ Fully operational on Mac mini  
**Tracks Imported:** 3,752  
**Tracks Analyzed:** ~135 completed, ~2,500 pending  
**Workers:** 6 parallel processors running continuously  

---

## 🔌 API Endpoints for Scraper Integration

### 1. Get Analysis Statistics

**Endpoint:** `GET /api/v1/stats`

Returns track counts by status - perfect for showing in your scraper dashboard.

```bash
curl http://localhost:8080/api/v1/stats
```

**Response:**
```json
{
  "total": 3752,
  "pending": 2500,
  "analyzing": 6,
  "completed": 135,
  "failed": 1100
}
```

**Fields:**
- `completed` - Number of successfully analyzed songs ✅
- `total` - Total tracks imported
- `pending` - Waiting for analysis
- `analyzing` - Currently being processed
- `failed` - Failed analysis (usually corrupted MP3s)

### 2. Get Detailed Statistics

**Endpoint:** `GET /api/v1/stats/detailed`

Includes processing performance metrics.

```json
{
  "total": 3752,
  "completed": 135,
  "pending": 2500,
  "analyzing": 6,
  "failed": 1100,
  "avg_processing_time_sec": "2.34",
  "tracks_per_hour": "256.78",
  "completion_percentage": "3.60"
}
```

### 3. Search Analyzed Tracks

**Endpoint:** `GET /api/v1/tracks/search?q=artist+name`

Search tracks that have been analyzed.

```json
{
  "tracks": [
    {
      "id": 123,
      "file_name": "track.mp3",
      "file_path": "/mnt/music/downloads/track.mp3",
      "analysis_status": "completed",
      "features": {
        "bpm": 128.5,
        "key": "C#",
        "energy": 0.847,
        "danceability": 0.76
      }
    }
  ],
  "count": 1
}
```

### 4. Get Single Track with Features

**Endpoint:** `GET /api/v1/tracks/:id`

Retrieve full analysis for a specific track.

---

## 💻 Node.js Integration Code

### Simple Statistics Fetch

```javascript
async function getAnalysisStats() {
  try {
    const response = await fetch('http://localhost:8080/api/v1/stats');
    const stats = await response.json();
    
    console.log(`Analyzed: ${stats.completed} / ${stats.total}`);
    return stats;
  } catch (error) {
    console.error('Analytics API error:', error);
    return { total: 0, completed: 0, pending: 0, failed: 0 };
  }
}
```

### Display in Dashboard

```javascript
import chalk from 'chalk';

async function displayAnalyticsProgress() {
  const stats = await fetch('http://localhost:8080/api/v1/stats/detailed')
    .then(r => r.json());
  
  console.log(chalk.cyan('\n📊 Audio Analysis Progress'));
  console.log(chalk.cyan('═══════════════════════════'));
  console.log(`Total Tracks:     ${chalk.white(stats.total.toLocaleString())}`);
  console.log(`Analyzed:         ${chalk.green(stats.completed.toLocaleString())}`);
  console.log(`Pending:          ${chalk.yellow(stats.pending.toLocaleString())}`);
  console.log(`Failed:           ${chalk.red(stats.failed.toLocaleString())}`);
  console.log(`Progress:         ${chalk.blue(stats.completion_percentage + '%')}`);
  console.log(`Processing Rate:  ${chalk.magenta(stats.tracks_per_hour + ' tracks/hr')}`);
}
```

### Periodic Monitoring

```javascript
// Check analysis progress every 30 seconds
setInterval(async () => {
  const stats = await fetch('http://localhost:8080/api/v1/stats').then(r => r.json());
  console.log(`[${new Date().toISOString()}] Analysis: ${stats.completed}/${stats.total}`);
}, 30000);
```

### With Error Handling and Retry

```javascript
async function getStatsWithRetry(maxRetries = 3) {
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      const response = await fetch('http://localhost:8080/api/v1/stats', {
        timeout: 5000
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      return await response.json();
    } catch (error) {
      console.error(`Attempt ${attempt} failed:`, error.message);
      
      if (attempt === maxRetries) {
        return { total: 0, completed: 0, pending: 0, failed: 0, error: true };
      }
      
      // Exponential backoff
      await new Promise(resolve => setTimeout(resolve, Math.pow(2, attempt) * 1000));
    }
  }
}
```

---

## 🗄️ Direct Database Access (Alternative)

If you prefer direct database queries instead of API:

```javascript
import pg from 'pg';

const pool = new pg.Pool({
  host: 'localhost',
  port: 5432,
  database: 'doyou_music',
  user: 'doyou',
  password: process.env.POSTGRES_PASSWORD
});

// Get statistics
async function getDbStats() {
  const result = await pool.query(`
    SELECT 
      COUNT(*) as total,
      COUNT(*) FILTER (WHERE analysis_status = 'completed') as completed,
      COUNT(*) FILTER (WHERE analysis_status = 'pending') as pending,
      COUNT(*) FILTER (WHERE analysis_status = 'failed') as failed
    FROM tracks
  `);
  
  return result.rows[0];
}

// Get recently analyzed tracks
async function getRecentlyAnalyzed(limit = 10) {
  const result = await pool.query(`
    SELECT t.file_name, af.bpm, af.key, af.energy, t.analyzed_at
    FROM tracks t
    JOIN audio_features af ON t.id = af.track_id
    WHERE t.analysis_status = 'completed'
    ORDER BY t.analyzed_at DESC
    LIMIT $1
  `, [limit]);
  
  return result.rows;
}
```

---

## 📡 Production Setup

**Mac Mini URL:** Replace `localhost` with your Mac mini's IP or hostname:

```javascript
const ANALYTICS_API_URL = process.env.ANALYTICS_API_URL || 'http://localhost:8080';

async function getStats() {
  const response = await fetch(`${ANALYTICS_API_URL}/api/v1/stats`);
  return response.json();
}
```

**Environment Variable:**
```bash
# In scraper .env
ANALYTICS_API_URL=http://192.168.1.100:8080
# or
ANALYTICS_API_URL=http://macmini.local:8080
```

---

## 🔍 What's Been Analyzed

### Query Analyzed Tracks

```bash
# Get all completed tracks
curl http://localhost:8080/api/v1/tracks?status=completed&limit=50

# Search by artist
curl http://localhost:8080/api/v1/tracks/search?q=artist+name

# Get failed tracks (to see what went wrong)
curl http://localhost:8080/api/v1/tracks/failed?limit=20
```

---

## 📚 Complete Documentation

- **[docs/API_USAGE.md](https://github.com/costuff/do-you-music-analytics/blob/main/docs/API_USAGE.md)** - Full API examples with Node.js code
- **[docs/DATABASE_USAGE.md](https://github.com/costuff/do-you-music-analytics/blob/main/docs/DATABASE_USAGE.md)** - Database schema and SQL queries
- **[README.md](https://github.com/costuff/do-you-music-analytics/blob/main/README.md)** - Project overview and architecture

---

## 🚀 Current System State

### Architecture
```
┌─────────────────────────────────────────────────────────┐
│                    Mac Mini (M2)                        │
│                                                         │
│  ┌──────────────┐  ┌────────────────┐  ┌───────────┐  │
│  │ PostgreSQL   │◄─┤ Analytics      │◄─┤ REST API  │  │
│  │ (Database)   │  │ Worker         │  │ :8080     │  │
│  │              │  │ 6 workers      │  └───────────┘  │
│  └──────────────┘  │ Continuous     │                  │
│                    └────────┬───────┘                  │
│                             │                          │
│                             ▼                          │
│                    ┌────────────────┐                  │
│                    │ NAS Storage    │                  │
│                    │ 9,875 MP3s     │                  │
│                    │ 65GB           │                  │
│                    └────────────────┘                  │
└─────────────────────────────────────────────────────────┘
              ▲
              │ API Calls
              │
   ┌──────────┴──────────┐
   │  Scraper Service    │
   │  (Node.js)          │
   │  - Display stats    │
   │  - Monitor progress │
   └─────────────────────┘
```

### Auto-Update Pipeline
```
git push → GitHub Actions → Build Docker Images → GHCR → Watchtower → Mac Mini (auto-deploy)
```

Every push to main automatically deploys within 5-10 minutes!

---

## ✅ Ready to Integrate!

Everything is documented and ready. Start with the simple stats endpoint:

```javascript
// Add this to your scraper
const analyticsStats = await fetch('http://localhost:8080/api/v1/stats').then(r => r.json());
console.log(`Songs analyzed: ${analyticsStats.completed}`);
```

See [docs/API_USAGE.md](API_USAGE.md) for complete examples! 🎵
