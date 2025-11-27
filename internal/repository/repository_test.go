package repository

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/abrezinsky/derbyvote/internal/errors"
	"github.com/abrezinsky/derbyvote/internal/models"
)

// newTestRepo creates a new in-memory repository for testing.
func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}
	return repo
}

// ==================== Voter Tests ====================

func TestGetVoterByQR_ExistingVoter(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a voter
	id, err := repo.CreateVoter(ctx, "TEST-QR1")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	// Get voter by QR code
	voterID, err := repo.GetVoterByQR(ctx, "TEST-QR1")
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}
	if voterID != id {
		t.Errorf("expected voter ID %d, got %d", id, voterID)
	}
}

func TestGetVoterByQR_NonExistent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetVoterByQR(ctx, "NON-EXISTENT")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateVoter_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, err := repo.CreateVoter(ctx, "SIMPLE-QR")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

func TestCreateVoter_DuplicateQR(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.CreateVoter(ctx, "DUPE-QR")
	if err != nil {
		t.Fatalf("first CreateVoter failed: %v", err)
	}

	// Attempt to create duplicate - should fail due to UNIQUE constraint
	_, err = repo.CreateVoter(ctx, "DUPE-QR")
	if err == nil {
		t.Error("expected error for duplicate QR code, got nil")
	}
}

func TestCreateVoterFull_AllFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, err := repo.CreateVoterFull(ctx, nil, "John Doe", "john@example.com", "judge", "FULL-QR1", "Test notes")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Verify fields via ListVoters
	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(voters))
	}

	v := voters[0]
	if v["qr_code"] != "FULL-QR1" {
		t.Errorf("expected qr_code 'FULL-QR1', got %v", v["qr_code"])
	}
	if v["name"] != "John Doe" {
		t.Errorf("expected name 'John Doe', got %v", v["name"])
	}
	if v["email"] != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got %v", v["email"])
	}
	if v["voter_type"] != "judge" {
		t.Errorf("expected voter_type 'judge', got %v", v["voter_type"])
	}
	if v["notes"] != "Test notes" {
		t.Errorf("expected notes 'Test notes', got %v", v["notes"])
	}
}

func TestCreateVoterFull_WithCarID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// First create a car
	err := repo.CreateCar(ctx, "101", "Test Racer", "Fast Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get car ID (should be 1)
	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	carID := cars[0].ID

	// Create voter with car
	id, err := repo.CreateVoterFull(ctx, &carID, "Car Owner", "", "racer", "CAR-OWNER", "")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Verify car association via ListVoters
	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(voters))
	}

	v := voters[0]
	if v["car_id"] != int64(carID) {
		t.Errorf("expected car_id %d, got %v", carID, v["car_id"])
	}
	if v["car_number"] != "101" {
		t.Errorf("expected car_number '101', got %v", v["car_number"])
	}
}

func TestCreateVoterFull_DBError(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force an error
	repo.db.Close()

	_, err := repo.CreateVoterFull(ctx, nil, "Test", "", "general", "QR", "")
	if err == nil {
		t.Error("expected error when database is closed")
	}
}

func TestUpdateVoter_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create voter
	id, err := repo.CreateVoterFull(ctx, nil, "Original Name", "original@test.com", "general", "UPDATE-QR", "")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}

	// Update voter
	err = repo.UpdateVoter(ctx, int(id), nil, "Updated Name", "updated@test.com", "judge", "Updated notes")
	if err != nil {
		t.Fatalf("UpdateVoter failed: %v", err)
	}

	// Verify update
	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(voters))
	}

	v := voters[0]
	if v["name"] != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %v", v["name"])
	}
	if v["email"] != "updated@test.com" {
		t.Errorf("expected email 'updated@test.com', got %v", v["email"])
	}
	if v["voter_type"] != "judge" {
		t.Errorf("expected voter_type 'judge', got %v", v["voter_type"])
	}
	if v["notes"] != "Updated notes" {
		t.Errorf("expected notes 'Updated notes', got %v", v["notes"])
	}
}

func TestDeleteVoter_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create voter
	id, err := repo.CreateVoter(ctx, "DELETE-QR")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	// Verify voter exists
	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(voters))
	}

	// Delete voter
	err = repo.DeleteVoter(ctx, id)
	if err != nil {
		t.Fatalf("DeleteVoter failed: %v", err)
	}

	// Verify voter is deleted
	voters, err = repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 0 {
		t.Errorf("expected 0 voters after delete, got %d", len(voters))
	}
}

func TestDeleteVoter_NonExistent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Delete non-existent voter should not error (no-op)
	err := repo.DeleteVoter(ctx, 99999)
	if err != nil {
		t.Errorf("DeleteVoter on non-existent should not error, got: %v", err)
	}
}

func TestListVoters_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 0 {
		t.Errorf("expected 0 voters, got %d", len(voters))
	}
}

func TestListVoters_Multiple(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create multiple voters
	for i := 0; i < 5; i++ {
		qr := "LIST-" + string(rune('A'+i))
		_, err := repo.CreateVoter(ctx, qr)
		if err != nil {
			t.Fatalf("CreateVoter failed: %v", err)
		}
	}

	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 5 {
		t.Errorf("expected 5 voters, got %d", len(voters))
	}
}

func TestListVoters_AllFieldCombinations(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a car to associate with some voters
	_ = repo.CreateCar(ctx, "100", "Test Racer", "Test Car", "")
	cars, _ := repo.ListCars(ctx)
	var carID int
	for _, car := range cars {
		if car.CarNumber == "100" {
			carID = car.ID
			break
		}
	}

	// Create voter with minimal fields (no car, name, email, notes)
	_, _ = repo.CreateVoter(ctx, "MIN-QR")

	// Create voter with all fields and associate with car
	voterID2, _ := repo.CreateVoterFull(ctx, &carID, "Full Name", "email@test.com", "general", "FULL-QR", "Some notes")

	// Create category and make one voter vote (to set last_voted_at)
	catID, _ := repo.CreateCategory(ctx, "TestCat", 1, nil, nil, nil)
	_ = repo.SaveVote(ctx, int(voterID2), int(catID), carID)

	// List voters and verify all field combinations are handled
	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}

	// Find the voters in the result
	var minVoter, fullVoter map[string]interface{}
	for _, v := range voters {
		qr := v["qr_code"].(string)
		if qr == "MIN-QR" {
			minVoter = v
		} else if qr == "FULL-QR" {
			fullVoter = v
		}
	}

	// Verify minimal voter (should not have car_id, name, email, notes)
	if minVoter == nil {
		t.Fatal("minimal voter not found")
	}
	if _, hasCarID := minVoter["car_id"]; hasCarID {
		t.Error("expected no car_id for minimal voter")
	}
	if _, hasName := minVoter["name"]; hasName {
		t.Error("expected no name for minimal voter")
	}
	if _, hasEmail := minVoter["email"]; hasEmail {
		t.Error("expected no email for minimal voter")
	}
	if _, hasNotes := minVoter["notes"]; hasNotes {
		t.Error("expected no notes for minimal voter")
	}
	if hasVoted, ok := minVoter["has_voted"].(bool); !ok || hasVoted {
		t.Errorf("expected has_voted=false for minimal voter, got %v", minVoter["has_voted"])
	}

	// Verify full voter (should have all fields including last_voted_at)
	if fullVoter == nil {
		t.Fatal("full voter not found")
	}
	if _, hasCarID := fullVoter["car_id"]; !hasCarID {
		t.Error("expected car_id for full voter")
	}
	if name, ok := fullVoter["name"].(string); !ok || name != "Full Name" {
		t.Errorf("expected name='Full Name', got %v", fullVoter["name"])
	}
	if email, ok := fullVoter["email"].(string); !ok || email != "email@test.com" {
		t.Errorf("expected email='email@test.com', got %v", fullVoter["email"])
	}
	if notes, ok := fullVoter["notes"].(string); !ok || notes != "Some notes" {
		t.Errorf("expected notes='Some notes', got %v", fullVoter["notes"])
	}
	if hasVoted, ok := fullVoter["has_voted"].(bool); !ok || !hasVoted {
		t.Errorf("expected has_voted=true for full voter, got %v", fullVoter["has_voted"])
	}
	if _, hasLastVoted := fullVoter["last_voted_at"]; !hasLastVoted {
		t.Error("expected last_voted_at for voter who has voted")
	}
}

func TestGetVoterQRCode_Existing(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, err := repo.CreateVoter(ctx, "GETQR-TEST")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	qrCode, err := repo.GetVoterQRCode(ctx, id)
	if err != nil {
		t.Fatalf("GetVoterQRCode failed: %v", err)
	}
	if qrCode != "GETQR-TEST" {
		t.Errorf("expected qr_code 'GETQR-TEST', got %q", qrCode)
	}
}

func TestGetVoterQRCode_NonExistent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetVoterQRCode(ctx, 99999)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetVoterByQRCode_Existing(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	expectedID, err := repo.CreateVoter(ctx, "BYQRCODE-1")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	id, exists, err := repo.GetVoterByQRCode(ctx, "BYQRCODE-1")
	if err != nil {
		t.Fatalf("GetVoterByQRCode failed: %v", err)
	}
	if !exists {
		t.Error("expected voter to exist")
	}
	if int(id) != expectedID {
		t.Errorf("expected ID %d, got %d", expectedID, id)
	}
}

func TestGetVoterByQRCode_NonExistent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, exists, err := repo.GetVoterByQRCode(ctx, "NON-EXISTENT")
	if err != nil {
		t.Fatalf("GetVoterByQRCode failed: %v", err)
	}
	if exists {
		t.Error("expected voter to not exist")
	}
}

// ==================== Category Tests ====================

func TestListCategories_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(categories))
	}
}

func TestCreateCategory_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, err := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Verify category exists
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].Name != "Best Design" {
		t.Errorf("expected name 'Best Design', got %q", categories[0].Name)
	}
	if categories[0].DisplayOrder != 1 {
		t.Errorf("expected display_order 1, got %d", categories[0].DisplayOrder)
	}
}

func TestCreateCategory_WithGroup(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a group first
	groupID, err := repo.CreateCategoryGroup(ctx, "Main Awards", "Primary awards", nil, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	gID := int(groupID)
	id, err := repo.CreateCategory(ctx, "Best Speed", 1, &gID, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Verify category has group
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].GroupID == nil || *categories[0].GroupID != gID {
		t.Errorf("expected group_id %d, got %v", gID, categories[0].GroupID)
	}
	if categories[0].GroupName != "Main Awards" {
		t.Errorf("expected group_name 'Main Awards', got %q", categories[0].GroupName)
	}
}

func TestCreateCategory_DBError(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force an error
	repo.db.Close()

	_, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err == nil {
		t.Error("expected error when database is closed")
	}
}

func TestUpdateCategory_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, err := repo.CreateCategory(ctx, "Original Name", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	err = repo.UpdateCategory(ctx, int(id), "Updated Name", 2, nil, nil, nil, true)
	if err != nil {
		t.Fatalf("UpdateCategory failed: %v", err)
	}

	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %q", categories[0].Name)
	}
	if categories[0].DisplayOrder != 2 {
		t.Errorf("expected display_order 2, got %d", categories[0].DisplayOrder)
	}
}

func TestDeleteCategory_SoftDelete(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, err := repo.CreateCategory(ctx, "To Delete", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Verify exists in active list
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	// Soft delete
	err = repo.DeleteCategory(ctx, int(id))
	if err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}

	// Should not appear in active list
	categories, err = repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("expected 0 active categories after soft delete, got %d", len(categories))
	}

	// But should appear in all categories list
	allCategories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}
	if len(allCategories) != 1 {
		t.Fatalf("expected 1 category in all list, got %d", len(allCategories))
	}
	if allCategories[0]["active"] != false {
		t.Error("expected active to be false after soft delete")
	}
}

func TestCategoryExists_True(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.CreateCategory(ctx, "Existing Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	exists, err := repo.CategoryExists(ctx, "Existing Category")
	if err != nil {
		t.Fatalf("CategoryExists failed: %v", err)
	}
	if !exists {
		t.Error("expected category to exist")
	}
}

func TestCategoryExists_False(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	exists, err := repo.CategoryExists(ctx, "Non-Existent Category")
	if err != nil {
		t.Fatalf("CategoryExists failed: %v", err)
	}
	if exists {
		t.Error("expected category to not exist")
	}
}

func TestListAllCategories_IncludesInactive(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create two categories
	id1, _ := repo.CreateCategory(ctx, "Category 1", 1, nil, nil, nil)
	_, _ = repo.CreateCategory(ctx, "Category 2", 2, nil, nil, nil)

	// Soft delete one
	_ = repo.DeleteCategory(ctx, int(id1))

	// ListAllCategories should return both
	allCategories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}
	if len(allCategories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(allCategories))
	}
}

// ==================== Category Group Tests ====================

func TestListCategoryGroups_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	groups, err := repo.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatalf("ListCategoryGroups failed: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

func TestCreateCategoryGroup_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, err := repo.CreateCategoryGroup(ctx, "Design Awards", "Categories for design", nil, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	groups, err := repo.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatalf("ListCategoryGroups failed: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "Design Awards" {
		t.Errorf("expected name 'Design Awards', got %q", groups[0].Name)
	}
	if groups[0].Description != "Categories for design" {
		t.Errorf("expected description 'Categories for design', got %q", groups[0].Description)
	}
}

func TestCreateCategoryGroup_WithExclusivityPool(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	poolID := 42
	id, err := repo.CreateCategoryGroup(ctx, "Exclusive Group", "", &poolID, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	group, err := repo.GetCategoryGroup(ctx, string(rune('0'+id)))
	if err != nil {
		t.Fatalf("GetCategoryGroup failed: %v", err)
	}
	if group == nil {
		t.Fatal("expected group to exist")
	}
	if group.ExclusivityPoolID == nil || *group.ExclusivityPoolID != poolID {
		t.Errorf("expected exclusivity_pool_id %d, got %v", poolID, group.ExclusivityPoolID)
	}
}

func TestCreateCategoryGroup_DBError(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force an error
	repo.db.Close()

	_, err := repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, nil, 1)
	if err == nil {
		t.Error("expected error when database is closed")
	}
}

func TestGetCategoryGroup_Existing(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, err := repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	group, err := repo.GetCategoryGroup(ctx, "1")
	if err != nil {
		t.Fatalf("GetCategoryGroup failed: %v", err)
	}
	if group == nil {
		t.Fatal("expected group to exist")
	}
	if int64(group.ID) != id {
		t.Errorf("expected ID %d, got %d", id, group.ID)
	}
}

func TestGetCategoryGroup_NonExistent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	group, err := repo.GetCategoryGroup(ctx, "99999")
	var appErr *errors.Error
	if !stderrors.As(err, &appErr) || appErr.Kind != errors.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-existent group, got: %v", err)
	}
	if group != nil {
		t.Error("expected nil group when not found")
	}
}

