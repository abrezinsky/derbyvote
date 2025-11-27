package logger

import (
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
)

// Logger defines the logging interface used throughout the application
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	SetLevel(level slog.Level)
	GetLevel() slog.Level
	EnableHTTPLogging()
	DisableHTTPLogging()
	IsHTTPLoggingEnabled() bool
}

// SlogLogger wraps slog.Logger to implement our Logger interface
type SlogLogger struct {
	logger          *slog.Logger
	level           *slog.LevelVar
	httpLogging     atomic.Bool
}

// New creates a new SlogLogger with default settings (info level)
func New() *SlogLogger {
	return NewWithLevel(slog.LevelInfo)
}

// NewWithLevel creates a new SlogLogger with a specific level
func NewWithLevel(level slog.Level) *SlogLogger {
	levelVar := &slog.LevelVar{}
	levelVar.Set(level)

	sl := &SlogLogger{
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: levelVar,
		})),
		level: levelVar,
	}
	sl.httpLogging.Store(false)
	return sl
}

// ParseLevel converts a string log level to slog.Level.
// Accepts: debug, info, warn, error (case-insensitive).
// Returns slog.LevelInfo if the level is not recognized.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// SetLevel changes the logging level dynamically
func (l *SlogLogger) SetLevel(level slog.Level) {
	l.level.Set(level)
}

// GetLevel returns the current logging level
func (l *SlogLogger) GetLevel() slog.Level {
	return l.level.Level()
}

// EnableHTTPLogging enables HTTP request logging
func (l *SlogLogger) EnableHTTPLogging() {
	l.httpLogging.Store(true)
}

// DisableHTTPLogging disables HTTP request logging
func (l *SlogLogger) DisableHTTPLogging() {
	l.httpLogging.Store(false)
}

// IsHTTPLoggingEnabled returns whether HTTP logging is enabled
func (l *SlogLogger) IsHTTPLoggingEnabled() bool {
	return l.httpLogging.Load()
}
