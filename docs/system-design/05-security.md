# 05 — Security Audit

Severity scale per CLAUDE.md §12/§13: CRITICAL / HIGH / MEDIUM / LOW / INFORMATIONAL.

---

## SEC-01 — Unauthenticated identity-minting endpoint enables account takeover

- **Severity: CRITICAL**
- **Location**: `backend/internal/handlers/apikeys.go:38-73` (`CreateKey`), `backend/internal/db/prisma_store.go:110-141` (`GetOrCreateUserByEmail`)
- **Root cause**: `POST /api/keys` accepts any syntactically-valid email (`net/mail.ParseAddress`, `shorten.go` uses the same package) with **no proof of ownership** — no email verification link, no OTP, no password. `GetOrCreateUserByEmail` transparently creates a `User` row for a never-before-seen email and immediately issues a valid API key bound to that user's `id`.
- **Impact**: Any unauthenticated attacker who knows or guesses a victim's email address (trivial — emails are not secret) can mint a working API key for that identity. That key then grants:
  - `GET /api/links` — full list of every link the victim has ever created (or that get created in the future once the attacker "owns" the identity for all `/api/shorten` calls made with that key)
  - The ability to create new links attributed to the victim's account
  - Because there is no way to list/see "your other keys" or revoke them, the victim has no visibility that an attacker has a live key for their identity, and no self-service revocation path exists at all (no `DELETE /api/keys/:id` route).
- **Evidence**: `apikeys.go:44` only validates *format* (`mail.ParseAddress`), not ownership. `db/prisma_store.go:126-134` creates the user unconditionally on first sight of the email, with a comment `// PasswordHash isn't part of Phase 2's scope (no login yet)`.
- **Recommendation**: Add real verification before an identity is usable — minimum viable, zero-cost fix: magic-link email verification (send a one-time signed token to the address, key is only issued after the link is clicked) using a free-tier transactional email provider (see `06-deployment.md` for GENUINELY FREE options). Until then, treat every "authenticated" action in this system as **only pseudo-authenticated** — it authenticates "possession of an API key that happens to be linked to an email," not "is this person."

## SEC-02a — Public, unauthenticated per-link analytics (BOLA / IDOR)

