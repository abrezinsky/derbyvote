package services_test

import (
	"strings"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/services"
)

func TestServiceError_Error(t *testing.T) {
	err := &services.ServiceError{Message: "test error message"}

	result := err.Error()

	if result != "test error message" {
		t.Errorf("expected 'test error message', got %q", result)
	}
}

func TestInvalidTableError_Error(t *testing.T) {
	err := &services.InvalidTableError{Table: "bad_table"}

	result := err.Error()

	if !strings.Contains(result, "bad_table") {
		t.Errorf("expected error to contain 'bad_table', got %q", result)
	}
	if !strings.Contains(result, "invalid table") {
		t.Errorf("expected error to mention 'invalid table', got %q", result)
	}
}

func TestPredefinedErrors(t *testing.T) {
	// Test that predefined errors return expected messages
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{"ErrInvalidTimerMinutes", services.ErrInvalidTimerMinutes, "minutes"},
		{"ErrNoTablesSpecified", services.ErrNoTablesSpecified, "tables"},
		{"ErrInvalidQRCount", services.ErrInvalidQRCount, "count"},
		{"ErrInvalidSeedType", services.ErrInvalidSeedType, "seed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if !strings.Contains(strings.ToLower(msg), tt.contains) {
				t.Errorf("expected error message to contain %q, got %q", tt.contains, msg)
			}
		})
	}
}