func TestUpdateCategoryGroup_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.CreateCategoryGroup(ctx, "Original", "Original desc", nil, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	err = repo.UpdateCategoryGroup(ctx, "1", "Updated", "Updated desc", nil, nil, 2)
	if err != nil {
		t.Fatalf("UpdateCategoryGroup failed: %v", err)
	}

	group, err := repo.GetCategoryGroup(ctx, "1")
	if err != nil {
		t.Fatalf("GetCategoryGroup failed: %v", err)
	}
	if group.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", group.Name)
	}
	if group.Description != "Updated desc" {
		t.Errorf("expected description 'Updated desc', got %q", group.Description)
	}
}

func TestDeleteCategoryGroup_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.CreateCategoryGroup(ctx, "To Delete", "", nil, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	err = repo.DeleteCategoryGroup(ctx, "1")
	if err != nil {
		t.Fatalf("DeleteCategoryGroup failed: %v", err)
	}

	group, err := repo.GetCategoryGroup(ctx, "1")
	var appErr *errors.Error
	if !stderrors.As(err, &appErr) || appErr.Kind != errors.ErrNotFound {
		t.Fatalf("expected ErrNotFound for deleted group, got: %v", err)
	}
	if group != nil {
		t.Error("expected nil group after deletion")
	}
}

// ==================== Car Tests ====================

func TestListCars_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 0 {
		t.Errorf("expected 0 cars, got %d", len(cars))
	}
}

func TestCreateCar_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.CreateCar(ctx, "42", "John Doe", "Speed Demon", "http://example.com/photo.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	if cars[0].CarNumber != "42" {
		t.Errorf("expected car_number '42', got %q", cars[0].CarNumber)
	}
	if cars[0].RacerName != "John Doe" {
		t.Errorf("expected racer_name 'John Doe', got %q", cars[0].RacerName)
	}
	if cars[0].CarName != "Speed Demon" {
		t.Errorf("expected car_name 'Speed Demon', got %q", cars[0].CarName)
	}
}

func TestCarExists_True(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.CreateCar(ctx, "99", "Racer", "Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	exists, err := repo.CarExists(ctx, "99")
	if err != nil {
		t.Fatalf("CarExists failed: %v", err)
	}
	if !exists {
		t.Error("expected car to exist")
	}
}

func TestCarExists_False(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	exists, err := repo.CarExists(ctx, "999")
	if err != nil {
		t.Fatalf("CarExists failed: %v", err)
	}
	if exists {
		t.Error("expected car to not exist")
	}
}

func TestUpsertCar_Insert(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.UpsertCar(ctx, 1001, "100", "Racer One", "Fast Car", "http://photo.com/1.jpg", "")
	if err != nil {
		t.Fatalf("UpsertCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	if cars[0].CarNumber != "100" {
		t.Errorf("expected car_number '100', got %q", cars[0].CarNumber)
	}
}

func TestUpsertCar_Update(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Insert
	err := repo.UpsertCar(ctx, 2001, "200", "Original Name", "Original Car", "", "")
	if err != nil {
		t.Fatalf("UpsertCar insert failed: %v", err)
	}

	// Update with same derbynet_racer_id
	err = repo.UpsertCar(ctx, 2001, "200", "Updated Name", "Updated Car", "http://new.jpg", "")
	if err != nil {
		t.Fatalf("UpsertCar update failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car (upsert should update), got %d", len(cars))
	}
	if cars[0].RacerName != "Updated Name" {
		t.Errorf("expected racer_name 'Updated Name', got %q", cars[0].RacerName)
	}
	if cars[0].CarName != "Updated Car" {
		t.Errorf("expected car_name 'Updated Car', got %q", cars[0].CarName)
	}
}

func TestGetCarByDerbyNetID_Existing(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.UpsertCar(ctx, 3001, "300", "Test Racer", "Test Car", "", "")
	if err != nil {
		t.Fatalf("UpsertCar failed: %v", err)
	}

	id, exists, err := repo.GetCarByDerbyNetID(ctx, 3001)
	if err != nil {
		t.Fatalf("GetCarByDerbyNetID failed: %v", err)
	}
	if !exists {
		t.Error("expected car to exist")
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

func TestGetCarByDerbyNetID_NonExistent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, exists, err := repo.GetCarByDerbyNetID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetCarByDerbyNetID failed: %v", err)
	}
	if exists {
		t.Error("expected car to not exist")
	}
}

// ==================== Vote Tests ====================

func TestSaveVote_NewVote(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create dependencies
	voterID, _ := repo.CreateVoter(ctx, "VOTE-QR1")
	categoryID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer", "Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Save vote
	err := repo.SaveVote(ctx, voterID, int(categoryID), carID)
	if err != nil {
		t.Fatalf("SaveVote failed: %v", err)
	}

	// Verify vote
	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}
	if len(votes) != 1 {
		t.Fatalf("expected 1 vote, got %d", len(votes))
	}
	if votes[int(categoryID)] != carID {
		t.Errorf("expected car_id %d for category %d, got %d", carID, categoryID, votes[int(categoryID)])
	}
}

func TestSaveVote_UpdateVote(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create dependencies
	voterID, _ := repo.CreateVoter(ctx, "VOTE-QR2")
	categoryID, _ := repo.CreateCategory(ctx, "Best Speed", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "")
	_ = repo.CreateCar(ctx, "2", "Racer 2", "Car 2", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Save initial vote
	err := repo.SaveVote(ctx, voterID, int(categoryID), car1ID)
	if err != nil {
		t.Fatalf("SaveVote failed: %v", err)
	}

	// Update vote to different car
	err = repo.SaveVote(ctx, voterID, int(categoryID), car2ID)
	if err != nil {
		t.Fatalf("SaveVote update failed: %v", err)
	}

	// Verify vote updated
	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}
	if len(votes) != 1 {
		t.Fatalf("expected 1 vote (updated, not added), got %d", len(votes))
	}
	if votes[int(categoryID)] != car2ID {
		t.Errorf("expected car_id %d after update, got %d", car2ID, votes[int(categoryID)])
	}
}

func TestSaveVote_DeleteVote(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create dependencies
	voterID, _ := repo.CreateVoter(ctx, "VOTE-QR3")
	categoryID, _ := repo.CreateCategory(ctx, "Best Color", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer", "Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Save vote
	_ = repo.SaveVote(ctx, voterID, int(categoryID), carID)

	// Delete vote by setting carID to 0
	err := repo.SaveVote(ctx, voterID, int(categoryID), 0)
	if err != nil {
		t.Fatalf("SaveVote delete failed: %v", err)
	}

	// Verify vote deleted
	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}
	if len(votes) != 0 {
		t.Errorf("expected 0 votes after delete, got %d", len(votes))
	}
}

func TestGetVoterVotes_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	voterID, _ := repo.CreateVoter(ctx, "NOVOTES-QR")

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}
	if len(votes) != 0 {
		t.Errorf("expected 0 votes, got %d", len(votes))
	}
}

func TestGetVoterVotes_MultipleCategories(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create dependencies
	voterID, _ := repo.CreateVoter(ctx, "MULTI-QR")
	cat1ID, _ := repo.CreateCategory(ctx, "Category 1", 1, nil, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Category 2", 2, nil, nil, nil)
	cat3ID, _ := repo.CreateCategory(ctx, "Category 3", 3, nil, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer", "Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Vote in multiple categories
	_ = repo.SaveVote(ctx, voterID, int(cat1ID), carID)
	_ = repo.SaveVote(ctx, voterID, int(cat2ID), carID)
	_ = repo.SaveVote(ctx, voterID, int(cat3ID), carID)

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}
	if len(votes) != 3 {
		t.Errorf("expected 3 votes, got %d", len(votes))
	}
}

func TestGetVoteResults_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	results, err := repo.GetVoteResults(ctx)
	if err != nil {
		t.Fatalf("GetVoteResults failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d categories", len(results))
	}
}

func TestGetVoteResults_WithVotes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create dependencies
	categoryID, _ := repo.CreateCategory(ctx, "Popular Vote", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "")
	_ = repo.CreateCar(ctx, "2", "Racer 2", "Car 2", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Create voters and votes
	for i := 0; i < 3; i++ {
		voterID, _ := repo.CreateVoter(ctx, "RESULTS-"+string(rune('A'+i)))
		_ = repo.SaveVote(ctx, voterID, int(categoryID), car1ID)
	}
	for i := 0; i < 2; i++ {
		voterID, _ := repo.CreateVoter(ctx, "RESULTS-"+string(rune('D'+i)))
		_ = repo.SaveVote(ctx, voterID, int(categoryID), car2ID)
	}

	results, err := repo.GetVoteResults(ctx)
	if err != nil {
		t.Fatalf("GetVoteResults failed: %v", err)
	}

	catResults := results[int(categoryID)]
	if catResults == nil {
		t.Fatal("expected results for category")
	}
	if catResults[car1ID] != 3 {
		t.Errorf("expected 3 votes for car1, got %d", catResults[car1ID])
	}
	if catResults[car2ID] != 2 {
		t.Errorf("expected 2 votes for car2, got %d", catResults[car2ID])
	}
}

func TestGetVoteResultsWithCars_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	results, err := repo.GetVoteResultsWithCars(ctx)
	if err != nil {
		t.Fatalf("GetVoteResultsWithCars failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d rows", len(results))
	}
}

func TestGetVoteResultsWithCars_WithVotes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create dependencies
	categoryID, _ := repo.CreateCategory(ctx, "Best in Show", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "42", "Winner Racer", "Champion Car", "http://photo.jpg")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	voterID, _ := repo.CreateVoter(ctx, "WITHCARS-QR")
	_ = repo.SaveVote(ctx, voterID, int(categoryID), carID)

	results, err := repo.GetVoteResultsWithCars(ctx)
	if err != nil {
		t.Fatalf("GetVoteResultsWithCars failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	row := results[0]
	if row.CategoryID != int(categoryID) {
		t.Errorf("expected category_id %d, got %d", categoryID, row.CategoryID)
	}
	if row.CarNumber != "42" {
		t.Errorf("expected car_number '42', got %q", row.CarNumber)
	}
	if row.RacerName != "Winner Racer" {
		t.Errorf("expected racer_name 'Winner Racer', got %q", row.RacerName)
	}
	if row.VoteCount != 1 {
		t.Errorf("expected vote_count 1, got %d", row.VoteCount)
	}
}

// ==================== Settings Tests ====================

func TestGetSetting_DefaultValues(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Default settings are inserted during migration
	votingOpen, err := repo.GetSetting(ctx, "voting_open")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if votingOpen != "true" {
		t.Errorf("expected default voting_open 'true', got %q", votingOpen)
	}

	// base_url is NOT set by default in repository - it's set by app.go
	// with the detected LAN IP address on startup
	baseURL, _ := repo.GetSetting(ctx, "base_url")
	if baseURL != "" {
		t.Errorf("expected base_url to be empty by default, got %q", baseURL)
	}
}

func TestSetSetting_NewValue(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.SetSetting(ctx, "custom_key", "custom_value")
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	value, err := repo.GetSetting(ctx, "custom_key")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if value != "custom_value" {
		t.Errorf("expected 'custom_value', got %q", value)
	}
}

func TestSetSetting_UpdateExisting(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Update default setting
	err := repo.SetSetting(ctx, "voting_open", "false")
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	value, err := repo.GetSetting(ctx, "voting_open")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if value != "false" {
		t.Errorf("expected 'false', got %q", value)
	}
}

func TestGetSetting_NonExistent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetSetting(ctx, "non_existent_key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for non-existent key, got %v", err)
	}
}

// ==================== Exclusivity Pool Tests ====================

func TestGetExclusivityPoolID_NoPool(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	categoryID, _ := repo.CreateCategory(ctx, "No Pool Category", 1, nil, nil, nil)

	_, hasPool, err := repo.GetExclusivityPoolID(ctx, int(categoryID))
	if err != nil {
		t.Fatalf("GetExclusivityPoolID failed: %v", err)
	}
	if hasPool {
		t.Error("expected no exclusivity pool")
	}
}

func TestGetExclusivityPoolID_WithPool(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	poolID := 1
	groupID, _ := repo.CreateCategoryGroup(ctx, "Exclusive Group", "", &poolID, nil, 1)
	gID := int(groupID)
	categoryID, _ := repo.CreateCategory(ctx, "Pooled Category", 1, &gID, nil, nil)

	pool, hasPool, err := repo.GetExclusivityPoolID(ctx, int(categoryID))
	if err != nil {
		t.Fatalf("GetExclusivityPoolID failed: %v", err)
	}
	if !hasPool {
		t.Error("expected exclusivity pool")
	}
	if int(pool) != poolID {
		t.Errorf("expected pool_id %d, got %d", poolID, pool)
	}
}

