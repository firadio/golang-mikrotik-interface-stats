# Mikrotik Interface Stats

Monitor Mikrotik interface traffic statistics in real-time.

## Features

- ✅ Connect to Mikrotik API
- ✅ Configurable interface list via .env
- ✅ Calculate per-second traffic rates (RX/TX)
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

# Display mode (optional)
DISPLAY_MODE=refresh  # or "append"

# Rate unit (optional)
RATE_UNIT=auto  # or "bps" (bits/s), "Bps" (Bytes/s)

# Rate scale (optional)
RATE_SCALE=auto  # or "k", "M", "G" for fixed scale

# Output mode (optional)
OUTPUT_MODE=terminal  # or "log"
```

**Configuration Options:**

- **INTERFACES**: Comma-separated list of interfaces to monitor (default: vlan2622,vlan2624)

- **DISPLAY_MODE**: How to display output
  - `refresh` (default) - Clear screen and redraw like `top`/`htop`
    - Uses ANSI escape codes for screen clearing
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
Interface       RX Rate              TX Rate
--------------------------------------------------------------------------------
vlan2622        26.43 KB/s           30.04 KB/s
vlan2624        3.19 MB/s            580.37 KB/s
--------------------------------------------------------------------------------
Press Ctrl+C to stop
```

**Append Mode:**
```
2025/11/07 01:09:12 Connected to Mikrotik at 175.100.109.154:65428

Monitoring interface traffic (Ctrl+C to stop):
================================================================================
[2025-11-07 01:09:13] vlan2622: RX: 15.82 KB/s  TX: 18.12 KB/s
[2025-11-07 01:09:13] vlan2624: RX: 3.00 MB/s  TX: 655.47 KB/s
[2025-11-07 01:09:14] vlan2622: RX: 16.60 KB/s  TX: 50.48 KB/s
[2025-11-07 01:09:14] vlan2624: RX: 3.64 MB/s  TX: 431.54 KB/s
```

**Fixed scale mode (RATE_SCALE=M):**
```
Mikrotik Interface Traffic Monitor
================================================================================
Time: 2025-11-07 01:08:36
--------------------------------------------------------------------------------
Interface       RX Rate              TX Rate
--------------------------------------------------------------------------------
vlan2622          0.03 MB/s            0.03 MB/s
vlan2624          3.19 MB/s            0.58 MB/s
--------------------------------------------------------------------------------
Press Ctrl+C to stop
```

**Log mode (OUTPUT_MODE=log):**
```
2025/11/07 01:09:12 Connected to Mikrotik at 175.100.109.154:65428
2025/11/07 01:09:12 Mikrotik Interface Traffic Monitor started
2025/11/07 01:09:13 interface=vlan2622 rx=15.82 KB/s tx=18.12 KB/s
2025/11/07 01:09:13 interface=vlan2624 rx=3.00 MB/s tx=655.47 KB/s
2025/11/07 01:09:14 interface=vlan2622 rx=16.60 KB/s tx=50.48 KB/s
2025/11/07 01:09:14 interface=vlan2624 rx=3.64 MB/s tx=431.54 KB/s
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

## API Query Format

The program uses the following Mikrotik API command format:
```
/interface/print
=stats
=.proplist=name,rx-byte,tx-byte
?name=vlan2622
?name=vlan2624
?#|
```

**Explanation:**
- `=stats` - Get real-time statistics (live counters)
- `=.proplist=name,rx-byte,tx-byte` - Only return these properties
- `?name=vlan2622` / `?name=vlan2624` - Filter by interface name
- `?#|` - OR operator (match either interface)

This filters results on the Mikrotik router before sending, reducing network traffic and processing time.

## License

MIT
