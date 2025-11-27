package services_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/errors"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/internal/repository/mock"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/testutil"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

func TestCarService_ListCars_Empty(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// List cars from empty database
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 0 {
		t.Errorf("expected 0 cars in empty database, got %d", len(cars))
	}
}

func TestCarService_ListCars_AfterSeeding(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Seed mock cars first
	count, err := svc.SeedMockCars(ctx)
	if err != nil {
		t.Fatalf("SeedMockCars failed: %v", err)
	}
	if count == 0 {
		t.Fatal("expected some cars to be seeded")
	}

	// Now list cars
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != count {
		t.Errorf("expected %d cars after seeding, got %d", count, len(cars))
	}

	// Verify car data is populated
	for _, car := range cars {
		if car.CarNumber == "" {
			t.Error("car has empty CarNumber")
		}
		if car.RacerName == "" {
			t.Error("car has empty RacerName")
		}
		if car.CarName == "" {
			t.Error("car has empty CarName")
		}
	}
}

func TestCarService_SeedMockCars_AddsFirstTime(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// First seed should add cars
	count, err := svc.SeedMockCars(ctx)
	if err != nil {
		t.Fatalf("SeedMockCars failed: %v", err)
	}
	if count == 0 {
		t.Error("expected some cars to be seeded on first call")
	}

	// Verify cars were actually added
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != count {
		t.Errorf("expected %d cars in database, got %d", count, len(cars))
	}
}

func TestCarService_SeedMockCars_NoDuplicates(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// First seed
	count1, err := svc.SeedMockCars(ctx)
	if err != nil {
		t.Fatalf("SeedMockCars first call failed: %v", err)
	}
	if count1 == 0 {
		t.Fatal("expected some cars to be seeded on first call")
	}

	// Second seed should not add duplicates
	count2, err := svc.SeedMockCars(ctx)
	if err != nil {
		t.Fatalf("SeedMockCars second call failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 new cars on second seed, got %d", count2)
	}

	// Verify total count remains the same
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != count1 {
		t.Errorf("expected %d total cars (no duplicates), got %d", count1, len(cars))
	}
}

func TestCarService_SyncFromDerbyNet_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://test-derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.TotalRacers != 10 {
		t.Errorf("expected 10 racers, got %d", result.TotalRacers)
	}
	if result.CarsCreated != 10 {
		t.Errorf("expected 10 cars created, got %d", result.CarsCreated)
	}

	// Verify cars were actually created
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 10 {
		t.Errorf("expected 10 cars in database, got %d", len(cars))
	}
}

func TestCarService_SyncFromDerbyNet_CreatesVoters(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://test-derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}
	if result.VotersCreated != 10 {
		t.Errorf("expected 10 voters created, got %d", result.VotersCreated)
	}

	// Verify voters exist via repo
	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 10 {
		t.Errorf("expected 10 voters in database, got %d", len(voters))
	}
}

func TestCarService_SyncFromDerbyNet_UpdatesExisting(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// First sync
	result1, err := svc.SyncFromDerbyNet(ctx, "http://test-derbynet.local")
	if err != nil {
		t.Fatalf("first SyncFromDerbyNet failed: %v", err)
	}
	if result1.CarsCreated != 10 {
		t.Fatalf("expected 10 cars created, got %d", result1.CarsCreated)
	}

	// Second sync should update, not create
	result2, err := svc.SyncFromDerbyNet(ctx, "http://test-derbynet.local")
	if err != nil {
		t.Fatalf("second SyncFromDerbyNet failed: %v", err)
	}
	if result2.CarsCreated != 0 {
		t.Errorf("expected 0 cars created on resync, got %d", result2.CarsCreated)
	}
	if result2.CarsUpdated != 10 {
		t.Errorf("expected 10 cars updated on resync, got %d", result2.CarsUpdated)
	}
}

func TestCarService_SyncFromDerbyNet_FetchError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient(
		derbynet.WithFetchError(stderrors.New("connection refused")),
	)
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://test-derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet returned error: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("expected status 'error', got %q", result.Status)
	}
	if result.Message == "" {
		t.Error("expected error message")
	}
}

