# URL Shortener — Request Flow Reference

> **What this is:** A repomix-style developer reference. Every API endpoint is traced from browser request to HTTP response, showing exactly which file, function, and line of code processes it. Not a tutorial — a map of the codebase.

---

## Tech Stack Snapshot

| Layer | Technology | Version |
|-------|-----------|---------|
| Language | Go (Golang) | 1.22 |
| Web Framework | Fiber v2 (built on valyala/fasthttp) | v2 |
| Database | PostgreSQL | 16-alpine |
| ORM | Prisma Client Go (steebchen/prisma-client-go) | v0.47.0 |
| Cache / Rate Limit / Events | Redis | 7-alpine |
| Redis Client | go-redis/v9 | v9 |
| Containerization | Docker + Docker Compose | — |

---

## Project File Map

```
cmd/api/main.go                     ← Server entry point, route registration, boot sequence
internal/config/config.go            ← Reads env vars into Config struct
internal/store/store.go             ← Persistence interfaces (LinkStore, UserStore, ApiKeyStore, etc.)
internal/db/prisma_store.go         ← Prisma-backed implementation of ALL store interfaces
internal/db/db_gen.go               ← Prisma-generated Go client (auto-generated, DO NOT EDIT)

internal/handlers/health.go          ← GET /health handler
internal/handlers/apikeys.go         ← POST /api/keys handler
internal/handlers/shorten.go         ← POST /api/shorten handler
internal/handlers/redirect.go        ← GET /:shortCode handler (the hot path)
internal/handlers/stats.go           ← GET /api/stats/* and /api/links/* handlers

internal/middleware/auth.go          ← OptionalAPIKeyAuth — Bearer token validation
internal/middleware/ratelimit.go      ← RateLimit — sliding-window per-key/per-IP limiter

internal/auth/apikey.go              ← GenerateAPIKey, HashKey, ExtractBearerToken
internal/redis/client.go             ← Redis connection wrapper (pool size, timeouts)
internal/redis/cache.go              ← URL cache (GET/SET/DEL) + alias reservation (SETNX)
internal/redis/counter.go            ← Global atomic INCR counter for short code IDs
internal/redis/ratelimit.go          ← Sliding-window-log rate limiter (sorted set)
internal/redis/stream.go             ← Redis Stream: push/read/ack click events
internal/redis/util.go               ← Helpers: randSuffix(), redisZ()

internal/shortcode/base62.go         ← Encode (uint64 → base62 string) and Decode
internal/worker/worker.go            ← Analytics worker: Redis Stream → Postgres

prisma/schema.prisma                 ← Database schema (4 models)
Dockerfile                           ← Multi-stage Go build (golang:1.22-alpine → alpine:3.19)
docker-compose.yml                   ← 3 services: postgres, redis, api
```

---

## Server Boot Sequence

**File:** `cmd/api/main.go`

```
 Browser Request
       │
       ▼
 ┌─────────────────────────────────────────────────────┐
 │  main() — cmd/api/main.go                           │
 │                                                     │
 │  1. godotenv.Load()                ← line 24        │
 │     Loads .env file into OS environment             │
 │                                                     │
 │  2. config.Load()                 ← line 26        │
 │     Reads PORT, BASE_URL, DATABASE_URL,             │
 │     REDIS_ADDR, REDIS_PASSWORD → Config struct     │
 │                                                     │
 │  3. redis.New(addr, pw, db)         ← line 28       │
 │     Creates Redis client (pool=50, minIdle=10)     │
 │                                                     │
 │  4. db.NewPrismaStore()           ← line 31        │
 │     Connects to Postgres via Prisma client          │
 │                                                     │
 │  5. worker.New(rdb, store).Start(ctx) ← lines 44-45 │
 │     Launches analytics worker goroutine             │
 │                                                     │
 │  6. fiber.New(...)                ← line 47        │
 │     Creates Fiber app (recover + logger middleware) │
 │                                                     │
 │  7. Route registration            ← lines 56-78     │
 │     Registers all 8 endpoints + per-route MW        │
 │                                                     │
 │  8. app.Listen(":8080")           ← line 92        │
 │     Starts HTTP server on configured port          │
 │                                                     │
 │  9. Graceful shutdown goroutine    ← lines 82-89   │
 │     Listens for SIGINT/SIGTERM → cancel ctx →       │
 │     stop worker → app.Shutdown()                    │
 └─────────────────────────────────────────────────────┘
```

### Boot code (annotated)

```go
// cmd/api/main.go:23-45
func main() {
    _ = godotenv.Load()                        // line 24 — load .env
    cfg := config.Load()                       // line 26 — env vars → Config

    rdb := redis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)  // line 28
    defer rdb.Close()

    linkStore, err := db.NewPrismaStore()       // line 31 — connect Postgres
    defer linkStore.Close()

    ctx, cancel := context.WithCancel(context.Background())  // line 41
    defer cancel()

    analyticsWorker := worker.New(rdb, linkStore)  // line 44
    analyticsWorker.Start(ctx)                      // line 45 — background goroutine

    app := fiber.New(fiber.Config{               // line 47
        AppName:      "urlshortener",
        ServerHeader: "urlshortener",
    })
```

### Config defaults (`internal/config/config.go:17-27`)

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | HTTP listen port |
| `BASE_URL` | `http://localhost:8080` | Prefix for generated short URLs |
| `ENV` | `development` | Environment label |
| `DATABASE_URL` | *(empty, required)* | PostgreSQL connection string |
| `REDIS_ADDR` | `localhost:6379` | Redis server address |
| `REDIS_PASSWORD` | *(empty)* | Redis password |
| `REDIS_DB` | `0` (hardcoded) | Redis database number |

---

## Global Middleware Stack

**File:** `cmd/api/main.go:52-53`

Every single request passes through these two middleware in order, regardless of the route:

```
 Incoming Request
       │
       ▼
 ┌──────────────────────────┐
 │ 1. recover.New()         │  ← fiber/v2/middleware/recover
 │    Catches Go panics,    │
 │    returns 500 instead   │
 │    of crashing the server│
 └──────────┬───────────────┘
            ▼
 ┌──────────────────────────┐
 │ 2. logger.New()          │  ← fiber/v2/middleware/logger
 │    Logs method, path,     │
 │    status, latency, IP   │
 │    to stdout              │
 └──────────┬───────────────┘
            ▼
    Per-route middleware
    (auth, rate limit, etc.)
            │
            ▼
       Handler
```

**Note:** There is NO global CORS middleware, NO body-parser middleware, NO compression. Fiber parses bodies on-demand via `c.BodyParser()`.

---

# ─────────────────────────────────────────────────────────────
# REQUEST FLOWS — every endpoint traced end-to-end
# ─────────────────────────────────────────────────────────────

---

## FLOW 1: GET /health — Health Check

**Route:** `cmd/api/main.go:56`
```go
app.Get("/health", health.Check)
```

**Per-route middleware:** None (only global recover + logger)

### Call chain

```
 GET /health
     │
     ▼
 recover.New()           ← global
 logger.New()            ← global
     │
     ▼
 HealthHandler.Check()   ← internal/handlers/health.go:19
     │
     ├─ context.WithTimeout(2s)
     │
     ├─ h.Redis.Ping(ctx)         ← internal/redis/client.go:32
     │   └─ rdb.Ping(ctx)        ← go-redis PING command
     │
     ├─ OK  → 200 {"status":"ok","redis":"ok"}
     └─ ERR → 503 {"status":"degraded","redis":"unreachable:..."}
```

### Handler code (annotated)

```go
// internal/handlers/health.go:19-37
func (h *HealthHandler) Check(c *fiber.Ctx) error {
    // Create a 2-second timeout context so we don't hang
    // if Redis is unreachable
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    status := "ok"
    code := fiber.StatusOK          // 200
    redisStatus := "ok"

    // Ping Redis — this is the only health check
    if err := h.Redis.Ping(ctx); err != nil {
        redisStatus = "unreachable: " + err.Error()
        status = "degraded"
        code = fiber.StatusServiceUnavailable  // 503
    }

    return c.Status(code).JSON(fiber.Map{
        "status": status,
        "redis":  redisStatus,
    })
}
```

### Responses

| Condition | Status | Body |
|-----------|--------|------|
| Redis reachable | `200` | `{"status":"ok","redis":"ok"}` |
| Redis unreachable | `503` | `{"status":"degraded","redis":"unreachable:..."}` |

---

## FLOW 2: POST /api/keys — Create API Key

**Route:** `cmd/api/main.go:65`
```go
app.Post("/api/keys", rateLimitMW, apiKeys.CreateKey)
```

**Per-route middleware:** `rateLimitMW` only (anonymous rate limit: 20/min per IP)

