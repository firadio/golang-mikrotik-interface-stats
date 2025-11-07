# Test history query API - Comparing Average Peak vs Burst Peak
Write-Host "Testing NEW statistics logic (avg_over_time -> max_over_time)" -ForegroundColor Cyan
Write-Host ""

$interface = "vlan2622"
$endTime = [int][double]::Parse((Get-Date -UFormat %s))
$startTime = $endTime - 3600  # 1 hour ago
$interval = "5m"

Write-Host "Parameters:" -ForegroundColor Yellow
Write-Host "  Interface: $interface"
Write-Host "  Start: $startTime ($(Get-Date -UnixTimeSeconds $startTime))"
Write-Host "  End: $endTime ($(Get-Date -UnixTimeSeconds $endTime))"
Write-Host "  Interval: $interval"
Write-Host ""

$url = "http://localhost:9999/api/history?interface=$interface&start=$startTime&end=$endTime&interval=$interval"

Write-Host "Calling: $url" -ForegroundColor Green
Write-Host ""

try {
    $response = Invoke-WebRequest -Uri $url -UseBasicParsing
    $json = $response.Content | ConvertFrom-Json

    Write-Host "Response received:" -ForegroundColor Green
    Write-Host "  Interface: $($json.interface)"
    Write-Host "  Interval: $($json.interval)"
    Write-Host "  Datapoints: $($json.datapoints.Count)"
    Write-Host ""

    if ($json.stats) {
        Write-Host "=== NEW Statistics (max_over_time for both) ===" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Download Statistics:" -ForegroundColor Cyan
        $dlAvgMbps = [math]::Round($json.stats.download_avg * 8 / 1000000, 2)
        $dlPeakMbps = [math]::Round($json.stats.download_peak * 8 / 1000000, 2)
        Write-Host "  Average Peak (sustained): $dlAvgMbps Mbps" -ForegroundColor Green
        Write-Host "  Burst Peak (instantaneous): $dlPeakMbps Mbps" -ForegroundColor Red
        Write-Host "  Difference: $([math]::Round($dlPeakMbps - $dlAvgMbps, 2)) Mbps ($([math]::Round(($dlPeakMbps - $dlAvgMbps) / $dlAvgMbps * 100, 1))%)" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Upload Statistics:" -ForegroundColor Cyan
        $ulAvgMbps = [math]::Round($json.stats.upload_avg * 8 / 1000000, 2)
        $ulPeakMbps = [math]::Round($json.stats.upload_peak * 8 / 1000000, 2)
        Write-Host "  Average Peak (sustained): $ulAvgMbps Mbps" -ForegroundColor Green
        Write-Host "  Burst Peak (instantaneous): $ulPeakMbps Mbps" -ForegroundColor Red
        Write-Host "  Difference: $([math]::Round($ulPeakMbps - $ulAvgMbps, 2)) Mbps ($([math]::Round(($ulPeakMbps - $ulAvgMbps) / $ulAvgMbps * 100, 1))%)" -ForegroundColor Yellow
    }

} catch {
    Write-Host "Error: $_" -ForegroundColor Red
    Write-Host $_.Exception.Message
}

Write-Host ""
