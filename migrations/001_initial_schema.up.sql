-- Create tracks table
CREATE TABLE IF NOT EXISTS tracks (
    id SERIAL PRIMARY KEY,
    file_path VARCHAR(512) NOT NULL UNIQUE,
    file_name VARCHAR(255) NOT NULL,
    title VARCHAR(255),
    artist VARCHAR(255),
    album VARCHAR(255),
    label VARCHAR(255),
    release_date DATE,
    file_size_bytes BIGINT,
    duration_seconds DECIMAL(10, 2),
    
    -- Timestamps
    downloaded_at TIMESTAMP DEFAULT NOW(),
    analyzed_at TIMESTAMP,
    
    -- Status tracking
    analysis_status VARCHAR(20) DEFAULT 'pending',
    analysis_error TEXT,
    
    -- Constraints
    CONSTRAINT unique_file_path UNIQUE (file_path),
    CONSTRAINT valid_analysis_status CHECK (analysis_status IN ('pending', 'analyzing', 'completed', 'failed'))
);

-- Indexes for tracks table
CREATE INDEX idx_tracks_analysis_status ON tracks(analysis_status);
CREATE INDEX idx_tracks_analyzed_at ON tracks(analyzed_at);
CREATE INDEX idx_tracks_artist ON tracks(artist);
CREATE INDEX idx_tracks_downloaded_at ON tracks(downloaded_at);
CREATE INDEX idx_tracks_file_name ON tracks(file_name);

-- Create audio_features table
CREATE TABLE IF NOT EXISTS audio_features (
    id SERIAL PRIMARY KEY,
    track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    
    -- Core features
    bpm DECIMAL(6, 2),
    key VARCHAR(10),
    mode VARCHAR(10),
    duration_seconds DECIMAL(10, 2),
    
    -- Spotify-like features (0.0-1.0 normalized)
    energy DECIMAL(5, 4),
    danceability DECIMAL(5, 4),
    valence DECIMAL(5, 4),
    acousticness DECIMAL(5, 4),
    instrumentalness DECIMAL(5, 4),
    speechiness DECIMAL(5, 4),
    liveness DECIMAL(5, 4),
    melodicness DECIMAL(5, 4),
    
    -- Technical features
    loudness_db DECIMAL(6, 2),
    tempo_confidence DECIMAL(5, 4),
    
    -- Analysis metadata
    analysis_version VARCHAR(20) DEFAULT '1.0.0',
    analyzed_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_track_features UNIQUE (track_id)
);

-- Indexes for audio_features table
CREATE INDEX idx_features_bpm ON audio_features(bpm);
CREATE INDEX idx_features_key ON audio_features(key);
CREATE INDEX idx_features_energy ON audio_features(energy);
CREATE INDEX idx_features_danceability ON audio_features(danceability);
CREATE INDEX idx_features_valence ON audio_features(valence);

-- Create analysis_jobs table
CREATE TABLE IF NOT EXISTS analysis_jobs (
    id SERIAL PRIMARY KEY,
    status VARCHAR(20) DEFAULT 'pending',
    total_tracks INTEGER DEFAULT 0,
    analyzed_tracks INTEGER DEFAULT 0,
    failed_tracks INTEGER DEFAULT 0,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    
    -- Constraints
    CONSTRAINT valid_job_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'))
);

-- Index for jobs table
CREATE INDEX idx_jobs_status ON analysis_jobs(status);
CREATE INDEX idx_jobs_started_at ON analysis_jobs(started_at);

-- Create views for common queries
CREATE OR REPLACE VIEW tracks_with_features AS
SELECT 
    t.*,
    af.bpm,
    af.key,
    af.mode,
    af.energy,
    af.danceability,
    af.valence,
    af.acousticness,
    af.instrumentalness,
    af.speechiness,
    af.liveness,
    af.melodicness,
    af.loudness_db
FROM tracks t
LEFT JOIN audio_features af ON t.id = af.track_id;

-- Create view for pending tracks
CREATE OR REPLACE VIEW pending_tracks AS
SELECT * FROM tracks
WHERE analysis_status = 'pending' AND analyzed_at IS NULL
ORDER BY downloaded_at DESC;

-- Create view for analysis statistics
CREATE OR REPLACE VIEW analysis_stats AS
SELECT
    COUNT(*) as total_tracks,
    COUNT(CASE WHEN analyzed_at IS NOT NULL THEN 1 END) as analyzed_tracks,
    COUNT(CASE WHEN analysis_status = 'pending' THEN 1 END) as pending_tracks,
    COUNT(CASE WHEN analysis_status = 'failed' THEN 1 END) as failed_tracks,
    ROUND(AVG(CASE WHEN analyzed_at IS NOT NULL THEN 100.0 ELSE 0 END), 2) as completion_percentage
FROM tracks;
