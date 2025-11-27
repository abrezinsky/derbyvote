package services_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/internal/repository/mock"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/testutil"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

func TestCategoryService_CreateCategory(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a category
	cat := services.Category{
		Name:         "Best Design",
		DisplayOrder: 1,
		Active:       true,
	}

	id, err := svc.CreateCategory(ctx, cat)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Verify it was created
	categories, err := svc.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].Name != "Best Design" {
		t.Errorf("expected name 'Best Design', got %q", categories[0].Name)
	}
}

func TestCategoryService_UpdateCategory(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a category
	id, err := svc.CreateCategory(ctx, services.Category{
		Name:         "Original Name",
		DisplayOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Update it
	err = svc.UpdateCategory(ctx, int(id), services.Category{
		Name:         "Updated Name",
		DisplayOrder: 2,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("UpdateCategory failed: %v", err)
	}

	// Verify the update
	categories, err := svc.ListCategories(ctx)
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

func TestCategoryService_DeleteCategory(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a category
	id, err := svc.CreateCategory(ctx, services.Category{
		Name:         "To Be Deleted",
		DisplayOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Delete it
	err = svc.DeleteCategory(ctx, int(id))
	if err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}

	// Verify it's gone from active list
	categories, err := svc.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("expected 0 categories after delete, got %d", len(categories))
	}
}

func TestCategoryService_SeedMockCategories(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// First seed should add categories
	count1, err := svc.SeedMockCategories(ctx)
	if err != nil {
		t.Fatalf("SeedMockCategories failed: %v", err)
	}
	if count1 == 0 {
		t.Error("expected some categories to be seeded")
	}

	// Second seed should add nothing (already exist)
	count2, err := svc.SeedMockCategories(ctx)
	if err != nil {
		t.Fatalf("SeedMockCategories second call failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 new categories on second seed, got %d", count2)
	}
}

func TestCategoryService_SeedMockCategories_WithDBError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Close the database to force errors
	repo.DB().Close()

	// Should return error when DB operations fail
	count, err := svc.SeedMockCategories(ctx)
	if err == nil {
		t.Fatal("expected error when database is closed, got nil")
	}
	if count != 0 {
		t.Errorf("expected 0 categories added when DB is closed, got %d", count)
	}
}

func TestCategoryService_SeedMockCategories_PartialSuccess(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Seed first, then close DB to cause errors on second seed attempt
	count1, err := svc.SeedMockCategories(ctx)
	if err != nil {
		t.Fatalf("First SeedMockCategories failed: %v", err)
	}
	if count1 == 0 {
		t.Error("expected some categories to be seeded")
	}

	// Now close the database and try to seed again
	// Categories already exist, so CategoryExists will be called but fail
	repo.DB().Close()

	count2, err := svc.SeedMockCategories(ctx)
	if err == nil {
		t.Error("expected error when checking existing categories with closed DB")
	}
	if count2 != 0 {
		t.Errorf("expected 0 categories added with closed DB, got %d", count2)
	}
}

func TestCategoryService_CategoryGroups(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a group
	group := services.CategoryGroup{
		Name:         "Speed Awards",
		Description:  "Categories related to speed",
		DisplayOrder: 1,
	}

	groupID, err := svc.CreateGroup(ctx, group)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if groupID <= 0 {
		t.Errorf("expected positive group ID, got %d", groupID)
	}

	// List groups
	groups, err := svc.ListGroups(ctx)
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "Speed Awards" {
		t.Errorf("expected group name 'Speed Awards', got %q", groups[0].Name)
	}
}

func TestCategoryService_ListAllCategories(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Create an active category
	id, err := svc.CreateCategory(ctx, services.Category{
		Name:         "Active Category",
		DisplayOrder: 1,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	// Delete it (soft delete - should still appear in ListAllCategories)
	err = svc.DeleteCategory(ctx, int(id))
	if err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}

	// ListCategories should return empty (only active)
	activeCategories, err := svc.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(activeCategories) != 0 {
		t.Errorf("expected 0 active categories, got %d", len(activeCategories))
	}

	// ListAllCategories should include the soft-deleted one
	allCategories, err := svc.ListAllCategories(ctx)
	if err != nil {
		t.Fatalf("ListAllCategories failed: %v", err)
	}
	if len(allCategories) == 0 {
		t.Error("expected at least 1 category in ListAllCategories")
	}
}

func TestCategoryService_GetGroup(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a group
	groupID, err := svc.CreateGroup(ctx, services.CategoryGroup{
		Name:         "Design Awards",
		Description:  "Categories for design",
		DisplayOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// Get the group
	group, err := svc.GetGroup(ctx, fmt.Sprintf("%d", groupID))
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}
	if group == nil {
		t.Fatal("expected group, got nil")
	}
	if group.Name != "Design Awards" {
		t.Errorf("expected name 'Design Awards', got %q", group.Name)
	}
	if group.Description != "Categories for design" {
		t.Errorf("expected description, got %q", group.Description)
	}
}

func TestCategoryService_GetGroup_NotFound(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Try to get a non-existent group
	group, err := svc.GetGroup(ctx, "99999")
	if err != nil {
		// Error is acceptable
		return
	}
	if group != nil {
		t.Error("expected nil for non-existent group")
	}
}

func TestCategoryService_UpdateGroup(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a group
	groupID, err := svc.CreateGroup(ctx, services.CategoryGroup{
		Name:         "Original Group",
		Description:  "Original description",
		DisplayOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// Update the group
	err = svc.UpdateGroup(ctx, fmt.Sprintf("%d", groupID), services.CategoryGroup{
		Name:         "Updated Group",
		Description:  "Updated description",
		DisplayOrder: 2,
	})
	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}

	// Verify the update
	group, err := svc.GetGroup(ctx, fmt.Sprintf("%d", groupID))
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}
	if group.Name != "Updated Group" {
		t.Errorf("expected name 'Updated Group', got %q", group.Name)
	}
	if group.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got %q", group.Description)
	}
}

func TestCategoryService_DeleteGroup(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	svc := services.NewCategoryService(log, repo, derbynet.NewMockClient())
	ctx := context.Background()

	// Create a group
	groupID, err := svc.CreateGroup(ctx, services.CategoryGroup{
		Name:         "To Delete",
		Description:  "Will be deleted",
		DisplayOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// Delete the group
	err = svc.DeleteGroup(ctx, fmt.Sprintf("%d", groupID))
	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	// Verify it's gone
	groups, err := svc.ListGroups(ctx)
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	for _, g := range groups {
		if g.Name == "To Delete" {
			t.Error("expected group to be deleted")
		}
	}
}

// ==================== SyncFromDerbyNet Tests ====================

func TestCategoryService_SyncFromDerbyNet_Success(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()

	// Create mock client with specific awards
	mockAwards := []derbynet.Award{
		{AwardID: 1, AwardName: "Best Design", Sort: 1},
		{AwardID: 2, AwardName: "Most Creative", Sort: 2},
		{AwardID: 3, AwardName: "Fastest Looking", Sort: 3},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithAwards(mockAwards))

	svc := services.NewCategoryService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.TotalAwards != 3 {
		t.Errorf("expected 3 total awards, got %d", result.TotalAwards)
	}
	if result.CategoriesCreated != 3 {
		t.Errorf("expected 3 categories created, got %d", result.CategoriesCreated)
	}
	if result.CategoriesUpdated != 0 {
		t.Errorf("expected 0 categories updated, got %d", result.CategoriesUpdated)
	}

	// Verify categories were created with DerbyNet award IDs
	categories, _ := svc.ListCategories(ctx)
	if len(categories) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(categories))
	}

	// Check that award IDs are linked
	for _, cat := range categories {
		if cat.DerbyNetAwardID == nil {
			t.Errorf("expected derbynet_award_id for category %q, got nil", cat.Name)
		}
	}
}

func TestCategoryService_SyncFromDerbyNet_UpdateExisting(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Pre-create a category with the same name
	_, _ = repo.CreateCategory(ctx, "Best Design", 10, nil, nil, nil)

	mockAwards := []derbynet.Award{
		{AwardID: 1, AwardName: "Best Design", Sort: 1},
		{AwardID: 2, AwardName: "New Award", Sort: 2},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithAwards(mockAwards))

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	if result.CategoriesCreated != 1 {
		t.Errorf("expected 1 category created (New Award), got %d", result.CategoriesCreated)
	}
	if result.CategoriesUpdated != 1 {
		t.Errorf("expected 1 category updated (Best Design), got %d", result.CategoriesUpdated)
	}

	// Verify existing category got linked to DerbyNet
	categories, _ := svc.ListCategories(ctx)
	for _, cat := range categories {
		if cat.Name == "Best Design" {
			if cat.DerbyNetAwardID == nil || *cat.DerbyNetAwardID != 1 {
				t.Errorf("expected derbynet_award_id 1 for Best Design, got %v", cat.DerbyNetAwardID)
			}
		}
	}
}

func TestCategoryService_SyncFromDerbyNet_FetchError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()

	// Create mock client that returns an error
	mockClient := derbynet.NewMockClient(derbynet.WithAwardsError(fmt.Errorf("connection refused")))

	svc := services.NewCategoryService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet returned error (should return result with error status): %v", err)
	}

	if result.Status != "error" {
		t.Errorf("expected status 'error', got %q", result.Status)
	}
	if result.Message == "" {
		t.Error("expected error message, got empty")
	}
}

func TestCategoryService_SyncFromDerbyNet_EmptyAwards(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()

	// Create mock client with no awards
	mockClient := derbynet.NewMockClient(derbynet.WithAwards([]derbynet.Award{}))

	svc := services.NewCategoryService(log, repo, mockClient)
	ctx := context.Background()

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.TotalAwards != 0 {
		t.Errorf("expected 0 total awards, got %d", result.TotalAwards)
	}
	if result.CategoriesCreated != 0 {
		t.Errorf("expected 0 categories created, got %d", result.CategoriesCreated)
	}
}

func TestCategoryService_SyncFromDerbyNet_SavesURL(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	mockClient := derbynet.NewMockClient(derbynet.WithAwards([]derbynet.Award{}))

	svc := services.NewCategoryService(log, repo, mockClient)
	ctx := context.Background()

	_, err := svc.SyncFromDerbyNet(ctx, "http://test-derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	// Verify URL was saved to settings
	savedURL, err := repo.GetSetting(ctx, "derbynet_url")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if savedURL != "http://test-derbynet.local" {
		t.Errorf("expected saved URL 'http://test-derbynet.local', got %q", savedURL)
	}
}

// ==================== PUSH to DerbyNet Tests ====================

func TestCategoryService_SyncFromDerbyNet_PushLocalCategories(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create local categories without derbynet_award_id
	_, _ = repo.CreateCategory(ctx, "Local Award 1", 1, nil, nil, nil)
	_, _ = repo.CreateCategory(ctx, "Local Award 2", 2, nil, nil, nil)

	// DerbyNet has no awards
	mockClient := derbynet.NewMockClient(derbynet.WithAwards([]derbynet.Award{}))

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	if result.AwardsCreated != 2 {
		t.Errorf("expected 2 awards created in DerbyNet, got %d", result.AwardsCreated)
	}

	// Verify awards were created in DerbyNet mock
	mockAwards := mockClient.GetAwards()
	if len(mockAwards) != 2 {
		t.Errorf("expected 2 awards in mock client, got %d", len(mockAwards))
	}

	// Verify local categories now have derbynet_award_id
	categories, _ := svc.ListCategories(ctx)
	for _, cat := range categories {
		if cat.DerbyNetAwardID == nil {
			t.Errorf("expected derbynet_award_id for category %q after push, got nil", cat.Name)
		}
	}
}

func TestCategoryService_SyncFromDerbyNet_LinkByName(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// DerbyNet has an award with the same name
	mockAwards := []derbynet.Award{
		{AwardID: 42, AwardName: "Best Design", Sort: 1},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithAwards(mockAwards))

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	// Should link existing, not create new
	if result.AwardsCreated != 0 {
		t.Errorf("expected 0 awards created (should link existing), got %d", result.AwardsCreated)
	}

	// Verify local category is now linked to the DerbyNet award
	categories, _ := svc.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].DerbyNetAwardID == nil || *categories[0].DerbyNetAwardID != 42 {
		t.Errorf("expected derbynet_award_id 42, got %v", categories[0].DerbyNetAwardID)
	}
}

func TestCategoryService_SyncFromDerbyNet_CreateAwardError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = repo.CreateCategory(ctx, "Local Award", 1, nil, nil, nil)

	// Mock client returns error on CreateAward
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
		derbynet.WithCreateAwardError(fmt.Errorf("DerbyNet API error")),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	// Now we expect an error since CreateAward failed
	if err == nil {
		t.Fatal("expected error when CreateAward fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create award") {
		t.Errorf("expected error about creating award, got: %v", err)
	}

	// Verify 0 awards were created
	if result.AwardsCreated != 0 {
		t.Errorf("expected 0 awards created due to error, got %d", result.AwardsCreated)
	}

	// Local category should remain unlinked
	categories, _ := svc.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].DerbyNetAwardID != nil {
		t.Errorf("expected nil derbynet_award_id after error, got %v", categories[0].DerbyNetAwardID)
	}
}

func TestCategoryService_SyncFromDerbyNet_FetchAwardTypesError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = repo.CreateCategory(ctx, "Local Award", 1, nil, nil, nil)

	// Mock client returns error on FetchAwardTypes but CreateAward works
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
		derbynet.WithAwardTypesError(fmt.Errorf("award types unavailable")),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	// Should still succeed - uses default award type
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
	// Should still create award using default type
	if result.AwardsCreated != 1 {
		t.Errorf("expected 1 award created (with default type), got %d", result.AwardsCreated)
	}
}

