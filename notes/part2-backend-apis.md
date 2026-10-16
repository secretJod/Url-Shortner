
# Part 2 — Backend & APIs

## 8. Every Real HTTP API, One by One

Everything the server exposes is registered in one place:
`backend/cmd/api/main.go`, lines 72–144. There's no separate router file —
Fiber routes are wired directly in `main()`. Route order matters a lot here
(see the comment at line 136), because `/:shortCode` (line 134) is a
catch-all: anything registered after it that isn't the final 404 handler
would silently become a short-code lookup instead. That's why every
`/api/*` route, every SPA route, and the static file server all sit above
it, and a bounded `app.Use(...)` 404 handler sits below it (lines 142–144)
to keep the Prometheus `path` label from exploding on scanner traffic.

| Method | Path | Middleware chain (in order) | Handler | Auth |
|---|---|---|---|---|
| GET | `/health` | — | `health.Check` | none |
| GET | `/metrics` | — | `promhttp.Handler()` (adapted) | none |
| GET | `/api/metrics` | — | inline closure → `appMetrics.Snapshot()` | none |
| POST | `/api/keys` | `rateLimitMW` | `apiKeys.CreateKey` | none (this *creates* the identity) |
| GET | `/api/keys/verify` | `rateLimitMW` | `apiKeys.VerifyKey` | token in query string |
| POST | `/api/shorten` | `authMW` (optional) → `rateLimitMW` | `shorten.Shorten` | optional Bearer key |
| GET | `/api/stats/top` | `rateLimitMW` | `stats.GetTopLinks` | none (public, aggregate-only) |
| GET | `/api/stats/:shortCode` | `requireAuthMW` → `rateLimitMW` | `stats.GetLinkStats` | required + ownership |
| GET | `/api/links` | `authMW` (optional) | `stats.GetUserLinks` | required in practice (401 inside handler if missing) |
| GET | `/api/links/:shortCode/clicks` | `requireAuthMW` → `rateLimitMW` | `stats.GetRecentClicks` | required + ownership |
| DELETE | `/api/links/:shortCode` | `requireAuthMW` → `rateLimitMW` | `stats.DeleteLink` | required + ownership |
| GET | `/`, `/login`, `/dashboard`, `/stats/*`, `/top`, `/admin` | — | `serveIndex` (SPA) | none |
| GET | `/:shortCode` | `rateLimitMW` | `redirect.Redirect` | none (catch-all) |
| any | `/*` (after everything above) | — | inline 404 closure | — |

Notice the asymmetry in middleware ordering: `/api/shorten` runs
`authMW` *before* `rateLimitMW` (main.go:98), and so does every stats route.
That ordering isn't cosmetic — `middleware.RateLimit` (ratelimit.go:40)
reads `GetAPIKey(c)` from `c.Locals`, which only exists if an auth
middleware ran first and populated it. Register them the other way around
and every authenticated request would silently fall back to the much
stricter anonymous IP bucket. This is called out explicitly in the
project's own internal/middleware doc comment (middleware/auth.go:10–15).

### Deeper look: `POST /api/shorten`

Request: `{"url": "...", "custom_alias": "optional", "expires_at": "optional RFC3339"}`
(shorten.go:30–34). Response on success is `201 Created` with
`{"short_url", "short_code", "long_url"}` (shorten.go:36–40, 96–100).

The handler (shorten.go:42–101) does, in order: parse body (400 on bad
JSON) → validate the URL is a real `http`/`https` URL via
`url.ParseRequestURI` (shorten.go:127–136, rejecting things like `javascript:`
or bare strings — this is the open-redirect/dangerous-protocol defense) →
parse `expires_at` if present (400 on bad format) → resolve a short code,
either reserving the custom alias or minting one from the Redis counter
(shorten.go:105–125, see chapter 13) → attach `UserID` from the
authenticated API key if present (shorten.go:82–84, so anonymous shortens
stay ownerless) → `CreateLink` in Postgres (shorten.go:86) → **write-through
cache** into Redis immediately (shorten.go:94) so the very first redirect
is already a cache hit instead of a guaranteed miss. If the Postgres insert
fails after a custom alias was reserved, the alias reservation is released
(shorten.go:87–90) so it doesn't get stuck locked for 30 seconds for
nothing.

