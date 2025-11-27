package models

// CategoryGroup represents a group of categories with optional exclusivity
type CategoryGroup struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	ExclusivityPoolID *int   `json:"exclusivity_pool_id"`
	MaxWinsPerCar     *int   `json:"max_wins_per_car,omitempty"`
	DisplayOrder      int    `json:"display_order"`
	Active            bool   `json:"active"`
}

// Category represents a voting category
type Category struct {
	ID                   int      `json:"id"`
	Name                 string   `json:"name"`
	DisplayOrder         int      `json:"display_order"`
	GroupID              *int     `json:"group_id"`
	GroupName            string   `json:"group_name,omitempty"`
	ExclusivityPoolID    *int     `json:"exclusivity_pool_id,omitempty"`
	DerbyNetAwardID      *int     `json:"derbynet_award_id,omitempty"`
	OverrideWinnerCarID  *int     `json:"override_winner_car_id,omitempty"`
	OverrideReason       string   `json:"override_reason,omitempty"`
	OverriddenAt         string   `json:"overridden_at,omitempty"`
	AllowedVoterTypes    []string `json:"allowed_voter_types,omitempty"` // Empty/nil means all types allowed
	AllowedRanks         []string `json:"allowed_ranks,omitempty"`       // Empty/nil means all ranks allowed
}

// Car represents a pinewood derby car
type Car struct {
	ID        int    `json:"id"`
	CarNumber string `json:"car_number"`
	RacerName string `json:"racer_name"`
	CarName   string `json:"car_name"`
	PhotoURL  string `json:"photo_url"`
	Rank      string `json:"rank"`
	Eligible  bool   `json:"eligible"`
}

// Vote represents a vote submission
type Vote struct {
	VoterQR    string `json:"voter_qr"`
	CategoryID int    `json:"category_id"`
	CarID      int    `json:"car_id"`
}

// VoteData represents the data sent to voters
type VoteData struct {
	Categories []Category  `json:"categories"`
	Cars       []Car       `json:"cars"`
	Votes      map[int]int `json:"votes"` // category_id -> car_id
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
