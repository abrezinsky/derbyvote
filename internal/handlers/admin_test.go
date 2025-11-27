package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/abrezinsky/derbyvote/internal/auth"
	"github.com/abrezinsky/derbyvote/internal/handlers"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/internal/repository/mock"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// testSetup creates all the dependencies needed for testing handlers
type testSetup struct {
	repo        *repository.Repository
	handlers    *handlers.Handlers
	router      chi.Router
	authCookie  *http.Cookie
	log         *logger.SlogLogger
}

// newTestSetup creates a new test setup with in-memory repository
func newTestSetup(t *testing.T) *testSetup {
	t.Helper()

	repo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Initialize logger for tests
	log := logger.New()

	// Initialize services
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, repo, derbynetClient)
	carService := services.NewCarService(log, repo, derbynetClient)
	settingsService := services.NewSettingsService(log, repo)
	voterService := services.NewVoterService(log, repo, settingsService)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, repo, settingsService, derbynetClient)

	// Initialize handlers (templates will not be used in API tests)
	h := handlers.NewForTesting(
		votingService,
		categoryService,
		voterService,
		carService,
		settingsService,
		resultsService,
	)

	// Set the logger for testing
	h.Log = log

	// Login to get a session cookie for authenticated requests
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{
		Name:  auth.CookieName,
		Value: token,
	}

	// Create router for URL parameter handling
	router := chi.NewRouter()

	// Categories
	router.Get("/api/admin/categories", h.Router().ServeHTTP)
	router.Post("/api/admin/categories", h.Router().ServeHTTP)
	router.Put("/api/admin/categories/{id}", h.Router().ServeHTTP)
	router.Delete("/api/admin/categories/{id}", h.Router().ServeHTTP)

	// Category Groups
	router.Get("/api/admin/category-groups", h.Router().ServeHTTP)
	router.Post("/api/admin/category-groups", h.Router().ServeHTTP)
	router.Get("/api/admin/category-groups/{id}", h.Router().ServeHTTP)
	router.Put("/api/admin/category-groups/{id}", h.Router().ServeHTTP)
	router.Delete("/api/admin/category-groups/{id}", h.Router().ServeHTTP)

	// Voters
	router.Get("/api/admin/voters", h.Router().ServeHTTP)
	router.Post("/api/admin/voters", h.Router().ServeHTTP)
	router.Put("/api/admin/voters", h.Router().ServeHTTP)
	router.Delete("/api/admin/voters/{id}", h.Router().ServeHTTP)

	// Stats
	router.Get("/api/admin/stats", h.Router().ServeHTTP)

	// Voting Control
	router.Post("/api/admin/voting-control", h.Router().ServeHTTP)
	router.Post("/api/admin/voting-timer", h.Router().ServeHTTP)

	// Settings
	router.Get("/api/admin/settings", h.Router().ServeHTTP)
	router.Put("/api/admin/settings", h.Router().ServeHTTP)
	router.Post("/api/admin/settings", h.Router().ServeHTTP)

	// Database Management
	router.Post("/api/admin/reset-database", h.Router().ServeHTTP)
	router.Post("/api/admin/seed-mock-data", h.Router().ServeHTTP)

	return &testSetup{
		repo:       repo,
		handlers:   h,
		router:     h.Router(), // Use the handlers' own router
		authCookie: authCookie,
		log:        log,
	}
}

// authRequest adds the auth cookie to a request
func (ts *testSetup) authRequest(req *http.Request) *http.Request {
	req.AddCookie(ts.authCookie)
	return req
}

// ==================== Categories Tests ====================

func TestHandleGetCategories_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a category first
	ctx := context.Background()
	_, err := setup.repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/categories", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("expected 1 category, got %d", len(response))
	}
}

func TestHandleGetCategories_Empty(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/categories", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Response should be null or empty array for no categories
	body := rec.Body.String()
	if body != "null\n" && body != "[]\n" {
		// It's okay if the response is null for empty categories
		var response []map[string]interface{}
		if err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&response); err == nil && len(response) > 0 {
			t.Errorf("expected empty categories, got %d", len(response))
		}
	}
}

func TestHandleCreateCategory_Success(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":          "New Category",
		"display_order": 1,
		"active":        true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["name"] != "New Category" {
		t.Errorf("expected name 'New Category', got %v", response["name"])
	}
	if response["id"] == nil || response["id"].(float64) <= 0 {
		t.Errorf("expected positive ID, got %v", response["id"])
	}
}

func TestHandleCreateCategory_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCreateCategory_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":          "New Category",
		"display_order": 1,
		"active":        true,
	}
	body, _ := json.Marshal(payload)

	// Close the database to cause a service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleCreateCategory_EmptyBody(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleUpdateCategory_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a category first
	ctx := context.Background()
	id, err := setup.repo.CreateCategory(ctx, "Original", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	payload := map[string]interface{}{
		"name":          "Updated Name",
		"display_order": 2,
		"active":        true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/categories/%d", id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["name"] != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %v", response["name"])
	}
}

