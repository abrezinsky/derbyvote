package app

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/abrezinsky/derbyvote/internal/auth"
	"github.com/abrezinsky/derbyvote/internal/handlers"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/repository"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/internal/websocket"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// App holds all application dependencies
type App struct {
	log            logger.Logger
	handlers       *handlers.Handlers
	repo           *repository.Repository
	cancelCountdown context.CancelFunc
}

// New creates and initializes a new application instance
func New(log logger.Logger, dbPath string, derbynetClient derbynet.Client, templatesFS, staticFS fs.FS, adminAuth *auth.Auth) (*App, error) {
	repo, err := repository.New(dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize services
	categoryService := services.NewCategoryService(log, repo, derbynetClient)
	carService := services.NewCarService(log, repo, derbynetClient)
	settingsService := services.NewSettingsService(log, repo)
	voterService := services.NewVoterService(log, repo, settingsService)
	votingService := services.NewVotingService(log, repo, categoryService, carService, settingsService)
	resultsService := services.NewResultsService(log, repo, settingsService, derbynetClient)

	// Initialize WebSocket hub with DI
	hub := websocket.New(log, settingsService)
	hub.Start()
	settingsService.SetBroadcaster(hub)

	// Start countdown with context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go hub.StartVotingCountdown(ctx)

	// Create static file server
	staticServer := handlers.NewStaticServer(staticFS)

	// Initialize handlers with hub
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
		hub,
		log,
	)
	if err != nil {
		cancel() // Clean up countdown goroutine
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	return &App{
		log:             log,
		handlers:        h,
		repo:            repo,
		cancelCountdown: cancel,
	}, nil
}

// Router returns the configured HTTP router
func (a *App) Router() chi.Router {
	return a.handlers.Router()
}

// Close performs graceful shutdown of app resources
func (a *App) Close() {
	if a.cancelCountdown != nil {
		a.cancelCountdown()
	}
}

// Run starts the HTTP server
func (a *App) Run(addr string) error {
	// Set default base URL if not configured, using detected LAN IP
	ip := getPreferredIP(realNetworkProvider{})
	baseURL := fmt.Sprintf("http://%s%s", ip, addr)
	a.setDefaultBaseURL(baseURL)

	a.log.Info("Server starting", "url", baseURL)
	a.log.Info("Admin URL", "url", baseURL+"/admin")
	return http.ListenAndServe(addr, a.Router())
}

// setDefaultBaseURL sets the base URL setting if not already configured
// or if current value uses localhost (which isn't useful for QR codes)
func (a *App) setDefaultBaseURL(baseURL string) {
	ctx := context.Background()
	existing, _ := a.repo.GetSetting(ctx, "base_url")

	// Set default if empty or if current value uses localhost
	needsUpdate := existing == "" || strings.Contains(existing, "localhost")
	if needsUpdate {
		if err := a.repo.SetSetting(ctx, "base_url", baseURL); err != nil {
			a.log.Warn("Failed to set default base_url", "error", err)
		} else {
			a.log.Info("Default base URL set", "url", baseURL)
		}
	}
}

// networkInterface wraps net.Interface for testing
type networkInterface interface {
	Flags() net.Flags
	Addrs() ([]net.Addr, error)
}

// realInterface wraps a real net.Interface
type realInterface struct {
	iface net.Interface
}

func (r realInterface) Flags() net.Flags {
	return r.iface.Flags
}

func (r realInterface) Addrs() ([]net.Addr, error) {
	return r.iface.Addrs()
}

// networkProvider is an interface for getting network interfaces (for testing)
type networkProvider interface {
	Interfaces() ([]networkInterface, error)
}

// realNetworkProvider implements networkProvider using actual net package
type realNetworkProvider struct{}

func (realNetworkProvider) Interfaces() ([]networkInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	result := make([]networkInterface, len(ifaces))
	for i, iface := range ifaces {
		result[i] = realInterface{iface: iface}
	}
	return result, nil
}

// getPreferredIP returns the best IP address for LAN access.
// Prefers private network addresses (192.168.x.x, 10.x.x.x, 172.16-31.x.x).
// Falls back to localhost if no suitable address is found.
func getPreferredIP(provider networkProvider) string {
	ifaces, err := provider.Interfaces()
	if err != nil {
		return "localhost"
	}

	var candidates []net.IP

	for _, iface := range ifaces {
		// Skip down, loopback, and point-to-point interfaces
		flags := iface.Flags()
		if flags&net.FlagUp == 0 || flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Only consider IPv4 addresses
			if ip == nil || ip.To4() == nil {
				continue
			}

			// Skip loopback
			if ip.IsLoopback() {
				continue
			}

			candidates = append(candidates, ip)
		}
	}

	// Prefer private network addresses
	for _, ip := range candidates {
		ipStr := ip.String()
		if strings.HasPrefix(ipStr, "192.168.") ||
			strings.HasPrefix(ipStr, "10.") ||
			isPrivate172(ip) {
			return ipStr
		}
	}

	// Fall back to any non-loopback if no private address found
	if len(candidates) > 0 {
		return candidates[0].String()
	}

	return "localhost"
}

// isPrivate172 checks if IP is in 172.16.0.0/12 range
func isPrivate172(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31
	}
	return false
}
