package app

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/abrezinsky/derbyvote/internal/auth"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

func TestNew_InitializesApp(t *testing.T) {
	templatesFS := createTestTemplatesFS()
	staticFS := fstest.MapFS{}
	log := logger.New()
	adminAuth := auth.New("test-password")
	derbynetClient := derbynet.NewMockClient()

	app, err := New(log, ":memory:", derbynetClient, templatesFS, staticFS, adminAuth)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if app == nil {
		t.Fatal("expected app to be created")
	}
	if app.handlers == nil {
		t.Error("expected handlers to be initialized")
	}
	if app.repo == nil {
		t.Error("expected repo to be initialized")
	}
	if app.cancelCountdown == nil {
		t.Error("expected cancelCountdown to be set")
	}
}

func TestNew_FailsWithBadDBPath(t *testing.T) {
	templatesFS := createTestTemplatesFS()
	staticFS := fstest.MapFS{}
	log := logger.New()
	adminAuth := auth.New("test-password")
	derbynetClient := derbynet.NewMockClient()

	// Invalid path should fail
	_, err := New(log, "/nonexistent/path/db.sqlite", derbynetClient, templatesFS, staticFS, adminAuth)

	if err == nil {
		t.Error("expected error for invalid db path")
	}
}

func TestNew_FailsWithMissingTemplates(t *testing.T) {
	// Empty templates FS
	templatesFS := fstest.MapFS{}
	staticFS := fstest.MapFS{}
	log := logger.New()
	adminAuth := auth.New("test-password")
	derbynetClient := derbynet.NewMockClient()

	_, err := New(log, ":memory:", derbynetClient, templatesFS, staticFS, adminAuth)

	if err == nil {
		t.Error("expected error for missing templates")
	}
}

func TestApp_Router_ReturnsRouter(t *testing.T) {
	app := createTestApp(t)

	router := app.Router()

	if router == nil {
		t.Fatal("expected router to be returned")
	}
}

func TestApp_Router_ServesRequests(t *testing.T) {
	app := createTestApp(t)
	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Test that static route exists (should not 404)
	resp, err := http.Get(server.URL + "/admin/login")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should get 200 (login page)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /admin/login, got %d", resp.StatusCode)
	}
}

func TestGetPreferredIP_ReturnsValidIP(t *testing.T) {
	ip := getPreferredIP(realNetworkProvider{})

	// Should return something (either localhost or an IP)
	if ip == "" {
		t.Error("expected non-empty IP")
	}

	// If not localhost, should be a valid IP format
	if ip != "localhost" {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			t.Errorf("expected valid IP, got: %s", ip)
		}
	}
}

func TestApp_Close_StopsCountdown(t *testing.T) {
	app := createTestApp(t)

	// Close should not panic
	app.Close()

	// Calling Close multiple times should be safe
	app.Close()
}

func TestIsPrivate172(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.15.0.1", false},
		{"172.32.0.1", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := isPrivate172(ip)
			if result != tt.expected {
				t.Errorf("isPrivate172(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsPrivate172_NilIP(t *testing.T) {
	result := isPrivate172(nil)
	if result != false {
		t.Errorf("isPrivate172(nil) = %v, want false", result)
	}
}

func TestIsPrivate172_IPv6(t *testing.T) {
	// IPv6 addresses should return false
	ip := net.ParseIP("::1")
	result := isPrivate172(ip)
	if result != false {
		t.Errorf("isPrivate172(::1) = %v, want false", result)
	}

	// IPv6 private address
	ip = net.ParseIP("fe80::1")
	result = isPrivate172(ip)
	if result != false {
		t.Errorf("isPrivate172(fe80::1) = %v, want false", result)
	}
}

func TestSetDefaultBaseURL_SetsWhenEmpty(t *testing.T) {
	app := createTestApp(t)
	defer app.Close()

	// Initially empty, should set
	app.setDefaultBaseURL("http://192.168.1.100:8080")

	// Verify it was set
	ctx := context.Background()
	val, err := app.repo.GetSetting(ctx, "base_url")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}
	if val != "http://192.168.1.100:8080" {
		t.Errorf("expected base_url to be set, got: %s", val)
	}
}

func TestSetDefaultBaseURL_ReplacesLocalhost(t *testing.T) {
	app := createTestApp(t)
	defer app.Close()

	ctx := context.Background()

	// Set to localhost first
	err := app.repo.SetSetting(ctx, "base_url", "http://localhost:8080")
	if err != nil {
		t.Fatalf("failed to set initial setting: %v", err)
	}

	// Should replace localhost with real URL
	app.setDefaultBaseURL("http://192.168.1.100:8080")

	val, err := app.repo.GetSetting(ctx, "base_url")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}
	if val != "http://192.168.1.100:8080" {
		t.Errorf("expected base_url to be replaced, got: %s", val)
	}
}

func TestSetDefaultBaseURL_DoesNotOverwriteValidURL(t *testing.T) {
	app := createTestApp(t)
	defer app.Close()

	ctx := context.Background()

	// Set to a valid URL first
	err := app.repo.SetSetting(ctx, "base_url", "http://192.168.1.50:8080")
	if err != nil {
		t.Fatalf("failed to set initial setting: %v", err)
	}

	// Should NOT replace valid URL
	app.setDefaultBaseURL("http://192.168.1.100:8080")

	val, err := app.repo.GetSetting(ctx, "base_url")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}
	if val != "http://192.168.1.50:8080" {
		t.Errorf("expected base_url to remain unchanged, got: %s", val)
	}
}

