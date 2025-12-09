package services_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/models"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/testutil"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// ============================================================================
// Integration Test: Full Voting Workflow
// ============================================================================

// TestIntegration_FullVotingWorkflow tests the complete voting lifecycle
func TestIntegration_FullVotingWorkflow(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	// Initialize all services
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	settingsSvc := services.NewSettingsService(log, repo)
	voterSvc := services.NewVoterService(log, repo, settingsSvc)
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)
	resultsSvc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())

	// Step 1: Create categories
	cat1ID, err := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Best Design",
		DisplayOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	cat2ID, err := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Most Creative",
		DisplayOrder: 2,
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Verify categories were created
	categories, err := categorySvc.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(categories))
	}

	// Step 2: Seed mock cars
	carsAdded, err := carSvc.SeedMockCars(ctx)
	if err != nil {
		t.Fatalf("SeedMockCars failed: %v", err)
	}
	if carsAdded == 0 {
		t.Fatal("expected some cars to be seeded")
	}

	cars, err := carSvc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) < 3 {
		t.Fatalf("expected at least 3 cars, got %d", len(cars))
	}

	// Step 3: Generate QR codes for voters
	qrCodes, err := voterSvc.GenerateQRCodes(ctx, 5)
	if err != nil {
		t.Fatalf("GenerateQRCodes failed: %v", err)
	}
	if len(qrCodes) != 5 {
		t.Fatalf("expected 5 QR codes, got %d", len(qrCodes))
	}

	// Step 4: Submit votes through VotingService
	// Voter 1, 2, 3 vote for car 1 in category 1
	// Voter 4, 5 vote for car 2 in category 1
	// All voters vote for car 3 in category 2
	for i, qr := range qrCodes {
		carIDForCat1 := cars[0].ID
		if i >= 3 {
			carIDForCat1 = cars[1].ID
		}

		// Vote in category 1
		vote1 := models.Vote{
			VoterQR:    qr,
			CategoryID: int(cat1ID),
			CarID:      carIDForCat1,
		}
		_, err := votingSvc.SubmitVote(ctx, vote1)
		if err != nil {
			t.Fatalf("SubmitVote cat1 for voter %d failed: %v", i, err)
		}

		// Vote in category 2
		vote2 := models.Vote{
			VoterQR:    qr,
			CategoryID: int(cat2ID),
			CarID:      cars[2].ID,
		}
		_, err = votingSvc.SubmitVote(ctx, vote2)
		if err != nil {
			t.Fatalf("SubmitVote cat2 for voter %d failed: %v", i, err)
		}
	}

	// Step 5: Verify results through ResultsService
	results, err := resultsSvc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	if len(results.Categories) != 2 {
		t.Fatalf("expected 2 categories in results, got %d", len(results.Categories))
	}

	// Verify category 1 results
	cat1Result := results.Categories[0]
	if cat1Result.TotalVotes != 5 {
		t.Errorf("category 1: expected 5 total votes, got %d", cat1Result.TotalVotes)
	}
	if len(cat1Result.Votes) < 2 {
		t.Errorf("category 1: expected at least 2 cars with votes, got %d", len(cat1Result.Votes))
	}

	// Verify category 2 results
	cat2Result := results.Categories[1]
	if cat2Result.TotalVotes != 5 {
		t.Errorf("category 2: expected 5 total votes, got %d", cat2Result.TotalVotes)
	}

	// Step 6: Verify winners
	winners, err := resultsSvc.GetWinners(ctx)
	if err != nil {
		t.Fatalf("GetWinners failed: %v", err)
	}

	if len(winners) != 2 {
		t.Fatalf("expected 2 winners, got %d", len(winners))
	}

	// Winner of category 1 should be car 1 (3 votes)
	winner1 := winners[0]["winner"].(map[string]interface{})
	if winner1["vote_count"] != 3 {
		t.Errorf("category 1 winner: expected 3 votes, got %v", winner1["vote_count"])
	}

	// Winner of category 2 should have 5 votes (all voters)
	winner2 := winners[1]["winner"].(map[string]interface{})
	if winner2["vote_count"] != 5 {
		t.Errorf("category 2 winner: expected 5 votes, got %v", winner2["vote_count"])
	}
}

// ============================================================================
// Integration Test: DerbyNet Sync Validation
// ============================================================================

// TestIntegration_DerbyNetSyncWithMock tests sync behavior with mock client
func TestIntegration_DerbyNetSyncWithMock(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	// Test successful sync with mock
	mockClient := derbynet.NewMockClient()
	carSvc := services.NewCarService(log, repo, mockClient)

	result, err := carSvc.SyncFromDerbyNet(ctx, "http://test.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.TotalRacers != 10 {
		t.Errorf("expected 10 racers from mock, got %d", result.TotalRacers)
	}

	// Test sync with error mock
	repo2 := testutil.NewTestRepository(t)
	errorClient := derbynet.NewMockClient(
		derbynet.WithFetchError(fmt.Errorf("connection refused")),
	)
	carSvc2 := services.NewCarService(log, repo2, errorClient)

	result2, err := carSvc2.SyncFromDerbyNet(ctx, "http://failing.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet returned unexpected error: %v", err)
	}
	if result2.Status != "error" {
		t.Errorf("expected status 'error', got %q", result2.Status)
	}
	if result2.Message == "" {
		t.Error("expected error message")
	}
}

// ============================================================================
// Integration Test: Settings Cascade
// ============================================================================

