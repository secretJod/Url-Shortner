# 📐 CONVENTIONS — Coding Patterns & How to Add Features

> **Purpose:** Documents the coding patterns used in this project. Follow these when adding new features.

---

## Architecture Pattern: Interface-Driven

**Rule:** Handlers NEVER import Prisma directly. They depend on `store.Store` interfaces.

```
handlers → store.Store (interface) ← db.PrismaStore (implementation)
```

**Why:** Testability (mock the store), flexibility (swap Prisma for another ORM).

**How to add a new store method:**
1. Add method to the appropriate interface in `internal/store/store.go`
2. Implement it in `internal/db/prisma_store.go`
3. Use it in the handler

---

## Handler Pattern

**All handlers follow this structure:**

```go
type XxxHandler struct {
    store store.SomeStore  // depends on interface, not Prisma
    redis *redis.Client    // if needed
}

func NewXxxHandler(s store.SomeStore) *XxxHandler {
    return &XxxHandler{store: s}
}

func (h *XxxHandler) HandlerMethod(c *fiber.Ctx) error {
    // 1. Parse request (params, body, query)
    // 2. Call store method
    // 3. Handle errors (ErrNotFound → 404, other → 500)
    // 4. Return JSON response
}
```

**Error handling pattern:**
```go
if err == store.ErrNotFound {
    return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}
if err != nil {
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "..."})
}
```

---

## Middleware Pattern

**Middleware returns `fiber.Handler`:**
```go
func SomeMiddleware(store store.SomeStore) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // do stuff
        return c.Next()  // or c.Status(401).JSON(...)
    }
}
```

**Middleware order in main.go:**
```
recover → logger → [authMW] → [rateLimitMW] → handler
```

---

## Prisma CreateOne Pattern

**⚠️ Critical:** Prisma's `CreateOne` takes required fields as individual args, but optional fields must go in a single variadic slice. You CANNOT mix individual args with a spread slice.

**Wrong (won't compile):**
```go
client.Link.CreateOne(
    Link.ShortCode.Set(code),
    Link.LongURL.Set(url),
    Link.CustomAlias.Set(true),  // ← this is a variadic arg
    Link.User.Link(...),          // ← this is also a variadic arg
)
```

**Right:**
```go
params := []LinkSetParam{
    Link.CustomAlias.Set(true),
}
if l.UserID != nil {
    params = append(params, Link.User.Link(...))
}
if l.ExpiresAt != nil {
    params = append(params, Link.ExpiresAt.Set(*l.ExpiresAt))
}

client.Link.CreateOne(
    Link.ShortCode.Set(code),  // required field (individual arg)
    Link.LongURL.Set(url),     // required field (individual arg)
    params...,                  // all optional fields (spread slice)
)
```

---

## Redis Cache Pattern

**Cache values are JSON, not plain strings:**
```go
type cachedLink struct {
    URL  string `json:"url"`
    ID   uint64 `json:"id"`
}
```

**Set:** `redis.Set(ctx, key, json.Marshal(cachedLink{url, id}), ttl)`
**Get:** `json.Unmarshal([]byte(val), &cachedLink)` — handle old plain-URL entries gracefully

---

## Fire-and-Forget Pattern (Analytics)

**Click events must NEVER block or fail a redirect:**
```go
func fireClickEvent(rdb *redis.Client, linkID uint64, referrer, ip string) {
    ev := ClickEvent{LinkID: linkID, Timestamp: time.Now().UnixNano(), ...}
    _ = rdb.PushClickEvent(context.Background(), ev)  // ignore errors
}
```

---

## Worker Pattern (Background Goroutine)

```go
func (w *Worker) Start(ctx context.Context) {
    go w.run(ctx)
}

func (w *Worker) run(ctx context.Context) {
    for {
        if ctx.Err() != nil { return }  // graceful shutdown
        // do work
        // handle redis.Nil as "no data, continue"
    }
}
```

---

## Testing Pattern

**Unit tests** (no external deps): `*_test.go` in same package
**Integration tests** (need Redis): skip if Redis not available
```go
func TestSomething(t *testing.T) {
    rdb := redis.New("localhost:6379", "", 0)
    if err := rdb.Ping(ctx); err != nil {
        t.Skip("Redis not available, skipping")
    }
    // test...
}
```

---

## Naming Conventions

| Type | Convention | Example |
|------|-----------|---------|
| Files | lowercase, no underscores | `prisma_store.go` |
| Packages | lowercase, single word | `store`, `handlers`, `redis` |
| Structs | PascalCase | `ShortenHandler`, `ClickEvent` |
| Functions | PascalCase (exported), camelCase (unexported) | `CreateLink`, `resolveShortCode` |
| Interfaces | PascalCase, often `XxxStore` | `LinkStore`, `StatsStore` |
| Constants | PascalCase | `StreamKey`, `ConsumerGroup` |

---

## Adding a New API Endpoint (Checklist)

1. **Add store method** to interface in `internal/store/store.go`
2. **Implement** in `internal/db/prisma_store.go`
3. **Create handler** in `internal/handlers/xxx.go`
4. **Register route** in `cmd/api/main.go`
5. **Add middleware** (auth, rate limit) as needed
6. **Test** with `curl` or write integration test
7. **Update docs**: `.ai/CODE_MAP.md`, `.ai/PROJECT_CONTEXT.md`, `KNOWLEDGE.md`

---

## Adding a New Database Table (Checklist)

1. **Add model** to `prisma/schema.prisma`
2. **Run**: `go run github.com/steebchen/prisma-client-go generate`
3. **Run**: `go run github.com/steebchen/prisma-client-go db push`
4. **Add type** to `internal/store/store.go`
5. **Add interface** to `internal/store/store.go`
6. **Implement** in `internal/db/prisma_store.go`
7. **Update docs**

---

## Import Aliases

When two packages have the same name, alias one:
```go
import (
    goredis "github.com/redis/go-redis/v9"  // external redis
    "github.com/yourorg/urlshortener/internal/redis"  // our redis
)
```

**Usage:** `goredis.Nil` (external), `redis.Client` (internal)