func TestFindConflictingVote_NoConflict(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	voterID, _ := repo.CreateVoter(ctx, "CONFLICT-QR1")
	poolID := 1
	groupID, _ := repo.CreateCategoryGroup(ctx, "Exclusive", "", &poolID, nil, 1)
	gID := int(groupID)
	categoryID, _ := repo.CreateCategory(ctx, "Cat 1", 1, &gID, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer", "Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	_, _, found, err := repo.FindConflictingVote(ctx, voterID, carID, int(categoryID), int64(poolID))
	if err != nil {
		t.Fatalf("FindConflictingVote failed: %v", err)
	}
	if found {
		t.Error("expected no conflict")
	}
}

func TestFindConflictingVote_WithConflict(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	voterID, _ := repo.CreateVoter(ctx, "CONFLICT-QR2")
	poolID := 1
	groupID, _ := repo.CreateCategoryGroup(ctx, "Exclusive", "", &poolID, nil, 1)
	gID := int(groupID)
	cat1ID, _ := repo.CreateCategory(ctx, "Cat 1", 1, &gID, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Cat 2", 2, &gID, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer", "Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Vote for car in cat1
	_ = repo.SaveVote(ctx, voterID, int(cat1ID), carID)

	// Try to vote for same car in cat2 (should find conflict)
	conflictCatID, conflictCatName, found, err := repo.FindConflictingVote(ctx, voterID, carID, int(cat2ID), int64(poolID))
	if err != nil {
		t.Fatalf("FindConflictingVote failed: %v", err)
	}
	if !found {
		t.Fatal("expected to find conflict")
	}
	if conflictCatID != int(cat1ID) {
		t.Errorf("expected conflict in category %d, got %d", cat1ID, conflictCatID)
	}
	if conflictCatName != "Cat 1" {
		t.Errorf("expected conflict category name 'Cat 1', got %q", conflictCatName)
	}
}

func TestClearConflictingVote_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	voterID, _ := repo.CreateVoter(ctx, "CLEAR-QR")
	categoryID, _ := repo.CreateCategory(ctx, "Clear Test", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer", "Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	_ = repo.SaveVote(ctx, voterID, int(categoryID), carID)

	// Clear the vote
	err := repo.ClearConflictingVote(ctx, voterID, int(categoryID), carID)
	if err != nil {
		t.Fatalf("ClearConflictingVote failed: %v", err)
	}

	// Verify vote is cleared
	votes, _ := repo.GetVoterVotes(ctx, voterID)
	if len(votes) != 0 {
		t.Errorf("expected 0 votes after clear, got %d", len(votes))
	}
}

// ==================== Stats Tests ====================

func TestGetVotingStats_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	stats, err := repo.GetVotingStats(ctx)
	if err != nil {
		t.Fatalf("GetVotingStats failed: %v", err)
	}

	if stats["total_voters"] != 0 {
		t.Errorf("expected 0 total_voters, got %v", stats["total_voters"])
	}
	if stats["voters_who_voted"] != 0 {
		t.Errorf("expected 0 voters_who_voted, got %v", stats["voters_who_voted"])
	}
	if stats["total_votes"] != 0 {
		t.Errorf("expected 0 total_votes, got %v", stats["total_votes"])
	}
}

func TestGetVotingStats_WithData(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create data
	categoryID, _ := repo.CreateCategory(ctx, "Stats Category", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer", "Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Create 3 voters, 2 who vote
	voter1ID, _ := repo.CreateVoter(ctx, "STATS-QR1")
	voter2ID, _ := repo.CreateVoter(ctx, "STATS-QR2")
	_, _ = repo.CreateVoter(ctx, "STATS-QR3") // doesn't vote

	_ = repo.SaveVote(ctx, voter1ID, int(categoryID), carID)
	_ = repo.SaveVote(ctx, voter2ID, int(categoryID), carID)

	stats, err := repo.GetVotingStats(ctx)
	if err != nil {
		t.Fatalf("GetVotingStats failed: %v", err)
	}

	if stats["total_voters"] != 3 {
		t.Errorf("expected 3 total_voters, got %v", stats["total_voters"])
	}
	if stats["voters_who_voted"] != 2 {
		t.Errorf("expected 2 voters_who_voted, got %v", stats["voters_who_voted"])
	}
	if stats["total_votes"] != 2 {
		t.Errorf("expected 2 total_votes, got %v", stats["total_votes"])
	}
	if stats["total_categories"] != 1 {
		t.Errorf("expected 1 total_categories, got %v", stats["total_categories"])
	}
	if stats["total_cars"] != 1 {
		t.Errorf("expected 1 total_cars, got %v", stats["total_cars"])
	}
}

// ==================== Database Management Tests ====================

func TestClearTable_Voters(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create some voters
	_, _ = repo.CreateVoter(ctx, "CLEAR-1")
	_, _ = repo.CreateVoter(ctx, "CLEAR-2")

	voters, _ := repo.ListVoters(ctx)
	if len(voters) != 2 {
		t.Fatalf("expected 2 voters before clear, got %d", len(voters))
	}

	// Clear table
	err := repo.ClearTable(ctx, "voters")
	if err != nil {
		t.Fatalf("ClearTable failed: %v", err)
	}

	voters, _ = repo.ListVoters(ctx)
	if len(voters) != 0 {
		t.Errorf("expected 0 voters after clear, got %d", len(voters))
	}
}

func TestInsertVoterIgnore_New(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.InsertVoterIgnore(ctx, "IGNORE-QR1")
	if err != nil {
		t.Fatalf("InsertVoterIgnore failed: %v", err)
	}

	voters, _ := repo.ListVoters(ctx)
	if len(voters) != 1 {
		t.Errorf("expected 1 voter, got %d", len(voters))
	}
}

func TestInsertVoterIgnore_Duplicate(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_ = repo.InsertVoterIgnore(ctx, "IGNORE-QR2")

	// Insert duplicate - should not error
	err := repo.InsertVoterIgnore(ctx, "IGNORE-QR2")
	if err != nil {
		t.Fatalf("InsertVoterIgnore on duplicate should not error, got: %v", err)
	}

	// Should still only have 1 voter
	voters, _ := repo.ListVoters(ctx)
	if len(voters) != 1 {
		t.Errorf("expected 1 voter (duplicate ignored), got %d", len(voters))
	}
}

func TestUpsertVoterForCar_New(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a car first
	err := repo.UpsertCar(ctx, 4001, "400", "Test Racer", "Test Car", "", "")
	if err != nil {
		t.Fatalf("UpsertCar failed: %v", err)
	}
	carID, _, _ := repo.GetCarByDerbyNetID(ctx, 4001)

	// Create voter for car
	err = repo.UpsertVoterForCar(ctx, carID, "Car Owner", "CAR-OWNER-QR")
	if err != nil {
		t.Fatalf("UpsertVoterForCar failed: %v", err)
	}

	voters, _ := repo.ListVoters(ctx)
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(voters))
	}
	if voters[0]["voter_type"] != "racer" {
		t.Errorf("expected voter_type 'racer', got %v", voters[0]["voter_type"])
	}
}

func TestUpsertVoterForCar_Update(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a car
	err := repo.UpsertCar(ctx, 5001, "500", "Original Racer", "Test Car", "", "")
	if err != nil {
		t.Fatalf("UpsertCar failed: %v", err)
	}
	carID, _, _ := repo.GetCarByDerbyNetID(ctx, 5001)

	// Create voter
	_ = repo.UpsertVoterForCar(ctx, carID, "Original Name", "UPSERT-VOTER-QR")

	// Update voter
	err = repo.UpsertVoterForCar(ctx, carID, "Updated Name", "UPSERT-VOTER-QR")
	if err != nil {
		t.Fatalf("UpsertVoterForCar update failed: %v", err)
	}

	voters, _ := repo.ListVoters(ctx)
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter (upsert should update), got %d", len(voters))
	}
	if voters[0]["name"] != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %v", voters[0]["name"])
	}
}

// ==================== DB Method Test ====================

func TestDB_ReturnsConnection(t *testing.T) {
	repo := newTestRepo(t)

	db := repo.DB()
	if db == nil {
		t.Error("expected non-nil database connection")
	}

	// Verify connection is usable
	err := db.Ping()
	if err != nil {
		t.Errorf("database ping failed: %v", err)
	}
}

// ==================== Additional Coverage Tests ====================

func TestListCategoryGroups_Multiple(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create multiple groups with different properties
	poolID := 1
	_, _ = repo.CreateCategoryGroup(ctx, "Group 1", "Description 1", nil, nil, 1)
	_, _ = repo.CreateCategoryGroup(ctx, "Group 2", "Description 2", &poolID, nil, 2)
	_, _ = repo.CreateCategoryGroup(ctx, "Group 3", "", nil, nil, 3)

	groups, err := repo.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatalf("ListCategoryGroups failed: %v", err)
	}
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// Verify ordering by display_order
	if groups[0].Name != "Group 1" {
		t.Errorf("expected first group 'Group 1', got %q", groups[0].Name)
	}
	if groups[1].Name != "Group 2" {
		t.Errorf("expected second group 'Group 2', got %q", groups[1].Name)
	}
	if groups[2].Name != "Group 3" {
		t.Errorf("expected third group 'Group 3', got %q", groups[2].Name)
	}

	// Verify exclusivity pool
	if groups[1].ExclusivityPoolID == nil || *groups[1].ExclusivityPoolID != poolID {
		t.Errorf("expected group 2 to have exclusivity pool %d", poolID)
	}
	if groups[0].ExclusivityPoolID != nil {
		t.Errorf("expected group 1 to have no exclusivity pool")
	}
}

func TestListCategories_OrderedByDisplayOrder(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create categories out of order
	_, _ = repo.CreateCategory(ctx, "Category C", 3, nil, nil, nil)
	_, _ = repo.CreateCategory(ctx, "Category A", 1, nil, nil, nil)
	_, _ = repo.CreateCategory(ctx, "Category B", 2, nil, nil, nil)

	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(categories))
	}

	// Verify ordering
	if categories[0].Name != "Category A" {
		t.Errorf("expected first category 'Category A', got %q", categories[0].Name)
	}
	if categories[1].Name != "Category B" {
		t.Errorf("expected second category 'Category B', got %q", categories[1].Name)
	}
	if categories[2].Name != "Category C" {
		t.Errorf("expected third category 'Category C', got %q", categories[2].Name)
	}
}

func TestListCategories_AllOptionalFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category group with exclusivity pool
	poolID := 1
	groupID, _ := repo.CreateCategoryGroup(ctx, "TestGroup", "Test Description", &poolID, nil, 1)

	// Create category without optional fields
	_, _ = repo.CreateCategory(ctx, "MinimalCat", 1, nil, nil, nil)

	// Create category with all optional fields
	derbynetAwardID := 100
	groupIDInt := int(groupID)
	catID2, _ := repo.CreateCategory(ctx, "FullCat", 2, &groupIDInt, nil, nil)

	// Set derbynet_award_id via direct DB update
	_, _ = repo.db.ExecContext(ctx, "UPDATE categories SET derbynet_award_id = ? WHERE id = ?", derbynetAwardID, catID2)

	// List categories and verify optional fields are handled
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}

	// Find our categories
	var minCat, fullCat *models.Category
	for i := range categories {
		if categories[i].Name == "MinimalCat" {
			minCat = &categories[i]
		} else if categories[i].Name == "FullCat" {
			fullCat = &categories[i]
		}
	}

	// Verify minimal category has no optional fields
	if minCat == nil {
		t.Fatal("minimal category not found")
	}
	if minCat.GroupID != nil {
		t.Error("expected no GroupID for minimal category")
	}
	if minCat.DerbyNetAwardID != nil {
		t.Error("expected no DerbyNetAwardID for minimal category")
	}
	if minCat.ExclusivityPoolID != nil {
		t.Error("expected no ExclusivityPoolID for minimal category")
	}

	// Verify full category has all optional fields
	if fullCat == nil {
		t.Fatal("full category not found")
	}
	if fullCat.GroupID == nil || *fullCat.GroupID != groupIDInt {
		t.Errorf("expected GroupID=%d, got %v", groupIDInt, fullCat.GroupID)
	}
	if fullCat.DerbyNetAwardID == nil || *fullCat.DerbyNetAwardID != derbynetAwardID {
		t.Errorf("expected DerbyNetAwardID=%d, got %v", derbynetAwardID, fullCat.DerbyNetAwardID)
	}
	if fullCat.ExclusivityPoolID == nil || *fullCat.ExclusivityPoolID != poolID {
		t.Errorf("expected ExclusivityPoolID=%d, got %v", poolID, fullCat.ExclusivityPoolID)
	}
	if fullCat.GroupName != "TestGroup" {
		t.Errorf("expected GroupName='TestGroup', got %q", fullCat.GroupName)
	}
}

func TestListAllCategories_WithGroupInfo(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create group and category with group
	groupID, _ := repo.CreateCategoryGroup(ctx, "Test Group", "Group Description", nil, nil, 1)
	gID := int(groupID)
	_, _ = repo.CreateCategory(ctx, "Grouped Category", 1, &gID, nil, nil)
	_, _ = repo.CreateCategory(ctx, "Ungrouped Category", 2, nil, nil, nil)

	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}

	// Check that group info is included
	for _, cat := range categories {
		if cat["name"] == "Grouped Category" {
			if cat["group_name"] != "Test Group" {
				t.Errorf("expected group_name 'Test Group', got %v", cat["group_name"])
			}
		}
	}
}