### Call chain

```
 POST /api/keys  {"email":"user@example.com"}
     │
     ▼
 recover.New()                          ← global
 logger.New()                           ← global
     │
     ▼
 RateLimit middleware                    ← internal/middleware/ratelimit.go:31
     ├─ No auth MW on this route → anonymous
     ├─ bucketKey = "ratelimit:ip:<IP>"
     ├─ Redis ZSET sliding-window check (20/min)
     ├─ Over limit → 429 {"error":"rate limit exceeded","retry_after":"..."}
     └─ Allowed → c.Next()
     │
     ▼
 APIKeyHandler.CreateKey()               ← internal/handlers/apikeys.go:38
     │
     ├─ c.BodyParser(&req)               ← Parse JSON: {"email":"..."}
     ├─ mail.ParseAddress(req.Email)     ← Validate email format
     │
     ├─ h.Store.GetOrCreateUserByEmail() ← internal/db/prisma_store.go:110
     │   ├─ User.FindUnique(Email)       ← SQL: SELECT * FROM users WHERE email=?
     │   ├─ Found → return existing User
     │   └─ ErrNotFound →
     │       User.CreateOne(Email, PasswordHash="")  ← SQL: INSERT INTO users...
     │
     ├─ auth.GenerateAPIKey()            ← internal/auth/apikey.go:27
     │   ├─ crypto/rand.Read(32 bytes)   ← 256 bits of entropy
     │   ├─ rawKey = "usk_" + hex(bytes)  ← e.g. "usk_a1b2c3..."
     │   └─ hash = SHA-256(rawKey)       ← hex-encoded
     │
     ├─ h.Store.CreateAPIKey()            ← internal/db/prisma_store.go:143
     │   └─ APIKey.CreateOne(KeyHash, UserID, Tier="standard")
     │       ← SQL: INSERT INTO api_keys (key_hash, user_id, rate_limit_tier) ...
     │
     └─ 201 {"api_key":"usk_...","warning":"save this key now — it will not be shown again"}
```

### Handler code (annotated)

```go
// internal/handlers/apikeys.go:38-73
func (h *APIKeyHandler) CreateKey(c *fiber.Ctx) error {
    var req createAPIKeyRequest            // {Email string}
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
    }

    // Validate email using Go's standard mail parser
    if _, err := mail.ParseAddress(req.Email); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "valid email is required"})
    }

    ctx := c.Context()

    // Find existing user or create a new one (no password in Phase 2)
    user, err := h.Store.GetOrCreateUserByEmail(ctx, req.Email)
    // → prisma_store.go:110-141

    // Generate raw key + hash
    rawKey, hash, err := auth.GenerateAPIKey()
    // → internal/auth/apikey.go:27-35

    // Persist only the hash — raw key is shown once and forgotten
    apiKey := &store.ApiKey{
        KeyHash:       hash,
        UserID:        user.ID,
        RateLimitTier: "standard",  // default tier
    }
    h.Store.CreateAPIKey(ctx, apiKey)
    // → prisma_store.go:143-162

    return c.Status(fiber.StatusCreated).JSON(createAPIKeyResponse{
        APIKey:  rawKey,   // raw key shown ONCE
        Warning: "save this key now — it will not be shown again",
    })
}
```

### Deep-dive: `auth.GenerateAPIKey()` (`internal/auth/apikey.go:27-35`)

```go
func GenerateAPIKey() (rawKey string, hash string, err error) {
    buf := make([]byte, 32)              // 32 bytes = 256 bits of entropy
    crypto/rand.Read(buf)                 // cryptographically secure random
    rawKey = "usk_" + hex.EncodeToString(buf)  // e.g. "usk_a1b2c3d4e5f6..."
    hash = HashKey(rawKey)               // SHA-256 hex
    return
}
```

### Deep-dive: `GetOrCreateUserByEmail()` (`internal/db/prisma_store.go:110-141`)

```go
func (s *PrismaStore) GetOrCreateUserByEmail(ctx context.Context, email string) (*store.User, error) {
    // Try to find existing user by email
    found, err := s.client.User.FindUnique(User.Email.Equals(email)).Exec(ctx)
    if err == nil {
        return &store.User{ID: uint64(found.ID), Email: found.Email, ...}, nil
    }
    // Not found → create new user with empty password hash (no login in Phase 2)
    if errors.Is(err, ErrNotFound) {
        created, _ := s.client.User.CreateOne(
            User.Email.Set(email),
            User.PasswordHash.Set(""),
        ).Exec(ctx)
        return &store.User{ID: uint64(created.ID), ...}, nil
    }
    return nil, err
}
```

### Request / Response

**Request:**
```json
POST /api/keys
Content-Type: application/json

{"email": "user@example.com"}
```

**Response (201):**
```json
{"api_key": "usk_a1b2c3d4e5f6...64hexchars...", "warning": "save this key now — it will not be shown again"}
```

---

## FLOW 3: POST /api/shorten — Create Short URL

**Route:** `cmd/api/main.go:68`
```go
app.Post("/api/shorten", authMW, rateLimitMW, shorten.Shorten)
```

**Per-route middleware:** `authMW` (OptionalAPIKeyAuth) → `rateLimitMW`

> **Why auth before rate-limit?** The rate limiter reads the API key from context to pick the right tier bucket (60/min for standard vs 20/min for anonymous). Auth must run first.

### Call chain

```
 POST /api/shorten  {"url":"https://google.com","custom_alias":"g","expires_at":"2026-12-31T23:59:59Z"}
  Authorization: Bearer usk_...   (optional)
     │
     ▼
 recover.New()                              ← global
 logger.New()                               ← global
     │
     ▼
 OptionalAPIKeyAuth                         ← internal/middleware/auth.go:22
     ├─ No header → anonymous (c.Next)
     ├─ Has header but malformed → 401
     ├─ Has header but invalid key → 401
     └─ Has valid key → c.Locals("apiKey", apiKey) → c.Next
     │
     ▼
 RateLimit middleware                        ← internal/middleware/ratelimit.go:31
     ├─ Authenticated → bucket "ratelimit:apikey:<id>", limit=60/min
     ├─ Anonymous    → bucket "ratelimit:ip:<IP>", limit=20/min
     ├─ Redis ZSET sliding-window check
     └─ Allowed → c.Next()
     │
     ▼
 ShortenHandler.Shorten()                   ← internal/handlers/shorten.go:42
     │
     ├─ c.BodyParser(&req)                   ← Parse JSON body
     │   ├─ req.URL         (required)
     │   ├─ req.CustomAlias (optional)
     │   └─ req.ExpiresAt   (optional, RFC3339)
     │
     ├─ validateURL(req.URL)                ← shorten.go:127-136
     │   ├─ Must be non-empty
     │   ├─ url.ParseRequestURI(raw)
     │   └─ Scheme must be "http" or "https"
     │
     ├─ time.Parse(RFC3339, req.ExpiresAt)   ← if provided
     │
     ├─ resolveShortCode(ctx, customAlias)  ← shorten.go:105-125
     │   │
     │   ├─ IF custom_alias provided:
     │   │   ├─ Regex check: ^[a-zA-Z0-9_-]{3,32}$
     │   │   ├─ Redis.ReserveAlias(alias)     ← redis/cache.go:96
     │   │   │   └─ SETNX urlshortener:alias_reserved:<alias> "1" 30s
     │   │   │       ├─ true  → alias reserved, proceed
     │   │   │       └─ false → 409 Conflict "alias already taken"
     │   │   └─ Return (alias, isCustom=true)
     │   │
     │   └─ IF no custom_alias (auto-generated):
     │       ├─ Redis.NextID(ctx)            ← redis/counter.go:14
     │       │   └─ INCR urlshortener:link_id_counter → atomic uint64
     │       └─ shortcode.Encode(id)         ← shortcode/base62.go:21
     │           └─ uint64 → base62 string (0→"0", 61→"Z", 62→"10")
     │
     ├─ If authenticated: link.UserID = &apiKey.UserID
     │
     ├─ h.Store.CreateLink(ctx, link)         ← db/prisma_store.go:44
     │   ├─ Link.CreateOne(ShortCode, LongURL, CustomAlias, ...)
     │   └─ SQL: INSERT INTO links (short_code, long_url, custom_alias, user_id, ...) ...
     │   ├─ On failure + custom alias → Redis.ReleaseAlias() to free reservation
     │   └─ Backfills link.ID and link.CreatedAt from created record
     │
     ├─ h.Redis.SetLongURL(ctx, code, url, id, expiresAt)  ← redis/cache.go:70
     │   ├─ TTL = min(24h, time until expiresAt)
     │   ├─ Serialize {url, id} as JSON
     │   └─ SET urlshortener:link:<shortCode> <json> <ttl>
     │
     └─ 201 {"short_url":"http://localhost:8080/g",
            "short_code":"g",
            "long_url":"https://google.com"}
```

