package main

import (
	"log"
	"time"
)

// Monitor monitors interface traffic and displays statistics
type Monitor struct {
	client          *MikrotikClient
	rateMap         map[string]*InterfaceRate
	interval        time.Duration
	interfaces      []string
	writer          OutputWriter
	debug           bool
	statsWindowSize int // Statistics window size in seconds
}

// NewMonitor creates a new traffic monitor
func NewMonitor(client *MikrotikClient, config *Config) *Monitor {
	// Create appropriate output writer based on config
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

// Start starts the monitoring loop
func (m *Monitor) Start() error {
	// Use ticker to avoid missed seconds
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Get initial stats
	stats, err := m.client.GetInterfaceStats(m.interfaces, m.debug)
	if err != nil {
		log.Printf("Warning: Failed to get initial stats: %v", err)
	} else {
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
	}

	// Write header
	m.writer.WriteHeader()

	// Display loop
	for range ticker.C {
		stats, err := m.client.GetInterfaceStats(m.interfaces, m.debug)
		if err != nil {
			log.Printf("Error getting stats: %v", err)
			continue
		}

		if len(stats) == 0 {
			// No matching interfaces found, skip silently
			continue
		}

		now := time.Now()

		// Build rate info map
		rateInfoMap := make(map[string]*RateInfo)

		// Calculate rates for each interface
		for _, stat := range stats {
			if prev, ok := m.rateMap[stat.Name]; ok {
				// Calculate time difference
				timeDiff := now.Sub(prev.LastTime).Seconds()
				if timeDiff > 0 {
					// Calculate rates (bytes per second)
					rxRate := float64(stat.RxByte-prev.LastRxByte) / timeDiff
					txRate := float64(stat.TxByte-prev.LastTxByte) / timeDiff

					// Update history (ring buffer) - store raw RX/TX rates
					prev.TxHistory[prev.HistoryIndex] = txRate
					prev.RxHistory[prev.HistoryIndex] = rxRate
					prev.HistoryIndex = (prev.HistoryIndex + 1) % m.statsWindowSize
					if prev.HistoryCount < m.statsWindowSize {
						prev.HistoryCount++
					}

					// Calculate average and peak from history
					var txSum, rxSum float64
					txPeak := txRate
					rxPeak := rxRate
					for i := 0; i < prev.HistoryCount; i++ {
						txSum += prev.TxHistory[i]
						rxSum += prev.RxHistory[i]
						if prev.TxHistory[i] > txPeak {
							txPeak = prev.TxHistory[i]
						}
						if prev.RxHistory[i] > rxPeak {
							rxPeak = prev.RxHistory[i]
						}
					}
					txAvg := txSum / float64(prev.HistoryCount)
					rxAvg := rxSum / float64(prev.HistoryCount)

					// Update stored values for next calculation
					prev.LastRxByte = stat.RxByte
					prev.LastTxByte = stat.TxByte
					prev.LastTime = now

					// Add to rate info map - use RX/TX naming
					rateInfoMap[stat.Name] = &RateInfo{
						InterfaceName: stat.Name,
						RxRate:        rxRate,
						TxRate:        txRate,
						RxAvg:         rxAvg, // RX average
						TxAvg:         txAvg, // TX average
						RxPeak:        rxPeak, // RX peak
						TxPeak:        txPeak, // TX peak
					}
				}
			} else {
				// Initialize for new interface
				m.rateMap[stat.Name] = &InterfaceRate{
					Name:       stat.Name,
					LastRxByte: stat.RxByte,
					LastTxByte: stat.TxByte,
					LastTime:   now,
					TxHistory:  make([]float64, m.statsWindowSize),
					RxHistory:  make([]float64, m.statsWindowSize),
				}
			}
		}

		// Write stats if we have any
		if len(rateInfoMap) > 0 {
			m.writer.WriteStats(now, rateInfoMap)
		}
	}

	return nil
}