func TestListAllCategories_AllOptionalFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category without optional fields
	_, _ = repo.CreateCategory(ctx, "MinimalCat", 1, nil, nil, nil)

	// Create category group
	groupID, _ := repo.CreateCategoryGroup(ctx, "TestGroup", "Test Description", nil, nil, 1)

	// Create category with all optional fields
	derbynetAwardID := 200
	gID := int(groupID)
	catID2, _ := repo.CreateCategory(ctx, "FullCat", 2, &gID, nil, nil)

	// Set derbynet_award_id via update
	_, _ = repo.db.ExecContext(ctx, "UPDATE categories SET derbynet_award_id = ? WHERE id = ?", derbynetAwardID, catID2)

	// List all categories and verify optional fields are handled
	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	// Find our categories
	var minCat, fullCat map[string]interface{}
	for _, cat := range categories {
		if cat["name"] == "MinimalCat" {
			minCat = cat
		} else if cat["name"] == "FullCat" {
			fullCat = cat
		}
	}

	// Verify minimal category has no optional fields
	if minCat == nil {
		t.Fatal("minimal category not found")
	}
	if _, hasGroupID := minCat["group_id"]; hasGroupID {
		t.Error("expected no group_id for minimal category")
	}
	if _, hasDerbynetAwardID := minCat["derbynet_award_id"]; hasDerbynetAwardID {
		t.Error("expected no derbynet_award_id for minimal category")
	}

	// Verify full category has all optional fields
	if fullCat == nil {
		t.Fatal("full category not found")
	}
	if groupIDVal, ok := fullCat["group_id"].(int); !ok || groupIDVal != gID {
		t.Errorf("expected group_id=%d, got %v", gID, fullCat["group_id"])
	}
	if derbynetVal, ok := fullCat["derbynet_award_id"].(int); !ok || derbynetVal != derbynetAwardID {
		t.Errorf("expected derbynet_award_id=%d, got %v", derbynetAwardID, fullCat["derbynet_award_id"])
	}
	if groupNameVal, ok := fullCat["group_name"].(string); !ok || groupNameVal != "TestGroup" {
		t.Errorf("expected group_name='TestGroup', got %v", fullCat["group_name"])
	}
}

func TestListVoters_WithCarInfo(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create car and voter with car
	err := repo.CreateCar(ctx, "123", "Car Racer", "Speedy Car", "http://photo.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	_, err = repo.CreateVoterFull(ctx, &carID, "Voter Name", "voter@test.com", "racer", "CARVOTER-QR", "")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}

	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(voters))
	}

	v := voters[0]
	if v["car_number"] != "123" {
		t.Errorf("expected car_number '123', got %v", v["car_number"])
	}
	if v["racer_name"] != "Car Racer" {
		t.Errorf("expected racer_name 'Car Racer', got %v", v["racer_name"])
	}
}

func TestListCars_Multiple(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create multiple cars
	_ = repo.CreateCar(ctx, "1", "Racer A", "Car A", "http://a.jpg")
	_ = repo.CreateCar(ctx, "2", "Racer B", "Car B", "")
	_ = repo.CreateCar(ctx, "3", "Racer C", "Car C", "http://c.jpg")

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 3 {
		t.Fatalf("expected 3 cars, got %d", len(cars))
	}

	// Verify car data
	for _, car := range cars {
		if car.CarNumber == "" {
			t.Error("expected non-empty car number")
		}
		if car.RacerName == "" {
			t.Error("expected non-empty racer name")
		}
		if car.CarName == "" {
			t.Error("expected non-empty car name")
		}
	}
}

func TestGetVoteResultsWithCars_MultipleCategories(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create multiple categories and cars
	cat1ID, _ := repo.CreateCategory(ctx, "Category 1", 1, nil, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Category 2", 2, nil, nil, nil)
	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "http://1.jpg")
	_ = repo.CreateCar(ctx, "2", "Racer 2", "Car 2", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Create votes
	voter1, _ := repo.CreateVoter(ctx, "MULTICATRES-1")
	voter2, _ := repo.CreateVoter(ctx, "MULTICATRES-2")
	_ = repo.SaveVote(ctx, voter1, int(cat1ID), car1ID)
	_ = repo.SaveVote(ctx, voter2, int(cat1ID), car1ID)
	_ = repo.SaveVote(ctx, voter1, int(cat2ID), car2ID)

	results, err := repo.GetVoteResultsWithCars(ctx)
	if err != nil {
		t.Fatalf("GetVoteResultsWithCars failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 result rows, got %d", len(results))
	}

	// Verify results include all expected fields
	for _, row := range results {
		if row.CategoryID == 0 {
			t.Error("expected non-zero category_id")
		}
		if row.CarID == 0 {
			t.Error("expected non-zero car_id")
		}
		if row.VoteCount == 0 {
			t.Error("expected non-zero vote_count")
		}
	}
}

func TestGetExclusivityPoolID_CategoryNotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Non-existent category should return sql.ErrNoRows
	_, _, err := repo.GetExclusivityPoolID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent category")
	}
}

func TestUpdateVoter_WithCarID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create car and voter
	_ = repo.CreateCar(ctx, "999", "Test Racer", "Test Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	voterID, _ := repo.CreateVoterFull(ctx, nil, "Original", "", "general", "UPVOTER-QR", "")

	// Update voter with car
	err := repo.UpdateVoter(ctx, int(voterID), &carID, "Updated", "test@test.com", "racer", "notes")
	if err != nil {
		t.Fatalf("UpdateVoter failed: %v", err)
	}

	voters, _ := repo.ListVoters(ctx)
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(voters))
	}

	v := voters[0]
	if v["car_id"] != int64(carID) {
		t.Errorf("expected car_id %d, got %v", carID, v["car_id"])
	}
}

// ==================== UpsertCategory Tests ====================

func TestUpsertCategory_NewCategoryWithAwardID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	awardID := 42
	created, err := repo.UpsertCategory(ctx, "New Award", 1, &awardID)
	if err != nil {
		t.Fatalf("UpsertCategory failed: %v", err)
	}
	if !created {
		t.Error("expected category to be created")
	}

	// Verify category exists with award ID
	categories, _ := repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].Name != "New Award" {
		t.Errorf("expected name 'New Award', got %q", categories[0].Name)
	}
	if categories[0].DerbyNetAwardID == nil || *categories[0].DerbyNetAwardID != awardID {
		t.Errorf("expected derbynet_award_id %d, got %v", awardID, categories[0].DerbyNetAwardID)
	}
}

func TestUpsertCategory_NewCategoryWithoutAwardID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	created, err := repo.UpsertCategory(ctx, "No Award", 1, nil)
	if err != nil {
		t.Fatalf("UpsertCategory failed: %v", err)
	}
	if !created {
		t.Error("expected category to be created")
	}

	categories, _ := repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].DerbyNetAwardID != nil {
		t.Errorf("expected nil derbynet_award_id, got %v", categories[0].DerbyNetAwardID)
	}
}

func TestUpsertCategory_UpdateExistingWithAwardID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category without award ID
	_, err := repo.CreateCategory(ctx, "Existing Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Upsert with award ID
	awardID := 99
	created, err := repo.UpsertCategory(ctx, "Existing Category", 2, &awardID)
	if err != nil {
		t.Fatalf("UpsertCategory failed: %v", err)
	}
	if created {
		t.Error("expected category to be updated, not created")
	}

	// Verify update
	categories, _ := repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].DisplayOrder != 2 {
		t.Errorf("expected display_order 2, got %d", categories[0].DisplayOrder)
	}
	if categories[0].DerbyNetAwardID == nil || *categories[0].DerbyNetAwardID != awardID {
		t.Errorf("expected derbynet_award_id %d, got %v", awardID, categories[0].DerbyNetAwardID)
	}
}

func TestUpsertCategory_PreservesExistingAwardID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create with award ID
	awardID := 123
	_, _ = repo.UpsertCategory(ctx, "Award Category", 1, &awardID)

	// Upsert without award ID (should preserve existing)
	_, err := repo.UpsertCategory(ctx, "Award Category", 2, nil)
	if err != nil {
		t.Fatalf("UpsertCategory failed: %v", err)
	}

	categories, _ := repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].DerbyNetAwardID == nil || *categories[0].DerbyNetAwardID != awardID {
		t.Errorf("expected preserved derbynet_award_id %d, got %v", awardID, categories[0].DerbyNetAwardID)
	}
}

func TestUpsertCategory_PreservesInactiveState(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create and delete category
	id, _ := repo.CreateCategory(ctx, "Inactive Category", 1, nil, nil, nil)
	_ = repo.DeleteCategory(ctx, int(id))

	// Verify inactive
	activeCategories, _ := repo.ListCategories(ctx)
	if len(activeCategories) != 0 {
		t.Fatalf("expected 0 active categories, got %d", len(activeCategories))
	}

	// Upsert should NOT reactivate - preserves user's choice to deactivate
	awardID := 50
	created, err := repo.UpsertCategory(ctx, "Inactive Category", 2, &awardID)
	if err != nil {
		t.Fatalf("UpsertCategory failed: %v", err)
	}
	if created {
		t.Error("expected category to be updated, not created")
	}

	// Verify still inactive (UpsertCategory should preserve active state)
	activeCategories, _ = repo.ListCategories(ctx)
	if len(activeCategories) != 0 {
		t.Fatalf("expected 0 active categories (should preserve inactive state), got %d", len(activeCategories))
	}
}

// ==================== GetWinnersForDerbyNet Tests ====================

func TestGetWinnersForDerbyNet_NoVotes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category but no votes
	awardID := 1
	_, _ = repo.UpsertCategory(ctx, "Empty Category", 1, &awardID)

	winners, err := repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		t.Fatalf("GetWinnersForDerbyNet failed: %v", err)
	}
	if len(winners) != 0 {
		t.Errorf("expected 0 winners (no votes), got %d", len(winners))
	}
}

func TestGetWinnersForDerbyNet_WithVotes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category with DerbyNet award ID
	awardID := 10
	_, _ = repo.UpsertCategory(ctx, "Best Design", 1, &awardID)
	categories, _ := repo.ListCategories(ctx)
	categoryID := categories[0].ID

	// Create car with DerbyNet racer ID
	_ = repo.UpsertCar(ctx, 100, "101", "Winner Racer", "Winner Car", "", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Create votes
	voter1, _ := repo.CreateVoter(ctx, "WINNER-QR1")
	voter2, _ := repo.CreateVoter(ctx, "WINNER-QR2")
	_ = repo.SaveVote(ctx, voter1, categoryID, carID)
	_ = repo.SaveVote(ctx, voter2, categoryID, carID)

	winners, err := repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		t.Fatalf("GetWinnersForDerbyNet failed: %v", err)
	}
	if len(winners) != 1 {
		t.Fatalf("expected 1 winner, got %d", len(winners))
	}

	w := winners[0]
	if w.CategoryName != "Best Design" {
		t.Errorf("expected category_name 'Best Design', got %q", w.CategoryName)
	}
	if w.DerbyNetAwardID == nil || *w.DerbyNetAwardID != awardID {
		t.Errorf("expected derbynet_award_id %d, got %v", awardID, w.DerbyNetAwardID)
	}
	if w.DerbyNetRacerID == nil || *w.DerbyNetRacerID != 100 {
		t.Errorf("expected derbynet_racer_id 100, got %v", w.DerbyNetRacerID)
	}
	if w.VoteCount != 2 {
		t.Errorf("expected vote_count 2, got %d", w.VoteCount)
	}
}

func TestGetWinnersForDerbyNet_MultipleCategories(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create categories
	awardID1 := 10
	awardID2 := 20
	_, _ = repo.UpsertCategory(ctx, "Category A", 1, &awardID1)
	_, _ = repo.UpsertCategory(ctx, "Category B", 2, &awardID2)
	categories, _ := repo.ListCategories(ctx)
	cat1ID := categories[0].ID
	cat2ID := categories[1].ID

	// Create cars
	_ = repo.UpsertCar(ctx, 101, "1", "Racer 1", "Car 1", "", "")
	_ = repo.UpsertCar(ctx, 102, "2", "Racer 2", "Car 2", "", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Create votes
	voter1, _ := repo.CreateVoter(ctx, "MULTI-WIN-1")
	voter2, _ := repo.CreateVoter(ctx, "MULTI-WIN-2")
	_ = repo.SaveVote(ctx, voter1, cat1ID, car1ID)
	_ = repo.SaveVote(ctx, voter2, cat1ID, car1ID)
	_ = repo.SaveVote(ctx, voter1, cat2ID, car2ID)

	winners, err := repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		t.Fatalf("GetWinnersForDerbyNet failed: %v", err)
	}
	if len(winners) != 2 {
		t.Fatalf("expected 2 winners, got %d", len(winners))
	}
}

func TestGetWinnersForDerbyNet_CategoryWithoutAwardID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category WITHOUT DerbyNet award ID
	_, _ = repo.CreateCategory(ctx, "Local Only", 1, nil, nil, nil)
	categories, _ := repo.ListCategories(ctx)
	categoryID := categories[0].ID

	// Create car and votes
	_ = repo.UpsertCar(ctx, 200, "201", "Local Racer", "Local Car", "", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	voter, _ := repo.CreateVoter(ctx, "LOCAL-QR")
	_ = repo.SaveVote(ctx, voter, categoryID, carID)

	winners, err := repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		t.Fatalf("GetWinnersForDerbyNet failed: %v", err)
	}
	if len(winners) != 1 {
		t.Fatalf("expected 1 winner, got %d", len(winners))
	}
	if winners[0].DerbyNetAwardID != nil {
		t.Errorf("expected nil derbynet_award_id for local category, got %v", winners[0].DerbyNetAwardID)
	}
}

func TestGetWinnersForDerbyNet_CarWithoutRacerID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category with award ID
	awardID := 30
	_, _ = repo.UpsertCategory(ctx, "Synced Category", 1, &awardID)
	categories, _ := repo.ListCategories(ctx)
	categoryID := categories[0].ID

	// Create car WITHOUT DerbyNet racer ID (local car)
	_ = repo.CreateCar(ctx, "999", "Manual Racer", "Manual Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	voter, _ := repo.CreateVoter(ctx, "MANUAL-QR")
	_ = repo.SaveVote(ctx, voter, categoryID, carID)

	winners, err := repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		t.Fatalf("GetWinnersForDerbyNet failed: %v", err)
	}
	if len(winners) != 1 {
		t.Fatalf("expected 1 winner, got %d", len(winners))
	}
	if winners[0].DerbyNetRacerID != nil {
		t.Errorf("expected nil derbynet_racer_id for manual car, got %v", winners[0].DerbyNetRacerID)
	}
}

// ==================== Car Eligibility Tests ====================

func TestListCars_IncludesEligibleField(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "")
	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	// New cars should be eligible by default
	if !cars[0].Eligible {
		t.Error("expected new car to be eligible by default")
	}
}

