ALTER TABLE tracks
    ADD COLUMN analysis_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN analysis_started_at TIMESTAMP,
    ADD COLUMN next_retry_at TIMESTAMP;

ALTER TABLE tracks
    ADD CONSTRAINT tracks_analysis_attempts_nonnegative CHECK (analysis_attempts >= 0);

CREATE INDEX idx_tracks_pending_retry
    ON tracks (next_retry_at, downloaded_at)
    WHERE analysis_status = 'pending';

CREATE INDEX idx_tracks_stale_analysis
    ON tracks (analysis_started_at)
    WHERE analysis_status = 'analyzing';