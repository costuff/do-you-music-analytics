package logger

import (
	"errors"
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

func TestNew_AllLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "invalid", ""}
	for _, level := range levels {
		l := New(level)
		if l == nil {
			t.Errorf("New(%q) returned nil", level)
		}
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
		{"", zerolog.InfoLevel},
		{"UNKNOWN", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		got := parseLogLevel(tt.input)
		if got != tt.want {
			t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLoggingMethods(t *testing.T) {
	l := New("debug")

	// All methods should not panic
	l.Debug("debug msg")
	l.Info("info msg")
	l.Warn("warn msg")
	l.Error("error msg")
}

func TestFormattedLoggingMethods(t *testing.T) {
	l := New("debug")

	// Formatted variants should not panic
	l.Debugf("debug %s %d", "test", 1)
	l.Infof("info %s %d", "test", 2)
	l.Warnf("warn %s %d", "test", 3)
	l.Errorf("error %s %d", "test", 4)
}

func TestErrorWithErr(t *testing.T) {
	l := New("debug")
	err := errors.New("test error")
	// Should not panic
	l.ErrorWithErr(err, "something failed")
}

func TestWith(t *testing.T) {
	l := New("info")
	l2 := l.With("key", "value")
	if l2 == nil {
		t.Error("With() returned nil")
	}
	// Should still log without panic
	l2.Info("with context")
}

func TestWithFields(t *testing.T) {
	l := New("info")
	l2 := l.WithFields(map[string]interface{}{
		"service": "test",
		"version": "1.0",
		"count":   42,
	})
	if l2 == nil {
		t.Error("WithFields() returned nil")
	}
	l2.Info("with multiple fields")
}

func TestGetZerolog(t *testing.T) {
	l := New("info")
	zl := l.GetZerolog()
	// GetZerolog should return a usable zerolog.Logger
	zl.Info().Msg("direct zerolog call")
}
