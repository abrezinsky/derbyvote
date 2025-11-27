package repository

import (
	"context"

	"github.com/abrezinsky/derbyvote/internal/models"
)

// CategoryRepository defines category data operations
type CategoryRepository interface {
	ListCategories(ctx context.Context) ([]models.Category, error)
	ListAllCategories(ctx context.Context) ([]map[string]interface{}, error)
	CreateCategory(ctx context.Context, name string, displayOrder int, groupID *int, allowedVoterTypes []string, allowedRanks []string) (int64, error)
	UpdateCategory(ctx context.Context, id int, name string, displayOrder int, groupID *int, allowedVoterTypes []string, allowedRanks []string, active bool) error
	DeleteCategory(ctx context.Context, id int) error
	CategoryExists(ctx context.Context, name string) (bool, error)
	UpsertCategory(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (created bool, err error)
	SetManualWinner(ctx context.Context, categoryID, carID int, reason string) error
	ClearManualWinner(ctx context.Context, categoryID int) error
	ListCategoryGroups(ctx context.Context) ([]models.CategoryGroup, error)
	GetCategoryGroup(ctx context.Context, id string) (*models.CategoryGroup, error)
	CreateCategoryGroup(ctx context.Context, name, description string, exclusivityPoolID *int, maxWinsPerCar *int, displayOrder int) (int64, error)
	UpdateCategoryGroup(ctx context.Context, id string, name, description string, exclusivityPoolID *int, maxWinsPerCar *int, displayOrder int) error
	DeleteCategoryGroup(ctx context.Context, id string) error
}

// VoterRepository defines voter data operations
type VoterRepository interface {
	ListVoters(ctx context.Context) ([]map[string]interface{}, error)
	GetVoterByQR(ctx context.Context, qrCode string) (int, error)
	GetVoterByQRCode(ctx context.Context, qrCode string) (int64, bool, error)
	GetVoterQRCode(ctx context.Context, id int) (string, error)
	GetVoterType(ctx context.Context, voterID int) (string, error)
	CreateVoter(ctx context.Context, qrCode string) (int, error)
	CreateVoterFull(ctx context.Context, carID *int, name, email, voterType, qrCode, notes string) (int64, error)
	UpdateVoter(ctx context.Context, id int, carID *int, name, email, voterType, notes string) error
	DeleteVoter(ctx context.Context, id int) error
	InsertVoterIgnore(ctx context.Context, qrCode string) error
	UpsertVoterForCar(ctx context.Context, carID int64, name, qrCode string) error
}

// CarRepository defines car data operations
type CarRepository interface {
	ListCars(ctx context.Context) ([]models.Car, error)
	ListEligibleCars(ctx context.Context) ([]models.Car, error)
	GetCar(ctx context.Context, id int) (*models.Car, error)
	GetCarByDerbyNetID(ctx context.Context, racerID int) (int64, bool, error)
	UpsertCar(ctx context.Context, derbynetRacerID int, carNumber, racerName, carName, photoURL, rank string) error
	CarExists(ctx context.Context, carNumber string) (bool, error)
	CreateCar(ctx context.Context, carNumber, racerName, carName, photoURL string) error
	UpdateCar(ctx context.Context, id int, carNumber, racerName, carName, photoURL, rank string) error
	SetCarEligibility(ctx context.Context, id int, eligible bool) error
	DeleteCar(ctx context.Context, id int) error
	CountVotesForCar(ctx context.Context, carID int) (int, error)
}

// VoteRepository defines vote data operations
type VoteRepository interface {
	GetVoterVotes(ctx context.Context, voterID int) (map[int]int, error)
	SaveVote(ctx context.Context, voterID, categoryID, carID int) error
	GetExclusivityPoolID(ctx context.Context, categoryID int) (int64, bool, error)
	FindConflictingVote(ctx context.Context, voterID, carID, categoryID int, poolID int64) (int, string, bool, error)
	ClearConflictingVote(ctx context.Context, voterID, categoryID, carID int) error
	GetVoteResults(ctx context.Context) (map[int]map[int]int, error)
	GetVoteResultsWithCars(ctx context.Context) ([]VoteResultRow, error)
	GetWinnersForDerbyNet(ctx context.Context) ([]WinnerForDerbyNet, error)
	CountVotesForCategory(ctx context.Context, categoryID int) (int, error)
}

// SettingsRepository defines settings data operations
type SettingsRepository interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	GetVotingStats(ctx context.Context) (map[string]interface{}, error)
	ClearTable(ctx context.Context, table string) error
}

// FullRepository combines all repository interfaces
// Use this when a service needs access to multiple domains
type FullRepository interface {
	CategoryRepository
	VoterRepository
	CarRepository
	VoteRepository
	SettingsRepository
}

// Ensure Repository implements all interfaces
var _ FullRepository = (*Repository)(nil)
