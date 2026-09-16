# 01 — Project Overview

## Status vocabulary (applies to every doc in `docs/system-design/`)

To keep every doc in this set consistent, findings and design items use exactly one of these markers:

| Marker | Meaning |
| --- | --- |
| ✅ **Implemented** | Confirmed present in the current code/config, verified by reading the actual source. |
| 🔜 **Proposed** | Not yet built. A design recommendation pending approval/implementation. |
| ⛔ **Removed** | Was implemented, then deliberately deleted from the repo. Kept as historical record. |
| 🐞 **Known issue** | Confirmed still broken/missing in the current code — a live bug or gap, not a future idea. |
| 🔶 **Partially resolved** | Part of the original finding was fixed; a specific, named part remains open. Used only when the split is real and worth calling out, never as a hedge. |
| ❓ **Unknown** | Could not be confirmed either way from the repository. |

This document set was originally written as a Phase 1 read-only audit, in which most findings were "CURRENT STATE problems" with "PROPOSED" fixes. Many of those proposed fixes have since been implemented in the code. Every doc in this set has been reconciled against the current code so that fixed items are marked ✅ **Implemented** and only genuinely outstanding items remain 🔜 **Proposed** or 🐞 **Known issue**. Design rationale and the original problem statements are preserved throughout — this is a reconciliation, not a deletion of history.

## What this system is (evidence-based)

A URL shortener with:

- Anonymous or magic-link-verified, API-key-authenticated link creation (custom aliases optional)
- Redis-cached, Postgres-backed redirects, rate-limited per IP
- Asynchronous click analytics via a Redis Stream + consumer group + background worker (with panic recovery)
- Owner-gated per-link analytics and link deletion, alongside an intentionally public aggregate leaderboard
- A React SPA served behind nginx, with the same Go binary handling API + static assets
- Prometheus/Grafana for infrastructure metrics, with provisioned dashboards and alert rules
- CI (GitHub Actions) that tests, builds, and publishes the Docker image to `ghcr.io`; no deploy stage exists today

## Confirmed technology stack (from source, not assumption)

| Layer                      | Technology                                                                                                                  | Evidence                                                                         |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Backend language/framework | Go 1.25 (module `go 1.25.0`), Fiber v2.52                                                                                   | `backend/go.mod`                                                                 |
| ORM                        | `steebchen/prisma-client-go` v0.47 (generated client checked into `.gitignore`d `backend/internal/db`)                      | `backend/prisma/schema.prisma`, `backend/internal/db/.gitignore`                 |
| Database                   | PostgreSQL 16 (alpine image)                                                                                                | `docker-compose.yml`                                                             |
| Cache / coordination       | Redis 7 (alpine, AOF persistence, `requirepass`-protected) via `go-redis/v9`                                                | `docker-compose.yml`, `backend/internal/redis/*.go`                              |
| Async processing           | Redis Stream (`urlshortener:click_events`) + consumer group (`analytics-workers`) drained by an in-process goroutine worker (panic-recovering, still single-consumer) | `backend/internal/redis/stream.go`, `backend/internal/worker/worker.go` |
| Email                      | SMTP via MailHog in local/dev (`internal/mail/`), used for magic-link verification                                          | `backend/internal/mail/`, `docker-compose.yml`                                   |
| Frontend                   | React 18 + Vite 5 + Tailwind 3 + Recharts 2 + framer-motion, built to static assets                                         | `frontend/package.json`                                                          |
| Serving frontend in prod   | Go binary via `app.Static("/", "./frontend/dist")` + manual SPA fallback routes, fronted by nginx                          | `backend/cmd/api/main.go`, `nginx/nginx.conf`                                    |
| Reverse proxy              | nginx (security headers, gzip, single entrypoint on `:80`)                                                                  | `nginx/nginx.conf`, `docker-compose.yml`                                         |
| Observability              | `prometheus/client_golang` counters/histograms exposed at `/metrics`; Prometheus + Grafana containers scrape/visualize, with provisioned dashboards and alert rules | `backend/internal/metrics/prometheus.go`, `docker-compose.yml`, `monitoring/` |
| CI/CD                      | GitHub Actions CI builds/tests and pushes the image to `ghcr.io` on `main`. No deploy workflow exists (one was built and removed — see `11-cicd.md`). | `.github/workflows/ci.yml`                                                       |
| Containerization           | Multi-stage Dockerfile (Node build stage → Go build stage → Alpine runtime, non-root `USER`)                                | `backend/Dockerfile`                                                             |

