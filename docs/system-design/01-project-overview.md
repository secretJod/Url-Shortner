# 01 — Project Overview

## What this system is (evidence-based)

A URL shortener with:

- Anonymous or API-key-authenticated link creation (custom aliases optional)
- Redis-cached, Postgres-backed redirects
- Asynchronous click analytics via a Redis Stream + consumer group + background worker
- A React SPA served by the same Go binary in production
- Prometheus/Grafana for infrastructure metrics

## Confirmed technology stack (from source, not assumption)

| Layer                      | Technology                                                                                                                  | Evidence                                                                         |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Backend language/framework | Go 1.25 (module `go 1.25.0`), Fiber v2.52                                                                                   | `backend/go.mod`                                                                 |
| ORM                        | `steebchen/prisma-client-go` v0.47 (generated client checked into `.gitignore`d `backend/internal/db`)                      | `backend/prisma/schema.prisma`, `backend/internal/db/.gitignore`                 |
| Database                   | PostgreSQL 16 (alpine image)                                                                                                | `docker-compose.yml`                                                             |
| Cache / coordination       | Redis 7 (alpine, AOF persistence) via `go-redis/v9`                                                                         | `docker-compose.yml`, `backend/internal/redis/*.go`                              |
| Async processing           | Redis Stream (`urlshortener:click_events`) + consumer group (`analytics-workers`) drained by an in-process goroutine worker | `backend/internal/redis/stream.go`, `backend/internal/worker/worker.go`          |
| Frontend                   | React 18 + Vite 5 + Tailwind 3 + Recharts 2 + framer-motion, built to static assets                                         | `frontend/package.json`                                                          |
| Serving frontend in prod   | Go binary via `app.Static("/", "./frontend/dist")` + manual SPA fallback routes                                             | `backend/cmd/api/main.go`                                                        |
| Observability              | `prometheus/client_golang` counters/histograms exposed at `/metrics`; Prometheus + Grafana containers scrape/visualize      | `backend/internal/metrics/prometheus.go`, `docker-compose.yml`, `prometheus.yml` |
| CI/CD                      | **None** — no `.github/` directory anywhere in the repo                                                                     | `find` of repo root                                                              |
| Containerization           | Multi-stage Dockerfile (Node build stage → Go build stage → Alpine runtime)                                                 | `backend/Dockerfile`                                                             |

## Repository layout

```text
backend/
  cmd/api/main.go            entrypoint, route registration, wiring
  internal/auth/             API key generation + hashing
  internal/config/           env var loading
  internal/db/               Prisma-generated client + PrismaStore (implements internal/store interfaces)
  internal/handlers/         HTTP handlers (apikeys, shorten, redirect, stats, health)
  internal/metrics/          legacy JSON metrics.Metrics + Prometheus collectors
  internal/middleware/       auth, rate limit, metrics (x2), prometheus
  internal/redis/            cache, counter (ID gen), rate limiter, stream, client
  internal/shortcode/        base62 encode/decode
  internal/store/            storage interfaces + domain structs
  internal/worker/           analytics stream consumer
  prisma/schema.prisma       DB schema (source of truth for tables)
frontend/
  src/pages/                 Landing, Login, Dashboard, Stats, TopLinks, Admin
  src/context/, hooks/       Auth (localStorage-based), Toast
  src/api/client.js          axios instance w/ Bearer token + 401 interceptor
docker-compose.yml           postgres, redis, api, prometheus, grafana
prometheus.yml                scrape config
```

## Deleted-but-uncommitted documentation

`git status` shows numerous docs deleted from the working tree but not committed (`.ai/*`, `KNOWLEDGE.md`, `PROJECT_OVERVIEW.md`, `README.md`, `docs/*`, etc.). Per CLAUDE.md §2, these are **not** used as evidence of current architecture in this audit. They were consulted read-only via `git show HEAD:<path>` only for historical color where noted, never as a substitute for reading the actual code — every claim in this document set is backed by a live source file.

## Functional feature classification

