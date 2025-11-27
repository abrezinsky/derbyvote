package handlers

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/abrezinsky/derbyvote/internal/auth"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/websocket"
)

// NewStaticServer creates a static file server from an fs.FS
func NewStaticServer(staticFS fs.FS) http.Handler {
	return http.FileServer(http.FS(staticFS))
}

// AdminPageData holds the data passed to admin templates
type AdminPageData struct {
	Title     string
	PageTitle string
	ActiveNav string
}

// Templates holds all parsed HTML templates
type Templates struct {
	Index           *template.Template
	Vote            *template.Template
	AdminLogin      *template.Template
	AdminDashboard  *template.Template
	AdminCategories *template.Template
	AdminCars       *template.Template
	AdminResults    *template.Template
	AdminVoters     *template.Template
	AdminSettings   *template.Template
}

// Handlers holds all HTTP handler dependencies
type Handlers struct {
	Voting       services.VotingServicer
	Category     services.CategoryServicer
	Voter        services.VoterServicer
	Car          services.CarServicer
	Settings     services.SettingsServicer
	Results      services.ResultsServicer
	Auth         *auth.Auth
	Hub          *websocket.Hub
	Log          HTTPLogger
	templates    *Templates
	staticServer http.Handler
}

// HTTPLogger is an interface for loggers that support HTTP logging control
type HTTPLogger interface {
	IsHTTPLoggingEnabled() bool
}

// New creates a new Handlers instance with all dependencies
func New(
	voting services.VotingServicer,
	category services.CategoryServicer,
	voter services.VoterServicer,
	car services.CarServicer,
	settings services.SettingsServicer,
	results services.ResultsServicer,
	templatesFS fs.FS,
	staticServer http.Handler,
	adminAuth *auth.Auth,
	hub *websocket.Hub,
	log HTTPLogger,
) (*Handlers, error) {
	templates, err := loadTemplates(templatesFS)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &Handlers{
		Voting:       voting,
		Category:     category,
		Voter:        voter,
		Car:          car,
		Settings:     settings,
		Results:      results,
		Auth:         adminAuth,
		Hub:          hub,
		Log:          log,
		templates:    templates,
		staticServer: staticServer,
	}, nil
}

// NoopHTTPLogger is a test logger that always returns false for HTTP logging
type NoopHTTPLogger struct{}

func (NoopHTTPLogger) IsHTTPLoggingEnabled() bool { return false }

// NewForTesting creates a Handlers instance without loading templates (for testing API endpoints)

func NewForTesting(
	voting services.VotingServicer,
	category services.CategoryServicer,
	voter services.VoterServicer,
	car services.CarServicer,
	settings services.SettingsServicer,
	results services.ResultsServicer,
) *Handlers {
	// Create a test auth with a known password
	testAuth := auth.New("test-password")
	return &Handlers{
		Voting:   voting,
		Category: category,
		Voter:    voter,
		Car:      car,
		Settings: settings,
		Results:  results,
		Auth:     testAuth,
		Log:      NoopHTTPLogger{},
		// templates left nil - API endpoints don't use templates
	}
}

// loadTemplates parses all templates once at startup
func loadTemplates(templatesFS fs.FS) (*Templates, error) {
	t := &Templates{}
	var err error

	if t.Index, err = template.ParseFS(templatesFS, "index.html"); err != nil {
		return nil, fmt.Errorf("index template: %w", err)
	}
	if t.Vote, err = template.ParseFS(templatesFS, "voter/vote.html"); err != nil {
		return nil, fmt.Errorf("vote template: %w", err)
	}
	if t.AdminLogin, err = template.ParseFS(templatesFS, "admin/login.html"); err != nil {
		return nil, fmt.Errorf("admin login template: %w", err)
	}
	if t.AdminDashboard, err = template.ParseFS(templatesFS, "admin/layout.html", "admin/dashboard.html"); err != nil {
		return nil, fmt.Errorf("admin dashboard template: %w", err)
	}
	if t.AdminCategories, err = template.ParseFS(templatesFS, "admin/layout.html", "admin/categories.html"); err != nil {
		return nil, fmt.Errorf("admin categories template: %w", err)
	}
	if t.AdminCars, err = template.ParseFS(templatesFS, "admin/layout.html", "admin/cars.html"); err != nil {
		return nil, fmt.Errorf("admin cars template: %w", err)
	}
	if t.AdminResults, err = template.ParseFS(templatesFS, "admin/layout.html", "admin/results.html"); err != nil {
		return nil, fmt.Errorf("admin results template: %w", err)
	}
	if t.AdminVoters, err = template.ParseFS(templatesFS, "admin/layout.html", "admin/voters.html"); err != nil {
		return nil, fmt.Errorf("admin voters template: %w", err)
	}
	if t.AdminSettings, err = template.ParseFS(templatesFS, "admin/layout.html", "admin/settings.html"); err != nil {
		return nil, fmt.Errorf("admin settings template: %w", err)
	}

	return t, nil
}
