package derbynet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abrezinsky/derbyvote/internal/logger"
)

// noopLogger implements logger.Logger but discards all output
type noopLogger struct{}

func (noopLogger) Debug(msg string, args ...any) {}
func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (n noopLogger) SetLevel(level slog.Level) {}
func (n noopLogger) GetLevel() slog.Level { return slog.LevelInfo }
func (n noopLogger) EnableHTTPLogging() {}
func (n noopLogger) DisableHTTPLogging() {}
func (n noopLogger) IsHTTPLoggingEnabled() bool { return false }

var _ logger.Logger = noopLogger{}

func TestHTTPClient_FetchRacers_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/action.php" {
			t.Errorf("expected path /action.php, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "racer.list" {
			t.Errorf("expected query=racer.list, got %s", r.URL.Query().Get("query"))
		}

		response := RacerListResponse{
			Racers: []Racer{
				{RacerID: 1, FirstName: "Test", LastName: "User", CarNumber: 101, CarName: "Test Car"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	racers, err := client.FetchRacers(context.Background())
	if err != nil {
		t.Fatalf("FetchRacers failed: %v", err)
	}

	if len(racers) != 1 {
		t.Fatalf("expected 1 racer, got %d", len(racers))
	}
	if racers[0].FirstName != "Test" {
		t.Errorf("expected FirstName 'Test', got %q", racers[0].FirstName)
	}
}

func TestHTTPClient_FetchRacers_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.FetchRacers(context.Background())
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestHTTPClient_FetchRacers_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.FetchRacers(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHTTPClient_FetchRacers_ConnectionError(t *testing.T) {
	client := NewHTTPClient("http://localhost:99999", noopLogger{})
	_, err := client.FetchRacers(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestHTTPClient_BaseURL(t *testing.T) {
	client := NewHTTPClient("http://example.com", noopLogger{})
	if client.BaseURL() != "http://example.com" {
		t.Errorf("expected base URL 'http://example.com', got %q", client.BaseURL())
	}
}

func TestMockClient_FetchRacers_Default(t *testing.T) {
	client := NewMockClient()
	racers, err := client.FetchRacers(context.Background())
	if err != nil {
		t.Fatalf("FetchRacers failed: %v", err)
	}
	if len(racers) != 10 {
		t.Errorf("expected 10 default racers, got %d", len(racers))
	}
}

func TestMockClient_FetchRacers_CustomRacers(t *testing.T) {
	customRacers := []Racer{
		{RacerID: 99, FirstName: "Custom", LastName: "Racer"},
	}
	client := NewMockClient(WithRacers(customRacers))
	racers, err := client.FetchRacers(context.Background())
	if err != nil {
		t.Fatalf("FetchRacers failed: %v", err)
	}
	if len(racers) != 1 {
		t.Fatalf("expected 1 racer, got %d", len(racers))
	}
	if racers[0].RacerID != 99 {
		t.Errorf("expected RacerID 99, got %d", racers[0].RacerID)
	}
}

func TestMockClient_FetchRacers_Error(t *testing.T) {
	testErr := errors.New("mock error")
	client := NewMockClient(WithFetchError(testErr))
	_, err := client.FetchRacers(context.Background())
	if err != testErr {
		t.Errorf("expected mock error, got %v", err)
	}
}

func TestMockClient_BaseURL(t *testing.T) {
	client := NewMockClient(WithBaseURL("http://test.local"))
	if client.BaseURL() != "http://test.local" {
		t.Errorf("expected 'http://test.local', got %q", client.BaseURL())
	}
}

func TestGenerateMockRacers(t *testing.T) {
	racers := GenerateMockRacers(25)
	if len(racers) != 25 {
		t.Fatalf("expected 25 racers, got %d", len(racers))
	}

	// Check sequential IDs
	for i, r := range racers {
		if r.RacerID != i+1 {
			t.Errorf("expected RacerID %d, got %d", i+1, r.RacerID)
		}
		if r.CarNumber != 100+i+1 {
			t.Errorf("expected CarNumber %d, got %d", 100+i+1, r.CarNumber)
		}
	}
}

func TestDefaultMockRacers(t *testing.T) {
	racers := DefaultMockRacers()
	if len(racers) != 10 {
		t.Fatalf("expected 10 racers, got %d", len(racers))
	}

	// Verify first racer
	if racers[0].FirstName != "Alex" || racers[0].LastName != "Johnson" {
		t.Errorf("unexpected first racer: %+v", racers[0])
	}
}

func TestClientInterface(t *testing.T) {
	// Verify both implementations satisfy the interface
	var _ Client = (*HTTPClient)(nil)
	var _ Client = (*MockClient)(nil)
}

// ==================== FetchAwards Tests ====================

func TestHTTPClient_FetchAwards_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/action.php" {
			t.Errorf("expected path /action.php, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "award.list" {
			t.Errorf("expected query=award.list, got %s", r.URL.Query().Get("query"))
		}

		response := AwardListResponse{
			Awards: []Award{
				{AwardID: 1, AwardName: "Best Design", AwardType: "Design", Sort: 1},
				{AwardID: 2, AwardName: "Most Creative", AwardType: "Design", Sort: 2},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	awards, err := client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards failed: %v", err)
	}

	if len(awards) != 2 {
		t.Fatalf("expected 2 awards, got %d", len(awards))
	}
	if awards[0].AwardName != "Best Design" {
		t.Errorf("expected AwardName 'Best Design', got %q", awards[0].AwardName)
	}
	if awards[1].AwardID != 2 {
		t.Errorf("expected AwardID 2, got %d", awards[1].AwardID)
	}
}

func TestHTTPClient_FetchAwards_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.FetchAwards(context.Background())
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestHTTPClient_FetchAwards_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.FetchAwards(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHTTPClient_FetchAwards_ConnectionError(t *testing.T) {
	client := NewHTTPClient("http://localhost:99999", noopLogger{})
	_, err := client.FetchAwards(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestMockClient_FetchAwards_Default(t *testing.T) {
	client := NewMockClient()
	awards, err := client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards failed: %v", err)
	}
	if len(awards) != 6 {
		t.Errorf("expected 6 default awards, got %d", len(awards))
	}
}

func TestMockClient_FetchAwards_CustomAwards(t *testing.T) {
	customAwards := []Award{
		{AwardID: 99, AwardName: "Custom Award"},
	}
	client := NewMockClient(WithAwards(customAwards))
	awards, err := client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards failed: %v", err)
	}
	if len(awards) != 1 {
		t.Fatalf("expected 1 award, got %d", len(awards))
	}
	if awards[0].AwardID != 99 {
		t.Errorf("expected AwardID 99, got %d", awards[0].AwardID)
	}
}

func TestMockClient_FetchAwards_Error(t *testing.T) {
	testErr := errors.New("awards error")
	client := NewMockClient(WithAwardsError(testErr))
	_, err := client.FetchAwards(context.Background())
	if err != testErr {
		t.Errorf("expected awards error, got %v", err)
	}
}

// ==================== SetAwardWinner Tests ====================

func TestHTTPClient_SetAwardWinner_Success(t *testing.T) {
	var receivedAwardID, receivedRacerID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/action.php" {
			t.Errorf("expected path /action.php, got %s", r.URL.Path)
		}

		// Parse form data
		r.ParseForm()
		if r.Form.Get("action") != "award.winner" {
			t.Errorf("expected action=award.winner, got %s", r.Form.Get("action"))
		}
		receivedAwardID = r.Form.Get("awardid")
		receivedRacerID = r.Form.Get("racerid")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"outcome":{"summary":"success"}}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	err := client.SetAwardWinner(context.Background(), 42, 123)
	if err != nil {
		t.Fatalf("SetAwardWinner failed: %v", err)
	}

	if receivedAwardID != "42" {
		t.Errorf("expected awardid '42', got %q", receivedAwardID)
	}
	if receivedRacerID != "123" {
		t.Errorf("expected racerid '123', got %q", receivedRacerID)
	}
}

func TestHTTPClient_SetAwardWinner_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	err := client.SetAwardWinner(context.Background(), 1, 1)
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestHTTPClient_SetAwardWinner_ConnectionError(t *testing.T) {
	client := NewHTTPClient("http://localhost:99999", noopLogger{})
	err := client.SetAwardWinner(context.Background(), 1, 1)
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestHTTPClient_SetAwardWinner_FailureOutcome(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"outcome":{"summary":"failure","code":"invalid-award","description":"Award does not exist"}}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	err := client.SetAwardWinner(context.Background(), 999, 123)
	if err == nil {
		t.Fatal("expected error for failure outcome")
	}
	if !strings.Contains(err.Error(), "Award does not exist") {
		t.Errorf("expected error to contain 'Award does not exist', got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid-award") {
		t.Errorf("expected error to contain 'invalid-award', got: %v", err)
	}
}

func TestHTTPClient_SetAwardWinner_NotAuthorized(t *testing.T) {
	loginAttempts := 0
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		action := r.Form.Get("action")

		if action == "role.login" {
			loginAttempts++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"outcome":{"summary":"success"}}`))
			return
		}

		if action == "award.winner" {
			requestCount++
			if requestCount == 1 {
				// First request returns notauthorized
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"outcome":{"summary":"failure","code":"notauthorized","description":"Not authorized"}}`))
			} else {
				// Second request (after re-auth) succeeds
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"outcome":{"summary":"success"}}`))
			}
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	// Set up authentication credentials
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Reset login attempts to only count re-authentication
	loginAttempts = 0

	// This should trigger a notauthorized, then re-auth, then succeed
	err = client.SetAwardWinner(context.Background(), 42, 123)
	if err != nil {
		t.Fatalf("SetAwardWinner should succeed after re-authentication, got: %v", err)
	}

	if loginAttempts != 1 {
		t.Errorf("expected 1 re-authentication attempt, got %d", loginAttempts)
	}
	if requestCount != 2 {
		t.Errorf("expected 2 award.winner requests (initial + retry), got %d", requestCount)
	}
}

func TestMockClient_SetAwardWinner_Success(t *testing.T) {
	client := NewMockClient()

	err := client.SetAwardWinner(context.Background(), 10, 100)
	if err != nil {
		t.Fatalf("SetAwardWinner failed: %v", err)
	}

	// Verify winner was recorded
	winners := client.GetAwardWinners()
	if winners[10] != 100 {
		t.Errorf("expected award 10 to have winner 100, got %v", winners)
	}
}

func TestMockClient_SetAwardWinner_MultipleWinners(t *testing.T) {
	client := NewMockClient()

	_ = client.SetAwardWinner(context.Background(), 1, 10)
	_ = client.SetAwardWinner(context.Background(), 2, 20)
	_ = client.SetAwardWinner(context.Background(), 3, 30)

	winners := client.GetAwardWinners()
	if len(winners) != 3 {
		t.Fatalf("expected 3 winners, got %d", len(winners))
	}
	if winners[1] != 10 || winners[2] != 20 || winners[3] != 30 {
		t.Errorf("unexpected winners map: %v", winners)
	}
}

func TestMockClient_SetAwardWinner_Error(t *testing.T) {
	testErr := errors.New("set winner error")
	client := NewMockClient(WithSetWinnerError(testErr))

	err := client.SetAwardWinner(context.Background(), 1, 1)
	if err != testErr {
		t.Errorf("expected set winner error, got %v", err)
	}
}

func TestMockClient_SetBaseURL(t *testing.T) {
	client := NewMockClient()
	client.SetBaseURL("http://new-url.local")
	if client.BaseURL() != "http://new-url.local" {
		t.Errorf("expected 'http://new-url.local', got %q", client.BaseURL())
	}
}

func TestHTTPClient_SetBaseURL(t *testing.T) {
	client := NewHTTPClient("http://original.local", noopLogger{})
	client.SetBaseURL("http://new-url.local")
	if client.BaseURL() != "http://new-url.local" {
		t.Errorf("expected 'http://new-url.local', got %q", client.BaseURL())
	}
}

func TestDefaultMockAwards(t *testing.T) {
	awards := DefaultMockAwards()
	if len(awards) != 6 {
		t.Fatalf("expected 6 awards, got %d", len(awards))
	}

	// Verify first award
	if awards[0].AwardName != "Most Creative" {
		t.Errorf("unexpected first award: %+v", awards[0])
	}
}

// ==================== Login Tests ====================

func TestHTTPClient_Login_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/action.php" {
			t.Errorf("expected path /action.php, got %s", r.URL.Path)
		}

		// Parse form data
		r.ParseForm()
		if r.Form.Get("action") != "role.login" {
			t.Errorf("expected action=role.login, got %s", r.Form.Get("action"))
		}
		if r.Form.Get("name") != "RaceCoordinator" {
			t.Errorf("expected name=RaceCoordinator, got %s", r.Form.Get("name"))
		}
		if r.Form.Get("password") != "secret123" {
			t.Errorf("expected password=secret123, got %s", r.Form.Get("password"))
		}

		// Return success response
		response := `{"outcome": {"summary": "success"}}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	err := client.Login(context.Background(), "RaceCoordinator", "secret123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
}

func TestHTTPClient_Login_IncorrectPassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"action": {"action": "role.login", "name": "RaceCoordinator"},
			"outcome": {
				"summary": "failure",
				"code": "login",
				"description": "Incorrect password"
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	err := client.Login(context.Background(), "RaceCoordinator", "wrongpassword")
	if err == nil {
		t.Fatal("expected error for incorrect password")
	}
	if !strings.Contains(err.Error(), "Incorrect password") {
		t.Errorf("expected error message to contain 'Incorrect password', got %v", err)
	}
}

func TestHTTPClient_Login_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestHTTPClient_Login_ConnectionError(t *testing.T) {
	client := NewHTTPClient("http://localhost:99999", noopLogger{})
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestMockClient_Login_Success(t *testing.T) {
	client := NewMockClient()
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
}

func TestMockClient_Login_Error(t *testing.T) {
	testErr := errors.New("auth error")
	client := NewMockClient(WithLoginError(testErr))
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err != testErr {
		t.Errorf("expected auth error, got %v", err)
	}
}

// ==================== FlexString Tests ====================

func TestFlexString_UnmarshalJSON_String(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`"Hello World"`), &fs)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if fs.String() != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", fs.String())
	}
}

func TestFlexString_UnmarshalJSON_Number(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`12345`), &fs)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if fs.String() != "12345" {
		t.Errorf("expected '12345', got %q", fs.String())
	}
}

