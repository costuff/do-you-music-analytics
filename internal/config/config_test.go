package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_WithDefaults(t *testing.T) {
	clearTestEnv(t)
	setupTestFiles(t)
	
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	
	if cfg.DBMaxConns != 20 {
		t.Errorf("DBMaxConns = %v, want 20", cfg.DBMaxConns)
	}
	if cfg.APIPort != 4000 {
		t.Errorf("APIPort = %v, want 4000", cfg.APIPort)
	}
	if cfg.BatchSize != 10 {
		t.Errorf("BatchSize = %v, want 10", cfg.BatchSize)
	}
}

func TestLoad_WithCustomEnv(t *testing.T) {
	clearTestEnv(t)
	tempDir := setupTestFiles(t)
	
	os.Setenv("DATABASE_URL", "postgresql://custom:pass@localhost/custom")
	os.Setenv("API_PORT", "8080")
	os.Setenv("BATCH_SIZE", "50")
	os.Setenv("MAX_WORKERS", "8")
	os.Setenv("NAS_MOUNT_PATH", tempDir)
	
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	
	if cfg.APIPort != 8080 {
		t.Errorf("APIPort = %v, want 8080", cfg.APIPort)
	}
	if cfg.BatchSize != 50 {
		t.Errorf("BatchSize = %v, want 50", cfg.BatchSize)
	}
	if cfg.MaxWorkers != 8 {
		t.Errorf("MaxWorkers = %v, want 8", cfg.MaxWorkers)
	}
}

func TestValidate_EmptyDatabaseURL(t *testing.T) {
	cfg := &Config{
		DatabaseURL: "",
	}
	
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error for empty DatabaseURL")
	}
}

func TestValidate_InvalidAPIPort(t *testing.T) {
	tests := []int{0, -1, 65536, 100000}
	
	for _, port := range tests {
		cfg := &Config{
			DatabaseURL: "postgresql://test",
			APIPort:     port,
		}
		
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() expected error for APIPort=%d", port)
		}
	}
}

func TestValidate_InvalidBatchSize(t *testing.T) {
	tests := []int{0, -5, 101, 999}
	
	for _, size := range tests {
		cfg := &Config{
			DatabaseURL: "postgresql://test",
			APIPort:     8080,
			BatchSize:   size,
		}
		
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() expected error for BatchSize=%d", size)
		}
	}
}

func TestValidate_InvalidMaxWorkers(t *testing.T) {
	tests := []int{0, -1, 33, 100}
	
	for _, workers := range tests {
		cfg := &Config{
			DatabaseURL: "postgresql://test",
			APIPort:     8080,
			BatchSize:   10,
			MaxWorkers:  workers,
		}
		
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() expected error for MaxWorkers=%d", workers)
		}
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		DatabaseURL: "postgresql://test",
		APIPort:     8080,
		BatchSize:   10,
		MaxWorkers:  4,
		LogLevel:    "invalid",
	}
	
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() expected error for invalid LogLevel")
	}
}

func TestValidate_ValidLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	tempDir := setupTestFiles(t)
	
	for _, level := range levels {
		cfg := &Config{
			DatabaseURL:          "postgresql://test",
			APIPort:              8080,
			BatchSize:            10,
			MaxWorkers:           4,
			LogLevel:             level,
			ValidateNASOnStartup: false,
			PythonPath:           filepath.Join(tempDir, "python3"),
			AnalysisScript:       filepath.Join(tempDir, "analyze.py"),
		}
		
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v for LogLevel=%s", err, level)
		}
	}
}