// TestIntegration_SettingsCascade tests voting status affecting behavior
func TestIntegration_SettingsCascade(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	resultsSvc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)

	// Step 1: Voting should be open by default
	isOpen, err := settingsSvc.IsVotingOpen(ctx)
	if err != nil {
		t.Fatalf("IsVotingOpen failed: %v", err)
	}
	if !isOpen {
		t.Error("expected voting to be open by default")
	}

	// Setup test data
	catID, _ := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "100", "Test Racer", "Test Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Step 2: Submit votes while voting is open
	vote := models.Vote{
		VoterQR:    "OPEN-001",
		CategoryID: int(catID),
		CarID:      carID,
	}
	_, err = votingSvc.SubmitVote(ctx, vote)
	if err != nil {
		t.Fatalf("SubmitVote while open failed: %v", err)
	}

	// Step 3: Close voting
	err = settingsSvc.CloseVoting(ctx)
	if err != nil {
		t.Fatalf("CloseVoting failed: %v", err)
	}

	isOpen, err = settingsSvc.IsVotingOpen(ctx)
	if err != nil {
		t.Fatalf("IsVotingOpen after close failed: %v", err)
	}
	if isOpen {
		t.Error("expected voting to be closed")
	}

	// Step 4: Verify stats reflect voting status
	stats, err := resultsSvc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if votingOpen, ok := stats["voting_open"]; ok {
		if votingOpen != false {
			t.Errorf("expected voting_open=false in stats, got %v", votingOpen)
		}
	}

	// Step 5: Reopen voting
	err = settingsSvc.OpenVoting(ctx)
	if err != nil {
		t.Fatalf("OpenVoting failed: %v", err)
	}

	isOpen, err = settingsSvc.IsVotingOpen(ctx)
	if err != nil {
		t.Fatalf("IsVotingOpen after reopen failed: %v", err)
	}
	if !isOpen {
		t.Error("expected voting to be open after reopening")
	}
}

// TestIntegration_TimerFunctionality tests voting timer functionality
func TestIntegration_TimerFunctionality(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)

	// Test valid timer start
	closeTime, err := settingsSvc.StartVotingTimer(ctx, 30)
	if err != nil {
		t.Fatalf("StartVotingTimer failed: %v", err)
	}
	if closeTime == "" {
		t.Error("expected non-empty close time")
	}

	// Verify voting is open after starting timer
	isOpen, err := settingsSvc.IsVotingOpen(ctx)
	if err != nil {
		t.Fatalf("IsVotingOpen failed: %v", err)
	}
	if !isOpen {
		t.Error("expected voting to be open after starting timer")
	}

	// Test invalid timer values
	invalidMinutes := []int{0, -1, 61, 100}
	for _, minutes := range invalidMinutes {
		_, err := settingsSvc.StartVotingTimer(ctx, minutes)
		if err != services.ErrInvalidTimerMinutes {
			t.Errorf("StartVotingTimer(%d): expected ErrInvalidTimerMinutes, got %v", minutes, err)
		}
	}

	// Test timer clear on close voting
	err = settingsSvc.CloseVoting(ctx)
	if err != nil {
		t.Fatalf("CloseVoting failed: %v", err)
	}

	// Verify timer was cleared
	timerEnd, err := settingsSvc.GetTimerEndTime(ctx)
	if err != nil {
		t.Fatalf("GetTimerEndTime failed: %v", err)
	}
	if timerEnd != 0 {
		t.Errorf("expected timer to be cleared (0), got %d", timerEnd)
	}
}

// ============================================================================
// Integration Test: Voter-Car Relationships
// ============================================================================

// TestIntegration_VoterCarRelationship tests voter linked to car behavior
func TestIntegration_VoterCarRelationship(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	voterSvc := services.NewVoterService(log, repo, settingsSvc)

	// Step 1: Create a car first
	err := repo.CreateCar(ctx, "200", "Jane Racer", "Speedy Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) == 0 {
		t.Fatal("expected at least one car")
	}
	carID := cars[0].ID

	// Step 2: Create voter linked to car
	voter := services.Voter{
		Name:   "Jane Racer",
		CarID:  &carID,
		QRCode: "CAR-VOTER",
	}
	voterID, qrCode, err := voterSvc.CreateVoter(ctx, voter)
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}
	if voterID <= 0 {
		t.Error("expected positive voter ID")
	}
	if qrCode != "CAR-VOTER" {
		t.Errorf("expected QR code 'CAR-VOTER', got %q", qrCode)
	}

	// Step 3: Verify voter appears in list
	voters, err := voterSvc.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(voters))
	}

	// Verify voter has car association
	v := voters[0]
	if v["name"] != "Jane Racer" {
		t.Errorf("expected name 'Jane Racer', got %v", v["name"])
	}

	// Step 4: Delete the voter
	err = voterSvc.DeleteVoter(ctx, int(voterID))
	if err != nil {
		t.Fatalf("DeleteVoter failed: %v", err)
	}

	// Step 5: Verify voter no longer exists
	voters, err = voterSvc.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters after delete failed: %v", err)
	}
	if len(voters) != 0 {
		t.Errorf("expected 0 voters after deletion, got %d", len(voters))
	}

	// Step 6: Verify car still exists (car wasn't affected by voter deletion)
	cars, err = repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars after voter delete failed: %v", err)
	}
	if len(cars) != 1 {
		t.Errorf("expected car to still exist after voter deletion, got %d cars", len(cars))
	}
}

// ============================================================================
// Integration Test: Category Group Exclusivity
// ============================================================================

