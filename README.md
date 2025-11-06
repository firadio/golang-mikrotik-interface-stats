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
- 🔜 Export data to VictoriaMetrics (planned)

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

## Output Examples

**Refresh Mode (default) with auto scale:**
```
Mikrotik Interface Traffic Monitor
================================================================================
Time: 2025-11-07 01:08:36
--------------------------------------------------------------------------------
Interface       Upload               Download
--------------------------------------------------------------------------------
vlan2622        30.04 KB/s           26.43 KB/s
vlan2624        580.37 KB/s          3.19 MB/s
--------------------------------------------------------------------------------
Press Ctrl+C to stop
```

Note: Display shows "Upload" (left) and "Download" (right) from user perspective. If an interface is configured as uplink, RX/TX are swapped automatically.

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
├── main.go              # Program entry point
├── config.go            # Configuration loading
├── client.go            # Mikrotik API client
├── stats.go             # Statistics data structures and formatting
├── monitor.go           # Monitoring logic
├── output.go            # Output abstraction (terminal/log modes)
├── terminal_windows.go  # Windows ANSI support (build tag: windows)
├── terminal_unix.go     # Unix ANSI stub (build tag: !windows)
├── go.mod               # Go module configuration
├── .env                 # Environment configuration
└── README.md            # Documentation
```

## Implementation Details

- Uses Mikrotik API protocol directly (no external dependencies)
- Implements proper MD5 challenge-response authentication
- Server-side filtering using Mikrotik API query syntax (reduces network overhead)
- Stores previous byte counts to calculate delta per second
- Uses `time.Ticker` for accurate 1-second intervals
- Configurable interface list via environment variables
- **Automatically enables ANSI support on Windows** via Windows API (no manual setup needed)
- Modular output system with OutputWriter interface:
  - TerminalOutput: Interactive display (refresh/append modes)
  - LogOutput: Service-friendly structured logging
- Configurable rate units (bits vs bytes) and scales (auto/fixed)
- Fixed-scale formatting with decimal alignment for easy reading
- **Efficient cursor control**: Uses ANSI escape sequences to move cursor instead of clearing screen
  - Reduces flicker and improves visual stability
  - Interfaces always displayed in alphabetical order (no jumping)
  - `\033[H` - move cursor to home position
  - `\033[J` - clear from cursor to end of screen

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
