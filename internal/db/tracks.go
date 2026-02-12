package db

import (
	"context"
	"fmt"
	"time"
)

type Track struct {
	ID             int
	FilePath       string
	AnalysisStatus string
}

// TrackWithFeatures represents a track with its audio features
type TrackWithFeatures struct {
	ID             int64      `json:"id"`
	FilePath       string     `json:"file_path"`
	FileName       string     `json:"file_name"`
	AnalysisStatus string     `json:"analysis_status"`
	AnalysisError  *string    `json:"analysis_error,omitempty"`
	DownloadedAt   *time.Time `json:"downloaded_at"`
	AnalyzedAt     *time.Time `json:"analyzed_at,omitempty"`
	
	// Audio features (will be null if not analyzed yet)
	Features *AudioFeatures `json:"features,omitempty"`
}

func (db *DB) GetTrackStats(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE analysis_status = 'pending') as pending,
			COUNT(*) FILTER (WHERE analysis_status = 'analyzing') as analyzing,
			COUNT(*) FILTER (WHERE analysis_status = 'completed') as completed,
			COUNT(*) FILTER (WHERE analysis_status = 'failed') as failed
		FROM tracks
	`
	
	var total, pending, analyzing, completed, failed int64
	err := db.pool.QueryRow(ctx, query).Scan(&total, &pending, &analyzing, &completed, &failed)
	if err != nil {
		return nil, fmt.Errorf("failed to get track stats: %w", err)
	}
	
	stats := map[string]int64{
		"total":     total,
		"pending":   pending,
		"analyzing": analyzing,
		"completed": completed,
		"failed":    failed,
	}
	
	return stats, nil
}

func (db *DB) GetPendingTracks(ctx context.Context, limit int) ([]*Track, error) {
	query := `
		SELECT id, file_path, analysis_status
		FROM tracks
		WHERE analysis_status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`
	
	rows, err := db.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending tracks: %w", err)
	}
	defer rows.Close()
	
	var tracks []*Track
	for rows.Next() {
		track := &Track{}
		if err := rows.Scan(&track.ID, &track.FilePath, &track.AnalysisStatus); err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, track)
	}
	
	return tracks, nil
}

// ListTracks returns paginated list of tracks with optional status filter
func (db *DB) ListTracks(ctx context.Context, limit, offset int, status string) ([]TrackWithFeatures, int64, error) {
	// Build query with optional status filter
	query := `
		SELECT t.id, t.file_path, t.file_name, t.analysis_status, t.analysis_error, 
		       t.downloaded_at, t.analyzed_at,
		       af.bpm, af.key, af.duration_seconds, af.loudness_db,
		       af.energy, af.danceability, af.speechiness, af.acousticness,
		       af.instrumentalness, af.liveness, af.valence, af.melodicness
		FROM tracks t
		LEFT JOIN audio_features af ON t.id = af.track_id
	`
	
	args := []interface{}{limit, offset}
	countArgs := []interface{}{}
	
	if status != "" {
		query += " WHERE t.analysis_status = $3"
		args = append(args, status)
		countArgs = append(countArgs, status)
	}
	
	query += " ORDER BY t.downloaded_at DESC LIMIT $1 OFFSET $2"
	
	// Get total count
	countQuery := "SELECT COUNT(*) FROM tracks"
	if status != "" {
		countQuery += " WHERE analysis_status = $1"
	}
	
	var total int64
	if err := db.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}
	
	// Get tracks
	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	var tracks []TrackWithFeatures
	for rows.Next() {
		var track TrackWithFeatures
		var features AudioFeatures
		var hasFeatures bool
		
		err := rows.Scan(
			&track.ID, &track.FilePath, &track.FileName, &track.AnalysisStatus, &track.AnalysisError,
			&track.DownloadedAt, &track.AnalyzedAt,
			&features.BPM, &features.Key, &features.DurationSeconds, &features.LoudnessDB,
			&features.Energy, &features.Danceability, &features.Speechiness,
			&features.Acousticness, &features.Instrumentalness, &features.Liveness,
			&features.Valence, &features.Melodicness,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan failed: %w", err)
		}
		
		// Check if features exist (BPM will be null if no features)
		if features.BPM != nil {
			hasFeatures = true
			track.Features = &features
		}
		
		if !hasFeatures {
			track.Features = nil
		}
		
		tracks = append(tracks, track)
	}
	
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration failed: %w", err)
	}
	
	return tracks, total, nil
}

// GetTrackByID returns a single track by ID with audio features
func (db *DB) GetTrackByID(ctx context.Context, trackID int64) (*TrackWithFeatures, error) {
	query := `
		SELECT t.id, t.file_path, t.file_name, t.analysis_status, t.analysis_error,
		       t.downloaded_at, t.analyzed_at,
		       af.bpm, af.key, af.duration_seconds, af.loudness_db,
		       af.energy, af.danceability, af.speechiness, af.acousticness,
		       af.instrumentalness, af.liveness, af.valence, af.melodicness
		FROM tracks t
		LEFT JOIN audio_features af ON t.id = af.track_id
		WHERE t.id = $1
	`
	
	var track TrackWithFeatures
	var features AudioFeatures
	
	err := db.pool.QueryRow(ctx, query, trackID).Scan(
		&track.ID, &track.FilePath, &track.FileName, &track.AnalysisStatus, &track.AnalysisError,
		&track.DownloadedAt, &track.AnalyzedAt,
		&features.BPM, &features.Key, &features.DurationSeconds,
		&features.LoudnessDB, &features.Energy, &features.Danceability, &features.Speechiness,
		&features.Acousticness, &features.Instrumentalness, &features.Liveness,
		&features.Valence, &features.Melodicness,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	
	// Check if features exist
	if features.BPM != nil {
		track.Features = &features
	}
	
	return &track, nil
}

// SearchTracks searches for tracks by filename or path (case-insensitive)
func (db *DB) SearchTracks(ctx context.Context, query string, limit int) ([]TrackWithFeatures, error) {
	searchQuery := `
		SELECT t.id, t.file_path, t.file_name, t.analysis_status, t.analysis_error,
		       t.downloaded_at, t.analyzed_at,
		       af.bpm, af.key, af.duration_seconds, af.loudness_db,
		       af.energy, af.danceability, af.speechiness, af.acousticness,
		       af.instrumentalness, af.liveness, af.valence, af.melodicness
		FROM tracks t
		LEFT JOIN audio_features af ON t.id = af.track_id
		WHERE LOWER(t.file_name) LIKE LOWER($1) OR LOWER(t.file_path) LIKE LOWER($1)
		ORDER BY t.downloaded_at DESC
		LIMIT $2
	`
	
	searchPattern := "%" + query + "%"
	rows, err := db.pool.Query(ctx, searchQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()
	
	var tracks []TrackWithFeatures
	for rows.Next() {
		var track TrackWithFeatures
		var features AudioFeatures
		
		err := rows.Scan(
			&track.ID, &track.FilePath, &track.FileName, &track.AnalysisStatus, &track.AnalysisError,
			&track.DownloadedAt, &track.AnalyzedAt,
			&features.BPM, &features.Key, &features.DurationSeconds,
			&features.LoudnessDB, &features.Energy, &features.Danceability, &features.Speechiness,
			&features.Acousticness, &features.Instrumentalness, &features.Liveness,
			&features.Valence, &features.Melodicness,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		
		if features.BPM != nil {
			track.Features = &features
		}
		
		tracks = append(tracks, track)
	}
	
	return tracks, nil
}

// GetFailedTracks returns tracks that failed analysis with error messages
func (db *DB) GetFailedTracks(ctx context.Context, limit int) ([]TrackWithFeatures, error) {
	query := `
		SELECT id, file_path, file_name, analysis_status, analysis_error,
		       downloaded_at, analyzed_at
		FROM tracks
		WHERE analysis_status = 'failed'
		ORDER BY analyzed_at DESC
		LIMIT $1
	`
	
	rows, err := db.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	var tracks []TrackWithFeatures
	for rows.Next() {
		var track TrackWithFeatures
		err := rows.Scan(
			&track.ID, &track.FilePath, &track.FileName, &track.AnalysisStatus, &track.AnalysisError,
			&track.DownloadedAt, &track.AnalyzedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		tracks = append(tracks, track)
	}
	
	return tracks, nil
}

// GetRecentCompletedTracks returns recently completed tracks for rate calculation
func (db *DB) GetRecentCompletedTracks(ctx context.Context, limit int) ([]TrackWithFeatures, error) {
	query := `
		SELECT id, file_path, file_name, analysis_status, analysis_error,
		       downloaded_at, analyzed_at
		FROM tracks
		WHERE analysis_status = 'completed' AND analyzed_at IS NOT NULL
		ORDER BY analyzed_at DESC
		LIMIT $1
	`
	
	rows, err := db.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	var tracks []TrackWithFeatures
	for rows.Next() {
		var track TrackWithFeatures
		err := rows.Scan(
			&track.ID, &track.FilePath, &track.FileName, &track.AnalysisStatus, &track.AnalysisError,
			&track.DownloadedAt, &track.AnalyzedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		tracks = append(tracks, track)
	}
	
	return tracks, nil
}
