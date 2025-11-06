package main

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

// InterfaceStats represents raw interface traffic counters from Mikrotik
type InterfaceStats struct {
	Name   string // Interface name (e.g., vlan2622, ether1)
	RxByte uint64 // Total received bytes
	TxByte uint64 // Total transmitted bytes
}

// InterfaceRate maintains rate calculation state for an interface
// Uses a ring buffer to track historical rates for statistics
type InterfaceRate struct {
	Name       string    // Interface name
	LastRxByte uint64    // Previous RX counter value
	LastTxByte uint64    // Previous TX counter value
	LastTime   time.Time // Timestamp of last update

	// Ring buffer for historical rates (bytes/second)
	TxHistory    []float64 // TX rate history
	RxHistory    []float64 // RX rate history
	HistoryIndex int       // Current position in ring buffer
	HistoryCount int       // Number of valid entries (0 to window size)
}

// GetInterfaceStats queries the Mikrotik router for interface statistics
// Returns raw byte counters for specified interfaces
func (c *MikrotikClient) GetInterfaceStats(interfaces []string, debug bool) ([]InterfaceStats, error) {
	// Build Mikrotik API command with server-side filtering
	// This reduces network traffic by filtering on the router
	//
	// Command structure:
	//   /interface/print       - Query interface data
	//   =stats                 - Get real-time statistics (live counters)
	//   =.proplist=...         - Only return specified properties
	//   ?name=iface1           - Filter by interface name
	//   ?name=iface2 ?#|       - OR operator (placed after each condition from 2nd onwards)
	cmd := []string{
		"/interface/print",
		"=stats",
		"=.proplist=name,rx-byte,tx-byte",
	}

	// Add interface filters with OR operators
	// Pattern: ?name=iface1 ?name=iface2 ?#| ?name=iface3 ?#|
	for i, iface := range interfaces {
		cmd = append(cmd, "?name="+iface)
		if i >= 1 {
			cmd = append(cmd, "?#|") // OR operator after each interface from 2nd onwards
		}
	}

	if debug {
		log.Printf("DEBUG: Mikrotik API command: %v", cmd)
	}

	// Send command and read response
	if err := c.sendCommand(cmd...); err != nil {
		return nil, fmt.Errorf("sendCommand failed: %w", err)
	}

	responses, err := c.readResponse()
	if err != nil {
		return nil, fmt.Errorf("readResponse failed: %w", err)
	}

	// Parse responses into InterfaceStats
	stats := make([]InterfaceStats, 0, len(responses))
	for _, resp := range responses {
		name := resp["name"]
		if name == "" {
			continue
		}

		rxByte, err := strconv.ParseUint(resp["rx-byte"], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rx-byte for %s: %w", name, err)
		}

		txByte, err := strconv.ParseUint(resp["tx-byte"], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tx-byte for %s: %w", name, err)
		}

		stats = append(stats, InterfaceStats{
			Name:   name,
			RxByte: rxByte,
			TxByte: txByte,
		})
	}

	return stats, nil
}

// FormatBytes converts bytes to human-readable format with auto-scaling (1024-based)
// Deprecated: Use FormatRate with appropriate parameters instead
func FormatBytes(bytes float64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%.2f B/s", bytes)
	}
	div, exp := float64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB/s", bytes/div, "KMGTPE"[exp])
}

// FormatRate formats traffic rate with unit suffix (for append/log modes)
// Converts bytes/sec to configured unit and scale, returns formatted string with unit
func FormatRate(bytesPerSec float64, rateUnit string, rateScale string) string {
	var value float64
	var unit string

	// Convert to bits or keep as bytes
	if rateUnit == "bps" {
		value = bytesPerSec * 8
		unit = "bps"
	} else {
		value = bytesPerSec
		unit = "B/s"
	}

	// Apply scale and format
	switch rateScale {
	case "k":
		return fmt.Sprintf("%7.2f k%s", value/1000, unit)
	case "M":
		return fmt.Sprintf("%7.2f M%s", value/1000000, unit)
	case "G":
		return fmt.Sprintf("%7.2f G%s", value/1000000000, unit)
	case "auto":
		// Auto scale based on value magnitude
		if value < 1000 {
			return fmt.Sprintf("%7.2f %s", value, unit)
		} else if value < 1000000 {
			return fmt.Sprintf("%7.2f k%s", value/1000, unit)
		} else if value < 1000000000 {
			return fmt.Sprintf("%7.2f M%s", value/1000000, unit)
		} else {
			return fmt.Sprintf("%7.2f G%s", value/1000000000, unit)
		}
	default:
		return fmt.Sprintf("%.2f %s", value, unit)
	}
}