func TestCarService_SyncFromDerbyNet_CustomRacers(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	customRacers := []derbynet.Racer{
		{RacerID: 1, FirstName: "Custom", LastName: "Racer", CarNumber: 999, CarName: "Custom Car"},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(customRacers))
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://test-derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}
	if result.TotalRacers != 1 {
		t.Errorf("expected 1 racer, got %d", result.TotalRacers)
	}

	cars, _ := svc.ListCars(ctx)
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	if cars[0].CarNumber != "999" {
		t.Errorf("expected car number '999', got %q", cars[0].CarNumber)
	}
	if cars[0].RacerName != "Custom Racer" {
		t.Errorf("expected racer name 'Custom Racer', got %q", cars[0].RacerName)
	}
}

func TestCarService_SyncFromDerbyNet_GeneratedRacers(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	generatedRacers := derbynet.GenerateMockRacers(50)
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(generatedRacers))
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://test-derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}
	if result.TotalRacers != 50 {
		t.Errorf("expected 50 racers, got %d", result.TotalRacers)
	}
	if result.CarsCreated != 50 {
		t.Errorf("expected 50 cars created, got %d", result.CarsCreated)
	}
}

func TestSyncResult_JSONIncludesZeroValues(t *testing.T) {
	// This test verifies the bug fix: SyncResult JSON must include zero values
	// for count fields, not omit them (which causes "undefined" in JavaScript)
	result := services.SyncResult{
		Status:        "success",
		CarsCreated:   0,
		CarsUpdated:   5,
		VotersCreated: 0,
		VotersUpdated: 5,
		TotalCars:     5,
		TotalVoters:   5,
		TotalRacers:   5,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal SyncResult: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify zero values are present in JSON (not omitted)
	requiredFields := []string{
		`"cars_created":0`,
		`"voters_created":0`,
		`"cars_updated":5`,
		`"voters_updated":5`,
	}

	for _, field := range requiredFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON missing expected field %s\nGot: %s", field, jsonStr)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCarService_SyncFromDerbyNet_WithPhotos(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	// Create racers with car photos
	racersWithPhotos := []derbynet.Racer{
		{RacerID: 1, FirstName: "Photo", LastName: "Racer", CarNumber: 101, CarName: "Photo Car", CarPhoto: "/photos/car1.jpg"},
		{RacerID: 2, FirstName: "NoPhoto", LastName: "Racer", CarNumber: 102, CarName: "No Photo Car", CarPhoto: ""},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(racersWithPhotos))
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local/test")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}
	if result.TotalRacers != 2 {
		t.Errorf("expected 2 racers, got %d", result.TotalRacers)
	}

	// Verify cars were created
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 2 {
		t.Fatalf("expected 2 cars, got %d", len(cars))
	}

	// Find the car with photo and verify URL
	for _, car := range cars {
		if car.CarNumber == "101" {
			if car.PhotoURL == "" {
				t.Error("expected photo URL for car 101")
			}
			// Photo URL should be combined with base URL
			if car.PhotoURL != "http://derbynet.local/test/photos/car1.jpg" {
				t.Errorf("expected full photo URL, got %q", car.PhotoURL)
			}
		}
		if car.CarNumber == "102" {
			if car.PhotoURL != "" {
				t.Errorf("expected empty photo URL for car 102, got %q", car.PhotoURL)
			}
		}
	}
}

func TestCarService_SyncFromDerbyNet_WithTrailingSlash(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	// Create racers with car photos with leading slash
	racersWithPhotos := []derbynet.Racer{
		{RacerID: 1, FirstName: "Test", LastName: "Racer", CarNumber: 201, CarName: "Test Car", CarPhoto: "/photos/car.jpg"},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(racersWithPhotos))
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// URL with trailing slash
	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local/test/")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}
	if result.TotalRacers != 1 {
		t.Errorf("expected 1 racer, got %d", result.TotalRacers)
	}

	cars, _ := svc.ListCars(ctx)
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}

	// Should not have double slashes
	if cars[0].PhotoURL == "" {
		t.Error("expected photo URL")
	}
	// Verify no double slashes in URL (trailing slash should be trimmed)
	if cars[0].PhotoURL != "http://derbynet.local/test/photos/car.jpg" {
		t.Errorf("unexpected photo URL format: %q", cars[0].PhotoURL)
	}
}

