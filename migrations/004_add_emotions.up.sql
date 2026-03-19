-- Migration 004: track_emotions table
-- Stores 8 deterministic emotional dimensions computed from audio features.
-- Each dimension is a DECIMAL(5,4) value in [0.0000, 1.0000].

CREATE TABLE IF NOT EXISTS track_emotions (
    track_id    INTEGER NOT NULL PRIMARY KEY REFERENCES tracks(id) ON DELETE CASCADE,

    -- Emotional dimensions (all 0–1)
    tension     DECIMAL(5,4) NOT NULL DEFAULT 0,
    melancholy  DECIMAL(5,4) NOT NULL DEFAULT 0,
    euphoria    DECIMAL(5,4) NOT NULL DEFAULT 0,
    aggression  DECIMAL(5,4) NOT NULL DEFAULT 0,
    serenity    DECIMAL(5,4) NOT NULL DEFAULT 0,
    nostalgia   DECIMAL(5,4) NOT NULL DEFAULT 0,
    sensuality  DECIMAL(5,4) NOT NULL DEFAULT 0,
    power       DECIMAL(5,4) NOT NULL DEFAULT 0,

    computed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for filtering/sorting by the most queried dimensions
CREATE INDEX idx_emotions_euphoria   ON track_emotions(euphoria);
CREATE INDEX idx_emotions_tension    ON track_emotions(tension);
CREATE INDEX idx_emotions_melancholy ON track_emotions(melancholy);
CREATE INDEX idx_emotions_energy     ON track_emotions(power);
