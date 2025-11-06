package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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
	Debug            bool     // Enable debug output (show API commands)

	// Optional output features (nil if disabled)
	Terminal        *TerminalConfig // Terminal interactive display
	Log             *LogConfig      // Structured logging
	Web             *WebConfig      // Web service
	VictoriaMetrics *VMConfig       // VictoriaMetrics integration
}

// TerminalConfig holds terminal output configuration
type TerminalConfig struct {
	Enabled   bool   // Enable terminal output
	Mode      string // "refresh" (like top) or "append" (like tail -f)
	RateUnit  string // "auto", "bps", "Bps"
	RateScale string // "auto", "k", "M", "G"
}

// LogConfig holds structured logging configuration
type LogConfig struct {
	Enabled   bool   // Enable structured logging
	Output    string // "stdout" or "file"
	File      string // File path if Output="file"
	Format    string // "json" or "text"
	RateUnit  string // "auto", "bps", "Bps"
	RateScale string // "auto", "k", "M", "G"
}

// WebConfig holds web service configuration
type WebConfig struct {
	Enabled        bool   // Enable web service
	ListenAddr     string // Listen address (e.g., ":8080")
	EnableRealtime bool   // Enable WebSocket real-time push
	EnableAPI      bool   // Enable REST API
	EnableStatic   bool   // Enable static file serving
}

// VMConfig holds VictoriaMetrics configuration
type VMConfig struct {
	Enabled       bool          // Enable VictoriaMetrics integration
	URL           string        // VictoriaMetrics endpoint
	ShortInterval time.Duration // Short-term aggregation interval (e.g., 10s)
	LongInterval  time.Duration // Long-term aggregation interval (e.g., 5m)
	EnableShort   bool          // Enable short-term aggregation
	EnableLong    bool          // Enable long-term aggregation
	Timeout       time.Duration // HTTP request timeout
	RetryCount    int           // Number of retries on failure
}

