package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// ResultsServiceRepository defines the repository methods needed by ResultsService
type ResultsServiceRepository interface {
	repository.CategoryRepository
	repository.CarRepository
	repository.VoteRepository
	repository.SettingsRepository
}

// ResultsService handles results and statistics business logic
type ResultsService struct {
	log      logger.Logger
	repo     ResultsServiceRepository
	settings SettingsServicer
	client   derbynet.Client
}

// NewResultsService creates a new ResultsService
func NewResultsService(log logger.Logger, repo ResultsServiceRepository, settings SettingsServicer, client derbynet.Client) *ResultsService {
	return &ResultsService{log: log, repo: repo, settings: settings, client: client}
}

// CarResult represents a car's vote result in a category
type CarResult struct {
	CarID     int    `json:"car_id"`
	CarNumber string `json:"car_number"`
	CarName   string `json:"car_name"`
	RacerName string `json:"racer_name"`
	PhotoURL  string `json:"photo_url"`
	VoteCount int    `json:"vote_count"`
	Rank      int    `json:"rank"`
}

// CategoryResult represents results for a single category
type CategoryResult struct {
	CategoryID          int         `json:"category_id"`
	CategoryName        string      `json:"category_name"`
	GroupID             *int        `json:"group_id,omitempty"`
	GroupName           string      `json:"group_name,omitempty"`
	TotalVotes          int         `json:"total_votes"`
	Votes               []CarResult `json:"votes"`
	HasOverride         bool        `json:"has_override"`
	OverrideCarID       *int        `json:"override_car_id,omitempty"`
	OverrideReason      string      `json:"override_reason,omitempty"`
	OverriddenAt        string      `json:"overridden_at,omitempty"`
}

// FullResults contains all voting results
type FullResults struct {
	Categories []CategoryResult       `json:"categories"`
	Stats      map[string]interface{} `json:"stats"`
}

// GetResults retrieves full voting results
func (s *ResultsService) GetResults(ctx context.Context) (*FullResults, error) {
	// Get categories
	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	// Get vote results with car details (single query, only cars with votes)
	voteRows, err := s.repo.GetVoteResultsWithCars(ctx)
	if err != nil {
		return nil, err
	}

	// Get stats
	stats, err := s.repo.GetVotingStats(ctx)
	if err != nil {
		return nil, err
	}

	// Group votes by category
	votesByCategory := make(map[int][]CarResult)
	totalByCategory := make(map[int]int)
	for _, row := range voteRows {
		votesByCategory[row.CategoryID] = append(votesByCategory[row.CategoryID], CarResult{
			CarID:     row.CarID,
			CarNumber: row.CarNumber,
			CarName:   row.CarName,
			RacerName: row.RacerName,
			PhotoURL:  row.PhotoURL,
			VoteCount: row.VoteCount,
		})
		totalByCategory[row.CategoryID] += row.VoteCount
	}

	// Build category results
	var categoryResults []CategoryResult
	for _, cat := range categories {
		votes := votesByCategory[cat.ID]

		// Assign ranks (already sorted by vote_count DESC from SQL)
		for i := range votes {
			votes[i].Rank = i + 1
		}

		hasOverride := cat.OverrideWinnerCarID != nil
		categoryResults = append(categoryResults, CategoryResult{
			CategoryID:     cat.ID,
			CategoryName:   cat.Name,
			GroupID:        cat.GroupID,
			GroupName:      cat.GroupName,
			TotalVotes:     totalByCategory[cat.ID],
			Votes:          votes,
			HasOverride:    hasOverride,
			OverrideCarID:  cat.OverrideWinnerCarID,
			OverrideReason: cat.OverrideReason,
			OverriddenAt:   cat.OverriddenAt,
		})
	}

	return &FullResults{
		Categories: categoryResults,
		Stats:      stats,
	}, nil
}

// GetCategoryResults retrieves results for a specific category
func (s *ResultsService) GetCategoryResults(ctx context.Context, categoryID int) (*CategoryResult, error) {
	results, err := s.GetResults(ctx)
	if err != nil {
		return nil, err
	}

	for _, cat := range results.Categories {
		if cat.CategoryID == categoryID {
			return &cat, nil
		}
	}
	return nil, nil
}

// GetStats retrieves voting statistics including voting_open status
func (s *ResultsService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := s.repo.GetVotingStats(ctx)
	if err != nil {
		return nil, err
	}

	// Add voting status
	if s.settings != nil {
		votingOpen, _ := s.settings.IsVotingOpen(ctx)
		stats["voting_open"] = votingOpen
	}

	return stats, nil
}

