@echo off
echo Starting debug instance on port 9999...
echo Press Ctrl+C to stop
mikrotik-stats.exe --env=.env.debug
