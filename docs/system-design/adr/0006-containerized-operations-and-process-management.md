# ADR-0006: Container-native process management (reject PM2), fully containerized dev/build/run lifecycle, Compose-first showcase

## Status
✅ **Accepted and implemented.** PM2 remains rejected. `docker-compose.yml` now has `healthcheck` blocks on all services (previously only `postgres`/`redis`); `restart: unless-stopped` and the existing SIGTERM handling in `main.go` are unchanged and already correct. The optional orchestration step-up (Docker Swarm or k3s/kind) discussed below remains proposed only — not adopted.

## Problem

The user asked whether PM2 should be used to manage this project's processes, and separately wants the project to be fully containerized (no host installs) with a strong "process management" story for a resume/production-readiness showcase. Two questions need an evidence-based answer:

1. Is PM2 an appropriate process manager for this stack?
2. What is the correct container-native equivalent, and what's the credible step-up for "process management at scale"?

## Evidence from the repository

- The backend is a **compiled Go binary** run via a 3-stage Docker build (`backend/Dockerfile`) — Go stage → Alpine runtime stage. There is no Node.js runtime process in production.
- The frontend is a **Vite-built React SPA** — static files (`frontend/dist`), served either by the Go binary itself (`app.Static`/`SendFile` in `backend/cmd/api/main.go`) or, in the proposed nginx layer (ADR-0005), by nginx. There is no `node` process serving the frontend at runtime, in current or proposed state.
- `docker-compose.yml` already defines 5 services (`postgres`, `redis`, `api`, `prometheus`, `grafana`), all with `restart: unless-stopped`. `postgres` and `redis` have `healthcheck` blocks (`pg_isready`, `redis-cli ping`); `api`, `prometheus`, `grafana` currently have **no** healthcheck (confirmed absent by reading the compose file).
- `backend/cmd/api/main.go:134-141` already implements graceful shutdown: `signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)` → on signal, `cancel()` (stops the analytics worker's context) → `app.Shutdown()` (Fiber drains in-flight requests before the process exits).
- An API `/health` endpoint already exists (`backend/internal/handlers/health.go`), currently Redis-only (see `08-reliability.md` for the Postgres-inclusion gap, a separate concern from this ADR).

## Options considered — process management

1. **PM2.** Rejected. PM2 is a Node.js process manager — it forks/monitors/restarts `node` processes and expects a JS entrypoint (or, via its `interpreter: none` fork mode, an arbitrary command, but this is not PM2's intended use and forfeits most of its actual features — clustering, log aggregation via `pm2 logs`, memory-based restarts tuned for V8 heaps). Concretely, in this repo:
   - There is no Node process to manage in the container that runs the API (a static Go binary).
   - There is no Node process to manage for the frontend (pre-built static files, not a running dev/preview server).
   - Running PM2 inside a container whose only job is to run one Go binary would mean installing a Node.js runtime into that image *solely* to run a process supervisor for a binary that doesn't need one — pure overhead (larger image, larger attack surface, another dependency to patch) with no corresponding benefit.
   - This is a recognizable anti-pattern / cargo-cult signal to anyone reviewing the repo: it reads as "used the process manager I know" rather than "used the process manager this stack needs." For a resume/production-readiness showcase specifically, an unjustified PM2 dependency undermines the "evidence-based engineering" narrative the rest of this audit is built on.
2. **A generic container-level init/supervisor inside the image (e.g. `tini`, `supervisord`, `s6-overlay`).** Not adopted as primary. `tini` (PID 1 signal-forwarding) has a legitimate narrow use case — but only if the container ran *more than one* foreground process or needed correct zombie-reaping for subprocesses, neither of which applies here: the Go binary is already PID 1 in its container, already handles SIGTERM/SIGINT itself (`main.go:134-141`), and spawns no subprocesses. Adding an init system would solve a problem this repo doesn't have.
3. **Container runtime as the process manager (Docker Compose's `restart` policy + healthchecks) — the container-native pattern.** Adopted. This is the industry-standard mapping of PM2's actual responsibilities onto containers:
   - **Crash-restart** (PM2's core job) → `restart: unless-stopped`, already present on all 5 services in `docker-compose.yml`.
   - **Liveness/readiness signaling** (PM2's `pm2 status`) → Docker `healthcheck` blocks, already present on `postgres`/`redis`; **proposed to add** to `api` (hitting the existing `/health` endpoint), `prometheus` (`/-/healthy`), and `grafana` (`/api/health`) — all endpoints these images already expose, no new code required for prometheus/grafana, and the API's endpoint already exists.
   - **Graceful reload/shutdown** (PM2's `pm2 reload`) → SIGTERM handling already implemented in `main.go:134-141`; Compose sends SIGTERM on `docker compose stop`/`down` by default, so this is already wired correctly, just not yet exercised via a documented healthcheck-aware rollout.
   - **"One process per container"** is the established container-native norm (see the Docker/OCI ecosystem's general guidance): each container does one job and is independently restartable/observable by the orchestrator, rather than one container hiding a multi-process supervisor tree from the orchestrator's view. Running PM2 (or any supervisor) *inside* a container works against this — it hides process state from `docker ps`/`docker inspect`/healthchecks and duplicates restart logic the container runtime already provides for free.

## Options considered — step-up orchestration (proposed, not adopted yet)

For a stronger "process management at scale" story than a single-host Compose file, two zero-cost, container-native options:

1. **Docker Swarm.** Can deploy the *same* `docker-compose.yml` nearly as-is via `docker stack deploy -c docker-compose.yml <stack>` (Swarm reads Compose v3 files natively). Adds: replica counts, rolling updates, declarative restart policies at the orchestrator level (`deploy.restart_policy`), and a multi-node story if ever needed — all with minimal new configuration surface beyond the existing compose file. Lowest-effort step-up.
2. **Lightweight Kubernetes (k3s or kind), local/free.** Stronger resume signal (Kubernetes experience is a more common industry ask than Swarm), but requires writing actual K8s manifests (Deployments, Services, ConfigMaps, health/readiness probes) — meaningfully more setup than Swarm reusing the existing compose file. Both k3s and kind are genuinely free (local binaries, no cloud spend), consistent with the project's zero-cost constraint.

Neither is required for correctness today — both are presented as an optional maturity step-up, distinct from the "reject PM2" decision, which stands on its own regardless of whether an orchestrator is ever adopted.

## Decision

Adopt, pending approval:

- **Reject PM2** for this stack — it manages Node.js processes; this stack has no long-running Node process to manage (Go binary backend, static-file frontend). Its presence would be unjustified overhead and an anti-pattern signal, not a production-readiness signal.
- **Adopt container-native process management**: rely on `restart: unless-stopped` (already present) for crash-restart, the existing SIGTERM handling in `main.go:134-141` for graceful shutdown, and **propose adding healthchecks to `api`, `prometheus`, and `grafana`** (currently only `postgres`/`redis` have them) as the direct container-native equivalent of PM2's status/liveness monitoring.
- **Propose, as an optional maturity step-up**, either Docker Swarm (`docker stack deploy` from the existing compose file — minimal extra complexity) or k3s/kind (stronger resume signal, more setup) as the credible "process management/orchestration at scale" story — not required for the base showcase to be honest and functional.

## Trade-offs

- **Container-native (`restart` + healthchecks) vs. PM2:** strictly better fit for this stack at zero added complexity — reuses mechanisms Docker already provides instead of introducing a language-mismatched dependency. No downside identified for this project's shape (single process per container, no clustering need within a container).
- **Swarm vs. k3s/kind:** Swarm reuses the existing compose file almost unchanged (fast, low-risk) but is a less common resume keyword today than Kubernetes; k3s/kind is a stronger signal but requires writing and maintaining separate K8s manifests, doubling the infra-as-code surface to keep in sync with the compose file. Both remain optional and are not prerequisites for the primary Compose-based showcase (see `06-deployment.md` PROPOSED "Fully containerized operations" section).
- **Not adding a general-purpose in-container init system (`tini`):** correct for the current single-process-per-container shape; would need revisiting only if a container's entrypoint ever spawned multiple long-lived subprocesses (not the case today for any of the 5 services).

## Next step
Present this decision alongside `06-deployment.md`'s PROPOSED "Fully containerized operations & process management" section for explicit Phase 2 approval before adding any healthcheck, compose `deploy.restart_policy` block, Swarm/k3s config, or CI change.
