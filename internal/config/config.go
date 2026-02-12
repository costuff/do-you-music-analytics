package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Database
	DatabaseURL string
	DBMaxConns  int
	DBTimeout   string
	
	// Storage
	NASMountPath          string
	ValidateNASOnStartup  bool
	
	// Analysis
	BatchSize       int
	MaxWorkers      int
	PythonPath      string
	AnalysisScript  string
	AnalysisTimeout string
	ScanInterval    string // How often to scan for new tracks (e.g., "5m", "1h")
	
	// Cleanup
	EnableCleanup   bool   // Enable automatic cleanup of missing files
	CleanupAction   string // Action for missing files: "mark" or "delete"
	CleanupInterval string // How often to run cleanup (e.g., "1h", "24h")
	
	// API
	APIPort    int
	LogLevel   string
	CORSOrigins string
	
	// Monitoring
	EnableMetrics bool
	MetricsPort   int
}

func Load() (*Config, error) {
	cfg := &Config{
		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://doyou:changeme@localhost:5432/doyou_music"),
		DBMaxConns:  getEnvInt("DB_MAX_CONNS", 20),
		DBTimeout:   getEnv("DB_TIMEOUT", "10s"),
		
		// Storage
		NASMountPath:         getEnv("NAS_MOUNT_PATH", "/mnt/music/downloads"),
		ValidateNASOnStartup: getEnvBool("VALIDATE_NAS_ON_STARTUP", true),
		
		// Analysis
		BatchSize:       getEnvInt("BATCH_SIZE", 10),
		MaxWorkers:      getEnvInt("MAX_WORKERS", 6),
		PythonPath:      getEnv("PYTHON_PATH", "/usr/bin/python3"),
		AnalysisScript:  getEnv("ANALYSIS_SCRIPT", "./scripts/analyze_audio.py"),
		AnalysisTimeout: getEnv("ANALYSIS_TIMEOUT", "300s"),
		ScanInterval:    getEnv("SCAN_INTERVAL", "30m"),
		
		// Cleanup
		EnableCleanup:   getEnvBool("ENABLE_CLEANUP", true),
		CleanupAction:   getEnv("CLEANUP_ACTION", "mark"),    // "mark" or "delete"
		CleanupInterval: getEnv("CLEANUP_INTERVAL", "24h"), // Scan for new tracks every 30 minutes
		
		// API
		APIPort:     getEnvInt("API_PORT", 4000),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:5173"),
		
		// Monitoring
		EnableMetrics: getEnvBool("ENABLE_METRICS", false),
		MetricsPort:   getEnvInt("METRICS_PORT", 9090),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	
	if c.APIPort < 1 || c.APIPort > 65535 {
		return fmt.Errorf("API_PORT must be between 1 and 65535, got %d", c.APIPort)
	}

	if c.BatchSize < 1 || c.BatchSize > 100 {
		return fmt.Errorf("BATCH_SIZE must be between 1 and 100, got %d", c.BatchSize)
	}
	
	if c.MaxWorkers < 1 || c.MaxWorkers > 32 {
		return fmt.Errorf("MAX_WORKERS must be between 1 and 32, got %d", c.MaxWorkers)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}
	
	// Validate NAS path if required
	if c.ValidateNASOnStartup {
		if _, err := os.Stat(c.NASMountPath); os.IsNotExist(err) {
			return fmt.Errorf("NAS mount path does not exist: %s (set VALIDATE_NAS_ON_STARTUP=false to skip)", c.NASMountPath)
		}
	}
	
	// Validate Python and analysis script paths
	if _, err := os.Stat(c.PythonPath); os.IsNotExist(err) {
		return fmt.Errorf("Python executable not found: %s", c.PythonPath)
	}
	
	if _, err := os.Stat(c.AnalysisScript); os.IsNotExist(err) {
		return fmt.Errorf("analysis script not found: %s", c.AnalysisScript)
	}

	return nil
}

func (c *Config) MaskPassword() string {
	// Mask password in database URL for logging
	if len(c.DatabaseURL) < 20 {
		return "***"
	}
	return c.DatabaseURL[:20] + "***"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
