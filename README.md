## Orders Service (Test Task)

Golang microservice that ingests orders from Kafka, persists them in Postgres, serves them over HTTP, and caches recent orders in-memory. Comes with Docker Compose to run Postgres and Redpanda (Kafka-compatible) locally.

### Features
- Kafka consumer (at-least-once) parsing JSON `Order`
- Postgres persistence with transactional upsert of aggregate (`orders`, `deliveries`, `payments`, `items`)
- HTTP API with read-through in-memory cache
- Static landing page
- Graceful shutdown

## Architecture
- `main.go`: wiring and lifecycle (startup, warmup, HTTP, consumer, shutdown)
- `config.go`: configuration (env + defaults)
- `models.go`: data structures (`Order`, `Delivery`, `Payment`, `Item`)
- `db.go`: Postgres pool + save/load logic
- `consumer.go`: Kafka reader loop (Segment `kafka-go`)
- `http.go`: HTTP server and routes (Gorilla Mux)
- `cashe.go`: simple concurrent cache with capacity limit
- `migrations/001_create_tables.sql`: database schema
- `static/`: static assets (index page)

## Prerequisites
- Docker Desktop (recommended) OR
- Go 1.20+, Postgres 15+, and Kafka (or Redpanda)

## Quick start (Docker)
This runs Postgres, Redpanda, and the service.

```powershell
cd "C:\Users\Василий\go-projects\Test task"
docker compose up --build -d
```

- Service listens on container port 8080, exposed on host port 8081 (see `docker-compose.yml`).
- Open UI: `http://localhost:8081/`

### Apply database schema (first run)
The service does not auto-migrate. Load the SQL into the Postgres container:

```powershell
docker cp migrations/001_create_tables.sql testtask-postgres-1:/tmp/001.sql
docker exec -i testtask-postgres-1 psql -U orders_user -d orders_db -f /tmp/001.sql | cat
```

### Produce sample order and query API
`sample_order.json` contains a valid `Order`. Produce it to the `orders` topic and query it.

```powershell
# Copy sample to Redpanda container and produce as single-line JSON
docker cp sample_order.json testtask-redpanda-1:/tmp/sample_order.json
docker exec testtask-redpanda-1 sh -lc "tr -d '\n' </tmp/sample_order.json | rpk topic produce orders"

# Fetch via API (PowerShell)
$uid = (Get-Content -Raw sample_order.json | ConvertFrom-Json).order_uid
Invoke-RestMethod -Method GET -Uri ("http://localhost:8081/orders/" + $uid) | ConvertTo-Json -Depth 6
```

### Useful commands
```powershell
# Status / Logs
docker compose ps
docker compose logs --no-color --tail=200 orders-service

# Stop / Remove
docker compose down
```

## Run locally without Docker
You need Postgres and Kafka running locally, and the schema applied.

1) Set configuration (defaults shown):
```powershell
$env:POSTGRES_DSN = "postgres://orders_user:orders_pass@localhost:5432/orders_db?sslmode=disable"
$env:KAFKA_BROKER = "localhost:9092"
```

2) Build and run:
```powershell
go build -o orders-service .
./orders-service
# or: go run .
```

HTTP will listen on `:8080` by default. Open `http://localhost:8080/`.

## API
- `GET /orders/{order_uid}` → returns `Order` as JSON
- `GET /` → serves `./static/index.html`
- `GET /static/...` → static assets

Responses are JSON, `Content-Type: application/json`.

## Configuration
Environment variables (with defaults):
- `POSTGRES_DSN` = `postgres://orders_user:orders_pass@localhost:5432/orders_db?sslmode=disable`
- `KAFKA_BROKER` = `localhost:9092`
- `KAFKA_TOPIC` = `orders` (fixed in code)
- `HTTP_ADDR` = `:8080` (configured in code)
- `CACHE_LIMIT` = `1000` (configured in code)
- `STARTUP_LOAD` = `100` (configured in code)

Note: values other than `POSTGRES_DSN` and `KAFKA_BROKER` are set inside `DefaultConfig()`.

## Cache behavior
- On startup, the service loads the latest `STARTUP_LOAD` orders into cache.
- `GET /orders/{id}` reads from cache first, otherwise fetches from DB and caches the result.
- Cache has a hard capacity; inserts are dropped when full (no eviction).

## Kafka semantics
- Invalid JSON and empty `order_uid` are committed (skipped) to avoid infinite retries.
- DB errors are NOT committed (at-least-once; message will be retried later).

## Project structure
```
Test task/
  cashe.go
  config.go
  consumer.go
  db.go
  docker-compose.yml
  Dockerfile
  go.mod / go.sum
  http.go
  insert_order.sql
  main.go
  migrations/
    001_create_tables.sql
  models.go
  sample_order.json
  static/
    index.html
```

## Development
```powershell
go fmt ./...
go vet ./...
go test ./...   # (no tests provided in this task)
```

## Troubleshooting
- Port 8081 already in use: change the host mapping in `docker-compose.yml` under `orders-service: ports`.
- Service starts before Postgres is ready: transient log "the database system is starting up"; it will keep running, and reads/writes will work once DB is ready.
- Producing sample JSON fails in PowerShell with `<` redirection: use the container command shown above (avoids PowerShell redirection).
- Docker not running: open Docker Desktop and retry `docker compose up -d`.

## License
MIT (or adapt as needed for your use case).