### Handler code (annotated)

```go
// internal/handlers/shorten.go:42-101
func (h *ShortenHandler) Shorten(c *fiber.Ctx) error {
    var req shortenRequest                 // {URL, CustomAlias, ExpiresAt}
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
    }

    // Step 1: Validate the URL (must be http:// or https://)
    longURL, err := validateURL(req.URL)    // → shorten.go:127-136
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }

    // Step 2: Parse optional expiration (RFC3339 format)
    var expiresAt *time.Time
    if req.ExpiresAt != "" {
        t, err := time.Parse(time.RFC3339, req.ExpiresAt)
        // ... error → 400
        expiresAt = &t
    }

    ctx := c.Context()

    // Step 3: Get a short code — either custom alias or auto-generated
    shortCode, isCustom, err := h.resolveShortCode(ctx, req.CustomAlias)
    // → shorten.go:105-125
    // Errors: ErrInvalidAlias → 400, ErrAliasTaken → 409, other → 500

    // Step 4: Build the Link struct
    link := &store.Link{
        ShortCode:   shortCode,
        LongURL:     longURL,
        CustomAlias: isCustom,
        ExpiresAt:   expiresAt,
    }

    // Step 5: Attach user identity if authenticated
    if apiKey := middleware.GetAPIKey(c); apiKey != nil {
        link.UserID = &apiKey.UserID       // associates link with user
    }

    // Step 6: Persist to PostgreSQL
    if err := h.Store.CreateLink(ctx, link); err != nil {
        if isCustom {
            _ = h.Redis.ReleaseAlias(ctx, shortCode)  // free reservation on failure
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save link"})
    }

    // Step 7: Write-through cache (so first redirect is a cache hit)
    _ = h.Redis.SetLongURL(ctx, shortCode, longURL, link.ID, expiresAt)

    // Step 8: Return the shortened URL
    return c.Status(fiber.StatusCreated).JSON(shortenResponse{
        ShortURL:  h.BaseURL + "/" + shortCode,
        ShortCode: shortCode,
        LongURL:   longURL,
    })
}
```

### Deep-dive: `resolveShortCode()` (`internal/handlers/shorten.go:105-125`)

```go
func (h *ShortenHandler) resolveShortCode(ctx context.Context, customAlias string) (code string, isCustom bool, err error) {
    if customAlias != "" {
        // --- CUSTOM ALIAS PATH ---
        if !customAliasPattern.MatchString(customAlias) {  // ^[a-zA-Z0-9_-]{3,32}$
            return "", false, store.ErrInvalidAlias
        }
        reserved, rErr := h.Redis.ReserveAlias(ctx, customAlias)
        // → redis/cache.go:96 — SETNX with 30s TTL
        if !reserved {
            return "", false, store.ErrAliasTaken  // someone else holds it
        }
        return customAlias, true, nil
    }

    // --- AUTO-GENERATED PATH ---
    id, rErr := h.Redis.NextID(ctx)
    // → redis/counter.go:14 — INCR urlshortener:link_id_counter
    return shortcode.Encode(id), false, nil
    // → shortcode/base62.go:21 — converts uint64 to base62 string
}
```

### Deep-dive: `Redis.ReserveAlias()` (`internal/redis/cache.go:96-102`)

```go
func (c *Client) ReserveAlias(ctx context.Context, alias string) (bool, error) {
    // SETNX = Set if Not eXists — atomic reservation
    // Key: urlshortener:alias_reserved:<alias>
    // TTL: 30 seconds (safety net if Postgres write crashes)
    ok, err := c.rdb.SetNX(ctx, aliasReservationKey(alias), "1", 30*time.Second).Result()
    return ok, nil  // true = won the reservation, false = taken
}
```

### Deep-dive: `Redis.NextID()` + `shortcode.Encode()`

```go
// internal/redis/counter.go:14-20
func (c *Client) NextID(ctx context.Context) (uint64, error) {
    val, _ := c.rdb.Incr(ctx, "urlshortener:link_id_counter").Result()
    return uint64(val), nil  // atomic — no duplicates under concurrency
}

// internal/shortcode/base62.go:21-37
// Alphabet: "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
// Examples: 0→"0", 61→"Z", 62→"10", 12345→"3d7"
func Encode(id uint64) string {
    // Divide by 62 repeatedly, collect remainders as characters, reverse
    digits := make([]byte, 0, 11)
    for id > 0 {
        digits = append(digits, alphabet[id%base])
        id /= base
    }
    // reverse digits to get most-significant-first
    return string(reverse(digits))
}
```

### Deep-dive: `CreateLink()` (`internal/db/prisma_store.go:44-74`)

```go
func (s *PrismaStore) CreateLink(ctx context.Context, l *store.Link) error {
    params := []LinkSetParam{
        Link.CustomAlias.Set(l.CustomAlias),
    }
    if l.UserID != nil {
        params = append(params, Link.User.Link(User.ID.Equals(types.BigInt(*l.UserID))))
    }
    if l.ExpiresAt != nil {
        params = append(params, Link.ExpiresAt.Set(*l.ExpiresAt))
    }
    if l.PasswordHash != nil {
        params = append(params, Link.PasswordHash.Set(*l.PasswordHash))
    }

    created, err := s.client.Link.CreateOne(
        Link.ShortCode.Set(l.ShortCode),
        Link.LongURL.Set(l.LongURL),
        params...,
    ).Exec(ctx)
    // SQL: INSERT INTO links (short_code, long_url, custom_alias, user_id, expires_at, ...) VALUES (...)

    l.ID = uint64(created.ID)       // backfill the auto-generated ID
    l.CreatedAt = created.CreatedAt // backfill the timestamp
    return nil
}
```

### Request / Response

**Request:**
```json
POST /api/shorten
Content-Type: application/json
Authorization: Bearer usk_...  (optional)

{"url": "https://google.com", "custom_alias": "g", "expires_at": "2026-12-31T23:59:59Z"}
```

**Response (201):**
```json
{
  "short_url": "http://localhost:8080/g",
  "short_code": "g",
  "long_url": "https://google.com"
}
```

**Error responses:**

| Condition | Status | Body |
|-----------|--------|------|
| Invalid body | `400` | `{"error":"invalid request body"}` |
| Missing/invalid URL | `400` | `{"error":"url is required"}` or `{"error":"url must be a valid http(s) URL"}` |
| Invalid alias format | `400` | `{"error":"custom alias must be 3-32 chars, letters/numbers/underscore/hyphen only"}` |
| Alias already taken | `409` | `{"error":"custom alias already taken"}` |
| DB write failure | `500` | `{"error":"failed to save link"}` |
| Rate limit exceeded | `429` | `{"error":"rate limit exceeded","retry_after":"..."}` |

---

## FLOW 4: GET /:shortCode — Redirect (The Hot Path)

**Route:** `cmd/api/main.go:71`
```go
app.Get("/:shortCode", redirect.Redirect)
```

**Per-route middleware:** None (only global recover + logger). **No rate limiting, no auth** — this must be as fast as possible.

> **Important:** This is a catch-all route registered AFTER all `/api/*` and `/health` routes. Fiber matches routes in registration order, so specific routes take priority.

### Call chain

```
 GET /abc123
     │
     ▼
 recover.New()                           ← global
 logger.New()                            ← global
     │
     ▼
 RedirectHandler.Redirect()              ← internal/handlers/redirect.go:23
     │
     ├─ c.Params("shortCode") → "abc123"
     │
     ├─ h.Redis.GetLongURL(ctx, "abc123")   ← redis/cache.go:47
     │   └─ GET urlshortener:link:abc123
     │       ├─ redis.Nil → ErrCacheMiss → fall through to Postgres
     │       ├─ Error → log, fall through to Postgres
     │       └─ Hit → parse JSON {url, id}
     │
     ├─ CACHE HIT path:
     │   ├─ if linkID > 0 → fireClickEvent(ctx, linkID, c)
     │   │   ├─ Build ClickEvent{LinkID, Timestamp, Referrer, IPHash}
     │   │   ├─ redis.HashIP(c.IP())          ← SHA-256, first 16 hex chars
     │   │   └─ Redis.PushClickEvent(ctx, ev)  ← stream.go:42
     │   │       └─ XADD urlshortener:click_events MAXLEN ~100000
     │   └─ return 302 Found → Location: <longURL>
     │
     └─ CACHE MISS path:
         ├─ h.Store.GetLinkByShortCode(ctx, code)  ← db/prisma_store.go:76
         │   ├─ Link.FindUnique(ShortCode)         ← SQL: SELECT * FROM links WHERE short_code=?
         │   ├─ ErrNotFound → 404 "short link not found or expired"
         │   ├─ Expired (ExpiresAt.Before(now)) → 404 "short link not found or expired"
         │   └─ Found → map to store.Link
         │
         ├─ h.Redis.SetLongURL(ctx, code, url, id, expiresAt)
         │   └─ SET urlshortener:link:<code> <json> <ttl>  ← backfill cache
         │
         ├─ fireClickEvent(ctx, linkID, c)          ← same as cache hit path
         │
         └─ return 302 Found → Location: <longURL>
```

