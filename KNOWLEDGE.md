# 📚 URL Shortener — Complete Knowledge File

> **Who is this for?** Someone who has never coded before. Every term is explained in plain English. If you read this from top to bottom, you'll understand what this project is, how it works, and what every single file does.

---

## Table of Contents

1. [What Is This Project?](#1-what-is-this-project)
2. [What Is a URL Shortener?](#2-what-is-a-url-shortener)
3. [The Big Picture (How It All Fits Together)](#3-the-big-picture-how-it-all-fits-together)
4. [Tools & Technologies Used (Explained Simply)](#4-tools--technologies-used-explained-simply)
5. [Project Folder Structure (Every File Explained)](#5-project-folder-structure-every-file-explained)
6. [How a Short URL Gets Created (Step by Step)](#6-how-a-short-url-gets-created-step-by-step)
7. [How a Redirect Works (Step by Step)](#7-how-a-redirect-works-step-by-step)
8. [API Key System (Authentication)](#8-api-key-system-authentication)
9. [Rate Limiting (Preventing Abuse)](#9-rate-limiting-preventing-abuse)
10. [The Database (What Data Is Stored)](#10-the-database-what-data-is-stored)
11. [Redis Cache (The Speed Layer)](#11-redis-cache-the-speed-layer)
12. [Base62 Encoding (How Short Codes Are Made)](#12-base62-encoding-how-short-codes-are-made)
13. [Docker & Deployment](#13-docker--deployment)
14. [Configuration & Environment Variables](#14-configuration--environment-variables)
15. [Testing](#15-testing)
16. [Build Phases & Current Status](#16-build-phases--current-status)
17. [Glossary of Terms](#17-glossary-of-terms)

---

## 1. What Is This Project?

This is a **URL Shortener** — a web service (a program that runs on the internet) that takes a long, ugly web address like:

```
https://www.example.com/some/really/long/path/that/goes/on/forever?param=value&another=thing
```

…and turns it into a short, clean one like:

```
http://localhost:8080/abc123
```

When someone clicks the short link, they get automatically redirected to the original long address.

Think of it like **Bitly** or **TinyURL** — this project does the same thing, but it's built from scratch using professional-grade tools.

---

## 2. What Is a URL Shortener?

Imagine you have a really long phone number: `+1 (555) 123-4567 ext. 8910`. You could save it in your phone as "Mom" — a short, easy-to-remember name that points to the full number.

A URL shortener does the same thing for web addresses:
- You give it a **long URL** (the full web address)
- It gives you back a **short code** (like `abc123`)
- When anyone visits `yoursite.com/abc123`, they get sent to the original long URL

**Why is this useful?**
- Short links fit in tweets, text messages, and QR codes
- They look cleaner in emails and presentations
- The service can track how many people clicked the link
- You can set expiration dates on links

---

## 3. The Big Picture (How It All Fits Together)

Here's a simple diagram of how the pieces connect:

```
                    ┌─────────────────────────────────────┐
                    │         Your Browser / App          │
                    │   (the person using the service)    │
                    └────────────┬────────────────────────┘
                                 │
                          HTTP requests
                    (like "shorten this URL"
                      or "go to this short link")
                                 │
                                 ▼
                    ┌─────────────────────────────────────┐
                    │      Go API Server (Fiber)          │
                    │  This is the "brain" — it receives  │
                    │  requests and decides what to do    │
                    └──────┬──────────────┬────────────────┘
                           │              │
              ┌────────────▼───┐   ┌──────▼──────────────┐
              │    Redis       │   │   PostgreSQL         │
              │  (super fast   │   │  (permanent storage) │
              │   memory cache)│   │  saves all data      │
              └────────────────┘   └──────────────────────┘
```

**In plain English:**
1. A person sends a request to the **Go API Server** (the main program)
2. The server checks **Redis** first (it's like short-term memory — very fast but temporary)
3. If Redis doesn't have the answer, the server checks **PostgreSQL** (it's like a filing cabinet — slower but permanent)
4. The server responds back to the person

---

## 4. Tools & Technologies Used (Explained Simply)

| Tool | What It Is | Why It's Used |
|------|-----------|---------------|
| **Go (Golang)** | A programming language made by Google | It's very fast and good at handling many requests at the same time |
| **Fiber** | A "web framework" for Go | It's like a pre-built house frame — instead of building everything from scratch, Fiber gives you the structure to handle web requests easily |
| **PostgreSQL** | A database (like a spreadsheet that stores data permanently) | This is where all the URL data, user accounts, and API keys are saved permanently |
| **Redis** | An "in-memory data store" (like super-fast temporary memory) | Used for caching (storing frequently-accessed data for quick retrieval), counting, and rate limiting |
| **Prisma** | An "ORM" (Object-Relational Mapper) | It's a translator between Go code and the database — instead of writing raw SQL queries, you write Go code and Prisma translates it |
| **Docker** | A tool that packages your app + all its dependencies into a "container" | Makes sure the app runs the same way on any computer, regardless of what's installed |
| **Docker Compose** | A tool for running multiple Docker containers together | Lets you start PostgreSQL, Redis, and the API all with one command |

---

## 5. Project Folder Structure (Every File Explained)

Here's the full directory tree with explanations:

```
Url-Shortner/
│
├── cmd/                          ← "cmd" stands for "command" — this is where the program starts
│   └── api/
│       └── main.go               ← THE entry point. This is the first file that runs.
│
├── internal/                     ← "internal" means these files are private to this project
│   │
│   ├── auth/                     ← Handles API key security
│   │   ├── apikey.go             ← Creates and hashes API keys
│   │   └── apikey_test.go        ← Tests for the API key functions
│   │
│   ├── config/                   ← Loads settings (like port number, database URL)
│   │   └── config.go             ← Reads environment variables and puts them in a Config struct
│   │
│   ├── db/                       ← Database connection code
│   │   ├── .gitignore            ← Tells Git to ignore generated files
│   │   └── prisma_store.go       ← The actual code that talks to PostgreSQL via Prisma
│   │
│   ├── handlers/                 ← "Handlers" = code that handles incoming web requests
│   │   ├── apikeys.go            ← Handles "create a new API key" requests
│   │   ├── health.go             ← Handles "is the server alive?" requests
│   │   ├── redirect.go           ← Handles "redirect me to the long URL" requests
│   │   ├── shorten.go            ← Handles "shorten this URL" requests
│   │   └── stats.go              ← Handles admin/stats API (Phase 4: click analytics, top links, user links)
│   │
│   ├── middleware/               ← Code that runs BEFORE your handler (like a security checkpoint)
│   │   ├── auth.go               ← Checks if the request has a valid API key
│   │   └── ratelimit.go          ← Checks if the requester has sent too many requests
│   │
│   ├── redis/                    ← All Redis-related code
│   │   ├── cache.go              ← Caches short code → long URL mappings
│   │   ├── client.go             ← Creates and configures the Redis connection
│   │   ├── counter.go            ← A global counter for generating unique IDs
│   │   ├── ratelimit.go          ← The actual rate limiting logic
│   │   ├── ratelimit_test.go     ← Tests for rate limiting
│   │   ├── stream.go             ← Redis Streams for click analytics (Phase 3)
│   │   ├── stream_test.go        ← Tests for Redis Stream operations
│   │   └── util.go               ← Small helper functions
│   │
│   ├── worker/                   ← Background workers (Phase 3)
│   │   └── worker.go             ← Analytics worker: reads click events from Redis Stream, writes to Postgres
│   │
│   ├── shortcode/                ← Converts numbers to short codes (like "abc123")
│   │   ├── base62.go             ← The encoding/decoding logic
│   │   └── base62_test.go        ← Tests for the encoding
│   │
│   └── store/                    ← Defines the "interface" (contract) for storage
│       └── store.go             ← Says WHAT the storage should do, not HOW
│
├── prisma/                       ← Prisma configuration
│   └── schema.prisma             ← Defines the database tables (like a blueprint)
│
├── .env.example                  ← A template for environment variables (copy to .env)
├── .gitignore                    ← Tells Git which files to ignore
├── api                           ← A compiled binary (the finished program). Not source code.
├── docker-compose.yml            ← Defines how to run PostgreSQL + Redis + API together
├── Dockerfile                    ← Recipe for building the API into a Docker container
├── go.mod                        ← Lists all Go dependencies (like a shopping list)
├── go.sum                        ← Checksums for dependencies (ensures they haven't been tampered with)
├── package.json                  ← Empty (not really used in this Go project)
├── PROJECT_OVERVIEW.md           ← The developer's own notes about the project
├── README.md                     ← Quick setup instructions
├── repomix-output.md             ← A merged copy of all source files (for AI analysis)
└── KNOWLEDGE.md                  ← YOU ARE HERE — this file!
```

---

## 6. How a Short URL Gets Created (Step by Step)

When someone sends a request to shorten a URL, here's exactly what happens:

### The Request
A person sends an HTTP POST request to `/api/shorten` with this JSON body:
```json
{
  "url": "https://www.example.com/very/long/address",
  "custom_alias": "mylink",
  "expires_at": "2025-12-31T23:59:59Z"
}
```

### The Flow (inside the code)

1. **Middleware runs first:**
   - `OptionalAPIKeyAuth` (in `internal/middleware/auth.go`) checks if the request has an API key. If it does, it validates it. If not, the request continues as "anonymous."
   - `RateLimit` (in `internal/middleware/ratelimit.go`) checks if the requester has sent too many requests recently. If they have, it returns a "429 Too Many Requests" error.

2. **Handler runs** (`internal/handlers/shorten.go`):
   - Parses the JSON body to get the URL, custom alias, and expiration date.
   - Validates the URL (must be a valid `http://` or `https://` URL).
   - If an expiration date was provided, it parses it.

3. **Short code generation** (`resolveShortCode` method):
   - **If a custom alias was provided:** Check it matches the pattern (3-32 chars, letters/numbers/underscore/hyphen only). Then try to "reserve" it in Redis (so no one else can grab it at the same time). If someone already has it, return a "409 Conflict" error.
   - **If no custom alias:** Ask Redis for the next number in a global counter (`INCR` command). Then convert that number to a short code using base62 encoding.

4. **Save to PostgreSQL** (`internal/db/prisma_store.go`):
   - Create a new row in the `links` table with the short code, long URL, user ID (if authenticated), custom alias flag, and expiration date.

5. **Write to Redis cache** (`internal/redis/cache.go`):
   - Store the `short_code → long_url` mapping in Redis so the very first redirect is already a cache hit.

6. **Return the response:**
   ```json
   {
     "short_url": "http://localhost:8080/abc123",
     "short_code": "abc123",
     "long_url": "https://www.example.com/very/long/address"
   }
   ```

---

## 7. How a Redirect Works (Step by Step)

When someone visits `http://localhost:8080/abc123`:

1. **Handler runs** (`internal/handlers/redirect.go`):
   - Extract the short code (`abc123`) from the URL.

2. **Check Redis cache first** (this is the fast path):
   - Ask Redis: "Do you have a long URL for the short code `abc123`?"
   - If Redis says yes → immediately redirect the person to the long URL with a `302 Found` response. This is super fast (sub-millisecond).

3. **Cache miss — fall back to PostgreSQL:**
   - If Redis doesn't have it, ask PostgreSQL: "Do you have a link with short code `abc123`?"
   - If PostgreSQL doesn't find it → return `404 Not Found`.
   - If the link has expired → also return `404 Not Found`.

4. **Backfill the cache:**
   - Store the result in Redis so the NEXT person who visits this short code gets a cache hit.

5. **Redirect:**
   - Send a `302 Found` response with the long URL in the `Location` header.
   - The browser automatically follows the redirect to the long URL.

---

## 8. API Key System (Authentication)

### What Is an API Key?

An API key is like a password for a program. Instead of a human typing a username and password, a program sends a secret key to prove who it is.

### How API Keys Work in This Project

**Creating a key** (`internal/auth/apikey.go`):

1. Generate 32 random bytes (256 bits of randomness) using Go's `crypto/rand`.
2. Convert those bytes to hexadecimal text.
3. Add a prefix `usk_` (stands for "url-shortener key") so keys are recognizable.
4. The raw key looks like: `usk_a1b2c3d4e5f6...` (64 hex characters after the prefix).
5. Hash the key using SHA-256 (a one-way mathematical function).
6. **Store ONLY the hash** in the database — never the raw key.
7. **Show the raw key to the user exactly once**. If they lose it, they need to create a new one.

**Why hash the key?**
If a hacker steals the database, they only get hashes — not the actual keys. They can't use hashes to impersonate users. This is the same principle as password hashing.

**Using a key:**
A client sends an HTTP header like:
```
Authorization: Bearer usk_a1b2c3d4e5f6...
```

The middleware (`internal/middleware/auth.go`):
1. Extracts the token from the header.
2. Hashes it with SHA-256.
3. Looks up the hash in the database.
4. If found → the request is authenticated. If not → `401 Unauthorized`.

**Important design choice:** Authentication is **optional**. You can shorten URLs without an API key (anonymous). But if you DO provide a key and it's invalid, you get rejected — no silent fallback to anonymous.

---

## 9. Rate Limiting (Preventing Abuse)

### What Is Rate Limiting?

Imagine a store that says "only 5 items per customer per day." Rate limiting does the same thing for web requests — it prevents any single person from overwhelming the server with too many requests.

### How It Works in This Project

The rate limiter uses a **sliding window** algorithm (in `internal/redis/ratelimit.go`):

Think of it like a moving time window. Instead of saying "you get 60 requests between 1:00 PM and 2:00 PM" (fixed window), it says "you get 60 requests in any rolling 60-minute period." This is more fair and prevents the "burst at the boundary" problem (where someone sends 60 requests at 1:59 PM and 60 more at 2:01 PM).

**Technical implementation:**
- Uses a Redis **sorted set** (a data structure that keeps items sorted by a score).
- Each request is added as a member with its timestamp as the score.
- Old entries (outside the window) are removed.
- The remaining count is compared against the limit.

**Rate limit tiers:**

| Who | Limit | Window |
|-----|-------|--------|
| Anonymous (no API key) | 20 requests | per minute |
| Standard tier API key | 60 requests | per minute |
| Pro tier API key | 600 requests | per minute |

**What happens when you exceed the limit?**
- You get a `429 Too Many Requests` response.
- The response includes a `Retry-After` header telling you how long to wait.

**Fail-open design:** If Redis itself is down, the rate limiter lets requests through. The philosophy is: a Redis hiccup shouldn't take down the entire API.

---

## 10. The Database (What Data Is Stored)

The database schema is defined in `prisma/schema.prisma`. There are 4 tables:

### Table: `links` (stores shortened URLs)
| Column | Type | Description |
|--------|------|-------------|
| `id` | BigInt (auto-increment) | Unique ID, also used as the base for the short code |
| `short_code` | String (unique) | The short code like `abc123` |
| `long_url` | String | The original long URL |
| `user_id` | BigInt? (nullable) | Who created this link (null = anonymous) |
| `custom_alias` | Boolean | `true` if the user chose the alias, `false` if auto-generated |
| `expires_at` | DateTime? (nullable) | When the link expires (null = never) |
| `password_hash` | String? (nullable) | For future password-protected links |
| `created_at` | DateTime | When the link was created |

### Table: `users` (stores user accounts)
| Column | Type | Description |
|--------|------|-------------|
| `id` | BigInt (auto-increment) | Unique ID |
| `email` | String (unique) | User's email address |
| `password_hash` | String | Placeholder for future password auth |
| `created_at` | DateTime | When the account was created |

### Table: `api_keys` (stores API keys)
| Column | Type | Description |
|--------|------|-------------|
| `id` | BigInt (auto-increment) | Unique ID |
| `key_hash` | String (unique) | SHA-256 hash of the API key (never the raw key!) |
| `user_id` | BigInt | Which user this key belongs to |
| `rate_limit_tier` | String | "standard" or "pro" — controls rate limits |
| `created_at` | DateTime | When the key was created |

### Table: `click_events` (stores analytics — for future use)
| Column | Type | Description |
|--------|------|-------------|
| `id` | BigInt (auto-increment) | Unique ID |
| `link_id` | BigInt | Which link was clicked |
| `timestamp` | DateTime | When the click happened |
| `referrer` | String? (nullable) | Where the visitor came from |
| `country` | String? (nullable) | Visitor's country |
| `device_type` | String? (nullable) | Phone/tablet/desktop |
| `ip_hash` | String? (nullable) | Hashed IP address (for privacy) |

---

## 11. Redis Cache (The Speed Layer)

Redis is used for four things in this project:

### 1. URL Cache (`internal/redis/cache.go`)
- **What it stores:** `short_code → long_url` mappings
- **Key format:** `urlshortener:link:<short_code>`
- **TTL (Time To Live):** 24 hours by default, or until the link's expiration date (whichever is sooner)
- **Why:** When someone visits a short link, checking Redis is much faster than checking PostgreSQL. Redis stores data in memory (RAM), while PostgreSQL reads from disk.

### 2. ID Counter (`internal/redis/counter.go`)
- **What it stores:** A single global counter that goes up by 1 each time a new link is created
- **Key:** `urlshortener:link_id_counter`
- **How:** Uses Redis `INCR` command, which is **atomic** — meaning even if 1000 requests come in at the exact same moment, each one gets a unique number. No duplicates, ever.
- **Why:** The counter value is converted to a short code using base62 encoding.

### 3. Rate Limiting (`internal/redis/ratelimit.go`)
- **What it stores:** Request timestamps in sorted sets
- **Key format:** `ratelimit:apikey:<id>` or `ratelimit:ip:<ip_address>`
- **How:** Each request is added with its timestamp. Old entries are trimmed. The count is compared to the limit.
- **Why:** Prevents abuse by limiting how many requests a person can make.

### 4. Alias Reservation (`internal/redis/cache.go`)
- **What it stores:** A temporary lock on a custom alias
- **Key format:** `urlshortener:alias_reserved:<alias>`
- **TTL:** 30 seconds (safety net in case the caller crashes)
- **How:** Uses Redis `SETNX` (Set if Not eXists) — atomically claims the alias. If someone else already has it, the claim fails.
- **Why:** Prevents two people from grabbing the same custom alias at the same time.

---

## 12. Base62 Encoding (How Short Codes Are Made)

### What Is Base62?

You're probably familiar with **base 10** (numbers 0-9) and **base 2** (0s and 1s). Base62 uses 62 different characters:

```
0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
```

That's: 10 digits (0-9) + 26 lowercase letters (a-z) + 26 uppercase letters (A-Z) = 62 characters.

### Why Base62?

Because it's **URL-safe** — all 62 characters can appear in a web address without needing special encoding. And it's **compact** — a small number like `999999` becomes just `4c91` in base62.

### How It Works (`internal/shortcode/base62.go`)

**Encoding** (number → short code):
```
ID 0     → "0"
ID 61    → "Z"
ID 62    → "10"
ID 12345 → "3d7"
```

The process is like converting a number to a different base:
1. Divide the number by 62.
2. The remainder gives you the rightmost character.
3. Repeat with the quotient for the remaining characters.
4. Reverse the result (since we built it right-to-left).

**Decoding** (short code → number):
The reverse process — multiply each character's position value by powers of 62.

**Why is this collision-free?**
Because every unique number produces a unique short code. Since the Redis counter gives unique numbers, the short codes are guaranteed unique. No need to check for collisions!

---

## 13. Docker & Deployment

### What Is Docker?

Docker is like a shipping container for software. Instead of worrying about what's installed on the computer, Docker packages your app + everything it needs into a "container" that runs the same way everywhere.

### Dockerfile (`Dockerfile`)

The Dockerfile has two stages (called "multi-stage build"):

**Stage 1: Builder** (uses `golang:1.22-alpine` image)
1. Sets up a working directory `/app`
2. Installs OpenSSL (needed by Prisma)
3. Copies `go.mod` and `go.sum` and downloads Go dependencies
4. Copies all source code
5. **Generates the Prisma client** — this is crucial! The Prisma client must be generated inside the container so the query-engine binary matches the container's platform (Alpine/Linux), not your development machine (e.g., macOS)
6. Compiles the Go program into a binary called `/urlshortener`

**Stage 2: Runner** (uses `alpine:3.19` image)
1. Installs CA certificates and OpenSSL
2. Copies just the compiled binary from the builder stage
3. Exposes port 8080
4. Runs the binary

**Why two stages?** The builder stage is big (has the Go compiler, all source code, etc.). The runner stage is tiny — it only has the compiled binary. This makes the final Docker image small and secure.

### Docker Compose (`docker-compose.yml`)

Docker Compose defines three services that run together:

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| `postgres` | `postgres:16-alpine` | 5432 | The PostgreSQL database |
| `redis` | `redis:7-alpine` | 6379 | The Redis cache |
| `api` | Built from `Dockerfile` | 8080 | The Go API server |

**Key details:**
- PostgreSQL and Redis have **health checks** — the API service waits until they're healthy before starting
- Data is persisted using **volumes** (`pg_data` for PostgreSQL, `redis_data` for Redis) — so your data survives even if the containers are restarted
- The `api` service has **environment overrides** — inside Docker's network, services are reachable by name (e.g., `postgres:5432`), not `localhost:5432`. The `.env` file uses `localhost` for running directly on your machine, but Docker Compose overrides this for container-to-container communication

---

## 14. Configuration & Environment Variables

### `.env.example` (template for `.env`)

| Variable | Default Value | What It Does |
|----------|--------------|--------------|
| `DATABASE_URL` | `postgresql://urlshortener:urlshortener_dev_pw@localhost:5432/urlshortener?schema=public` | Tells Prisma how to connect to PostgreSQL (username, password, host, port, database name) |
| `REDIS_ADDR` | `localhost:6379` | The address of the Redis server |
| `REDIS_PASSWORD` | (empty) | Password for Redis (empty = no password) |
| `REDIS_DB` | `0` | Which Redis database to use (Redis has 16 numbered databases) |
| `PORT` | `8080` | What port the API server listens on |
| `BASE_URL` | `http://localhost:8080` | The base URL for generating short links (e.g., `http://localhost:8080/abc123`) |
| `ENV` | `development` | The environment name (development, production, etc.) |

### How Config Is Loaded (`internal/config/config.go`)

1. `main.go` calls `godotenv.Load()` — this reads the `.env` file and loads the variables into the environment
2. `config.Load()` reads each variable using `os.LookupEnv()` — if a variable isn't set, it uses a default value
3. The config is stored in a `Config` struct (a Go struct is like a container for related values)

---

## 15. Testing

The project has unit tests for the most critical pieces:

### API Key Tests (`internal/auth/apikey_test.go`)
- **TestGenerateAPIKeyIsRandomAndConsistentlyHashed:** Generates two API keys and checks:
  - They're different (randomness works)
  - They have the `usk_` prefix
  - Hashing the raw key gives the same hash that was returned
  - Two different keys produce different hashes
- **TestExtractBearerToken:** Tests parsing the `Authorization: Bearer <key>` header with various inputs (valid, malformed, empty)

### Base62 Tests (`internal/shortcode/base62_test.go`)
- **TestEncodeDecodeRoundTrip:** Encodes a number, decodes the result, and checks it matches the original. Tests with edge cases: 0, 1, 61, 62, 63, 12345, 999999999, and the maximum uint64 value.
- **TestEncodeIsDeterministicAndUnique:** Encodes 10,000 numbers and checks no two produce the same code.
- **TestDecodeInvalidCharacter:** Checks that invalid characters (like `!`) cause an error.

### Rate Limiter Tests (`internal/redis/ratelimit_test.go`)
- **TestAllow_UnderLimit:** Sends 5 requests with a limit of 5 — all should be allowed.
- **TestAllow_OverLimit:** Sends 3 requests with a limit of 3 (all allowed), then a 4th (should be blocked).
- **TestAllow_WindowSlides:** Sends 2 requests with a limit of 2 in a 500ms window, checks the 3rd is blocked, waits for the window to expire, then checks a new request is allowed.

### Redis Stream Tests (`internal/redis/stream_test.go`) — added in Phase 3
- **TestPushAndReadClickEvent:** Pushes a click event to the Redis Stream, reads it back with a consumer group, and verifies the data matches.
- **TestHashIP:** Checks IP hashing is deterministic (same IP → same hash), unique (different IPs → different hashes), and the right length (16 hex chars).
- **TestStreamLen:** Verifies the stream length is 0 when empty and 3 after pushing 3 events.

**Note:** Redis tests require a running Redis instance. If Redis isn't available, the tests are skipped (not failed).

---

## 16. Build Phases & Current Status

The project was built in phases:

| Phase | Status | What Was Done |
|-------|--------|--------------|
| **Phase 0** | ✅ Complete | Project scaffold, Docker setup, Fiber server, Redis client, `/health` endpoint |
| **Phase 1** | ✅ Complete | Core shorten + redirect API (base62 encoding, PostgreSQL via Prisma, Redis cache). Verified working end-to-end. |
| **Phase 2** | ✅ Complete | API key authentication + rate limiting (sliding window). Code complete, builds clean. |
| **Phase 3** | ✅ Complete | Async analytics pipeline (Redis Streams → worker → PostgreSQL). Verified live end-to-end (click events flowing from redirects to Postgres). |
| **Phase 4** | ✅ Complete | Admin/Stats API (link analytics, top links, user links, recent clicks). Verified live end-to-end. |
| **Phase 5** | ⬜ Not started | Load testing, caching tuning, observability |
| **Phase 6** | ⬜ Not started | Deployment hardening (finalized Docker Compose, optional Kubernetes) |

---

## 17. Glossary of Terms

| Term | Simple Explanation |
|------|-------------------|
| **API** | A way for programs to talk to each other over the internet |
| **API Key** | A secret password for a program (not a human) |
| **Base62** | A way to represent numbers using 62 characters (0-9, a-z, A-Z) |
| **Cache** | A fast, temporary storage for frequently-accessed data |
| **Cache Miss** | When the cache doesn't have what you're looking for (you have to check the slower database) |
| **Container** | A self-contained package of software that runs the same way everywhere |
| **Docker** | A tool for creating and running containers |
| **Environment Variable** | A setting stored outside the code, usually in a `.env` file |
| **Framework** | Pre-written code that gives you a structure to build on (like a house frame) |
| **Go (Golang)** | A programming language created by Google, known for speed and simplicity |
| **Handler** | Code that handles a specific type of web request |
| **Hash** | A one-way mathematical function that turns data into a fixed-size string. You can't reverse it. |
| **HTTP** | The protocol (set of rules) for how web browsers and servers talk to each other |
| **JSON** | A format for writing data that both humans and computers can read easily |
| **Middleware** | Code that runs between receiving a request and handling it (like a security checkpoint) |
| **ORM** | Object-Relational Mapper — a tool that translates between code and database queries |
| **PostgreSQL** | A popular, powerful database system that stores data permanently |
| **Prisma** | The specific ORM used in this project |
| **Redis** | A super-fast, in-memory data store (like temporary memory for your app) |
| **Rate Limiting** | Preventing someone from sending too many requests in a short time |
| **Redirect** | Automatically sending a visitor from one URL to another |
| **SHA-256** | A specific hash function that produces a 256-bit (64-character hexadecimal) output |
| **Sliding Window** | A rate limiting approach where the time window moves with each request (more fair than fixed windows) |
| **Struct** | A Go type that groups related values together (like a record or object) |
| **TTL (Time To Live)** | How long a piece of data should be kept before it's automatically deleted |
| **URL** | Uniform Resource Locator — a web address |

---

## Quick Reference: API Endpoints

| Method | Path | Auth Required | What It Does |
|--------|------|---------------|--------------|
| `GET` | `/health` | No | Checks if the server and Redis are alive |
| `POST` | `/api/keys` | No | Creates a new API key (body: `{"email": "you@example.com"}`) |
| `POST` | `/api/shorten` | Optional | Shortens a URL (body: `{"url": "...", "custom_alias": "...", "expires_at": "..."}`) |
| `GET` | `/:shortCode` | No | Redirects to the original long URL |
| `GET` | `/api/stats/:shortCode` | No | Returns click statistics for a link (total clicks, unique IPs, top referrers, daily chart) |
| `GET` | `/api/stats/top?limit=N` | No | Returns the most-clicked links |
| `GET` | `/api/links` | Yes (API key) | Returns all links created by the authenticated user |
| `GET` | `/api/links/:shortCode/clicks?limit=N` | No | Returns recent click events for a link |

---

## Quick Reference: How to Run This Project

### Option 1: Using Docker (easiest)

```bash
# 1. Copy the environment template
cp .env.example .env

# 2. Start everything (PostgreSQL + Redis + API)
docker-compose up --build

# 3. In another terminal, create the database tables (one-time only)
go run github.com/steebchen/prisma-client-go db push
```

### Option 2: Running locally (for development)

```bash
# 1. Copy the environment template
cp .env.example .env

# 2. Start just PostgreSQL and Redis
docker-compose up -d postgres redis

# 3. Generate the Prisma client (one-time, after changing schema)
go run github.com/steebchen/prisma-client-go generate

# 4. Create the database tables
go run github.com/steebchen/prisma-client-go db push

# 5. Build and run
go build ./...
go run ./cmd/api
```

### Testing it out

```bash
# Check if the server is alive
curl http://localhost:8080/health

# Shorten a URL
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.google.com"}'

# Visit the short URL (will redirect)
curl -L http://localhost:8080/<short_code>
```

---

> **Final note:** This project is a learning exercise in building a production-grade web service. It demonstrates real-world patterns like caching, rate limiting, API key authentication, and containerized deployment — all built from scratch with the Go programming language.