func TestSetCarEligibility_SetToFalse(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set eligibility to false
	err := repo.SetCarEligibility(ctx, carID, false)
	if err != nil {
		t.Fatalf("SetCarEligibility failed: %v", err)
	}

	// Verify car is now ineligible
	cars, _ = repo.ListCars(ctx)
	if cars[0].Eligible {
		t.Error("expected car to be ineligible after setting to false")
	}
}

func TestSetCarEligibility_SetToTrue(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set to ineligible then back to eligible
	_ = repo.SetCarEligibility(ctx, carID, false)
	err := repo.SetCarEligibility(ctx, carID, true)
	if err != nil {
		t.Fatalf("SetCarEligibility failed: %v", err)
	}

	// Verify car is now eligible
	cars, _ = repo.ListCars(ctx)
	if !cars[0].Eligible {
		t.Error("expected car to be eligible after setting to true")
	}
}

func TestListEligibleCars_FiltersIneligible(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create 3 cars
	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "")
	_ = repo.CreateCar(ctx, "2", "Racer 2", "Car 2", "")
	_ = repo.CreateCar(ctx, "3", "Racer 3", "Car 3", "")
	cars, _ := repo.ListCars(ctx)

	// Make one ineligible
	_ = repo.SetCarEligibility(ctx, cars[1].ID, false)

	// ListCars should return all 3
	allCars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(allCars) != 3 {
		t.Errorf("expected 3 cars from ListCars, got %d", len(allCars))
	}

	// ListEligibleCars should return only 2
	eligibleCars, err := repo.ListEligibleCars(ctx)
	if err != nil {
		t.Fatalf("ListEligibleCars failed: %v", err)
	}
	if len(eligibleCars) != 2 {
		t.Errorf("expected 2 eligible cars, got %d", len(eligibleCars))
	}

	// Verify the ineligible car is not in the list
	for _, car := range eligibleCars {
		if car.CarNumber == "2" {
			t.Error("expected ineligible car to be excluded from ListEligibleCars")
		}
	}
}

func TestListEligibleCars_Empty(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	cars, err := repo.ListEligibleCars(ctx)
	if err != nil {
		t.Fatalf("ListEligibleCars failed: %v", err)
	}
	if len(cars) != 0 {
		t.Errorf("expected 0 eligible cars, got %d", len(cars))
	}
}

func TestListEligibleCars_AllIneligible(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create cars and make all ineligible
	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "")
	_ = repo.CreateCar(ctx, "2", "Racer 2", "Car 2", "")
	cars, _ := repo.ListCars(ctx)
	_ = repo.SetCarEligibility(ctx, cars[0].ID, false)
	_ = repo.SetCarEligibility(ctx, cars[1].ID, false)

	eligibleCars, err := repo.ListEligibleCars(ctx)
	if err != nil {
		t.Fatalf("ListEligibleCars failed: %v", err)
	}
	if len(eligibleCars) != 0 {
		t.Errorf("expected 0 eligible cars when all are ineligible, got %d", len(eligibleCars))
	}
}

func TestGetCar_IncludesEligibleField(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_ = repo.CreateCar(ctx, "1", "Racer 1", "Car 1", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	car, err := repo.GetCar(ctx, carID)
	if err != nil {
		t.Fatalf("GetCar failed: %v", err)
	}
	if car == nil {
		t.Fatal("expected car to exist")
	}
	// New cars should be eligible by default
	if !car.Eligible {
		t.Error("expected GetCar to return eligible=true for new car")
	}

	// Set to ineligible and verify
	_ = repo.SetCarEligibility(ctx, carID, false)
	car, _ = repo.GetCar(ctx, carID)
	if car.Eligible {
		t.Error("expected GetCar to return eligible=false after setting")
	}
}

// ==================== UpdateCar Tests ====================

func TestUpdateCar_Success(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a car
	_ = repo.CreateCar(ctx, "101", "Original Racer", "Original Car", "http://original.com/photo.jpg")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Update the car
	err := repo.UpdateCar(ctx, carID, "102", "Updated Racer", "Updated Car", "http://updated.com/photo.jpg", "")
	if err != nil {
		t.Fatalf("UpdateCar failed: %v", err)
	}

	// Verify the update
	car, err := repo.GetCar(ctx, carID)
	if err != nil {
		t.Fatalf("GetCar failed: %v", err)
	}

	if car.CarNumber != "102" {
		t.Errorf("expected car_number '102', got '%s'", car.CarNumber)
	}
	if car.RacerName != "Updated Racer" {
		t.Errorf("expected racer_name 'Updated Racer', got '%s'", car.RacerName)
	}
	if car.CarName != "Updated Car" {
		t.Errorf("expected car_name 'Updated Car', got '%s'", car.CarName)
	}
	if car.PhotoURL != "http://updated.com/photo.jpg" {
		t.Errorf("expected photo_url 'http://updated.com/photo.jpg', got '%s'", car.PhotoURL)
	}
}

func TestUpdateCar_NonExistentCar(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Try to update a car that doesn't exist
	err := repo.UpdateCar(ctx, 99999, "102", "Updated Racer", "Updated Car", "", "")
	// Should not return an error (SQLite UPDATE on non-existent row succeeds)
	if err != nil {
		t.Errorf("UpdateCar on non-existent car returned error: %v", err)
	}
}

func TestUpdateCar_EmptyFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a car with data
	_ = repo.CreateCar(ctx, "101", "Original Racer", "Original Car", "http://original.com/photo.jpg")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Update with empty fields
	err := repo.UpdateCar(ctx, carID, "", "", "", "", "")
	if err != nil {
		t.Fatalf("UpdateCar with empty fields failed: %v", err)
	}

	// Verify empty fields were set
	car, _ := repo.GetCar(ctx, carID)
	if car.CarNumber != "" {
		t.Errorf("expected empty car_number, got '%s'", car.CarNumber)
	}
	if car.RacerName != "" {
		t.Errorf("expected empty racer_name, got '%s'", car.RacerName)
	}
	if car.CarName != "" {
		t.Errorf("expected empty car_name, got '%s'", car.CarName)
	}
	if car.PhotoURL != "" {
		t.Errorf("expected empty photo_url, got '%s'", car.PhotoURL)
	}
}

func TestUpdateCar_ClosedDB(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database
	repo.db.Close()

	// Try to update - should fail
	err := repo.UpdateCar(ctx, 1, "101", "Racer", "Car", "", "")
	if err == nil {
		t.Error("expected error when updating with closed database")
	}
}

// ==================== DeleteCar Tests ====================

func TestDeleteCar_Success(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a car
	_ = repo.CreateCar(ctx, "101", "Racer 1", "Car 1", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Verify car is active
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}

	// Delete the car (soft delete)
	err := repo.DeleteCar(ctx, carID)
	if err != nil {
		t.Fatalf("DeleteCar failed: %v", err)
	}

	// Verify car no longer appears in ListCars (active = 0)
	cars, _ = repo.ListCars(ctx)
	if len(cars) != 0 {
		t.Errorf("expected 0 active cars after delete, got %d", len(cars))
	}

	// Verify the car still exists in DB but is inactive
	var active bool
	err = repo.db.QueryRowContext(ctx, `SELECT active FROM cars WHERE id = ?`, carID).Scan(&active)
	if err != nil {
		t.Fatalf("failed to query car: %v", err)
	}
	if active {
		t.Error("expected active=false after delete")
	}
}

func TestDeleteCar_NonExistentCar(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Try to delete a car that doesn't exist
	err := repo.DeleteCar(ctx, 99999)
	// Should not return an error (SQLite UPDATE on non-existent row succeeds)
	if err != nil {
		t.Errorf("DeleteCar on non-existent car returned error: %v", err)
	}
}

func TestDeleteCar_MultipleDelete(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a car
	_ = repo.CreateCar(ctx, "101", "Racer 1", "Car 1", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Delete twice
	err := repo.DeleteCar(ctx, carID)
	if err != nil {
		t.Fatalf("first DeleteCar failed: %v", err)
	}

	err = repo.DeleteCar(ctx, carID)
	if err != nil {
		t.Fatalf("second DeleteCar failed: %v", err)
	}

	// Should still be inactive
	cars, _ = repo.ListCars(ctx)
	if len(cars) != 0 {
		t.Errorf("expected 0 active cars after double delete, got %d", len(cars))
	}
}

func TestDeleteCar_ClosedDB(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database
	repo.db.Close()

	// Try to delete - should fail
	err := repo.DeleteCar(ctx, 1)
	if err == nil {
		t.Error("expected error when deleting with closed database")
	}
}

func TestDeleteCar_DoesNotAffectOtherCars(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create multiple cars
	_ = repo.CreateCar(ctx, "101", "Racer 1", "Car 1", "")
	_ = repo.CreateCar(ctx, "102", "Racer 2", "Car 2", "")
	_ = repo.CreateCar(ctx, "103", "Racer 3", "Car 3", "")
	cars, _ := repo.ListCars(ctx)

	// Delete the middle car
	err := repo.DeleteCar(ctx, cars[1].ID)
	if err != nil {
		t.Fatalf("DeleteCar failed: %v", err)
	}

	// Verify only 2 cars remain
	remainingCars, _ := repo.ListCars(ctx)
	if len(remainingCars) != 2 {
		t.Errorf("expected 2 active cars after deleting one, got %d", len(remainingCars))
	}

	// Verify the correct cars remain
	carNumbers := make(map[string]bool)
	for _, car := range remainingCars {
		carNumbers[car.CarNumber] = true
	}
	if !carNumbers["101"] || !carNumbers["103"] {
		t.Error("expected cars 101 and 103 to remain active")
	}
	if carNumbers["102"] {
		t.Error("expected car 102 to be deleted")
	}
}

// ==================== New Method Tests ====================

func TestClose_Success(t *testing.T) {
	repo := newTestRepo(t)

	err := repo.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestClose_NilDB(t *testing.T) {
	repo := &Repository{db: nil}

	err := repo.Close()
	if err != nil {
		t.Fatalf("Close with nil db should not error, got: %v", err)
	}
}

func TestPing_Success(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestClearTable_InvalidTable(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.ClearTable(ctx, "malicious_table")
	if err != ErrInvalidTable {
		t.Fatalf("expected ErrInvalidTable, got: %v", err)
	}
}

func TestClearTable_SQLInjection(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Try SQL injection attack
	err := repo.ClearTable(ctx, "voters; DROP TABLE cars;")
	if err != ErrInvalidTable {
		t.Fatalf("expected ErrInvalidTable for SQL injection attempt, got: %v", err)
	}

	// Verify cars table still exists
	_, err = repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("cars table was affected by injection attempt: %v", err)
	}
}

func TestNew_InvalidPath(t *testing.T) {
	// Try to create repo with invalid path
	_, err := New("/invalid/path/that/does/not/exist/test.db")
	if err == nil {
		t.Fatal("expected error when creating repo with invalid path, got nil")
	}
}

func TestGetVotingStats_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.GetVotingStats(ctx)
	if err == nil {
		t.Fatal("expected error from GetVotingStats on closed DB, got nil")
	}
}

func TestListVoters_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.ListVoters(ctx)
	if err == nil {
		t.Fatal("expected error from ListVoters on closed DB, got nil")
	}
}

func TestListCategories_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.ListCategories(ctx)
	if err == nil {
		t.Fatal("expected error from ListCategories on closed DB, got nil")
	}
}

func TestListAllCategories_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.ListAllCategories(ctx)
	if err == nil {
		t.Fatal("expected error from ListAllCategories on closed DB, got nil")
	}
}

func TestListCategoryGroups_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.ListCategoryGroups(ctx)
	if err == nil {
		t.Fatal("expected error from ListCategoryGroups on closed DB, got nil")
	}
}

func TestListCars_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.ListCars(ctx)
	if err == nil {
		t.Fatal("expected error from ListCars on closed DB, got nil")
	}
}

func TestListEligibleCars_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.ListEligibleCars(ctx)
	if err == nil {
		t.Fatal("expected error from ListEligibleCars on closed DB, got nil")
	}
}

func TestGetVoterVotes_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.GetVoterVotes(ctx, 1)
	if err == nil {
		t.Fatal("expected error from GetVoterVotes on closed DB, got nil")
	}
}

func TestGetVoteResults_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.GetVoteResults(ctx)
	if err == nil {
		t.Fatal("expected error from GetVoteResults on closed DB, got nil")
	}
}

func TestGetVoteResultsWithCars_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.GetVoteResultsWithCars(ctx)
	if err == nil {
		t.Fatal("expected error from GetVoteResultsWithCars on closed DB, got nil")
	}
}

func TestGetWinnersForDerbyNet_Errors(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Close the database to force errors
	repo.Close()

	_, err := repo.GetWinnersForDerbyNet(ctx)
	if err == nil {
		t.Fatal("expected error from GetWinnersForDerbyNet on closed DB, got nil")
	}
}

func TestSaveVote_UpdateLastVotedError(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create voter, car and category
	voterID, _ := repo.CreateVoter(ctx, "VOTE-TEST")
	_ = repo.CreateCar(ctx, "100", "Racer", "Car", "")
	catID, _ := repo.CreateCategory(ctx, "Test", 1, nil, nil, nil)

	// Use UpsertCar with derbynet ID so we can look it up
	_ = repo.UpsertCar(ctx, 100, "100", "Racer", "Car", "", "")
	carID, _, _ := repo.GetCarByDerbyNetID(ctx, 100)

	// Save vote successfully
	err := repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("SaveVote failed: %v", err)
	}

	// Verify vote was saved
	votes, _ := repo.GetVoterVotes(ctx, voterID)
	if votes[int(catID)] != int(carID) {
		t.Error("vote was not saved correctly")
	}
}

