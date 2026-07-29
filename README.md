# URL Shortener (Enterprise)

Go + Fiber + Prisma + PostgreSQL + Redis. See `PROJECT_OVERVIEW.md` for full architecture, phase plan, and session log — **read that file first** if you're picking this project back up.

## Local setup

```bash
cp .env.example .env
docker-compose up -d postgres redis

# REQUIRED before first build — generates the Prisma Go client into internal/db
go run github.com/steebchen/prisma-client-go generate

# push the schema to Postgres (dev-friendly, no migration files)
go run github.com/steebchen/prisma-client-go db push

go build ./...
go run ./cmd/api
```

## API

- `GET  /health` — Redis connectivity check
- `POST /api/keys` — body: `{"email": "you@example.com"}` — issues a new API key (shown once)
- `POST /api/shorten` — optional `Authorization: Bearer <api_key>` header; body: `{"url": "https://...", "custom_alias": "optional", "expires_at": "optional RFC3339"}`
- `GET  /:shortCode` — redirects to the long URL (cache-first via Redis, Postgres fallback)

Rate limits: 20 req/min per IP (anonymous), 60 req/min per key (standard tier), 600 req/min (pro tier).

## Full stack via Docker

```bash
docker-compose up --build
```

## Status

Phase 2 complete: API key issuance, optional bearer-token auth, Redis sliding-window rate limiting.
See `PROJECT_OVERVIEW.md` §4 for the full phase tracker and §10 for what to verify next.
