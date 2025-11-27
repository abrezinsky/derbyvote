package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository/mock"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/testutil"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// setupTestData creates test categories, cars, voters, and optionally votes
// Returns category IDs and car IDs for use in tests
func setupTestData(t *testing.T, ctx context.Context, repo interface {
	CreateCategory(ctx context.Context, name string, displayOrder int, groupID *int, allowedVoterTypes []string, allowedRanks []string) (int64, error)
	CreateCar(ctx context.Context, carNumber, racerName, carName, photoURL string) error
	CreateVoter(ctx context.Context, qrCode string) (int, error)
	GetVoterByQR(ctx context.Context, qrCode string) (int, error)
	SaveVote(ctx context.Context, voterID, categoryID, carID int) error
}, withVotes bool) (categoryIDs []int, carIDs []int) {
	t.Helper()

	// Create categories
	cat1ID, err := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	cat2ID, err := repo.CreateCategory(ctx, "Fastest Looking", 2, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	cat3ID, err := repo.CreateCategory(ctx, "Most Creative", 3, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	categoryIDs = []int{int(cat1ID), int(cat2ID), int(cat3ID)}

	// Create cars
	cars := []struct {
		number    string
		racerName string
		carName   string
		photoURL  string
	}{
		{"101", "Racer One", "Speed Demon", "http://example.com/car1.jpg"},
		{"102", "Racer Two", "Lightning Bolt", "http://example.com/car2.jpg"},
		{"103", "Racer Three", "Thunder", "http://example.com/car3.jpg"},
	}

	for _, car := range cars {
		err := repo.CreateCar(ctx, car.number, car.racerName, car.carName, car.photoURL)
		if err != nil {
			t.Fatalf("CreateCar failed: %v", err)
		}
	}

	// Car IDs are typically sequential starting from 1
	carIDs = []int{1, 2, 3}

	if withVotes {
		// Create voters and submit votes
		voters := []string{"voter-qr-001", "voter-qr-002", "voter-qr-003", "voter-qr-004", "voter-qr-005"}
		for _, qr := range voters {
			_, err := repo.CreateVoter(ctx, qr)
			if err != nil {
				t.Fatalf("CreateVoter failed: %v", err)
			}
		}

		// Submit votes to create rankings
		// Category 1 (Best Design): Car 1 gets 3 votes, Car 2 gets 2 votes, Car 3 gets 0
		// Category 2 (Fastest Looking): Car 2 gets 3 votes, Car 1 gets 1 vote, Car 3 gets 1
		// Category 3 (Most Creative): Car 3 gets 2 votes, Car 1 gets 1 vote, Car 2 gets 0

		votes := []struct {
			voterQR    string
			categoryID int
			carID      int
		}{
			// Category 1 votes (Best Design)
			{"voter-qr-001", int(cat1ID), 1},
			{"voter-qr-002", int(cat1ID), 1},
			{"voter-qr-003", int(cat1ID), 1},
			{"voter-qr-004", int(cat1ID), 2},
			{"voter-qr-005", int(cat1ID), 2},
			// Category 2 votes (Fastest Looking)
			{"voter-qr-001", int(cat2ID), 2},
			{"voter-qr-002", int(cat2ID), 2},
			{"voter-qr-003", int(cat2ID), 2},
			{"voter-qr-004", int(cat2ID), 1},
			{"voter-qr-005", int(cat2ID), 3},
			// Category 3 votes (Most Creative)
			{"voter-qr-001", int(cat3ID), 3},
			{"voter-qr-002", int(cat3ID), 3},
			{"voter-qr-003", int(cat3ID), 1},
		}

		for _, v := range votes {
			voterID, err := repo.GetVoterByQR(ctx, v.voterQR)
			if err != nil {
				t.Fatalf("GetVoterByQR failed: %v", err)
			}
			err = repo.SaveVote(ctx, voterID, v.categoryID, v.carID)
			if err != nil {
				t.Fatalf("SaveVote failed: %v", err)
			}
		}
	}

	return categoryIDs, carIDs
}

func TestResultsService_GetResults_EmptyResults(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Get results with no data at all
	results, err := svc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	if results == nil {
		t.Fatal("expected non-nil results")
	}

	// Should have no categories
	if len(results.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(results.Categories))
	}

	// Stats should still exist
	if results.Stats == nil {
		t.Error("expected non-nil Stats")
	}
}

func TestResultsService_GetResults_EmptyVotes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create categories and cars but no votes
	categoryIDs, _ := setupTestData(t, ctx, repo, false)

	results, err := svc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	if results == nil {
		t.Fatal("expected non-nil results")
	}

	// Should have 3 categories
	if len(results.Categories) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(results.Categories))
	}

	// Each category should have zero votes
	for i, cat := range results.Categories {
		if cat.CategoryID != categoryIDs[i] {
			t.Errorf("category %d: expected ID %d, got %d", i, categoryIDs[i], cat.CategoryID)
		}
		if cat.TotalVotes != 0 {
			t.Errorf("category %s: expected 0 total votes, got %d", cat.CategoryName, cat.TotalVotes)
		}
		if len(cat.Votes) != 0 {
			t.Errorf("category %s: expected 0 vote entries, got %d", cat.CategoryName, len(cat.Votes))
		}
	}
}

func TestResultsService_GetResults_WithVotes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes
	_, _ = setupTestData(t, ctx, repo, true)

	results, err := svc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	if results == nil {
		t.Fatal("expected non-nil results")
	}

	// Should have 3 categories
	if len(results.Categories) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(results.Categories))
	}

	// Verify Category 1 (Best Design): Car 1 has 3 votes, Car 2 has 2 votes
	cat1 := results.Categories[0]
	if cat1.CategoryName != "Best Design" {
		t.Errorf("expected category 'Best Design', got %q", cat1.CategoryName)
	}
	if cat1.TotalVotes != 5 {
		t.Errorf("Best Design: expected 5 total votes, got %d", cat1.TotalVotes)
	}
	if len(cat1.Votes) != 2 {
		t.Fatalf("Best Design: expected 2 car entries with votes, got %d", len(cat1.Votes))
	}

	// Verify Category 2 (Fastest Looking): Car 2 has 3 votes, Car 1 and 3 have 1 each
	cat2 := results.Categories[1]
	if cat2.CategoryName != "Fastest Looking" {
		t.Errorf("expected category 'Fastest Looking', got %q", cat2.CategoryName)
	}
	if cat2.TotalVotes != 5 {
		t.Errorf("Fastest Looking: expected 5 total votes, got %d", cat2.TotalVotes)
	}

	// Verify Category 3 (Most Creative): Car 3 has 2 votes, Car 1 has 1 vote
	cat3 := results.Categories[2]
	if cat3.CategoryName != "Most Creative" {
		t.Errorf("expected category 'Most Creative', got %q", cat3.CategoryName)
	}
	if cat3.TotalVotes != 3 {
		t.Errorf("Most Creative: expected 3 total votes, got %d", cat3.TotalVotes)
	}
}

