package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/abrezinsky/derbyvote/internal/app"
	"github.com/abrezinsky/derbyvote/internal/auth"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
	"github.com/abrezinsky/derbyvote/web"
)

// ANSI escape codes
const (
	clearLine = "\033[2K"
	moveUp    = "\033[%dA"
	reset     = "\033[0m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	green     = "\033[32m"
	cyan      = "\033[36m"
	bold      = "\033[1m"
)

// showStartupAnimation displays DerbyVote logo then animated race
func showStartupAnimation(skipRace bool) {
	width := 62
	border := ""
	for i := 0; i < width; i++ {
		border += "═"
	}

	// Show DerbyVote logo first (centered in 62 chars)
	logo := []string{
		"      ____            _          __     __    _          ",
		"     |  _ \\  ___ _ __| |__  _   _\\ \\   / /__ | |_ ___    ",
		"     | | | |/ _ \\ '__| '_ \\| | | |\\ \\ / / _ \\| __/ _ \\   ",
		"     | |_| |  __/ |  | |_) | |_| | \\ V / (_) | ||  __/   ",
		"     |____/ \\___|_|  |_.__/ \\__, |  \\_/ \\___/ \\__\\___|   ",
		"                            |___/                        ",
	}

	fmt.Printf("\n  %s╔%s╗%s\n", cyan, border, reset)
	for _, line := range logo {
		for len(line) < width {
			line += " "
		}
		fmt.Printf("  %s║%s%s%s║%s\n", cyan, yellow, line, cyan, reset)
	}
	fmt.Printf("  %s╚%s╝%s\n", cyan, border, reset)

	if skipRace {
		fmt.Print("\n")
		return
	}

	// For race animation, change bottom to middle divider and continue
	fmt.Printf(moveUp, 1)
	fmt.Printf("%s  %s╠%s╣%s\n", clearLine, cyan, border, reset)

	// Car designs
	cars := []struct {
		art   string
		color string
	}{
		{`__/¯¯\__`, red},
		{`=<[##]>=`, blue},
		{`-=[==]=-`, green},
	}

	trackLen := 62
	finishLine := "║"

	// Print initial track
	for i := 0; i < 3; i++ {
		track := ""
		for j := 0; j < trackLen; j++ {
			track += " "
		}
		fmt.Printf("  %s║%s%s%s\n", cyan, track, finishLine, reset)
	}
	fmt.Printf("  %s╚%s╝%s\n", cyan, border, reset)

	// Move cursor back up
	fmt.Printf(moveUp, 4)

	// Animate cars racing with randomized speeds (all different)
	rand.Seed(time.Now().UnixNano())
	positions := []int{0, 0, 0}
	speeds := []int{3, 4, 5}
	rand.Shuffle(len(speeds), func(i, j int) { speeds[i], speeds[j] = speeds[j], speeds[i] })
	finished := []bool{false, false, false}
	finishTimes := []int{0, 0, 0}
	finishPos := trackLen - 8 // Cars finish at right edge

	for frame := 0; frame < 20; frame++ {
		for i := range positions {
			if !finished[i] {
				positions[i] += speeds[i]
				if positions[i] >= finishPos {
					positions[i] = finishPos
					finished[i] = true
					finishTimes[i] = frame
				}
			}
		}

		for i, car := range cars {
			padding := ""
			for j := 0; j < positions[i]; j++ {
				padding += " "
			}
			remaining := trackLen - positions[i] - 8
			if remaining < 0 {
				remaining = 0
			}
			trail := ""
			for j := 0; j < remaining; j++ {
				trail += " "
			}
			fmt.Printf("%s  %s║%s%s%s%s%s║%s\n", clearLine, cyan, padding, car.color, car.art, reset, trail, reset)
		}
		fmt.Printf("%s  %s╚%s╝%s\n", clearLine, cyan, border, reset)

		if frame < 19 {
			fmt.Printf(moveUp, 4)
		}

		time.Sleep(80 * time.Millisecond)
	}

	// Find winner (lowest finish time)
	winner := 0
	for i := 1; i < 3; i++ {
		if finishTimes[i] < finishTimes[winner] {
			winner = i
		}
	}

	// Redraw with times (inside the box)
	fmt.Printf(moveUp, 4)
	for i, car := range cars {
		t := float64(finishTimes[i]*80+400) / 1000.0
		winnerStr := "      "
		if i == winner {
			winnerStr = yellow + "WINNER" + reset
		}
		// Format: ║ 0.72s WINNER [padding] car ║
		// 62 total - 8 car - 13 (time+winner) = 41 spaces
		timeStr := fmt.Sprintf(" %.2fs %s", t, winnerStr)
		spaces := ""
		for j := 0; j < 41; j++ {
			spaces += " "
		}
		fmt.Printf("%s  %s║%s%s%s%s%s║%s\n", clearLine, cyan, timeStr, spaces, car.color, car.art, cyan, reset)
	}
	fmt.Printf("%s  %s╚%s╝%s\n\n", clearLine, cyan, border, reset)
}

var (
	version = "dev"
)

// cycleLogLevel cycles through debug -> info -> warn -> error
func cycleLogLevel(appLog *logger.SlogLogger) {
	current := appLog.GetLevel()
	var next string
	var nextLevel string

	switch current.String() {
	case "DEBUG":
		nextLevel = "info"
		next = "info"
	case "INFO":
		nextLevel = "warn"
		next = "warn"
	case "WARN":
		nextLevel = "error"
		next = "error"
	case "ERROR":
		nextLevel = "debug"
		next = "debug"
	default:
		nextLevel = "info"
		next = "info"
	}

	appLog.SetLevel(logger.ParseLevel(nextLevel))
	fmt.Printf("%sLog level: %s%s%s\n", green, yellow, next, reset)
}

// printKeyboardHelp displays all available keyboard shortcuts
func printKeyboardHelp() {
	fmt.Printf("\n%s%s  Keyboard Shortcuts:%s\n", bold, green, reset)
	fmt.Printf("    %sa%s      - Open admin page in browser\n", cyan, reset)
	fmt.Printf("    %sh%s      - Toggle HTTP request logging\n", cyan, reset)
	fmt.Printf("    %sl%s      - Cycle log level (debug → info → warn → error)\n", cyan, reset)
	fmt.Printf("    %sq%s      - Quit server\n", cyan, reset)
	fmt.Printf("    %s?%s      - Show this help\n\n", cyan, reset)
}

func main() {
	port := flag.Int("port", 8081, "HTTP server port")
	dbPath := flag.String("db", "voting.db", "SQLite database path")
	adminPw := flag.String("adminpw", "", "Admin password (auto-generated if not set)")
	logLevel := flag.String("loglevel", "info", "Log level (debug, info, warn, error)")
	noAnimate := flag.Bool("noanimate", false, "Show logo only, skip race animation")
	noKeyboard := flag.Bool("nokeyboard", false, "Disable keyboard shortcuts")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `DerbyVote - Pinewood Derby Voting System

Usage:
  derbyvote [options]

Options:
  -port int      HTTP server port (default 8081)
  -db string     SQLite database path (default "voting.db")
  -adminpw str   Admin password (auto-generated if not set)
  -loglevel str  Log level: debug, info, warn, error (default "info")
  -noanimate     Show logo only, skip race animation
  -nokeyboard    Disable keyboard shortcuts
  -version       Show version and exit
  -help          Show this help message

Keyboard Shortcuts (when enabled):
  a              Open admin page in browser
  h              Toggle HTTP request logging
  l              Cycle log level (debug → info → warn → error)
  q              Quit server
  ?              Show keyboard help

Examples:
  derbyvote                          # Run on port 8081 with voting.db
  derbyvote -port 8080               # Run on port 8080
  derbyvote -db /data/derby.db       # Use custom database path
  derbyvote -adminpw secret123       # Use specific admin password
  derbyvote -nokeyboard              # Disable keyboard shortcuts
  derbyvote -port 80 -db prod.db     # Production example

`)
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("derbyvote %s\n", version)
		os.Exit(0)
	}

	// Show startup animation or just logo
	showStartupAnimation(*noAnimate)

	// Setup admin authentication
	password := *adminPw
	if password == "" {
		password = auth.GeneratePassword()
	}
	adminAuth := auth.New(password)

	// Create logger with specified level
	appLog := logger.NewWithLevel(logger.ParseLevel(*logLevel))

	// Create DerbyNet client - URL is set dynamically from settings
	derbynetClient := derbynet.NewHTTPClient("", appLog)

	a, err := app.New(appLog, *dbPath, derbynetClient, web.GetTemplatesFS(), web.GetStaticFS(), adminAuth)
	if err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	appLog.Info("Admin password", "password", password)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- a.Run(addr)
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)

	// Get base URL for browser opening
	adminURL := fmt.Sprintf("http://localhost:%d/admin", *port)

	// Print keyboard shortcuts and start listener (unless disabled)
	if !*noKeyboard {
		fmt.Printf("\n%s%s  Keyboard shortcuts:%s\n", bold, green, reset)
		fmt.Printf("    %sa%s      - Open admin page in browser\n", cyan, reset)
		fmt.Printf("    %sh%s      - Toggle HTTP request logging\n", cyan, reset)
		fmt.Printf("    %sl%s      - Cycle log level (debug → info → warn → error)\n", cyan, reset)
		fmt.Printf("    %sq%s      - Quit server\n", cyan, reset)
		fmt.Printf("    %s?%s      - Show help\n\n", cyan, reset)

		// Start keyboard listener in goroutine
		go listenForKeyboard(adminURL, appLog)
	} else {
		fmt.Printf("\n%sKeyboard shortcuts disabled (use -nokeyboard=false to enable)%s\n\n", yellow, reset)
	}

	// Wait for server error or signal
	if err := <-serverErr; err != nil {
		log.Fatal(err)
	}
}