// TestIntegration_CategoryGroupExclusivity tests exclusivity pool conflict handling
func TestIntegration_CategoryGroupExclusivity(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Step 1: Create category group with exclusivity pool
	poolID := 1
	groupID, err := categorySvc.CreateGroup(ctx, services.CategoryGroup{
		Name:              "Speed Awards",
		Description:       "Speed-related categories",
		ExclusivityPoolID: &poolID,
		DisplayOrder:      1,
	})
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// Step 2: Create categories in the group
	groupIDInt := int(groupID)
	cat1ID, err := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Fastest Looking",
		DisplayOrder: 1,
		GroupID:      &groupIDInt,
	})
	if err != nil {
		t.Fatalf("CreateCategory 1 failed: %v", err)
	}

	cat2ID, err := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Most Aerodynamic",
		DisplayOrder: 2,
		GroupID:      &groupIDInt,
	})
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}

	cat3ID, err := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Speed Demon Award",
		DisplayOrder: 3,
		GroupID:      &groupIDInt,
	})
	if err != nil {
		t.Fatalf("CreateCategory 3 failed: %v", err)
	}

	// Step 3: Create a test car
	err = repo.CreateCar(ctx, "300", "Speed Racer", "Fast Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	voterQR := "EXCL-001"

	// Step 4: Vote for car in first category
	vote1 := models.Vote{
		VoterQR:    voterQR,
		CategoryID: int(cat1ID),
		CarID:      carID,
	}
	result1, err := votingSvc.SubmitVote(ctx, vote1)
	if err != nil {
		t.Fatalf("SubmitVote 1 failed: %v", err)
	}
	if result1.ConflictCleared {
		t.Error("expected no conflict on first vote")
	}

	// Step 5: Vote for SAME car in second category (should trigger conflict)
	vote2 := models.Vote{
		VoterQR:    voterQR,
		CategoryID: int(cat2ID),
		CarID:      carID,
	}
	result2, err := votingSvc.SubmitVote(ctx, vote2)
	if err != nil {
		t.Fatalf("SubmitVote 2 failed: %v", err)
	}
	if !result2.ConflictCleared {
		t.Error("expected conflict to be detected and cleared")
	}
	if result2.ConflictCategoryID != int(cat1ID) {
		t.Errorf("expected conflict category ID %d, got %d", cat1ID, result2.ConflictCategoryID)
	}

	// Step 6: Vote for SAME car in third category (should clear vote from cat2)
	vote3 := models.Vote{
		VoterQR:    voterQR,
		CategoryID: int(cat3ID),
		CarID:      carID,
	}
	result3, err := votingSvc.SubmitVote(ctx, vote3)
	if err != nil {
		t.Fatalf("SubmitVote 3 failed: %v", err)
	}
	if !result3.ConflictCleared {
		t.Error("expected conflict to be detected and cleared")
	}
	if result3.ConflictCategoryID != int(cat2ID) {
		t.Errorf("expected conflict category ID %d, got %d", cat2ID, result3.ConflictCategoryID)
	}

	// Step 7: Verify only one vote remains (in cat3)
	voterID, err := repo.GetVoterByQR(ctx, voterQR)
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}

	if len(votes) != 1 {
		t.Errorf("expected 1 vote after exclusivity handling, got %d", len(votes))
	}
	if votes[int(cat3ID)] != carID {
		t.Errorf("expected vote in category %d to be car %d", cat3ID, carID)
	}
}