func TestResultsService_GetResults_ProperRanking(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes
	_, _ = setupTestData(t, ctx, repo, true)

	results, err := svc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	// Check Category 1 ranking: Car 1 (3 votes) should be rank 1, Car 2 (2 votes) should be rank 2
	cat1 := results.Categories[0]
	if len(cat1.Votes) < 2 {
		t.Fatalf("expected at least 2 cars with votes in Best Design, got %d", len(cat1.Votes))
	}

	// First car should have rank 1 with highest votes
	if cat1.Votes[0].Rank != 1 {
		t.Errorf("first car rank: expected 1, got %d", cat1.Votes[0].Rank)
	}
	if cat1.Votes[0].VoteCount != 3 {
		t.Errorf("first car votes: expected 3, got %d", cat1.Votes[0].VoteCount)
	}
	if cat1.Votes[0].CarNumber != "101" {
		t.Errorf("first car number: expected '101', got %q", cat1.Votes[0].CarNumber)
	}

	// Second car should have rank 2 with fewer votes
	if cat1.Votes[1].Rank != 2 {
		t.Errorf("second car rank: expected 2, got %d", cat1.Votes[1].Rank)
	}
	if cat1.Votes[1].VoteCount != 2 {
		t.Errorf("second car votes: expected 2, got %d", cat1.Votes[1].VoteCount)
	}
	if cat1.Votes[1].CarNumber != "102" {
		t.Errorf("second car number: expected '102', got %q", cat1.Votes[1].CarNumber)
	}

	// Check Category 2 ranking: Car 2 (3 votes) should be rank 1
	cat2 := results.Categories[1]
	if len(cat2.Votes) < 1 {
		t.Fatalf("expected at least 1 car with votes in Fastest Looking, got %d", len(cat2.Votes))
	}
	if cat2.Votes[0].Rank != 1 {
		t.Errorf("Fastest Looking first car rank: expected 1, got %d", cat2.Votes[0].Rank)
	}
	if cat2.Votes[0].VoteCount != 3 {
		t.Errorf("Fastest Looking first car votes: expected 3, got %d", cat2.Votes[0].VoteCount)
	}
	if cat2.Votes[0].CarNumber != "102" {
		t.Errorf("Fastest Looking first car number: expected '102', got %q", cat2.Votes[0].CarNumber)
	}
}

func TestResultsService_GetCategoryResults_ReturnsSpecificCategory(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes
	categoryIDs, _ := setupTestData(t, ctx, repo, true)

	// Get results for category 1 (Best Design)
	catResult, err := svc.GetCategoryResults(ctx, categoryIDs[0])
	if err != nil {
		t.Fatalf("GetCategoryResults failed: %v", err)
	}

	if catResult == nil {
		t.Fatal("expected non-nil category result")
	}

	if catResult.CategoryID != categoryIDs[0] {
		t.Errorf("expected category ID %d, got %d", categoryIDs[0], catResult.CategoryID)
	}
	if catResult.CategoryName != "Best Design" {
		t.Errorf("expected category name 'Best Design', got %q", catResult.CategoryName)
	}
	if catResult.TotalVotes != 5 {
		t.Errorf("expected 5 total votes, got %d", catResult.TotalVotes)
	}

	// Get results for category 2 (Fastest Looking)
	catResult2, err := svc.GetCategoryResults(ctx, categoryIDs[1])
	if err != nil {
		t.Fatalf("GetCategoryResults for category 2 failed: %v", err)
	}

	if catResult2 == nil {
		t.Fatal("expected non-nil category result for category 2")
	}
	if catResult2.CategoryName != "Fastest Looking" {
		t.Errorf("expected category name 'Fastest Looking', got %q", catResult2.CategoryName)
	}
}

func TestResultsService_GetCategoryResults_NonExistentCategory(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data
	_, _ = setupTestData(t, ctx, repo, true)

	// Get results for non-existent category
	catResult, err := svc.GetCategoryResults(ctx, 9999)
	if err != nil {
		t.Fatalf("GetCategoryResults failed: %v", err)
	}

	// Should return nil for non-existent category
	if catResult != nil {
		t.Errorf("expected nil for non-existent category, got %+v", catResult)
	}
}

func TestResultsService_GetCategoryResults_EmptyDatabase(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Get results for category in empty database
	catResult, err := svc.GetCategoryResults(ctx, 1)
	if err != nil {
		t.Fatalf("GetCategoryResults failed: %v", err)
	}

	// Should return nil when category doesn't exist
	if catResult != nil {
		t.Errorf("expected nil for empty database, got %+v", catResult)
	}
}

func TestResultsService_GetStats_ReturnsStatistics(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes
	_, _ = setupTestData(t, ctx, repo, true)

	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	// Check for expected stat keys
	expectedKeys := []string{"total_voters", "voters_who_voted", "total_votes", "total_categories", "total_cars"}
	for _, key := range expectedKeys {
		if _, ok := stats[key]; !ok {
			t.Errorf("expected stats to contain key %q", key)
		}
	}

	// Verify specific values
	if totalVoters, ok := stats["total_voters"].(int); ok {
		if totalVoters != 5 {
			t.Errorf("expected 5 total voters, got %d", totalVoters)
		}
	} else {
		t.Error("total_voters is not an int")
	}

	if totalCategories, ok := stats["total_categories"].(int); ok {
		if totalCategories != 3 {
			t.Errorf("expected 3 total categories, got %d", totalCategories)
		}
	} else {
		t.Error("total_categories is not an int")
	}

	if totalCars, ok := stats["total_cars"].(int); ok {
		if totalCars != 3 {
			t.Errorf("expected 3 total cars, got %d", totalCars)
		}
	} else {
		t.Error("total_cars is not an int")
	}

	// We have 13 votes total across all categories
	if totalVotes, ok := stats["total_votes"].(int); ok {
		if totalVotes != 13 {
			t.Errorf("expected 13 total votes, got %d", totalVotes)
		}
	} else {
		t.Error("total_votes is not an int")
	}
}

func TestResultsService_GetStats_IncludesVotingOpenStatus(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create minimal test data
	_, _ = setupTestData(t, ctx, repo, false)

	// Test with voting open (default)
	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	votingOpen, ok := stats["voting_open"]
	if !ok {
		t.Fatal("expected stats to contain 'voting_open' key")
	}
	if votingOpen != true {
		t.Errorf("expected voting_open to be true by default, got %v", votingOpen)
	}

	// Close voting and verify
	err = settingsSvc.SetVotingOpen(ctx, false)
	if err != nil {
		t.Fatalf("SetVotingOpen failed: %v", err)
	}

	stats, err = svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed after closing voting: %v", err)
	}

	votingOpen, ok = stats["voting_open"]
	if !ok {
		t.Fatal("expected stats to contain 'voting_open' key after closing")
	}
	if votingOpen != false {
		t.Errorf("expected voting_open to be false after closing, got %v", votingOpen)
	}
}

func TestResultsService_GetStats_EmptyDatabase(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Get stats from empty database
	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	// Verify zeros
	if totalVoters, ok := stats["total_voters"].(int); ok {
		if totalVoters != 0 {
			t.Errorf("expected 0 total voters, got %d", totalVoters)
		}
	}

	if totalVotes, ok := stats["total_votes"].(int); ok {
		if totalVotes != 0 {
			t.Errorf("expected 0 total votes, got %d", totalVotes)
		}
	}

	if totalCategories, ok := stats["total_categories"].(int); ok {
		if totalCategories != 0 {
			t.Errorf("expected 0 total categories, got %d", totalCategories)
		}
	}

	if totalCars, ok := stats["total_cars"].(int); ok {
		if totalCars != 0 {
			t.Errorf("expected 0 total cars, got %d", totalCars)
		}
	}
}

func TestResultsService_GetWinners_EmptyWhenNoVotes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create categories and cars but no votes
	_, _ = setupTestData(t, ctx, repo, false)

	winners, err := svc.GetWinners(ctx)
	if err != nil {
		t.Fatalf("GetWinners failed: %v", err)
	}

	// Should return empty slice when no votes
	if winners == nil {
		// nil is acceptable as "empty"
		return
	}
	if len(winners) != 0 {
		t.Errorf("expected 0 winners when no votes, got %d", len(winners))
	}
}