func TestUpsertCategory_UpdateExistingError(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a category first
	catID, err := repo.CreateCategory(ctx, "Test Cat", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Upsert should update existing (by name)
	created, err := repo.UpsertCategory(ctx, "Test Cat", 2, nil)
	if err != nil {
		t.Fatalf("UpsertCategory failed: %v", err)
	}
	if created {
		t.Error("expected created=false for existing category, got true")
	}

	// Verify the display order was updated
	categories, _ := repo.ListAllCategories(ctx)
	for _, cat := range categories {
		if cat["id"] == int(catID) {
			if cat["display_order"] != 2 {
				t.Errorf("expected display_order=2, got %d", cat["display_order"])
			}
		}
	}
}

func TestFindConflictingVote_NoConflictFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create necessary data
	voterID, _ := repo.CreateVoter(ctx, "CONFLICT-TEST")
	_ = repo.UpsertCar(ctx, 100, "100", "Racer", "Car", "", "")
	carID, _, _ := repo.GetCarByDerbyNetID(ctx, 100)
	catID, _ := repo.CreateCategory(ctx, "Test", 1, nil, nil, nil)

	// Find conflict when none exists
	conflictCatID, conflictName, hasConflict, err := repo.FindConflictingVote(ctx, voterID, int(carID), int(catID), 1)
	if err != nil {
		t.Fatalf("FindConflictingVote failed: %v", err)
	}
	if hasConflict {
		t.Errorf("expected no conflict, but found conflict in category %d (%s)", conflictCatID, conflictName)
	}
}

func TestGetCar_WithNullFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create car with minimal fields (null racer_name, car_name, photo_url)
	_ = repo.CreateCar(ctx, "999", "", "", "")

	// Get the car - this should handle null fields
	cars, _ := repo.ListCars(ctx)
	var carID int
	for _, car := range cars {
		if car.CarNumber == "999" {
			carID = car.ID
			break
		}
	}

	car, err := repo.GetCar(ctx, carID)
	if err != nil {
		t.Fatalf("GetCar failed: %v", err)
	}
	if car == nil {
		t.Fatal("expected car, got nil")
	}
	if car.RacerName != "" {
		t.Errorf("expected empty racer_name, got %s", car.RacerName)
	}
}

func TestGetCar_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	car, err := repo.GetCar(ctx, 99999)
	var appErr *errors.Error
	if !stderrors.As(err, &appErr) || appErr.Kind != errors.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-existent car, got: %v", err)
	}
	if car != nil {
		t.Error("expected nil car when not found")
	}
}

func TestGetCarByDerbyNetID_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, found, err := repo.GetCarByDerbyNetID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetCarByDerbyNetID failed: %v", err)
	}
	if found {
		t.Error("expected not found, got found")
	}
	if id != 0 {
		t.Errorf("expected id=0, got %d", id)
	}
}

func TestGetVoterByQRCode_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	id, found, err := repo.GetVoterByQRCode(ctx, "NOT-EXISTS")
	if err != nil {
		t.Fatalf("GetVoterByQRCode failed: %v", err)
	}
	if found {
		t.Error("expected not found, got found")
	}
	if id != 0 {
		t.Errorf("expected id=0, got %d", id)
	}
}

// ==================== Manual Winner Override Tests ====================

func TestSetManualWinner_Success(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "42", "John Smith", "Speed Demon", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set manual winner
	err := repo.SetManualWinner(ctx, int(catID), carID, "Resolved tie")
	if err != nil {
		t.Fatalf("SetManualWinner failed: %v", err)
	}

	// Verify override was set
	categories, _ := repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	cat := categories[0]
	if cat.OverrideWinnerCarID == nil {
		t.Fatal("expected override_winner_car_id to be set")
	}
	if *cat.OverrideWinnerCarID != carID {
		t.Errorf("expected override_winner_car_id=%d, got %d", carID, *cat.OverrideWinnerCarID)
	}
	if cat.OverrideReason != "Resolved tie" {
		t.Errorf("expected override_reason='Resolved tie', got '%s'", cat.OverrideReason)
	}
	if cat.OverriddenAt == "" {
		t.Error("expected overridden_at to be set")
	}
}

func TestSetManualWinner_UpdateExisting(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and two cars
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "42", "John", "Car A", "")
	_ = repo.CreateCar(ctx, "7", "Sarah", "Car B", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Set first manual winner
	repo.SetManualWinner(ctx, int(catID), car1ID, "First choice")

	// Update to second winner
	err := repo.SetManualWinner(ctx, int(catID), car2ID, "Changed to runner-up")
	if err != nil {
		t.Fatalf("SetManualWinner (update) failed: %v", err)
	}

	// Verify override was updated
	categories, _ := repo.ListCategories(ctx)
	cat := categories[0]

	if cat.OverrideWinnerCarID == nil {
		t.Fatal("expected override_winner_car_id to be set")
	}
	if *cat.OverrideWinnerCarID != car2ID {
		t.Errorf("expected car2ID=%d, got %d", car2ID, *cat.OverrideWinnerCarID)
	}
	if cat.OverrideReason != "Changed to runner-up" {
		t.Errorf("expected updated reason, got '%s'", cat.OverrideReason)
	}
}

func TestClearManualWinner_Success(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "42", "John", "Speed Demon", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set manual winner
	repo.SetManualWinner(ctx, int(catID), carID, "Test reason")

	// Clear override
	err := repo.ClearManualWinner(ctx, int(catID))
	if err != nil {
		t.Fatalf("ClearManualWinner failed: %v", err)
	}

	// Verify override was cleared
	categories, _ := repo.ListCategories(ctx)
	cat := categories[0]

	if cat.OverrideWinnerCarID != nil {
		t.Errorf("expected override_winner_car_id to be nil, got %v", *cat.OverrideWinnerCarID)
	}
	if cat.OverrideReason != "" {
		t.Errorf("expected override_reason to be empty, got '%s'", cat.OverrideReason)
	}
	if cat.OverriddenAt != "" {
		t.Errorf("expected overridden_at to be empty, got '%s'", cat.OverriddenAt)
	}
}

func TestClearManualWinner_NoOverrideExists(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category without override
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// Clear non-existent override (should not error)
	err := repo.ClearManualWinner(ctx, int(catID))
	if err != nil {
		t.Fatalf("ClearManualWinner failed: %v", err)
	}

	// Verify still no override
	categories, _ := repo.ListCategories(ctx)
	cat := categories[0]

	if cat.OverrideWinnerCarID != nil {
		t.Error("expected override_winner_car_id to remain nil")
	}
}

func TestGetWinnersForDerbyNet_WithOverride(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category with derbynet_award_id
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	awardID := 100
	repo.UpsertCategory(ctx, "Best Design", 1, &awardID)

	// Create two cars (car numbers chosen so they sort in order matching derbynet_racer_id)
	repo.UpsertCar(ctx, 1, "7", "John", "Car A", "", "")
	repo.UpsertCar(ctx, 2, "42", "Sarah", "Car B", "", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Create voters and votes (car1 gets more votes)
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")

	repo.SaveVote(ctx, v1, int(catID), car1ID) // Car A: 2 votes
	repo.SaveVote(ctx, v2, int(catID), car1ID)
	repo.SaveVote(ctx, v3, int(catID), car2ID) // Car B: 1 vote

	// Set manual override to car2 (not the vote winner)
	repo.SetManualWinner(ctx, int(catID), car2ID, "Resolved tie")

	// Get winners
	winners, err := repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		t.Fatalf("GetWinnersForDerbyNet failed: %v", err)
	}

	if len(winners) != 1 {
		t.Fatalf("expected 1 winner, got %d", len(winners))
	}

	winner := winners[0]

	// Should use the override (car2), not the vote leader (car1)
	if winner.CarID != car2ID {
		t.Errorf("expected override winner car2ID=%d, got %d", car2ID, winner.CarID)
	}
	if winner.CategoryID != int(catID) {
		t.Errorf("expected categoryID=%d, got %d", catID, winner.CategoryID)
	}
	if winner.DerbyNetAwardID == nil || *winner.DerbyNetAwardID != awardID {
		t.Errorf("expected derbynet_award_id=%d, got %v", awardID, winner.DerbyNetAwardID)
	}
	if winner.DerbyNetRacerID == nil {
		t.Error("expected derbynet_racer_id to be set")
	} else if *winner.DerbyNetRacerID != 2 {
		t.Errorf("expected derbynet_racer_id=2 for car2, got %d", *winner.DerbyNetRacerID)
	}
}

func TestGetWinnersForDerbyNet_WithoutOverride(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	awardID := 100
	repo.UpsertCategory(ctx, "Best Design", 1, &awardID)

	// Create two cars (car numbers chosen so they sort in order matching derbynet_racer_id)
	repo.UpsertCar(ctx, 1, "7", "John", "Car A", "", "")
	repo.UpsertCar(ctx, 2, "42", "Sarah", "Car B", "", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Create votes (car1 wins)
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")

	repo.SaveVote(ctx, v1, int(catID), car1ID) // Car A: 2 votes
	repo.SaveVote(ctx, v2, int(catID), car1ID)
	repo.SaveVote(ctx, v3, int(catID), car2ID) // Car B: 1 vote

	// No override set

	// Get winners
	winners, err := repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		t.Fatalf("GetWinnersForDerbyNet failed: %v", err)
	}

	if len(winners) != 1 {
		t.Fatalf("expected 1 winner, got %d", len(winners))
	}

	winner := winners[0]

	// Should use vote winner (car1)
	if winner.CarID != car1ID {
		t.Errorf("expected vote winner car1ID=%d, got %d", car1ID, winner.CarID)
	}
	if winner.VoteCount != 2 {
		t.Errorf("expected vote_count=2, got %d", winner.VoteCount)
	}
}

func TestGetWinnersForDerbyNet_OverrideWithNoVotes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	awardID := 100
	repo.UpsertCategory(ctx, "Best Design", 1, &awardID)

	// Create car
	repo.UpsertCar(ctx, 1, "42", "John", "Car A", "", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// No votes, but set manual override
	repo.SetManualWinner(ctx, int(catID), carID, "Manual selection with no votes")

	// Get winners
	winners, err := repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		t.Fatalf("GetWinnersForDerbyNet failed: %v", err)
	}

	if len(winners) != 1 {
		t.Fatalf("expected 1 winner (override), got %d", len(winners))
	}

	winner := winners[0]

	// Should use the override even though there are no votes
	if winner.CarID != carID {
		t.Errorf("expected override carID=%d, got %d", carID, winner.CarID)
	}
	if winner.VoteCount != 0 {
		t.Errorf("expected vote_count=0 (no votes), got %d", winner.VoteCount)
	}
}

func TestSetManualWinner_WithInactiveCar(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "42", "John", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set manual override
	repo.SetManualWinner(ctx, int(catID), carID, "Test override")

	// Verify override is set
	categories, _ := repo.ListCategories(ctx)
	if categories[0].OverrideWinnerCarID == nil {
		t.Fatal("expected override to be set before car deletion")
	}

	// Soft delete the car (sets active=0, doesn't actually delete the row)
	err := repo.DeleteCar(ctx, carID)
	if err != nil {
		t.Fatalf("failed to delete car: %v", err)
	}

	// Verify override is still set (soft delete doesn't trigger foreign key constraint)
	// This is expected behavior - the override references the car ID, which still exists in DB
	categories, _ = repo.ListCategories(ctx)
	if categories[0].OverrideWinnerCarID == nil {
		t.Error("override should still be set after soft delete")
	}
	if *categories[0].OverrideWinnerCarID != carID {
		t.Errorf("expected override_winner_car_id=%d, got %v", carID, *categories[0].OverrideWinnerCarID)
	}

	// The application should handle inactive cars gracefully in the UI/results display
}

func TestListCategories_IncludesOverrideFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "42", "John", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set override
	repo.SetManualWinner(ctx, int(catID), carID, "Test reason")

	// List categories
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}

	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	cat := categories[0]

	// Verify override fields are populated
	if cat.OverrideWinnerCarID == nil {
		t.Fatal("expected override_winner_car_id to be set")
	}
	if *cat.OverrideWinnerCarID != carID {
		t.Errorf("expected carID=%d, got %d", carID, *cat.OverrideWinnerCarID)
	}
	if cat.OverrideReason != "Test reason" {
		t.Errorf("expected 'Test reason', got '%s'", cat.OverrideReason)
	}
	if cat.OverriddenAt == "" {
		t.Error("expected overridden_at to be set")
	}
}

func TestListAllCategories_IncludesOverrideFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "42", "John", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set override
	repo.SetManualWinner(ctx, int(catID), carID, "Admin choice")

	// List all categories
	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	cat := categories[0]

	// Verify override fields are in the map
	overrideCarID, ok := cat["override_winner_car_id"]
	if !ok {
		t.Fatal("expected override_winner_car_id in map")
	}
	if overrideCarID != carID {
		t.Errorf("expected carID=%d, got %v", carID, overrideCarID)
	}

	overrideReason, ok := cat["override_reason"]
	if !ok {
		t.Fatal("expected override_reason in map")
	}
	if overrideReason != "Admin choice" {
		t.Errorf("expected 'Admin choice', got %v", overrideReason)
	}

	overriddenAt, ok := cat["overridden_at"]
	if !ok {
		t.Fatal("expected overridden_at in map")
	}
	if overriddenAt == "" {
		t.Error("expected overridden_at to be set")
	}
}
// ==================== Category Group max_wins_per_car Tests ====================