func TestCategoryService_SyncFromDerbyNet_MixedPullPush(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = repo.CreateCategory(ctx, "Local Only Award", 1, nil, nil, nil)

	// DerbyNet has different awards
	mockAwards := []derbynet.Award{
		{AwardID: 1, AwardName: "DerbyNet Award 1", Sort: 1},
		{AwardID: 2, AwardName: "DerbyNet Award 2", Sort: 2},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithAwards(mockAwards))

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	// Should pull 2 from DerbyNet and push 1 to DerbyNet
	if result.CategoriesCreated != 2 {
		t.Errorf("expected 2 categories created (from DerbyNet), got %d", result.CategoriesCreated)
	}
	if result.AwardsCreated != 1 {
		t.Errorf("expected 1 award created in DerbyNet, got %d", result.AwardsCreated)
	}

	// Should have 3 total categories now
	categories, _ := svc.ListCategories(ctx)
	if len(categories) != 3 {
		t.Errorf("expected 3 total categories, got %d", len(categories))
	}

	// All should have derbynet_award_id
	for _, cat := range categories {
		if cat.DerbyNetAwardID == nil {
			t.Errorf("expected derbynet_award_id for category %q, got nil", cat.Name)
		}
	}
}

// ==================== Auth Error Handling Tests ====================

