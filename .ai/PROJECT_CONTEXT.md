# 🧠 PROJECT CONTEXT — Read This First

> **Purpose:** This file gives any AI agent or developer complete project context in under 2 minutes. No need to loop through files — everything you need to start working is here.

---

## TL;DR

**URL Shortener** built in **Go** with **Fiber** framework, **PostgreSQL** via **Prisma ORM**, **Redis** for caching/rate-limiting/analytics, and **Docker** for deployment. Phases 0-4 complete. Phase 5 (load testing, observability) is next.

---

## Stack (Locked)

| Layer | Tech | Version |
|-------|------|---------|
| Language | Go | 1.22 |
| Web Framework | Fiber | v2.52.14 |
| ORM | Prisma (prisma-client-go) | latest |
| Database | PostgreSQL | 16-alpine |
| Cache/Queue | Redis | 7-alpine |
| Containerization | Docker + Docker Compose | — |

---

## Architecture (One Diagram)

```
Client → Go API (Fiber, stateless)
              ├─ Redis: cache (short_code→long_url), counter (INCR), rate limiting (sorted set), streams (click events)
              ├─ PostgreSQL: links, users, api_keys, click_events (via Prisma)
              └─ Analytics Worker (goroutine): Redis Stream → Postgres
```

**Hot path (redirect):** Redis cache → 302 redirect → fire click event to Redis Stream (async)
**Write path (shorten):** Redis INCR → base62 encode → Prisma write → Redis cache write-through

---

## All API Endpoints (8 total)

| Method | Path | Auth | Handler | Phase |
|--------|------|------|---------|-------|
| GET | `/health` | No | `handlers.HealthHandler.Check` | 0 |
| POST | `/api/keys` | No | `handlers.APIKeyHandler.CreateKey` | 2 |
| POST | `/api/shorten` | Optional | `handlers.ShortenHandler.Shorten` | 1 |
| GET | `/:shortCode` | No | `handlers.RedirectHandler.Redirect` | 1 |
| GET | `/api/stats/:shortCode` | No | `handlers.StatsHandler.GetLinkStats` | 4 |
| GET | `/api/stats/top` | No | `handlers.StatsHandler.GetTopLinks` | 4 |
| GET | `/api/links` | Required | `handlers.StatsHandler.GetUserLinks` | 4 |
| GET | `/api/links/:shortCode/clicks` | No | `handlers.StatsHandler.GetRecentClicks` | 4 |

**Middleware chain:** `recover → logger → OptionalAPIKeyAuth → RateLimit → handler`

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

## Build Phases Status

| Phase | Status | What |
|-------|--------|------|
| 0 | ✅ | Scaffold, Docker, Fiber, Redis, /health |
| 1 | ✅ | Core shorten + redirect (base62, Prisma, Redis cache) |
| 2 | ✅ | API key auth + rate limiting (sliding window) |
| 3 | ✅ | Async analytics (Redis Streams → worker → Postgres) |
| 4 | ✅ | Admin/Stats API (4 endpoints) |
| 5 | ⬜ | Load testing, caching tuning, observability |
| 6 | ⬜ | Deployment hardening (K8s or finalized Docker Compose) |

---

## Quick Start (3 Commands)

```bash
git clone https://github.com/secretJod/Url-Shortner.git && cd Url-Shortner
cp .env.example .env && docker-compose up --build
go run github.com/steebchen/prisma-client-go db push  # one-time: create tables
```

Server runs at `http://localhost:8080`

---

## Key Design Decisions

1. **Interfaces over implementations** — `store.Store` interface decouples handlers from Prisma
2. **Fire-and-forget analytics** — click events pushed to Redis Stream, never block redirects
3. **Fail-open rate limiting** — Redis down = requests pass through (availability > strictness)
4. **Optional auth** — anonymous links allowed; invalid API key = rejected (no silent fallback)
5. **Cache stores JSON** — `{"url":"...","id":123}` so redirect handler has link_id for click events
6. **Consumer groups for analytics** — horizontal scaling + crash recovery via Redis XREADGROUP

---

## File Count

- **Go source files:** 20 (including tests)
- **Config files:** 5 (go.mod, go.sum, Dockerfile, docker-compose.yml, prisma/schema.prisma)
- **Documentation:** 3 (PROJECT_OVERVIEW.md, KNOWLEDGE.md, README.md)
- **AI context:** 6 files in `.ai/` folder

---

## What to Read Next

| If you want to... | Read this |
|--------------------|-----------|
| Understand every file | `.ai/CODE_MAP.md` |
| Follow coding patterns | `.ai/CONVENTIONS.md` |
| Avoid known bugs | `.ai/GOTCHAS.md` |
| Start coding immediately | `.ai/QUICKSTART.md` |
| Know how to work on this project | `.ai/AGENT_INSTRUCTIONS.md` |