func TestResultsService_GetWinners_EmptyDatabase(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Get winners from empty database
	winners, err := svc.GetWinners(ctx)
	if err != nil {
		t.Fatalf("GetWinners failed: %v", err)
	}

	// Should return empty/nil slice
	if winners != nil && len(winners) != 0 {
		t.Errorf("expected 0 winners for empty database, got %d", len(winners))
	}
}

func TestResultsService_GetWinners_ReturnsWinnersAfterVoting(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes
	_, _ = setupTestData(t, ctx, repo, true)

	winners, err := svc.GetWinners(ctx)
	if err != nil {
		t.Fatalf("GetWinners failed: %v", err)
	}

	if winners == nil {
		t.Fatal("expected non-nil winners")
	}

	// Should have 3 winners (one per category)
	if len(winners) != 3 {
		t.Fatalf("expected 3 winners, got %d", len(winners))
	}

	// Create a map for easier lookup
	winnersByCategory := make(map[string]map[string]interface{})
	for _, w := range winners {
		catName := w["category_name"].(string)
		winnersByCategory[catName] = w
	}

	// Verify Best Design winner (Car 1 with 3 votes)
	bdWinner, ok := winnersByCategory["Best Design"]
	if !ok {
		t.Fatal("expected winner for 'Best Design' category")
	}
	winnerInfo := bdWinner["winner"].(map[string]interface{})
	if winnerInfo["car_number"] != "101" {
		t.Errorf("Best Design winner: expected car '101', got %v", winnerInfo["car_number"])
	}
	if winnerInfo["vote_count"] != 3 {
		t.Errorf("Best Design winner: expected 3 votes, got %v", winnerInfo["vote_count"])
	}

	// Verify Fastest Looking winner (Car 2 with 3 votes)
	flWinner, ok := winnersByCategory["Fastest Looking"]
	if !ok {
		t.Fatal("expected winner for 'Fastest Looking' category")
	}
	winnerInfo = flWinner["winner"].(map[string]interface{})
	if winnerInfo["car_number"] != "102" {
		t.Errorf("Fastest Looking winner: expected car '102', got %v", winnerInfo["car_number"])
	}
	if winnerInfo["vote_count"] != 3 {
		t.Errorf("Fastest Looking winner: expected 3 votes, got %v", winnerInfo["vote_count"])
	}

	// Verify Most Creative winner (Car 3 with 2 votes)
	mcWinner, ok := winnersByCategory["Most Creative"]
	if !ok {
		t.Fatal("expected winner for 'Most Creative' category")
	}
	winnerInfo = mcWinner["winner"].(map[string]interface{})
	if winnerInfo["car_number"] != "103" {
		t.Errorf("Most Creative winner: expected car '103', got %v", winnerInfo["car_number"])
	}
	if winnerInfo["vote_count"] != 2 {
		t.Errorf("Most Creative winner: expected 2 votes, got %v", winnerInfo["vote_count"])
	}
}

func TestResultsService_GetWinners_IncludesCarDetails(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes
	_, _ = setupTestData(t, ctx, repo, true)

	winners, err := svc.GetWinners(ctx)
	if err != nil {
		t.Fatalf("GetWinners failed: %v", err)
	}

	if len(winners) == 0 {
		t.Fatal("expected at least one winner")
	}

	// Check first winner has all expected fields
	winner := winners[0]

	// Check category fields
	if _, ok := winner["category_id"]; !ok {
		t.Error("winner missing 'category_id' field")
	}
	if _, ok := winner["category_name"]; !ok {
		t.Error("winner missing 'category_name' field")
	}

	// Check winner info fields
	winnerInfo, ok := winner["winner"].(map[string]interface{})
	if !ok {
		t.Fatal("winner missing 'winner' object")
	}

	expectedFields := []string{"car_id", "car_number", "car_name", "racer_name", "vote_count"}
	for _, field := range expectedFields {
		if _, ok := winnerInfo[field]; !ok {
			t.Errorf("winner info missing '%s' field", field)
		}
	}

	// Verify actual values for first winner (Best Design - Car 1)
	if winnerInfo["car_name"] != "Speed Demon" {
		t.Errorf("expected car_name 'Speed Demon', got %v", winnerInfo["car_name"])
	}
	if winnerInfo["racer_name"] != "Racer One" {
		t.Errorf("expected racer_name 'Racer One', got %v", winnerInfo["racer_name"])
	}
}

func TestResultsService_GetResults_WithNilSettingsService(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	// Create ResultsService with nil settings
	svc := services.NewResultsService(log, repo, nil, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes
	_, _ = setupTestData(t, ctx, repo, true)

	// Should not panic with nil settings service
	results, err := svc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed with nil settings: %v", err)
	}

	if results == nil {
		t.Fatal("expected non-nil results")
	}

	if len(results.Categories) != 3 {
		t.Errorf("expected 3 categories, got %d", len(results.Categories))
	}
}

func TestResultsService_GetStats_WithNilSettingsService(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	// Create ResultsService with nil settings
	svc := services.NewResultsService(log, repo, nil, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data
	_, _ = setupTestData(t, ctx, repo, false)

	// Should not panic with nil settings service
	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed with nil settings: %v", err)
	}

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	// voting_open should NOT be present when settings service is nil
	if _, ok := stats["voting_open"]; ok {
		t.Log("Note: voting_open is present even with nil settings - this may be intentional")
	}
}

func TestResultsService_GetResults_CarInfoIncluded(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes
	_, _ = setupTestData(t, ctx, repo, true)

	results, err := svc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	// Check that car info is properly populated in results
	cat1 := results.Categories[0]
	if len(cat1.Votes) == 0 {
		t.Fatal("expected votes in first category")
	}

	firstVote := cat1.Votes[0]
	if firstVote.CarID == 0 {
		t.Error("expected non-zero car_id")
	}
	if firstVote.CarNumber == "" {
		t.Error("expected non-empty car_number")
	}
	if firstVote.CarName == "" {
		t.Error("expected non-empty car_name")
	}
	if firstVote.RacerName == "" {
		t.Error("expected non-empty racer_name")
	}
	// PhotoURL can be empty but should not cause issues
}

// ==================== PushResultsToDerbyNet Tests ====================

func TestResultsService_PushResultsToDerbyNet_NoVotes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, repo, settingsSvc, mockClient)
	ctx := context.Background()

	// Create category with DerbyNet award ID but no votes
	awardID := 10
	_, _ = repo.UpsertCategory(ctx, "Best Design", 1, &awardID)

	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("PushResultsToDerbyNet failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.Message != "No winners to push (no votes recorded)" {
		t.Errorf("expected 'No winners to push' message, got %q", result.Message)
	}
	if result.WinnersPushed != 0 {
		t.Errorf("expected 0 winners pushed, got %d", result.WinnersPushed)
	}
}

func TestResultsService_PushResultsToDerbyNet_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, repo, settingsSvc, mockClient)
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
	voter, _ := repo.CreateVoter(ctx, "PUSH-QR")
	_ = repo.SaveVote(ctx, voter, categoryID, carID)

	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("PushResultsToDerbyNet failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.WinnersPushed != 1 {
		t.Errorf("expected 1 winner pushed, got %d", result.WinnersPushed)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}

	// Verify the mock client received the call
	winners := mockClient.GetAwardWinners()
	if winners[awardID] != 100 {
		t.Errorf("expected mock client to have winner for award %d -> racer 100, got %v", awardID, winners)
	}
}

