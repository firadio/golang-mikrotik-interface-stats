@echo off
echo Testing history query API...
echo.
echo Using interface: vlan2622
echo Time range: Last 24 hours
echo Interval: 30m
echo.

set /a "END=%date:~0,4%%date:~5,2%%date:~8,2%"
set /a "START=%END%-1"

REM Get current timestamp (approximate)
for /f %%i in ('powershell -Command "[int][double]::Parse((Get-Date -UFormat %%s))"') do set END_TS=%%i
set /a "START_TS=%END_TS%-86400"

echo Start timestamp: %START_TS%
echo End timestamp: %END_TS%
echo.
echo Calling API...
echo.

curl "http://localhost:9999/api/history?interface=vlan2622&start=%START_TS%&end=%END_TS%&interval=30m"

echo.
echo.
echo Done!
pause
