# 📁 CODE MAP — Every File Explained

> **Purpose:** Complete map of every source file, what it does, key functions, and dependencies. An agent can read this and know exactly where to make changes without exploring the codebase.

---

## Entry Point

### `cmd/api/main.go`
**What it does:** Starts the entire application — connects to Redis + Postgres, starts analytics worker, sets up Fiber routes, handles graceful shutdown.

**Key flow:**
```
main() → godotenv.Load() → config.Load() → redis.New() → db.NewPrismaStore()
  → worker.New(rdb, linkStore).Start(ctx) → fiber.New() → register routes → app.Listen(:8080)
```

**Routes registered:**
- `GET /health` → healthHandler.Check
- `POST /api/keys` → apiKeyHandler.CreateKey (with rateLimitMW)
- `POST /api/shorten` → shortenHandler.Shorten (with authMW + rateLimitMW)
- `GET /:shortCode` → redirectHandler.Redirect
- `GET /api/stats/top` → statsHandler.GetTopLinks (with rateLimitMW)
- `GET /api/stats/:shortCode` → statsHandler.GetLinkStats (with rateLimitMW)
- `GET /api/links` → statsHandler.GetUserLinks (with authMW + rateLimitMW)
- `GET /api/links/:shortCode/clicks` → statsHandler.GetRecentClicks (with rateLimitMW)

**Graceful shutdown:** SIGINT/SIGTERM → cancel context (stops worker) → app.Shutdown()

**Dependencies:** config, db, handlers, middleware, redis, worker

---

## internal/config/

### `internal/config/config.go`
**What it does:** Loads environment variables into a `Config` struct with defaults.

**Key type:**
```go
type Config struct {
    Port, Env, BaseURL, DatabaseURL, RedisAddr, RedisPassword string
    RedisDB int
}
```

**Function:** `Load() Config` — reads env vars, falls back to defaults if not set.

---

## internal/store/

### `internal/store/store.go`
**What it does:** Defines ALL storage interfaces (contracts). Handlers depend on these, never on Prisma directly. This is the **most important file for understanding the architecture**.

**Interfaces:**
- `LinkStore` — `CreateLink(ctx, *Link)`, `GetLinkByShortCode(ctx, shortCode) (*Link, error)`
- `UserStore` — `GetOrCreateUserByEmail(ctx, email) (*User, error)`
- `ApiKeyStore` — `CreateAPIKey(ctx, *ApiKey)`, `GetAPIKeyByHash(ctx, hash) (*ApiKey, error)`
- `ClickEventStore` — `CreateClickEvent(ctx, *ClickEvent) error`
- `StatsStore` — `GetLinkStats`, `GetTopLinks`, `GetUserLinks`, `GetRecentClickEvents`
- `Store` — combines all above interfaces

**Types:** `Link`, `User`, `ApiKey`, `ClickEvent`, `LinkStats`, `DailyClicks`

**Errors:** `ErrNotFound`, `ErrAliasTaken`, `ErrInvalidAlias`

---

## internal/db/

### `internal/db/prisma_store.go`
**What it does:** Implements ALL store interfaces using Prisma client (PostgreSQL). This is the ONLY file that talks to the database.

**Key type:** `PrismaStore struct { client *PrismaClient }`

**Functions:**
- `NewPrismaStore() (*PrismaStore, error)` — connects to Postgres
- `Close() error`
- `CreateLink` — uses `client.Link.CreateOne()` with variadic params slice
- `GetLinkByShortCode` — uses `client.Link.FindUnique()`, checks expiry
- `GetOrCreateUserByEmail` — find first, create if not found
- `CreateAPIKey` — stores hashed key with rate limit tier
- `GetAPIKeyByHash` — lookup by hash for auth middleware
- `CreateClickEvent` — called by analytics worker, uses `client.ClickEvent.CreateOne()`
- `GetLinkStats` — aggregates clicks, unique IPs, top referrers, daily counts
- `GetTopLinks` — counts clicks per link, sorts by count descending
- `GetUserLinks` — finds links by user_id
- `GetRecentClickEvents` — fetches events, sorts by timestamp in Go (not Prisma)

**⚠️ Gotcha:** Prisma's `CreateOne` requires optional params in a single variadic slice — can't mix individual args with spread. See `.ai/GOTCHAS.md`.

---

## internal/handlers/

### `internal/handlers/health.go`
**What it does:** Health check endpoint. Pings Redis and returns status.
- `HealthHandler struct { redis *redis.Client }`
- `Check(c *fiber.Ctx) error` — returns `{"status":"ok","redis":"ok"}`

