DROP INDEX IF EXISTS idx_tracks_stale_analysis;
DROP INDEX IF EXISTS idx_tracks_pending_retry;
ALTER TABLE tracks DROP CONSTRAINT IF EXISTS tracks_analysis_attempts_nonnegative;
ALTER TABLE tracks
    DROP COLUMN IF EXISTS next_retry_at,
    DROP COLUMN IF EXISTS analysis_started_at,
    DROP COLUMN IF EXISTS analysis_attempts;