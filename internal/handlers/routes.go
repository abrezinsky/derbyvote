package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// conditionalHTTPLogger only logs HTTP requests when HTTP logging is enabled
func (h *Handlers) conditionalHTTPLogger(next http.Handler) http.Handler {
	logger := middleware.Logger(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.Log != nil && h.Log.IsHTTPLoggingEnabled() {
			logger.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// Router returns a configured chi router with all routes
func (h *Handlers) Router() chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(h.conditionalHTTPLogger) // Custom conditional HTTP logger
	r.Use(middleware.Recoverer)
	r.Use(middleware.RedirectSlashes)
	r.Use(middleware.Timeout(60 * time.Second))

	// Static files (served from embedded filesystem)
	r.Handle("/static/*", http.StripPrefix("/static/", h.staticServer))

	// Home page
	r.Get("/", h.handleIndex)

	// WebSocket
	r.Get("/ws", h.Hub.ServeWs)

	// Voting pages (public)
	r.Get("/vote/new", h.handleGenerateVoteCode) // Must come before /vote/{qrCode}
	r.Get("/vote/{qrCode}", h.handleVotePage)

	// Voting API (public)
	r.Get("/api/vote-data/{qrCode}", h.handleGetVoteData)
	r.Post("/api/vote", h.handleSubmitVote)

	// Car photo proxy (public)
	r.Get("/cars/{id}/photo", h.handleCarPhoto)

	// Auth routes (public)
	r.Get("/admin/login", h.handleLoginPage)
	r.Post("/admin/login", h.handleLogin)
	r.Post("/admin/logout", h.handleLogout)

	// Admin pages (protected)
	r.Group(func(r chi.Router) {
		r.Use(h.Auth.RequireAuth)
		r.Get("/admin", h.handleAdminDashboard)
		r.Get("/admin/categories", h.handleAdminCategories)
		r.Get("/admin/cars", h.handleAdminCars)
		r.Get("/admin/results", h.handleAdminResults)
		r.Get("/admin/voters", h.handleAdminVoters)
		r.Get("/admin/settings", h.handleAdminSettings)
	})

	// Admin API (protected)
	r.Group(func(r chi.Router) {
		r.Use(h.Auth.RequireAuthAPI)

		// Categories
		r.Get("/api/admin/categories", h.handleGetCategories)
		r.Post("/api/admin/categories", h.handleCreateCategory)
		r.Put("/api/admin/categories/{id}", h.handleUpdateCategory)
		r.Delete("/api/admin/categories/{id}", h.handleDeleteCategory)

		// Category Groups
		r.Get("/api/admin/category-groups", h.handleGetCategoryGroups)
		r.Post("/api/admin/category-groups", h.handleCreateCategoryGroup)
		r.Get("/api/admin/category-groups/{id}", h.handleGetCategoryGroup)
		r.Put("/api/admin/category-groups/{id}", h.handleUpdateCategoryGroup)
		r.Delete("/api/admin/category-groups/{id}", h.handleDeleteCategoryGroup)

		// Voting Control
		r.Post("/api/admin/voting-control", h.handleSetVotingStatus)
		r.Post("/api/admin/voting-timer", h.handleSetVotingTimer)

		// Stats & Results
		r.Get("/api/admin/stats", h.handleGetStats)
		r.Get("/api/admin/results", h.handleGetResults)
		r.Get("/api/admin/results/conflicts", h.handleGetConflicts)
		r.Get("/api/admin/results/overrides", h.handleGetOverrides)
		r.Post("/api/admin/results/override-winner", h.handleOverrideWinner)
		r.Delete("/api/admin/results/override-winner/{categoryID}", h.handleClearOverride)

		// DerbyNet
		r.Post("/api/admin/sync-derbynet", h.handleSyncDerbyNet)
		r.Post("/api/admin/sync-categories-derbynet", h.handleSyncCategoriesDerbyNet)
		r.Post("/api/admin/push-results-derbynet", h.handlePushResultsDerbyNet)
		r.Post("/api/admin/test-derbynet", h.handleTestDerbyNet)

		// QR Codes
		r.Post("/api/admin/generate-qr", h.handleGenerateQRCodes)
		r.Get("/api/admin/voters/{id}/qr", h.handleGetQRImage)
		r.Get("/api/admin/open-voting-qr", h.handleGetOpenVotingQR)

		// Settings
		r.Get("/api/admin/settings", h.handleGetSettings)
		r.Post("/api/admin/settings", h.handleUpdateSettings)
		r.Put("/api/admin/settings", h.handleUpdateSettings)
		r.Get("/api/admin/voter-types", h.handleGetVoterTypes)

		// Database Management
		r.Post("/api/admin/reset-database", h.handleResetDatabase)
		r.Post("/api/admin/seed-mock-data", h.handleSeedMockData)

		// Voters
		r.Get("/api/admin/voters", h.handleGetVoters)
		r.Post("/api/admin/voters", h.handleCreateVoter)
		r.Put("/api/admin/voters", h.handleUpdateVoter)
		r.Delete("/api/admin/voters/{id}", h.handleDeleteVoter)

		// Cars
		r.Get("/api/admin/cars", h.handleGetCars)
		r.Get("/api/admin/cars/{id}", h.handleGetCar)
		r.Post("/api/admin/cars", h.handleCreateCar)
		r.Put("/api/admin/cars/{id}", h.handleUpdateCar)
		r.Put("/api/admin/cars/{id}/eligibility", h.handleSetCarEligibility)
		r.Delete("/api/admin/cars/{id}", h.handleDeleteCar)
	})

	return r
}
