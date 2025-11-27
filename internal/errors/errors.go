package errors

import "fmt"

// Kind represents the type of error
type Kind int

const (
	ErrInternal Kind = iota
	ErrNotFound
	ErrValidation
	ErrConflict
	ErrInvalidInput
)

// Error is an application-level error with a kind for classification
type Error struct {
	Kind    Kind
	Message string
	Err     error // underlying error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Constructor functions for common error types

func NotFound(msg string) *Error {
	return &Error{Kind: ErrNotFound, Message: msg}
}

func NotFoundf(format string, args ...interface{}) *Error {
	return &Error{Kind: ErrNotFound, Message: fmt.Sprintf(format, args...)}
}

func Validation(msg string) *Error {
	return &Error{Kind: ErrValidation, Message: msg}
}

func Validationf(format string, args ...interface{}) *Error {
	return &Error{Kind: ErrValidation, Message: fmt.Sprintf(format, args...)}
}

func Conflict(msg string) *Error {
	return &Error{Kind: ErrConflict, Message: msg}
}

func Conflictf(format string, args ...interface{}) *Error {
	return &Error{Kind: ErrConflict, Message: fmt.Sprintf(format, args...)}
}

func InvalidInput(msg string) *Error {
	return &Error{Kind: ErrInvalidInput, Message: msg}
}

func InvalidInputf(format string, args ...interface{}) *Error {
	return &Error{Kind: ErrInvalidInput, Message: fmt.Sprintf(format, args...)}
}

func Internal(err error) *Error {
	return &Error{Kind: ErrInternal, Message: "internal error", Err: err}
}

func Internalf(format string, args ...interface{}) *Error {
	return &Error{Kind: ErrInternal, Message: fmt.Sprintf(format, args...)}
}

// Wrap wraps an error with additional context
func Wrap(err error, kind Kind, msg string) *Error {
	return &Error{Kind: kind, Message: msg, Err: err}
}
