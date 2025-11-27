package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/models"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/internal/repository/mock"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/testutil"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// setupVotingService creates a VotingService with all dependencies for testing
func setupVotingService(t *testing.T) (*services.VotingService, *services.CategoryService, *services.CarService, *services.SettingsService, *repository.Repository) {
	t.Helper()
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, repo, derbynetClient)
	carSvc := services.NewCarService(log, repo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, repo)
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)
	return votingSvc, categorySvc, carSvc, settingsSvc, repo
}

// TestGetOrCreateVoter_CreatesNewVoter tests that a new voter is created if not exists
func TestGetOrCreateVoter_CreatesNewVoter(t *testing.T) {
	votingSvc, _, _, _, _ := setupVotingService(t)
	ctx := context.Background()

	qrCode := "XX-YYY"
	voterID, err := votingSvc.GetOrCreateVoter(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetOrCreateVoter failed: %v", err)
	}

	if voterID <= 0 {
		t.Errorf("expected positive voter ID, got %d", voterID)
	}
}

// TestGetOrCreateVoter_ReturnsExistingVoter tests that an existing voter is returned
func TestGetOrCreateVoter_ReturnsExistingVoter(t *testing.T) {
	votingSvc, _, _, _, _ := setupVotingService(t)
	ctx := context.Background()

	qrCode := "AA-BBB"

	// First call creates the voter
	voterID1, err := votingSvc.GetOrCreateVoter(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetOrCreateVoter first call failed: %v", err)
	}

	// Second call should return the same voter
	voterID2, err := votingSvc.GetOrCreateVoter(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetOrCreateVoter second call failed: %v", err)
	}

	if voterID1 != voterID2 {
		t.Errorf("expected same voter ID, got %d and %d", voterID1, voterID2)
	}
}

// TestGetVoteData_ReturnsCategoriesAndCars tests that categories and cars are returned
func TestGetVoteData_ReturnsCategoriesAndCars(t *testing.T) {
	votingSvc, _, _, _, repo := setupVotingService(t)
	ctx := context.Background()

	// Create test categories
	catID1, err := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	catID2, err := repo.CreateCategory(ctx, "Most Creative", 2, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create test cars
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "http://example.com/photo1.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	err = repo.CreateCar(ctx, "102", "Jane Doe", "Speed Demon", "http://example.com/photo2.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get vote data
	qrCode := "CC-DDD"
	voteData, err := votingSvc.GetVoteData(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Verify categories
	if len(voteData.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(voteData.Categories))
	}

	// Verify the categories have correct IDs
	foundCat1 := false
	foundCat2 := false
	for _, cat := range voteData.Categories {
		if cat.ID == int(catID1) && cat.Name == "Best Design" {
			foundCat1 = true
		}
		if cat.ID == int(catID2) && cat.Name == "Most Creative" {
			foundCat2 = true
		}
	}
	if !foundCat1 || !foundCat2 {
		t.Error("expected to find both created categories")
	}

	// Verify cars
	if len(voteData.Cars) != 2 {
		t.Errorf("expected 2 cars, got %d", len(voteData.Cars))
	}

	// Verify votes map is initialized but empty for new voter
	if voteData.Votes == nil {
		t.Error("expected Votes map to be initialized")
	}
	if len(voteData.Votes) != 0 {
		t.Errorf("expected 0 votes for new voter, got %d", len(voteData.Votes))
	}
}

// TestGetVoteData_ReturnsExistingVotes tests that existing votes are returned for a voter
func TestGetVoteData_ReturnsExistingVotes(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create test category
	catID, err := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create test car
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get car ID (it will be 1 since it's the first car)
	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) == 0 {
		t.Fatal("expected at least one car")
	}
	carID := cars[0].ID

	// Submit a vote first
	qrCode := "EE-FFF"
	vote := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID),
		CarID:      carID,
	}
	_, err = votingSvc.SubmitVote(ctx, vote)
	if err != nil {
		t.Fatalf("SubmitVote failed: %v", err)
	}

	// Now get vote data
	voteData, err := votingSvc.GetVoteData(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Verify the vote is returned
	if len(voteData.Votes) != 1 {
		t.Errorf("expected 1 vote, got %d", len(voteData.Votes))
	}

	if voteData.Votes[int(catID)] != carID {
		t.Errorf("expected vote for category %d to be car %d, got %d", catID, carID, voteData.Votes[int(catID)])
	}
}

