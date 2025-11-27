package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestListVoters_ScanError tests row scanning error
func TestListVoters_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock query that returns a row with wrong types to cause scan error
	rows := sqlmock.NewRows([]string{"id", "car_id", "name", "email", "voter_type", "qr_code", "notes", "created_at", "last_voted_at", "car_number", "racer_name"}).
		AddRow("not-a-number", nil, nil, nil, "general", "QR", nil, nil, nil, nil, nil) // id should be int, not string

	mock.ExpectQuery("SELECT (.+) FROM voters").WillReturnRows(rows)

	voters, err := repo.ListVoters(ctx)

	// Should return nil/empty list and continue on scan error (no error returned)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(voters) != 0 {
		t.Errorf("expected empty voters list on scan error, got %d voters", len(voters))
	}
}

// TestListCategories_ScanError tests row scanning error
func TestListCategories_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock query with invalid data type to trigger scan error
	rows := sqlmock.NewRows([]string{"id", "name", "display_order", "group_id", "derbynet_award_id", "name", "exclusivity_pool_id"}).
		AddRow("bad-id", "Cat", 1, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT (.+) FROM categories").WillReturnRows(rows)

	_, err = repo.ListCategories(ctx)

	// Should return error on scan failure
	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestListAllCategories_ScanError tests row scanning error
func TestListAllCategories_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock query with invalid data type
	rows := sqlmock.NewRows([]string{"id", "name", "display_order", "group_id", "derbynet_award_id", "active", "group_name"}).
		AddRow("bad-id", "Cat", 1, nil, nil, true, nil)

	mock.ExpectQuery("SELECT (.+) FROM categories").WillReturnRows(rows)

	_, err = repo.ListAllCategories(ctx)

	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestListCategoryGroups_ScanError tests row scanning error
func TestListCategoryGroups_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "name", "description", "exclusivity_pool_id", "display_order", "active"}).
		AddRow("bad-id", "Group", nil, nil, 1, true)

	mock.ExpectQuery("SELECT (.+) FROM category_groups").WillReturnRows(rows)

	_, err = repo.ListCategoryGroups(ctx)

	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestListCars_ScanError tests row scanning error
func TestListCars_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "car_number", "racer_name", "car_name", "photo_url", "eligible"}).
		AddRow("bad-id", "101", nil, nil, nil, true)

	mock.ExpectQuery("SELECT (.+) FROM cars WHERE active").WillReturnRows(rows)

	_, err = repo.ListCars(ctx)

	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestListEligibleCars_ScanError tests row scanning error
func TestListEligibleCars_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "car_number", "racer_name", "car_name", "photo_url", "eligible"}).
		AddRow("bad-id", "101", nil, nil, nil, true)

	mock.ExpectQuery("SELECT (.+) FROM cars WHERE active = 1 AND COALESCE").WillReturnRows(rows)

	_, err = repo.ListEligibleCars(ctx)

	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestGetVoterVotes_ScanError tests row scanning error
func TestGetVoterVotes_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"category_id", "car_id"}).
		AddRow("bad-id", 1)

	mock.ExpectQuery("SELECT category_id, car_id FROM votes").WillReturnRows(rows)

	_, err = repo.GetVoterVotes(ctx, 1)

	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestGetVoteResults_ScanError tests row scanning error
func TestGetVoteResults_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"category_id", "car_id", "vote_count"}).
		AddRow("bad-id", 1, 5)

	mock.ExpectQuery("SELECT category_id, car_id, COUNT").WillReturnRows(rows)

	_, err = repo.GetVoteResults(ctx)

	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestGetVoteResultsWithCars_ScanError tests row scanning error
func TestGetVoteResultsWithCars_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"category_id", "car_id", "car_number", "car_name", "racer_name", "photo_url", "vote_count"}).
		AddRow("bad-id", 1, "101", nil, nil, nil, 5)

	mock.ExpectQuery("SELECT (.+) FROM votes").WillReturnRows(rows)

	_, err = repo.GetVoteResultsWithCars(ctx)

	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestGetWinnersForDerbyNet_ScanError tests row scanning error
func TestGetWinnersForDerbyNet_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "name", "derbynet_award_id", "car_id", "derbynet_racer_id", "vote_count"}).
		AddRow("bad-id", "Cat", nil, 1, nil, 5)

	mock.ExpectQuery("WITH ranked_votes AS").WillReturnRows(rows)

	_, err = repo.GetWinnersForDerbyNet(ctx)

	if err == nil {
		t.Error("expected error from scan failure, got nil")
	}
}

