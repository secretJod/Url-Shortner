# 10 — Testing Strategy

Status markers follow `01-project-overview.md`'s vocabulary.

## CURRENT STATE — Confirmed test inventory

Repository-wide search confirms **7 Go test files**, up from the original audit's 4 — three new handler-level unit tests were added alongside the security fixes:

| File | What it covers | Kind |
| --- | --- | --- |
| `backend/internal/auth/apikey_test.go` | `GenerateAPIKey` randomness/prefix/hash-consistency, `ExtractBearerToken` parsing | Unit |
| `backend/internal/redis/ratelimit_test.go` | Sliding-window `Allow()` under/over limit, window sliding — requires a live Redis (skips if unreachable) | Integration-lite |
| `backend/internal/redis/stream_test.go` | Push/read/ack round-trip on the Redis Stream, `HashIP` determinism, `StreamLen` — requires live Redis | Integration-lite |
| `backend/internal/shortcode/base62_test.go` | Encode/decode round-trip, uniqueness over 10k sequential IDs, invalid-character rejection | Unit |
| `backend/internal/handlers/apikeys_test.go` | ✅ New: verification-token generation/hashing, expiry logic | Unit |
| `backend/internal/handlers/shorten_test.go` | ✅ New: `validateURL` scheme/format validation | Unit |
| `backend/internal/handlers/stats_test.go` | ✅ New: `ownsLink` authorization logic (the exact SEC-02a/b regression check the original audit called for) | Unit |

**Still confirmed absent** (verified by `find`/`grep`, not assumed):

- **No `prisma_store.go` tests** — the still-unfixed `GetTopLinks`/`GetLinkStats` full-table-scan aggregation logic, and every SQL interaction, remain untested. The `GetOrCreateUserByEmail` fix (now an atomic upsert) has no regression test proving it no longer races.
- **No middleware tests** — `OptionalAPIKeyAuth`, `RequireAPIKeyAuth`, `RateLimit` (the middleware wrapper), CORS, metrics middleware — unchanged.
- **No worker tests** — `worker.go`'s batching, ack, panic-recovery, and error-handling/retry logic is still untested, including the new `recover()` behavior itself.
- **No integration tests** — nothing spins up the full API + Postgres + Redis stack and exercises an end-to-end flow (create → redirect → stats → delete). Unchanged.
- **No frontend tests** — zero `*.test.jsx`/`*.spec.jsx` files anywhere under `frontend/`; no Vitest/Jest/RTL dependency. Unchanged.
- **No E2E tests** — unchanged.
- **No load/performance tests** — the scalability claims in `07-scalability.md` remain architectural analysis, not measured. Unchanged.
- **No concurrency tests** — the `GetOrCreateUserByEmail` fix has no `-race` test with concurrent goroutines proving the race is actually closed. Unchanged gap, now more important since the fix itself is unverified by test.
- **CI now runs the existing tests** — ✅ this part of the original audit's "no CI" gap is resolved (`.github/workflows/ci.yml` runs `go test ./...` on every push/PR), though it does not yet spin up Postgres/Redis service containers for the integration-lite tests, which still silently skip in CI exactly as they did locally.

## Engineering maturity signal (feeds into overall maturity classification)

Test coverage has grown from "two lowest-risk pure-function packages" to include three new handler-level unit tests, and those tests directly cover some of the auth/authorization logic this project's security fixes depend on (`TestOwnsLink`, verification-token expiry). This is real progress, but it is still narrow: no integration tests, no frontend tests, no concurrency tests, and the highest-risk untested code (`prisma_store.go`'s aggregation queries, the worker) is unchanged from the original audit. Net assessment: **INTERMEDIATE** testing maturity, up from BEGINNER-to-INTERMEDIATE, still the biggest drag on the project's overall engineering-maturity classification in `01-project-overview.md`.

## PROPOSED — Testing strategy by layer (still open)

1. **Unit tests** (no infra required):
   - `prisma_store.go`'s pure transformation logic (once `GetTopLinks`/`GetLinkStats` are refactored per `03-data-model.md` to real SQL aggregation, the remaining Go-side mapping logic should still be unit-tested).
   - Remaining request validation logic not yet covered.

2. **Integration tests** (real Postgres + Redis, run via Docker Compose in CI):
   - Full `PrismaStore` CRUD + a concurrency test for `GetOrCreateUserByEmail` (run with `-race` and concurrent goroutines) to actually verify the upsert fix holds under contention, not just read as correct.
   - `AnalyticsWorker` end-to-end, including a test that intentionally triggers the new `recover()` path and asserts the worker resumes processing afterward.

3. **API/handler tests** (Fiber's `app.Test()`):
   - Auth middleware ordering.
   - Regression tests locking in the now-fixed SEC-01/SEC-02a/SEC-02b behavior (e.g. `TestGetLinkStats_RequiresOwnership` should now pass — add it as a permanent regression guard, not just document the fix in prose).
   - Status code correctness for the two new endpoints (`GET /api/keys/verify`, `DELETE /api/links/:shortCode`).

4. **Frontend tests** (Vitest + React Testing Library): still not started. `AuthContext`/`ProtectedRoute` behavior, a regression test for the still-broken `AdminPage.jsx` once it's fixed, and a test asserting `DashboardPage.jsx`'s delete flow calls the real `DELETE` endpoint (now that it does — this should be locked in with a test, not just visually confirmed).

5. **Security tests**: several of the original findings this section named (SEC-01, SEC-02a/b) are now fixed in code but still lack automated regression tests beyond the new `TestOwnsLink` unit test — add handler-level tests that actually exercise the HTTP layer (auth header required, 404 for non-owner) rather than only the underlying `ownsLink` helper.

6. **Performance/load tests**: still not started. A minimal `k6` script exercising `/:shortCode` and `/api/stats/top` at increasing concurrency remains the way to convert `07-scalability.md`'s UNKNOWN items into measured facts.

7. **CI wiring**: ✅ partially done — `go test ./...` now runs in CI. Still missing: Postgres/Redis service containers in the CI job (so the integration-lite Redis tests actually run instead of skipping) and `npm test` for the frontend (no frontend tests exist yet to run).