// TestSubmitVote_RecordsNewVote tests that a new vote is recorded successfully
func TestSubmitVote_RecordsNewVote(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create test category
	catID, err := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create test car
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	// Submit vote
	qrCode := "GG-HHH"
	vote := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID),
		CarID:      carID,
	}

	result, err := votingSvc.SubmitVote(ctx, vote)
	if err != nil {
		t.Fatalf("SubmitVote failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}

	if result.Message != "Vote recorded" {
		t.Errorf("expected message 'Vote recorded', got %q", result.Message)
	}

	if result.ConflictCleared {
		t.Error("expected ConflictCleared to be false for new vote")
	}

	// Verify the vote was saved
	voterID, err := repo.GetVoterByQR(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}

	if votes[int(catID)] != carID {
		t.Errorf("expected vote for category %d to be car %d, got %d", catID, carID, votes[int(catID)])
	}
}

// TestSubmitVote_UpdatesExistingVote tests that voting for a different car in the same category updates the vote
func TestSubmitVote_UpdatesExistingVote(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create test category
	catID, err := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create two test cars
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}
	err = repo.CreateCar(ctx, "102", "Jane Doe", "Speed Demon", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID1 := cars[0].ID
	carID2 := cars[1].ID

	qrCode := "II-JJJ"

	// Vote for first car
	vote1 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID),
		CarID:      carID1,
	}
	_, err = votingSvc.SubmitVote(ctx, vote1)
	if err != nil {
		t.Fatalf("SubmitVote first car failed: %v", err)
	}

	// Vote for second car (same category)
	vote2 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID),
		CarID:      carID2,
	}
	result, err := votingSvc.SubmitVote(ctx, vote2)
	if err != nil {
		t.Fatalf("SubmitVote second car failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}

	// Verify the vote was updated to the second car
	voterID, err := repo.GetVoterByQR(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}

	if votes[int(catID)] != carID2 {
		t.Errorf("expected vote to be updated to car %d, got %d", carID2, votes[int(catID)])
	}
}

// TestSubmitVote_HandlesDeselection tests that voting with carID=0 removes the vote
func TestSubmitVote_HandlesDeselection(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create test category
	catID, err := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create test car
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	qrCode := "KK-LLL"

	// First, submit a vote
	vote := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID),
		CarID:      carID,
	}
	_, err = votingSvc.SubmitVote(ctx, vote)
	if err != nil {
		t.Fatalf("SubmitVote failed: %v", err)
	}

	// Verify vote exists
	voterID, err := repo.GetVoterByQR(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}
	if len(votes) != 1 {
		t.Errorf("expected 1 vote before deselection, got %d", len(votes))
	}

	// Now deselect (vote with carID = 0)
	deselect := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID),
		CarID:      0,
	}
	result, err := votingSvc.SubmitVote(ctx, deselect)
	if err != nil {
		t.Fatalf("SubmitVote deselect failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}

	// Verify the vote was removed
	votes, err = repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes after deselect failed: %v", err)
	}

	if len(votes) != 0 {
		t.Errorf("expected 0 votes after deselection, got %d", len(votes))
	}
}

// TestSubmitVote_HandlesExclusivityConflicts tests the exclusivity pool conflict handling
func TestSubmitVote_HandlesExclusivityConflicts(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create a category group with an exclusivity pool
	poolID := 1
	groupID, err := repo.CreateCategoryGroup(ctx, "Speed Awards", "Speed related categories", &poolID, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	// Create two categories in the same exclusivity pool (via the group)
	groupIDInt := int(groupID)
	catID1, err := repo.CreateCategory(ctx, "Fastest Looking", 1, &groupIDInt, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}

	catID2, err := repo.CreateCategory(ctx, "Most Aerodynamic", 2, &groupIDInt, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Create a test car
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	qrCode := "MM-NNN"

	// Vote for the car in first category
	vote1 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID1),
		CarID:      carID,
	}
	result1, err := votingSvc.SubmitVote(ctx, vote1)
	if err != nil {
		t.Fatalf("SubmitVote first category failed: %v", err)
	}

	if result1.ConflictCleared {
		t.Error("expected no conflict on first vote")
	}

	// Vote for the SAME car in second category (should trigger conflict)
	vote2 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID2),
		CarID:      carID,
	}
	result2, err := votingSvc.SubmitVote(ctx, vote2)
	if err != nil {
		t.Fatalf("SubmitVote second category failed: %v", err)
	}

	// Verify conflict was detected and cleared
	if !result2.ConflictCleared {
		t.Error("expected ConflictCleared to be true")
	}

	if result2.ConflictCategoryID != int(catID1) {
		t.Errorf("expected conflict category ID %d, got %d", catID1, result2.ConflictCategoryID)
	}

	if result2.ConflictCategoryName != "Fastest Looking" {
		t.Errorf("expected conflict category name 'Fastest Looking', got %q", result2.ConflictCategoryName)
	}

	// Verify the vote in the first category was cleared
	voterID, err := repo.GetVoterByQR(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}

	// Should only have vote in second category now
	if len(votes) != 1 {
		t.Errorf("expected 1 vote after conflict resolution, got %d", len(votes))
	}

	if votes[int(catID1)] != 0 {
		t.Errorf("expected vote in category %d to be cleared, but it's %d", catID1, votes[int(catID1)])
	}

	if votes[int(catID2)] != carID {
		t.Errorf("expected vote in category %d to be car %d, got %d", catID2, carID, votes[int(catID2)])
	}
}

