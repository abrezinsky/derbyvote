package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/internal/repository/mock"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/testutil"
)

func TestSettingsService_VotingOpen(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Default should be open (true)
	open, err := svc.IsVotingOpen(ctx)
	if err != nil {
		t.Fatalf("IsVotingOpen failed: %v", err)
	}
	if !open {
		t.Error("expected voting to be open by default")
	}

	// Close voting
	err = svc.SetVotingOpen(ctx, false)
	if err != nil {
		t.Fatalf("SetVotingOpen(false) failed: %v", err)
	}

	// Verify closed
	open, err = svc.IsVotingOpen(ctx)
	if err != nil {
		t.Fatalf("IsVotingOpen failed: %v", err)
	}
	if open {
		t.Error("expected voting to be closed")
	}

	// Reopen voting
	err = svc.SetVotingOpen(ctx, true)
	if err != nil {
		t.Fatalf("SetVotingOpen(true) failed: %v", err)
	}

	// Verify open
	open, err = svc.IsVotingOpen(ctx)
	if err != nil {
		t.Fatalf("IsVotingOpen failed: %v", err)
	}
	if !open {
		t.Error("expected voting to be open again")
	}
}

func TestSettingsService_DerbyNetURL(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	testURL := "http://localhost:9000/derbynet"

	// Set URL
	err := svc.SetDerbyNetURL(ctx, testURL)
	if err != nil {
		t.Fatalf("SetDerbyNetURL failed: %v", err)
	}

	// Get URL
	url, err := svc.GetDerbyNetURL(ctx)
	if err != nil {
		t.Fatalf("GetDerbyNetURL failed: %v", err)
	}
	if url != testURL {
		t.Errorf("expected URL %q, got %q", testURL, url)
	}
}

func TestSettingsService_BaseURL(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Default should be empty (app.go sets it with detected IP on startup)
	url, err := svc.GetBaseURL(ctx)
	if err != nil {
		t.Fatalf("GetBaseURL failed: %v", err)
	}
	if url != "" {
		t.Errorf("expected empty default URL, got %q", url)
	}

	// Set custom URL
	customURL := "https://voting.example.com"
	err = svc.SetBaseURL(ctx, customURL)
	if err != nil {
		t.Fatalf("SetBaseURL failed: %v", err)
	}

	// Verify custom URL
	url, err = svc.GetBaseURL(ctx)
	if err != nil {
		t.Fatalf("GetBaseURL failed: %v", err)
	}
	if url != customURL {
		t.Errorf("expected URL %q, got %q", customURL, url)
	}
}

func TestSettingsService_StartVotingTimer_InvalidMinutes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Test invalid minutes
	testCases := []int{0, -1, 61, 100}
	for _, minutes := range testCases {
		_, err := svc.StartVotingTimer(ctx, minutes)
		if err == nil {
			t.Errorf("expected error for %d minutes, got nil", minutes)
		}
	}
}

func TestSettingsService_ResetTables_InvalidTable(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Try to reset an invalid table
	_, err := svc.ResetTables(ctx, []string{"invalid_table"})
	if err == nil {
		t.Error("expected error for invalid table, got nil")
	}

	// Verify it's an InvalidTableError
	if _, ok := err.(*services.InvalidTableError); !ok {
		t.Errorf("expected InvalidTableError, got %T", err)
	}
}

func TestSettingsService_ResetTables_EmptyList(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Try to reset with empty list
	_, err := svc.ResetTables(ctx, []string{})
	if err == nil {
		t.Error("expected error for empty table list, got nil")
	}
}

func TestSettingsService_SetBroadcaster(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)

	// Create a mock broadcaster
	mock := &mockBroadcaster{}
	svc.SetBroadcaster(mock)

	// Trigger a broadcast via OpenVoting
	ctx := context.Background()
	err := svc.OpenVoting(ctx)
	if err != nil {
		t.Fatalf("OpenVoting failed: %v", err)
	}

	// Verify broadcast was called
	if !mock.called {
		t.Error("expected broadcaster to be called")
	}
	if !mock.lastOpen {
		t.Error("expected lastOpen to be true")
	}
}

