package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Commander is an interface for executing commands (for testing)
type Commander interface {
	Start(name string, args ...string) error
}

// RealCommander executes actual commands
type RealCommander struct{}

// Start executes a command and starts it
func (RealCommander) Start(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}

var defaultCommander Commander = RealCommander{}

// Open opens the specified URL in the default browser
func Open(url string) error {
	return OpenWithCommander(url, defaultCommander, runtime.GOOS)
}

// OpenWithCommander opens the URL using the specified commander and OS (for testing)
func OpenWithCommander(url string, commander Commander, goos string) error {
	var name string
	var args []string

	switch goos {
	case "linux":
		name = "xdg-open"
		args = []string{url}
	case "darwin": // macOS
		name = "open"
		args = []string{url}
	case "windows":
		name = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported platform: %s", goos)
	}

	return commander.Start(name, args...)
}