- **Severity: HIGH**
- **Location**: `main.go:102` (`app.Get("/api/stats/:shortCode", rateLimitMW, stats.GetLinkStats)`), `stats.go:34-62`
- **Root cause**: No `authMW` and no ownership check — any caller who knows a short code gets full click analytics for that link (total clicks, unique IPs count, top 5 referrers, daily click timeline).
- **Impact**: Short codes are base62-encoded sequential IDs (`shortcode/base62.go`, fed by a Redis `INCR` counter) — they are **enumerable in numeric order**, not random. An attacker can walk `Encode(1)`, `Encode(2)`, ... and pull analytics for every link ever created on the platform, including referrer strings that may embed sensitive data (session params, internal URLs, campaign tracking with PII in query strings) leaked by the referring page.
- **Evidence**: `shortcode/base62.go:21-37` confirms sequential encoding of a monotonic counter (`redis/counter.go`'s `INCR`); no randomization anywhere in the ID-minting path for non-custom aliases.
- **Recommendation**: Require ownership (API key matching `link.UserID`) for the analytics endpoints, or explicitly redesign short codes to be non-sequential (random suffix, or HMAC-derived) if "stats are public by design" is confirmed as intended product behavior in Phase 2 approval — either fix independently closes the enumeration half of this finding.

## SEC-02b — Public, unauthenticated recent-click-events endpoint leaks IP hashes/referrers

- **Severity: HIGH**
- **Location**: `main.go:104` (`app.Get("/api/links/:shortCode/clicks", rateLimitMW, stats.GetRecentClicks)`), `stats.go:117-155`
- **Root cause**: Same missing-auth pattern as SEC-02a, but exposes raw-ish per-event data (`ip_hash`, `referrer`, `device_type`, `country`) rather than aggregates.
- **Impact**: `ip_hash` is a truncated SHA-256 of the visitor's IP (`redis/stream.go HashIP`, 64 bits / 16 hex chars). While one-way, 64 bits of hash space over a small, guessable IPv4 range (or a known set of candidate visitor IPs, e.g. "did person X at IP Y click this link") is crackable by brute-force pre-image search — an attacker with a candidate IP list can confirm "did IP 1.2.3.4 click this specific link" by hashing candidates and comparing, defeating the anonymization intent. Combined with public reachability (no auth), this turns a "privacy-preserving" analytics field into a re-identification tool against anyone with a suspect-IP list.
- **Recommendation**: Same auth/ownership fix as SEC-02a. Independently, consider salting `HashIP` with a server-side secret (`HMAC-SHA256(secret, ip)` instead of plain `SHA256(ip)`) to prevent brute-force IP confirmation even if the endpoint is later made public/aggregate-only by design.

## SEC-03 — Password fields present in schema, never enforced anywhere (misleading security posture)

- **Severity: MEDIUM**
- **Location**: `prisma/schema.prisma:19` (`Link.passwordHash`), `prisma/schema.prisma:30` (`User.passwordHash`); confirmed via `grep -rn "PasswordHash" backend/internal --include="*.go"` → only read/written in `prisma_store.go` as pass-through storage, **never compared against anything** in any handler.
- **Root cause**: Schema anticipates password-protected links and password-based login; neither was implemented; the fields are silently non-functional.
- **Impact**: Not directly exploitable, but represents a **false sense of security** — a link created with an intended password (if any UI ever surfaces this field, none currently does) would redirect with zero enforcement. An operator or future engineer could reasonably (and wrongly) assume this is a working feature.
- **Recommendation**: Either implement enforcement (check password on redirect, prompt via a interstitial page, using bcrypt/argon2 rather than plain hash-and-store) or drop the columns until the feature is actually built. Decision belongs in Phase 2 approval, not silently resolved by an agent.

## SEC-04 — No privilege separation for `/admin`

- **Severity: MEDIUM**
- **Location**: `frontend/src/components/ProtectedRoute.jsx` (only checks `isAuthenticated`, i.e. "has *some* API key in localStorage"); `main.go:118` (`/admin` served via `serveIndex`, no server-side gate at all beyond the SPA shell); confirmed via `grep -rn "role\|Role\|isAdmin\|IsAdmin\|Admin" backend/internal --include="*.go"` → zero matches for any authorization concept.
- **Root cause**: No `role` field on `User`, no admin allow-list, no middleware distinguishing "any authenticated user" from "an operator."
- **Impact**: Today `/admin` only shows metrics (which are also broken, see `04-api-design.md`), so actual damage is currently low — but this is an architectural gap: **any** user who mints an API key (trivially, per SEC-01) can reach `/admin` in the UI. If admin functionality grows (it's a natural next feature for a URL shortener — e.g., viewing all users' links, disabling abusive links), there is currently zero backend enforcement mechanism to build on safely without adding one first.
- **Recommendation**: Add a `role` enum (`user`/`admin`) to `User`, an admin-only middleware, and gate any future privileged Go route with it before adding privileged functionality — not after.

## SEC-05 — Docker Compose ships hardcoded, plaintext dev credentials with no Redis AUTH

- **Severity**: MEDIUM in the current **local-dev-only** context (compose file is Docker-first per CLAUDE.md §4 and never described as a production deployment artifact); would be **CRITICAL** if this compose file were ever deployed as-is to any environment reachable from the internet.
- **Location**: `docker-compose.yml:8` (`POSTGRES_PASSWORD: urlshortener_dev_pw`, plaintext in the compose file itself, not even in `.env`), `docker-compose.yml:11,25` (both Postgres `5432` and Redis `6379` published to the host), `backend/.env.example:6` (`REDIS_PASSWORD=""` — Redis has no `requirepass` set anywhere, confirmed absent from the `redis-server` command args in `docker-compose.yml:26`).
- **Impact**: As pure local dev this is a completely normal, low-risk pattern. The finding is that **nothing in the repo marks this compose file as dev-only or provides a hardened production variant** — there's no `docker-compose.prod.yml`, no documented "change these before deploying" checklist, no secrets-management story. A developer copying this file verbatim onto a public VPS (a very plausible zero-cost deployment path, e.g. a free-tier cloud VM) would expose an unauthenticated Redis instance and a default-password Postgres instance directly to the internet.
- **Recommendation**: Produce a `docker-compose.prod.yml` (or profile) that: binds Postgres/Redis ports to `127.0.0.1` only (or removes host port publishing entirely, relying on the Docker network), requires `REDIS_PASSWORD`/`requirepass` to be set from a real secret, and never hardcodes a password inline in the compose file (use `${POSTGRES_PASSWORD}` from `.env`, which is already the pattern used for the `api` service's `env_file: .env`, just not applied consistently to `postgres`/`redis`).

## SEC-06 — No rate limiting on the redirect route

- **Severity: MEDIUM**
- **Location**: `main.go:122` — `app.Get("/:shortCode", ...)` has no `rateLimitMW` in its handler chain (every other API route does).
- **Impact**: The redirect path is the system's highest-traffic, most-exposed endpoint. Without rate limiting it can be used to: (a) brute-force-enumerate short codes at unlimited speed to harvest valid links (compounding SEC-02a/b), (b) hammer the Postgres fallback path on repeated cache misses for non-existent codes, driving unnecessary DB load, (c) be used as a redirect-amplification vector against arbitrary target URLs (each hit generates one outbound-looking request from every client that follows the 302, i.e. not server-side SSRF, but still a traffic-amplification/abuse vector against the target site).
- **Recommendation**: Apply `rateLimitMW` (per-IP) to the redirect route, tuned higher than the API anon tier (redirects are the expected common case, not the exception) — e.g. a distinct, more generous "redirect" tier ceiling with the same sliding-window mechanism already implemented.

## SEC-07 — CORS hardcoded to localhost; `/metrics` and `/health` unauthenticated and internet-exposed if the port is published

- **Severity**: LOW as pure current-state observation (CORS being localhost-only is *safe* today, just not deployment-ready); flagged as a **deployment blocker**, not a live bug, per CLAUDE.md §12 distinguishing findings from blockers.
- **Location**: `main.go:57-61` — `AllowOrigins: "http://localhost:5173,http://localhost:3000,http://localhost:8080"` is a fixed string, not env-driven.
- **Impact**: The API will simply reject cross-origin requests from any real deployed frontend domain (browser CORS preflight failure) — a functional blocker, not directly a vulnerability, but listed here because getting CORS *wrong* in the other direction (e.g. `AllowOrigins: "*"` with credentials) is the common failure mode teams introduce while "fixing" this, so it's flagged for careful handling in Phase 3.
- **Recommendation**: Make `AllowOrigins` env-driven (`cfg.AllowedOrigins`), explicit allow-list per deployment environment, never a wildcard combined with credentialed requests. Additionally, put `/metrics` behind either network isolation (Prometheus scrapes over the private Docker network only, port not published to the host) or basic auth — it is currently reachable by anyone who can reach the API's public port.

## SEC-08 — No structured logging; no correlation/request IDs; sensitive info in log lines

- **Severity: LOW** (an observability/forensics gap more than a direct vulnerability — tracked in depth in `09-observability.md`)
- **Location**: Confirmed via `grep -rn "structured logging" backend --include="*.go"` → two explicit deferral comments (`middleware/ratelimit.go:55`, `handlers/redirect.go:42`). Only Fiber's default `logger.New()` access log and scattered `log.Printf` in `main.go`/`worker.go`.
- **Impact**: In a real incident (e.g. investigating SEC-01/02 exploitation), there is no request-ID correlation across the access log, application errors, and the analytics worker's log lines — makes any forensic reconstruction materially harder.
- **Recommendation**: See `09-observability.md` for the structured logging design.

## Reviewed and found NOT to be issues (evidence-based, to avoid over-claiming per CLAUDE.md §37)

- **API key generation** (`internal/auth/apikey.go`): 256 bits of `crypto/rand` entropy, SHA-256 hash-at-rest, only-shown-once pattern, prefix for secret-scanning recognizability — this is solid, industry-standard design. No finding.
- **SQL/NoSQL injection**: Prisma-Go's typed query builder is used throughout (`Link.ShortCode.Equals(...)`, etc.) — no raw string concatenation into queries found anywhere in `prisma_store.go`. No finding.
- **Open redirect**: `validateURL` (`shorten.go:127-136`) restricts scheme to `http`/`https` only via `url.ParseRequestURI`, rejecting `javascript:`, `data:`, `file:`, etc. This *is* still "the whole product is a redirector by design" — that's expected/intended behavior for a URL shortener, not a vulnerability, and scheme validation correctly blocks dangerous-protocol injection. No finding beyond noting it as a strength.
- **XSS**: React's default JSX escaping is used throughout the reviewed frontend files; no `dangerouslySetInnerHTML` found in any reviewed component. No finding (not exhaustively verified against every component file, flagged as **UNKNOWN** for files not read in this pass rather than asserted clean).
- **CSRF**: Not applicable in the traditional sense — the API is Bearer-token authenticated (not cookie-session based), so classic CSRF (which relies on browsers auto-attaching cookies) does not apply to the authenticated endpoints. No finding.