type mockBroadcaster struct {
	called     bool
	lastOpen   bool
	lastCloseTime string
}

func (m *mockBroadcaster) BroadcastVotingStatus(open bool, closeTime string) {
	m.called = true
	m.lastOpen = open
	m.lastCloseTime = closeTime
}

func TestSettingsService_GetSetSetting(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set an arbitrary setting
	err := svc.SetSetting(ctx, "custom_key", "custom_value")
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	// Get the setting
	value, err := svc.GetSetting(ctx, "custom_key")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if value != "custom_value" {
		t.Errorf("expected 'custom_value', got %q", value)
	}
}

func TestSettingsService_TimerEndTime(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Default should be 0
	endTime, err := svc.GetTimerEndTime(ctx)
	if err != nil {
		t.Fatalf("GetTimerEndTime failed: %v", err)
	}
	if endTime != 0 {
		t.Errorf("expected 0, got %d", endTime)
	}

	// Set timer end time
	testTime := int64(1700000000)
	err = svc.SetTimerEndTime(ctx, testTime)
	if err != nil {
		t.Fatalf("SetTimerEndTime failed: %v", err)
	}

	// Verify it was set
	endTime, err = svc.GetTimerEndTime(ctx)
	if err != nil {
		t.Fatalf("GetTimerEndTime failed: %v", err)
	}
	if endTime != testTime {
		t.Errorf("expected %d, got %d", testTime, endTime)
	}

	// Clear timer
	err = svc.ClearTimer(ctx)
	if err != nil {
		t.Fatalf("ClearTimer failed: %v", err)
	}

	// Verify it was cleared
	endTime, err = svc.GetTimerEndTime(ctx)
	if err != nil {
		t.Fatalf("GetTimerEndTime failed: %v", err)
	}
	if endTime != 0 {
		t.Errorf("expected 0 after clear, got %d", endTime)
	}
}

func TestSettingsService_AllSettings(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set some values
	svc.SetDerbyNetURL(ctx, "http://derbynet.local")
	svc.SetBaseURL(ctx, "http://voting.local")
	svc.SetVotingOpen(ctx, false)

	// Get all settings
	settings, err := svc.AllSettings(ctx)
	if err != nil {
		t.Fatalf("AllSettings failed: %v", err)
	}

	if settings["voting_open"] != false {
		t.Errorf("expected voting_open=false, got %v", settings["voting_open"])
	}
	if settings["derbynet_url"] != "http://derbynet.local" {
		t.Errorf("expected derbynet_url, got %v", settings["derbynet_url"])
	}
	if settings["base_url"] != "http://voting.local" {
		t.Errorf("expected base_url, got %v", settings["base_url"])
	}
	if _, ok := settings["timer_end"]; !ok {
		t.Error("expected timer_end key in settings")
	}
}

func TestSettingsService_UpdateSettings(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Update settings
	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetURL: "http://new-derbynet.local",
		BaseURL:     "http://new-voting.local",
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify
	url, _ := svc.GetDerbyNetURL(ctx)
	if url != "http://new-derbynet.local" {
		t.Errorf("expected new derbynet URL, got %q", url)
	}

	baseURL, _ := svc.GetBaseURL(ctx)
	if baseURL != "http://new-voting.local" {
		t.Errorf("expected new base URL, got %q", baseURL)
	}
}

func TestSettingsService_UpdateSettings_Partial(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set initial values
	svc.SetDerbyNetURL(ctx, "http://original.local")
	svc.SetBaseURL(ctx, "http://original-base.local")

	// Update only derbynet URL
	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetURL: "http://updated.local",
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify derbynet URL changed
	url, _ := svc.GetDerbyNetURL(ctx)
	if url != "http://updated.local" {
		t.Errorf("expected updated derbynet URL, got %q", url)
	}

	// Verify base URL unchanged
	baseURL, _ := svc.GetBaseURL(ctx)
	if baseURL != "http://original-base.local" {
		t.Errorf("expected original base URL, got %q", baseURL)
	}
}

