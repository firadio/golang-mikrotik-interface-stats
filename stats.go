package main

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

// InterfaceStats represents interface traffic statistics
type InterfaceStats struct {
	Name   string
	RxByte uint64
	TxByte uint64
}

// InterfaceRate stores previous statistics for rate calculation
type InterfaceRate struct {
	Name       string
	LastRxByte uint64
	LastTxByte uint64
	LastTime   time.Time
}

// GetInterfaceStats queries the Mikrotik router for interface statistics
func (c *MikrotikClient) GetInterfaceStats(interfaces []string, debug bool) ([]InterfaceStats, error) {
	// Query with server-side filtering using Mikrotik API query syntax
	// =stats              : get real-time statistics (live counters)
	// =.proplist=         : only return specified properties (name, rx-byte, tx-byte)
	// ?name=              : filter where name equals the value
	// ?#|                 : OR operator (matches if any condition is true)
	// This sends only the filtered results from Mikrotik, reducing network traffic

	// Build command with dynamic interface list
	cmd := []string{
		"/interface/print",
		"=stats",
		"=.proplist=name,rx-byte,tx-byte",
	}

	// Add filter for each interface
	// Mikrotik API OR syntax: ?name=iface1 ?name=iface2 ?#| ?name=iface3 ?#|
	// The OR operator ?#| comes AFTER each condition (starting from the second)
	for i, iface := range interfaces {
		cmd = append(cmd, "?name="+iface)
		// Add OR operator after each interface starting from the second one
		if i >= 1 {
			cmd = append(cmd, "?#|")
		}
	}

	// Debug: print command for troubleshooting (if enabled)
	if debug {
		log.Printf("DEBUG: Mikrotik API command: %v", cmd)
	}

	err := c.sendCommand(cmd...)
	if err != nil {
		return nil, fmt.Errorf("sendCommand failed: %w", err)
	}

	responses, err := c.readResponse()
	if err != nil {
		return nil, fmt.Errorf("readResponse failed: %w", err)
	}

	var stats []InterfaceStats
	for _, resp := range responses {
		name := resp["name"]
		rxByteStr := resp["rx-byte"]
		txByteStr := resp["tx-byte"]

		if name == "" {
			continue
		}

		rxByte, err := strconv.ParseUint(rxByteStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rx-byte for %s: %w", name, err)
		}

		txByte, err := strconv.ParseUint(txByteStr, 10, 64)
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

// FormatBytes converts bytes to human-readable format (auto scale)
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

// FormatRate formats rate according to configuration
func FormatRate(bytesPerSec float64, rateUnit string, rateScale string) string {
	var value float64
	var unit string

	// Convert to bits or keep as bytes
	if rateUnit == "bps" {
		value = bytesPerSec * 8 // Convert to bits
		unit = "bps"
	} else {
		value = bytesPerSec
		unit = "B/s"
	}

	// Apply scale
	switch rateScale {
	case "k":
		value = value / 1000
		return fmt.Sprintf("%7.2f %c%s", value, 'k', unit)
	case "M":
		value = value / 1000000
		return fmt.Sprintf("%7.2f %c%s", value, 'M', unit)
	case "G":
		value = value / 1000000000
		return fmt.Sprintf("%7.2f %c%s", value, 'G', unit)
	case "auto":
		// Auto scale
		if value < 1000 {
			return fmt.Sprintf("%7.2f %s", value, unit)
		} else if value < 1000000 {
			return fmt.Sprintf("%7.2f %c%s", value/1000, 'k', unit)
		} else if value < 1000000000 {
			return fmt.Sprintf("%7.2f %c%s", value/1000000, 'M', unit)
		} else {
			return fmt.Sprintf("%7.2f %c%s", value/1000000000, 'G', unit)
		}
	default:
		return fmt.Sprintf("%.2f %s", value, unit)
	}
}
