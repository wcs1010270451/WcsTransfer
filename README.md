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
- `ENABLE_DOCS=true`
- `ENABLE_ADMIN_DEBUG=true`
- `ADMIN_BOOTSTRAP_USERNAME=admin`
- `ADMIN_BOOTSTRAP_PASSWORD=change-me-admin-password`
- `ADMIN_BOOTSTRAP_DISPLAY_NAME=Platform Admin`
- `AUTH_TOKEN_SECRET=change-me-portal-secret`
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
- `POST /v1/gemini/generate-content`
- `POST /v1/gemini/stream-generate-content`
- `POST /v1/messages`
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
- `GET /admin/reconciliation/tenants`

## Local Development Stack

```powershell
docker compose up -d --build
```

## Environment File Layout

Use different `.env` files for different responsibilities. Do not merge frontend and backend runtime config into one file.

- Repository root `.env`
  - used by `docker-compose.yml`
  - only for local Docker Compose development
  - stores compose-layer variables such as PostgreSQL, Redis, backend container env, and bootstrap admin env
- `backend/.env`
  - used when running the backend directly with `go run ./cmd/server`
  - stores backend-only runtime config such as DB, Redis, auth secrets, and bootstrap admin settings
- `frontend/.env`
  - used when running the frontend directly with `npm run dev`
  - should only contain `VITE_` variables intended for the frontend build
- Repository root `.env.prod`
  - used by `docker-compose.prod.yml`
  - production compose entrypoint
  - should be the main production environment file instead of relying on `backend/.env` or `frontend/.env`

Recommended usage:

- local direct run:
  - backend reads `backend/.env`
  - frontend reads `frontend/.env`
- local Docker Compose:
  - compose reads repository root `.env`
- production Docker Compose:
  - compose reads repository root `.env.prod`

Services:

- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Backend API: `http://localhost:8080`

The first PostgreSQL startup automatically applies the backend migrations, including client API key and quota schemas in:

- `backend/migrations/0002_client_api_keys.up.sql`
- `backend/migrations/0003_client_key_quotas.up.sql`
- `backend/migrations/0006_add_anthropic_provider_type.up.sql`
- `backend/migrations/0016_add_gemini_provider_type.up.sql`
- `backend/migrations/0007_tenants_and_tenant_users.up.sql`

In production, recommended defaults are:

- `ENABLE_DOCS=false`
- `ENABLE_ADMIN_DEBUG=false`
- configure `AUTH_TOKEN_SECRET` with a strong random value
- optionally set `ADMIN_BOOTSTRAP_USERNAME` / `ADMIN_BOOTSTRAP_PASSWORD` once to create or reset the initial admin account

## Tenant Portal

The repository now includes a first-pass tenant portal for self-service client key management.

Tenant user flow:

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

## Gemini Provider Setup

WcsTransfer now supports the Gemini official native REST API directly.

Recommended provider configuration:

- `provider_type`: `gemini`
- `base_url`: `https://generativelanguage.googleapis.com`
- `extra_config`:

```json
{
  "gemini_api_version": "v1beta"
}
```

Important:

- do not set `base_url` to a full endpoint such as `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-pro:generateContent`
- the gateway appends `/v1beta/models/{model}:generateContent` automatically
- the public gateway routes are:
  - `POST /v1/gemini/generate-content`
  - `POST /v1/gemini/stream-generate-content`

Example public request:

```powershell
curl.exe -X POST http://localhost:3210/v1/gemini/generate-content `
  -H "Authorization: Bearer <client_api_key>" `
  -H "Content-Type: application/json" `
  -d "{\"model\":\"gemini-2.5-pro\",\"contents\":[{\"role\":\"user\",\"parts\":[{\"text\":\"hello\"}]}]}"
```

## Production Deployment

The repository now includes a production stack in `docker-compose.prod.yml` with:

- PostgreSQL
- PostgreSQL backup worker
- Redis
- one-shot migration runner
- backend API
- frontend static console
- Caddy edge proxy with automatic HTTPS

Supporting files:

- `backend/scripts/run-migrations.sh`
- `deploy/backup/backup-postgres.sh`
- `deploy/backup/restore-postgres.sh`
- `deploy/release/preflight-prod.sh`
- `deploy/release/smoke-test.sh`
- `deploy/RELEASE_RUNBOOK.md`
- `deploy/BACKUP_AND_RECOVERY.md`
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
   - `AUTH_TOKEN_SECRET`
   - `ADMIN_BOOTSTRAP_USERNAME`
   - `ADMIN_BOOTSTRAP_PASSWORD`
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
- PostgreSQL logical backups are written periodically to the `postgres_backup_data` volume

Recommended public routes:

- `https://your-domain/console/`
- `https://your-domain/docs`
- `https://your-domain/redoc`
- `https://your-domain/openapi.json`

