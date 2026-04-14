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
- multi-key failover, bounded retry, and temporary unhealthy-key cooldown

Provider routing, database integration, key management, and logging persistence can be added on top of this structure.

## Database

Initial PostgreSQL schema migrations are available in `migrations/`:

- `0001_init.up.sql`
- `0001_init.down.sql`
- `0002_client_api_keys.up.sql`
- `0002_client_api_keys.down.sql`
- `0003_client_key_quotas.up.sql`
- `0003_client_key_quotas.down.sql`

The schema currently includes:

- `admin_users`
- `providers`
- `provider_keys`
- `client_api_keys`
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
- `GET /admin/client-keys`
- `POST /admin/client-keys`
- `GET /admin/keys`
- `POST /admin/keys`
- `GET /admin/models`
- `POST /admin/models`
- `GET /admin/logs`

Chat proxy requests made through `/v1/chat/completions` are now written into `request_logs`.

## Public API Auth

`/v1/*` routes now require one of these headers when PostgreSQL-backed auth is enabled:

- `Authorization: Bearer <client_api_key>`
- `X-API-Key: <client_api_key>`

Create business-side keys from the `Client Keys` page in the admin console or through the `/admin/client-keys` API. New keys are shown in plain text only once at creation time.

## Client Quotas

Client keys now support:

- `rpm_limit`: requests per minute
- `daily_request_limit`: total requests per UTC day
- `daily_token_limit`: total tokens per UTC day

The gateway checks RPM and daily request limits before proxying the request, then adds token usage after a successful or failed model call when usage is available.

## Routing Resilience

`/v1/chat/completions` now includes a first-pass resilience layer:

- active keys are ordered by priority and weight
- failed keys can fall through to the next available key
- the last candidate key gets one bounded retry for transient failures
- keys that hit `429`, `401`, `403`, network errors, or upstream `5xx` are temporarily cooled down in memory
- routing decisions are recorded in request log metadata

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
