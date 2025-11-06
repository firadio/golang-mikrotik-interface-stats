package main

import (
	"log"
	"time"
)

// Monitor handles traffic monitoring and rate calculation
type Monitor struct {
	client          *MikrotikClient              // Mikrotik API client
	rateMap         map[string]*InterfaceRate    // Interface rate tracking state
	interval        time.Duration                // Monitoring interval (1 second)
	interfaces      []string                     // List of interfaces to monitor
	writer          OutputWriter                 // Output handler (terminal/log/metrics)
	debug           bool                         // Enable debug logging
	statsWindowSize int                          // Statistics window size in seconds
}

// NewMonitor creates a new traffic monitor with appropriate output writer
func NewMonitor(client *MikrotikClient, config *Config) *Monitor {
	// Select output writer based on configuration
	var writer OutputWriter
	if config.OutputMode == "log" {
		writer = NewLogOutput(config.RateUnit, config.RateScale, config.UplinkInterfaces, config.StatsWindowSize)
	} else {
		refreshMode := config.DisplayMode != "append"
		writer = NewTerminalOutput(refreshMode, config.RateUnit, config.RateScale, config.UplinkInterfaces, config.StatsWindowSize)
	}

	return &Monitor{
		client:          client,
		rateMap:         make(map[string]*InterfaceRate),
		interval:        1 * time.Second,
		interfaces:      config.Interfaces,
		writer:          writer,
		debug:           config.Debug,
		statsWindowSize: config.StatsWindowSize,
	}
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

	// Write output header
	m.writer.WriteHeader()

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

	// Write stats if we have any
	if len(rateInfoMap) > 0 {
		m.writer.WriteStats(now, rateInfoMap)
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
