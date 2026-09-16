# ADR-0001: Rewrite analytics aggregation from in-process full-table scans to indexed SQL

## Status
🔶 **Partially implemented.** `GetRecentClickEvents` was rewritten as a DB-side `ORDER BY timestamp DESC LIMIT N` query backed by the new `click_events(link_id, timestamp)` index (Option 2, as decided below). `GetTopLinks` and `GetLinkStats` were **not** rewritten — they still call `FindMany()` with no `WHERE`/`LIMIT` and aggregate/sort in Go. The index this ADR called for exists; two of the three query rewrites it justified are still outstanding. See `03-data-model.md`/`07-scalability.md` for current detail.

## Problem
`GetTopLinks`, `GetLinkStats`, and `GetRecentClickEvents` (`backend/internal/db/prisma_store.go`) each call `FindMany()` with no `WHERE`/`LIMIT` pushed to Postgres, loading entire tables into Go memory and aggregating/sorting there (confirmed by direct code reading — see `03-data-model.md` and `07-scalability.md`). This does not scale past a few thousand click events and is a confirmed data-loss/DoS risk at higher volumes (public, unauthenticated, rate-limit-bypassable via distinct IPs).

## Options considered

1. **Do nothing until it breaks.** Rejected — premature optimization is generally to be avoided, but this is not premature; it's an evidenced, not hypothetical, unbounded-scan pattern already present at zero data volume, with severity growing linearly with usage. Waiting converts a code-review-catchable issue into a production incident.
2. **Add SQL-side `COUNT`/`GROUP BY`/`ORDER BY`/`LIMIT` via Prisma-Go raw SQL (`client.Prisma.QueryRaw`).** Prisma-Go's typed query builder has no native aggregate/group-by support (confirmed: no `GroupBy` method found anywhere in the generated client usage across `prisma_store.go`), but it does expose raw SQL execution, keeping the rest of the Prisma-based data-access pattern intact.
3. **Introduce a separate SQL library (e.g. `sqlx`/`pgx`) alongside Prisma just for analytics queries.** Rejected for now — adds a second data-access pattern/dependency for a problem option 2 already solves without architectural churn; revisit only if raw SQL usage grows large enough to justify a dedicated query layer.
4. **Pre-aggregate into a summary table updated by the analytics worker** (e.g. `link_daily_stats(link_id, date, click_count)` upserted per event) to avoid aggregate queries entirely. Attractive for `GetTopLinks`/daily-chart use cases at very high scale, but adds write-side complexity (the worker must now do a read-modify-write or `ON CONFLICT` upsert per event) not justified until option 2's straightforward indexed aggregate queries are shown insufficient by an actual load test.

## Decision
Adopt **Option 2** now: raw parameterized SQL (via Prisma-Go's `QueryRaw`) for the three offending methods, backed by the new indexes proposed in `03-data-model.md` (`click_events(link_id, timestamp)`). Revisit Option 4 only if a future load test shows Option 2 insufficient at the actual observed scale — do not build it preemptively.

## Trade-offs
- Raw SQL bypasses Prisma-Go's type safety for these three queries — mitigated by keeping them narrowly scoped and covered by integration tests (per `10-testing-strategy.md`) that assert result shape.
- Requires a migration to add the new index — a schema change, which requires explicit user approval before being applied, not silently run.
