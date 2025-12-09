//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/abrezinsky/derbyvote/internal/browser"
	"github.com/abrezinsky/derbyvote/internal/logger"
)

// listenForKeyboard listens for keyboard input on Windows
func listenForKeyboard(adminURL string, appLog *logger.SlogLogger) {
	// Simple line-based reading on Windows (terminal manipulation is more complex)
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
			os.Exit(0)
		case "?":
			printKeyboardHelp()
		}
	}
}