// TestGetVotingStats_QueryErrors tests all query error paths
func TestGetVotingStats_QueryErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Test error on first query (total_voters)
	mock.ExpectQuery("SELECT COUNT.*FROM voters").WillReturnError(errors.New("query error"))

	_, err = repo.GetVotingStats(ctx)
	if err == nil {
		t.Error("expected error from first query, got nil")
	}

	// Test error on second query (voters_who_voted)
	mock.ExpectQuery("SELECT COUNT.*FROM voters").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery("SELECT COUNT.*DISTINCT voter_id.*FROM votes").WillReturnError(errors.New("query error"))

	_, err = repo.GetVotingStats(ctx)
	if err == nil {
		t.Error("expected error from second query, got nil")
	}

	// Test error on third query (total_votes)
	mock.ExpectQuery("SELECT COUNT.*FROM voters").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery("SELECT COUNT.*DISTINCT voter_id.*FROM votes").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery("SELECT COUNT.*FROM votes$").WillReturnError(errors.New("query error"))

	_, err = repo.GetVotingStats(ctx)
	if err == nil {
		t.Error("expected error from third query, got nil")
	}

	// Test error on fourth query (total_categories)
	mock.ExpectQuery("SELECT COUNT.*FROM voters").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery("SELECT COUNT.*DISTINCT voter_id.*FROM votes").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery("SELECT COUNT.*FROM votes$").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))
	mock.ExpectQuery("SELECT COUNT.*FROM categories WHERE active").WillReturnError(errors.New("query error"))

	_, err = repo.GetVotingStats(ctx)
	if err == nil {
		t.Error("expected error from fourth query, got nil")
	}

	// Test error on fifth query (total_cars)
	mock.ExpectQuery("SELECT COUNT.*FROM voters").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery("SELECT COUNT.*DISTINCT voter_id.*FROM votes").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery("SELECT COUNT.*FROM votes$").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))
	mock.ExpectQuery("SELECT COUNT.*FROM categories WHERE active").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery("SELECT COUNT.*FROM cars WHERE active").WillReturnError(errors.New("query error"))

	_, err = repo.GetVotingStats(ctx)
	if err == nil {
		t.Error("expected error from fifth query, got nil")
	}
}

// TestNew_MigrationError tests migration failure
func TestNew_MigrationError(t *testing.T) {
	// Test with invalid database path that will fail during migration
	_, err := New("/proc/invalid/path/test.db")
	if err == nil {
		t.Error("expected error when migration fails, got nil")
	}
}

// TestSaveVote_DeleteError tests delete vote error
func TestSaveVote_DeleteError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock delete error when carID is 0
	mock.ExpectExec("DELETE FROM votes WHERE voter_id").
		WillReturnError(errors.New("delete error"))

	err = repo.SaveVote(ctx, 1, 1, 0)
	if err == nil {
		t.Error("expected error from delete, got nil")
	}
}

// TestSaveVote_InsertError tests insert vote error
func TestSaveVote_InsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock insert error
	mock.ExpectExec("INSERT INTO votes").
		WillReturnError(errors.New("insert error"))

	err = repo.SaveVote(ctx, 1, 1, 1)
	if err == nil {
		t.Error("expected error from insert, got nil")
	}
}

// TestUpsertCategory_CategoryExistsError tests error checking if category exists
func TestUpsertCategory_CategoryExistsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock CategoryExists to return error
	mock.ExpectQuery("SELECT EXISTS.*FROM categories WHERE name").
		WillReturnError(errors.New("query error"))

	_, err = repo.UpsertCategory(ctx, "Test", 1, nil)
	if err == nil {
		t.Error("expected error from CategoryExists, got nil")
	}
}

// TestGetCategoryGroup_ScanError tests GetCategoryGroup scan error
func TestGetCategoryGroup_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock query with bad data type
	rows := sqlmock.NewRows([]string{"id", "name", "description", "exclusivity_pool_id", "display_order", "active"}).
		AddRow("bad-id", "Group", nil, nil, 1, true)

	mock.ExpectQuery("SELECT (.+) FROM category_groups WHERE id").WillReturnRows(rows)

	_, err = repo.GetCategoryGroup(ctx, "1")
	if err == nil {
		t.Error("expected error from scan, got nil")
	}
}

// TestGetCarByDerbyNetID_QueryError tests query error
func TestGetCarByDerbyNetID_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	mock.ExpectQuery("SELECT id FROM cars WHERE derbynet_racer_id").
		WillReturnError(errors.New("query error"))

	_, _, err = repo.GetCarByDerbyNetID(ctx, 1)
	if err == nil {
		t.Error("expected error from query, got nil")
	}
}

// TestGetVoterByQRCode_QueryError tests query error
func TestGetVoterByQRCode_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	mock.ExpectQuery("SELECT id FROM voters WHERE qr_code").
		WillReturnError(errors.New("query error"))

	_, _, err = repo.GetVoterByQRCode(ctx, "QR")
	if err == nil {
		t.Error("expected error from query, got nil")
	}
}

