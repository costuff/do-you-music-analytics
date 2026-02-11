package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog logger with application-specific methods
type Logger struct {
	logger zerolog.Logger
}

// New creates a new logger instance
func New(level string) *Logger {
	// Set time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Configure log level
	logLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	// Create pretty console writer for development
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()

	return &Logger{logger: logger}
}

// NewProduction creates a production logger (JSON output, no caller info)
func NewProduction(level string) *Logger {
	logLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()

	return &Logger{logger: logger}
}

// parseLogLevel converts string level to zerolog level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// ErrorWithErr logs an error with the error object
func (l *Logger) ErrorWithErr(err error, msg string) {
	l.logger.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal().Msgf(format, args...)
}

// With returns a new logger with additional context fields
func (l *Logger) With(key string, value interface{}) *Logger {
	return &Logger{
		logger: l.logger.With().Interface(key, value).Logger(),
	}
}

// WithFields returns a new logger with multiple context fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{logger: ctx.Logger()}
}

// GetZerolog returns the underlying zerolog logger
func (l *Logger) GetZerolog() zerolog.Logger {
	return l.logger
}

// Global logger instance
var global *Logger

// Init initializes the global logger
func Init(level string, isProduction bool) {
	if isProduction {
		global = NewProduction(level)
	} else {
		global = New(level)
	}
	log.Logger = global.logger
}

// Global convenience functions that use the global logger

// Debug logs a debug message using the global logger
func Debug(msg string) {
	if global != nil {
		global.Debug(msg)
	}
}

// Debugf logs a formatted debug message using the global logger
func Debugf(format string, args ...interface{}) {
	if global != nil {
		global.Debugf(format, args...)
	}
}

// Info logs an info message using the global logger
func Info(msg string) {
	if global != nil {
		global.Info(msg)
	}
}

// Infof logs a formatted info message using the global logger
func Infof(format string, args ...interface{}) {
	if global != nil {
		global.Infof(format, args...)
	}
}

// Warn logs a warning message using the global logger
func Warn(msg string) {
	if global != nil {
		global.Warn(msg)
	}
}

// Warnf logs a formatted warning message using the global logger
func Warnf(format string, args ...interface{}) {
	if global != nil {
		global.Warnf(format, args...)
	}
}

// Error logs an error message using the global logger
func Error(msg string) {
	if global != nil {
		global.Error(msg)
	}
}

// Errorf logs a formatted error message using the global logger
func Errorf(format string, args ...interface{}) {
	if global != nil {
		global.Errorf(format, args...)
	}
}

// ErrorWithErr logs an error with the error object using the global logger
func ErrorWithErr(err error, msg string) {
	if global != nil {
		global.ErrorWithErr(err, msg)
	}
}

// Fatal logs a fatal message and exits using the global logger
func Fatal(msg string) {
	if global != nil {
		global.Fatal(msg)
	}
}

// Fatalf logs a formatted fatal message and exits using the global logger
func Fatalf(format string, args ...interface{}) {
	if global != nil {
		global.Fatalf(format, args...)
	}
}