func TestFlexString_UnmarshalJSON_Float(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`123.45`), &fs)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if fs.String() != "123.45" {
		t.Errorf("expected '123.45', got %q", fs.String())
	}
}

func TestFlexString_UnmarshalJSON_Null(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`null`), &fs)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if fs.String() != "" {
		t.Errorf("expected empty string for null, got %q", fs.String())
	}
}

func TestFlexString_UnmarshalJSON_Invalid(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`{"invalid": "object"}`), &fs)
	if err == nil {
		t.Fatal("expected error for invalid JSON object")
	}
}

func TestFlexString_String(t *testing.T) {
	fs := FlexString("test value")
	if fs.String() != "test value" {
		t.Errorf("expected 'test value', got %q", fs.String())
	}
}

func TestFlexString_InRacer(t *testing.T) {
	// Test FlexString when used in a Racer struct with JSON unmarshaling
	jsonData := `{"racerid": 1, "firstname": "Test", "lastname": "User", "carnumber": 101, "carname": 42}`
	var racer Racer
	err := json.Unmarshal([]byte(jsonData), &racer)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if racer.CarName.String() != "42" {
		t.Errorf("expected carname '42', got %q", racer.CarName.String())
	}
}

// ==================== NewHTTPClientWithHTTPClient Tests ====================

func TestNewHTTPClientWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 60}
	client := NewHTTPClientWithHTTPClient("http://custom.local", customClient, noopLogger{})

	if client.BaseURL() != "http://custom.local" {
		t.Errorf("expected base URL 'http://custom.local', got %q", client.BaseURL())
	}
}

// ==================== FetchAwardTypes Tests ====================

func TestHTTPClient_FetchAwardTypes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/action.php" {
			t.Errorf("expected path /action.php, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "award.list" {
			t.Errorf("expected query=award.list, got %s", r.URL.Query().Get("query"))
		}

		response := AwardListResponse{
			Awards: []Award{},
			AwardTypes: []AwardType{
				{AwardTypeID: 1, AwardType: "Design"},
				{AwardTypeID: 2, AwardType: "Speed"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	awardTypes, err := client.FetchAwardTypes(context.Background())
	if err != nil {
		t.Fatalf("FetchAwardTypes failed: %v", err)
	}

	if len(awardTypes) != 2 {
		t.Fatalf("expected 2 award types, got %d", len(awardTypes))
	}
	if awardTypes[0].AwardType != "Design" {
		t.Errorf("expected first award type 'Design', got %q", awardTypes[0].AwardType)
	}
}

func TestHTTPClient_FetchAwardTypes_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.FetchAwardTypes(context.Background())
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestHTTPClient_FetchAwardTypes_ConnectionError(t *testing.T) {
	client := NewHTTPClient("http://localhost:99999", noopLogger{})
	_, err := client.FetchAwardTypes(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestHTTPClient_FetchAwardTypes_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.FetchAwardTypes(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMockClient_FetchAwardTypes_Default(t *testing.T) {
	client := NewMockClient()
	awardTypes, err := client.FetchAwardTypes(context.Background())
	if err != nil {
		t.Fatalf("FetchAwardTypes failed: %v", err)
	}
	if len(awardTypes) != 3 {
		t.Errorf("expected 3 default award types, got %d", len(awardTypes))
	}
}

func TestMockClient_FetchAwardTypes_Custom(t *testing.T) {
	customTypes := []AwardType{
		{AwardTypeID: 99, AwardType: "Custom Type"},
	}
	client := NewMockClient(WithAwardTypes(customTypes))
	awardTypes, err := client.FetchAwardTypes(context.Background())
	if err != nil {
		t.Fatalf("FetchAwardTypes failed: %v", err)
	}
	if len(awardTypes) != 1 {
		t.Fatalf("expected 1 award type, got %d", len(awardTypes))
	}
	if awardTypes[0].AwardTypeID != 99 {
		t.Errorf("expected AwardTypeID 99, got %d", awardTypes[0].AwardTypeID)
	}
}

func TestMockClient_FetchAwardTypes_Error(t *testing.T) {
	testErr := errors.New("award types error")
	client := NewMockClient(WithAwardTypesError(testErr))
	_, err := client.FetchAwardTypes(context.Background())
	if err != testErr {
		t.Errorf("expected award types error, got %v", err)
	}
}

// ==================== CreateAward Tests ====================

func TestHTTPClient_CreateAward_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		r.ParseForm()
		if r.Form.Get("action") != "award.edit" {
			t.Errorf("expected action=award.edit, got %s", r.Form.Get("action"))
		}
		if r.Form.Get("awardid") != "new" {
			t.Errorf("expected awardid=new, got %s", r.Form.Get("awardid"))
		}
		if r.Form.Get("name") != "Test Award" {
			t.Errorf("expected name='Test Award', got %s", r.Form.Get("name"))
		}

		response := CreateAwardResponse{
			Awards: []Award{
				{AwardID: 42, AwardName: "Test Award", AwardType: "Design"},
			},
			Outcome: Outcome{Summary: "success"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	awardID, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err != nil {
		t.Fatalf("CreateAward failed: %v", err)
	}
	if awardID != 42 {
		t.Errorf("expected award ID 42, got %d", awardID)
	}
}

func TestHTTPClient_CreateAward_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestHTTPClient_CreateAward_ConnectionError(t *testing.T) {
	client := NewHTTPClient("http://localhost:99999", noopLogger{})
	_, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestHTTPClient_CreateAward_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHTTPClient_CreateAward_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CreateAwardResponse{
			Outcome: Outcome{
				Summary:     "failure",
				Code:        "error",
				Description: "Award already exists",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err == nil {
		t.Fatal("expected error for DerbyNet failure response")
	}
	if !strings.Contains(err.Error(), "Award already exists") {
		t.Errorf("expected error message to contain 'Award already exists', got %v", err)
	}
}

func TestHTTPClient_CreateAward_AwardNotInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CreateAwardResponse{
			Awards: []Award{
				{AwardID: 1, AwardName: "Different Award"},
			},
			Outcome: Outcome{Summary: "success"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	_, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err == nil {
		t.Fatal("expected error when award not found in response")
	}
	if !strings.Contains(err.Error(), "not found in response") {
		t.Errorf("expected error about award not found, got %v", err)
	}
}

func TestHTTPClient_CreateAward_NotAuthorized(t *testing.T) {
	loginAttempts := 0
	requestCount := 0
	const testAwardName = "Test Award"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		action := r.Form.Get("action")

		if action == "role.login" {
			loginAttempts++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"outcome":{"summary":"success"}}`))
			return
		}

		if action == "award.edit" {
			requestCount++
			if requestCount == 1 {
				// First request returns notauthorized
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"outcome":{"summary":"failure","code":"notauthorized","description":"Not authorized"}}`))
			} else {
				// Second request (after re-auth) succeeds
				response := CreateAwardResponse{
					Awards: []Award{
						{AwardID: 42, AwardName: testAwardName},
					},
					Outcome: Outcome{Summary: "success"},
				}
				json.NewEncoder(w).Encode(response)
			}
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	// Set up authentication credentials
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Reset login attempts to only count re-authentication
	loginAttempts = 0

	// This should trigger a notauthorized, then re-auth, then succeed
	awardID, err := client.CreateAward(context.Background(), testAwardName, 1)
	if err != nil {
		t.Fatalf("CreateAward should succeed after re-authentication, got: %v", err)
	}

	if awardID != 42 {
		t.Errorf("expected award ID 42, got %d", awardID)
	}

	if loginAttempts != 1 {
		t.Errorf("expected 1 re-authentication attempt, got %d", loginAttempts)
	}
	if requestCount != 2 {
		t.Errorf("expected 2 award.edit requests (initial + retry), got %d", requestCount)
	}
}

func TestMockClient_CreateAward_Success(t *testing.T) {
	client := NewMockClient()

	awardID, err := client.CreateAward(context.Background(), "New Test Award", 1)
	if err != nil {
		t.Fatalf("CreateAward failed: %v", err)
	}
	if awardID == 0 {
		t.Error("expected non-zero award ID")
	}

	// Verify award was added
	awards := client.GetAwards()
	found := false
	for _, a := range awards {
		if a.AwardName == "New Test Award" {
			found = true
			if a.AwardID != awardID {
				t.Errorf("expected award ID %d, got %d", awardID, a.AwardID)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find 'New Test Award' in awards list")
	}
}

func TestMockClient_CreateAward_WithAwardType(t *testing.T) {
	customTypes := []AwardType{
		{AwardTypeID: 5, AwardType: "Custom Category"},
	}
	client := NewMockClient(WithAwardTypes(customTypes))

	awardID, err := client.CreateAward(context.Background(), "Custom Award", 5)
	if err != nil {
		t.Fatalf("CreateAward failed: %v", err)
	}

	awards := client.GetAwards()
	for _, a := range awards {
		if a.AwardID == awardID {
			if a.AwardType != "Custom Category" {
				t.Errorf("expected award type 'Custom Category', got %q", a.AwardType)
			}
			break
		}
	}
}

func TestMockClient_CreateAward_Error(t *testing.T) {
	testErr := errors.New("create award error")
	client := NewMockClient(WithCreateAwardError(testErr))

	_, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err != testErr {
		t.Errorf("expected create award error, got %v", err)
	}
}

func TestMockClient_GetAwards(t *testing.T) {
	client := NewMockClient()

	awards := client.GetAwards()
	if len(awards) != 6 {
		t.Errorf("expected 6 default awards, got %d", len(awards))
	}
}

func TestDefaultMockAwardTypes(t *testing.T) {
	awardTypes := DefaultMockAwardTypes()
	if len(awardTypes) != 3 {
		t.Fatalf("expected 3 award types, got %d", len(awardTypes))
	}

	if awardTypes[0].AwardType != "Design" {
		t.Errorf("expected first award type 'Design', got %q", awardTypes[0].AwardType)
	}
}

// ==================== Body Read Error Tests ====================

// failingReader is a reader that always returns an error
type failingReader struct{}

func (f failingReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

// failingReadCloser wraps failingReader to implement io.ReadCloser
type failingReadCloser struct {
	failingReader
}

func (f failingReadCloser) Close() error {
	return nil
}

// Custom transport that returns a response with a failing body
type failingBodyTransport struct {
	statusCode int
}

func (t failingBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.statusCode,
		Body:       failingReadCloser{},
		Header:     make(http.Header),
	}, nil
}

func TestHTTPClient_FetchAwardTypes_ReadBodyError(t *testing.T) {
	customClient := &http.Client{
		Transport: failingBodyTransport{statusCode: http.StatusOK},
	}
	client := NewHTTPClientWithHTTPClient("http://test.local", customClient, noopLogger{})

	_, err := client.FetchAwardTypes(context.Background())
	if err == nil {
		t.Fatal("expected error for body read failure")
	}
	if !strings.Contains(err.Error(), "failed to read response") {
		t.Errorf("expected 'failed to read response' error, got: %v", err)
	}
}

func TestHTTPClient_CreateAward_ReadBodyError(t *testing.T) {
	customClient := &http.Client{
		Transport: failingBodyTransport{statusCode: http.StatusOK},
	}
	client := NewHTTPClientWithHTTPClient("http://test.local", customClient, noopLogger{})

	_, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err == nil {
		t.Fatal("expected error for body read failure")
	}
	if !strings.Contains(err.Error(), "failed to read response") {
		t.Errorf("expected 'failed to read response' error, got: %v", err)
	}
}

func TestHTTPClient_FetchRacers_ReadBodyError(t *testing.T) {
	customClient := &http.Client{
		Transport: failingBodyTransport{statusCode: http.StatusOK},
	}
	client := NewHTTPClientWithHTTPClient("http://test.local", customClient, noopLogger{})

	_, err := client.FetchRacers(context.Background())
	if err == nil {
		t.Fatal("expected error for body read failure")
	}
	if !strings.Contains(err.Error(), "failed to read response") {
		t.Errorf("expected 'failed to read response' error, got: %v", err)
	}
}

func TestHTTPClient_FetchAwards_ReadBodyError(t *testing.T) {
	customClient := &http.Client{
		Transport: failingBodyTransport{statusCode: http.StatusOK},
	}
	client := NewHTTPClientWithHTTPClient("http://test.local", customClient, noopLogger{})

	_, err := client.FetchAwards(context.Background())
	if err == nil {
		t.Fatal("expected error for body read failure")
	}
	if !strings.Contains(err.Error(), "failed to read response") {
		t.Errorf("expected 'failed to read response' error, got: %v", err)
	}
}

func TestHTTPClient_Login_ReadBodyError(t *testing.T) {
	customClient := &http.Client{
		Transport: failingBodyTransport{statusCode: http.StatusOK},
	}
	client := NewHTTPClientWithHTTPClient("http://test.local", customClient, noopLogger{})

	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err == nil {
		t.Fatal("expected error for body read failure")
	}
	if !strings.Contains(err.Error(), "failed to read response") {
		t.Errorf("expected 'failed to read response' error, got: %v", err)
	}
}

func TestHTTPClient_Login_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse login response") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// ==================== FlexString Additional Tests ====================

func TestFlexString_UnmarshalJSON_Array(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`[1, 2, 3]`), &fs)
	if err == nil {
		t.Fatal("expected error for JSON array")
	}
	if !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Errorf("expected 'cannot unmarshal' error, got: %v", err)
	}
}

func TestFlexString_UnmarshalJSON_Boolean(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`true`), &fs)
	if err == nil {
		t.Fatal("expected error for JSON boolean")
	}
	if !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Errorf("expected 'cannot unmarshal' error, got: %v", err)
	}
}

func TestFlexString_UnmarshalJSON_Object(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`{"key": "value"}`), &fs)
	if err == nil {
		t.Fatal("expected error for JSON object")
	}
	if !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Errorf("expected 'cannot unmarshal' error, got: %v", err)
	}
}

