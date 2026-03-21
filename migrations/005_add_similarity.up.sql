-- Migration 005: track_similarity table
-- Stores pre-computed cosine similarity scores between tracks in emotion space.
-- Each row (track_a_id, track_b_id) represents that track_b is among the
-- top-k most similar tracks to track_a. Both (a,b) and (b,a) are stored
-- independently so that each track's neighbourhood can be queried with
-- a simple WHERE track_a_id = $1 using the index below.

CREATE TABLE IF NOT EXISTS track_similarity (
    track_a_id  BIGINT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    track_b_id  BIGINT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    score       DECIMAL(6,5) NOT NULL,
    computed_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    PRIMARY KEY (track_a_id, track_b_id),
    CHECK (track_a_id <> track_b_id)
);

-- Primary lookup: "give me the top-N neighbours for track X"
CREATE INDEX IF NOT EXISTS idx_track_similarity_a_score
    ON track_similarity (track_a_id, score DESC);
