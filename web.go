package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================================
// Web Server Implementation
// ============================================================================

// Embed static files into binary (production mode)
//go:embed web
var embeddedFS embed.FS

// WebServer handles HTTP/WebSocket server for real-time monitoring
type WebServer struct {
	config           *WebConfig
	uplinkInterfaces map[string]bool
	server           *http.Server
	vmClient         *VMClient         // For historical data queries
	userConfig       *UserConfigManager // For user configuration management

	// WebSocket client management
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader

	// Latest stats cache
	latestStats   map[string]*RateInfo
	latestTime    time.Time
	latestStatsMu sync.RWMutex
}

// getWebFS returns the appropriate file system (local or embedded)
// Developer mode: If "web" directory exists, use local files for hot-reload
// Production mode: Use embedded files from binary
func getWebFS() (http.FileSystem, bool) {
	const webDir = "web"

	// Check if web directory exists (developer mode)
	if stat, err := os.Stat(webDir); err == nil && stat.IsDir() {
		log.Printf("[Web] Developer mode: Using local files from '%s/' directory", webDir)
		log.Printf("[Web] 💡 Tip: Remove '%s/' directory to test production mode (embedded files)", webDir)
		return http.Dir(webDir), true
	}

	// Production mode: use embedded files
	log.Println("[Web] Production mode: Using embedded files from binary")

	// Strip "web" prefix from embedded FS
	webContent, err := fs.Sub(embeddedFS, webDir)
	if err != nil {
		log.Printf("[Web] Warning: Failed to access embedded files: %v", err)
		return nil, false
	}

	return http.FS(webContent), false
}

// NewWebServer creates a new web server
func NewWebServer(config *WebConfig, uplinkInterfaces []string, vmClient *VMClient) *WebServer {
	log.Printf("[Web] Web server initialized (addr: %s)", config.ListenAddr)

	// Convert uplink interface list to set
	uplinkSet := make(map[string]bool, len(uplinkInterfaces))
	for _, iface := range uplinkInterfaces {
		uplinkSet[iface] = true
	}

	// Initialize user configuration manager
	userConfigMgr, err := NewUserConfigManager()
	if err != nil {
		log.Printf("[Web] Warning: Failed to initialize user config: %v", err)
	}

	ws := &WebServer{
		config:           config,
		uplinkInterfaces: uplinkSet,
		vmClient:         vmClient,
		userConfig:       userConfigMgr,
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
		// Get appropriate file system (local or embedded)
		webFS, isDev := getWebFS()
		if webFS != nil {
			fileServer := http.FileServer(webFS)
			mux.Handle("/", fileServer)

			// Log mode for clarity
			if isDev {
				log.Println("[Web] Static files: Hot-reload enabled (changes take effect immediately)")
			} else {
				log.Println("[Web] Static files: Serving from embedded binary")
			}
		} else {
			log.Println("[Web] ERROR: Failed to initialize file system")
		}
	}

	if config.EnableAPI {
		mux.HandleFunc("/api/current", ws.handleCurrentStats)
		mux.HandleFunc("/api/history", ws.handleHistoryQuery)
		mux.HandleFunc("/api/config/labels", ws.handleInterfaceLabels)
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
		var uploadRate, downloadRate float64

		// Convert RX/TX to Upload/Download based on interface type
		if w.uplinkInterfaces[name] {
			// Uplink: no swap
			uploadRate = info.TxRate
			downloadRate = info.RxRate
		} else {
			// Downlink: swap TX/RX
			uploadRate = info.RxRate
			downloadRate = info.TxRate
		}

		interfaces[name] = map[string]interface{}{
			"upload_rate":   uploadRate,
			"download_rate": downloadRate,
		}
	}

	return map[string]interface{}{
		"timestamp":  timestamp.Format(time.RFC3339),
		"interfaces": interfaces,
	}
}

// handleHistoryQuery returns historical statistics from VictoriaMetrics
func (w *WebServer) handleHistoryQuery(rw http.ResponseWriter, r *http.Request) {
	// Check if VM is enabled
	if w.vmClient == nil {
		http.Error(rw, "VictoriaMetrics not enabled", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	interfaceName := query.Get("interface")
	startStr := query.Get("start")
	endStr := query.Get("end")
	interval := query.Get("interval")

	// Validate required parameters
	if interfaceName == "" {
		http.Error(rw, "Missing 'interface' parameter", http.StatusBadRequest)
		return
	}

	// Parse time range
	var start, end time.Time
	var err error

	if startStr == "" {
		// Default to last 24 hours
		end = time.Now()
		start = end.Add(-24 * time.Hour)
	} else {
		// Try parsing as Unix timestamp (seconds)
		if startInt, err2 := strconv.ParseInt(startStr, 10, 64); err2 == nil {
			start = time.Unix(startInt, 0)
		} else {
			// Try parsing as RFC3339
			start, err = time.Parse(time.RFC3339, startStr)
			if err != nil {
				http.Error(rw, "Invalid 'start' time format", http.StatusBadRequest)
				return
			}
		}

		if endStr == "" {
			end = time.Now()
		} else {
			if endInt, err2 := strconv.ParseInt(endStr, 10, 64); err2 == nil {
				end = time.Unix(endInt, 0)
			} else {
				end, err = time.Parse(time.RFC3339, endStr)
				if err != nil {
					http.Error(rw, "Invalid 'end' time format", http.StatusBadRequest)
					return
				}
			}
		}
	}

	// Validate time range
	if start.After(end) {
		http.Error(rw, "Start time must be before end time", http.StatusBadRequest)
		return
	}

	// Default interval to auto
	if interval == "" {
		interval = "auto"
	}

	// Query VictoriaMetrics
	resp, err := w.vmClient.QueryHistory(HistoryQueryParams{
		Interface: interfaceName,
		Start:     start,
		End:       end,
		Interval:  interval,
	})

	if err != nil {
		log.Printf("[Web] History query error: %v", err)
		http.Error(rw, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to display format (swap RX/TX if needed)
	w.convertHistoryToDisplayFormat(resp)

	// Return JSON response
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(resp)
}

// convertHistoryToDisplayFormat converts RX/TX to Upload/Download for history data
func (w *WebServer) convertHistoryToDisplayFormat(resp *HistoryResponse) {
	isUplink := w.uplinkInterfaces[resp.Interface]

	for i := range resp.DataPoints {
		dp := &resp.DataPoints[i]

		if isUplink {
			// Uplink: TX=Upload, RX=Download (no swap)
			// Already correct
		} else {
			// Downlink: TX=Download, RX=Upload (need swap)
			dp.UploadAvg, dp.DownloadAvg = dp.DownloadAvg, dp.UploadAvg
			dp.UploadPeak, dp.DownloadPeak = dp.DownloadPeak, dp.UploadPeak
		}
	}
}

// ============================================================================
// User Configuration API
// ============================================================================

// handleInterfaceLabels handles GET and PUT requests for interface labels
func (ws *WebServer) handleInterfaceLabels(w http.ResponseWriter, r *http.Request) {
	if ws.userConfig == nil {
		http.Error(w, "User configuration not initialized", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Return all interface labels
		labels := ws.userConfig.GetAllInterfaceLabels()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(labels); err != nil {
			log.Printf("[Web] Error encoding interface labels: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

	case http.MethodPut:
		// Update interface labels
		var labels map[string]string
		if err := json.NewDecoder(r.Body).Decode(&labels); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		if err := ws.userConfig.UpdateInterfaceLabels(labels); err != nil {
			log.Printf("[Web] Error updating interface labels: %v", err)
			http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
