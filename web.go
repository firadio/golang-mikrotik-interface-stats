package main

import (
	"log"
	"time"
)

// ============================================================================
// Web Server (stub implementation)
// ============================================================================

// WebServer handles HTTP/WebSocket server for real-time monitoring
type WebServer struct {
	config           *WebConfig
	uplinkInterfaces map[string]bool
}

// NewWebServer creates a new web server
func NewWebServer(config *WebConfig, uplinkInterfaces []string) *WebServer {
	log.Printf("[Web] Web server initialized (addr: %s)", config.ListenAddr)

	// Convert uplink interface list to set
	uplinkSet := make(map[string]bool, len(uplinkInterfaces))
	for _, iface := range uplinkInterfaces {
		uplinkSet[iface] = true
	}

	return &WebServer{
		config:           config,
		uplinkInterfaces: uplinkSet,
	}
}

// Start starts the web server (non-blocking)
func (w *WebServer) Start() error {
	log.Printf("[Web] Starting web server on %s", w.config.ListenAddr)
	// TODO: Implement actual HTTP server with routes:
	// - GET  /              -> index.html
	// - GET  /api/current   -> current stats (JSON)
	// - WS   /api/realtime  -> WebSocket real-time push
	// - GET  /api/history/* -> proxy to VictoriaMetrics
	return nil
}

// Stop stops the web server gracefully
func (w *WebServer) Stop() error {
	log.Println("[Web] Stopping web server")
	// TODO: Implement graceful shutdown
	return nil
}

// BroadcastStats broadcasts statistics to all connected WebSocket clients
func (w *WebServer) BroadcastStats(timestamp time.Time, stats map[string]*RateInfo) {
	// TODO: Implement WebSocket broadcast
	// This should format stats and send to all connected clients
}
