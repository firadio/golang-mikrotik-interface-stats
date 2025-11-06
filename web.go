package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================================
// Web Server Implementation
// ============================================================================

// WebServer handles HTTP/WebSocket server for real-time monitoring
type WebServer struct {
	config           *WebConfig
	uplinkInterfaces map[string]bool
	server           *http.Server

	// WebSocket client management
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader

	// Latest stats cache
	latestStats   map[string]*RateInfo
	latestTime    time.Time
	latestStatsMu sync.RWMutex
}

// NewWebServer creates a new web server
func NewWebServer(config *WebConfig, uplinkInterfaces []string) *WebServer {
	log.Printf("[Web] Web server initialized (addr: %s)", config.ListenAddr)

	// Convert uplink interface list to set
	uplinkSet := make(map[string]bool, len(uplinkInterfaces))
	for _, iface := range uplinkInterfaces {
		uplinkSet[iface] = true
	}

	ws := &WebServer{
		config:           config,
		uplinkInterfaces: uplinkSet,
		clients:          make(map[*websocket.Conn]bool),
		latestStats:      make(map[string]*RateInfo),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}

	// Setup HTTP server
	mux := http.NewServeMux()

	// Register routes based on enabled features
	if config.EnableStatic {
		// Serve static files from web/ directory
		fs := http.FileServer(http.Dir("web"))
		mux.Handle("/", fs)
		mux.Handle("/static/", fs)
	}

	if config.EnableAPI {
		mux.HandleFunc("/api/current", ws.handleCurrentStats)
	}

	if config.EnableRealtime {
		mux.HandleFunc("/api/realtime", ws.handleWebSocket)
	}

	ws.server = &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}

	return ws
}

// Start starts the web server (non-blocking)
func (w *WebServer) Start() error {
	log.Printf("[Web] Starting web server on %s", w.config.ListenAddr)

	// Start server in goroutine
	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Web] Server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the web server gracefully
func (w *WebServer) Stop() error {
	log.Println("[Web] Stopping web server")

	// Close all WebSocket connections
	w.clientsMu.Lock()
	for client := range w.clients {
		client.Close()
	}
	w.clients = make(map[*websocket.Conn]bool)
	w.clientsMu.Unlock()

	// Shutdown HTTP server
	if w.server != nil {
		return w.server.Close()
	}

	return nil
}

// BroadcastStats broadcasts statistics to all connected WebSocket clients
func (w *WebServer) BroadcastStats(timestamp time.Time, stats map[string]*RateInfo) {
	// Update cache
	w.latestStatsMu.Lock()
	w.latestStats = stats
	w.latestTime = timestamp
	w.latestStatsMu.Unlock()

	// Broadcast to WebSocket clients if enabled
	if !w.config.EnableRealtime {
		return
	}

	// Convert to display format
	data := w.convertToDisplayFormat(timestamp, stats)

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[Web] Failed to marshal stats: %v", err)
		return
	}

	// Broadcast to all clients
	w.clientsMu.RLock()
	defer w.clientsMu.RUnlock()

	for client := range w.clients {
		err := client.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			log.Printf("[Web] WebSocket write error: %v", err)
			// Client will be removed on next read/write
		}
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// handleCurrentStats returns current statistics as JSON
func (w *WebServer) handleCurrentStats(rw http.ResponseWriter, r *http.Request) {
	w.latestStatsMu.RLock()
	stats := w.latestStats
	timestamp := w.latestTime
	w.latestStatsMu.RUnlock()

	data := w.convertToDisplayFormat(timestamp, stats)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(data)
}

// handleWebSocket handles WebSocket connections
func (w *WebServer) handleWebSocket(rw http.ResponseWriter, r *http.Request) {
	conn, err := w.upgrader.Upgrade(rw, r, nil)
	if err != nil {
		log.Printf("[Web] WebSocket upgrade error: %v", err)
		return
	}

	// Register client
	w.clientsMu.Lock()
	w.clients[conn] = true
	clientCount := len(w.clients)
	w.clientsMu.Unlock()

	log.Printf("[Web] New WebSocket connection (total: %d)", clientCount)

	// Send current stats immediately
	w.latestStatsMu.RLock()
	stats := w.latestStats
	timestamp := w.latestTime
	w.latestStatsMu.RUnlock()

	if len(stats) > 0 {
		data := w.convertToDisplayFormat(timestamp, stats)
		if jsonData, err := json.Marshal(data); err == nil {
			conn.WriteMessage(websocket.TextMessage, jsonData)
		}
	}

	// Handle client disconnect
	go func() {
		defer func() {
			w.clientsMu.Lock()
			delete(w.clients, conn)
			clientCount := len(w.clients)
			w.clientsMu.Unlock()
			conn.Close()
			log.Printf("[Web] WebSocket disconnected (remaining: %d)", clientCount)
		}()

		// Read loop (just to detect disconnect)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// ============================================================================
// Helper Functions
// ============================================================================

// convertToDisplayFormat converts RateInfo to display format with Upload/Download
func (w *WebServer) convertToDisplayFormat(timestamp time.Time, stats map[string]*RateInfo) map[string]interface{} {
	interfaces := make(map[string]interface{})

	for name, info := range stats {
		var uploadRate, downloadRate, uploadAvg, downloadAvg, uploadPeak, downloadPeak float64

		// Convert RX/TX to Upload/Download based on interface type
		if w.uplinkInterfaces[name] {
			// Uplink: no swap
			uploadRate = info.TxRate
			downloadRate = info.RxRate
			uploadAvg = info.TxAvg
			downloadAvg = info.RxAvg
			uploadPeak = info.TxPeak
			downloadPeak = info.RxPeak
		} else {
			// Downlink: swap TX/RX
			uploadRate = info.RxRate
			downloadRate = info.TxRate
			uploadAvg = info.RxAvg
			downloadAvg = info.TxAvg
			uploadPeak = info.RxPeak
			downloadPeak = info.TxPeak
		}

		interfaces[name] = map[string]interface{}{
			"upload_rate":     uploadRate,
			"download_rate":   downloadRate,
			"upload_avg":      uploadAvg,
			"download_avg":    downloadAvg,
			"upload_peak":     uploadPeak,
			"download_peak":   downloadPeak,
			"upload_mbps":     uploadRate * 8 / 1000000,  // Convert to Mbps
			"download_mbps":   downloadRate * 8 / 1000000,
		}
	}

	return map[string]interface{}{
		"timestamp":  timestamp.Format(time.RFC3339),
		"interfaces": interfaces,
	}
}
