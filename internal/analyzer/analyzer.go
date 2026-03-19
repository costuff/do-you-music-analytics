package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// Analyzer handles audio analysis using Python librosa
type Analyzer struct {
	pythonPath      string
	scriptPath      string
	analysisTimeout time.Duration
}

// AudioFeatures represents the output from Python analysis
type AudioFeatures struct {
	FilePath        string  `json:"file_path"`
	FileName        string  `json:"file_name"`
	DurationSeconds float64 `json:"duration_seconds"`

	// Core detected features
	BPM  *float64 `json:"bpm"`
	Key  *string  `json:"key"`
	Mode *string  `json:"mode"`

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

	// Raw spectral features
	SpectralCentroidMean *float64 `json:"spectral_centroid_mean"`
	SpectralCentroidStd  *float64 `json:"spectral_centroid_std"`
	SpectralRolloffMean  *float64 `json:"spectral_rolloff_mean"`
	ZeroCrossingRateMean *float64 `json:"zero_crossing_rate_mean"`
	RMSEnergyMean        *float64 `json:"rms_energy_mean"`
	RMSEnergyStd         *float64 `json:"rms_energy_std"`

	// MFCCs 0–12 (mean and standard deviation per coefficient)
	MFCC0Mean  *float64 `json:"mfcc_0_mean"`
	MFCC0Std   *float64 `json:"mfcc_0_std"`
	MFCC1Mean  *float64 `json:"mfcc_1_mean"`
	MFCC1Std   *float64 `json:"mfcc_1_std"`
	MFCC2Mean  *float64 `json:"mfcc_2_mean"`
	MFCC2Std   *float64 `json:"mfcc_2_std"`
	MFCC3Mean  *float64 `json:"mfcc_3_mean"`
	MFCC3Std   *float64 `json:"mfcc_3_std"`
	MFCC4Mean  *float64 `json:"mfcc_4_mean"`
	MFCC4Std   *float64 `json:"mfcc_4_std"`
	MFCC5Mean  *float64 `json:"mfcc_5_mean"`
	MFCC5Std   *float64 `json:"mfcc_5_std"`
	MFCC6Mean  *float64 `json:"mfcc_6_mean"`
	MFCC6Std   *float64 `json:"mfcc_6_std"`
	MFCC7Mean  *float64 `json:"mfcc_7_mean"`
	MFCC7Std   *float64 `json:"mfcc_7_std"`
	MFCC8Mean  *float64 `json:"mfcc_8_mean"`
	MFCC8Std   *float64 `json:"mfcc_8_std"`
	MFCC9Mean  *float64 `json:"mfcc_9_mean"`
	MFCC9Std   *float64 `json:"mfcc_9_std"`
	MFCC10Mean *float64 `json:"mfcc_10_mean"`
	MFCC10Std  *float64 `json:"mfcc_10_std"`
	MFCC11Mean *float64 `json:"mfcc_11_mean"`
	MFCC11Std  *float64 `json:"mfcc_11_std"`
	MFCC12Mean *float64 `json:"mfcc_12_mean"`
	MFCC12Std  *float64 `json:"mfcc_12_std"`

	Error *string `json:"error"`
}

// New creates a new Analyzer
func New(pythonPath, scriptPath string, timeout time.Duration) *Analyzer {
	return &Analyzer{
		pythonPath:      pythonPath,
		scriptPath:      scriptPath,
		analysisTimeout: timeout,
	}
}

// AnalyzeFile runs Python analysis on a single audio file
func (a *Analyzer) AnalyzeFile(ctx context.Context, filePath string) (*AudioFeatures, error) {
	ctx, cancel := context.WithTimeout(ctx, a.analysisTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, a.pythonPath, a.scriptPath, filePath, "--single", "--quiet")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("analysis timeout after %v", a.analysisTimeout)
		}
		return nil, fmt.Errorf("python execution failed: %w (output: %s)", err, string(output))
	}

	var features AudioFeatures
	if err := json.Unmarshal(output, &features); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w (output: %s)", err, string(output))
	}

	if features.Error != nil {
		return &features, fmt.Errorf("analysis error: %s", *features.Error)
	}

	return &features, nil
}

// Validate checks if the analyzer is properly configured
func (a *Analyzer) Validate() error {
	if _, err := exec.LookPath(a.pythonPath); err != nil {
		return fmt.Errorf("python not found: %w", err)
	}
	return nil
}