★ **interview tip** — "Why release the alias reservation on a failed
insert instead of just letting the 30-second TTL expire it?" Because a
client retrying immediately (a very normal thing for an HTTP client to do
on a 500) would get a spurious 409 "already taken" for up to 30 seconds,
on an alias nobody actually holds. Releasing it makes the retry path
correct, not just eventually-correct.

### Deeper look: `GET /:shortCode` (the redirect)

No request body. Two outcomes: `302 Found` with a `Location` header on
success, or JSON `404`/`500` on failure (redirect.go:38, 51, 54, 64). This
is covered in full line-by-line detail in chapter 9's second trace — the
short version is cache-first (redirect.go:30), Postgres on miss with a
cache backfill (redirect.go:47–58), and a fire-and-forget click event
pushed onto a Redis Stream either way (redirect.go:35, 61).

### Deeper look: the magic-link pair — `POST /api/keys` + `GET /api/keys/verify`

This project has no passwords. Identity is: email → magic link → API key.
`POST /api/keys` (apikeys.go:63–105) takes `{"email": "..."}`, validates it
with `net/mail.ParseAddress` (apikeys.go:69), and calls
`GetOrCreateUserByEmail` (apikeys.go:75), which is itself implemented as a
Prisma `Upsert` rather than find-then-create specifically to avoid a
TOCTOU race on the `email` unique constraint when two requests for the
same brand-new address land concurrently (prisma_store.go:144–149, 150–160).
It then generates a random 32-byte token, stores only its SHA-256 hash
(apikeys.go:80–92, same "never persist the raw secret" pattern used for
API keys), and emails a link containing the *raw* token
(apikeys.go:94–95). The response is `202 Accepted` with no key in it —
the key doesn't exist yet.

