package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNew_DefaultsToInfoLevel(t *testing.T) {
	log := New()

	if log == nil {
		t.Fatal("expected logger to be created")
	}
	if log.logger == nil {
		t.Error("expected slog.Logger to be set")
	}
}

func TestNewWithLevel_SetsLevel(t *testing.T) {
	tests := []struct {
		level    slog.Level
		name     string
	}{
		{slog.LevelDebug, "debug"},
		{slog.LevelInfo, "info"},
		{slog.LevelWarn, "warn"},
		{slog.LevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := NewWithLevel(tt.level)
			if log == nil {
				t.Fatal("expected logger to be created")
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"Debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},    // Default
		{"", slog.LevelInfo},           // Default
		{"invalid", slog.LevelInfo},    // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSlogLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = (*SlogLogger)(nil)
}

func TestSlogLogger_LogMethods(t *testing.T) {
	// Create a logger that writes to a buffer for verification
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	log := &SlogLogger{logger: slog.New(handler)}

	tests := []struct {
		name   string
		fn     func(string, ...any)
		level  string
	}{
		{"Debug", log.Debug, "DEBUG"},
		{"Info", log.Info, "INFO"},
		{"Warn", log.Warn, "WARN"},
		{"Error", log.Error, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.fn("test message", "key", "value")

			output := buf.String()
			if !strings.Contains(output, tt.level) {
				t.Errorf("expected output to contain %q, got: %s", tt.level, output)
			}
			if !strings.Contains(output, "test message") {
				t.Errorf("expected output to contain message, got: %s", output)
			}
			if !strings.Contains(output, "key=value") {
				t.Errorf("expected output to contain key=value, got: %s", output)
			}
		})
	}
}

func TestSlogLogger_LevelFiltering(t *testing.T) {
	// Create logger at WARN level
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	log := &SlogLogger{logger: slog.New(handler)}

	// Debug and Info should be filtered
	log.Debug("debug message")
	log.Info("info message")

	if buf.Len() > 0 {
		t.Errorf("expected debug/info to be filtered at WARN level, got: %s", buf.String())
	}

	// Warn and Error should pass through
	log.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("expected warn message to be logged")
	}

	buf.Reset()
	log.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Error("expected error message to be logged")
	}
}

func TestSlogLogger_SetLevel(t *testing.T) {
	log := New()

	// Set to debug
	log.SetLevel(slog.LevelDebug)
	if log.GetLevel() != slog.LevelDebug {
		t.Errorf("expected level to be Debug, got %v", log.GetLevel())
	}

	// Set to warn
	log.SetLevel(slog.LevelWarn)
	if log.GetLevel() != slog.LevelWarn {
		t.Errorf("expected level to be Warn, got %v", log.GetLevel())
	}
}

func TestSlogLogger_GetLevel(t *testing.T) {
	// Test default level
	log := New()
	if log.GetLevel() != slog.LevelInfo {
		t.Errorf("expected default level to be Info, got %v", log.GetLevel())
	}

	// Test custom level
	log = NewWithLevel(slog.LevelError)
	if log.GetLevel() != slog.LevelError {
		t.Errorf("expected level to be Error, got %v", log.GetLevel())
	}
}

func TestSlogLogger_HTTPLogging(t *testing.T) {
	log := New()

	// Should be disabled by default
	if log.IsHTTPLoggingEnabled() {
		t.Error("expected HTTP logging to be disabled by default")
	}

	// Enable it
	log.EnableHTTPLogging()
	if !log.IsHTTPLoggingEnabled() {
		t.Error("expected HTTP logging to be enabled")
	}

	// Disable it
	log.DisableHTTPLogging()
	if log.IsHTTPLoggingEnabled() {
		t.Error("expected HTTP logging to be disabled")
	}
}
