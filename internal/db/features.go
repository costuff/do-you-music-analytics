package db

import (
	"context"
	"fmt"
	"time"
)

// AudioFeatures represents audio analysis results
type AudioFeatures struct {
	TrackID         int      `json:"track_id"`
	BPM             *float64 `json:"bpm"`
	Key             *string  `json:"key"`
	Mode            *string  `json:"mode"`
	DurationSeconds *float64 `json:"duration_seconds"`

	// Spotify-like high-level features (0–1)
	Energy           *float64 `json:"energy"`
	Danceability     *float64 `json:"danceability"`
	Valence          *float64 `json:"valence"`
	Acousticness     *float64 `json:"acousticness"`
	Instrumentalness *float64 `json:"instrumentalness"`
	Speechiness      *float64 `json:"speechiness"`
	Liveness         *float64 `json:"liveness"`
	Melodicness      *float64 `json:"melodicness"`
	LoudnessDB       *float64 `json:"loudness_db"`
	AnalysisVersion  string   `json:"analysis_version,omitempty"`

	// Raw spectral features (omitted from JSON when nil to keep API responses lean)
	SpectralCentroidMean *float64 `json:"spectral_centroid_mean,omitempty"`
	SpectralCentroidStd  *float64 `json:"spectral_centroid_std,omitempty"`
	SpectralRolloffMean  *float64 `json:"spectral_rolloff_mean,omitempty"`
	ZeroCrossingRateMean *float64 `json:"zero_crossing_rate_mean,omitempty"`
	RMSEnergyMean        *float64 `json:"rms_energy_mean,omitempty"`
	RMSEnergyStd         *float64 `json:"rms_energy_std,omitempty"`

	// MFCCs 0–12 (mean and standard deviation per coefficient)
	MFCC0Mean  *float64 `json:"mfcc_0_mean,omitempty"`
	MFCC0Std   *float64 `json:"mfcc_0_std,omitempty"`
	MFCC1Mean  *float64 `json:"mfcc_1_mean,omitempty"`
	MFCC1Std   *float64 `json:"mfcc_1_std,omitempty"`
	MFCC2Mean  *float64 `json:"mfcc_2_mean,omitempty"`
	MFCC2Std   *float64 `json:"mfcc_2_std,omitempty"`
	MFCC3Mean  *float64 `json:"mfcc_3_mean,omitempty"`
	MFCC3Std   *float64 `json:"mfcc_3_std,omitempty"`
	MFCC4Mean  *float64 `json:"mfcc_4_mean,omitempty"`
	MFCC4Std   *float64 `json:"mfcc_4_std,omitempty"`
	MFCC5Mean  *float64 `json:"mfcc_5_mean,omitempty"`
	MFCC5Std   *float64 `json:"mfcc_5_std,omitempty"`
	MFCC6Mean  *float64 `json:"mfcc_6_mean,omitempty"`
	MFCC6Std   *float64 `json:"mfcc_6_std,omitempty"`
	MFCC7Mean  *float64 `json:"mfcc_7_mean,omitempty"`
	MFCC7Std   *float64 `json:"mfcc_7_std,omitempty"`
	MFCC8Mean  *float64 `json:"mfcc_8_mean,omitempty"`
	MFCC8Std   *float64 `json:"mfcc_8_std,omitempty"`
	MFCC9Mean  *float64 `json:"mfcc_9_mean,omitempty"`
	MFCC9Std   *float64 `json:"mfcc_9_std,omitempty"`
	MFCC10Mean *float64 `json:"mfcc_10_mean,omitempty"`
	MFCC10Std  *float64 `json:"mfcc_10_std,omitempty"`
	MFCC11Mean *float64 `json:"mfcc_11_mean,omitempty"`
	MFCC11Std  *float64 `json:"mfcc_11_std,omitempty"`
	MFCC12Mean *float64 `json:"mfcc_12_mean,omitempty"`
	MFCC12Std  *float64 `json:"mfcc_12_std,omitempty"`
}

