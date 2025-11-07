package main

import (
	"log"
	"time"
)

// Monitor handles traffic monitoring and rate calculation
type Monitor struct {
	client           *MikrotikClient           // Mikrotik API client
	rateMap          map[string]*InterfaceRate // Interface rate tracking state
	interval         time.Duration             // Monitoring interval (1 second)
	interfaces       []string                  // List of interfaces to monitor
	uplinkInterfaces map[string]bool           // Uplink interface set
	debug            bool                      // Enable debug logging
	statsWindowSize  int                       // Statistics window size in seconds

	// Optional output components (nil if disabled)
	terminalWriter *TerminalOutput     // Terminal output
	logWriter      *StructuredLogger   // Structured log output
	webServer      *WebServer          // Web server
	vmClient       *VMClient           // VictoriaMetrics client
	aggregator     *TimeWindowAggregator // Time window aggregator
}

// NewMonitor creates a new traffic monitor with appropriate output handlers
func NewMonitor(client *MikrotikClient, config *Config) *Monitor {
	m := &Monitor{
		client:           client,
		rateMap:          make(map[string]*InterfaceRate),
		interval:         1 * time.Second,
		interfaces:       config.Interfaces,
		uplinkInterfaces: toSet(config.UplinkInterfaces),
		debug:            config.Debug,
		statsWindowSize:  config.StatsWindowSize,
	}

	// Initialize terminal output if enabled
	if config.Terminal != nil {
		refreshMode := config.Terminal.Mode == "refresh"
		m.terminalWriter = NewTerminalOutput(
			refreshMode,
			config.Terminal.RateUnit,
			config.Terminal.RateScale,
			config.UplinkInterfaces,
			config.StatsWindowSize,
		)
	}

	// Initialize log output if enabled
	if config.Log != nil {
		m.logWriter = NewStructuredLogger(config.Log, config.UplinkInterfaces)
	}

	// Initialize VictoriaMetrics if enabled (BEFORE web server to ensure vmClient is available)
	if config.VictoriaMetrics != nil {
		m.vmClient = NewVMClient(config.VictoriaMetrics)
		m.aggregator = NewTimeWindowAggregator(config.VictoriaMetrics.Interval)
	}

	// Initialize web server if enabled (AFTER VictoriaMetrics to get vmClient)
	if config.Web != nil {
		m.webServer = NewWebServer(config.Web, config.UplinkInterfaces, m.vmClient)
	}

	return m
}

// toSet converts a slice to a set (map[string]bool)
func toSet(list []string) map[string]bool {
	set := make(map[string]bool, len(list))
	for _, item := range list {
		set[item] = true
	}
	return set
}

// Start begins the monitoring loop
// Queries interfaces every second and calculates rates
func (m *Monitor) Start() error {
	// Use ticker for precise 1-second intervals
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Initialize rate tracking with first stats
	if err := m.initializeRates(); err != nil {
		log.Printf("Warning: Failed to get initial stats: %v", err)
	}

	// Start web server if enabled
	if m.webServer != nil {
		if err := m.webServer.Start(); err != nil {
			log.Printf("Warning: Failed to start web server: %v", err)
		}
		defer m.webServer.Stop()
	}

	// Write header for terminal/log output
	if m.terminalWriter != nil {
		m.terminalWriter.WriteHeader()
	}
	if m.logWriter != nil {
		m.logWriter.WriteHeader()
	}

	// Main monitoring loop
	for range ticker.C {
		if err := m.updateAndDisplay(); err != nil {
			log.Printf("Error in monitoring loop: %v", err)
		}
	}

	return nil
}

// initializeRates fetches initial statistics to establish baseline
func (m *Monitor) initializeRates() error {
	stats, err := m.client.GetInterfaceStats(m.interfaces, m.debug)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, stat := range stats {
		m.rateMap[stat.Name] = &InterfaceRate{
			Name:       stat.Name,
			LastRxByte: stat.RxByte,
			LastTxByte: stat.TxByte,
			LastTime:   now,
			TxHistory:  make([]float64, m.statsWindowSize),
			RxHistory:  make([]float64, m.statsWindowSize),
		}
	}

	return nil
}

