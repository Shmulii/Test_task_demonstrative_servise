# Test script to get order data
Write-Host "Testing Orders Service API..." -ForegroundColor Green

# Test 1: Get order via API
Write-Host "`n1. Testing API endpoint:" -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/orders/b563feb7b2b84b6test" -Method GET
    Write-Host "Status: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "Response:" -ForegroundColor Cyan
    $response.Content | ConvertFrom-Json | ConvertTo-Json -Depth 10
} catch {
    Write-Host "API Error: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 2: Get order directly from database
Write-Host "`n2. Getting order from database:" -ForegroundColor Yellow
$dbQuery = @"
SELECT 
    o.order_uid,
    o.track_number,
    o.customer_id,
    o.date_created,
    d.name as delivery_name,
    d.city as delivery_city,
    p.amount as payment_amount,
    p.currency as payment_currency
FROM orders o
LEFT JOIN deliveries d ON o.order_uid = d.order_uid
LEFT JOIN payments p ON o.order_uid = p.order_uid
WHERE o.order_uid = 'b563feb7b2b84b6test';
"@

$dbResult = docker exec testtask-postgres-1 psql -U orders_user -d orders_db -t -c $dbQuery
Write-Host "Database result:" -ForegroundColor Cyan
$dbResult

# Test 3: Get order items
Write-Host "`n3. Getting order items:" -ForegroundColor Yellow
$itemsQuery = "SELECT name, price, brand, status FROM items WHERE order_uid = 'b563feb7b2b84b6test';"
$itemsResult = docker exec testtask-postgres-1 psql -U orders_user -d orders_db -t -c $itemsQuery
Write-Host "Items:" -ForegroundColor Cyan
$itemsResult

Write-Host "`nAPI is available at: http://localhost:8080/" -ForegroundColor Green
Write-Host "Order endpoint: http://localhost:8080/orders/{order_uid}" -ForegroundColor Green
