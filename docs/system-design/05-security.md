# 05 — Security Audit

Severity scale: CRITICAL / HIGH / MEDIUM / LOW / INFORMATIONAL. Status markers follow `01-project-overview.md`'s vocabulary. This document was originally a Phase 1 audit; each finding below has been reconciled against the current code — resolved findings are marked ✅ **Implemented** with the fix described, unresolved ones remain open.

---

## SEC-01 — Unauthenticated identity-minting endpoint enables account takeover

- **Status: ✅ Implemented (resolved)**
- **Original severity: CRITICAL**
- **Location**: `backend/internal/handlers/apikeys.go`, `backend/internal/db/prisma_store.go`
- **Original root cause**: `POST /api/keys` accepted any syntactically-valid email with no proof of ownership, and immediately issued a working API key.
- **Fix implemented**: `POST /api/keys` no longer issues a key. It resolves/creates the user, mints a one-time signed verification token (15-minute TTL, hashed at rest in the new `verification_tokens` table), and emails a magic link via `internal/mail` (MailHog locally). The API key is only generated and returned by the new `GET /api/keys/verify?token=...` endpoint, which consumes the token (single-use, deleted immediately) and marks the user verified. An attacker who only knows a victim's email address can no longer obtain a key for that identity without also controlling the mailbox.
- **Residual notes**: there is still no self-service key listing/revocation (`DELETE /api/keys/:id`) — a verified user who later suspects a leaked key has no way to revoke it. Not a regression, just an unaddressed follow-on concern.

## SEC-02a — Public, unauthenticated per-link analytics (BOLA / IDOR)

- **Status: ✅ Implemented (resolved)**
- **Original severity: HIGH**
- **Location**: `main.go` (`app.Get("/api/stats/:shortCode", requireAuthMW, rateLimitMW, stats.GetLinkStats)`), `stats.go GetLinkStats`
- **Fix implemented**: the route now requires a valid API key (`requireAuthMW`) and the handler checks `ownsLink(link, apiKey)` — a link with no owner, or owned by a different user, returns `404` (not `403`), deliberately indistinguishable from "short code doesn't exist" so the endpoint doesn't confirm which short codes are valid to non-owners.
- **Residual notes**: short codes remain sequentially enumerable (`shortcode/base62.go`, fed by a Redis `INCR` counter) — this fix closes the *authorization* gap, not the enumerability of IDs themselves. `GET /api/stats/top` remains intentionally public (aggregate-only, no per-link ownership implied), which is a deliberate, documented asymmetry (see ADR-0004), not an oversight.

## SEC-02b — Public, unauthenticated recent-click-events endpoint leaks IP hashes/referrers

- **Status: ✅ Implemented (resolved)**
- **Original severity: HIGH**
- **Location**: `main.go` (`app.Get("/api/links/:shortCode/clicks", requireAuthMW, rateLimitMW, stats.GetRecentClicks)`), `stats.go GetRecentClicks`
- **Fix implemented**: same ownership-gating pattern as SEC-02a. Independently, `redis.HashIP` now takes a server-side secret (`IP_HASH_SECRET`, HMAC-style construction) rather than a plain unsalted `SHA256(ip)` — this was the independent hardening recommended in the original finding, closing the brute-force IP-confirmation risk even if this data were ever made public again by a future product decision.

## SEC-03 — Password fields present in schema, never enforced anywhere (misleading security posture)

- **Status: 🐞 Known issue (unresolved)**
- **Severity: MEDIUM**
- **Location**: `prisma/schema.prisma` (`Link.passwordHash`, `User.passwordHash`)
- **Current state**: unchanged from the original audit. `User.passwordHash` is still always written as an empty string (`GetOrCreateUserByEmail`'s upsert explicitly sets `""`, with a comment noting it's out of scope). `Link.passwordHash` is still round-tripped by `CreateLink`/`GetLinkByShortCode` but never compared against anything in any handler.
- **Recommendation**: unchanged — either implement enforcement (bcrypt/argon2, prompt on redirect) or drop the columns. Still a Phase 2 decision, not silently resolved.

## SEC-04 — No privilege separation for `/admin`

