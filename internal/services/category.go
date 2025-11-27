package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/models"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// CategoryServiceRepository defines the repository methods needed by CategoryService
type CategoryServiceRepository interface {
	repository.CategoryRepository
	repository.SettingsRepository
	repository.VoteRepository
}

// CategoryService handles category-related business logic
type CategoryService struct {
	log    logger.Logger
	repo   CategoryServiceRepository
	client derbynet.Client
}

// NewCategoryService creates a new CategoryService
func NewCategoryService(log logger.Logger, repo CategoryServiceRepository, client derbynet.Client) *CategoryService {
	return &CategoryService{log: log, repo: repo, client: client}
}

// CategorySyncResult contains the result of a DerbyNet category sync
type CategorySyncResult struct {
	Status            string `json:"status"`
	Message           string `json:"message,omitempty"`
	CategoriesCreated int    `json:"categories_created"`
	CategoriesUpdated int    `json:"categories_updated"`
	AwardsCreated     int    `json:"awards_created"`     // awards created in DerbyNet
	TotalCategories   int    `json:"total_categories"`
	TotalAwards       int    `json:"total_awards"`
	AuthError         string `json:"auth_error,omitempty"` // DerbyNet authentication error if any
}

// Category represents a category for create/update operations
type Category struct {
	Name              string
	DisplayOrder      int
	GroupID           *int
	Active            bool
	AllowedVoterTypes []string
	AllowedRanks      []string
}

// CategoryGroup represents a category group for create/update operations
type CategoryGroup struct {
	Name              string
	Description       string
	ExclusivityPoolID *int
	MaxWinsPerCar     *int
	DisplayOrder      int
}

// ListCategories returns all active categories
func (s *CategoryService) ListCategories(ctx context.Context) ([]models.Category, error) {
	return s.repo.ListCategories(ctx)
}

// ListAllCategories returns all categories including inactive
func (s *CategoryService) ListAllCategories(ctx context.Context) ([]map[string]interface{}, error) {
	return s.repo.ListAllCategories(ctx)
}

// CreateCategory creates a new category
func (s *CategoryService) CreateCategory(ctx context.Context, cat Category) (int64, error) {
	return s.repo.CreateCategory(ctx, cat.Name, cat.DisplayOrder, cat.GroupID, cat.AllowedVoterTypes, cat.AllowedRanks)
}

// UpdateCategory updates a category
func (s *CategoryService) UpdateCategory(ctx context.Context, id int, cat Category) error {
	return s.repo.UpdateCategory(ctx, id, cat.Name, cat.DisplayOrder, cat.GroupID, cat.AllowedVoterTypes, cat.AllowedRanks, cat.Active)
}

// DeleteCategory soft-deletes a category
func (s *CategoryService) DeleteCategory(ctx context.Context, id int) error {
	return s.repo.DeleteCategory(ctx, id)
}

// CountVotesForCategory returns the number of votes in a category
func (s *CategoryService) CountVotesForCategory(ctx context.Context, categoryID int) (int, error) {
	return s.repo.CountVotesForCategory(ctx, categoryID)
}

// ListGroups returns all category groups
func (s *CategoryService) ListGroups(ctx context.Context) ([]models.CategoryGroup, error) {
	return s.repo.ListCategoryGroups(ctx)
}

// GetGroup retrieves a category group by ID
func (s *CategoryService) GetGroup(ctx context.Context, id string) (*models.CategoryGroup, error) {
	return s.repo.GetCategoryGroup(ctx, id)
}

// CreateGroup creates a new category group
func (s *CategoryService) CreateGroup(ctx context.Context, group CategoryGroup) (int64, error) {
	return s.repo.CreateCategoryGroup(ctx, group.Name, group.Description, group.ExclusivityPoolID, group.MaxWinsPerCar, group.DisplayOrder)
}

// UpdateGroup updates a category group
func (s *CategoryService) UpdateGroup(ctx context.Context, id string, group CategoryGroup) error {
	return s.repo.UpdateCategoryGroup(ctx, id, group.Name, group.Description, group.ExclusivityPoolID, group.MaxWinsPerCar, group.DisplayOrder)
}

// DeleteGroup deletes a category group
func (s *CategoryService) DeleteGroup(ctx context.Context, id string) error {
	return s.repo.DeleteCategoryGroup(ctx, id)
}

// SeedMockCategories seeds mock category data
func (s *CategoryService) SeedMockCategories(ctx context.Context) (int, error) {
	mockCategories := []struct {
		Name         string
		DisplayOrder int
	}{
		{"Most Creative", 1},
		{"Best Paint Job", 2},
		{"Best Design", 3},
		{"Fastest Looking", 4},
		{"Most Unique", 5},
		{"Best Theme", 6},
	}

	var addedCount int
	var firstError error
	for _, cat := range mockCategories {
		exists, err := s.repo.CategoryExists(ctx, cat.Name)
		if err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("failed to check if category exists: %w", err)
			}
			continue
		}
		if !exists {
			_, err := s.repo.CreateCategory(ctx, cat.Name, cat.DisplayOrder, nil, nil, nil)
			if err != nil {
				if firstError == nil {
					firstError = fmt.Errorf("failed to create category %q: %w", cat.Name, err)
				}
			} else {
				addedCount++
			}
		}
	}

	return addedCount, firstError
}

