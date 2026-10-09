# Coin Transfer Service

Сервис переводов монет между аккаунтами игроков. Условие задания — в [TASK.md](TASK.md).

## Запуск

```sh
docker compose up --build
```

Сервис доступен на `http://localhost:3000`, PostgreSQL — на `localhost:5432`.

Локально без Docker:

```sh
go run ./app/cmd/main.go
```

## Эндпоинты

| Метод  | Путь                                  | Описание                    |
|--------|---------------------------------------|-----------------------------|
| `POST` | `/api/v1/transfers`                   | Создать перевод             |
| `GET`  | `/api/v1/transfers/{idempotency_key}` | Получить перевод            |
| `GET`  | `/api/v1/accounts/{id}`               | Получить аккаунт с балансом |
| `GET`  | `/livez`                              | Liveness probe              |
| `GET`  | `/readyz`                             | Readiness probe             |
| `GET`  | `/docs`                               | Документация API            |
| `GET`  | `/openapi.json`                       | OpenAPI-спецификация        |
