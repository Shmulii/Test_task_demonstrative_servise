# Simple cache performance check: first request (MISS) vs repeated (HIT)

$ErrorActionPreference = 'Stop'

$uid = (Get-Content -Raw sample_order.json | ConvertFrom-Json).order_uid

Write-Host "First request (expected MISS)..." -ForegroundColor Yellow
$t1 = Measure-Command { Invoke-WebRequest -Uri ("http://localhost:8081/order/" + $uid) -Method GET -UseBasicParsing }
Write-Host ("Time: {0} ms" -f [int]$t1.TotalMilliseconds) -ForegroundColor Green

Write-Host "Second request (expected HIT)..." -ForegroundColor Yellow
$t2 = Measure-Command { Invoke-WebRequest -Uri ("http://localhost:8081/order/" + $uid) -Method GET -UseBasicParsing }
Write-Host ("Time: {0} ms" -f [int]$t2.TotalMilliseconds) -ForegroundColor Green

if ($t2.TotalMilliseconds -lt $t1.TotalMilliseconds) {
  Write-Host "Cache appears faster on repeat request." -ForegroundColor Green
} else {
  Write-Host "No speedup observed; check logs and DB connectivity." -ForegroundColor Yellow
}