// TestIntegration_ExclusivityAcrossMultiplePools tests multiple exclusivity pools
func TestIntegration_ExclusivityAcrossMultiplePools(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create two separate pools
	pool1 := 1
	pool2 := 2

	group1ID, err := categorySvc.CreateGroup(ctx, services.CategoryGroup{
		Name:              "Speed Awards",
		ExclusivityPoolID: &pool1,
		DisplayOrder:      1,
	})
	if err != nil {
		t.Fatalf("CreateGroup 1 failed: %v", err)
	}

	group2ID, err := categorySvc.CreateGroup(ctx, services.CategoryGroup{
		Name:              "Design Awards",
		ExclusivityPoolID: &pool2,
		DisplayOrder:      2,
	})
	if err != nil {
		t.Fatalf("CreateGroup 2 failed: %v", err)
	}

	// Create categories in each pool
	g1Int := int(group1ID)
	g2Int := int(group2ID)

	cat1ID, _ := categorySvc.CreateCategory(ctx, services.Category{Name: "Fast 1", GroupID: &g1Int, DisplayOrder: 1})
	cat2ID, _ := categorySvc.CreateCategory(ctx, services.Category{Name: "Fast 2", GroupID: &g1Int, DisplayOrder: 2})
	cat3ID, _ := categorySvc.CreateCategory(ctx, services.Category{Name: "Design 1", GroupID: &g2Int, DisplayOrder: 3})
	cat4ID, _ := categorySvc.CreateCategory(ctx, services.Category{Name: "Design 2", GroupID: &g2Int, DisplayOrder: 4})

	// Create a car
	_ = repo.CreateCar(ctx, "400", "Multi Pool", "Pool Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	voterQR := "MULTI-POOL"

	// Vote for car in both pools - should be allowed (different pools)
	votingSvc.SubmitVote(ctx, models.Vote{VoterQR: voterQR, CategoryID: int(cat1ID), CarID: carID})
	votingSvc.SubmitVote(ctx, models.Vote{VoterQR: voterQR, CategoryID: int(cat3ID), CarID: carID})

	voterID, _ := repo.GetVoterByQR(ctx, voterQR)
	votes, _ := repo.GetVoterVotes(ctx, voterID)

	// Should have votes in both pools
	if len(votes) != 2 {
		t.Errorf("expected 2 votes (one per pool), got %d", len(votes))
	}

	// Now vote in same pool - should trigger conflict
	result, _ := votingSvc.SubmitVote(ctx, models.Vote{VoterQR: voterQR, CategoryID: int(cat2ID), CarID: carID})
	if !result.ConflictCleared {
		t.Error("expected conflict within pool 1")
	}

	// Vote in other pool - should also trigger conflict
	result2, _ := votingSvc.SubmitVote(ctx, models.Vote{VoterQR: voterQR, CategoryID: int(cat4ID), CarID: carID})
	if !result2.ConflictCleared {
		t.Error("expected conflict within pool 2")
	}

	// Final state: votes in cat2 and cat4
	votes, _ = repo.GetVoterVotes(ctx, voterID)
	if len(votes) != 2 {
		t.Errorf("expected 2 votes after conflicts, got %d", len(votes))
	}
}

// ============================================================================
// Integration Test: Concurrent Operations
// ============================================================================

// TestIntegration_ConcurrentVoting tests multiple voters voting simultaneously
func TestIntegration_ConcurrentVoting(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)
	resultsSvc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Setup test data
	catID, err := repo.CreateCategory(ctx, "Concurrent Test", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Create multiple cars
	for i := 0; i < 3; i++ {
		err := repo.CreateCar(ctx, fmt.Sprintf("C%d", i), fmt.Sprintf("Racer %d", i), fmt.Sprintf("Car %d", i), "")
		if err != nil {
			t.Fatalf("CreateCar %d failed: %v", i, err)
		}
	}

	cars, _ := repo.ListCars(ctx)
	if len(cars) < 3 {
		t.Fatal("expected at least 3 cars")
	}

	// Concurrent voting
	numVoters := 50
	var wg sync.WaitGroup
	errors := make(chan error, numVoters)

	for i := 0; i < numVoters; i++ {
		wg.Add(1)
		go func(voterNum int) {
			defer wg.Done()

			qrCode := fmt.Sprintf("CONC-%03d", voterNum)
			carIdx := voterNum % 3 // Distribute votes across 3 cars

			vote := models.Vote{
				VoterQR:    qrCode,
				CategoryID: int(catID),
				CarID:      cars[carIdx].ID,
			}

			_, err := votingSvc.SubmitVote(ctx, vote)
			if err != nil {
				errors <- fmt.Errorf("voter %d: %w", voterNum, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent vote error: %v", err)
	}

	// Verify vote counts are accurate
	results, err := resultsSvc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	if len(results.Categories) == 0 {
		t.Fatal("expected at least one category in results")
	}

	catResult := results.Categories[0]
	if catResult.TotalVotes != numVoters {
		t.Errorf("expected %d total votes, got %d", numVoters, catResult.TotalVotes)
	}

	// Verify distribution (should be roughly equal since we use modulo 3)
	expectedPerCar := numVoters / 3
	for _, carVote := range catResult.Votes {
		// Allow some variance due to modulo distribution
		if carVote.VoteCount < expectedPerCar-1 || carVote.VoteCount > expectedPerCar+1 {
			t.Errorf("car %s: expected ~%d votes, got %d", carVote.CarNumber, expectedPerCar, carVote.VoteCount)
		}
	}
}

// TestIntegration_ConcurrentVoteUpdates tests concurrent vote changes by same voter
func TestIntegration_ConcurrentVoteUpdates(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Setup
	catID, _ := repo.CreateCategory(ctx, "Update Test", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "U1", "Racer 1", "Car 1", "")
	_ = repo.CreateCar(ctx, "U2", "Racer 2", "Car 2", "")
	cars, _ := repo.ListCars(ctx)

	qrCode := "UPDATE-001"
	var wg sync.WaitGroup
	iterations := 20

	// Rapidly change votes back and forth
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()

			carID := cars[iter%2].ID
			vote := models.Vote{
				VoterQR:    qrCode,
				CategoryID: int(catID),
				CarID:      carID,
			}
			votingSvc.SubmitVote(ctx, vote)
		}(i)
	}

	wg.Wait()

	// Verify voter has exactly one vote (the last one wins)
	voterID, err := repo.GetVoterByQR(ctx, qrCode)
	if err != nil {
		t.Fatalf("GetVoterByQR failed: %v", err)
	}

	votes, err := repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		t.Fatalf("GetVoterVotes failed: %v", err)
	}

	if len(votes) != 1 {
		t.Errorf("expected exactly 1 vote after concurrent updates, got %d", len(votes))
	}
}

// ============================================================================
// Integration Test: Stats Accuracy
// ============================================================================

// TestIntegration_StatsAccuracy tests that statistics are accurate after various operations
func TestIntegration_StatsAccuracy(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	resultsSvc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)
	voterSvc := services.NewVoterService(log, repo, settingsSvc)

	// Initial stats should be zero
	stats, err := resultsSvc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats["total_voters"].(int) != 0 {
		t.Errorf("initial total_voters: expected 0, got %v", stats["total_voters"])
	}
	if stats["total_categories"].(int) != 0 {
		t.Errorf("initial total_categories: expected 0, got %v", stats["total_categories"])
	}

	// Add categories
	categorySvc.CreateCategory(ctx, services.Category{Name: "Cat 1", DisplayOrder: 1})
	categorySvc.CreateCategory(ctx, services.Category{Name: "Cat 2", DisplayOrder: 2})

	// Add cars
	carSvc.SeedMockCars(ctx)

	// Generate voters
	voterSvc.GenerateQRCodes(ctx, 10)

	// Check updated stats
	stats, _ = resultsSvc.GetStats(ctx)
	if stats["total_categories"].(int) != 2 {
		t.Errorf("total_categories: expected 2, got %v", stats["total_categories"])
	}
	if stats["total_voters"].(int) != 10 {
		t.Errorf("total_voters: expected 10, got %v", stats["total_voters"])
	}

	// Submit some votes
	cats, _ := categorySvc.ListCategories(ctx)
	cars, _ := carSvc.ListCars(ctx)

	for i := 0; i < 5; i++ {
		qr := fmt.Sprintf("STAT-%03d", i)
		votingSvc.SubmitVote(ctx, models.Vote{
			VoterQR:    qr,
			CategoryID: cats[0].ID,
			CarID:      cars[0].ID,
		})
	}

	// Check vote stats
	stats, _ = resultsSvc.GetStats(ctx)
	if stats["total_votes"].(int) != 5 {
		t.Errorf("total_votes: expected 5, got %v", stats["total_votes"])
	}

	// voters_who_voted should be 5 (new voters created by voting)
	if stats["voters_who_voted"].(int) != 5 {
		t.Errorf("voters_who_voted: expected 5, got %v", stats["voters_who_voted"])
	}
}

