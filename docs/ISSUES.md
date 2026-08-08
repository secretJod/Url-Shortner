# 🐛 Issue Tracker

> A log of past bugs, features, and chores tracked during development — the way a real project tracks its work. Items are ordered newest → oldest.

---

## Open Issues

### [CHORE] #12 — Add Prometheus + Grafana local monitoring stack
- **Status:** Done
- **Labels:** observability, dev-tools
- **Description:** Add `prometheus.yml` + docker-compose services so we can scrape `/metrics` and view dashboards locally. App's `/metrics` endpoint already emits Prometheus-format data (commit `7cb9c50`).
- **Related:** #11

### [FEATURE] #11 — Expose Prometheus metrics from the app
- **Status:** Done (merged in `7cb9c50`)
- **Labels:** observability
- **Description:** Add Prometheus-format metrics (http requests, redirects, rate-limit, analytics worker) so Grafana can visualize them.

---

## Done Issues

### [BUG] #10 — Stored links on dashboard opened a blank LinkSnip page
- **Status:** Fixed
- **Labels:** bug, frontend
- **Description:** Dashboard/Top Links built short URLs with `window.location.origin` (the Vite dev server on :5173), so clicking them loaded the SPA instead of redirecting. Fixed by pointing them at the backend (`http://localhost:8080`).
- **Files:** `frontend/src/components/LinkTable.jsx`, `frontend/src/pages/TopLinksPage.jsx`

### [BUG] #9 — "Analyze" word invisible on landing hero
- **Status:** Fixed
- **Labels:** bug, frontend
- **Description:** Used `text-transparent bg-clip-text` gradient which didn't render in some browsers, making the word invisible. Switched to solid `text-brand-600` with a pulse.
- **Files:** `frontend/src/pages/LandingPage.jsx`

### [BUG] #8 — Rate limiter can block after Redis restart
- **Status:** Won't fix for now (fail-open behavior is intentional)
- **Labels:** bug, rate-limit
- **Description:** On Redis hiccup the middleware fails open so the API stays up. Logging needs to be added later (structured logging phase).

### [FEATURE] #7 — Admin/Stats API (click analytics endpoints)
- **Status:** Done
- **Labels:** feature, api
- **Description:** Added `GET /api/stats/:code`, `/api/stats/top`, `/api/links`, `/api/links/:code/clicks`.

### [FEATURE] #6 — Async click analytics pipeline
- **Status:** Done
- **Labels:** feature, analytics
- **Description:** Redis Stream + worker writes click events to Postgres. Redirect hot path stays fast.

### [BUG] #5 — Prisma query engine "binary not found" in Docker
- **Status:** Fixed
- **Labels:** bug, docker, prisma
- **Description:** `prisma generate` on host embedded macOS engine. Fix: generate inside the Docker build stage.
- **Files:** `Dockerfile`

### [BUG] #4 — API container couldn't reach Postgres/Redis
- **Status:** Fixed
- **Labels:** bug, docker
- **Description:** `localhost` inside Docker doesn't work; overrode `DATABASE_URL`/`REDIS_ADDR` in compose with service names.
- **Files:** `docker-compose.yml`

### [BUG] #3 — "relation public.links does not exist"
- **Status:** Fixed
- **Labels:** bug, prisma
- **Description:** `prisma db push` hadn't been run — only `generate`. Ran `db push` to create tables.

### [BUG] #2 — Prisma `CreateOne` variadic mixing compile error
- **Status:** Fixed
- **Labels:** bug, prisma
- **Description:** Go can't mix individual variadic args with a spread slice. Moved optional fields into one slice.
- **Files:** `internal/db/prisma_store.go`

### [FEATURE] #1 — API keys + rate limiting
- **Status:** Done
- **Labels:** feature, auth
- **Description:** `usk_`-prefixed keys, SHA-256 hashing, sliding-window rate limiter in Redis, optional auth middleware.