func TestCategoryService_SyncFromDerbyNet_AuthError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Configure DerbyNet credentials
	_ = repo.SetSetting(ctx, "derbynet_role", "RaceCoordinator")
	_ = repo.SetSetting(ctx, "derbynet_password", "secret123")

	// Create a local category that would need to be pushed
	_, _ = repo.CreateCategory(ctx, "Local Award", 1, nil, nil, nil)

	// Mock client returns login error
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
		derbynet.WithLoginError(fmt.Errorf("DerbyNet login failed: Incorrect password")),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet should not return error: %v", err)
	}

	// Sync should succeed but with auth error reported
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}

	// AuthError should be populated
	if result.AuthError == "" {
		t.Error("expected AuthError to be set when login fails")
	}
	if !strings.Contains(result.AuthError, "DerbyNet login failed: Incorrect password") {
		t.Errorf("expected AuthError to contain 'DerbyNet login failed: Incorrect password', got: %q", result.AuthError)
	}
}

func TestCategoryService_SyncFromDerbyNet_NoCredentials_NoAuthError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// No DerbyNet credentials configured

	// Create a local category
	_, _ = repo.CreateCategory(ctx, "Local Award", 1, nil, nil, nil)

	// Mock client that would fail login if called
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
		derbynet.WithLoginError(fmt.Errorf("should not be called")),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet should not return error: %v", err)
	}

	// AuthError should be empty since no credentials were provided
	if result.AuthError != "" {
		t.Errorf("expected empty AuthError when no credentials, got %q", result.AuthError)
	}
}

