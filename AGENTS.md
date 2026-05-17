# DPM — Music Streaming Microservice

**Стек:** Go 1.25 + PostgreSQL 16 + Yandex S3 + Nginx

**Архитектура:** Hexagonal (ports & adapters)

```
Nginx (HTTPS) → Go App (:3003) → PostgreSQL
                                → Yandex S3
```

## Ключевые пакеты

| Пакет | Роль |
|---|---|
| `internal/cmd/main.go` | Точка входа, DI, запуск сервера |
| `internal/cmd/shutdown/` | Graceful shutdown (не подключен) |
| `internal/config/` | Загрузка конфига (.env + config.yaml) |
| `internal/adapters/http/` | HTTP handlers, middleware, routes |
| `internal/adapters/repo/postgres/` | PostgreSQL (pgx), SQL-запросы |
| `internal/adapters/repo/objectStorage/` | S3 (AWS SDK v2), presigned URLs |
| `internal/services/` | Бизнес-логика |
| `internal/models/` | Модели данных |
| `pkg/api/v1/` | Сгенерированный код из OpenAPI (oapi-codegen) |
| `api/openapi.yaml` | OpenAPI 3.0.1 спецификация |
| `html/` | Статический фронтенд (HTML + JS) |

## Паттерны

- DI через интерфейсы — сервисы принимают узкие интерфейсы репозиториев
- Functional options для конфигурации (WithMinConns, WithMaxConns)
- Cookie-based JWT (Access-Token) + oapi-validator
- Worker pool (5 горутин) для multipart upload
- Presigned URLs для стриминга аудио (24h)
- BTREE Counter — денормализованные счётчики (лайки, избранное, прослушивания)
- oapi-codegen strict-server для валидации запросов/ответов

## Инфраструктура

- Docker Compose: app + postgres + nginx
- Nginx: reverse proxy с самоподписанным TLS, раздаёт статику
- БД: таблицы users, music, users_music_likes, favor, listening_history
