# 08 — Reliability

Status markers follow `01-project-overview.md`'s vocabulary.

## CURRENT STATE — Failure mode analysis (evidence-based)

| Failure | Current behavior | Evidence | Status |
|---|---|---|---|
| Redis unreachable | Rate limiter **fails open** (all requests allowed, unlimited) | `middleware/ratelimit.go`, explicit comment "Fail open: a Redis hiccup shouldn't take down the whole API" | Unchanged. Reasonable availability-over-strictness trade-off, but still not observable — no metric/log distinguishing "fail open due to error" from "normal allow." |
| Redis unreachable, redirect path | Cache-miss path falls through to Postgres — redirects keep working, just slower | `handlers/redirect.go` | Unchanged. Graceful degradation is architecturally present but still not observable (structured logging still missing, see `09-observability.md`). |
| Redis unreachable, shorten path | **Hard failure** — `resolveShortCode` propagates the Redis error straight to a `500` | `shorten.go` | Unchanged. Single point of failure: all URL creation stops if Redis is down, no fallback ID-generation strategy. |
| Postgres unreachable | Redirects on cache hit still work; cache misses `500`; all writes `500` | `redirect.go`, `handlers/apikeys.go` | Unchanged expected behavior given Postgres is the system of record. |
| Health check while Postgres is down | ✅ **Fixed** — `/health` now checks **both** Redis and Postgres, returning `503` if either is unreachable | `handlers/health.go` — `HealthHandler.DB` is a `pinger` (`Ping(ctx)`) satisfied by `PrismaStore`, checked alongside Redis | Closes the original "orchestrator would keep routing to a broken instance" gap. |
| Analytics worker crash | ✅ **Fixed** — `worker.go`'s `run()` now wraps its loop body in a `defer recover()` that logs and restarts the loop (`go w.run(ctx)` again) if it hasn't been cancelled | `worker/worker.go` | Closes the original "unrecovered panic kills the whole API process" finding. The worker still runs as a goroutine inside the API process rather than a separate one (see Scaling row below) — recovery is fixed, isolation/independent scaling is not. |
| Analytics worker horizontal scaling | Still a single goroutine, single named consumer (`worker-1`), hardcoded | `worker/worker.go New()` | 🔜 Proposed, unchanged (see ADR-0003) — Option 1 (recover) of that ADR is done; Option 2 (extract `cmd/worker/main.go`) is not. |
| Graceful shutdown | Implemented for the HTTP server: SIGINT/SIGTERM → `cancel()` (stops the worker's context) → `app.Shutdown()` (Fiber lets in-flight requests finish) | `main.go` | Unchanged, correct pattern, no finding. |
| Retries / backoff on transient errors | **None** anywhere in the reviewed code for Postgres or Redis calls, aside from the worker's blanket `time.Sleep(1s)` on any `ReadClickEvents` error | `worker.go` | Unchanged — still a blunt instrument, no exponential backoff, no distinction between transient and permanent failure classes. |
| Idempotency | `POST /api/shorten` is not idempotent (no idempotency key) | `shorten.go` | Unchanged, low severity. |
| Backups / restore / DR | **No backup strategy found anywhere in the repo** | `docker-compose.yml` (only a bare named volume, no backup service/sidecar) | Unchanged — still a real gap for any deployment beyond throwaway local dev. |

## SLO/SLA/RTO/RPO — CURRENT STATE

**None are defined anywhere in the repository.** Unchanged from the original audit. This section proposes reasonable **starting-point targets**, explicitly labeled as assumptions requiring user sign-off.

## PROPOSED — targets (ASSUMPTIONS, pending approval)

| Metric | Proposed target | Rationale / assumption basis |
|---|---|---|
| Redirect availability (SLO) | 99.5% monthly | ASSUMPTION: reasonable for a free-tier, single-region, no-SLA-backing-infra deployment. |
| Redirect p95 latency (SLO) | < 150ms on cache hit, < 500ms on cache miss | ASSUMPTION: not measured — no load test has been run. |
| Write-path (shorten/keys) availability (SLO) | 99% monthly | ASSUMPTION: lower than redirect since it depends synchronously on both Postgres and Redis with no fallback path. |
| RPO | 24 hours | ASSUMPTION: **currently unachievable** — no backup automation exists at all (unchanged). |
| RTO | 4 hours | ASSUMPTION: reasonable for a zero-cost, no on-call, single-maintainer project. |
| SLA | **Not proposed** — committing to an external SLA on free-tier infra would be dishonest; recommend internal SLOs only. | |

## Reliability fixes — status update

1. **Add Postgres to `/health`** — ✅ **Done.** Was the highest-leverage, lowest-effort fix; it's implemented. Still proposed as a further refinement: split liveness (`/health/live`, always 200 if the process is up) from readiness (`/health/ready`, checks both dependencies) rather than one combined endpoint — 🔜 Proposed, not required but a clean next step.
2. **Wrap the analytics worker's `run()` loop in a `recover()`** — ✅ **Done.**
3. **Add a scheduled Postgres backup** — 🔜 Proposed, unchanged. Still no `pg_dump` cron or managed-provider backup wired up.
4. **Fallback ID generation strategy if Redis is down** — 🔜 Proposed, unchanged. Still a hard dependency; not yet decided whether the added complexity is justified (Phase 2 decision, still open).
5. **Horizontally scale the analytics worker** — 🔜 Proposed, unchanged (ADR-0003 Option 2). The code already supports parameterizing `ConsumerName`; the separate `cmd/worker` binary/container has not been built.

## Container-native process management (PM2 rejected) — status

**✅ Implemented**, per `adr/0006-containerized-operations-and-process-management.md`. What that ADR proposed has since landed:

| PM2 responsibility | Container-native equivalent | Status |
|---|---|---|
| Crash-restart a dead process | `restart: unless-stopped` (Compose) | ✅ Present on all services in `docker-compose.yml` |
| Liveness/readiness status | Docker `healthcheck` | ✅ Present on `postgres`, `redis`, `migrate` (one-shot), `api`, `nginx` via its dependency chain, `prometheus`, and `grafana` — the original gap ("only postgres/redis have healthchecks") is closed |
| Graceful reload (`pm2 reload`) | SIGTERM handling in the process itself | ✅ Already implemented, unchanged |
| Process supervision inside one unit | "One process per container" | ✅ Unchanged, already the shape of this repo |

The optional orchestration step-up (Docker Swarm or k3s/kind) discussed in ADR-0006 remains 🔜 **Proposed** and has not been adopted — the project still runs as a single-host Compose stack, which is consistent with the current, undecided deployment story (see `06-deployment.md`).
