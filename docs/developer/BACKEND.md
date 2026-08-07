# 🔧 Backend Developer Guide

> **Purpose:** Technical reference for the Go backend. Covers architecture, API endpoints, database, Redis, and the analytics worker.

---

## Stack

| Layer | Tech |
|-------|------|
| Language | Go 1.22 |
| Web Framework | Fiber v2.52 |
| ORM | Prisma (prisma-client-go) |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |
| Containerization | Docker + Docker Compose |

---

## Architecture

```
Client → Go API (Fiber, stateless)
              ├─ Redis: cache (short_code→long_url), counter (INCR), rate limiting (sorted set), streams (click events)
              ├─ PostgreSQL: links, users, api_keys, click_events (via Prisma)
              └─ Analytics Worker (goroutine): Redis Stream → Postgres
```

**Hot path (redirect):** Redis cache → 302 redirect → fire click event to Redis Stream (async)
**Write path (shorten):** Redis INCR → base62 encode → Prisma write → Redis cache write-through

---

## API Endpoints (9 total)

| Method | Path | Auth | Handler | Phase |
|--------|------|------|---------|-------|
| GET | `/health` | No | `handlers.HealthHandler.Check` | 0 |
| GET | `/metrics` | No | inline (metrics snapshot) | 5 |
| POST | `/api/keys` | No | `handlers.APIKeyHandler.CreateKey` | 2 |
| POST | `/api/shorten` | Optional | `handlers.ShortenHandler.Shorten` | 1 |
| GET | `/:shortCode` | No | `handlers.RedirectHandler.Redirect` | 1 |
| GET | `/api/stats/:shortCode` | No | `handlers.StatsHandler.GetLinkStats` | 4 |
| GET | `/api/stats/top` | No | `handlers.StatsHandler.GetTopLinks` | 4 |
| GET | `/api/links` | Required | `handlers.StatsHandler.GetUserLinks` | 4 |
| GET | `/api/links/:shortCode/clicks` | No | `handlers.StatsHandler.GetRecentClicks` | 4 |

**Middleware chain:** `recover → logger → cors → metrics → [authMW] → [rateLimitMW] → handler`

---

## Database Schema (4 tables)

```
links:        id, short_code (unique), long_url, user_id?, custom_alias, expires_at?, password_hash?, created_at
users:        id, email (unique), password_hash, created_at
api_keys:     id, key_hash (unique), user_id, rate_limit_tier, created_at
click_events: id, link_id, timestamp, referrer?, country?, device_type?, ip_hash?
```

**Relationships:** User 1→N Links, User 1→N ApiKeys, Link 1→N ClickEvents

---

## Redis Usage

| Purpose | Key Pattern | Command |
|---------|-------------|---------|
| URL cache | `urlshortener:link:<short_code>` | GET/SET (JSON `{"url":"...","id":123}`) |
| ID counter | `urlshortener:link_id_counter` | INCR |
| Rate limiting | `ratelimit:apikey:<id>` / `ratelimit:ip:<ip>` | ZADD/ZREMRANGEBYSCORE/ZCARD |
| Alias reservation | `urlshortener:alias_reserved:<alias>` | SETNX (30s TTL) |
| Click events stream | `urlshortener:click_events` | XADD/XREADGROUP/XACK |

---

## Analytics Worker

- Reads click events from Redis Stream in batches (100 at a time, 5s block)
- Writes each to Postgres via `CreateClickEvent`
- ACKs successfully-written events (XACK)
- Failed writes are NOT ACKed (stay in pending list for retry)
- Graceful shutdown via context cancellation

---

## Key Files

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Entry point — wires everything, registers routes, graceful shutdown |
| `internal/store/store.go` | All storage interfaces (contracts) |
| `internal/db/prisma_store.go` | Prisma implementation of all store interfaces |
| `internal/handlers/*.go` | HTTP handlers (health, shorten, redirect, apikeys, stats) |
| `internal/middleware/*.go` | Auth, rate limit, metrics middleware |
| `internal/redis/*.go` | Redis client, cache, counter, ratelimit, stream |
| `internal/worker/worker.go` | Analytics worker (Redis Stream → Postgres) |
| `internal/shortcode/base62.go` | Base62 encode/decode |
| `internal/auth/apikey.go` | API key generation + hashing |
| `internal/metrics/metrics.go` | Atomic counters for observability |

---

## How to Run

```bash
cd backend
cp .env.example .env   # set DATABASE_URL, REDIS_ADDR, etc.
go run github.com/steebchen/prisma-client-go generate
go run github.com/steebchen/prisma-client-go db push
go run ./cmd/api
```

Server runs at `http://localhost:8080`

---

## Common Gotchas

See [`.ai/GOTCHAS.md`](../../.ai/GOTCHAS.md) for all 15 known bugs and solutions. Key ones:

1. **Prisma CreateOne** — required fields = individual args, optional fields = single variadic slice
2. **types.BigInt()** — required for Prisma BigInt fields
3. **XGroupCreateMkStream** — use this, not XGroupCreate (avoids NOGROUP error)
4. **redis.Nil** — normal timeout from XReadGroup, not an error
5. **Import alias** — external go-redis is `goredis`, internal is `redis`