### Handler code (annotated)

```go
// internal/handlers/redirect.go:23-59
func (h *RedirectHandler) Redirect(c *fiber.Ctx) error {
    shortCode := c.Params("shortCode")     // extract from URL path
    ctx := c.Context()

    // === STEP 1: Cache-first lookup (serves almost all traffic) ===
    longURL, linkID, err := h.Redis.GetLongURL(ctx, shortCode)
    // → redis/cache.go:47 — GET urlshortener:link:<shortCode>
    if err == nil {
        // Cache HIT — fire analytics and redirect immediately
        if linkID > 0 {                   // old cache entries may have linkID=0
            h.fireClickEvent(ctx, linkID, c)
        }
        return c.Redirect(longURL, fiber.StatusFound)  // 302
    }
    if !errors.Is(err, redis.ErrCacheMiss) {
        // Redis error (not a miss) — log and fall through to Postgres
        // Don't fail the redirect just because Redis is having a bad day
    }

    // === STEP 2: Cache miss — fall back to Postgres ===
    link, err := h.Store.GetLinkByShortCode(ctx, shortCode)
    // → db/prisma_store.go:76-108
    if err != nil {
        if errors.Is(err, store.ErrNotFound) {
            return c.Status(fiber.StatusNotFound).JSON(
                fiber.Map{"error": "short link not found or expired"})
        }
        return c.Status(fiber.StatusInternalServerError).JSON(
            fiber.Map{"error": "failed to resolve short link"})
    }

    // === STEP 3: Backfill cache so next request is a hit ===
    _ = h.Redis.SetLongURL(ctx, link.ShortCode, link.LongURL, link.ID, link.ExpiresAt)

    // === STEP 4: Fire analytics (fire-and-forget) ===
    h.fireClickEvent(ctx, link.ID, c)

    // === STEP 5: Redirect ===
    return c.Redirect(link.LongURL, fiber.StatusFound)  // 302
}
```

### Deep-dive: `fireClickEvent()` (`internal/handlers/redirect.go:65-73`)

```go
// Fire-and-forget: errors are IGNORED. Analytics must never block or fail a redirect.
func (h *RedirectHandler) fireClickEvent(ctx context.Context, linkID uint64, c *fiber.Ctx) {
    ev := redis.ClickEvent{
        LinkID:    linkID,
        Timestamp: time.Now().UnixNano(),
        Referrer:  c.Get("Referer"),        // HTTP Referer header
        IPHash:    redis.HashIP(c.IP()),    // SHA-256 truncated to 16 hex chars
    }
    _ = h.Redis.PushClickEvent(ctx, ev)    // XADD to Redis Stream
}
```

### Deep-dive: `Redis.GetLongURL()` (`internal/redis/cache.go:47-64`)

```go
func (c *Client) GetLongURL(ctx context.Context, shortCode string) (longURL string, linkID uint64, err error) {
    val, err := c.rdb.Get(ctx, cacheKey(shortCode)).Result()
    // cacheKey = "urlshortener:link:" + shortCode

    if errors.Is(err, redis.Nil) {
        return "", 0, ErrCacheMiss  // key doesn't exist
    }
    if err != nil {
        return "", 0, err          // actual Redis error
    }

    // Try to parse as JSON (new format: {"url":"...","id":123})
    var cl cachedLink
    if json.Unmarshal([]byte(val), &cl) == nil && cl.LongURL != "" {
        return cl.LongURL, cl.LinkID, nil
    }

    // Old format (plain URL string) — backward compatible, linkID=0
    return val, 0, nil
}
```

### Deep-dive: `GetLinkByShortCode()` (`internal/db/prisma_store.go:76-108`)

```go
func (s *PrismaStore) GetLinkByShortCode(ctx context.Context, shortCode string) (*store.Link, error) {
    found, err := s.client.Link.FindUnique(Link.ShortCode.Equals(shortCode)).Exec(ctx)
    // SQL: SELECT * FROM links WHERE short_code = ? LIMIT 1

    if errors.Is(err, ErrNotFound) {
        return nil, store.ErrNotFound
    }

    result := &store.Link{
        ID: uint64(found.ID), ShortCode: found.ShortCode,
        LongURL: found.LongURL, CustomAlias: found.CustomAlias,
        CreatedAt: found.CreatedAt,
    }
    // Map optional fields (nullable in DB)
    if expiresAt, ok := found.ExpiresAt(); ok { result.ExpiresAt = &expiresAt }
    if passwordHash, ok := found.PasswordHash(); ok { result.PasswordHash = &passwordHash }

    // Expired links are treated as non-existent
    if result.ExpiresAt != nil && result.ExpiresAt.Before(time.Now()) {
        return nil, store.ErrNotFound
    }

    return result, nil
}
```

### Responses

| Condition | Status | Body / Header |
|-----------|--------|---------------|
| Cache hit | `302 Found` | `Location: <longURL>` |
| Cache miss, link found | `302 Found` | `Location: <longURL>` (after backfilling cache) |
| Link not found | `404` | `{"error":"short link not found or expired"}` |
| DB error | `500` | `{"error":"failed to resolve short link"}` |

---

## FLOW 5: GET /api/stats/:shortCode — Link Analytics

**Route:** `cmd/api/main.go:76`
```go
app.Get("/api/stats/:shortCode", rateLimitMW, stats.GetLinkStats)
```

**Per-route middleware:** `rateLimitMW` only (anonymous: 20/min)

### Call chain

```
 GET /api/stats/abc123
     │
     ▼
 recover.New()  →  logger.New()  →  RateLimit (20/min anonymous)
     │
     ▼
 StatsHandler.GetLinkStats()              ← internal/handlers/stats.go:34
     │
     ├─ c.Params("shortCode") → "abc123"
     │
     ├─ h.Store.GetLinkStats(ctx, shortCode)  ← db/prisma_store.go:224
     │   │
     │   ├─ GetLinkByShortCode(shortCode)     ← find the link first
     │   │   └─ Link.FindUnique(ShortCode)
     │   │
     │   ├─ ClickEvent.FindMany(LinkID)       ← fetch ALL click events for this link
     │   │   └─ SQL: SELECT * FROM click_events WHERE link_id = ?
     │   │
     │   ├─ In-memory aggregation:
     │   │   ├─ TotalClicks = len(allEvents)
     │   │   ├─ UniqueIPs = count(distinct IPHash values)
     │   │   ├─ ReferrerTop = top 5 by count (insertion sort)
     │   │   └─ DailyClicks = grouped by "2006-01-02", sorted ascending
     │   │
     │   └─ Return *store.LinkStats
     │
     └─ 200 {"short_code":"abc123","long_url":"...","total_clicks":42,
            "unique_ips":18,"top_referrers":["google.com",...],
            "daily_clicks":[{"date":"2026-08-01","count":5},...]}
```

### Handler code (annotated)

```go
// internal/handlers/stats.go:34-62
func (h *StatsHandler) GetLinkStats(c *fiber.Ctx) error {
    shortCode := c.Params("shortCode")

    stats, err := h.Store.GetLinkStats(c.Context(), shortCode)
    // → db/prisma_store.go:224-299 — fetches link + all clicks, aggregates in-memory
    if err != nil {
        if err == store.ErrNotFound {
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch link stats"})
    }

    // Format DailyClicks as JSON-friendly date strings
    daily := make([]fiber.Map, 0, len(stats.DailyClicks))
    for _, dc := range stats.DailyClicks {
        daily = append(daily, fiber.Map{"date": dc.Date.Format("2006-01-02"), "count": dc.Count})
    }

    return c.JSON(fiber.Map{
        "short_code":    stats.Link.ShortCode,
        "long_url":      stats.Link.LongURL,
        "total_clicks":  stats.TotalClicks,
        "unique_ips":    stats.UniqueIPs,
        "top_referrers": stats.ReferrerTop,     // top 5 referrers
        "daily_clicks":  daily,
    })
}
```

---

## FLOW 6: GET /api/stats/top — Top Links

**Route:** `cmd/api/main.go:75`
```go
app.Get("/api/stats/top", rateLimitMW, stats.GetTopLinks)
```