func TestCarService_GetCar_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create a car first
	err := svc.CreateCar(ctx, "100", "Test Racer", "Test Car", "http://photo.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get the car list to find the ID
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}

	// Get the car by ID
	car, err := svc.GetCar(ctx, cars[0].ID)
	if err != nil {
		t.Fatalf("GetCar failed: %v", err)
	}
	if car == nil {
		t.Fatal("expected car to be returned")
	}
	if car.CarNumber != "100" {
		t.Errorf("expected car number '100', got %q", car.CarNumber)
	}
	if car.RacerName != "Test Racer" {
		t.Errorf("expected racer name 'Test Racer', got %q", car.RacerName)
	}
	if car.CarName != "Test Car" {
		t.Errorf("expected car name 'Test Car', got %q", car.CarName)
	}
	if car.PhotoURL != "http://photo.jpg" {
		t.Errorf("expected photo URL 'http://photo.jpg', got %q", car.PhotoURL)
	}
}

func TestCarService_GetCar_NotFound(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Try to get a non-existent car
	car, err := svc.GetCar(ctx, 99999)
	var appErr *errors.Error
	if !stderrors.As(err, &appErr) || appErr.Kind != errors.ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-existent car, got: %v", err)
	}
	if car != nil {
		t.Error("expected nil car for non-existent ID")
	}
}

func TestCarService_CreateCar_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create a car
	err := svc.CreateCar(ctx, "200", "New Racer", "Speed Demon", "http://example.com/photo.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Verify car was created
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}

	car := cars[0]
	if car.CarNumber != "200" {
		t.Errorf("expected car number '200', got %q", car.CarNumber)
	}
	if car.RacerName != "New Racer" {
		t.Errorf("expected racer name 'New Racer', got %q", car.RacerName)
	}
	if car.CarName != "Speed Demon" {
		t.Errorf("expected car name 'Speed Demon', got %q", car.CarName)
	}
	if car.PhotoURL != "http://example.com/photo.jpg" {
		t.Errorf("expected photo URL, got %q", car.PhotoURL)
	}
}