func TestSettingsService_OpenVoting(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Start with voting closed
	svc.SetVotingOpen(ctx, false)

	// Open voting
	err := svc.OpenVoting(ctx)
	if err != nil {
		t.Fatalf("OpenVoting failed: %v", err)
	}

	// Verify
	open, _ := svc.IsVotingOpen(ctx)
	if !open {
		t.Error("expected voting to be open")
	}
}

func TestSettingsService_CloseVoting(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Start with voting open and a timer
	svc.SetVotingOpen(ctx, true)
	svc.SetTimerEndTime(ctx, 1700000000)

	// Close voting
	err := svc.CloseVoting(ctx)
	if err != nil {
		t.Fatalf("CloseVoting failed: %v", err)
	}

	// Verify voting closed
	open, _ := svc.IsVotingOpen(ctx)
	if open {
		t.Error("expected voting to be closed")
	}

	// Verify timer cleared
	endTime, _ := svc.GetTimerEndTime(ctx)
	if endTime != 0 {
		t.Errorf("expected timer to be cleared, got %d", endTime)
	}
}

func TestSettingsService_StartVotingTimer_Valid(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Start with voting closed
	svc.SetVotingOpen(ctx, false)

	// Start timer
	closeTime, err := svc.StartVotingTimer(ctx, 5)
	if err != nil {
		t.Fatalf("StartVotingTimer failed: %v", err)
	}

	// Should return a close time string
	if closeTime == "" {
		t.Error("expected close time to be set")
	}

	// Verify voting is now open
	open, _ := svc.IsVotingOpen(ctx)
	if !open {
		t.Error("expected voting to be open after starting timer")
	}
}

