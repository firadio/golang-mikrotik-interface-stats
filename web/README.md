# Web Interface

This directory contains the static files for the web monitoring interface.

## 📦 Embedded vs Local Files

The web interface supports **dual-mode operation**:

### Production Mode (Embedded)
- Static files are **embedded into the binary** at compile time
- **Single executable** - no need to distribute the `web/` directory
- Files served directly from memory (faster)
- To use: Simply **delete or move the `web/` directory** before running

### Developer Mode (Local Files)
- If `web/` directory exists at runtime, files are loaded from disk
- **Hot-reload enabled** - changes take effect immediately (just refresh browser)
- Perfect for frontend development and testing
- To use: Keep the `web/` directory in the same location as the executable

**The program automatically detects which mode to use at startup.**

## Directory Structure

```
web/
├── index.html              # Main HTML page
└── static/
    ├── css/
    │   └── style.css       # Stylesheet
    └── js/
        └── app.js          # WebSocket client and UI logic
```

## Features

- **Real-time line charts** with Chart.js (60-second history)
- **WebSocket live updates** (`/api/realtime`) with auto-reconnect
- **Modern card-based UI** inspired by mainstream monitoring tools
- **Animated status indicator** with pulse effect
- **Current/Average/Peak statistics** in separate metric cards
- **Color scheme**:
  - Upload: Red (#ef4444)
  - Download: Green (#10b981)
  - Background: Slate dark theme
  - Charts: Smooth gradient fills
- **Responsive design** adapts to mobile and desktop
- **Hover tooltips** on charts showing exact values

## API Endpoints

### WebSocket Real-time Push
- **Endpoint**: `ws://localhost:8080/api/realtime`
- **Protocol**: WebSocket
- **Data format**:
```json
{
  "timestamp": "2025-11-07T12:34:56Z",
  "interfaces": {
    "vlan2622": {
      "upload_rate": 1234567.89,
      "download_rate": 9876543.21,
      "upload_avg": 1000000.00,
      "download_avg": 8000000.00,
      "upload_peak": 2000000.00,
      "download_peak": 10000000.00,
      "upload_mbps": 9.88,
      "download_mbps": 79.01
    }
  }
}
```

### REST API - Current Stats
- **Endpoint**: `GET /api/current`
- **Protocol**: HTTP
- **Response**: Same JSON format as WebSocket

## Configuration

Enable web server in `.env`:
```env
WEB_ENABLED=true
WEB_LISTEN_ADDR=:8080
WEB_ENABLE_REALTIME=true   # WebSocket real-time push
WEB_ENABLE_API=true        # REST API
WEB_ENABLE_STATIC=true     # Static file serving
```

## Chart Configuration

The real-time charts are powered by Chart.js 4.4.0 and configured in `app.js`:

- **Data retention**: 60 seconds (configurable via `MAX_DATA_POINTS`)
- **Update rate**: Every second via WebSocket
- **Chart type**: Line chart with smooth curves (tension: 0.4)
- **Y-axis**: Auto-scales in Mbps
- **X-axis**: Time labels (HH:MM:SS format)
- **Animation**: Disabled for smooth real-time updates

### Customizing Charts

Edit `app.js` to modify chart behavior:

```javascript
// Change data retention
const MAX_DATA_POINTS = 120; // Show last 2 minutes

// Change colors
const CHART_COLORS = {
    upload: 'rgb(239, 68, 68)',    // Red
    download: 'rgb(16, 185, 129)', // Green
    // ... customize as needed
};
```

## Development Workflow

### Quick Start

1. **Keep `web/` directory present** during development
2. **Start the program** - it will detect developer mode automatically
3. **Edit files** in `web/` directory
4. **Refresh browser** - changes appear immediately (no rebuild needed!)

Example log output in developer mode:
```
[Web] Developer mode: Using local files from 'web/' directory
[Web] 💡 Tip: Remove 'web/' directory to test production mode (embedded files)
[Web] Static files: Hot-reload enabled (changes take effect immediately)
```

### Building for Production

```bash
# Compile with embedded files
go build -o mikrotik-stats.exe

# Distribute only the exe (web/ directory not needed)
# Users can delete/move web/ directory - the program will work from embedded files
```

### Adding New Features

1. **Add CSS**: Edit `static/css/style.css` (uses CSS custom properties for easy theming)
2. **Add JavaScript**: Edit `static/js/app.js` or create new files
3. **Modify HTML**: Edit `index.html`
4. **Test changes**: Just refresh browser (no rebuild needed in developer mode)

### Adding Icons/Images

Create `static/images/` directory and add your assets:
```bash
mkdir web/static/images
# Add your images to web/static/images/
```

Reference in HTML:
```html
<img src="/static/images/logo.png" alt="Logo">
```

### Adding Third-party Libraries

You can add external libraries (e.g., Chart.js for graphs):

**Option 1: CDN (Recommended for development)**
```html
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
```

**Option 2: Local files (Recommended for production)**
```bash
# Download library to web/static/js/
curl -o web/static/js/chart.min.js https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js
```

Then reference in HTML:
```html
<script src="/static/js/chart.min.js"></script>
```

## Future Enhancements

- [ ] Historical data charts (query VictoriaMetrics)
- [ ] Drill-down from 5-minute to 10-second data
- [ ] Configurable time ranges (1h, 6h, 24h, 7d)
- [ ] Export data as CSV/JSON
- [ ] Dark/Light theme toggle
- [ ] Multiple interface grouping/tabs