**Per-route middleware:** `rateLimitMW` only (anonymous: 20/min)

### Call chain

```
 GET /api/stats/top?limit=5
     │
     ▼
 recover.New()  →  logger.New()  →  RateLimit
     │
     ▼
 StatsHandler.GetTopLinks()               ← internal/handlers/stats.go:66
     │
     ├─ c.Query("limit") → "5" (default: 10, max: 100)
     │
     ├─ h.Store.GetTopLinks(ctx, 5)       ← db/prisma_store.go:302
     │   ├─ Link.FindMany()               ← fetch ALL links
     │   ├─ ClickEvent.FindMany()         ← fetch ALL click events
     │   ├─ Build clickCounts map by LinkID
     │   ├─ Sort by TotalClicks descending (insertion sort)
     │   └─ Truncate to limit
     │
     └─ 200 {"top_links":[{"short_code":"abc","long_url":"...","total_clicks":42},...]}
```

### Handler code (annotated)

```go
// internal/handlers/stats.go:66-89
func (h *StatsHandler) GetTopLinks(c *fiber.Ctx) error {
    limit := 10
    if l := c.Query("limit"); l != "" {
        if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
            limit = parsed
        }
    }

    topLinks, err := h.Store.GetTopLinks(c.Context(), limit)
    // → db/prisma_store.go:302-357 — clamps limit to [1,100], fetches all links+clicks, sorts, truncates
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch top links"})
    }

    results := make([]fiber.Map, 0, len(topLinks))
    for _, ls := range topLinks {
        results = append(results, fiber.Map{
            "short_code":   ls.Link.ShortCode,
            "long_url":     ls.Link.LongURL,
            "total_clicks": ls.TotalClicks,
        })
    }
    return c.JSON(fiber.Map{"top_links": results})
}
```

---

## FLOW 7: GET /api/links — User's Links

**Route:** `cmd/api/main.go:77`
```go
app.Get("/api/links", authMW, rateLimitMW, stats.GetUserLinks)
```

**Per-route middleware:** `authMW` (required) → `rateLimitMW`

> **This is the only endpoint where auth is REQUIRED (not optional).** Anonymous requests get 401.

### Call chain

```
 GET /api/links
 Authorization: Bearer usk_...
     │
     ▼
 recover.New()  →  logger.New()
     │
     ▼
 OptionalAPIKeyAuth          ← must have valid key (no anonymous fallback here)
     └─ No key → 401 "API key required"   (enforced in handler)
     │
     ▼
 RateLimit (60/min for standard key)
     │
     ▼
 StatsHandler.GetUserLinks()              ← internal/handlers/stats.go:93
     │
     ├─ middleware.GetAPIKey(c)           ← get authenticated key from context
     │   └─ nil → 401 "API key required"
     │
     ├─ h.Store.GetUserLinks(ctx, userID) ← db/prisma_store.go:360
     │   └─ Link.FindMany(UserID.Equals(userID))
     │       └─ SQL: SELECT * FROM links WHERE user_id = ?
     │
     └─ 200 {"links":[{"short_code":"abc","long_url":"...","custom_alias":false,"created_at":"..."},...]}
```

### Handler code (annotated)

```go
// internal/handlers/stats.go:93-115
func (h *StatsHandler) GetUserLinks(c *fiber.Ctx) error {
    apiKey := middleware.GetAPIKey(c)      // ← from auth middleware's c.Locals()
    if apiKey == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "API key required"})
    }

    links, err := h.Store.GetUserLinks(c.Context(), apiKey.UserID)
    // → db/prisma_store.go:360-387
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch user links"})
    }

    results := make([]fiber.Map, 0, len(links))
    for _, link := range links {
        results = append(results, fiber.Map{
            "short_code":   link.ShortCode,
            "long_url":     link.LongURL,
            "custom_alias": link.CustomAlias,
            "created_at":   link.CreatedAt,
        })
    }
    return c.JSON(fiber.Map{"links": results})
}
```

---

## FLOW 8: GET /api/links/:shortCode/clicks — Recent Click Events

**Route:** `cmd/api/main.go:78`
```go
app.Get("/api/links/:shortCode/clicks", rateLimitMW, stats.GetRecentClicks)
```

**Per-route middleware:** `rateLimitMW` only (anonymous: 20/min)

### Call chain

```
 GET /api/links/abc123/clicks?limit=10
     │
     ▼
 recover.New()  →  logger.New()  →  RateLimit (20/min)
     │
     ▼
 StatsHandler.GetRecentClicks()          ← internal/handlers/stats.go:119
     │
     ├─ c.Params("shortCode") → "abc123"
     │
     ├─ getLinkByShortCode(store, ctx, "abc123")  ← stats.go:25
     │   ├─ Type-assert StatsStore to LinkStore
     │   └─ Store.GetLinkByShortCode(ctx, code)
     │       └─ ErrNotFound → 404
     │
     ├─ c.Query("limit") → "10" (default: 20, max: 100)
     │
     ├─ h.Store.GetRecentClickEvents(ctx, linkID, 10)  ← db/prisma_store.go:390
     │   ├─ ClickEvent.FindMany(LinkID)
     │   ├─ Sort by timestamp descending (most recent first)
     │   └─ Truncate to limit
     │
     └─ 200 {"clicks":[{"timestamp":"...","referrer":"google.com","country":"","device_type":"","ip_hash":"abc123..."},...]}
```

### Handler code (annotated)

```go
// internal/handlers/stats.go:119-155
func (h *StatsHandler) GetRecentClicks(c *fiber.Ctx) error {
    shortCode := c.Params("shortCode")

    // First get the link to find its database ID
    link, err := getLinkByShortCode(h.Store, c.Context(), shortCode)
    // → stats.go:25 — type-asserts StatsStore to LinkStore, calls GetLinkByShortCode
    if err != nil {
        if err == store.ErrNotFound {
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to find link"})
    }

    limit := 20
    if l := c.Query("limit"); l != "" {
        if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
            limit = parsed
        }
    }

    events, err := h.Store.GetRecentClickEvents(c.Context(), link.ID, limit)
    // → db/prisma_store.go:390-437 — clamps limit to [1,100], fetches all, sorts desc, truncates
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch click events"})
    }

    results := make([]fiber.Map, 0, len(events))
    for _, ev := range events {
        results = append(results, fiber.Map{
            "timestamp":   ev.Timestamp,
            "referrer":    ev.Referrer,
            "country":     ev.Country,
            "device_type": ev.DeviceType,
            "ip_hash":     ev.IPHash,
        })
    }
    return c.JSON(fiber.Map{"clicks": results})
}
```

---

# ─────────────────────────────────────────────────────────────
# BACKGROUND PROCESSES
# ─────────────────────────────────────────────────────────────

## Analytics Worker (Redis Stream → PostgreSQL)

**File:** `internal/worker/worker.go`
**Started at:** `cmd/api/main.go:44-45`

This is NOT an HTTP endpoint — it's a background goroutine that runs alongside the server, consuming click events pushed by the redirect handler.

### Architecture

```
 Redirect Handler                Redis Stream                    Analytics Worker
 ─────────────────              ─────────────                    ────────────────
                                urlshortener:click_events
 fireClickEvent() ───XADD───►  [event1, event2, event3, ...]  ───XREADGROUP──►  worker-1
 (fire-and-forget)             Consumer Group: analytics-workers      │
                                                               ───for each event──►
                                                                    │
                                                                    ▼
                                                               CreateClickEvent()
                                                               INSERT INTO click_events
                                                                    │
                                                                    ▼
                                                               XACK (remove from pending)
```

### Worker loop (annotated)

```go
// internal/worker/worker.go:45-110
func (w *AnalyticsWorker) run(ctx context.Context) {
    for {
        if ctx.Err() != nil { return }     // context cancelled → shutdown

        // Read up to 100 events, block for 5s if empty
        ids, events, err := w.Redis.ReadClickEvents(ctx, w.ConsumerName, w.BatchSize, w.BlockTime)
        // → redis/stream.go:77
        //   └─ XGROUPCREATE (idempotent, creates stream + group if needed)
        //   └─ XREADGROUP GROUP analytics-workers worker-1 STREAMS urlshortener:click_events >

        if err == goredis.Nil { continue }  // timeout with no events — normal
        if err != nil { time.Sleep(1*time.Second); continue }  // real error, back off

        // Write each event to Postgres individually (Prisma has no bulk API)
        var ackIDs []string
        for i, ev := range events {
            clickEvent := &store.ClickEvent{
                LinkID:    ev.LinkID,
                Timestamp: time.Unix(0, ev.Timestamp),   // nanoseconds → time.Time
                Referrer:  ev.Referrer,
                Country:   ev.Country,
                DeviceType: ev.DeviceType,
                IPHash:    ev.IPHash,
            }

            if err := w.Store.CreateClickEvent(ctx, clickEvent); err != nil {
                // Failed write → skip but DON'T ACK
                // Event stays in pending list for retry
                log.Printf("failed to write click event (stream id=%s): %v", ids[i], err)
                continue
            }
            ackIDs = append(ackIDs, ids[i])
        }

        // ACK all successfully written events
        if len(ackIDs) > 0 {
            w.Redis.AckClickEvents(ctx, ackIDs...)
            // → redis/stream.go:114 — XACK urlshortener:click_events analytics-workers id1 id2 ...
        }
    }
}
```

