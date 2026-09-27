# TripGo — лабораторная работа 1

HTTP-сервис для создания, получения и завершения поездок. Данные хранятся в
PostgreSQL, миграции выполняются Goose, а HTTP-типы и серверные интерфейсы
генерируются из OpenAPI

## API

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/api/v1/trips` | создать поездку |
| `GET` | `/api/v1/trips/{tripId}` | получить поездку |
| `POST` | `/api/v1/trips/{tripId}/finish` | завершить поездку |
| `GET` | `/health` | проверить процесс без обращения к БД |
| `GET` | `/ready` | проверить доступность PostgreSQL |

Контракт находится в `contracts/openapi/trip-service.openapi.yaml`. Ошибки API
возвращаются в `application/problem+json`; внутренние ошибки и stack trace
остаются только в логах

## Требования

- Go версии, указанной в `go.mod`
- Docker
- GNU Make и `tripgoctl`
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

`tripgoctl environment start` создаёт `.env` с фактическим `DATABASE_URL`.
Корневой `.env.example` составлен на основе
`config/trip-service.env.example` из репозитория курса. Makefile сначала
загружает `.env.example`, затем переопределяет значения из `.env`

Сервис доступен по адресу `http://localhost:8080`

## Команды

| Команда | Назначение |
|---|---|
| `make generate` | сгенерировать типы и chi-интерфейсы из OpenAPI |
| `make migrate` | применить новые миграции |
| `make migrate-down` | откатить последнюю миграцию |
| `make migrate-status` | показать состояние миграций |
| `make run` | запустить сервис |
| `make build` | собрать все пакеты |
| `make test` | дважды запустить тесты с race detector |

## Переменные окружения

Все переменные обязательны. Безопасные локальные значения находятся в
`.env.example`

| Переменная | Пример | Назначение |
|---|---|---|
| `HTTP_ADDR` | `:8080` | адрес HTTP-сервера |
| `LOG_LEVEL` | `info` | уровень логирования |
| `SHUTDOWN_TIMEOUT` | `10s` | общий бюджет graceful shutdown |
| `HTTP_READ_TIMEOUT` | `10s` | таймаут чтения запроса |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | таймаут чтения заголовков |
| `HTTP_WRITE_TIMEOUT` | `15s` | таймаут записи ответа |
| `HTTP_IDLE_TIMEOUT` | `60s` | таймаут keep-alive соединения |
| `DATABASE_URL` | `postgres://...` | строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | `10` | максимальный размер пула |
| `DATABASE_MIN_CONNS` | `2` | минимальный размер пула |
| `DATABASE_MAX_CONN_LIFETIME` | `30m` | максимальное время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | `5s` | таймаут подключения и стартового `Ping` |
| `DATABASE_QUERY_TIMEOUT` | `3s` | таймаут запросов к БД |

Сервис не стартует, если обязательная переменная отсутствует или имеет
некорректное значение

## Проверка

```bash
make generate
make build
make test
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
HTTP-сервера и закрытие пула PostgreSQL. `service stopped` записывается после
обоих действий; при исчерпании бюджета процесс завершается принудительно

## Решения

### Уровень изоляции

Транзакции используют `READ COMMITTED` — стандартный уровень PostgreSQL.
Необходимые конкурентные гарантии обеспечиваются ограничением БД и условным
`UPDATE`, поэтому более строгий уровень изоляции не требуется

### Менеджер транзакций

`TxManager.Do` открывает транзакцию и помещает её в `context.Context`.
Репозиторий берёт из контекста транзакцию либо использует пул, если транзакции
нет. `nil` из callback приводит к `COMMIT`, ошибка или паника — к `ROLLBACK`.
Вложенный `Do` переиспользует текущую транзакцию. Бизнес-сервис не зависит от
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
