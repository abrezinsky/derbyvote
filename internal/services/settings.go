package services

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
)

// Broadcaster defines the interface for broadcasting messages to clients
type Broadcaster interface {
	BroadcastVotingStatus(open bool, closeTime string)
}

// SettingsService handles settings-related business logic
type SettingsService struct {
	log         logger.Logger
	repo        repository.SettingsRepository
	broadcaster Broadcaster
}

// NewSettingsService creates a new SettingsService
func NewSettingsService(log logger.Logger, repo repository.SettingsRepository) *SettingsService {
	return &SettingsService{log: log, repo: repo}
}

// SetBroadcaster sets the broadcaster for sending updates to clients
func (s *SettingsService) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// IsVotingOpen checks if voting is currently open
func (s *SettingsService) IsVotingOpen(ctx context.Context) (bool, error) {
	value, err := s.repo.GetSetting(ctx, "voting_open")
	if err != nil {
		if err == repository.ErrNotFound {
			return true, nil // Default to open if setting doesn't exist
		}
		return false, err // Propagate database errors
	}
	return value == "true", nil
}

// SetVotingOpen sets the voting open status
func (s *SettingsService) SetVotingOpen(ctx context.Context, open bool) error {
	value := "false"
	if open {
		value = "true"
	}
	return s.repo.SetSetting(ctx, "voting_open", value)
}

// GetDerbyNetURL returns the configured DerbyNet URL
func (s *SettingsService) GetDerbyNetURL(ctx context.Context) (string, error) {
	return s.repo.GetSetting(ctx, "derbynet_url")
}

// SetDerbyNetURL saves the DerbyNet URL
func (s *SettingsService) SetDerbyNetURL(ctx context.Context, url string) error {
	return s.repo.SetSetting(ctx, "derbynet_url", url)
}

// GetBaseURL returns the application base URL
func (s *SettingsService) GetBaseURL(ctx context.Context) (string, error) {
	value, err := s.repo.GetSetting(ctx, "base_url")
	if err != nil {
		if err == repository.ErrNotFound {
			return "", nil // No default - setting not yet configured
		}
		return "", err // Propagate database errors
	}
	return value, nil
}

// SetBaseURL saves the application base URL
func (s *SettingsService) SetBaseURL(ctx context.Context, url string) error {
	return s.repo.SetSetting(ctx, "base_url", url)
}

// GetSetting retrieves an arbitrary setting
func (s *SettingsService) GetSetting(ctx context.Context, key string) (string, error) {
	return s.repo.GetSetting(ctx, key)
}

// SetSetting saves an arbitrary setting
func (s *SettingsService) SetSetting(ctx context.Context, key, value string) error {
	return s.repo.SetSetting(ctx, key, value)
}

// GetTimerEndTime returns the timer end timestamp (Unix seconds)
func (s *SettingsService) GetTimerEndTime(ctx context.Context) (int64, error) {
	value, err := s.repo.GetSetting(ctx, "timer_end")
	if err != nil {
		if err == repository.ErrNotFound {
			return 0, nil // No timer set
		}
		return 0, err // Propagate database errors
	}
	endTime, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, nil // Invalid value, treat as no timer
	}
	return endTime, nil
}

// SetTimerEndTime sets the timer end timestamp
func (s *SettingsService) SetTimerEndTime(ctx context.Context, endTime int64) error {
	return s.repo.SetSetting(ctx, "timer_end", strconv.FormatInt(endTime, 10))
}

// ClearTimer clears the timer
func (s *SettingsService) ClearTimer(ctx context.Context) error {
	return s.repo.SetSetting(ctx, "timer_end", "0")
}

// RequireRegisteredQR checks if voting requires pre-registered QR codes
func (s *SettingsService) RequireRegisteredQR(ctx context.Context) (bool, error) {
	value, err := s.repo.GetSetting(ctx, "require_registered_qr")
	if err != nil {
		if err == repository.ErrNotFound {
			return false, nil // Default to false (allow any QR code)
		}
		return false, err // Propagate database errors
	}
	return value == "true", nil
}

