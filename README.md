# TripGo — лабораторная работа 2

HTTP-сервис для создания, получения и завершения поездок. Данные хранятся в
PostgreSQL, миграции выполняются Goose, а HTTP-типы и серверные интерфейсы
генерируются из OpenAPI. Код разделён на слои, покрыт тестами и подключён к
OpenTelemetry

## API

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/api/v1/trips` | создать поездку |
| `GET` | `/api/v1/trips/{tripId}` | получить поездку |
| `POST` | `/api/v1/trips/{tripId}/finish` | завершить поездку |
| `GET` | `/health` | проверить процесс без обращения к БД |
| `GET` | `/ready` | проверить доступность PostgreSQL |

Контракт находится в `contracts/openapi/trip-service.openapi.yaml`. Ошибки API
возвращаются в `application/problem+json`; внутренние ошибки и stack trace не
попадают в ответ клиенту

## Требования

- Go версии, указанной в `go.mod`
- Docker
- GNU Make и `tripgoctl`
- `golangci-lint` v2
- `curl`, `jq` и `psql` для ручной проверки

На Windows команды выполняются внутри WSL2 с включённой интеграцией Docker
Desktop. Makefile использует `/bin/bash`

## Запуск

Из корня репозитория:

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect

go mod download
make generate
make migrate
make run
```

`tripgoctl environment start` создаёт `.env` с фактическими адресами PostgreSQL
и OTLP. Корневой `.env.example` содержит локальные значения. Makefile сначала
загружает `.env.example`, затем переопределяет значения из `.env`

Сервис доступен по адресу `http://localhost:8080`

## Docker

Образ собирается из корня репозитория:

```bash
docker build -f deploy/Dockerfile -t trip-service:local .
```

PostgreSQL и observability поднимаются через `tripgoctl`, миграции применяются
командой `make migrate`. В Docker Desktop адрес `localhost` из сгенерированного
`.env` заменяется на адрес хоста, потому что внутри контейнера `localhost`
указывает на сам контейнер:

```bash
tripgoctl environment start
make migrate

source .env
CONTAINER_DATABASE_URL="${DATABASE_URL/localhost/host.docker.internal}"
CONTAINER_OTEL_ENDPOINT="${OTEL_EXPORTER_OTLP_ENDPOINT/localhost/host.docker.internal}"

docker run --rm -d \
  --name trip-service \
  --add-host=host.docker.internal:host-gateway \
  --env-file .env.example \
  --env-file .env \
  -e DATABASE_URL="$CONTAINER_DATABASE_URL" \
  -e OTEL_EXPORTER_OTLP_ENDPOINT="$CONTAINER_OTEL_ENDPOINT" \
  -p 8080:8080 \
  trip-service:local

curl -i http://localhost:8080/ready
docker stop --timeout 15 trip-service
```

Итоговый образ построен на `scratch` и запускается от пользователя
`65532:65532`

На Linux с нативным Docker контейнер запускается с `--network host` и адресами
из `.env` без замены `localhost`

## Команды

| Команда | Назначение |
|---|---|
| `make generate` | сгенерировать OpenAPI-код и моки |
| `make migrate` | применить новые миграции |
| `make migrate-down` | откатить последнюю миграцию |
| `make migrate-status` | показать состояние миграций |
| `make run` | запустить сервис |
| `make build` | собрать все пакеты |
| `make test` | дважды запустить unit-тесты с race detector и вывести покрытие domain/usecase |
| `make test-integration` | запустить интеграционные тесты repository на PostgreSQL |
| `make lint` | запустить `golangci-lint` |

## Архитектура

- `internal/domain/trip` — бизнес-модель, инварианты и доменные ошибки
- `internal/model` — пассивные DTO между слоями
- `internal/usecase/trip/create|get|finish` — сценарии приложения
- `internal/http/trip/create|get|finish` — HTTP-обработчики
- `internal/http/domainerror` — единое преобразование ошибок в API Problem
- `internal/repository/trip` — SQL и преобразование данных PostgreSQL
- `internal/infra` — пул, транзакции и OpenTelemetry
- `internal/container` — создание и связывание зависимостей
- `cmd/trip-service` — конфигурация и жизненный цикл процесса

Зависимости направлены к бизнес-логике: domain не знает о HTTP, PostgreSQL и
OpenTelemetry. Интерфейсы зависимостей объявлены на стороне потребителя,
потому что потребитель определяет необходимый ему контракт. Контракты usecase
и HTTP-обработчиков находятся в `contract.go`

`api.*` — транспортные типы OpenAPI, `model.*` — межслойные DTO, а
`domain.Trip` — модель с бизнес-правилами. `container` создаёт конкретные
реализации и связывает их, а `main` управляет запуском и остановкой процесса

## Тесты

```bash
make test

tripgoctl environment start
make migrate
make test-integration

make lint
```