| Feature                                                       | Status                                                       | Evidence                                                                                                                                                                                            |
| ------------------------------------------------------------- | ------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Anonymous URL shortening                                      | IMPLEMENTED                                                  | `handlers/shorten.go`, no auth required on `/api/shorten` beyond optional key                                                                                                                       |
| Authenticated URL shortening (attributed to a user)           | IMPLEMENTED                                                  | `shorten.go:82-84` sets `UserID` from `middleware.GetAPIKey`                                                                                                                                        |
| Custom alias                                                  | IMPLEMENTED                                                  | `shorten.go` `resolveShortCode`, Redis `SetNX` reservation                                                                                                                                          |
| Link expiry (`expires_at`)                                    | IMPLEMENTED                                                  | `shorten.go`, enforced in `GetLinkByShortCode` (treated as not-found post-expiry)                                                                                                                   |
| Redirect w/ cache-aside                                       | IMPLEMENTED                                                  | `handlers/redirect.go`                                                                                                                                                                              |
| Click analytics (async)                                       | IMPLEMENTED                                                  | Redis Stream → `worker.go` → Postgres `ClickEvent`                                                                                                                                                  |
| Per-link stats (`/api/stats/:shortCode`)                      | IMPLEMENTED but **UNAUTHENTICATED**                          | `stats.go GetLinkStats`, route has no `authMW`                                                                                                                                                      |
| Top links (`/api/stats/top`)                                  | IMPLEMENTED, public by design                                | `stats.go GetTopLinks`                                                                                                                                                                              |
| User's own links (`/api/links`)                               | IMPLEMENTED                                                  | `stats.go GetUserLinks`, requires API key                                                                                                                                                           |
| Recent click events per link (`/api/links/:shortCode/clicks`) | IMPLEMENTED but **UNAUTHENTICATED**                          | `stats.go GetRecentClicks`, no `authMW` on route                                                                                                                                                    |
| API-key "login" (email → key, no verification)                | IMPLEMENTED (as an identity-minting endpoint, not real auth) | `handlers/apikeys.go`                                                                                                                                                                               |
| Password-based user login                                     | **MISSING / SCHEMA-ONLY**                                    | `User.passwordHash` field exists in `schema.prisma`; `prisma_store.go:130` always writes `""`; grep shows zero reads of `PasswordHash` for `User` anywhere; no login handler exists                 |
| Password-protected links                                      | **MISSING / SCHEMA-ONLY**                                    | `Link.passwordHash` exists in schema and is round-tripped by `CreateLink`/`GetLinkByShortCode`, but no handler ever checks it — a "password protected" link redirects with zero password prompt     |
| Delete a link                                                 | **BROKEN (frontend fakes it)**                               | `DashboardPage.jsx handleDelete` only does `setLinks(links.filter(...))` — no API call. No `DELETE` route exists in `main.go` at all. Refreshing the page brings the "deleted" link right back      |
| Admin role / privilege separation                             | **MISSING**                                                  | No `role`/`isAdmin` field in `schema.prisma`; grep for role/admin concepts in `backend/internal` returns nothing; `/admin` frontend route is gated only by `ProtectedRoute` (any authenticated key) |
| System metrics UI (`/admin` page)                             | **BROKEN**                                                   | `AdminPage.jsx` expects JSON (`metrics.uptime_seconds`, `metrics.cache_hit_rate`, …) but `GET /metrics` returns raw Prometheus text-exposition format (see `04-api-design.md`)                      |
| Structured logging                                            | **MISSING (explicitly deferred in comments)**                | `grep "structured logging"` hits two `// TODO`-style comments in `ratelimit.go` and `redirect.go`; only Fiber's default access logger (`logger.New()`) and stray `log.Printf` exist                 |
| CI/CD pipeline                                                | MISSING                                                      | No `.github/` directory                                                                                                                                                                             |
| Automated tests beyond 4 unit-test files                      | MISSING                                                      | See `10-testing-strategy.md`                                                                                                                                                                        |

## Engineering maturity classification

**Classification: INTERMEDIATE**, with pockets of SENIOR-level design (Redis-based ID generation, sliding-window rate limiting, stream-based async analytics, cache-aside redirect path, multi-stage Docker build with a documented Alpine/musl Prisma gotcha) undermined by BEGINNER-level gaps (no admin/authz model, no tests beyond leaf utilities, unbounded full-table scans in every analytics query, a UI feature that reads the wrong endpoint format, a fake delete). See `system-design` docs 05–10 for full evidence per CLAUDE.md §19.