func TestCategoryService_SyncFromDerbyNet_AuthSuccess_NoAuthError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Configure DerbyNet credentials
	_ = repo.SetSetting(ctx, "derbynet_role", "RaceCoordinator")
	_ = repo.SetSetting(ctx, "derbynet_password", "correctpassword")

	// Create a local category
	_, _ = repo.CreateCategory(ctx, "Local Award", 1, nil, nil, nil)

	// Mock client with successful login (no error)
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet should not return error: %v", err)
	}

	// AuthError should be empty on successful login
	if result.AuthError != "" {
		t.Errorf("expected empty AuthError on successful login, got %q", result.AuthError)
	}

	// Award should still be created since login succeeded
	if result.AwardsCreated != 1 {
		t.Errorf("expected 1 award created after successful auth, got %d", result.AwardsCreated)
	}
}

func TestCategoryService_SyncFromDerbyNet_AuthError_PartialRole(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Only role configured, no password - login should not be attempted
	_ = repo.SetSetting(ctx, "derbynet_role", "RaceCoordinator")

	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
		derbynet.WithLoginError(fmt.Errorf("should not be called")),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet should not return error: %v", err)
	}

	// AuthError should be empty since credentials incomplete
	if result.AuthError != "" {
		t.Errorf("expected empty AuthError when password missing, got %q", result.AuthError)
	}
}

