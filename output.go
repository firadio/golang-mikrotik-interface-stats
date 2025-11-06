package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// clearScreen clears the terminal screen using ANSI escape codes
func clearScreen() {
	// Use ANSI escape codes to clear screen
	// This works on:
	// - Linux/macOS terminals
	// - Windows 10+ with Virtual Terminal Processing enabled
	// - Windows Terminal, PowerShell 7+, Git Bash, etc.
	//
	// \033[2J - clear entire screen
	// \033[H - move cursor to home position (1,1)
	fmt.Print("\033[2J\033[H")
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
	refreshMode bool
	rateUnit    string
	rateScale   string
}

// NewTerminalOutput creates a new terminal output handler
func NewTerminalOutput(refreshMode bool, rateUnit, rateScale string) *TerminalOutput {
	return &TerminalOutput{
		refreshMode: refreshMode,
		rateUnit:    rateUnit,
		rateScale:   rateScale,
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

	if t.refreshMode {
		// Refresh mode: clear and redraw
		clearScreen()
		fmt.Println("Mikrotik Interface Traffic Monitor")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("Time: %s\n", timeStr)
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("%-15s %-20s %-20s\n", "Interface", "RX Rate", "TX Rate")
		fmt.Println(strings.Repeat("-", 80))

		for _, info := range stats {
			rxFormatted := FormatRate(info.RxRate, t.rateUnit, t.rateScale)
			txFormatted := FormatRate(info.TxRate, t.rateUnit, t.rateScale)
			fmt.Printf("%-15s %-20s %-20s\n", info.InterfaceName, rxFormatted, txFormatted)
		}

		fmt.Println(strings.Repeat("-", 80))
		fmt.Println("Press Ctrl+C to stop")
	} else {
		// Append mode: add new lines
		for _, info := range stats {
			rxFormatted := FormatRate(info.RxRate, t.rateUnit, t.rateScale)
			txFormatted := FormatRate(info.TxRate, t.rateUnit, t.rateScale)
			fmt.Printf("[%s] %s: RX: %s  TX: %s\n",
				timeStr, info.InterfaceName, rxFormatted, txFormatted)
		}
	}
}

func (t *TerminalOutput) Close() {
	// Nothing to close for terminal output
}

// LogOutput handles log-style output
type LogOutput struct {
	rateUnit  string
	rateScale string
}

// NewLogOutput creates a new log output handler
func NewLogOutput(rateUnit, rateScale string) *LogOutput {
	return &LogOutput{
		rateUnit:  rateUnit,
		rateScale: rateScale,
	}
}

func (l *LogOutput) WriteHeader() {
	log.Println("Mikrotik Interface Traffic Monitor started")
}

func (l *LogOutput) WriteStats(timestamp time.Time, stats map[string]*RateInfo) {
	for _, info := range stats {
		rxFormatted := FormatRate(info.RxRate, l.rateUnit, l.rateScale)
		txFormatted := FormatRate(info.TxRate, l.rateUnit, l.rateScale)
		log.Printf("interface=%s rx=%s tx=%s", info.InterfaceName, rxFormatted, txFormatted)
	}
}

func (l *LogOutput) Close() {
	log.Println("Mikrotik Interface Traffic Monitor stopped")
}
