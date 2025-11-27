package repository

import "errors"

// ErrNotFound is returned when a requested record is not found in the repository.
// This abstracts away the underlying storage implementation (SQL, NoSQL, etc.)
// from the service layer.
var ErrNotFound = errors.New("record not found")

// ErrInvalidTable is returned when attempting to clear a table that is not whitelisted.
// This prevents SQL injection attacks.
var ErrInvalidTable = errors.New("invalid table name")
