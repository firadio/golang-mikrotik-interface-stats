package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// getUnitSuffix returns the unit suffix for display
func getUnitSuffix(rateUnit string, rateScale string) string {
	var unit string
	if rateUnit == "bps" {
		unit = "bps"
	} else {
		unit = "B/s"
	}

	switch rateScale {
	case "k":
		return "k" + unit
	case "M":
		return "M" + unit
	case "G":
		return "G" + unit
	case "auto":
		return "auto-" + unit
	default:
		return unit
	}
}

// formatNumeric formats rate as numeric value only (no unit)
func formatNumeric(bytesPerSec float64, rateUnit string, rateScale string) string {
	var value float64

	// Convert to bits or keep as bytes
	if rateUnit == "bps" {
		value = bytesPerSec * 8
	} else {
		value = bytesPerSec
	}

	// Apply scale
	switch rateScale {
	case "k":
		value = value / 1000
		return fmt.Sprintf("%.2f", value)
	case "M":
		value = value / 1000000
		return fmt.Sprintf("%.2f", value)
	case "G":
		value = value / 1000000000
		return fmt.Sprintf("%.2f", value)
	case "auto":
		// Auto scale - pick appropriate unit
		if value < 1000 {
			return fmt.Sprintf("%.2f", value)
		} else if value < 1000000 {
			return fmt.Sprintf("%.2fk", value/1000)
		} else if value < 1000000000 {
			return fmt.Sprintf("%.2fM", value/1000000)
		} else {
			return fmt.Sprintf("%.2fG", value/1000000000)
		}
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

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
	RxAvg         float64 // Average RX rate (over stats window)
	TxAvg         float64 // Average TX rate (over stats window)
	RxPeak        float64 // Peak RX rate (over stats window)
	TxPeak        float64 // Peak TX rate (over stats window)
}

// TerminalOutput handles terminal output (refresh or append mode)
type TerminalOutput struct {
	refreshMode      bool
	rateUnit         string
	rateScale        string
	uplinkInterfaces map[string]bool // Set of uplink interface names
	statsWindowSize  int             // Statistics window size in seconds
}

// NewTerminalOutput creates a new terminal output handler
func NewTerminalOutput(refreshMode bool, rateUnit, rateScale string, uplinkInterfaces []string, statsWindowSize int) *TerminalOutput {
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
		statsWindowSize:  statsWindowSize,
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

		// Display Time, Unit and Window size on one line
		unitSuffix := getUnitSuffix(t.rateUnit, t.rateScale)
		fmt.Printf("Time: %s | Unit: %s | Window: %ds\n", timeStr, unitSuffix, t.statsWindowSize)

		fmt.Println(strings.Repeat("-", 80))
		// Header: 10+10+10+10+10+10+10 = 70 chars (留10字符余量)
		// Fixed column headers
		fmt.Printf("%-10s %10s %10s %10s %10s %10s %10s\n",
			"Interface", "Up", "Down", "UpAvg", "DnAvg", "UpPeak", "DnPeak")
		fmt.Println(strings.Repeat("-", 80))

		for _, name := range names {
			info := stats[name]
			var downloadRate, uploadRate, uploadAvg, downloadAvg, uploadPeak, downloadPeak float64

			// Check if this is an uplink interface
			if t.uplinkInterfaces[name] {
				// Uplink (WAN to ISP): TX=Upload (to internet), RX=Download (from internet)
				// This is the "normal" understanding, no swap needed
				downloadRate = info.RxRate
				uploadRate = info.TxRate
				downloadAvg = info.RxAvg
				uploadAvg = info.TxAvg
				downloadPeak = info.RxPeak
				uploadPeak = info.TxPeak
			} else {
				// Downlink (to users/LAN): TX=Download (data to user), RX=Upload (data from user)
				// From user perspective, needs swap
				downloadRate = info.TxRate
				uploadRate = info.RxRate
				downloadAvg = info.TxAvg
				uploadAvg = info.RxAvg
				downloadPeak = info.TxPeak
				uploadPeak = info.RxPeak
			}

			// Format rates as numeric values only (no unit suffix)
			uploadStr := formatNumeric(uploadRate, t.rateUnit, t.rateScale)
			downloadStr := formatNumeric(downloadRate, t.rateUnit, t.rateScale)
			uploadAvgStr := formatNumeric(uploadAvg, t.rateUnit, t.rateScale)
			downloadAvgStr := formatNumeric(downloadAvg, t.rateUnit, t.rateScale)
			uploadPeakStr := formatNumeric(uploadPeak, t.rateUnit, t.rateScale)
			downloadPeakStr := formatNumeric(downloadPeak, t.rateUnit, t.rateScale)

			// Truncate interface name if needed
			ifName := info.InterfaceName
			if len(ifName) > 10 {
				ifName = ifName[:10]
			}

			// Left-align interface name, right-align all numeric values
			fmt.Printf("%-10s %10s %10s %10s %10s %10s %10s\n",
				ifName, uploadStr, downloadStr, uploadAvgStr, downloadAvgStr, uploadPeakStr, downloadPeakStr)
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
	statsWindowSize  int             // Statistics window size in seconds (unused in log mode)
}

// NewLogOutput creates a new log output handler
func NewLogOutput(rateUnit, rateScale string, uplinkInterfaces []string, statsWindowSize int) *LogOutput {
	// Build uplink interface set for fast lookup
	uplinkSet := make(map[string]bool)
	for _, iface := range uplinkInterfaces {
		uplinkSet[iface] = true
	}

	return &LogOutput{
		rateUnit:         rateUnit,
		rateScale:        rateScale,
		uplinkInterfaces: uplinkSet,
		statsWindowSize:  statsWindowSize,
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
