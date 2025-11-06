# Deployment Guide

This guide explains how to build and deploy the Mikrotik Interface Monitor.

## Build for Production

### Standard Build (Recommended)

```bash
go build -o mikrotik-stats.exe
```

This creates a **single executable** with all static web files embedded inside.

**File size:** ~8.7 MB (includes all web assets)

### Deployment Files

For production deployment, you only need:
- ✅ `mikrotik-stats.exe` (the compiled binary)
- ✅ `.env` file (your configuration)

You do **NOT** need:
- ❌ `web/` directory (files are embedded in the exe)
- ❌ Go source code
- ❌ go.mod / go.sum

## Distribution

### Option 1: Binary Only (Simplest)

```bash
# On build machine
go build -o mikrotik-stats.exe

# Copy to production server (only these 2 files)
scp mikrotik-stats.exe user@server:/opt/mikrotik-stats/
scp .env user@server:/opt/mikrotik-stats/

# On production server
cd /opt/mikrotik-stats
chmod +x mikrotik-stats.exe
./mikrotik-stats.exe
```

### Option 2: Include Source (For Development)

If you want to develop/modify on the production server:

```bash
# Copy everything
scp -r . user@server:/opt/mikrotik-stats/

# On production server - the web/ directory will enable developer mode
cd /opt/mikrotik-stats
go build -o mikrotik-stats.exe
./mikrotik-stats.exe  # Will use local web/ files (hot-reload enabled)
```

## Developer vs Production Mode

The program automatically detects which mode to use:

### Production Mode (Embedded Files)

**Condition:** `web/` directory does NOT exist in the same directory as the executable

**Behavior:**
- Serves static files from embedded binary
- Faster (files in memory)
- Single-file distribution
- No hot-reload

**Logs:**
```
[Web] Production mode: Using embedded files from binary
[Web] Static files: Serving from embedded binary
```

### Developer Mode (Local Files)

**Condition:** `web/` directory EXISTS in the same directory as the executable

**Behavior:**
- Serves static files from disk
- Hot-reload enabled (changes take effect on browser refresh)
- Perfect for frontend development
- No rebuild needed for HTML/CSS/JS changes

**Logs:**
```
[Web] Developer mode: Using local files from 'web/' directory
[Web] 💡 Tip: Remove 'web/' directory to test production mode (embedded files)
[Web] Static files: Hot-reload enabled (changes take effect immediately)
```

## Testing Both Modes

### Test Production Mode

```bash
# Build first
go build -o mikrotik-stats.exe

# Remove/rename web directory
mv web web.backup

# Run - will use embedded files
./mikrotik-stats.exe
```

### Test Developer Mode

```bash
# Restore web directory
mv web.backup web

# Run - will use local files
./mikrotik-stats.exe

# Now you can edit web/index.html and just refresh browser!
```

## Building Optimized Binaries

### Smaller Binary Size

```bash
# Strip debug symbols and compress
go build -ldflags="-s -w" -o mikrotik-stats.exe

# Further compress with UPX (optional)
upx --best --lzma mikrotik-stats.exe
```

**Size comparison:**
- Normal build: ~8.7 MB
- With `-ldflags="-s -w"`: ~6.5 MB
- With UPX: ~2.5 MB

### Cross-compilation

Build for different platforms:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mikrotik-stats-linux

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o mikrotik-stats.exe

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o mikrotik-stats-macos

# Raspberry Pi (ARM)
GOOS=linux GOARCH=arm GOARM=7 go build -o mikrotik-stats-rpi
```

## Running as a Service

### systemd (Linux)

Create `/etc/systemd/system/mikrotik-stats.service`:

```ini
[Unit]
Description=Mikrotik Interface Monitor
After=network.target

[Service]
Type=simple
User=mikrotik
WorkingDirectory=/opt/mikrotik-stats
ExecStart=/opt/mikrotik-stats/mikrotik-stats.exe
Restart=on-failure
RestartSec=10s

# Environment variables (or use .env file)
Environment="MIKROTIK_HOST=192.168.1.1"
Environment="MIKROTIK_PORT=8728"
Environment="LOG_ENABLED=true"
Environment="LOG_OUTPUT=stdout"
Environment="WEB_ENABLED=true"

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable mikrotik-stats
sudo systemctl start mikrotik-stats
sudo systemctl status mikrotik-stats
```

### Docker (Optional)

Create `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o mikrotik-stats

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mikrotik-stats .
COPY .env .
EXPOSE 8080
CMD ["./mikrotik-stats"]
```

Build and run:
```bash
docker build -t mikrotik-stats .
docker run -d \
  --name mikrotik-stats \
  -p 8080:8080 \
  --env-file .env \
  --restart unless-stopped \
  mikrotik-stats
```

## Security Considerations

### File Permissions

```bash
# Restrict access to config file (contains password)
chmod 600 .env

# Executable permissions
chmod 755 mikrotik-stats.exe
```

### Firewall Rules

```bash
# Allow only specific IPs to access web interface
sudo ufw allow from 192.168.1.0/24 to any port 8080

# Or use nginx as reverse proxy with authentication
```

### Environment Variables

Instead of `.env` file, use environment variables:

```bash
# More secure for production
export MIKROTIK_HOST=192.168.1.1
export MIKROTIK_PASSWORD=$(cat /secure/password.txt)
./mikrotik-stats.exe
```

## Troubleshooting

### Web Interface 404

**Symptom:** Browser shows 404 for all pages

**Solution 1:** Check if web/ directory exists and logs show correct mode
```bash
ls -la web/
# Should show web/index.html, web/static/, etc.
```

**Solution 2:** Rebuild with embed
```bash
# Ensure web/ directory exists during build
go build -o mikrotik-stats.exe
# Now you can delete web/ directory
```

### Files Not Updating

**Symptom:** Changes to HTML/CSS/JS don't appear

**Developer mode:** Clear browser cache (Ctrl+F5)

**Production mode:** You need to rebuild!
```bash
go build -o mikrotik-stats.exe
# Embedded files only update on recompile
```

## Version Management

Tag releases for easy deployment:

```bash
# Create release
git tag v0.0.1
git push origin v0.0.1

# Build with version info
go build -ldflags="-X main.Version=v0.0.1" -o mikrotik-stats.exe
```

## Backup and Recovery

### Backup

```bash
# Backup only what's needed
tar czf mikrotik-stats-backup.tar.gz \
  mikrotik-stats.exe \
  .env
```

### Restore

```bash
tar xzf mikrotik-stats-backup.tar.gz
./mikrotik-stats.exe
```

## Summary

**For production:**
- ✅ Single binary deployment (`mikrotik-stats.exe` + `.env`)
- ✅ No web/ directory needed
- ✅ Fast (memory-based file serving)

**For development:**
- ✅ Keep web/ directory present
- ✅ Hot-reload enabled
- ✅ No rebuild needed for frontend changes
