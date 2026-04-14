# WcsTransfer

WcsTransfer is a model gateway project with:

- `backend/`: Go + Gin gateway service
- `frontend/`: React admin console

The current repository includes a runnable backend skeleton with:

- health and version endpoints
- OpenAI-style API placeholders
- admin API placeholders
- basic request ID and admin auth middleware
- PostgreSQL schema migration
- Docker Compose for local development

## Backend Quick Start

```powershell
cd backend
go mod tidy
go run ./cmd/server
```

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
