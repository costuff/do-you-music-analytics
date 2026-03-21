-- Migration 006: Add 2D projection coordinates for the hierarchical emotion map.
-- proj_x / proj_y are in [-1, 1] (arousal × valence axes).
-- zone = dominant emotion name for that track.

CREATE TABLE IF NOT EXISTS track_projections (
    track_id    INTEGER NOT NULL PRIMARY KEY REFERENCES tracks(id) ON DELETE CASCADE,
    proj_x      DECIMAL(8,5) NOT NULL DEFAULT 0,
    proj_y      DECIMAL(8,5) NOT NULL DEFAULT 0,
    zone        VARCHAR(50)  NOT NULL DEFAULT '',
    computed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_track_projections_zone ON track_projections(zone);