func TestCarService_CreateCar_EmptyPhotoURL(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create a car without photo URL
	err := svc.CreateCar(ctx, "201", "No Photo Racer", "Plain Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Verify car was created with empty photo
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	if cars[0].PhotoURL != "" {
		t.Errorf("expected empty photo URL, got %q", cars[0].PhotoURL)
	}
}

func TestCarService_UpdateCar_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create a car first
	err := svc.CreateCar(ctx, "300", "Original Racer", "Original Car", "http://original.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get the car ID
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	// Update the car
	err = svc.UpdateCar(ctx, carID, "301", "Updated Racer", "Updated Car", "http://updated.jpg", "")
	if err != nil {
		t.Fatalf("UpdateCar failed: %v", err)
	}

	// Verify the update
	car, err := svc.GetCar(ctx, carID)
	if err != nil {
		t.Fatalf("GetCar failed: %v", err)
	}
	if car.CarNumber != "301" {
		t.Errorf("expected car number '301', got %q", car.CarNumber)
	}
	if car.RacerName != "Updated Racer" {
		t.Errorf("expected racer name 'Updated Racer', got %q", car.RacerName)
	}
	if car.CarName != "Updated Car" {
		t.Errorf("expected car name 'Updated Car', got %q", car.CarName)
	}
	if car.PhotoURL != "http://updated.jpg" {
		t.Errorf("expected photo URL 'http://updated.jpg', got %q", car.PhotoURL)
	}
}

func TestCarService_UpdateCar_PartialUpdate(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create a car first
	err := svc.CreateCar(ctx, "400", "Keep Racer", "Keep Car", "http://keep.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get the car ID
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	// Update only the car name
	err = svc.UpdateCar(ctx, carID, "400", "Keep Racer", "New Car Name", "http://keep.jpg", "")
	if err != nil {
		t.Fatalf("UpdateCar failed: %v", err)
	}

	// Verify the update only changed car name
	car, err := svc.GetCar(ctx, carID)
	if err != nil {
		t.Fatalf("GetCar failed: %v", err)
	}
	if car.CarNumber != "400" {
		t.Errorf("car number should not change, got %q", car.CarNumber)
	}
	if car.RacerName != "Keep Racer" {
		t.Errorf("racer name should not change, got %q", car.RacerName)
	}
	if car.CarName != "New Car Name" {
		t.Errorf("expected car name 'New Car Name', got %q", car.CarName)
	}
}

func TestCarService_DeleteCar_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create a car first
	err := svc.CreateCar(ctx, "500", "Delete Me", "Goodbye Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get the car ID
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	carID := cars[0].ID

	// Delete the car
	err = svc.DeleteCar(ctx, carID)
	if err != nil {
		t.Fatalf("DeleteCar failed: %v", err)
	}

	// Verify car is no longer in list (soft deleted)
	cars, err = svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 0 {
		t.Errorf("expected 0 cars after delete, got %d", len(cars))
	}
}

func TestCarService_DeleteCar_NonExistent(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Try to delete a non-existent car (should not error - idempotent)
	err := svc.DeleteCar(ctx, 99999)
	if err != nil {
		t.Errorf("DeleteCar on non-existent car should not error: %v", err)
	}
}

func TestCarService_ListEligibleCars_AllEligible(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create cars (eligible by default)
	err := svc.CreateCar(ctx, "600", "Racer 1", "Car 1", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}
	err = svc.CreateCar(ctx, "601", "Racer 2", "Car 2", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// List eligible cars
	cars, err := svc.ListEligibleCars(ctx)
	if err != nil {
		t.Fatalf("ListEligibleCars failed: %v", err)
	}
	if len(cars) != 2 {
		t.Errorf("expected 2 eligible cars, got %d", len(cars))
	}
}

func TestCarService_ListEligibleCars_SomeIneligible(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create cars
	err := svc.CreateCar(ctx, "700", "Eligible Racer", "Eligible Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}
	err = svc.CreateCar(ctx, "701", "Ineligible Racer", "Ineligible Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get car IDs
	allCars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}

	// Find the ineligible car ID
	var ineligibleID int
	for _, car := range allCars {
		if car.CarNumber == "701" {
			ineligibleID = car.ID
			break
		}
	}

	// Set one car as ineligible
	err = svc.SetCarEligibility(ctx, ineligibleID, false)
	if err != nil {
		t.Fatalf("SetCarEligibility failed: %v", err)
	}

	// List eligible cars
	eligibleCars, err := svc.ListEligibleCars(ctx)
	if err != nil {
		t.Fatalf("ListEligibleCars failed: %v", err)
	}
	if len(eligibleCars) != 1 {
		t.Errorf("expected 1 eligible car, got %d", len(eligibleCars))
	}
	if eligibleCars[0].CarNumber != "700" {
		t.Errorf("expected eligible car '700', got %q", eligibleCars[0].CarNumber)
	}
}

func TestCarService_ListEligibleCars_Empty(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// List eligible cars from empty database
	cars, err := svc.ListEligibleCars(ctx)
	if err != nil {
		t.Fatalf("ListEligibleCars failed: %v", err)
	}
	if len(cars) != 0 {
		t.Errorf("expected 0 eligible cars in empty database, got %d", len(cars))
	}
}

func TestCarService_SetCarEligibility_Enable(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create a car
	err := svc.CreateCar(ctx, "800", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Get car ID
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	carID := cars[0].ID

	// Disable eligibility
	err = svc.SetCarEligibility(ctx, carID, false)
	if err != nil {
		t.Fatalf("SetCarEligibility(false) failed: %v", err)
	}

	// Verify disabled
	eligibleCars, _ := svc.ListEligibleCars(ctx)
	if len(eligibleCars) != 0 {
		t.Errorf("expected 0 eligible cars after disabling, got %d", len(eligibleCars))
	}

	// Re-enable eligibility
	err = svc.SetCarEligibility(ctx, carID, true)
	if err != nil {
		t.Fatalf("SetCarEligibility(true) failed: %v", err)
	}

	// Verify enabled
	eligibleCars, _ = svc.ListEligibleCars(ctx)
	if len(eligibleCars) != 1 {
		t.Errorf("expected 1 eligible car after re-enabling, got %d", len(eligibleCars))
	}
}

func TestCarService_SetCarEligibility_MultipleCars(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create multiple cars
	for i := 0; i < 5; i++ {
		err := svc.CreateCar(ctx, "90"+string(rune('0'+i)), "Racer "+string(rune('A'+i)), "Car "+string(rune('A'+i)), "")
		if err != nil {
			t.Fatalf("CreateCar failed: %v", err)
		}
	}

	// Get all cars
	cars, err := svc.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 5 {
		t.Fatalf("expected 5 cars, got %d", len(cars))
	}

	// Disable first 3 cars
	for i := 0; i < 3; i++ {
		err = svc.SetCarEligibility(ctx, cars[i].ID, false)
		if err != nil {
			t.Fatalf("SetCarEligibility failed: %v", err)
		}
	}

	// Verify only 2 are eligible
	eligibleCars, err := svc.ListEligibleCars(ctx)
	if err != nil {
		t.Fatalf("ListEligibleCars failed: %v", err)
	}
	if len(eligibleCars) != 2 {
		t.Errorf("expected 2 eligible cars, got %d", len(eligibleCars))
	}
}

// ===== Error Path Tests using Mock Repository =====

func TestCarService_SeedMockCars_CarExistsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	mockClient := derbynet.NewMockClient()
	svc := services.NewCarService(log, mockRepo, mockClient)

	// Configure mock to fail on CarExists check
	mockRepo.CarExistsError = stderrors.New("database error checking car")

	count, err := svc.SeedMockCars(ctx)
	if err == nil {
		t.Fatal("expected error when CarExists fails, got nil")
	}
	if count != 0 {
		t.Errorf("expected 0 cars added when check fails, got %d", count)
	}
}

func TestCarService_SeedMockCars_CreateCarError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	mockClient := derbynet.NewMockClient()
	svc := services.NewCarService(log, mockRepo, mockClient)

	// Configure mock to fail on CreateCar
	mockRepo.CreateCarError = stderrors.New("database error creating car")

	count, err := svc.SeedMockCars(ctx)
	if err == nil {
		t.Fatal("expected error when CreateCar fails, got nil")
	}
	if count != 0 {
		t.Errorf("expected 0 cars added when creation fails, got %d", count)
	}
}

func TestCarService_SyncFromDerbyNet_GetCarByDerbyNetIDError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	customRacers := []derbynet.Racer{
		{RacerID: 1, FirstName: "Test", LastName: "Racer", CarNumber: 101, CarName: "Test Car"},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(customRacers))
	svc := services.NewCarService(log, mockRepo, mockClient)

	// Configure mock to fail on GetCarByDerbyNetID
	mockRepo.GetCarByDerbyNetIDError = stderrors.New("database error checking car")

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when GetCarByDerbyNetID fails, got nil")
	}
}

func TestCarService_SyncFromDerbyNet_UpsertCarError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	customRacers := []derbynet.Racer{
		{RacerID: 1, FirstName: "Test", LastName: "Racer", CarNumber: 101, CarName: "Test Car"},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(customRacers))
	svc := services.NewCarService(log, mockRepo, mockClient)

	// Configure mock to fail on UpsertCar
	mockRepo.UpsertCarError = stderrors.New("database error upserting car")

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when UpsertCar fails, got nil")
	}
}

