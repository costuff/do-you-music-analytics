package db

import (
	"context"
	"fmt"
	"time"
)

// AudioFeatures represents audio analysis results
type AudioFeatures struct {
	TrackID          int
	BPM              *float64
	Key              *string
	DurationSeconds  *float64
	Energy           *float64
	Danceability     *float64
	Valence          *float64
	Acousticness     *float64
	Instrumentalness *float64
	Speechiness      *float64
	Liveness         *float64
	Melodicness      *float64
	LoudnessDB       *float64
	AnalysisVersion  string
}

// SaveAudioFeatures inserts or updates audio features for a track
func (db *DB) SaveAudioFeatures(ctx context.Context, features *AudioFeatures) error {
	query := `
		INSERT INTO audio_features (
			track_id, bpm, key, duration_seconds,
			energy, danceability, valence, acousticness,
			instrumentalness, speechiness, liveness, melodicness,
			loudness_db, analysis_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (track_id) DO UPDATE SET
			bpm = EXCLUDED.bpm,
			key = EXCLUDED.key,
			duration_seconds = EXCLUDED.duration_seconds,
			energy = EXCLUDED.energy,
			danceability = EXCLUDED.danceability,
			valence = EXCLUDED.valence,
			acousticness = EXCLUDED.acousticness,
			instrumentalness = EXCLUDED.instrumentalness,
			speechiness = EXCLUDED.speechiness,
			liveness = EXCLUDED.liveness,
			melodicness = EXCLUDED.melodicness,
			loudness_db = EXCLUDED.loudness_db,
			analysis_version = EXCLUDED.analysis_version,
			analyzed_at = NOW()`

	_, err := db.pool.Exec(
		ctx, query,
		features.TrackID, features.BPM, features.Key, features.DurationSeconds,
		features.Energy, features.Danceability, features.Valence, features.Acousticness,
		features.Instrumentalness, features.Speechiness, features.Liveness,
		features.Melodicness, features.LoudnessDB, features.AnalysisVersion,
	)

	if err != nil {
		return fmt.Errorf("failed to save audio features: %w", err)
	}

	return nil
}

// CreateTrack inserts a new track
func (db *DB) CreateTrack(ctx context.Context, filePath, fileName string) (int, error) {
	query := `
		INSERT INTO tracks (file_path, file_name, analysis_status)
		VALUES ($1, $2, 'pending')
		RETURNING id`

	var trackID int
	err := db.pool.QueryRow(ctx, query, filePath, fileName).Scan(&trackID)
	if err != nil {
		return 0, fmt.Errorf("failed to create track: %w", err)
	}

	return trackID, nil
}

// UpdateTrackStatus updates the analysis status of a track
func (db *DB) UpdateTrackStatus(ctx context.Context, trackID int, status string, errorMsg *string) error {
	var analyzedAt *time.Time
	if status == "completed" {
		now := time.Now()
		analyzedAt = &now
	}

	query := `
		UPDATE tracks
		SET analysis_status = $1,
		    analysis_error = $2,
		    analyzed_at = COALESCE($3, analyzed_at)
		WHERE id = $4`

	_, err := db.pool.Exec(ctx, query, status, errorMsg, analyzedAt, trackID)
	if err != nil {
		return fmt.Errorf("failed to update track status: %w", err)
	}

	return nil
}

// GetTrackByPath retrieves a track by file path
func (db *DB) GetTrackByPath(ctx context.Context, filePath string) (*Track, error) {
	query := `SELECT id, file_path, analysis_status FROM tracks WHERE file_path = $1`

	track := &Track{}
	err := db.pool.QueryRow(ctx, query, filePath).Scan(&track.ID, &track.FilePath, &track.AnalysisStatus)
	if err != nil {
		return nil, fmt.Errorf("track not found: %w", err)
	}

	return track, nil
}