### Redis Stream operations (`internal/redis/stream.go`)

```go
// PushClickEvent — called by redirect handler (fire-and-forget)
// → stream.go:42-60
func (c *Client) PushClickEvent(ctx context.Context, ev ClickEvent) error {
    data, _ := json.Marshal(ev)
    return c.rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: "urlshortener:click_events",
        MaxLen: 100000,      // cap stream at ~100k entries (safety net)
        Approx: true,        // trim in batches for efficiency
        Values: map[string]interface{}{"data": string(data)},
    }).Err()
}

// ReadClickEvents — called by analytics worker
// → stream.go:77-109
// XGROUPCREATE (idempotent) → XREADGROUP (blocking, 5s) → decode JSON → return IDs + events

// AckClickEvents — called after successful Postgres writes
// → stream.go:114-119
// XACK removes events from consumer group's pending list

// HashIP — SHA-256 of IP, truncated to first 16 hex chars (64 bits)
// → stream.go:65-71
```

---

# ─────────────────────────────────────────────────────────────
# MIDDLEWARE DEEP-DIVE
# ─────────────────────────────────────────────────────────────

## OptionalAPIKeyAuth

**File:** `internal/middleware/auth.go:22-48`

This middleware is "optional" in the sense that requests without a key pass through as anonymous. But if a key IS presented, it MUST be valid — no silent fallback.

```
 Request with Authorization header?
     │
     ├─ NO header → c.Next() as anonymous
     │
     └─ HAS header:
         │
         ├─ auth.ExtractBearerToken(header)     ← internal/auth/apikey.go:46
         │   ├─ Check "Bearer " prefix
         │   ├─ Trim and extract token
         │   └─ Empty/missing → ErrMalformedKey → 401
         │
         ├─ auth.HashKey(rawKey)                ← internal/auth/apikey.go:38
         │   └─ SHA-256(rawKey) → hex string
         │
         ├─ Store.GetAPIKeyByHash(ctx, hash)    ← db/prisma_store.go:164
         │   ├─ APIKey.FindUnique(KeyHash)       ← SQL: SELECT * FROM api_keys WHERE key_hash=?
         │   ├─ Found → c.Locals("apiKey", apiKey) → c.Next()
         │   └─ ErrNotFound → 401 "invalid API key"
         │
         └─ Other error → 500
```

### Full code

```go
// internal/middleware/auth.go:22-48
func OptionalAPIKeyAuth(s store.ApiKeyStore) fiber.Handler {
    return func(c *fiber.Ctx) error {
        header := c.Get("Authorization")
        if header == "" {
            return c.Next()  // anonymous — rate limiter falls back to per-IP
        }

        rawKey, err := auth.ExtractBearerToken(header)   // → auth/apikey.go:46
        if err != nil {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "malformed Authorization header, expected: Bearer <api_key>",
            })
        }

        hash := auth.HashKey(rawKey)                     // → auth/apikey.go:38
        apiKey, err := s.GetAPIKeyByHash(c.Context(), hash)  // → db/prisma_store.go:164
        if err != nil {
            if err == store.ErrNotFound {
                return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid API key"})
            }
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to validate API key"})
        }

        c.Locals(LocalsAPIKey, apiKey)   // store for handlers + rate limiter
        return c.Next()
    }
}

// internal/middleware/auth.go:51-56
func GetAPIKey(c *fiber.Ctx) *store.ApiKey {
    if v, ok := c.Locals(LocalsAPIKey).(*store.ApiKey); ok {
        return v
    }
    return nil  // anonymous
}
```

---

## RateLimit

**File:** `internal/middleware/ratelimit.go:31-67`

Must run AFTER OptionalAPIKeyAuth so it can read the authenticated key from context.

### Rate tiers

| Tier | Limit | Window | Bucket key format |
|------|-------|--------|-------------------|
| Anonymous (no key) | 20/min | 1 minute | `ratelimit:ip:<IP>` |
| `standard` | 60/min | 1 minute | `ratelimit:apikey:<keyID>` |
| `pro` | 600/min | 1 minute | `ratelimit:apikey:<keyID>` |

Unknown tiers fall back to "standard".

### Full code

```go
// internal/middleware/ratelimit.go:31-67
func RateLimit(rdb *redis.Client) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var bucketKey string
        var limit int
        var window time.Duration

        if apiKey := GetAPIKey(c); apiKey != nil {
            // Authenticated — use tier-based bucket
            tier, ok := tierLimits[apiKey.RateLimitTier]
            if !ok { tier = tierLimits["standard"] }
            bucketKey = fmt.Sprintf("ratelimit:apikey:%d", apiKey.ID)
            limit, window = tier.limit, tier.window
        } else {
            // Anonymous — use IP-based bucket
            bucketKey = fmt.Sprintf("ratelimit:ip:%s", c.IP())
            limit, window = anonLimit.limit, anonLimit.window
        }

        allowed, retryAfter, err := rdb.Allow(c.Context(), bucketKey, limit, window)
        // → redis/ratelimit.go:19

        if err != nil {
            // FAIL OPEN — Redis hiccup shouldn't take down the API
            return c.Next()
        }
        if !allowed {
            c.Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
            return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
                "error":       "rate limit exceeded",
                "retry_after": retryAfter.String(),
            })
        }

        return c.Next()
    }
}
```

---

## Sliding-Window-Log Algorithm

**File:** `internal/redis/ratelimit.go:19-55`

Uses a Redis sorted set (ZSET) where each member is a request timestamp and the score is that timestamp. Old entries outside the window are trimmed, then the count is compared against the limit.

### Full code (annotated)

```go
// internal/redis/ratelimit.go:19-55
func (c *Client) Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration, err error) {
    now := time.Now()
    windowStart := now.Add(-window)
    member := fmt.Sprintf("%d-%s", now.UnixNano(), randSuffix())
    // randSuffix = 4 random bytes as hex — prevents collisions at same nanosecond

    // All 4 commands run atomically in a transaction pipeline
    pipe := c.rdb.TxPipeline()
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))  // 1. trim old
    countCmd := pipe.ZCard(ctx, key)                                                  // 2. count remaining
    pipe.ZAdd(ctx, key, redisZ(float64(now.UnixNano()), member))                     // 3. add new entry
    pipe.Expire(ctx, key, window)                                                    // 4. auto-cleanup key

    if _, err := pipe.Exec(ctx); err != nil {
        return false, 0, err
    }

    count := countCmd.Val()
    if int(count) >= limit {
        // OVER LIMIT — remove the entry we just optimistically added
        c.rdb.ZRem(ctx, key, member)

        // Calculate retry-after from the oldest entry still in the window
        oldest, err := c.rdb.ZRangeWithScores(ctx, key, 0, 0).Result()
        if err == nil && len(oldest) > 0 {
            oldestTime := time.Unix(0, int64(oldest[0].Score))
            retryAfter = window - now.Sub(oldestTime)
            if retryAfter < 0 { retryAfter = 0 }
        } else {
            retryAfter = window
        }
        return false, retryAfter, nil
    }

    return true, 0, nil  // ALLOWED
}
```

**Why sliding-window instead of fixed-window?** Fixed windows have a "burst at boundary" problem — 60 requests at 1:59:59 and 60 more at 2:00:01 = 120 requests in 2 seconds. Sliding window prevents this.

---

# ─────────────────────────────────────────────────────────────
# REDIS KEY MAP
# ─────────────────────────────────────────────────────────────

| Key Pattern | Type | Redis Command | TTL | Purpose |
|-------------|------|---------------|-----|---------|
| `urlshortener:link:<shortCode>` | String | GET / SET | 24h (or link expiry) | Cached URL+ID for redirects |
| `urlshortener:alias_reserved:<alias>` | String | SETNX / DEL | 30s | Atomic custom alias reservation |
| `urlshortener:link_id_counter` | Counter | INCR | Permanent | Global atomic ID for short codes |
| `urlshortener:click_events` | Stream | XADD / XREADGROUP / XACK / XLEN | MAXLEN ~100000 | Click analytics event queue |
| `ratelimit:apikey:<id>` | Sorted Set | ZADD / ZREM / ZCARD / ZRANGE | = window (1min) | Per-API-key rate limit |
| `ratelimit:ip:<ip>` | Sorted Set | ZADD / ZREM / ZCARD / ZRANGE | = window (1min) | Per-IP anonymous rate limit |

