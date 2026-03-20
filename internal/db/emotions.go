package db

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// TrackEmotions holds the 8 emotional dimensions for a track.
// All values are in [0, 1] and are computed deterministically from AudioFeatures.
type TrackEmotions struct {
	TrackID    int       `json:"track_id"`
	Tension    float64   `json:"tension"`
	Melancholy float64   `json:"melancholy"`
	Euphoria   float64   `json:"euphoria"`
	Aggression float64   `json:"aggression"`
	Serenity   float64   `json:"serenity"`
	Nostalgia  float64   `json:"nostalgia"`
	Sensuality float64   `json:"sensuality"`
	Power      float64   `json:"power"`
	ComputedAt time.Time `json:"computed_at"`
}

// EmotionDistribution holds aggregate statistics for a single emotion dimension.
type EmotionDistribution struct {
	Dimension string  `json:"dimension"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Avg       float64 `json:"avg"`
	Count     int64   `json:"count"`
}

// ComputeEmotions derives the 8 emotional dimensions deterministically from
// the given AudioFeatures using the mapping matrix defined in the project TODO.
//
// All input feature pointers may be nil; sensible defaults are used in that case.
// All returned values are clamped to [0, 1].
func ComputeEmotions(f *AudioFeatures) TrackEmotions {
	pf := func(p *float64, def float64) float64 {
		if p != nil {
			return *p
		}
		return def
	}

	clamp := func(v float64) float64 {
		return math.Max(0, math.Min(1, v))
	}

	energy          := pf(f.Energy, 0.5)
	valence         := pf(f.Valence, 0.5)
	danceability    := pf(f.Danceability, 0.5)
	acousticness    := pf(f.Acousticness, 0.5)
	speechiness     := pf(f.Speechiness, 0.3)
	liveness        := pf(f.Liveness, 0.3)
	melodicness     := pf(f.Melodicness, 0.5)

	bpmNorm := 0.5
	if f.BPM != nil {
		bpmNorm = math.Min(1.0, *f.BPM/200.0)
	}

	modeIsMajor := 0.0
	if f.Mode != nil && *f.Mode == "major" {
		modeIsMajor = 1.0
	}

	// Mapping matrix (defined in docs/TODO.md)
	tension    := clamp(0.4*energy + 0.3*(1-valence) + 0.2*bpmNorm + 0.1*speechiness)
	melancholy := clamp(0.5*(1-valence) + 0.3*(1-energy) + 0.2*(1-modeIsMajor))
	euphoria   := clamp(0.4*valence + 0.3*energy + 0.2*danceability + 0.1*(1-speechiness))
	aggression := clamp(0.4*energy + 0.3*(1-acousticness) + 0.2*bpmNorm + 0.1*(1-valence))
	// serenity uses tension that was just computed — no circular dependency
	serenity   := clamp(0.4*acousticness + 0.3*(1-energy) + 0.2*valence + 0.1*(1-tension))
	nostalgia  := clamp(0.3*acousticness + 0.3*melodicness + 0.2*(1-valence) + 0.2*(1-danceability))
	sensuality := clamp(0.4*valence + 0.3*(1-bpmNorm) + 0.2*melodicness + 0.1*liveness)
	power      := clamp(0.5*energy + 0.3*(1-acousticness) + 0.2*bpmNorm)

	return TrackEmotions{
		TrackID:    f.TrackID,
		Tension:    tension,
		Melancholy: melancholy,
		Euphoria:   euphoria,
		Aggression: aggression,
		Serenity:   serenity,
		Nostalgia:  nostalgia,
		Sensuality: sensuality,
		Power:      power,
		ComputedAt: time.Now(),
	}
}

// SaveEmotions upserts emotion dimensions for a track.
func (db *DB) SaveEmotions(ctx context.Context, e *TrackEmotions) error {
	query := `
		INSERT INTO track_emotions
			(track_id, tension, melancholy, euphoria, aggression,
			 serenity, nostalgia, sensuality, power, computed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (track_id) DO UPDATE SET
			tension    = EXCLUDED.tension,
			melancholy = EXCLUDED.melancholy,
			euphoria   = EXCLUDED.euphoria,
			aggression = EXCLUDED.aggression,
			serenity   = EXCLUDED.serenity,
			nostalgia  = EXCLUDED.nostalgia,
			sensuality = EXCLUDED.sensuality,
			power      = EXCLUDED.power,
			computed_at = NOW()
	`
	_, err := db.pool.Exec(ctx, query,
		e.TrackID, e.Tension, e.Melancholy, e.Euphoria, e.Aggression,
		e.Serenity, e.Nostalgia, e.Sensuality, e.Power)
	if err != nil {
		return fmt.Errorf("save emotions for track %d: %w", e.TrackID, err)
	}
	return nil
}

// GetEmotions returns the emotion record for a single track.
// Returns (nil, nil) when the track has no computed emotions yet.
func (db *DB) GetEmotions(ctx context.Context, trackID int64) (*TrackEmotions, error) {
	query := `
		SELECT track_id, tension, melancholy, euphoria, aggression,
		       serenity, nostalgia, sensuality, power, computed_at
		FROM track_emotions
		WHERE track_id = $1
	`
	var e TrackEmotions
	err := db.pool.QueryRow(ctx, query, trackID).Scan(
		&e.TrackID, &e.Tension, &e.Melancholy, &e.Euphoria, &e.Aggression,
		&e.Serenity, &e.Nostalgia, &e.Sensuality, &e.Power, &e.ComputedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, fmt.Errorf("get emotions for track %d: %w", trackID, err)
	}
	return &e, nil
}

// GetEmotionsDistribution returns min/max/avg/count for every emotion dimension.
// Returns an empty slice when the table has no rows yet.
func (db *DB) GetEmotionsDistribution(ctx context.Context) ([]EmotionDistribution, error) {
	// Use COALESCE so that MIN/MAX/AVG return 0 instead of NULL on an empty table,
	// allowing clean scanning into float64 without nullable wrapper types.
	query := `
		SELECT 'tension'    AS dim, COALESCE(MIN(tension),0),    COALESCE(MAX(tension),0),    COALESCE(AVG(tension),0),    COUNT(*) FROM track_emotions
		UNION ALL
		SELECT 'melancholy',        COALESCE(MIN(melancholy),0), COALESCE(MAX(melancholy),0), COALESCE(AVG(melancholy),0), COUNT(*) FROM track_emotions
		UNION ALL
		SELECT 'euphoria',          COALESCE(MIN(euphoria),0),   COALESCE(MAX(euphoria),0),   COALESCE(AVG(euphoria),0),   COUNT(*) FROM track_emotions
		UNION ALL
		SELECT 'aggression',        COALESCE(MIN(aggression),0), COALESCE(MAX(aggression),0), COALESCE(AVG(aggression),0), COUNT(*) FROM track_emotions
		UNION ALL
		SELECT 'serenity',          COALESCE(MIN(serenity),0),   COALESCE(MAX(serenity),0),   COALESCE(AVG(serenity),0),   COUNT(*) FROM track_emotions
		UNION ALL
		SELECT 'nostalgia',         COALESCE(MIN(nostalgia),0),  COALESCE(MAX(nostalgia),0),  COALESCE(AVG(nostalgia),0),  COUNT(*) FROM track_emotions
		UNION ALL
		SELECT 'sensuality',        COALESCE(MIN(sensuality),0), COALESCE(MAX(sensuality),0), COALESCE(AVG(sensuality),0), COUNT(*) FROM track_emotions
		UNION ALL
		SELECT 'power',             COALESCE(MIN(power),0),      COALESCE(MAX(power),0),      COALESCE(AVG(power),0),      COUNT(*) FROM track_emotions
	`
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("emotions distribution query: %w", err)
	}
	defer rows.Close()

	var dist []EmotionDistribution
	for rows.Next() {
		var d EmotionDistribution
		if err := rows.Scan(&d.Dimension, &d.Min, &d.Max, &d.Avg, &d.Count); err != nil {
			return nil, fmt.Errorf("scan distribution row: %w", err)
		}
		dist = append(dist, d)
	}
	return dist, rows.Err()
}
