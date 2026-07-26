# URL Shortener (Enterprise)

Go + Fiber + Prisma + PostgreSQL + Redis. See `PROJECT_OVERVIEW.md` for full architecture, phase plan, and session log — **read that file first** if you're picking this project back up.

## Local setup

```bash
cp .env.example .env
docker-compose up -d postgres redis

# generate Prisma client (Go) — once schema is finalized in Phase 1
go run github.com/steebchen/prisma-client-go generate

go run ./cmd/api
```

Health check: `GET http://localhost:8080/health`

## Full stack via Docker

```bash
docker-compose up --build
```

## Status

Phase 0 complete: project scaffold, Docker Compose (Postgres + Redis), Fiber server with `/health`, Prisma schema drafted. See `PROJECT_OVERVIEW.md` §4 for phase tracker.