func TestFlexString_UnmarshalJSON_EmptyString(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`""`), &fs)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if fs.String() != "" {
		t.Errorf("expected empty string, got %q", fs.String())
	}
}

func TestFlexString_UnmarshalJSON_Zero(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`0`), &fs)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if fs.String() != "0" {
		t.Errorf("expected '0', got %q", fs.String())
	}
}

func TestFlexString_UnmarshalJSON_NegativeNumber(t *testing.T) {
	var fs FlexString
	err := json.Unmarshal([]byte(`-42`), &fs)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if fs.String() != "-42" {
		t.Errorf("expected '-42', got %q", fs.String())
	}
}

// ==================== Invalid URL Tests (NewRequestWithContext errors) ====================

func TestHTTPClient_Login_InvalidURL(t *testing.T) {
	// URL with control character causes NewRequestWithContext to fail
	client := NewHTTPClient("http://test\x00.local", noopLogger{})
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "failed to create login request") {
		t.Errorf("expected 'failed to create login request' error, got: %v", err)
	}
}

func TestHTTPClient_FetchRacers_InvalidURL(t *testing.T) {
	client := NewHTTPClient("http://test\x00.local", noopLogger{})
	_, err := client.FetchRacers(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("expected 'failed to create request' error, got: %v", err)
	}
}

