# Deploying with GitHub CI/CD to a self-hosted machine (Windows + WSL)

This sets up an automatic pipeline: **you `git push` → GitHub runs a job on the
target machine → the full stack redeploys there with `docker compose`.**

There are two workflows in `.github/workflows/`:

- **ci.yml** — runs on GitHub's own cloud runners: builds and tests the image,
  and pushes it to GitHub Container Registry (ghcr.io). No setup needed.
- **deploy.yml** — runs on a **self-hosted runner** (the target machine). It
  checks out the repo there and runs `docker compose up -d --build`.

## How it works (plain language)

- **GitHub Actions** = automation that runs when you push.
- A **runner** = the worker that executes a job. GitHub has cloud runners; a
  **self-hosted runner** is *your own* machine. We use one so the deploy happens
  on the target PC.
- The runner is a small program that logs into GitHub and waits for jobs. When
  you push to `main`, GitHub sends the deploy job to it, and it runs the deploy
  commands locally. The machine must be on and the runner running.

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
5. **Start the runner.** Either run `./run.sh` (keeps running in that terminal),
   or install it as a background service so it survives reboots:
   `sudo ./svc.sh install && sudo ./svc.sh start`.
6. Confirm on GitHub (Settings → Actions → Runners) that the runner shows
   **Idle / online**.

## Deploy

- `git push origin main` (or Actions tab → **Deploy** → Run workflow).
- Watch the repo's **Actions** tab: the `deploy` job runs on your runner and
  ends with `docker compose ps` showing the stack up.
- On the machine, the app is at http://localhost (Grafana :3000, MailHog :8025,
  Prometheus :9090).

## Notes

- The machine must be **on** and the **runner running** for a deploy to happen;
  otherwise the job queues until it is.
- `deploy.yml` copies `.env.example` → `.env` for a demo. For real secrets, use
  GitHub repository secrets and write them into the env step instead of the
  placeholder values.
- To reach it beyond that machine, run a free tunnel there:
  `cloudflared tunnel --url http://localhost:80` (or `ngrok http 80`).
