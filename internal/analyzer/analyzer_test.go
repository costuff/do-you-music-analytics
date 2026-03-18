package analyzer

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	a := New("python3", "script.py", 30*time.Second)
	if a == nil {
		t.Error("New() returned nil")
	}
	if a.pythonPath != "python3" {
		t.Errorf("pythonPath = %q, want %q", a.pythonPath, "python3")
	}
	if a.scriptPath != "script.py" {
		t.Errorf("scriptPath = %q, want %q", a.scriptPath, "script.py")
	}
	if a.analysisTimeout != 30*time.Second {
		t.Errorf("analysisTimeout = %v, want %v", a.analysisTimeout, 30*time.Second)
	}
}

// writePythonScript creates a temporary Python script file with the given content.
func writePythonScript(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test_*.py")
	if err != nil {
		t.Fatalf("failed to create temp script: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	f.Close()
	return f.Name()
}

// findPython3 returns the path to python3 or skips the test if unavailable.
func findPython3(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not in PATH")
	}
	return p
}

func TestAnalyzeFile_Success(t *testing.T) {
	python := findPython3(t)
	script := writePythonScript(t, `
import sys, json
print(json.dumps({
    "file_path": sys.argv[1],
    "file_name": "test.mp3",
    "duration_seconds": 180.0,
    "bpm": 128.5,
    "key": "Am",
    "energy": 0.8,
    "danceability": 0.7,
    "valence": 0.5,
    "acousticness": 0.1,
    "instrumentalness": 0.9,
    "speechiness": 0.05,
    "liveness": 0.1,
    "melodicness": 0.6,
    "rms_energy_mean": 0.3,
    "loudness_db": -8.5
}))
`)

	a := New(python, script, 10*time.Second)
	features, err := a.AnalyzeFile(context.Background(), "/some/test.mp3")
	if err != nil {
		t.Fatalf("AnalyzeFile() unexpected error: %v", err)
	}
	if features == nil {
		t.Fatal("AnalyzeFile() returned nil features")
	}
	if features.BPM == nil || *features.BPM != 128.5 {
		t.Errorf("BPM = %v, want 128.5", features.BPM)
	}
	if features.Key == nil || *features.Key != "Am" {
		t.Errorf("Key = %v, want Am", features.Key)
	}
	if features.DurationSeconds != 180.0 {
		t.Errorf("DurationSeconds = %v, want 180.0", features.DurationSeconds)
	}
	if features.LoudnessDB == nil || *features.LoudnessDB != -8.5 {
		t.Errorf("LoudnessDB = %v, want -8.5", features.LoudnessDB)
	}
}

func TestAnalyzeFile_PythonNotFound(t *testing.T) {
	a := New("/nonexistent/python/binary", "script.py", 5*time.Second)
	_, err := a.AnalyzeFile(context.Background(), "/some/test.mp3")
	if err == nil {
		t.Error("AnalyzeFile() expected error for missing python binary, got nil")
	}
}

func TestAnalyzeFile_BadJSON(t *testing.T) {
	python := findPython3(t)
	script := writePythonScript(t, `print("this is not valid json at all")`)

	a := New(python, script, 10*time.Second)
	_, err := a.AnalyzeFile(context.Background(), "/some/test.mp3")
	if err == nil {
		t.Error("AnalyzeFile() expected error for bad JSON, got nil")
	}
	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "json") && !strings.Contains(errMsg, "parse") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

func TestAnalyzeFile_AnalysisError(t *testing.T) {
	python := findPython3(t)
	script := writePythonScript(t, `
import json
print(json.dumps({
    "file_path": "",
    "file_name": "",
    "duration_seconds": 0,
    "error": "corrupted file: cannot decode audio"
}))
`)

	a := New(python, script, 10*time.Second)
	features, err := a.AnalyzeFile(context.Background(), "/corrupted.mp3")
	if err == nil {
		t.Error("AnalyzeFile() expected error when JSON contains error field, got nil")
	}
	if !strings.Contains(err.Error(), "analysis error") {
		t.Errorf("expected 'analysis error' in message, got: %v", err)
	}
	// features should still be returned alongside the error
	if features == nil {
		t.Error("expected features to be non-nil even when error field is set")
	}
}

func TestAnalyzeFile_Timeout(t *testing.T) {
	python := findPython3(t)
	script := writePythonScript(t, `
import time
time.sleep(10)
print("{}")
`)

	a := New(python, script, 150*time.Millisecond)
	_, err := a.AnalyzeFile(context.Background(), "/some/test.mp3")
	if err == nil {
		t.Error("AnalyzeFile() expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected 'timeout' in error message, got: %v", err)
	}
}

func TestAnalyzeFile_ScriptNotFound(t *testing.T) {
	python := findPython3(t)
	a := New(python, "/nonexistent/path/script.py", 5*time.Second)
	_, err := a.AnalyzeFile(context.Background(), "/some/test.mp3")
	if err == nil {
		t.Error("AnalyzeFile() expected error for missing script, got nil")
	}
}

func TestAnalyzeFile_ContextCancelled(t *testing.T) {
	python := findPython3(t)
	script := writePythonScript(t, `
import time
time.sleep(10)
print("{}")
`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	a := New(python, script, 30*time.Second)
	_, err := a.AnalyzeFile(ctx, "/some/test.mp3")
	if err == nil {
		t.Error("AnalyzeFile() expected error for cancelled context, got nil")
	}
}

func TestValidate_Valid(t *testing.T) {
	python := findPython3(t)
	a := New(python, "script.py", 10*time.Second)
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() unexpected error for valid python path: %v", err)
	}
}

func TestValidate_NotFound(t *testing.T) {
	a := New("/totally/nonexistent/python", "script.py", 10*time.Second)
	err := a.Validate()
	if err == nil {
		t.Error("Validate() expected error for missing python, got nil")
	}
	if !strings.Contains(err.Error(), "python not found") {
		t.Errorf("expected 'python not found' in error, got: %v", err)
	}
}
