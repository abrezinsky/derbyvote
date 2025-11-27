package handlers

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/abrezinsky/derbyvote/internal/errors"
	"github.com/abrezinsky/derbyvote/internal/services"
)

// Error codes for standardized API error responses
const (
	ErrCodeBadRequest       = "BAD_REQUEST"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeValidation       = "VALIDATION_ERROR"
	ErrCodeInternalServer   = "INTERNAL_SERVER_ERROR"
	ErrCodeVotingClosed     = "VOTING_CLOSED"
	ErrCodeAlreadyVoted     = "ALREADY_VOTED"
	ErrCodeInvalidQRCode    = "INVALID_QR_CODE"
)

// APIError represents an error with an HTTP status code and error code
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"error"`
}

func (e *APIError) Error() string {
	return e.Message
}

// Common errors
var (
	ErrBadRequest       = &APIError{Status: http.StatusBadRequest, Code: ErrCodeBadRequest, Message: "Bad request"}
	ErrUnauthorized     = &APIError{Status: http.StatusUnauthorized, Code: ErrCodeUnauthorized, Message: "Unauthorized"}
	ErrNotFound         = &APIError{Status: http.StatusNotFound, Code: ErrCodeNotFound, Message: "Not found"}
	ErrInternalServer   = &APIError{Status: http.StatusInternalServerError, Code: ErrCodeInternalServer, Message: "Internal server error"}
)

// NewAPIError creates a new API error with custom message and code
func NewAPIError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// BadRequest creates a 400 error with custom message and auto-assigned error code
func BadRequest(message string) *APIError {
	code := ErrCodeBadRequest

	// Auto-assign specific error codes based on message content
	if strings.Contains(strings.ToLower(message), "qr code") {
		code = ErrCodeInvalidQRCode
	} else if strings.Contains(strings.ToLower(message), "validation") || strings.Contains(strings.ToLower(message), "invalid") {
		code = ErrCodeValidation
	}

	return &APIError{Status: http.StatusBadRequest, Code: code, Message: message}
}

// Unauthorized creates a 401 error with custom message
func Unauthorized(message string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Code: ErrCodeUnauthorized, Message: message}
}

// NotFound creates a 404 error with custom message
func NotFound(message string) *APIError {
	return &APIError{Status: http.StatusNotFound, Code: ErrCodeNotFound, Message: message}
}

// Conflict creates a 409 error with custom message
func Conflict(message string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: ErrCodeConflict, Message: message}
}

// InternalError creates a 500 error, logs the original error
func InternalError(err error) *APIError {
	log.Printf("Internal error: %v", err)
	return &APIError{Status: http.StatusInternalServerError, Code: ErrCodeInternalServer, Message: "Internal server error"}
}

// respondJSON writes a JSON response with the given status code
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondOK writes a 200 OK JSON response
func respondOK(w http.ResponseWriter, data interface{}) {
	respondJSON(w, http.StatusOK, data)
}

// respondCreated writes a 201 Created JSON response
func respondCreated(w http.ResponseWriter, data interface{}) {
	respondJSON(w, http.StatusCreated, data)
}

// respondSuccess writes a 200 OK with a message
func respondSuccess(w http.ResponseWriter, message string) {
	respondJSON(w, http.StatusOK, map[string]string{"message": message})
}

// respondDeleted writes a 204 No Content response
func respondDeleted(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// respondError writes an error response
func respondError(w http.ResponseWriter, err error) {
	if apiErr, ok := err.(*APIError); ok {
		respondJSON(w, apiErr.Status, apiErr)
		return
	}
	// Convert service errors to appropriate API errors
	apiErr := ToAPIError(err)
	respondJSON(w, apiErr.Status, apiErr)
}

// decodeJSON decodes JSON from request body into the target
func decodeJSON(r *http.Request, target interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		if err == io.EOF {
			return BadRequest("Request body is empty")
		}
		return BadRequest("Invalid JSON: " + err.Error())
	}
	return nil
}

// parseIntParam extracts and parses an integer URL parameter
func parseIntParam(r *http.Request, name string) (int, error) {
	param := chi.URLParam(r, name)
	if param == "" {
		return 0, BadRequest("Missing " + name + " parameter")
	}
	id, err := strconv.Atoi(param)
	if err != nil {
		return 0, BadRequest("Invalid " + name + " parameter")
	}
	return id, nil
}

// ToAPIError converts service errors to appropriate API errors
func ToAPIError(err error) *APIError {
	// Check for application errors first
	var appErr *errors.Error
	if stderrors.As(err, &appErr) {
		switch appErr.Kind {
		case errors.ErrNotFound:
			return NotFound(appErr.Message)
		case errors.ErrValidation, errors.ErrInvalidInput:
			return &APIError{Status: http.StatusBadRequest, Code: ErrCodeValidation, Message: appErr.Message}
		case errors.ErrConflict:
			return Conflict(appErr.Message)
		default:
			return InternalError(err)
		}
	}

	// Legacy service errors (can migrate these over time)
	if svcErr, ok := err.(*services.ServiceError); ok {
		// Map specific service error types to error codes
		if svcErr.Message == "Voting is closed" {
			return &APIError{Status: http.StatusBadRequest, Code: ErrCodeVotingClosed, Message: svcErr.Message}
		}
		if svcErr.Message == "You have already voted in this category" {
			return &APIError{Status: http.StatusBadRequest, Code: ErrCodeAlreadyVoted, Message: svcErr.Message}
		}
		return BadRequest(svcErr.Message)
	}
	if tableErr, ok := err.(*services.InvalidTableError); ok {
		return BadRequest(tableErr.Error())
	}

	return InternalError(err)
}
