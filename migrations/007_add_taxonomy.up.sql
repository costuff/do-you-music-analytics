-- Migration 007: Add taxonomy nodes and track labels for the AI curation system.

-- taxonomy_nodes: hierarchical tree of emotion/mood labels.
--   source = 'auto' (seeded) or 'user' (added via curation UI).
CREATE TABLE IF NOT EXISTS taxonomy_nodes (
    id          INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    parent_id   INTEGER REFERENCES taxonomy_nodes(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    source      VARCHAR(20) NOT NULL DEFAULT 'auto'
                    CHECK (source IN ('auto', 'user')),
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Unique (parent, lower(name)) so the same label cannot appear twice under one parent.
CREATE UNIQUE INDEX IF NOT EXISTS idx_taxonomy_parent_name
    ON taxonomy_nodes(COALESCE(parent_id, -1), lower(name));

-- track_labels: one row per track with the computed auto-label and optional user curation.
--   label_path      = user-curated path (JSONB array e.g. ["euphoria","summer vibez"])
--   auto_label_path = system-computed path (currently just the dominant emotion zone)
CREATE TABLE IF NOT EXISTS track_labels (
    id              INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    track_id        INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    label_path      JSONB NOT NULL DEFAULT '[]',
    auto_label_path JSONB NOT NULL DEFAULT '[]',
    curated_by      TEXT,
    notes           TEXT,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(track_id)
);

CREATE INDEX IF NOT EXISTS idx_track_labels_label_path
    ON track_labels USING gin(label_path);

-- Seed the 8 root-level emotion nodes that mirror the emotion dimensions.
INSERT INTO taxonomy_nodes (name, description, source) VALUES
    ('euphoria',   'High energy, positive, danceable tracks',      'auto'),
    ('tension',    'High energy, negative or suspenseful tracks',   'auto'),
    ('melancholy', 'Slow, sad, introspective tracks',               'auto'),
    ('aggression', 'Assertive, high-energy, dark tracks',           'auto'),
    ('serenity',   'Calm, peaceful, acoustic tracks',               'auto'),
    ('nostalgia',  'Melodic, acoustic, bittersweet tracks',         'auto'),
    ('sensuality', 'Smooth, melodic, moderate tempo tracks',        'auto'),
    ('power',      'Energetic, driving, strong rhythm tracks',      'auto')
ON CONFLICT DO NOTHING;
