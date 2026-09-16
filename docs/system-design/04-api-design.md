# 04 — API Design (built from `backend/cmd/api/main.go` route registrations + handler source)

Status markers follow `01-project-overview.md`'s vocabulary.

## CURRENT STATE — Full endpoint inventory

| Method | Path | Auth | Rate limit | Request | Success | Errors | Notes / Issues |
|---|---|---|---|---|---|---|---|
| GET | `/health` | none | none | — | `200 {status, redis, database}` or `503 {status:"degraded", ...}` | — | ✅ Now checks **both** Redis and Postgres (`handlers/health.go`) — closes the original "Postgres never checked" gap (see `08-reliability.md`). Still leaks raw backend error strings in the response body — minor info disclosure to any anonymous caller, unchanged. |
| GET | `/metrics` | none | none | — | `200 text/plain; version=0.0.4` (Prometheus exposition format) | `500 JSON` on gather/encode failure | Publicly exposes internal metrics with **no auth** — acceptable for a private Prometheus scrape target, a real exposure if this port is published directly (see `05-security.md` SEC-07, still open). Response format mismatch with the frontend Admin page is unchanged — see below. |
| POST | `/api/keys` | none (rate-limited only) | yes, per-IP anon tier | `{email}` | `202 {message: "verification email sent"}` | `400` invalid email, `500`, `502` mail send failure | ✅ No longer mints a key directly — sends a magic-link verification email instead. Closes SEC-01 (see `05-security.md`, resolved). |
| GET | `/api/keys/verify` | none (possession of the emailed token is the proof) | yes | `?token=...` | `201 {api_key, warning}` | `400` invalid/expired/already-used token, `500` | New endpoint (✅ Implemented) — issues the API key only after the one-time token is consumed. |
| POST | `/api/shorten` | optional (Bearer, validated if present) | yes (per key or per IP) | `{url, custom_alias?, expires_at?}` | `201 {short_url, short_code, long_url}` | `400` invalid url/alias/expiry format, `409` alias taken, `500` | Anonymous shortening allowed by design (product decision, not a bug) — unchanged. |
| GET | `/api/stats/top` | none | yes (per-IP anon tier) | `?limit=N` (default 10, capped 100) | `200 {top_links: [...]}` | `500` | Public by intentional design — a "top links" leaderboard. Still the most expensive endpoint in the system: backing query is an unbounded full-table scan (see `07-scalability.md`, unchanged). |
| GET | `/api/stats/:shortCode` | ✅ **required** (Bearer + ownership) | yes | — | `200 {short_code, long_url, total_clicks, unique_ips, top_referrers, daily_clicks}` | `404` not found **or** not owned (indistinguishable by design, to avoid confirming which short codes exist), `500` | ✅ Now owner-gated (`stats.go GetLinkStats` + `ownsLink` check) — closes SEC-02a (see `05-security.md`, resolved). |
| GET | `/api/links` | required (Bearer) | yes | — | `200 {links:[...]}` | `401` no/invalid key, `500` | Correctly scoped to `apiKey.UserID` server-side — unchanged, was already fine. |
| GET | `/api/links/:shortCode/clicks` | ✅ **required** (Bearer + ownership) | yes | `?limit=N` (default 20, capped 100) | `200 {clicks:[...]}` | `404` not found/not owned, `500` | ✅ Now owner-gated, same pattern as `/api/stats/:shortCode` — closes SEC-02b (see `05-security.md`, resolved). Query itself is now DB-side `ORDER BY ... LIMIT` (see `03-data-model.md`). |
| DELETE | `/api/links/:shortCode` | ✅ required (Bearer + ownership) | yes | — | `204 No Content` | `404` not found/not owned, `500` | New endpoint (✅ Implemented) — hard-deletes the link and invalidates its Redis cache entry. Backs the frontend delete button, which previously only updated local state. |
| GET | `/:shortCode` | none | ✅ yes (added) | — | `302` redirect | `404` JSON if not found/expired, falls back to SPA `index.html` for non-short-code paths | ✅ Now rate-limited (`rateLimitMW` in the handler chain) — closes SEC-06 (see `05-security.md`, resolved). |
| GET | `/`, `/login`, `/dashboard`, `/stats/*`, `/top`, `/admin` | none | none | — | serves `frontend/dist/index.html` | — | Purely static SPA shell serving; page-level access control is client-side only (`ProtectedRoute.jsx`) — expected for an SPA, real authorization now lives server-side for all endpoints that need it (`/api/links*`, `/api/stats/:shortCode`), unlike the original audit's finding. |
| Static | `/*` (via `app.Static`) | none | none | — | serves `frontend/dist/*` | — | Registered before the SPA/shortcode fallback routes, unchanged. |

## Known bug, still present: `/metrics` format vs. frontend `AdminPage.jsx` expectation

This was flagged in the original audit and **has not been fixed**:

- **Backend**: `/metrics` still gathers `prometheus.DefaultGatherer` and encodes standard Prometheus text-exposition format, content-type `text/plain; version=0.0.4`.
- **Frontend** (`AdminPage.jsx`): still does `const res = await apiClient.get('/metrics'); setMetrics(res.data);` then reads `metrics.uptime_seconds`, `metrics.status_codes`, `metrics.avg_latency_ms`, `metrics.cache_hit_rate`, `metrics.events_processed`, `metrics.events_failed` — a JSON shape that a Prometheus-text response can never satisfy. The Admin page is still functionally broken.
- **Root cause, unchanged**: the legacy JSON-producing type `internal/metrics.Metrics`/`Metrics.Snapshot()` still exists and is still wired into every request via `middleware/metrics.go` (`main.go`), computing state that is never read by any HTTP handler — confirmed still dead code.
- **Classification: 🐞 Known issue, Severity: MEDIUM** — unchanged from the original audit.

## Cross-cutting API issues — status update

- **Inconsistent auth enforcement across "similar" endpoints — ✅ Resolved.** `/api/stats/:shortCode` and `/api/links/:shortCode/clicks` now require ownership, matching `/api/links`. `/api/stats/top` remains intentionally public (aggregate-only), which is now a documented, deliberate asymmetry rather than an apparent oversight.
- **No pagination** anywhere except a bare `limit` query param — unchanged, acceptable at current scale, a scalability concern at higher volumes (see `07-scalability.md`).
- **No idempotency key support** on `POST /api/shorten` — unchanged, low severity.
- **Error response shape is consistent** (`{"error": "..."}"`) across handlers, including the two new endpoints — unchanged strength.
- **No API versioning** (`/api/...` not `/api/v1/...`) — unchanged, still fine pre-1.0.

## PROPOSED — Remaining API work

1. Rewrite `GetTopLinks`'s backing query (SQL-side aggregation, see `03-data-model.md`) — the authorization decision for this doc's original item #1 is resolved, but the performance dimension of `/api/stats/top` remains open.
2. Fix or remove the Admin metrics page: either add a JSON `/api/admin/metrics` endpoint backed by `internal/metrics.Metrics.Snapshot()` (gated by real admin auth, which doesn't exist yet — see `05-security.md` SEC-04), or delete the dead JSON metrics code path and rebuild `AdminPage.jsx` against Prometheus's HTTP API or a Grafana embed instead.
3. Introduce `/api/v1` versioning prefix before any public/external API consumers exist.
