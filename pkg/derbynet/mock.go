package derbynet

import (
	"context"
	"fmt"
)

// MockClient is a mock DerbyNet client for testing
type MockClient struct {
	racers           []Racer
	awards           []Award
	awardTypes       []AwardType
	baseURL          string
	fetchErr         error
	awardsErr        error
	awardTypesErr    error
	createAwardErr   error
	setWinnerErr     error
	loginErr         error
	awardWinners     map[int]int // awardID -> racerID
	nextAwardID      int         // counter for generating new award IDs
	credentialsSet   bool        // tracks if SetCredentials was called
}

// MockOption configures the mock client
type MockOption func(*MockClient)

// WithRacers sets the racers to return
func WithRacers(racers []Racer) MockOption {
	return func(m *MockClient) {
		m.racers = racers
	}
}

// WithFetchError sets an error to return from FetchRacers
func WithFetchError(err error) MockOption {
	return func(m *MockClient) {
		m.fetchErr = err
	}
}

// WithAwards sets the awards to return
func WithAwards(awards []Award) MockOption {
	return func(m *MockClient) {
		m.awards = awards
	}
}

// WithAwardsError sets an error to return from FetchAwards
func WithAwardsError(err error) MockOption {
	return func(m *MockClient) {
		m.awardsErr = err
	}
}

// WithSetWinnerError sets an error to return from SetAwardWinner
func WithSetWinnerError(err error) MockOption {
	return func(m *MockClient) {
		m.setWinnerErr = err
	}
}

// WithAwardTypes sets the award types to return
func WithAwardTypes(awardTypes []AwardType) MockOption {
	return func(m *MockClient) {
		m.awardTypes = awardTypes
	}
}

// WithAwardTypesError sets an error to return from FetchAwardTypes
func WithAwardTypesError(err error) MockOption {
	return func(m *MockClient) {
		m.awardTypesErr = err
	}
}

// WithCreateAwardError sets an error to return from CreateAward
func WithCreateAwardError(err error) MockOption {
	return func(m *MockClient) {
		m.createAwardErr = err
	}
}

// WithBaseURL sets the base URL
func WithBaseURL(url string) MockOption {
	return func(m *MockClient) {
		m.baseURL = url
	}
}

// WithLoginError sets an error to return from Login
func WithLoginError(err error) MockOption {
	return func(m *MockClient) {
		m.loginErr = err
	}
}

