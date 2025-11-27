package testutil

import (
	"testing"

	"github.com/abrezinsky/derbyvote/internal/repository"
)

// NewTestRepository creates a new in-memory repository for testing.
// Each call creates a fresh database with all migrations applied.
func NewTestRepository(t *testing.T) *repository.Repository {
	t.Helper()

	repo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	t.Cleanup(func() {
		// Repository doesn't expose Close, but :memory: dbs are cleaned up automatically
	})

	return repo
}