// ============================================================================
// Integration Test: Vote Deselection
// ============================================================================

// TestIntegration_VoteDeselection tests removing a vote by voting for carID=0
func TestIntegration_VoteDeselection(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)
	resultsSvc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Setup
	catID, _ := repo.CreateCategory(ctx, "Deselect Test", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "D1", "Deselect Racer", "Deselect Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	voterQR := "DESELECT-001"

	// Vote for a car
	_, err := votingSvc.SubmitVote(ctx, models.Vote{
		VoterQR:    voterQR,
		CategoryID: int(catID),
		CarID:      carID,
	})
	if err != nil {
		t.Fatalf("SubmitVote failed: %v", err)
	}

	// Verify vote exists
	results, _ := resultsSvc.GetResults(ctx)
	if results.Categories[0].TotalVotes != 1 {
		t.Errorf("expected 1 vote before deselection, got %d", results.Categories[0].TotalVotes)
	}

	// Deselect (vote with carID=0)
	_, err = votingSvc.SubmitVote(ctx, models.Vote{
		VoterQR:    voterQR,
		CategoryID: int(catID),
		CarID:      0,
	})
	if err != nil {
		t.Fatalf("Deselect failed: %v", err)
	}

	// Verify vote was removed
	results, _ = resultsSvc.GetResults(ctx)
	if results.Categories[0].TotalVotes != 0 {
		t.Errorf("expected 0 votes after deselection, got %d", results.Categories[0].TotalVotes)
	}

	// Verify voter still exists but has no votes
	voterID, _ := repo.GetVoterByQR(ctx, voterQR)
	votes, _ := repo.GetVoterVotes(ctx, voterID)
	if len(votes) != 0 {
		t.Errorf("expected 0 votes for voter after deselection, got %d", len(votes))
	}
}

// ============================================================================
// Integration Test: Multiple Voters Same Car
// ============================================================================

// TestIntegration_MultipleVotersSameCar tests that multiple voters can vote for the same car
func TestIntegration_MultipleVotersSameCar(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)
	resultsSvc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Setup
	catID, _ := repo.CreateCategory(ctx, "Popular Vote", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "P1", "Popular Racer", "Popular Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Multiple voters vote for the same car
	numVoters := 25
	for i := 0; i < numVoters; i++ {
		qr := fmt.Sprintf("POP-%03d", i)
		_, err := votingSvc.SubmitVote(ctx, models.Vote{
			VoterQR:    qr,
			CategoryID: int(catID),
			CarID:      carID,
		})
		if err != nil {
			t.Fatalf("SubmitVote for voter %d failed: %v", i, err)
		}
	}

	// Verify results
	results, err := resultsSvc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	if len(results.Categories) == 0 {
		t.Fatal("expected at least one category")
	}

	catResult := results.Categories[0]
	if catResult.TotalVotes != numVoters {
		t.Errorf("expected %d total votes, got %d", numVoters, catResult.TotalVotes)
	}

	if len(catResult.Votes) != 1 {
		t.Errorf("expected 1 car with votes, got %d", len(catResult.Votes))
	}

	if catResult.Votes[0].VoteCount != numVoters {
		t.Errorf("expected car to have %d votes, got %d", numVoters, catResult.Votes[0].VoteCount)
	}

	// Verify winners
	winners, _ := resultsSvc.GetWinners(ctx)
	if len(winners) != 1 {
		t.Errorf("expected 1 winner, got %d", len(winners))
	}
	if winners[0]["winner"].(map[string]interface{})["vote_count"] != numVoters {
		t.Errorf("winner should have %d votes", numVoters)
	}
}

// ============================================================================
// Integration Test: Category Without Group (No Exclusivity)
// ============================================================================

// TestIntegration_CategoriesWithoutExclusivity tests voting without exclusivity constraints
func TestIntegration_CategoriesWithoutExclusivity(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)

	// Open voting so we can submit votes
	settingsSvc.OpenVoting(ctx)

	// Create categories WITHOUT groups (no exclusivity)
	cat1ID, _ := categorySvc.CreateCategory(ctx, services.Category{Name: "No Group 1", DisplayOrder: 1})
	cat2ID, _ := categorySvc.CreateCategory(ctx, services.Category{Name: "No Group 2", DisplayOrder: 2})
	cat3ID, _ := categorySvc.CreateCategory(ctx, services.Category{Name: "No Group 3", DisplayOrder: 3})

	// Create a car
	_ = repo.CreateCar(ctx, "NG1", "No Group Racer", "No Group Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	voterQR := "NO-GROUP"

	// Vote for same car in ALL categories - should be allowed
	vote1, _ := votingSvc.SubmitVote(ctx, models.Vote{VoterQR: voterQR, CategoryID: int(cat1ID), CarID: carID})
	if vote1.ConflictCleared {
		t.Error("expected no conflict for category without group")
	}

	vote2, _ := votingSvc.SubmitVote(ctx, models.Vote{VoterQR: voterQR, CategoryID: int(cat2ID), CarID: carID})
	if vote2.ConflictCleared {
		t.Error("expected no conflict for category without group")
	}

	vote3, _ := votingSvc.SubmitVote(ctx, models.Vote{VoterQR: voterQR, CategoryID: int(cat3ID), CarID: carID})
	if vote3.ConflictCleared {
		t.Error("expected no conflict for category without group")
	}

	// Verify all 3 votes exist
	voterID, _ := repo.GetVoterByQR(ctx, voterQR)
	votes, _ := repo.GetVoterVotes(ctx, voterID)

	if len(votes) != 3 {
		t.Errorf("expected 3 votes (one per category), got %d", len(votes))
	}

	// All votes should be for the same car
	for catID, votedCarID := range votes {
		if votedCarID != carID {
			t.Errorf("category %d: expected car %d, got %d", catID, carID, votedCarID)
		}
	}
}