func TestValidate_NASPath(t *testing.T) {
	tempDir := setupTestFiles(t)
	
	t.Run("succeeds with existing path", func(t *testing.T) {
		cfg := &Config{
			DatabaseURL:          "postgresql://test",
			APIPort:              8080,
			BatchSize:            10,
			MaxWorkers:           4,
			LogLevel:             "info",
			NASMountPath:         tempDir,
			ValidateNASOnStartup: true,
			PythonPath:           filepath.Join(tempDir, "python3"),
			AnalysisScript:       filepath.Join(tempDir, "analyze.py"),
		}
		
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v for existing NAS path", err)
		}
	})
	
	t.Run("fails with non-existent path when validation enabled", func(t *testing.T) {
		cfg := &Config{
			DatabaseURL:          "postgresql://test",
			APIPort:              8080,
			BatchSize:            10,
			MaxWorkers:           4,
			LogLevel:             "info",
			NASMountPath:         "/non/existent/path",
			ValidateNASOnStartup: true,
			PythonPath:           filepath.Join(tempDir, "python3"),
			AnalysisScript:       filepath.Join(tempDir, "analyze.py"),
		}
		
		if err := cfg.Validate(); err == nil {
			t.Error("Validate() expected error for non-existent NAS path")
		}
	})
	
	t.Run("skips validation when disabled", func(t *testing.T) {
		cfg := &Config{
			DatabaseURL:          "postgresql://test",
			APIPort:              8080,
			BatchSize:            10,
			MaxWorkers:           4,
			LogLevel:             "info",
			NASMountPath:         "/non/existent/path",
			ValidateNASOnStartup: false,
			PythonPath:           filepath.Join(tempDir, "python3"),
			AnalysisScript:       filepath.Join(tempDir, "analyze.py"),
		}
		
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v when NAS validation disabled", err)
		}
	})
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{
			url:  "postgresql://user:secretpass@localhost:5432/dbname",
			want: "postgresql://user:se***",
		},
		{
			url:  "short",
			want: "***",
		},
	}
	
	for _, tt := range tests {
		cfg := &Config{DatabaseURL: tt.url}
		got := cfg.MaskPassword()
		if got != tt.want {
			t.Errorf("MaskPassword() = %v, want %v", got, tt.want)
		}
	}
}

func TestGetEnvHelpers(t *testing.T) {
	clearTestEnv(t)
	
	t.Run("getEnv", func(t *testing.T) {
		os.Setenv("TEST_VAR", "value")
		if got := getEnv("TEST_VAR", "default"); got != "value" {
			t.Errorf("getEnv() = %v, want value", got)
		}
		
		os.Unsetenv("TEST_VAR")
		if got := getEnv("TEST_VAR", "default"); got != "default" {
			t.Errorf("getEnv() = %v, want default", got)
		}
	})
	
	t.Run("getEnvInt", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		if got := getEnvInt("TEST_INT", 10); got != 42 {
			t.Errorf("getEnvInt() = %v, want 42", got)
		}
		
		os.Setenv("TEST_INT", "invalid")
		if got := getEnvInt("TEST_INT", 10); got != 10 {
			t.Errorf("getEnvInt() = %v, want default 10", got)
		}
		
		os.Unsetenv("TEST_INT")
		if got := getEnvInt("TEST_INT", 10); got != 10 {
			t.Errorf("getEnvInt() = %v, want default 10", got)
		}
	})
	
	t.Run("getEnvBool", func(t *testing.T) {
		os.Setenv("TEST_BOOL", "true")
		if got := getEnvBool("TEST_BOOL", false); !got {
			t.Error("getEnvBool() = false, want true")
		}
		
		os.Setenv("TEST_BOOL", "false")
		if got := getEnvBool("TEST_BOOL", true); got {
			t.Error("getEnvBool() = true, want false")
		}
		
		os.Setenv("TEST_BOOL", "invalid")
		if got := getEnvBool("TEST_BOOL", true); !got {
			t.Error("getEnvBool() = false, want default true")
		}
		
		os.Unsetenv("TEST_BOOL")
		if got := getEnvBool("TEST_BOOL", true); !got {
			t.Error("getEnvBool() = false, want default true")
		}
	})
}

// Helper functions

func setupTestFiles(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	
	pythonPath := filepath.Join(tempDir, "python3")
	scriptPath := filepath.Join(tempDir, "analyze.py")
	
	if err := os.WriteFile(pythonPath, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	
	os.Setenv("PYTHON_PATH", pythonPath)
	os.Setenv("ANALYSIS_SCRIPT", scriptPath)
	os.Setenv("NAS_MOUNT_PATH", tempDir)
	
	return tempDir
}

func clearTestEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"DATABASE_URL", "DB_MAX_CONNS", "DB_TIMEOUT",
		"NAS_MOUNT_PATH", "VALIDATE_NAS_ON_STARTUP",
		"BATCH_SIZE", "MAX_WORKERS", "PYTHON_PATH", "ANALYSIS_SCRIPT", "ANALYSIS_TIMEOUT",
		"API_PORT", "LOG_LEVEL", "CORS_ORIGINS",
		"ENABLE_METRICS", "METRICS_PORT",
		"TEST_VAR", "TEST_INT", "TEST_BOOL",
	}
	for _, key := range envVars {
		os.Unsetenv(key)
	}
}