## Repository layout

```text
backend/
  cmd/api/main.go            entrypoint, route registration, wiring
  internal/auth/             API key generation + hashing
  internal/config/           env var loading
  internal/db/               Prisma-generated client + PrismaStore (implements internal/store interfaces)
  internal/handlers/         HTTP handlers (apikeys, shorten, redirect, stats, health)
  internal/mail/             SMTP mailer (magic-link verification emails)
  internal/metrics/          legacy JSON metrics.Metrics (dead code, see 04/09) + Prometheus collectors
  internal/middleware/       auth, rate limit, metrics (x2), prometheus
  internal/redis/            cache, counter (ID gen), rate limiter, stream, client
  internal/shortcode/        base62 encode/decode
  internal/store/            storage interfaces + domain structs
  internal/worker/           analytics stream consumer (panic-recovering goroutine)
  prisma/schema.prisma       DB schema (source of truth for tables)
frontend/
  src/pages/                 Landing, Login, Dashboard, Stats, TopLinks, Admin
  src/context/, hooks/       Auth (localStorage-based), Toast
  src/api/client.js          axios instance w/ Bearer token + 401 interceptor
docker-compose.yml           postgres, redis, mailhog, migrate, api, nginx, prometheus, grafana
nginx/nginx.conf              reverse proxy: security headers, gzip
monitoring/                   Prometheus alert rules + provisioned Grafana dashboards
.github/workflows/ci.yml      test, build, push image to ghcr.io (no deploy stage)
```

## Functional feature classification

