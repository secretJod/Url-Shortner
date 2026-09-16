# Simulated VM deploy target

This directory builds a container that plays the role of a cloud VM (think
EC2): a Linux host running Docker, with a self-hosted GitHub Actions runner
registered against this repository. The `deploy.yml` workflow's job
(`runs-on: self-hosted`) executes on it, pulling the image CI pushed to
`ghcr.io` and bringing up the full `docker compose` stack right there.

Everything below runs on your own machine, under your own Docker. No cloud
account, no card, $0.

## 1. Get a runner registration token

GitHub repo → **Settings → Actions → Runners → New self-hosted runner**.
Copy the token shown there (it's short-lived, generate a fresh one when it
expires).

## 2. Build the runner image

```bash
docker build -t urlshortener-vm-runner ./deploy/runner
```

## 3. Run it

The runner needs to talk to the Docker daemon so it can run `docker compose`
for the deploy step. Mount the host's Docker socket in:

```bash
docker run -d \
  --name urlshortener-vm \
  -e REPO_URL="https://github.com/<owner>/<repo>" \
  -e RUNNER_TOKEN="<token from step 1>" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  urlshortener-vm-runner
```

Check it registered: repo → Settings → Actions → Runners, it should show as
`simulated-vm`, online.

## 4. Trigger a deploy

Push to `main` (or run the `Deploy` workflow manually via
`workflow_dispatch`). The `deploy.yml` job lands on this runner, logs in to
`ghcr.io`, pulls the freshly-built image, and runs
`docker compose up -d --build` — the same stack (`api`, `postgres`, `redis`,
`mailhog`, `nginx`, `prometheus`, `grafana`) comes up on this container as
it would on a real VM.

## 5. Verify

From inside (or exposing ports from) the runner container:

```bash
docker exec urlshortener-vm docker compose ps
```

Grafana/Prometheus/nginx are then reachable the same way as any local
`docker compose up` — see the root `README.md`.

## Token rotation

Registration tokens expire quickly. If the container restarts and fails to
register, generate a new token and re-run step 3 (or re-`docker exec` a
fresh `config.sh` call with the new token).

## Required GitHub Secrets (production deploy)

`deploy.yml` no longer copies `.env.example` to `.env` on the deploy target.
`.env.example` contains publicly-known placeholder values (they're in the
git history), so shipping them to production would mean running with
world-readable passwords, a reversible analytics IP hash, and links pointing
at `localhost`. Instead, the workflow generates `.env` at deploy time from
GitHub Actions Secrets, and the deploy fails loudly if any required secret
is missing.

The backend also enforces this at startup: when `ENV=production`, it refuses
to start (fatal error, non-zero exit) if any of these values are empty or
still equal a known dev placeholder (see `backend/internal/config/config.go`).
This is the safety net if the workflow is ever changed to skip a secret.

Set these in **repo → Settings → Secrets and variables → Actions → New
repository secret**:

| Secret | Purpose | How to generate |
| --- | --- | --- |
| `DATABASE_URL` | Full Postgres connection string used by the API in production (`postgresql://user:pass@host:5432/db?schema=public`). Must not reuse the dev user/password. | Build from your production Postgres credentials; generate the password with `openssl rand -base64 32`. |
| `REDIS_PASSWORD` | Password for the production Redis instance (session/cache store). | `openssl rand -base64 32` |
| `IP_HASH_SECRET` | HMAC secret used to hash visitor IPs before storing them for analytics. A known/placeholder value makes the hash reversible (a privacy problem), so this must be a strong, private secret. | `openssl rand -hex 32` |
| `GF_SECURITY_ADMIN_PASSWORD` | Grafana admin login password. | `openssl rand -base64 24` |
| `SMTP_HOST` | Hostname of the real production SMTP provider (never MailHog in production). | Provided by your email provider (e.g. SES, Postmark, SendGrid). |
| `SMTP_PORT` | Port for the production SMTP provider. | Provided by your email provider. |
| `SMTP_USER` | Username/API key for SMTP auth (if required by your provider). | Provided by your email provider. |
| `SMTP_PASSWORD` | Password/API secret for SMTP auth (if required by your provider). | Provided by your email provider. |
| `MAIL_FROM` | "From" address used for verification/notification email. | Your chosen sender address, e.g. `no-reply@yourdomain.com`. |
| `BASE_URL` | Public base URL of the deployed app, used to build short links. Must not be `localhost`. | Your production domain, e.g. `https://short.yourdomain.com`. |
| `CORS_ALLOWED_ORIGINS` (optional) | Comma-separated list of allowed frontend origins in production. | Your production frontend origin(s). |

None of these values are invented or stored in this repo — they must be
created by a maintainer with real production credentials.