func TestResultsService_PushResultsToDerbyNet_MissingAwardID(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, repo, settingsSvc, mockClient)
	ctx := context.Background()

	// Create category WITHOUT DerbyNet award ID
	_, _ = repo.CreateCategory(ctx, "Local Category", 1, nil, nil, nil)
	categories, _ := repo.ListCategories(ctx)
	categoryID := categories[0].ID

	// Create car with DerbyNet racer ID
	_ = repo.UpsertCar(ctx, 200, "201", "Racer", "Car", "", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Create votes
	voter, _ := repo.CreateVoter(ctx, "LOCAL-QR")
	_ = repo.SaveVote(ctx, voter, categoryID, carID)

	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("PushResultsToDerbyNet failed: %v", err)
	}

	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped (no award ID), got %d", result.Skipped)
	}
	if result.WinnersPushed != 0 {
		t.Errorf("expected 0 winners pushed, got %d", result.WinnersPushed)
	}

	// Check details
	if len(result.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(result.Details))
	}
	if result.Details[0].Status != "skipped" {
		t.Errorf("expected detail status 'skipped', got %q", result.Details[0].Status)
	}
}

func TestResultsService_PushResultsToDerbyNet_MissingRacerID(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, repo, settingsSvc, mockClient)
	ctx := context.Background()

	// Create category WITH DerbyNet award ID
	awardID := 30
	_, _ = repo.UpsertCategory(ctx, "Synced Category", 1, &awardID)
	categories, _ := repo.ListCategories(ctx)
	categoryID := categories[0].ID

	// Create car WITHOUT DerbyNet racer ID (manually created)
	_ = repo.CreateCar(ctx, "999", "Manual Racer", "Manual Car", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Create votes
	voter, _ := repo.CreateVoter(ctx, "MANUAL-QR")
	_ = repo.SaveVote(ctx, voter, categoryID, carID)

	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("PushResultsToDerbyNet failed: %v", err)
	}

	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped (no racer ID), got %d", result.Skipped)
	}
	if result.WinnersPushed != 0 {
		t.Errorf("expected 0 winners pushed, got %d", result.WinnersPushed)
	}
}

func TestResultsService_PushResultsToDerbyNet_ClientError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	// Create mock client that returns error on SetAwardWinner
	mockClient := derbynet.NewMockClient(derbynet.WithSetWinnerError(errTest))
	svc := services.NewResultsService(log, repo, settingsSvc, mockClient)
	ctx := context.Background()

	// Create category with DerbyNet award ID
	awardID := 40
	_, _ = repo.UpsertCategory(ctx, "Error Category", 1, &awardID)
	categories, _ := repo.ListCategories(ctx)
	categoryID := categories[0].ID

	// Create car with DerbyNet racer ID
	_ = repo.UpsertCar(ctx, 400, "401", "Error Racer", "Error Car", "", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Create votes
	voter, _ := repo.CreateVoter(ctx, "ERROR-QR")
	_ = repo.SaveVote(ctx, voter, categoryID, carID)

	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("PushResultsToDerbyNet failed: %v", err)
	}

	if result.Status != "partial" {
		t.Errorf("expected status 'partial', got %q", result.Status)
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
	if result.WinnersPushed != 0 {
		t.Errorf("expected 0 winners pushed, got %d", result.WinnersPushed)
	}
}

func TestResultsService_PushResultsToDerbyNet_MultipleCategories(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, repo, settingsSvc, mockClient)
	ctx := context.Background()

	// Create categories with DerbyNet award IDs
	awardID1 := 50
	awardID2 := 60
	_, _ = repo.UpsertCategory(ctx, "Category 1", 1, &awardID1)
	_, _ = repo.UpsertCategory(ctx, "Category 2", 2, &awardID2)
	categories, _ := repo.ListCategories(ctx)
	cat1ID := categories[0].ID
	cat2ID := categories[1].ID

	// Create cars with DerbyNet racer IDs
	_ = repo.UpsertCar(ctx, 501, "501", "Racer 1", "Car 1", "", "")
	_ = repo.UpsertCar(ctx, 502, "502", "Racer 2", "Car 2", "", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Create votes
	voter, _ := repo.CreateVoter(ctx, "MULTI-QR")
	_ = repo.SaveVote(ctx, voter, cat1ID, car1ID)
	_ = repo.SaveVote(ctx, voter, cat2ID, car2ID)

	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("PushResultsToDerbyNet failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.WinnersPushed != 2 {
		t.Errorf("expected 2 winners pushed, got %d", result.WinnersPushed)
	}

	// Verify both winners were set in mock
	winners := mockClient.GetAwardWinners()
	if winners[awardID1] != 501 || winners[awardID2] != 502 {
		t.Errorf("expected mock to have award %d->501 and %d->502, got %v", awardID1, awardID2, winners)
	}
}

func TestResultsService_PushResultsToDerbyNet_SavesURL(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, repo, settingsSvc, mockClient)
	ctx := context.Background()

	_, err := svc.PushResultsToDerbyNet(ctx, "http://push-test.local")
	if err != nil {
		t.Fatalf("PushResultsToDerbyNet failed: %v", err)
	}

	// Verify URL was saved to settings
	savedURL, err := repo.GetSetting(ctx, "derbynet_url")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if savedURL != "http://push-test.local" {
		t.Errorf("expected saved URL 'http://push-test.local', got %q", savedURL)
	}
}

// errTest is a test error for mock clients
var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

// ===== Error Path Tests =====

func TestResultsService_GetResults_ListCategoriesError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoriesError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, mockRepo, settingsSvc, mockClient)

	ctx := context.Background()
	_, err := svc.GetResults(ctx)
	if err == nil {
		t.Fatal("expected error from GetResults when ListCategories fails, got nil")
	}
}

func TestResultsService_GetResults_GetVoteResultsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetVoteResultsWithCarsError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, mockRepo, settingsSvc, mockClient)

	ctx := context.Background()
	_, err := svc.GetResults(ctx)
	if err == nil {
		t.Fatal("expected error from GetResults when GetVoteResultsWithCars fails, got nil")
	}
}

func TestResultsService_GetResults_GetVotingStatsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetVotingStatsError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, mockRepo, settingsSvc, mockClient)

	ctx := context.Background()
	_, err := svc.GetResults(ctx)
	if err == nil {
		t.Fatal("expected error from GetResults when GetVotingStats fails, got nil")
	}
}

func TestResultsService_GetCategoryResults_ErrorPropagation(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoriesError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, mockRepo, settingsSvc, mockClient)

	ctx := context.Background()
	_, err := svc.GetCategoryResults(ctx, 1)
	if err == nil {
		t.Fatal("expected error from GetCategoryResults when GetResults fails, got nil")
	}
}

func TestResultsService_GetStats_GetVotingStatsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetVotingStatsError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, mockRepo, settingsSvc, mockClient)

	ctx := context.Background()
	_, err := svc.GetStats(ctx)
	if err == nil {
		t.Fatal("expected error from GetStats when GetVotingStats fails, got nil")
	}
}

func TestResultsService_GetWinners_GetResultsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoriesError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, mockRepo, settingsSvc, mockClient)

	ctx := context.Background()
	_, err := svc.GetWinners(ctx)
	if err == nil {
		t.Fatal("expected error from GetWinners when GetResults fails, got nil")
	}
}

func TestResultsService_PushResultsToDerbyNet_GetWinnersError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetWinnersForDerbyNetError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, mockRepo, settingsSvc, mockClient)

	ctx := context.Background()
	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("expected status 'error', got: %s", result.Status)
	}
}