// TestSubmitVote_NoConflictDifferentCars tests that voting for different cars in same pool is allowed
func TestSubmitVote_NoConflictDifferentCars(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create a category group with an exclusivity pool
	poolID := 1
	groupID, err := repo.CreateCategoryGroup(ctx, "Speed Awards", "Speed related categories", &poolID, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	// Create two categories in the same exclusivity pool
	groupIDInt := int(groupID)
	catID1, err := repo.CreateCategory(ctx, "Fastest Looking", 1, &groupIDInt, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}

	catID2, err := repo.CreateCategory(ctx, "Most Aerodynamic", 2, &groupIDInt, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Create two test cars
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	if err != nil {
		t.Fatalf("CreateCar 1 failed: %v", err)
	}
	err = repo.CreateCar(ctx, "102", "Jane Doe", "Speed Demon", "")
	if err != nil {
		t.Fatalf("CreateCar 2 failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID1 := cars[0].ID
	carID2 := cars[1].ID

	qrCode := "OO-PPP"

	// Vote for car1 in first category
	vote1 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID1),
		CarID:      carID1,
	}
	_, err = votingSvc.SubmitVote(ctx, vote1)
	if err != nil {
		t.Fatalf("SubmitVote first category failed: %v", err)
	}

	// Vote for DIFFERENT car in second category (should NOT trigger conflict)
	vote2 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID2),
		CarID:      carID2,
	}
	result2, err := votingSvc.SubmitVote(ctx, vote2)
	if err != nil {
		t.Fatalf("SubmitVote second category failed: %v", err)
	}

	// No conflict should occur
	if result2.ConflictCleared {
		t.Error("expected no conflict when voting for different cars")
	}

	// Both votes should exist
	voterID, err := repo.GetVoterByQR(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}

	if len(votes) != 2 {
		t.Errorf("expected 2 votes, got %d", len(votes))
	}

	if votes[int(catID1)] != carID1 {
		t.Errorf("expected vote in category %d to be car %d, got %d", catID1, carID1, votes[int(catID1)])
	}

	if votes[int(catID2)] != carID2 {
		t.Errorf("expected vote in category %d to be car %d, got %d", catID2, carID2, votes[int(catID2)])
	}
}

// TestSubmitVote_NoConflictNoPool tests that categories without exclusivity pool don't conflict
func TestSubmitVote_NoConflictNoPool(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create two categories WITHOUT a group (no exclusivity pool)
	catID1, err := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}

	catID2, err := repo.CreateCategory(ctx, "Most Creative", 2, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Create a test car
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	qrCode := "QQ-RRR"

	// Vote for same car in both categories
	vote1 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID1),
		CarID:      carID,
	}
	_, err = votingSvc.SubmitVote(ctx, vote1)
	if err != nil {
		t.Fatalf("SubmitVote first category failed: %v", err)
	}

	vote2 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID2),
		CarID:      carID,
	}
	result2, err := votingSvc.SubmitVote(ctx, vote2)
	if err != nil {
		t.Fatalf("SubmitVote second category failed: %v", err)
	}

	// No conflict should occur (no exclusivity pool)
	if result2.ConflictCleared {
		t.Error("expected no conflict when categories have no exclusivity pool")
	}

	// Both votes should exist
	voterID, err := repo.GetVoterByQR(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}

	if len(votes) != 2 {
		t.Errorf("expected 2 votes, got %d", len(votes))
	}
}

