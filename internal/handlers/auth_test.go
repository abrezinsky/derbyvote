package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/abrezinsky/derbyvote/internal/auth"
	"github.com/abrezinsky/derbyvote/internal/handlers"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/websocket"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// ==================== handleLoginPage Tests ====================

func TestHandleLoginPage_AlreadyLoggedIn(t *testing.T) {
	setup := newTestSetupWithTemplates(t)

	// Create authenticated request
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should redirect to /admin
	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/admin" {
		t.Errorf("expected redirect to /admin, got %s", location)
	}
}

func TestHandleLoginPage_NotLoggedIn(t *testing.T) {
	setup := newTestSetupWithTemplates(t)

	// Create unauthenticated request
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should render login page
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify it's HTML content
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") && contentType != "" {
		// Content-Type might not be set explicitly
		body := rec.Body.String()
		if !strings.Contains(body, "<html>") && !strings.Contains(body, "Login") {
			t.Errorf("expected HTML login page, got: %s", body)
		}
	}
}

// ==================== handleLogin Tests ====================

func TestHandleLogin_Success(t *testing.T) {
	setup := newTestSetupWithTemplates(t)

	// Submit login form with correct password
	form := url.Values{}
	form.Set("password", "test-password")

	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should redirect to /admin
	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/admin" {
		t.Errorf("expected redirect to /admin, got %s", location)
	}

	// Should set session cookie
	cookies := rec.Result().Cookies()
	foundSessionCookie := false
	for _, cookie := range cookies {
		if cookie.Name == auth.CookieName {
			foundSessionCookie = true
			if cookie.Value == "" {
				t.Error("expected non-empty session cookie value")
			}
			break
		}
	}

	if !foundSessionCookie {
		t.Error("expected session cookie to be set")
	}
}

func TestHandleLogin_InvalidPassword(t *testing.T) {
	setup := newTestSetupWithTemplates(t)

	// Submit login form with wrong password
	form := url.Values{}
	form.Set("password", "wrong-password")

	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should render login page with error
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Should show error message
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid password") {
		t.Errorf("expected error message in response, got: %s", body)
	}
}

func TestHandleLogin_EmptyPassword(t *testing.T) {
	setup := newTestSetupWithTemplates(t)

	// Submit login form with empty password
	form := url.Values{}
	form.Set("password", "")

	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should render login page with error
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Should show error message
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid password") {
		t.Errorf("expected error message in response, got: %s", body)
	}
}

// ==================== handleLogout Tests ====================

func TestHandleLogout_WithValidSession(t *testing.T) {
	setup := newTestSetupWithTemplates(t)

	// Create logout request with valid session
	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	req.AddCookie(setup.authCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should redirect to login page
	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/admin/login" {
		t.Errorf("expected redirect to /admin/login, got %s", location)
	}

	// Should clear session cookie
	cookies := rec.Result().Cookies()
	foundClearedCookie := false
	for _, cookie := range cookies {
		if cookie.Name == auth.CookieName {
			foundClearedCookie = true
			// Cookie should be cleared (MaxAge < 0 or empty value)
			if cookie.MaxAge >= 0 && cookie.Value != "" {
				t.Error("expected session cookie to be cleared")
			}
			break
		}
	}

	if !foundClearedCookie {
		t.Error("expected session cookie to be cleared")
	}
}

func TestHandleLogout_WithoutSession(t *testing.T) {
	setup := newTestSetupWithTemplates(t)

	// Create logout request without session cookie
	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should still redirect to login page
	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/admin/login" {
		t.Errorf("expected redirect to /admin/login, got %s", location)
	}
}

func TestHandleLogout_WithInvalidSession(t *testing.T) {
	setup := newTestSetupWithTemplates(t)

	// Create logout request with invalid session cookie
	invalidCookie := &http.Cookie{
		Name:  auth.CookieName,
		Value: "invalid-token-xyz",
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	req.AddCookie(invalidCookie)
	rec := httptest.NewRecorder()

	setup.router.ServeHTTP(rec, req)

	// Should still redirect to login page
	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/admin/login" {
		t.Errorf("expected redirect to /admin/login, got %s", location)
	}
}

// ==================== Helper for Tests with Templates ====================

type testSetupWithTemplates struct {
	repo       *repository.Repository
	handlers   *handlers.Handlers
	router     http.Handler
	authCookie *http.Cookie
}

func newTestSetupWithTemplates(t *testing.T) *testSetupWithTemplates {
	t.Helper()

	repo, err := repository.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Initialize logger for tests
	log := logger.New()

	// Initialize services
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, repo, derbynetClient)
	carService := services.NewCarService(log, repo, derbynetClient)
	settingsService := services.NewSettingsService(log, repo)
	voterService := services.NewVoterService(log, repo, settingsService)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, repo, settingsService, derbynetClient)

	// Create template filesystem for auth pages
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login{{if .Error}} - {{.Error}}{{end}}</body></html>`),
		},
		"admin/layout.html": &fstest.MapFile{
			Data: []byte(`<html><body>{{template "content" .}}</body></html>{{define "content"}}{{end}}`),
		},
		"admin/dashboard.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Dashboard{{end}}`),
		},
		"admin/categories.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Categories{{end}}`),
		},
		"admin/cars.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Cars{{end}}`),
		},
		"admin/results.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Results{{end}}`),
		},
		"admin/voters.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Voters{{end}}`),
		},
		"admin/settings.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Settings{{end}}`),
		},
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

	// Create auth
	adminAuth := auth.New("test-password")
	hub := websocket.New(log, settingsService)

	// Initialize handlers with templates
	h, err := handlers.New(
		votingService,
		categoryService,
		voterService,
		carService,
		settingsService,
		resultsService,
		templatesFS,
		staticServer,
		adminAuth,
		hub, handlers.NoopHTTPLogger{},
	)
	if err != nil {
		t.Fatalf("failed to create handlers: %v", err)
	}

	// Login to get a session cookie for authenticated requests
	token, _ := h.Auth.Login("test-password")
	authCookie := &http.Cookie{
		Name:  auth.CookieName,
		Value: token,
	}

	return &testSetupWithTemplates{
		repo:       repo,
		handlers:   h,
		router:     h.Router(),
		authCookie: authCookie,
	}
}
