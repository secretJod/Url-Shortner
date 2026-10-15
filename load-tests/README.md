# Load tests

k6 scripts run against the real, live deployment (not localhost, not
Codespaces) — see each script's header comment for exact invocation.

## Results — 2026-10-15, against the self-hosted deployment via Cloudflare quick tunnel

**`rate-limit-validation.js`** — 30 rapid requests from one client:
- 19 allowed (302), 11 correctly rejected (429), every rejection carrying a
  `Retry-After` header — matches the anonymous 20 req/min cap
  (`backend/internal/middleware/ratelimit.go`) exactly.
- 100% of checks passed.

**`redirect-latency.js`** — 11 requests over 60s, deliberately kept under
the rate limit to measure legitimate-traffic latency rather than rejection
behavior:
- 0% errors, 100% successful redirects.
- p95 969ms, median 240ms.

**Caveat, stated plainly:** the quick-tunnel path (browser → Cloudflare edge
→ `trycloudflare.com` tunnel → Colima VM → nginx → api) adds real latency
that a production network path or a named tunnel would not have. These
numbers measure *this* path on this date, not the backend's raw capacity.
