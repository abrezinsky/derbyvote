package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/models"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// CarServiceRepository defines the repository methods needed by CarService
type CarServiceRepository interface {
	repository.CarRepository
	repository.VoterRepository
	repository.SettingsRepository
}

// CarService handles car-related business logic
type CarService struct {
	log    logger.Logger
	repo   CarServiceRepository
	client derbynet.Client
}

// NewCarService creates a new CarService
func NewCarService(log logger.Logger, repo CarServiceRepository, client derbynet.Client) *CarService {
	return &CarService{log: log, repo: repo, client: client}
}

// SyncResult contains the result of a DerbyNet sync
type SyncResult struct {
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
	CarsCreated   int    `json:"cars_created"`
	CarsUpdated   int    `json:"cars_updated"`
	VotersCreated int    `json:"voters_created"`
	VotersUpdated int    `json:"voters_updated"`
	TotalCars     int    `json:"total_cars"`
	TotalVoters   int    `json:"total_voters"`
	TotalRacers   int    `json:"total_racers"`
}

// PhotoData contains photo metadata and content
type PhotoData struct {
	Data        []byte
	ContentType string
}

// ListCars returns all active cars
func (s *CarService) ListCars(ctx context.Context) ([]models.Car, error) {
	return s.repo.ListCars(ctx)
}

// GetCar returns a car by ID
func (s *CarService) GetCar(ctx context.Context, id int) (*models.Car, error) {
	return s.repo.GetCar(ctx, id)
}

// CreateCar creates a new car
func (s *CarService) CreateCar(ctx context.Context, carNumber, racerName, carName, photoURL string) error {
	return s.repo.CreateCar(ctx, carNumber, racerName, carName, photoURL)
}

// UpdateCar updates a car
func (s *CarService) UpdateCar(ctx context.Context, id int, carNumber, racerName, carName, photoURL, rank string) error {
	return s.repo.UpdateCar(ctx, id, carNumber, racerName, carName, photoURL, rank)
}

// DeleteCar soft deletes a car
func (s *CarService) DeleteCar(ctx context.Context, id int) error {
	return s.repo.DeleteCar(ctx, id)
}

// ListEligibleCars returns all active and eligible cars (for voting)
func (s *CarService) ListEligibleCars(ctx context.Context) ([]models.Car, error) {
	return s.repo.ListEligibleCars(ctx)
}

// SetCarEligibility updates a car's eligibility for voting
func (s *CarService) SetCarEligibility(ctx context.Context, id int, eligible bool) error {
	return s.repo.SetCarEligibility(ctx, id, eligible)
}

// CountVotesForCar returns the number of votes a car has received
func (s *CarService) CountVotesForCar(ctx context.Context, carID int) (int, error) {
	return s.repo.CountVotesForCar(ctx, carID)
}

// GetCarPhoto fetches the photo for a car, returning nil if photo is unavailable
func (s *CarService) GetCarPhoto(ctx context.Context, id int) (*PhotoData, error) {
	// Get car from database
	car, err := s.repo.GetCar(ctx, id)
	if err != nil || car.PhotoURL == "" {
		return nil, fmt.Errorf("car photo not available")
	}

	// Fetch the photo from the source
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(car.PhotoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch photo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("photo fetch returned status %d", resp.StatusCode)
	}

	// Read the photo data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read photo data: %w", err)
	}

	// Get content type, default to image/jpeg if not specified
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	return &PhotoData{
		Data:        data,
		ContentType: contentType,
	}, nil
}

// SyncFromDerbyNet syncs cars from DerbyNet using the provided URL
func (s *CarService) SyncFromDerbyNet(ctx context.Context, derbyNetURL string) (*SyncResult, error) {
	// Set the URL on the client
	s.client.SetBaseURL(derbyNetURL)
	baseURL := derbyNetURL

	// Save DerbyNet URL to settings
	if err := s.repo.SetSetting(ctx, "derbynet_url", baseURL); err != nil {
		return nil, fmt.Errorf("failed to save DerbyNet URL: %w", err)
	}

	// Fetch racers from DerbyNet
	racers, err := s.client.FetchRacers(ctx)
	if err != nil {
		return &SyncResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to fetch from DerbyNet: %v", err),
		}, nil
	}

	s.log.Info("Fetched racers from DerbyNet", "count", len(racers))

	// Process racers
	result := &SyncResult{Status: "success", TotalRacers: len(racers)}
	var firstError error

	for _, racer := range racers {
		racerName := fmt.Sprintf("%s %s", racer.FirstName, racer.LastName)
		carNumber := fmt.Sprintf("%d", racer.CarNumber)
		photoURL := ""
		if racer.CarPhoto != "" {
			cleanBaseURL := strings.TrimSuffix(baseURL, "/")
			photoPath := strings.TrimPrefix(racer.CarPhoto, "/")
			photoURL = fmt.Sprintf("%s/%s", cleanBaseURL, photoPath)
		}

		// Check if car already exists
		_, carExisted, err := s.repo.GetCarByDerbyNetID(ctx, racer.RacerID)
		if err != nil {
			s.log.Error("Error checking car", "racer_id", racer.RacerID, "error", err)
			if firstError == nil {
				firstError = fmt.Errorf("failed to check car for racer %d: %w", racer.RacerID, err)
			}
			continue
		}

		// Get rank from DerbyNet
		rank := racer.Rank

		// Upsert car
		if err := s.repo.UpsertCar(ctx, racer.RacerID, carNumber, racerName, racer.CarName.String(), photoURL, rank); err != nil {
			s.log.Error("Error syncing racer", "racer_id", racer.RacerID, "name", racerName, "error", err)
			if firstError == nil {
				firstError = fmt.Errorf("failed to sync racer %d: %w", racer.RacerID, err)
			}
			continue
		}

		if carExisted {
			result.CarsUpdated++
		} else {
			result.CarsCreated++
		}

		// Get the car ID
		carID, _, err := s.repo.GetCarByDerbyNetID(ctx, racer.RacerID)
		if err != nil {
			s.log.Error("Error getting car ID for racer", "racer_id", racer.RacerID, "error", err)
			if firstError == nil {
				firstError = fmt.Errorf("failed to get car ID for racer %d: %w", racer.RacerID, err)
			}
			continue
		}

		// Generate QR code for voter
		qrCode := GenerateReadableCode(fmt.Sprintf("car-%d-%d", racer.RacerID, carID))

		// Check if voter exists
		_, voterExisted, err := s.repo.GetVoterByQRCode(ctx, qrCode)
		if err != nil {
			s.log.Error("Error checking voter for racer", "racer_id", racer.RacerID, "error", err)
			if firstError == nil {
				firstError = fmt.Errorf("failed to check voter for racer %d: %w", racer.RacerID, err)
			}
			continue
		}

		// Upsert voter
		if err := s.repo.UpsertVoterForCar(ctx, carID, racerName, qrCode); err != nil {
			s.log.Error("Error creating/updating voter for racer", "racer_id", racer.RacerID, "name", racerName, "error", err)
			if firstError == nil {
				firstError = fmt.Errorf("failed to upsert voter for racer %d: %w", racer.RacerID, err)
			}
		} else {
			if voterExisted {
				result.VotersUpdated++
			} else {
				result.VotersCreated++
			}
		}
	}

	result.TotalCars = result.CarsCreated + result.CarsUpdated
	result.TotalVoters = result.VotersCreated + result.VotersUpdated

	s.log.Info("Sync complete", "cars_created", result.CarsCreated, "cars_updated", result.CarsUpdated,
		"voters_created", result.VotersCreated, "voters_updated", result.VotersUpdated)

	return result, firstError
}