func TestSettingsService_ResetTables_ValidTables(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Reset valid tables
	result, err := svc.ResetTables(ctx, []string{"votes", "voters"})
	if err != nil {
		t.Fatalf("ResetTables failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Should contain both tables
	if len(result.Tables) < 2 {
		t.Errorf("expected at least 2 tables, got %d", len(result.Tables))
	}
}

func TestSettingsService_ResetTables_AutoAddsVotes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Reset cars should auto-add votes
	result, err := svc.ResetTables(ctx, []string{"cars"})
	if err != nil {
		t.Fatalf("ResetTables failed: %v", err)
	}

	// Should have votes first, then cars
	if len(result.Tables) != 2 {
		t.Fatalf("expected 2 tables (votes + cars), got %d", len(result.Tables))
	}
	if result.Tables[0] != "votes" {
		t.Errorf("expected first table to be 'votes', got %q", result.Tables[0])
	}
}

func TestSettingsService_UpdateSettings_OnlyBaseURL(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set initial values
	svc.SetDerbyNetURL(ctx, "http://original.local")
	svc.SetBaseURL(ctx, "http://original-base.local")

	// Update only base URL
	err := svc.UpdateSettings(ctx, services.Settings{
		BaseURL: "http://updated-base.local",
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify derbynet URL unchanged
	url, _ := svc.GetDerbyNetURL(ctx)
	if url != "http://original.local" {
		t.Errorf("expected original derbynet URL, got %q", url)
	}

	// Verify base URL changed
	baseURL, _ := svc.GetBaseURL(ctx)
	if baseURL != "http://updated-base.local" {
		t.Errorf("expected updated base URL, got %q", baseURL)
	}
}

func TestSettingsService_CloseVoting_WithBroadcaster(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set up mock broadcaster
	mock := &mockBroadcaster{}
	svc.SetBroadcaster(mock)

	// Open voting first
	svc.SetVotingOpen(ctx, true)

	// Close voting
	err := svc.CloseVoting(ctx)
	if err != nil {
		t.Fatalf("CloseVoting failed: %v", err)
	}

	// Verify broadcaster was called
	if !mock.called {
		t.Error("expected broadcaster to be called")
	}
	if mock.lastOpen {
		t.Error("expected lastOpen to be false for CloseVoting")
	}
}

func TestSettingsService_ResetTables_ActuallyDeletesData(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Create some cars
	err := repo.CreateCar(ctx, "101", "Driver 1", "Test Car 1", "http://example.com/1.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}
	err = repo.CreateCar(ctx, "102", "Driver 2", "Test Car 2", "http://example.com/2.jpg")
	if err != nil {
		t.Fatalf("CreateCar failed: %v", err)
	}

	// Verify cars exist
	cars, err := repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars failed: %v", err)
	}
	if len(cars) != 2 {
		t.Fatalf("expected 2 cars before reset, got %d", len(cars))
	}

	// Reset cars table (should also reset votes)
	result, err := svc.ResetTables(ctx, []string{"cars"})
	if err != nil {
		t.Fatalf("ResetTables failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify cars are deleted
	cars, err = repo.ListCars(ctx)
	if err != nil {
		t.Fatalf("ListCars after reset failed: %v", err)
	}
	if len(cars) != 0 {
		t.Errorf("expected 0 cars after reset, got %d", len(cars))
	}
}

func TestSettingsService_ResetTables_DeletesVoters(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Create some voters
	_, err := repo.CreateVoter(ctx, "VOTER-001")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}
	_, err = repo.CreateVoter(ctx, "VOTER-002")
	if err != nil {
		t.Fatalf("CreateVoter failed: %v", err)
	}

	// Verify voters exist
	voters, err := repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}
	if len(voters) != 2 {
		t.Fatalf("expected 2 voters before reset, got %d", len(voters))
	}

	// Reset voters table (should also reset votes)
	result, err := svc.ResetTables(ctx, []string{"voters"})
	if err != nil {
		t.Fatalf("ResetTables failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify voters are deleted
	voters, err = repo.ListVoters(ctx)
	if err != nil {
		t.Fatalf("ListVoters after reset failed: %v", err)
	}
	if len(voters) != 0 {
		t.Errorf("expected 0 voters after reset, got %d", len(voters))
	}
}

func TestSettingsService_ResetTables_DeletesCategories(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Create some categories
	_, err := repo.CreateCategory(ctx, "Test Category 1", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	_, err = repo.CreateCategory(ctx, "Test Category 2", 2, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Verify categories exist
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("expected 2 categories before reset, got %d", len(categories))
	}

	// Reset categories table (should also reset votes)
	result, err := svc.ResetTables(ctx, []string{"categories"})
	if err != nil {
		t.Fatalf("ResetTables failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify categories are deleted
	categories, err = repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories after reset failed: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("expected 0 categories after reset, got %d", len(categories))
	}
}

func TestSettingsService_UpdateSettings_AllFields(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	requireReg := true

	// Update all settings at once
	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetURL:         "http://derbynet-all.local",
		BaseURL:             "http://voting-all.local",
		DerbyNetRole:        "admin",
		DerbyNetPassword:    "password123",
		RequireRegisteredQR: &requireReg,
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify all fields
	url, _ := svc.GetDerbyNetURL(ctx)
	if url != "http://derbynet-all.local" {
		t.Errorf("expected derbynet URL, got %q", url)
	}

	baseURL, _ := svc.GetBaseURL(ctx)
	if baseURL != "http://voting-all.local" {
		t.Errorf("expected base URL, got %q", baseURL)
	}

	role, _ := svc.GetSetting(ctx, "derbynet_role")
	if role != "admin" {
		t.Errorf("expected derbynet_role=admin, got %q", role)
	}

	password, _ := svc.GetSetting(ctx, "derbynet_password")
	if password != "password123" {
		t.Errorf("expected password, got %q", password)
	}

	requireQR, _ := svc.RequireRegisteredQR(ctx)
	if !requireQR {
		t.Error("expected RequireRegisteredQR to be true")
	}
}

func TestSettingsService_UpdateSettings_OnlyDerbyNetRole(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set initial URL
	svc.SetDerbyNetURL(ctx, "http://original.local")

	// Update only role
	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetRole: "viewer",
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify role changed
	role, _ := svc.GetSetting(ctx, "derbynet_role")
	if role != "viewer" {
		t.Errorf("expected derbynet_role=viewer, got %q", role)
	}

	// Verify URL unchanged
	url, _ := svc.GetDerbyNetURL(ctx)
	if url != "http://original.local" {
		t.Errorf("expected original URL, got %q", url)
	}
}

func TestSettingsService_UpdateSettings_OnlyDerbyNetPassword(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Update only password
	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetPassword: "newpass456",
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify password changed
	password, _ := svc.GetSetting(ctx, "derbynet_password")
	if password != "newpass456" {
		t.Errorf("expected password, got %q", password)
	}
}

func TestSettingsService_UpdateSettings_OnlyRequireRegisteredQR(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	requireReg := true

	// Update only RequireRegisteredQR
	err := svc.UpdateSettings(ctx, services.Settings{
		RequireRegisteredQR: &requireReg,
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify it changed
	requireQR, _ := svc.RequireRegisteredQR(ctx)
	if !requireQR {
		t.Error("expected RequireRegisteredQR to be true")
	}
}

func TestSettingsService_RequireRegisteredQR_Default(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Default should be false
	require, err := svc.RequireRegisteredQR(ctx)
	if err != nil {
		t.Fatalf("RequireRegisteredQR failed: %v", err)
	}
	if require {
		t.Error("expected default RequireRegisteredQR to be false")
	}
}

func TestSettingsService_RequireRegisteredQR_Toggle(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set to true
	err := svc.SetRequireRegisteredQR(ctx, true)
	if err != nil {
		t.Fatalf("SetRequireRegisteredQR(true) failed: %v", err)
	}

	require, _ := svc.RequireRegisteredQR(ctx)
	if !require {
		t.Error("expected RequireRegisteredQR to be true")
	}

	// Set back to false
	err = svc.SetRequireRegisteredQR(ctx, false)
	if err != nil {
		t.Fatalf("SetRequireRegisteredQR(false) failed: %v", err)
	}

	require, _ = svc.RequireRegisteredQR(ctx)
	if require {
		t.Error("expected RequireRegisteredQR to be false")
	}
}

func TestSettingsService_GetTimerEndTime_InvalidValue(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set an invalid timer value
	repo.SetSetting(ctx, "timer_end", "not-a-number")

	// Should return 0 on parse error
	endTime, err := svc.GetTimerEndTime(ctx)
	if err != nil {
		t.Fatalf("GetTimerEndTime failed: %v", err)
	}
	if endTime != 0 {
		t.Errorf("expected 0 for invalid value, got %d", endTime)
	}
}

// ===== Error Path Tests =====

func TestSettingsService_OpenVoting_SetError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.SetSettingError = errors.New("database error")

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	err := svc.OpenVoting(ctx)
	if err == nil {
		t.Fatal("expected error from OpenVoting, got nil")
	}
}

func TestSettingsService_CloseVoting_SetError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.SetSettingError = errors.New("database error")

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	err := svc.CloseVoting(ctx)
	if err == nil {
		t.Fatal("expected error from CloseVoting, got nil")
	}
}

func TestSettingsService_StartVotingTimer_SetCloseTimeError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.SetSettingError = errors.New("database error")

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	_, err := svc.StartVotingTimer(ctx, 5)
	if err == nil {
		t.Fatal("expected error from StartVotingTimer, got nil")
	}
}

func TestSettingsService_StartVotingTimer_SetVotingOpenError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	// Fail on second SetSetting call (voting_open)
	callCount := 0
	originalSetSetting := mockRepo.FullRepository.SetSetting
	mockRepo.FullRepository = &mockSettingsRepo{
		FullRepository: mockRepo.FullRepository,
		setSetting: func(ctx context.Context, key, value string) error {
			callCount++
			if callCount > 1 {
				return errors.New("database error")
			}
			return originalSetSetting(ctx, key, value)
		},
	}

	_, err := svc.StartVotingTimer(ctx, 5)
	if err == nil {
		t.Fatal("expected error from StartVotingTimer on SetVotingOpen, got nil")
	}
}

func TestSettingsService_UpdateSettings_DerbyNetURLError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.SetSettingError = errors.New("database error")

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetURL: "http://test.local",
	})
	if err == nil {
		t.Fatal("expected error from UpdateSettings, got nil")
	}
}

func TestSettingsService_UpdateSettings_BaseURLError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	// First setting succeeds, second fails
	callCount := 0
	originalSetSetting := mockRepo.FullRepository.SetSetting
	mockRepo.FullRepository = &mockSettingsRepo{
		FullRepository: mockRepo.FullRepository,
		setSetting: func(ctx context.Context, key, value string) error {
			callCount++
			if callCount > 1 {
				return errors.New("database error")
			}
			return originalSetSetting(ctx, key, value)
		},
	}

	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetURL: "http://test.local",
		BaseURL:     "http://base.local",
	})
	if err == nil {
		t.Fatal("expected error from UpdateSettings, got nil")
	}
}

