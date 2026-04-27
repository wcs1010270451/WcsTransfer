# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

WcsTransfer is an AI model gateway — a self-hosted proxy that sits between API clients and upstream LLM providers (OpenAI, Anthropic, Gemini). It exposes OpenAI-compatible endpoints, manages provider key rotation and health tracking, handles multi-tenant billing via a wallet system, and provides an admin console.

## Development Commands

### Backend (Go)

```bash
cd backend
go mod tidy          # sync dependencies
go run ./cmd/server  # start the server
go test ./...        # run all tests
go test ./internal/service/reconciliation/...  # run a single package's tests
go build ./...       # compile check
```

Requires Go 1.23+ (toolchain 1.24.5).

### Frontend (React + Vite)

```bash
cd frontend
npm install
npm run dev      # dev server (default: http://localhost:5173)
npm run build    # production build
npm run preview  # preview the production build
```

### Local Docker Compose (full stack)

```bash
docker compose up -d --build
```

Services: PostgreSQL on `localhost:5432`, Redis on `localhost:6379`, backend API on `http://localhost:8080`.

## Architecture

### Backend (`backend/`)

Entry point: `cmd/server/main.go` → `internal/app/server.go` → `internal/router/router.go`

**Layer structure:**
- `internal/config/` — all config loaded from env vars; `config.Config` is the single config struct passed everywhere
- `internal/platform/` — dependency wiring: PostgreSQL pool + Redis client
- `internal/repository/interfaces.go` — store interfaces (AdminStore, ClientAuthStore, PublicModelStore, RequestLogWriter, TenantClientKeyStore, etc.)
- `internal/repository/postgres/store.go` — single `Store` struct that implements all interfaces; migrations auto-run on startup
- `internal/entity/gateway.go` — all domain types (Provider, ProviderKey, Model, ClientAPIKey, Tenant, RequestLog, etc.)
- `internal/api/` — Gin handlers, one sub-package per route group: `admin/`, `adminauth/`, `openai/`, `system/`, `tenant/`
- `internal/middleware/` — request ID, CORS, admin JWT auth, tenant JWT auth, public API key auth, quota enforcement
- `internal/service/` — background services: reconciliation, alerting (webhook), provideralert, walletalert, billingalert, dependencyalert, clientquota (Redis-backed), keyhealth (in-memory cooldown tracker)

**Request flow for proxy endpoints (`/v1/*`):**
1. `middleware.PublicAPIAuth` — validates the client API key against Postgres, attaches `ClientAPIKey` to context
2. `middleware.PublicAPIQuota` — checks RPM/daily token/daily request limits via Redis
3. `openai.Handler` — resolves model route from DB, picks a provider key (with cooldown awareness via `keyhealth.Tracker`), forwards request to upstream, logs result and debits tenant wallet

**Provider types:** `openai`, `anthropic`, `gemini` — each has dedicated proxy logic in `internal/api/openai/` (`handler.go`, `anthropic.go`, `gemini.go`)

**Auth:** Both admin and tenant users authenticate with JWT; the same `AUTH_TOKEN_SECRET` signs both. Admin tokens use `adminauth` service, tenant tokens use `tenantauth` service.

**Background services** (all start via `server.go` on boot if enabled):
- `reconciliation` — periodically cross-checks wallet balance against ledger entries
- `provideralert` — detects rolling-window 429/5xx anomalies per provider
- `walletalert` — detects wallet-empty and reserve-insufficient spikes per tenant
- `billingalert` — detects request logs with missing ledger debit rows
- `dependencyalert` — pings Postgres and Redis, alerts on failure
- All alert services push to a webhook via `alerting.WebhookNotifier` (supports `generic`, `wecom`, `feishu`)

**Migrations:** Sequential numbered SQL files in `backend/migrations/`. Applied automatically at startup via `postgres/bootstrap.go`.

### Frontend (`frontend/src/`)

React 19 + Vite + Ant Design 5 + React Router 7 + Zustand.

**Two separate auth contexts:**
- Admin console (`/dashboard`, `/providers`, `/tenants`, `/keys`, `/models`, `/client-keys`, `/logs`, `/debug`, `/docs`) — guarded by `AdminGuard`, token in `adminAuthStore`
- Tenant portal (`/portal/keys`) — guarded by `PortalGuard`, token in `portalAuthStore`

`api/client.js` — Axios instance, reads `VITE_API_BASE_URL` from env, attaches bearer token from the appropriate store.

### Environment Files

| File | Used when |
|---|---|
| `backend/.env` | `go run ./cmd/server` direct run |
| `frontend/.env` | `npm run dev` direct run |
| `.env` (repo root) | `docker compose up` (local) |
| `.env.prod` (repo root) | `docker compose -f docker-compose.prod.yml up` (production) |

CORS defaults allow `localhost:3211`. The frontend dev port is `5173` by default (set `VITE_API_BASE_URL` in `frontend/.env` to point at the backend).

## Key Conventions

- All repository access goes through the interfaces in `internal/repository/interfaces.go`; the Postgres store in `internal/repository/postgres/store.go` is the only implementation
- `entity.go` is the single source of truth for all domain types — add new fields there and in the corresponding migration
- Config validation in `config.Validate()` enforces production-only rules (no wildcard CORS, no insecure secrets, no docs enabled)
- `keyhealth.Tracker` is in-memory only; key cooldown state is lost on restart by design
- `clientquota.Service` uses Redis sliding-window counters; quota keys are keyed by client API key ID
- Alert services are all opt-in via env vars; they default to enabled in `APP_ENV=production` and disabled otherwise
