# ⚠️ GOTCHAS — All Known Bugs, Problems & Solutions

> **Purpose:** Every problem encountered during development, with root cause and fix. Read this BEFORE writing Prisma queries, Redis Stream code, or Docker configs.

---

## 1. Prisma CreateOne Variadic Mixing

**Problem:** Go compiler error when calling `CreateOne` with both individual optional args AND a spread slice.

**Root cause:** Go disallows mixing individual variadic arguments with a spread slice in the same function call. Prisma's `CreateOne` takes required fields as individual args and optional fields as variadic `...SetParam`.

**Error:**
```
cannot use Link.CustomAlias.Set(true) (type LinkSetParam) as type LinkCreateOneSetParam)
```

**Fix:** Put ALL optional params in a single slice, spread with `...`:
```go
params := []LinkSetParam{Link.CustomAlias.Set(true)}
if l.UserID != nil {
    params = append(params, Link.User.Link(...))
}
client.Link.CreateOne(
    Link.ShortCode.Set(code),  // required
    Link.LongURL.Set(url),     // required
    params...,                  // all optional
)
```

**Files affected:** `internal/db/prisma_store.go` (CreateLink, CreateClickEvent)

---

## 2. Prisma BigInt Conversion

**Problem:** `User.ID.Equals(intValue)` doesn't compile.

**Root cause:** Prisma's BigInt fields require `types.BigInt()` wrapper, not plain Go `int`.

**Error:**
```
cannot use userID (type uint64) as type BigIntValue
```

**Fix:**
```go
import "github.com/steebchen/prisma-client-go/runtime/types"

Link.UserID.Equals(types.BigInt(userID))
```

**Files affected:** `internal/db/prisma_store.go` (all methods using IDs)

---

## 3. "ensure: no binary found" at Container Runtime

**Problem:** Docker container starts but crashes with "ensure: no binary found".

**Root cause:** `prisma generate` was run on the host (macOS/arm64), embedding the wrong platform's query-engine binary. The container runs Linux, so the binary doesn't match.

**Fix:** Move `prisma generate` INTO the Dockerfile's builder stage:
```dockerfile
RUN go run github.com/steebchen/prisma-client-go prefetch
RUN go run github.com/steebchen/prisma-client-go generate
```

**Files affected:** `Dockerfile`

---

## 4. Docker Networking — localhost vs Service Names

**Problem:** API container can't connect to Postgres/Redis using `localhost`.

**Root cause:** Inside Docker's network, services are reachable by service name (e.g., `postgres:5432`), not `localhost`. The `.env` file uses `localhost` for host-side `go run`, which is wrong inside containers.

**Fix:** Add `environment:` overrides in `docker-compose.yml`:
```yaml
api:
  environment:
    DATABASE_URL: postgresql://urlshortener:urlshortener_dev_pw@postgres:5432/urlshortener?schema=public
    REDIS_ADDR: redis:6379
```

**Files affected:** `docker-compose.yml`

---

## 5. Missing Postgres Tables — generate vs db push

**Problem:** `relation "public.links" does not exist` error.

**Root cause:** `prisma generate` (builds Go client) and `prisma db push` (creates tables) are SEPARATE steps. Only running `generate` doesn't create tables.

**Fix:** Run both:
```bash
go run github.com/steebchen/prisma-client-go generate  # builds Go client
go run github.com/steebchen/prisma-client-go db push   # creates tables in Postgres
```

**Note:** `db push` must be run with `DATABASE_URL` pointing to the actual Postgres instance (localhost:5432 if running host-side, postgres:5432 if inside Docker).

---

## 6. Redis Stream NOGROUP Error

**Problem:** `NOGROUP No such key 'urlshortener:click_events' or consumer group 'analytics-workers'` spammed in logs.

**Root cause:** `XGroupCreate` fails when the stream doesn't exist yet (before any click event has been pushed). The worker calls `XGroupCreate` on every poll, and it fails every time until the first event creates the stream.

**Fix:** Use `XGroupCreateMkStream` instead of `XGroupCreate`:
```go
// Wrong:
c.rdb.XGroupCreate(ctx, StreamKey, ConsumerGroup, "$")

// Right:
c.rdb.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "$")
```

`MkStream` creates the stream if it doesn't exist, preventing the NOGROUP error.

**Files affected:** `internal/redis/stream.go`

---

## 7. redis.Nil Log Spam in Worker

**Problem:** `analytics worker: ReadClickEvents error: redis: nil` logged every 5 seconds.

