$response = curl.exe -s "http://localhost:9999/api/history?interface=vlan2622&start=1762499300&end=1762502900&interval=5m"
$json = $response | ConvertFrom-Json

Write-Host "Total points: $($json.datapoints.Count)" -ForegroundColor Yellow
Write-Host ""
Write-Host "Timestamps:" -ForegroundColor Cyan

foreach ($dp in $json.datapoints) {
    Write-Host $dp.timestamp
}
