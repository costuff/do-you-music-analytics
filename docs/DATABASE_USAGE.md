# Database Usage Guide

## Overview
This service analyzes MP3 audio files and extracts musical features (BPM, key, energy, danceability, etc.) into a PostgreSQL database.

## Database Connection
```bash
Host: localhost (or doyou-analytics-db from Docker network)
Port: 5432
Database: doyou_music
User: doyou
Password: REDACTED_ROTATED_CREDENTIAL
```

## Schema

### Table: `tracks`
Stores metadata about each MP3 file.

```sql
CREATE TABLE tracks (
    id SERIAL PRIMARY KEY,
    file_path VARCHAR(512) NOT NULL UNIQUE,
    file_name VARCHAR(255) NOT NULL,
    title VARCHAR(255),
    artist VARCHAR(255),
    album VARCHAR(255),
    label VARCHAR(255),
    release_date DATE,
    file_size_bytes BIGINT,
    duration_seconds NUMERIC(10,2),
    downloaded_at TIMESTAMP DEFAULT NOW(),
    analyzed_at TIMESTAMP,
    analysis_status VARCHAR(20) DEFAULT 'pending',
    analysis_error TEXT
);
```

**Analysis Status Values:**
- `pending` - Waiting to be analyzed
- `analyzing` - Currently being processed
- `completed` - Successfully analyzed
- `failed` - Analysis failed (corrupted file, etc.)

### Table: `audio_features`
Stores extracted audio features for each track.

```sql
CREATE TABLE audio_features (
    id SERIAL PRIMARY KEY,
    track_id INTEGER REFERENCES tracks(id) ON DELETE CASCADE,
    bpm NUMERIC(6,2),                -- Tempo (beats per minute)
    key VARCHAR(3),                  -- Musical key (C, D, E, F, G, A, B with # or b)
    mode VARCHAR(10),                -- Major/Minor
    duration_seconds NUMERIC(10,2),  -- Track length
    energy NUMERIC(6,4),             -- 0.0-1.0: High energy = loud, fast, noisy
    danceability NUMERIC(6,4),       -- 0.0-1.0: How suitable for dancing
    valence NUMERIC(6,4),            -- 0.0-1.0: Musical positivity (happy vs sad)
    acousticness NUMERIC(6,4),       -- 0.0-1.0: Confidence of acoustic vs electronic
    instrumentalness NUMERIC(6,4),   -- 0.0-1.0: Likelihood of no vocals
    speechiness NUMERIC(6,4),        -- 0.0-1.0: Presence of spoken words
    liveness NUMERIC(6,4),           -- 0.0-1.0: Presence of audience (live vs studio)
    melodicness NUMERIC(6,4),        -- 0.0-1.0: Melodic complexity
    loudness_db NUMERIC(6,2),        -- Overall loudness in decibels
    tempo_confidence NUMERIC(6,4),   -- Confidence in BPM detection
    analysis_version VARCHAR(10),    -- Version of analysis algorithm
    analyzed_at TIMESTAMP DEFAULT NOW()
);
```

## Common Queries

### Get All Completed Analyses
```sql
SELECT 
    t.file_name,
    t.artist,
    t.title,
    af.bpm,
    af.key,
    af.energy,
    af.danceability,
    af.valence,
    t.analyzed_at
FROM tracks t
JOIN audio_features af ON t.id = af.track_id
WHERE t.analysis_status = 'completed'
ORDER BY t.analyzed_at DESC;
```

### Find High Energy Danceable Tracks
```sql
SELECT 
    t.file_name,
    af.bpm,
    af.energy,
    af.danceability
FROM tracks t
JOIN audio_features af ON t.id = af.track_id
WHERE t.analysis_status = 'completed'
    AND af.energy > 0.7
    AND af.danceability > 0.8
ORDER BY af.danceability DESC;
```

### Get Tracks by BPM Range
```sql
SELECT 
    t.file_name,
    af.bpm,
    af.key,
    af.energy
FROM tracks t
JOIN audio_features af ON t.id = af.track_id
WHERE t.analysis_status = 'completed'
    AND af.bpm BETWEEN 120 AND 130
ORDER BY af.bpm;
```

### Analysis Progress Stats
```sql
SELECT 
    analysis_status,
    COUNT(*) as count,
    ROUND(COUNT(*)::numeric / (SELECT COUNT(*) FROM tracks)::numeric * 100, 2) as percentage
FROM tracks
GROUP BY analysis_status
ORDER BY analysis_status;
```

### Average Features by Key
```sql
SELECT 
    af.key,
    COUNT(*) as track_count,
    ROUND(AVG(af.bpm), 2) as avg_bpm,
    ROUND(AVG(af.energy), 4) as avg_energy,
    ROUND(AVG(af.danceability), 4) as avg_danceability
FROM tracks t
JOIN audio_features af ON t.id = af.track_id
WHERE t.analysis_status = 'completed'
GROUP BY af.key
ORDER BY track_count DESC;
```

## API Endpoints

The API server runs on port 8080 and provides RESTful access to the data.

### Health Check
```bash
GET http://localhost:8080/health
```

### Get Track Analysis (Example - check actual implementation)
```bash
# Get track by ID
GET http://localhost:8080/api/v1/tracks/{id}

# Search tracks by criteria
GET http://localhost:8080/api/v1/tracks?bpm_min=120&bpm_max=130&energy_min=0.7

# Get audio features for track
GET http://localhost:8080/api/v1/tracks/{id}/features
```

