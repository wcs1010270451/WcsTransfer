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
