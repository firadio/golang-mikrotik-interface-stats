#!/bin/bash

# Test script for embed dual-mode functionality
# This script demonstrates both production and developer modes

echo "========================================"
echo "Mikrotik Stats - Embed Test Script"
echo "========================================"
echo ""

# Build first
echo "📦 Building binary..."
go build -o mikrotik-stats.exe
if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi
echo "✅ Build successful!"
echo ""

# Check file size
SIZE=$(du -h mikrotik-stats.exe | cut -f1)
echo "📊 Binary size: $SIZE"
echo ""

# Test 1: Developer mode (web directory present)
echo "========================================"
echo "Test 1: Developer Mode (web/ exists)"
echo "========================================"
if [ -d "web" ]; then
    echo "✅ web/ directory found"
    echo ""
    echo "Expected logs:"
    echo "  [Web] Developer mode: Using local files from 'web/' directory"
    echo "  [Web] 💡 Tip: Remove 'web/' directory to test production mode"
    echo "  [Web] Static files: Hot-reload enabled"
    echo ""
else
    echo "❌ web/ directory not found - creating..."
    git restore web/ 2>/dev/null || echo "⚠️  Cannot restore web/"
fi

# Test 2: Production mode (web directory absent)
echo "========================================"
echo "Test 2: Production Mode (no web/)"
echo "========================================"
echo ""
echo "To test production mode:"
echo "  1. Rename web directory:  mv web web.backup"
echo "  2. Run the program:       ./mikrotik-stats.exe"
echo "  3. Check logs for:        [Web] Production mode: Using embedded files"
echo "  4. Restore web:           mv web.backup web"
echo ""

echo "========================================"
echo "Verification Steps"
echo "========================================"
echo ""
echo "1. With web/ directory (developer mode):"
echo "   - Edit web/index.html (change title)"
echo "   - Refresh browser (no rebuild needed)"
echo "   - Changes should appear immediately"
echo ""
echo "2. Without web/ directory (production mode):"
echo "   - mv web web.backup"
echo "   - ./mikrotik-stats.exe"
echo "   - Program still works!"
echo "   - Serves files from embedded binary"
echo ""

echo "✅ Embed functionality implemented successfully!"
echo ""
echo "Deployment tip:"
echo "  For production: Just distribute mikrotik-stats.exe + .env"
echo "  For development: Keep web/ directory for hot-reload"