Domain проверяется table-driven тестами без моков. Usecase и HTTP-обработчики
проверяются unit-тестами с `gomock`. Repository и rollback транзакции
проверяются на PostgreSQL из `tripgoctl`; эти тесты помечены
`//go:build integration`, используют уникальные данные и удаляют их после
выполнения. `make test` выводит покрытие domain/usecase, которое должно быть не
ниже 50%

## Наблюдаемость

Сервис пишет структурированные JSON-логи с `trace_id` и `span_id`. Middleware
подключены в порядке `Recover → Tracing → Logging → Metrics`. HTTP-запрос
создаёт корневой span, а запросы к PostgreSQL — дочерние spans

Обязательные метрики:

- `http.server.request.duration`
- `http.server.active_requests`
- `db.client.operation.duration`
- `tripgo.trips.created`
- `tripgo.trips.completed`
- `tripgo.trips.active`

В `http.route` записывается шаблон маршрута без UUID. Недоступность OTLP
collector не мешает обработке HTTP-запросов. Адреса Jaeger и Grafana выводит
`tripgoctl connect`

## Переменные окружения

Все переменные обязательны. Безопасные локальные значения находятся в
`.env.example`

| Переменная | Пример | Назначение |
|---|---|---|
| `HTTP_ADDR` | `:8080` | адрес HTTP-сервера |
| `LOG_LEVEL` | `info` | уровень логирования |
| `SHUTDOWN_TIMEOUT` | `10s` | общий бюджет graceful shutdown |
| `DATABASE_URL` | `postgres://...` | строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | `10` | максимальный размер пула |
| `DATABASE_MIN_CONNS` | `2` | минимальный размер пула |
| `DATABASE_MAX_CONN_LIFETIME` | `30m` | максимальное время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | `5s` | таймаут подключения и стартового `Ping` |
| `DATABASE_QUERY_TIMEOUT` | `3s` | таймаут запросов к БД |
| `HTTP_READ_TIMEOUT` | `10s` | таймаут чтения запроса |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | таймаут чтения заголовков |
| `HTTP_WRITE_TIMEOUT` | `15s` | таймаут записи ответа |
| `HTTP_IDLE_TIMEOUT` | `60s` | таймаут keep-alive соединения |
| `OTEL_SERVICE_NAME` | `trip-service` | имя сервиса в телеметрии |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | адрес OTLP collector |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | протокол OTLP exporter |
| `OTEL_RESOURCE_ATTRIBUTES` | `service.namespace=tripgo,deployment.environment=local` | атрибуты ресурса OpenTelemetry |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | стратегия сэмплирования трейсов |

Сервис не стартует, если обязательная переменная отсутствует или имеет
некорректное значение

## Проверка

```bash
make generate
make build
make test
make lint
make migrate-status
```

Проверка HTTP:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready

curl -i -X POST http://localhost:8080/api/v1/trips \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
    "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
    "start_point":{"latitude":59.9398,"longitude":30.3146},
    "end_point":{"latitude":59.929,"longitude":30.3626},
    "price":1450
  }'

curl -i http://localhost:8080/api/v1/trips/<TRIP_ID>
curl -i -X POST http://localhost:8080/api/v1/trips/<TRIP_ID>/finish
```

Полный откат миграций удаляет локальные данные:

```bash
make migrate-down
make migrate-down
make migrate
```

При остановленной БД `/health` продолжает возвращать `200`, а `/ready`
возвращает `503`:

```bash
tripgoctl environment stop
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
tripgoctl environment start
```

По `SIGINT` или `SIGTERM` один `SHUTDOWN_TIMEOUT` ограничивает остановку
HTTP-сервера, закрытие пула PostgreSQL и телеметрии. `service stopped`
записывается после успешного завершения всех действий. При исчерпании бюджета
ошибка записывается в лог и процесс завершается с кодом `1`

## Решения

### Уровень изоляции

Транзакции используют `READ COMMITTED` — стандартный уровень PostgreSQL.
Необходимые конкурентные гарантии обеспечиваются ограничением БД и условным
`UPDATE`, поэтому более строгий уровень изоляции не требуется

### Менеджер транзакций

`TxManager.Do` открывает транзакцию и помещает её в `context.Context`.
Репозиторий берёт из контекста транзакцию либо использует пул, если транзакции
нет. `nil` из callback приводит к `COMMIT`, ошибка или паника — к `ROLLBACK`.
Вложенный `Do` переиспользует текущую транзакцию. Бизнес-логика не зависит от
`pgx` и не принимает транзакцию аргументом

Создание поездки и начальной записи истории выполняется в одной транзакции

### Конкурентные ограничения

Частичный уникальный индекс `trips_driver_id_active_unique_idx` запрещает две
активные поездки одного водителя. Ошибка PostgreSQL `23505` по этому индексу
преобразуется в `driver_busy`

Завершение выполняется одним `UPDATE` с условием `status = 'active'`. Поэтому
при двух параллельных запросах только один возвращает `200`, второй —
`409 trip_completed`, а `finished_at` не перезаписывается

## Остановка окружения

```bash
tripgoctl environment stop
```