// ============================================================================
// Integration Test: Complete End-to-End Workflow
// ============================================================================

// TestIntegration_CompleteEndToEndWorkflow tests the entire voting lifecycle
// from setup through to pushing results to DerbyNet
func TestIntegration_CompleteEndToEndWorkflow(t *testing.T) {
	// Step 1: Start with blank database
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	// Step 2: Connect to mocked DerbyNet instance
	mockClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, repo, mockClient)
	carSvc := services.NewCarService(log, repo, mockClient)
	settingsSvc := services.NewSettingsService(log, repo)
	voterSvc := services.NewVoterService(log, repo, settingsSvc)
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)
	resultsSvc := services.NewResultsService(log, repo, settingsSvc, mockClient)

	// Step 3: Import cars and categories from DerbyNet
	syncResult, err := carSvc.SyncFromDerbyNet(ctx, "http://mock-derbynet.local")
	if err != nil {
		t.Fatalf("Failed to sync cars from DerbyNet: %v", err)
	}
	if syncResult.TotalRacers != 10 {
		t.Errorf("Expected 10 racers from mock, got %d", syncResult.TotalRacers)
	}

	cars, err := carSvc.ListCars(ctx)
	if err != nil {
		t.Fatalf("Failed to list cars: %v", err)
	}
	if len(cars) < 5 {
		t.Fatalf("Expected at least 5 cars, got %d", len(cars))
	}

	// Step 4: Create category groups with exclusive vote groups
	poolID1 := 1
	poolID2 := 2

	// Group 1: Speed awards (exclusivity pool 1)
	group1ID, err := categorySvc.CreateGroup(ctx, services.CategoryGroup{
		Name:              "Speed Awards",
		Description:       "Awards for fastest-looking cars",
		ExclusivityPoolID: &poolID1,
		DisplayOrder:      1,
	})
	if err != nil {
		t.Fatalf("Failed to create group 1: %v", err)
	}

	// Group 2: Design awards (exclusivity pool 2)
	group2ID, err := categorySvc.CreateGroup(ctx, services.CategoryGroup{
		Name:              "Design Awards",
		Description:       "Awards for best design",
		ExclusivityPoolID: &poolID2,
		DisplayOrder:      2,
	})
	if err != nil {
		t.Fatalf("Failed to create group 2: %v", err)
	}

	// Group 3: Special awards (no exclusivity)
	group3ID, err := categorySvc.CreateGroup(ctx, services.CategoryGroup{
		Name:         "Special Awards",
		Description:  "Special recognition awards",
		DisplayOrder: 3,
	})
	if err != nil {
		t.Fatalf("Failed to create group 3: %v", err)
	}

	// Create categories in each group
	g1Int := int(group1ID)
	g2Int := int(group2ID)
	g3Int := int(group3ID)

	cat1ID, _ := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Fastest Looking",
		GroupID:      &g1Int,
		DisplayOrder: 1,
	})
	cat2ID, _ := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Most Aerodynamic",
		GroupID:      &g1Int,
		DisplayOrder: 2,
	})
	cat3ID, _ := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Best Design",
		GroupID:      &g2Int,
		DisplayOrder: 3,
	})
	categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Best Paint Job",
		GroupID:      &g2Int,
		DisplayOrder: 4,
	})
	cat5ID, _ := categorySvc.CreateCategory(ctx, services.Category{
		Name:         "Most Creative",
		GroupID:      &g3Int,
		DisplayOrder: 5,
	})

	// Step 5: Add additional voters and check QR codes are generated
	qrCodes, err := voterSvc.GenerateQRCodes(ctx, 20)
	if err != nil {
		t.Fatalf("Failed to generate QR codes: %v", err)
	}
	if len(qrCodes) != 20 {
		t.Fatalf("Expected 20 QR codes, got %d", len(qrCodes))
	}

	// Verify QR codes are unique and properly formatted
	qrMap := make(map[string]bool)
	for _, qr := range qrCodes {
		if qrMap[qr] {
			t.Errorf("Duplicate QR code found: %s", qr)
		}
		qrMap[qr] = true
		if len(qr) == 0 {
			t.Error("Empty QR code generated")
		}
	}

	// Step 6: Run actual voting
	// Voter 0-4: Vote for car 0 in cat1, car 1 in cat3
	// Voter 5-9: Vote for car 1 in cat1, car 2 in cat3
	// Voter 10-14: Vote for car 2 in cat1, car 0 in cat3
	// Voter 15-19: Vote for car 0 in cat1, car 1 in cat3 (creates tie in cat1)
	for i := 0; i < 20; i++ {
		var car1Idx, car3Idx int
		if i < 5 {
			car1Idx, car3Idx = 0, 1
		} else if i < 10 {
			car1Idx, car3Idx = 1, 2
		} else if i < 15 {
			car1Idx, car3Idx = 2, 0
		} else {
			car1Idx, car3Idx = 0, 1 // Creates tie in cat1
		}

		_, err := votingSvc.SubmitVote(ctx, models.Vote{
			VoterQR:    qrCodes[i],
			CategoryID: int(cat1ID),
			CarID:      cars[car1Idx].ID,
		})
		if err != nil {
			t.Fatalf("Vote submission failed for voter %d: %v", i, err)
		}

		_, err = votingSvc.SubmitVote(ctx, models.Vote{
			VoterQR:    qrCodes[i],
			CategoryID: int(cat3ID),
			CarID:      cars[car3Idx].ID,
		})
		if err != nil {
			t.Fatalf("Vote submission failed for voter %d cat3: %v", i, err)
		}
	}

	// Step 7: Have users change their mind and vote for different people
	// First 5 voters change their vote in cat1
	for i := 0; i < 5; i++ {
		_, err := votingSvc.SubmitVote(ctx, models.Vote{
			VoterQR:    qrCodes[i],
			CategoryID: int(cat1ID),
			CarID:      cars[2].ID, // Change to car 2
		})
		if err != nil {
			t.Fatalf("Vote change failed for voter %d: %v", i, err)
		}
	}

	// Step 8: Test exclusivity - vote for same car in exclusive categories
	_, err = votingSvc.SubmitVote(ctx, models.Vote{
		VoterQR:    qrCodes[0],
		CategoryID: int(cat2ID),
		CarID:      cars[2].ID, // Same car as cat1, should trigger conflict
	})
	if err != nil {
		t.Fatalf("Exclusivity vote failed: %v", err)
	}

	// Verify conflict was handled - voter should only have vote in cat2 now for pool 1
	voterID, _ := repo.GetVoterByQR(ctx, qrCodes[0])
	votes, _ := repo.GetVoterVotes(ctx, voterID)
	if votes[int(cat1ID)] != 0 {
		t.Errorf("Expected cat1 vote to be cleared due to exclusivity, but found vote")
	}
	if votes[int(cat2ID)] != cars[2].ID {
		t.Errorf("Expected vote in cat2 for car %d", cars[2].ID)
	}

	// Step 9: Close voting and try to vote after it's closed
	err = settingsSvc.CloseVoting(ctx)
	if err != nil {
		t.Fatalf("Failed to close voting: %v", err)
	}

	// Attempt to vote after closing - should be rejected
	_, err = votingSvc.SubmitVote(ctx, models.Vote{
		VoterQR:    qrCodes[19],
		CategoryID: int(cat5ID),
		CarID:      cars[0].ID,
	})
	if err == nil {
		t.Error("Expected error when voting after close, but got none")
	}
	if err != services.ErrVotingClosed {
		t.Errorf("Expected ErrVotingClosed, got: %v", err)
	}

	// Step 10: Resolve the vote, including ties and conflicts
	results, err := resultsSvc.GetResults(ctx)
	if err != nil {
		t.Fatalf("Failed to get results: %v", err)
	}

	// Check for ties
	ties, err := resultsSvc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("Failed to detect ties: %v", err)
	}

	// We should have a tie in cat1 (car 0 and car 2 both have 9-10 votes after changes)
	hasTie := false
	for _, tie := range ties {
		if tie.CategoryID == int(cat1ID) {
			hasTie = true
			if len(tie.TiedCars) < 2 {
				t.Errorf("Expected tie with at least 2 cars, got %d", len(tie.TiedCars))
			}
		}
	}

	// Resolve the tie manually
	if hasTie {
		err = resultsSvc.SetManualWinner(ctx, int(cat1ID), cars[0].ID, "Broke tie by coin flip")
		if err != nil {
			t.Fatalf("Failed to set manual winner: %v", err)
		}

		// Verify manual winner was set
		categories, _ := repo.ListCategories(ctx)
		for _, cat := range categories {
			if cat.ID == int(cat1ID) {
				if cat.OverrideWinnerCarID == nil || *cat.OverrideWinnerCarID != cars[0].ID {
					t.Errorf("Manual winner not properly set for cat1")
				}
				if cat.OverrideReason != "Broke tie by coin flip" {
					t.Errorf("Override reason not set correctly")
				}
			}
		}
	}

	// Verify results structure
	if len(results.Categories) != 5 {
		t.Errorf("Expected 5 categories in results, got %d", len(results.Categories))
	}

	// Step 11: Push results back to DerbyNet
	// Note: With mock client, we need to first sync categories to get DerbyNet award IDs
	// For this test, we'll manually link categories to awards
	awards := mockClient.GetAwards()
	if len(awards) > 0 {
		// Link first category to first award
		repo.UpdateCategory(ctx, int(cat1ID), "Fastest Looking", 1, &g1Int, nil, nil, true)
	}

	// Push results
	pushResult, err := resultsSvc.PushResultsToDerbyNet(ctx, "http://mock-derbynet.local")
	if err != nil {
		t.Fatalf("Failed to push results to DerbyNet: %v", err)
	}

	// Verify push result
	if pushResult.Status != "success" && pushResult.Status != "partial" {
		t.Errorf("Expected success or partial status, got: %s", pushResult.Status)
	}

	// Most categories won't have DerbyNet links, so we expect some skipped
	if pushResult.Skipped == 0 && pushResult.WinnersPushed == 0 {
		t.Error("Expected either skipped or pushed winners, got neither")
	}

	t.Logf("Integration test completed successfully - Winners pushed: %d, Skipped: %d, Errors: %d",
		pushResult.WinnersPushed, pushResult.Skipped, pushResult.Errors)
}