`GET /api/keys/verify?token=...` (apikeys.go:110–159) is what actually
issues the key: it hashes the incoming token, looks up the
`VerificationToken` row by that hash, checks expiry (1 hour — see the long
comment at apikeys.go:19–32 explaining *why* 1 hour instead of the more
obviously-secure 15 minutes: this deployment relays mail through MailHog
manually, so a human sits in the loop between "email sent" and "link
clicked"), deletes the token immediately so it can't be replayed
(apikeys.go:133, *before* even marking the user verified — consume-first
ordering), marks the user verified, and only then calls
`auth.GenerateAPIKey()` (apikeys.go:141) which returns the raw key once
and its hash for storage. The raw key is returned in the `201` response
body and never persisted anywhere (apikeys.go:155–158) — if the user loses
it, there is no recovery path, only "request a new verification email and
get a new key."

★ **interview tip** — "What stops someone from hitting `/api/keys/verify`
twice with the same token and getting two API keys?" The token row is
deleted (apikeys.go:133) before the key is issued, and the lookup
(apikeys.go:119) returns `store.ErrNotFound` for a deleted token — so the
second request gets a clean 400 "invalid or already-used token," not a
second key.

---

## 9. The Request Lifecycle, End to End

### Trace 1: `POST /api/shorten`

1. **nginx** receives the request on port 80 and proxies it to `api:8080`
   over the Docker network (service name, not `localhost` — see §4/CLAUDE.md
   and the service map in the codebase guide).
2. Fiber's middleware chain runs in registration order (main.go:54–70):
   - `recover.New()` — wraps everything downstream in a panic recoverer so
     a bug in a handler returns a 500 instead of killing the process.
   - `logger.New()` — structured access logging.
   - `cors.New(...)` — validates `Origin` against `CORS_ALLOWED_ORIGINS`
     (env-driven, main.go:59–63) and sets the CORS headers.
   - `middleware.MetricsMiddleware(appMetrics)` — legacy atomic-counter
     metrics (main.go:67).
   - `middleware.PrometheusMetricsMiddleware()` — records the Prometheus
     HTTP histograms/counters with the route's path label (main.go:70).
3. The route-specific chain then runs: `authMW` (`OptionalAPIKeyAuth`,
   auth.go:22–48) reads `Authorization`, and if present, extracts the
   bearer token, hashes it, and looks it up via `GetAPIKeyByHash`
   (one Postgres query) — on success it stashes the `*store.ApiKey` in
   `c.Locals(LocalsAPIKey)`. No header at all just calls `c.Next()` as
   anonymous. A header that's *present but invalid* is rejected outright
   with 401 (auth.go:30–34, 39–43) — deliberately not silently downgraded
   to anonymous.
4. `rateLimitMW` (`RateLimit`, ratelimit.go) reads the API key that step 3
   may have put in locals, picks a bucket+limit (chapter 11), and calls
   `rdb.Allow(...)` — one Redis round trip (a `TxPipeline` with 4 commands).
   429 with `Retry-After` on rejection, otherwise `c.Next()`.
5. `shorten.Shorten` (shorten.go:42) runs: parses and validates the body,
   then **one of two possible Redis calls** — either `ReserveAlias`
   (`SETNX`) for a custom alias, or `NextID` (`INCR`) for an auto code —
   then **one Postgres write** (`CreateLink`, an `INSERT`), then **one more
   Redis write** (`SetLongURL`, the write-through cache). So the order on
   the wire is: Redis (alias reserve or counter incr) → Postgres (insert)
   → Redis (cache write). If the Postgres insert fails and a custom alias
   was reserved, there's a cleanup Redis call (`ReleaseAlias`) after it.
6. The handler returns the `shortenResponse` JSON directly — Fiber's
   `c.JSON(...)` writes to the underlying `fasthttp` response and returns;
   there's no separate serialization step the way an Express callback would
   need.
7. nginx passes the response straight back to the client unmodified.

### Trace 2: `GET /:shortCode` — cache hit

1. nginx → `api:8080`, same middleware stack as above minus the auth
   middleware (this route doesn't register one — redirects are always
   anonymous as far as auth goes) — just `rateLimitMW` (main.go:134).
2. `redirect.Redirect` (redirect.go:25) calls `h.Redis.GetLongURL(ctx,
   shortCode)` (cache.go:47) — one Redis `GET`. It comes back with a
   JSON-decoded `{url, id}` (cache.go:56–60).
3. `fireClickEvent` (redirect.go:71–80) builds a `ClickEvent` struct and
   calls `PushClickEvent`, which is one Redis `XADD` (stream.go:53–61).
   This happens *before* the redirect response is sent, but errors from it
   are explicitly discarded (redirect.go:79) — a Redis Stream outage never
   fails a redirect.
4. `c.Redirect(longURL, fiber.StatusFound)` (redirect.go:38) — `302`, done.
   **Zero Postgres round trips on this path.**

### Trace 2b: `GET /:shortCode` — cache miss

1–2. Same as above, but `GetLongURL` returns `redis.ErrCacheMiss`
   (cache.go:49–50, 14).
3. Falls through to `h.Store.GetLinkByShortCode(ctx, shortCode)`
   (redirect.go:47) — one Postgres `SELECT ... WHERE short_code = $1`
   (prisma_store.go:76–79, backed by the unique index on `short_code`).
   If not found (or expired — prisma_store.go:106–109 treats an expired
   link as not-found), returns `404` immediately with no cache write.
4. On success: `SetLongURL` backfills the cache (redirect.go:58) — one
   Redis `SET` with a TTL — *before* firing the click event, so step 5
   can't somehow race ahead of the cache being warm.
5. `fireClickEvent` — one Redis `XADD`, same as the hit path.
6. `302` redirect.

So a cache miss costs exactly one Postgres read and two Redis writes (cache
backfill + stream push) instead of the hit path's one Redis read and one
Redis write.

---

## 10. How "Callbacks" Get a Response, and How the Response Gets Processed

This is really two different programming models stitched together by one
HTTP response, and conflating them is a common source of confusion.

**Backend (Go/Fiber) — not a callback model at all.** A Fiber handler is a
plain function, `func(c *fiber.Ctx) error` (every handler in this codebase
has that exact signature — e.g. `func (h *ShortenHandler) Shorten(c
*fiber.Ctx) error` at shorten.go:42). There's no "send a response, then
later call back" — the function *is* the entire request lifecycle. Inside
it, `c.JSON(...)`, `c.Redirect(...)`, `c.Status(...).JSON(...)` etc. write
bytes directly onto the underlying `fasthttp.RequestCtx`'s response buffer,
synchronously, in whatever order the code calls them. The function then
`return`s (usually `return c.JSON(...)`, returning that write call's own
error value so Fiber's error-handling middleware can see if the write
itself failed). Once the handler returns, Fiber flushes the buffered
response to the socket. There is no event loop handing control back to a
callback later — the goroutine that's running this handler *is* the thing
serving the socket, for the duration of the request.

**Frontend (React/axios) — this is where the actual async callback model
lives.** `frontend/src/api/client.js` wraps `axios.create(...)` with two
interceptors (client.js:10–46): a request interceptor that reads the API
key out of `localStorage` and attaches `Authorization: Bearer <key>`
(client.js:12–14), and a response interceptor that normalizes errors — a
`401` clears the stored key and redirects to `/login` (client.js:27–32), a
`429` gets rewritten to a friendly "too many requests" message
(client.js:36–37), and anything else gets shaped into `{status, message,
data}` and rejected (client.js:42). Every call site in the React app calls
`apiClient.post(...)` / `apiClient.get(...)`, which returns a `Promise`.
Whatever component made the call either `await`s it inside an `async`
function or chains `.then()/.catch()` — and it's *that* resolution, not
anything on the Go side, that actually updates React state and re-renders
the UI.

**Bridging the two halves:** by the time axios's promise resolves on the
frontend, the Go handler that produced the response has already returned
and its goroutine is gone — the "connection" between them is purely the
HTTP response bytes that traveled over the wire (through nginx) in between.
The backend's synchronous write-and-return model and the frontend's
asynchronous promise-resolution model never share any runtime state; the
HTTP response is the entire contract.

★ **interview tip** — "If a Fiber handler panics halfway through writing a
partial response, what does the client see?" `recover.New()` (main.go:54)
catches the panic before it reaches the OS-level connection handler and
converts it into a `500`, but only if the panic happens *before* Fiber has
already started flushing bytes — once the status line and headers are
written, you can't retroactively rewrite the status code. In practice,
because every handler here builds the full response fiber.Map before
calling `c.JSON`/`c.Redirect` once, there's no "half-written" response
case in this codebase.

---

## 11. Optimization #1 — Rate Limiting

**The problem:** with no limiter, one script hammering `/api/shorten` or
scraping sequential short codes off `/:shortCode` can monopolize the
process, degrade latency for everyone else, and (for the redirect path) do
cheap enumeration — so a limiter sits in front of effectively every route
that touches a store (`rateLimitMW` on `/api/keys`, `/api/keys/verify`,
`/api/shorten`, every `/api/stats/*` and `/api/links*` route, and
`/:shortCode` — see the table in chapter 8).

**The exact algorithm** — sliding-window log, in
`backend/internal/redis/ratelimit.go:19–55`:

```
ZREMRANGEBYSCORE key 0 <now - window>   // drop entries older than the window
ZCARD key                                // count what's left
ZADD key <now_ns> <member>               // record this request optimistically
EXPIRE key <window>                      // so idle keys don't live forever
```

All four run in a single `TxPipeline` (ratelimit.go:24–30) — one Redis
round trip, not four. The count returned by `ZCARD` is read *before* the
`ZADD` landed logically (Redis pipelines execute commands in order but the
client reads `countCmd.Val()` after `Exec`, so it reflects the count prior
to this request's own entry — ratelimit.go:34). If that count is already
`>= limit`, the request is over the limit; the member that was
optimistically added gets removed again with `ZREM` (ratelimit.go:38) so a
rejected request doesn't itself count against the window. `retryAfter` is
then estimated from the oldest surviving entry's score (ratelimit.go:41–50).

Each sorted-set member is `"<unix_nanos>-<4 random hex bytes>"`
(ratelimit.go:22, util.go:17–21) — the random suffix exists purely to
avoid two requests landing on the exact same nanosecond and colliding as
the *same* sorted-set member (which would silently undercount, since a set
member is unique).

**Why sliding-window-log beats fixed-window:** a fixed window (e.g. "60
requests per calendar minute") lets a client burst 60 requests at 0:59 and
another 60 at 1:01 — 120 requests in 2 seconds, still "compliant." The
sliding log has no window boundary to burst across: it always asks "how
many requests happened in the last 60 seconds, as of right now," so a
client literally cannot get more than `limit` requests into *any*
60-second span. The cost is `O(log N)` per request for the sorted-set ops
instead of `O(1)` for a fixed counter — a reasonable trade at the
request volumes a per-client limiter itself has to handle (tens to
hundreds of requests per minute, not millions).

**The three tiers** (middleware/ratelimit.go:16–27):

| Tier | Limit | Window | Who |
|---|---|---|---|
| anonymous | 20 | 1 minute | per-IP, no API key |
| `standard` | 60 | 1 minute | default tier for new keys |
| `pro` | 600 | 1 minute | elevated tier |

The bucket key is `ratelimit:apikey:<id>` when authenticated or
`ratelimit:ip:<ip>` when anonymous (middleware/ratelimit.go:45, 48) — so an
authenticated user's limit follows their key across IPs/devices, while an
anonymous caller is bucketed purely by IP.

**What happens when Redis is down:** `rdb.Allow(...)` returns an error,
and the middleware's response to that is `return c.Next()`
(middleware/ratelimit.go:53–57) — it **fails open**. The comment right
there calls this out explicitly: "a Redis hiccup shouldn't take down the
whole API." This is the right call for this system specifically because
the alternative (fail closed, reject everything) would turn a transient
cache outage into a full API outage for every legitimate user, in exchange
for blocking abuse that Redis being down already makes harder to sustain
at scale anyway (an attacker hammering a degraded backend has bigger
problems than the rate limiter). The trade-off being accepted is: during a
Redis outage, abuse protection is temporarily off.

★ **interview tip** — "Why remove the optimistically-added member on
rejection instead of just not adding it until the count check passes?"
Because the count check and the add happen in the *same* pipelined
round-trip for performance — splitting it into "count, then decide, then
maybe add" would mean two round trips and a race window between them where
two concurrent requests could both see a count just under the limit and
both get admitted, overshooting it.

---

## 12. Optimization #2 — Caching the Redirect Path

**The problem:** a redirect is the single highest-volume endpoint in a URL
shortener by a wide margin — every click on every link ever created hits
it. If every one of those went straight to Postgres, the database becomes
the bottleneck the moment traffic is more than "a few requests a second,"
and a transient DB slowdown would directly translate into slow redirects
for every user clicking any link at that moment.

**Cache-aside, implemented in `backend/internal/redis/cache.go`:** the
redirect handler checks Redis first (`GetLongURL`, cache.go:47); on a miss
it reads Postgres and then writes the result back into Redis
(`SetLongURL`, cache.go:70) so the *next* request for that code is a hit.
Redis is never the system of record — Postgres is — so Redis can be
flushed entirely and the system degrades to "always a cache miss," not
"data loss."

**Exact TTL logic** (cache.go:70–85): default TTL is `defaultCacheTTL = 24
* time.Hour` (cache.go:29). If the link has an explicit `expiresAt`, the
cache entry's TTL is capped to `min(time.Until(expiresAt), 24h)`
(cache.go:72–75) — so a link expiring in 10 minutes gets cached for 10
minutes, not 24 hours (which would let a redirect serve a link past its
intended expiry), while a link with no expiry or a far-future one still
gets periodically refreshed from Postgres rather than living in cache
forever.

**Why the cached value is `{url, id}`, not just the bare URL string**
(cache.go:31–38): the redirect handler needs the link's database ID to
attach to the click event it pushes onto the analytics stream
(redirect.go:34–36). Without the ID cached alongside the URL, a cache *hit*
would still need a Postgres lookup just to get the ID for analytics —
which would defeat half the point of caching. The struct is JSON-encoded
(`cachedLink`, cache.go:35–38) and there's explicit backward-compatibility
handling for the pre-this-feature plain-string cache format: if JSON
decoding fails or comes back with an empty URL, the raw string is treated
as a bare URL with `linkID=0` (cache.go:56–63), and the redirect handler
skips firing a click event when `linkID` is `0` (redirect.go:33–34) rather
than writing a garbage/zero `link_id`.

**Invalidation on delete:** `DeleteLink` (stats.go:191–223) deletes the row
from Postgres first, then calls `h.Redis.InvalidateLongURL(...)`
(stats.go:218–219), which is a plain Redis `DEL` on the cache key
(cache.go:88–90). The order matters — deleting the DB row first and then
invalidating the cache means a redirect that lands in the gap between
those two steps would still see the (not-yet-invalidated) cached value and
serve a stale redirect for the link's remaining TTL window, which is the
known trade-off of cache-aside invalidation versus a stricter
write-through-with-lock scheme; this codebase accepts that window since a
deleted link still briefly redirecting is low severity compared to adding
write coordination to the hot path.

---

## 13. Optimization #3 — Collision-Free ID Generation

**The problem:** two approaches that look reasonable both have a real
failure mode. Auto-incrementing in Postgres means every single shorten
request needs a synchronous DB round trip just to get an ID, on top of the
insert itself. And if two concurrent requests pick a custom alias
independently (e.g. both check "is `/promo` taken?" then both insert), one
of them wins and one gets a unique-constraint violation at best, or a
silently overwritten link at worst, depending on how the insert is
written.

**Redis `INCR` + base62, collision-free by construction, not
probabilistically** (counter.go:7–20, base62.go): `NextID` calls Redis
`INCR` on a single global counter key — `INCR` is atomic in Redis, so two
concurrent callers are *guaranteed* distinct integers, never the same one.
There's no hashing, no randomness, no birthday-paradox collision
probability to reason about the way there would be with e.g. a random
6-character code — every counter value maps to exactly one base62 string
and vice versa (`Encode`/`Decode`, base62.go:21–51), so uniqueness is a
property of the counter being atomic, not a property of the encoding being
"unlikely to repeat."

**`SETNX`-based custom alias reservation and the exact race it prevents**
(cache.go:92–107, used from shorten.go:105–118): a custom alias isn't just
checked against Postgres and then inserted — that would leave a window
where two requests for the same alias both see "not taken" and both
proceed to insert, with the database's unique constraint on `short_code`
(schema.prisma:13) being the only thing stopping a duplicate, and even
then only one request finds out it lost *after* doing the Postgres work.
Instead, `ReserveAlias` does a Redis `SETNX` (`SET ... NX`) on
`urlshortener:alias_reserved:<alias>` with a 30-second TTL
(cache.go:96–102) — `SETNX` is atomic, so of two concurrent callers
exactly one gets `true` back. The loser gets `store.ErrAliasTaken`
immediately (shorten.go:114–116) — a fast `409`, no wasted Postgres work.
The winner proceeds to the Postgres insert; if that insert fails for any
other reason, `ReleaseAlias` (cache.go:105–107) frees the reservation
immediately rather than waiting out the 30-second TTL (shorten.go:87–90).

★ **interview tip** — "Why 30 seconds for the reservation TTL, and why is
it a safety net rather than the actual authority on whether an alias is
taken?" Postgres's `UNIQUE` constraint on `short_code` is the real
authority — the Redis reservation only needs to survive long enough to
cover "SETNX, then one Postgres insert." Thirty seconds is comfortably
longer than that round trip, so the TTL only matters if the process
crashes between the two steps, in which case the alias becomes reservable
again instead of being permanently stuck.

---

## 14. Optimization #4 — Decoupling Analytics from the Hot Path

**The problem:** a redirect is latency-sensitive in a way that writing an
analytics row is not — the user is sitting there waiting for their browser
to follow the `Location` header. If the redirect handler did a synchronous
`INSERT INTO click_events` on every single hit, every redirect's latency
would be bounded by Postgres write latency, and any Postgres slowdown
(lock contention, a slow query from somewhere else, a failover) would
directly slow down the one thing users are actually waiting on.

**Redis Streams + consumer group**, split across the producer
(`PushClickEvent`, stream.go:43–61, called fire-and-forget from both the
cache-hit and cache-miss branches of the redirect handler,
redirect.go:35/61) and the consumer (`internal/worker/worker.go`):

- **Producer side:** `XADD urlshortener:click_events MAXLEN~100000 data=<json>`
  (stream.go:53–60). `MAXLEN ~ 100000` caps the stream so a long worker
  outage doesn't let it grow unbounded — it's described in the code as a
  safety net, not a normal operating ceiling (stream.go:49–51).
- **Consumer side:** the worker goroutine (worker.go:51–130, started
  in-process by `main.go:46–47 — there is no separate worker container)
  loops calling `ReadClickEvents`, which does `XGROUPCREATE ... MKSTREAM`
  (idempotent, creates the group/stream on first use, stream.go:87) then
  `XREADGROUP GROUP analytics-workers ... COUNT 100 BLOCK 5s`
  (stream.go:89–95). For each event it writes a row to Postgres
  (`CreateClickEvent`, worker.go:108), and only events that wrote
  successfully get collected into `ackIDs` and acknowledged with `XACK`
  (worker.go:119, 124–128, stream.go:119–124). A failed write is simply
  *not* acked, which leaves it in the consumer group's pending-entries
  list for redelivery rather than being dropped (worker.go:109–116) — this
  is explicitly how **at-least-once delivery** is achieved: nothing is
  acked until it's durably in Postgres.

**Crash recovery:** because unacked entries stay in the pending list tied
to the consumer group (not to the specific worker process), if the worker
process dies mid-batch, those events are still retrievable — a
restarted worker (or a second one, since the group model supports multiple
consumers) can claim them again. The worker's own `run()` loop additionally
recovers from in-process panics (`defer recover()`, worker.go:52–59) and
restarts itself rather than taking the whole process down with it — note
the comment there: an unrecovered panic in *any* goroutine, including this
one, would kill the entire Go process regardless of Fiber's own
`recover()` middleware, since that only guards the HTTP-handling
goroutines.

**Why this makes analytics specifically "eventually consistent," and what
that means in plain terms:** a click is immediately durable in Redis (the
stream) the instant it happens, but it isn't durable in Postgres — and
therefore not visible to `/api/stats/:shortCode` or `/api/stats/top` —
until the worker has drained it, acknowledged it, and committed the write,
which happens on a 5-second poll cycle in batches of up to 100
(worker.go:32–34, 69–70). In plain terms: if you click a link and
immediately check its stats, the click count might not include your click
yet; check again a few seconds later and it will. The redirect itself,
though, is never delayed by any of this — that's the entire point of the
split.

---

## 15. Database Schema

`backend/prisma/schema.prisma` defines four tables, all accessed through
`backend/internal/db/prisma_store.go` (the only implementation of the
interfaces in `internal/store/store.go`).

### `links` (model `Link`, schema.prisma:11–25)

| Field | Type | Notes |
|---|---|---|
| `id` | BigInt, PK, autoincrement | |
| `short_code` | String, **`@unique`** | the redirect lookup key |
| `long_url` | String | |
| `user_id` | BigInt?, FK → `users.id` | nullable — anonymous shortens have no owner |
| `custom_alias` | Boolean, default false | |
| `expires_at` | DateTime? | checked in `GetLinkByShortCode` (prisma_store.go:106–109) |
| `password_hash` | String? | present in the schema/struct but no handler reads or sets it as a redirect gate — link-level password protection is modeled, not wired up |
| `created_at` | DateTime, default now() | |

Indexes: `@@index([userId])` (schema.prisma:23) plus the implicit unique
index backing `@unique` on `short_code`. The `userId` index exists because
`GetUserLinks` (prisma_store.go:463–468) does
`Link.UserID.Equals(...).OrderBy(Link.CreatedAt.Order(SortOrderDesc))` —
without it, every "show me my links" call would be a full table scan. The
`short_code` unique index is what makes `GetLinkByShortCode`
(prisma_store.go:76–79), the redirect hot path's cache-miss fallback, an
index lookup rather than a scan — this is the single most latency-critical
query in the schema.

### `users` (model `User`, schema.prisma:27–38)

`email` is `@unique` (schema.prisma:29) — this backs `GetOrCreateUserByEmail`'s
`Upsert` (prisma_store.go:150–160), which relies on that constraint to make
concurrent first-time signups race-safe rather than raising a duplicate-row
bug. `password_hash` is stored but always set to `""` (prisma_store.go:156)
— there's no password login in this system; the field is a schema leftover
from before the magic-link model, noted directly in the code comment
(prisma_store.go:154–155).

### `verification_tokens` (model `VerificationToken`, schema.prisma:40–50)

`token_hash` is `@unique` (schema.prisma:42) — `GetVerificationTokenByHash`
(prisma_store.go:199–202) looks up by it directly, and uniqueness also
means two independently-generated tokens can never collide into the same
lookup key. `@@index([userId])` (schema.prisma:48) isn't exercised by any
query in `prisma_store.go` today (there's no "list a user's pending
tokens" query) — it's defined for the FK relation but not yet used by a
hot query; flag this as schema surface ahead of actual usage rather than a
load-bearing index.

### `api_keys` (model `ApiKey`, schema.prisma:52–62)

`key_hash` is `@unique` (schema.prisma:54), backing `GetAPIKeyByHash`
(prisma_store.go:262–265) — this runs on **every single authenticated
request** via `OptionalAPIKeyAuth`/`RequireAPIKeyAuth`
(middleware/auth.go:37, 77), so it needs to be an index lookup, not a
scan. `rate_limit_tier` defaults to `"standard"` (schema.prisma:57,
mirrored in `CreateAPIKey`, prisma_store.go:242–244) and is what the rate
limit middleware reads to pick a bucket (middleware/ratelimit.go:41).
`@@index([userId])` (schema.prisma:60) exists for the FK relation but, like
`verification_tokens`, has no direct query exercising it in
`prisma_store.go` yet (there's no "list a user's API keys" endpoint).

### `click_events` (model `ClickEvent`, schema.prisma:64–78)

| Field | Notes |
|---|---|
| `link_id` | FK → `links.id` |
| `timestamp` | default now(), but in practice set explicitly by the worker to the event's actual click time (worker.go:100) |
| `referrer`, `country`, `device_type`, `ip_hash` | all nullable, best-effort |

Three indexes here, each justified by a real query: `@@index([linkId])`
(schema.prisma:74) backs `GetLinkStats`'s per-link fetch
(`ClickEvent.LinkID.Equals(...)`, prisma_store.go:334–336) and `DeleteLink`'s
cascade-style cleanup (`ClickEvent.LinkID.Equals(...).Delete()`,
prisma_store.go:127–131, needed because Postgres would otherwise reject
the link delete on the FK constraint). `@@index([timestamp])`
(schema.prisma:75) isn't directly exercised by a `WHERE timestamp`
predicate anywhere in `prisma_store.go` today — the daily-grouping logic
in `GetLinkStats` (prisma_store.go:376–388) buckets by day in Go after
fetching all events for a link, not via a timestamp-range query. The
composite `@@index([linkId, timestamp])` (schema.prisma:76) is what
actually backs `GetRecentClickEvents`
(`ClickEvent.LinkID.Equals(...).OrderBy(ClickEvent.Timestamp.Order(SortOrderDesc)).Take(limit)`,
prisma_store.go:510–514) — filter by `link_id` and sort by `timestamp`
without a separate in-memory sort, which the code's own comment calls out
directly ("backed by the ClickEvent(linkId, timestamp) composite index
... rather than fetching everything and sorting in Go," prisma_store.go:507–509).

Worth flagging as a real finding, not just description: `GetLinkStats`
(prisma_store.go:322–397) and `GetTopLinks` (prisma_store.go:400–459) both
load **every** click event for a link (or for all links, in `GetTopLinks`'s
case — `s.client.ClickEvent.FindMany().Exec(ctx)` with no filter at all,
prisma_store.go:416) into memory and aggregate in Go, rather than using
`COUNT`/`GROUP BY` at the database level. The `timestamp` index exists but
isn't leveraged by either of these — they're full per-link (or full-table)
reads followed by Go-side counting and insertion sort. This works fine at
current data volumes but is the clearest N+1-adjacent / scale-limiting
pattern in the analytics read path, and the index design on `click_events`
is ahead of what the queries actually use.
