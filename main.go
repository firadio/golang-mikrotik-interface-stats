package main

import (
	"log"
)

// Mikrotik Interface Traffic Monitor
// Monitors Mikrotik router interface traffic and displays real-time statistics
// Supports multiple output modes: terminal (refresh/append), log, and metrics

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
