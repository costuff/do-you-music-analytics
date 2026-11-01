package db

import (
	"encoding/json"
	"testing"
)

func TestAudioFeatures_JSONSerialization(t *testing.T) {
	// Test that AudioFeatures correctly serialize/deserialize
	original := &AudioFeatures{
		TrackID:          123,
		BPM:              floatPtr(128.5),
		Key:              stringPtr("Am"),
		DurationSeconds:  floatPtr(301.5),
		Energy:           floatPtr(0.89),
		Danceability:     floatPtr(0.75),
		Valence:          floatPtr(0.45),
		Acousticness:     floatPtr(0.02),
		Instrumentalness: floatPtr(0.95),
		Speechiness:      floatPtr(0.03),
		Liveness:         floatPtr(0.12),
		Melodicness:      floatPtr(0.67),
		LoudnessDB:       floatPtr(-6.5),
		AnalysisVersion:  "1.0",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify JSON structure
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	// Check required fields exist
	requiredFields := []string{
		"track_id", "bpm", "key", "energy", "danceability",
		"valence", "acousticness", "instrumentalness",
		"speechiness", "liveness",
	}

	for _, field := range requiredFields {
		if _, ok := jsonMap[field]; !ok {
			t.Errorf("Missing required field in JSON: %s", field)
		}
	}

	// Unmarshal back to struct
	var decoded AudioFeatures
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal to struct failed: %v", err)
	}

	// Verify values match
	if decoded.BPM == nil || original.BPM == nil || *decoded.BPM != *original.BPM {
		t.Errorf("BPM mismatch: got %v, want %v",
			ptrToValue(decoded.BPM), ptrToValue(original.BPM))
	}
	if decoded.Key == nil || original.Key == nil || *decoded.Key != *original.Key {
		t.Errorf("Key mismatch: got %v, want %v",
			ptrToString(decoded.Key), ptrToString(original.Key))
	}
	if decoded.Energy == nil || original.Energy == nil || *decoded.Energy != *original.Energy {
		t.Errorf("Energy mismatch: got %v, want %v",
			ptrToValue(decoded.Energy), ptrToValue(original.Energy))
	}
}