func TestCarService_SyncFromDerbyNet_UpsertVoterForCarError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	customRacers := []derbynet.Racer{
		{RacerID: 1, FirstName: "Test", LastName: "Racer", CarNumber: 101, CarName: "Test Car"},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(customRacers))
	svc := services.NewCarService(log, mockRepo, mockClient)

	// Configure mock to fail on UpsertVoterForCar
	mockRepo.UpsertVoterForCarError = stderrors.New("database error upserting voter")

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when UpsertVoterForCar fails, got nil")
	}
}

func TestCarService_SyncFromDerbyNet_SetSettingError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	mockClient := derbynet.NewMockClient()
	svc := services.NewCarService(log, mockRepo, mockClient)

	// Configure mock to fail on SetSetting (saving DerbyNet URL)
	mockRepo.SetSettingError = stderrors.New("database error saving URL")

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when SetSetting fails, got nil")
	}
	if !contains(err.Error(), "failed to save DerbyNet URL") {
		t.Errorf("expected error about saving URL, got: %v", err)
	}
}

func TestCarService_SyncFromDerbyNet_GetCarByDerbyNetIDErrorAfterUpsert(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	customRacers := []derbynet.Racer{
		{RacerID: 1, FirstName: "Test", LastName: "Racer", CarNumber: 101, CarName: "Test Car"},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(customRacers))
	svc := services.NewCarService(log, mockRepo, mockClient)

	// We need to fail GetCarByDerbyNetID only on the second call (after UpsertCar succeeds)
	// First call checks if car exists, second call gets the ID for voter creation
	callCount := 0
	originalGetCarByDerbyNetID := mockRepo.FullRepository.GetCarByDerbyNetID

	mockRepo.FullRepository = &mockCarRepo{
		FullRepository: mockRepo.FullRepository,
		getCarByDerbyNetID: func(ctx context.Context, derbyNetID int) (int64, bool, error) {
			callCount++
			if callCount > 1 {
				return 0, false, stderrors.New("database error getting car ID")
			}
			return originalGetCarByDerbyNetID(ctx, derbyNetID)
		},
	}

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when GetCarByDerbyNetID fails on second call, got nil")
	}
}

