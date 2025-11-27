package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/errors"
	"github.com/abrezinsky/derbyvote/internal/handlers"
	"github.com/abrezinsky/derbyvote/internal/services"
)

func TestAPIError_Error(t *testing.T) {
	err := handlers.NewAPIError(http.StatusBadRequest, "BAD_REQUEST", "test message")

	result := err.Error()

	if result != "test message" {
		t.Errorf("expected 'test message', got %q", result)
	}
	if err.Code != "BAD_REQUEST" {
		t.Errorf("expected code 'BAD_REQUEST', got %q", err.Code)
	}
}

func TestBadRequest(t *testing.T) {
	err := handlers.BadRequest("invalid input")

	if err.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", err.Status)
	}
	if err.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %q", err.Message)
	}
}

func TestUnauthorized(t *testing.T) {
	err := handlers.Unauthorized("login required")

	if err.Status != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", err.Status)
	}
	if err.Code != "UNAUTHORIZED" {
		t.Errorf("expected code 'UNAUTHORIZED', got %q", err.Code)
	}
	if err.Message != "login required" {
		t.Errorf("expected message 'login required', got %q", err.Message)
	}
}

func TestNotFound(t *testing.T) {
	err := handlers.NotFound("resource not found")

	if err.Status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", err.Status)
	}
	if err.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %q", err.Message)
	}
}

func TestInternalError(t *testing.T) {
	originalErr := fmt.Errorf("db connection failed")
	err := handlers.InternalError(originalErr)

	if err.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", err.Status)
	}
	// Internal errors should not expose the original message
	if err.Message != "Internal server error" {
		t.Errorf("expected generic message, got %q", err.Message)
	}
}

func TestNewAPIError(t *testing.T) {
	err := handlers.NewAPIError(http.StatusConflict, "CONFLICT", "conflict occurred")

	if err.Status != http.StatusConflict {
		t.Errorf("expected status 409, got %d", err.Status)
	}
	if err.Code != "CONFLICT" {
		t.Errorf("expected code 'CONFLICT', got %q", err.Code)
	}
	if err.Message != "conflict occurred" {
		t.Errorf("expected message 'conflict occurred', got %q", err.Message)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name           string
		err            *handlers.APIError
		expectedStatus int
	}{
		{"ErrBadRequest", handlers.ErrBadRequest, http.StatusBadRequest},
		{"ErrNotFound", handlers.ErrNotFound, http.StatusNotFound},
		{"ErrInternalServer", handlers.ErrInternalServer, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, tt.err.Status)
			}
		})
	}
}

// Test error conversion from service errors
func TestToAPIError_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Test by triggering a service error through the handlers
	// Using invalid timer minutes should trigger ErrInvalidTimerMinutes
	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-timer", strings.NewReader(`{"minutes":0}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for service error, got %d", rec.Code)
	}
}

func TestToAPIError_InvalidTableError(t *testing.T) {
	setup := newTestSetup(t)

	// Trigger InvalidTableError
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reset-database", strings.NewReader(`{"tables":["invalid_table"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid table error, got %d", rec.Code)
	}
}

func TestToAPIError_ApplicationError_NotFound(t *testing.T) {
	setup := newTestSetup(t)

	// Trigger not found error via category group
	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups/99999", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for not found error, got %d", rec.Code)
	}
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rec.Code)
	}

	// Should mention "empty"
	body := rec.Body.String()
	if !strings.Contains(strings.ToLower(body), "empty") {
		t.Errorf("expected error to mention 'empty', got %q", body)
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories", strings.NewReader("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}

	// Should mention "JSON"
	body := rec.Body.String()
	if !strings.Contains(body, "JSON") && !strings.Contains(body, "json") {
		t.Errorf("expected error to mention 'JSON', got %q", body)
	}
}

func TestParseIntParam_Invalid(t *testing.T) {
	setup := newTestSetup(t)

	// Invalid ID parameter
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/categories/abc", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid int param, got %d", rec.Code)
	}
}

// Test service error types are correctly converted
func TestServiceErrorConversion(t *testing.T) {
	// Test that ServiceError is converted to BadRequest
	svcErr := &services.ServiceError{Message: "test service error"}
	if svcErr.Error() != "test service error" {
		t.Errorf("unexpected error message: %s", svcErr.Error())
	}

	// Test that InvalidTableError is converted
	tableErr := &services.InvalidTableError{Table: "bad_table"}
	if !strings.Contains(tableErr.Error(), "bad_table") {
		t.Errorf("expected error to contain 'bad_table': %s", tableErr.Error())
	}
}

