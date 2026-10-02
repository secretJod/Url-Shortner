# System Design Topics — Index

A concise, interview-ready index of the system-design concepts this URL shortener actually demonstrates, with a pointer to where each one lives in the code or design docs. Items are marked **[implemented]** or **[proposed]** — only what is genuinely built is claimed as done.

---

## 1. Core Shortening

- **Unique ID generation via Redis atomic counter** [implemented] — `INCR` on a single global key (`urlshortener:link_id_counter`) hands out monotonically increasing, collision-free IDs without DB locking. `backend/internal/redis/counter.go`.
- **Base62 encoding** [implemented] — numeric IDs are converted to compact `[0-9a-zA-Z]` short codes (and back, for tooling). Collision-free by construction since encoding is a bijection. `backend/internal/shortcode/base62.go`.
- **Custom alias reservation via SETNX** [implemented] — a custom alias is atomically claimed with `SETNX` (30s TTL safety net) before being committed to Postgres, preventing two concurrent requests from winning the same alias. `backend/internal/redis/cache.go` (`ReserveAlias`).
- See `docs/system-design/03-data-model.md`, `04-api-design.md`.

## 2. Caching

- **Cache-aside / read-through on redirects** [implemented] — the redirect handler checks Redis first (`GetLongURL`); on a miss it falls back to Postgres and backfills the cache. `backend/internal/redis/cache.go`, `backend/internal/handlers/redirect.go`.
- **Write-through on create** [implemented] — `SetLongURL` populates the cache at link-creation time, not just on first read.
- **TTL strategy** [implemented] — default 24h TTL, capped to the link's own expiry when it has one, so long-lived links still refresh periodically from Postgres.
- **Cache invalidation on delete** [implemented] — `InvalidateLongURL` removes the cache entry when a link is deleted, avoiding stale redirects.
- **Cached value carries link ID** [implemented] — the cache stores `{url, id}` (not just the URL) so a cache hit can still emit an accurate click event without a DB round-trip; backward-compatible with older plain-string cache entries.
- See `docs/system-design/02-high-level-design.md`, `07-scalability.md`.

## 3. Rate Limiting

- **Sliding-window-log algorithm** [implemented] — implemented with a Redis sorted set: expired entries trimmed (`ZREMRANGEBYSCORE`), current count checked (`ZCARD`), new request recorded (`ZADD`) — avoids the fixed-window boundary-burst problem. `backend/internal/redis/ratelimit.go`.
- **Per-API-key vs per-IP tiers** [implemented] — limiter keys are scoped (`ratelimit:apikey:<id>` vs `ratelimit:ip:<addr>`), applied via `backend/internal/middleware/ratelimit.go`.
- **Fail-open behavior** [implemented] — if Redis is unreachable, requests are allowed through rather than blocking all traffic (reliability over strictness for a non-critical control).
- See `docs/system-design/05-security.md`, `08-reliability.md`.

## 4. Reverse Proxy / Edge

- **Nginx as single entry point** [implemented] — fronts the API/frontend, only nginx's port is exposed externally. `nginx/nginx.conf`.
- **Security headers, gzip, real client IP (X-Forwarded-For)** [implemented] — configured in the same nginx layer so the app can trust `X-Forwarded-For` for rate limiting and IP hashing.
- See ADR `0005-nginx-reverse-proxy-and-deployment.md`, `docs/system-design/06-deployment.md`.

## 5. Asynchronous Processing

- **Redirect hot path decoupled from analytics writes** [implemented] — the redirect handler fires a click event onto a Redis Stream (`XADD`, fire-and-forget) instead of writing to Postgres inline. `backend/internal/redis/stream.go`.
- **Consumer group processing (XREADGROUP/XACK)** [implemented] — a background worker reads batches via `XReadGroup`, writes each event to Postgres, and only `XAck`s successfully written events (failed ones stay pending for reprocessing). `backend/internal/worker/worker.go`.
- **Stream bounded with MAXLEN ~** [implemented] — `XAdd` caps the stream at ~100k entries as a safety net if the worker falls behind.
- **Batching** [implemented] — worker reads up to 100 events per poll with a 5s block, not one-by-one.
- See ADR `0003-decouple-analytics-worker.md`.

## 6. Data Modeling & Indexing

- **Schema**: `links`, `users`, `api_keys`, `click_events`, `verification_tokens`, with foreign keys tying links/keys/tokens to owning users. `backend/prisma/schema.prisma`.
- **Indexes** [implemented] — `click_events` is indexed on `linkId`, `timestamp`, and the composite `(linkId, timestamp)`; `links`, `api_keys`, `verification_tokens` are indexed on `userId`.
- **Open item** [partially implemented] — top-links and per-link stats queries still perform full scans / aggregate without fully leveraging the composite index at scale; tracked as a known gap. See ADR `0001-analytics-query-rewrite.md` and `docs/system-design/03-data-model.md`.

