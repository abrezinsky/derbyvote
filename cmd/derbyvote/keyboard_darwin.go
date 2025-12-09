//go:build darwin
// +build darwin

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"github.com/abrezinsky/derbyvote/internal/browser"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"golang.org/x/sys/unix"
)

// listenForKeyboard listens for keyboard input and performs actions
func listenForKeyboard(adminURL string, appLog *logger.SlogLogger) {
	// Get the current terminal state
	fd := int(os.Stdin.Fd())
	oldState, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		// Can't get terminal state, silently return
		return
	}

	// Create a new state based on the old one
	newState := *oldState
	// Disable canonical mode (line buffering) and echo
	newState.Lflag &^= unix.ICANON | unix.ECHO
	// Set minimum characters to read to 1
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0

	// Apply the new state
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &newState); err != nil {
		return
	}

	// Restore old state when done
	defer unix.IoctlSetTermios(fd, unix.TIOCSETA, oldState)

	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		input := strings.ToLower(string(buf[0]))
		switch input {
		case "a":
			fmt.Printf("%sOpening admin page in browser...%s\n", cyan, reset)
			if err := browser.Open(adminURL); err != nil {
				fmt.Printf("%sError opening browser: %v%s\n", red, err, reset)
			}
		case "h":
			if appLog.IsHTTPLoggingEnabled() {
				appLog.DisableHTTPLogging()
				fmt.Printf("%sHTTP logging disabled%s\n", yellow, reset)
			} else {
				appLog.EnableHTTPLogging()
				fmt.Printf("%sHTTP logging enabled%s\n", green, reset)
			}
		case "l":
			cycleLogLevel(appLog)
		case "q":
			fmt.Printf("%sShutting down server...%s\n", yellow, reset)
			unix.IoctlSetTermios(fd, unix.TIOCSETA, oldState)
			os.Exit(0)
		case "?":
			printKeyboardHelp()
		case "\x03": // Ctrl+C
			fmt.Printf("%sShutting down server...%s\n", yellow, reset)
			unix.IoctlSetTermios(fd, unix.TIOCSETA, oldState)
			os.Exit(0)
		}
	}
}