func TestCategoryGroup_CreateWithMaxWins_AndReadBack(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	maxWins := 2
	id, err := repo.CreateCategoryGroup(ctx, "Design Awards", "Design categories", nil, &maxWins, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	group, err := repo.GetCategoryGroup(ctx, fmt.Sprintf("%d", id))
	if err != nil {
		t.Fatalf("GetCategoryGroup failed: %v", err)
	}

	if group.MaxWinsPerCar == nil {
		t.Fatal("expected max_wins_per_car to be set, got nil")
	}
	if *group.MaxWinsPerCar != maxWins {
		t.Errorf("expected max_wins_per_car=%d, got %d", maxWins, *group.MaxWinsPerCar)
	}
}

func TestCategoryGroup_UpdateMaxWins_FromNullToValue(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create without max_wins_per_car
	id, err := repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	// Update to set max_wins_per_car
	newMaxWins := 1
	err = repo.UpdateCategoryGroup(ctx, fmt.Sprintf("%d", id), "Updated Group", "Updated desc", nil, &newMaxWins, 2)
	if err != nil {
		t.Fatalf("UpdateCategoryGroup failed: %v", err)
	}

	// Verify update
	group, _ := repo.GetCategoryGroup(ctx, fmt.Sprintf("%d", id))
	if group.MaxWinsPerCar == nil {
		t.Fatal("expected max_wins_per_car to be set after update, got nil")
	}
	if *group.MaxWinsPerCar != newMaxWins {
		t.Errorf("expected max_wins_per_car=%d, got %d", newMaxWins, *group.MaxWinsPerCar)
	}
}

func TestCategoryGroup_UpdateMaxWins_FromValueToNull(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create with max_wins_per_car
	maxWins := 2
	id, err := repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, &maxWins, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	// Update to clear max_wins_per_car
	err = repo.UpdateCategoryGroup(ctx, fmt.Sprintf("%d", id), "Updated Group", "Updated desc", nil, nil, 2)
	if err != nil {
		t.Fatalf("UpdateCategoryGroup failed: %v", err)
	}

	// Verify it was cleared
	group, _ := repo.GetCategoryGroup(ctx, fmt.Sprintf("%d", id))
	if group.MaxWinsPerCar != nil {
		t.Errorf("expected max_wins_per_car to be nil after clear, got %d", *group.MaxWinsPerCar)
	}
}

func TestCategoryGroup_UpdateMaxWins_FromValueToValue(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create with max_wins_per_car = 1
	oldMaxWins := 1
	id, err := repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, &oldMaxWins, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	// Update to max_wins_per_car = 3
	newMaxWins := 3
	err = repo.UpdateCategoryGroup(ctx, fmt.Sprintf("%d", id), "Updated Group", "Updated desc", nil, &newMaxWins, 2)
	if err != nil {
		t.Fatalf("UpdateCategoryGroup failed: %v", err)
	}

	// Verify update
	group, _ := repo.GetCategoryGroup(ctx, fmt.Sprintf("%d", id))
	if group.MaxWinsPerCar == nil {
		t.Fatal("expected max_wins_per_car to be set, got nil")
	}
	if *group.MaxWinsPerCar != newMaxWins {
		t.Errorf("expected max_wins_per_car=%d, got %d", newMaxWins, *group.MaxWinsPerCar)
	}
}

func TestCategoryGroup_ListIncludesMaxWinsPerCar(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create groups with different max_wins_per_car values
	maxWins1 := 1
	maxWins2 := 2
	repo.CreateCategoryGroup(ctx, "Group 1", "Has max=1", nil, &maxWins1, 1)
	repo.CreateCategoryGroup(ctx, "Group 2", "Has max=2", nil, &maxWins2, 2)
	repo.CreateCategoryGroup(ctx, "Group 3", "No max", nil, nil, 3)

	groups, err := repo.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatalf("ListCategoryGroups failed: %v", err)
	}

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// Verify each group has correct max_wins_per_car
	foundGroup1, foundGroup2, foundGroup3 := false, false, false
	for _, g := range groups {
		if g.Name == "Group 1" {
			foundGroup1 = true
			if g.MaxWinsPerCar == nil || *g.MaxWinsPerCar != maxWins1 {
				t.Errorf("Group 1: expected max_wins_per_car=%d, got %v", maxWins1, g.MaxWinsPerCar)
			}
		}
		if g.Name == "Group 2" {
			foundGroup2 = true
			if g.MaxWinsPerCar == nil || *g.MaxWinsPerCar != maxWins2 {
				t.Errorf("Group 2: expected max_wins_per_car=%d, got %v", maxWins2, g.MaxWinsPerCar)
			}
		}
		if g.Name == "Group 3" {
			foundGroup3 = true
			if g.MaxWinsPerCar != nil {
				t.Errorf("Group 3: expected max_wins_per_car=nil, got %v", g.MaxWinsPerCar)
			}
		}
	}

	if !foundGroup1 || !foundGroup2 || !foundGroup3 {
		t.Error("did not find all expected groups")
	}
}

// ==================== Manual Override Repository Tests ====================

func TestSetManualWinner_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Set manual winner
	err := repo.SetManualWinner(ctx, int(catID), cars[0].ID, "Resolved tie")
	if err != nil {
		t.Fatalf("SetManualWinner failed: %v", err)
	}

	// Verify it was set
	categories, _ := repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	cat := categories[0]
	if cat.OverrideWinnerCarID == nil {
		t.Fatal("expected override_winner_car_id to be set, got nil")
	}
	if *cat.OverrideWinnerCarID != cars[0].ID {
		t.Errorf("expected override_winner_car_id=%d, got %d", cars[0].ID, *cat.OverrideWinnerCarID)
	}
	if cat.OverrideReason != "Resolved tie" {
		t.Errorf("expected override_reason='Resolved tie', got '%s'", cat.OverrideReason)
	}
	if cat.OverriddenAt == "" {
		t.Error("expected overridden_at to be set, got empty string")
	}
}

func TestClearManualWinner_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Set and then clear manual winner
	repo.SetManualWinner(ctx, int(catID), cars[0].ID, "Test")
	err := repo.ClearManualWinner(ctx, int(catID))
	if err != nil {
		t.Fatalf("ClearManualWinner failed: %v", err)
	}

	// Verify it was cleared
	categories, _ := repo.ListCategories(ctx)
	cat := categories[0]
	if cat.OverrideWinnerCarID != nil {
		t.Errorf("expected override_winner_car_id to be nil, got %d", *cat.OverrideWinnerCarID)
	}
	if cat.OverrideReason != "" {
		t.Errorf("expected override_reason to be empty, got '%s'", cat.OverrideReason)
	}
	if cat.OverriddenAt != "" {
		t.Errorf("expected overridden_at to be empty, got '%s'", cat.OverriddenAt)
	}
}

func TestClearManualWinner_WhenNoOverride(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category without override
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// Try to clear (should not error)
	err := repo.ClearManualWinner(ctx, int(catID))
	if err != nil {
		t.Errorf("ClearManualWinner should not error when no override exists, got: %v", err)
	}
}

func TestManualWinner_ForeignKeyConstraint_DeleteCar(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set manual winner
	repo.SetManualWinner(ctx, int(catID), carID, "Test")

	// Soft-delete the car (sets active=0, doesn't trigger FK constraint)
	err := repo.DeleteCar(ctx, carID)
	if err != nil {
		t.Fatalf("DeleteCar failed: %v", err)
	}

	// Verify override remains (because DeleteCar is a soft delete)
	allCategories, _ := repo.ListAllCategories(ctx)
	if len(allCategories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(allCategories))
	}
	cat := allCategories[0]
	overrideCarID := cat["override_winner_car_id"]
	if overrideCarID == nil {
		t.Error("expected override to remain after soft delete, but it was NULL")
	}
}

func TestListCategories_ReturnsOverrideFields(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create categories
	cat1ID, _ := repo.CreateCategory(ctx, "Category 1", 1, nil, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Category 2", 2, nil, nil, nil)

	// Create cars
	repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Set override on category 1
	repo.SetManualWinner(ctx, int(cat1ID), cars[0].ID, "Test reason")

	// List categories
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}

	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}

	// Find category 1 and 2
	var cat1, cat2 *models.Category
	for i := range categories {
		if categories[i].ID == int(cat1ID) {
			cat1 = &categories[i]
		}
		if categories[i].ID == int(cat2ID) {
			cat2 = &categories[i]
		}
	}

	// Verify category 1 has override
	if cat1.OverrideWinnerCarID == nil {
		t.Error("category 1: expected override to be set")
	}
	if cat1.OverrideReason != "Test reason" {
		t.Errorf("category 1: expected override_reason='Test reason', got '%s'", cat1.OverrideReason)
	}

	// Verify category 2 does not have override
	if cat2.OverrideWinnerCarID != nil {
		t.Error("category 2: expected no override")
	}
}

// ==================== Error Path Tests with sqlmock ====================

func TestMigrate_ExecutionFailure(t *testing.T) {
	// Create a mock database that will fail on a specific migration
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Expect first migration to succeed
	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect second migration to fail
	mock.ExpectExec(".*").WillReturnError(fmt.Errorf("migration failed"))

	// Create repository with mock db
	repo := &Repository{db: db}
	err = repo.migrate()

	if err == nil {
		t.Error("expected migrate to fail, but it succeeded")
	}

	if err.Error() != "migration failed" {
		t.Errorf("expected error 'migration failed', got '%v'", err)
	}

	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestClearManualWinner_DatabaseError(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Expect the UPDATE to fail
	mock.ExpectExec("UPDATE categories").
		WillReturnError(fmt.Errorf("database locked"))

	repo := &Repository{db: db}
	err = repo.ClearManualWinner(context.Background(), 1)

	if err == nil {
		t.Error("expected ClearManualWinner to fail, but it succeeded")
	}

	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNew_InvalidDatabasePath(t *testing.T) {
	// Try to create a database in a directory that doesn't exist
	_, err := New("/nonexistent/path/to/database.db")
	if err == nil {
		t.Error("expected New to fail with invalid path, but it succeeded")
	}
}

func TestNew_ForeignKeyPragmaSuccess(t *testing.T) {
	// Create a database to verify foreign keys are enabled
	repo := newTestRepo(t)

	// Try to insert a vote with invalid car_id (should fail due to foreign key constraint)
	_, err := repo.db.Exec("INSERT INTO votes (voter_id, category_id, car_id) VALUES (9999, 9999, 9999)")
	if err == nil {
		t.Error("expected foreign key constraint to be enforced, but insert succeeded")
	}
}

func TestNew_MigrateFailsReadOnlyDB(t *testing.T) {
	// Create a temporary read-only database file
	tmpfile, err := os.CreateTemp("", "readonly-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Make it read-only
	if err := os.Chmod(tmpfile.Name(), 0444); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}

	// Try to create repository - should fail during migration
	_, err = New(tmpfile.Name())
	if err == nil {
		t.Error("expected New to fail with read-only database, but it succeeded")
	}
}

// ==================== Voter Types Tests ====================

func TestCreateCategoryWithVoterTypes_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	voterTypes := []string{"general", "racer", "Race Committee"}
	catID, err := repo.CreateCategory(ctx, "Special Award", 1, nil, voterTypes, nil)
	if err != nil {
		t.Fatalf("CreateCategoryWithVoterTypes failed: %v", err)
	}
	if catID == 0 {
		t.Fatal("expected non-zero category ID")
	}

	// Verify it was created with voter types
	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	allowedTypes, ok := categories[0]["allowed_voter_types"].([]string)
	if !ok {
		t.Fatal("expected allowed_voter_types to be []string")
	}
	if len(allowedTypes) != 3 {
		t.Fatalf("expected 3 voter types, got %d", len(allowedTypes))
	}
	if allowedTypes[0] != "general" || allowedTypes[1] != "racer" || allowedTypes[2] != "Race Committee" {
		t.Errorf("unexpected voter types: %v", allowedTypes)
	}
}

func TestCreateCategoryWithVoterTypes_EmptyTypes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create with empty voter types array (should allow all voters)
	_, err := repo.CreateCategory(ctx, "Open Award", 1, nil, []string{}, nil)
	if err != nil {
		t.Fatalf("CreateCategoryWithVoterTypes failed: %v", err)
	}

	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	// When no voter types are set, the field should not be present in the map
	_, hasField := categories[0]["allowed_voter_types"]
	if hasField {
		t.Error("expected allowed_voter_types to not be present for empty types")
	}
}

func TestUpdateCategoryWithVoterTypes_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category without voter types
	catID, _ := repo.CreateCategory(ctx, "Award", 1, nil, nil, nil)

	// Update with voter types
	voterTypes := []string{"racer", "Cubmaster"}
	err := repo.UpdateCategory(ctx, int(catID), "Updated Award", 2, nil, voterTypes, nil, true)
	if err != nil {
		t.Fatalf("UpdateCategoryWithVoterTypes failed: %v", err)
	}

	// Verify update
	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	cat := categories[0]
	if cat["name"] != "Updated Award" {
		t.Errorf("expected name 'Updated Award', got %v", cat["name"])
	}
	if cat["display_order"] != 2 {
		t.Errorf("expected display_order 2, got %v", cat["display_order"])
	}

	allowedTypes, ok := cat["allowed_voter_types"].([]string)
	if !ok {
		t.Fatal("expected allowed_voter_types to be []string")
	}
	if len(allowedTypes) != 2 {
		t.Fatalf("expected 2 voter types, got %d", len(allowedTypes))
	}
}

func TestUpdateCategoryWithVoterTypes_ClearTypes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create category with voter types
	voterTypes := []string{"general", "racer"}
	catID, _ := repo.CreateCategory(ctx, "Award", 1, nil, voterTypes, nil)

	// Update to clear voter types
	err := repo.UpdateCategory(ctx, int(catID), "Award", 1, nil, []string{}, nil, true)
	if err != nil {
		t.Fatalf("UpdateCategoryWithVoterTypes failed: %v", err)
	}

	// Verify types were cleared
	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	_, hasField := categories[0]["allowed_voter_types"]
	if hasField {
		t.Error("expected allowed_voter_types to be cleared")
	}
}

func TestGetVoterType_ExistingVoter(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create voter with specific type
	voterID, _ := repo.CreateVoterFull(ctx, nil, "John", "john@example.com", "racer", "QR123", "")

	voterType, err := repo.GetVoterType(ctx, int(voterID))
	if err != nil {
		t.Fatalf("GetVoterType failed: %v", err)
	}
	if voterType != "racer" {
		t.Errorf("expected voter type 'racer', got %s", voterType)
	}
}

