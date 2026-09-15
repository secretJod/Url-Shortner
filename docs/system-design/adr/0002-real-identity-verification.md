# ADR-0002: Introduce real identity verification before API-key issuance

## Status
Proposed — pending Phase 2 approval (this is a product-scope decision as much as a technical one, and must be explicitly approved per CLAUDE.md §31, not silently implemented).

## Problem
`POST /api/keys` mints a working API key for any syntactically valid, unverified email (SEC-01 in `05-security.md`) — an account-takeover-shaped vulnerability confirmed by direct code reading of `handlers/apikeys.go` and `db/prisma_store.go`'s `GetOrCreateUserByEmail`.

## Options considered

1. **Do nothing / accept as a known limitation of an anonymous-friendly tool.** Valid *only if* the product intentionally never treats "owns this email" as a security boundary anywhere (i.e., never uses email identity to gate anything sensitive). Rejected as the default, because the system already treats the resulting API key as sufficient to list a user's private link inventory (`GET /api/links`) — that already crosses into "this identity is trusted with private data," which is incompatible with zero-verification issuance.
2. **Magic-link email verification.** User submits email → server generates a signed, single-use, short-TTL token → emails a verification link → key is issued only after the link is visited. Requires an outbound email capability.
3. **OTP (one-time code) via email.** Similar trust model to option 2, marginally better UX (no link-click required across devices), same email-sending dependency.
4. **Full password-based auth with the already-existing (but unused) `User.passwordHash` field.** Rejected as the *sole* fix — a password alone doesn't prove email ownership at signup time (someone could still set a password for `victim@example.com` without ever proving they control that inbox), so this doesn't close SEC-01 by itself; could be layered on top of option 2/3 later as an additional convenience (remembering a password vs. re-verifying by email every time), but is a separate concern.

## Decision
Adopt **Option 2 (magic-link email verification)** as the minimum fix, since it requires no new persistent secret (no password to hash/store/reset-flow to build) and directly closes the "any email works" gap with the least new surface area. Requires selecting a genuinely-free-tier transactional email provider (e.g. Resend's free tier, or Amazon SES's free tier if within its usage cap, both evaluated for GENUINELY FREE / TEMPORARILY FREE / PAID-AFTER-LIMIT status per CLAUDE.md §21 during Phase 2, not decided here) — this ADR does not pick a specific provider, only the mechanism.

## Trade-offs
- Adds an external dependency (an email-sending provider) where none existed before — must be verified against the ₹0/$0 constraint (CLAUDE.md §20) before being locked in.
- Adds latency/friction to the "get an API key" flow (previously instant, now requires checking email) — an intentional trade-off, since the previous instant flow is the vulnerability itself.
- Does not retroactively fix keys already issued under the old flow for email addresses the actual owner never claimed — a one-time cleanup/notification concern for Phase 3 if this ships to an existing user base (not applicable yet, since this is pre-production).
