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

- **Web Service** (WEB_ENABLED) - Stub implementation
  - HTTP server framework
  - WebSocket real-time push (planned)
  - REST API for historical queries (planned)
  - Static web UI (planned)

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