// updateAndDisplay fetches new stats, calculates rates, and displays results
func (m *Monitor) updateAndDisplay() error {
	stats, err := m.client.GetInterfaceStats(m.interfaces, m.debug)
	if err != nil {
		return err
	}

	if len(stats) == 0 {
		return nil // No matching interfaces
	}

	now := time.Now()
	rateInfoMap := m.calculateRates(stats, now)

	if len(rateInfoMap) == 0 {
		return nil
	}

	// 1. Terminal output (if enabled)
	if m.terminalWriter != nil {
		m.terminalWriter.WriteStats(now, rateInfoMap)
	}

	// 2. Structured log output (if enabled)
	if m.logWriter != nil {
		m.logWriter.WriteStats(now, rateInfoMap)
	}

	// 3. WebSocket push (if enabled)
	if m.webServer != nil {
		m.webServer.BroadcastStats(now, rateInfoMap)
	}

	// 4. VictoriaMetrics aggregation (if enabled)
	if m.aggregator != nil {
		for ifaceName, rateInfo := range rateInfoMap {
			m.aggregator.AddSample(now, ifaceName, rateInfo.RxRate, rateInfo.TxRate)
		}

		// Check for completed windows and send to VM
		if windows := m.aggregator.GetCompletedWindows(); len(windows) > 0 {
			for _, window := range windows {
				if err := m.vmClient.SendMetrics(window); err != nil {
					log.Printf("[VM] Failed to send metrics: %v", err)
				}
			}
		}
	}

	return nil
}

// calculateRates computes current rates and statistics from raw counters
func (m *Monitor) calculateRates(stats []InterfaceStats, now time.Time) map[string]*RateInfo {
	rateInfoMap := make(map[string]*RateInfo, len(stats))

	for _, stat := range stats {
		prev, exists := m.rateMap[stat.Name]
		if !exists {
			// Initialize new interface
			m.rateMap[stat.Name] = &InterfaceRate{
				Name:       stat.Name,
				LastRxByte: stat.RxByte,
				LastTxByte: stat.TxByte,
				LastTime:   now,
				TxHistory:  make([]float64, m.statsWindowSize),
				RxHistory:  make([]float64, m.statsWindowSize),
			}
			continue
		}

		// Calculate time delta
		timeDiff := now.Sub(prev.LastTime).Seconds()
		if timeDiff <= 0 {
			continue
		}

		// Calculate instantaneous rates (bytes/second)
		rxRate := float64(stat.RxByte-prev.LastRxByte) / timeDiff
		txRate := float64(stat.TxByte-prev.LastTxByte) / timeDiff

		// Update ring buffer with new rates
		prev.TxHistory[prev.HistoryIndex] = txRate
		prev.RxHistory[prev.HistoryIndex] = rxRate
		prev.HistoryIndex = (prev.HistoryIndex + 1) % m.statsWindowSize
		if prev.HistoryCount < m.statsWindowSize {
			prev.HistoryCount++
		}

		// Calculate statistics from history
		txAvg, txPeak := m.calculateStats(prev.TxHistory, prev.HistoryCount)
		rxAvg, rxPeak := m.calculateStats(prev.RxHistory, prev.HistoryCount)

		// Update baseline for next iteration
		prev.LastRxByte = stat.RxByte
		prev.LastTxByte = stat.TxByte
		prev.LastTime = now

		// Store calculated rate info
		rateInfoMap[stat.Name] = &RateInfo{
			InterfaceName: stat.Name,
			RxRate:        rxRate,
			TxRate:        txRate,
			RxAvg:         rxAvg,
			TxAvg:         txAvg,
			RxPeak:        rxPeak,
			TxPeak:        txPeak,
		}
	}

	return rateInfoMap
}

// calculateStats computes average and peak from a history buffer
func (m *Monitor) calculateStats(history []float64, count int) (avg float64, peak float64) {
	if count == 0 {
		return 0, 0
	}

	var sum float64
	peak = history[0]

	for i := 0; i < count; i++ {
		sum += history[i]
		if history[i] > peak {
			peak = history[i]
		}
	}

	avg = sum / float64(count)
	return avg, peak
}
