# ADR-0004: Authorization model for per-link analytics endpoints

## Status
✅ **Accepted and implemented — as Option 1, not the Option 2 recommended below.** Both `GET /api/stats/:shortCode` and `GET /api/links/:shortCode/clicks` now require the caller to own the link (`stats.go`'s `ownsLink` check, 404 for non-owners) — full ownership gating, not the "keep aggregates public, strip only identifying fields" middle ground this ADR recommended. `GET /api/stats/top` remains public and aggregate-only, unaffected. See `05-security.md` SEC-02a/SEC-02b (resolved) and `04-api-design.md`.

## Problem
`GET /api/stats/:shortCode` and `GET /api/links/:shortCode/clicks` are currently public and unauthenticated (SEC-02a/SEC-02b in `05-security.md`), while `GET /api/links` correctly requires ownership. It is unclear from the code whether public per-link stats were an intentional product decision (a "public leaderboard"-style feature, consistent with `GET /api/stats/top` which does appear intentionally public) or an oversight, since no comment or test documents the intent either way.

## Options considered

1. **Require ownership (API key matching `link.UserID`) on both endpoints**, matching `/api/links`'s existing pattern. Simplest, most consistent with the rest of the API, closes SEC-02a/b completely. Downside: removes a potentially-intended "anyone can check how their shared short link is performing" feature (some public URL shorteners intentionally expose basic click counts for any link, similar to link-shortener products like Bitly's public stats pages).
2. **Keep aggregate stats public but strip identifying detail** — `/api/stats/:shortCode` continues returning `total_clicks`/`daily_clicks` (harmless aggregate), but drops `top_referrers` (can leak sensitive referring-page query strings) and `/api/links/:shortCode/clicks` (raw per-event `ip_hash`/`referrer`/`device_type`) becomes owner-only unconditionally, since there is no aggregate-safe version of per-event data.
3. **Fully owner-gated, with an explicit opt-in "make stats public" flag per link** (new `Link.statsPublic` boolean, default `false`) — gives link creators the choice, closest to matching how real products (Bitly, etc.) handle this, but requires a schema change and new UI, more scope than a security fix strictly requires.

## Decision
Recommend **Option 2** as the default fix (least scope, closes the concrete data-exposure risk while preserving the apparently-intentional "public aggregate stats" feature implied by `GET /api/stats/top` already existing as a public leaderboard), with **Option 3 offered to the user as a richer alternative** if product requirements call for per-link control rather than a global policy. This decision is explicitly deferred to Phase 2 approval rather than picked unilaterally, since it changes user-facing behavior (a currently-public feature becoming owner-gated in the `top_referrers`/click-events case), not just an internal fix.

## Trade-offs
- Option 2 still leaves short codes sequentially enumerable (`shortcode/base62.go`), so `total_clicks`/`daily_clicks` for *any* link remains discoverable by enumeration even after this fix — a separate, independent decision about whether short codes should be randomized is out of scope for this ADR and not assumed; flagged for the user's awareness only.
- Whichever option is chosen, add the authorization regression tests proposed in `10-testing-strategy.md` so the chosen policy can't silently regress.