func TestGetVoterType_DefaultGeneral(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create voter without specifying type (should default to general)
	voterID, _ := repo.CreateVoter(ctx, "QR456")

	voterType, err := repo.GetVoterType(ctx, int(voterID))
	if err != nil {
		t.Fatalf("GetVoterType failed: %v", err)
	}
	if voterType != "general" {
		t.Errorf("expected voter type 'general', got %s", voterType)
	}
}

func TestGetVoterType_NonExistent(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetVoterType(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent voter, got nil")
	}
}

func TestListCategories_InvalidJSON(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Directly insert a category with invalid JSON for allowed_voter_types
	db := repo.DB()
	_, err := db.ExecContext(ctx,
		`INSERT INTO categories (name, display_order, allowed_voter_types, active) VALUES (?, ?, ?, 1)`,
		"Bad JSON Category", 1, `{invalid json`)
	if err != nil {
		t.Fatalf("failed to insert test category: %v", err)
	}

	// ListCategories should return error when trying to unmarshal invalid JSON
	_, err = repo.ListCategories(ctx)
	if err == nil {
		t.Fatal("expected error from ListCategories with invalid JSON, got nil")
	}
}

func TestListAllCategories_WithVoterTypes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create categories with different voter type configurations
	repo.CreateCategory(ctx, "Racer Only", 1, nil, []string{"racer"}, nil)
	repo.CreateCategory(ctx, "All Voters", 2, nil, nil, nil)
	repo.CreateCategory(ctx, "Multiple Types", 3, nil, []string{"general", "racer", "Cubmaster"}, nil)

	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}
	if len(categories) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(categories))
	}

	// Verify first category has racer only
	cat1Types, ok := categories[0]["allowed_voter_types"].([]string)
	if !ok {
		t.Fatal("expected allowed_voter_types for cat1")
	}
	if len(cat1Types) != 1 || cat1Types[0] != "racer" {
		t.Errorf("expected ['racer'], got %v", cat1Types)
	}

	// Verify second category has no voter type restrictions
	_, hasCat2Types := categories[1]["allowed_voter_types"]
	if hasCat2Types {
		t.Error("expected no allowed_voter_types for cat2")
	}

	// Verify third category has multiple types
	cat3Types, ok := categories[2]["allowed_voter_types"].([]string)
	if !ok {
		t.Fatal("expected allowed_voter_types for cat3")
	}
	if len(cat3Types) != 3 {
		t.Errorf("expected 3 types, got %d", len(cat3Types))
	}
}

// ==================== Additional Error Path Tests ====================

func TestGetVoterType_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Expect query to fail with a database error (not ErrNoRows)
	mock.ExpectQuery("SELECT voter_type FROM voters WHERE id = ?").
		WillReturnError(fmt.Errorf("database connection lost"))

	repo := &Repository{db: db}
	ctx := context.Background()

	_, err = repo.GetVoterType(ctx, 123)
	if err == nil {
		t.Fatal("expected error from GetVoterType with database error, got nil")
	}
	if err.Error() != "database connection lost" {
		t.Errorf("expected 'database connection lost' error, got: %v", err)
	}
}

func TestGetVoterType_WithNullValue(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Return a row with NULL voter_type
	rows := sqlmock.NewRows([]string{"voter_type"}).AddRow(nil)
	mock.ExpectQuery("SELECT voter_type FROM voters WHERE id = ?").
		WillReturnRows(rows)

	repo := &Repository{db: db}
	ctx := context.Background()

	voterType, err := repo.GetVoterType(ctx, 123)
	if err != nil {
		t.Fatalf("GetVoterType failed: %v", err)
	}
	if voterType != "general" {
		t.Errorf("expected 'general' for NULL voter_type, got %s", voterType)
	}
}

func TestGetVoterType_WithValidValue(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Return a row with valid voter_type
	rows := sqlmock.NewRows([]string{"voter_type"}).AddRow("racer")
	mock.ExpectQuery("SELECT voter_type FROM voters WHERE id = ?").
		WillReturnRows(rows)

	repo := &Repository{db: db}
	ctx := context.Background()

	voterType, err := repo.GetVoterType(ctx, 123)
	if err != nil {
		t.Fatalf("GetVoterType failed: %v", err)
	}
	if voterType != "racer" {
		t.Errorf("expected 'racer', got %s", voterType)
	}
}

// ==================== Allowed Ranks Tests ====================

func TestCreateCategoryWithRanks_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	ranks := []string{"Tiger", "Lion", "Bear"}
	catID, err := repo.CreateCategory(ctx, "Rank-Specific Award", 1, nil, nil, ranks)
	if err != nil {
		t.Fatalf("CreateCategory with ranks failed: %v", err)
	}
	if catID == 0 {
		t.Fatal("expected non-zero category ID")
	}

	// Verify it was created with ranks using ListCategories
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	if len(categories[0].AllowedRanks) != 3 {
		t.Fatalf("expected 3 ranks, got %d", len(categories[0].AllowedRanks))
	}
	if categories[0].AllowedRanks[0] != "Tiger" || categories[0].AllowedRanks[1] != "Lion" || categories[0].AllowedRanks[2] != "Bear" {
		t.Errorf("unexpected ranks: %v", categories[0].AllowedRanks)
	}
}

func TestCreateCategoryWithRanks_InListAllCategories(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	ranks := []string{"Wolf", "Webelos"}
	catID, err := repo.CreateCategory(ctx, "Older Scouts Award", 1, nil, nil, ranks)
	if err != nil {
		t.Fatalf("CreateCategory with ranks failed: %v", err)
	}
	if catID == 0 {
		t.Fatal("expected non-zero category ID")
	}

	// Verify it was created with ranks using ListAllCategories
	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	allowedRanks, ok := categories[0]["allowed_ranks"].([]string)
	if !ok {
		t.Fatal("expected allowed_ranks to be []string")
	}
	if len(allowedRanks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(allowedRanks))
	}
	if allowedRanks[0] != "Wolf" || allowedRanks[1] != "Webelos" {
		t.Errorf("unexpected ranks: %v", allowedRanks)
	}
}

func TestCreateCategoryWithRanks_EmptyRanks(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create with empty ranks array (should allow all ranks)
	_, err := repo.CreateCategory(ctx, "Open Award", 1, nil, nil, []string{})
	if err != nil {
		t.Fatalf("CreateCategory with empty ranks failed: %v", err)
	}

	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	// When no ranks are set, the field should not be present in the map
	_, hasField := categories[0]["allowed_ranks"]
	if hasField {
		t.Error("expected allowed_ranks to not be present for empty ranks")
	}
}

func TestUpdateCategoryWithRanks_Basic(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a category without ranks
	catID, err := repo.CreateCategory(ctx, "Test Award", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Update it with ranks
	ranks := []string{"Tiger", "Wolf"}
	err = repo.UpdateCategory(ctx, int(catID), "Test Award", 1, nil, nil, ranks, true)
	if err != nil {
		t.Fatalf("UpdateCategory with ranks failed: %v", err)
	}

	// Verify ranks were set
	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	allowedRanks, ok := categories[0]["allowed_ranks"].([]string)
	if !ok {
		t.Fatal("expected allowed_ranks to be []string")
	}
	if len(allowedRanks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(allowedRanks))
	}
	if allowedRanks[0] != "Tiger" || allowedRanks[1] != "Wolf" {
		t.Errorf("unexpected ranks: %v", allowedRanks)
	}
}

func TestUpdateCategoryWithRanks_ClearRanks(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create a category with ranks
	ranks := []string{"Tiger", "Lion"}
	catID, err := repo.CreateCategory(ctx, "Test Award", 1, nil, nil, ranks)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Update it to clear ranks (empty array)
	err = repo.UpdateCategory(ctx, int(catID), "Test Award", 1, nil, nil, []string{}, true)
	if err != nil {
		t.Fatalf("UpdateCategory to clear ranks failed: %v", err)
	}

	// Verify ranks were cleared
	categories, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	_, hasField := categories[0]["allowed_ranks"]
	if hasField {
		t.Error("expected allowed_ranks to be cleared")
	}
}

func TestCategoryWithBothVoterTypesAndRanks(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	voterTypes := []string{"racer", "general"}
	ranks := []string{"Tiger", "Bear"}
	catID, err := repo.CreateCategory(ctx, "Combined Award", 1, nil, voterTypes, ranks)
	if err != nil {
		t.Fatalf("CreateCategory with both failed: %v", err)
	}
	if catID == 0 {
		t.Fatal("expected non-zero category ID")
	}

	// Verify both fields in ListCategories
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	if len(categories[0].AllowedVoterTypes) != 2 {
		t.Fatalf("expected 2 voter types, got %d", len(categories[0].AllowedVoterTypes))
	}
	if len(categories[0].AllowedRanks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(categories[0].AllowedRanks))
	}

	// Verify both fields in ListAllCategories
	allCats, err := repo.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}

	allowedTypes, ok := allCats[0]["allowed_voter_types"].([]string)
	if !ok || len(allowedTypes) != 2 {
		t.Errorf("expected 2 allowed_voter_types, got %v", allCats[0]["allowed_voter_types"])
	}

	allowedRanks, ok := allCats[0]["allowed_ranks"].([]string)
	if !ok || len(allowedRanks) != 2 {
		t.Errorf("expected 2 allowed_ranks, got %v", allCats[0]["allowed_ranks"])
	}
}

func TestListCategories_InvalidRanksJSON(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Directly insert a category with invalid JSON for allowed_ranks
	db := repo.DB()
	_, err := db.ExecContext(ctx,
		`INSERT INTO categories (name, display_order, allowed_ranks, active) VALUES (?, ?, ?, 1)`,
		"Bad Ranks JSON Category", 1, `{invalid json`)
	if err != nil {
		t.Fatalf("failed to insert test category: %v", err)
	}

	// ListCategories should return error when trying to unmarshal invalid JSON
	_, err = repo.ListCategories(ctx)
	if err == nil {
		t.Fatal("expected error from ListCategories with invalid ranks JSON, got nil")
	}
}

func TestDeleteVoter_WithVotes(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create voter
	voterID, err := repo.CreateVoter(ctx, "VOTER-WITH-VOTES")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	// Create car
	err = repo.CreateCar(ctx, "100", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Create category
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Save vote
	err = repo.SaveVote(ctx, voterID, int(catID), carID)
	if err != nil {
		t.Fatalf("SaveVote failed: %v", err)
	}

	// Verify vote exists
	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}
	if len(votes) != 1 {
		t.Fatalf("expected 1 vote, got %d", len(votes))
	}

	// Delete voter (should also delete their votes)
	err = repo.DeleteVoter(ctx, voterID)
	if err != nil {
		t.Fatalf("DeleteVoter failed: %v", err)
	}

	// Verify voter is deleted
	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 0 {
		t.Errorf("expected 0 voters after delete, got %d", len(voters))
	}

	// Verify votes are also deleted
	votes, err = repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}
	if len(votes) != 0 {
		t.Errorf("expected 0 votes after voter delete, got %d", len(votes))
	}
}

func TestCountVotesForCar(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	ctx := context.Background()

	// Create voter
	voterID, err := repo.CreateVoter(ctx, "QR123")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	// Create car
	err = repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Create categories
	catID1, err := repo.CreateCategory(ctx, "Category 1", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	catID2, err := repo.CreateCategory(ctx, "Category 2", 2, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Initially, car should have 0 votes
	count, err := repo.CountVotesForCar(ctx, carID)
	if err != nil {
		t.Fatalf("CountVotesForCar failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 votes, got %d", count)
	}

	// Add first vote
	err = repo.SaveVote(ctx, voterID, int(catID1), carID)
	if err != nil {
		t.Fatalf("SaveVote failed: %v", err)
	}

	// Should have 1 vote now
	count, err = repo.CountVotesForCar(ctx, carID)
	if err != nil {
		t.Fatalf("CountVotesForCar failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 vote, got %d", count)
	}

	// Add second vote from same voter in different category
	err = repo.SaveVote(ctx, voterID, int(catID2), carID)
	if err != nil {
		t.Fatalf("SaveVote failed: %v", err)
	}

	// Should have 2 votes now
	count, err = repo.CountVotesForCar(ctx, carID)
	if err != nil {
		t.Fatalf("CountVotesForCar failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 votes, got %d", count)
	}
}

func TestCountVotesForCategory(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	ctx := context.Background()

	// Create voter
	voterID, err := repo.CreateVoter(ctx, "QR123")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	// Create cars
	err = repo.CreateCar(ctx, "101", "Test Racer 1", "Test Car 1", "")
	if err != nil {
		t.Fatalf("CreateCar 1 failed: %v", err)
	}
	err = repo.CreateCar(ctx, "102", "Test Racer 2", "Test Car 2", "")
	if err != nil {
		t.Fatalf("CreateCar 2 failed: %v", err)
	}
	cars, _ := repo.ListCars(ctx)
	carID1 := cars[0].ID
	carID2 := cars[1].ID

	// Create category
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Initially, category should have 0 votes
	count, err := repo.CountVotesForCategory(ctx, int(catID))
	if err != nil {
		t.Fatalf("CountVotesForCategory failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 votes, got %d", count)
	}

	// Add first vote
	err = repo.SaveVote(ctx, voterID, int(catID), carID1)
	if err != nil {
		t.Fatalf("SaveVote 1 failed: %v", err)
	}

	// Should have 1 vote now
	count, err = repo.CountVotesForCategory(ctx, int(catID))
	if err != nil {
		t.Fatalf("CountVotesForCategory failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 vote, got %d", count)
	}

	// Create second voter
	voterID2, err := repo.CreateVoter(ctx, "QR456")
	if err != nil {
		t.Fatalf("CreateVoter 2 failed: %v", err)
	}

	// Add vote from second voter for different car in same category
	err = repo.SaveVote(ctx, voterID2, int(catID), carID2)
	if err != nil {
		t.Fatalf("SaveVote 2 failed: %v", err)
	}

	// Should have 2 votes now
	count, err = repo.CountVotesForCategory(ctx, int(catID))
	if err != nil {
		t.Fatalf("CountVotesForCategory failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 votes, got %d", count)
	}
}