// TestSubmitVote_MultipleVotersIndependent tests that different voters don't affect each other
func TestSubmitVote_MultipleVotersIndependent(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create a category group with an exclusivity pool
	poolID := 1
	groupID, err := repo.CreateCategoryGroup(ctx, "Speed Awards", "Speed related categories", &poolID, nil, 1)
	if err != nil {
		t.Fatalf("CreateCategoryGroup failed: %v", err)
	}

	// Create two categories in the same exclusivity pool
	groupIDInt := int(groupID)
	catID1, err := repo.CreateCategory(ctx, "Fastest Looking", 1, &groupIDInt, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}

	catID2, err := repo.CreateCategory(ctx, "Most Aerodynamic", 2, &groupIDInt, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Create a test car
	err = repo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	// Voter 1 votes for car in category 1
	voter1QR := "SS-TTT"
	vote1 := models.Vote{
		VoterQR:    voter1QR,
		CategoryID: int(catID1),
		CarID:      carID,
	}
	_, err = votingSvc.SubmitVote(ctx, vote1)
	if err != nil {
		t.Fatalf("Voter1 SubmitVote failed: %v", err)
	}

	// Voter 2 votes for same car in category 2
	voter2QR := "UU-VVV"
	vote2 := models.Vote{
		VoterQR:    voter2QR,
		CategoryID: int(catID2),
		CarID:      carID,
	}
	result2, err := votingSvc.SubmitVote(ctx, vote2)
	if err != nil {
		t.Fatalf("Voter2 SubmitVote failed: %v", err)
	}

	// No conflict for voter 2 (different voter)
	if result2.ConflictCleared {
		t.Error("expected no conflict for different voter")
	}

	// Verify voter 1's vote is still intact
	voter1ID, err := repo.GetVoterByQR(ctx, voter1QR)
	if err != nil {
		t.Fatalf("GetVoterByQR voter1 failed: %v", err)
	}

	votes1, err := repo.GetVoterVotes(ctx, voter1ID)
	if err != nil {
		t.Fatalf("GetVoterVotes voter1 failed: %v", err)
	}

	if votes1[int(catID1)] != carID {
		t.Error("voter1's vote should not be affected by voter2's vote")
	}
}

// TestGetVoteData_EmptyDatabase tests GetVoteData with no data
func TestGetVoteData_EmptyDatabase(t *testing.T) {
	votingSvc, _, _, _, _ := setupVotingService(t)
	ctx := context.Background()

	qrCode := "WW-XXX"
	voteData, err := votingSvc.GetVoteData(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Should return empty slices/maps, not nil
	if voteData.Categories == nil {
		// Categories can be nil when empty, that's OK
	}
	if voteData.Cars == nil {
		// Cars can be nil when empty, that's OK
	}
	if voteData.Votes == nil {
		t.Error("expected Votes map to be initialized (not nil)")
	}

	if len(voteData.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(voteData.Categories))
	}
	if len(voteData.Cars) != 0 {
		t.Errorf("expected 0 cars, got %d", len(voteData.Cars))
	}
	if len(voteData.Votes) != 0 {
		t.Errorf("expected 0 votes, got %d", len(voteData.Votes))
	}
}

// TestSubmitVote_VotingClosed tests that voting is rejected when voting is closed
func TestSubmitVote_VotingClosed(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Explicitly close voting (default is open)
	settingsSvc.CloseVoting(ctx)

	// Create test category
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create test car
	err = repo.CreateCar(ctx, "201", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	// Try to submit vote while voting is closed
	qrCode := "CLOSED-TEST"
	vote := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID),
		CarID:      carID,
	}

	result, err := votingSvc.SubmitVote(ctx, vote)

	// Should fail with ErrVotingClosed
	if err != services.ErrVotingClosed {
		t.Errorf("expected ErrVotingClosed, got: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when voting is closed")
	}

	// Now open voting and verify vote succeeds
	settingsSvc.OpenVoting(ctx)

	result, err = votingSvc.SubmitVote(ctx, vote)
	if err != nil {
		t.Fatalf("SubmitVote failed after opening voting: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

// TestGetOrCreateVoter_MultipleUniqueCodes tests creating multiple voters with unique codes
func TestGetOrCreateVoter_MultipleUniqueCodes(t *testing.T) {
	votingSvc, _, _, _, _ := setupVotingService(t)
	ctx := context.Background()

	codes := []string{"AA-111", "BB-222", "CC-333"}
	voterIDs := make(map[string]int)

	for _, code := range codes {
		id, err := votingSvc.GetOrCreateVoter(ctx, code)
		if err != nil {
			t.Fatalf("GetOrCreateVoter(%s) failed: %v", code, err)
		}
		voterIDs[code] = id
	}

	// All IDs should be unique
	seen := make(map[int]bool)
	for code, id := range voterIDs {
		if seen[id] {
			t.Errorf("duplicate voter ID %d for code %s", id, code)
		}
		seen[id] = true
	}
}

// ==================== Car Eligibility Tests ====================

// TestSubmitVote_IneligibleCar tests that voting for an ineligible car is rejected
func TestSubmitVote_IneligibleCar(t *testing.T) {
	votingSvc, _, _, _, repo := setupVotingService(t)
	ctx := context.Background()

	// Create test category
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create test car and make it ineligible
	err = repo.CreateCar(ctx, "301", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set car as ineligible
	err = repo.SetCarEligibility(ctx, carID, false)
	if err != nil {
		t.Fatalf("SetCarEligibility failed: %v", err)
	}

	// Try to vote for the ineligible car
	vote := models.Vote{
		VoterQR:    "INELIG-QR1",
		CategoryID: int(catID),
		CarID:      carID,
	}

	_, err = votingSvc.SubmitVote(ctx, vote)
	if err != services.ErrCarNotEligible {
		t.Errorf("expected ErrCarNotEligible, got: %v", err)
	}
}

// TestSubmitVote_EligibleCar tests that voting for an eligible car succeeds
func TestSubmitVote_EligibleCar(t *testing.T) {
	votingSvc, _, _, _, repo := setupVotingService(t)
	ctx := context.Background()

	// Create test category
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create test car (eligible by default)
	err = repo.CreateCar(ctx, "302", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Vote for the eligible car
	vote := models.Vote{
		VoterQR:    "ELIG-QR1",
		CategoryID: int(catID),
		CarID:      carID,
	}

	result, err := votingSvc.SubmitVote(ctx, vote)
	if err != nil {
		t.Fatalf("SubmitVote failed: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

// TestGetVoteData_FiltersIneligibleCars tests that GetVoteData excludes ineligible cars
func TestGetVoteData_FiltersIneligibleCars(t *testing.T) {
	votingSvc, _, _, _, repo := setupVotingService(t)
	ctx := context.Background()

	// Create test category
	_, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create 3 cars
	err = repo.CreateCar(ctx, "401", "Racer 1", "Car 1", "")
	if err != nil {
		t.Fatalf("CreateCar 1 failed: %v", err)
	}
	err = repo.CreateCar(ctx, "402", "Racer 2", "Car 2", "")
	if err != nil {
		t.Fatalf("CreateCar 2 failed: %v", err)
	}
	err = repo.CreateCar(ctx, "403", "Racer 3", "Car 3", "")
	if err != nil {
		t.Fatalf("CreateCar 3 failed: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	if len(cars) != 3 {
		t.Fatalf("expected 3 cars, got %d", len(cars))
	}

	// Make one car ineligible
	err = repo.SetCarEligibility(ctx, cars[1].ID, false)
	if err != nil {
		t.Fatalf("SetCarEligibility failed: %v", err)
	}

	// Get vote data
	voteData, err := votingSvc.GetVoteData(ctx, "FILTER-QR")
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Should only have 2 cars (the eligible ones)
	if len(voteData.Cars) != 2 {
		t.Errorf("expected 2 eligible cars in vote data, got %d", len(voteData.Cars))
	}

	// Verify the ineligible car is not in the list
	for _, car := range voteData.Cars {
		if car.CarNumber == "402" {
			t.Error("expected ineligible car to be excluded from vote data")
		}
	}
}

// TestSubmitVote_CarNotFound tests that voting for a non-existent car fails
func TestSubmitVote_CarNotFound(t *testing.T) {
	votingSvc, _, _, _, repo := setupVotingService(t)
	ctx := context.Background()

	// Create test category
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Try to vote for a non-existent car
	vote := models.Vote{
		VoterQR:    "NOTFOUND-QR",
		CategoryID: int(catID),
		CarID:      99999,
	}

	_, err = votingSvc.SubmitVote(ctx, vote)
	if err != services.ErrCarNotFound {
		t.Errorf("expected ErrCarNotFound, got: %v", err)
	}
}

// TestGetOrCreateVoter_RequireRegisteredQR_RejectsUnregistered tests that unregistered QR codes are rejected when the setting is enabled
func TestGetOrCreateVoter_RequireRegisteredQR_RejectsUnregistered(t *testing.T) {
	votingSvc, _, _, settingsSvc, _ := setupVotingService(t)
	ctx := context.Background()

	// Enable the require_registered_qr setting
	err := settingsSvc.SetRequireRegisteredQR(ctx, true)
	if err != nil {
		t.Fatalf("SetRequireRegisteredQR failed: %v", err)
	}

	// Try to use an unregistered QR code
	_, err = votingSvc.GetOrCreateVoter(ctx, "UNREG-QR-123")
	if err != services.ErrUnregisteredQR {
		t.Errorf("expected ErrUnregisteredQR, got: %v", err)
	}
}

// TestGetOrCreateVoter_RequireRegisteredQR_AllowsRegistered tests that registered QR codes work when the setting is enabled
func TestGetOrCreateVoter_RequireRegisteredQR_AllowsRegistered(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// First create a voter with this QR code (before enabling the setting)
	qrCode := "PREREG-QR-456"
	_, err := repo.CreateVoter(ctx, qrCode)
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	// Now enable the require_registered_qr setting
	err = settingsSvc.SetRequireRegisteredQR(ctx, true)
	if err != nil {
		t.Fatalf("SetRequireRegisteredQR failed: %v", err)
	}

	// Try to use the pre-registered QR code - should succeed
	voterID, err := votingSvc.GetOrCreateVoter(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetOrCreateVoter failed for registered QR: %v", err)
	}
	if voterID <= 0 {
		t.Errorf("expected positive voter ID, got %d", voterID)
	}
}

// TestGetOrCreateVoter_RequireRegisteredQR_DisabledAllowsAny tests that any QR code works when the setting is disabled
func TestGetOrCreateVoter_RequireRegisteredQR_DisabledAllowsAny(t *testing.T) {
	votingSvc, _, _, settingsSvc, _ := setupVotingService(t)
	ctx := context.Background()

	// Ensure the setting is disabled (default)
	err := settingsSvc.SetRequireRegisteredQR(ctx, false)
	if err != nil {
		t.Fatalf("SetRequireRegisteredQR failed: %v", err)
	}

	// Try to use a new unregistered QR code - should auto-create
	voterID, err := votingSvc.GetOrCreateVoter(ctx, "NEW-QR-789")
	if err != nil {
		t.Fatalf("GetOrCreateVoter failed: %v", err)
	}
	if voterID <= 0 {
		t.Errorf("expected positive voter ID, got %d", voterID)
	}
}

// TestSubmitVote_RequireRegisteredQR_RejectsUnregistered tests that votes from unregistered QR codes are rejected
func TestSubmitVote_RequireRegisteredQR_RejectsUnregistered(t *testing.T) {
	votingSvc, _, _, settingsSvc, repo := setupVotingService(t)
	ctx := context.Background()

	// Open voting
	settingsSvc.OpenVoting(ctx)

	// Enable the require_registered_qr setting
	err := settingsSvc.SetRequireRegisteredQR(ctx, true)
	if err != nil {
		t.Fatalf("SetRequireRegisteredQR failed: %v", err)
	}

	// Create a category and car
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	err = repo.CreateCar(ctx, "101", "John Smith", "Test Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}

	// Try to submit a vote with unregistered QR code
	vote := models.Vote{
		VoterQR:    "UNREG-VOTE-QR",
		CategoryID: int(catID),
		CarID:      cars[0].ID,
	}

	_, err = votingSvc.SubmitVote(ctx, vote)
	if err != services.ErrUnregisteredQR {
		t.Errorf("expected ErrUnregisteredQR, got: %v", err)
	}
}

// TestGetVoteData_RequireRegisteredQR_RejectsUnregistered tests that GetVoteData rejects unregistered QR codes
func TestGetVoteData_RequireRegisteredQR_RejectsUnregistered(t *testing.T) {
	votingSvc, _, _, settingsSvc, _ := setupVotingService(t)
	ctx := context.Background()

	// Enable the require_registered_qr setting
	err := settingsSvc.SetRequireRegisteredQR(ctx, true)
	if err != nil {
		t.Fatalf("SetRequireRegisteredQR failed: %v", err)
	}

	// Try to get vote data with unregistered QR code
	_, err = votingSvc.GetVoteData(ctx, "UNREG-DATA-QR")
	if err != services.ErrUnregisteredQR {
		t.Errorf("expected ErrUnregisteredQR, got: %v", err)
	}
}

// ===== Error Path Tests =====

func TestGetVoteData_GetVoterTypeError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetVoterTypeError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()
	_, err := votingSvc.GetVoteData(ctx, "TEST-QR")
	if err == nil {
		t.Fatal("expected error from GetVoteData when GetVoterType fails, got nil")
	}
}

func TestGetVoteData_ListCategoriesError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoriesError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()
	_, err := votingSvc.GetVoteData(ctx, "TEST-QR")
	if err == nil {
		t.Fatal("expected error from GetVoteData when ListCategories fails, got nil")
	}
}

func TestGetVoteData_ListEligibleCarsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListEligibleCarsError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()
	_, err := votingSvc.GetVoteData(ctx, "TEST-QR")
	if err == nil {
		t.Fatal("expected error from GetVoteData when ListEligibleCars fails, got nil")
	}
}

func TestGetVoteData_GetVoterVotesError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetVoterVotesError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()
	_, err := votingSvc.GetVoteData(ctx, "TEST-QR")
	if err == nil {
		t.Fatal("expected error from GetVoteData when GetVoterVotes fails, got nil")
	}
}


func TestSubmitVote_IsVotingOpenError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetSettingError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()
	vote := models.Vote{
		VoterQR:    "TEST-QR",
		CategoryID: 1,
		CarID:      1,
	}
	_, err := votingSvc.SubmitVote(ctx, vote)
	if err == nil {
		t.Fatal("expected error from SubmitVote when IsVotingOpen check fails, got nil")
	}
}

func TestSubmitVote_SaveVoteError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.SaveVoteError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()

	// Open voting first
	settingsSvc.OpenVoting(ctx)

	// Create test category and car
	catID, _ := realRepo.CreateCategory(ctx, "Test Cat", 1, nil, nil, nil)
	realRepo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	cars, _ := realRepo.ListCars(ctx)

	vote := models.Vote{
		VoterQR:    "TEST-QR",
		CategoryID: int(catID),
		CarID:      cars[0].ID,
	}
	_, err := votingSvc.SubmitVote(ctx, vote)
	if err == nil {
		t.Fatal("expected error from SubmitVote when SaveVote fails, got nil")
	}
}

func TestCheckExclusivityConflict_GetExclusivityPoolIDError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetExclusivityPoolIDError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()

	// Open voting first
	settingsSvc.OpenVoting(ctx)

	// Create test category and car
	catID, _ := realRepo.CreateCategory(ctx, "Test Cat", 1, nil, nil, nil)
	realRepo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	cars, _ := realRepo.ListCars(ctx)

	vote := models.Vote{
		VoterQR:    "TEST-QR",
		CategoryID: int(catID),
		CarID:      cars[0].ID,
	}
	_, err := votingSvc.SubmitVote(ctx, vote)
	if err == nil {
		t.Fatal("expected error from SubmitVote when GetExclusivityPoolID fails, got nil")
	}
}

func TestSubmitVote_GetCarError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetCarError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()

	// Open voting first
	settingsSvc.OpenVoting(ctx)

	// Create test category and car
	catID, _ := realRepo.CreateCategory(ctx, "Test Cat", 1, nil, nil, nil)
	realRepo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	cars, _ := realRepo.ListCars(ctx)

	vote := models.Vote{
		VoterQR:    "TEST-QR",
		CategoryID: int(catID),
		CarID:      cars[0].ID,
	}
	_, err := votingSvc.SubmitVote(ctx, vote)
	if err == nil {
		t.Fatal("expected error from SubmitVote when GetCar fails, got nil")
	}
}

func TestSubmitVote_ClearConflictingVoteError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()

	// Open voting first
	settingsSvc.OpenVoting(ctx)

	// Create a category group with an exclusivity pool
	poolID := 1
	groupID, _ := realRepo.CreateCategoryGroup(ctx, "Speed Awards", "Speed related categories", &poolID, nil, 1)

	// Create two categories in the same exclusivity pool (via the group)
	groupIDInt := int(groupID)
	catID1, _ := realRepo.CreateCategory(ctx, "Fastest Looking", 1, &groupIDInt, nil, nil)
	catID2, _ := realRepo.CreateCategory(ctx, "Most Aerodynamic", 2, &groupIDInt, nil, nil)

	// Create a test car
	realRepo.CreateCar(ctx, "101", "John Smith", "Lightning Bolt", "")
	cars, _ := realRepo.ListCars(ctx)
	carID := cars[0].ID

	qrCode := "CONFLICT-ERR-QR"

	// Vote for the car in first category
	vote1 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID1),
		CarID:      carID,
	}
	votingSvc.SubmitVote(ctx, vote1)

	// Now inject error for clearing conflicting vote
	mockRepo.ClearConflictingVoteError = errors.New("database error")

	// Vote for the SAME car in second category (should trigger conflict and error)
	vote2 := models.Vote{
		VoterQR:    qrCode,
		CategoryID: int(catID2),
		CarID:      carID,
	}
	_, err := votingSvc.SubmitVote(ctx, vote2)
	if err == nil {
		t.Fatal("expected error from SubmitVote when ClearConflictingVote fails, got nil")
	}
}

