package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
)

// VoterService handles voter-related business logic
type VoterService struct {
	log        logger.Logger
	repo       repository.VoterRepository
	settings   SettingsServicer
	randReader io.Reader // for testing: defaults to crypto/rand.Reader
}

// NewVoterService creates a new VoterService
func NewVoterService(log logger.Logger, repo repository.VoterRepository, settings SettingsServicer) *VoterService {
	return &VoterService{
		log:        log,
		repo:       repo,
		settings:   settings,
		randReader: rand.Reader,
	}
}

// SetRandReader sets a custom random reader (for testing)
func (s *VoterService) SetRandReader(reader io.Reader) {
	s.randReader = reader
}

// Voter represents a voter for create/update operations
type Voter struct {
	ID        int
	CarID     *int
	Name      string
	Email     string
	VoterType string
	QRCode    string
	Notes     string
}

// ListVoters returns all voters with car info
func (s *VoterService) ListVoters(ctx context.Context) ([]map[string]interface{}, error) {
	return s.repo.ListVoters(ctx)
}

// CreateVoter creates a new voter
func (s *VoterService) CreateVoter(ctx context.Context, voter Voter) (int64, string, error) {
	// Generate QR code if not provided
	if voter.QRCode == "" {
		timestamp := time.Now().UnixNano()
		seed := fmt.Sprintf("voter-%d-%s", timestamp, voter.Name)
		voter.QRCode = GenerateReadableCode(seed)
	}

	// Set default voter type
	if voter.VoterType == "" {
		voter.VoterType = "general"
	}

	id, err := s.repo.CreateVoterFull(ctx, voter.CarID, voter.Name, voter.Email, voter.VoterType, voter.QRCode, voter.Notes)
	return id, voter.QRCode, err
}

// UpdateVoter updates a voter
func (s *VoterService) UpdateVoter(ctx context.Context, voter Voter) error {
	return s.repo.UpdateVoter(ctx, voter.ID, voter.CarID, voter.Name, voter.Email, voter.VoterType, voter.Notes)
}

// DeleteVoter deletes a voter
func (s *VoterService) DeleteVoter(ctx context.Context, id int) error {
	return s.repo.DeleteVoter(ctx, id)
}

// GenerateQRCodes generates multiple QR codes and creates voters
func (s *VoterService) GenerateQRCodes(ctx context.Context, count int) ([]string, error) {
	if count <= 0 || count > 200 {
		return nil, ErrInvalidQRCount
	}

	qrCodes := make([]string, count)
	timestamp := time.Now().UnixNano()

	for i := 0; i < count; i++ {
		seed := fmt.Sprintf("bulk-%d-%d", timestamp, i)
		qrCode := GenerateReadableCode(seed)
		qrCodes[i] = qrCode

		if err := s.repo.InsertVoterIgnore(ctx, qrCode); err != nil {
			s.log.Error("Error creating voter", "qr_code", qrCode, "error", err)
		}
	}

	return qrCodes, nil
}

// GenerateReadableCode creates a short, readable code from input data
// Uses only clear characters (no O/0/I/1/L) - format: XX-YYY
func GenerateReadableCode(seed string) string {
	const chars = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

	hash := sha256.Sum256([]byte(seed))
	num := binary.BigEndian.Uint64(hash[:8])

	code := make([]byte, 5)
	for i := 0; i < 5; i++ {
		code[i] = chars[num%uint64(len(chars))]
		num /= uint64(len(chars))
	}

	return fmt.Sprintf("%s-%s", string(code[:2]), string(code[2:]))
}

// GenerateQRImage generates a QR code PNG image for a voter by ID
func (s *VoterService) GenerateQRImage(ctx context.Context, voterID int) ([]byte, error) {
	qrCode, err := s.repo.GetVoterQRCode(ctx, voterID)
	if err != nil {
		return nil, fmt.Errorf("voter not found: %w", err)
	}

	baseURL, err := s.settings.GetBaseURL(ctx)
	if err != nil || baseURL == "" {
		return nil, fmt.Errorf("base_url not configured")
	}
	votingURL := fmt.Sprintf("%s/vote/%s", strings.TrimSuffix(baseURL, "/"), qrCode)
	return qrcode.Encode(votingURL, qrcode.Medium, 256)
}

// GenerateUniqueCode generates a unique random code that doesn't exist in the database
// This should only be called when require_registered_qr is disabled (open voting mode)
func (s *VoterService) GenerateUniqueCode(ctx context.Context) (string, error) {
	// Check if open voting is allowed
	requireRegistered, err := s.settings.RequireRegisteredQR(ctx)
	if err != nil {
		return "", fmt.Errorf("error checking settings: %w", err)
	}
	if requireRegistered {
		return "", ErrOpenVotingDisabled
	}

	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		// Generate a random 8-character hex code
		bytes := make([]byte, 4)
		if _, err := s.randReader.Read(bytes); err != nil {
			return "", fmt.Errorf("failed to generate random code: %w", err)
		}
		code := hex.EncodeToString(bytes)

		// Check if this code already exists
		_, err := s.repo.GetVoterByQR(ctx, code)
		if err == repository.ErrNotFound {
			// Code doesn't exist - perfect!
			return code, nil
		}
		if err != nil {
			// Database error
			return "", fmt.Errorf("error checking code uniqueness: %w", err)
		}

		// Code exists, try again
		s.log.Debug("Generated code already exists, retrying", "code", code, "attempt", i+1)
	}

	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxRetries)
}

// GenerateDynamicQRImage generates a QR code for /vote/new URL
// This allows anyone to scan and get their own unique code (open voting mode)
func (s *VoterService) GenerateDynamicQRImage(ctx context.Context) ([]byte, error) {
	// Check if open voting is allowed
	requireRegistered, err := s.settings.RequireRegisteredQR(ctx)
	if err != nil {
		return nil, fmt.Errorf("error checking settings: %w", err)
	}
	if requireRegistered {
		return nil, ErrOpenVotingDisabled
	}

	baseURL, err := s.settings.GetBaseURL(ctx)
	if err != nil || baseURL == "" {
		return nil, fmt.Errorf("base_url not configured")
	}

	voteURL := fmt.Sprintf("%s/vote/new", strings.TrimSuffix(baseURL, "/"))
	return qrcode.Encode(voteURL, qrcode.Medium, 256)
}
