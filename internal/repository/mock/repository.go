package mock

import (
	"context"

	"github.com/abrezinsky/derbyvote/internal/models"
	"github.com/abrezinsky/derbyvote/internal/repository"
)

// Repository wraps a real repository and allows injecting errors for testing.
// This provides a flexible way to test error paths without complex database manipulation.
//
// Usage:
//
//	realRepo := testutil.NewTestRepository(t)
//	mockRepo := mock.NewRepository(realRepo)
//	mockRepo.UpsertCategoryError = errors.New("database error")
//	svc := services.NewCategoryService(log, mockRepo, mockClient)
//	_, err := svc.SyncFromDerbyNet(ctx, "http://derbynet.local")
//	// err will now contain the injected error
type Repository struct {
	repository.FullRepository

	// ===== Category Errors =====
	UpsertCategoryError      error
	ListCategoriesError      error
	CategoryExistsError      error
	CreateCategoryError      error
	DeleteCategoryError      error
	GetCategoryGroupError    error
	UpdateCategoryGroupError error
	DeleteCategoryGroupError error
	ListCategoryGroupsError  error

	// ===== Car Errors =====
	CarExistsError          error
	CreateCarError          error
	GetCarByDerbyNetIDError error
	UpsertCarError          error
	DeleteCarError          error
	SetCarEligibilityError  error

	// ===== Voter Errors =====
	GetVoterByQRCodeError   error
	GetVoterByQRError       error
	UpsertVoterForCarError  error
	InsertVoterIgnoreError  error
	GetVoterQRCodeError     error
	GetVoterTypeError       error

	// ===== Settings Errors =====
	GetSettingError error
	SetSettingError error
	ClearTableError error

	// ===== Vote Errors =====
	ListEligibleCarsError       error
	GetVoterVotesError          error
	SaveVoteError               error
	GetVoteResultsError         error
	GetExclusivityPoolIDError   error
	ClearConflictingVoteError   error
	GetCarError                 error
	CreateVoterError            error
	CountVotesForCarError       error
	CountVotesForCategoryError  error

	// ===== Results Errors =====
	ListCarsError               error
	UpdateCarError              error
	GetVoteResultsWithCarsError error
	GetVotingStatsError         error
	GetWinnersForDerbyNetError  error
	ClearManualWinnerError      error
}

// NewRepository creates a mock repository wrapping a real one
func NewRepository(real repository.FullRepository) *Repository {
	return &Repository{
		FullRepository: real,
	}
}

// ===== Category Methods =====

