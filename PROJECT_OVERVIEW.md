# URL Shortener — Project Overview

> **Purpose of this file:** This is the single source of truth for the project.
> At the start of every session, read this file first to restore full context
> instead of re-explaining the project. Update it at the end of every phase.

---

## 1. Stack Decisions (locked in)

| Layer | Choice | Notes |
|---|---|---|
| Language | Go | Chosen for raw speed + concurrency |
| Web framework | Fiber | Fast, Express-like ergonomics |
| ORM | Prisma (prisma-client-go) | Community-maintained Go client. Fallback plan: swap to `ent` if we hit maturity issues |
| Primary DB | PostgreSQL | Persistent store for links, users, API keys |
| Cache / hot path | Redis | Redirect cache, ID counter, rate limiting |
| Containerization | Docker + Docker Compose | Local dev + eventual deploy |
| Analytics queue | Redis Streams (or lightweight pub/sub) | Async click tracking, doesn't block redirects |

---

## 2. Architecture

```
Client → Load Balancer → Go API (stateless, N instances)
                              ├─ Redis: short_code lookup cache (GET, sub-ms)
                              ├─ Redis: INCR counter → base62 encode → short_code
                              ├─ Redis: rate limiting (sliding window per API key/IP)
                              ├─ Postgres (via Prisma): source of truth for links/users/keys
                              └─ Redis Stream → Analytics Worker → Postgres analytics tables
```

**Hot path (redirect) flow:**
1. Request hits `GET /:shortCode`
2. Check Redis cache → if hit, 302 redirect immediately (~sub-ms)
3. If miss, query Postgres, backfill Redis cache, then redirect
4. Fire-and-forget: push click event to Redis Stream for async analytics

**Write path (shorten) flow:**
1. `POST /api/shorten` with long URL (+ optional custom alias/expiry)
2. If custom alias: check availability, else `INCR` global counter in Redis
3. Base62-encode counter value → short code (collision-free by construction)
4. Write to Postgres via Prisma, write-through to Redis cache
5. Return short URL

---

## 3. Data Model (initial draft)

**Link**
- id (bigint, PK — same value used for base62 short code)
- short_code (unique, indexed)
- long_url
- user_id (nullable, FK — anonymous links allowed)
- custom_alias (bool)
- expires_at (nullable)
- password_hash (nullable)
- created_at

**User**
- id, email, password_hash, created_at

**ApiKey**
- id, key_hash, user_id, rate_limit_tier, created_at

**ClickEvent** (analytics, written async)
- id, link_id, timestamp, referrer, country, device_type, ip_hash

---

## 4. Build Phases & Status

