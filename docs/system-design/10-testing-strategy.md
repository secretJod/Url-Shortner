# 10 — Testing Strategy

## CURRENT STATE — Confirmed test inventory

Repository-wide search confirms **exactly 4 test files exist, all Go unit tests on leaf utility packages**, and zero tests of any other kind:

| File                                        | What it covers                                                                                                                                                               | Kind                                   |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| `backend/internal/auth/apikey_test.go`      | `GenerateAPIKey` randomness/prefix/hash-consistency, `ExtractBearerToken` parsing                                                                                            | Unit                                   |
| `backend/internal/redis/ratelimit_test.go`  | Sliding-window `Allow()` under/over limit, window sliding — **requires a live Redis** (`testClient` calls `c.Ping` and `t.Skipf`s if unreachable, `ratelimit_test.go:11-15`) | Integration-lite (skips without infra) |
| `backend/internal/redis/stream_test.go`     | Push/read/ack round-trip on the Redis Stream, `HashIP` determinism, `StreamLen` — also requires live Redis, same skip pattern                                                | Integration-lite                       |
| `backend/internal/shortcode/base62_test.go` | Encode/decode round-trip, uniqueness over 10k sequential IDs, invalid-character rejection                                                                                    | Unit                                   |

**Confirmed absent** (verified by `find`/`grep`, not assumed):

- **No handler tests** — `shorten.go`, `redirect.go`, `stats.go`, `apikeys.go`, `health.go` have zero test coverage. The most security-and-correctness-critical code in the repo (auth middleware ordering, ownership checks, validation) is entirely untested.
- **No `prisma_store.go` tests** — the full-table-scan aggregation logic, the `GetOrCreateUserByEmail` race condition (see `03-data-model.md`), and every SQL interaction are untested.
- **No middleware tests** — `OptionalAPIKeyAuth`, `RateLimit` (the middleware wrapper, as opposed to the underlying `redis.Allow` which _is_ tested), CORS, metrics middleware.
- **No worker tests** — `worker.go`'s batching, ack, and error-handling/retry logic (the `time.Sleep(1s)` backoff, partial-batch-failure handling) is untested.
- **No integration tests** — nothing spins up the full API + Postgres + Redis stack and exercises an end-to-end flow (create → redirect → stats).
- **No frontend tests** — zero `*.test.jsx`/`*.spec.jsx` files anywhere under `frontend/`; no Vitest/Jest/React Testing Library dependency in `frontend/package.json`.
- **No E2E tests** — no Playwright/Cypress config or dependency anywhere.
- **No security tests** — no automated check for the auth/authorization gaps documented in `05-security.md` (e.g. a test asserting `/api/stats/:shortCode` requires ownership would have caught SEC-02a as a regression the moment it's fixed).
- **No load/performance tests** — the scalability claims in `07-scalability.md` are architectural analysis, not measured; there is no `k6`/`vegeta`/`hey` script or config anywhere in the repo to validate them.
- **No concurrency tests** beyond what the Redis rate-limit tests incidentally exercise sequentially (not concurrently) — the `GetOrCreateUserByEmail` race (`03-data-model.md`) has no test reproducing it, e.g. via `go test -race` with concurrent goroutines hammering `POST /api/keys` with the same new email.
- **No CI** to run even the 4 existing tests automatically — confirmed via absence of `.github/`. Tests exist but nothing enforces they pass before merge.

## Engineering maturity signal (feeds into CLAUDE.md §19 classification)

Four unit-test files covering only the two lowest-risk, pure-function packages (base62 codec, API key crypto) plus two Redis-dependent tests that silently skip when infra isn't available (meaning they may never have run in whatever environment produced this repo's history) is consistent with **BEGINNER-to-INTERMEDIATE** testing maturity, dragging down an otherwise more sophisticated architecture — reinforces the overall INTERMEDIATE engineering-maturity classification in `01-project-overview.md`.

## PROPOSED PRODUCTION STATE — Testing strategy by layer

1. **Unit tests** (no infra required, fast, run on every commit):
   - `prisma_store.go`'s pure transformation logic (once refactored per `03-data-model.md` to real SQL aggregation, the remaining Go-side mapping logic should still be unit-tested)
   - Request validation logic in every handler (`validateURL`, alias pattern, expiry parsing) — currently exercised only implicitly
   - `redis.HashIP`, rate-limit tier resolution logic in `middleware/ratelimit.go`

2. **Integration tests** (real Postgres + Redis, run via Docker Compose in CI, Docker-first per CLAUDE.md §4):
   - Full `PrismaStore` CRUD + the `GetOrCreateUserByEmail` concurrency race, run with `-race` and concurrent goroutines to reproduce and then verify the fix from `03-data-model.md`
   - `AnalyticsWorker` end-to-end: push events to the stream, run the worker, assert Postgres rows land and are ACKed; assert a single bad event doesn't block the batch (already-implemented behavior, currently unverified by any test)

3. **API/handler tests** (using Fiber's `app.Test()` httptest-style harness, no real network needed, Postgres/Redis can be the same Compose-provided instances as integration tests):
   - Auth middleware ordering (does `RateLimit` correctly read `OptionalAPIKeyAuth`'s context value)
   - **Authorization regression tests** for every finding in `05-security.md` — e.g. `TestGetLinkStats_RequiresOwnership` (currently would fail, documenting the gap as an explicit, visible red test until fixed — a good practice for tracking known issues as code rather than only as prose)
   - Status code correctness for every documented endpoint in `04-api-design.md`

4. **Frontend tests** (Vitest + React Testing Library — both free, no paid service, integrate naturally with the existing Vite toolchain):
   - `AuthContext`/`ProtectedRoute` behavior
   - Regression test for the `AdminPage.jsx` bug once fixed (assert the page renders real numbers given a mocked JSON metrics response, not the raw Prometheus text)
   - `DashboardPage.jsx` delete flow once implemented for real (assert it calls the DELETE endpoint, not just local state)

5. **Security tests**: encode the SEC-01 through SEC-07 findings as automated tests where feasible (e.g. SEC-01's "any email works" is inherently hard to "fix" with just tests since the intended fix requires new functionality — email verification — but the _regression_ — "an API key for user A can list user B's `/api/links`" — is directly testable today and should be, as it's a concrete authorization boundary check independent of whether email verification ships).

6. **Performance/load tests**: a minimal `k6` (free, open source) script exercising `/:shortCode` redirect and `/api/stats/top` at increasing concurrency, run manually before each production deploy milestone (not necessarily in CI, given free-tier CI minute constraints) — needed to convert the `07-scalability.md` UNKNOWN items (connection pool behavior, actual QPS ceilings) into measured facts.

7. **CI wiring**: GitHub Actions (GENUINELY FREE per `06-deployment.md`) running `go vet`, `go test ./...` (unit + integration via a Postgres/Redis service container, which GitHub Actions provides free), and `npm test` for frontend, on every PR — currently **zero** CI exists, so this is a net-new addition, not a fix to something broken.

Implementation of all of the above is a Phase 3 concern once the user approves — this document defines the target, per CLAUDE.md §18.