// SetRequireRegisteredQR sets whether voting requires pre-registered QR codes
func (s *SettingsService) SetRequireRegisteredQR(ctx context.Context, require bool) error {
	value := "false"
	if require {
		value = "true"
	}
	return s.repo.SetSetting(ctx, "require_registered_qr", value)
}

// AllSettings returns commonly used settings as a map
func (s *SettingsService) AllSettings(ctx context.Context) (map[string]interface{}, error) {
	settings := make(map[string]interface{})

	votingOpen, _ := s.IsVotingOpen(ctx)
	settings["voting_open"] = votingOpen

	derbyNetURL, _ := s.GetDerbyNetURL(ctx)
	settings["derbynet_url"] = derbyNetURL

	baseURL, _ := s.GetBaseURL(ctx)
	settings["base_url"] = baseURL

	timerEnd, _ := s.GetTimerEndTime(ctx)
	settings["timer_end"] = timerEnd

	requireRegisteredQR, _ := s.RequireRegisteredQR(ctx)
	settings["require_registered_qr"] = requireRegisteredQR

	return settings, nil
}

// OpenVoting opens voting and broadcasts the status change
func (s *SettingsService) OpenVoting(ctx context.Context) error {
	if err := s.SetVotingOpen(ctx, true); err != nil {
		return err
	}
	s.broadcast(true, "")
	return nil
}

// CloseVoting closes voting, clears the timer, and broadcasts the status change
func (s *SettingsService) CloseVoting(ctx context.Context) error {
	if err := s.SetVotingOpen(ctx, false); err != nil {
		return err
	}
	s.ClearTimer(ctx)
	s.SetSetting(ctx, "voting_close_time", "")
	s.broadcast(false, "")
	return nil
}

// StartVotingTimer starts a voting timer for the specified minutes, opens voting, and broadcasts
func (s *SettingsService) StartVotingTimer(ctx context.Context, minutes int) (string, error) {
	if minutes <= 0 || minutes > 60 {
		return "", ErrInvalidTimerMinutes
	}

	closeTime := time.Now().Add(time.Duration(minutes) * time.Minute)
	closeTimeStr := closeTime.Format(time.RFC3339)

	if err := s.SetSetting(ctx, "voting_close_time", closeTimeStr); err != nil {
		return "", err
	}

	if err := s.SetVotingOpen(ctx, true); err != nil {
		return "", err
	}

	s.broadcast(true, closeTimeStr)
	return closeTimeStr, nil
}

// Settings represents application settings for update operations
type Settings struct {
	DerbyNetURL         string
	BaseURL             string
	DerbyNetRole        string
	DerbyNetPassword    string
	RequireRegisteredQR *bool
	VotingInstructions  string
	VoterTypes          []string
}

// UpdateSettings updates multiple settings at once
func (s *SettingsService) UpdateSettings(ctx context.Context, settings Settings) error {
	if settings.DerbyNetURL != "" {
		if err := s.SetDerbyNetURL(ctx, settings.DerbyNetURL); err != nil {
			return err
		}
	}
	if settings.BaseURL != "" {
		if err := s.SetBaseURL(ctx, settings.BaseURL); err != nil {
			return err
		}
	}
	if settings.DerbyNetRole != "" {
		if err := s.SetSetting(ctx, "derbynet_role", settings.DerbyNetRole); err != nil {
			return err
		}
	}
	if settings.DerbyNetPassword != "" {
		if err := s.SetSetting(ctx, "derbynet_password", settings.DerbyNetPassword); err != nil {
			return err
		}
	}
	if settings.RequireRegisteredQR != nil {
		if err := s.SetRequireRegisteredQR(ctx, *settings.RequireRegisteredQR); err != nil {
			return err
		}
	}
	if settings.VotingInstructions != "" {
		if err := s.SetSetting(ctx, "voting_instructions", settings.VotingInstructions); err != nil {
			return err
		}
	}
	if len(settings.VoterTypes) > 0 {
		if err := s.SetVoterTypes(ctx, settings.VoterTypes); err != nil {
			return err
		}
	}
	return nil
}