func TestHTTPClient_FetchAwards_InvalidURL(t *testing.T) {
	client := NewHTTPClient("http://test\x00.local", noopLogger{})
	_, err := client.FetchAwards(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("expected 'failed to create request' error, got: %v", err)
	}
}

func TestHTTPClient_FetchAwardTypes_InvalidURL(t *testing.T) {
	client := NewHTTPClient("http://test\x00.local", noopLogger{})
	_, err := client.FetchAwardTypes(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("expected 'failed to create request' error, got: %v", err)
	}
}

func TestHTTPClient_CreateAward_InvalidURL(t *testing.T) {
	client := NewHTTPClient("http://test\x00.local", noopLogger{})
	_, err := client.CreateAward(context.Background(), "Test", 1)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("expected 'failed to create request' error, got: %v", err)
	}
}

func TestHTTPClient_SetAwardWinner_InvalidURL(t *testing.T) {
	client := NewHTTPClient("http://test\x00.local", noopLogger{})
	err := client.SetAwardWinner(context.Background(), 1, 1)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("expected 'failed to create request' error, got: %v", err)
	}
}

// ==================== doRequest Authentication Failure Tests ====================

func TestHTTPClient_SetAwardWinner_InitialAuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return auth failure
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"outcome":{"summary":"failure","code":"authfailed","description":"Invalid password"}}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	// Set credentials without calling Login first
	client.role = "RaceCoordinator"
	client.password = "wrong"
	client.authenticated = false

	err := client.SetAwardWinner(context.Background(), 42, 123)
	if err == nil {
		t.Fatal("expected error when initial authentication fails")
	}
	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' error, got: %v", err)
	}
}