- [x] **Phase 0** — Scaffold, this overview file, Docker setup, Fiber server + Redis client wired, `/health` endpoint, builds clean
- [x] **Phase 1** — Core shorten + redirect API (base62 ID gen, Postgres via Prisma, Redis cache) *(verified working end-to-end: POST /api/shorten → Postgres write → cache write-through → GET redirect → 302 confirmed on Karan's machine)*
- [x] **Phase 2** — Auth (API keys) + rate limiting *(code complete, builds clean — needs the same local Prisma codegen re-run + a live test, see session log)*
- [x] **Phase 3** — Async analytics pipeline (Redis Streams → worker → Postgres) *(verified live end-to-end: click events flowing from redirects to Postgres)*
- [x] **Phase 4** — Admin/stats API *(verified live: GET /api/stats/:code, /api/stats/top, /api/links, /api/links/:code/clicks all working)*
- [ ] **Phase 5** — Load testing, caching tuning, observability (metrics/logging)
- [ ] **Phase 6** — Deployment hardening (Docker Compose finalized, optional K8s manifests)

---

## 5. Project Structure

```
urlshortener/
├── PROJECT_OVERVIEW.md      ← you are here
├── cmd/api/                 ← main.go, app entrypoint
├── internal/
│   ├── handlers/             ← HTTP handlers: health, shorten, redirect
│   ├── redis/                  ← Redis client, ID counter (INCR), cache, alias reservation
│   ├── shortcode/              ← base62 encode/decode (unit tested)
│   ├── store/                  ← LinkStore interface (decouples handlers from Prisma)
│   ├── db/                     ← Prisma-backed LinkStore implementation (needs codegen — see §8)
│   ├── middleware/             ← auth, rate limiting, logging (Phase 2+)
│   └── config/                 ← env/config loading
├── prisma/
│   └── schema.prisma          ← DB schema (source of truth for Postgres), output → internal/db
├── docker-compose.yml
├── go.mod
└── .env.example
```

---

## 6. Open Decisions / Things to Revisit
- Prisma Go client maturity — revisit if we hit blockers, fallback is `ent`
- Custom domains — deferred to a later phase
- Deployment target (K8s vs simple Docker Compose on a VM) — not yet decided

---

## 7. Session Log
- **Session 1**: Locked stack (Go + Fiber + Prisma + Postgres + Redis). Scaffolded project structure.
  Phase 0 completed: `go.mod` set up, Fiber + go-redis wired, `/health` endpoint live, `docker-compose.yml`
  (Postgres + Redis + api), Dockerfile (multi-stage), Prisma schema drafted (Link/User/ApiKey/ClickEvent).
  Verified with `go build` and `go vet` — both clean. Next session: Phase 1 (base62 short code generation
  via Redis INCR, wire Prisma client, implement real `/api/shorten` and `/:shortCode` handlers).

- **Session 3**: Debugged Phase 1 all the way to a verified working end-to-end test. In order:
  1. Fixed two real Prisma-client-go usage bugs found on first real build (couldn't be caught in the
     sandbox since codegen wasn't possible there): (a) `CreateOne` — Go disallows mixing an individual
     variadic arg with a spread slice in one call; moved all optional `Set()` params into one slice.
     (b) `User.ID.Equals` needed an explicit `types.BigInt` conversion, not a plain Go `int`.
  2. Fixed "ensure: no binary found" at container runtime — `prisma generate` had been run on the host
     (macOS/arm64), embedding the wrong platform's query-engine binary. Fix: moved `prisma-client-go
     prefetch` + `generate` into the Dockerfile's builder stage itself (`golang:1.22-alpine`), per
     Prisma's documented Docker pattern, so the embedded engine matches the container's actual platform.
  3. Fixed api container connecting to `localhost:5432`/`localhost:6379` instead of the Postgres/Redis
     *containers* — `.env` is correct for host-side `go run`, but wrong once loaded into a container via
     `env_file` (inside Docker's network, services are reachable by service name). Fix: added an
     `environment:` override block on the `api` service in `docker-compose.yml` for just those two vars.
  4. Fixed `relation "public.links" does not exist` — `prisma generate` (builds the Go client) and
     `prisma db push` (creates the actual Postgres tables) are separate steps; only the former had been
     run. Running `db push` locally (needs `DATABASE_URL` pointing to `localhost:5432`, i.e. host-side,
     while `docker compose`'s Postgres port is mapped out to the host) created the tables.
  5. **Verified end-to-end**: `POST /api/shorten` → `201` with real `short_url`/`short_code`. `GET
     /<short_code>` → `302 Found` with correct `Location` header. Full pipeline confirmed working.

  **Workflow note for next time**: after `docker compose up --build`, if tables don't exist yet, run
  `go run github.com/steebchen/prisma-client-go db push` locally (host-side, not in Docker) once, with
  the Postgres container already up and its port mapped to the host. This is a manual step for now —
  worth switching to proper `prisma migrate` files in a later phase so it's automatic.

## 9. Repo & credentials note
This project lives at `github.com/secretJod/Url-Shortner` (public repo). Claude pushes commits directly
using a short-lived, repo-scoped fine-grained PAT that Karan provides fresh each session — the token is
never stored between sessions, and Karan revokes it after each session ends.

## 10. Session Log (continued)
- **Session 4**: Phase 2 built — API keys + rate limiting.

  **Scope decision**: no full signup/login/password flow yet. `POST /api/keys` takes just an email,
  get-or-creates a `User` (with an empty password placeholder — full auth is a separate future decision,
  not part of Phase 2), generates a random 256-bit API key, and returns the raw key exactly once. Only
  its SHA-256 hash is ever stored. This matches how most API-first products (Stripe, bit.ly, etc.) start.

  **What was built**:
  - `internal/auth` — API key generation (`usk_`-prefixed, 256-bit random) + SHA-256 hashing + bearer
    token extraction. Unit tested (randomness, hash consistency, header parsing edge cases).
  - `internal/redis/ratelimit.go` — sliding-window-log rate limiter using a Redis sorted set (ZADD +
    ZREMRANGEBYSCORE + ZCARD), more accurate than fixed-window counters (no boundary-burst problem).
    Integration-tested against a real local Redis (under-limit, over-limit, window-slide-expiry cases).
  - `internal/store` — extended with `User`/`ApiKey` types and `UserStore`/`ApiKeyStore` interfaces;
    `PrismaStore` now implements the full combined `Store` interface.
  - `internal/middleware/auth.go` — `OptionalAPIKeyAuth`: anonymous requests pass through; requests
    WITH an `Authorization` header that fails to validate are rejected (no silent fallback to anonymous
    for a malformed/invalid key — that would hide misconfigured clients).
  - `internal/middleware/ratelimit.go` — per-API-key-tier limits (`standard`=60/min, `pro`=600/min) or
    per-IP for anonymous (20/min). Fails open on Redis errors (a cache hiccup shouldn't take down the API).
  - `POST /api/keys` and `POST /api/shorten` now go through `authMW` (shorten only) then `rateLimitMW`.
    Links created with a valid API key get `UserID` set, associating them with that user.

  **Verified in sandbox**: all new packages (`auth`, `redis` incl. rate limiter, `store`, `middleware`,
  `handlers`, `cmd/api`) build and vet clean; `internal/db` fails only on the same known missing-codegen
  symbols as before (confirms no new bugs, same as the Phase 1 pattern). **Not yet live-tested end-to-end**
  on Karan's machine — do that next: `git pull`, rebuild, then hit `POST /api/keys` and confirm rate
  limiting kicks in on `POST /api/shorten` after enough anonymous requests.

- **Session 5**: Phase 3 built — async analytics pipeline (Redis Streams → worker → Postgres).

  **What was built**:
  - `internal/redis/stream.go` — Redis Stream producer (`PushClickEvent` using `XADD` with `MAXLEN ~100000`
    for automatic trimming) and consumer (`ReadClickEvents` using `XReadGroup` with a consumer group for
    horizontal scaling + crash recovery). Also includes `HashIP` (SHA-256, first 16 hex chars) for
    privacy-preserving IP hashing, `AckClickEvents` (`XACK`), and `StreamLen` for monitoring.
  - `internal/redis/stream_test.go` — integration tests: push/read round-trip, IP hashing determinism +
    uniqueness, stream length tracking. Tests skip gracefully if Redis isn't reachable.
  - `internal/store/store.go` — extended with `ClickEvent` struct and `ClickEventStore` interface;
    `Store` now includes `ClickEventStore`.
  - `internal/db/prisma_store.go` — `CreateClickEvent` implementation using Prisma's `ClickEvent.CreateOne`
    (link relation as first arg, optional fields as variadic params — same pattern as `CreateLink`).
  - `internal/worker/worker.go` — `AnalyticsWorker`: background goroutine that reads click events from
    the Redis Stream in batches (100 at a time, 5s block), writes each to Postgres via `CreateClickEvent`,
    and ACKs successfully-written events. Failed writes are NOT ACKed (stay in pending list for retry).
    Graceful shutdown via context cancellation.
  - `internal/redis/cache.go` — cache value format changed from plain URL string to JSON object
    `{"url":"...","id":123}` so the redirect handler has the link ID for click events even on cache hits.
    Backward compatible: old plain-URL cache entries are parsed with linkID=0 (click event skipped).
  - `internal/handlers/redirect.go` — replaced the Phase 3 TODO with `fireClickEvent` method that pushes
    a `ClickEvent` (with link ID, referrer, hashed IP) to the Redis Stream. Fire-and-forget: errors are
    ignored, redirect is never blocked.
  - `internal/handlers/shorten.go` — `SetLongURL` call updated to pass `link.ID` (new cache signature).
  - `cmd/api/main.go` — starts the `AnalyticsWorker` alongside the API server with a cancellable context.
    Added graceful shutdown on SIGINT/SIGTERM (cancels worker context + shuts down Fiber).

  **Verified in sandbox**: all packages (except `internal/db` which needs Prisma codegen) build and vet
  clean. **Not yet live-tested end-to-end** on Karan's machine.
- **Session 6**: Phase 4 built — Admin/Stats API.

  **What was built**:
  - `internal/store/store.go` — added `LinkStats`, `DailyClicks` types and `StatsStore` interface
    with methods: `GetLinkStats`, `GetTopLinks`, `GetUserLinks`, `GetRecentClickEvents`. Combined
    `Store` interface now includes `StatsStore`.
  - `internal/db/prisma_store.go` — implemented all StatsStore methods using Prisma queries:
    `GetLinkStats` aggregates clicks, unique IPs, top referrers, and daily click counts.
    `GetTopLinks` sorts all links by click count (top N). `GetUserLinks` finds links by user ID.
    `GetRecentClickEvents` fetches recent click events (sorted by timestamp descending, in Go).
  - `internal/handlers/stats.go` — new `StatsHandler` with routes:
    - `GET /api/stats/:shortCode` — aggregated click stats for a link (total clicks, unique IPs, top referrers, daily click chart)
    - `GET /api/stats/top?limit=N` — most-clicked links
    - `GET /api/links` — all links for the authenticated user (requires API key)
    - `GET /api/links/:shortCode/clicks?limit=N` — recent click events for a link
  - `internal/db/prisma_store.go` — fixed `QueryOrderDesc` compiler error by sorting results in Go
    instead of using Prisma's OrderBy (which wasn't available for the generated code).

  **Verified live on Karan's machine**: all 4 endpoints tested with real PostgreSQL data
  (click events recorded in prior tests). `GET /api/stats/s` returned correct totals,
  `GET /api/stats/top?limit=5` returned links sorted by clicks, `GET /api/links/s/clicks`
  returned recent events, and `GET /api/links` with a new API key returned an empty list.

## 8. Required local step before this runs: generate the Prisma client

This sandbox's network can't reach Prisma's binary CDN, so the Prisma Go client has never actually been
generated. On your own machine (normal internet access), from the project root:

```bash
go run github.com/steebchen/prisma-client-go generate
go build ./...
```

This downloads Prisma's CLI/query-engine binaries and writes the generated client into `internal/db`
(per the `output` path set in `prisma/schema.prisma`). After that, `go build ./...` should succeed
end-to-end. If it doesn't, paste the error back in and we'll debug it together.



### Dev environment note
Go module downloads needed `GOPROXY=direct GOSUMDB=off` plus `replace` directives in `go.mod` mapping
`golang.org/x/*` → `github.com/golang/*` mirrors, because the build sandbox only allowlists `github.com`/
`codeload.github.com`, not `proxy.golang.org` or `golang.org`. Keep these replace directives — remove only
if building outside this constrained network.