**Root cause:** `XReadGroup` returns `redis.Nil` when the block timeout expires with no new events. This is NORMAL behavior (no data available), not an error. The worker was logging it as an error.

**Fix:** Check for `redis.Nil` and continue silently:
```go
if err == goredis.Nil {
    continue  // normal timeout, no events
}
log.Printf("error: %v", err)  // only log real errors
```

**Files affected:** `internal/worker/worker.go`

---

## 8. Import Name Conflict — redis vs redis

**Problem:** `redis redeclared in this block` compiler error.

**Root cause:** Both `github.com/redis/go-redis/v9` (external library) and `github.com/yourorg/urlshortener/internal/redis` (our package) are named `redis`. Go can't have two imports with the same name.

**Fix:** Alias the external library:
```go
import (
    goredis "github.com/redis/go-redis/v9"
    "github.com/yourorg/urlshortener/internal/redis"
)
```

**Usage:** `goredis.Nil` (external), `redis.Client` (internal)

**Files affected:** `internal/worker/worker.go`

---

## 9. QueryOrderDesc Not Available

**Problem:** `undefined: QueryOrderDesc` compiler error.

**Root cause:** The generated Prisma client doesn't expose `QueryOrderDesc` as a top-level constant in this version of prisma-client-go.

**Fix:** Sort in Go instead of using Prisma's OrderBy:
```go
// Instead of .OrderBy(ClickEvent.Timestamp.Order(QueryOrderDesc))
// Fetch all, sort in Go:
for i := 1; i < len(found); i++ {
    for j := i; j > 0 && found[j].Timestamp.After(found[j-1].Timestamp); j-- {
        found[j], found[j-1] = found[j-1], found[j]
    }
}
```

**Files affected:** `internal/db/prisma_store.go` (GetRecentClickEvents)

---

## 10. Port 8080 Already in Use

**Problem:** `listen tcp :8080: bind: address already in use`

**Root cause:** Old server process or Docker container still running and holding port 8080.

**Fix:**
```bash
# Kill any process on port 8080
lsof -ti:8080 | xargs kill -9

# Or stop Docker containers
docker-compose down
```

---

## 11. Cache Format Change — Backward Compatibility

**Problem:** After changing cache from plain URL string to JSON `{"url":"...","id":123}`, old cache entries break.

**Root cause:** Old entries are plain strings, new code tries to JSON-parse them.

**Fix:** Handle both formats in `GetLongURL`:
```go
var cached cachedLink
if err := json.Unmarshal([]byte(val), &cached); err != nil {
    // Old format: plain URL string
    return val, 0, nil  // linkID=0, click event will be skipped
}
return cached.URL, cached.ID, nil
```

**Files affected:** `internal/redis/cache.go`

---

## 12. StatsHandler Needs LinkStore but Only Has StatsStore

**Problem:** `GetRecentClicks` needs to look up a link by short code to get its ID, but `StatsHandler` only has `StatsStore` which doesn't include `GetLinkByShortCode`.

**Fix:** Type assertion — PrismaStore implements both interfaces:
```go
func getLinkByShortCode(s store.StatsStore, ctx context.Context, shortCode string) (*store.Link, error) {
    if ls, ok := s.(store.LinkStore); ok {
        return ls.GetLinkByShortCode(ctx, shortCode)
    }
    return nil, store.ErrNotFound
}
```

**Files affected:** `internal/handlers/stats.go`

---

## 13. Prisma Codegen Required After Schema Changes

**Problem:** Build fails with undefined types after changing `schema.prisma`.

**Root cause:** The Prisma Go client is GENERATED from the schema. Changing the schema doesn't automatically update the generated code.

**Fix:** Always run after schema changes:
```bash
go run github.com/steebchen/prisma-client-go generate
go run github.com/steebchen/prisma-client-go db push
```

---

## 14. .gitignore for Generated Prisma Files

**Problem:** Generated Prisma client files in `internal/db/prisma/` shouldn't be committed.

**Fix:** `.gitignore` includes:
```
internal/db/prisma/
```

But `internal/db/.gitignore` (separate file) also ignores generated files. Both are needed.

---

## 15. Go Module Replace Directives

**Problem:** Go module downloads fail in restricted network environments.

**Root cause:** Some environments only allowlist `github.com` but not `proxy.golang.org` or `golang.org`.

**Fix:** `go.mod` has `replace` directives mapping `golang.org/x/*` → `github.com/golang/*` mirrors. Keep these — remove only if building outside the constrained network.