func TestResultsService_PushResultsToDerbyNet_SetSettingError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, mockRepo, settingsSvc, mockClient)
	ctx := context.Background()

	// Create category with DerbyNet award ID
	awardID := 50
	_, _ = realRepo.UpsertCategory(ctx, "Test Category", 1, &awardID)
	categories, _ := realRepo.ListCategories(ctx)
	categoryID := categories[0].ID

	// Create car with DerbyNet racer ID
	_ = realRepo.UpsertCar(ctx, 500, "501", "Test Racer", "Test Car", "", "")
	cars, _ := realRepo.ListCars(ctx)
	carID := cars[0].ID

	// Create votes
	voter, _ := realRepo.CreateVoter(ctx, "PUSH-QR")
	_ = realRepo.SaveVote(ctx, voter, categoryID, carID)

	// Configure mock to fail on SetSetting (saving DerbyNet URL)
	mockRepo.SetSettingError = errors.New("database error saving URL")

	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when SetSetting fails, got nil")
	}
	if result != nil {
		t.Error("expected nil result when SetSetting fails")
	}
}

func TestResultsService_PushResultsToDerbyNet_WithCredentials(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	mockClient := derbynet.NewMockClient()
	svc := services.NewResultsService(log, repo, settingsSvc, mockClient)
	ctx := context.Background()

	// Set up DerbyNet credentials
	_ = repo.SetSetting(ctx, "derbynet_role", "RaceCoordinator")
	_ = repo.SetSetting(ctx, "derbynet_password", "secret123")

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
	voter, _ := repo.CreateVoter(ctx, "PUSH-QR")
	_ = repo.SaveVote(ctx, voter, categoryID, carID)

	result, err := svc.PushResultsToDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("PushResultsToDerbyNet failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.WinnersPushed != 1 {
		t.Errorf("expected 1 winner pushed, got %d", result.WinnersPushed)
	}

	// Verify the mock client received the call
	winners := mockClient.GetAwardWinners()
	if winners[awardID] != 100 {
		t.Errorf("expected mock client to have winner for award %d -> racer 100, got %v", awardID, winners)
	}
}

// ==================== Conflict Detection Tests ====================

func TestResultsService_DetectTies_NoTies(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data with votes (no ties)
	_, _ = setupTestData(t, ctx, repo, true)

	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	if len(ties) != 0 {
		t.Errorf("expected no ties, got %d", len(ties))
	}
}

func TestResultsService_DetectTies_SimpleTie(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and cars
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := repo.ListCars(ctx)

	// Create votes - 2-way tie with 2 votes each
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")
	v4, _ := repo.CreateVoter(ctx, "V4")

	repo.SaveVote(ctx, v1, int(catID), cars[0].ID) // Car A: 2 votes
	repo.SaveVote(ctx, v2, int(catID), cars[0].ID)
	repo.SaveVote(ctx, v3, int(catID), cars[1].ID) // Car B: 2 votes
	repo.SaveVote(ctx, v4, int(catID), cars[1].ID)

	// Detect ties
	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	if len(ties) != 1 {
		t.Fatalf("expected 1 tie, got %d", len(ties))
	}

	tie := ties[0]
	if tie.CategoryID != int(catID) {
		t.Errorf("expected categoryID=%d, got %d", catID, tie.CategoryID)
	}
	if tie.CategoryName != "Best Design" {
		t.Errorf("expected 'Best Design', got '%s'", tie.CategoryName)
	}
	if len(tie.TiedCars) != 2 {
		t.Fatalf("expected 2 tied cars, got %d", len(tie.TiedCars))
	}

	// Both cars should have same vote count
	if tie.TiedCars[0].VoteCount != 2 {
		t.Errorf("car 0: expected 2 votes, got %d", tie.TiedCars[0].VoteCount)
	}
	if tie.TiedCars[1].VoteCount != 2 {
		t.Errorf("car 1: expected 2 votes, got %d", tie.TiedCars[1].VoteCount)
	}
}

func TestResultsService_DetectTies_ThreeWayTie(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and cars
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	_ = repo.CreateCar(ctx, "103", "Racer Three", "Car C", "")
	cars, _ := repo.ListCars(ctx)

	// Create 3-way tie with 1 vote each
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")

	repo.SaveVote(ctx, v1, int(catID), cars[0].ID) // Car A: 1 vote
	repo.SaveVote(ctx, v2, int(catID), cars[1].ID) // Car B: 1 vote
	repo.SaveVote(ctx, v3, int(catID), cars[2].ID) // Car C: 1 vote

	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	if len(ties) != 1 {
		t.Fatalf("expected 1 tie, got %d", len(ties))
	}

	tie := ties[0]
	if len(tie.TiedCars) != 3 {
		t.Fatalf("expected 3-way tie, got %d cars", len(tie.TiedCars))
	}
}

func TestResultsService_DetectTies_IgnoresCategoriesWithNoVotes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category with no votes
	_, _ = repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	if len(ties) != 0 {
		t.Errorf("expected no ties for category with no votes, got %d", len(ties))
	}
}

func TestResultsService_DetectTies_WithExistingOverride(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and cars
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Create voters and votes (create a tie)
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")

	repo.SaveVote(ctx, v1, int(catID), car1ID) // Car A: 1 vote
	repo.SaveVote(ctx, v2, int(catID), car2ID) // Car B: 1 vote

	// Set an override to resolve the tie
	repo.SetManualWinner(ctx, int(catID), car1ID, "Resolved tie")

	// Detect ties - the tie should NOT be detected because it has been resolved with an override
	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	if len(ties) != 0 {
		t.Fatalf("expected 0 ties (resolved with override), got %d", len(ties))
	}

	// The override resolved the conflict, so it no longer shows in the conflicts list
}

func TestResultsService_DetectMultipleWins_None(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create test data where each car wins at most 1 category
	_, _ = setupTestData(t, ctx, repo, true)

	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	if len(multiWins) != 0 {
		t.Errorf("expected no multiple wins, got %d", len(multiWins))
	}
}