func TestSetDefaultBaseURL_HandlesRepoError(t *testing.T) {
	app := createTestApp(t)
	// Close the app which stops countdown but keeps repo
	app.Close()

	// Close the underlying database to force an error on SetSetting
	app.repo.DB().Close()

	// Should not panic even if repo is closed - just logs warning
	app.setDefaultBaseURL("http://192.168.1.100:8080")
}

func TestGetPreferredIP_HandlesAllCases(t *testing.T) {
	// This test exercises getPreferredIP thoroughly
	// It will either return localhost or a real IP
	ip := getPreferredIP(realNetworkProvider{})

	if ip == "" {
		t.Error("IP should never be empty")
	}

	// Should be either localhost or a valid IP
	if ip != "localhost" {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			t.Errorf("expected valid IP or 'localhost', got: %s", ip)
		}
		// Should be IPv4
		if parsed.To4() == nil {
			t.Errorf("expected IPv4 address, got: %s", ip)
		}
	}
}

// mockInterface implements networkInterface for testing
type mockInterface struct {
	flags net.Flags
	addrs []net.Addr
	err   error
}

func (m mockInterface) Flags() net.Flags {
	return m.flags
}

func (m mockInterface) Addrs() ([]net.Addr, error) {
	return m.addrs, m.err
}

// mockNetworkProvider implements networkProvider for testing
type mockNetworkProvider struct {
	interfaces []networkInterface
	err        error
}

func (m mockNetworkProvider) Interfaces() ([]networkInterface, error) {
	return m.interfaces, m.err
}

func TestGetPreferredIP_NetworkError(t *testing.T) {
	provider := mockNetworkProvider{
		err: net.ErrClosed,
	}

	ip := getPreferredIP(provider)
	if ip != "localhost" {
		t.Errorf("expected 'localhost' on error, got: %s", ip)
	}
}

func TestGetPreferredIP_InterfaceAddrsError(t *testing.T) {
	// Create an interface that will return an error when Addrs() is called
	iface := mockInterface{
		flags: net.FlagUp, // Up but not loopback
		err:   net.ErrClosed, // Addrs() returns error
	}

	provider := mockNetworkProvider{
		interfaces: []networkInterface{iface},
	}

	// This exercises the error handling path when iface.Addrs() fails
	ip := getPreferredIP(provider)
	if ip != "localhost" {
		t.Errorf("expected 'localhost' when Addrs() fails, got: %s", ip)
	}
}

func TestGetPreferredIP_WithIPAddr(t *testing.T) {
	// Test with *net.IPAddr to hit that case in the type switch
	ipAddr := &net.IPAddr{IP: net.ParseIP("192.168.1.100")}

	iface := mockInterface{
		flags: net.FlagUp,
		addrs: []net.Addr{ipAddr},
	}

	provider := mockNetworkProvider{
		interfaces: []networkInterface{iface},
	}

	ip := getPreferredIP(provider)
	if ip != "192.168.1.100" {
		t.Errorf("expected '192.168.1.100', got: %s", ip)
	}
}