### All Redis commands used (18 total)

| Command | File | Purpose |
|---------|------|---------|
| `PING` | `redis/client.go:32` | Health check |
| `GET` | `redis/cache.go:48` | Lookup cached long URL |
| `SET` | `redis/cache.go:82-84` | Cache long URL + link ID |
| `DEL` | `redis/cache.go:89, 106` | Invalidate cache / release alias |
| `SETNX` | `redis/cache.go:97` | Atomic alias reservation |
| `INCR` | `redis/counter.go:15` | Atomic ID counter |
| `ZREMRANGEBYSCORE` | `redis/ratelimit.go:25` | Trim old rate limit entries |
| `ZCARD` | `redis/ratelimit.go:26` | Count requests in window |
| `ZADD` | `redis/ratelimit.go:27` | Add request to rate limit set |
| `EXPIRE` | `redis/ratelimit.go:28` | Set TTL on rate limit key |
| `ZREM` | `redis/ratelimit.go:38` | Remove rejected request |
| `ZRANGEWITHSCORES` | `redis/ratelimit.go:41` | Get oldest entry for retry-after |
| `XADD` | `redis/stream.go:52` | Push click event |
| `XGROUPCREATEMKSTREAM` | `redis/stream.go:82` | Create consumer group |
| `XREADGROUP` | `redis/stream.go:84` | Read events from stream |
| `XACK` | `redis/stream.go:114` | Acknowledge processed events |
| `XLEN` | `redis/stream.go:123` | Stream length for monitoring |

---

# ─────────────────────────────────────────────────────────────
# DATABASE SCHEMA
# ─────────────────────────────────────────────────────────────

**File:** `prisma/schema.prisma` — PostgreSQL 16

### Table: `links`

| Column | Prisma Type | DB Column | Constraints | Notes |
|--------|-------------|-----------|-------------|-------|
| `id` | BigInt | `id` | PK, autoincrement | Also used as base for auto-generated short codes |
| `shortCode` | String | `short_code` | **@unique** | The short code or custom alias |
| `longUrl` | String | `long_url` | required | Original URL |
| `userId` | BigInt? | `user_id` | nullable, FK→users | NULL = anonymous link |
| `customAlias` | Boolean | `custom_alias` | default false | True if user chose the alias |
| `expiresAt` | DateTime? | `expires_at` | nullable | NULL = never expires |
| `passwordHash` | String? | `password_hash` | nullable | Future: password-protected links |
| `createdAt` | DateTime | `created_at` | default now() | |

**Indexes:** `@@unique([shortCode])`, `@@index([userId])`

### Table: `users`

| Column | Prisma Type | DB Column | Constraints |
|--------|-------------|-----------|-------------|
| `id` | BigInt | `id` | PK, autoincrement |
| `email` | String | `email` | **@unique** |
| `passwordHash` | String | `password_hash` | required (empty string in Phase 2) |
| `createdAt` | DateTime | `created_at` | default now() |

**Relations:** has many `links`, has many `apiKeys`

### Table: `api_keys`

| Column | Prisma Type | DB Column | Constraints | Notes |
|--------|-------------|-----------|-------------|-------|
| `id` | BigInt | `id` | PK, autoincrement | |
| `keyHash` | String | `key_hash` | **@unique** | SHA-256 hash of the API key |
| `userId` | BigInt | `user_id` | FK→users, required | |
| `rateLimitTier` | String | `rate_limit_tier` | default "standard" | "standard" or "pro" |
| `createdAt` | DateTime | `created_at` | default now() | |

**Indexes:** `@@unique([keyHash])`, `@@index([userId])`

### Table: `click_events`

| Column | Prisma Type | DB Column | Constraints | Notes |
|--------|-------------|-----------|-------------|-------|
| `id` | BigInt | `id` | PK, autoincrement | |
| `linkId` | BigInt | `link_id` | FK→links, required | |
| `timestamp` | DateTime | `timestamp` | default now() | |
| `referrer` | String? | `referrer` | nullable | HTTP Referer header |
| `country` | String? | `country` | nullable | Future: GeoIP lookup |
| `deviceType` | String? | `device_type` | nullable | Future: User-Agent parsing |
| `ipHash` | String? | `ip_hash` | nullable | SHA-256, first 16 hex chars |

**Indexes:** `@@index([linkId])`

---

# ─────────────────────────────────────────────────────────────
# STORE INTERFACE LAYER
# ─────────────────────────────────────────────────────────────

**File:** `internal/store/store.go`

Handlers and middleware depend on these interfaces — **never on Prisma directly**. This decouples handler logic from the database implementation (enabling mocking in tests or swapping backends).

### Interface hierarchy

```
Store (combines all 5)
 ├── LinkStore
 │   ├── CreateLink(ctx, *Link) error
 │   └── GetLinkByShortCode(ctx, shortCode) (*Link, error)
 │
 ├── UserStore
 │   └── GetOrCreateUserByEmail(ctx, email) (*User, error)
 │
 ├── ApiKeyStore
 │   ├── CreateAPIKey(ctx, *ApiKey) error
 │   └── GetAPIKeyByHash(ctx, hash) (*ApiKey, error)
 │
 ├── ClickEventStore
 │   └── CreateClickEvent(ctx, *ClickEvent) error
 │
 └── StatsStore
     ├── GetLinkStats(ctx, shortCode) (*LinkStats, error)
     ├── GetTopLinks(ctx, limit) ([]*LinkStats, error)
     ├── GetUserLinks(ctx, userID) ([]*Link, error)
     └── GetRecentClickEvents(ctx, linkID, limit) ([]*ClickEvent, error)
```

### Implementation: `db.PrismaStore` (`internal/db/prisma_store.go`)

A single struct implements ALL 5 interfaces. Created via `db.NewPrismaStore()` which connects to Postgres using `DATABASE_URL`.

### Data models (decoupled from Prisma)

| Model | Defined at | Purpose |
|-------|-----------|---------|
| `store.Link` | `store.go:20-29` | Mirrors Prisma Link, used by handlers |
| `store.User` | `store.go:32-36` | Mirrors Prisma User |
| `store.ApiKey` | `store.go:41-47` | Mirrors Prisma ApiKey, carries RateLimitTier |
| `store.ClickEvent` | `store.go:53-61` | Mirrors Prisma ClickEvent |
| `store.LinkStats` | `store.go:100-106` | Aggregated analytics (TotalClicks, UniqueIPs, etc.) |
| `store.DailyClicks` | `store.go:109-112` | Date+count pair for daily chart |

### Sentinel errors

```go
// store.go:12-16
var (
    ErrNotFound     = errors.New("store: link not found")       // 404
    ErrAliasTaken   = errors.New("store: short code / alias already taken")  // 409
    ErrInvalidAlias = errors.New("store: custom alias format invalid")        // 400
)
```

---

# ─────────────────────────────────────────────────────────────
# DOCKER ARCHITECTURE
# ─────────────────────────────────────────────────────────────

### Service topology

```
                 ┌──────────────────────────┐
   Port 8080     │   urlshortener-api        │
 ◄──────────────►│   (Go binary on alpine)   │
                 │                           │
                 │   cmd/api/main.go          │
                 │   Listens on :8080        │
                 └────────┬─────────┬─────────┘
                          │         │
                    postgres:5432  redis:6379
                    (Docker DNS)  (Docker DNS)
                          │         │
                 ┌────────▼────┐  ┌─▼────────────────┐
   Port 5432     │ urlshortener│  │ urlshortener-redis│
 ◄──────────────►│ -postgres  │  │ redis:7-alpine    │
                 │ postgres:16 │  │ AOF persistence   │
                 │ pg_data vol │  │ redis_data vol    │
                 └─────────────┘  └──────────────────┘
```

### Service details (`docker-compose.yml`)

| Service | Image | Container | Host Port | Volume | Health Check |
|---------|-------|-----------|-----------|--------|-------------|
| postgres | `postgres:16-alpine` | urlshortener-postgres | 5432 | pg_data → /var/lib/postgresql/data | `pg_isready -U urlshortener` (5s) |
| redis | `redis:7-alpine` | urlshortener-redis | 6379 | redis_data → /data | `redis-cli ping` (5s) |
| api | (built from Dockerfile) | urlshortener-api | 8080 | none | none (depends_on healthy) |

### Environment overrides (inside Docker)