func TestGetOrCreateVoter_CreateVoterError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.CreateVoterError = errors.New("database error")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()

	_, err := votingSvc.GetOrCreateVoter(ctx, "NEW-QR-ERROR")
	if err == nil {
		t.Fatal("expected error from GetOrCreateVoter when CreateVoter fails, got nil")
	}
}

// TestGetOrCreateVoter_RequireRegisteredQRError tests that database errors from RequireRegisteredQR are propagated
func TestGetOrCreateVoter_RequireRegisteredQRError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)

	// Inject a non-ErrNotFound error for GetSetting
	// This simulates a real database error (connection lost, disk full, etc.)
	mockRepo.GetSettingError = errors.New("database connection lost")

	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, mockRepo, derbynetClient)
	carSvc := services.NewCarService(log, mockRepo, derbynetClient)
	settingsSvc := services.NewSettingsService(log, mockRepo)
	votingSvc := services.NewVotingService(log, mockRepo, categorySvc, carSvc, settingsSvc)

	ctx := context.Background()

	// Try to get/create voter with an unregistered QR code
	// This will trigger: GetVoterByQR -> ErrNotFound -> RequireRegisteredQR -> database error
	_, err := votingSvc.GetOrCreateVoter(ctx, "UNREGISTERED-QR")
	if err == nil {
		t.Fatal("expected error from GetOrCreateVoter when RequireRegisteredQR fails with DB error, got nil")
	}
	if err.Error() != "database connection lost" {
		t.Errorf("expected 'database connection lost' error, got: %v", err)
	}
}