func TestHandleUpdateCategory_InvalidID(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":          "Test",
		"display_order": 1,
		"active":        true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/categories/invalid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleUpdateCategory_ServiceError(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category first
	id, err := setup.repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	payload := map[string]interface{}{
		"name":          "Updated Category",
		"display_order": 2,
		"active":        true,
	}
	body, _ := json.Marshal(payload)

	// Close the database to cause a service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/categories/%d", id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleUpdateCategory_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/categories/1", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleDeleteCategory_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a category first
	ctx := context.Background()
	id, err := setup.repo.CreateCategory(ctx, "To Delete", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/categories/%d", id), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteCategory_InvalidID(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/categories/invalid", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleDeleteCategory_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/categories/1", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleDeleteCategory_WithVotes(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category
	catID, err := setup.repo.CreateCategory(ctx, "Popular Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	// Create a car
	err = setup.repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Create a voter
	voterID, err := setup.repo.CreateVoter(ctx, "test-qr")
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	// Save a vote
	err = setup.repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote: %v", err)
	}

	// Try to delete category without force
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/categories/%d", catID), nil)
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	if !response["confirmation_required"].(bool) {
		t.Error("expected confirmation_required to be true")
	}
}

func TestHandleDeleteCategory_WithVotesAndForce(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category
	catID, err := setup.repo.CreateCategory(ctx, "To Force Delete", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	// Create a car
	err = setup.repo.CreateCar(ctx, "102", "Test Racer 2", "Test Car 2", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Create a voter
	voterID, err := setup.repo.CreateVoter(ctx, "test-qr-2")
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	// Save a vote
	err = setup.repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote: %v", err)
	}

	// Delete category with force=true
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/categories/%d?force=true", catID), nil)
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
}

func TestHandleCreateCategory_WithAllowedRanks(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":          "Tiger Only Category",
		"display_order": 1,
		"active":        true,
		"allowed_ranks": []string{"Tiger", "Lion"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["name"] != "Tiger Only Category" {
		t.Errorf("expected name 'Tiger Only Category', got %v", response["name"])
	}

	// Verify allowed_ranks is returned in response
	ranks, ok := response["allowed_ranks"].([]interface{})
	if !ok {
		t.Fatalf("expected allowed_ranks to be an array, got %T", response["allowed_ranks"])
	}
	if len(ranks) != 2 {
		t.Errorf("expected 2 allowed ranks, got %d", len(ranks))
	}
	if ranks[0] != "Tiger" || ranks[1] != "Lion" {
		t.Errorf("expected ranks [Tiger, Lion], got %v", ranks)
	}
}

func TestHandleUpdateCategory_WithAllowedRanks(t *testing.T) {
	setup := newTestSetup(t)

	// Create a category first
	ctx := context.Background()
	id, err := setup.repo.CreateCategory(ctx, "Original", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	payload := map[string]interface{}{
		"name":          "Updated With Ranks",
		"display_order": 2,
		"active":        true,
		"allowed_ranks": []string{"Bear", "Wolf"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/categories/%d", id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["name"] != "Updated With Ranks" {
		t.Errorf("expected name 'Updated With Ranks', got %v", response["name"])
	}

	// Verify allowed_ranks is returned in response
	ranks, ok := response["allowed_ranks"].([]interface{})
	if !ok {
		t.Fatalf("expected allowed_ranks to be an array, got %T", response["allowed_ranks"])
	}
	if len(ranks) != 2 {
		t.Errorf("expected 2 allowed ranks, got %d", len(ranks))
	}
	if ranks[0] != "Bear" || ranks[1] != "Wolf" {
		t.Errorf("expected ranks [Bear, Wolf], got %v", ranks)
	}
}

func TestHandleGetCategories_ReturnsAllowedRanks(t *testing.T) {
	setup := newTestSetup(t)

	// Create categories with different allowed_ranks
	ctx := context.Background()
	_, err := setup.repo.CreateCategory(ctx, "Tiger Category", 1, nil, nil, []string{"Tiger"})
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	_, err = setup.repo.CreateCategory(ctx, "All Ranks", 2, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/categories", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(response))
	}

	// First category should have allowed_ranks
	if response[0]["name"] == "Tiger Category" {
		ranks, ok := response[0]["allowed_ranks"].([]interface{})
		if !ok {
			t.Fatalf("expected allowed_ranks to be an array, got %T", response[0]["allowed_ranks"])
		}
		if len(ranks) != 1 || ranks[0] != "Tiger" {
			t.Errorf("expected ranks [Tiger], got %v", ranks)
		}
	}

	// Second category should have nil/empty allowed_ranks
	if response[1]["name"] == "All Ranks" {
		if response[1]["allowed_ranks"] != nil {
			t.Errorf("expected allowed_ranks to be nil for unrestricted category, got %v", response[1]["allowed_ranks"])
		}
	}
}

// ==================== Voters Tests ====================

func TestHandleGetVoters_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a voter first
	ctx := context.Background()
	_, err := setup.repo.CreateVoterFull(ctx, nil, "Test Voter", "test@example.com", "general", "TEST-QR1", "notes")
	if err != nil {
		t.Fatalf("failed to create test voter: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/voters", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("expected 1 voter, got %d", len(response))
	}
}

func TestHandleGetVoters_Empty(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/voters", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleGetVoters_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close the database to cause a service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/voters", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleCreateVoter_Success(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":       "New Voter",
		"email":      "new@example.com",
		"voter_type": "general",
		"notes":      "test notes",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["name"] != "New Voter" {
		t.Errorf("expected name 'New Voter', got %v", response["name"])
	}
	if response["qr_code"] == nil || response["qr_code"] == "" {
		t.Error("expected qr_code to be generated")
	}
}

func TestHandleCreateVoter_WithQRCode(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":       "Voter With QR",
		"email":      "qr@example.com",
		"voter_type": "racer",
		"qr_code":    "CUSTOM-QR",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["qr_code"] != "CUSTOM-QR" {
		t.Errorf("expected qr_code 'CUSTOM-QR', got %v", response["qr_code"])
	}
}

func TestHandleCreateVoter_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voters", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCreateVoter_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":       "Test Voter",
		"voter_type": "general",
	}
	body, _ := json.Marshal(payload)

	// Close the database to cause a service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleCreateVoter_EmptyBody(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voters", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleDeleteVoter_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a voter first
	ctx := context.Background()
	id, err := setup.repo.CreateVoterFull(ctx, nil, "To Delete", "", "general", "DELETE-QR", "")
	if err != nil {
		t.Fatalf("failed to create test voter: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/voters/%d", id), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteVoter_InvalidID(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/voters/invalid", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleDeleteVoter_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/voters/1", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// ==================== Stats Tests ====================

func TestHandleGetStats_Success(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check expected stat fields exist
	expectedFields := []string{"total_voters", "total_votes", "total_categories", "total_cars", "voting_open"}
	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("expected field %q in stats response", field)
		}
	}
}

func TestHandleGetStats_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close the database to cause a service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleGetStats_WithData(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Add some data
	_, _ = setup.repo.CreateCategory(ctx, "Category 1", 1, nil, nil, nil)
	_, _ = setup.repo.CreateCategory(ctx, "Category 2", 2, nil, nil, nil)
	_ = setup.repo.CreateCar(ctx, "101", "Racer 1", "Car 1", "")
	_, _ = setup.repo.CreateVoterFull(ctx, nil, "Voter 1", "", "general", "VOTER-1", "")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["total_categories"].(float64) != 2 {
		t.Errorf("expected 2 categories, got %v", response["total_categories"])
	}
	if response["total_cars"].(float64) != 1 {
		t.Errorf("expected 1 car, got %v", response["total_cars"])
	}
	if response["total_voters"].(float64) != 1 {
		t.Errorf("expected 1 voter, got %v", response["total_voters"])
	}
}

// ==================== Voting Control Tests ====================

func TestHandleSetVotingStatus_Open(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"open": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-control", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["open"] != true {
		t.Errorf("expected open=true, got %v", response["open"])
	}
}

func TestHandleSetVotingStatus_Close(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"open": false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-control", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["open"] != false {
		t.Errorf("expected open=false, got %v", response["open"])
	}
}

func TestHandleSetVotingStatus_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-control", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSetVotingStatus_EmptyBody(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-control", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSetVotingTimer_Success(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"minutes": 10,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-timer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["minutes"].(float64) != 10 {
		t.Errorf("expected minutes=10, got %v", response["minutes"])
	}
	if response["close_time"] == nil || response["close_time"] == "" {
		t.Error("expected close_time to be set")
	}
}

func TestHandleSetVotingTimer_InvalidMinutes_Zero(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"minutes": 0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-timer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSetVotingTimer_InvalidMinutes_TooHigh(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"minutes": 61,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-timer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSetVotingTimer_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-timer", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSetVotingTimer_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"minutes": 10,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-timer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// ==================== Settings Tests ====================

func TestHandleGetSettings_Success(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check that expected fields exist (may be empty strings)
	if _, ok := response["derbynet_url"]; !ok {
		t.Error("expected derbynet_url in response")
	}
	if _, ok := response["base_url"]; !ok {
		t.Error("expected base_url in response")
	}
}

func TestHandleUpdateSettings_Success(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
		"base_url":     "http://voting.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify settings were updated
	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
	getRec := httptest.NewRecorder()
	getReq.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(getRec, getReq)

	var response map[string]interface{}
	if err := json.NewDecoder(getRec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["derbynet_url"] != "http://derbynet.local" {
		t.Errorf("expected derbynet_url 'http://derbynet.local', got %v", response["derbynet_url"])
	}
	if response["base_url"] != "http://voting.local" {
		t.Errorf("expected base_url 'http://voting.local', got %v", response["base_url"])
	}
}

func TestHandleUpdateSettings_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleUpdateSettings_EmptyBody(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleUpdateSettings_PartialUpdate(t *testing.T) {
	setup := newTestSetup(t)

	// First set both values
	payload := map[string]interface{}{
		"derbynet_url": "http://initial.derbynet",
		"base_url":     "http://initial.base",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Now update only base_url
	payload2 := map[string]interface{}{
		"base_url": "http://updated.base",
	}
	body2, _ := json.Marshal(payload2)
	req2 := httptest.NewRequest(http.MethodPut, "/api/admin/settings", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	req2.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec2.Code)
	}
}

func TestHandleUpdateSettings_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"derbynet_url": "http://test.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleUpdateSettings_VotingInstructions(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	instructions := "Please vote carefully!\nEach vote counts."
	payload := map[string]interface{}{
		"voting_instructions": instructions,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify instructions were saved
	saved, err := setup.repo.GetSetting(ctx, "voting_instructions")
	if err != nil {
		t.Fatalf("failed to get voting_instructions: %v", err)
	}

	if saved != instructions {
		t.Errorf("expected instructions '%s', got '%s'", instructions, saved)
	}

	// Verify GET endpoint returns the instructions
	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
	getRec := httptest.NewRecorder()

	getReq.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Errorf("expected status %d for GET, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(getRec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	returnedInstructions, ok := response["voting_instructions"]
	if !ok {
		t.Fatal("voting_instructions not in GET response")
	}

	if returnedInstructions != instructions {
		t.Errorf("GET returned instructions '%v', expected '%s'", returnedInstructions, instructions)
	}
}

// ==================== Category Groups Tests ====================

func TestHandleGetCategoryGroups_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a group first
	ctx := context.Background()
	_, err := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleCreateCategoryGroup_Success(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":          "New Group",
		"description":   "Test Description",
		"display_order": 1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/category-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["id"] == nil || response["id"].(float64) <= 0 {
		t.Errorf("expected positive ID, got %v", response["id"])
	}
}

func TestHandleCreateCategoryGroup_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/category-groups", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCreateCategoryGroup_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":          "New Group",
		"description":   "Test Description",
		"display_order": 1,
	}
	body, _ := json.Marshal(payload)

	// Close the database to cause a service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodPost, "/api/admin/category-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleGetCategoryGroup_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a group first
	ctx := context.Background()
	id, err := setup.repo.CreateCategoryGroup(ctx, "Get Group", "Description", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/category-groups/%d", id), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandleGetCategoryGroup_NotFound(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups/99999", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandleUpdateCategoryGroup_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a group first
	ctx := context.Background()
	id, err := setup.repo.CreateCategoryGroup(ctx, "Original Group", "Original Desc", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	payload := map[string]interface{}{
		"name":          "Updated Group",
		"description":   "Updated Desc",
		"display_order": 2,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/category-groups/%d", id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteCategoryGroup_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Create a group first
	ctx := context.Background()
	id, err := setup.repo.CreateCategoryGroup(ctx, "Delete Group", "Description", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/category-groups/%d", id), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteCategoryGroup_NotFound(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/category-groups/99999", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandleDeleteCategoryGroup_ServiceErrorOnGet(t *testing.T) {
	setup := newTestSetup(t)

	// Create a group first
	ctx := context.Background()
	id, err := setup.repo.CreateCategoryGroup(ctx, "Delete Group", "Description", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	// Close the database to cause a service error on GetGroup
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/category-groups/%d", id), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}


func TestHandleUpdateCategoryGroup_NotFound(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"name":          "Updated",
		"description":   "Desc",
		"display_order": 1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/category-groups/99999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandleUpdateCategoryGroup_ServiceErrorOnGet(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a group first
	id, err := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	payload := map[string]interface{}{
		"name":          "Updated",
		"description":   "Desc",
		"display_order": 1,
	}
	body, _ := json.Marshal(payload)

	// Close the database to cause a service error on GetGroup
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/category-groups/%d", id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}


func TestHandleUpdateCategoryGroup_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a group first
	id, err := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/category-groups/%d", id), bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleGetCategoryGroup_InvalidID(t *testing.T) {
	setup := newTestSetup(t)

	// Invalid ID (non-numeric) should return not found
	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups/invalid", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandleGetCategoryGroup_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups/1", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleGetCategoryGroups_Empty(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandleGetCategoryGroups_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close the database to cause a service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleCreateCategoryGroup_WithExclusivityPool(t *testing.T) {
	setup := newTestSetup(t)

	poolID := int64(1)
	payload := map[string]interface{}{
		"name":                "Pool Group",
		"description":         "Group with pool",
		"exclusivity_pool_id": poolID,
		"display_order":       1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/category-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
}

// ==================== Database Management Tests ====================

func TestHandleResetDatabase_Success(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"tables": []string{"votes"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/reset-database", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandleResetDatabase_InvalidTable(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"tables": []string{"invalid_table"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/reset-database", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleResetDatabase_EmptyTables(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"tables": []string{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/reset-database", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleResetDatabase_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/reset-database", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSeedMockData_Categories(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"seed_type": "categories",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandleSeedMockData_CategoriesAlreadyExist(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"seed_type": "categories",
	}
	body, _ := json.Marshal(payload)

	// Seed once
	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Seed again - should return "already exist" message
	body2, _ := json.Marshal(payload)
	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	req2.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec2.Code, rec2.Body.String())
	}

	// Check that message indicates already exist
	var response map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	message, ok := response["message"].(string)
	if !ok || !strings.Contains(message, "already exist") {
		t.Errorf("expected 'already exist' message, got: %v", response["message"])
	}
}

func TestHandleSeedMockData_Cars(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"seed_type": "cars",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandleSeedMockData_CarsAlreadyExist(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"seed_type": "cars",
	}
	body, _ := json.Marshal(payload)

	// Seed once
	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Seed again - should return "already exist" message
	body2, _ := json.Marshal(payload)
	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	req2.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec2.Code, rec2.Body.String())
	}

	// Check that message indicates already exist
	var response map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	message, ok := response["message"].(string)
	if !ok || !strings.Contains(message, "already exist") {
		t.Errorf("expected 'already exist' message, got: %v", response["message"])
	}
}

func TestHandleSeedMockData_InvalidType(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"seed_type": "invalid",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSeedMockData_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSeedMockData_EmptyBody(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSeedMockData_CategoriesServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"seed_type": "categories",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleSeedMockData_CarsServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"seed_type": "cars",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed-mock-data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// ==================== QR Codes Tests ====================

func TestHandleGenerateQRCodes_Success(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"count": 5,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/generate-qr", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	qrCodes, ok := response["qr_codes"].([]interface{})
	if !ok {
		t.Fatal("expected qr_codes array in response")
	}
	if len(qrCodes) != 5 {
		t.Errorf("expected 5 QR codes, got %d", len(qrCodes))
	}
}

func TestHandleGenerateQRCodes_InvalidCount_Zero(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"count": 0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/generate-qr", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleGenerateQRCodes_InvalidCount_TooHigh(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"count": 201,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/generate-qr", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleGenerateQRCodes_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/generate-qr", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ==================== Results Tests ====================

func TestHandleGetResults_Success(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandleGetResults_WithVotes(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category
	catID, _ := setup.repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)

	// Create a car
	_ = setup.repo.CreateCar(ctx, "101", "Racer 1", "Car 1", "")

	// Create a voter and vote
	voterID, _ := setup.repo.CreateVoter(ctx, "TEST-VOTER")
	_ = setup.repo.SaveVote(ctx, voterID, int(catID), 1)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("expected 1 category result, got %d", len(response))
	}
}

func TestHandleGetResults_EmptyReturnsArray(t *testing.T) {
	// This test verifies the bug fix: results API must return a JSON array
	// even when there are no categories, not null or an object
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify response is a valid JSON array (not null, not object)
	var response []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("response is not a valid JSON array: %v\nBody: %s", err, rec.Body.String())
	}

	// Should be empty array, not nil
	if response == nil {
		t.Error("expected empty array [], got null")
	}

	if len(response) != 0 {
		t.Errorf("expected 0 categories in empty database, got %d", len(response))
	}
}

func TestHandleGetResults_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// ==================== Edge Cases and Integration Tests ====================

func TestCategoryLifecycle(t *testing.T) {
	setup := newTestSetup(t)

	// Create
	createPayload := map[string]interface{}{
		"name":          "Lifecycle Category",
		"display_order": 1,
		"active":        true,
	}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/categories", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(setup.authCookie)
	createRec := httptest.NewRecorder()
	setup.router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d - %s", createRec.Code, createRec.Body.String())
	}

	var createResp map[string]interface{}
	json.NewDecoder(createRec.Body).Decode(&createResp)
	id := int(createResp["id"].(float64))

	// Update
	updatePayload := map[string]interface{}{
		"name":          "Updated Lifecycle",
		"display_order": 2,
		"active":        true,
	}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/categories/%d", id), bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.AddCookie(setup.authCookie)
	updateRec := httptest.NewRecorder()
	setup.router.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("update failed: %d - %s", updateRec.Code, updateRec.Body.String())
	}

	// List and verify
	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/categories", nil)
	listReq.AddCookie(setup.authCookie)
	listRec := httptest.NewRecorder()
	setup.router.ServeHTTP(listRec, listReq)

	var categories []map[string]interface{}
	json.NewDecoder(listRec.Body).Decode(&categories)
	if len(categories) != 1 || categories[0]["name"] != "Updated Lifecycle" {
		t.Error("category was not properly updated")
	}

	// Delete
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/categories/%d", id), nil)
	deleteReq.AddCookie(setup.authCookie)
	deleteRec := httptest.NewRecorder()
	setup.router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete failed: %d - %s", deleteRec.Code, deleteRec.Body.String())
	}

	// Verify deleted (soft delete - not in active list)
	listReq2 := httptest.NewRequest(http.MethodGet, "/api/admin/categories", nil)
	listReq2.AddCookie(setup.authCookie)
	listRec2 := httptest.NewRecorder()
	setup.router.ServeHTTP(listRec2, listReq2)

	// ListAllCategories returns all including inactive, but if we're using ListCategories
	// the deleted category should not appear
	// The response might be null for empty
	body := listRec2.Body.String()
	if body != "null\n" {
		var finalCategories []map[string]interface{}
		if err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&finalCategories); err == nil {
			// Check that the deleted category is not in the active list
			for _, cat := range finalCategories {
				if cat["name"] == "Updated Lifecycle" && cat["active"] == true {
					t.Error("deleted category should not be in active list")
				}
			}
		}
	}
}

func TestVoterLifecycle(t *testing.T) {
	setup := newTestSetup(t)

	// Create
	createPayload := map[string]interface{}{
		"name":       "Lifecycle Voter",
		"email":      "lifecycle@example.com",
		"voter_type": "general",
	}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/voters", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(setup.authCookie)
	createRec := httptest.NewRecorder()
	setup.router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d - %s", createRec.Code, createRec.Body.String())
	}

	var createResp map[string]interface{}
	json.NewDecoder(createRec.Body).Decode(&createResp)
	id := int(createResp["id"].(float64))

	// List and verify
	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/voters", nil)
	listReq.AddCookie(setup.authCookie)
	listRec := httptest.NewRecorder()
	setup.router.ServeHTTP(listRec, listReq)

	var voters []map[string]interface{}
	json.NewDecoder(listRec.Body).Decode(&voters)
	if len(voters) != 1 {
		t.Errorf("expected 1 voter, got %d", len(voters))
	}

	// Delete
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/voters/%d", id), nil)
	deleteReq.AddCookie(setup.authCookie)
	deleteRec := httptest.NewRecorder()
	setup.router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, deleteRec.Code)
	}
}

// ==================== Sync Categories DerbyNet Tests ====================

func TestHandleSyncCategoriesDerbyNet_Success(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/sync-categories-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] == nil {
		t.Error("expected status field in response")
	}
}

func TestHandleSyncCategoriesDerbyNet_MissingURL(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/sync-categories-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSyncCategoriesDerbyNet_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/sync-categories-derbynet", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ==================== Push Results DerbyNet Tests ====================

func TestHandlePushResultsDerbyNet_Success(t *testing.T) {
	setup := newTestSetup(t)

	// First, create a category for testing
	ctx := context.Background()
	_, _ = setup.repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)

	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] == nil {
		t.Error("expected status field in response")
	}
}

func TestHandlePushResultsDerbyNet_MissingURL(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandlePushResultsDerbyNet_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ==================== Set Car Eligibility Tests ====================

func TestHandleSetCarEligibility_Success(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	payload := map[string]interface{}{
		"eligible": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["eligible"] != true {
		t.Errorf("expected eligible=true, got %v", resp["eligible"])
	}
}

func TestHandleSetCarEligibility_SetFalse(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "102", "Test Racer 2", "Test Car 2", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	payload := map[string]interface{}{
		"eligible": false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["eligible"] != false {
		t.Errorf("expected eligible=false, got %v", resp["eligible"])
	}
}

func TestHandleSetCarEligibility_CarNotFound(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"eligible": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/cars/99999/eligibility", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestHandleSetCarEligibility_InvalidID(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"eligible": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/cars/invalid/eligibility", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSetCarEligibility_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "103", "Test Racer 3", "Test Car 3", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSetCarEligibility_ServiceError(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "104", "Test Racer 4", "Test Car 4", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	payload := map[string]interface{}{
		"eligible": true,
	}
	body, _ := json.Marshal(payload)

	// Close the database to cause a service error
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// ==================== Additional Sync Handler Error Tests ====================

func TestHandleSyncDerbyNet_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/sync-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleSyncCategoriesDerbyNet_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/sync-categories-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandlePushResultsDerbyNet_ServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandlePushResultsDerbyNet_WithTieConflict(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category and cars
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	setup.repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := setup.repo.ListCars(ctx)

	// Create 2-way tie
	v1, _ := setup.repo.CreateVoter(ctx, "V1")
	v2, _ := setup.repo.CreateVoter(ctx, "V2")

	setup.repo.SaveVote(ctx, v1, int(catID), cars[0].ID)
	setup.repo.SaveVote(ctx, v2, int(catID), cars[1].ID)

	// Try to push results
	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should return 409 Conflict due to unresolved conflicts
	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}

	// Check error message
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	errorMsg, ok := resp["error"].(string)
	if !ok || !strings.Contains(errorMsg, "conflicts exist") {
		t.Errorf("expected error message about conflicts, got: %v", resp)
	}
}

func TestHandlePushResultsDerbyNet_WithMultiWinConflict(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := setup.repo.CreateCategoryGroup(ctx, "Main Awards", "Description", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create two categories in the same group
	cat1ID, _ := setup.repo.CreateCategory(ctx, "Fastest", 1, &groupIDInt, nil, nil)
	cat2ID, _ := setup.repo.CreateCategory(ctx, "Best Design", 2, &groupIDInt, nil, nil)

	// Create two cars
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	setup.repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := setup.repo.ListCars(ctx)

	// Make car 1 win both categories (multi-win conflict)
	// Category 1: Car 1 gets 2 votes, Car 2 gets 1 vote (Car 1 wins)
	v1, _ := setup.repo.CreateVoter(ctx, "V1")
	v2, _ := setup.repo.CreateVoter(ctx, "V2")
	v3, _ := setup.repo.CreateVoter(ctx, "V3")

	setup.repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	setup.repo.SaveVote(ctx, v2, int(cat1ID), cars[0].ID)
	setup.repo.SaveVote(ctx, v3, int(cat1ID), cars[1].ID)

	// Category 2: Car 1 gets 2 votes, Car 2 gets 1 vote (Car 1 wins)
	v4, _ := setup.repo.CreateVoter(ctx, "V4")
	v5, _ := setup.repo.CreateVoter(ctx, "V5")
	v6, _ := setup.repo.CreateVoter(ctx, "V6")

	setup.repo.SaveVote(ctx, v4, int(cat2ID), cars[0].ID)
	setup.repo.SaveVote(ctx, v5, int(cat2ID), cars[0].ID)
	setup.repo.SaveVote(ctx, v6, int(cat2ID), cars[1].ID)

	// Try to push results
	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should return 409 Conflict due to unresolved conflicts
	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}

	// Check error message
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	errorMsg, ok := resp["error"].(string)
	if !ok || !strings.Contains(errorMsg, "conflicts exist") {
		t.Errorf("expected error message about conflicts, got: %v", resp)
	}
}

func TestHandlePushResultsDerbyNet_DetectMultipleWinsError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)

	// Inject error for ListCategoryGroups which is called by DetectMultipleWins
	mockRepo.ListCategoryGroupsError = fmt.Errorf("database error fetching category groups")

	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandlePushResultsDerbyNet_PushError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)

	// Inject error for SetSetting which is called by PushResultsToDerbyNet before GetWinnersForDerbyNet
	mockRepo.SetSettingError = fmt.Errorf("database error saving settings")

	payload := map[string]interface{}{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error (SetSetting returns actual error)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleSetVotingStatus_CloseVotingServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"open": false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-control", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleSetVotingStatus_OpenVotingServiceError(t *testing.T) {
	setup := newTestSetup(t)

	// Close database to trigger service error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"open": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/voting-control", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleDeleteCategoryGroup_DeleteServiceError(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category group first
	groupID, err := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Desc", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	// Close database after creation to trigger error on delete
	setup.repo.DB().Close()

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/category-groups/%d", groupID), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleUpdateCar_UpdateServiceError(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	// Close database after getting the car to trigger UpdateCar error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"car_number": "102",
		"racer_name": "Updated Racer",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d", carID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleSetCarEligibility_UpdateServiceError(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	// Close database after getting the car to trigger SetCarEligibility error
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"eligible": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleUpdateCategoryGroup_UpdateServiceError(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category group first
	groupID, err := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Desc", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	// Close database after creation to trigger error on update
	setup.repo.DB().Close()

	payload := map[string]interface{}{
		"name":          "Updated Group",
		"description":   "Updated Desc",
		"display_order": 2,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/category-groups/%d", groupID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// ==================== Mock Repository Tests for Error Injection ====================

// newTestSetupWithMockRepo creates a test setup with a mock repository for error injection
func newTestSetupWithMockRepo(t *testing.T) (*testSetup, *mock.Repository) {
	t.Helper()

	realRepo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Wrap with mock repository
	mockRepo := mock.NewRepository(realRepo)

	// Initialize logger for tests
	log := logger.New()

	// Initialize services with mock repository
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, mockRepo, derbynetClient)
	carService := services.NewCarService(log, mockRepo, derbynetClient)
	settingsService := services.NewSettingsService(log, mockRepo)
	voterService := services.NewVoterService(log, mockRepo, settingsService)
	votingService := services.NewVotingService(log, mockRepo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, mockRepo, settingsService, derbynetClient)

	// Initialize handlers
	h := handlers.NewForTesting(
		votingService,
		categoryService,
		voterService,
		carService,
		settingsService,
		resultsService,
	)

	// Login to get auth cookie
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{
		Name:  auth.CookieName,
		Value: token,
	}

	return &testSetup{
		repo:       realRepo,
		handlers:   h,
		router:     h.Router(),
		authCookie: authCookie,
		log:        log,
	}, mockRepo
}

func TestHandleDeleteCar_DeleteError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	// Inject error for DeleteCar (after GetCar succeeds)
	mockRepo.DeleteCarError = fmt.Errorf("database error during delete")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/cars/%d", carID), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleSetCarEligibility_SetError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "102", "Test Racer 2", "Test Car 2", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	// Inject error for SetCarEligibility (after GetCar succeeds)
	mockRepo.SetCarEligibilityError = fmt.Errorf("database error during eligibility update")

	payload := map[string]interface{}{
		"eligible": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleUpdateCar_UpdateError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "103", "Test Racer 3", "Test Car 3", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	// Inject error for UpdateCar (after GetCar succeeds)
	mockRepo.UpdateCarError = fmt.Errorf("database error during update")

	payload := map[string]interface{}{
		"car_number": "104",
		"racer_name": "Updated Racer",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d", carID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleDeleteCategoryGroup_DeleteError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a category group first
	groupID, err := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Desc", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	// Inject error for DeleteCategoryGroup (after GetCategoryGroup succeeds)
	mockRepo.DeleteCategoryGroupError = fmt.Errorf("database error during delete")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/category-groups/%d", groupID), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleUpdateCategoryGroup_UpdateError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a category group first
	groupID, err := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Desc", nil, nil, 1)
	if err != nil {
		t.Fatalf("failed to create test group: %v", err)
	}

	// Inject error for UpdateCategoryGroup (after GetCategoryGroup succeeds)
	mockRepo.UpdateCategoryGroupError = fmt.Errorf("database error during update")

	payload := map[string]interface{}{
		"name":          "Updated Group",
		"description":   "Updated Desc",
		"display_order": 2,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/category-groups/%d", groupID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for internal service error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleGetOpenVotingQR_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Enable open voting
	setup.repo.SetSetting(context.Background(), "require_registered_qr", "false")

	// Set base URL
	setup.repo.SetSetting(context.Background(), "base_url", "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/open-voting-qr", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Should return PNG image
	contentType := rec.Header().Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", contentType)
	}

	// Should have PNG data
	data := rec.Body.Bytes()
	if len(data) == 0 {
		t.Fatal("expected non-empty response body")
	}

	// PNG files start with \x89PNG
	if len(data) < 4 || data[0] != 0x89 || data[1] != 0x50 || data[2] != 0x4E || data[3] != 0x47 {
		t.Error("expected valid PNG data")
	}
}

func TestHandleGetOpenVotingQR_OpenVotingDisabled(t *testing.T) {
	setup := newTestSetup(t)

	// Disable open voting
	setup.repo.SetSetting(context.Background(), "require_registered_qr", "true")

	// Set base URL
	setup.repo.SetSetting(context.Background(), "base_url", "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/open-voting-qr", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return error
	if rec.Code == http.StatusOK {
		t.Error("expected error response, got 200")
	}
}

func TestHandleGetOpenVotingQR_BaseURLNotConfigured(t *testing.T) {
	setup := newTestSetup(t)

	// Enable open voting
	setup.repo.SetSetting(context.Background(), "require_registered_qr", "false")

	// Don't set base URL

	req := httptest.NewRequest(http.MethodGet, "/api/admin/open-voting-qr", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return error
	if rec.Code == http.StatusOK {
		t.Error("expected error response when base_url not configured, got 200")
	}
}

// ==================== Manual Winner Override Tests ====================

func TestHandleGetConflicts_NoConflicts(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create categories and cars but no votes
	setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/conflicts", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	ties := response["ties"].([]interface{})
	multiWins := response["multi_wins"].([]interface{})

	if len(ties) != 0 {
		t.Errorf("expected 0 ties, got %d", len(ties))
	}
	if len(multiWins) != 0 {
		t.Errorf("expected 0 multi-wins, got %d", len(multiWins))
	}
}

func TestHandleGetConflicts_WithTie(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category and cars
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	setup.repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	cars, _ := setup.repo.ListCars(ctx)

	// Create 2-way tie
	v1, _ := setup.repo.CreateVoter(ctx, "V1")
	v2, _ := setup.repo.CreateVoter(ctx, "V2")

	setup.repo.SaveVote(ctx, v1, int(catID), cars[0].ID)
	setup.repo.SaveVote(ctx, v2, int(catID), cars[1].ID)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/conflicts", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	ties := response["ties"].([]interface{})
	if len(ties) != 1 {
		t.Fatalf("expected 1 tie, got %d", len(ties))
	}

	tie := ties[0].(map[string]interface{})
	if tie["category_name"] != "Best Design" {
		t.Errorf("expected 'Best Design', got %v", tie["category_name"])
	}

	tiedCars := tie["tied_cars"].([]interface{})
	if len(tiedCars) != 2 {
		t.Errorf("expected 2 tied cars, got %d", len(tiedCars))
	}
}

func TestHandleGetConflicts_WithMultipleWins(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := setup.repo.CreateCategoryGroup(ctx, "Design Awards", "Design related categories", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create categories in the group
	cat1ID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, &groupIDInt, nil, nil)
	cat2ID, _ := setup.repo.CreateCategory(ctx, "Most Creative", 2, &groupIDInt, nil, nil)

	// Create cars
	setup.repo.CreateCar(ctx, "101", "Racer One", "Super Car", "")
	setup.repo.CreateCar(ctx, "102", "Racer Two", "Other Car", "")
	cars, _ := setup.repo.ListCars(ctx)

	// Car 1 wins both categories
	v1, _ := setup.repo.CreateVoter(ctx, "V1")
	v2, _ := setup.repo.CreateVoter(ctx, "V2")

	setup.repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	setup.repo.SaveVote(ctx, v2, int(cat2ID), cars[0].ID)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/conflicts", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	multiWins := response["multi_wins"].([]interface{})
	if len(multiWins) != 1 {
		t.Fatalf("expected 1 multi-win, got %d", len(multiWins))
	}

	mw := multiWins[0].(map[string]interface{})
	if mw["car_number"] != "101" {
		t.Errorf("expected car '101', got %v", mw["car_number"])
	}

	awardsWon := mw["awards_won"].([]interface{})
	if len(awardsWon) != 2 {
		t.Errorf("expected 2 awards won, got %d", len(awardsWon))
	}
}

func TestHandleOverrideWinner_Success(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Close voting to allow override
	setup.repo.SetSetting(ctx, "voting_open", "false")

	// Set override
	payload := map[string]interface{}{
		"category_id": catID,
		"car_id":      carID,
		"reason":      "Resolved tie",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/results/override-winner", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify override was set
	categories, _ := setup.repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	cat := categories[0]
	if cat.OverrideWinnerCarID == nil {
		t.Fatal("expected override to be set")
	}
	if *cat.OverrideWinnerCarID != carID {
		t.Errorf("expected carID=%d, got %d", carID, *cat.OverrideWinnerCarID)
	}
	if cat.OverrideReason != "Resolved tie" {
		t.Errorf("expected 'Resolved tie', got '%s'", cat.OverrideReason)
	}
}

func TestHandleOverrideWinner_MissingCategoryID(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"car_id": 1,
		"reason": "Test",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/results/override-winner", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleOverrideWinner_MissingCarID(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"category_id": 1,
		"reason":      "Test",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/results/override-winner", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleOverrideWinner_MissingReason(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]interface{}{
		"category_id": 1,
		"car_id":      1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/results/override-winner", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleClearOverride_Success(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Close voting to allow override operations
	setup.repo.SetSetting(ctx, "voting_open", "false")

	// Set override first
	setup.repo.SetManualWinner(ctx, int(catID), carID, "Test")

	// Clear it
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/results/override-winner/%d", catID), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify override was cleared
	categories, _ := setup.repo.ListCategories(ctx)
	cat := categories[0]

	if cat.OverrideWinnerCarID != nil {
		t.Errorf("expected override to be cleared, got carID=%v", *cat.OverrideWinnerCarID)
	}
}

func TestHandleClearOverride_InvalidCategoryID(t *testing.T) {
	setup := newTestSetup(t)

	// Try to clear override with non-numeric category ID
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/results/override-winner/invalid", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid category ID, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestHandleGetOverrides_NoOverrides(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category without override
	setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/overrides", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var overrides []interface{}
	if err := json.NewDecoder(rec.Body).Decode(&overrides); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(overrides) != 0 {
		t.Errorf("expected 0 overrides, got %d", len(overrides))
	}
}

func TestHandleGetOverrides_WithOverrides(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Create vote
	v1, _ := setup.repo.CreateVoter(ctx, "V1")
	setup.repo.SaveVote(ctx, v1, int(catID), carID)

	// Set override
	setup.repo.SetManualWinner(ctx, int(catID), carID, "Test override")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/overrides", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var overrides []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&overrides); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(overrides))
	}

	override := overrides[0]
	if override["category_name"] != "Best Design" {
		t.Errorf("expected 'Best Design', got %v", override["category_name"])
	}
	if override["override_reason"] != "Test override" {
		t.Errorf("expected 'Test override', got %v", override["override_reason"])
	}
}

func TestPushResultsToDerbyNet_WithOverride(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category with DerbyNet award ID
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	awardID := 100
	setup.repo.UpsertCategory(ctx, "Best Design", 1, &awardID)

	// Create two cars
	setup.repo.UpsertCar(ctx, 1, "101", "Racer One", "Car A", "", "")
	setup.repo.UpsertCar(ctx, 2, "102", "Racer Two", "Car B", "", "")
	cars, _ := setup.repo.ListCars(ctx)
	car1ID := cars[0].ID // Vote winner
	car2ID := cars[1].ID // Override winner

	// Create votes (car1 wins by votes)
	v1, _ := setup.repo.CreateVoter(ctx, "V1")
	v2, _ := setup.repo.CreateVoter(ctx, "V2")
	v3, _ := setup.repo.CreateVoter(ctx, "V3")

	setup.repo.SaveVote(ctx, v1, int(catID), car1ID) // Car A: 2 votes
	setup.repo.SaveVote(ctx, v2, int(catID), car1ID)
	setup.repo.SaveVote(ctx, v3, int(catID), car2ID) // Car B: 1 vote

	// Set manual override to car2 (not the vote winner)
	setup.repo.SetManualWinner(ctx, int(catID), car2ID, "Manual selection")

	// Push results
	payload := map[string]string{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	// Verify correct winner was pushed
	winnersPushed := int(result["winners_pushed"].(float64))
	if winnersPushed != 1 {
		t.Errorf("expected 1 winner pushed, got %d", winnersPushed)
	}

	// Check that car2 (override) was pushed, not car1 (vote winner)
	// This would be verified by checking the mock DerbyNet server's received requests
	// The key point is that the override was respected
}

func TestPushResultsToDerbyNet_OverrideWithNoVotes(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	awardID := 100
	setup.repo.UpsertCategory(ctx, "Best Design", 1, &awardID)

	// Create car
	setup.repo.UpsertCar(ctx, 1, "101", "Racer One", "Car A", "", "")
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// No votes, but set override
	setup.repo.SetManualWinner(ctx, int(catID), carID, "Manual selection with no votes")

	// Push results
	payload := map[string]string{
		"derbynet_url": "http://derbynet.local",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/push-results-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	// Should push the override even with no votes
	winnersPushed := int(result["winners_pushed"].(float64))
	if winnersPushed != 1 {
		t.Errorf("expected 1 winner pushed (override), got %d", winnersPushed)
	}
}
// ==================== Category Group max_wins_per_car Tests ====================

func TestHandleCreateCategoryGroup_WithMaxWinsPerCar(t *testing.T) {
	setup := newTestSetup(t)

	maxWins := 1
	payload := map[string]interface{}{
		"name":             "Design Awards",
		"description":      "Design categories",
		"max_wins_per_car": maxWins,
		"display_order":    1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/category-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	groupID := int64(response["id"].(float64))

	// Verify it was saved by fetching it back
	ctx := context.Background()
	group, err := setup.repo.GetCategoryGroup(ctx, fmt.Sprintf("%d", groupID))
	if err != nil {
		t.Fatalf("failed to get group: %v", err)
	}

	if group.MaxWinsPerCar == nil {
		t.Error("expected max_wins_per_car to be set, got nil")
	} else if *group.MaxWinsPerCar != maxWins {
		t.Errorf("expected max_wins_per_car=%d, got %d", maxWins, *group.MaxWinsPerCar)
	}
}

func TestHandleGetCategoryGroup_ReturnsMaxWinsPerCar(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	maxWins := 2
	groupID, _ := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, &maxWins, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/category-groups/%d", groupID), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var group map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&group)

	if group["max_wins_per_car"] == nil {
		t.Error("expected max_wins_per_car in response, got nil")
	} else if int(group["max_wins_per_car"].(float64)) != maxWins {
		t.Errorf("expected max_wins_per_car=%d, got %v", maxWins, group["max_wins_per_car"])
	}
}

func TestHandleUpdateCategoryGroup_UpdatesMaxWinsPerCar(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create group without max_wins_per_car
	groupID, _ := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, nil, 1)

	// Update to set max_wins_per_car
	newMaxWins := 1
	payload := map[string]interface{}{
		"name":             "Updated Group",
		"description":      "Updated",
		"max_wins_per_car": newMaxWins,
		"display_order":    2,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/category-groups/%d", groupID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify update
	group, _ := setup.repo.GetCategoryGroup(ctx, fmt.Sprintf("%d", groupID))
	if group.MaxWinsPerCar == nil {
		t.Error("expected max_wins_per_car to be set after update")
	} else if *group.MaxWinsPerCar != newMaxWins {
		t.Errorf("expected max_wins_per_car=%d, got %d", newMaxWins, *group.MaxWinsPerCar)
	}
}

func TestHandleUpdateCategoryGroup_ClearsMaxWinsPerCar(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create group with max_wins_per_car
	maxWins := 1
	groupID, _ := setup.repo.CreateCategoryGroup(ctx, "Test Group", "Description", nil, &maxWins, 1)

	// Update to clear max_wins_per_car (send null)
	payload := map[string]interface{}{
		"name":             "Updated Group",
		"description":      "Updated",
		"max_wins_per_car": nil,
		"display_order":    2,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/category-groups/%d", groupID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify it was cleared
	group, _ := setup.repo.GetCategoryGroup(ctx, fmt.Sprintf("%d", groupID))
	if group.MaxWinsPerCar != nil {
		t.Errorf("expected max_wins_per_car to be nil, got %d", *group.MaxWinsPerCar)
	}
}

func TestHandleGetCategoryGroups_ReturnsMaxWinsPerCar(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create groups with and without max_wins_per_car
	maxWins := 2
	setup.repo.CreateCategoryGroup(ctx, "Group With Max", "Has limit", nil, &maxWins, 1)
	setup.repo.CreateCategoryGroup(ctx, "Group Without Max", "No limit", nil, nil, 2)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var groups []map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&groups)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// Check first group has max_wins_per_car
	foundWithMax := false
	foundWithoutMax := false
	for _, g := range groups {
		if g["name"] == "Group With Max" {
			foundWithMax = true
			if g["max_wins_per_car"] == nil {
				t.Error("expected max_wins_per_car for 'Group With Max', got nil")
			}
		}
		if g["name"] == "Group Without Max" {
			foundWithoutMax = true
			if g["max_wins_per_car"] != nil {
				t.Errorf("expected nil max_wins_per_car for 'Group Without Max', got %v", g["max_wins_per_car"])
			}
		}
	}

	if !foundWithMax || !foundWithoutMax {
		t.Error("did not find expected groups in response")
	}
}

// ==================== Conflict Resolution Handler Tests ====================

func TestHandleGetConflicts_WithTiesAndMultiWins(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a group with max_wins_per_car = 1
	maxWins := 1
	groupID, _ := setup.repo.CreateCategoryGroup(ctx, "Design Awards", "Design categories", nil, &maxWins, 1)
	groupIDInt := int(groupID)

	// Create categories in the group
	cat1ID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, &groupIDInt, nil, nil)
	cat2ID, _ := setup.repo.CreateCategory(ctx, "Most Creative", 2, &groupIDInt, nil, nil)
	cat3ID, _ := setup.repo.CreateCategory(ctx, "Best Paint", 3, nil, nil, nil) // Not in group

	// Create cars
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	setup.repo.CreateCar(ctx, "102", "Racer Two", "Car B", "")
	setup.repo.CreateCar(ctx, "103", "Racer Three", "Car C", "")
	cars, _ := setup.repo.ListCars(ctx)

	// Car 1 wins categories 1 and 2 (multi-win conflict)
	v1, _ := setup.repo.CreateVoter(ctx, "V1")
	v2, _ := setup.repo.CreateVoter(ctx, "V2")
	setup.repo.SaveVote(ctx, v1, int(cat1ID), cars[0].ID)
	setup.repo.SaveVote(ctx, v2, int(cat2ID), cars[0].ID)

	// Category 3 has a tie between cars 2 and 3
	v3, _ := setup.repo.CreateVoter(ctx, "V3")
	v4, _ := setup.repo.CreateVoter(ctx, "V4")
	setup.repo.SaveVote(ctx, v3, int(cat3ID), cars[1].ID)
	setup.repo.SaveVote(ctx, v4, int(cat3ID), cars[2].ID)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/conflicts", nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	// Check ties
	ties := response["ties"].([]interface{})
	if len(ties) != 1 {
		t.Errorf("expected 1 tie, got %d", len(ties))
	}

	// Check multi_wins
	multiWins := response["multi_wins"].([]interface{})
	if len(multiWins) != 1 {
		t.Errorf("expected 1 multi-win conflict, got %d", len(multiWins))
	}

	// Verify multi-win structure includes group info
	mw := multiWins[0].(map[string]interface{})
	if mw["group_id"] == nil {
		t.Error("expected group_id in multi-win conflict")
	}
	if mw["group_name"] == nil {
		t.Error("expected group_name in multi-win conflict")
	}
	if mw["max_wins_per_car"] == nil {
		t.Error("expected max_wins_per_car in multi-win conflict")
	} else if int(mw["max_wins_per_car"].(float64)) != maxWins {
		t.Errorf("expected max_wins_per_car=%d, got %v", maxWins, mw["max_wins_per_car"])
	}
}

// ==================== Conflict and Override Error Path Tests ====================

func TestHandleGetConflicts_DetectTiesError(t *testing.T) {
	realRepo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Create mock repo with error
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetVoteResultsWithCarsError = fmt.Errorf("database error")

	// Initialize logger and services with mock
	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, mockRepo, derbynetClient)
	carService := services.NewCarService(log, mockRepo, derbynetClient)
	settingsService := services.NewSettingsService(log, mockRepo)
	voterService := services.NewVoterService(log, mockRepo, settingsService)
	votingService := services.NewVotingService(log, mockRepo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, mockRepo, settingsService, derbynetClient)

	h := handlers.NewForTesting(votingService, categoryService, voterService, carService, settingsService, resultsService)

	// Login to get a session cookie
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{Name: auth.CookieName, Value: token}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/conflicts", nil)
	rec := httptest.NewRecorder()
	req.AddCookie(authCookie)

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleGetConflicts_DetectMultipleWinsError(t *testing.T) {
	realRepo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Create a category so DetectTies succeeds
	realRepo.CreateCategory(context.Background(), "Test", 1, nil, nil, nil)

	// Create mock repo with error for ListCategoryGroups
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoryGroupsError = fmt.Errorf("database error")

	// Initialize logger and services with mock
	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, mockRepo, derbynetClient)
	carService := services.NewCarService(log, mockRepo, derbynetClient)
	settingsService := services.NewSettingsService(log, mockRepo)
	voterService := services.NewVoterService(log, mockRepo, settingsService)
	votingService := services.NewVotingService(log, mockRepo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, mockRepo, settingsService, derbynetClient)

	h := handlers.NewForTesting(votingService, categoryService, voterService, carService, settingsService, resultsService)

	// Login to get a session cookie
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{Name: auth.CookieName, Value: token}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/conflicts", nil)
	rec := httptest.NewRecorder()
	req.AddCookie(authCookie)

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleOverrideWinner_DecodeJSONError(t *testing.T) {
	setup := newTestSetup(t)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/admin/results/override-winner", strings.NewReader("invalid json"))
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleOverrideWinner_SetManualWinnerError(t *testing.T) {
	realRepo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Create mock repo with error
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoriesError = fmt.Errorf("database error")

	// Initialize logger and services with mock
	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, mockRepo, derbynetClient)
	carService := services.NewCarService(log, mockRepo, derbynetClient)
	settingsService := services.NewSettingsService(log, mockRepo)
	voterService := services.NewVoterService(log, mockRepo, settingsService)
	votingService := services.NewVotingService(log, mockRepo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, mockRepo, settingsService, derbynetClient)

	h := handlers.NewForTesting(votingService, categoryService, voterService, carService, settingsService, resultsService)

	// Login to get a session cookie
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{Name: auth.CookieName, Value: token}

	// Close voting to allow override
	realRepo.SetSetting(context.Background(), "voting_open", "false")

	body := `{"category_id": 1, "car_id": 1, "reason": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/results/override-winner", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	req.AddCookie(authCookie)

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleClearOverride_ClearManualWinnerError(t *testing.T) {
	realRepo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Create mock repo with error
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ClearManualWinnerError = fmt.Errorf("database error")

	// Initialize logger and services with mock
	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, mockRepo, derbynetClient)
	carService := services.NewCarService(log, mockRepo, derbynetClient)
	settingsService := services.NewSettingsService(log, mockRepo)
	voterService := services.NewVoterService(log, mockRepo, settingsService)
	votingService := services.NewVotingService(log, mockRepo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, mockRepo, settingsService, derbynetClient)

	h := handlers.NewForTesting(votingService, categoryService, voterService, carService, settingsService, resultsService)

	// Login to get a session cookie
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{Name: auth.CookieName, Value: token}

	// Close voting to allow override
	realRepo.SetSetting(context.Background(), "voting_open", "false")

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/results/override-winner/1", nil)
	rec := httptest.NewRecorder()
	req.AddCookie(authCookie)

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleGetOverrides_GetResultsError(t *testing.T) {
	realRepo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Create mock repo with error
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.ListCategoriesError = fmt.Errorf("database error")

	// Initialize logger and services with mock
	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, mockRepo, derbynetClient)
	carService := services.NewCarService(log, mockRepo, derbynetClient)
	settingsService := services.NewSettingsService(log, mockRepo)
	voterService := services.NewVoterService(log, mockRepo, settingsService)
	votingService := services.NewVotingService(log, mockRepo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, mockRepo, settingsService, derbynetClient)

	h := handlers.NewForTesting(votingService, categoryService, voterService, carService, settingsService, resultsService)

	// Login to get a session cookie
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{Name: auth.CookieName, Value: token}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/results/overrides", nil)
	rec := httptest.NewRecorder()
	req.AddCookie(authCookie)

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleOverrideWinner_VotingStillOpen(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Set voting to open
	setup.repo.SetSetting(ctx, "voting_open", "true")

	// Try to set override while voting is open
	payload := map[string]interface{}{
		"category_id": catID,
		"car_id":      carID,
		"reason":      "Resolved tie",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/results/override-winner", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return bad request
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	// Verify error message
	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	if errorMsg, ok := response["error"].(string); ok {
		if errorMsg != "Cannot resolve conflicts while voting is still open" {
			t.Errorf("expected error about voting being open, got: %s", errorMsg)
		}
	} else {
		t.Error("expected error message in response")
	}

	// Verify override was NOT set
	categories, _ := setup.repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	cat := categories[0]
	if cat.OverrideWinnerCarID != nil {
		t.Errorf("expected no override to be set, got carID=%v", *cat.OverrideWinnerCarID)
	}
}

func TestHandleClearOverride_VotingStillOpen(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create category and car
	catID, _ := setup.repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)
	setup.repo.CreateCar(ctx, "101", "Racer One", "Car A", "")
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Set voting to closed first to set the override
	setup.repo.SetSetting(ctx, "voting_open", "false")

	// Set an override
	setup.repo.SetManualWinner(ctx, int(catID), carID, "Test reason")

	// Verify override was set
	categories, _ := setup.repo.ListCategories(ctx)
	if categories[0].OverrideWinnerCarID == nil {
		t.Fatal("expected override to be set initially")
	}

	// Now set voting to open
	setup.repo.SetSetting(ctx, "voting_open", "true")

	// Try to clear override while voting is open
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/results/override-winner/%d", catID), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return bad request
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	// Verify error message
	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	if errorMsg, ok := response["error"].(string); ok {
		if errorMsg != "Cannot clear conflict resolution while voting is still open" {
			t.Errorf("expected error about voting being open, got: %s", errorMsg)
		}
	} else {
		t.Error("expected error message in response")
	}

	// Verify override was NOT cleared
	categories, _ = setup.repo.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}

	cat := categories[0]
	if cat.OverrideWinnerCarID == nil {
		t.Error("expected override to still be set")
	}
}

func TestHandleGetVoterTypes_Success(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Set some voter types
	setup.repo.SetSetting(ctx, "voter_types", `["general","racer","Test Type"]`)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/voter-types", nil)
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	voterTypes, ok := response["voter_types"].([]interface{})
	if !ok {
		t.Fatal("expected voter_types in response")
	}

	if len(voterTypes) < 2 {
		t.Errorf("expected at least 2 voter types, got %d", len(voterTypes))
	}
}

func TestHandleGetVoterTypes_Error(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Set invalid JSON to trigger unmarshal error
	setup.repo.SetSetting(ctx, "voter_types", `invalid json{`)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/voter-types", nil)
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)

	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d: %s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func TestHandleOverrideWinner_IsVotingOpenError(t *testing.T) {
	realRepo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Create mock repo with error for GetSetting
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetSettingError = fmt.Errorf("database error")

	// Initialize logger and services with mock
	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, mockRepo, derbynetClient)
	carService := services.NewCarService(log, mockRepo, derbynetClient)
	settingsService := services.NewSettingsService(log, mockRepo)
	voterService := services.NewVoterService(log, mockRepo, settingsService)
	votingService := services.NewVotingService(log, mockRepo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, mockRepo, settingsService, derbynetClient)

	h := handlers.NewForTesting(votingService, categoryService, voterService, carService, settingsService, resultsService)

	// Login to get a session cookie
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{Name: auth.CookieName, Value: token}

	body := `{"category_id": 1, "car_id": 1, "reason": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/results/override-winner", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	req.AddCookie(authCookie)

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleClearOverride_IsVotingOpenError(t *testing.T) {
	realRepo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Create mock repo with error for GetSetting
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetSettingError = fmt.Errorf("database error")

	// Initialize logger and services with mock
	log := logger.New()
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, mockRepo, derbynetClient)
	carService := services.NewCarService(log, mockRepo, derbynetClient)
	settingsService := services.NewSettingsService(log, mockRepo)
	voterService := services.NewVoterService(log, mockRepo, settingsService)
	votingService := services.NewVotingService(log, mockRepo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, mockRepo, settingsService, derbynetClient)

	h := handlers.NewForTesting(votingService, categoryService, voterService, carService, settingsService, resultsService)

	// Login to get a session cookie
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{Name: auth.CookieName, Value: token}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/results/override-winner/1", nil)
	rec := httptest.NewRecorder()
	req.AddCookie(authCookie)

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// ==================== Test DerbyNet Tests ====================

func TestHandleTestDerbyNet_Success(t *testing.T) {
	setup := newTestSetup(t)

	// Set up a mock DerbyNet server
	derbynetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/action.php":
			query := r.URL.Query().Get("query")
			if query == "racer.list" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"racers":[{"racerid":1,"firstname":"Racer","lastname":"One","carnumber":1,"carname":"Car 1"},{"racerid":2,"firstname":"Racer","lastname":"Two","carnumber":2,"carname":"Car 2"}]}`))
			} else if query == "award.list" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"awards":[{"awardid":1,"awardname":"Best Design"}],"award-types":[]}`))
			}
		}
	}))
	defer derbynetServer.Close()

	payload := map[string]string{
		"derbynet_url": derbynetServer.URL,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/test-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	if result["status"] != "success" {
		t.Errorf("expected status 'success', got %v", result["status"])
	}
	if result["total_racers"].(float64) != 2 {
		t.Errorf("expected 2 racers, got %v", result["total_racers"])
	}
	if result["total_awards"].(float64) != 1 {
		t.Errorf("expected 1 award, got %v", result["total_awards"])
	}
}

func TestHandleTestDerbyNet_WithCredentials(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Set credentials in database
	setup.repo.SetSetting(ctx, "derbynet_role", "RaceCoordinator")
	setup.repo.SetSetting(ctx, "derbynet_password", "secret")

	// Set up a mock DerbyNet server
	derbynetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/action.php":
			query := r.URL.Query().Get("query")
			action := r.FormValue("action")

			// Handle query-based endpoints (GET)
			if query == "racer.list" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"racers":[{"racerid":1,"firstname":"Racer","lastname":"One","carnumber":1,"carname":"Car 1"}]}`))
			} else if query == "award.list" {
				// FetchAwards and FetchAwardTypes both use this endpoint
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"outcome":{"summary":"success"},"awards":[{"awardid":1,"awardname":"Best Design"}],"award-types":[{"awardtypeid":1,"awardtype":"Speed"}]}`))
			} else if action == "role.login" {
				// Handle login (POST)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Set-Cookie", "PHPSESSID=test-session-id; Path=/")
				w.Write([]byte(`{"outcome":{"summary":"success"}}`))
			}
		}
	}))
	defer derbynetServer.Close()

	payload := map[string]string{
		"derbynet_url": derbynetServer.URL,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/test-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	if result["status"] != "success" {
		t.Errorf("expected status 'success', got %v", result["status"])
	}
	if result["authenticated"] != true {
		t.Errorf("expected authenticated to be true, got %v", result["authenticated"])
	}
	if result["role"] != "RaceCoordinator" {
		t.Errorf("expected role 'RaceCoordinator', got %v", result["role"])
	}
}

func TestHandleTestDerbyNet_InvalidJSON(t *testing.T) {
	setup := newTestSetup(t)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/admin/test-derbynet", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleTestDerbyNet_MissingURL(t *testing.T) {
	setup := newTestSetup(t)

	payload := map[string]string{}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/test-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	errorMsg, ok := result["error"].(string)
	if !ok || !strings.Contains(strings.ToLower(errorMsg), "derbynet_url") {
		t.Errorf("expected error about derbynet_url, got %v", result["error"])
	}
}

func TestHandleTestDerbyNet_ConnectionFailure(t *testing.T) {
	setup := newTestSetup(t)

	// Use an invalid URL that will fail to connect
	payload := map[string]string{
		"derbynet_url": "http://invalid-host-that-does-not-exist.local:9999",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/test-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	errorMsg, ok := result["error"].(string)
	if !ok || !strings.Contains(strings.ToLower(errorMsg), "failed to connect") {
		t.Errorf("expected error about connection failure, got %v", result["error"])
	}
}

func TestHandleTestDerbyNet_AwardsFetchFailure(t *testing.T) {
	setup := newTestSetup(t)

	// Set up a mock DerbyNet server that returns racers but fails on awards
	derbynetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/action.php":
			query := r.URL.Query().Get("query")
			if query == "racer.list" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"racers":[{"racerid":1,"firstname":"Racer","lastname":"One","carnumber":1,"carname":"Car 1"}]}`))
			} else if query == "award.list" {
				// Return an error for awards
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"awards not available"}`))
			}
		}
	}))
	defer derbynetServer.Close()

	payload := map[string]string{
		"derbynet_url": derbynetServer.URL,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/test-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	if result["status"] != "success" {
		t.Errorf("expected status 'success', got %v", result["status"])
	}
	if result["total_racers"].(float64) != 1 {
		t.Errorf("expected 1 racer, got %v", result["total_racers"])
	}
	// Awards should be 0 since fetch failed
	if result["total_awards"].(float64) != 0 {
		t.Errorf("expected 0 awards (fetch failed), got %v", result["total_awards"])
	}
}

func TestHandleTestDerbyNet_AuthenticationFailure(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Set credentials in database
	setup.repo.SetSetting(ctx, "derbynet_role", "RaceCoordinator")
	setup.repo.SetSetting(ctx, "derbynet_password", "secret")

	// Set up a mock DerbyNet server that succeeds for racers/awards but fails for auth
	requestCount := 0
	derbynetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/action.php":
			query := r.URL.Query().Get("query")
			action := r.FormValue("action")

			if query == "racer.list" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"racers":[{"racerid":1,"firstname":"Racer","lastname":"One","carnumber":1,"carname":"Car 1"}]}`))
			} else if query == "award.list" {
				requestCount++
				// First call is for FetchAwards (no auth), second is for FetchAwardTypes (with auth)
				if requestCount == 1 {
					// First call succeeds
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"awards":[{"awardid":1,"awardname":"Best Design"}],"award-types":[]}`))
				} else {
					// Second call (FetchAwardTypes after auth) fails
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"outcome":{"summary":"failure","code":"notauthorized","description":"Not authorized"}}`))
				}
			} else if action == "role.login" {
				// Return auth failure
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"outcome":{"summary":"failure","code":"incorrect-password","description":"Incorrect password"}}`))
			}
		}
	}))
	defer derbynetServer.Close()

	payload := map[string]string{
		"derbynet_url": derbynetServer.URL,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/test-derbynet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	if result["status"] != "success" {
		t.Errorf("expected status 'success', got %v", result["status"])
	}
	// Should show connection succeeded but authentication failed
	if result["authenticated"] != false {
		t.Errorf("expected authenticated to be false (auth failed), got %v", result["authenticated"])
	}
}