func TestCategoryService_SyncFromDerbyNet_SetSettingError(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Close DB to cause SetSetting to fail
	repo.DB().Close()

	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when SetSetting fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to save DerbyNet URL") {
		t.Errorf("expected error about saving URL, got: %v", err)
	}
}

func TestCategoryService_SyncFromDerbyNet_AwardWithZeroSort(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Set up DerbyNet with an award that has Sort=0 (should use AwardID instead)
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{
			{AwardID: 42, AwardName: "Test Award", Sort: 0}, // Sort=0, should use AwardID for display_order
		}),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	if result.CategoriesCreated != 1 {
		t.Errorf("expected 1 category created, got %d", result.CategoriesCreated)
	}

	// Verify the category was created with AwardID as display_order
	categories, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].DisplayOrder != 42 {
		t.Errorf("expected display_order 42 (from AwardID), got %d", categories[0].DisplayOrder)
	}
}

func TestCategoryService_SyncFromDerbyNet_EmptyAwardTypes(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create local category without DerbyNet link
	_, _ = repo.CreateCategory(ctx, "Local Award", 1, nil, nil, nil)

	// Mock client that returns empty award types
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
		derbynet.WithAwardTypes([]derbynet.AwardType{}), // Empty award types
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	// Should still create award with default type ID 1
	if result.AwardsCreated != 1 {
		t.Errorf("expected 1 award created, got %d", result.AwardsCreated)
	}
}

func TestCategoryService_SyncFromDerbyNet_WithDebugLogging(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Set DerbyNet credentials to trigger auth path
	_ = repo.SetSetting(ctx, "derbynet_role", "RaceCoordinator")
	_ = repo.SetSetting(ctx, "derbynet_password", "test123")

	// Create local category
	_, _ = repo.CreateCategory(ctx, "Local Award", 1, nil, nil, nil)

	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
	)

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	// Should authenticate successfully
	if result.AuthError != "" {
		t.Errorf("expected no auth error with valid credentials, got %q", result.AuthError)
	}

	// Should create award
	if result.AwardsCreated != 1 {
		t.Errorf("expected 1 award created, got %d", result.AwardsCreated)
	}
}

func TestCategoryService_SyncFromDerbyNet_UpsertCategoryError_PullPhase(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	// Set up DerbyNet with awards
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{
			{AwardID: 1, AwardName: "Award One", Sort: 1},
		}),
	)

	svc := services.NewCategoryService(log, mockRepo, mockClient)

	// Configure mock to fail on UpsertCategory (during pull phase)
	mockRepo.UpsertCategoryError = errors.New("database error during upsert")

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when UpsertCategory fails during pull, got nil")
	}
	if !strings.Contains(err.Error(), "failed to sync award") {
		t.Errorf("expected error about syncing award, got: %v", err)
	}
}

