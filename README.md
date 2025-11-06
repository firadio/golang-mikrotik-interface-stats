# Mikrotik Interface Stats

Monitor Mikrotik interface traffic statistics in real-time.

## Features

- ✅ Connect to Mikrotik API
- ✅ Configurable interface list via .env
- ✅ Calculate per-second traffic rates
- ✅ **User-friendly Download/Upload display** (automatically handles uplink/downlink interfaces)
- ✅ Multiple display modes:
  - Refresh mode (like top/htop)
  - Append mode (like tail -f)
  - Log mode (for services/daemons)
- ✅ Configurable rate units (bits vs bytes per second)
- ✅ Auto-scaling or fixed-scale display with decimal alignment
- ✅ Precise timing using time.Ticker (no missed seconds)
- ✅ Real-time monitoring with 1-second intervals
- ✅ Modular output system for easy extension
- ✅ **Web interface with real-time Chart.js graphs**
- ✅ **Historical data storage and query via VictoriaMetrics**
- ✅ **Embedded static files** (single-file distribution with developer mode support)

## Configuration

Create a `.env` file in the project root or set environment variables:

```env
MIKROTIK_HOST=175.100.109.154
MIKROTIK_PORT=65428
MIKROTIK_USERNAME=your_username
MIKROTIK_PASSWORD=your_password

# Interface list (comma-separated)
INTERFACES=vlan2622,vlan2624

# Uplink interfaces (optional, comma-separated)
UPLINK_INTERFACES=

# Display mode (optional)
DISPLAY_MODE=refresh  # or "append"

# Rate unit (optional)
RATE_UNIT=auto  # or "bps" (bits/s), "Bps" (Bytes/s)

# Rate scale (optional)
RATE_SCALE=auto  # or "k", "M", "G" for fixed scale

# Output mode (optional)
OUTPUT_MODE=terminal  # or "log"

# Debug mode (optional)
DEBUG=false  # or "true" to see API commands

# Web interface (optional)
WEB_ENABLED=true
WEB_LISTEN_ADDR=:8080

# VictoriaMetrics (optional)
VM_ENABLED=true
VM_URL=http://localhost:8428
VM_SHORT_INTERVAL=10s
VM_LONG_INTERVAL=300s
VM_ENABLE_SHORT=true
VM_ENABLE_LONG=true
```

**Configuration Options:**

- **INTERFACES**: Comma-separated list of interfaces to monitor (default: vlan2622,vlan2624)

- **UPLINK_INTERFACES**: Comma-separated list of uplink interfaces (optional)
  - **Uplink (WAN to ISP)**: TX=Upload, RX=Download (normal understanding)
  - **Downlink (LAN/VLAN to users)**: TX=Download (to user), RX=Upload (from user) - needs swap
  - Example: `UPLINK_INTERFACES=ether1,sfp1` if ether1 and sfp1 connect to ISP
  - Leave empty if all monitored interfaces are downlink (e.g., LANs, VLANs)
  - **Why needed?** For downlink interfaces, the router sends data TO users (TX), which is actually user's Download. The router receives data FROM users (RX), which is actually user's Upload.

- **DISPLAY_MODE**: How to display output
  - `refresh` (default) - Redraw display like `top`/`htop`
    - Uses ANSI cursor control (moves to home position and overwrites)
    - No full screen clear, reduces flicker
    - **Recommended terminals:**
      - Windows: Windows Terminal, PowerShell 7+, Git Bash
      - Linux/macOS: Any standard terminal
  - `append` - Append new lines like `tail -f`
    - Suitable for logging and redirecting to files

- **RATE_UNIT**: Display unit for traffic rates
  - `auto` (default) - Uses Bytes per second (B/s)
  - `bps` - Bits per second (multiplies by 8)
  - `Bps` - Bytes per second

