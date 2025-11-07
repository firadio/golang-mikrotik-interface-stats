# Test script to verify the interval fix

Write-Host "Testing interval/step fix..." -ForegroundColor Yellow
Write-Host ""

# Test with 5m interval (should return ~13 points)
Write-Host "Test 1: 5m interval (expect ~13 points)" -ForegroundColor Cyan
$response = curl.exe -s "http://localhost:9999/api/history?interface=vlan2622&start=1762499300&end=1762502900&interval=5m"
$json = $response | ConvertFrom-Json
$pointCount = $json.datapoints.Count
Write-Host "  Returned: $pointCount points" -ForegroundColor $(if ($pointCount -lt 20) { "Green" } else { "Red" })
Write-Host ""

# Test with 1m interval (should return ~61 points)
Write-Host "Test 2: 1m interval (expect ~61 points)" -ForegroundColor Cyan
$response = curl.exe -s "http://localhost:9999/api/history?interface=vlan2622&start=1762499300&end=1762502900&interval=1m"
$json = $response | ConvertFrom-Json
$pointCount = $json.datapoints.Count
Write-Host "  Returned: $pointCount points" -ForegroundColor $(if ($pointCount -gt 50 -and $pointCount -lt 70) { "Green" } else { "Red" })
Write-Host ""

# Test with 10s interval (should return ~361 points)
Write-Host "Test 3: 10s interval (expect ~361 points)" -ForegroundColor Cyan
$response = curl.exe -s "http://localhost:9999/api/history?interface=vlan2622&start=1762499300&end=1762502900&interval=10s"
$json = $response | ConvertFrom-Json
$pointCount = $json.datapoints.Count
Write-Host "  Returned: $pointCount points" -ForegroundColor $(if ($pointCount -gt 350) { "Green" } else { "Red" })