### `internal/handlers/shorten.go`
**What it does:** Shortens URLs. Validates URL, generates short code (custom or auto), saves to Postgres, writes to Redis cache.
- `ShortenHandler struct { store store.LinkStore, redis *redis.Client, baseURL string }`
- `Shorten(c *fiber.Ctx) error` — main handler
- `resolveShortCode(c *fiber.Ctx, req ShortenRequest) (string, error)` — custom alias or INCR+base62

**Flow:** Parse JSON → validate URL → resolveShortCode → store.CreateLink → redis.SetLongURL → return JSON

### `internal/handlers/redirect.go`
**What it does:** Redirects short codes to long URLs. Cache-first, falls back to Postgres, fires click event.
- `RedirectHandler struct { store store.LinkStore, redis *redis.Client }`
- `Redirect(c *fiber.Ctx) error` — main handler
- `fireClickEvent(redis, linkID, referrer, ip)` — pushes to Redis Stream (fire-and-forget)

**Flow:** Get shortCode → redis.GetLongURL → if miss: store.GetLinkByShortCode → redis.SetLongURL → fireClickEvent → 302 redirect

### `internal/handlers/apikeys.go`
**What it does:** Creates API keys. Generates random key, hashes it, stores hash, returns raw key once.
- `APIKeyHandler struct { store store.Store }`
- `CreateKey(c *fiber.Ctx) error` — takes email, creates user if needed, generates key

### `internal/handlers/stats.go`
**What it does:** Admin/Stats API (Phase 4). 4 endpoints for analytics.
- `StatsHandler struct { Store store.StatsStore }`
- `GetLinkStats` — total clicks, unique IPs, top referrers, daily chart
- `GetTopLinks` — most-clicked links
- `GetUserLinks` — links for authenticated user (requires API key)
- `GetRecentClicks` — recent click events for a link

**⚠️ Gotcha:** Uses type assertion `s.(store.LinkStore)` to access GetLinkByShortCode from StatsStore. See `.ai/GOTCHAS.md`.

---

## internal/middleware/

### `internal/middleware/auth.go`
**What it does:** Optional API key authentication. Extracts bearer token, hashes it, looks up in store.
- `OptionalAPIKeyAuth(store store.ApiKeyStore) fiber.Handler`
- `GetAPIKey(c *fiber.Ctx) *store.ApiKey` — returns nil if anonymous, *ApiKey if authenticated

**Behavior:** No auth header → anonymous (pass through). Invalid auth header → 401. Valid → set apiKey in Locals.

### `internal/middleware/ratelimit.go`
**What it does:** Sliding window rate limiting using Redis sorted sets.
- `RateLimit(rdb *redis.Client) fiber.Handler`
- Tiers: anonymous=20/min, standard=60/min, pro=600/min
- **Fail-open:** if Redis is down, requests pass through

---

## internal/redis/

### `internal/redis/client.go`
**What it does:** Creates and configures the Redis client connection.
- `Client struct { rdb *redis.Client }`
- `New(addr, password string, db int) *Client`
- `Close() error`

### `internal/redis/cache.go`
**What it does:** URL cache (short_code → JSON with long_url + link_id). Also handles alias reservation.
- `GetLongURL(shortCode) (url string, linkID uint64, err error)` — parses JSON cache value
- `SetLongURL(shortCode, longURL string, linkID uint64, ttl time.Duration) error`
- `ReserveAlias(alias string) error` — SETNX with 30s TTL
- `ReleaseAlias(alias string) error`

**⚠️ Gotcha:** Cache value is JSON `{"url":"...","id":123}`, not plain URL. Old plain-URL entries are parsed with linkID=0. See `.ai/GOTCHAS.md`.

### `internal/redis/counter.go`
**What it does:** Global atomic counter for generating unique link IDs.
- `NextID() (uint64, error)` — uses Redis INCR, atomic and collision-free

### `internal/redis/ratelimit.go`
**What it does:** Sliding window rate limiter using Redis sorted sets.
- `Allow(key string, limit int, window time.Duration) (bool, error)`
- Algorithm: ZADD timestamp → ZREMRANGEBYSCORE (trim old) → ZCARD (count) → compare to limit

### `internal/redis/stream.go`
**What it does:** Redis Stream producer/consumer for click analytics (Phase 3).
- `PushClickEvent(ev ClickEvent) error` — XADD with MAXLEN ~100000
- `ReadClickEvents(consumer, count, block) (ids, events, error)` — XReadGroup with consumer group
- `AckClickEvents(ids ...string) error` — XACK
- `HashIP(ip string) string` — SHA-256, first 16 hex chars
- `StreamLen() (int64, error)` — XLEN