| Feature                                                       | Status                                                       | Evidence                                                                                                                                                                                            |
| --------------------------------------------------------------| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Anonymous URL shortening                                      | ✅ Implemented                                                | `handlers/shorten.go`, no auth required on `/api/shorten` beyond optional key                                                                                                                       |
| Authenticated URL shortening (attributed to a user)           | ✅ Implemented                                                | `shorten.go` sets `UserID` from `middleware.GetAPIKey`                                                                                                                                               |
| Custom alias                                                  | ✅ Implemented                                                | `shorten.go` `resolveShortCode`, Redis `SetNX` reservation                                                                                                                                          |
| Link expiry (`expires_at`)                                    | ✅ Implemented                                                | `shorten.go`, enforced in `GetLinkByShortCode` (treated as not-found post-expiry)                                                                                                                   |
| Redirect w/ cache-aside                                       | ✅ Implemented                                                | `handlers/redirect.go`                                                                                                                                                                              |
| Rate limiting on the redirect route                           | ✅ Implemented                                                | `main.go`: `app.Get("/:shortCode", rateLimitMW, ...)` — previously the one unprotected high-traffic route, now covered (see `05-security.md` SEC-06, resolved) |
| Click analytics (async)                                       | ✅ Implemented                                                | Redis Stream → `worker.go` → Postgres `ClickEvent`                                                                                                                                                  |
| Analytics worker crash isolation                               | ✅ Implemented                                                | `worker.go`'s `run()` wraps its loop in `recover()` and restarts itself, so a panic no longer takes the whole API process down (see `08-reliability.md`) |
| Analytics worker horizontal scaling                            | 🔜 Proposed                                                   | Still a single goroutine with a hardcoded `ConsumerName: "worker-1"` inside the API process — not yet extracted into its own deployable (see `08-reliability.md`, ADR-0003) |
| Email verification before API-key issuance                    | ✅ Implemented                                                | `handlers/apikeys.go`: `POST /api/keys` only sends a magic-link email; `GET /api/keys/verify` issues the key after the token is consumed (see `05-security.md` SEC-01, resolved) |
| Per-link stats (`/api/stats/:shortCode`)                       | ✅ Implemented, owner-gated                                   | `stats.go GetLinkStats`, route requires auth + `ownsLink` check, 404s for non-owners (see `05-security.md` SEC-02a, resolved) |
| Top links (`/api/stats/top`)                                  | ✅ Implemented, public by design                              | `stats.go GetTopLinks` — intentionally public, aggregate-only, no ownership required |
| User's own links (`/api/links`)                               | ✅ Implemented                                                | `stats.go GetUserLinks`, requires API key                                                                                                                                                           |
| Recent click events per link (`/api/links/:shortCode/clicks`) | ✅ Implemented, owner-gated                                   | `stats.go GetRecentClicks`, requires auth + `ownsLink` check (see `05-security.md` SEC-02b, resolved) |
| Delete a link                                                 | ✅ Implemented                                                | `DELETE /api/links/:shortCode` (owner-gated, hard delete, invalidates cache) wired in `main.go`/`stats.go DeleteLink`; `DashboardPage.jsx handleDelete` calls the real endpoint instead of only updating local state |
| IP hash salting for click events                               | ✅ Implemented                                                | `redis.HashIP` now takes an `IPHashSecret` (HMAC-style), configured via `IP_HASH_SECRET`, instead of a plain unsalted hash |
| Analytics query performance (`GetRecentClickEvents`)            | ✅ Implemented                                                | Now DB-side `ORDER BY timestamp DESC LIMIT N` backed by the `click_events(link_id, timestamp)` composite index, instead of an unbounded fetch + in-Go sort |
| Analytics query performance (`GetTopLinks`, `GetLinkStats`)     | 🐞 Known issue                                                | Still `FindMany()` with no `WHERE`/`LIMIT` pushed to SQL — `GetTopLinks` loads **every** link and **every** click event into Go memory to count/sort; `GetLinkStats` loads **every** click event for a link to aggregate/sort in Go. Not fixed — see `03-data-model.md`/`07-scalability.md`/ADR-0001 |
| Password-based user login                                     | 🐞 Known issue / schema-only                                  | `User.passwordHash` field exists in `schema.prisma`; `GetOrCreateUserByEmail` always writes `""`; no login handler exists — unchanged since the original audit |
| Password-protected links                                      | 🐞 Known issue / schema-only                                  | `Link.passwordHash` exists in schema and is round-tripped by `CreateLink`/`GetLinkByShortCode`, but no handler ever checks it — unchanged since the original audit |
| Admin role / privilege separation                             | 🐞 Known issue                                                | No `role`/`isAdmin` field in `schema.prisma`; no authorization concept found anywhere in `backend/internal`; `/admin` frontend route is gated only by `ProtectedRoute` (any authenticated key) — unchanged |
| System metrics UI (`/admin` page)                              | 🐞 Known issue                                                | `AdminPage.jsx` still expects JSON (`metrics.uptime_seconds`, `metrics.cache_hit_rate`, …) but `GET /metrics` still returns raw Prometheus text-exposition format — unchanged, see `04-api-design.md` |
| Structured logging                                             | 🐞 Known issue                                                | Still only Fiber's default access logger (`logger.New()`) and scattered `log.Printf`; deferral comments in `ratelimit.go`/`redirect.go` remain — unchanged |
| CI pipeline (test/build/push image)                            | ✅ Implemented                                                | `.github/workflows/ci.yml` — `go test`, `docker build`, push to `ghcr.io` on `main`                                                                                                                  |
| CD / deployment pipeline                                       | ⛔ Removed                                                    | A self-hosted-runner/simulated-VM deploy stage was implemented and then deleted (`deploy/`, `.github/workflows/deploy.yml`); deployment approach is currently undecided — see `06-deployment.md`/`11-cicd.md` |
| Automated tests beyond a handful of unit-test files             | 🐞 Known issue                                                | See `10-testing-strategy.md`                                                                                                                                                                        |

## Engineering maturity classification

**Classification: INTERMEDIATE, trending toward SENIOR** in the hardened areas. The original audit's SENIOR-level design touches (Redis-based ID generation, sliding-window rate limiting, stream-based async analytics, cache-aside redirect path, multi-stage Docker build) are now joined by a real security-fix pass (magic-link verification, owner-gated analytics, salted IP hashing, rate-limited redirects, non-root container, env-driven CORS, worker panic recovery, Postgres+Redis health checks) and real operational maturity (CI publishing images, Prometheus alerting, provisioned Grafana dashboards, MailHog for local email testing). What still holds this back from SENIOR/PRODUCTION READY: the `GetTopLinks`/`GetLinkStats` full-table-scan queries are unfixed, there is no automated test coverage beyond a few leaf-utility files, the `/admin` metrics UI is still broken, structured logging/request IDs are still missing, and the deployment story is currently undecided after the self-hosted-runner approach was tried and removed. See `system-design` docs 05–11 for full evidence.
