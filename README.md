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
- `POST /api/shorten` — body: `{"url": "https://...", "custom_alias": "optional", "expires_at": "optional RFC3339"}`
- `GET  /:shortCode` — redirects to the long URL (cache-first via Redis, Postgres fallback)

## Full stack via Docker

```bash
docker-compose up --build
```

## Status

Phase 1 complete: base62 short code generation, Redis cache + counter, Prisma-backed Postgres storage,
real `/api/shorten` and `/:shortCode` handlers. **Note:** run `go run github.com/steebchen/prisma-client-go generate`
before your first build — see `PROJECT_OVERVIEW.md` §8. See `PROJECT_OVERVIEW.md` §4 for the full phase tracker.
