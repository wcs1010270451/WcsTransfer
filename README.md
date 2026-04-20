# WcsTransfer

WcsTransfer is a model gateway project with:

- `backend/`: Go + Gin gateway service
- `frontend/`: React admin console

The current repository includes a runnable backend skeleton with:

- health and version endpoints
- OpenAI-compatible model listing and proxy endpoints
- admin configuration and observability APIs
- basic request ID and admin auth middleware
- PostgreSQL schema migration
- Docker Compose for local development

## Backend Quick Start

```powershell
cd backend
go mod tidy
go run ./cmd/server
```

Recommended local toolchain:

- Go `1.24.5+`

Optional environment variables:

- `APP_ENV=development`
- `HTTP_PORT=8080`
- `GIN_MODE=debug`
- `ADMIN_TOKEN=change-me`
- `DATABASE_URL=postgres://wcstransfer:wcstransfer@localhost:5432/wcstransfer?sslmode=disable`
- `REDIS_ADDR=localhost:6379`
- `HTTP_READ_TIMEOUT=15s`
- `HTTP_WRITE_TIMEOUT=60s`
- `HTTP_SHUTDOWN_TIMEOUT=10s`

Available endpoints:

- `GET /healthz`
- `GET /version`
- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/messages`
- `POST /portal/auth/register`
- `POST /portal/auth/login`
- `GET /portal/me`
- `GET /portal/client-keys`
- `GET /admin/client-keys`
- `POST /admin/client-keys`
- `GET /admin/providers`
- `POST /admin/providers`
- `GET /admin/keys`
- `POST /admin/keys`
- `GET /admin/models`
- `POST /admin/models`
- `GET /admin/logs`

## Local Development Stack

```powershell
docker compose up -d --build
```

Services:

- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Backend API: `http://localhost:8080`

The first PostgreSQL startup automatically applies the backend migrations, including client API key and quota schemas in:

- `backend/migrations/0002_client_api_keys.up.sql`
- `backend/migrations/0003_client_key_quotas.up.sql`
- `backend/migrations/0006_add_anthropic_provider_type.up.sql`
- `backend/migrations/0007_tenants_and_tenant_users.up.sql`

## Tenant Portal

The repository now includes a first-pass tenant portal for self-service client key management.

Tenant user flow:

- register a workspace with `POST /portal/auth/register`
- sign in with `POST /portal/auth/login`
- call `GET /portal/me` with the returned bearer token
- create and disable tenant-owned client keys through `/portal/client-keys`

Frontend routes under the console base path:

- `/portal/login`
- `/portal/keys`

Backend requirement:

- set `AUTH_TOKEN_SECRET` to a non-default value before deploying beyond local development

## Anthropic Provider Setup

WcsTransfer supports Anthropic's official Messages API directly.

Recommended provider configuration:

- `provider_type`: `anthropic`
- `base_url`: `https://api.anthropic.com`
- `extra_config`:

```json
{
  "anthropic_version": "2023-06-01"
}
```

Important:

- do not set `base_url` to a full endpoint such as `https://api.anthropic.com/v1/messages`
- the gateway appends `/v1/messages` automatically
- Anthropic provider keys should be Claude Console API keys

Example public request:

```powershell
curl.exe -X POST http://localhost:3210/v1/messages `
  -H "Authorization: Bearer <client_api_key>" `
  -H "Content-Type: application/json" `
  -d "{\"model\":\"claude-sonnet-4\",\"max_tokens\":1024,\"messages\":[{\"role\":\"user\",\"content\":\"hello\"}]}"
```

## Production Deployment

The repository now includes a production stack in `docker-compose.prod.yml` with:

- PostgreSQL
- Redis
- one-shot migration runner
- backend API
- frontend static console
- Caddy edge proxy with automatic HTTPS

Supporting files:

- `backend/scripts/run-migrations.sh`
- `frontend/Dockerfile`
- `frontend/nginx.conf`
- `deploy/Caddyfile`
- `.env.prod.example`

Deployment steps:

1. Copy `.env.prod.example` to `.env.prod`
2. Set at least:
   - `DOMAIN`
   - `PUBLIC_BASE_URL`
   - `POSTGRES_PASSWORD`
   - `ADMIN_TOKEN`
   - `ADMIN_UI_USER`
   - `ADMIN_UI_PASSWORD_HASH`
   - `ACME_EMAIL`
3. Bring up the production stack:

```powershell
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build
```

What the production stack does:

- Caddy terminates HTTPS on `80/443`
- frontend console is served at `/console/`
- `/console/*` is protected with Caddy Basic Auth
- backend is only exposed internally to the proxy
- migrations are applied automatically before backend startup

Recommended public routes:

- `https://your-domain/console/`
- `https://your-domain/docs`
- `https://your-domain/redoc`
- `https://your-domain/openapi.json`

Operational notes:

- `ADMIN_TOKEN` should be rotated away from any development value
- do not inject `ADMIN_TOKEN` into frontend build args or static files
- generate `ADMIN_UI_PASSWORD_HASH` with `caddy hash-password --plaintext 'your-password'`
- `PUBLIC_BASE_URL` should match the final external origin exactly
- current CORS policy is set from `PUBLIC_BASE_URL`
- if you already run an external reverse proxy or load balancer, you can reuse the backend/frontend services and skip Caddy
