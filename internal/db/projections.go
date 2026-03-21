package db

import (
	"context"
	"fmt"
	"math"
	"time"
)

// TrackProjection holds a 2D coordinate for a single track, derived from its
// emotion vector. Used by the hierarchical emotion map in the frontend.
//
// Axes:
//   - ProjX = arousal  : (euphoria + power + aggression - serenity - nostalgia - melancholy) / 3
//   - ProjY = valence  : (euphoria + sensuality + serenity - tension - aggression - melancholy) / 3
//
// Both axes are clamped to [-1, 1]. Positive X = high arousal, positive Y = positive valence.
type TrackProjection struct {
	TrackID    int64     `json:"track_id"`
	ProjX      float64   `json:"proj_x"`
	ProjY      float64   `json:"proj_y"`
	Zone       string    `json:"zone"`
	ComputedAt time.Time `json:"computed_at"`
}

// ComputeProjection derives a 2D coordinate and dominant zone from a TrackEmotions record.
func ComputeProjection(e TrackEmotions) TrackProjection {
	clamp := func(v float64) float64 { return math.Max(-1, math.Min(1, v)) }

	x := clamp((e.Euphoria + e.Power + e.Aggression - e.Serenity - e.Nostalgia - e.Melancholy) / 3.0)
	y := clamp((e.Euphoria + e.Sensuality + e.Serenity - e.Tension - e.Aggression - e.Melancholy) / 3.0)

	return TrackProjection{
		TrackID:    int64(e.TrackID),
		ProjX:      x,
		ProjY:      y,
		Zone:       dominantEmotion(e),
		ComputedAt: time.Now(),
	}
}

func dominantEmotion(e TrackEmotions) string {
	candidates := []struct {
		name string
		val  float64
	}{
		{"euphoria", e.Euphoria},
		{"tension", e.Tension},
		{"melancholy", e.Melancholy},
		{"aggression", e.Aggression},
		{"serenity", e.Serenity},
		{"nostalgia", e.Nostalgia},
		{"sensuality", e.Sensuality},
		{"power", e.Power},
	}
	best := "serenity"
	maxVal := -1.0
	for _, c := range candidates {
		if c.val > maxVal {
			maxVal = c.val
			best = c.name
		}
	}
	return best
}

// SaveProjection upserts one track projection row.
func (db *DB) SaveProjection(ctx context.Context, p TrackProjection) error {
	query := `
		INSERT INTO track_projections (track_id, proj_x, proj_y, zone, computed_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (track_id) DO UPDATE SET
			proj_x      = EXCLUDED.proj_x,
			proj_y      = EXCLUDED.proj_y,
			zone        = EXCLUDED.zone,
			computed_at = EXCLUDED.computed_at
	`
	_, err := db.pool.Exec(ctx, query, p.TrackID, p.ProjX, p.ProjY, p.Zone, p.ComputedAt)
	if err != nil {
		return fmt.Errorf("save projection for track %d: %w", p.TrackID, err)
	}
	return nil
}

// GetAllProjections returns up to limit track projections ordered by track_id.
func (db *DB) GetAllProjections(ctx context.Context, limit int) ([]TrackProjection, error) {
	if limit <= 0 || limit > 20000 {
		limit = 10000
	}
	query := `SELECT track_id, proj_x, proj_y, zone, computed_at
	          FROM track_projections ORDER BY track_id LIMIT $1`
	rows, err := db.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query projections: %w", err)
	}
	defer rows.Close()

	var out []TrackProjection
	for rows.Next() {
		var p TrackProjection
		if err := rows.Scan(&p.TrackID, &p.ProjX, &p.ProjY, &p.Zone, &p.ComputedAt); err != nil {
			return nil, fmt.Errorf("scan projection: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ComputeAndSaveAllProjections recomputes projections for every track that has
// emotions. Returns the number of projections saved.
func (db *DB) ComputeAndSaveAllProjections(ctx context.Context) (int, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT track_id, tension, melancholy, euphoria, aggression,
		       serenity, nostalgia, sensuality, power, computed_at
		FROM track_emotions
	`)
	if err != nil {
		return 0, fmt.Errorf("query emotions for projection: %w", err)
	}
	defer rows.Close()

	var projections []TrackProjection
	for rows.Next() {
		var e TrackEmotions
		if err := rows.Scan(
			&e.TrackID, &e.Tension, &e.Melancholy, &e.Euphoria, &e.Aggression,
			&e.Serenity, &e.Nostalgia, &e.Sensuality, &e.Power, &e.ComputedAt,
		); err != nil {
			return 0, fmt.Errorf("scan emotion row: %w", err)
		}
		projections = append(projections, ComputeProjection(e))
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	count := 0
	for _, p := range projections {
		if err := db.SaveProjection(ctx, p); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
