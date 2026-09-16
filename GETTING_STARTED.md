# Getting Started

A URL shortener with link analytics, rate limiting, and a full observability
stack — Go/Fiber API, Postgres via Prisma, Redis, and a Vite frontend behind
nginx.

Three ways to run it:

- **Option A** — run everything directly on your host (no Docker). Needs Go,
  Node, Postgres, Redis installed locally.
- **Option B** — run everything in Docker (recommended, easiest). Needs only
  Docker + Docker Compose.
- **Option C** — push to GitHub and let CI build/test the image and push it
  to `ghcr.io`. Needs a GitHub repo.

---

## Option A — Run locally without Docker

### 1. Install prerequisites

Example (macOS/Homebrew); use your platform's package manager otherwise:

```bash
brew install go node postgresql@16 redis
```

You need: Go 1.25, Node 20, PostgreSQL 16, Redis 7.

### 2. Start Postgres and Redis

```bash
brew services start postgresql@16 redis
```

Create the database/user matching `backend/.env.example`:

```bash
psql postgres -c "CREATE USER urlshortener WITH PASSWORD 'urlshortener_dev_pw';"
psql postgres -c "CREATE DATABASE urlshortener OWNER urlshortener;"
```

### 3. Configure and run the backend

```bash
cd backend
cp .env.example .env
```

Edit `.env` so it points at your local services:

```bash
DATABASE_URL="postgresql://urlshortener:urlshortener_dev_pw@localhost:5432/urlshortener?schema=public"
REDIS_ADDR="localhost:6379"
```

(`.env.example` already defaults to these `localhost` values, so usually no
edit is needed — just double check.)

Generate the Prisma client and apply the schema:

```bash
go run github.com/steebchen/prisma-client-go prefetch   # downloads the query-engine binary
go run github.com/steebchen/prisma-client-go generate    # generates Go client code in internal/db
go run github.com/steebchen/prisma-client-go db push     # applies schema.prisma to the database
```

Run the API:

```bash
go run ./cmd/api
```

The API listens on `:8080`.

### 4. Run the frontend

```bash
cd frontend
npm install
npm run dev
```

Vite's dev server runs on `:5173`. The API's default
`CORS_ALLOWED_ORIGINS` already includes `http://localhost:5173`, so no extra
config is needed.

### 5. Email / verification links

There's no MailHog container in this path. Either:

- run a standalone MailHog binary and point `SMTP_HOST=localhost` /
  `SMTP_PORT=1025` at it, or
- point `SMTP_HOST`/`SMTP_PORT` at any local SMTP sink.

Either way, grab the verification/reset link from wherever your sink shows
sent mail — that link is what matters, not the mail itself.

### 6. URLs

| Service  | URL                     |
|----------|-------------------------|
| Frontend | http://localhost:5173   |
| API      | http://localhost:8080   |

---

## Option B — Run locally with Docker (easiest)

Nothing needs to be installed on the host except Docker + Docker Compose —
Go, Node, Postgres, Redis, and Prisma all run in containers.

```bash
cp .env.example .env
cp backend/.env.example backend/.env
docker compose up --build
```

The `migrate` service applies the Prisma schema automatically before the API
starts — no manual `db push` needed.

Services:

| Service         | URL                     |
|------------------|-------------------------|
| App (via nginx)  | http://localhost:80    |
| API directly     | http://localhost:8080  |
| Grafana          | http://localhost:3000  |
| Prometheus       | http://localhost:9090  |
| MailHog UI       | http://localhost:8025  |

Stop everything:

```bash
docker compose down       # keep data volumes
docker compose down -v    # also drop Postgres/Redis data
```

---

## Option C — Ship it through GitHub CI

```bash
git push origin main
```

`.github/workflows/ci.yml` runs `go test`, builds the Docker image, and (on
`main`) pushes it to `ghcr.io` tagged with the commit SHA and `latest`.
There is currently no deployment step beyond that — actually running the
built image somewhere is undecided (see `docs/system-design/06-deployment.md`).
To run it yourself right now, use Option B locally.

---

## Common commands / troubleshooting

- View logs for a service: `docker compose logs -f api`
- Restart one service after a change: `docker compose up -d --build api`
- Re-run the schema migration manually: `docker compose run --rm migrate`
- Env vars live in `.env` (root, Docker-facing) and `backend/.env` (host-facing,
  used by `go run ./cmd/api` directly)
- Grab a verification/reset email link: open the MailHog UI at
  `http://localhost:8025` (Docker path) or your local SMTP sink (non-Docker
  path)