// NewMockClient creates a new mock DerbyNet client
func NewMockClient(opts ...MockOption) *MockClient {
	m := &MockClient{
		baseURL:     "http://mock-derbynet.local",
		racers:      DefaultMockRacers(),
		awards:      DefaultMockAwards(),
		awardTypes:  DefaultMockAwardTypes(),
		nextAwardID: 100, // Start at 100 to avoid conflicts with existing awards
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// BaseURL returns the configured base URL
func (m *MockClient) BaseURL() string {
	return m.baseURL
}

// SetBaseURL updates the base URL
func (m *MockClient) SetBaseURL(url string) {
	m.baseURL = url
}

// SetCredentials configures authentication credentials
func (m *MockClient) SetCredentials(role, password string) {
	m.credentialsSet = true
}

// Login simulates DerbyNet authentication (always succeeds unless error is set)
func (m *MockClient) Login(ctx context.Context, role, password string) error {
	if m.loginErr != nil {
		return m.loginErr
	}
	return nil
}

// FetchRacers returns the configured mock racers or error
func (m *MockClient) FetchRacers(ctx context.Context) ([]Racer, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.racers, nil
}

// FetchAwards returns the configured mock awards or error
func (m *MockClient) FetchAwards(ctx context.Context) ([]Award, error) {
	if m.awardsErr != nil {
		return nil, m.awardsErr
	}
	return m.awards, nil
}

// FetchAwardTypes returns the configured mock award types or error
func (m *MockClient) FetchAwardTypes(ctx context.Context) ([]AwardType, error) {
	if m.awardTypesErr != nil {
		return nil, m.awardTypesErr
	}
	return m.awardTypes, nil
}

// CreateAward creates a new award in the mock client and returns its ID
func (m *MockClient) CreateAward(ctx context.Context, name string, awardTypeID int) (int, error) {
	// Simulate authentication failure if credentials were set and loginErr is set
	if m.credentialsSet && m.loginErr != nil {
		return 0, fmt.Errorf("failed to authenticate: %w", m.loginErr)
	}

	if m.createAwardErr != nil {
		return 0, m.createAwardErr
	}

	// Generate new award ID
	m.nextAwardID++
	newAwardID := m.nextAwardID

	// Find award type name
	awardTypeName := "Design" // default
	for _, at := range m.awardTypes {
		if at.AwardTypeID == awardTypeID {
			awardTypeName = at.AwardType
			break
		}
	}

	// Add new award to the list
	newAward := Award{
		AwardID:   newAwardID,
		AwardName: name,
		AwardType: awardTypeName,
		Sort:      len(m.awards) + 1,
	}
	m.awards = append(m.awards, newAward)

	return newAwardID, nil
}

// GetAwards returns the current awards (for testing)
func (m *MockClient) GetAwards() []Award {
	return m.awards
}

// SetAwardWinner records a winner for an award in the mock client
func (m *MockClient) SetAwardWinner(ctx context.Context, awardID, racerID int) error {
	// Simulate authentication failure if credentials were set and loginErr is set
	if m.credentialsSet && m.loginErr != nil {
		return fmt.Errorf("failed to authenticate: %w", m.loginErr)
	}

	if m.setWinnerErr != nil {
		return m.setWinnerErr
	}
	if m.awardWinners == nil {
		m.awardWinners = make(map[int]int)
	}
	m.awardWinners[awardID] = racerID
	return nil
}

// GetAwardWinners returns the recorded award winners (for testing)
func (m *MockClient) GetAwardWinners() map[int]int {
	return m.awardWinners
}

// DefaultMockRacers returns a set of sample racers for testing
func DefaultMockRacers() []Racer {
	return []Racer{
		{
			RacerID:   1,
			FirstName: "Alex",
			LastName:  "Johnson",
			CarNumber: 101,
			CarName:   FlexString("Lightning Bolt"),
			CarPhoto:  "cars/car_101.jpg",
		},
		{
			RacerID:   2,
			FirstName: "Sarah",
			LastName:  "Williams",
			CarNumber: 102,
			CarName:   FlexString("Red Rocket"),
			CarPhoto:  "cars/car_102.jpg",
		},
		{
			RacerID:   3,
			FirstName: "Mike",
			LastName:  "Chen",
			CarNumber: 103,
			CarName:   FlexString("Blue Thunder"),
			CarPhoto:  "cars/car_103.jpg",
		},
		{
			RacerID:   4,
			FirstName: "Emma",
			LastName:  "Davis",
			CarNumber: 104,
			CarName:   FlexString("Pink Panther"),
			CarPhoto:  "cars/car_104.jpg",
		},
		{
			RacerID:   5,
			FirstName: "James",
			LastName:  "Brown",
			CarNumber: 105,
			CarName:   FlexString("Green Machine"),
			CarPhoto:  "cars/car_105.jpg",
		},
		{
			RacerID:   6,
			FirstName: "Olivia",
			LastName:  "Martinez",
			CarNumber: 106,
			CarName:   FlexString("Purple Haze"),
			CarPhoto:  "cars/car_106.jpg",
		},
		{
			RacerID:   7,
			FirstName: "Noah",
			LastName:  "Wilson",
			CarNumber: 107,
			CarName:   FlexString("Golden Arrow"),
			CarPhoto:  "cars/car_107.jpg",
		},
		{
			RacerID:   8,
			FirstName: "Sophia",
			LastName:  "Garcia",
			CarNumber: 108,
			CarName:   FlexString("Silver Bullet"),
			CarPhoto:  "cars/car_108.jpg",
		},
		{
			RacerID:   9,
			FirstName: "Liam",
			LastName:  "Anderson",
			CarNumber: 109,
			CarName:   FlexString("Black Hawk"),
			CarPhoto:  "cars/car_109.jpg",
		},
		{
			RacerID:   10,
			FirstName: "Ava",
			LastName:  "Taylor",
			CarNumber: 110,
			CarName:   FlexString("White Lightning"),
			CarPhoto:  "cars/car_110.jpg",
		},
	}
}

// GenerateMockRacers generates n mock racers with sequential IDs
func GenerateMockRacers(n int) []Racer {
	firstNames := []string{"Alex", "Sarah", "Mike", "Emma", "James", "Olivia", "Noah", "Sophia", "Liam", "Ava"}
	lastNames := []string{"Johnson", "Williams", "Chen", "Davis", "Brown", "Martinez", "Wilson", "Garcia", "Anderson", "Taylor"}
	carNames := []string{"Lightning", "Rocket", "Thunder", "Panther", "Machine", "Haze", "Arrow", "Bullet", "Hawk", "Storm"}
	colors := []string{"Red", "Blue", "Green", "Yellow", "Purple", "Orange", "Silver", "Gold", "Black", "White"}

	racers := make([]Racer, n)
	for i := 0; i < n; i++ {
		racers[i] = Racer{
			RacerID:   i + 1,
			FirstName: firstNames[i%len(firstNames)],
			LastName:  lastNames[i%len(lastNames)],
			CarNumber: 100 + i + 1,
			CarName:   FlexString(fmt.Sprintf("%s %s", colors[i%len(colors)], carNames[i%len(carNames)])),
			CarPhoto:  fmt.Sprintf("cars/car_%d.jpg", 100+i+1),
		}
	}
	return racers
}

// DefaultMockAwards returns a set of sample awards for testing
func DefaultMockAwards() []Award {
	return []Award{
		{
			AwardID:   1,
			AwardName: "Most Creative",
			AwardType: "Design",
			Sort:      1,
		},
		{
			AwardID:   2,
			AwardName: "Best Paint Job",
			AwardType: "Design",
			Sort:      2,
		},
		{
			AwardID:   3,
			AwardName: "Best Design",
			AwardType: "Design",
			Sort:      3,
		},
		{
			AwardID:   4,
			AwardName: "Fastest Looking",
			AwardType: "Design",
			Sort:      4,
		},
		{
			AwardID:   5,
			AwardName: "Most Unique",
			AwardType: "Design",
			Sort:      5,
		},
		{
			AwardID:   6,
			AwardName: "Best Theme",
			AwardType: "Design",
			Sort:      6,
		},
	}
}

// DefaultMockAwardTypes returns a set of sample award types for testing
func DefaultMockAwardTypes() []AwardType {
	return []AwardType{
		{AwardTypeID: 1, AwardType: "Design"},
		{AwardTypeID: 2, AwardType: "Speed"},
		{AwardTypeID: 3, AwardType: "Showmanship"},
	}
}

// Ensure MockClient implements Client
var _ Client = (*MockClient)(nil)