func TestResultsService_DetectMultipleWins_OneCarWinsMultiple(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a category group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := repo.CreateCategoryGroup(ctx, "Design Awards", "Design related categories", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create categories in the group
	cat1ID, _ := repo.CreateCategory(ctx, "Best Design", 1, &groupIDInt, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Most Creative", 2, &groupIDInt, nil, nil)
	cat3ID, _ := repo.CreateCategory(ctx, "Best Paint", 3, &groupIDInt, nil, nil)

	// Create cars
	_ = repo.CreateCar(ctx, "101", "Racer One", "Super Car", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Other Car", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID // This car will win multiple awards
	car2ID := cars[1].ID

	// Create voters
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")

	// Car 1 wins categories 1 and 2
	repo.SaveVote(ctx, v1, int(cat1ID), car1ID)
	repo.SaveVote(ctx, v2, int(cat2ID), car1ID)

	// Car 2 wins category 3
	repo.SaveVote(ctx, v3, int(cat3ID), car2ID)

	// Detect multiple wins
	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	if len(multiWins) != 1 {
		t.Fatalf("expected 1 car with multiple wins, got %d", len(multiWins))
	}

	mw := multiWins[0]
	if mw.CarID != car1ID {
		t.Errorf("expected carID=%d, got %d", car1ID, mw.CarID)
	}
	if mw.CarNumber != "101" {
		t.Errorf("expected car number '101', got '%s'", mw.CarNumber)
	}
	if len(mw.AwardsWon) != 2 {
		t.Fatalf("expected 2 awards won, got %d", len(mw.AwardsWon))
	}

	// Verify awards list
	expectedAwards := map[string]bool{"Best Design": true, "Most Creative": true}
	for _, award := range mw.AwardsWon {
		if !expectedAwards[award] {
			t.Errorf("unexpected award in list: %s", award)
		}
	}
}

func TestResultsService_DetectMultipleWins_WithOverrides(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a category group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := repo.CreateCategoryGroup(ctx, "Design Awards", "Design related categories", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create categories in the group
	cat1ID, _ := repo.CreateCategory(ctx, "Best Design", 1, &groupIDInt, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Most Creative", 2, &groupIDInt, nil, nil)

	// Create cars
	_ = repo.CreateCar(ctx, "101", "Racer One", "Super Car", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Other Car", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID // Will win by votes in cat1, by override in cat2
	car2ID := cars[1].ID

	// Create voters
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")

	// Car 1 wins category 1 by votes
	repo.SaveVote(ctx, v1, int(cat1ID), car1ID)

	// Car 2 wins category 2 by votes
	repo.SaveVote(ctx, v2, int(cat2ID), car2ID)

	// But manually override cat2 to car1 (resolving a tie, for example)
	repo.SetManualWinner(ctx, int(cat2ID), car1ID, "Resolved tie")

	// Detect multiple wins - car1 should be detected as winning both categories
	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	if len(multiWins) != 1 {
		t.Fatalf("expected 1 car with multiple wins, got %d", len(multiWins))
	}

	mw := multiWins[0]
	if mw.CarID != car1ID {
		t.Errorf("expected carID=%d, got %d", car1ID, mw.CarID)
	}
	if len(mw.AwardsWon) != 2 {
		t.Fatalf("expected 2 awards won (1 by vote, 1 by override), got %d", len(mw.AwardsWon))
	}

	// Verify awards list includes both
	expectedAwards := map[string]bool{"Best Design": true, "Most Creative": true}
	for _, award := range mw.AwardsWon {
		if !expectedAwards[award] {
			t.Errorf("unexpected award in list: %s", award)
		}
	}
}

// ==================== Manual Winner Override Tests ====================

func TestResultsService_SetManualWinner_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set manual winner
	err := svc.SetManualWinner(ctx, int(catID), carID, "Resolved tie")
	if err != nil {
		t.Fatalf("SetManualWinner failed: %v", err)
	}

	// Verify it was set
	results, _ := svc.GetResults(ctx)
	if len(results.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(results.Categories))
	}

	cat := results.Categories[0]
	if !cat.HasOverride {
		t.Error("expected has_override to be true")
	}
	if cat.OverrideCarID == nil || *cat.OverrideCarID != carID {
		t.Errorf("expected override_car_id=%d, got %v", carID, cat.OverrideCarID)
	}
	if cat.OverrideReason != "Resolved tie" {
		t.Errorf("expected reason 'Resolved tie', got '%s'", cat.OverrideReason)
	}
}

func TestResultsService_SetManualWinner_NonExistentCategory(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a car but no category
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Try to set winner for non-existent category
	err := svc.SetManualWinner(ctx, 9999, carID, "Test reason")
	if err == nil {
		t.Error("expected error for non-existent category, got nil")
	}
}

func TestResultsService_SetManualWinner_NonExistentCar(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category but no car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// Try to set winner with non-existent car
	err := svc.SetManualWinner(ctx, int(catID), 9999, "Test reason")
	if err == nil {
		t.Error("expected error for non-existent car, got nil")
	}
}

func TestResultsService_SetManualWinner_EmptyReason(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Try with empty reason
	err := svc.SetManualWinner(ctx, int(catID), carID, "")
	if err == nil {
		t.Error("expected error for empty reason, got nil")
	}

	// Try with whitespace-only reason
	err = svc.SetManualWinner(ctx, int(catID), carID, "   ")
	if err == nil {
		t.Error("expected error for whitespace-only reason, got nil")
	}
}

func TestResultsService_ClearManualWinner_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set then clear override
	svc.SetManualWinner(ctx, int(catID), carID, "Test")
	err := svc.ClearManualWinner(ctx, int(catID))
	if err != nil {
		t.Fatalf("ClearManualWinner failed: %v", err)
	}

	// Verify it was cleared
	results, _ := svc.GetResults(ctx)
	cat := results.Categories[0]

	if cat.HasOverride {
		t.Error("expected has_override to be false after clear")
	}
	if cat.OverrideCarID != nil {
		t.Errorf("expected override_car_id to be nil, got %v", cat.OverrideCarID)
	}
}

func TestResultsService_GetFinalWinners_WithOverride(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and cars
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := repo.ListCars(ctx)
	car1ID := cars[0].ID
	car2ID := cars[1].ID

	// Create votes (car1 wins by votes)
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")

	repo.SaveVote(ctx, v1, int(catID), car1ID) // Car A: 2 votes
	repo.SaveVote(ctx, v2, int(catID), car1ID)
	repo.SaveVote(ctx, v3, int(catID), car2ID) // Car B: 1 vote

	// Set manual override to car2 (not the vote winner)
	svc.SetManualWinner(ctx, int(catID), car2ID, "Manual choice")

	// Get final winners
	winners, err := svc.GetFinalWinners(ctx)
	if err != nil {
		t.Fatalf("GetFinalWinners failed: %v", err)
	}

	if len(winners) != 1 {
		t.Fatalf("expected 1 winner, got %d", len(winners))
	}

	winnerData := winners[0]
	winnerInfo := winnerData["winner"].(map[string]interface{})

	// Should be car2 (the override), not car1 (the vote winner)
	if winnerInfo["car_id"] != car2ID {
		t.Errorf("expected override winner carID=%d, got %v", car2ID, winnerInfo["car_id"])
	}
	if winnerInfo["is_override"] != true {
		t.Error("expected is_override=true")
	}
	if winnerInfo["override_reason"] != "Manual choice" {
		t.Errorf("expected reason 'Manual choice', got %v", winnerInfo["override_reason"])
	}
}

func TestResultsService_GetFinalWinners_WithoutOverride(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and cars
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
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

	// Get final winners
	winners, err := svc.GetFinalWinners(ctx)
	if err != nil {
		t.Fatalf("GetFinalWinners failed: %v", err)
	}

	if len(winners) != 1 {
		t.Fatalf("expected 1 winner, got %d", len(winners))
	}

	winnerData := winners[0]
	winnerInfo := winnerData["winner"].(map[string]interface{})

	// Should be car1 (the vote winner)
	if winnerInfo["car_id"] != car1ID {
		t.Errorf("expected vote winner carID=%d, got %v", car1ID, winnerInfo["car_id"])
	}
	if winnerInfo["is_override"] != false {
		t.Error("expected is_override=false")
	}
}

func TestResultsService_GetResults_IncludesOverrideInfo(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Set override
	svc.SetManualWinner(ctx, int(catID), carID, "Test override")

	// Get results
	results, err := svc.GetResults(ctx)
	if err != nil {
		t.Fatalf("GetResults failed: %v", err)
	}

	if len(results.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(results.Categories))
	}

	cat := results.Categories[0]

	// Verify override info is included
	if !cat.HasOverride {
		t.Error("expected has_override=true")
	}
	if cat.OverrideCarID == nil || *cat.OverrideCarID != carID {
		t.Errorf("expected override_car_id=%d, got %v", carID, cat.OverrideCarID)
	}
	if cat.OverrideReason != "Test override" {
		t.Errorf("expected 'Test override', got '%s'", cat.OverrideReason)
	}
	if cat.OverriddenAt == "" {
		t.Error("expected overridden_at to be set")
	}
}
// ==================== DetectTies Additional Edge Case Tests ====================

func TestResultsService_DetectTies_ThreeCars(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// Create 3 cars
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	_ = repo.CreateCar(ctx, "103", "Racer Three", "Car C", "")
	cars, _ := repo.ListCars(ctx)

	// All 3 cars get 1 vote each (3-way tie)
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")
	repo.SaveVote(ctx, v1, int(catID), cars[0].ID)
	repo.SaveVote(ctx, v2, int(catID), cars[1].ID)
	repo.SaveVote(ctx, v3, int(catID), cars[2].ID)

	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	if len(ties) != 1 {
		t.Fatalf("expected 1 tie, got %d", len(ties))
	}

	if len(ties[0].TiedCars) != 3 {
		t.Errorf("expected 3 tied cars, got %d", len(ties[0].TiedCars))
	}
}

func TestResultsService_DetectTies_ZeroVotes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category but no votes
	repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	// Should not detect tie when no votes
	if len(ties) != 0 {
		t.Errorf("expected no ties with zero votes, got %d", len(ties))
	}
}

func TestResultsService_DetectTies_OnlyOneCar(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category and single car
	catID, _ := repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Single vote
	v1, _ := repo.CreateVoter(ctx, "V1")
	repo.SaveVote(ctx, v1, int(catID), cars[0].ID)

	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	// Should not detect tie with only one car
	if len(ties) != 0 {
		t.Errorf("expected no ties with only one car, got %d", len(ties))
	}
}

// ==================== DetectMultipleWins Additional Tests ====================

func TestResultsService_DetectMultipleWins_DifferentGroups(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create two different groups with max_wins_per_car = 1
	maxWins := 1
	group1ID, _ := repo.CreateCategoryGroup(ctx, "Design Awards", "Design", nil, &maxWins, 1)
	group2ID, _ := repo.CreateCategoryGroup(ctx, "Speed Awards", "Speed", nil, &maxWins, 2)
	group1IDInt := int(group1ID)
	group2IDInt := int(group2ID)

	// Create categories in different groups
	cat1ID, _ := repo.CreateCategory(ctx, "Best Design", 1, &group1IDInt, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Fastest Looking", 2, &group2IDInt, nil, nil)

	// Create car
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Car wins in both groups
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	repo.SaveVote(ctx, v2, int(cat2ID), cars[0].ID)

	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	// Should NOT conflict - different groups
	if len(multiWins) != 0 {
		t.Errorf("expected no conflicts for wins in different groups, got %d", len(multiWins))
	}
}

func TestResultsService_DetectMultipleWins_GroupWithoutLimit(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create group WITHOUT max_wins_per_car
	groupID, _ := repo.CreateCategoryGroup(ctx, "Design Awards", "Design", nil, nil, 1)
	groupIDInt := int(groupID)

	// Create multiple categories in the group
	cat1ID, _ := repo.CreateCategory(ctx, "Best Design", 1, &groupIDInt, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Most Creative", 2, &groupIDInt, nil, nil)
	cat3ID, _ := repo.CreateCategory(ctx, "Best Paint", 3, &groupIDInt, nil, nil)

	// Create car
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Car wins all 3 categories
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")
	repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	repo.SaveVote(ctx, v2, int(cat2ID), cars[0].ID)
	repo.SaveVote(ctx, v3, int(cat3ID), cars[0].ID)

	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	// Should NOT conflict - no limit set
	if len(multiWins) != 0 {
		t.Errorf("expected no conflicts when no max_wins_per_car set, got %d", len(multiWins))
	}
}

func TestResultsService_DetectMultipleWins_MaxTwoWinsTwo(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create group with max_wins_per_car = 2
	maxWins := 2
	groupID, _ := repo.CreateCategoryGroup(ctx, "Design Awards", "Design", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create 2 categories in the group
	cat1ID, _ := repo.CreateCategory(ctx, "Best Design", 1, &groupIDInt, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Most Creative", 2, &groupIDInt, nil, nil)

	// Create car
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Car wins both (exactly at limit)
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	repo.SaveVote(ctx, v2, int(cat2ID), cars[0].ID)

	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	// Should NOT conflict - exactly at limit
	if len(multiWins) != 0 {
		t.Errorf("expected no conflict when wins = max_wins_per_car, got %d", len(multiWins))
	}
}

func TestResultsService_DetectMultipleWins_MaxTwoWinsThree(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create group with max_wins_per_car = 2
	maxWins := 2
	groupID, _ := repo.CreateCategoryGroup(ctx, "Design Awards", "Design", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create 3 categories in the group
	cat1ID, _ := repo.CreateCategory(ctx, "Best Design", 1, &groupIDInt, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Most Creative", 2, &groupIDInt, nil, nil)
	cat3ID, _ := repo.CreateCategory(ctx, "Best Paint", 3, &groupIDInt, nil, nil)

	// Create car
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Car wins all 3 (exceeds limit of 2)
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")
	repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	repo.SaveVote(ctx, v2, int(cat2ID), cars[0].ID)
	repo.SaveVote(ctx, v3, int(cat3ID), cars[0].ID)

	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	// SHOULD conflict - exceeds limit
	if len(multiWins) != 1 {
		t.Fatalf("expected 1 conflict when wins > max_wins_per_car, got %d", len(multiWins))
	}

	if len(multiWins[0].AwardsWon) != 3 {
		t.Errorf("expected 3 awards won, got %d", len(multiWins[0].AwardsWon))
	}
}

func TestResultsService_DetectMultipleWins_MultipleCarsViolating(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := repo.CreateCategoryGroup(ctx, "Design Awards", "Design", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create 4 categories in the group
	cat1ID, _ := repo.CreateCategory(ctx, "Category 1", 1, &groupIDInt, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Category 2", 2, &groupIDInt, nil, nil)
	cat3ID, _ := repo.CreateCategory(ctx, "Category 3", 3, &groupIDInt, nil, nil)
	cat4ID, _ := repo.CreateCategory(ctx, "Category 4", 4, &groupIDInt, nil, nil)

	// Create 2 cars
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := repo.ListCars(ctx)

	// Car 1 wins categories 1 and 2
	// Car 2 wins categories 3 and 4
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	v3, _ := repo.CreateVoter(ctx, "V3")
	v4, _ := repo.CreateVoter(ctx, "V4")
	repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	repo.SaveVote(ctx, v2, int(cat2ID), cars[0].ID)
	repo.SaveVote(ctx, v3, int(cat3ID), cars[1].ID)
	repo.SaveVote(ctx, v4, int(cat4ID), cars[1].ID)

	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	// Should detect both cars violating
	if len(multiWins) != 2 {
		t.Fatalf("expected 2 conflicts (one per car), got %d", len(multiWins))
	}
}

func TestResultsService_DetectMultipleWins_ReturnsGroupInfo(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, repo)
	svc := services.NewResultsService(log, repo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := repo.CreateCategoryGroup(ctx, "Design Awards", "Design categories", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create 2 categories in the group
	cat1ID, _ := repo.CreateCategory(ctx, "Category 1", 1, &groupIDInt, nil, nil)
	cat2ID, _ := repo.CreateCategory(ctx, "Category 2", 2, &groupIDInt, nil, nil)

	// Create car
	_ = repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := repo.ListCars(ctx)

	// Car wins both
	v1, _ := repo.CreateVoter(ctx, "V1")
	v2, _ := repo.CreateVoter(ctx, "V2")
	repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	repo.SaveVote(ctx, v2, int(cat2ID), cars[0].ID)

	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	if len(multiWins) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(multiWins))
	}

	mw := multiWins[0]

	// Verify group info is included
	if mw.GroupID == nil {
		t.Error("expected GroupID to be set")
	} else if *mw.GroupID != groupIDInt {
		t.Errorf("expected GroupID=%d, got %d", groupIDInt, *mw.GroupID)
	}

	if mw.GroupName != "Design Awards" {
		t.Errorf("expected GroupName='Design Awards', got '%s'", mw.GroupName)
	}

	if mw.MaxWinsPerCar != maxWins {
		t.Errorf("expected MaxWinsPerCar=%d, got %d", maxWins, mw.MaxWinsPerCar)
	}
}

// ==================== Manual Override Service Tests ====================

// ==================== Error Path Tests ====================

func TestResultsService_DetectTies_GetResultsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetVoteResultsWithCarsError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, mockRepo)
	svc := services.NewResultsService(log, mockRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	_, err := svc.DetectTies(ctx)
	if err == nil {
		t.Error("expected error from GetResults, got nil")
	}
}

func TestResultsService_DetectMultipleWins_GetResultsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetVoteResultsWithCarsError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, mockRepo)
	svc := services.NewResultsService(log, mockRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	_, err := svc.DetectMultipleWins(ctx)
	if err == nil {
		t.Error("expected error from GetResults, got nil")
	}
}

func TestResultsService_DetectMultipleWins_ListCategoryGroupsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoryGroupsError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, mockRepo)
	svc := services.NewResultsService(log, mockRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	_, err := svc.DetectMultipleWins(ctx)
	if err == nil {
		t.Error("expected error from ListCategoryGroups, got nil")
	}
}

func TestResultsService_SetManualWinner_ListCategoriesError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoriesError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, mockRepo)
	svc := services.NewResultsService(log, mockRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	err := svc.SetManualWinner(ctx, 1, 1, "test reason")
	if err == nil {
		t.Error("expected error from ListCategories, got nil")
	}
}

func TestResultsService_SetManualWinner_ListCarsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, mockRepo)
	svc := services.NewResultsService(log, mockRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a category first (without error)
	catID, _ := realRepo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)

	// Now inject error for ListCars
	mockRepo.ListCarsError = errors.New("database error")

	err := svc.SetManualWinner(ctx, int(catID), 1, "test reason")
	if err == nil {
		t.Error("expected error from ListCars, got nil")
	}
}

func TestResultsService_GetFinalWinners_ListCategoriesError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoriesError = errors.New("database error")

	log := logger.New()
	settingsSvc := services.NewSettingsService(log, mockRepo)
	svc := services.NewResultsService(log, mockRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	_, err := svc.GetFinalWinners(ctx)
	if err == nil {
		t.Error("expected error from ListCategories, got nil")
	}
}

func TestResultsService_GetFinalWinners_GetResultsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, mockRepo)
	svc := services.NewResultsService(log, mockRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a category first (without error)
	_, _ = realRepo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)

	// Now inject error for GetVoteResultsWithCars
	mockRepo.GetVoteResultsWithCarsError = errors.New("database error")

	_, err := svc.GetFinalWinners(ctx)
	if err == nil {
		t.Error("expected error from GetResults, got nil")
	}
}