- **RATE_SCALE**: Scale for displaying rates
  - `auto` (default) - Automatically scales (B/s, KB/s, MB/s, GB/s)
  - `k` - Fixed kilobit/kilobyte scale
  - `M` - Fixed megabit/megabyte scale (good for high-speed interfaces)
  - `G` - Fixed gigabit/gigabyte scale
  - Fixed scales use 7.2f format for decimal alignment (e.g., "  12.34 Mbps")

- **OUTPUT_MODE**: Output format
  - `terminal` (default) - Formatted table output for interactive use
  - `log` - Log-style output for running as a service/daemon

- **DEBUG**: Enable debug output
  - `false` (default) - Normal operation
  - `true` or `1` - Print Mikrotik API commands being sent (useful for troubleshooting)

- **WEB_ENABLED**: Enable web interface
  - `true` - Enable real-time web dashboard with Chart.js graphs
  - `false` (default) - Disable web interface

- **WEB_LISTEN_ADDR**: Web server listen address
  - Default: `:8080` - Listen on all interfaces, port 8080
  - Example: `localhost:8080` - Only accessible from localhost

- **VM_ENABLED**: Enable VictoriaMetrics integration
  - `true` - Enable metrics aggregation and push to VictoriaMetrics
  - `false` (default) - Disable VictoriaMetrics integration

- **VM_URL**: VictoriaMetrics server URL
  - Default: `http://localhost:8428`
  - Must include protocol (http/https)

- **VM_SHORT_INTERVAL**: Short-term aggregation interval
  - Default: `10s` - 10-second windows for detailed monitoring
  - Suitable for <1 hour time ranges

- **VM_LONG_INTERVAL**: Long-term aggregation interval
  - Default: `300s` (5 minutes) - 5-minute windows for historical data
  - Suitable for >1 hour time ranges

- **VM_ENABLE_SHORT**: Enable short-term aggregation
  - `true` (default) - Enable 10-second aggregation
  - `false` - Disable short-term metrics

- **VM_ENABLE_LONG**: Enable long-term aggregation
  - `true` (default) - Enable 5-minute aggregation
  - `false` - Disable long-term metrics

See `.env.example` for reference.

### Windows Terminal Support

**Good news:** The program automatically enables Virtual Terminal Processing on Windows!

This means refresh mode will work in:
- ✅ Windows CMD (automatically enabled)
- ✅ Windows Terminal
- ✅ PowerShell (all versions)
- ✅ Git Bash
- ✅ Any Windows console

No manual configuration needed! The program uses Windows API to enable ANSI support at startup.

If you still see escape codes like `[2J[H]` (very rare), you can switch to append mode:
```bash
DISPLAY_MODE=append
```

## Usage

Run without building (recommended for development):
```bash
go run .
```

Build and run:
```bash
go build -o mikrotik-stats
./mikrotik-stats
```

### Web Interface

When web interface is enabled (`WEB_ENABLED=true`), access the dashboard at:
```
http://localhost:8080
```

**Features:**
- ✅ Real-time monitoring with 60-second rolling charts
- ✅ WebSocket connection for live updates (1-second refresh)
- ✅ Historical data query interface
- ✅ Interactive Chart.js graphs with zoom/pan
- ✅ Time range selector (1h, 6h, 24h, 7d, 30d, custom)
- ✅ Per-interface statistics (Average + Peak rates)
- ✅ Modern dark theme with responsive design
- ✅ Automatic reconnection on disconnect

**Historical Data:**
Click the "📊 History" button on any interface card to:
- View historical data from VictoriaMetrics
- Select time ranges (1 hour to 30 days)
- Choose aggregation interval (auto, 10s, 5min)
- See 4 metrics: Upload/Download Average and Peak
- Export-ready data visualization

**Developer Mode:**
- If `web/` directory exists: Uses local files (hot-reload)
- Otherwise: Uses embedded files from binary
- No separate static file distribution needed

## Output Examples

