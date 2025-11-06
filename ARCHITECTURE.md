# Architecture Documentation

## Overview

This application monitors Mikrotik router interface traffic in real-time, calculating rates and statistics with support for multiple output formats.

## Code Structure

```
├── main.go              # Application entry point
├── client.go            # Mikrotik API client implementation
├── config.go            # Configuration management
├── stats.go             # Data structures and rate formatting
├── monitor.go           # Monitoring logic and rate calculation
├── output.go            # Output abstraction (terminal/log/metrics)
├── terminal_windows.go  # Windows ANSI support
└── terminal_unix.go     # Unix ANSI stub
```

## Architecture Layers

### 1. Connection Layer (`client.go`)
- **MikrotikClient**: Implements Mikrotik RouterOS API protocol
- Handles TCP connection, authentication, and API communication
- Uses length-encoded words with MD5 challenge-response auth
- Methods:
  - `NewMikrotikClient()`: Connect and authenticate
  - `sendCommand()`: Send API commands
  - `readResponse()`: Parse API responses
  - `GetInterfaceStats()`: Query interface statistics

### 2. Configuration Layer (`config.go`)
- **Config**: Application configuration structure
- Loads from `.env` file and environment variables
- Validates and provides defaults for all settings
- Helper functions:
  - `loadEnvFile()`: Parse .env file
  - `parseCommaSeparated()`: Parse interface lists
  - `parseIntWithDefault()`: Parse bounded integers

### 3. Data Layer (`stats.go`)
- **InterfaceStats**: Raw byte counters from Mikrotik
- **InterfaceRate**: Rate calculation state with ring buffer
- Data flow:
  ```
  Raw counters → Delta calculation → Rates (bytes/s) → Statistics (avg/peak)
  ```
- Formatting functions:
  - `FormatRate()`: Rate with unit suffix (for logs)
  - `formatNumeric()`: Numeric-only (for tables)

### 4. Monitoring Layer (`monitor.go`)
- **Monitor**: Core monitoring loop
- Responsibilities:
  - Query interfaces every second using time.Ticker
  - Calculate rates from counter deltas
  - Maintain ring buffer for statistics window
  - Compute average and peak values
- Key methods:
  - `Start()`: Main monitoring loop
  - `initializeRates()`: Bootstrap rate tracking
  - `updateAndDisplay()`: Fetch and display stats
  - `calculateRates()`: Compute rates from counters
  - `calculateStats()`: Compute avg/peak from history

### 5. Output Layer (`output.go`)
- **OutputWriter Interface**: Abstraction for different outputs
  - `WriteHeader()`: Initialize output
  - `WriteStats()`: Display statistics
  - `Close()`: Cleanup

- **TerminalOutput**: Interactive terminal display
  - Refresh mode: Overwrites screen (like `top`)
  - Append mode: Appends lines (like `tail -f`)
  - RX/TX to Upload/Download conversion

- **LogOutput**: Structured logging for daemons
  - Key-value format for parsing
  - Suitable for systemd services

- **Future: MetricsOutput** (for VictoriaMetrics)
  - Push metrics via HTTP API
  - Labels for interface name and type

### 6. UI Layer (`main.go`)
- Application initialization
- Error handling and logging
- Lifecycle management

## Data Flow

```
┌─────────────────┐
│  Mikrotik API   │
└────────┬────────┘
         │ Raw counters (rx-byte, tx-byte)
         ▼
┌─────────────────┐
│  InterfaceStats │
└────────┬────────┘
         │ Delta calculation
         ▼
┌─────────────────┐
│ InterfaceRate   │ Ring buffer for history
│  - RxRate       │
│  - TxRate       │
└────────┬────────┘
         │ Statistics calculation
         ▼
┌─────────────────┐
│   RateInfo      │
│  - RxRate       │ Current rates
│  - TxRate       │
│  - RxAvg/TxAvg  │ Window averages
│  - RxPeak/TxPeak│ Window peaks
└────────┬────────┘
         │ Display conversion
         ▼
┌─────────────────┐
│  OutputWriter   │
│  - Terminal     │ User-friendly display
│  - Log          │ Structured logging
│  - Metrics      │ Time-series metrics
└─────────────────┘
```

## Key Design Decisions

### 1. RX/TX vs Upload/Download
- **Data layer**: Always uses RX/TX (raw counters)
- **Display layer**: Converts to Upload/Download based on interface type
  - **Uplink (WAN)**: TX=Upload, RX=Download (no swap)
  - **Downlink (LAN)**: TX=Download, RX=Upload (swap needed)

### 2. Ring Buffer for Statistics
- Fixed-size slice allocated once
- Circular indexing with modulo operator
- Tracks valid entry count during warmup
- Configurable window size (1-60 seconds)

### 3. Output Abstraction
- Interface-based design for extensibility
- Easy to add new outputs (metrics, CSV, JSON, etc.)
- Each output controls its own formatting

### 4. Configuration Philosophy
- Environment variables for runtime config
- .env file for development convenience
- Validation with sensible defaults
- Helper functions for parsing

## Preparing for VictoriaMetrics

### Metrics Format
VictoriaMetrics expects Prometheus-compatible metrics:

```
interface_rx_rate{interface="vlan2622",type="downlink"} 1234567.89
interface_tx_rate{interface="vlan2622",type="downlink"} 987654.32
interface_rx_avg{interface="vlan2622",type="downlink"} 1200000.00
interface_tx_avg{interface="vlan2622",type="downlink"} 950000.00
interface_rx_peak{interface="vlan2622",type="downlink"} 1500000.00
interface_tx_peak{interface="vlan2622",type="downlink"} 1100000.00
```

### Implementation Plan

1. **Create MetricsOutput** (implements OutputWriter)
   ```go
   type MetricsOutput struct {
       endpoint         string // VictoriaMetrics URL
       client          *http.Client
       uplinkInterfaces map[string]bool
   }
   ```

2. **WriteStats Method**
   - Format metrics in Prometheus format
   - Add labels: interface name, type (uplink/downlink)
   - POST to VictoriaMetrics `/api/v1/import/prometheus`

3. **Configuration**
   ```env
   OUTPUT_MODE=metrics
   VICTORIA_METRICS_URL=http://localhost:8428
   ```

4. **Batching**
   - Collect all interface metrics
   - Single HTTP request per interval
   - Error handling with retry logic

## Testing Strategy

1. **Unit tests**: Rate calculation logic
2. **Integration tests**: Mikrotik API communication
3. **Manual tests**: Various configurations and displays
4. **Load tests**: Many interfaces, long runtime

## Performance Considerations

- Ring buffer avoids allocations
- Map lookups are O(1) with interface name keys
- String formatting only when displaying
- No goroutines (single-threaded by design)
- Ticker ensures precise intervals

## Future Enhancements

1. **VictoriaMetrics integration** (next step)
2. **HTTP API**: REST API for querying current stats
3. **Alerting**: Thresholds for email/webhook notifications
4. **Web UI**: Real-time dashboard with charts
5. **Multiple routers**: Monitor multiple Mikrotik devices
6. **Historical data**: SQLite for local time-series storage