// TestFindConflictingVote_QueryError tests query error
func TestFindConflictingVote_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	mock.ExpectQuery("SELECT (.+) FROM votes").
		WillReturnError(errors.New("query error"))

	_, _, _, err = repo.FindConflictingVote(ctx, 1, 1, 1, 1)
	if err == nil {
		t.Error("expected error from query, got nil")
	}
}

// TestGetCar_ScanError tests scan error in GetCar
func TestGetCar_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "car_number", "racer_name", "car_name", "photo_url", "eligible"}).
		AddRow("bad-id", "101", nil, nil, nil, true)

	mock.ExpectQuery("SELECT (.+) FROM cars WHERE id").WillReturnRows(rows)

	_, err = repo.GetCar(ctx, 1)
	if err == nil {
		t.Error("expected error from scan, got nil")
	}
}

// TestNew_OpenError tests database connection error path
// Note: sql.Open with sqlite3 driver uses lazy initialization, so errors
// typically manifest during the first database operation (PRAGMA or migration)
// rather than during sql.Open itself. This test verifies error handling
// when the database connection fails during initialization.
func TestNew_PragmaOrMigrationError(t *testing.T) {
	// Use an invalid path that will fail during PRAGMA or migration
	_, err := New("/nonexistent/path/to/database.db")
	if err == nil {
		t.Error("expected error from database initialization, got nil")
	}
}

// TestListVoters_QueryError tests query error in ListVoters
func TestListVoters_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	mock.ExpectQuery("SELECT (.+) FROM voters").
		WillReturnError(errors.New("query error"))

	_, err = repo.ListVoters(ctx)
	if err == nil {
		t.Error("expected error from query, got nil")
	}
}

// TestListCategories_QueryError tests query error in ListCategories
func TestListCategories_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	mock.ExpectQuery("SELECT (.+) FROM categories").
		WillReturnError(errors.New("query error"))

	_, err = repo.ListCategories(ctx)
	if err == nil {
		t.Error("expected error from query, got nil")
	}
}

// TestListAllCategories_QueryError tests query error in ListAllCategories
func TestListAllCategories_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	mock.ExpectQuery("SELECT (.+) FROM categories").
		WillReturnError(errors.New("query error"))

	_, err = repo.ListAllCategories(ctx)
	if err == nil {
		t.Error("expected error from query, got nil")
	}
}

// TestGetCar_QueryError tests query error in GetCar
func TestGetCar_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	mock.ExpectQuery("SELECT (.+) FROM cars WHERE id").
		WillReturnError(errors.New("query error"))

	_, err = repo.GetCar(ctx, 1)
	if err == nil {
		t.Error("expected error from query, got nil")
	}
}

// TestMigrate_SettingsInsertError tests error when inserting default settings
func TestMigrate_SettingsInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Expect all CREATE TABLE statements to succeed
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS voters").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS cars").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS category_groups").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS categories").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS votes").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS settings").WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect CREATE INDEX statements
	for i := 0; i < 5; i++ {
		mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
	}

	// Expect ALTER TABLE statements (additionalMigrations - these are allowed to fail)
	for i := 0; i < 8; i++ {
		mock.ExpectExec("ALTER TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	}

	// First settings insert succeeds, second one fails
	mock.ExpectExec("INSERT OR IGNORE INTO settings").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT OR IGNORE INTO settings").WillReturnError(errors.New("insert error"))

	repo := &Repository{db: db}
	err = repo.migrate()
	if err == nil {
		t.Error("expected error from settings insert, got nil")
	}
}

// TestDeleteVoter_VoteDeleteError tests error when deleting voter's votes fails
func TestDeleteVoter_VoteDeleteError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock the DELETE votes query to fail
	mock.ExpectExec("DELETE FROM votes WHERE voter_id").
		WithArgs(1).
		WillReturnError(errors.New("failed to delete votes"))

	err = repo.DeleteVoter(ctx, 1)
	if err == nil {
		t.Error("expected error from vote delete, got nil")
	}
	if err.Error() != "failed to delete votes" {
		t.Errorf("expected 'failed to delete votes', got '%v'", err)
	}
}

// TestDeleteVoter_VoterDeleteError tests error when deleting voter fails
func TestDeleteVoter_VoterDeleteError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := &Repository{db: db}
	ctx := context.Background()

	// Mock successful vote deletion
	mock.ExpectExec("DELETE FROM votes WHERE voter_id").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock the DELETE voter query to fail
	mock.ExpectExec("DELETE FROM voters WHERE id").
		WithArgs(1).
		WillReturnError(errors.New("failed to delete voter"))

	err = repo.DeleteVoter(ctx, 1)
	if err == nil {
		t.Error("expected error from voter delete, got nil")
	}
	if err.Error() != "failed to delete voter" {
		t.Errorf("expected 'failed to delete voter', got '%v'", err)
	}
}
