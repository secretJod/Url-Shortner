# 🎯 NEXT AGENT PROMPT — Copy This to Start a New Session

> **Purpose:** Give this exact prompt to the next AI agent (Claude, GPT, etc.) when starting a new session on this project. It restricts the agent to the right files and gives full context in minimal tokens.

---

## Copy-Paste This Prompt to the Next Agent:

```
You are working on a URL Shortener project built in Go (Fiber + Prisma + PostgreSQL + Redis + Docker).

BEFORE DOING ANYTHING, read these files IN ORDER:
1. .ai/PROJECT_CONTEXT.md — full project context (2 min read)
2. .ai/CODE_MAP.md — every file explained
3. .ai/GOTCHAS.md — all 15 known bugs and solutions
4. .ai/CONVENTIONS.md — coding patterns to follow
5. .ai/AGENT_INSTRUCTIONS.md — rules and restrictions

DO NOT read every source file. The .ai/ folder has everything you need.
Only read source files when you need to make changes to a specific file.

RULES:
- Handlers depend on store.Store interfaces, NEVER import Prisma in handlers
- Prisma CreateOne: required fields = individual args, optional fields = single variadic slice
- Use types.BigInt() for Prisma BigInt fields
- Redis Stream: use XGroupCreateMkStream (not XGroupCreate)
- redis.Nil from XReadGroup is normal (no data), NOT an error
- Alias external go-redis as "goredis" to avoid name conflict with internal redis package
- Click events are fire-and-forget — NEVER block a redirect
- Always run `go build ./...` and `go vet ./...` before completing

CURRENT STATE: Phases 0-4 complete. Phase 5 (load testing, observability) is next.

To start the project:
  docker-compose up -d postgres redis
  go run github.com/steebchen/prisma-client-go generate
  go run github.com/steebchen/prisma-client-go db push
  go run ./cmd/api

What I want you to do: [DESCRIBE YOUR TASK HERE]
```

---

## How to Restrict the Next Agent

### Option 1: Minimal Restriction (Recommended)
Just give the prompt above. The `.ai/` files contain all the rules and gotchas. The agent will follow them.

### Option 2: Strict Restriction
Add these lines to the prompt:
```
STRICT RULES:
- Do NOT modify any file in .ai/ folder without my permission
- Do NOT change the architecture (interface-driven, handlers never import Prisma)
- Do NOT remove any existing tests
- Do NOT change the database schema without my approval
- ALWAYS run go build and go vet before completing
- ALWAYS update .ai/CODE_MAP.md and .ai/PROJECT_CONTEXT.md after making changes
- Ask me before starting any new phase
```

### Option 3: Task-Specific Restriction
```
SCOPE: Only work on [SPECIFIC TASK]. Do not touch any other files.
FILES YOU CAN EDIT: [list specific files]
FILES YOU CANNOT TOUCH: [list files]
```

---

## How to Update .ai/ Files After Each Phase

After completing any phase or major change, update these files:

### 1. `.ai/PROJECT_CONTEXT.md`
- Update the "Build Phases Status" table (mark phase as ✅)
- Add new endpoints to the "All API Endpoints" table
- Update the "Quick Start" if setup changed

### 2. `.ai/CODE_MAP.md`
- Add new files to the map
- Update existing file descriptions if they changed
- Add new ⚠️ Gotcha references if you hit new issues

### 3. `.ai/GOTCHAS.md`
- Add any new bugs/problems you encountered
- Number them sequentially (next would be #16, #17, etc.)

### 4. `.ai/CONVENTIONS.md`
- Add new patterns if you introduced new coding conventions
- Update checklists if the workflow changed

### 5. `.ai/AGENT_INSTRUCTIONS.md`
- Update "Current State" table
- Update "Phase X: What's Next" section
- Add new "Don't Do These" items if you discovered new gotchas

### 6. `.ai/QUICKSTART.md`
- Update if setup commands changed
- Add new test commands if you added endpoints

---

## Phase 5 Implementation Guide

When the next agent starts Phase 5 (Load Testing, Caching Tuning, Observability), they should:

### Step 1: Add Prometheus Metrics
```
New file: internal/metrics/metrics.go
- Request counter (total, by endpoint, by status code)
- Request duration histogram
- Cache hit/miss counter
- Redis stream length gauge
- Worker events processed counter
```

### Step 2: Add Structured Logging
```
Replace Fiber's default logger with slog
- Log JSON instead of text
- Add request ID for tracing
- Log cache hit/miss, DB queries, stream events
```

### Step 3: Improve Health Check
```
Update: internal/handlers/health.go
- Check Postgres connectivity (not just Redis)
- Return version, uptime, handler count
- Add /health/ready (readiness) vs /health/live (liveness)
```

### Step 4: Load Testing
```
Use `hey` or `vegeta` to test:
- Redirect throughput (cache hit vs miss)
- Shorten throughput
- Rate limiting behavior under load
- Worker drain rate under load
```

### Step 5: Cache Tuning
```
- Measure cache hit rate
- Tune TTL (currently 24h default)
- Consider cache warming on startup
- Add cache stamp protection (singleflight)
```

### After Phase 5, Update .ai/ Files:
1. Mark Phase 5 as ✅ in PROJECT_CONTEXT.md
2. Add new files to CODE_MAP.md (metrics.go, etc.)
3. Add any new gotchas to GOTCHAS.md
4. Update AGENT_INSTRUCTIONS.md with Phase 6 as next
5. Update QUICKSTART.md if new commands needed