func TestHTTPClient_CreateAward_InitialAuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return auth failure
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"outcome":{"summary":"failure","code":"authfailed","description":"Invalid password"}}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	// Set credentials without calling Login first
	client.role = "RaceCoordinator"
	client.password = "wrong"
	client.authenticated = false

	_, err := client.CreateAward(context.Background(), "Test Award", 1)
	if err == nil {
		t.Fatal("expected error when initial authentication fails")
	}
	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' error, got: %v", err)
	}
}

func TestHTTPClient_SetAwardWinner_ReauthenticationFailure(t *testing.T) {
	authAttempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		action := r.Form.Get("action")

		if action == "role.login" {
			authAttempts++
			if authAttempts == 1 {
				// First login succeeds
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"outcome":{"summary":"success"}}`))
			} else {
				// Re-authentication fails
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"outcome":{"summary":"failure","code":"authfailed","description":"Invalid password"}}`))
			}
			return
		}

		if action == "award.winner" {
			// Always return notauthorized
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"outcome":{"summary":"failure","code":"notauthorized","description":"Not authorized"}}`))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, noopLogger{})
	// Initial login succeeds
	err := client.Login(context.Background(), "RaceCoordinator", "password")
	if err != nil {
		t.Fatalf("Initial login failed: %v", err)
	}

	// This should get notauthorized, try to re-auth (which fails), and return error
	err = client.SetAwardWinner(context.Background(), 42, 123)
	if err == nil {
		t.Fatal("expected error when re-authentication fails")
	}
	if !strings.Contains(err.Error(), "failed to re-authenticate") {
		t.Errorf("expected 'failed to re-authenticate' error, got: %v", err)
	}

	if authAttempts != 2 {
		t.Errorf("expected 2 login attempts, got %d", authAttempts)
	}
}

func TestHTTPClient_SetCredentials(t *testing.T) {
	log := logger.New()
	client := NewHTTPClient("http://localhost:8080", log)

	// Test setting credentials
	client.SetCredentials("RaceCoordinator", "password123")

	// Verify credentials were set (internal state, can't directly test but ensures function runs)
	// The function is simple and doesn't return anything to verify
	// This test just ensures it doesn't panic
}

func TestMockClient_SetCredentials(t *testing.T) {
	mock := NewMockClient()

	// Test setting credentials
	mock.SetCredentials("RaceCoordinator", "password123")

	// Verify credentials were set (internal state, can't directly test but ensures function runs)
	// This test just ensures it doesn't panic
}

func TestMockClient_CreateAward_WithMatchingAwardType(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	// The mock has default award types, one of which is ID 1 ("Design")
	// This should hit the break statement when finding the matching type
	awardID, err := mock.CreateAward(ctx, "Best Design", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if awardID == 0 {
		t.Error("expected non-zero award ID")
	}

	// Verify the award was created with correct type name
	awards := mock.GetAwards()
	var found bool
	for _, award := range awards {
		if award.AwardID == awardID && award.AwardName == "Best Design" {
			found = true
			if award.AwardType != "Design" {
				t.Errorf("expected award type 'Design', got %q", award.AwardType)
			}
			break
		}
	}
	if !found {
		t.Error("created award not found in awards list")
	}
}

func TestMockClient_SetAwardWinner_InitializesMap(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	// First call should initialize the awardWinners map
	err := mock.SetAwardWinner(ctx, 1, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	winners := mock.GetAwardWinners()
	if winners == nil {
		t.Fatal("award winners map should be initialized")
	}

	if winners[1] != 100 {
		t.Errorf("expected winner 100 for award 1, got %d", winners[1])
	}

	// Second call should use existing map
	err = mock.SetAwardWinner(ctx, 2, 200)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if len(winners) != 2 {
		t.Errorf("expected 2 winners, got %d", len(winners))
	}
}

func TestMockClient_CreateAward_WithoutMatchingAwardType(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	// Use an award type ID that doesn't exist in defaults (e.g., 999)
	// This should use the default "Design" type
	awardID, err := mock.CreateAward(ctx, "Unknown Type Award", 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if awardID == 0 {
		t.Error("expected non-zero award ID")
	}

	// Verify the award was created with default type name
	awards := mock.GetAwards()
	var found bool
	for _, award := range awards {
		if award.AwardID == awardID && award.AwardName == "Unknown Type Award" {
			found = true
			if award.AwardType != "Design" {
				t.Errorf("expected default award type 'Design', got %q", award.AwardType)
			}
			break
		}
	}
	if !found {
		t.Error("created award not found in awards list")
	}
}

func TestMockClient_CreateAward_AuthenticationFails(t *testing.T) {
	mock := NewMockClient(WithLoginError(fmt.Errorf("auth failed")))
	ctx := context.Background()

	// Set credentials to trigger authentication check
	mock.SetCredentials("Admin", "wrong")

	// CreateAward should fail with authentication error
	_, err := mock.CreateAward(ctx, "Test Award", 1)
	if err == nil {
		t.Fatal("expected authentication error")
	}

	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' error, got: %v", err)
	}
}

func TestMockClient_SetAwardWinner_AuthenticationFails(t *testing.T) {
	mock := NewMockClient(WithLoginError(fmt.Errorf("auth failed")))
	ctx := context.Background()

	// Set credentials to trigger authentication check
	mock.SetCredentials("Admin", "wrong")

	// SetAwardWinner should fail with authentication error
	err := mock.SetAwardWinner(ctx, 1, 100)
	if err == nil {
		t.Fatal("expected authentication error")
	}

	if !strings.Contains(err.Error(), "failed to authenticate") {
		t.Errorf("expected 'failed to authenticate' error, got: %v", err)
	}
}