func TestCarService_SyncFromDerbyNet_GetVoterByQRCodeError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	customRacers := []derbynet.Racer{
		{RacerID: 1, FirstName: "Test", LastName: "Racer", CarNumber: 101, CarName: "Test Car"},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithRacers(customRacers))
	svc := services.NewCarService(log, mockRepo, mockClient)

	// Configure mock to fail on GetVoterByQRCode
	mockRepo.GetVoterByQRCodeError = stderrors.New("database error checking voter")

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when GetVoterByQRCode fails, got nil")
	}
}

// mockCarRepo wraps a repository to inject specific errors for car operations
type mockCarRepo struct {
	repository.FullRepository
	getCarByDerbyNetID func(ctx context.Context, derbyNetID int) (int64, bool, error)
}

func (m *mockCarRepo) GetCarByDerbyNetID(ctx context.Context, derbyNetID int) (int64, bool, error) {
	if m.getCarByDerbyNetID != nil {
		return m.getCarByDerbyNetID(ctx, derbyNetID)
	}
	return m.FullRepository.GetCarByDerbyNetID(ctx, derbyNetID)
}

// ===== GetCarPhoto Tests =====

func TestCarService_GetCarPhoto_CarNotFound(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Try to get photo for non-existent car
	photo, err := svc.GetCarPhoto(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent car, got nil")
	}
	if photo != nil {
		t.Errorf("expected nil photo for non-existent car, got %+v", photo)
	}
}

func TestCarService_GetCarPhoto_CarWithoutPhotoURL(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create car without photo URL
	err := repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Try to get photo
	photo, err := svc.GetCarPhoto(ctx, carID)
	if err == nil {
		t.Fatal("expected error for car without photo URL, got nil")
	}
	if photo != nil {
		t.Errorf("expected nil photo for car without photo URL, got %+v", photo)
	}
}

func TestCarService_GetCarPhoto_HTTPFetchFailure(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create car with invalid photo URL (will fail to fetch)
	err := repo.CreateCar(ctx, "102", "Test Racer", "Test Car", "http://invalid-host-that-does-not-exist.local/photo.jpg")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Try to get photo
	photo, err := svc.GetCarPhoto(ctx, carID)
	if err == nil {
		t.Fatal("expected error for HTTP fetch failure, got nil")
	}
	if photo != nil {
		t.Errorf("expected nil photo for HTTP fetch failure, got %+v", photo)
	}
}

func TestCarService_GetCarPhoto_HTTPNon200Status(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create test server that returns 404
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	// Create car with photo URL pointing to test server
	err := repo.CreateCar(ctx, "103", "Test Racer", "Test Car", testServer.URL+"/photo.jpg")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Try to get photo
	photo, err := svc.GetCarPhoto(ctx, carID)
	if err == nil {
		t.Fatal("expected error for HTTP 404 status, got nil")
	}
	if photo != nil {
		t.Errorf("expected nil photo for HTTP 404 status, got %+v", photo)
	}
}

