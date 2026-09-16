# 07 — Scalability

Status markers follow `01-project-overview.md`'s vocabulary.

## The central bottleneck — partially fixed, partially still open

`backend/internal/db/prisma_store.go` originally had three methods that loaded **entire tables** into Go process memory on every request. Status now:

- **`GetRecentClickEvents` — ✅ Fixed.** Now `ClickEvent.FindMany(...).OrderBy(Timestamp.Desc).Take(limit)` — DB-side ordering and limiting, backed by the new `click_events(link_id, timestamp)` composite index. No more unbounded fetch or in-Go sort.
- **`GetTopLinks` — 🐞 Still unfixed.** `s.client.Link.FindMany().Exec(ctx)` — **all links**, plus `s.client.ClickEvent.FindMany().Exec(ctx)` — **all click events across all links, ever** — just to count clicks per link and return the top N. Confirmed: still no `Take()`/`Where()` call against `ClickEvent` in this function; sorting is still an O(n²) insertion sort in Go.
- **`GetLinkStats` — 🐞 Still unfixed.** Still loads **all click events for one link** (bounded by link, unbounded by count) to compute total clicks, unique IPs, top referrers, and a daily-click histogram, all in Go, with an O(n²) insertion sort for both the referrer ranking and the daily-click ordering.

This is confirmed by direct code reading, not assumption. The single highest-impact fix identified in the original audit (ADR-0001) is **half-applied**: the index exists and one of the three offending queries was rewritten to use it; the other two — including `GetTopLinks`, which backs a fully public, unauthenticated endpoint — were not.

## Scenario: 100 users

- **Assumption**: light usage, ~a few hundred links, thousands of click events total.
- **Impact**: `GetTopLinks` and `GetLinkStats` are still doing full-table scans, but at this row count it's invisible — sub-10ms in practice. **No bottleneck yet.** This scale still masks the underlying architectural problem, unchanged from the original audit.

## Scenario: 1,000 users

- **Assumption**: ~10x the links/clicks of the 100-user case — tens of thousands of click events.
- **Impact**: `GetTopLinks`'s "fetch every click event ever recorded, on every call" starts to matter — every call to `/api/stats/top` (still public and unauthenticated by design, now at least rate-limited per the SEC-06 fix — see `05-security.md`) pulls tens of thousands of rows from Postgres into Go memory. Latency moves from imperceptible to noticeable, and there's still no caching layer in front of it (confirmed: `GetTopLinks`/`GetLinkStats` are never written through Redis, unlike the redirect path).
- **First real bottleneck appears here**: unchanged from the original audit — Prisma-Go's default pool sizing is still unconfigured (no `connection_limit` param), still **UNKNOWN, needs a load test before production sign-off.**

## Scenario: 10,000 users

- **Impact**: The two still-unfixed full-table-scan endpoints become a **genuine production incident risk**, exactly as originally assessed. `/api/stats/top` is now rate-limited (per SEC-06's fix), which caps *anonymous per-IP* abuse somewhat, but does not cap aggregate load from many distinct IPs or from organic traffic growth — each request is still O(total click events in the system).
- The redirect hot path itself (cache-aside via Redis, now also rate-limited) continues to scale fine independently of this problem.

## Scenario: 100,000 users

- **Impact**: The analytics query pattern for `GetTopLinks`/`GetLinkStats` is **not viable at all** at this scale without the SQL-side aggregation fix (see `03-data-model.md`, ADR-0001). Every other component remains a legitimately scalable pattern needing only routine tuning:
  - **Redis `INCR` counter for ID generation**: unchanged, correctly atomic, scales linearly.
  - **Sliding-window rate limiter**: unchanged, O(log N) per request, correctly justified.
  - **Analytics worker**: still a **single goroutine, single named consumer (`worker-1`)** — unchanged from the original audit, though it now recovers from panics (see `08-reliability.md`) rather than being able to crash the whole API process. At 100k-user scale, ingestion volume could still exceed one consumer's drain rate, risking the same `MAXLEN`-triggered data loss described in the original audit. Horizontal scaling of the worker remains 🔜 Proposed (ADR-0003).
  - **Postgres**: without the `GetTopLinks`/`GetLinkStats` fix, no amount of Postgres hardware fixes an O(all rows) query pattern.

## Summary ranking of scalability findings by impact

1. **CRITICAL for production, still open**: `GetTopLinks`/`GetLinkStats` full-table scans — fix before any real traffic (see `03-data-model.md` proposed fix, ADR-0001). `GetRecentClickEvents` is no longer part of this finding — it was fixed.
2. **HIGH, still open**: Single, non-horizontally-scaled analytics worker with a bounded stream that can silently drop data under sustained backlog (crash-resilience was fixed; horizontal scaling was not — see `08-reliability.md`, ADR-0003).
3. **MEDIUM, still open**: No caching layer for the frequently-hit `/api/stats/top`.
4. **LOW at current evidenced scale, worth monitoring, unchanged**: Prisma-Go connection pool sizing is unconfigured/default — needs an actual load test to move out of UNKNOWN status.

No capacity numbers (e.g. "handles X req/s") are asserted anywhere in this document without a load test — none has been run.
