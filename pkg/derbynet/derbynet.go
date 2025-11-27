// Package derbynet provides a client for interacting with DerbyNet racing software.
package derbynet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/abrezinsky/derbyvote/internal/logger"
)

// FlexString is a string type that can be unmarshaled from either a string or a number.
// This handles DerbyNet API inconsistency where some fields may be returned as numbers.
type FlexString string

// UnmarshalJSON implements json.Unmarshaler for FlexString
func (f *FlexString) UnmarshalJSON(data []byte) error {
	// Handle null first (before other unmarshal attempts)
	if string(data) == "null" {
		*f = ""
		return nil
	}

	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexString(s)
		return nil
	}

	// Try number (int or float)
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexString(n.String())
		return nil
	}

	return fmt.Errorf("FlexString: cannot unmarshal %s", string(data))
}

// String returns the string value
func (f FlexString) String() string {
	return string(f)
}

// Racer represents a racer from DerbyNet
type Racer struct {
	RacerID   int        `json:"racerid"`
	FirstName string     `json:"firstname"`
	LastName  string     `json:"lastname"`
	CarNumber int        `json:"carnumber"`
	CarName   FlexString `json:"carname"`
	CarPhoto  string     `json:"car_photo"`
	Rank      string     `json:"rank"` // Den/rank (e.g., "Tiger", "Lion", "Bear")
}

// RacerListResponse is the response from the racer.list API
type RacerListResponse struct {
	Racers []Racer `json:"racers"`
}

// Award represents an award/category from DerbyNet
type Award struct {
	AwardID   int    `json:"awardid"`
	AwardName string `json:"awardname"`
	AwardType string `json:"awardtype"`
	ClassID   int    `json:"classid"`
	ClassName string `json:"class"`
	RankID    int    `json:"rankid"`
	RankName  string `json:"rank"`
	Sort      int    `json:"sort"`
}

// AwardListResponse is the response from the award.list API
type AwardListResponse struct {
	Awards     []Award     `json:"awards"`
	AwardTypes []AwardType `json:"award-types"`
}

// AwardType represents an award type from DerbyNet
type AwardType struct {
	AwardTypeID int    `json:"awardtypeid"`
	AwardType   string `json:"awardtype"`
}

