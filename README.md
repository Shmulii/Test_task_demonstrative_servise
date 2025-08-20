## Сервер запросов (Тестовое задание)

Микросервис написаный на ЯП Golang, который принимает заказы из Kafka, сохраняет их в Postgres, обслуживает по HTTP и кэширует последние заказы в памяти. Поставляется с Docker Compose для локального запуска Postgres и Redpanda (совместимых с Kafka).

### Features
- Kafka/Redpanda потребитель (at-least-once) анализирующий JSON `Order`
- Сохранеине данных Postgres вместе с транзакционной обработкой и вставкой характеристик(`orders`, `deliveries`, `payments`, `items`)
- HTTP API со сквозным чтением кэша в памяти
- Статическая целевая (посадочная) страница
- Нормальное заверщение работы 

## Архитектура 
- `main.go`: подключение и жизненый цикл (startup, warmup, HTTP, consumer, shutdown)
- `config.go`: конфигурация (env + defaults)
- `models.go`: структуры данных (`Order`, `Delivery`, `Payment`, `Item`)
- `db.go`: Postgres pool + логика загрузки/сохранения
- `consumer.go`: цикл чтения Kafka/Redpanda (сегмент `kafka-go`)
- `http.go`: HTTP-сервер и маршруты (Gorilla Mux)
- `cashe.go`: простой параллельный кэш с ограничением емкости
- `migrations/001_create_tables.sql`: схема базы данных
- `static/`: статистические ресурсы (index page)

## Системные требования
- Docker Desktop (реккомендуется)
- Go 1.20+, Postgres 15+ и Kafka (или Redpanda)

## Quick start (Docker)
Запускает Postgres, Redpanda и сервис.

```powershell
cd "C:\Users\Василий\go-projects\Test task"
docker compose up --build -d
```

- Служба прослушивает порт контейнера 8080, доступ к которому осуществляется через порт хоста 8081 (см. `docker-compose.yml`).
- Открывает UI: `http://localhost:8081/`

### Применение схемы базы данных (first run)
Служба не выполняет автоматическую миграцию. Загрузите SQL-запрос в контейнер Postgres:

```powershell
docker cp migrations/001_create_tables.sql testtask-postgres-1:/tmp/001.sql
docker exec -i testtask-postgres-1 psql -U orders_user -d orders_db -f /tmp/001.sql
```

### Создание простого заказа и запрос API
Вариант A — через Kafka/Redpanda (если настроено):

```powershell
docker cp sample_order.json testtask-redpanda-1:/tmp/sample_order.json
docker exec testtask-redpanda-1 sh -lc 'p=$(tr -d "\n" </tmp/sample_order.json); printf "%s\n" "$p" | rpk topic produce orders --brokers redpanda:9092'

$uid = (Get-Content -Raw sample_order.json | ConvertFrom-Json).order_uid
Invoke-RestMethod -Method GET -Uri ("http://localhost:8081/orders/" + $uid) | ConvertTo-Json -Depth 6
```

Вариант B — без Kafka (напрямую через SQL), удобно для локальной проверки:

```powershell
docker cp insert_order_fixed.sql testtask-postgres-1:/tmp/insert_order_fixed.sql
docker exec -i testtask-postgres-1 psql -U orders_user -d orders_db -f /tmp/insert_order_fixed.sql

$uid = (Get-Content -Raw sample_order.json | ConvertFrom-Json).order_uid
Invoke-RestMethod -Method GET -Uri ("http://localhost:8081/orders/" + $uid) | ConvertTo-Json -Depth 6
```

### Полезные команды
```powershell
# Статусы / логи
docker compose ps
docker compose logs --no-color --tail=200 orders-service

# Остановка / Удаление
docker compose down
```

## Запуск тестов
В отдельном терминале, пока сервис запущен:

```powershell
./test_simple.ps1
./test_cache.ps1
./test_api.ps1
./test_kafka.ps1
```

## Запуск локально без Docker
Необходимо запустить Postgres и Kafka/Redpanda локально, а также применить схему:

1) Настройте конфигурацию (пример, значения по умолчанию):
```powershell
$env:POSTGRES_DSN = "postgres://orders_user:orders_pass@localhost:5432/orders_db?sslmode=disable"
$env:KAFKA_BROKER = "localhost:9092"
```

2) Сброка и запуск: 
```powershell
go build -o orders-service .
./orders-service
# либо: go run .
```

По умолчанию HTTP будет прослушивать порт `:8080`. Откройте `http://localhost:8080/`.

## API
- `GET /orders/{order_uid}` → returns `Order` as JSON
- `GET /` → serves `./static/index.html`
- `GET /static/...` → static assets

Ответы в формате JSON, `Content-Type: application/json`.

## Конфигурация
Переменные среды (со значениями по умолчанию):
- `POSTGRES_DSN` = `postgres://orders_user:orders_pass@localhost:5432/orders_db?sslmode=disable`
- `KAFKA_BROKER` = `localhost:9092`
- `KAFKA_TOPIC` = `orders` (исправлено в коде)
- `HTTP_ADDR` = `:8080` (настроено в коде)
- `CACHE_LIMIT` = `1000` (настроено в коде)
- `STARTUP_LOAD` = `100` (настроено в коде)

Примечание: значения, отличные от `POSTGRES_DSN` и `KAFKA_BROKER` задаются внутри `DefaultConfig()`.

## Поведение кэша
- При запуске сервис загружает `STARTUP_LOAD` последние заказы в кэш.
- `GET /orders/{id}` сначала считывает данные из кэша, иначе извлекает данные из базы данных и кэширует результат.
- Кэш имеет жесткую емкость. Вставки удаляются при заполнении(без вытеснения).

## Семантика Kafka/Redpanda
- Недопустимые JSON и пустые `order_uid` фиксируются (пропускаются) для избежания бесконечных повторов.
- Ошибки базы данных НЕ фикисруются (at-least-once; будет повторено позже).

## Структура проекта
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
  insert_order_fixed.sql
  main.go
  migrations/
    001_create_tables.sql
  models.go
  sample_order.json
  static/
    index.html
```

## Устранение неполадок
- Если порт 8081 уже используется: измените сопоставление хоста в `docker-compose.yml` в разделе `orders-service: ports`.
- Сервис запускается до готовности Postgres: временный log "система базы данных запускается"; он продолжит работу и запись/чтение будут доступны после готовности базы данных.
- Создание примера JSON файла в PowerShell завершается ошибкой с перенаправлением `<`. Использнуйте команду контейнера показанную выше (avoid PowerShell rediraction).
- Docker не запускается: откройте Docker Desktop и повторите команду `docker compose up -d`.