func TestGetVoteData_IncludesInstructions(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	
	ctx := context.Background()
	
	// Create services
	settingsService := services.NewSettingsService(log, repo)
	categoryService := services.NewCategoryService(log, repo, nil)
	carService := services.NewCarService(log, repo, nil)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)
	
	// Create test data
	repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	repo.CreateCar(ctx, "101", "Racer 1", "Car 1", "")
	repo.CreateVoter(ctx, "TEST-QR")
	
	// Set voting instructions
	instructions := "Please vote carefully!\nEach vote counts."
	err := settingsService.SetSetting(ctx, "voting_instructions", instructions)
	if err != nil {
		t.Fatalf("failed to set instructions: %v", err)
	}
	
	// Get vote data
	voteData, err := votingService.GetVoteData(ctx, "TEST-QR")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	
	// Verify instructions are included
	if voteData.Instructions != instructions {
		t.Errorf("expected instructions '%s', got '%s'", instructions, voteData.Instructions)
	}
}

func TestGetVoteData_NoInstructions(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()

	ctx := context.Background()

	// Create services
	settingsService := services.NewSettingsService(log, repo)
	categoryService := services.NewCategoryService(log, repo, nil)
	carService := services.NewCarService(log, repo, nil)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)

	// Create test data
	repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	repo.CreateCar(ctx, "101", "Racer 1", "Car 1", "")
	repo.CreateVoter(ctx, "TEST-QR")

	// Don't set any instructions

	// Get vote data
	voteData, err := votingService.GetVoteData(ctx, "TEST-QR")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify instructions are empty
	if voteData.Instructions != "" {
		t.Errorf("expected empty instructions, got '%s'", voteData.Instructions)
	}
}