The `.env` file uses `localhost` for local development. Docker Compose overrides these to use Docker service names:

| Variable | .env (local) | Docker override |
|----------|-------------|------------------|
| `DATABASE_URL` | `...@localhost:5432/...` | `...@postgres:5432/...` |
| `REDIS_ADDR` | `localhost:6379` | `redis:6379` |

### Dockerfile (multi-stage build)

| Stage | Base Image | Purpose |
|-------|-----------|---------|
| Builder | `golang:1.22-alpine` | Install openssl → `go mod download` → Prisma generate → `CGO_ENABLED=0 go build` |
| Runner | `alpine:3.19` | Copy binary → expose 8080 → `./urlshortener` |

> **Critical:** Prisma client is generated INSIDE the Docker build container so the query-engine binary matches Alpine/musl, not the host OS (e.g., macOS).

---

# ─────────────────────────────────────────────────────────────
# COMPLETE ROUTE MAP (with middleware per route)
# ─────────────────────────────────────────────────────────────

| # | Method | Path | Auth | Rate Limit | Handler | File:Line |
|---|--------|------|------|-----------|---------|-----------|
| 1 | GET | `/health` | No | No | `health.Check` | health.go:19 |
| 2 | POST | `/api/keys` | No | Yes (20/min anon) | `apiKeys.CreateKey` | apikeys.go:38 |
| 3 | POST | `/api/shorten` | Optional | Yes (20 or 60/min) | `shorten.Shorten` | shorten.go:42 |
| 4 | GET | `/:shortCode` | No | **No** | `redirect.Redirect` | redirect.go:23 |
| 5 | GET | `/api/stats/top` | No | Yes (20/min anon) | `stats.GetTopLinks` | stats.go:66 |
| 6 | GET | `/api/stats/:shortCode` | No | Yes (20/min anon) | `stats.GetLinkStats` | stats.go:34 |
| 7 | GET | `/api/links` | **Required** | Yes (60/min) | `stats.GetUserLinks` | stats.go:93 |
| 8 | GET | `/api/links/:shortCode/clicks` | No | Yes (20/min anon) | `stats.GetRecentClicks` | stats.go:119 |

---

# ─────────────────────────────────────────────────────────────
# APPENDIX
# ─────────────────────────────────────────────────────────────

## How to Run This Project

### Option 1: Docker (easiest)

```bash
cp .env.example .env
docker-compose up --build
go run github.com/steebchen/prisma-client-go db push    # one-time: create tables
```

### Option 2: Local development

```bash
cp .env.example .env
docker-compose up -d postgres redis                       # start databases only
go run github.com/steebchen/prisma-client-go generate     # generate Prisma client
go run github.com/steebchen/prisma-client-go db push      # create tables
go run ./cmd/api                                          # start server
```

### Testing it out

```bash
# Health check
curl http://localhost:8080/health

# Create an API key
curl -X POST http://localhost:8080/api/keys \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'

# Shorten a URL (anonymous)
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.google.com"}'

# Shorten a URL (authenticated)
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer usk_your_key_here" \
  -d '{"url": "https://www.google.com"}'

# Visit the short URL (redirects)
curl -L http://localhost:8080/<short_code>

# Link stats
curl http://localhost:8080/api/stats/<short_code>

# Top links
curl http://localhost:8080/api/stats/top?limit=5

# Your links (requires API key)
curl http://localhost:8080/api/links \
  -H "Authorization: Bearer usk_your_key_here"

# Recent clicks for a link
curl http://localhost:8080/api/links/<short_code>/clicks?limit=10
```

---

## Testing

| Test File | What It Tests |
|-----------|---------------|
| `internal/auth/apikey_test.go` | Key generation randomness, consistent hashing, Bearer token extraction |
| `internal/shortcode/base62_test.go` | Encode/decode round-trip, uniqueness across 10k IDs, invalid chars |
| `internal/redis/ratelimit_test.go` | Under-limit allows, over-limit blocks, window slides correctly |
| `internal/redis/stream_test.go` | Push+read round-trip, IP hashing, stream length tracking |

> Redis tests require a running Redis instance. If Redis is unavailable, tests are skipped (not failed).

---

## Build Phases & Current Status

| Phase | Status | What Was Done |
|-------|--------|---------------|
| **Phase 0** | ✅ Complete | Project scaffold, Docker setup, Fiber server, Redis client, `/health` endpoint |
| **Phase 1** | ✅ Complete | Core shorten + redirect API (base62, PostgreSQL via Prisma, Redis cache) |
| **Phase 2** | ✅ Complete | API key authentication + sliding-window rate limiting |
| **Phase 3** | ✅ Complete | Async analytics pipeline (Redis Streams → worker → PostgreSQL) |
| **Phase 4** | ✅ Complete | Admin/Stats API (link analytics, top links, user links, recent clicks) |
| **Phase 5** | ⬜ Not started | Load testing, caching tuning, observability |
| **Phase 6** | ⬜ Not started | Deployment hardening (finalized Docker Compose, optional Kubernetes) |


---

## 18. Frontend (LinkSnip UI) — What It Is & How It Works

### What is the frontend?

The frontend is the **visual part** of the app — the web pages you actually see and click in your browser. It is built with **React** (a popular JavaScript library for building user interfaces) and **Vite** (a fast build tool).

When you open http://localhost:5173, you see the **LinkSnip** UI — the landing page with the URL shortening box, the login page, the dashboard, the stats charts, etc.

### Frontend Pages (6)

| Page | URL | What It Does |
|------|-----|--------------|
| Landing | / | Hero section + the Shorten a URL input box |
| Login | /login | Enter your email to get an API key |
| Dashboard | /dashboard | Shows all YOUR shortened links (requires API key) |
| Stats | /stats/abc123 | Charts and analytics for one specific link |
| Top Links | /top | The most-clicked links across the whole app |
| Admin | /admin | Shows app health and performance metrics |

### How the frontend talks to the backend

Your Browser -> Vite Dev Server (frontend, port 5173) -> Go Backend API (port 8080) -> PostgreSQL / Redis

In development, the frontend runs on port 5173 and the backend runs on port 8080. Vite automatically forwards any request starting with /api to the backend.

### Auth — API Keys

- You get an API key by entering your email on the Login page
- The key is stored in your browser localStorage
- Every API call automatically includes Authorization: Bearer usk_...
- If the key is missing or invalid, you get redirected to the Login page

### Key Frontend Files

| File | What It Does |
|------|--------------|
| frontend/src/App.jsx | Defines all the routes (which page shows at which URL) |
| frontend/src/main.jsx | The starting point — mounts React into the HTML page |
| frontend/src/api/client.js | The postman — handles all API calls, adds auth header |
| frontend/src/context/AuthContext.jsx | Stores your API key state (logged in / logged out) |
| frontend/src/pages/LandingPage.jsx | The homepage with the shorten form |
| frontend/src/pages/DashboardPage.jsx | Your links table |
| frontend/src/pages/StatsPage.jsx | The analytics charts for a link |
| frontend/src/components/LinkTable.jsx | The table showing your links (with copy/delete/stats buttons) |

---

## 19. How to Run the FULL App (Both Parts)

You need two terminals because the frontend and backend are separate servers:

Terminal 1 — run the backend (port 8080):
  cd backend
  go run ./cmd/api

Terminal 2 — run the frontend (port 5173):
  cd frontend
  npm install     (only the first time)
  npm run dev

Then open http://localhost:5173 in your browser.

---

## 20. Project Structure (Current, Full-Stack)

Url-Shortner/
  backend/              <- Go API (the brain)
    cmd/api/main.go     <- Server entry point
    internal/           <- All Go packages
    prisma/             <- Database schema
    go.mod              <- Go dependencies
    Dockerfile          <- Container build
  frontend/             <- React app (the face)
    src/                <- All React source
    vite.config.js      <- Dev server + proxy config
    package.json        <- Frontend dependencies
    tailwind.config.js  <- Styling config
  docs/                 <- All documentation
  .ai/                  <- AI agent context files
  KNOWLEDGE.md          <- This file
  docker-compose.yml    <- Runs postgres + redis + api
  QWEN_FRONTEND_PROMPT.md <- The prompt used to generate the frontend

---

## 21. Current Phase Status and What is Next

| Phase | Status | What |
|-------|--------|------|
| 0 | Done | Scaffold, Docker, Fiber, Redis |
| 1 | Done | Core shorten + redirect API |
| 2 | Done | API key auth + rate limiting |
| 3 | Done | Async analytics pipeline |
| 4 | Done | Admin/Stats API |
| 5 | Done | Observability (metrics) |
| 6 | Done | Frontend UI (LinkSnip) + backend wiring |
| 7 | NEXT | Deployment — take the app live on the internet |