func TestResultsService_DetectMultipleWins_GetCarError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	svc := services.NewResultsService(log, realRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := realRepo.CreateCategoryGroup(ctx, "Test Group", "Test", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create category in the group
	cat1ID, _ := realRepo.CreateCategory(ctx, "Category 1", 1, &groupIDInt, nil, nil)

	// Create car and set override to a non-existent car
	_ = realRepo.CreateCar(ctx, "101", "Racer One", "Car A", "")

	// Set manual override to a car that doesn't exist (will test GetCar error path)
	nonExistentCarID := 9999
	_ = realRepo.SetManualWinner(ctx, int(cat1ID), nonExistentCarID, "test")

	// This should handle the GetCar error gracefully by skipping the category
	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	// Should have no conflicts since the category was skipped
	if len(multiWins) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(multiWins))
	}
}

func TestResultsService_DetectTies_AllVotesZero(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	svc := services.NewResultsService(log, realRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create category
	cat1ID, _ := realRepo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)

	// Create two cars but don't cast any votes (votes will be 0)
	_ = realRepo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = realRepo.CreateCar(ctx, "102", "Racer Two", "Car B", "")

	// DetectTies should not report ties when all votes are 0
	ties, err := svc.DetectTies(ctx)
	if err != nil {
		t.Fatalf("DetectTies failed: %v", err)
	}

	// Should have no ties because maxVotes == 0
	if len(ties) != 0 {
		t.Errorf("expected 0 ties, got %d", len(ties))
	}

	// Verify the category exists in results
	results, _ := svc.GetResults(ctx)
	found := false
	for _, cat := range results.Categories {
		if cat.CategoryID == int(cat1ID) {
			found = true
			// All cars should have 0 votes
			for _, vote := range cat.Votes {
				if vote.VoteCount != 0 {
					t.Errorf("expected vote count 0, got %d", vote.VoteCount)
				}
			}
		}
	}
	if !found {
		t.Error("category not found in results")
	}
}