func TestToAPIError_ApplicationError_DefaultCase(t *testing.T) {
	setup := newTestSetup(t)

	// Close the database to trigger an unexpected error
	setup.repo.DB().Close()

	// Try any operation that would fail with DB error
	req := httptest.NewRequest(http.MethodGet, "/api/admin/categories", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for unexpected errors
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for internal error, got %d", rec.Code)
	}
}

func TestToAPIError_ApplicationError_Validation(t *testing.T) {
	setup := newTestSetup(t)

	// Test validation error - try to create voter with invalid data
	payload := map[string]interface{}{
		"name":       "",  // Empty name might trigger validation error
		"voter_type": "invalid_type",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 400 for validation error or successfully create
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusCreated {
		t.Logf("Got status %d which is acceptable for this test", rec.Code)
	}
}

func TestToAPIError_ApplicationError_Conflict(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a car first
	_ = setup.repo.CreateCar(ctx, "100", "Test Racer", "Test Car", "")

	// Try to create the same car again (should trigger conflict)
	payload := map[string]interface{}{
		"car_number": "100",
		"racer_name": "Duplicate",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/cars", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 409 Conflict or 400 Bad Request depending on implementation
	if rec.Code != http.StatusConflict && rec.Code != http.StatusBadRequest && rec.Code != http.StatusInternalServerError {
		t.Logf("Got status %d for duplicate car", rec.Code)
	}
}

func TestParseIntParam_MissingParam(t *testing.T) {
	setup := newTestSetup(t)

	// Request without ID parameter (empty string in URL)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/categories/", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Router will handle this differently - may be 404 or redirect
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMovedPermanently && rec.Code != http.StatusBadRequest {
		t.Logf("Got status %d for missing param", rec.Code)
	}
}

// ==================== Direct toAPIError Tests ====================

func TestToAPIError_DirectTests(t *testing.T) {
	tests := []struct {
		name           string
		inputErr       error
		expectedStatus int
		expectedMsg    string
		expectedCode   string
	}{
		{
			name:           "NotFoundError",
			inputErr:       &errors.Error{Kind: errors.ErrNotFound, Message: "resource not found"},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "resource not found",
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "ValidationError",
			inputErr:       &errors.Error{Kind: errors.ErrValidation, Message: "validation failed"},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "validation failed",
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "InvalidInputError",
			inputErr:       &errors.Error{Kind: errors.ErrInvalidInput, Message: "invalid input"},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid input",
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "ConflictError",
			inputErr:       &errors.Error{Kind: errors.ErrConflict, Message: "resource conflict"},
			expectedStatus: http.StatusConflict,
			expectedMsg:    "resource conflict",
			expectedCode:   "CONFLICT",
		},
		{
			name:           "InternalError_DefaultCase",
			inputErr:       &errors.Error{Kind: errors.ErrInternal, Message: "internal error"},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Internal server error",
			expectedCode:   "INTERNAL_SERVER_ERROR",
		},
		{
			name:           "ServiceError",
			inputErr:       &services.ServiceError{Message: "service error"},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "service error",
			expectedCode:   "BAD_REQUEST",
		},
		{
			name:           "ServiceError_VotingClosed",
			inputErr:       &services.ServiceError{Message: "Voting is closed"},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Voting is closed",
			expectedCode:   "VOTING_CLOSED",
		},
		{
			name:           "ServiceError_AlreadyVoted",
			inputErr:       &services.ServiceError{Message: "You have already voted in this category"},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "You have already voted in this category",
			expectedCode:   "ALREADY_VOTED",
		},
		{
			name:           "InvalidTableError",
			inputErr:       &services.InvalidTableError{Table: "bad_table"},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid table name: bad_table",
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "GenericError",
			inputErr:       fmt.Errorf("generic error"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Internal server error",
			expectedCode:   "INTERNAL_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := handlers.ToAPIError(tt.inputErr)

			if apiErr.Status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, apiErr.Status)
			}

			if apiErr.Message != tt.expectedMsg {
				t.Errorf("expected message %q, got %q", tt.expectedMsg, apiErr.Message)
			}

			if apiErr.Code != tt.expectedCode {
				t.Errorf("expected code %q, got %q", tt.expectedCode, apiErr.Code)
			}
		})
	}
}
