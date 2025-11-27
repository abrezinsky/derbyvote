package services

import (
	"context"
	stderrors "errors"

	"github.com/abrezinsky/derbyvote/internal/errors"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/models"
	"github.com/abrezinsky/derbyvote/internal/repository"
)

// VotingServiceRepository defines the repository methods needed by VotingService
type VotingServiceRepository interface {
	repository.VoterRepository
	repository.VoteRepository
	repository.CategoryRepository
	repository.CarRepository
	ListEligibleCars(ctx context.Context) ([]models.Car, error)
}

// VotingService handles vote-related business logic
type VotingService struct {
	log      logger.Logger
	repo     VotingServiceRepository
	category CategoryServicer
	car      CarServicer
	settings SettingsServicer
}

// NewVotingService creates a new VotingService
func NewVotingService(log logger.Logger, repo VotingServiceRepository, category CategoryServicer, car CarServicer, settings SettingsServicer) *VotingService {
	return &VotingService{
		log:      log,
		repo:     repo,
		category: category,
		car:      car,
		settings: settings,
	}
}

// VoteData contains all data needed for the voting interface
type VoteData struct {
	Categories   []models.Category `json:"categories"`
	Cars         []models.Car      `json:"cars"`
	Votes        map[int]int       `json:"votes"`
	Instructions string            `json:"instructions,omitempty"`
}

// VoteResult contains the result of a vote submission
type VoteResult struct {
	Status               string `json:"status"`
	Message              string `json:"message"`
	ConflictCleared      bool   `json:"conflict_cleared,omitempty"`
	ConflictCategoryID   int    `json:"conflict_category_id,omitempty"`
	ConflictCategoryName string `json:"conflict_category_name,omitempty"`
}

// GetVoteData retrieves all data needed for voting
func (s *VotingService) GetVoteData(ctx context.Context, qrCode string) (*VoteData, error) {
	// Get or create voter
	voterID, err := s.GetOrCreateVoter(ctx, qrCode)
	if err != nil {
		return nil, err
	}

	// Get voter type
	voterType, err := s.repo.GetVoterType(ctx, voterID)
	if err != nil {
		return nil, err
	}

	// Get categories
	allCategories, err := s.repo.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	// Filter categories based on voter type
	categories := filterCategoriesByVoterType(allCategories, voterType)

	// Get only eligible cars for voting
	cars, err := s.repo.ListEligibleCars(ctx)
	if err != nil {
		return nil, err
	}

	// Get existing votes
	votes, err := s.repo.GetVoterVotes(ctx, voterID)
	if err != nil {
		return nil, err
	}

	// Get voting instructions (if configured)
	instructions, _ := s.settings.GetSetting(ctx, "voting_instructions")

	return &VoteData{
		Categories:   categories,
		Cars:         cars,
		Votes:        votes,
		Instructions: instructions,
	}, nil
}

// filterCategoriesByVoterType filters categories to only include those allowed for the voter type
func filterCategoriesByVoterType(categories []models.Category, voterType string) []models.Category {
	var filtered []models.Category
	for _, cat := range categories {
		// If no allowed types specified, category is available to all
		if len(cat.AllowedVoterTypes) == 0 {
			filtered = append(filtered, cat)
			continue
		}

		// Check if voter type is in the allowed list
		for _, allowedType := range cat.AllowedVoterTypes {
			if allowedType == voterType {
				filtered = append(filtered, cat)
				break
			}
		}
	}
	return filtered
}

// GetOrCreateVoter gets an existing voter or creates a new one based on settings
func (s *VotingService) GetOrCreateVoter(ctx context.Context, qrCode string) (int, error) {
	voterID, err := s.repo.GetVoterByQR(ctx, qrCode)
	if err == repository.ErrNotFound {
		// Check if we require pre-registered QR codes
		requireRegistered, settingsErr := s.settings.RequireRegisteredQR(ctx)
		if settingsErr != nil {
			return 0, settingsErr
		}
		if requireRegistered {
			return 0, ErrUnregisteredQR
		}
		return s.repo.CreateVoter(ctx, qrCode)
	}
	return voterID, err
}

// SubmitVote processes a vote submission with exclusivity conflict handling
func (s *VotingService) SubmitVote(ctx context.Context, vote models.Vote) (*VoteResult, error) {
	// Check if voting is open
	open, err := s.settings.IsVotingOpen(ctx)
	if err != nil {
		return nil, err
	}
	if !open {
		return nil, ErrVotingClosed
	}

	// Get or create voter
	voterID, err := s.GetOrCreateVoter(ctx, vote.VoterQR)
	if err != nil {
		return nil, err
	}

	var conflictCategoryID int
	var conflictCategoryName string
	var hadConflict bool

	// Check for eligibility and exclusivity conflicts (only if not deselecting)
	if vote.CarID != 0 {
		// Check if car is eligible for voting
		car, err := s.repo.GetCar(ctx, vote.CarID)
		if err != nil {
			// Check if it's a not found error and convert to service-specific error
			var appErr *errors.Error
			if stderrors.As(err, &appErr) && appErr.Kind == errors.ErrNotFound {
				return nil, ErrCarNotFound
			}
			return nil, err
		}
		if !car.Eligible {
			return nil, ErrCarNotEligible
		}
		conflictCategoryID, conflictCategoryName, hadConflict, err = s.checkExclusivityConflict(ctx, voterID, vote.CarID, vote.CategoryID)
		if err != nil {
			return nil, err
		}

		// Clear conflicting vote if found
		if hadConflict {
			if err := s.repo.ClearConflictingVote(ctx, voterID, conflictCategoryID, vote.CarID); err != nil {
				return nil, err
			}
			s.log.Info("Cleared conflicting vote", "voter_id", voterID, "category", conflictCategoryID, "car", vote.CarID)
		}
	}

	// Save the vote
	if err := s.repo.SaveVote(ctx, voterID, vote.CategoryID, vote.CarID); err != nil {
		return nil, err
	}

	s.log.Info("Vote recorded", "qr", vote.VoterQR, "voter_id", voterID, "category", vote.CategoryID, "car", vote.CarID)

	result := &VoteResult{
		Status:  "success",
		Message: "Vote recorded",
	}

	if hadConflict {
		result.ConflictCleared = true
		result.ConflictCategoryID = conflictCategoryID
		result.ConflictCategoryName = conflictCategoryName
	}

	return result, nil
}

// checkExclusivityConflict checks if voting for a car in a category conflicts with existing votes
func (s *VotingService) checkExclusivityConflict(ctx context.Context, voterID, carID, categoryID int) (conflictCategoryID int, conflictCategoryName string, hasConflict bool, err error) {
	// Get the exclusivity pool for the target category
	poolID, hasPool, err := s.repo.GetExclusivityPoolID(ctx, categoryID)
	if err != nil {
		return 0, "", false, err
	}

	// If no exclusivity pool, no conflict is possible
	if !hasPool {
		return 0, "", false, nil
	}

	// Check for conflicting votes
	return s.repo.FindConflictingVote(ctx, voterID, carID, categoryID, poolID)
}
