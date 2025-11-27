package errors

import (
	"errors"
	"fmt"
	"testing"
)

// =============================================================================
// Test Error Types and Constructors
// =============================================================================

func TestNotFound(t *testing.T) {
	err := NotFound("resource not found")

	if err.Kind != ErrNotFound {
		t.Errorf("expected Kind to be ErrNotFound (%d), got %d", ErrNotFound, err.Kind)
	}
	if err.Message != "resource not found" {
		t.Errorf("expected Message to be 'resource not found', got '%s'", err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestNotFoundf(t *testing.T) {
	err := NotFoundf("user %d not found", 123)

	if err.Kind != ErrNotFound {
		t.Errorf("expected Kind to be ErrNotFound (%d), got %d", ErrNotFound, err.Kind)
	}
	if err.Message != "user 123 not found" {
		t.Errorf("expected Message to be 'user 123 not found', got '%s'", err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestValidation(t *testing.T) {
	err := Validation("invalid email format")

	if err.Kind != ErrValidation {
		t.Errorf("expected Kind to be ErrValidation (%d), got %d", ErrValidation, err.Kind)
	}
	if err.Message != "invalid email format" {
		t.Errorf("expected Message to be 'invalid email format', got '%s'", err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestValidationf(t *testing.T) {
	err := Validationf("field %s must be at least %d characters", "password", 8)

	if err.Kind != ErrValidation {
		t.Errorf("expected Kind to be ErrValidation (%d), got %d", ErrValidation, err.Kind)
	}
	expectedMsg := "field password must be at least 8 characters"
	if err.Message != expectedMsg {
		t.Errorf("expected Message to be '%s', got '%s'", expectedMsg, err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestConflict(t *testing.T) {
	err := Conflict("resource already exists")

	if err.Kind != ErrConflict {
		t.Errorf("expected Kind to be ErrConflict (%d), got %d", ErrConflict, err.Kind)
	}
	if err.Message != "resource already exists" {
		t.Errorf("expected Message to be 'resource already exists', got '%s'", err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestConflictf(t *testing.T) {
	err := Conflictf("user with email %s already exists", "test@example.com")

	if err.Kind != ErrConflict {
		t.Errorf("expected Kind to be ErrConflict (%d), got %d", ErrConflict, err.Kind)
	}
	expectedMsg := "user with email test@example.com already exists"
	if err.Message != expectedMsg {
		t.Errorf("expected Message to be '%s', got '%s'", expectedMsg, err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestInvalidInput(t *testing.T) {
	err := InvalidInput("missing required field")

	if err.Kind != ErrInvalidInput {
		t.Errorf("expected Kind to be ErrInvalidInput (%d), got %d", ErrInvalidInput, err.Kind)
	}
	if err.Message != "missing required field" {
		t.Errorf("expected Message to be 'missing required field', got '%s'", err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestInvalidInputf(t *testing.T) {
	err := InvalidInputf("invalid value %q for field %s", "abc", "age")

	if err.Kind != ErrInvalidInput {
		t.Errorf("expected Kind to be ErrInvalidInput (%d), got %d", ErrInvalidInput, err.Kind)
	}
	expectedMsg := `invalid value "abc" for field age`
	if err.Message != expectedMsg {
		t.Errorf("expected Message to be '%s', got '%s'", expectedMsg, err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestInternal(t *testing.T) {
	underlyingErr := fmt.Errorf("database connection failed")
	err := Internal(underlyingErr)

	if err.Kind != ErrInternal {
		t.Errorf("expected Kind to be ErrInternal (%d), got %d", ErrInternal, err.Kind)
	}
	if err.Message != "internal error" {
		t.Errorf("expected Message to be 'internal error', got '%s'", err.Message)
	}
	if err.Err != underlyingErr {
		t.Errorf("expected Err to be %v, got %v", underlyingErr, err.Err)
	}
}

func TestInternalWithNilError(t *testing.T) {
	err := Internal(nil)

	if err.Kind != ErrInternal {
		t.Errorf("expected Kind to be ErrInternal (%d), got %d", ErrInternal, err.Kind)
	}
	if err.Message != "internal error" {
		t.Errorf("expected Message to be 'internal error', got '%s'", err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

func TestInternalf(t *testing.T) {
	err := Internalf("failed to process request: %s", "timeout")

	if err.Kind != ErrInternal {
		t.Errorf("expected Kind to be ErrInternal (%d), got %d", ErrInternal, err.Kind)
	}
	expectedMsg := "failed to process request: timeout"
	if err.Message != expectedMsg {
		t.Errorf("expected Message to be '%s', got '%s'", expectedMsg, err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

// =============================================================================
// Test Wrap Function
// =============================================================================

func TestWrap(t *testing.T) {
	underlyingErr := fmt.Errorf("original error")
	err := Wrap(underlyingErr, ErrNotFound, "wrapped context")

	if err.Kind != ErrNotFound {
		t.Errorf("expected Kind to be ErrNotFound (%d), got %d", ErrNotFound, err.Kind)
	}
	if err.Message != "wrapped context" {
		t.Errorf("expected Message to be 'wrapped context', got '%s'", err.Message)
	}
	if err.Err != underlyingErr {
		t.Errorf("expected Err to be %v, got %v", underlyingErr, err.Err)
	}
}

func TestWrapWithDifferentKinds(t *testing.T) {
	testCases := []struct {
		name string
		kind Kind
	}{
		{"ErrInternal", ErrInternal},
		{"ErrNotFound", ErrNotFound},
		{"ErrValidation", ErrValidation},
		{"ErrConflict", ErrConflict},
		{"ErrInvalidInput", ErrInvalidInput},
	}

	underlyingErr := fmt.Errorf("base error")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Wrap(underlyingErr, tc.kind, "test message")
			if err.Kind != tc.kind {
				t.Errorf("expected Kind to be %d, got %d", tc.kind, err.Kind)
			}
		})
	}
}

func TestWrapWithNilError(t *testing.T) {
	err := Wrap(nil, ErrValidation, "no underlying error")

	if err.Kind != ErrValidation {
		t.Errorf("expected Kind to be ErrValidation (%d), got %d", ErrValidation, err.Kind)
	}
	if err.Message != "no underlying error" {
		t.Errorf("expected Message to be 'no underlying error', got '%s'", err.Message)
	}
	if err.Err != nil {
		t.Errorf("expected Err to be nil, got %v", err.Err)
	}
}

// =============================================================================
// Test Error Interface
// =============================================================================

func TestErrorMethod_WithoutWrappedError(t *testing.T) {
	err := &Error{
		Kind:    ErrNotFound,
		Message: "user not found",
		Err:     nil,
	}

	expected := "user not found"
	if err.Error() != expected {
		t.Errorf("expected Error() to return '%s', got '%s'", expected, err.Error())
	}
}

func TestErrorMethod_WithWrappedError(t *testing.T) {
	underlyingErr := fmt.Errorf("database query failed")
	err := &Error{
		Kind:    ErrInternal,
		Message: "failed to fetch user",
		Err:     underlyingErr,
	}

	expected := "failed to fetch user: database query failed"
	if err.Error() != expected {
		t.Errorf("expected Error() to return '%s', got '%s'", expected, err.Error())
	}
}

func TestErrorMethod_WithNestedWrappedError(t *testing.T) {
	innerErr := fmt.Errorf("connection refused")
	middleErr := fmt.Errorf("database error: %w", innerErr)
	err := &Error{
		Kind:    ErrInternal,
		Message: "service unavailable",
		Err:     middleErr,
	}

	expected := "service unavailable: database error: connection refused"
	if err.Error() != expected {
		t.Errorf("expected Error() to return '%s', got '%s'", expected, err.Error())
	}
}

func TestUnwrap(t *testing.T) {
	underlyingErr := fmt.Errorf("original error")
	err := &Error{
		Kind:    ErrInternal,
		Message: "wrapper",
		Err:     underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("expected Unwrap() to return %v, got %v", underlyingErr, unwrapped)
	}
}

func TestUnwrap_NilError(t *testing.T) {
	err := &Error{
		Kind:    ErrNotFound,
		Message: "not found",
		Err:     nil,
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("expected Unwrap() to return nil, got %v", unwrapped)
	}
}

// =============================================================================
// Test Error Type Checking with errors.As
// =============================================================================

func TestErrorsAs_DirectError(t *testing.T) {
	err := NotFound("user not found")

	var appErr *Error
	if !errors.As(err, &appErr) {
		t.Error("expected errors.As to return true for *Error")
	}
	if appErr.Kind != ErrNotFound {
		t.Errorf("expected Kind to be ErrNotFound, got %d", appErr.Kind)
	}
}

func TestErrorsAs_WrappedError(t *testing.T) {
	innerErr := fmt.Errorf("db error")
	appErr := Wrap(innerErr, ErrInternal, "service error")
	wrappedErr := fmt.Errorf("handler error: %w", appErr)

	var extractedErr *Error
	if !errors.As(wrappedErr, &extractedErr) {
		t.Error("expected errors.As to return true for wrapped *Error")
	}
	if extractedErr.Kind != ErrInternal {
		t.Errorf("expected Kind to be ErrInternal, got %d", extractedErr.Kind)
	}
}

func TestErrorsAs_NonAppError(t *testing.T) {
	err := fmt.Errorf("regular error")

	var appErr *Error
	if errors.As(err, &appErr) {
		t.Error("expected errors.As to return false for non-*Error")
	}
}

func TestErrorsAs_NilError(t *testing.T) {
	var err error = nil

	var appErr *Error
	if errors.As(err, &appErr) {
		t.Error("expected errors.As to return false for nil error")
	}
}

// =============================================================================
// Test Kind in Switch Statements
// =============================================================================

func TestKindSwitch(t *testing.T) {
	testCases := []struct {
		name         string
		err          *Error
		expectedKind Kind
	}{
		{"ErrInternal", Internal(nil), ErrInternal},
		{"ErrNotFound", NotFound("test"), ErrNotFound},
		{"ErrValidation", Validation("test"), ErrValidation},
		{"ErrConflict", Conflict("test"), ErrConflict},
		{"ErrInvalidInput", InvalidInput("test"), ErrInvalidInput},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var matched Kind
			switch tc.err.Kind {
			case ErrInternal:
				matched = ErrInternal
			case ErrNotFound:
				matched = ErrNotFound
			case ErrValidation:
				matched = ErrValidation
			case ErrConflict:
				matched = ErrConflict
			case ErrInvalidInput:
				matched = ErrInvalidInput
			default:
				t.Fatal("unexpected error kind")
			}

			if matched != tc.expectedKind {
				t.Errorf("expected kind %d, got %d", tc.expectedKind, matched)
			}
		})
	}
}

func TestKindSwitchFromExtractedError(t *testing.T) {
	err := NotFoundf("resource %s not found", "item-123")
	wrappedErr := fmt.Errorf("handler: %w", err)

	var appErr *Error
	if !errors.As(wrappedErr, &appErr) {
		t.Fatal("expected to extract *Error from wrapped error")
	}

	switch appErr.Kind {
	case ErrNotFound:
		// Expected case
	default:
		t.Errorf("expected ErrNotFound kind, got %d", appErr.Kind)
	}
}

// =============================================================================
// Test Formatting Functions (f variants)
// =============================================================================

func TestNotFoundf_Formatting(t *testing.T) {
	testCases := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "single string arg",
			format:   "user %s not found",
			args:     []interface{}{"john"},
			expected: "user john not found",
		},
		{
			name:     "single int arg",
			format:   "item %d not found",
			args:     []interface{}{42},
			expected: "item 42 not found",
		},
		{
			name:     "multiple args",
			format:   "%s with id %d not found in %s",
			args:     []interface{}{"User", 123, "database"},
			expected: "User with id 123 not found in database",
		},
		{
			name:     "no args",
			format:   "resource not found",
			args:     []interface{}{},
			expected: "resource not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := NotFoundf(tc.format, tc.args...)
			if err.Message != tc.expected {
				t.Errorf("expected Message '%s', got '%s'", tc.expected, err.Message)
			}
		})
	}
}

func TestValidationf_MultipleArguments(t *testing.T) {
	testCases := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "field validation with bounds",
			format:   "field %s must be between %d and %d",
			args:     []interface{}{"age", 0, 150},
			expected: "field age must be between 0 and 150",
		},
		{
			name:     "type validation",
			format:   "expected %s, got %T",
			args:     []interface{}{"string", 123},
			expected: "expected string, got int",
		},
		{
			name:     "quoted value",
			format:   "invalid value %q for %s",
			args:     []interface{}{"not-a-number", "count"},
			expected: `invalid value "not-a-number" for count`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validationf(tc.format, tc.args...)
			if err.Message != tc.expected {
				t.Errorf("expected Message '%s', got '%s'", tc.expected, err.Message)
			}
			if err.Kind != ErrValidation {
				t.Errorf("expected Kind ErrValidation, got %d", err.Kind)
			}
		})
	}
}

func TestConflictf_Formatting(t *testing.T) {
	err := Conflictf("entity %s with key %s=%v already exists", "User", "email", "test@example.com")
	expected := "entity User with key email=test@example.com already exists"
	if err.Message != expected {
		t.Errorf("expected Message '%s', got '%s'", expected, err.Message)
	}
}

func TestInvalidInputf_Formatting(t *testing.T) {
	err := InvalidInputf("parameter %s cannot be %s", "count", "negative")
	expected := "parameter count cannot be negative"
	if err.Message != expected {
		t.Errorf("expected Message '%s', got '%s'", expected, err.Message)
	}
}

func TestInternalf_Formatting(t *testing.T) {
	err := Internalf("failed to connect to %s:%d after %d retries", "localhost", 5432, 3)
	expected := "failed to connect to localhost:5432 after 3 retries"
	if err.Message != expected {
		t.Errorf("expected Message '%s', got '%s'", expected, err.Message)
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestEmptyMessage(t *testing.T) {
	err := NotFound("")

	if err.Kind != ErrNotFound {
		t.Errorf("expected Kind to be ErrNotFound, got %d", err.Kind)
	}
	if err.Message != "" {
		t.Errorf("expected empty Message, got '%s'", err.Message)
	}
	if err.Error() != "" {
		t.Errorf("expected Error() to return empty string, got '%s'", err.Error())
	}
}

func TestEmptyFormatString(t *testing.T) {
	err := NotFoundf("")

	if err.Message != "" {
		t.Errorf("expected empty Message, got '%s'", err.Message)
	}
}

func TestSpecialCharactersInMessage(t *testing.T) {
	specialMsg := "error with special chars: <>&\"'\n\t%s"
	err := Validation(specialMsg)

	if err.Message != specialMsg {
		t.Errorf("expected Message to preserve special chars, got '%s'", err.Message)
	}
}

func TestUnicodeInMessage(t *testing.T) {
	unicodeMsg := "error: user not found"
	err := NotFound(unicodeMsg)

	if err.Message != unicodeMsg {
		t.Errorf("expected Message to preserve unicode, got '%s'", err.Message)
	}
}

func TestKindConstants(t *testing.T) {
	// Verify the Kind constants have expected iota values
	if ErrInternal != 0 {
		t.Errorf("expected ErrInternal to be 0, got %d", ErrInternal)
	}
	if ErrNotFound != 1 {
		t.Errorf("expected ErrNotFound to be 1, got %d", ErrNotFound)
	}
	if ErrValidation != 2 {
		t.Errorf("expected ErrValidation to be 2, got %d", ErrValidation)
	}
	if ErrConflict != 3 {
		t.Errorf("expected ErrConflict to be 3, got %d", ErrConflict)
	}
	if ErrInvalidInput != 4 {
		t.Errorf("expected ErrInvalidInput to be 4, got %d", ErrInvalidInput)
	}
}

// =============================================================================
// Test errors.Is compatibility (chain unwrapping)
// =============================================================================

func TestErrorsIs_WithWrappedStandardError(t *testing.T) {
	sentinelErr := fmt.Errorf("sentinel error")
	appErr := Wrap(sentinelErr, ErrInternal, "application error")

	if !errors.Is(appErr, sentinelErr) {
		t.Error("expected errors.Is to find sentinel error in chain")
	}
}

func TestErrorsIs_DeeplyNestedError(t *testing.T) {
	sentinelErr := fmt.Errorf("sentinel error")
	level1 := fmt.Errorf("level 1: %w", sentinelErr)
	level2 := Wrap(level1, ErrInternal, "level 2")
	level3 := fmt.Errorf("level 3: %w", level2)

	if !errors.Is(level3, sentinelErr) {
		t.Error("expected errors.Is to find sentinel error in deeply nested chain")
	}
}

// =============================================================================
// Test that Error satisfies the error interface
// =============================================================================

func TestErrorImplementsErrorInterface(t *testing.T) {
	var _ error = &Error{}
	var _ error = NotFound("test")
	var _ error = Validation("test")
	var _ error = Conflict("test")
	var _ error = InvalidInput("test")
	var _ error = Internal(nil)
	var _ error = Internalf("test")
}

// =============================================================================
// Table-driven test for all constructor functions
// =============================================================================

func TestAllConstructors(t *testing.T) {
	underlyingErr := fmt.Errorf("underlying")

	testCases := []struct {
		name         string
		constructor  func() *Error
		expectedKind Kind
		checkMessage string
		hasErr       bool
	}{
		{
			name:         "NotFound",
			constructor:  func() *Error { return NotFound("msg") },
			expectedKind: ErrNotFound,
			checkMessage: "msg",
			hasErr:       false,
		},
		{
			name:         "NotFoundf",
			constructor:  func() *Error { return NotFoundf("msg %d", 1) },
			expectedKind: ErrNotFound,
			checkMessage: "msg 1",
			hasErr:       false,
		},
		{
			name:         "Validation",
			constructor:  func() *Error { return Validation("msg") },
			expectedKind: ErrValidation,
			checkMessage: "msg",
			hasErr:       false,
		},
		{
			name:         "Validationf",
			constructor:  func() *Error { return Validationf("msg %d", 1) },
			expectedKind: ErrValidation,
			checkMessage: "msg 1",
			hasErr:       false,
		},
		{
			name:         "Conflict",
			constructor:  func() *Error { return Conflict("msg") },
			expectedKind: ErrConflict,
			checkMessage: "msg",
			hasErr:       false,
		},
		{
			name:         "Conflictf",
			constructor:  func() *Error { return Conflictf("msg %d", 1) },
			expectedKind: ErrConflict,
			checkMessage: "msg 1",
			hasErr:       false,
		},
		{
			name:         "InvalidInput",
			constructor:  func() *Error { return InvalidInput("msg") },
			expectedKind: ErrInvalidInput,
			checkMessage: "msg",
			hasErr:       false,
		},
		{
			name:         "InvalidInputf",
			constructor:  func() *Error { return InvalidInputf("msg %d", 1) },
			expectedKind: ErrInvalidInput,
			checkMessage: "msg 1",
			hasErr:       false,
		},
		{
			name:         "Internal",
			constructor:  func() *Error { return Internal(underlyingErr) },
			expectedKind: ErrInternal,
			checkMessage: "internal error",
			hasErr:       true,
		},
		{
			name:         "Internalf",
			constructor:  func() *Error { return Internalf("msg %d", 1) },
			expectedKind: ErrInternal,
			checkMessage: "msg 1",
			hasErr:       false,
		},
		{
			name:         "Wrap_NotFound",
			constructor:  func() *Error { return Wrap(underlyingErr, ErrNotFound, "msg") },
			expectedKind: ErrNotFound,
			checkMessage: "msg",
			hasErr:       true,
		},
		{
			name:         "Wrap_Validation",
			constructor:  func() *Error { return Wrap(underlyingErr, ErrValidation, "msg") },
			expectedKind: ErrValidation,
			checkMessage: "msg",
			hasErr:       true,
		},
		{
			name:         "Wrap_Conflict",
			constructor:  func() *Error { return Wrap(underlyingErr, ErrConflict, "msg") },
			expectedKind: ErrConflict,
			checkMessage: "msg",
			hasErr:       true,
		},
		{
			name:         "Wrap_InvalidInput",
			constructor:  func() *Error { return Wrap(underlyingErr, ErrInvalidInput, "msg") },
			expectedKind: ErrInvalidInput,
			checkMessage: "msg",
			hasErr:       true,
		},
		{
			name:         "Wrap_Internal",
			constructor:  func() *Error { return Wrap(underlyingErr, ErrInternal, "msg") },
			expectedKind: ErrInternal,
			checkMessage: "msg",
			hasErr:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.constructor()

			if err.Kind != tc.expectedKind {
				t.Errorf("expected Kind %d, got %d", tc.expectedKind, err.Kind)
			}
			if err.Message != tc.checkMessage {
				t.Errorf("expected Message '%s', got '%s'", tc.checkMessage, err.Message)
			}
			if tc.hasErr && err.Err == nil {
				t.Error("expected Err to be non-nil")
			}
			if !tc.hasErr && err.Err != nil {
				t.Errorf("expected Err to be nil, got %v", err.Err)
			}
		})
	}
}
