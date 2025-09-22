# Project Template

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white&labelColor=0b1021)](https://go.dev)
[![Chi](https://img.shields.io/badge/chi-router-3b5bdb?labelColor=0b1021)](https://github.com/go-chi/chi)
[![Gorm](https://img.shields.io/badge/Gorm-Postgres-0C68C7?logo=postgresql&logoColor=white&labelColor=0b1021)](https://gorm.io)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white&labelColor=0b1021)](https://redis.io)
[![Next.js](https://img.shields.io/badge/Next.js-15-000000?logo=nextdotjs&logoColor=white&labelColor=0b1021)](https://nextjs.org)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white&labelColor=0b1021)](https://react.dev)

> Template project I use to bootstrap new apps fast. It handles auth, team management, frontend dashboard with basic features (notifications, account management, etc.)


A full‑stack template with a Go (Chi + Gorm + Postgres + Redis) backend and a Next.js 15 frontend

> ⚠️ Actively evolving; APIs and UX may change.

## What's inside
- Backend: Chi router, middleware (CORS, rate‑limit, auth), Gorm models/migrations, sessions/JWT, OAuth (Goth), email (Resend)
- Frontend: Next.js App Router, Tailwind, API proxy via rewrites, cookie‑based auth
- Infra: Postgres, Redis, Docker Compose

## Quick start
- Create a `.env` at repo root (see minimal example below). Then:

```bash
docker compose up --build
# App at http://localhost:3000, API at http://localhost:8080
```

Local dev without Docker:

```bash
# Backend
cd backend && go run ./...

# Frontend (in another terminal)
cd frontend && npm install
BACKEND_URL=http://localhost:8080 npm run dev
```

## Stack
- Go 1.24, Chi router, Gorm (Postgres), Redis, JWT, Goth (OAuth), Resend (email)
- Next.js 15, React 19, Tailwind CSS

## Environment variables
The backend loads configuration from a `.env` file (see `backend/config/config.go`). Many are required. Below is a minimal example for local development. Adjust as needed.

```env
# --- Server ---
BACKEND_ADDR=:8080
# BACKEND_PUBLIC_URL=http://localhost:8080 # optional override when not using WorkOS-managed redirects
APP_ENV=dev
APP_READ_TIMEOUT_S=15
APP_WRITE_TIMEOUT_S=30
APP_IDLE_TIMEOUT_S=60

# --- Database ---
DB_NAME=dbname
DB_USER=postgres
DB_PASS=postgres
DB_HOST=localhost
DB_PORT=5432

# --- Redis ---
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASS=
REDIS_DB=0
REDIS_SECRET=dev-redis-secret-change-me

# --- App metadata ---
APP_NAME=AppName
APP_URL=localhost:3000

# --- Secrets ---
RESEND_API_KEY=your-resend-api-key
WORKOS_API_KEY=your-workos-api-key
WORKOS_CLIENT_ID=your-workos-client-id
WORKOS_COOKIE_SECRET=replace-with-long-random-string
WORKOS_GOOGLE_CONNECTION_ID=optional-google-connection-id
WORKOS_GITHUB_CONNECTION_ID=optional-github-connection-id
WORKOS_DEFAULT_REDIRECT_PATH=/auth/ready
```

Frontend uses a proxy rewrite (see `frontend/next.config.ts`):
- `NEXT_PUBLIC_API_BASE_URL` defaults to `/api`.
- `BACKEND_URL` defaults to `http://backend:8080` (good for Docker). In local dev, you can set `BACKEND_URL=http://localhost:8080`.

## Quick start with Docker
1. Create `.env` in the repo root using the example above.
2. Build and start:
   ```bash
   docker compose up --build
   ```
3. Open the app: http://localhost:3000

Notes:
- `docker-compose.yml` builds both `backend` and `frontend`, and brings up Postgres and Redis.
- The frontend proxies `/api/*` to the backend inside the Compose network.

## API overview
Key routes (see `backend/api/routes.go`):
- `GET /health`
- `POST /feedback` (requires verified account)
- `POST /auth/signup`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `POST /auth/password/reset`
- `POST /auth/password/confirm`
- `POST /auth/verify/resend`
- `POST /auth/verify/send` (requires auth)
- `POST /auth/verify/confirm` (requires auth)
- `GET /auth/me` (requires auth)
- `GET /dashboard/overview` (requires verified account)
- Team management under `/teams/*` (requires verified account)
- Account management under `/account/*` (requires verified account)
- Notifications under `/notifications/*` (requires verified account)

Global middleware includes CORS, rate limiting, real IP, recoverer, and auth (see `backend/api/routes.go`)

## Troubleshooting
- 401/403 on protected routes: ensure cookies are being sent. The frontend uses same-origin proxying via Next rewrites.
- 5xx on startup: check `.env` completeness; many fields are required by `config.Load()`.
- DB connection errors: verify Postgres envs and that the service is reachable.
- Email/OAuth flows: require valid external service credentials for Resend and Google
