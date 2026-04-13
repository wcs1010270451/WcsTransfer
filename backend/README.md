# Backend

This is the Go + Gin backend for the WcsTransfer model gateway.

## Run

```powershell
go mod tidy
go run ./cmd/server
```

The backend now auto-loads `backend/.env` when present.

## Environment Variables

- `APP_ENV`: application environment, defaults to `development`
- `HTTP_PORT`: HTTP port, defaults to `8080`
- `GIN_MODE`: Gin mode, defaults to `debug`
- `ADMIN_TOKEN`: bearer token for `/admin/*` routes. Empty means auth is disabled for now.
- `CORS_ALLOWED_ORIGINS`: comma-separated frontend origins allowed by CORS
- `HTTP_READ_TIMEOUT`: defaults to `15s`
- `HTTP_WRITE_TIMEOUT`: defaults to `60s`
- `HTTP_SHUTDOWN_TIMEOUT`: defaults to `10s`
- `DATABASE_URL`: PostgreSQL DSN
- `DATABASE_MAX_CONNS`: defaults to `20`
- `DATABASE_MIN_CONNS`: defaults to `2`
- `REDIS_ADDR`: Redis address, defaults to `localhost:6379`
- `REDIS_PASSWORD`: Redis password, defaults to empty
- `REDIS_DB`: Redis DB index, defaults to `0`
- `DEPENDENCY_TIMEOUT`: dependency ping timeout, defaults to `3s`

## Current Scope

The current skeleton focuses on:

- app bootstrap
- configuration loading
- routing and middleware
- PostgreSQL and Redis dependency wiring
- DB-backed admin configuration APIs
- OpenAI-style model listing and chat proxy
- request log persistence for chat completions
- streaming proxy support for `stream: true`

Provider routing, database integration, key management, and logging persistence can be added on top of this structure.

## Database

Initial PostgreSQL schema migrations are available in `migrations/`:

- `0001_init.up.sql`
- `0001_init.down.sql`

The schema currently includes:

- `admin_users`
- `providers`
- `provider_keys`
- `models`
- `request_logs`

## Docker Compose

From the repository root:

```powershell
docker compose up -d --build
```

The Compose stack starts:

- PostgreSQL
- Redis
- backend

## Current Admin APIs

- `GET /admin/providers`
- `POST /admin/providers`
- `GET /admin/keys`
- `POST /admin/keys`
- `GET /admin/models`
- `POST /admin/models`
- `GET /admin/logs`

Chat proxy requests made through `/v1/chat/completions` are now written into `request_logs`.

## Dev Seed

For local testing, you can import `scripts/dev_seed.sql`.

Before running it, replace the placeholder API key in that file with your real upstream key.

## Quick Smoke Test

After seeding the database and starting the backend, you can test the proxy with:

```powershell
curl.exe -X POST http://localhost:3210/v1/chat/completions `
  -H "Content-Type: application/json" `
  -d "{\"model\":\"gpt-4o-mini\",\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}"
```
