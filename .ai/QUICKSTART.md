# 🚀 QUICKSTART — Get Running in 3 Commands

> **Purpose:** Minimal setup to get the project running locally.

---

## Option 1: Docker (Recommended)

```bash
# 1. Clone and enter
git clone https://github.com/secretJod/Url-Shortner.git && cd Url-Shortner

# 2. Copy env and start everything
cp .env.example .env && docker-compose up --build

# 3. Create database tables (one-time, in another terminal)
go run github.com/steebchen/prisma-client-go db push
```

Server: `http://localhost:8080`

---

## Option 2: Local Development

```bash
# 1. Clone
git clone https://github.com/secretJod/Url-Shortner.git && cd Url-Shortner

# 2. Start only Postgres + Redis
docker-compose up -d postgres redis

# 3. Generate Prisma client + create tables + run
cp .env.example .env
go run github.com/steebchen/prisma-client-go generate
go run github.com/steebchen/prisma-client-go db push
go run ./cmd/api
```

Server: `http://localhost:8080`

---

## Test It Works

```bash
# Health check
curl http://localhost:8080/health
# → {"redis":"ok","status":"ok"}

# Shorten a URL
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.google.com"}'
# → {"short_code":"x","short_url":"http://localhost:8080/x",...}

# Redirect
curl -L http://localhost:8080/x
# → redirects to google.com

# Check stats
curl http://localhost:8080/api/stats/x
# → {"total_clicks":1,"unique_ips":1,...}
```

---

## Stop Everything

```bash
# Stop containers
docker-compose down

# Stop local server
lsof -ti:8080 | xargs kill -9

# Remove all data (fresh start)
docker-compose down -v
```

---

## Common Issues

| Problem | Fix |
|---------|-----|
| Port 8080 in use | `lsof -ti:8080 \| xargs kill -9` |
| Tables don't exist | `go run github.com/steebchen/prisma-client-go db push` |
| Build fails (undefined types) | `go run github.com/steebchen/prisma-client-go generate` |
| Can't connect to Postgres | Check `.env` has `localhost:5432` (not `postgres:5432`) |
| Docker can't connect | Check `docker-compose.yml` has `environment:` overrides |