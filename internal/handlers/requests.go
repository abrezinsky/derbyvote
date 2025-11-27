package handlers

// CategoryCreateRequest represents a request to create a category
type CategoryCreateRequest struct {
	Name               string   `json:"name"`
	DisplayOrder       int      `json:"display_order"`
	GroupID            *int     `json:"group_id"`
	Active             bool     `json:"active"`
	AllowedVoterTypes  []string `json:"allowed_voter_types,omitempty"`
	AllowedRanks       []string `json:"allowed_ranks,omitempty"`
}

// CategoryUpdateRequest represents a request to update a category
type CategoryUpdateRequest struct {
	Name               string   `json:"name"`
	DisplayOrder       int      `json:"display_order"`
	GroupID            *int     `json:"group_id"`
	Active             bool     `json:"active"`
	AllowedVoterTypes  []string `json:"allowed_voter_types,omitempty"`
	AllowedRanks       []string `json:"allowed_ranks,omitempty"`
}

// CategoryGroupCreateRequest represents a request to create a category group
type CategoryGroupCreateRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	ExclusivityPoolID *int   `json:"exclusivity_pool_id"`
	MaxWinsPerCar     *int   `json:"max_wins_per_car"`
	DisplayOrder      int    `json:"display_order"`
}

// CategoryGroupUpdateRequest represents a request to update a category group
type CategoryGroupUpdateRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	ExclusivityPoolID *int   `json:"exclusivity_pool_id"`
	MaxWinsPerCar     *int   `json:"max_wins_per_car"`
	DisplayOrder      int    `json:"display_order"`
}

// VotingStatusRequest represents a request to set voting open/closed
type VotingStatusRequest struct {
	Open bool `json:"open"`
}

// VotingTimerRequest represents a request to start a voting timer
type VotingTimerRequest struct {
	Minutes int `json:"minutes"`
}

// DerbyNetSyncRequest represents a request to sync from DerbyNet
type DerbyNetSyncRequest struct {
	DerbyNetURL string `json:"derbynet_url"`
}

// QRCodeGenerateRequest represents a request to generate QR codes
type QRCodeGenerateRequest struct {
	Count int `json:"count"`
}

// SettingsUpdateRequest represents a request to update settings
type SettingsUpdateRequest struct {
	DerbyNetURL         string   `json:"derbynet_url"`
	BaseURL             string   `json:"base_url"`
	DerbyNetRole        string   `json:"derbynet_role"`
	DerbyNetPassword    string   `json:"derbynet_password"`
	RequireRegisteredQR *bool    `json:"require_registered_qr"`
	VotingInstructions  string   `json:"voting_instructions"`
	VoterTypes          []string `json:"voter_types"`
}

// DatabaseResetRequest represents a request to reset database tables
type DatabaseResetRequest struct {
	Tables []string `json:"tables"`
}

// VoterCreateRequest represents a request to create a voter
type VoterCreateRequest struct {
	CarID     *int   `json:"car_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	VoterType string `json:"voter_type"`
	QRCode    string `json:"qr_code"`
	Notes     string `json:"notes"`
}

// VoterUpdateRequest represents a request to update a voter
type VoterUpdateRequest struct {
	ID        int    `json:"id"`
	CarID     *int   `json:"car_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	VoterType string `json:"voter_type"`
	Notes     string `json:"notes"`
}

// VoteSubmitRequest represents a request to submit a vote
type VoteSubmitRequest struct {
	VoterQR    string `json:"voter_qr"`
	CategoryID int    `json:"category_id"`
	CarID      int    `json:"car_id"`
}

// SeedMockDataRequest represents a request to seed mock data
type SeedMockDataRequest struct {
	SeedType string `json:"seed_type"`
}

// CarCreateRequest represents a request to create a car
type CarCreateRequest struct {
	CarNumber string `json:"car_number"`
	RacerName string `json:"racer_name"`
	CarName   string `json:"car_name"`
	PhotoURL  string `json:"photo_url"`
	Rank      string `json:"rank"`
}

// CarUpdateRequest represents a request to update a car
type CarUpdateRequest struct {
	CarNumber string `json:"car_number"`
	RacerName string `json:"racer_name"`
	CarName   string `json:"car_name"`
	PhotoURL  string `json:"photo_url"`
	Rank      string `json:"rank"`
}

// CarEligibilityRequest represents a request to set car eligibility
type CarEligibilityRequest struct {
	Eligible bool `json:"eligible"`
	Force    bool `json:"force"`
}
