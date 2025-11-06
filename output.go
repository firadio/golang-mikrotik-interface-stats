package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// clearScreen clears the terminal screen using ANSI escape codes
func clearScreen() {
	// Use ANSI escape codes to clear screen and move cursor to home
	// This works on:
	// - Linux/macOS terminals
	// - Windows 10+ with Virtual Terminal Processing enabled
	// - Windows Terminal, PowerShell 7+, Git Bash, etc.
	//
	// \033[2J - clear entire screen
	// \033[H - move cursor to home position (1,1)
	fmt.Print("\033[2J\033[H")
}

// moveCursorHome moves cursor to top-left without clearing screen
func moveCursorHome() {
	// \033[H - move cursor to home position (1,1)
	// This allows overwriting previous content without clearing
	// More efficient than full screen clear, reduces flicker
	fmt.Print("\033[H")
}

// OutputWriter defines the interface for output handling
type OutputWriter interface {
	WriteHeader()
	WriteStats(timestamp time.Time, stats map[string]*RateInfo)
	Close()
}

// RateInfo holds rate information for display
type RateInfo struct {
	InterfaceName string
	RxRate        float64
	TxRate        float64
}

// TerminalOutput handles terminal output (refresh or append mode)
type TerminalOutput struct {
	refreshMode      bool
	rateUnit         string
	rateScale        string
	uplinkInterfaces map[string]bool // Set of uplink interface names
}

// NewTerminalOutput creates a new terminal output handler
func NewTerminalOutput(refreshMode bool, rateUnit, rateScale string, uplinkInterfaces []string) *TerminalOutput {
	// Build uplink interface set for fast lookup
	uplinkSet := make(map[string]bool)
	for _, iface := range uplinkInterfaces {
		uplinkSet[iface] = true
	}

	return &TerminalOutput{
		refreshMode:      refreshMode,
		rateUnit:         rateUnit,
		rateScale:        rateScale,
		uplinkInterfaces: uplinkSet,
	}
}

func (t *TerminalOutput) WriteHeader() {
	if t.refreshMode {
		clearScreen()
		fmt.Println("Mikrotik Interface Traffic Monitor")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println("Initializing...")
	} else {
		fmt.Println("\nMonitoring interface traffic (Ctrl+C to stop):")
		fmt.Println(strings.Repeat("=", 80))
	}
}

func (t *TerminalOutput) WriteStats(timestamp time.Time, stats map[string]*RateInfo) {
	timeStr := timestamp.Format("2006-01-02 15:04:05")

	// Sort interface names for consistent ordering
	names := make([]string, 0, len(stats))
	for name := range stats {
		names = append(names, name)
	}
	sort.Strings(names)

	if t.refreshMode {
		// Refresh mode: move cursor to home and overwrite
		// Use moveCursorHome instead of clearScreen to reduce flicker
		moveCursorHome()
		fmt.Println("Mikrotik Interface Traffic Monitor")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("Time: %s\n", timeStr)
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("%-15s %-20s %-20s\n", "Interface", "Upload", "Download")
		fmt.Println(strings.Repeat("-", 80))

		for _, name := range names {
			info := stats[name]
			var downloadRate, uploadRate float64

			// Check if this is an uplink interface
			if t.uplinkInterfaces[name] {
				// Uplink (WAN to ISP): TX=Upload (to internet), RX=Download (from internet)
				// This is the "normal" understanding, no swap needed
				downloadRate = info.RxRate
				uploadRate = info.TxRate
			} else {
				// Downlink (to users/LAN): TX=Download (data to user), RX=Upload (data from user)
				// From user perspective, needs swap
				downloadRate = info.TxRate
				uploadRate = info.RxRate
			}

			downloadFormatted := FormatRate(downloadRate, t.rateUnit, t.rateScale)
			uploadFormatted := FormatRate(uploadRate, t.rateUnit, t.rateScale)
			fmt.Printf("%-15s %-20s %-20s\n", info.InterfaceName, uploadFormatted, downloadFormatted)
		}

		fmt.Println(strings.Repeat("-", 80))
		fmt.Println("Press Ctrl+C to stop")
		// Clear any remaining lines from previous output (if interface count decreased)
		fmt.Print("\033[J")
	} else {
		// Append mode: add new lines
		for _, name := range names {
			info := stats[name]
			var downloadRate, uploadRate float64

			// Check if this is an uplink interface
			if t.uplinkInterfaces[name] {
				// Uplink (WAN to ISP): TX=Upload (to internet), RX=Download (from internet)
				// This is the "normal" understanding, no swap needed
				downloadRate = info.RxRate
				uploadRate = info.TxRate
			} else {
				// Downlink (to users/LAN): TX=Download (data to user), RX=Upload (data from user)
				// From user perspective, needs swap
				downloadRate = info.TxRate
				uploadRate = info.RxRate
			}

			downloadFormatted := FormatRate(downloadRate, t.rateUnit, t.rateScale)
			uploadFormatted := FormatRate(uploadRate, t.rateUnit, t.rateScale)
			fmt.Printf("[%s] %s: Upload: %s  Download: %s\n",
				timeStr, info.InterfaceName, uploadFormatted, downloadFormatted)
		}
	}
}

func (t *TerminalOutput) Close() {
	// Nothing to close for terminal output
}

// LogOutput handles log-style output
type LogOutput struct {
	rateUnit         string
	rateScale        string
	uplinkInterfaces map[string]bool // Set of uplink interface names
}

// NewLogOutput creates a new log output handler
func NewLogOutput(rateUnit, rateScale string, uplinkInterfaces []string) *LogOutput {
	// Build uplink interface set for fast lookup
	uplinkSet := make(map[string]bool)
	for _, iface := range uplinkInterfaces {
		uplinkSet[iface] = true
	}

	return &LogOutput{
		rateUnit:         rateUnit,
		rateScale:        rateScale,
		uplinkInterfaces: uplinkSet,
	}
}

func (l *LogOutput) WriteHeader() {
	log.Println("Mikrotik Interface Traffic Monitor started")
}

func (l *LogOutput) WriteStats(timestamp time.Time, stats map[string]*RateInfo) {
	// Sort interface names for consistent ordering
	names := make([]string, 0, len(stats))
	for name := range stats {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		info := stats[name]
		var downloadRate, uploadRate float64

		// Check if this is an uplink interface
		if l.uplinkInterfaces[name] {
			// Uplink (WAN to ISP): TX=Upload (to internet), RX=Download (from internet)
			// This is the "normal" understanding, no swap needed
			downloadRate = info.RxRate
			uploadRate = info.TxRate
		} else {
			// Downlink (to users/LAN): TX=Download (data to user), RX=Upload (data from user)
			// From user perspective, needs swap
			downloadRate = info.TxRate
			uploadRate = info.RxRate
		}

		downloadFormatted := FormatRate(downloadRate, l.rateUnit, l.rateScale)
		uploadFormatted := FormatRate(uploadRate, l.rateUnit, l.rateScale)
		log.Printf("interface=%s upload=%s download=%s", info.InterfaceName, uploadFormatted, downloadFormatted)
	}
}

func (l *LogOutput) Close() {
	log.Println("Mikrotik Interface Traffic Monitor stopped")
}