// GetWinners returns the top winner for each category
func (s *ResultsService) GetWinners(ctx context.Context) ([]map[string]interface{}, error) {
	results, err := s.GetResults(ctx)
	if err != nil {
		return nil, err
	}

	var winners []map[string]interface{}
	for _, cat := range results.Categories {
		if len(cat.Votes) > 0 && cat.Votes[0].VoteCount > 0 {
			winners = append(winners, map[string]interface{}{
				"category_id":   cat.CategoryID,
				"category_name": cat.CategoryName,
				"winner": map[string]interface{}{
					"car_id":     cat.Votes[0].CarID,
					"car_number": cat.Votes[0].CarNumber,
					"car_name":   cat.Votes[0].CarName,
					"racer_name": cat.Votes[0].RacerName,
					"vote_count": cat.Votes[0].VoteCount,
				},
			})
		}
	}

	return winners, nil
}

// ResultsPushResult contains the result of pushing results to DerbyNet
type ResultsPushResult struct {
	Status        string              `json:"status"`
	Message       string              `json:"message,omitempty"`
	WinnersPushed int                 `json:"winners_pushed"`
	Skipped       int                 `json:"skipped"`
	Errors        int                 `json:"errors"`
	Details       []ResultsPushDetail `json:"details,omitempty"`
}

