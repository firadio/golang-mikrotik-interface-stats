# Mikrotik Interface Stats

Monitor Mikrotik interface traffic statistics in real-time.

## Features

- ✅ Connect to Mikrotik API
- ✅ Monitor specific VLAN interfaces (vlan2622, vlan2624)
- ✅ Calculate per-second traffic rates (RX/TX)
- ✅ Human-readable output format (B/s, KB/s, MB/s, GB/s)
- ✅ Precise timing using time.Ticker (no missed seconds)
- ✅ Real-time monitoring with 1-second intervals
- 🔜 Export data to VictoriaMetrics (planned)

## Configuration

Create a `.env` file in the project root or set environment variables:

```env
MIKROTIK_HOST=175.100.109.154
MIKROTIK_PORT=65428
MIKROTIK_USERNAME=your_username
MIKROTIK_PASSWORD=your_password
```

See `.env.example` for reference.

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

## Output Example

```
2025/11/06 22:26:02 Connected to Mikrotik at 175.100.109.154:65428

Monitoring interface traffic (Ctrl+C to stop):
================================================================================
[2025-11-06 22:26:05] vlan2622: RX: 552.49 KB/s  TX: 332.49 KB/s
[2025-11-06 22:26:05] vlan2624: RX: 7.58 MB/s  TX: 2.42 MB/s
[2025-11-06 22:26:06] vlan2622: RX: 302.43 KB/s  TX: 160.64 KB/s
[2025-11-06 22:26:06] vlan2624: RX: 3.45 MB/s  TX: 1.17 MB/s
```

## Requirements

- Go 1.21 or later
- Access to Mikrotik Router with API enabled
- Valid Mikrotik credentials

## Implementation Details

- Uses Mikrotik API protocol directly (no external dependencies)
- Implements proper MD5 challenge-response authentication
- Server-side filtering using Mikrotik API query syntax (reduces network overhead)
- Stores previous byte counts to calculate delta per second
- Uses `time.Ticker` for accurate 1-second intervals
- Filters interfaces by name to monitor only vlan2622 and vlan2624

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