// ============================================================================
// Integration Test: DerbyNet Results Push with Categories Linked
// ============================================================================

// TestIntegration_PushResultsToDerbyNet tests pushing voting results to DerbyNet
func TestIntegration_PushResultsToDerbyNet(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	mockClient := derbynet.NewMockClient()
	categorySvc := services.NewCategoryService(log, repo, mockClient)
	carSvc := services.NewCarService(log, repo, mockClient)
	settingsSvc := services.NewSettingsService(log, repo)
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)
	resultsSvc := services.NewResultsService(log, repo, settingsSvc, mockClient)

	// Sync cars from DerbyNet (creates cars with DerbyNet racer IDs)
	syncResult, err := carSvc.SyncFromDerbyNet(ctx, "http://mock-derbynet.local")
	if err != nil {
		t.Fatalf("Failed to sync cars: %v", err)
	}
	if syncResult.Status != "success" {
		t.Fatalf("Sync failed: %s", syncResult.Message)
	}

	cars, _ := carSvc.ListCars(ctx)

	// Sync categories from DerbyNet awards
	catSyncResult, err := categorySvc.SyncFromDerbyNet(ctx, "http://mock-derbynet.local")
	if err != nil {
		t.Fatalf("Failed to sync categories: %v", err)
	}
	if catSyncResult.CategoriesCreated == 0 && catSyncResult.CategoriesUpdated == 0 {
		t.Fatal("Expected categories to be synced from DerbyNet")
	}

	categories, _ := categorySvc.ListCategories(ctx)
	if len(categories) == 0 {
		t.Fatal("No categories after import")
	}

	// Submit votes for first 3 categories
	for i := 0; i < 3 && i < len(categories); i++ {
		cat := categories[i]
		// 5 votes for car 0, 3 votes for car 1, 2 votes for car 2
		for v := 0; v < 10; v++ {
			var carIdx int
			if v < 5 {
				carIdx = 0
			} else if v < 8 {
				carIdx = 1
			} else {
				carIdx = 2
			}

			qr := fmt.Sprintf("PUSH-TEST-%d-%d", i, v)
			_, err := votingSvc.SubmitVote(ctx, models.Vote{
				VoterQR:    qr,
				CategoryID: cat.ID,
				CarID:      cars[carIdx].ID,
			})
			if err != nil {
				t.Fatalf("Failed to submit vote: %v", err)
			}
		}
	}

	// Push results to DerbyNet
	pushResult, err := resultsSvc.PushResultsToDerbyNet(ctx, "http://mock-derbynet.local")
	if err != nil {
		t.Fatalf("Failed to push results: %v", err)
	}

	// Should successfully push winners for categories that were imported
	if pushResult.WinnersPushed == 0 {
		t.Error("Expected at least some winners to be pushed")
	}

	// Verify winners were recorded in mock client
	winners := mockClient.GetAwardWinners()
	if len(winners) == 0 {
		t.Error("Expected award winners to be recorded in mock client")
	}

	t.Logf("Successfully pushed %d winners to DerbyNet", pushResult.WinnersPushed)
}