// Outcome represents the outcome/status from a DerbyNet API call
type Outcome struct {
	Summary     string `json:"summary"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// CreateAwardResponse is the response from creating an award
type CreateAwardResponse struct {
	Awards  []Award `json:"awards"`
	Outcome Outcome `json:"outcome"`
}

// GenericResponse is a generic DerbyNet API response with outcome
type GenericResponse struct {
	Outcome Outcome `json:"outcome"`
}

// Client defines the interface for DerbyNet operations
type Client interface {
	// Login authenticates with DerbyNet using role-based login
	Login(ctx context.Context, role, password string) error
	// SetCredentials configures authentication credentials for automatic login
	SetCredentials(role, password string)
	// FetchRacers retrieves all racers from DerbyNet
	FetchRacers(ctx context.Context) ([]Racer, error)
	// FetchAwards retrieves all awards/categories from DerbyNet
	FetchAwards(ctx context.Context) ([]Award, error)
	// FetchAwardTypes retrieves all award types from DerbyNet
	FetchAwardTypes(ctx context.Context) ([]AwardType, error)
	// CreateAward creates a new award in DerbyNet and returns the new award ID
	CreateAward(ctx context.Context, name string, awardTypeID int) (int, error)
	// SetAwardWinner assigns a winner (racer) to an award in DerbyNet
	SetAwardWinner(ctx context.Context, awardID, racerID int) error
	// BaseURL returns the configured DerbyNet base URL
	BaseURL() string
	// SetBaseURL updates the DerbyNet base URL
	SetBaseURL(url string)
}

// HTTPClient is a real HTTP client for DerbyNet
type HTTPClient struct {
	baseURL       string
	httpClient    *http.Client
	log           logger.Logger
	role          string
	password      string
	authenticated bool
}

// NewHTTPClient creates a new DerbyNet HTTP client with cookie support
func NewHTTPClient(baseURL string, log logger.Logger) *HTTPClient {
	jar, _ := cookiejar.New(nil)
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
		log: log,
	}
}

// NewHTTPClientWithHTTPClient creates a new DerbyNet client with a custom http.Client
func NewHTTPClientWithHTTPClient(baseURL string, httpClient *http.Client, log logger.Logger) *HTTPClient {
	return &HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		log:        log,
	}
}

// BaseURL returns the configured DerbyNet base URL
func (c *HTTPClient) BaseURL() string {
	return c.baseURL
}

// SetBaseURL updates the DerbyNet base URL
func (c *HTTPClient) SetBaseURL(url string) {
	c.baseURL = url
}

// SetCredentials configures authentication credentials for automatic login
func (c *HTTPClient) SetCredentials(role, password string) {
	c.role = role
	c.password = password
	c.authenticated = false // Reset auth state when credentials change
}

// doRequest executes an HTTP POST request to DerbyNet and handles common error checking
// It validates the HTTP status, parses the JSON response, and checks the outcome field for failures
// Automatically re-authenticates if the session has expired
func (c *HTTPClient) doRequest(ctx context.Context, action string, params url.Values, response interface{}) error {
	// Ensure we're authenticated before making the request
	if !c.authenticated && c.role != "" && c.password != "" {
		c.log.Debug("Not authenticated, logging in before request")
		if err := c.Login(ctx, c.role, c.password); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	apiURL := fmt.Sprintf("%s/action.php", c.baseURL)
	params.Set("action", action)

	c.log.Debug("DerbyNet request", "method", "POST", "url", apiURL, "action", action, "body", params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to DerbyNet: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	c.log.Debug("DerbyNet response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DerbyNet returned status %d: %s", resp.StatusCode, string(body))
	}

	// First check if there's a failure outcome
	var outcomeCheck struct {
		Outcome Outcome `json:"outcome"`
	}
	if err := json.Unmarshal(body, &outcomeCheck); err == nil {
		// If we get "notauthorized", try to re-authenticate and retry once
		if outcomeCheck.Outcome.Code == "notauthorized" && c.role != "" && c.password != "" {
			c.log.Debug("Session expired, re-authenticating")
			c.authenticated = false
			if err := c.Login(ctx, c.role, c.password); err != nil {
				return fmt.Errorf("failed to re-authenticate: %w", err)
			}
			// Retry the original request
			return c.doRequest(ctx, action, params, response)
		}

		if outcomeCheck.Outcome.Summary == "failure" {
			return fmt.Errorf("DerbyNet error: %s (%s)", outcomeCheck.Outcome.Description, outcomeCheck.Outcome.Code)
		}
	}

	// Parse the full response into the provided struct
	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// LoginResponse represents the response from a DerbyNet login
type LoginResponse struct {
	Outcome Outcome `json:"outcome"`
}

// Login authenticates with DerbyNet using role-based login and stores credentials
func (c *HTTPClient) Login(ctx context.Context, role, password string) error {
	// Perform the actual login
	params := url.Values{}
	params.Set("name", role)
	params.Set("password", password)

	apiURL := fmt.Sprintf("%s/action.php", c.baseURL)
	params.Set("action", "role.login")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to DerbyNet: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	c.log.Debug("DerbyNet login response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DerbyNet returned status %d: %s", resp.StatusCode, string(body))
	}

	var response LoginResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if response.Outcome.Summary == "failure" {
		return fmt.Errorf("DerbyNet login failed: %s (%s)", response.Outcome.Description, response.Outcome.Code)
	}

	// Save credentials for re-authentication
	c.role = role
	c.password = password
	c.authenticated = true

	c.log.Info("DerbyNet login successful", "role", role)
	return nil
}

// FetchRacers retrieves all racers from DerbyNet
func (c *HTTPClient) FetchRacers(ctx context.Context) ([]Racer, error) {
	reqURL := fmt.Sprintf("%s/action.php?query=racer.list&render=200x200", c.baseURL)

	c.log.Debug("DerbyNet request", "method", "GET", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DerbyNet: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	c.log.Debug("DerbyNet response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DerbyNet returned status %d", resp.StatusCode)
	}

	var response RacerListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Racers, nil
}

// FetchAwards retrieves all awards/categories from DerbyNet
func (c *HTTPClient) FetchAwards(ctx context.Context) ([]Award, error) {
	reqURL := fmt.Sprintf("%s/action.php?query=award.list", c.baseURL)

	c.log.Debug("DerbyNet request", "method", "GET", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DerbyNet: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	c.log.Debug("DerbyNet response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DerbyNet returned status %d", resp.StatusCode)
	}

	var response AwardListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Awards, nil
}

// FetchAwardTypes retrieves all award types from DerbyNet
func (c *HTTPClient) FetchAwardTypes(ctx context.Context) ([]AwardType, error) {
	reqURL := fmt.Sprintf("%s/action.php?query=award.list", c.baseURL)

	c.log.Debug("DerbyNet request", "method", "GET", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DerbyNet: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	c.log.Debug("DerbyNet response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DerbyNet returned status %d", resp.StatusCode)
	}

	var response AwardListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.AwardTypes, nil
}

// CreateAward creates a new award in DerbyNet and returns the new award ID
func (c *HTTPClient) CreateAward(ctx context.Context, name string, awardTypeID int) (int, error) {
	params := url.Values{}
	params.Set("awardid", "new")
	params.Set("name", name)
	params.Set("awardtypeid", fmt.Sprintf("%d", awardTypeID))

	var response CreateAwardResponse
	if err := c.doRequest(ctx, "award.edit", params, &response); err != nil {
		return 0, err
	}

	c.log.Debug("CreateAward parsed response", "awards_count", len(response.Awards))

	// Find the award with matching name to get its ID
	for _, award := range response.Awards {
		if award.AwardName == name {
			c.log.Debug("Found matching award", "name", name, "award_id", award.AwardID)
			return award.AwardID, nil
		}
	}

	// Debug: list what awards we got back
	var awardNames []string
	for _, award := range response.Awards {
		awardNames = append(awardNames, award.AwardName)
	}

	return 0, fmt.Errorf("award created but ID not found in response (looking for %q, got %d awards: %v)", name, len(response.Awards), awardNames)
}

// SetAwardWinner assigns a winner (racer) to an award in DerbyNet
func (c *HTTPClient) SetAwardWinner(ctx context.Context, awardID, racerID int) error {
	params := url.Values{}
	params.Set("awardid", fmt.Sprintf("%d", awardID))
	params.Set("racerid", fmt.Sprintf("%d", racerID))

	var response GenericResponse
	return c.doRequest(ctx, "award.winner", params, &response)
}

// Ensure HTTPClient implements Client
var _ Client = (*HTTPClient)(nil)
