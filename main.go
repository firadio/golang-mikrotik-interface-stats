package main

import (
	"log"
)

func main() {
	// Enable ANSI escape sequences on Windows
	if err := enableANSI(); err != nil {
		log.Printf("Warning: Failed to enable ANSI support: %v", err)
	}

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Mikrotik
	client, err := NewMikrotikClient(config)
	if err != nil {
		log.Fatalf("Failed to connect to Mikrotik: %v", err)
	}
	defer client.Close()

	log.Printf("Connected to Mikrotik at %s:%s", config.Host, config.Port)

	// Start monitoring
	monitor := NewMonitor(client, config)
	if err := monitor.Start(); err != nil {
		log.Fatalf("Monitor error: %v", err)
	}
}