// ============================================================================
// Integration Test: Voting After Close and Reopening
// ============================================================================

// TestIntegration_VotingClosedAndReopened tests the complete close/reopen cycle
func TestIntegration_VotingClosedAndReopened(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	ctx := context.Background()
	log := logger.New()

	settingsSvc := services.NewSettingsService(log, repo)
	categorySvc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	carSvc := services.NewCarService(log, repo, derbynet.NewMockClient())
	votingSvc := services.NewVotingService(log, repo, categorySvc, carSvc, settingsSvc)

	// Create test data
	catID, _ := categorySvc.CreateCategory(ctx, services.Category{Name: "Test Cat", DisplayOrder: 1})
	_ = repo.CreateCar(ctx, "100", "Test Racer", "Test Car", "")
	cars, _ := repo.ListCars(ctx)

	// Initial vote while open
	_, err := votingSvc.SubmitVote(ctx, models.Vote{
		VoterQR:    "VOTER-1",
		CategoryID: int(catID),
		CarID:      cars[0].ID,
	})
	if err != nil {
		t.Fatalf("Initial vote failed: %v", err)
	}

	// Close voting
	err = settingsSvc.CloseVoting(ctx)
	if err != nil {
		t.Fatalf("Failed to close voting: %v", err)
	}

	// Try to submit new vote - should fail
	_, err = votingSvc.SubmitVote(ctx, models.Vote{
		VoterQR:    "VOTER-2",
		CategoryID: int(catID),
		CarID:      cars[0].ID,
	})
	if err != services.ErrVotingClosed {
		t.Errorf("Expected ErrVotingClosed, got: %v", err)
	}

	// Try to change existing vote - should also fail
	_, err = votingSvc.SubmitVote(ctx, models.Vote{
		VoterQR:    "VOTER-1",
		CategoryID: int(catID),
		CarID:      cars[0].ID,
	})
	if err != services.ErrVotingClosed {
		t.Errorf("Expected ErrVotingClosed when changing vote, got: %v", err)
	}

	// Reopen voting
	err = settingsSvc.OpenVoting(ctx)
	if err != nil {
		t.Fatalf("Failed to reopen voting: %v", err)
	}

	// Now voting should work again
	_, err = votingSvc.SubmitVote(ctx, models.Vote{
		VoterQR:    "VOTER-2",
		CategoryID: int(catID),
		CarID:      cars[0].ID,
	})
	if err != nil {
		t.Errorf("Vote after reopening failed: %v", err)
	}

	// Verify both votes exist
	voterID1, _ := repo.GetVoterByQR(ctx, "VOTER-1")
	voterID2, _ := repo.GetVoterByQR(ctx, "VOTER-2")
	votes1, _ := repo.GetVoterVotes(ctx, voterID1)
	votes2, _ := repo.GetVoterVotes(ctx, voterID2)

	if len(votes1) != 1 || len(votes2) != 1 {
		t.Errorf("Expected both voters to have 1 vote each")
	}
}