// SyncFromDerbyNet syncs categories bi-directionally with DerbyNet awards
// - Pull: Awards from DerbyNet are created/linked as local categories
// - Push: Local categories without derbynet_award_id are created as awards in DerbyNet
func (s *CategoryService) SyncFromDerbyNet(ctx context.Context, derbyNetURL string) (*CategorySyncResult, error) {
	// Set the URL on the client
	s.client.SetBaseURL(derbyNetURL)

	// Save DerbyNet URL to settings
	if err := s.repo.SetSetting(ctx, "derbynet_url", derbyNetURL); err != nil {
		return nil, fmt.Errorf("failed to save DerbyNet URL: %w", err)
	}

	result := &CategorySyncResult{Status: "success"}
	var firstError error

	// ========== PULL: DerbyNet -> DerbyVote ==========
	// Fetch awards from DerbyNet
	awards, err := s.client.FetchAwards(ctx)
	if err != nil {
		return &CategorySyncResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to fetch awards from DerbyNet: %v", err),
		}, nil
	}

	s.log.Info("Fetched awards from DerbyNet", "count", len(awards))
	result.TotalAwards = len(awards)

	// Build a set of award names for checking existing awards
	awardNameSet := make(map[string]int) // name -> awardID
	for _, award := range awards {
		awardNameSet[award.AwardName] = award.AwardID
	}

	// Process awards: create/update local categories
	for _, award := range awards {
		displayOrder := award.Sort
		if displayOrder == 0 {
			displayOrder = award.AwardID
		}

		awardID := award.AwardID
		created, err := s.repo.UpsertCategory(ctx, award.AwardName, displayOrder, &awardID)
		if err != nil {
			s.log.Error("Error syncing award", "award_id", award.AwardID, "name", award.AwardName, "error", err)
			if firstError == nil {
				firstError = fmt.Errorf("failed to sync award %q: %w", award.AwardName, err)
			}
			continue
		}

		if created {
			result.CategoriesCreated++
		} else {
			result.CategoriesUpdated++
		}
	}

	// ========== PUSH: DerbyVote -> DerbyNet ==========
	// Configure credentials for automatic authentication
	derbyNetRole, _ := s.repo.GetSetting(ctx, "derbynet_role")
	derbyNetPassword, _ := s.repo.GetSetting(ctx, "derbynet_password")
	if derbyNetRole != "" && derbyNetPassword != "" {
		s.log.Debug("Configuring DerbyNet credentials", "role", derbyNetRole)
		s.client.SetCredentials(derbyNetRole, derbyNetPassword)
	}

	// Get all local categories
	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		s.log.Error("Failed to list categories for push sync", "error", err)
		if firstError == nil {
			firstError = fmt.Errorf("failed to list categories: %w", err)
		}
	} else {
		// Get award types from DerbyNet (need to know available types)
		awardTypes, err := s.client.FetchAwardTypes(ctx)
		if err != nil {
			s.log.Warn("Failed to fetch award types, using default", "error", err)
			// Use default award type ID 1 (typically "Design")
			awardTypes = []derbynet.AwardType{{AwardTypeID: 1, AwardType: "Design"}}
		}

		// Default to first award type (usually "Design")
		defaultAwardTypeID := 1
		if len(awardTypes) > 0 {
			defaultAwardTypeID = awardTypes[0].AwardTypeID
		}

		// Push categories without derbynet_award_id to DerbyNet
		for _, cat := range categories {
			if cat.DerbyNetAwardID != nil {
				continue // Already linked
			}

			// Check if an award with this name already exists in DerbyNet
			if existingAwardID, exists := awardNameSet[cat.Name]; exists {
				// Link to existing award
				s.log.Info("Linking existing category to DerbyNet award", "category", cat.Name, "award_id", existingAwardID)
				_, err := s.repo.UpsertCategory(ctx, cat.Name, cat.DisplayOrder, &existingAwardID)
				if err != nil {
					s.log.Error("Failed to link category to award", "category", cat.Name, "error", err)
					if firstError == nil {
						firstError = fmt.Errorf("failed to link category %q to award: %w", cat.Name, err)
					}
				} else {
					result.CategoriesUpdated++
				}
				continue
			}

			// Create new award in DerbyNet
			s.log.Info("Creating award in DerbyNet", "category", cat.Name)
			newAwardID, err := s.client.CreateAward(ctx, cat.Name, defaultAwardTypeID)
			if err != nil {
				s.log.Error("Failed to create award in DerbyNet", "category", cat.Name, "error", err)
				// Check if it's an authentication error
				errMsg := err.Error()
				isAuthError := strings.Contains(errMsg, "failed to authenticate") ||
					strings.Contains(errMsg, "failed to re-authenticate") ||
					strings.Contains(errMsg, "notauthorized")

				if isAuthError {
					// Record auth error but don't fail the sync
					if result.AuthError == "" {
						result.AuthError = errMsg
					}
				} else {
					// Non-auth errors should fail the sync
					if firstError == nil {
						firstError = fmt.Errorf("failed to create award %q in DerbyNet: %w", cat.Name, err)
					}
				}
				continue
			}

			// Update local category with new award ID
			_, err = s.repo.UpsertCategory(ctx, cat.Name, cat.DisplayOrder, &newAwardID)
			if err != nil {
				s.log.Error("Failed to update category with award ID", "category", cat.Name, "award_id", newAwardID, "error", err)
				if firstError == nil {
					firstError = fmt.Errorf("failed to update category %q with award ID: %w", cat.Name, err)
				}
				continue
			}

			result.AwardsCreated++
			s.log.Info("Created award in DerbyNet", "category", cat.Name, "award_id", newAwardID)
		}
	}

	result.TotalCategories = result.CategoriesCreated + result.CategoriesUpdated

	s.log.Info("Category sync complete",
		"categories_created", result.CategoriesCreated,
		"categories_updated", result.CategoriesUpdated,
		"awards_created", result.AwardsCreated)

	return result, firstError
}
