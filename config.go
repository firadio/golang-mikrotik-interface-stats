package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config holds the Mikrotik connection configuration
type Config struct {
	Host       string
	Port       string
	Username   string
	Password   string
	Interfaces []string // List of interfaces to monitor

	// Display settings
	DisplayMode      string   // "refresh" or "append"
	RateUnit         string   // "auto", "bps", "Bps" (bits or Bytes per second)
	RateScale        string   // "auto", "k", "M", "G" (fixed scale)
	OutputMode       string   // "terminal" or "log"
	Debug            bool     // Enable debug output
	UplinkInterfaces []string // List of uplink interfaces (RX=Upload, TX=Download)
}

// LoadConfig loads configuration from .env file or environment variables
func LoadConfig() (*Config, error) {
	// Try to load from .env file
	if file, err := os.Open(".env"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		envMap := make(map[string]string)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Override with .env values if present
		for k, v := range envMap {
			os.Setenv(k, v)
		}
	}

	host := os.Getenv("MIKROTIK_HOST")
	port := os.Getenv("MIKROTIK_PORT")
	username := os.Getenv("MIKROTIK_USERNAME")
	password := os.Getenv("MIKROTIK_PASSWORD")

	if host == "" || port == "" || username == "" || password == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	// Parse interface list (comma-separated)
	interfacesStr := os.Getenv("INTERFACES")
	if interfacesStr == "" {
		interfacesStr = "vlan2622,vlan2624" // default
	}
	interfaces := strings.Split(interfacesStr, ",")
	for i := range interfaces {
		interfaces[i] = strings.TrimSpace(interfaces[i])
	}

	// Display settings with defaults
	displayMode := os.Getenv("DISPLAY_MODE")
	if displayMode == "" {
		displayMode = "refresh"
	}

	rateUnit := os.Getenv("RATE_UNIT")
	if rateUnit == "" {
		rateUnit = "auto" // auto, bps (bits), Bps (Bytes)
	}

	rateScale := os.Getenv("RATE_SCALE")
	if rateScale == "" {
		rateScale = "auto" // auto, k, M, G
	}

	outputMode := os.Getenv("OUTPUT_MODE")
	if outputMode == "" {
		outputMode = "terminal" // terminal or log
	}

	debug := false
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1" {
		debug = true
	}

	// Parse uplink interface list (comma-separated)
	// Uplink (WAN to ISP): TX=Upload (发到互联网), RX=Download (从互联网收)
	// Downlink (LAN/VLAN to users): TX=Download (发给用户即用户下载), RX=Upload (收用户的即用户上传)
	uplinkInterfacesStr := os.Getenv("UPLINK_INTERFACES")
	var uplinkInterfaces []string
	if uplinkInterfacesStr != "" {
		uplinkInterfaces = strings.Split(uplinkInterfacesStr, ",")
		for i := range uplinkInterfaces {
			uplinkInterfaces[i] = strings.TrimSpace(uplinkInterfaces[i])
		}
	}

	return &Config{
		Host:             host,
		Port:             port,
		Username:         username,
		Password:         password,
		Interfaces:       interfaces,
		DisplayMode:      displayMode,
		RateUnit:         rateUnit,
		RateScale:        rateScale,
		OutputMode:       outputMode,
		Debug:            debug,
		UplinkInterfaces: uplinkInterfaces,
	}, nil
}
