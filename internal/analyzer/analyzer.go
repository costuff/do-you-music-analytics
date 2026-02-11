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
	FilePath         string   `json:"file_path"`
	FileName         string   `json:"file_name"`
	DurationSeconds  float64  `json:"duration_seconds"`
	BPM              *float64 `json:"bpm"`
	Key              *string  `json:"key"`
	Energy           *float64 `json:"energy"`
	Danceability     *float64 `json:"danceability"`
	Valence          *float64 `json:"valence"`
	Acousticness     *float64 `json:"acousticness"`
	Instrumentalness *float64 `json:"instrumentalness"`
	Speechiness      *float64 `json:"speechiness"`
	Liveness         *float64 `json:"liveness"`
	Melodicness      *float64 `json:"melodicness"`
	RMSEnergyMean    *float64 `json:"rms_energy_mean"`
	LoudnessDB       *float64 `json:"loudness_db"`
	Error            *string  `json:"error"`
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
