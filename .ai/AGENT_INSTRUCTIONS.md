# 🤖 AGENT INSTRUCTIONS — How to Work on This Project

> **Purpose:** Instructions for any AI agent (Claude, GPT, etc.) working on this project. Follow these to be productive immediately.

---

## START HERE

1. **Read `.ai/PROJECT_CONTEXT.md` first** — gives you the full picture in 2 minutes
2. **Read `.ai/CODE_MAP.md`** — tells you which file does what
3. **Read `.ai/GOTCHAS.md`** — prevents you from repeating known mistakes
4. **Read `.ai/CONVENTIONS.md`** — shows you the coding patterns to follow

**Do NOT** read every source file. The `.ai/` folder has everything you need. Only read source files when you need to make changes to a specific file.

---

## Project Rules

### 1. Interface-Driven Architecture
- Handlers depend on `store.Store` interfaces, NEVER on Prisma directly
- To add a new DB operation: add to interface in `store.go` → implement in `prisma_store.go`
- This makes the code testable and swappable

### 2. Prisma CreateOne Pattern
- Required fields = individual args
- Optional fields = single variadic slice spread with `...`
- NEVER mix individual optional args with a spread slice (won't compile)
- See `.ai/GOTCHAS.md` #1 for details

### 3. Redis Stream Analytics
- Click events are fire-and-forget — NEVER block or fail a redirect
- Worker handles `redis.Nil` as "no data" (normal timeout), NOT an error
- Use `XGroupCreateMkStream` (not `XGroupCreate`) to auto-create streams

### 4. Import Aliases
- External `github.com/redis/go-redis/v9` → alias as `goredis`
- Internal `internal/redis` → keep as `redis`
- Usage: `goredis.Nil`, `redis.Client`

### 5. Error Handling
- `store.ErrNotFound` → HTTP 404
- Other errors → HTTP 500
- Never expose internal errors to the client (return generic messages)

### 6. Testing
- Unit tests: no external deps, run with `go test ./...`
- Integration tests: need Redis, skip gracefully if not available
- Always test with `go build ./...` and `go vet ./...` before completing

---

## How to Add a New Feature

### New API Endpoint
1. Add method to store interface (`internal/store/store.go`)
2. Implement in PrismaStore (`internal/db/prisma_store.go`)
3. Create handler method (`internal/handlers/xxx.go`)
4. Register route in `cmd/api/main.go`
5. Add middleware (auth, rate limit) as needed
6. Test with `curl`
7. Update `.ai/CODE_MAP.md` and `.ai/PROJECT_CONTEXT.md`

### New Database Table
1. Add model to `prisma/schema.prisma`
2. Run: `go run github.com/steebchen/prisma-client-go generate`
3. Run: `go run github.com/steebchen/prisma-client-go db push`
4. Add type + interface to `internal/store/store.go`
5. Implement in `internal/db/prisma_store.go`
6. Update `.ai/CODE_MAP.md`

### New Background Worker
1. Create `internal/worker/xxx.go`
2. Follow the `AnalyticsWorker` pattern (context cancellation, graceful shutdown)
3. Start in `cmd/api/main.go` with cancellable context
4. Handle `redis.Nil` as normal (not error)

---

## Build & Verify Checklist

Before completing any task, run:
```bash
go build ./...    # must exit 0
go vet ./...      # must exit 0
```

If either fails, fix before proceeding.

---

## Current State (as of Phase 4)

| Phase | Status |
|-------|--------|
| 0: Scaffold | ✅ |
| 1: Core API | ✅ |
| 2: Auth + Rate Limit | ✅ |
| 3: Analytics Pipeline | ✅ |
| 4: Stats API | ✅ |
| 5: Load Testing + Observability | ⬜ Next |
| 6: Deployment Hardening | ⬜ |

---

## Phase 5: What's Next

**Load Testing, Caching Tuning, Observability** would include:
- Prometheus metrics (request count, latency, cache hit/miss)
- Structured logging (replace Fiber's default logger with `slog`)
- Load testing with `hey` or `vegeta`
- Cache TTL tuning
- Redis connection pool sizing
- Health check improvements (check Postgres too, not just Redis)

---

## Don't Do These

- ❌ Don't import Prisma in handlers — use store interfaces
- ❌ Don't block redirects with analytics writes
- ❌ Don't log `redis.Nil` as an error
- ❌ Don't use `XGroupCreate` — use `XGroupCreateMkStream`
- ❌ Don't mix individual args with spread slice in Prisma `CreateOne`
- ❌ Don't forget `types.BigInt()` for Prisma BigInt fields
- ❌ Don't commit `.env` or generated Prisma files
- ❌ Don't run `prisma generate` on host for Docker builds (do it in Dockerfile)

---

## File Priority (Read Order)

1. `.ai/PROJECT_CONTEXT.md` — master context
2. `.ai/CODE_MAP.md` — file map
3. `.ai/GOTCHAS.md` — known issues
4. `.ai/CONVENTIONS.md` — patterns
5. `PROJECT_OVERVIEW.md` — session log
6. Source files — only when making changes