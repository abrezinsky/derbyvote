package handlers_test

import (
	"net/http"
	"net/http/httptest"
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

// Template rendering tests for page handlers

func setupHandlersWithTemplates(t *testing.T) (*handlers.Handlers, *http.Cookie) {
	t.Helper()

	// Create test templates
	templatesFS := fstest.MapFS{
		"index.html":             &fstest.MapFile{Data: []byte(`<html><body><h1>Index Page</h1></body></html>`)},
		"voter/vote.html":        &fstest.MapFile{Data: []byte(`<html><body><h1>Vote Page</h1></body></html>`)},
		"admin/login.html":       &fstest.MapFile{Data: []byte(`<html><body><h1>Login Page</h1></body></html>`)},
		"admin/layout.html":      &fstest.MapFile{Data: []byte(`{{define "admin"}}<html><body><h1>{{.PageTitle}}</h1>{{template "content" .}}</body></html>{{end}}`),
		},
		"admin/dashboard.html":  &fstest.MapFile{Data: []byte(`{{define "content"}}<div>Dashboard Content</div>{{end}}`)},
		"admin/categories.html": &fstest.MapFile{Data: []byte(`{{define "content"}}<div>Categories Content</div>{{end}}`)},
		"admin/cars.html":       &fstest.MapFile{Data: []byte(`{{define "content"}}<div>Cars Content</div>{{end}}`)},
		"admin/results.html":    &fstest.MapFile{Data: []byte(`{{define "content"}}<div>Results Content</div>{{end}}`)},
		"admin/voters.html":     &fstest.MapFile{Data: []byte(`{{define "content"}}<div>Voters Content</div>{{end}}`)},
		"admin/settings.html":   &fstest.MapFile{Data: []byte(`{{define "content"}}<div>Settings Content</div>{{end}}`)},
	}

	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

	// Setup dependencies
	repo, _ := repository.New(":memory:")
	log := logger.New()
	settingsService := services.NewSettingsService(log, repo)
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, repo, derbynetClient)
	carService := services.NewCarService(log, repo, derbynetClient)
	voterService := services.NewVoterService(log, repo, settingsService)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, repo, settingsService, derbynetClient)
	adminAuth := auth.New("test-password")
	hub := websocket.New(log, settingsService)

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

	// Login to get auth cookie
	token, _ := adminAuth.Login("test-password")
	authCookie := &http.Cookie{
		Name:  auth.CookieName,
		Value: token,
	}

	return h, authCookie
}

func TestHandleIndex(t *testing.T) {
	h, _ := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Index Page") {
		t.Error("expected index page content")
	}
}

func TestHandleAdminDashboard(t *testing.T) {
	h, authCookie := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(authCookie)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Admin Dashboard") {
		t.Error("expected dashboard title in response")
	}
	if !strings.Contains(body, "Dashboard Content") {
		t.Error("expected dashboard content in response")
	}
}

func TestHandleAdminCategories(t *testing.T) {
	h, authCookie := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	req.AddCookie(authCookie)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Manage Categories") {
		t.Error("expected categories title in response")
	}
}

func TestHandleAdminResults(t *testing.T) {
	h, authCookie := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/results", nil)
	req.AddCookie(authCookie)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Voting Results") {
		t.Error("expected results title in response")
	}
}

func TestHandleAdminVoters(t *testing.T) {
	h, authCookie := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/voters", nil)
	req.AddCookie(authCookie)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Manage Voters") {
		t.Error("expected voters title in response")
	}
}

func TestHandleAdminSettings(t *testing.T) {
	h, authCookie := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
	req.AddCookie(authCookie)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Admin Settings") {
		t.Error("expected settings title in response")
	}
}

func TestHandleAdminCars(t *testing.T) {
	h, authCookie := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/cars", nil)
	req.AddCookie(authCookie)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Cars Content") {
		t.Error("expected cars content in response")
	}
}

func TestHandleLoginPage(t *testing.T) {
	h, _ := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Login Page") {
		t.Error("expected login page content")
	}
}

func TestHandleVotePage(t *testing.T) {
	h, _ := setupHandlersWithTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/vote/TEST-QR", nil)
	rec := httptest.NewRecorder()

	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Vote Page") {
		t.Error("expected vote page content")
	}
}
