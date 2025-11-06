package main

import (
	"fmt"
	"log"
	"strings"
)

const (
	Version = "v0.0.1"
)

// Mikrotik Interface Traffic Monitor
// Monitors Mikrotik router interface traffic and displays real-time statistics
// Supports multiple output modes: terminal, structured logging, web UI, and VictoriaMetrics

func main() {
	// Enable ANSI escape sequences on Windows for color/cursor control
	if err := enableANSI(); err != nil {
		log.Printf("Warning: Failed to enable ANSI support: %v", err)
	}

	// Load configuration from .env file and environment variables
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Print startup information
	printStartupInfo(config)

	// Establish connection to Mikrotik router via API
	client, err := NewMikrotikClient(config)
	if err != nil {
		log.Fatalf("Failed to connect to Mikrotik: %v", err)
	}
	defer client.Close()

	log.Printf("Connected to Mikrotik at %s:%s", config.Host, config.Port)

	// Create and start monitoring loop
	monitor := NewMonitor(client, config)
	if err := monitor.Start(); err != nil {
		log.Fatalf("Monitor error: %v", err)
	}
}

// printStartupInfo prints application startup information
func printStartupInfo(config *Config) {
	log.Println("========================================")
	log.Printf("Mikrotik Interface Traffic Monitor %s", Version)
	log.Println("========================================")
	log.Printf("Monitoring %d interface(s): %s", len(config.Interfaces), strings.Join(config.Interfaces, ", "))

	// Print enabled features
	var features []string

	if config.Terminal != nil {
		features = append(features, fmt.Sprintf("Terminal (%s mode)", config.Terminal.Mode))
	}

	if config.Log != nil {
		features = append(features, fmt.Sprintf("Structured Log (%s → %s)", config.Log.Format, config.Log.Output))
	}

	if config.Web != nil {
		webFeatures := []string{}
		if config.Web.EnableRealtime {
			webFeatures = append(webFeatures, "realtime")
		}
		if config.Web.EnableAPI {
			webFeatures = append(webFeatures, "api")
		}
		if config.Web.EnableStatic {
			webFeatures = append(webFeatures, "static")
		}
		features = append(features, fmt.Sprintf("Web (%s on %s)",
			strings.Join(webFeatures, "+"), config.Web.ListenAddr))
	}

	if config.VictoriaMetrics != nil {
		vmFeatures := []string{}
		if config.VictoriaMetrics.EnableShort {
			vmFeatures = append(vmFeatures, fmt.Sprintf("short:%v", config.VictoriaMetrics.ShortInterval))
		}
		if config.VictoriaMetrics.EnableLong {
			vmFeatures = append(vmFeatures, fmt.Sprintf("long:%v", config.VictoriaMetrics.LongInterval))
		}
		features = append(features, fmt.Sprintf("VictoriaMetrics (%s)", strings.Join(vmFeatures, "+")))
	}

	if len(features) == 0 {
		log.Println("Enabled Features: None (running in silent mode)")
		log.Println("")
		log.Println("💡 Tip: Enable features via environment variables:")
		log.Println("  - TERMINAL_ENABLED=true  (interactive display)")
		log.Println("  - LOG_ENABLED=true       (structured logging)")
		log.Println("  - WEB_ENABLED=true       (web interface)")
		log.Println("  - VM_ENABLED=true        (metrics storage)")
		log.Println("")
	} else {
		log.Printf("Enabled Features: %s", strings.Join(features, ", "))
	}

	log.Println("========================================")
}
