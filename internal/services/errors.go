package services

import "fmt"

// Service errors
var (
	ErrInvalidTimerMinutes = &ServiceError{Message: "minutes must be between 1 and 60"}
	ErrNoTablesSpecified   = &ServiceError{Message: "no tables specified"}
	ErrInvalidQRCount      = &ServiceError{Message: "count must be between 1 and 200"}
	ErrInvalidSeedType     = &ServiceError{Message: "invalid seed type"}
	ErrVotingClosed        = &ServiceError{Message: "voting is currently closed"}
	ErrCarNotEligible      = &ServiceError{Message: "car is not eligible for voting"}
	ErrCarNotFound         = &ServiceError{Message: "car not found"}
	ErrUnregisteredQR      = &ServiceError{Message: "QR code is not registered"}
	ErrOpenVotingDisabled  = &ServiceError{Message: "open voting is disabled - only pre-registered QR codes are allowed"}
)

// ServiceError represents a service-level error
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// InvalidTableError represents an invalid table name error
type InvalidTableError struct {
	Table string
}

func (e *InvalidTableError) Error() string {
	return fmt.Sprintf("invalid table name: %s", e.Table)
}
