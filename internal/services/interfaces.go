package services

import (
	"context"

	"github.com/abrezinsky/derbyvote/internal/models"
)

// CategoryServicer defines the interface for category operations
type CategoryServicer interface {
	ListCategories(ctx context.Context) ([]models.Category, error)
	ListAllCategories(ctx context.Context) ([]map[string]interface{}, error)
	CreateCategory(ctx context.Context, cat Category) (int64, error)
	UpdateCategory(ctx context.Context, id int, cat Category) error
	DeleteCategory(ctx context.Context, id int) error
	CountVotesForCategory(ctx context.Context, categoryID int) (int, error)
	ListGroups(ctx context.Context) ([]models.CategoryGroup, error)
	GetGroup(ctx context.Context, id string) (*models.CategoryGroup, error)
	CreateGroup(ctx context.Context, group CategoryGroup) (int64, error)
	UpdateGroup(ctx context.Context, id string, group CategoryGroup) error
	DeleteGroup(ctx context.Context, id string) error
	SeedMockCategories(ctx context.Context) (int, error)
	SyncFromDerbyNet(ctx context.Context, derbyNetURL string) (*CategorySyncResult, error)
}

// CarServicer defines the interface for car operations
type CarServicer interface {
	ListCars(ctx context.Context) ([]models.Car, error)
	ListEligibleCars(ctx context.Context) ([]models.Car, error)
	GetCar(ctx context.Context, id int) (*models.Car, error)
	GetCarPhoto(ctx context.Context, id int) (*PhotoData, error)
	CreateCar(ctx context.Context, carNumber, racerName, carName, photoURL string) error
	UpdateCar(ctx context.Context, id int, carNumber, racerName, carName, photoURL, rank string) error
	SetCarEligibility(ctx context.Context, id int, eligible bool) error
	DeleteCar(ctx context.Context, id int) error
	CountVotesForCar(ctx context.Context, carID int) (int, error)
	SyncFromDerbyNet(ctx context.Context, derbyNetURL string) (*SyncResult, error)
	SeedMockCars(ctx context.Context) (int, error)
}

// VoterServicer defines the interface for voter operations
type VoterServicer interface {
	ListVoters(ctx context.Context) ([]map[string]interface{}, error)
	CreateVoter(ctx context.Context, voter Voter) (int64, string, error)
	UpdateVoter(ctx context.Context, voter Voter) error
	DeleteVoter(ctx context.Context, id int) error
	GenerateQRCodes(ctx context.Context, count int) ([]string, error)
	GenerateQRImage(ctx context.Context, voterID int) ([]byte, error)
	GenerateUniqueCode(ctx context.Context) (string, error)
	GenerateDynamicQRImage(ctx context.Context) ([]byte, error)
}

// VotingServicer defines the interface for voting operations
type VotingServicer interface {
	GetVoteData(ctx context.Context, qrCode string) (*VoteData, error)
	GetOrCreateVoter(ctx context.Context, qrCode string) (int, error)
	SubmitVote(ctx context.Context, vote models.Vote) (*VoteResult, error)
}

// SettingsServicer defines the interface for settings operations
type SettingsServicer interface {
	IsVotingOpen(ctx context.Context) (bool, error)
	SetVotingOpen(ctx context.Context, open bool) error
	GetDerbyNetURL(ctx context.Context) (string, error)
	SetDerbyNetURL(ctx context.Context, url string) error
	GetBaseURL(ctx context.Context) (string, error)
	SetBaseURL(ctx context.Context, url string) error
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	GetTimerEndTime(ctx context.Context) (int64, error)
	SetTimerEndTime(ctx context.Context, endTime int64) error
	ClearTimer(ctx context.Context) error
	AllSettings(ctx context.Context) (map[string]interface{}, error)
	OpenVoting(ctx context.Context) error
	CloseVoting(ctx context.Context) error
	StartVotingTimer(ctx context.Context, minutes int) (string, error)
	UpdateSettings(ctx context.Context, settings Settings) error
	ResetTables(ctx context.Context, tables []string) (*ResetTablesResult, error)
	SetBroadcaster(b Broadcaster)
	RequireRegisteredQR(ctx context.Context) (bool, error)
	GetVoterTypes(ctx context.Context) ([]string, error)
	SetVoterTypes(ctx context.Context, types []string) error
}

// ResultsServicer defines the interface for results operations
type ResultsServicer interface {
	GetResults(ctx context.Context) (*FullResults, error)
	GetCategoryResults(ctx context.Context, categoryID int) (*CategoryResult, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
	GetWinners(ctx context.Context) ([]map[string]interface{}, error)
	GetFinalWinners(ctx context.Context) ([]map[string]interface{}, error)
	PushResultsToDerbyNet(ctx context.Context, derbyNetURL string) (*ResultsPushResult, error)
	DetectTies(ctx context.Context) ([]TieConflict, error)
	DetectMultipleWins(ctx context.Context) ([]MultiWinConflict, error)
	SetManualWinner(ctx context.Context, categoryID, carID int, reason string) error
	ClearManualWinner(ctx context.Context, categoryID int) error
}

// Ensure concrete types implement interfaces
var (
	_ CategoryServicer = (*CategoryService)(nil)
	_ CarServicer      = (*CarService)(nil)
	_ VoterServicer    = (*VoterService)(nil)
	_ VotingServicer   = (*VotingService)(nil)
	_ SettingsServicer = (*SettingsService)(nil)
	_ ResultsServicer  = (*ResultsService)(nil)
)
