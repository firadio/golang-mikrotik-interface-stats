# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.0.1] - 2025-11-07

### Added
- Initial release
- Core monitoring functionality
  - Connect to Mikrotik API
  - Query interface statistics every second
  - Calculate traffic rates (current, average, peak)
  - Support uplink/downlink interface distinction

- **Terminal Output** (TERMINAL_ENABLED)
  - Refresh mode (like top/htop)
  - Append mode (like tail -f)
  - Configurable units (bps/Bps) and scales (auto/k/M/G)
  - Real-time statistics with 10-second window

- **Structured Logging** (LOG_ENABLED)
  - JSON and text formats
  - Output to stdout or file
  - Suitable for systemd services

- **Web Service** (WEB_ENABLED)
  - HTTP server with route management
  - WebSocket real-time push with auto-reconnect
  - REST API for current statistics (/api/current)
  - **Dual-mode static file serving** (Go 1.16+ embed)
    - Production mode: Files embedded in binary (single exe distribution)
    - Developer mode: Hot-reload from local `web/` directory (instant changes)
    - Automatic mode detection at startup
  - Modular frontend (HTML/CSS/JS separated)
  - **Real-time line charts** powered by Chart.js 4.4.0
    - Smooth 60-second historical view
    - Upload/Download dual-line visualization
    - Gradient fills and hover tooltips
    - Auto-scaling Y-axis in Mbps
  - **Modern card-based UI** inspired by mainstream monitoring tools
    - Animated status indicator with pulse effect
    - Current/Average/Peak statistics badges
    - Responsive grid layout
    - Slate dark theme with CSS custom properties
  - Thread-safe client connection management
  - Stats caching for immediate delivery to new connections
  - Easy to extend and customize

- **VictoriaMetrics Integration** (VM_ENABLED) - Stub implementation
  - Time window aggregator (planned)
  - Short-term aggregation (10s intervals)
  - Long-term aggregation (5m intervals)
  - Push metrics to VictoriaMetrics (planned)

- Configuration management
  - Environment variable support
  - .env file support
  - Feature toggles (all disabled by default)
  - Comprehensive validation

### Features
- All output features disabled by default (silent mode)
- Terminal and Log can be used together (Terminal to screen, Log to file)
- Terminal + Log(stdout) conflict detection (cannot both write to stdout)
- Web and VictoriaMetrics can be enabled independently
- Clear startup information showing enabled features
- Helpful tips when running in silent mode

### Documentation
- Comprehensive README with usage examples
- Architecture documentation (ARCHITECTURE.md)
- Detailed .env.example with all options

[v0.0.1]: https://github.com/yourusername/mikrotik-interface-stats/releases/tag/v0.0.1
