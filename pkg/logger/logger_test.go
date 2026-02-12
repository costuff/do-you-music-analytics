package logger

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	logger := New("info")
	if logger == nil {
		t.Error("New() returned nil")
	}
}

func TestNewProduction(t *testing.T) {
	logger := NewProduction("info")
	if logger == nil {
		t.Error("NewProduction() returned nil")
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  zerolog.Level
	}{
		{"debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"invalid", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		got := parseLogLevel(tt.input)
		if got != tt.want {
			t.Errorf("parseLogLevel(%s) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLoggingMethods(t *testing.T) {
	logger := New("debug")

	// These should not panic
	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")
}