// ==================== Voter Type Filtering Tests ====================

// TestGetVoteData_FiltersCategoriesByVoterType_NoRestrictions tests that categories with no allowed_voter_types are shown to all voters
func TestGetVoteData_FiltersCategoriesByVoterType_NoRestrictions(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create services
	settingsService := services.NewSettingsService(log, repo)
	categoryService := services.NewCategoryService(log, repo, nil)
	carService := services.NewCarService(log, repo, nil)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)

	// Create categories with no voter type restrictions
	catID1, err := repo.CreateCategory(ctx, "Open Category 1", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}
	catID2, err := repo.CreateCategory(ctx, "Open Category 2", 2, nil, []string{}, nil)
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Create a voter with type "racer"
	voterID, err := repo.CreateVoterFull(ctx, nil, "Test Racer", "", "racer", "RACER-QR", "")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}

	// Get vote data
	voteData, err := votingService.GetVoteData(ctx, "RACER-QR")
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Should see both categories
	if len(voteData.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(voteData.Categories))
	}

	foundCat1 := false
	foundCat2 := false
	for _, cat := range voteData.Categories {
		if cat.ID == int(catID1) {
			foundCat1 = true
		}
		if cat.ID == int(catID2) {
			foundCat2 = true
		}
	}
	if !foundCat1 || !foundCat2 {
		t.Error("expected to find both categories for voter with no restrictions")
	}

	_ = voterID // Suppress unused warning
}

// TestGetVoteData_FiltersCategoriesByVoterType_MatchingType tests that voters see categories they have access to
func TestGetVoteData_FiltersCategoriesByVoterType_MatchingType(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create services
	settingsService := services.NewSettingsService(log, repo)
	categoryService := services.NewCategoryService(log, repo, nil)
	carService := services.NewCarService(log, repo, nil)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)

	// Create categories with voter type restrictions
	catID1, err := repo.CreateCategory(ctx, "Racer Only Category", 1, nil, []string{"racer"}, nil)
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}
	catID2, err := repo.CreateCategory(ctx, "Committee Only Category", 2, nil, []string{"committee"}, nil)
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Create a voter with type "racer"
	_, err = repo.CreateVoterFull(ctx, nil, "Test Racer", "", "racer", "RACER-QR-2", "")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}

	// Get vote data for racer
	voteData, err := votingService.GetVoteData(ctx, "RACER-QR-2")
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Should only see racer category
	if len(voteData.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(voteData.Categories))
	}

	if len(voteData.Categories) > 0 && voteData.Categories[0].ID != int(catID1) {
		t.Errorf("expected to see racer category %d, got %d", catID1, voteData.Categories[0].ID)
	}

	// Verify committee category is not visible
	for _, cat := range voteData.Categories {
		if cat.ID == int(catID2) {
			t.Error("racer should not see committee-only category")
		}
	}
}

