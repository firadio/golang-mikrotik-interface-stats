package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	// Mikrotik connection settings
	Host     string // Mikrotik router hostname/IP
	Port     string // Mikrotik API port
	Username string // Authentication username
	Password string // Authentication password

	// Monitoring settings
	Interfaces       []string // List of interfaces to monitor
	UplinkInterfaces []string // Uplink interfaces (WAN ports) for RX/TX interpretation
	StatsWindowSize  int      // Statistics window size in seconds (default 10, max 60)

	// Display settings
	DisplayMode string // "refresh" (like top) or "append" (like tail -f)
	OutputMode  string // "terminal" (formatted) or "log" (structured)
	RateUnit    string // "auto", "bps" (bits/s), "Bps" (Bytes/s)
	RateScale   string // "auto", "k", "M", "G" (fixed scale)

	// Debug settings
	Debug bool // Enable debug output (show API commands)
}

// LoadConfig loads configuration from .env file and environment variables
// .env file values are loaded first, then environment variables can override them
func LoadConfig() (*Config, error) {
	// Load .env file if present (optional)
	loadEnvFile(".env")

	// Parse and validate configuration
	config := &Config{}

	// Required connection settings
	config.Host = os.Getenv("MIKROTIK_HOST")
	config.Port = os.Getenv("MIKROTIK_PORT")
	config.Username = os.Getenv("MIKROTIK_USERNAME")
	config.Password = os.Getenv("MIKROTIK_PASSWORD")

	if config.Host == "" || config.Port == "" || config.Username == "" || config.Password == "" {
		return nil, fmt.Errorf("missing required environment variables: MIKROTIK_HOST, MIKROTIK_PORT, MIKROTIK_USERNAME, MIKROTIK_PASSWORD")
	}

	// Monitoring settings
	config.Interfaces = parseCommaSeparated(os.Getenv("INTERFACES"), "vlan2622,vlan2624")
	config.UplinkInterfaces = parseCommaSeparated(os.Getenv("UPLINK_INTERFACES"), "")
	config.StatsWindowSize = parseIntWithDefault(os.Getenv("STATS_WINDOW_SIZE"), 10, 1, 60)

	// Display settings with defaults
	config.DisplayMode = getEnvOrDefault("DISPLAY_MODE", "refresh")
	config.OutputMode = getEnvOrDefault("OUTPUT_MODE", "terminal")
	config.RateUnit = getEnvOrDefault("RATE_UNIT", "auto")
	config.RateScale = getEnvOrDefault("RATE_SCALE", "auto")

	// Debug settings
	config.Debug = os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1"

	return config, nil
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return // File doesn't exist, use environment variables only
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Only set if not already in environment
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(value, defaultValue string) []string {
	if value == "" {
		value = defaultValue
	}
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseIntWithDefault parses an integer with min/max bounds
func parseIntWithDefault(value string, defaultValue, min, max int) int {
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	if intValue < min {
		return min
	}
	if intValue > max {
		return max
	}
	return intValue
}
