# 09 — Observability

Status markers follow `01-project-overview.md`'s vocabulary.

## CURRENT STATE

| Pillar | Status | Evidence |
|---|---|---|
| Metrics | ✅ Implemented (Prometheus) + dead legacy JSON path (unchanged) | `internal/metrics/prometheus.go` defines `urlshortener_http_requests_total`, `urlshortener_http_request_duration_seconds`, `urlshortener_redirects_total`, `urlshortener_rate_limit_total`, `urlshortener_analytics_events_{processed,failed}_total`, `urlshortener_analytics_processing_duration_seconds`. Correctly registered via `promauto` and incremented at the right call sites. Route-path label cardinality (not raw URL) correctly avoids a common Prometheus cardinality mistake. |
| Metrics (legacy) | 🐞 Still dead code, unchanged | `internal/metrics.Metrics`/`Snapshot()` is still wired into every request via `middleware/metrics.go` (`main.go`), computing counters never read by any HTTP handler — confirmed still dead code, still backs the still-broken `AdminPage.jsx` expectation (see `04-api-design.md`). |
| Structured logging | 🐞 Still missing, unchanged | Same deferral comments still present (`middleware/ratelimit.go`, `handlers/redirect.go`). Only Fiber's `logger.New()` default access log plus scattered `log.Printf`/`log.Println`. |
| Request/correlation IDs | 🐞 Still missing, unchanged | No `X-Request-ID` generation or propagation found anywhere. Fiber's built-in `requestid` middleware still isn't used. |
| Health metrics | ✅ Improved | `/health` now checks **both** Redis and Postgres (was Redis-only) — see `08-reliability.md`. Still one combined endpoint rather than a liveness/readiness split (that split remains 🔜 Proposed). |
| Error metrics | Partial, unchanged | HTTP status codes are captured as a Prometheus label — 4xx/5xx rates are derivable. No structured error-type breakdown since structured logging still doesn't exist. |
| Resource metrics (CPU/mem/goroutines) | Unchanged | `client_golang`'s default registry includes Go runtime collectors (GC, goroutines, memory) via `promauto`'s defaults — likely true, not runtime-confirmed in this pass. |
| Alerting | ✅ Implemented | `monitoring/prometheus/alerts.yml` now defines alert rules (mounted into the `prometheus` container in `docker-compose.yml`), closing the original "Prometheus deployed but zero alerting rules" gap. |
| Tracing | Still missing, not currently justified, unchanged | No distributed tracing — reasonable to omit at current single-process scale; would become justified if the analytics worker is ever split into its own service (still 🔜 Proposed, see ADR-0003). |
| Dashboards | ✅ Implemented | `monitoring/grafana/provisioning/dashboards/urlshortener.json` + `dashboards.yml` + `monitoring/grafana/provisioning/datasources/datasource.yml` are now checked into the repo and mounted into the Grafana container — closes the original "zero dashboards/datasources provisioned" gap. Grafana admin credentials are now sourced from `GF_SECURITY_ADMIN_PASSWORD` (env-driven), closing the original "hardcoded admin/admin" gap. |

## PROPOSED — Remaining observability work

1. **Structured logging** using `log/slog` (Go 1.21+, already compatible with this project's Go 1.25 toolchain, zero new dependency) emitting JSON with fields: `request_id`, `method`, `path`, `status`, `latency_ms`, `api_key_id` (if authenticated), `short_code` (where applicable), `error` (structured). Replace every `log.Printf`/silent-swallow comment identified above. Still not started.
2. **Request ID middleware** (Fiber's built-in `requestid`, already available transitively, no new dependency), propagated into structured log fields and returned as an `X-Request-ID` response header. Still not started.
3. **Fix or retire the legacy JSON metrics path** (`internal/metrics.Metrics`) — either back a real `/api/admin/metrics` JSON endpoint with it (auth-gated, see `05-security.md` SEC-04, still unresolved) or delete it. Still not started.
4. **Liveness/readiness split** for `/health` (`/health/live` vs `/health/ready`) now that both dependencies are checked — a refinement on top of the fix already shipped, not a correctness gap.