- **Status: 🐞 Known issue (unresolved)**
- **Severity: MEDIUM**
- **Location**: `frontend/src/components/ProtectedRoute.jsx`, `main.go` (`/admin` served via `serveIndex`, no server-side gate)
- **Current state**: unchanged. No `role` field on `User` in `schema.prisma`; no admin/authorization concept found anywhere in `backend/internal`. Any user who verifies an email (now requires mailbox control, per SEC-01's fix, but still no privilege tiering) can reach `/admin` in the UI.
- **Recommendation**: unchanged — add a `role` enum, an admin-only middleware, and gate any future privileged route with it before adding privileged functionality.

## SEC-05 — Docker Compose dev credentials and exposed ports

- **Status: 🔶 Partially resolved**
- **Severity**: MEDIUM in the current local-dev-only context; would be CRITICAL if deployed as-is to the public internet.
- **What was fixed**: ✅ Redis now requires a password — `docker-compose.yml`'s `redis` service passes `--requirepass "${REDIS_PASSWORD}"`, and the API/health checks authenticate with it. This closes the original "Redis has no AUTH at all" half of the finding.
- **What remains unresolved**: 🐞 Postgres's password is still hardcoded in plaintext in `docker-compose.yml` (`POSTGRES_PASSWORD: urlshortener_dev_pw`) rather than sourced from `.env`, and both Postgres (`5432`) and Redis (`6379`) still publish their ports to the host. There is still no `docker-compose.prod.yml`/hardened profile, no documented "change these before deploying" checklist. A developer copying this file verbatim onto a public host would still expose a default-password Postgres instance directly to the internet (Redis is now at least password-protected).
- **Recommendation**: unchanged for the Postgres half — move `POSTGRES_PASSWORD` to `${POSTGRES_PASSWORD}` from `.env` (the pattern already used for `REDIS_PASSWORD` and the `api` service), and consider binding both database ports to `127.0.0.1` only or removing host publishing entirely for any non-local-dev profile.

## SEC-06 — No rate limiting on the redirect route

- **Status: ✅ Implemented (resolved)**
- **Original severity: MEDIUM**
- **Location**: `main.go` — `app.Get("/:shortCode", rateLimitMW, ...)`
- **Fix implemented**: the redirect route now has `rateLimitMW` in its handler chain, same as every other route. Closes the original enumeration/scraping/DoS-amplification concern.

## SEC-07 — CORS hardcoded to localhost; `/metrics` and `/health` unauthenticated

- **Status: 🔶 Partially resolved**
- **What was fixed**: ✅ `AllowOrigins` is now env-driven (`CORS_ALLOWED_ORIGINS` via `config.Load()`), not a fixed string — this was the primary "deployment blocker" half of the original finding, and it's closed. The default value still lists localhost origins, which is correct and expected for local dev; a real deployment sets the env var to its real origin(s).
- **What remains unresolved**: 🐞 `/metrics` is still unauthenticated and reachable from the public internet in any deployment that publishes the API's port directly (not currently mitigated by network isolation or basic auth). This is unchanged from the original audit.
- **Recommendation**: put `/metrics` behind network isolation (private Docker network only, as is already true for local Compose since only nginx's `:80` needs to be public) or basic auth once a real deployment target is chosen.

## SEC-08 — No structured logging; no correlation/request IDs

- **Status: 🐞 Known issue (unresolved)**
- **Severity: LOW**
- **Current state**: unchanged. Only Fiber's default `logger.New()` access log and scattered `log.Printf` in `main.go`/`worker.go`. No request-ID generation/propagation. See `09-observability.md` for the design.

## New since the original audit: hardening items not originally flagged

- **✅ Non-root container user.** `backend/Dockerfile`'s runtime stage now creates and switches to `appuser`/`appgroup` (`USER appuser`) instead of running as root — closes a gap noted in `06-deployment.md`'s original CURRENT STATE section, not separately numbered there at the time.
- **✅ nginx security headers.** `nginx/nginx.conf` sets `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, and a `Content-Security-Policy` on all responses passed through it.

## Reviewed and found NOT to be issues (evidence-based, to avoid over-claiming)

- **API key generation** (`internal/auth/apikey.go`): 256 bits of `crypto/rand` entropy, SHA-256 hash-at-rest, only-shown-once pattern, prefix for secret-scanning recognizability — this is solid, industry-standard design. No finding. Unchanged.
- **SQL/NoSQL injection**: Prisma-Go's typed query builder is used throughout — no raw string concatenation into queries found anywhere in `prisma_store.go`. No finding. Unchanged.
- **Open redirect**: `validateURL` restricts scheme to `http`/`https` only, rejecting `javascript:`/`data:`/`file:`. Expected/intended redirector behavior, not a vulnerability. No finding. Unchanged.
- **XSS**: React's default JSX escaping is used throughout the reviewed frontend files; no `dangerouslySetInnerHTML` found. No finding (not exhaustively verified against every component file — flagged **UNKNOWN** for files not read, not asserted clean). Unchanged.
- **CSRF**: Not applicable — Bearer-token authenticated API, not cookie-session based. No finding. Unchanged.