func TestCarService_GetCarPhoto_SuccessWithContentType(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create test server that serves image with content type
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-png-data"))
	}))
	defer testServer.Close()

	// Create car with photo URL pointing to test server
	err := repo.CreateCar(ctx, "104", "Test Racer", "Test Car", testServer.URL+"/photo.png")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Get photo
	photo, err := svc.GetCarPhoto(ctx, carID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if photo == nil {
		t.Fatal("expected photo data, got nil")
	}

	// Verify content type is forwarded
	if photo.ContentType != "image/png" {
		t.Errorf("expected ContentType image/png, got %s", photo.ContentType)
	}

	// Verify data is correct
	if string(photo.Data) != "fake-png-data" {
		t.Errorf("expected Data 'fake-png-data', got %s", string(photo.Data))
	}
}

func TestCarService_GetCarPhoto_SuccessWithEmptyContentType(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create test server that serves image without content type (should default to image/jpeg)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set Content-Type to empty string
		w.Header().Set("Content-Type", "")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-jpeg-data"))
	}))
	defer testServer.Close()

	// Create car with photo URL pointing to test server
	err := repo.CreateCar(ctx, "105", "Test Racer", "Test Car", testServer.URL+"/photo.jpg")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Get photo
	photo, err := svc.GetCarPhoto(ctx, carID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if photo == nil {
		t.Fatal("expected photo data, got nil")
	}

	// Verify content type defaults to image/jpeg
	if photo.ContentType != "image/jpeg" {
		t.Errorf("expected ContentType image/jpeg (default), got %s", photo.ContentType)
	}

	// Verify data is correct
	if string(photo.Data) != "fake-jpeg-data" {
		t.Errorf("expected Data 'fake-jpeg-data', got %s", string(photo.Data))
	}
}

func TestCarService_GetCarPhoto_ReadBodyError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create test server that hijacks connection to cause read error
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the hijacker
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter doesn't support hijacking")
		}

		// Hijack the connection
		conn, bufrw, err := hijacker.Hijack()
		if err != nil {
			t.Fatalf("Failed to hijack: %v", err)
		}
		defer conn.Close()

		// Send HTTP response headers manually with Content-Length
		response := "HTTP/1.1 200 OK\r\n" +
			"Content-Type: image/png\r\n" +
			"Content-Length: 100\r\n" + // Claim 100 bytes
			"\r\n" +
			"partial" // Only send 7 bytes, then close connection

		bufrw.WriteString(response)
		bufrw.Flush()
		// Connection will be closed when handler exits, causing EOF before Content-Length bytes are read
	}))
	defer testServer.Close()

	// Create car with photo URL pointing to test server
	err := repo.CreateCar(ctx, "106", "Test Racer", "Test Car", testServer.URL+"/photo.png")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}

	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Try to get photo - io.ReadAll should fail due to unexpected EOF
	photo, err := svc.GetCarPhoto(ctx, carID)
	if err == nil {
		t.Fatal("expected error for incomplete read, got nil")
	}
	if photo != nil {
		t.Errorf("expected nil photo for incomplete read, got %+v", photo)
	}
}

func TestCarService_CountVotesForCar(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCarService(log, repo, mockClient)
	ctx := context.Background()

	// Create a category
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Create a car
	err = repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

	// Initially should have 0 votes
	count, err := svc.CountVotesForCar(ctx, int(carID))
	if err != nil {
		t.Fatalf("CountVotesForCar failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 votes, got %d", count)
	}

	// Create a voter and add a vote
	voterID, err := repo.CreateVoter(ctx, "test-qr")
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}
	err = repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote: %v", err)
	}

	// Should now have 1 vote
	count, err = svc.CountVotesForCar(ctx, int(carID))
	if err != nil {
		t.Fatalf("CountVotesForCar failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 vote, got %d", count)
	}

	// Add another voter and vote
	voterID2, err := repo.CreateVoter(ctx, "test-qr-2")
	if err != nil {
		t.Fatalf("failed to create voter 2: %v", err)
	}
	err = repo.SaveVote(ctx, voterID2, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote 2: %v", err)
	}

	// Should now have 2 votes
	count, err = svc.CountVotesForCar(ctx, int(carID))
	if err != nil {
		t.Fatalf("CountVotesForCar failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 votes, got %d", count)
	}
}