## 7. AuthN / AuthZ

- **Passwordless API-key authentication** [implemented] — opaque bearer tokens, stored hashed (SHA-256) at rest so the raw key is never recoverable from the DB. `backend/internal/middleware/auth.go`, `backend/internal/handlers/apikeys.go`.
- **Magic-link email verification** [implemented] — verification tokens table backs an email-based identity flow (no passwords). See ADR `0002-real-identity-verification.md`.
- **Ownership-based authorization (BOLA/IDOR prevention)** [implemented] — per-link endpoints check that the requesting user/API key owns the link before allowing read/update/delete. See ADR `0004-authorization-model-for-analytics-endpoints.md`.
- **Gaps** [not implemented] — no JWT (sessions are opaque tokens, not self-contained/signed), no admin role / privileged-user tier yet.
- See `docs/system-design/05-security.md`.

## 8. Reliability

- **Graceful shutdown** [implemented] — the API listens for SIGTERM and drains in-flight requests before exiting. `backend/cmd/api/main.go`.
- **Worker panic recovery** [implemented] — the analytics worker's loop recovers from panics and restarts itself rather than crashing the process. `backend/internal/worker/worker.go`.
- **Health checks** [implemented] — Postgres + Redis connectivity checks exposed via a health endpoint. `backend/internal/handlers/health.go`.
- **Fail-open rate limiting, container restart policies, compose healthchecks** [implemented] — see `docker-compose.yml`.
- **Gap** [proposed only] — formal SLO/SLA/RTO/RPO targets are documented as goals, not yet operationally enforced. `docs/system-design/08-reliability.md`.

## 9. Observability

- **Prometheus metrics** [implemented] — custom counters/histograms (e.g. analytics events processed/failed, processing duration) with deliberately controlled label cardinality; exposed at `/metrics`. `backend/internal/middleware/prometheus.go`, `backend/internal/metrics`.
- **Grafana dashboards + alert rules** [implemented] — provisioned under `monitoring/`.
- **Gap** [not implemented] — structured logging and request/correlation IDs are still missing; current logging is plain `log.Printf`.
- See `docs/system-design/09-observability.md`.

## 10. Scalability Considerations

- **Stateless API + shared Redis/Postgres** [implemented] — API instances hold no local state, making the API tier horizontally scalable behind nginx.
- **Worker horizontal-scale readiness** [design-only] — the consumer-group design (`ConsumerGroup`, per-consumer names) supports running multiple worker instances, but today the worker still runs in-process alongside the API (`New("worker-1", ...)` started from `main.go`), not as an independently scaled service.
- See `docs/system-design/07-scalability.md` for the 100 → 100,000 user scaling analysis.

## 11. Privacy / Security Hygiene

- **HMAC-salted IP hashing** [implemented] — click-event IPs are hashed with HMAC-SHA256 keyed by a secret (`IP_HASH_SECRET`), truncated to 64 bits — enough for dedup without storing raw IPs. `backend/internal/redis/stream.go` (`HashIP`).
- **Secrets via environment**, **non-root container**, **CORS via env**, **scheme allowlist / open-redirect prevention** on submitted long URLs [implemented] — see `backend/internal/handlers/shorten.go`, `Dockerfile`, `docs/system-design/05-security.md`.

## 12. Containerization & CI/CD

- **Multi-stage Docker build** [implemented] — frontend build stage + Go build stage composed into a minimal runtime image.
- **One-shot migrate service** [implemented] — a dedicated compose service runs Prisma migrations before the app starts, rather than migrating in-process.
- **docker-compose orchestration** [implemented] — API, Postgres, Redis, nginx, worker (in-process), monitoring stack wired together. `docker-compose.yml`.
- **GitHub Actions CI** [implemented] — build, test, and image push to `ghcr.io`. `.github/workflows/ci.yml`, ADR `0007-cicd-pipeline.md`.
- **Gap** — actual deployment target (where the built image runs in production) is currently undecided. `docs/system-design/06-deployment.md`.

## 13. Trade-off Decisions (ADRs)

- `0001` — rewrite analytics queries for the indexed schema instead of ad-hoc aggregation.
- `0002` — adopt real (magic-link) identity verification instead of unverified email claims.
- `0003` — decouple click-analytics writes from the redirect path via a Redis Stream + worker.
- `0004` — define an ownership-based authorization model for analytics endpoints.
- `0005` — put nginx in front as the reverse proxy / single deployment entry point.
- `0006` — run operations (migrations, worker) as containerized processes rather than manual steps.
- `0007` — CI/CD pipeline design (build, test, publish to ghcr.io).

---

## Where to read the full detail

See `docs/system-design/01-project-overview.md` through `11-cicd.md`, and `docs/system-design/adr/` for the full write-ups behind every item above.