func TestCategoryService_SyncFromDerbyNet_ListCategoriesError_PushPhase(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	// Set up DerbyNet with no awards (so pull phase succeeds)
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{}),
	)

	svc := services.NewCategoryService(log, mockRepo, mockClient)

	// Configure mock to fail on ListCategories (during push phase)
	mockRepo.ListCategoriesError = errors.New("database error listing categories")

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when ListCategories fails during push, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list categories") {
		t.Errorf("expected error about listing categories, got: %v", err)
	}
}

func TestCategoryService_SeedMockCategories_CategoryExistsError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	svc := services.NewCategoryService(log, mockRepo, derbynet.NewMockClient())

	// Configure mock to fail on CategoryExists check
	mockRepo.CategoryExistsError = errors.New("database error checking existence")

	count, err := svc.SeedMockCategories(ctx)
	if err == nil {
		t.Fatal("expected error when CategoryExists fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to check if category exists") {
		t.Errorf("expected error about checking category exists, got: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 categories added when check fails, got %d", count)
	}
}

func TestCategoryService_SeedMockCategories_CreateCategoryError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	svc := services.NewCategoryService(log, mockRepo, derbynet.NewMockClient())

	// Configure mock to fail on CreateCategory
	mockRepo.CreateCategoryError = errors.New("database error creating category")

	count, err := svc.SeedMockCategories(ctx)
	if err == nil {
		t.Fatal("expected error when CreateCategory fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create category") {
		t.Errorf("expected error about creating category, got: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 categories added when creation fails, got %d", count)
	}
}

func TestCategoryService_SyncFromDerbyNet_LinkExistingByName(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = repo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// DerbyNet has an award with the same name
	mockAwards := []derbynet.Award{
		{AwardID: 42, AwardName: "Best Design", Sort: 1},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithAwards(mockAwards))

	svc := services.NewCategoryService(log, repo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err != nil {
		t.Fatalf("SyncFromDerbyNet failed: %v", err)
	}

	// Should have pulled the award (creating/updating local category) and linked by name (updating again)
	// The local "Best Design" should be updated with derbynet_award_id=42
	if result.CategoriesUpdated < 1 {
		t.Errorf("expected at least 1 category updated (linked by name), got %d", result.CategoriesUpdated)
	}

	// Verify local category is now linked to the DerbyNet award
	categories, _ := svc.ListCategories(ctx)
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].DerbyNetAwardID == nil || *categories[0].DerbyNetAwardID != 42 {
		t.Errorf("expected derbynet_award_id 42, got %v", categories[0].DerbyNetAwardID)
	}
}