func TestSettingsService_UpdateSettings_DerbyNetRoleError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	// Fail on setting derbynet_role
	callCount := 0
	originalSetSetting := mockRepo.FullRepository.SetSetting
	mockRepo.FullRepository = &mockSettingsRepo{
		FullRepository: mockRepo.FullRepository,
		setSetting: func(ctx context.Context, key, value string) error {
			callCount++
			if key == "derbynet_role" {
				return errors.New("database error")
			}
			return originalSetSetting(ctx, key, value)
		},
	}

	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetRole: "admin",
	})
	if err == nil {
		t.Fatal("expected error from UpdateSettings, got nil")
	}
}

func TestSettingsService_UpdateSettings_DerbyNetPasswordError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	// Fail on setting derbynet_password
	originalSetSetting := mockRepo.FullRepository.SetSetting
	mockRepo.FullRepository = &mockSettingsRepo{
		FullRepository: mockRepo.FullRepository,
		setSetting: func(ctx context.Context, key, value string) error {
			if key == "derbynet_password" {
				return errors.New("database error")
			}
			return originalSetSetting(ctx, key, value)
		},
	}

	err := svc.UpdateSettings(ctx, services.Settings{
		DerbyNetPassword: "secret",
	})
	if err == nil {
		t.Fatal("expected error from UpdateSettings, got nil")
	}
}