// ResultsPushDetail contains detail for one category's push result
type ResultsPushDetail struct {
	CategoryName string `json:"category_name"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

// PushResultsToDerbyNet pushes voting results to DerbyNet as award winners
func (s *ResultsService) PushResultsToDerbyNet(ctx context.Context, derbyNetURL string) (*ResultsPushResult, error) {
	// Set the URL on the client
	s.client.SetBaseURL(derbyNetURL)

	// Configure credentials for automatic authentication
	derbyNetRole, _ := s.repo.GetSetting(ctx, "derbynet_role")
	derbyNetPassword, _ := s.repo.GetSetting(ctx, "derbynet_password")
	if derbyNetRole != "" && derbyNetPassword != "" {
		s.client.SetCredentials(derbyNetRole, derbyNetPassword)
	}

	// Save DerbyNet URL to settings
	if err := s.repo.SetSetting(ctx, "derbynet_url", derbyNetURL); err != nil {
		return nil, fmt.Errorf("failed to save DerbyNet URL: %w", err)
	}

	// Get winners with DerbyNet IDs
	winners, err := s.repo.GetWinnersForDerbyNet(ctx)
	if err != nil {
		return &ResultsPushResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to get winners: %v", err),
		}, nil
	}

	if len(winners) == 0 {
		return &ResultsPushResult{
			Status:  "success",
			Message: "No winners to push (no votes recorded)",
		}, nil
	}

	s.log.Info("Pushing results to DerbyNet", "count", len(winners))

	result := &ResultsPushResult{Status: "success"}

	for _, w := range winners {
		detail := ResultsPushDetail{CategoryName: w.CategoryName}

		// Check if we have the required DerbyNet IDs
		if w.DerbyNetAwardID == nil {
			detail.Status = "skipped"
			detail.Message = "Category not linked to DerbyNet (sync categories first)"
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}
		if w.DerbyNetRacerID == nil {
			detail.Status = "skipped"
			detail.Message = "Winning car not linked to DerbyNet (sync cars first)"
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}

		// Push to DerbyNet
		err := s.client.SetAwardWinner(ctx, *w.DerbyNetAwardID, *w.DerbyNetRacerID)
		if err != nil {
			s.log.Error("Error pushing winner to DerbyNet",
				"category", w.CategoryName,
				"award_id", *w.DerbyNetAwardID,
				"racer_id", *w.DerbyNetRacerID,
				"error", err)
			detail.Status = "error"
			detail.Message = err.Error()
			result.Errors++
		} else {
			s.log.Info("Pushed winner to DerbyNet",
				"category", w.CategoryName,
				"award_id", *w.DerbyNetAwardID,
				"racer_id", *w.DerbyNetRacerID)
			detail.Status = "success"
			result.WinnersPushed++
		}
		result.Details = append(result.Details, detail)
	}

	if result.Errors > 0 {
		result.Status = "partial"
		result.Message = fmt.Sprintf("%d winners pushed, %d skipped, %d errors", result.WinnersPushed, result.Skipped, result.Errors)
	} else if result.Skipped > 0 {
		result.Message = fmt.Sprintf("%d winners pushed, %d skipped (missing DerbyNet links)", result.WinnersPushed, result.Skipped)
	}

	return result, nil
}

// TieConflict represents a category with tied vote counts
type TieConflict struct {
	CategoryID   int         `json:"category_id"`
	CategoryName string      `json:"category_name"`
	TiedCars     []CarResult `json:"tied_cars"`
}

// MultiWinConflict represents a car winning multiple awards (exceeding group limit)
type MultiWinConflict struct {
	CarID         int      `json:"car_id"`
	CarNumber     string   `json:"car_number"`
	RacerName     string   `json:"racer_name"`
	AwardsWon     []string `json:"awards_won"`
	CategoryIDs   []int    `json:"category_ids"`
	GroupID       *int     `json:"group_id,omitempty"`
	GroupName     string   `json:"group_name,omitempty"`
	MaxWinsPerCar int      `json:"max_wins_per_car"`
}

// DetectTies finds categories where multiple cars share the highest vote count
func (s *ResultsService) DetectTies(ctx context.Context) ([]TieConflict, error) {
	results, err := s.GetResults(ctx)
	if err != nil {
		return nil, err
	}

	var ties []TieConflict
	for _, cat := range results.Categories {
		// Skip categories that already have a manual override
		if cat.HasOverride {
			continue
		}

		if len(cat.Votes) < 2 {
			continue // Need at least 2 cars to have a tie
		}

		// Safe to access [0] because we checked len >= 2 above
		maxVotes := cat.Votes[0].VoteCount

		// Find all cars with the max vote count
		var tiedCars []CarResult
		for _, vote := range cat.Votes {
			if vote.VoteCount == maxVotes {
				tiedCars = append(tiedCars, vote)
			} else {
				break // Votes are sorted DESC, so we can stop
			}
		}

		// If more than one car has max votes, it's a tie
		if len(tiedCars) > 1 {
			ties = append(ties, TieConflict{
				CategoryID:   cat.CategoryID,
				CategoryName: cat.CategoryName,
				TiedCars:     tiedCars,
			})
		}
	}

	return ties, nil
}

// DetectMultipleWins finds cars that won multiple awards exceeding group limits
func (s *ResultsService) DetectMultipleWins(ctx context.Context) ([]MultiWinConflict, error) {
	results, err := s.GetResults(ctx)
	if err != nil {
		return nil, err
	}

	// Get category groups to check max_wins_per_car limits
	groups, err := s.repo.ListCategoryGroups(ctx)
	if err != nil {
		return nil, err
	}

	// Map group_id to max_wins_per_car
	groupLimits := make(map[int]int)
	groupNames := make(map[int]string)
	for _, g := range groups {
		if g.MaxWinsPerCar != nil && *g.MaxWinsPerCar > 0 {
			groupLimits[g.ID] = *g.MaxWinsPerCar
			groupNames[g.ID] = g.Name
		}
	}

	// Track wins per (car_id, group_id)
	type winKey struct {
		carID   int
		groupID int
	}
	carGroupWins := make(map[winKey]struct {
		carNumber   string
		racerName   string
		groupName   string
		awards      []string
		categoryIDs []int
	})

	for _, cat := range results.Categories {
		// Skip categories not in a group or groups without max_wins_per_car limit
		if cat.GroupID == nil {
			continue
		}
		maxWins, hasLimit := groupLimits[*cat.GroupID]
		if !hasLimit {
			continue
		}

		var winnerCarID int
		var winnerCarNumber, winnerRacerName string

		// Check for manual override first
		if cat.HasOverride && cat.OverrideCarID != nil {
			winnerCarID = *cat.OverrideCarID
			// Find car details from votes first
			found := false
			for _, vote := range cat.Votes {
				if vote.CarID == winnerCarID {
					winnerCarNumber = vote.CarNumber
					winnerRacerName = vote.RacerName
					found = true
					break
				}
			}
			if !found {
				// Override car might not have votes in this category, fetch car details
				car, err := s.repo.GetCar(ctx, winnerCarID)
				if err != nil || car == nil {
					continue // Skip if car not found
				}
				winnerCarNumber = car.CarNumber
				winnerRacerName = car.RacerName
			}
		} else {
			// Use vote winner
			if len(cat.Votes) == 0 || cat.Votes[0].VoteCount == 0 {
				continue // No winner
			}
			winnerCarID = cat.Votes[0].CarID
			winnerCarNumber = cat.Votes[0].CarNumber
			winnerRacerName = cat.Votes[0].RacerName
		}

		key := winKey{carID: winnerCarID, groupID: *cat.GroupID}
		entry := carGroupWins[key]
		entry.carNumber = winnerCarNumber
		entry.racerName = winnerRacerName
		entry.groupName = cat.GroupName
		entry.awards = append(entry.awards, cat.CategoryName)
		entry.categoryIDs = append(entry.categoryIDs, cat.CategoryID)
		carGroupWins[key] = entry

		// Check if this car has exceeded the limit for this group
		_ = maxWins // Will use in next loop
	}

	// Find violations where car wins exceed group limit
	var multiWins []MultiWinConflict
	for key, entry := range carGroupWins {
		maxWins := groupLimits[key.groupID]
		if len(entry.awards) > maxWins {
			multiWins = append(multiWins, MultiWinConflict{
				CarID:         key.carID,
				CarNumber:     entry.carNumber,
				RacerName:     entry.racerName,
				AwardsWon:     entry.awards,
				CategoryIDs:   entry.categoryIDs,
				GroupID:       &key.groupID,
				GroupName:     entry.groupName,
				MaxWinsPerCar: maxWins,
			})
		}
	}

	return multiWins, nil
}

// SetManualWinner sets a manual winner override for a category
func (s *ResultsService) SetManualWinner(ctx context.Context, categoryID, carID int, reason string) error {
	// Validate reason is not empty
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("reason cannot be empty")
	}

	// Verify category exists
	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify category: %w", err)
	}
	categoryExists := false
	for _, cat := range categories {
		if cat.ID == categoryID {
			categoryExists = true
			break
		}
	}
	if !categoryExists {
		return fmt.Errorf("category %d not found", categoryID)
	}

	// Verify car exists
	cars, err := s.repo.ListCars(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify car: %w", err)
	}
	carExists := false
	for _, car := range cars {
		if car.ID == carID {
			carExists = true
			break
		}
	}
	if !carExists {
		return fmt.Errorf("car %d not found", carID)
	}

	return s.repo.SetManualWinner(ctx, categoryID, carID, reason)
}

// ClearManualWinner removes the manual winner override for a category
func (s *ResultsService) ClearManualWinner(ctx context.Context, categoryID int) error {
	return s.repo.ClearManualWinner(ctx, categoryID)
}

// GetFinalWinners returns the winner for each category, respecting manual overrides
func (s *ResultsService) GetFinalWinners(ctx context.Context) ([]map[string]interface{}, error) {
	// Get categories (includes override fields)
	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	// Get vote results
	results, err := s.GetResults(ctx)
	if err != nil {
		return nil, err
	}

	// Map category results by ID for easy lookup
	resultsByCategory := make(map[int]*CategoryResult)
	for i := range results.Categories {
		cat := &results.Categories[i]
		resultsByCategory[cat.CategoryID] = cat
	}

	var winners []map[string]interface{}
	for _, cat := range categories {
		var winner map[string]interface{}

		// Check if there's a manual override
		if cat.OverrideWinnerCarID != nil {
			// Use the manual override
			// We need to get car details - find in the category results
			catResult := resultsByCategory[cat.ID]
			if catResult != nil {
				// Find the car in the votes
				for _, vote := range catResult.Votes {
					if vote.CarID == *cat.OverrideWinnerCarID {
						winner = map[string]interface{}{
							"car_id":     vote.CarID,
							"car_number": vote.CarNumber,
							"car_name":   vote.CarName,
							"racer_name": vote.RacerName,
							"vote_count": vote.VoteCount,
							"is_override": true,
							"override_reason": cat.OverrideReason,
						}
						break
					}
				}
			}
		} else {
			// Use vote-based winner
			catResult := resultsByCategory[cat.ID]
			if catResult != nil && len(catResult.Votes) > 0 && catResult.Votes[0].VoteCount > 0 {
				vote := catResult.Votes[0]
				winner = map[string]interface{}{
					"car_id":     vote.CarID,
					"car_number": vote.CarNumber,
					"car_name":   vote.CarName,
					"racer_name": vote.RacerName,
					"vote_count": vote.VoteCount,
					"is_override": false,
				}
			}
		}

		if winner != nil {
			winners = append(winners, map[string]interface{}{
				"category_id":   cat.ID,
				"category_name": cat.Name,
				"winner":        winner,
			})
		}
	}

	return winners, nil
}