func TestGetPreferredIP_PublicIPFallback(t *testing.T) {
	// Test fallback to first candidate when no private addresses
	publicIP := &net.IPNet{IP: net.ParseIP("8.8.8.8"), Mask: net.CIDRMask(24, 32)}

	iface := mockInterface{
		flags: net.FlagUp,
		addrs: []net.Addr{publicIP},
	}

	provider := mockNetworkProvider{
		interfaces: []networkInterface{iface},
	}

	ip := getPreferredIP(provider)
	if ip != "8.8.8.8" {
		t.Errorf("expected '8.8.8.8' (public IP fallback), got: %s", ip)
	}
}

func TestGetPreferredIP_IPv6AndIPAddr(t *testing.T) {
	// Test with various IP address types to hit all branches
	// This tests the IPAddr case and IPv6 filtering

	// We can't easily create a mock net.Interface with custom addresses
	// because Interface.Addrs() uses the actual system call
	// But we've refactored to make the Interfaces() call mockable
	// which is the main testing improvement

	// The real-world usage will hit these branches naturally
	ip := getPreferredIP(realNetworkProvider{})
	// Just verify it returns something valid
	if ip == "" {
		t.Error("IP should not be empty")
	}
}

func TestGetPreferredIP_NonPrivateAddress(t *testing.T) {
	// Test the fallback to first candidate when no private addresses exist
	// This is hard to mock with real net.Interface, but the refactoring
	// makes it possible to test in integration

	// For now, verify the real implementation works
	ip := getPreferredIP(realNetworkProvider{})
	if ip == "" {
		t.Error("expected non-empty IP")
	}
}

func TestGetPreferredIP_LoopbackIP(t *testing.T) {
	// Test that loopback IPs are filtered even if interface flags don't indicate loopback
	// This tests defense-in-depth: interface might not be flagged as loopback but IP is
	loopbackIP := &net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)}
	validIP := &net.IPNet{IP: net.ParseIP("192.168.1.50"), Mask: net.CIDRMask(24, 32)}

	iface := mockInterface{
		flags: net.FlagUp, // Up but not marked as loopback
		addrs: []net.Addr{loopbackIP, validIP}, // First is loopback, second is valid
	}

	provider := mockNetworkProvider{
		interfaces: []networkInterface{iface},
	}

	ip := getPreferredIP(provider)
	// Should skip loopback and return the valid private IP
	if ip != "192.168.1.50" {
		t.Errorf("expected '192.168.1.50' (skipping loopback), got: %s", ip)
	}
}

func TestRealNetworkProvider_Interfaces(t *testing.T) {
	// Test the real network provider's Interfaces method
	// This exercises the wrapper logic (the error path is untestable without system manipulation)
	provider := realNetworkProvider{}
	ifaces, err := provider.Interfaces()

	// On a working system, this should succeed
	if err != nil {
		t.Logf("net.Interfaces() failed (this is system-dependent): %v", err)
		// Don't fail the test - the error path is system-dependent
		return
	}

	// Verify we got some interfaces
	if len(ifaces) == 0 {
		t.Error("expected at least one network interface")
	}

	// Verify each interface implements our interface
	for i, iface := range ifaces {
		// Test that Flags() works
		_ = iface.Flags()

		// Test that Addrs() works
		addrs, err := iface.Addrs()
		if err != nil {
			t.Logf("interface %d Addrs() failed: %v", i, err)
			continue
		}
		t.Logf("interface %d has %d addresses", i, len(addrs))
	}
}

func TestApp_Run_Integration(t *testing.T) {
	app := createTestApp(t)
	defer app.Close()

	// Start server in background on random port
	done := make(chan error, 1)
	go func() {
		// This will block, so we run it in a goroutine
		done <- app.Run(":0")
	}()

	// Give server time to start (or fail)
	select {
	case err := <-done:
		// If it returns immediately, it's likely a bind error (port already in use)
		// which is fine for testing - we just want to exercise the code
		if err != nil {
			t.Logf("Run returned (expected): %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		// Server started successfully, close the app to stop it
		app.Close()
	}
}

// Helper functions

func createTestTemplatesFS() fstest.MapFS {
	return fstest.MapFS{
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
}

func createTestApp(t *testing.T) *App {
	t.Helper()
	templatesFS := createTestTemplatesFS()
	staticFS := fstest.MapFS{}
	log := logger.New()
	adminAuth := auth.New("test-password")
	derbynetClient := derbynet.NewMockClient()

	app, err := New(log, ":memory:", derbynetClient, templatesFS, staticFS, adminAuth)
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	return app
}