**Refresh Mode (default) with 7-column numeric display:**
```
Mikrotik Interface Traffic Monitor
================================================================================
Time: 2025-11-07 01:08:36
Unit: MB/s
--------------------------------------------------------------------------------
Interface          Up       Down      UpAvg      DnAvg     UpPeak     DnPeak
--------------------------------------------------------------------------------
vlan2622         0.03       0.03       0.03       0.03       0.04       0.03
vlan2624         0.58       3.19       0.52       3.10       0.65       3.50
--------------------------------------------------------------------------------
Press Ctrl+C to stop
```

**Features:**
- **Unit on top**: Single unit display (e.g., "MB/s", "kbps") applies to all columns
- **Pure numeric**: All values are numbers with .00 decimal format
- **Right-aligned**: Easy to compare values visually
- **Real-time rates**: Current upload/download speeds (Up/Down)
- **10-second averages**: UpAvg/DnAvg - smoothed rates over last 10 seconds
- **10-second peaks**: UpPeak/DnPeak - maximum speeds in last 10 seconds
- **80-column display**: 7 columns × 10 chars = 70 chars (fits standard terminals)

Note: Display shows "Upload" and "Download" from user perspective. If an interface is configured as uplink, RX/TX are swapped automatically.

**Append Mode:**
```
2025/11/07 01:09:12 Connected to Mikrotik at 175.100.109.154:65428

Monitoring interface traffic (Ctrl+C to stop):
================================================================================
[2025-11-07 01:09:13] vlan2622: Upload: 18.12 KB/s  Download: 15.82 KB/s
[2025-11-07 01:09:13] vlan2624: Upload: 655.47 KB/s  Download: 3.00 MB/s
[2025-11-07 01:09:14] vlan2622: Upload: 50.48 KB/s  Download: 16.60 KB/s
[2025-11-07 01:09:14] vlan2624: Upload: 431.54 KB/s  Download: 3.64 MB/s
```

**Fixed scale mode (RATE_SCALE=M):**
```
Mikrotik Interface Traffic Monitor
================================================================================
Time: 2025-11-07 01:08:36
--------------------------------------------------------------------------------
Interface       Upload               Download
--------------------------------------------------------------------------------
vlan2622          0.03 MB/s            0.03 MB/s
vlan2624          0.58 MB/s            3.19 MB/s
--------------------------------------------------------------------------------
Press Ctrl+C to stop
```

**Log mode (OUTPUT_MODE=log):**
```
2025/11/07 01:09:12 Connected to Mikrotik at 175.100.109.154:65428
2025/11/07 01:09:12 Mikrotik Interface Traffic Monitor started
2025/11/07 01:09:13 interface=vlan2622 upload=18.12 KB/s download=15.82 KB/s
2025/11/07 01:09:13 interface=vlan2624 upload=655.47 KB/s download=3.00 MB/s
2025/11/07 01:09:14 interface=vlan2622 upload=50.48 KB/s download=16.60 KB/s
2025/11/07 01:09:14 interface=vlan2624 upload=431.54 KB/s download=3.64 MB/s
```

## Requirements

- Go 1.21 or later
- Access to Mikrotik Router with API enabled
- Valid Mikrotik credentials

## Project Structure

```
.
├── main.go                 # Program entry point
├── config.go               # Configuration loading
├── client.go               # Mikrotik API client
├── stats.go                # Statistics data structures and formatting
├── monitor.go              # Monitoring logic
├── output.go               # Output abstraction (terminal/log modes)
├── web.go                  # Web server with WebSocket + embedded files
├── vm.go                   # VictoriaMetrics client and aggregation
├── terminal_windows.go     # Windows ANSI support (build tag: windows)
├── terminal_unix.go        # Unix ANSI stub (build tag: !windows)
├── web/                    # Web interface files (embedded)
│   ├── index.html          # Main HTML structure
│   └── static/
│       ├── css/
│       │   └── style.css   # Dark theme styles
│       └── js/
│           └── app.js      # Real-time + historical data logic
├── go.mod                  # Go module configuration
├── .env                    # Environment configuration
├── DEPLOYMENT.md           # Deployment guide
└── README.md               # Documentation
```