func TestAudioFeatures_NullableFields(t *testing.T) {
	tests := []struct {
		name     string
		features *AudioFeatures
		checkFn  func(*testing.T, []byte)
	}{
		{
			name: "all fields populated",
			features: &AudioFeatures{
				TrackID:         1,
				BPM:             floatPtr(120.0),
				Key:             stringPtr("C"),
				DurationSeconds: floatPtr(180.0),
				Melodicness:     floatPtr(0.5),
				LoudnessDB:      floatPtr(-8.0),
			},
			checkFn: func(t *testing.T, data []byte) {
				if !contains(data, "duration_seconds") {
					t.Error("Expected duration_seconds in JSON")
				}
				if !contains(data, "melodicness") {
					t.Error("Expected melodicness in JSON")
				}
				if !contains(data, "loudness_db") {
					t.Error("Expected loudness_db in JSON")
				}
			},
		},
		{
			name: "nullable fields null",
			features: &AudioFeatures{
				TrackID:         2,
				BPM:             floatPtr(140.0),
				Key:             stringPtr("F#"),
				DurationSeconds: nil,
				Melodicness:     nil,
				LoudnessDB:      nil,
			},
			checkFn: func(t *testing.T, data []byte) {
				var jsonMap map[string]interface{}
				if err := json.Unmarshal(data, &jsonMap); err != nil {
					t.Fatal(err)
				}
				// These fields should be present as null
				if _, ok := jsonMap["duration_seconds"]; !ok {
					t.Error("duration_seconds should be present (as null)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.features)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			tt.checkFn(t, data)
		})
	}
}

func TestTrackWithFeatures_JSONResponse(t *testing.T) {
	// Test complete API response structure
	track := &TrackWithFeatures{
		ID:             123,
		FilePath:       "/mnt/music/downloads/track.mp3",
		FileName:       "track.mp3",
		AnalysisStatus: "completed",
		Features: &AudioFeatures{
			TrackID:          123,
			BPM:              floatPtr(161.5),
			Key:              stringPtr("D#"),
			DurationSeconds:  floatPtr(301.5),
			Energy:           floatPtr(0.3989),
			Danceability:     floatPtr(0.6012),
			Valence:          floatPtr(0.1546),
			Acousticness:     floatPtr(0.0452),
			Instrumentalness: floatPtr(0.987),
			Speechiness:      floatPtr(0.0234),
			Liveness:         floatPtr(0.1123),
			Melodicness:      floatPtr(0.456),
			LoudnessDB:       floatPtr(-8.5),
			AnalysisVersion:  "1.0",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(track)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Parse as map to verify structure
	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify top-level fields
	if response["id"] != float64(123) {
		t.Errorf("id = %v, want 123", response["id"])
	}

	if response["file_name"] != "track.mp3" {
		t.Errorf("file_name = %v, want track.mp3", response["file_name"])
	}

	if response["analysis_status"] != "completed" {
		t.Errorf("analysis_status = %v, want completed", response["analysis_status"])
	}

	// Verify nested features
	features, ok := response["features"].(map[string]interface{})
	if !ok {
		t.Fatal("features is not an object")
	}

	if features["bpm"] != 161.5 {
		t.Errorf("BPM = %v, want 161.5", features["bpm"])
	}

	if features["key"] != "D#" {
		t.Errorf("Key = %v, want D#", features["key"])
	}

	// Verify all 13 audio features are present
	expectedFeatures := []string{
		"bpm", "key", "duration_seconds", "energy",
		"danceability", "valence", "acousticness",
		"instrumentalness", "speechiness", "liveness",
		"melodicness", "loudness_db",
	}

	for _, field := range expectedFeatures {
		if _, ok := features[field]; !ok {
			t.Errorf("Missing audio feature in JSON: %s", field)
		}
	}
}

func TestJSON_NumberPrecision(t *testing.T) {
	// Verify float precision is maintained
	features := &AudioFeatures{
		TrackID:      1,
		BPM:          floatPtr(128.567891234), // High precision
		Energy:       floatPtr(0.123456789),
		Danceability: floatPtr(0.987654321),
	}

	data, err := json.Marshal(features)
	if err != nil {
		t.Fatal(err)
	}

	var decoded AudioFeatures
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	// BPM should preserve reasonable precision
	if decoded.BPM == nil || features.BPM == nil {
		t.Fatal("BPM should not be nil")
	}
	if diff := abs(*decoded.BPM - *features.BPM); diff > 0.0001 {
		t.Errorf("BPM precision loss: got %v, want %v (diff: %v)",
			*decoded.BPM, *features.BPM, diff)
	}

	// Feature values should preserve precision
	if decoded.Energy == nil || features.Energy == nil {
		t.Fatal("Energy should not be nil")
	}
	if diff := abs(*decoded.Energy - *features.Energy); diff > 0.000001 {
		t.Errorf("Energy precision loss: got %v, want %v",
			*decoded.Energy, *features.Energy)
	}
}

func TestJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		checkFn func(*testing.T, *AudioFeatures)
	}{
		{
			name:    "BPM as integer",
			input:   `{"track_id":1,"bpm":120,"key":"C"}`,
			wantErr: false,
			checkFn: func(t *testing.T, af *AudioFeatures) {
				if af.BPM == nil || *af.BPM != 120.0 {
					t.Errorf("BPM = %v, want 120.0", af.BPM)
				}
			},
		},
		{
			name:    "very high BPM",
			input:   `{"track_id":1,"bpm":200.5,"key":"D"}`,
			wantErr: false,
			checkFn: func(t *testing.T, af *AudioFeatures) {
				if af.BPM == nil || *af.BPM != 200.5 {
					t.Errorf("BPM = %v, want 200.5", af.BPM)
				}
			},
		},
		{
			name:    "very low BPM",
			input:   `{"track_id":1,"bpm":60.0,"key":"E"}`,
			wantErr: false,
			checkFn: func(t *testing.T, af *AudioFeatures) {
				if af.BPM == nil || *af.BPM != 60.0 {
					t.Errorf("BPM = %v, want 60.0", af.BPM)
				}
			},
		},
		{
			name:    "feature values at boundaries",
			input:   `{"track_id":1,"bpm":120,"key":"C","energy":1.0,"danceability":0.0}`,
			wantErr: false,
			checkFn: func(t *testing.T, af *AudioFeatures) {
				if af.Energy == nil || *af.Energy != 1.0 {
					t.Errorf("Energy = %v, want 1.0", af.Energy)
				}
				if af.Danceability == nil || *af.Danceability != 0.0 {
					t.Errorf("Danceability = %v, want 0.0", af.Danceability)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var features AudioFeatures
			err := json.Unmarshal([]byte(tt.input), &features)

			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFn != nil {
				tt.checkFn(t, &features)
			}
		})
	}
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}

func contains(data []byte, substr string) bool {
	return len(data) > 0 && len(substr) > 0 && string(data) != "" && jsonContains(string(data), substr)
}

func jsonContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func ptrToValue(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