**⚠️ Gotcha:** Uses `XGroupCreateMkStream` (not `XGroupCreate`) to auto-create stream. See `.ai/GOTCHAS.md`.

### `internal/redis/util.go`
**What it does:** Small helper functions (e.g., parsing durations).

### `internal/redis/ratelimit_test.go`
**What it does:** Integration tests for rate limiter. Skips if Redis not available.
- TestAllow_UnderLimit, TestAllow_OverLimit, TestAllow_WindowSlides

### `internal/redis/stream_test.go`
**What it does:** Integration tests for Redis Stream operations.
- TestPushAndReadClickEvent, TestHashIP, TestStreamLen

---

## internal/worker/

### `internal/worker/worker.go`
**What it does:** Background analytics worker. Reads click events from Redis Stream, writes to Postgres, ACKs.
- `AnalyticsWorker struct { Redis, Store, ConsumerName, BatchSize, BlockTime }`
- `New(rdb, store) *AnalyticsWorker` — defaults: batch=100, block=5s
- `Start(ctx)` — launches goroutine
- `run(ctx)` — main loop: ReadClickEvents → CreateClickEvent for each → AckClickEvents

**⚠️ Gotcha:** `redis.Nil` from XReadGroup is normal (timeout, no events). Don't log as error. See `.ai/GOTCHAS.md`.

---

## internal/auth/

### `internal/auth/apikey.go`
**What it does:** API key generation and hashing.
- `GenerateAPIKey() (rawKey, hash string, err error)` — 256-bit random, `usk_` prefix, SHA-256 hash
- `HashAPIKey(raw string) string` — SHA-256 hex
- `ExtractBearerToken(authHeader string) (string, error)` — parses `Bearer <key>`

### `internal/auth/apikey_test.go`
**What it does:** Unit tests for API key generation and bearer token extraction.

---

## internal/shortcode/

### `internal/shortcode/base62.go`
**What it does:** Base62 encoding/decoding (0-9, a-z, A-Z = 62 chars).
- `Encode(n uint64) string` — number → short code
- `Decode(s string) (uint64, error)` — short code → number

**Example:** 0→"0", 61→"Z", 62→"10", 12345→"3d7"

### `internal/shortcode/base62_test.go`
**What it does:** Unit tests for base62 encode/decode round-trips, uniqueness, invalid characters.

---

## internal/config/

### `internal/config/config.go`
**What it does:** Loads env vars into Config struct with defaults. See above.

---

## prisma/

### `prisma/schema.prisma`
**What it does:** Database schema definition (source of truth for Postgres tables).
- 4 models: Link, User, ApiKey, ClickEvent
- Relations: User 1→N Link, User 1→N ApiKey, Link 1→N ClickEvent
- Output: `../internal/db` (generated Prisma client goes here)

**⚠️ Gotcha:** After changing schema, must run `go run github.com/steebchen/prisma-client-go generate` then `db push`. See `.ai/GOTCHAS.md`.

---

## Docker Files

### `Dockerfile`
**What it does:** Multi-stage build. Stage 1 (builder): compiles Go + generates Prisma client. Stage 2 (runner): minimal Alpine with just the binary.

**⚠️ Gotcha:** Prisma client MUST be generated inside Docker (platform match). See `.ai/GOTCHAS.md`.

### `docker-compose.yml`
**What it does:** Defines 3 services: postgres, redis, api. Health checks, volumes, env overrides for container networking.

**⚠️ Gotcha:** Inside Docker, services are reachable by name (postgres:5432), not localhost. See `.ai/GOTCHAS.md`.

---

## Documentation Files

| File | Purpose |
|------|---------|
| `PROJECT_OVERVIEW.md` | Developer's session log, phase tracking, architecture decisions |
| `KNOWLEDGE.md` | 660-line layman's guide for first-time coders |
| `README.md` | Quick setup instructions |
| `.ai/PROJECT_CONTEXT.md` | Master context for AI agents (read first) |
| `.ai/CODE_MAP.md` | This file — every source file explained |
| `.ai/CONVENTIONS.md` | Coding patterns and conventions |
| `.ai/GOTCHAS.md` | All known bugs, gotchas, and solutions |
| `.ai/QUICKSTART.md` | 3-command setup guide |
| `.ai/AGENT_INSTRUCTIONS.md` | How to work on this project |