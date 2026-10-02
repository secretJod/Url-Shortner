# Deploying with GitHub CI/CD to a self-hosted machine (Windows + WSL)

This sets up an automatic pipeline: **you `git push` → GitHub builds & pushes an
image → a runner on your own machine pulls it and redeploys the stack with
`docker compose`.**

There are two workflows in `.github/workflows/`:

- **ci.yml** — runs on GitHub's own cloud runners: builds and tests the image,
  and pushes it to GitHub Container Registry (`ghcr.io/secretjod/url-shortner`,
  tagged `:<commit-sha>` and `:latest`). No setup needed.
- **deploy.yml** — runs on a **self-hosted runner** (the target machine). It
  writes a real `.env` from GitHub Actions Secrets, sets `ENV=production`,
  pulls the image `ci.yml` just built, and brings the stack up with the
  **production hardening overlay** (`docker-compose.prod.yml`) — never the
  bare `docker-compose.yml`.

## How it works (plain language)

- **GitHub Actions** = automation that runs when you push.
- A **runner** = the worker that executes a job. GitHub has cloud runners; a
  **self-hosted runner** is *your own* machine. We use one so the deploy happens
  on the target PC.
- The runner is a small program that logs into GitHub and waits for jobs. When
  you push to `main`, GitHub sends the deploy job to it, and it runs the deploy
  commands locally. The machine must be on and the runner running.

`deploy.yml` triggers only on `push: branches: [main]` and manual
`workflow_dispatch` — deliberately **not** on `pull_request`. A self-hosted
runner executes fork PR code on your own machine if `pull_request`/
`pull_request_target` is ever added here, so don't add it.

## One-time setup on the target machine (Windows + WSL2)

1. **Install Docker Desktop for Windows** and enable **WSL 2 integration**
   (Docker Desktop → Settings → Resources → WSL Integration → enable your
   distro). Launch Docker Desktop so it's running.
2. **Open your WSL (Ubuntu) terminal.** Confirm Docker works there:
   `docker ps` should run without error.
3. **Clone the repo in WSL:**
   `git clone https://github.com/secretJod/Url-Shortner && cd Url-Shortner`
4. **Register the self-hosted runner** (this is the GitHub connection):
   - In the browser: GitHub → your repo → **Settings → Actions → Runners →
     New self-hosted runner → Linux**.
   - It shows a set of commands (download, `./config.sh --url ... --token ...`,
     `./run.sh`). Run those **inside the WSL terminal**. Use the token shown
     (it expires quickly; generate a fresh one if needed).
   - When it asks for labels, the defaults are fine (`self-hosted`).
5. **Install the runner as a systemd service** so it survives reboots and
   doesn't depend on a terminal staying open:
   ```bash
   sudo ./svc.sh install
   sudo ./svc.sh start
   sudo ./svc.sh status   # confirm it's active
   ```
   (`./run.sh` in the foreground also works for a one-off test, but the
   service install is what makes deploys reliable across restarts.)
6. Confirm on GitHub (Settings → Actions → Runners) that the runner shows
   **Idle / online**.

## Required GitHub Actions Secrets

Create these under **Settings → Secrets and variables → Actions → New
repository secret**. `deploy.yml` writes them into a fresh `.env` on the
runner at deploy time — nothing here is ever committed to the repo.

| Secret | Purpose |
| --- | --- |
| `POSTGRES_PASSWORD` | Real Postgres password (replaces the dev default `urlshortener_dev_pw`) |
| `REDIS_PASSWORD` | Real Redis `requirepass` value |
| `IP_HASH_SECRET` | HMAC secret for hashing click IPs before storage |
| `BASE_URL` | Public URL of the deployment, e.g. `https://links.example.com` — must NOT contain `localhost`/`127.0.0.1` or `config.go`'s `validateForProduction()` refuses to start |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins allowed to call the API (your real frontend origin(s)) |
| `GF_SECURITY_ADMIN_PASSWORD` | Grafana admin password |
| `MAIL_FROM` | From-address used for magic-link emails |