Operational notes:

- generate `ADMIN_UI_PASSWORD_HASH` with `caddy hash-password --plaintext 'your-password'`
- `PUBLIC_BASE_URL` should match the final external origin exactly
- current CORS policy is set from `PUBLIC_BASE_URL`
- if you already run an external reverse proxy or load balancer, you can reuse the backend/frontend services and skip Caddy
- backup and restore steps are documented in `deploy/BACKUP_AND_RECOVERY.md`
- tenant wallet reconciliation SQL/script lives under `deploy/reconciliation/`
- scheduled tenant wallet reconciliation can be enabled with:
  - `RECONCILIATION_ENABLED=true`
  - `RECONCILIATION_INTERVAL=1h`
  - `RECONCILIATION_DIFF_THRESHOLD=0.0001`
- provider anomaly alerts can be enabled with:
  - `PROVIDER_ALERT_ENABLED=true`
  - `PROVIDER_ALERT_WINDOW=5m`
  - `PROVIDER_ALERT_INTERVAL=1m`
  - `PROVIDER_ALERT_MIN_REQUESTS=10`
  - `PROVIDER_ALERT_429_THRESHOLD=0.2`
  - `PROVIDER_ALERT_5XX_THRESHOLD=0.2`
- tenant wallet / reserve block alerts can be enabled with:
  - `TENANT_WALLET_ALERT_ENABLED=true`
  - `TENANT_WALLET_ALERT_WINDOW=5m`
  - `TENANT_WALLET_ALERT_INTERVAL=1m`
  - `TENANT_WALLET_ALERT_MIN_BLOCKS=5`
  - `TENANT_RESERVE_ALERT_MIN_BLOCKS=5`
- billing debit anomaly alerts can be enabled with:
  - `BILLING_ALERT_ENABLED=true`
  - `BILLING_ALERT_WINDOW=10m`
  - `BILLING_ALERT_INTERVAL=1m`
  - `BILLING_ALERT_MIN_COUNT=1`
  - `BILLING_ALERT_MIN_AMOUNT=0.01`
- dependency alerts can be enabled with:
  - `DEPENDENCY_ALERT_ENABLED=true`
  - `DEPENDENCY_ALERT_INTERVAL=1m`
- external `/healthz` availability monitor can be enabled in production compose with:
  - `HEALTHCHECK_URL=http://caddy/healthz`
  - `HEALTHCHECK_INTERVAL=30`
- reconciliation mismatches can be pushed to a webhook with:
  - `ALERT_WEBHOOK_URL=https://...`
  - `ALERT_WEBHOOK_PROVIDER=generic|wecom|feishu`
  - `ALERT_WEBHOOK_TIMEOUT=5s`
- provider anomaly alerts currently aggregate recent `request_logs` by provider and trigger on rolling-window `429` / `5xx` ratios
- tenant wallet alerts currently aggregate recent `request_logs` by tenant and trigger on rolling-window spikes of:
  - `wallet_empty`
  - `wallet_below_minimum`
  - `wallet_reserve_insufficient`
- billing anomaly alerts currently aggregate recent successful `request_logs` with `billable_amount > 0` and trigger when:
  - the request has no matching `tenant_wallet_ledger` debit row
  - the missing debit count or missing billable amount exceeds the configured threshold
- dependency alerts currently monitor:
  - PostgreSQL ping failures
  - Redis ping failures
- external health watcher monitors the public health endpoint and alerts when `/healthz` becomes unavailable
- recommended webhook targets:
  - `wecom`: 企业微信机器人地址
  - `feishu`: 飞书机器人地址
  - `generic`: your own alert receiver or automation service
- current implementation sends reconciliation mismatch, provider anomaly, tenant wallet / reserve anomaly, billing debit anomaly, and dependency anomaly alerts through webhook when configured; if no webhook is configured, alerts remain in backend logs only