// SaveAudioFeatures inserts or updates audio features for a track
func (db *DB) SaveAudioFeatures(ctx context.Context, features *AudioFeatures) error {
	query := `
		INSERT INTO audio_features (
			track_id, bpm, key, mode, duration_seconds,
			energy, danceability, valence, acousticness,
			instrumentalness, speechiness, liveness, melodicness,
			loudness_db, analysis_version,
			spectral_centroid_mean, spectral_centroid_std, spectral_rolloff_mean,
			zero_crossing_rate_mean, rms_energy_mean, rms_energy_std,
			mfcc_0_mean,  mfcc_0_std,
			mfcc_1_mean,  mfcc_1_std,
			mfcc_2_mean,  mfcc_2_std,
			mfcc_3_mean,  mfcc_3_std,
			mfcc_4_mean,  mfcc_4_std,
			mfcc_5_mean,  mfcc_5_std,
			mfcc_6_mean,  mfcc_6_std,
			mfcc_7_mean,  mfcc_7_std,
			mfcc_8_mean,  mfcc_8_std,
			mfcc_9_mean,  mfcc_9_std,
			mfcc_10_mean, mfcc_10_std,
			mfcc_11_mean, mfcc_11_std,
			mfcc_12_mean, mfcc_12_std
		) VALUES (
			$1,  $2,  $3,  $4,  $5,
			$6,  $7,  $8,  $9,
			$10, $11, $12, $13,
			$14, $15,
			$16, $17, $18,
			$19, $20, $21,
			$22, $23,
			$24, $25,
			$26, $27,
			$28, $29,
			$30, $31,
			$32, $33,
			$34, $35,
			$36, $37,
			$38, $39,
			$40, $41,
			$42, $43,
			$44, $45,
			$46, $47
		)
		ON CONFLICT (track_id) DO UPDATE SET
			bpm                   = EXCLUDED.bpm,
			key                   = EXCLUDED.key,
			mode                  = EXCLUDED.mode,
			duration_seconds      = EXCLUDED.duration_seconds,
			energy                = EXCLUDED.energy,
			danceability          = EXCLUDED.danceability,
			valence               = EXCLUDED.valence,
			acousticness          = EXCLUDED.acousticness,
			instrumentalness      = EXCLUDED.instrumentalness,
			speechiness           = EXCLUDED.speechiness,
			liveness              = EXCLUDED.liveness,
			melodicness           = EXCLUDED.melodicness,
			loudness_db           = EXCLUDED.loudness_db,
			analysis_version      = EXCLUDED.analysis_version,
			spectral_centroid_mean  = EXCLUDED.spectral_centroid_mean,
			spectral_centroid_std   = EXCLUDED.spectral_centroid_std,
			spectral_rolloff_mean   = EXCLUDED.spectral_rolloff_mean,
			zero_crossing_rate_mean = EXCLUDED.zero_crossing_rate_mean,
			rms_energy_mean         = EXCLUDED.rms_energy_mean,
			rms_energy_std          = EXCLUDED.rms_energy_std,
			mfcc_0_mean  = EXCLUDED.mfcc_0_mean,  mfcc_0_std  = EXCLUDED.mfcc_0_std,
			mfcc_1_mean  = EXCLUDED.mfcc_1_mean,  mfcc_1_std  = EXCLUDED.mfcc_1_std,
			mfcc_2_mean  = EXCLUDED.mfcc_2_mean,  mfcc_2_std  = EXCLUDED.mfcc_2_std,
			mfcc_3_mean  = EXCLUDED.mfcc_3_mean,  mfcc_3_std  = EXCLUDED.mfcc_3_std,
			mfcc_4_mean  = EXCLUDED.mfcc_4_mean,  mfcc_4_std  = EXCLUDED.mfcc_4_std,
			mfcc_5_mean  = EXCLUDED.mfcc_5_mean,  mfcc_5_std  = EXCLUDED.mfcc_5_std,
			mfcc_6_mean  = EXCLUDED.mfcc_6_mean,  mfcc_6_std  = EXCLUDED.mfcc_6_std,
			mfcc_7_mean  = EXCLUDED.mfcc_7_mean,  mfcc_7_std  = EXCLUDED.mfcc_7_std,
			mfcc_8_mean  = EXCLUDED.mfcc_8_mean,  mfcc_8_std  = EXCLUDED.mfcc_8_std,
			mfcc_9_mean  = EXCLUDED.mfcc_9_mean,  mfcc_9_std  = EXCLUDED.mfcc_9_std,
			mfcc_10_mean = EXCLUDED.mfcc_10_mean, mfcc_10_std = EXCLUDED.mfcc_10_std,
			mfcc_11_mean = EXCLUDED.mfcc_11_mean, mfcc_11_std = EXCLUDED.mfcc_11_std,
			mfcc_12_mean = EXCLUDED.mfcc_12_mean, mfcc_12_std = EXCLUDED.mfcc_12_std,
			analyzed_at  = NOW()`

	_, err := db.pool.Exec(
		ctx, query,
		features.TrackID, features.BPM, features.Key, features.Mode, features.DurationSeconds,
		features.Energy, features.Danceability, features.Valence, features.Acousticness,
		features.Instrumentalness, features.Speechiness, features.Liveness,
		features.Melodicness, features.LoudnessDB, features.AnalysisVersion,
		features.SpectralCentroidMean, features.SpectralCentroidStd, features.SpectralRolloffMean,
		features.ZeroCrossingRateMean, features.RMSEnergyMean, features.RMSEnergyStd,
		features.MFCC0Mean, features.MFCC0Std,
		features.MFCC1Mean, features.MFCC1Std,
		features.MFCC2Mean, features.MFCC2Std,
		features.MFCC3Mean, features.MFCC3Std,
		features.MFCC4Mean, features.MFCC4Std,
		features.MFCC5Mean, features.MFCC5Std,
		features.MFCC6Mean, features.MFCC6Std,
		features.MFCC7Mean, features.MFCC7Std,
		features.MFCC8Mean, features.MFCC8Std,
		features.MFCC9Mean, features.MFCC9Std,
		features.MFCC10Mean, features.MFCC10Std,
		features.MFCC11Mean, features.MFCC11Std,
		features.MFCC12Mean, features.MFCC12Std,
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
		SET analysis_status = $1::text,
		    analysis_error = $2,
		    analyzed_at = COALESCE($3, analyzed_at),
		    analysis_duration_seconds = CASE
		        WHEN $1::text = 'analyzing' THEN NULL
		        WHEN $1::text = 'completed' AND analysis_started_at IS NOT NULL THEN EXTRACT(EPOCH FROM NOW() - analysis_started_at)
		        ELSE analysis_duration_seconds
		    END,
		    analysis_started_at = CASE WHEN $1::text = 'analyzing' THEN analysis_started_at ELSE NULL END,
		    next_retry_at = CASE WHEN $1::text = 'pending' THEN next_retry_at ELSE NULL END
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
