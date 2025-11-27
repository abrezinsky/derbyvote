package handlers

// CategoryResponse is the JSON response for category operations
type CategoryResponse struct {
	ID                int64    `json:"id"`
	Name              string   `json:"name"`
	DisplayOrder      int      `json:"display_order"`
	GroupID           *int     `json:"group_id"`
	Active            bool     `json:"active"`
	AllowedVoterTypes []string `json:"allowed_voter_types,omitempty"`
	AllowedRanks      []string `json:"allowed_ranks,omitempty"`
}

// CategoryGroupResponse is the response for category group operations
type CategoryGroupResponse struct {
	ID int64 `json:"id"`
}

// VotingStatusResponse is the response for voting status changes
type VotingStatusResponse struct {
	Open bool `json:"open"`
}

// VotingTimerResponse is the response for setting a voting timer
type VotingTimerResponse struct {
	CloseTime string `json:"close_time"`
	Minutes   int    `json:"minutes"`
}

// QRCodesResponse is the response for QR code generation
type QRCodesResponse struct {
	QRCodes []string `json:"qr_codes"`
}

// SettingsResponse is the response for settings
type SettingsResponse struct {
	DerbyNetURL         string   `json:"derbynet_url"`
	BaseURL             string   `json:"base_url"`
	DerbyNetRole        string   `json:"derbynet_role,omitempty"`
	RequireRegisteredQR bool     `json:"require_registered_qr"`
	VotingInstructions  string   `json:"voting_instructions,omitempty"`
	VoterTypes          []string `json:"voter_types,omitempty"`
}

// VoterResponse is the response for voter operations
type VoterResponse struct {
	ID        int64  `json:"id"`
	CarID     *int   `json:"car_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	VoterType string `json:"voter_type"`
	QRCode    string `json:"qr_code"`
	Notes     string `json:"notes"`
}

// CarResponse is the response for car operations
type CarResponse struct {
	ID        int    `json:"id"`
	CarNumber string `json:"car_number"`
	RacerName string `json:"racer_name"`
	CarName   string `json:"car_name"`
	PhotoURL  string `json:"photo_url"`
	Rank      string `json:"rank"`
}

// ConflictsResponse is the response for the conflicts detection endpoint
type ConflictsResponse struct {
	Ties      []TieConflictResponse      `json:"ties"`
	MultiWins []MultiWinConflictResponse `json:"multi_wins"`
}

// TieConflictResponse represents a category with tied vote counts
type TieConflictResponse struct {
	CategoryID   int              `json:"category_id"`
	CategoryName string           `json:"category_name"`
	TiedCars     []TiedCarResponse `json:"tied_cars"`
}

// TiedCarResponse represents a car in a tie
type TiedCarResponse struct {
	CarID     int    `json:"car_id"`
	CarNumber string `json:"car_number"`
	CarName   string `json:"car_name"`
	RacerName string `json:"racer_name"`
	VoteCount int    `json:"vote_count"`
}

// MultiWinConflictResponse represents a car winning multiple awards
type MultiWinConflictResponse struct {
	CarID         int      `json:"car_id"`
	CarNumber     string   `json:"car_number"`
	RacerName     string   `json:"racer_name"`
	AwardsWon     []string `json:"awards_won"`
	CategoryIDs   []int    `json:"category_ids"`
	GroupID       *int     `json:"group_id,omitempty"`
	GroupName     string   `json:"group_name,omitempty"`
	MaxWinsPerCar int      `json:"max_wins_per_car"`
}

// OverrideWinnerRequest is the request body for setting a manual winner
type OverrideWinnerRequest struct {
	CategoryID int    `json:"category_id"`
	CarID      int    `json:"car_id"`
	Reason     string `json:"reason"`
}

// OverrideResponse is the response for override operations
type OverrideResponse struct {
	CategoryID          int    `json:"category_id"`
	CategoryName        string `json:"category_name"`
	OverrideCarID       *int   `json:"override_car_id"`
	OverrideCarNumber   string `json:"override_car_number,omitempty"`
	OverrideRacerName   string `json:"override_racer_name,omitempty"`
	OverrideReason      string `json:"override_reason,omitempty"`
	OverriddenAt        string `json:"overridden_at,omitempty"`
}
