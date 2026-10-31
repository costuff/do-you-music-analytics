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

func TestValidate_InvalidRetrySettings(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "attempt count", cfg: Config{DatabaseURL: "postgresql://test", MaxAnalysisAttempts: 21}},
		{name: "retry delay", cfg: Config{DatabaseURL: "postgresql://test", RetryBaseDelay: "soon"}},
		{name: "lease timeout", cfg: Config{DatabaseURL: "postgresql://test", AnalysisLeaseTimeout: "0s"}},
		{name: "lease below analyzer timeout", cfg: Config{DatabaseURL: "postgresql://test", AnalysisTimeout: "5m", AnalysisLeaseTimeout: "5m"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); err == nil {
				t.Fatal("Validate() expected error for invalid retry configuration")
			}
		})
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
		"SCAN_INTERVAL",
	}
	for _, key := range envVars {
		os.Unsetenv(key)
	}
}

func TestScanInterval(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
		wantErr  bool
	}{
		{
			name:     "default value",
			envValue: "",
			expected: "30m",
			wantErr:  false,
		},
		{
			name:     "custom minutes",
			envValue: "5m",
			expected: "5m",
			wantErr:  false,
		},
		{
			name:     "custom hours",
			envValue: "2h",
			expected: "2h",
			wantErr:  false,
		},
		{
			name:     "mixed duration",
			envValue: "1h30m",
			expected: "1h30m",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTestEnv(t)
			setupTestFiles(t)

			if tt.envValue != "" {
				os.Setenv("SCAN_INTERVAL", tt.envValue)
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg.ScanInterval != tt.expected {
				t.Errorf("ScanInterval = %v, want %v", cfg.ScanInterval, tt.expected)
			}
		})
	}
}

func TestPathHandling_LocalVsDocker(t *testing.T) {
	tests := []struct {
		name        string
		nasPath     string
		description string
	}{
		{
			name:        "local macOS path",
			nasPath:     "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks",
			description: "Local development path on Mac mini",
		},
		{
			name:        "docker mount path",
			nasPath:     "/mnt/music/downloads",
			description: "Production Docker container mount",
		},
		{
			name:        "linux absolute path",
			nasPath:     "/media/nas/music",
			description: "Linux NAS mount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTestEnv(t)

			// Create temp dir to represent NAS path
			tempDir := t.TempDir()

			// Create dummy Python script and executable for validation
			pythonPath := filepath.Join(tempDir, "python3")
			scriptPath := filepath.Join(tempDir, "analyze_audio.py")

			// Create dummy files
			if err := os.WriteFile(pythonPath, []byte("#!/bin/sh\n"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(scriptPath, []byte("# dummy\n"), 0644); err != nil {
				t.Fatal(err)
			}

			os.Setenv("NAS_MOUNT_PATH", tempDir)
			os.Setenv("VALIDATE_NAS_ON_STARTUP", "true")
			os.Setenv("PYTHON_PATH", pythonPath)
			os.Setenv("ANALYSIS_SCRIPT", scriptPath)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			// Verify path handling works
			if cfg.NASMountPath != tempDir {
				t.Errorf("NASMountPath = %v, want %v", cfg.NASMountPath, tempDir)
			}

			// Verify validation can be disabled for Docker
			os.Setenv("VALIDATE_NAS_ON_STARTUP", "false")
			os.Setenv("NAS_MOUNT_PATH", tt.nasPath) // Use non-existent path

			cfg2, err := Load()
			if err != nil {
				t.Fatalf("Load() with disabled validation failed: %v", err)
			}

			// Should not validate when disabled
			if err := cfg2.Validate(); err != nil {
				t.Errorf("Validate() failed when validation disabled: %v", err)
			}
		})
	}
}

func TestConfig_DockerEnvironment(t *testing.T) {
	clearTestEnv(t)
	tempDir := t.TempDir()

	// Create required files for validation
	pythonPath := filepath.Join(tempDir, "python3")
	scriptPath := filepath.Join(tempDir, "scripts", "analyze_audio.py")

	// Create Python executable
	os.WriteFile(pythonPath, []byte("#!/bin/sh\n"), 0755)

	// Create analysis script
	os.MkdirAll(filepath.Dir(scriptPath), 0755)
	os.WriteFile(scriptPath, []byte("#!/usr/bin/env python3"), 0755)

	// Simulate Docker environment variables
	os.Setenv("NAS_MOUNT_PATH", tempDir)
	os.Setenv("PYTHON_PATH", pythonPath)
	os.Setenv("ANALYSIS_SCRIPT", scriptPath)
	os.Setenv("DATABASE_URL", "postgresql://doyou:password@postgres:5432/doyou_music")
	os.Setenv("SCAN_INTERVAL", "30m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() in Docker env failed: %v", err)
	}

	// Verify Docker-style paths are accepted
	if cfg.NASMountPath != tempDir {
		t.Errorf("Docker NAS path = %v, want %v", cfg.NASMountPath, tempDir)
	}

	if cfg.ScanInterval != "30m" {
		t.Errorf("ScanInterval = %v, want 30m", cfg.ScanInterval)
	}
}

func TestConfig_MaskPasswordForLogging(t *testing.T) {
	tests := []struct {
		name        string
		dbURL       string
		expectMask  bool
		description string
	}{
		{
			name:        "password in URL",
			dbURL:       "postgresql://user:secretpass@localhost:5432/db",
			expectMask:  true,
			description: "Should mask password in logs",
		},
		{
			name:        "no password",
			dbURL:       "postgresql://user@localhost:5432/db",
			expectMask:  false,
			description: "No password to mask",
		},
		{
			name:        "docker service name",
			dbURL:       "postgresql://doyou:password@postgres:5432/doyou_music",
			expectMask:  true,
			description: "Docker compose service name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTestEnv(t)
			setupTestFiles(t)

			os.Setenv("DATABASE_URL", tt.dbURL)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			masked := cfg.MaskPassword()

			if tt.expectMask {
				// Should contain masked placeholder
				if masked == tt.dbURL {
					t.Error("Password was not masked")
				}
			}

			// Masked string should never contain actual password
			if tt.expectMask && masked == tt.dbURL {
				t.Error("MaskPassword() did not modify URL with password")
			}
		})
	}
}
func TestLoad_ShallowScan_DefaultFalse(t *testing.T) {
	clearTestEnv(t)
	setupTestFiles(t)
	// Ensure env var is not set
	os.Unsetenv("SHALLOW_SCAN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ShallowScan != false {
		t.Errorf("ShallowScan default = %v, want false", cfg.ShallowScan)
	}
}

func TestLoad_ShallowScan_EnvTrue(t *testing.T) {
	clearTestEnv(t)
	setupTestFiles(t)
	os.Setenv("SHALLOW_SCAN", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.ShallowScan {
		t.Errorf("ShallowScan = false, want true when SHALLOW_SCAN=true")
	}
}

func TestLoad_ShallowScan_Env1(t *testing.T) {
	clearTestEnv(t)
	setupTestFiles(t)
	os.Setenv("SHALLOW_SCAN", "1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.ShallowScan {
		t.Errorf("ShallowScan = false, want true when SHALLOW_SCAN=1")
	}
}

func TestLoad_ShallowScan_EnvFalse(t *testing.T) {
	clearTestEnv(t)
	setupTestFiles(t)
	os.Setenv("SHALLOW_SCAN", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ShallowScan {
		t.Errorf("ShallowScan = true, want false when SHALLOW_SCAN=false")
	}
}
