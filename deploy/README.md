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
