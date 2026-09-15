# 07 — Scalability

## The central bottleneck (applies to every scenario below)

`backend/internal/db/prisma_store.go` has three methods that load **entire tables** into Go process memory on every single request, with no `WHERE` bound (beyond a link-id filter for the single-link case) and no `LIMIT` pushed to SQL:

- `GetTopLinks` (`prisma_store.go:302-357`): `s.client.Link.FindMany().Exec(ctx)` — **all links**, plus `s.client.ClickEvent.FindMany().Exec(ctx)` — **all click events across all links, ever** — just to count clicks per link and return the top N. Confirmed: no `Take()`/`Limit()`/`Where()` call anywhere in this function against `ClickEvent`.
- `GetLinkStats` (`prisma_store.go:224-299`): loads **all click events for one link** (bounded by link, unbounded by count) to compute total clicks, unique IPs, top referrers, and a daily-click histogram — all in Go, with an O(n²) insertion sort for both the referrer ranking and the daily-click ordering.
- `GetRecentClickEvents` (`prisma_store.go:390-437`): loads **all click events for one link** (again fully unbounded), sorts them in Go with an O(n²) insertion sort, then truncates to `limit` — the `LIMIT` is applied *after* the full unbounded fetch and full in-memory sort, not before.

This is confirmed by direct code reading, not assumption — every one of these three methods is an evidenced full-scan.

## Scenario: 100 users

- **Assumption**: light usage, ~a few hundred links, thousands of click events total.
- **Impact**: `GetTopLinks` and `GetLinkStats` are already doing full-table scans, but at this row count (low thousands) it's invisible — sub-10ms in practice. **No bottleneck yet.** This scale masks the underlying architectural problem, which is exactly why it's dangerous — it will not surface in early testing/demo.

## Scenario: 1,000 users

- **Assumption**: ~10x the links/clicks of the 100-user case if usage patterns hold — tens of thousands of click events.
- **Impact**: `GetTopLinks`'s "fetch every click event ever recorded, on every call" starts to matter — every call to `/api/stats/top` (a **public, unauthenticated, rate-limited-but-not-scale-limited** endpoint per `04-api-design.md`) now pulls tens of thousands of rows from Postgres into Go memory, builds a map, sorts, discards almost all of it. Latency moves from imperceptible to noticeable (tens to low hundreds of ms), and — because this endpoint has no per-endpoint concurrency cap — concurrent callers each pay this full cost independently, with no caching layer in front of it at all (confirmed: `GetTopLinks`/`GetLinkStats` are never written through Redis, unlike the redirect path).
- **First real bottleneck appears here**: Postgres connection pool exhaustion risk if `GetTopLinks` is hit concurrently by several clients while each holds a connection open for the (now slower) full scan — Prisma-Go's default pool sizing was not found configured anywhere in `config.go` or the `DATABASE_URL` connection string (no `connection_limit` param set), so this is running on Prisma's client defaults, unverified against actual load in this Phase 1 (no load test was run, per the read-only constraint) — labeled **UNKNOWN, needs a load test before production sign-off.**

## Scenario: 10,000 users

- **Impact**: The full-table-scan endpoints become a **genuine production incident risk**. Depending on link/click volume this could mean hundreds of thousands to low millions of `click_events` rows scanned per `/api/stats/top` or `/api/stats/:shortCode` call. Because these endpoints are public and unauthenticated (SEC-02a/b), there is nothing stopping either organic traffic growth *or* a trivial abuse pattern (repeatedly hitting `/api/stats/top`) from turning this into a self-inflicted denial-of-service: each request is O(total click events in the system), and the system has no cap on how many times per minute this O(n) work can be triggered beyond the generic per-IP rate limit (20/min anonymous) — 20 full-table scans per minute per anonymous IP, with no limit on the number of distinct IPs, **is already enough to degrade shared Postgres capacity** at this data volume.
- The redirect hot path itself (cache-aside via Redis) scales fine independently of this problem — that part of the architecture is sound and does not need redesign at this scale.

## Scenario: 100,000 users

- **Impact**: The analytics query pattern described above is **not viable at all** at this scale without the fix in `03-data-model.md` (SQL-side aggregation with indexes). Every other component (Redis cache, Redis counter, sliding-window rate limiter, Redis Stream) is a legitimately scalable pattern that would need only routine tuning (pool sizes, Stream consumer count) rather than redesign:
  - **Redis `INCR` counter for ID generation**: correctly atomic, no lock contention, scales linearly — no redesign needed, though a single global counter key means all ID-minting funnels through one Redis key; at extreme write rates this could become a single hot key, but Redis single-key ops are fast enough (single-digit microseconds) that this is very unlikely to be the limiting factor before other components fail first. **UNKNOWN precisely at what QPS this would bind — no load test performed.**
  - **Sliding-window rate limiter** (sorted-set based, `redis/ratelimit.go`): O(log N) per request as documented in its own code comment — correctly justified trade-off, scales fine.
  - **Analytics worker**: currently a **single goroutine, single named consumer (`worker-1`)** (`worker/worker.go:31`, hardcoded). At 100k-user scale, click-event ingestion volume could exceed what one consumer can drain from the stream (`BatchSize: 100`, `BlockTime: 5s` — `worker.go:32-33`), causing the stream to grow toward its `MAXLEN ~100000` cap (`redis/stream.go:54`) and start **dropping the oldest unprocessed click events** (Redis Streams with `MAXLEN` silently trim old entries once the cap is hit, even if they haven't been consumed) — this is a real, evidenced data-loss risk at sustained high click volume with the worker falling behind, not a hypothetical.
  - **Postgres**: without the indexing/aggregation fix in `03-data-model.md`, no scale of Postgres hardware fixes an O(all rows) query pattern — this must be fixed in code, not by throwing money at bigger instances (which would also violate the ₹0/$0 constraint).

## Summary ranking of scalability findings by impact

1. **CRITICAL for production**: `GetTopLinks`/`GetLinkStats`/`GetRecentClickEvents` full-table scans — fix before any real traffic (see `03-data-model.md` proposed fix #1).
2. **HIGH**: Single, non-horizontally-scaled analytics worker with a bounded stream that silently drops data under sustained backlog.
3. **MEDIUM**: No caching layer for the frequently-hit `/api/stats/top` (a natural cache-aside candidate, short TTL, same pattern already proven for the redirect path).
4. **LOW at current evidenced scale, worth monitoring**: Prisma-Go connection pool sizing is unconfigured/default — needs an actual load test to move out of UNKNOWN status.

No capacity numbers (e.g. "handles X req/s") are asserted anywhere in this document without a load test, per CLAUDE.md §15's "do not claim capacity without evidence" — none was run in this read-only Phase 1.