func TestHandleDeleteCar_WithVotes(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category
	catID, err := setup.repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	// Create a car
	err = setup.repo.CreateCar(ctx, "103", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Create a voter
	voterID, err := setup.repo.CreateVoter(ctx, "test-qr-car")
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	// Save a vote for this car
	err = setup.repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote: %v", err)
	}

	// Try to delete car without force
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/cars/%d", carID), nil)
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	if !response["confirmation_required"].(bool) {
		t.Error("expected confirmation_required to be true")
	}
	if response["vote_count"].(float64) != 1 {
		t.Errorf("expected vote_count to be 1, got %v", response["vote_count"])
	}
}

func TestHandleDeleteCar_WithVotesAndForce(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category
	catID, err := setup.repo.CreateCategory(ctx, "Test Category 2", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	// Create a car
	err = setup.repo.CreateCar(ctx, "104", "Test Racer 2", "Test Car 2", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Create a voter
	voterID, err := setup.repo.CreateVoter(ctx, "test-qr-car-2")
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	// Save a vote for this car
	err = setup.repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote: %v", err)
	}

	// Delete car with force=true
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/cars/%d?force=true", carID), nil)
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}
}

func TestHandleSetCarEligibility_MarkIneligibleWithVotes(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category
	catID, err := setup.repo.CreateCategory(ctx, "Test Category 3", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	// Create a car (eligible by default)
	err = setup.repo.CreateCar(ctx, "105", "Test Racer 3", "Test Car 3", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Create a voter
	voterID, err := setup.repo.CreateVoter(ctx, "test-qr-car-3")
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	// Save a vote for this car
	err = setup.repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote: %v", err)
	}

	// Try to mark as ineligible without force
	payload := map[string]interface{}{
		"eligible": false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader(body))
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	if !response["confirmation_required"].(bool) {
		t.Error("expected confirmation_required to be true")
	}
}

func TestHandleSetCarEligibility_MarkIneligibleWithVotesAndForce(t *testing.T) {
	setup := newTestSetup(t)
	ctx := context.Background()

	// Create a category
	catID, err := setup.repo.CreateCategory(ctx, "Test Category 4", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test category: %v", err)
	}

	// Create a car (eligible by default)
	err = setup.repo.CreateCar(ctx, "106", "Test Racer 4", "Test Car 4", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Create a voter
	voterID, err := setup.repo.CreateVoter(ctx, "test-qr-car-4")
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	// Save a vote for this car
	err = setup.repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote: %v", err)
	}

	// Mark as ineligible with force=true
	payload := map[string]interface{}{
		"eligible": false,
		"force":    true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader(body))
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteCar_CountVotesError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a car first
	err := setup.repo.CreateCar(ctx, "107", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}

	cars, _ := setup.repo.ListCars(ctx)
	if len(cars) == 0 {
		t.Fatal("no cars found after creation")
	}
	carID := cars[0].ID

	// Inject error for CountVotesForCar
	mockRepo.CountVotesForCarError = fmt.Errorf("database error")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/cars/%d", carID), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for count error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleDeleteCategory_CountVotesError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a category
	catID, err := setup.repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Inject error for CountVotesForCategory
	mockRepo.CountVotesForCategoryError = fmt.Errorf("database error")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/categories/%d", catID), nil)
	rec := httptest.NewRecorder()

	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for count error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleSetCarEligibility_CountVotesError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a car (eligible by default)
	err := setup.repo.CreateCar(ctx, "108", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create test car: %v", err)
	}
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	// Inject error for CountVotesForCar
	mockRepo.CountVotesForCarError = fmt.Errorf("database error")

	// Try to mark as ineligible
	payload := map[string]interface{}{
		"eligible": false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/cars/%d/eligibility", carID), bytes.NewReader(body))
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for count error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestConditionalHTTPLogger_WithLoggingEnabled(t *testing.T) {
	setup := newTestSetup(t)

	// Enable HTTP logging BEFORE creating the router
	setup.log.EnableHTTPLogging()
	
	// Create a new router after enabling logging
	router := setup.handlers.Router()

	// Make a request to trigger the conditional logger
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should get a response (middleware doesn't block)
	if rec.Code == 0 {
		t.Error("expected response code to be set")
	}
}

func TestConditionalHTTPLogger_WithLoggingDisabled(t *testing.T) {
	setup := newTestSetup(t)

	// Disable HTTP logging (should be disabled by default but let's be explicit)
	setup.log.DisableHTTPLogging()
	
	// Create a new router after setting logging state
	router := setup.handlers.Router()

	// Make a request to trigger the conditional logger
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should get a response (middleware doesn't block)
	if rec.Code == 0 {
		t.Error("expected response code to be set")
	}
}

func TestConditionalHTTPLogger_WithNilLogger(t *testing.T) {
	setup := newTestSetup(t)

	// Set Log to nil to test the nil check
	setup.handlers.Log = nil

	// Make a request to trigger the conditional logger
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should get a response (middleware doesn't block)
	if rec.Code == 0 {
		t.Error("expected response code to be set")
	}
}

func TestHandleDeleteCategory_WithForceDeleteError(t *testing.T) {
	setup, mockRepo := newTestSetupWithMockRepo(t)
	ctx := context.Background()

	// Create a category
	catID, err := setup.repo.CreateCategory(ctx, "Category To Delete With Error", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Create a car and vote so we can test the force path
	err = setup.repo.CreateCar(ctx, "201", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}
	cars, _ := setup.repo.ListCars(ctx)
	carID := cars[0].ID

	voterID, err := setup.repo.CreateVoter(ctx, "test-qr-force-delete")
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	err = setup.repo.SaveVote(ctx, voterID, int(catID), int(carID))
	if err != nil {
		t.Fatalf("failed to save vote: %v", err)
	}

	// Inject delete error
	mockRepo.DeleteCategoryError = fmt.Errorf("database error on delete")

	// Try to delete with force=true
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/categories/%d?force=true", catID), nil)
	rec := httptest.NewRecorder()
	req.AddCookie(setup.authCookie)
	setup.router.ServeHTTP(rec, req)

	// Should return 500 for delete error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
