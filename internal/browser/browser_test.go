package browser

import (
	"fmt"
	"strings"
	"testing"
)

// mockCommander records command executions for testing
type mockCommander struct {
	lastCommand string
	lastArgs    []string
	startError  error
}

func (m *mockCommander) Start(name string, args ...string) error {
	m.lastCommand = name
	m.lastArgs = args
	return m.startError
}

func TestOpenWithCommander_Linux(t *testing.T) {
	mock := &mockCommander{}
	url := "http://localhost:8081/admin"

	err := OpenWithCommander(url, mock, "linux")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if mock.lastCommand != "xdg-open" {
		t.Errorf("expected command 'xdg-open', got '%s'", mock.lastCommand)
	}

	if len(mock.lastArgs) != 1 || mock.lastArgs[0] != url {
		t.Errorf("expected args [%s], got %v", url, mock.lastArgs)
	}
}

func TestOpenWithCommander_Darwin(t *testing.T) {
	mock := &mockCommander{}
	url := "http://localhost:8081/admin"

	err := OpenWithCommander(url, mock, "darwin")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if mock.lastCommand != "open" {
		t.Errorf("expected command 'open', got '%s'", mock.lastCommand)
	}

	if len(mock.lastArgs) != 1 || mock.lastArgs[0] != url {
		t.Errorf("expected args [%s], got %v", url, mock.lastArgs)
	}
}

func TestOpenWithCommander_Windows(t *testing.T) {
	mock := &mockCommander{}
	url := "http://localhost:8081/admin"

	err := OpenWithCommander(url, mock, "windows")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if mock.lastCommand != "rundll32" {
		t.Errorf("expected command 'rundll32', got '%s'", mock.lastCommand)
	}

	expectedArgs := []string{"url.dll,FileProtocolHandler", url}
	if len(mock.lastArgs) != 2 || mock.lastArgs[0] != expectedArgs[0] || mock.lastArgs[1] != expectedArgs[1] {
		t.Errorf("expected args %v, got %v", expectedArgs, mock.lastArgs)
	}
}

func TestOpenWithCommander_UnsupportedPlatform(t *testing.T) {
	mock := &mockCommander{}
	url := "http://localhost:8081/admin"

	err := OpenWithCommander(url, mock, "freebsd")

	if err == nil {
		t.Fatal("expected error for unsupported platform, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported platform") {
		t.Errorf("expected 'unsupported platform' in error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "freebsd") {
		t.Errorf("expected platform name 'freebsd' in error, got: %v", err)
	}
}

func TestOpenWithCommander_CommandError(t *testing.T) {
	mock := &mockCommander{
		startError: fmt.Errorf("command execution failed"),
	}
	url := "http://localhost:8081/admin"

	err := OpenWithCommander(url, mock, "linux")

	if err == nil {
		t.Fatal("expected error from commander, got nil")
	}

	if err.Error() != "command execution failed" {
		t.Errorf("expected 'command execution failed', got: %v", err)
	}
}

func TestOpenWithCommander_DifferentURLs(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost:8081/admin"},
		{"LAN IP", "http://192.168.1.100:8081/admin"},
		{"with path", "http://localhost:8081/admin/settings"},
		{"HTTPS", "https://example.com/admin"},
		{"with query", "http://localhost:8081/admin?tab=settings"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockCommander{}
			err := OpenWithCommander(tc.url, mock, "linux")

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if mock.lastArgs[0] != tc.url {
				t.Errorf("expected URL '%s', got '%s'", tc.url, mock.lastArgs[0])
			}
		})
	}
}

func TestOpen_CallsOpenWithCommander(t *testing.T) {
	// Save and restore the original commander
	originalCommander := defaultCommander
	defer func() { defaultCommander = originalCommander }()

	// Replace with mock
	mock := &mockCommander{}
	defaultCommander = mock

	url := "http://localhost:8081/admin"
	err := Open(url)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the mock was called
	if mock.lastCommand == "" {
		t.Error("expected commander to be called, but it wasn't")
	}

	// Verify URL was passed through
	found := false
	for _, arg := range mock.lastArgs {
		if arg == url {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected URL '%s' in args, got %v", url, mock.lastArgs)
	}
}

func TestOpen_UsesRuntimeGOOS(t *testing.T) {
	// This test verifies that Open() works without errors on the current platform
	// We can't verify the browser actually opens without mocking, but we can
	// verify it doesn't panic or return unexpected errors

	// Note: This will attempt to open the browser on the test machine
	// Comment this test out if running in CI without display
	t.Skip("Skipping actual browser open test - uncomment to test manually")

	err := Open("http://localhost:8081/admin")
	if err != nil {
		t.Logf("Open returned error (may be expected in CI): %v", err)
	}
}

func TestRealCommander_Start(t *testing.T) {
	commander := RealCommander{}

	// Test with a safe command that exists on all platforms
	// We'll use a command that's likely to fail but won't cause harm
	err := commander.Start("nonexistent-command-xyz-123")

	// We expect an error because the command doesn't exist
	if err == nil {
		t.Error("expected error for nonexistent command, got nil")
	}
}