// LoadConfig loads configuration from .env file and environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if present (optional)
	loadEnvFile(".env")

	// Parse and validate configuration
	config := &Config{}

	// Load core settings
	if err := loadCoreConfig(config); err != nil {
		return nil, err
	}

	// Load optional features
	loadTerminalConfig(config)
	loadLogConfig(config)
	loadWebConfig(config)
	loadVMConfig(config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// loadCoreConfig loads required core configuration
func loadCoreConfig(config *Config) error {
	config.Host = os.Getenv("MIKROTIK_HOST")
	config.Port = os.Getenv("MIKROTIK_PORT")
	config.Username = os.Getenv("MIKROTIK_USERNAME")
	config.Password = os.Getenv("MIKROTIK_PASSWORD")

	if config.Host == "" || config.Port == "" || config.Username == "" || config.Password == "" {
		return fmt.Errorf("missing required environment variables: MIKROTIK_HOST, MIKROTIK_PORT, MIKROTIK_USERNAME, MIKROTIK_PASSWORD")
	}

	config.Interfaces = parseCommaSeparated(os.Getenv("INTERFACES"), "vlan2622,vlan2624")
	config.UplinkInterfaces = parseCommaSeparated(os.Getenv("UPLINK_INTERFACES"), "")
	config.StatsWindowSize = parseIntWithDefault(os.Getenv("STATS_WINDOW_SIZE"), 10, 1, 60)
	config.Debug = parseBool(os.Getenv("DEBUG"), false)

	return nil
}

// loadTerminalConfig loads terminal output configuration
func loadTerminalConfig(config *Config) {
	enabled := parseBool(os.Getenv("TERMINAL_ENABLED"), false)
	if !enabled {
		config.Terminal = nil
		return
	}

	config.Terminal = &TerminalConfig{
		Enabled:   true,
		Mode:      getEnvOrDefault("TERMINAL_MODE", "refresh"),
		RateUnit:  getEnvOrDefault("TERMINAL_RATE_UNIT", "auto"),
		RateScale: getEnvOrDefault("TERMINAL_RATE_SCALE", "auto"),
	}
}

// loadLogConfig loads structured logging configuration
func loadLogConfig(config *Config) {
	enabled := parseBool(os.Getenv("LOG_ENABLED"), false)
	if !enabled {
		config.Log = nil
		return
	}

	config.Log = &LogConfig{
		Enabled:   true,
		Output:    getEnvOrDefault("LOG_OUTPUT", "stdout"),
		File:      getEnvOrDefault("LOG_FILE", "/var/log/mikrotik-stats.log"),
		Format:    getEnvOrDefault("LOG_FORMAT", "text"),
		RateUnit:  getEnvOrDefault("LOG_RATE_UNIT", "auto"),
		RateScale: getEnvOrDefault("LOG_RATE_SCALE", "auto"),
	}
}

// loadWebConfig loads web service configuration
func loadWebConfig(config *Config) {
	enabled := parseBool(os.Getenv("WEB_ENABLED"), false)
	if !enabled {
		config.Web = nil
		return
	}

	config.Web = &WebConfig{
		Enabled:        true,
		ListenAddr:     getEnvOrDefault("WEB_LISTEN_ADDR", ":8080"),
		EnableRealtime: parseBool(os.Getenv("WEB_ENABLE_REALTIME"), true),
		EnableAPI:      parseBool(os.Getenv("WEB_ENABLE_API"), true),
		EnableStatic:   parseBool(os.Getenv("WEB_ENABLE_STATIC"), true),
	}
}

// loadVMConfig loads VictoriaMetrics configuration
func loadVMConfig(config *Config) {
	enabled := parseBool(os.Getenv("VM_ENABLED"), false)
	if !enabled {
		config.VictoriaMetrics = nil
		return
	}

	config.VictoriaMetrics = &VMConfig{
		Enabled:       true,
		URL:           getEnvOrDefault("VM_URL", "http://localhost:8428"),
		ShortInterval: parseDuration(os.Getenv("VM_SHORT_INTERVAL"), 10*time.Second),
		LongInterval:  parseDuration(os.Getenv("VM_LONG_INTERVAL"), 5*time.Minute),
		EnableShort:   parseBool(os.Getenv("VM_ENABLE_SHORT"), true),
		EnableLong:    parseBool(os.Getenv("VM_ENABLE_LONG"), true),
		Timeout:       parseDuration(os.Getenv("VM_TIMEOUT"), 5*time.Second),
		RetryCount:    parseIntWithDefault(os.Getenv("VM_RETRY_COUNT"), 3, 0, 10),
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Check for output conflicts: Terminal + Log(stdout) will cause display issues
	if c.Terminal != nil && c.Terminal.Enabled && c.Log != nil && c.Log.Enabled && c.Log.Output == "stdout" {
		return fmt.Errorf("TERMINAL_ENABLED and LOG_ENABLED with LOG_OUTPUT=stdout cannot both be true (output conflict)")
	}

	// Validate terminal config
	if c.Terminal != nil {
		if c.Terminal.Mode != "refresh" && c.Terminal.Mode != "append" {
			return fmt.Errorf("invalid TERMINAL_MODE: %s (must be 'refresh' or 'append')", c.Terminal.Mode)
		}
	}

	// Validate log config
	if c.Log != nil {
		if c.Log.Output != "stdout" && c.Log.Output != "file" {
			return fmt.Errorf("invalid LOG_OUTPUT: %s (must be 'stdout' or 'file')", c.Log.Output)
		}
		if c.Log.Output == "file" && c.Log.File == "" {
			return fmt.Errorf("LOG_FILE must be specified when LOG_OUTPUT=file")
		}
		if c.Log.Format != "json" && c.Log.Format != "text" {
			return fmt.Errorf("invalid LOG_FORMAT: %s (must be 'json' or 'text')", c.Log.Format)
		}
	}

	// Validate web config
	if c.Web != nil {
		// At least one web feature must be enabled
		if !c.Web.EnableRealtime && !c.Web.EnableAPI && !c.Web.EnableStatic {
			return fmt.Errorf("at least one web feature must be enabled (WEB_ENABLE_REALTIME, WEB_ENABLE_API, or WEB_ENABLE_STATIC)")
		}
	}

	// Validate VM config
	if c.VictoriaMetrics != nil {
		if c.VictoriaMetrics.URL == "" {
			return fmt.Errorf("VM_URL must be specified when VM_ENABLED=true")
		}
		if c.VictoriaMetrics.ShortInterval < 1*time.Second {
			return fmt.Errorf("VM_SHORT_INTERVAL must be at least 1 second")
		}
		if c.VictoriaMetrics.LongInterval < c.VictoriaMetrics.ShortInterval {
			return fmt.Errorf("VM_LONG_INTERVAL must be >= VM_SHORT_INTERVAL")
		}
	}

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

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

// parseBool parses a boolean value
func parseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1"
}

// parseDuration parses a duration value
func parseDuration(value string, defaultValue time.Duration) time.Duration {
	if value == "" {
		return defaultValue
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as duration string (e.g., "10s", "5m")
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}

	return defaultValue
}