func TestSettingsService_UpdateSettings_RequireRegisteredQRError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	// Fail on setting require_registered_qr
	originalSetSetting := mockRepo.FullRepository.SetSetting
	mockRepo.FullRepository = &mockSettingsRepo{
		FullRepository: mockRepo.FullRepository,
		setSetting: func(ctx context.Context, key, value string) error {
			if key == "require_registered_qr" {
				return errors.New("database error")
			}
			return originalSetSetting(ctx, key, value)
		},
	}

	requireReg := true
	err := svc.UpdateSettings(ctx, services.Settings{
		RequireRegisteredQR: &requireReg,
	})
	if err == nil {
		t.Fatal("expected error from UpdateSettings, got nil")
	}
}

func TestSettingsService_UpdateSettings_OnlyVotingInstructions(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	instructions := "Please vote carefully!\nEach vote counts."
	err := svc.UpdateSettings(ctx, services.Settings{
		VotingInstructions: instructions,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify it was saved
	saved, err := repo.GetSetting(ctx, "voting_instructions")
	if err != nil {
		t.Fatalf("failed to get voting_instructions: %v", err)
	}

	if saved != instructions {
		t.Errorf("expected instructions '%s', got '%s'", instructions, saved)
	}
}

func TestSettingsService_UpdateSettings_VotingInstructionsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	// Fail on setting voting_instructions
	originalSetSetting := mockRepo.FullRepository.SetSetting
	mockRepo.FullRepository = &mockSettingsRepo{
		FullRepository: mockRepo.FullRepository,
		setSetting: func(ctx context.Context, key, value string) error {
			if key == "voting_instructions" {
				return errors.New("database error")
			}
			return originalSetSetting(ctx, key, value)
		},
	}

	err := svc.UpdateSettings(ctx, services.Settings{
		VotingInstructions: "Some instructions",
	})
	if err == nil {
		t.Fatal("expected error from UpdateSettings, got nil")
	}
}

// mockSettingsRepo wraps a repository to inject specific errors
type mockSettingsRepo struct {
	repository.FullRepository
	setSetting func(ctx context.Context, key, value string) error
}

func (m *mockSettingsRepo) SetSetting(ctx context.Context, key, value string) error {
	if m.setSetting != nil {
		return m.setSetting(ctx, key, value)
	}
	return m.FullRepository.SetSetting(ctx, key, value)
}

func TestSettingsService_ResetTables_ClearTableError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	// Create some cars
	_ = realRepo.CreateCar(ctx, "101", "Driver 1", "Test Car 1", "")

	// Configure mock to fail on ClearTable
	mockRepo.ClearTableError = errors.New("database error clearing table")

	_, err := svc.ResetTables(ctx, []string{"cars"})
	if err == nil {
		t.Fatal("expected error when ClearTable fails, got nil")
	}
}
func TestSettingsService_IsVotingOpen_NotFound(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	// Inject ErrNotFound to simulate setting doesn't exist
	mockRepo.GetSettingError = repository.ErrNotFound

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	open, err := svc.IsVotingOpen(ctx)
	if err != nil {
		t.Fatalf("expected no error when setting not found, got: %v", err)
	}
	if !open {
		t.Error("expected voting to default to open when setting doesn't exist")
	}
}