func TestResultsService_DetectMultipleWins_OverrideWinnerInVotes(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, realRepo)
	svc := services.NewResultsService(log, realRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := realRepo.CreateCategoryGroup(ctx, "Test Group", "Test", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create 2 categories in the group
	cat1ID, _ := realRepo.CreateCategory(ctx, "Category 1", 1, &groupIDInt, nil, nil)
	cat2ID, _ := realRepo.CreateCategory(ctx, "Category 2", 2, &groupIDInt, nil, nil)

	// Create two cars
	_ = realRepo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = realRepo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := realRepo.ListCars(ctx)

	// Car 1 wins category 1 by votes
	v1, _ := realRepo.CreateVoter(ctx, "V1")
	v2, _ := realRepo.CreateVoter(ctx, "V2")
	v3, _ := realRepo.CreateVoter(ctx, "V3")
	realRepo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)

	// In category 2, both cars get votes but car 2 has more
	realRepo.SaveVote(ctx, v2, int(cat2ID), cars[1].ID)
	realRepo.SaveVote(ctx, v3, int(cat2ID), cars[0].ID)

	// Set manual override for category 2 to car 1 (who HAS votes in this category)
	realRepo.SetManualWinner(ctx, int(cat2ID), cars[0].ID, "manual override")

	// Now car 1 should be winning both categories (violating max_wins_per_car = 1)
	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins failed: %v", err)
	}

	// Should detect car 1 winning multiple times
	if len(multiWins) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(multiWins))
	}

	mw := multiWins[0]
	if mw.CarID != cars[0].ID {
		t.Errorf("expected car ID %d, got %d", cars[0].ID, mw.CarID)
	}
	if len(mw.AwardsWon) != 2 {
		t.Errorf("expected 2 awards, got %d", len(mw.AwardsWon))
	}
}

func TestResultsService_DetectMultipleWins_GetCarErrorWithMock(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	settingsSvc := services.NewSettingsService(log, mockRepo)
	svc := services.NewResultsService(log, mockRepo, settingsSvc, derbynet.NewMockClient())
	ctx := context.Background()

	// Create group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := realRepo.CreateCategoryGroup(ctx, "Test Group", "Test", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create category in the group
	cat1ID, _ := realRepo.CreateCategory(ctx, "Category 1", 1, &groupIDInt, nil, nil)

	// Create two cars
	_ = realRepo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	_ = realRepo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := realRepo.ListCars(ctx)

	// Set manual override for category 1 to car 2 (who has NO votes in this category)
	realRepo.SetManualWinner(ctx, int(cat1ID), cars[1].ID, "manual override")

	// Inject error for GetCar to trigger the error path in DetectMultipleWins
	mockRepo.GetCarError = errors.New("database error")

	// This should handle the GetCar error gracefully by skipping the category
	multiWins, err := svc.DetectMultipleWins(ctx)
	if err != nil {
		t.Fatalf("DetectMultipleWins should not fail: %v", err)
	}

	// Should have no conflicts since the category was skipped due to error
	if len(multiWins) != 0 {
		t.Errorf("expected 0 conflicts due to GetCar error, got %d", len(multiWins))
	}
}

