# My Subs

![Go](https://img.shields.io/badge/Go-1.25.0-blue)
![Postgres](https://img.shields.io/badge/Postgres-16-blue)
![Docker](https://img.shields.io/badge/Docker-ready-blue)
![CI](https://github.com/EgorLis/my-subs/actions/workflows/go-tests.yml/badge.svg)

**My Subs** --- учебный проект для управления подписками с
использованием Go, PostgreSQL и Docker.\
Проект построен с учётом лучших практик: миграции БД, тесты, CI/CD,
документация через Swagger.

------------------------------------------------------------------------

## 🚀 Стек технологий

-   **Язык:** Go 1.25.0
-   **База данных:** PostgreSQL
    -   [pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool)
        --- пул соединений
    -   [golang-migrate](https://github.com/golang-migrate/migrate) ---
        миграции
-   **Инфраструктура:** Docker, docker-compose
-   **Веб-сервер:** стандартный `net/http`
-   **Логгирование:** встроенный логгер Go
-   **Тестирование:** unit-тесты с использованием mock-репозитория
-   **Taskfile:** автоматизация рутинных задач
-   **CI:** GitHub Actions (запуск тестов при каждом коммите/PR)
-   **Документация:** Swagger (через swaggo/swag)

------------------------------------------------------------------------

## 📂 Структура проекта

``` bash
my-subs/
├── cmd/                # Точка входа приложения
├── configs/            # Примеры конфигов (.env.example, .env.docker.example)
├── deployments/docker/ # Dockerfile, docker-compose.yml
├── internal/           # Внутренняя логика (app, config, domain, infra, transport)
│   ├── app/            # Builder приложения
│   ├── config/         # Конфиги, загрузка ENV
│   ├── docs/           # Swagger (генерируется)
│   ├── domain/         # Доменные сущности и интерфейсы
│   ├── infra/          # Репозитории (mock, postgres)
│   └── transport/      # HTTP API (handlers, middleware, v1)
├── Taskfile.yml        # Сценарии для запуска и управления
├── README.md           # Документация
└── go.mod / go.sum     # Зависимости
```

------------------------------------------------------------------------

## ⚙️ Запуск проекта

### 1. Подготовка окружения

Копирование примеров конфигураций в корневую папку (`.env` и `.env.docker`)

``` bash
task env
```

### 2. Прогон тестов

``` bash
task test       # локально (go test)
task test:docker # внутри Dockerfile (stage test)
```

### 3. Запуск приложения и БД

``` bash
task up         # форграунд
task up:detached # в фоне
```

После запуска сервер будет доступен на `http://localhost:8001`.

### 4. Swagger-документация

Генерация доков (уже лежит в репозитории):

``` bash
task swagger
```

Открыть Swagger UI:

    http://localhost:8001/swagger/index.html

### 5. Управление контейнерами

``` bash
task logs        # логи
task ps          # статус
task stop        # остановить контейнеры
task down        # удалить контейнеры (volume сохраняется)
task down:volumes # снести контейнеры и volumes (очистка postgres_data)
```

------------------------------------------------------------------------

## 🧪 CI/CD

GitHub Actions запускает unit-тесты при каждом push/PR.\
Файл workflow: [go-tests.yml](.github/workflows/go-tests.yml).

------------------------------------------------------------------------

## 🧪 Тестирование

- Обеспечено покрытие unit‑тестами HTTP‑хендлера **subscriptions**.
- В тестах используется **mock‑репозиторий** вместо Postgres.
- Проверяются сценарии: **OK**, ошибки **валидации (400)**, **not found (404)**, **timeout (504)**, **internal error (500)**.
- На CI (GitHub Actions) тесты запускаются при каждом push/PR.

Запуск локально:
```bash
go test ./... -race -cover
```

Покрытие (HTML‑отчёт):
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

------------------------------------------------------------------------

## 📡 Примеры API-запросов

Все эндпоинты принимают/возвращают `application/json`.

### Схемы ответов

```jsonc
// SubscriptionDTO
{
  "service_name": "string",
  "price": 0,
  "user_id": "GUID",
  "start_date": "MM-YYYY", // YearMonth
  "end_date": "MM-YYYY"    // YearMonth
}

// CUDResponse (Create/Update/Delete)
{
  "subscription_id": "GUID",
  "status": "subscription created | subscription updated | subscription deleted"
}

// ListResponse
{
  "subscriptions": [ SubscriptionDTO, ... ]
}

// TotalCostResponse
{
  "service_name": "string",
  "user_id": "GUID",
  "from": "MM-YYYY",
  "to": "MM-YYYY",
  "total_cost": 0
}

// Error (общая форма ошибок)
{ "error": "message" }
```

---

### 1) Создать подписку — `POST /v1/subscriptions`

**Заголовки запроса**
```
Content-Type: application/json
```

**Тело запроса**
```json
{
  "service_name": "Yandex Plus",
  "price": 400,
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
  "start_date": "07-2025",
  "end_date": "12-2025"
}
```

**Ответы сервера**
- `200 OK`
  ```json
  { "subscription_id": "3ba9941a-9fbb-4f7e-9d2e-0e5f6b2e49a2", "status": "subscription created" }
  ```
- `400 Bad Request`
  ```json
  { "error": "service_name: must not be empty" }
  ```
- `504 Gateway Timeout`
  ```json
  { "error": "request timed out" }
  ```
- `500 Internal Server Error`
  ```json
  { "error": "" }
  ```

---

### 2) Получить информацию о подписке — `GET /v1/subscriptions/{id}`

**Параметры**
- `id` — GUID

**Ответы сервера**
- `200 OK`
  ```json
  {
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025",
    "end_date": "12-2025"
  }
  ```
- `400 Bad Request`
  ```json
  { "error": "id: invalid GUID" }
  ```
- `404 Not Found`
  ```json
  { "error": "not found" }
  ```
- `504 Gateway Timeout`
  ```json
  { "error": "request timed out" }
  ```
- `500 Internal Server Error`
  ```json
  { "error": "" }
  ```

---

### 3) Обновить информацию о подписке — `PUT /v1/subscriptions/{id}`

**Заголовки запроса**
```
Content-Type: application/json
```

**Тело запроса**
```json
{
  "id": "3ba9941a-9fbb-4f7e-9d2e-0e5f6b2e49a2",
  "service_name": "Yandex Plus Premium",
  "price": 500,
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
  "start_date": "07-2025",
  "end_date": "12-2025"
}
```

**Ответы сервера**
- `200 OK`
  ```json
  { "subscription_id": "3ba9941a-9fbb-4f7e-9d2e-0e5f6b2e49a2", "status": "subscription updated" }
  ```
- `400 Bad Request`
  ```json
  { "error": "date range: start_date must be <= end_date" }
  ```
- `404 Not Found`
  ```json
  { "error": "not found" }
  ```
- `504 Gateway Timeout`
  ```json
  { "error": "request timed out" }
  ```
- `500 Internal Server Error`
  ```json
  { "error": "" }
  ```

---

### 4) Удалить подписку — `DELETE /v1/subscriptions/{id}`

**Параметры**
- `id` — GUID

**Ответы сервера**
- `200 OK`
  ```json
  { "subscription_id": "3ba9941a-9fbb-4f7e-9d2e-0e5f6b2e49a2", "status": "subscription deleted" }
  ```
- `400 Bad Request`
  ```json
  { "error": "id: invalid GUID" }
  ```
- `404 Not Found`
  ```json
  { "error": "not found" }
  ```
- `504 Gateway Timeout`
  ```json
  { "error": "request timed out" }
  ```
- `500 Internal Server Error`
  ```json
  { "error": "" }
  ```

---

### 5) Список всех существующих подписок — `GET /v1/subscriptions`

**Ответы сервера**
- `200 OK`
  ```json
  {
    "subscriptions": [
      {
        "service_name": "Yandex Plus",
        "price": 400,
        "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
        "start_date": "07-2025",
        "end_date": "12-2025"
      },
      {
        "service_name": "Spotify",
        "price": 300,
        "user_id": "7f2d0a07-0f8b-4b28-8c77-5f8f1e0c3a21",
        "start_date": "01-2025",
        "end_date": "06-2025"
      }
    ]
  }
  ```
- `504 Gateway Timeout`
  ```json
  { "error": "request timed out" }
  ```
- `500 Internal Server Error`
  ```json
  { "error": "" }
  ```

---

### 6) Общая стоимость подписок (пользователь и название сервиса) — `GET /v1/subscriptions/totalcost`

**Параметры запроса**
- `user_id` (GUID) — обязателен  
- `service_name` (string) — обязателен  
- `from` (MM-YYYY) — обязателен  
- `to` (MM-YYYY) — обязателен

**Ответы сервера**
- `200 OK`
  ```json
  {
    "service_name": "Yandex Plus",
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "from": "07-2025",
    "to": "12-2025",
    "total_cost": 700
  }
  ```
- `400 Bad Request`
  ```json
  { "error": "from: invalid format, expected MM-YYYY" }
  ```
- `504 Gateway Timeout`
  ```json
  { "error": "request timed out" }
  ```
- `500 Internal Server Error`
  ```json
  { "error": "" }
  ```
------------------------------------------------------------------------

## 📖 Полезные команды

``` bash
task help          # список доступных задач
task clean         # очистить dangling-образы
task swagger       # генерация Swagger доков
task test          # локальные тесты
task test:docker   # тесты внутри docker stage
```

------------------------------------------------------------------------

## 📌 Репозиторий

[🔗 GitHub: EgorLis/my-subs](https://github.com/EgorLis/my-subs)
