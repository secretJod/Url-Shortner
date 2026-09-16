# ADR-0007: Committed CI/CD pipeline — GitHub Actions → ghcr.io → self-hosted-runner-as-simulated-VM → Compose deploy → Prometheus/Grafana

## Status

**Accepted, implemented, then reverted.** The self-hosted-runner/simulated-VM CD pipeline this ADR describes was built (`deploy/` directory, `.github/workflows/deploy.yml`) and later deliberately removed from the repository. What remains today from this ADR's decision is only the **CI** half — test/build/push to `ghcr.io`, in `.github/workflows/ci.yml`. Deployment target is undecided again, left for a future decision (see `06-deployment.md`). This ADR supersedes `adr/0005-nginx-reverse-proxy-and-deployment.md`'s deployment-target section for the purpose of *how the project was deployed while this design was active*; ADR-0005's nginx-scope decision is separate and unaffected by any of this. The rest of this document is preserved as the historical decision record.

## Problem

`06-deployment.md` and ADR-0005 previously documented a **menu** of possible deployment targets (Oracle Cloud Always-Free, Render, Google Cloud Run) with their respective card/cold-start trade-offs, left open for Phase 2 approval. The user has now made a decision: rather than choosing among external cloud providers, the project will use a single, concrete, fully local, fully containerized, $0, no-card CI/CD pipeline that still produces a genuine, demonstrable "push → build → deploy → monitor" resume artifact — without depending on any cloud account.

## Options considered

1. **Oracle Cloud Always-Free VM** (previously documented as the primary target in ADR-0005). **Rejected.** Requires a card on file at account signup, which conflicts with the project's disqualification of card-required services, even though the compute itself doesn't auto-bill. The user has decided not to accept this trade-off.
2. **Render free web service** (previously documented as the no-card fallback in ADR-0005). **Rejected.** No card required, but the free tier sleeps after 15 minutes idle, producing 30-60s+ cold starts on both redirects and the Grafana dashboard — undermines the "always demonstrable" showcase goal, and depends on an external provider's continued free-tier policy.
3. **Google Cloud Run** (previously documented as a conditional alternative in ADR-0005). **Rejected.** Requires a billing account/card at signup regardless of whether usage stays within the free allowance, which is the exact "requires payment method, bills automatically past a limit" pattern the zero-cost constraint disqualifies.
4. **Fully local, containerized self-hosted-runner pipeline** (this ADR). **Adopted.** No cloud account, no card, anywhere. Uses only: (a) GitHub Actions' free CI runners and free `ghcr.io` registry (both tied to the user's existing free GitHub account, no card), and (b) a container the user runs on their own machine, standing in for a cloud VM, registered as a GitHub Actions self-hosted runner. All compute for the "deploy target" is the user's own hardware — there is no external biller to trigger, ever.

## Decision

Adopt, as the single committed pipeline (see `11-cicd.md` for the full design):

- **CI**: on push to the GitHub repo, a GitHub Actions workflow runs `go test ./...` against the backend, then `docker build` using the existing multi-stage `backend/Dockerfile`, then pushes the resulting image to **GitHub Container Registry (ghcr.io)**, authenticated via the workflow's own `GITHUB_TOKEN` — no separate account or secret needed.
- **Deploy target**: a **containerized Linux "server"** running Docker, hosting a **self-hosted GitHub Actions runner**. This container is the stand-in for a cloud VM (e.g. AWS EC2) but runs entirely on the user's own machine, at $0, under their own control.
- **CD**: the pipeline's CD stage runs on that self-hosted runner: `docker compose pull` to fetch the freshly-pushed image from ghcr.io, then `docker compose up -d` to bring up the full stack (`api + postgres + redis + prometheus + grafana + nginx`) on the simulated-VM host.
- **Monitoring**: Prometheus + Grafana run inside that same containerized environment, exactly as they do in the current `docker-compose.yml`. Live dashboards, plus a green CI/CD pipeline run, are the demonstrable showcase artifact (via screenshots/recordings), reproducible by anyone who clones the repo — no external account required to reproduce any part of the flow.
- This design is documented as the single decided plan in `11-cicd.md`, not as one option among several; `06-deployment.md` now points to it instead of presenting a provider menu for the deploy-target decision.

## Reasoning

- **Strict zero-cost compliance.** Every previously-evaluated cloud option carried either a hard card requirement (Oracle, Cloud Run) or a functional trade-off that undermines the showcase goal (Render's cold starts). Running the "VM" as a container the user owns removes the external-provider dependency entirely — there is no biller in the loop, so there is nothing to disqualify.
- **Still a genuine CI/CD story.** This is not a downgrade to "just run it locally" — it is a real, automated pipeline: a GitHub-hosted build/test/push stage, a registry, and an automated pull-and-redeploy stage triggered by the same workflow, using the actual GitHub Actions product (self-hosted runners are a first-class, supported GitHub Actions feature, not a workaround).
- **Reproducibility.** Anyone can clone the repo, register their own self-hosted runner container, and reproduce the entire push→build→deploy→monitor flow without needing to sign up for anything beyond a free GitHub account they likely already have.
- **Consistent with the project's Docker-first standard.** The "VM" being a container, not a bare-metal or cloud host, keeps the entire pipeline inside the project's existing Docker-first operating model.

## Trade-offs

- **Not a publicly reachable URL.** Because the "VM" is a container on the user's own machine, there is no live public domain a third party can visit without also running the stack themselves — this is a deliberate trade against the "always-on, publicly reachable" goal that motivated evaluating Oracle/Render/Cloud Run in ADR-0005. The showcase artifact is therefore screenshots/recordings of a real, working pipeline and dashboard, not a persistent public link.
- **Self-hosted runner operational responsibility.** Unlike GitHub-hosted runners, a self-hosted runner is the user's responsibility to keep running, patched, and secured (GitHub's own guidance recommends caution with self-hosted runners on public repos, since workflow code from PRs could execute on them) — for this project's private/personal use this risk is acceptable, but it is a real operational difference from a GitHub-hosted runner and should be scoped (e.g. restrict to `push` on trusted branches, not arbitrary `pull_request` from forks).
- **Runner container must be running for CD to execute.** If the user's machine/runner container is off, the CD stage simply won't have a runner to execute the deploy job on — this is a known and accepted limitation of a self-hosted deploy target, not a bug; CI (test/build/push) still succeeds independently on GitHub-hosted runners regardless of the runner's availability.

## Next step

This design was implemented (CI + CD, per `11-cicd.md`) and the CD half was subsequently removed. Current status and any future deployment-target decision start from `06-deployment.md`, not from this ADR — the CI half remains in production use, unchanged.
