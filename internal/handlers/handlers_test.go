package handlers_test

import (
	"net/http"
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

func createTestTemplatesFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":             &fstest.MapFile{Data: []byte(`<html><body>Index</body></html>`)},
		"voter/vote.html":        &fstest.MapFile{Data: []byte(`<html><body>Vote</body></html>`)},
		"admin/login.html":       &fstest.MapFile{Data: []byte(`<html><body>Login</body></html>`)},
		"admin/layout.html":      &fstest.MapFile{Data: []byte(`<html><body>{{template "content" .}}</body></html>{{define "content"}}{{end}}`)},
		"admin/dashboard.html":   &fstest.MapFile{Data: []byte(`{{define "content"}}Dashboard{{end}}`)},
		"admin/categories.html":  &fstest.MapFile{Data: []byte(`{{define "content"}}Categories{{end}}`)},
		"admin/cars.html":        &fstest.MapFile{Data: []byte(`{{define "content"}}Cars{{end}}`)},
		"admin/results.html":     &fstest.MapFile{Data: []byte(`{{define "content"}}Results{{end}}`)},
		"admin/voters.html":      &fstest.MapFile{Data: []byte(`{{define "content"}}Voters{{end}}`)},
		"admin/settings.html":    &fstest.MapFile{Data: []byte(`{{define "content"}}Settings{{end}}`)},
	}
}

func TestNew_WithValidTemplates(t *testing.T) {
	// Create mock filesystem with all required templates
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
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
		t.Fatalf("expected no error, got: %v", err)
	}
	if h == nil {
		t.Fatal("expected handlers to be created")
	}
}

func TestNew_WithMissingVoteTemplate(t *testing.T) {
	// Missing voter/vote.html
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
		},
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "vote template") {
		t.Errorf("expected error to mention 'vote template', got: %v", err)
	}
}

func TestNew_WithMissingAdminTemplate(t *testing.T) {
	// Has vote but missing admin/login.html
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing admin template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "admin login template") {
		t.Errorf("expected error to mention 'admin login template', got: %v", err)
	}
}

func TestNew_WithInvalidTemplateContent(t *testing.T) {
	// Template with invalid syntax
	templatesFS := fstest.MapFS{
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html>{{.InvalidSyntax`), // Invalid template
		},
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for invalid template syntax")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
}

func TestNew_WithMissingIndexTemplate(t *testing.T) {
	// Missing index.html
	templatesFS := fstest.MapFS{
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
		},
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing index template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "index template") {
		t.Errorf("expected error to mention 'index template', got: %v", err)
	}
}

func TestNew_WithMissingAdminDashboardTemplate(t *testing.T) {
	// Has index, vote, and login, but missing admin/dashboard.html
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
		},
		"admin/layout.html": &fstest.MapFile{
			Data: []byte(`<html><body>Layout</body></html>`),
		},
		// Missing admin/dashboard.html
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing admin dashboard template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "admin dashboard template") {
		t.Errorf("expected error to mention 'admin dashboard template', got: %v", err)
	}
}

func TestNew_WithMissingAdminCategoriesTemplate(t *testing.T) {
	// Has all templates up to dashboard, but missing admin/categories.html
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
		},
		"admin/layout.html": &fstest.MapFile{
			Data: []byte(`<html><body>Layout</body></html>`),
		},
		"admin/dashboard.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Dashboard{{end}}`),
		},
		// Missing admin/categories.html
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing admin categories template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "admin categories template") {
		t.Errorf("expected error to mention 'admin categories template', got: %v", err)
	}
}

func TestNew_WithMissingAdminCarsTemplate(t *testing.T) {
	// Has all templates up to categories, but missing admin/cars.html
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
		},
		"admin/layout.html": &fstest.MapFile{
			Data: []byte(`<html><body>Layout</body></html>`),
		},
		"admin/dashboard.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Dashboard{{end}}`),
		},
		"admin/categories.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}Categories{{end}}`),
		},
		// Missing admin/cars.html
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing admin cars template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "admin cars template") {
		t.Errorf("expected error to mention 'admin cars template', got: %v", err)
	}
}

func TestNew_WithMissingAdminResultsTemplate(t *testing.T) {
	// Has all templates up to cars, but missing admin/results.html
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
		},
		"admin/layout.html": &fstest.MapFile{
			Data: []byte(`<html><body>Layout</body></html>`),
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
		// Missing admin/results.html
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing admin results template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "admin results template") {
		t.Errorf("expected error to mention 'admin results template', got: %v", err)
	}
}

func TestNew_WithMissingAdminVotersTemplate(t *testing.T) {
	// Has all templates up to results, but missing admin/voters.html
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
		},
		"admin/layout.html": &fstest.MapFile{
			Data: []byte(`<html><body>Layout</body></html>`),
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
		// Missing admin/voters.html
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing admin voters template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "admin voters template") {
		t.Errorf("expected error to mention 'admin voters template', got: %v", err)
	}
}

func TestNew_WithMissingAdminSettingsTemplate(t *testing.T) {
	// Has all templates except admin/settings.html
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
		},
		"admin/layout.html": &fstest.MapFile{
			Data: []byte(`<html><body>Layout</body></html>`),
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
		// Missing admin/settings.html
	}
	staticFS := fstest.MapFS{}
	staticServer := handlers.NewStaticServer(staticFS)

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

	if err == nil {
		t.Fatal("expected error for missing admin settings template")
	}
	if h != nil {
		t.Error("expected nil handlers on error")
	}
	if !strings.Contains(err.Error(), "admin settings template") {
		t.Errorf("expected error to mention 'admin settings template', got: %v", err)
	}
}

func TestNew_HubInjection(t *testing.T) {
	templatesFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<html><body>Index</body></html>`),
		},
		"voter/vote.html": &fstest.MapFile{
			Data: []byte(`<html><body>Vote</body></html>`),
		},
		"admin/login.html": &fstest.MapFile{
			Data: []byte(`<html><body>Login</body></html>`),
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
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify hub was injected
	if h.Hub != hub {
		t.Error("expected hub to be injected into handlers")
	}
}

func TestNewStaticServer(t *testing.T) {
	staticFS := fstest.MapFS{
		"test.txt": &fstest.MapFile{Data: []byte("hello")},
	}

	server := handlers.NewStaticServer(staticFS)

	if server == nil {
		t.Fatal("expected static server to be created")
	}

	// Verify it implements http.Handler
	var _ http.Handler = server
}

func TestNew_WithCustomStaticServer(t *testing.T) {
	templatesFS := createTestTemplatesFS()

	// Create a custom static server (could be a mock, CDN proxy, etc.)
	customServer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("custom static"))
	})

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
		customServer, // Custom static server injected
		adminAuth,
		hub, handlers.NoopHTTPLogger{},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected handlers to be created")
	}
}

func TestNewForTesting_NoTemplatesNeeded(t *testing.T) {
	repo, _ := repository.New(":memory:")
	log := logger.New()
	settingsService := services.NewSettingsService(log, repo)
	derbynetClient := derbynet.NewMockClient()
	categoryService := services.NewCategoryService(log, repo, derbynetClient)
	carService := services.NewCarService(log, repo, derbynetClient)
	voterService := services.NewVoterService(log, repo, settingsService)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, repo, settingsService, derbynetClient)

	// Should not panic even without templates
	h := handlers.NewForTesting(
		votingService,
		categoryService,
		voterService,
		carService,
		settingsService,
		resultsService,
	)

	if h == nil {
		t.Fatal("expected handlers to be created")
	}
	if h.Auth == nil {
		t.Error("expected test auth to be created")
	}
}
