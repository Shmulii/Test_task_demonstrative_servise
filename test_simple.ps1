# Simple API test
Write-Host "Testing API..." -ForegroundColor Green

try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/orders/b563feb7b2b84b6test" -Method GET -UseBasicParsing
    Write-Host "Status: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "Response:" -ForegroundColor Cyan
    $response.Content
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Status Code: $($_.Exception.Response.StatusCode)" -ForegroundColor Red
}

Write-Host "`nWeb interface: http://localhost:8080/" -ForegroundColor Yellow
Write-Host "Enter 'b563feb7b2b84b6test' in the input field and click 'Get'" -ForegroundColor Yellow
