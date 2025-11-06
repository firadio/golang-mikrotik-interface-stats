package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// Helper Functions
// ============================================================================

// getUnitSuffix returns the unit suffix string for display headers
func getUnitSuffix(rateUnit string, rateScale string) string {
	var baseUnit string
	if rateUnit == "bps" {
		baseUnit = "bps"
	} else {
		baseUnit = "B/s"
	}

	switch rateScale {
	case "k":
		return "k" + baseUnit
	case "M":
		return "M" + baseUnit
	case "G":
		return "G" + baseUnit
	case "auto":
		return "auto-" + baseUnit
	default:
		return baseUnit
	}
}

// formatNumeric formats rate as numeric value only (no unit suffix)
// Used for table display where unit is shown in header
func formatNumeric(bytesPerSec float64, rateUnit string, rateScale string) string {
	var value float64

	// Convert to bits or keep as bytes
	if rateUnit == "bps" {
		value = bytesPerSec * 8
	} else {
		value = bytesPerSec
	}

	// Apply scale and format
	switch rateScale {
	case "k":
		return fmt.Sprintf("%.2f", value/1000)
	case "M":
		return fmt.Sprintf("%.2f", value/1000000)
	case "G":
		return fmt.Sprintf("%.2f", value/1000000000)
	case "auto":
		// Auto scale - includes unit suffix in value
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

// ANSI escape code utilities for terminal control

// clearScreen clears the entire terminal screen using ANSI codes
// \033[2J - clear entire screen
// \033[H  - move cursor to home position (1,1)
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// moveCursorHome moves cursor to top-left without clearing screen
// \033[H - move cursor to home position (1,1)
// More efficient than clearScreen, reduces flicker
func moveCursorHome() {
	fmt.Print("\033[H")
}

// ============================================================================
// Output Interface
// ============================================================================

// OutputWriter defines the interface for output implementations
// Allows multiple output formats (terminal, log, metrics, etc.)
type OutputWriter interface {
	WriteHeader()                                          // Initialize output (print headers, etc.)
	WriteStats(timestamp time.Time, stats map[string]*RateInfo) // Write statistics
	Close()                                                // Cleanup resources
}

// RateInfo holds calculated rate information for an interface
// All rates are in bytes/second (RX/TX naming)
// Display layer converts to Upload/Download based on interface type
type RateInfo struct {
	InterfaceName string  // Interface name
	RxRate        float64 // Current RX rate (bytes/s)
	TxRate        float64 // Current TX rate (bytes/s)
	RxAvg         float64 // Average RX rate over stats window
	TxAvg         float64 // Average TX rate over stats window
	RxPeak        float64 // Peak RX rate over stats window
	TxPeak        float64 // Peak TX rate over stats window
}

// ============================================================================
// Terminal Output (refresh/append modes)
// ============================================================================

// TerminalOutput implements OutputWriter for terminal display
type TerminalOutput struct {
	refreshMode      bool            // true = refresh mode (like top), false = append mode (like tail -f)
	rateUnit         string          // "bps" or "Bps"
	rateScale        string          // "auto", "k", "M", "G"
	uplinkInterfaces map[string]bool // Set of uplink interface names for RX/TX swapping
	statsWindowSize  int             // Statistics window size in seconds
}

// NewTerminalOutput creates a new terminal output handler
func NewTerminalOutput(refreshMode bool, rateUnit, rateScale string, uplinkInterfaces []string, statsWindowSize int) *TerminalOutput {
	// Convert uplink interface list to set for O(1) lookup
	uplinkSet := make(map[string]bool, len(uplinkInterfaces))
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

			// Convert RX/TX to Upload/Download based on interface type
			//
			// Uplink (WAN to ISP):
			//   - TX = Upload to internet
			//   - RX = Download from internet
			//   - No swap needed (matches user expectation)
			//
			// Downlink (LAN/VLAN to users):
			//   - TX = Download (router sends to user)
			//   - RX = Upload (router receives from user)
			//   - Swap needed for user perspective
			if t.uplinkInterfaces[name] {
				// Uplink: no swap
				uploadRate = info.TxRate
				downloadRate = info.RxRate
				uploadAvg = info.TxAvg
				downloadAvg = info.RxAvg
				uploadPeak = info.TxPeak
				downloadPeak = info.RxPeak
			} else {
				// Downlink: swap TX/RX
				uploadRate = info.RxRate
				downloadRate = info.TxRate
				uploadAvg = info.RxAvg
				downloadAvg = info.TxAvg
				uploadPeak = info.RxPeak
				downloadPeak = info.TxPeak
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

// ============================================================================
// Log Output (for services/daemons)
// ============================================================================

// LogOutput implements OutputWriter for structured logging
// Suitable for running as a service or daemon
type LogOutput struct {
	rateUnit         string          // "bps" or "Bps"
	rateScale        string          // "auto", "k", "M", "G"
	uplinkInterfaces map[string]bool // Set of uplink interface names for RX/TX swapping
	statsWindowSize  int             // Statistics window size (unused in log mode)
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

// ============================================================================
// Structured Logger (for LOG_ENABLED mode)
// ============================================================================

// StructuredLogger implements structured logging output
// Suitable for running as a service with JSON or text format
type StructuredLogger struct {
	config           *LogConfig
	uplinkInterfaces map[string]bool
	writer           *log.Logger
	file             *os.File // Only used if Output="file"
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *LogConfig, uplinkInterfaces []string) *StructuredLogger {
	// Convert uplink interface list to set for O(1) lookup
	uplinkSet := make(map[string]bool, len(uplinkInterfaces))
	for _, iface := range uplinkInterfaces {
		uplinkSet[iface] = true
	}

	logger := &StructuredLogger{
		config:           config,
		uplinkInterfaces: uplinkSet,
	}

	// Setup output writer
	if config.Output == "file" {
		// Open log file with append mode
		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file %s: %v", config.File, err)
		}
		logger.file = file
		logger.writer = log.New(file, "", 0) // No prefix, we'll format ourselves
	} else {
		// Use stdout
		logger.writer = log.New(os.Stdout, "", 0)
	}

	return logger
}

// WriteHeader initializes logging
func (s *StructuredLogger) WriteHeader() {
	if s.config.Format == "json" {
		s.writer.Printf(`{"level":"info","msg":"Mikrotik Interface Traffic Monitor started"}`)
	} else {
		s.writer.Printf("%s [INFO] Mikrotik Interface Traffic Monitor started", time.Now().Format(time.RFC3339))
	}
}

// WriteStats writes statistics in structured format
func (s *StructuredLogger) WriteStats(timestamp time.Time, stats map[string]*RateInfo) {
	// Sort interface names for consistent ordering
	names := make([]string, 0, len(stats))
	for name := range stats {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		info := stats[name]
		var downloadRate, uploadRate float64

		// Convert RX/TX to Upload/Download based on interface type
		if s.uplinkInterfaces[name] {
			// Uplink: no swap
			uploadRate = info.TxRate
			downloadRate = info.RxRate
		} else {
			// Downlink: swap TX/RX
			uploadRate = info.RxRate
			downloadRate = info.TxRate
		}

		// Format based on configured format
		if s.config.Format == "json" {
			s.writeJSON(timestamp, info.InterfaceName, uploadRate, downloadRate)
		} else {
			s.writeText(timestamp, info.InterfaceName, uploadRate, downloadRate)
		}
	}
}

// writeJSON writes a JSON log entry
func (s *StructuredLogger) writeJSON(timestamp time.Time, iface string, uploadRate, downloadRate float64) {
	// Format rates
	uploadFormatted := FormatRate(uploadRate, s.config.RateUnit, s.config.RateScale)
	downloadFormatted := FormatRate(downloadRate, s.config.RateUnit, s.config.RateScale)

	// Write JSON (single line)
	s.writer.Printf(`{"time":"%s","interface":"%s","upload":"%s","download":"%s","upload_bps":%.0f,"download_bps":%.0f}`,
		timestamp.Format(time.RFC3339),
		iface,
		strings.TrimSpace(uploadFormatted),
		strings.TrimSpace(downloadFormatted),
		uploadRate*8,   // Convert to bits for numeric field
		downloadRate*8,
	)
}

// writeText writes a text log entry
func (s *StructuredLogger) writeText(timestamp time.Time, iface string, uploadRate, downloadRate float64) {
	// Format rates
	uploadFormatted := FormatRate(uploadRate, s.config.RateUnit, s.config.RateScale)
	downloadFormatted := FormatRate(downloadRate, s.config.RateUnit, s.config.RateScale)

	// Write text format
	s.writer.Printf("%s interface=%s upload=%s download=%s",
		timestamp.Format(time.RFC3339),
		iface,
		strings.TrimSpace(uploadFormatted),
		strings.TrimSpace(downloadFormatted),
	)
}

// Close closes the logger
func (s *StructuredLogger) Close() {
	if s.config.Format == "json" {
		s.writer.Printf(`{"level":"info","msg":"Mikrotik Interface Traffic Monitor stopped"}`)
	} else {
		s.writer.Printf("%s [INFO] Mikrotik Interface Traffic Monitor stopped", time.Now().Format(time.RFC3339))
	}

	// Close file if opened
	if s.file != nil {
		s.file.Close()
	}
}
