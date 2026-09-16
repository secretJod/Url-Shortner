# ADR-0005: Introduce nginx reverse proxy (TLS, security headers, optional URL masking) and deploy the containerized stack to an always-on free host

## Status

**Split decision, two independent parts:**

- **nginx reverse proxy (headers/routing scope): Accepted and implemented.** `nginx/nginx.conf` now runs in front of the API in `docker-compose.yml`, providing security headers, gzip, and a single public entrypoint. TLS termination is not yet configured — deployment-target-dependent. URL masking/cloaking was **not adopted**; the shipped config keeps standard 302-redirect semantics, with the masking option documented and disabled by default (see `02-high-level-design.md`).
- **Deployment target (Oracle/Render/Cloud Run options below): Superseded, then moot.** This ADR's deployment-target decision was superseded by `adr/0007-cicd-pipeline.md`'s self-hosted-runner design. That design was itself implemented and later removed. The options below are preserved as the original evaluation; they reflect neither the current plan (there is none — see `06-deployment.md`) nor an active recommendation.

## Problem
Two related gaps exist in the current deployment posture (see `02-high-level-design.md` and `06-deployment.md`):

1. **No reverse proxy / edge layer.** Fiber listens on `:8080` with no TLS, no security headers, and hardcoded-localhost CORS (`05-security.md` SEC-07) — a hard blocker to any real deployment. Separately, the user has requested a specific product behavior — keeping the **short URL visible in the browser address bar** while serving the destination's content underneath ("masking"/"cloaking") — which the current 302-redirect mechanism (`backend/internal/handlers/redirect.go`) cannot provide by definition, since a 302 instructs the browser to navigate away and replace the address bar.
2. **No deployment target.** `docker-compose.yml` runs the full stack (postgres, redis, api, prometheus, grafana) locally only; there is no hosting decision, and the project's Prometheus/Grafana dashboards are not reachable or demonstrable anywhere.

## Options considered — nginx and masking

1. **No reverse proxy; keep Fiber directly exposed.** Rejected — leaves TLS, security headers, and CORS-origin flexibility unsolved, which `06-deployment.md` already identifies as blockers independent of masking.
2. **nginx reverse proxy, redirect semantics unchanged (302 passthrough), used only for TLS/headers/routing.** Solves the deployment blockers cleanly with no change to product behavior or security profile. Does **not** satisfy the masking request.
3. **nginx reverse proxy + Approach A: proxy_pass destination content under the short-URL path, with `sub_filter` rewriting relative links.** Satisfies masking for simple destinations. Breaks unpredictably for destinations with JS-driven asset loading, protocol-relative URLs, or service workers bound to their own origin; increases bandwidth cost (full response bodies flow through the proxy instead of a few-hundred-byte 302); raises phishing/cloaking classification risk with browsers and scanners; raises consent/liability questions for displaying third-party content under this project's domain.
4. **nginx reverse proxy + Approach B: full-page iframe shell served at the short-URL path.** Simpler to implement than Approach A but fails completely against any destination sending `X-Frame-Options`/CSP `frame-ancestors` (a large share of real sites) — no generic nginx-side fix without also proxying (converging back to Option 3).

## Decision — nginx and masking

Adopted (Option 2's scope) and implemented:

- **nginx as a reverse proxy** in front of the API for security headers, gzip, and routing (`nginx/nginx.conf`, wired into `docker-compose.yml`) — directly addresses the cited blockers. TLS termination was not part of this scope and remains deployment-target-dependent.
- **URL masking (Options 3/4) was not adopted.** Both mechanisms remain documented in `02-high-level-design.md` with their honest trade-offs, kept available as a reference if a future product decision revisits this — the shipped config keeps standard 302-redirect semantics with masking explicitly commented out and disabled by default.

## Options considered — deployment target (historical; see Status above)

1. **Fly.io.** Rejected — no longer offers an uncapped always-free tier; requires a card with automatic billing past a small allowance, failing the zero-cost constraint.
2. **Render free web service.** No card required; sleeps after 15 min idle causing 30-60s+ cold starts on both redirects and Grafana access. Genuinely free but undermines the "always-on, showcase-quality" goal.
3. **Google Cloud Run.** Free-tier-genuine only for scale-to-zero traffic (still has cold starts) or requires min-instances≥1 for always-on, which consumes the free allowance faster and risks billing without a hard $0 budget alert enforced. Card required at signup regardless.
4. **Oracle Cloud "Always Free" VM (Ampere ARM shape).** Permanently free compute (not a trial/credit), sized enough (up to 4 OCPU/24GB RAM) to run the full 6-container stack (api+postgres+redis+prometheus+grafana+nginx) on one host, genuinely always-on with no cold starts. Requires a card on file at account creation (Oracle states Always-Free usage itself does not auto-charge) and carries a documented, discretionary risk of Oracle reclaiming resources judged idle — a real but limited risk for an actively-serving instance.

## Decision — deployment target (historical; superseded, see Status)

At the time this ADR was written, the recorded decision was: primary target Oracle Cloud Always-Free VM, fallback Render free tier, with the Oracle card-at-signup conflict flagged but not resolved. **This decision was superseded before implementation** by `adr/0007-cicd-pipeline.md`'s self-hosted-runner design, which was itself later implemented and then removed. No cloud-hosted target from this list was ever deployed. Preserved here as the original evaluation only.

## Trade-offs

- **nginx (proxy scope only):** minimal — adds one more container/config surface to operate and monitor, in exchange for solving header/CORS blockers that are prerequisites for any deployment target regardless of masking. This trade-off was accepted and is now realized in the running stack.
- **Masking (not adopted):** fragile, higher operational/security/legal exposure, higher bandwidth cost, real risk of phishing/cloaking misclassification, and no partial/safe middle ground — documented in full in `02-high-level-design.md`. This is why it wasn't adopted.
- **Oracle Always-Free / Render fallback:** historical only — neither was ever provisioned; see `06-deployment.md` for current (undecided) deployment status.

## Next step
The nginx/masking decision is implemented and closed. The deployment-target question is open again — see `06-deployment.md` for current status before making any new infrastructure decision.
