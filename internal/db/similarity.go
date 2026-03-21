package db

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// TrackSimilarityResult is a neighbour with its similarity score and track data.
type TrackSimilarityResult struct {
	TrackID    int64              `json:"track_id"`
	Score      float64            `json:"score"`
	ComputedAt time.Time          `json:"computed_at"`
	Track      *TrackWithFeatures `json:"track,omitempty"`
}

// cosineSim returns the cosine similarity of two equal-length float64 slices.
// Returns 0 for zero-magnitude vectors.
func cosineSim(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// toVector converts a TrackEmotions record to a float64 slice for similarity
// computation. Order must stay stable — it defines the vector space.
func toVector(e TrackEmotions) []float64 {
	return []float64{
		e.Tension, e.Melancholy, e.Euphoria, e.Aggression,
		e.Serenity, e.Nostalgia, e.Sensuality, e.Power,
	}
}

type neighborPair struct {
	trackID int
	score   float64
}

// GetAllEmotionVectors loads every row from track_emotions without pagination.
// Used by ComputeSimilarity to build the full similarity matrix.
func (db *DB) GetAllEmotionVectors(ctx context.Context) ([]TrackEmotions, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT track_id, tension, melancholy, euphoria, aggression,
		       serenity, nostalgia, sensuality, power
		FROM   track_emotions
		ORDER  BY track_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query emotion vectors: %w", err)
	}
	defer rows.Close()

	var out []TrackEmotions
	for rows.Next() {
		var e TrackEmotions
		if err := rows.Scan(
			&e.TrackID, &e.Tension, &e.Melancholy, &e.Euphoria, &e.Aggression,
			&e.Serenity, &e.Nostalgia, &e.Sensuality, &e.Power,
		); err != nil {
			return nil, fmt.Errorf("scan emotion vector: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ComputeSimilarity loads all track emotion vectors, computes cosine similarity
// for every pair, keeps the top-k neighbours per track, and upserts the results
// into track_similarity (one row per directed edge: source → neighbour).
//
// Returns the total number of rows upserted (= n_tracks × k, or fewer if some
// tracks have fewer than k neighbours).
func (db *DB) ComputeSimilarity(ctx context.Context, k int) (int, error) {
	if k <= 0 {
		k = 20
	}

	emotions, err := db.GetAllEmotionVectors(ctx)
	if err != nil {
		return 0, fmt.Errorf("load emotion vectors: %w", err)
	}
	n := len(emotions)
	if n == 0 {
		return 0, nil
	}

	// Build matrix of vectors once to avoid repeated struct field access.
	vectors := make([][]float64, n)
	for i, e := range emotions {
		vectors[i] = toVector(e)
	}

	topK := min(k, n-1)

	// Collect all (source, neighbour, score) triples.
	type edge struct {
		a, b  int64
		score float64
	}
	pairs := make([]edge, 0, n*topK)

	for i := 0; i < n; i++ {
		neighbors := make([]neighborPair, 0, n-1)
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			s := cosineSim(vectors[i], vectors[j])
			neighbors = append(neighbors, neighborPair{emotions[j].TrackID, s})
		}
		sort.Slice(neighbors, func(a, b int) bool {
			return neighbors[a].score > neighbors[b].score
		})
		if len(neighbors) > topK {
			neighbors = neighbors[:topK]
		}
		for _, nb := range neighbors {
			pairs = append(pairs, edge{
				a:     int64(emotions[i].TrackID),
				b:     int64(nb.trackID),
				score: nb.score,
			})
		}
	}

	// Batch upsert in transactions of up to 1 000 rows.
	const batchSize = 1000
	inserted := 0
	for start := 0; start < len(pairs); start += batchSize {
		end := start + batchSize
		if end > len(pairs) {
			end = len(pairs)
		}
		batch := pairs[start:end]

		tx, err := db.pool.Begin(ctx)
		if err != nil {
			return inserted, fmt.Errorf("begin similarity batch tx: %w", err)
		}

		for _, p := range batch {
			if _, err := tx.Exec(ctx, `
				INSERT INTO track_similarity (track_a_id, track_b_id, score, computed_at)
				VALUES ($1, $2, $3, NOW())
				ON CONFLICT (track_a_id, track_b_id) DO UPDATE
					SET score = EXCLUDED.score, computed_at = EXCLUDED.computed_at
			`, p.a, p.b, p.score); err != nil {
				_ = tx.Rollback(ctx)
				return inserted, fmt.Errorf("upsert similarity (%d,%d): %w", p.a, p.b, err)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			return inserted, fmt.Errorf("commit similarity batch: %w", err)
		}
		inserted += len(batch)
	}

	return inserted, nil
}

// GetSimilarTracks returns the top-n pre-computed neighbours for the given
// track, ordered by descending score, including full track + feature data.
func (db *DB) GetSimilarTracks(ctx context.Context, trackID int64, n int) ([]TrackSimilarityResult, error) {
	if n <= 0 {
		n = 10
	}
	if n > 50 {
		n = 50
	}

	rows, err := db.pool.Query(ctx, `
		SELECT ts.track_b_id, ts.score, ts.computed_at,
		       t.id, t.file_path, t.file_name, t.analysis_status, t.analysis_error,
		       t.downloaded_at, t.analyzed_at,
		       af.bpm, af.key, af.mode, af.duration_seconds, af.loudness_db,
		       af.energy, af.danceability, af.speechiness, af.acousticness,
		       af.instrumentalness, af.liveness, af.valence, af.melodicness
		FROM   track_similarity ts
		JOIN   tracks           t  ON t.id  = ts.track_b_id
		LEFT   JOIN audio_features af ON af.track_id = ts.track_b_id
		WHERE  ts.track_a_id = $1
		ORDER  BY ts.score DESC
		LIMIT  $2
	`, trackID, n)
	if err != nil {
		return nil, fmt.Errorf("get similar tracks for %d: %w", trackID, err)
	}
	defer rows.Close()

	var result []TrackSimilarityResult
	for rows.Next() {
		var r TrackSimilarityResult
		var track TrackWithFeatures
		var features AudioFeatures

		if err := rows.Scan(
			&r.TrackID, &r.Score, &r.ComputedAt,
			&track.ID, &track.FilePath, &track.FileName, &track.AnalysisStatus, &track.AnalysisError,
			&track.DownloadedAt, &track.AnalyzedAt,
			&features.BPM, &features.Key, &features.Mode, &features.DurationSeconds, &features.LoudnessDB,
			&features.Energy, &features.Danceability, &features.Speechiness, &features.Acousticness,
			&features.Instrumentalness, &features.Liveness, &features.Valence, &features.Melodicness,
		); err != nil {
			return nil, fmt.Errorf("scan similar track row: %w", err)
		}

		if features.BPM != nil {
			track.Features = &features
		}
		r.Track = &track
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// SimilarityStats holds summary information about the similarity table.
type SimilarityStats struct {
	TotalPairs    int64      `json:"total_pairs"`
	CoveredTracks int64      `json:"covered_tracks"`
	LastComputed  *time.Time `json:"last_computed,omitempty"`
}

// GetSimilarityStats returns a summary of the current similarity table state.
func (db *DB) GetSimilarityStats(ctx context.Context) (SimilarityStats, error) {
	var stats SimilarityStats
	err := db.pool.QueryRow(ctx, `
		SELECT COUNT(*) AS total_pairs,
		       COUNT(DISTINCT track_a_id) AS covered_tracks,
		       MAX(computed_at) AS last_computed
		FROM track_similarity
	`).Scan(&stats.TotalPairs, &stats.CoveredTracks, &stats.LastComputed)
	if err != nil {
		return stats, fmt.Errorf("similarity stats: %w", err)
	}
	return stats, nil
}
