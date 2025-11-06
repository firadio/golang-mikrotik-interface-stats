# Web Interface Features

## Overview

The web interface provides a modern, real-time monitoring experience with live charts and statistics, inspired by mainstream network monitoring tools.

## Main Features

### 1. Real-time Line Charts

Each interface displays a smooth line chart showing:
- **Upload traffic** (Red line) - Traffic sent FROM the interface
- **Download traffic** (Green line) - Traffic received BY the interface
- **60-second rolling window** - Shows the last minute of activity
- **Smooth curves** - Uses cubic Bézier interpolation for visual appeal
- **Gradient fills** - Subtle background gradients under each line
- **Auto-scaling Y-axis** - Automatically adjusts scale based on traffic volume

**Interaction:**
- Hover over chart to see exact values at any time point
- Tooltip shows both upload and download rates simultaneously
- Time labels on X-axis (HH:MM:SS format)
- Values displayed in Mbps on Y-axis

### 2. Live Statistics Badges

At the top of each interface card:
- **↑ Upload** - Current upload rate (updated every second)
- **↓ Download** - Current download rate (updated every second)
- Color-coded: Upload in red, Download in green

### 3. Statistical Metrics

Below the chart, four metric cards display:
- **Average Upload** - Mean upload rate over the stats window (default 10s)
- **Average Download** - Mean download rate over the stats window
- **Peak Upload** - Maximum upload rate recorded in the window
- **Peak Download** - Maximum download rate recorded in the window

### 4. Connection Status

Top-right corner shows connection state:
- **Green dot (Connected)** - WebSocket connection active
- **Red dot (Reconnecting)** - Connection lost, attempting reconnect
- Animated pulse effect on status indicator

### 5. Responsive Design

The interface adapts to different screen sizes:
- **Desktop (>1200px)**: 2-column grid layout
- **Tablet (768px-1200px)**: Single column
- **Mobile (<768px)**: Optimized for touch, vertical metrics

## Technical Details

### Chart Configuration

```javascript
MAX_DATA_POINTS = 60        // 60 data points = 60 seconds
Update frequency = 1 second // WebSocket pushes every second
Chart type = Line           // With area fill
Tension = 0.4              // Curve smoothness (0 = straight lines, 1 = very curved)
```

### Color Scheme

The interface uses a carefully selected color palette:

**Background:**
- Primary: `#0f172a` (Slate 900)
- Secondary: `#1e293b` (Slate 800)
- Card: `#334155` (Slate 700)

**Traffic:**
- Upload: `#ef4444` (Red 500) - Represents data sent OUT
- Download: `#10b981` (Green 500) - Represents data received IN

**Text:**
- Primary: `#f1f5f9` (Slate 100)
- Secondary: `#94a3b8` (Slate 400)

### Data Flow

```
Mikrotik API → Go Backend → WebSocket → Browser
                   ↓
              Stats Cache (for new connections)
                   ↓
              Chart.js Rendering
```

1. Backend queries Mikrotik every second
2. Stats calculated (current, avg, peak)
3. WebSocket broadcasts to all connected clients
4. JavaScript updates chart and metrics
5. Chart.js renders smooth animations

## Performance

- **Lightweight**: Chart.js is loaded from CDN (no heavy dependencies)
- **Efficient updates**: Only changed data is redrawn
- **Smooth animations**: 60 FPS rendering with hardware acceleration
- **Low bandwidth**: ~500 bytes per update (JSON)
- **Responsive**: Updates appear within 100ms of backend calculation

## Browser Compatibility

Tested and working on:
- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Opera 76+

Requires:
- WebSocket support
- Canvas 2D context
- ES6 JavaScript (arrow functions, template literals, etc.)

## Keyboard Shortcuts

None implemented yet. Future enhancements could include:
- `F` - Toggle fullscreen
- `P` - Pause/resume updates
- `R` - Reset chart view
- `1-9` - Switch between interfaces

## Future Enhancements

Planned features:
- [ ] Historical data view (query VictoriaMetrics)
- [ ] Time range selector (1h, 6h, 24h, 7d, 30d)
- [ ] Drill-down from 5-minute averages to 10-second detail
- [ ] CSV/JSON export
- [ ] Dark/Light theme toggle
- [ ] Interface grouping/tabs
- [ ] Bandwidth alerts/notifications
- [ ] Comparison view (overlay multiple interfaces)
- [ ] Mobile app (PWA)