// SeedMockCars seeds mock car data
func (s *CarService) SeedMockCars(ctx context.Context) (int, error) {
	mockCars := []struct {
		CarNumber string
		RacerName string
		CarName   string
		PhotoURL  string
	}{
		{"101", "Alex Johnson", "Lightning Bolt", "https://placehold.co/300x300/3b82f6/ffffff?text=101"},
		{"102", "Sarah Williams", "Red Rocket", "https://placehold.co/300x300/ef4444/ffffff?text=102"},
		{"103", "Mike Chen", "Blue Thunder", "https://placehold.co/300x300/3b82f6/ffffff?text=103"},
		{"104", "Emma Davis", "Pink Panther", "https://placehold.co/300x300/ec4899/ffffff?text=104"},
		{"105", "James Brown", "Green Machine", "https://placehold.co/300x300/22c55e/ffffff?text=105"},
		{"106", "Olivia Martinez", "Purple Haze", "https://placehold.co/300x300/a855f7/ffffff?text=106"},
		{"107", "Noah Wilson", "Golden Arrow", "https://placehold.co/300x300/eab308/ffffff?text=107"},
		{"108", "Sophia Garcia", "Silver Bullet", "https://placehold.co/300x300/94a3b8/ffffff?text=108"},
		{"109", "Liam Anderson", "Black Hawk", "https://placehold.co/300x300/1f2937/ffffff?text=109"},
		{"110", "Ava Taylor", "White Lightning", "https://placehold.co/300x300/f3f4f6/1f2937?text=110"},
		{"111", "Ethan Thomas", "Orange Crush", "https://placehold.co/300x300/f97316/ffffff?text=111"},
		{"112", "Isabella Moore", "Teal Dream", "https://placehold.co/300x300/14b8a6/ffffff?text=112"},
		{"113", "Mason Jackson", "Crimson Comet", "https://placehold.co/300x300/dc2626/ffffff?text=113"},
		{"114", "Mia White", "Indigo Star", "https://placehold.co/300x300/6366f1/ffffff?text=114"},
		{"115", "Lucas Harris", "Lime Streak", "https://placehold.co/300x300/84cc16/ffffff?text=115"},
		{"116", "Charlotte Martin", "Magenta Magic", "https://placehold.co/300x300/db2777/ffffff?text=116"},
		{"117", "Benjamin Lee", "Cyan Speed", "https://placehold.co/300x300/06b6d4/ffffff?text=117"},
		{"118", "Amelia Walker", "Bronze Blaze", "https://placehold.co/300x300/92400e/ffffff?text=118"},
		{"119", "Henry Hall", "Emerald Express", "https://placehold.co/300x300/059669/ffffff?text=119"},
		{"120", "Harper Allen", "Ruby Racer", "https://placehold.co/300x300/be123c/ffffff?text=120"},
	}

	var addedCount int
	var firstError error
	for _, car := range mockCars {
		exists, err := s.repo.CarExists(ctx, car.CarNumber)
		if err != nil {
			s.log.Error("Error checking car", "car_number", car.CarNumber, "error", err)
			if firstError == nil {
				firstError = fmt.Errorf("failed to check if car exists: %w", err)
			}
			continue
		}
		if !exists {
			if err := s.repo.CreateCar(ctx, car.CarNumber, car.RacerName, car.CarName, car.PhotoURL); err != nil {
				s.log.Error("Error seeding car", "car_number", car.CarNumber, "error", err)
				if firstError == nil {
					firstError = fmt.Errorf("failed to create car %q: %w", car.CarNumber, err)
				}
			} else {
				addedCount++
			}
		}
	}

	return addedCount, firstError
}
