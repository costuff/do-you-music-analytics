-- Drop views
DROP VIEW IF EXISTS analysis_stats;
DROP VIEW IF EXISTS pending_tracks;
DROP VIEW IF EXISTS tracks_with_features;

-- Drop tables (reverse order due to foreign keys)
DROP TABLE IF EXISTS analysis_jobs;
DROP TABLE IF EXISTS audio_features;
DROP TABLE IF EXISTS tracks;