Those seven are required — the deploy job checks each is non-empty before
writing `.env` and fails loudly (rather than silently deploying with blanks)
if any are missing. `docker-compose.prod.yml` additionally enforces several
of them again at compose-render time via `${VAR:?...}`, so a secret deleted
after being set once is still caught.

### Optional: switching mail to a real relay

| Secret | Purpose |
| --- | --- |
| `SMTP_HOST` | Relay host, e.g. `smtp-relay.brevo.com` |
| `SMTP_PORT` | Relay port — `587` for STARTTLS |
| `SMTP_USER` | Relay login |
| `SMTP_PASSWORD` | Relay password / API key (NOT your account password) |

Leave all four unset and the stack uses the in-network MailHog catcher
(`SMTP_HOST` defaults to `mailhog`, `SMTP_PORT` to `1025`, credentials to
empty). Set all four and `internal/mail/mail.go` switches to its STARTTLS +
AUTH PLAIN path automatically — no code change, no rebuild.

They are deliberately NOT in the required list: an empty `SMTP_USER` is a
meaningful value (it selects the unauthenticated path), so a `${VAR:?...}`
guard would reject exactly the configuration we want.

### Reading verification mail in the current mode

MailHog runs in production but publishes **no ports** — its web UI exists
only on the internal Docker network. That, not obscurity, is what protects
it: the UI lists every verification link for every user, so anyone who could
reach it could claim any account.

To read mail, forward the port over SSH deliberately:

```bash
ssh -L 8025:localhost:8025 <user>@<host>   # then open http://localhost:8025
```

Never proxy 8025 through nginx and never expose it via the tunnel. Magic
links are valid for 1 hour (`internal/handlers/apikeys.go`), which is the
window you have to relay the code to the user by hand.

`ENV=production` is written by the workflow itself, not stored as a secret —
this is what activates `config.go`'s placeholder-secret guard, so make sure
none of the secrets above are ever left as the `.env.example` demo values.

## Deploy

- `git push origin main` (or Actions tab → **Deploy** → Run workflow).
- Watch the repo's **Actions** tab. The `deploy` job:
  1. writes `.env` from the secrets above,
  2. runs `docker compose -f docker-compose.yml -f docker-compose.prod.yml pull`
     to fetch the image `ci.yml` built for that commit,
  3. runs `... up -d --remove-orphans` **with the prod overlay** — this is
     mandatory: it strips host port publishing from postgres/redis/
     prometheus/grafana/api (only nginx stays published), swaps in the
     ghcr.io image instead of building locally, removes MailHog, and skips
     the unattended `prisma db push` against production data,
  4. polls `docker compose ps` until no container is still `starting` and
     none report `unhealthy`,
  5. curls `http://localhost/health` (through nginx, the only published
     entrypoint) with retries, and **fails the job** if the app never
     answers.
- On the machine, the app is reachable at `http://localhost` (proxied by
  nginx). Grafana, Prometheus, Postgres, and Redis are **not** published to
  the host in production — reach them with `docker compose exec` or
  `docker compose port` from the machine itself if needed.

### Manually verifying a deploy

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
docker compose -f docker-compose.yml -f docker-compose.prod.yml config   # renders the merged config; sanity-check secrets resolved and no ports leaked
curl -i http://localhost/health
```

## Notes

- The machine must be **on** and the **runner running** for a deploy to happen;
  otherwise the job queues until it is.
- Schema changes are **not** applied automatically in production (the
  `migrate` service is replaced with a no-op by the prod overlay — see the
  comments in `docker-compose.prod.yml`). Apply `prisma db push` or a real
  migration deliberately and separately, reviewed, against production data.
- To reach the deployment from outside that machine without opening router
  ports or getting a static IP, use a **Cloudflare Tunnel**
  (`cloudflared tunnel --url http://localhost:80`, or a named tunnel via
  `cloudflared tunnel run`): it's outbound-only from the runner machine (so it
  works behind CGNAT/most home routers), free, and terminates TLS at
  Cloudflare's edge — so nginx itself can stay HTTP-only as configured. Avoid
  the generic `ngrok http 80` approach for anything beyond a throwaway demo:
  free ngrok URLs rotate on every restart and are rate-limited.
