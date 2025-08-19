# Produce sample order to Kafka and query API

$ErrorActionPreference = 'Stop'

$sample = "sample_order.json"
if (-not (Test-Path $sample)) {
  Write-Error "sample_order.json not found"
}

Write-Host "Copying sample to redpanda..." -ForegroundColor Yellow
docker cp sample_order.json testtask-redpanda-1:/tmp/sample_order.json | Out-Null

Write-Host "Producing to topic 'orders'..." -ForegroundColor Yellow
docker exec testtask-redpanda-1 sh -lc "tr -d '\n' </tmp/sample_order.json | rpk topic produce orders" | Write-Host

$uid = (Get-Content -Raw sample_order.json | ConvertFrom-Json).order_uid
Write-Host "Querying API for $uid ..." -ForegroundColor Yellow
$resp = Invoke-WebRequest -Uri ("http://localhost:8081/order/" + $uid) -Method GET -UseBasicParsing
Write-Host ("Status: {0}  X-Cache: {1}" -f $resp.StatusCode, $resp.Headers['X-Cache']) -ForegroundColor Green
$resp.Content | Write-Host