func TestSettingsService_IsVotingOpen_DatabaseError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetSettingError = errors.New("database connection lost")

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	_, err := svc.IsVotingOpen(ctx)
	if err == nil {
		t.Fatal("expected error when GetSetting fails with database error, got nil")
	}
	if err.Error() != "database connection lost" {
		t.Errorf("expected 'database connection lost', got: %v", err)
	}
}

func TestSettingsService_GetTimerEndTime_DatabaseError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.GetSettingError = errors.New("database connection lost")

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	_, err := svc.GetTimerEndTime(ctx)
	if err == nil {
		t.Fatal("expected error when GetSetting fails with database error, got nil")
	}
	if err.Error() != "database connection lost" {
		t.Errorf("expected 'database connection lost', got: %v", err)
	}
}

// ==================== Voter Types Tests ====================

func TestGetVoterTypes_Default(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	voterTypes, err := svc.GetVoterTypes(ctx)
	if err != nil {
		t.Fatalf("GetVoterTypes failed: %v", err)
	}

	// Should return defaults: general, racer, Race Committee, Cubmaster
	expected := []string{"general", "racer", "Race Committee", "Cubmaster"}
	if len(voterTypes) != len(expected) {
		t.Fatalf("expected %d voter types, got %d", len(expected), len(voterTypes))
	}
	for i, vt := range expected {
		if voterTypes[i] != vt {
			t.Errorf("expected voter type[%d] to be %s, got %s", i, vt, voterTypes[i])
		}
	}
}

func TestGetVoterTypes_CustomTypes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set custom voter types
	customTypes := []string{"general", "racer", "Leader", "Parent"}
	err := svc.SetVoterTypes(ctx, customTypes)
	if err != nil {
		t.Fatalf("SetVoterTypes failed: %v", err)
	}

	// Get them back
	voterTypes, err := svc.GetVoterTypes(ctx)
	if err != nil {
		t.Fatalf("GetVoterTypes failed: %v", err)
	}

	if len(voterTypes) != len(customTypes) {
		t.Fatalf("expected %d voter types, got %d", len(customTypes), len(voterTypes))
	}
	for i, vt := range customTypes {
		if voterTypes[i] != vt {
			t.Errorf("expected voter type[%d] to be %s, got %s", i, vt, voterTypes[i])
		}
	}
}

func TestGetVoterTypes_EnsuresRequiredTypes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Try to set without required types
	incompleteTypes := []string{"Leader", "Parent"}
	err := svc.SetVoterTypes(ctx, incompleteTypes)
	if err != nil {
		t.Fatalf("SetVoterTypes failed: %v", err)
	}

	// Check what was actually stored in the database
	storedJSON, err := svc.GetSetting(ctx, "voter_types")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	t.Logf("Stored JSON: %s", storedJSON)

	// Try getting the setting directly first
	rawSetting, _ := repo.GetSetting(ctx, "voter_types")
	t.Logf("Raw setting from repo: %s", rawSetting)

	// Get them back - should have general and racer added
	voterTypes, err := svc.GetVoterTypes(ctx)
	if err != nil {
		t.Fatalf("GetVoterTypes failed: %v", err)
	}
	t.Logf("Retrieved voter types: %v", voterTypes)

	// Verify we have at least 4 types (general, racer, Leader, Parent)
	if len(voterTypes) < 4 {
		t.Fatalf("expected at least 4 voter types, got %d: %v", len(voterTypes), voterTypes)
	}

	hasGeneral := false
	hasRacer := false
	hasLeader := false
	hasParent := false
	for _, vt := range voterTypes {
		if vt == "general" {
			hasGeneral = true
		}
		if vt == "racer" {
			hasRacer = true
		}
		if vt == "Leader" {
			hasLeader = true
		}
		if vt == "Parent" {
			hasParent = true
		}
	}

	if !hasGeneral {
		t.Errorf("expected 'general' to be in voter types, got: %v", voterTypes)
	}
	if !hasRacer {
		t.Errorf("expected 'racer' to be in voter types, got: %v", voterTypes)
	}
	if !hasLeader {
		t.Errorf("expected 'Leader' to be in voter types, got: %v", voterTypes)
	}
	if !hasParent {
		t.Errorf("expected 'Parent' to be in voter types, got: %v", voterTypes)
	}
}