// ResetTablesResult contains the result of a database reset
type ResetTablesResult struct {
	Tables  []string
	Message string
}

// ValidTables defines which tables can be reset
var ValidTables = map[string]bool{
	"votes": true, "voters": true, "cars": true, "categories": true, "settings": true,
}

// ResetTables validates and resets the specified database tables
func (s *SettingsService) ResetTables(ctx context.Context, tables []string) (*ResetTablesResult, error) {
	if len(tables) == 0 {
		return nil, ErrNoTablesSpecified
	}

	// Validate tables
	var tablesToReset []string
	for _, table := range tables {
		if !ValidTables[table] {
			return nil, &InvalidTableError{Table: table}
		}
		tablesToReset = append(tablesToReset, table)
	}

	// Auto-add votes table if dependent tables are being reset
	needsVotesCleared := false
	for _, table := range tablesToReset {
		if table == "cars" || table == "voters" || table == "categories" {
			needsVotesCleared = true
			break
		}
	}

	if needsVotesCleared && !containsTable(tablesToReset, "votes") {
		tablesToReset = append([]string{"votes"}, tablesToReset...)
	}

	// Close voting if votes or settings are being reset
	if containsTable(tablesToReset, "votes") || containsTable(tablesToReset, "settings") {
		s.SetVotingOpen(ctx, false)
		s.ClearTimer(ctx)
	}

	// Delete data from each table
	for _, table := range tablesToReset {
		if err := s.repo.ClearTable(ctx, table); err != nil {
			return nil, err
		}
	}

	return &ResetTablesResult{
		Tables:  tablesToReset,
		Message: "Successfully deleted data from tables",
	}, nil
}

func containsTable(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// broadcast sends voting status to all connected clients
func (s *SettingsService) broadcast(open bool, closeTime string) {
	if s.broadcaster != nil {
		s.broadcaster.BroadcastVotingStatus(open, closeTime)
	}
}

// GetVoterTypes returns the list of voter types
// Returns default types if not configured
func (s *SettingsService) GetVoterTypes(ctx context.Context) ([]string, error) {
	voterTypesJSON, err := s.GetSetting(ctx, "voter_types")
	if err != nil || voterTypesJSON == "" {
		// Return defaults: general and racer are always included, plus 2 defaults
		return []string{"general", "racer", "Race Committee", "Cubmaster"}, nil
	}

	var voterTypes []string
	if err := json.Unmarshal([]byte(voterTypesJSON), &voterTypes); err != nil {
		return nil, err
	}

	// Ensure general and racer are always present
	return ensureRequiredVoterTypes(voterTypes), nil
}

// SetVoterTypes sets the list of voter types
// Always ensures "general" and "racer" are included
func (s *SettingsService) SetVoterTypes(ctx context.Context, types []string) error {
	// Ensure general and racer are always present
	types = ensureRequiredVoterTypes(types)

	jsonData, _ := json.Marshal(types) // Marshal on []string never fails

	return s.SetSetting(ctx, "voter_types", string(jsonData))
}

// ensureRequiredVoterTypes ensures "general" and "racer" are always in the list
func ensureRequiredVoterTypes(types []string) []string {
	hasGeneral := false
	hasRacer := false

	for _, t := range types {
		if t == "general" {
			hasGeneral = true
		}
		if t == "racer" {
			hasRacer = true
		}
	}

	result := make([]string, 0, len(types)+2)

	// Always add general and racer first if not in input
	if !hasGeneral {
		result = append(result, "general")
	}
	if !hasRacer {
		result = append(result, "racer")
	}

	// Add all types from input
	for _, t := range types {
		result = append(result, t)
	}

	return result
}