## Implementation Details

### Core Monitoring
- Uses Mikrotik API protocol directly (no external dependencies)
- Implements proper MD5 challenge-response authentication
- Server-side filtering using Mikrotik API query syntax (reduces network overhead)
- Stores previous byte counts to calculate delta per second
- Uses `time.Ticker` for accurate 1-second intervals
- Configurable interface list via environment variables
- **Automatically enables ANSI support on Windows** via Windows API (no manual setup needed)

### Output System
- Modular output system with OutputWriter interface:
  - TerminalOutput: Interactive display (refresh/append modes)
  - LogOutput: Service-friendly structured logging
  - WebServer: Real-time WebSocket dashboard
- Configurable rate units (bits vs bytes) and scales (auto/fixed)
- Fixed-scale formatting with decimal alignment for easy reading
- **Efficient cursor control**: Uses ANSI escape sequences to move cursor instead of clearing screen
  - Reduces flicker and improves visual stability
  - Interfaces always displayed in alphabetical order (no jumping)

### Web Interface
- **Embedded static files**: Go 1.16+ `//go:embed` directive
- **Developer mode**: Auto-detects `web/` directory for hot-reload
- **Production mode**: Uses embedded files from binary (9.5MB total)
- **WebSocket**: Real-time push of interface statistics
- **Chart.js 4.4.0**: Modern, responsive graphs with time axis
- **Dark theme**: Slate color scheme optimized for monitoring

### VictoriaMetrics Integration
- **Fixed time-boundary aggregation**: Windows aligned to intervals (not sliding)
- **Dual-interval support**: 10s (short-term) + 300s (long-term)
- **Prometheus format**: Compatible with standard VM import API
- **Retry logic**: Automatic retry with exponential backoff
- **Query API**: PromQL-based historical data retrieval
- **Auto-interval selection**: Chooses appropriate granularity based on time range
- **Metrics exported**:
  - `mikrotik_interface_rx_rate_avg{interface,interval}` - Average download rate
  - `mikrotik_interface_rx_rate_peak{interface,interval}` - Peak download rate
  - `mikrotik_interface_rx_rate_min{interface,interval}` - Minimum download rate
  - `mikrotik_interface_tx_rate_avg{interface,interval}` - Average upload rate
  - `mikrotik_interface_tx_rate_peak{interface,interval}` - Peak upload rate
  - `mikrotik_interface_tx_rate_min{interface,interval}` - Minimum upload rate
  - `mikrotik_interface_sample_count{interface,interval}` - Number of samples

## API Query Format

The program uses the following Mikrotik API command format:
```
/interface/print
=stats
=.proplist=name,rx-byte,tx-byte
?name=vlan2622
?name=vlan2624
?#|
?name=vlan2626
?#|
```

**Explanation:**
- `=stats` - Get real-time statistics (live counters)
- `=.proplist=name,rx-byte,tx-byte` - Only return these properties
- `?name=<interface>` - Filter by interface name
- `?#|` - OR operator placed **after each interface starting from the second one**

**Important:** The OR operator `?#|` must be placed **after each interface starting from the second one**. This allows the query to match interface1 OR interface2 OR interface3, etc.

Format pattern:
- 1 interface: `?name=iface1`
- 2 interfaces: `?name=iface1 ?name=iface2 ?#|`
- 3 interfaces: `?name=iface1 ?name=iface2 ?#| ?name=iface3 ?#|`

This filters results on the Mikrotik router before sending, reducing network traffic and processing time.

**Troubleshooting:** If you encounter issues with multiple interfaces, enable debug mode by setting `DEBUG=true` in your .env file. This will print the actual API commands being sent to help diagnose the problem.

## License

MIT