func TestSetVoterTypes_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	voterTypes := []string{"general", "racer", "Custom Type"}
	err := svc.SetVoterTypes(ctx, voterTypes)
	if err != nil {
		t.Fatalf("SetVoterTypes failed: %v", err)
	}

	// Verify we can retrieve them
	retrieved, err := svc.GetVoterTypes(ctx)
	if err != nil {
		t.Fatalf("GetVoterTypes failed: %v", err)
	}

	if len(retrieved) != len(voterTypes) {
		t.Errorf("expected %d voter types, got %d", len(voterTypes), len(retrieved))
	}
	for i, vt := range voterTypes {
		if retrieved[i] != vt {
			t.Errorf("expected voter type[%d] to be %s, got %s", i, vt, retrieved[i])
		}
	}
}

// TestUpdateSettings_OnlyVoterTypes tests UpdateSettings with only VoterTypes field
func TestUpdateSettings_OnlyVoterTypes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	settings := services.Settings{
		VoterTypes: []string{"general", "racer", "Committee Member"},
	}

	err := svc.UpdateSettings(ctx, settings)
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify voter types were set
	voterTypes, err := svc.GetVoterTypes(ctx)
	if err != nil {
		t.Fatalf("GetVoterTypes failed: %v", err)
	}

	if len(voterTypes) != 3 {
		t.Errorf("expected 3 voter types, got %d", len(voterTypes))
	}

	hasCommittee := false
	for _, vt := range voterTypes {
		if vt == "Committee Member" {
			hasCommittee = true
			break
		}
	}
	if !hasCommittee {
		t.Error("expected 'Committee Member' to be in voter types")
	}
}

// TestUpdateSettings_VoterTypesError tests UpdateSettings error path when SetVoterTypes fails
func TestUpdateSettings_VoterTypesError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.SetSettingError = errors.New("database error")

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	settings := services.Settings{
		VoterTypes: []string{"general", "racer", "New Type"},
	}

	err := svc.UpdateSettings(ctx, settings)
	if err == nil {
		t.Fatal("expected error from UpdateSettings when SetVoterTypes fails, got nil")
	}
}

// TestSetVoterTypes_SetSettingError tests SetVoterTypes error path when SetSetting fails
func TestSetVoterTypes_SetSettingError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	mockRepo.SetSettingError = errors.New("database error")

	log := logger.New()
	svc := services.NewSettingsService(log, mockRepo)
	ctx := context.Background()

	err := svc.SetVoterTypes(ctx, []string{"general", "racer", "Test"})
	if err == nil {
		t.Fatal("expected error from SetVoterTypes when SetSetting fails, got nil")
	}
}

// TestGetVoterTypes_UnmarshalError tests GetVoterTypes with invalid JSON
func TestGetVoterTypes_UnmarshalError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewSettingsService(log, repo)
	ctx := context.Background()

	// Set invalid JSON for voter_types
	err := repo.SetSetting(ctx, "voter_types", `{invalid json`)
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	// GetVoterTypes should return error
	_, err = svc.GetVoterTypes(ctx)
	if err == nil {
		t.Fatal("expected error from GetVoterTypes with invalid JSON, got nil")
	}
}