func TestCategoryService_SyncFromDerbyNet_UpsertCategoryErrorAfterCreateAward(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = realRepo.CreateCategory(ctx, "Local Award", 1, nil, nil, nil)

	// DerbyNet has no awards (so CreateAward will be called)
	mockClient := derbynet.NewMockClient(derbynet.WithAwards([]derbynet.Award{}))

	svc := services.NewCategoryService(log, mockRepo, mockClient)

	// Configure mock to fail on UpsertCategory after CreateAward succeeds
	// Since DerbyNet has no awards, the only UpsertCategory call will be after CreateAward
	originalUpsertCategory := mockRepo.FullRepository.UpsertCategory

	mockRepo.FullRepository = &mockCategoryRepo{
		FullRepository: mockRepo.FullRepository,
		upsertCategory: func(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (bool, error) {
			// This call happens after CreateAward in DerbyNet, should fail
			if derbynetAwardID != nil {
				return false, errors.New("database error updating category with award ID")
			}
			return originalUpsertCategory(ctx, name, displayOrder, derbynetAwardID)
		},
	}

	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
	if err == nil {
		t.Fatal("expected error when UpsertCategory fails after CreateAward, got nil")
	}
	if !strings.Contains(err.Error(), "failed to update category") {
		t.Errorf("expected error about updating category, got: %v", err)
	}
}

// TestCategoryService_SyncFromDerbyNet_ListCategoriesError tests error when ListCategories fails during PUSH
func TestCategoryService_SyncFromDerbyNet_ListCategoriesError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)

	ctx := context.Background()
	log := logger.New()

	// Setup mock DerbyNet client with some awards
	mockClient := derbynet.NewMockClient(
		derbynet.WithAwards([]derbynet.Award{
			{AwardID: 1, AwardName: "Fastest", Sort: 10},
		}),
	)

	// Configure credentials for push sync
	realRepo.SetSetting(ctx, "derbynet_role", "admin")
	realRepo.SetSetting(ctx, "derbynet_password", "pass")

	// Inject error for ListCategories (happens during PUSH phase)
	mockRepo.ListCategoriesError = errors.New("database error listing categories")

	svc := services.NewCategoryService(log, mockRepo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")

	// Should return error from ListCategories
	if err == nil {
		t.Fatal("expected error when ListCategories fails during PUSH, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list categories") {
		t.Errorf("expected error about listing categories, got: %v", err)
	}

	// Result should still contain PULL phase data
	if result == nil {
		t.Fatal("expected result to be returned even with error")
	}
	if result.TotalAwards != 1 {
		t.Errorf("expected 1 award from PULL phase, got %d", result.TotalAwards)
	}
}

// TestCategoryService_SyncFromDerbyNet_LinkExistingAwardError tests error when linking to existing award fails
func TestCategoryService_SyncFromDerbyNet_LinkExistingAwardError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = realRepo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// DerbyNet has an award with the same name
	mockAwards := []derbynet.Award{
		{AwardID: 42, AwardName: "Best Design", Sort: 1},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithAwards(mockAwards))

	// Inject error for ALL UpsertCategory calls with award ID 42
	// This will cause both PULL and PUSH to fail when trying to link
	originalUpsertCategory := mockRepo.FullRepository.UpsertCategory

	mockRepo.FullRepository = &mockCategoryRepo{
		FullRepository: mockRepo.FullRepository,
		upsertCategory: func(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (bool, error) {
			// Fail any UpsertCategory for award 42 (both PULL and PUSH phases)
			if derbynetAwardID != nil && *derbynetAwardID == 42 {
				return false, errors.New("database error linking category to award")
			}
			return originalUpsertCategory(ctx, name, displayOrder, derbynetAwardID)
		},
	}

	svc := services.NewCategoryService(log, mockRepo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")

	// Should return error from failed link (either PULL or PUSH phase)
	if err == nil {
		t.Fatal("expected error when linking category to award fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to") {
		t.Errorf("expected error about failing to sync/link category, got: %v", err)
	}

	// No categories should be successfully created/updated due to errors
	if result.CategoriesCreated != 0 {
		t.Errorf("expected 0 categories created (error during PULL), got %d", result.CategoriesCreated)
	}
	if result.CategoriesUpdated != 0 {
		t.Errorf("expected 0 categories updated (error during PUSH), got %d", result.CategoriesUpdated)
	}
}

// mockCategoryRepo wraps a repository to inject specific errors for category operations
type mockCategoryRepo struct {
	repository.FullRepository
	upsertCategory func(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (bool, error)
}

func (m *mockCategoryRepo) UpsertCategory(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (bool, error) {
	if m.upsertCategory != nil {
		return m.upsertCategory(ctx, name, displayOrder, derbynetAwardID)
	}
	return m.FullRepository.UpsertCategory(ctx, name, displayOrder, derbynetAwardID)
}

// TestCategoryService_SyncFromDerbyNet_LinkExistingAwardFailureFirstError tests PUSH linking failure as first error
func TestCategoryService_SyncFromDerbyNet_LinkExistingAwardFailureFirstError(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	mockRepo := mock.NewRepository(realRepo)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = realRepo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// DerbyNet has awards
	mockAwards := []derbynet.Award{
		{AwardID: 42, AwardName: "Best Design", Sort: 1},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithAwards(mockAwards))

	// Track calls to UpsertCategory
	pullCallCount := 0
	originalUpsert := mockRepo.FullRepository.UpsertCategory

	mockRepo.FullRepository = &mockCategoryRepo{
		FullRepository: mockRepo.FullRepository,
		upsertCategory: func(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (bool, error) {
			if derbynetAwardID != nil && *derbynetAwardID == 42 {
				pullCallCount++
				if pullCallCount == 1 {
					// First call (PULL) - succeed but don't actually update to keep derbynet_award_id nil
					return true, nil
				}
				// Second call (PUSH) - fail with error
				return false, errors.New("database error linking during PUSH")
			}
			return originalUpsert(ctx, name, displayOrder, derbynetAwardID)
		},
	}

	svc := services.NewCategoryService(log, mockRepo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")

	// Should return error from PUSH phase
	if err == nil {
		t.Fatal("expected error when linking fails during PUSH, got nil")
	}
	if !strings.Contains(err.Error(), "failed to link") {
		t.Errorf("expected error about failed linking, got: %v", err)
	}

	// Verify this was the first error (PULL succeeded)
	if result.CategoriesCreated != 1 {
		t.Errorf("expected 1 category created during PULL, got %d", result.CategoriesCreated)
	}
}

// TestCategoryService_SyncFromDerbyNet_LinkExistingAwardSuccess tests successful linking during PUSH phase
func TestCategoryService_SyncFromDerbyNet_LinkExistingAwardSuccess(t *testing.T) {
	realRepo := testutil.NewTestRepository(t)
	log := logger.New()
	ctx := context.Background()

	// Create local category without derbynet_award_id
	_, _ = realRepo.CreateCategory(ctx, "Best Design", 1, nil, nil, nil)

	// DerbyNet has awards, including one with the same name as local category
	// But we'll use a mock repo to prevent PULL from setting the derbynet_award_id
	mockRepo := mock.NewRepository(realRepo)

	// Track if PULL or PUSH called UpsertCategory
	pullCalled := false
	pushCalled := false
	originalUpsert := mockRepo.FullRepository.UpsertCategory

	mockRepo.FullRepository = &mockCategoryRepo{
		FullRepository: mockRepo.FullRepository,
		upsertCategory: func(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (bool, error) {
			if derbynetAwardID != nil && *derbynetAwardID == 42 && !pullCalled {
				// First call during PULL - succeed but don't actually update
				pullCalled = true
				return true, nil // Return success but don't call real UpsertCategory
			}
			// Second call during PUSH - actually do the link
			if derbynetAwardID != nil && *derbynetAwardID == 42 && pullCalled {
				pushCalled = true
			}
			return originalUpsert(ctx, name, displayOrder, derbynetAwardID)
		},
	}

	mockAwards := []derbynet.Award{
		{AwardID: 42, AwardName: "Best Design", Sort: 1},
	}
	mockClient := derbynet.NewMockClient(derbynet.WithAwards(mockAwards))

	svc := services.NewCategoryService(log, mockRepo, mockClient)

	result, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")

	// Should succeed
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify PUSH linking was called
	if !pushCalled {
		t.Error("expected PUSH phase to link existing award, but it wasn't called")
	}

	// Category should be linked (updated) during PUSH phase
	if result.CategoriesUpdated < 1 {
		t.Errorf("expected at least 1 category updated (linked during PUSH), got %d", result.CategoriesUpdated)
	}
}

func TestCategoryService_CountVotesForCategory(t *testing.T) {
	repo := testutil.NewTestRepository(t)
	mockClient := derbynet.NewMockClient()
	log := logger.New()
	svc := services.NewCategoryService(log, repo, mockClient)
	ctx := context.Background()

	// Create a category
	catID, err := repo.CreateCategory(ctx, "Test Category", 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Initially should have 0 votes
	count, err := svc.CountVotesForCategory(ctx, int(catID))
	if err != nil {
		t.Fatalf("CountVotesForCategory failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 votes, got %d", count)
	}

	// Create a car
	err = repo.CreateCar(ctx, "101", "Test Racer", "Test Car", "")
	if err != nil {
		t.Fatalf("failed to create car: %v", err)
	}
	cars, _ := repo.ListCars(ctx)
	carID := cars[0].ID

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
	count, err = svc.CountVotesForCategory(ctx, int(catID))
	if err != nil {
		t.Fatalf("CountVotesForCategory failed: %v", err)
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
	count, err = svc.CountVotesForCategory(ctx, int(catID))
	if err != nil {
		t.Fatalf("CountVotesForCategory failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 votes, got %d", count)
	}
}