## Integration with Scraper Service

### Recommended Workflow

1. **Scraper downloads MP3** → Saves to `/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks/`
2. **Analytics worker auto-scans** → Every 30 minutes, discovers new files
3. **Worker imports to DB** → Adds to `tracks` table with status='pending'
4. **Worker analyzes** → Extracts audio features, saves to `audio_features` table
5. **Scraper queries results** → Uses DB or API to get analysis data

### Direct Database Access from Scraper (Python)

```python
import psycopg2
from psycopg2.extras import RealDictCursor

# Connect to database
conn = psycopg2.connect(
    host="localhost",
    port=5432,
    database="doyou_music",
    user="doyou",
    password="REDACTED_ROTATED_CREDENTIAL"
)

# Query completed analyses
with conn.cursor(cursor_factory=RealDictCursor) as cur:
    cur.execute("""
        SELECT 
            t.file_name,
            t.file_path,
            af.bpm,
            af.key,
            af.energy,
            af.danceability,
            af.valence
        FROM tracks t
        JOIN audio_features af ON t.id = af.track_id
        WHERE t.analysis_status = 'completed'
        ORDER BY t.analyzed_at DESC
        LIMIT 100
    """)
    tracks = cur.fetchall()
    
    for track in tracks:
        print(f"{track['file_name']}: {track['bpm']} BPM, {track['key']} key")

conn.close()
```

### Query Specific Track After Download

```python
def get_track_analysis(file_path: str):
    """Get analysis for a specific file path."""
    with conn.cursor(cursor_factory=RealDictCursor) as cur:
        cur.execute("""
            SELECT 
                t.*,
                af.*
            FROM tracks t
            LEFT JOIN audio_features af ON t.id = af.track_id
            WHERE t.file_path = %s
        """, (file_path,))
        return cur.fetchone()

# After scraper downloads a file
file_path = "/mnt/music/downloads/doyou-world-radio-tracks/new-track.mp3"
analysis = get_track_analysis(file_path)

if analysis and analysis['analysis_status'] == 'completed':
    print(f"BPM: {analysis['bpm']}, Energy: {analysis['energy']}")
elif analysis and analysis['analysis_status'] == 'pending':
    print("Analysis in progress, check back later")
else:
    print("Track not yet imported")
```

### Manual Track Import (If Needed)

If the scraper needs to immediately trigger import/analysis:

```bash
# Run import command manually
docker exec doyou-analytics-worker /app/import-tracks /mnt/music/downloads/doyou-world-radio-tracks

# Or from host:
cd /Users/costasskoutelis/srv/do-you-music-analytics
DATABASE_URL="postgresql://doyou:REDACTED_ROTATED_CREDENTIAL@localhost:5432/doyou_music" \
VALIDATE_NAS_ON_STARTUP=false \
go run cmd/import-tracks/main.go /Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks
```

## Feature Interpretation Guide

### Energy (0.0 - 1.0)
- **0.0-0.3**: Calm, quiet, meditative
- **0.3-0.6**: Moderate energy
- **0.6-0.8**: Energetic, upbeat
- **0.8-1.0**: Very intense, loud, aggressive

### Danceability (0.0 - 1.0)
- **0.0-0.3**: Not suitable for dancing
- **0.3-0.6**: Moderate groove
- **0.6-0.8**: Danceable
- **0.8-1.0**: Highly danceable, club-ready

### Valence (0.0 - 1.0)
- **0.0-0.3**: Sad, dark, negative
- **0.3-0.6**: Neutral mood
- **0.6-0.8**: Positive, uplifting
- **0.8-1.0**: Very happy, euphoric

### Acousticness (0.0 - 1.0)
- **0.0-0.3**: Electronic, synthesized
- **0.3-0.7**: Mix of acoustic and electronic
- **0.7-1.0**: Primarily acoustic instruments

### Instrumentalness (0.0 - 1.0)
- **0.0-0.3**: Contains vocals
- **0.3-0.7**: Some vocals, some instrumental
- **0.7-1.0**: Likely instrumental only

## Monitoring

### Check Worker Status
```bash
docker ps --filter name=doyou-analytics
docker logs doyou-analytics-worker --tail 50
```

### View Analysis Progress
```bash
docker exec doyou-analytics-db psql -U doyou -d doyou_music -c \
  "SELECT analysis_status, COUNT(*) FROM tracks GROUP BY analysis_status;"
```

### Recent Completions
```bash
docker exec doyou-analytics-db psql -U doyou -d doyou_music -c \
  "SELECT file_name, analyzed_at FROM tracks 
   WHERE analysis_status = 'completed' 
   ORDER BY analyzed_at DESC LIMIT 10;"
```

## Troubleshooting

### Track Status is 'failed'
Check the error message:
```sql
SELECT file_name, analysis_error 
FROM tracks 
WHERE analysis_status = 'failed' 
LIMIT 10;
```

Common causes:
- Corrupted MP3 file (segmentation fault)
- Malformed audio headers
- Insufficient file permissions

### Worker Not Processing
1. Check worker is running: `docker ps | grep worker`
2. Check worker logs: `docker logs doyou-analytics-worker -f`
3. Restart if stuck: `docker restart doyou-analytics-worker`

### Database Connection Issues
```bash
# Test connection
docker exec doyou-analytics-db psql -U doyou -d doyou_music -c "SELECT 1;"

# Check database is healthy
docker ps --filter name=db
```