func (m *Repository) UpsertCategory(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (bool, error) {
	if m.UpsertCategoryError != nil {
		return false, m.UpsertCategoryError
	}
	return m.FullRepository.UpsertCategory(ctx, name, displayOrder, derbynetAwardID)
}

func (m *Repository) ListCategories(ctx context.Context) ([]models.Category, error) {
	if m.ListCategoriesError != nil {
		return nil, m.ListCategoriesError
	}
	return m.FullRepository.ListCategories(ctx)
}

func (m *Repository) CategoryExists(ctx context.Context, name string) (bool, error) {
	if m.CategoryExistsError != nil {
		return false, m.CategoryExistsError
	}
	return m.FullRepository.CategoryExists(ctx, name)
}

func (m *Repository) CreateCategory(ctx context.Context, name string, displayOrder int, groupID *int, allowedVoterTypes []string, allowedRanks []string) (int64, error) {
	if m.CreateCategoryError != nil {
		return 0, m.CreateCategoryError
	}
	return m.FullRepository.CreateCategory(ctx, name, displayOrder, groupID, allowedVoterTypes, allowedRanks)
}

func (m *Repository) DeleteCategory(ctx context.Context, id int) error {
	if m.DeleteCategoryError != nil {
		return m.DeleteCategoryError
	}
	return m.FullRepository.DeleteCategory(ctx, id)
}

func (m *Repository) ListCategoryGroups(ctx context.Context) ([]models.CategoryGroup, error) {
	if m.ListCategoryGroupsError != nil {
		return nil, m.ListCategoryGroupsError
	}
	return m.FullRepository.ListCategoryGroups(ctx)
}

// ===== Car Methods =====

func (m *Repository) CarExists(ctx context.Context, carNumber string) (bool, error) {
	if m.CarExistsError != nil {
		return false, m.CarExistsError
	}
	return m.FullRepository.CarExists(ctx, carNumber)
}

func (m *Repository) CreateCar(ctx context.Context, carNumber, racerName, carName, photoURL string) error {
	if m.CreateCarError != nil {
		return m.CreateCarError
	}
	return m.FullRepository.CreateCar(ctx, carNumber, racerName, carName, photoURL)
}

func (m *Repository) GetCarByDerbyNetID(ctx context.Context, derbyNetID int) (int64, bool, error) {
	if m.GetCarByDerbyNetIDError != nil {
		return 0, false, m.GetCarByDerbyNetIDError
	}
	return m.FullRepository.GetCarByDerbyNetID(ctx, derbyNetID)
}

func (m *Repository) UpsertCar(ctx context.Context, derbyNetID int, carNumber, racerName, carName, photoURL, rank string) error {
	if m.UpsertCarError != nil {
		return m.UpsertCarError
	}
	return m.FullRepository.UpsertCar(ctx, derbyNetID, carNumber, racerName, carName, photoURL, rank)
}

// ===== Voter Methods =====

func (m *Repository) GetVoterByQRCode(ctx context.Context, qrCode string) (int64, bool, error) {
	if m.GetVoterByQRCodeError != nil {
		return 0, false, m.GetVoterByQRCodeError
	}
	return m.FullRepository.GetVoterByQRCode(ctx, qrCode)
}

func (m *Repository) UpsertVoterForCar(ctx context.Context, carID int64, name, qrCode string) error {
	if m.UpsertVoterForCarError != nil {
		return m.UpsertVoterForCarError
	}
	return m.FullRepository.UpsertVoterForCar(ctx, carID, name, qrCode)
}

// ===== Settings Methods =====

func (m *Repository) GetSetting(ctx context.Context, key string) (string, error) {
	if m.GetSettingError != nil {
		return "", m.GetSettingError
	}
	return m.FullRepository.GetSetting(ctx, key)
}

func (m *Repository) SetSetting(ctx context.Context, key, value string) error {
	if m.SetSettingError != nil {
		return m.SetSettingError
	}
	return m.FullRepository.SetSetting(ctx, key, value)
}

// ===== Vote Methods =====

func (m *Repository) ListEligibleCars(ctx context.Context) ([]models.Car, error) {
	if m.ListEligibleCarsError != nil {
		return nil, m.ListEligibleCarsError
	}
	return m.FullRepository.ListEligibleCars(ctx)
}

func (m *Repository) GetVoterVotes(ctx context.Context, voterID int) (map[int]int, error) {
	if m.GetVoterVotesError != nil {
		return nil, m.GetVoterVotesError
	}
	return m.FullRepository.GetVoterVotes(ctx, voterID)
}

func (m *Repository) SaveVote(ctx context.Context, voterID, categoryID, carID int) error {
	if m.SaveVoteError != nil {
		return m.SaveVoteError
	}
	return m.FullRepository.SaveVote(ctx, voterID, categoryID, carID)
}

func (m *Repository) GetVoteResults(ctx context.Context) (map[int]map[int]int, error) {
	if m.GetVoteResultsError != nil {
		return nil, m.GetVoteResultsError
	}
	return m.FullRepository.GetVoteResults(ctx)
}

// ===== Car Methods (Additional) =====

func (m *Repository) ListCars(ctx context.Context) ([]models.Car, error) {
	if m.ListCarsError != nil {
		return nil, m.ListCarsError
	}
	return m.FullRepository.ListCars(ctx)
}

func (m *Repository) UpdateCar(ctx context.Context, id int, carNumber, racerName, carName, photoURL, rank string) error {
	if m.UpdateCarError != nil {
		return m.UpdateCarError
	}
	return m.FullRepository.UpdateCar(ctx, id, carNumber, racerName, carName, photoURL, rank)
}

func (m *Repository) GetExclusivityPoolID(ctx context.Context, categoryID int) (int64, bool, error) {
	if m.GetExclusivityPoolIDError != nil {
		return 0, false, m.GetExclusivityPoolIDError
	}
	return m.FullRepository.GetExclusivityPoolID(ctx, categoryID)
}

func (m *Repository) ClearConflictingVote(ctx context.Context, voterID, conflictCategoryID, carID int) error {
	if m.ClearConflictingVoteError != nil {
		return m.ClearConflictingVoteError
	}
	return m.FullRepository.ClearConflictingVote(ctx, voterID, conflictCategoryID, carID)
}

func (m *Repository) GetCar(ctx context.Context, carID int) (*models.Car, error) {
	if m.GetCarError != nil {
		return nil, m.GetCarError
	}
	return m.FullRepository.GetCar(ctx, carID)
}

func (m *Repository) CreateVoter(ctx context.Context, qrCode string) (int, error) {
	if m.CreateVoterError != nil {
		return 0, m.CreateVoterError
	}
	return m.FullRepository.CreateVoter(ctx, qrCode)
}

func (m *Repository) GetVoterByQR(ctx context.Context, qrCode string) (int, error) {
	if m.GetVoterByQRError != nil {
		return 0, m.GetVoterByQRError
	}
	return m.FullRepository.GetVoterByQR(ctx, qrCode)
}

func (m *Repository) GetVoterType(ctx context.Context, voterID int) (string, error) {
	if m.GetVoterTypeError != nil {
		return "", m.GetVoterTypeError
	}
	return m.FullRepository.GetVoterType(ctx, voterID)
}

func (m *Repository) GetVoteResultsWithCars(ctx context.Context) ([]repository.VoteResultRow, error) {
	if m.GetVoteResultsWithCarsError != nil {
		return nil, m.GetVoteResultsWithCarsError
	}
	return m.FullRepository.GetVoteResultsWithCars(ctx)
}

func (m *Repository) GetVotingStats(ctx context.Context) (map[string]interface{}, error) {
	if m.GetVotingStatsError != nil {
		return nil, m.GetVotingStatsError
	}
	return m.FullRepository.GetVotingStats(ctx)
}

func (m *Repository) GetWinnersForDerbyNet(ctx context.Context) ([]repository.WinnerForDerbyNet, error) {
	if m.GetWinnersForDerbyNetError != nil {
		return nil, m.GetWinnersForDerbyNetError
	}
	return m.FullRepository.GetWinnersForDerbyNet(ctx)
}

func (m *Repository) InsertVoterIgnore(ctx context.Context, qrCode string) error {
	if m.InsertVoterIgnoreError != nil {
		return m.InsertVoterIgnoreError
	}
	return m.FullRepository.InsertVoterIgnore(ctx, qrCode)
}

func (m *Repository) GetVoterQRCode(ctx context.Context, voterID int) (string, error) {
	if m.GetVoterQRCodeError != nil {
		return "", m.GetVoterQRCodeError
	}
	return m.FullRepository.GetVoterQRCode(ctx, voterID)
}

func (m *Repository) DeleteCar(ctx context.Context, id int) error {
	if m.DeleteCarError != nil {
		return m.DeleteCarError
	}
	return m.FullRepository.DeleteCar(ctx, id)
}

func (m *Repository) SetCarEligibility(ctx context.Context, id int, eligible bool) error {
	if m.SetCarEligibilityError != nil {
		return m.SetCarEligibilityError
	}
	return m.FullRepository.SetCarEligibility(ctx, id, eligible)
}

func (m *Repository) GetCategoryGroup(ctx context.Context, id string) (*models.CategoryGroup, error) {
	if m.GetCategoryGroupError != nil {
		return nil, m.GetCategoryGroupError
	}
	return m.FullRepository.GetCategoryGroup(ctx, id)
}

func (m *Repository) UpdateCategoryGroup(ctx context.Context, id, name, description string, exclusivityPoolID *int, maxWinsPerCar *int, displayOrder int) error {
	if m.UpdateCategoryGroupError != nil {
		return m.UpdateCategoryGroupError
	}
	return m.FullRepository.UpdateCategoryGroup(ctx, id, name, description, exclusivityPoolID, maxWinsPerCar, displayOrder)
}

func (m *Repository) DeleteCategoryGroup(ctx context.Context, id string) error {
	if m.DeleteCategoryGroupError != nil {
		return m.DeleteCategoryGroupError
	}
	return m.FullRepository.DeleteCategoryGroup(ctx, id)
}

func (m *Repository) ClearTable(ctx context.Context, table string) error {
	if m.ClearTableError != nil {
		return m.ClearTableError
	}
	return m.FullRepository.ClearTable(ctx, table)
}

func (m *Repository) ClearManualWinner(ctx context.Context, categoryID int) error {
	if m.ClearManualWinnerError != nil {
		return m.ClearManualWinnerError
	}
	return m.FullRepository.ClearManualWinner(ctx, categoryID)
}

func (m *Repository) CountVotesForCar(ctx context.Context, carID int) (int, error) {
	if m.CountVotesForCarError != nil {
		return 0, m.CountVotesForCarError
	}
	return m.FullRepository.CountVotesForCar(ctx, carID)
}

func (m *Repository) CountVotesForCategory(ctx context.Context, categoryID int) (int, error) {
	if m.CountVotesForCategoryError != nil {
		return 0, m.CountVotesForCategoryError
	}
	return m.FullRepository.CountVotesForCategory(ctx, categoryID)
}
