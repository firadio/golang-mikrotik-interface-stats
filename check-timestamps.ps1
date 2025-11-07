# Check timestamp alignment
$response = curl.exe -s "http://localhost:9999/api/history?interface=vlan2622&start=1762499300&end=1762502900&interval=5m"
$json = $response | ConvertFrom-Json

Write-Host "Checking timestamp alignment for 5m interval:" -ForegroundColor Yellow
Write-Host "Start: 1762499300 ($(([DateTimeOffset]::FromUnixTimeSeconds(1762499300).DateTime)))" -ForegroundColor Cyan
Write-Host "End:   1762502900 ($(([DateTimeOffset]::FromUnixTimeSeconds(1762502900).DateTime)))" -ForegroundColor Cyan
Write-Host ""

Write-Host "First 5 data points:" -ForegroundColor Green
for ($i = 0; $i -lt [Math]::Min(5, $json.datapoints.Count); $i++) {
    $ts = $json.datapoints[$i].timestamp
    $dt = [DateTime]::Parse($ts)
    $unix = [long]([DateTimeOffset]$dt).ToUnixTimeSeconds()
    Write-Host "  $i : $ts (unix: $unix)"
}

Write-Host ""
Write-Host "Last 5 data points:" -ForegroundColor Green
$start = [Math]::Max(0, $json.datapoints.Count - 5)
for ($i = $start; $i -lt $json.datapoints.Count; $i++) {
    $ts = $json.datapoints[$i].timestamp
    $dt = [DateTime]::Parse($ts)
    $unix = [long]([DateTimeOffset]$dt).ToUnixTimeSeconds()
    Write-Host "  $i : $ts (unix: $unix)"
}

Write-Host ""
Write-Host "Checking 5-minute alignment:" -ForegroundColor Yellow
$aligned = $true
foreach ($dp in $json.datapoints) {
    $dt = [DateTime]::Parse($dp.timestamp)
    $unix = [long]([DateTimeOffset]$dt).ToUnixTimeSeconds()
    if (($unix % 300) -ne 0) {
        Write-Host "  NOT ALIGNED: $($dp.timestamp) (unix: $unix, remainder: $($unix % 300))" -ForegroundColor Red
        $aligned = $false
    }
}

if ($aligned) {
    Write-Host "  All timestamps are aligned to 5-minute boundaries!" -ForegroundColor Green
}