// TestGetVoteData_FiltersCategoriesByVoterType_NonMatchingType tests that voters don't see categories they don't have access to
func TestGetVoteData_FiltersCategoriesByVoterType_NonMatchingType(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create services
	settingsService := services.NewSettingsService(log, repo)
	categoryService := services.NewCategoryService(log, repo, nil)
	carService := services.NewCarService(log, repo, nil)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)

	// Create categories with voter type restrictions
	_, err := repo.CreateCategory(ctx, "Committee Only 1", 1, nil, []string{"committee"}, nil)
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}
	_, err = repo.CreateCategory(ctx, "Committee Only 2", 2, nil, []string{"committee", "admin"}, nil)
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Create a voter with type "general"
	_, err = repo.CreateVoterFull(ctx, nil, "General Voter", "", "general", "GENERAL-QR", "")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}

	// Get vote data for general voter
	voteData, err := votingService.GetVoteData(ctx, "GENERAL-QR")
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Should see no categories
	if len(voteData.Categories) != 0 {
		t.Errorf("expected 0 categories for general voter, got %d", len(voteData.Categories))
	}
}

// TestGetVoteData_FiltersCategoriesByVoterType_MixedCategories tests filtering with a mix of restricted and unrestricted categories
func TestGetVoteData_FiltersCategoriesByVoterType_MixedCategories(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create services
	settingsService := services.NewSettingsService(log, repo)
	categoryService := services.NewCategoryService(log, repo, nil)
	carService := services.NewCarService(log, repo, nil)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)

	// Create mix of categories
	catID1, err := repo.CreateCategory(ctx, "Everyone Category", 1, nil, nil, nil) // No restrictions
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}
	catID2, err := repo.CreateCategory(ctx, "Racer Category", 2, nil, []string{"racer"}, nil) // Racer only
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}
	_, err = repo.CreateCategory(ctx, "Committee Category", 3, nil, []string{"committee"}, nil) // Committee only
	if err != nil {
		t.Fatalf("CreateCategory 3 failed: %v", err)
	}
	catID4, err := repo.CreateCategory(ctx, "Racer or General", 4, nil, []string{"racer", "general"}, nil) // Racer or general
	if err != nil {
		t.Fatalf("CreateCategory 4 failed: %v", err)
	}

	// Create a voter with type "racer"
	_, err = repo.CreateVoterFull(ctx, nil, "Test Racer", "", "racer", "RACER-QR-3", "")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}

	// Get vote data for racer
	voteData, err := votingService.GetVoteData(ctx, "RACER-QR-3")
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Should see 3 categories: Everyone, Racer, and Racer or General
	if len(voteData.Categories) != 3 {
		t.Errorf("expected 3 categories, got %d", len(voteData.Categories))
	}

	foundEveryone := false
	foundRacer := false
	foundRacerOrGeneral := false
	foundCommittee := false

	for _, cat := range voteData.Categories {
		if cat.ID == int(catID1) {
			foundEveryone = true
		}
		if cat.ID == int(catID2) {
			foundRacer = true
		}
		if cat.ID == int(catID4) {
			foundRacerOrGeneral = true
		}
		if cat.Name == "Committee Category" {
			foundCommittee = true
		}
	}

	if !foundEveryone {
		t.Error("racer should see 'Everyone Category'")
	}
	if !foundRacer {
		t.Error("racer should see 'Racer Category'")
	}
	if !foundRacerOrGeneral {
		t.Error("racer should see 'Racer or General' category")
	}
	if foundCommittee {
		t.Error("racer should not see 'Committee Category'")
	}
}

// TestGetVoteData_FiltersCategoriesByVoterType_EmptyList tests an edge case with empty categories list
func TestGetVoteData_FiltersCategoriesByVoterType_EmptyList(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create services
	settingsService := services.NewSettingsService(log, repo)
	categoryService := services.NewCategoryService(log, repo, nil)
	carService := services.NewCarService(log, repo, nil)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)

	// Don't create any categories

	// Create a voter with type "racer"
	_, err := repo.CreateVoterFull(ctx, nil, "Test Racer", "", "racer", "EMPTY-QR", "")
	if err != nil {
		t.Fatalf("CreateVoterFull failed: %v", err)
	}

	// Get vote data
	voteData, err := votingService.GetVoteData(ctx, "EMPTY-QR")
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Should see no categories
	if len(voteData.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(voteData.Categories))
	}
}

// TestGetVoteData_FiltersCategoriesByVoterType_DefaultVoterType tests filtering for default voter type
func TestGetVoteData_FiltersCategoriesByVoterType_DefaultVoterType(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create services
	settingsService := services.NewSettingsService(log, repo)
	categoryService := services.NewCategoryService(log, repo, nil)
	carService := services.NewCarService(log, repo, nil)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)

	// Create categories
	catID1, err := repo.CreateCategory(ctx, "General Category", 1, nil, []string{"general"}, nil)
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}
	_, err = repo.CreateCategory(ctx, "Racer Category", 2, nil, []string{"racer"}, nil)
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	// Create a voter using simple CreateVoter (which defaults to "general" type)
	_, err = repo.CreateVoter(ctx, "DEFAULT-QR")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	// Get vote data
	voteData, err := votingService.GetVoteData(ctx, "DEFAULT-QR")
	if err != nil {
		t.Fatalf("GetVoteData failed: %v", err)
	}

	// Should only see general category (default voter type is "general")
	if len(voteData.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(voteData.Categories))
	}

	if len(voteData.Categories) > 0 && voteData.Categories[0].ID != int(catID1) {
		t.Errorf("expected to see general category %d, got %d", catID1, voteData.Categories[0].ID)
	}
}
