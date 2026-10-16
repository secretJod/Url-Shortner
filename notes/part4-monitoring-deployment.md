
# Part 4 — Monitoring, CI/CD, Deployment, and Load Testing

The first three parts (wherever they live) presumably covered the shortener
itself: the redirect hot path, the auth model, the data layer. This part
covers everything that turns "a Go binary that works on my machine" into
"a thing with a public URL, a dashboard, and an automated pipeline that
redeploys it." That's nginx, Prometheus/Grafana, GitHub Actions, and a real
(if modest) load test against a real (if temporary) deployment.

One thing up front, because it shapes how to read everything below: this
repo's own `DEPLOY.md` and several of its `docs/system-design/adr/*.md`
files describe an *earlier* deployment story — a Windows+WSL2 self-hosted
runner, an Oracle/Render/Cloud-Run provider menu, a CD pipeline that was
"accepted, implemented, then reverted." None of that is what's running
today. The actual commit history (`git log --oneline --all`, checked while
writing this) shows the deploy workflow came back on October 15, 2026
(`8d97404`), and the real deploy target turned out to be an Apple Silicon
(arm64) machine, not Windows — confirmed by commit `6e67a8f`'s own message,
quoted in full in Chapter 25. Per this project's own ground rule (§36,
`CLAUDE.md`): code and git history are the source of truth, not stale docs.
Where the two disagree below, that's flagged explicitly rather than quietly
picking one.

---

## 21. nginx — why a reverse proxy sits in front of the API at all

### The problem it solves

Without nginx, `docker-compose.prod.yml` would have no choice but to publish
the Go API container's port 8080 straight to the internet. That's not
hypothetical — look at what the prod overlay actually does to every *other*
service:

```yaml
# docker-compose.prod.yml:58-64
postgres:
  ports: !reset []
redis:
  ports: !reset []
```

Postgres and Redis lose their host port mappings entirely in production.
They're reachable only over the internal Docker network, by service name.
The API can't disappear the same way — something has to be public, or
there's no product. nginx is that something: one container whose only job
is to be the public entrypoint, so the app server itself never has to be.

That buys three concrete things without writing a line of Go:

1. **A single, auditable point of public exposure.** One container, one
   config file, one thing to reason about when asking "what can the
   internet reach?"
2. **Headers and policy at the edge, for free.** Security headers, gzip,
   routing — done once in nginx config instead of as middleware in every
   backend language/framework that might ever sit behind it.
3. **A seam for TLS later**, without having to touch the Go binary. TLS
   termination is a proxy-layer concern; putting nginx in front now means
   adding HTTPS later is a config change at the edge, not an application
   change.

The alternative — exposing Fiber directly — isn't wrong in the sense of
"won't run." It's wrong in the sense that TLS, security headers, and
origin-handling logic would all have to be either hand-rolled in Go
middleware or simply absent. A reverse proxy is the industry's standard
answer to "give me these things without writing them."

### Walking `nginx/nginx.conf` line by line

```nginx
worker_processes auto;          # one worker per CPU core — nginx scales
                                 # itself to the host, no tuning needed here

events {
    worker_connections 1024;    # max simultaneous connections per worker
}

http {
    include       /etc/nginx/mime.types;   # correct Content-Type by file ext
    default_type  application/octet-stream;

    sendfile        on;         # kernel-level file send, skips a userspace copy
    keepalive_timeout  65;      # how long to hold idle keep-alive connections

    gzip on;
    gzip_types text/plain text/css application/json application/javascript
               text/xml application/xml application/xml+rss text/javascript;
    gzip_min_length 256;        # don't bother compressing tiny responses

    upstream api_backend {
        server api:8080;        # Docker service name, not localhost — resolved
                                 # via Docker's embedded DNS on the compose network
    }

    server {
        listen 80;
        server_name _;          # catch-all: respond regardless of Host header
```

That `upstream api_backend { server api:8080; }` block is worth pausing on,
because it's the clearest instance in the whole stack of the
`CLAUDE.md` Docker-networking rule (§4): "services must communicate using
Docker networking/service names rather than incorrectly using `localhost`."
`api` here isn't a hostname that happens to work — it's the literal
Docker Compose service name, resolved by Docker's internal DNS. On the host
machine, `api` means nothing; inside the compose network, it's the only
address nginx needs for the whole backend.

Security headers, applied to every response:

```nginx
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Content-Security-Policy "default-src 'self'; img-src 'self' data:; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'" always;
```

- `X-Frame-Options: DENY` — this site can never be framed by another site.
  Kills classic clickjacking.
- `X-Content-Type-Options: nosniff` — stops browsers from guessing a
  different MIME type than the one the server declared, which closes off a
  class of content-sniffing XSS.
- `Referrer-Policy: strict-origin-when-cross-origin` — don't leak full URLs
  (which, for a URL shortener, could include meaningful path segments) to
  third-party `Referer` headers on cross-origin navigation.
- The CSP is a real allowlist, not a report-only stub: scripts, styles,
  images, and XHR/fetch (`connect-src`) are all restricted to `'self'`
  (images additionally allow `data:` URIs). `style-src` needs
  `'unsafe-inline'` — consistent with a Vite/Tailwind-built SPA that injects
  inline styles at runtime.

The proxy block itself:

```nginx
location / {
    proxy_pass http://api_backend;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

This is the part that does the actual forwarding, and the four
`proxy_set_header` lines are what make the forwarding *honest*. Without
them, every request the Go app sees would appear to originate from nginx
itself (`127.0.0.1` or the Docker bridge IP), not the real client:

- `Host $host` — preserves the original Host header the client sent.
- `X-Real-IP $remote_addr` — the immediate connecting IP (nginx's own view).
- `X-Forwarded-For $proxy_add_x_forwarded_for` — appends to any existing
  forwarding chain rather than overwriting it, so a chain of proxies stays
  traceable.
- `X-Forwarded-Proto $scheme` — tells the backend whether the original
  request was http or https, which matters once TLS terminates upstream of
  nginx (see below) and the Go app needs to know the client actually used
  HTTPS even though the hop it sees is plain HTTP.

This is a concrete instance of "things a proxy does for free": reproducing
this logic in Go middleware is possible, but it's the kind of boilerplate
every single backend behind every single reverse proxy in the world
re-implements identically — which is exactly the kind of thing a proxy
layer exists to centralize.

### Explicit on TLS: nginx here is HTTP-only

`nginx.conf` has exactly one `listen` directive — `listen 80;` — and no
`ssl_certificate` anywhere in the file. There is no TLS termination inside
this container, deliberately. `DEPLOY.md`'s "Notes" section spells out the
actual plan:

> use a **Cloudflare Tunnel** (`cloudflared tunnel --url http://localhost:80`
> ...): it's outbound-only from the runner machine ... and terminates TLS at
> Cloudflare's edge — so nginx itself can stay HTTP-only as configured.

The same applies to the other environment this project runs in, GitHub
Codespaces, whose own edge handles HTTPS for the forwarded port before
traffic ever reaches this container. Either way, nginx's job stops at being
the single internal entrypoint; TLS is somebody else's edge, by design —
ADR-0005 confirms this explicitly: "TLS termination is not yet configured —
deployment-target-dependent."

> **★ interview tip**
> **Q: If nginx has no TLS config, isn't that site serving plaintext HTTP?**
> A: From nginx's own perspective, yes — it only speaks HTTP on port 80.
> But that's not the end of the path a real client takes. In this project's
> actual operating mode, the public-facing hop (Cloudflare's tunnel edge, or
> GitHub Codespaces' own edge) terminates HTTPS *before* traffic reaches this
> machine at all, and reaches nginx over the tunnel's own encrypted channel.
> So the browser-to-edge leg is HTTPS; the edge-to-nginx leg, running over a
> private tunnel on the operator's own machine rather than the open
> internet, is HTTP. That's a legitimate, common pattern (it's exactly how
> most CDN/tunnel setups work) — but it's worth being able to say precisely
> *where* TLS terminates rather than asserting "it's HTTPS" as a blanket
> claim about the whole path.

> **★ interview tip**
> **Q: Why have nginx at all if Cloudflare's tunnel is already acting as a
> proxy?**
> A: The tunnel solves "how do I get traffic from the internet to my
> machine without a static IP or opened router ports" — it's a network-layer
> concern. nginx solves a different layer: once traffic *is* on the machine,
> what single internal entrypoint enforces headers, gzip, and routing before
> anything touches the app? Swap the tunnel for a VM with a real public IP
> tomorrow and nginx's job is unchanged; the tunnel is incidental to how
> traffic currently arrives, nginx is structural to how it's handled once it
> does.

---

## 22. Metrics, and what they actually tell you

There are genuinely two separate metrics surfaces in this codebase, and
they serve two different consumers. Worth being precise about both, because
it's easy to conflate them.

### `/metrics` — Prometheus exposition format

Defined in `backend/internal/metrics/prometheus.go` (71 lines), registered
via `promauto` (safe to call more than once — idempotent registration).
These are the *real* metric names, copied verbatim, not paraphrased:

```go
// backend/internal/metrics/prometheus.go
HTTPRequestsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{Name: "urlshortener_http_requests_total", ...},
    []string{"method", "path", "status"},
)

HTTPRequestDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "urlshortener_http_request_duration_seconds",
        Buckets: prometheus.DefBuckets,
    },
    []string{"method", "path"},
)

RedirectsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{Name: "urlshortener_redirects_total", ...},
    []string{"result"},
)

RateLimitTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{Name: "urlshortener_rate_limit_total", ...},
    []string{"result"},
)

AnalyticsEventsProcessed = promauto.NewCounter(
    prometheus.CounterOpts{Name: "urlshortener_analytics_events_processed_total", ...},
)
AnalyticsEventsFailed = promauto.NewCounter(
    prometheus.CounterOpts{Name: "urlshortener_analytics_events_failed_total", ...},
)
AnalyticsProcessingDuration = promauto.NewHistogram(
    prometheus.HistogramOpts{
        Name: "urlshortener_analytics_processing_duration_seconds",
        Buckets: prometheus.DefBuckets,
    },
)
```

Six metrics total, all namespaced `urlshortener_*`. Two are label-free
(`AnalyticsEventsProcessed`/`Failed`), the rest carry labels sized to stay
low-cardinality — `method`/`path`/`status` on the HTTP ones, `result` on
redirects and rate-limit decisions. That "`path`, not raw URL" choice is
deliberate and documented right in the middleware that populates it:

```go
// backend/internal/middleware/prometheus.go:19-23
// Resolve the registered route path (not the raw URL)
routePath := c.Route().Path
if routePath == "" {
    routePath = c.Path()
}
```

`c.Route().Path` is Fiber's *registered* route pattern — `/:shortCode`, not
`/gh-mirror` or `/abc123`. If this instead recorded the raw path, every
distinct short code ever visited would mint a new Prometheus time series
forever, a textbook cardinality explosion that would eventually make the
`/metrics` endpoint itself slow and Prometheus's storage grow unboundedly.
Using the route pattern caps the `path` label to the fixed set of routes
the app actually registers, regardless of how many short links exist.

The middleware wires both metrics on every request:

```go
// backend/internal/middleware/prometheus.go:11-31
func PrometheusMetricsMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        err := c.Next()
        routePath := c.Route().Path
        if routePath == "" { routePath = c.Path() }
        method := c.Method()
        status := strconv.Itoa(c.Response().StatusCode())
        metrics.HTTPRequestsTotal.WithLabelValues(method, routePath, status).Inc()
        metrics.HTTPRequestDuration.WithLabelValues(method, routePath).Observe(time.Since(start).Seconds())
        return err
    }
}
```

This is Prometheus's text exposition format — the standard a Prometheus
server scrapes directly, line-oriented `metric_name{labels} value`, built
by the `promhttp` handler the Prometheus client library provides. It's
machine-first: meant to be pulled by Prometheus on an interval, not read by
a human in a browser tab (though you can, and it's a reasonable way to
sanity-check what's actually being recorded).

### `/api/metrics` — JSON, for the admin dashboard

A completely separate, hand-rolled metrics system lives in
`backend/internal/metrics/metrics.go` (92 lines): atomic counters
(`atomic.Int64`), no external dependency, a `Snapshot()` method that
returns a `map[string]interface{}` — uptime, total requests, a
status-code breakdown, average/max latency, cache hit rate, events
processed/failed. This is what the frontend's admin dashboard actually
fetches and renders as numbers on a page; it's JSON because a React
component calling `fetch()` wants JSON, not a Prometheus exposition blob it
would have to parse by hand.

### Why both exist, instead of just one

They're not redundant — they're answering different questions for different
readers:

- `/metrics` (Prometheus format) exists so **Prometheus** can scrape it on
  an interval, store it as time series, and let Grafana query *trends over
  time* — rate of requests per second, p95 latency, rate-limit rejection
  rate over the last hour. That's the whole value proposition of a
  time-series database: questions shaped like "how has this changed."
- `/api/metrics` (JSON) exists so **a browser tab** can get a cheap,
  self-contained snapshot — "what's true right now" — without standing up
  Prometheus/Grafana at all. It's useful specifically when there's no
  monitoring stack running (e.g. a quick local check), and it's a much
  smaller, simpler dependency for the frontend than teaching a React
  component to parse Prometheus's text format or proxy through a Grafana
  panel.

Two independent recording paths, though — `Metrics.RecordRequest()` (used
for the JSON snapshot) and `HTTPRequestsTotal`/`HTTPRequestDuration` (used
for Prometheus) are two different counters tracking the same event from two
different call sites, not one feeding the other. They should broadly agree
in shape (both count every request), but they're not guaranteed to be
byte-for-byte identical — worth knowing if the two ever appear to disagree
on an exact number.

### What `prometheus.yml` actually scrapes

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "/etc/prometheus/alerts.yml"

scrape_configs:
  - job_name: 'urlshortener'
    static_configs:
      - targets: ['api:8080']
        labels:
          app: 'urlshortener'
```

One job, one static target: `api:8080` — again the Docker service name, not
`localhost`. Every 15 seconds, Prometheus pulls whatever's currently at
`http://api:8080/metrics`. The `rule_files` line references
`/etc/prometheus/alerts.yml`, which is a path *inside the Prometheus
container*, not a file found anywhere in this repo — so either that file is
mounted some other way not visible here, or it's a forward-reference to
alerting rules that don't exist yet. Worth flagging as `UNKNOWN` rather than
assuming either way.

### What Grafana actually provisions

Two files under `monitoring/grafana/provisioning/`:

```yaml
# monitoring/grafana/provisioning/datasources/datasource.yml
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
```

Again, `prometheus:9090` — service name. Grafana doesn't need to be told
where Prometheus lives beyond the Docker network address.

```yaml
# monitoring/grafana/provisioning/dashboards/dashboards.yml
providers:
  - name: urlshortener
    type: file
    options:
      path: /etc/grafana/provisioning/dashboards
```

And the one dashboard that path resolves to,
`monitoring/grafana/provisioning/dashboards/urlshortener.json` — title "URL
Shortener Overview," four panels, `refresh: "10s"`, default time window
`now-1h` to `now`:

1. **HTTP Request Rate** — `sum(rate(urlshortener_http_requests_total[1m])) by (path, status)`
2. **Redirects by Result** — `sum(rate(urlshortener_redirects_total[1m])) by (result)`
3. **Rate Limit Decisions** — `sum(rate(urlshortener_rate_limit_total[1m])) by (result)`
4. **API Up** — `up{job="urlshortener"}`

All four queries map directly onto the metric names shown above — nothing
invented, nothing provisioned that the backend doesn't actually emit.

> **★ interview tip**
> **Q: Why does the HTTP Request Rate panel use `rate()` instead of just
> plotting the raw counter?**
> A: `urlshortener_http_requests_total` only ever goes up — it's a counter.
> Plotted raw, it's a monotonically climbing line that tells you "how many
> requests total since the process started," which flattens any interesting
> signal about *current* traffic into an ever-steepening slope. `rate(...[1m])`
> converts it into "requests per second, averaged over the trailing minute" —
> a value that can go up *and* down, which is what you actually want on a
> dashboard meant to show live traffic shape.

---

## 23. The observability story, end to end — tracing one real panel

Walking the full path for a single concrete number, rather than describing
it abstractly:

1. **A request lands.** Say `GET /abc123` — a redirect lookup.
   `backend/cmd/api/main.go` has already registered
   `PrometheusMetricsMiddleware()` ahead of the catch-all `/:shortCode`
   route (route-ordering matters here exactly as the codebase guide notes —
   a route registered *after* the catch-all would never reach this
   middleware at all).
2. **The middleware records it**
   (`backend/internal/middleware/prometheus.go:11-31`, quoted in full
   above). It captures `start := time.Now()`, calls `c.Next()` to let the
   actual redirect handler run, then — once the handler returns — reads the
   *registered route path* (`/:shortCode`, not `/abc123`), the HTTP method
   (`GET`), and the final status code (`302`, or `404` if the code doesn't
   exist). Two things happen in memory inside the API process at this
   point: `HTTPRequestsTotal.WithLabelValues("GET", "/:shortCode", "302").Inc()`
   bumps a counter by one, and
   `HTTPRequestDuration.WithLabelValues("GET", "/:shortCode").Observe(...)`
   records this request's latency into a histogram bucket. Nothing is sent
   anywhere yet — this is purely an in-process counter increment.
3. **Prometheus scrapes `/metrics` every 15 seconds** (`prometheus.yml`'s
   `scrape_interval: 15s`, target `api:8080`). On its next tick, Prometheus
   makes an HTTP GET to `http://api:8080/metrics`, which the
   `promhttp` handler inside the Go binary answers by dumping the current
   in-memory state of every registered metric — including the counter this
   request just bumped — as plain text. Prometheus parses that text,
   timestamps it, and appends it to its own time-series storage.
4. **Grafana queries Prometheus.** The "HTTP Request Rate" panel's query —
   `sum(rate(urlshortener_http_requests_total[1m])) by (path, status)` —
   isn't evaluated once; Grafana re-runs it on the dashboard's own refresh
   cycle (`"refresh": "10s"` in `urlshortener.json`). Each time it fires,
   Grafana sends that PromQL expression to `http://prometheus:9090` (the
   datasource URL from `datasource.yml`), and Prometheus computes the
   per-second rate of that counter over the trailing 1-minute window,
   grouped by `path` and `status`, from the time series it's been
   accumulating via scrapes.
5. **The panel updates.** The line for `{path="/:shortCode", status="302"}`
   ticks up, reflecting that this redirect — and every other `/:shortCode`
   hit in the trailing minute — is now part of the plotted rate. Nothing
   about this redirect hits a database write on this path (the click event
   itself goes onto the Redis Stream for the analytics worker, a completely
   separate flow); the number on this specific panel comes entirely from
   the in-memory Prometheus counter the middleware bumped in step 2.

That's the full loop, concretely: **app counter → scrape → PromQL
aggregation → panel.** No step in that chain touches Postgres, and no step
is instantaneous — there's a real, bounded lag (up to the 15s scrape
interval, then up to the panel's own 10s refresh) between "a redirect
happened" and "the dashboard shows it."

> **★ interview tip**
> **Q: How stale can a Grafana panel be relative to real traffic?**
> A: Bounded by two independent intervals stacking: up to 15s for
> Prometheus to next scrape `/metrics` (`prometheus.yml`'s
> `scrape_interval: 15s`), plus up to 10s for the dashboard's own panel
> refresh (`urlshortener.json`'s `"refresh": "10s"`). Worst case, a request
> that just happened can take up to ~25 seconds to visibly move a line on
> the dashboard. That's a deliberate, reasonable trade for a low-traffic
> showcase app — it would be the first knob to tighten for anything
> latency-sensitive.

---

## 24. The CI/CD pipeline, fully

Two workflow files, two different triggers, chained together deliberately.

### `ci.yml` — test, then build/push

Trigger (`.github/workflows/ci.yml:3-7`): `push` to `main`, or any
`pull_request` targeting `main`.

**`test` job** (`ci.yml:23-81`), on `ubuntu-latest`:

1. `actions/checkout@v4`.
2. `actions/setup-go@v5`, Go 1.25, with `cache: false` — and that's not an
   oversight, it's a documented, currently-unresolved debugging decision.
   The comment (`ci.yml:31-43`) is worth reading in full: every CI run had
   failed at "Run tests" since the very first commit, yet the identical
   steps reproduce clean locally on two different architectures — the one
   variable that *can't* be reproduced locally is GitHub's own module
   cache, and this repo's `go.mod` has non-standard `replace` directives
   pointing at GitHub mirror repos instead of the normal module proxy,
   "exactly the shape that can produce a stale/bad cache entry." Disabling
   caching was the test of that hypothesis.
3. `go mod download` (`backend/`).
4. **Generate the Prisma client** (`ci.yml:49-70`) — `prefetch` then
   `generate`. The comment here documents the *actual root cause* that was
   finally confirmed, quoting the exact failure from a genuinely fresh
   clone: `internal/db/prisma_store.go:27:10: undefined: PrismaClient`.
   `internal/db/*_gen.go` is gitignored (per the codebase guide) and only
   exists after this generation step runs — without it, `internal/db` and
   anything importing it (`cmd/api`) fail to even *compile*, not merely
   fail a test. Commit `0ed94c7` is this exact fix, and its message
   explicitly notes a prior, wrong fix (disabling Go caching, for what
   turned out to be an unrelated reason) that's "superseded by this
   commit" — a documented false lead left visible rather than scrubbed.
5. `go test ./... -v` (`ci.yml:72-81`) — no `|| echo` swallow, by explicit
   design: the comment states plainly that a prior version of this step did
   swallow failures, which meant "a broken build could still be tagged and
   pushed to GHCR, and then deployed by deploy.yml." (The codebase guide's
   own note — "`go test ./...` (currently swallows failure via `|| echo`)"
   — describes that earlier state; the committed workflow on disk today
   does not have that fallback.)

**`build` job** (`ci.yml:83-143`), `needs: test` — only runs if tests pass:

1. Checkout.
2. `docker/setup-qemu-action@v3` — cross-architecture emulation, needed
   because GitHub's hosted runners are amd64-only and the deploy target
   turned out to be arm64 (see Chapter 25).
3. `docker/setup-buildx-action@v3`.
4. **Smoke-test build** (`ci.yml:105-118`): `push: false`, `load: true`,
   single-platform (the runner's own amd64) — just confirms
   `backend/Dockerfile` still builds, loaded into the local daemon for
   inspection. Deliberately *not* multi-platform, because `load: true`
   can't accept a multi-arch manifest list.
5. Login to `ghcr.io`, gated `if: github.event_name == 'push' && github.ref == 'refs/heads/main'`
   — PRs never get this far, only pushes to `main`.
6. **Real push** (`ci.yml:128-143`): `push: true`,
   `platforms: linux/amd64,linux/arm64`, tagged both
   `${{ github.sha }}` and `latest`. The `env.IMAGE_NAME` this all resolves
   under is hardcoded lowercase — `ghcr.io/secretjod/url-shortner` — and
   the comment at `ci.yml:9-20` explains why it isn't the interpolated
   `ghcr.io/${{ github.repository }}`: the actual GitHub repo is
   `secretJod/Url-Shortner`, mixed case, and Docker image references must
   be entirely lowercase — the interpolated form produced an invalid
   reference and silently broke the push step.

### `deploy.yml` — triggered by CI's success, not by the push itself

Trigger (`deploy.yml:22-27`): `workflow_run` on workflow `"CI"`,
`types: [completed]`, `branches: [main]` — plus manual
`workflow_dispatch`. The header comment (`deploy.yml:8-14`) explains the
ordering choice precisely: both workflows fire off the same push, but this
job — a checkout and a file write — would reach `docker compose pull`
within seconds, "long before a hosted runner finishes a Node+Go+Prisma
multi-stage build and a GHCR push." Pulling a commit-SHA tag that doesn't
exist yet fails with "manifest unknown" and aborts. `workflow_run` makes
the dependency on CI's completion explicit instead of racing it.

A `concurrency` group (`deploy.yml:31-33`) limits this to one deploy at a
time — two overlapping `compose up` runs would fight over container names
and volumes.

The job itself (`runs-on: self-hosted`), gated
(`deploy.yml:40-42`) to only actually run when
`github.event.workflow_run.conclusion == 'success'` or a manual dispatch —
otherwise a `workflow_run` event fires on CI *failure* too, which would
deploy a red build.

**Checkout pinned to the exact commit CI built** (`deploy.yml:45-50`):
`ref: ${{ github.event.workflow_run.head_sha || github.sha }}` — because the
default ref for a `workflow_run` event is the branch tip, which may have
moved since CI started.

**Writing `.env` from secrets** (`deploy.yml:52-117`) — this is the step
worth walking carefully, because the technique is deliberate, not
incidental:

```yaml
env:
  POSTGRES_PASSWORD: ${{ secrets.POSTGRES_PASSWORD }}
  REDIS_PASSWORD: ${{ secrets.REDIS_PASSWORD }}
  # ... nine more
run: |
  set -euo pipefail
  required_vars="POSTGRES_PASSWORD REDIS_PASSWORD IP_HASH_SECRET BASE_URL CORS_ALLOWED_ORIGINS GF_SECURITY_ADMIN_PASSWORD MAIL_FROM"
  missing=""
  for var_name in $required_vars; do
    if [ -z "${!var_name:-}" ]; then
      missing="$missing $var_name"
    fi
  done
  if [ -n "$missing" ]; then
    echo "::error::Missing required secret(s):$missing — ..."
    exit 1
  fi
  umask 077
  { echo "ENV=production"; echo "POSTGRES_PASSWORD=${POSTGRES_PASSWORD}"; ... } > .env
  chmod 600 .env
```

Secrets reach this script via a step-level `env:` mapping — `$VAR`, shell
variables — never `${{ secrets.X }}` interpolated directly into the script
body. The comment (`deploy.yml:52-63`) states why: a secret value
containing `$`, a quote, or a newline must not be able to break the script
or leak into the workflow log, which direct `${{ }}` interpolation into a
multi-line `run:` block risks. The `"${!var_name:-}"` syntax
(`deploy.yml:90-96`) is **indirect expansion** — it reads the value of the
variable *named by* `$var_name` (e.g. `POSTGRES_PASSWORD`) without the
script's own text ever containing a secret literal, letting one loop check
all seven required vars instead of seven repeated `if` blocks each naming a
secret directly.

`umask 077` before the write, and `chmod 600 .env` after, mean the file
exists without world/group read access from the moment it's created.
`ENV=production` is set unconditionally here — this is what turns on
`config.go`'s `validateForProduction()` guard, per the codebase guide:
without it, that guard never runs and the app would boot happily on
committed dev placeholder secrets.

SMTP vars are deliberately *not* in `required_vars` — the comment
(`deploy.yml:79-86`) explains that leaving them unset is what selects the
MailHog fallback path, which is the actual current operating mode (see
Chapter 27).

**Pull, then deploy with the mandatory prod overlay**
(`deploy.yml:119-139`):

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --remove-orphans
```

Always both files, never the bare compose file — `docker-compose.prod.yml`
is what strips host ports from postgres/redis/prometheus/grafana/api,
swaps the ghcr.io image in for a local build, disables MailHog's ports, and
replaces the `migrate` service with a no-op (see Chapter 25 for why that
last one matters on the very first deploy).

**Health verification, not just "did `up` exit 0"**
(`deploy.yml:144-193`): a loop (up to 24 iterations, 5s apart — a two-minute
budget) polling `docker compose ps --format '{{.Name}} {{.Health}}'`,
failing immediately on any `unhealthy` status, and separately failing if
anything is still `starting` after the full budget. Then a second loop (up
to 12 iterations, 5s apart) that actually `curl`s
`http://localhost/health` **through nginx** — the comment
(`deploy.yml:170-175`) is explicit that this is deliberate: "A successful
`docker compose up` does not guarantee the app is actually reachable, so
hit it for real and fail the job if it doesn't." Both failure branches dump
the last 100–200 lines of `docker compose logs` before exiting non-zero, so
a failed deploy run leaves a diagnosable trail in the Actions log rather
than a bare exit code.

> **★ interview tip**
> **Q: Why `workflow_run` instead of just giving `deploy.yml` the same
> `push: branches: [main]` trigger as `ci.yml`?**
> A: Because they'd then be racing each other on the same event, and the
> deploy job (checkout + write a file + `docker compose pull`) finishes in
> seconds — long before CI's own multi-stage Docker build and GHCR push
> complete. Pulling a tag that doesn't exist yet fails outright. Making
> deploy depend on CI's own completion event, rather than mirroring CI's
> trigger, makes the ordering a structural guarantee instead of a timing
> coincidence that happens to usually work.

---

## 25. The real deployment story

Stated plainly, in the order it actually happened — cross-checked against
`git log --oneline --all` rather than assumed:

**1. The self-hosted runner needed an actual logged-in desktop session, not
just SSH access.** This is a real operational quirk of GitHub's self-hosted
runner on macOS: a runner installed as a `launchd` service under a user
context that isn't an active, logged-in GUI session can fail to execute
Docker-dependent jobs correctly — Docker Desktop's own daemon socket is
only reliably available to processes running inside a real logged-in
session, not a bare background/headless one. That's a macOS platform
detail, not a Docker Compose or GitHub Actions config problem, and it's
exactly the kind of thing that looks like a CI bug until you realize it's
an OS session-management quirk. (This detail comes from the operator's own
account of standing the runner up; it doesn't appear as commit text in the
repo's history, so it's reported here as told, not independently verified
against a commit.)

**2. The first real deploy attempt failed on an architecture mismatch.**
Commit `6e67a8f`'s message says this plainly — quoting it directly because
it's the clearest first-hand account in the repo:

> deploy.yml's first real run against an Apple Silicon (arm64) machine
> failed with "no matching manifest for linux/arm64/v8 in the manifest list
> entries" when pulling ghcr.io/secretjod/url-shortner. Every official base
> image used here (postgres, redis, nginx, grafana, prometheus, golang)
> already ships multi-arch — only our own CI-built image didn't, because
> GitHub's hosted runners are amd64-only and docker/build-push-action
> defaults to publishing just the runner's own architecture.

The fix was exactly what Chapter 24 already walked through: `setup-qemu-action`
plus `platforms: linux/amd64,linux/arm64` on the real push step
(`ci.yml:92-140`), merged via PR #6 (`51bc89f`) on the same day,
October 15, 2026. This is also the detail that settles the Windows-vs-Mac
question raised in the intro: `DEPLOY.md` and the deploy.yml header comment
both describe a WSL2/Windows target, but the one commit that actually
documents a real failed deploy attempt names an Apple Silicon (arm64)
machine explicitly. The repo's own documentation is stale on this point —
treat the commit, not the doc, as what happened (`CLAUDE.md` §36).

**3. The production overlay deliberately disables automatic schema push,
so a brand-new database needed one manual, deliberate `prisma db push`
before the stack could come up healthy.** This follows directly from how
`docker-compose.prod.yml`'s `migrate` service is defined
(`docker-compose.prod.yml:91-98`):

```yaml
migrate:
  environment:
    DATABASE_URL: "postgresql://urlshortener:${POSTGRES_PASSWORD:?...}@postgres:5432/urlshortener?schema=public"
  command: sh -c "echo 'Skipping automatic schema push in production. Apply schema changes deliberately...' && exit 0"
```

In dev, `docker-compose.yml`'s `migrate` service actually runs
`prisma db push`, gating `api` via `depends_on:
service_completed_successfully` (per the codebase guide). In prod, that
command is replaced with a no-op that exits 0 immediately — satisfying
`api`'s dependency condition without ever touching the schema. The header
comment (`docker-compose.prod.yml:39-55`) explains why: `db push`
reconciles the live schema with *no migration history and no rollback
path* — fine against throwaway dev data, a real risk of silent data loss
against anything that matters. Deliberate consequence: a database that has
never seen the schema applied (a fresh Postgres volume, which is exactly
what a first deploy to a new machine starts with) will **not** get a
schema from `docker compose up` alone in production. That has to be a
conscious, separate step — exactly what happened on this deploy: the
operator ran `prisma db push` by hand, once, deliberately, against the new
database, before the stack reported healthy. This is the overlay doing
precisely what its own comments say it's for — trading automatic
convenience for a loud, manual gate on anything that touches schema in a
real environment.

> **★ interview tip**
> **Q: Isn't disabling the automatic migration step in production a
> regression — doesn't dev get something prod doesn't?**
> A: It's the opposite of a regression; it's the entire point. `prisma db
> push` is a *schema-reconciliation* tool with no migration history and no
> rollback — safe against data you're willing to throw away (dev), unsafe
> against data you're not (prod, where a column rename or type change could
> silently destroy rows with nothing to revert to). The overlay doesn't
> remove schema management from prod — it removes the *automatic,
> unattended* version of it, replacing it with a conscious, separate,
> reviewed step. That trade — slower, but loud and deliberate instead of
> fast and silent — is exactly what you want once there's real data to
> lose.

---

## 26. Load testing, for real

`load-tests/README.md` records results from **2026-10-15, against the
self-hosted deployment via Cloudflare quick tunnel** — the same day as the
multi-arch fix and the deploy workflow's reintroduction, per the git log.
Two scripts, both k6.

### `rate-limit-validation.js` — confirming the sliding-window limiter

30 rapid requests, one virtual user, `redirects: 0` so k6 sees the raw 302
instead of following it:

```js
export const options = {
  scenarios: {
    burst: { executor: 'per-vu-iterations', vus: 1, iterations: 30, maxDuration: '30s' },
  },
};
export default function () {
  const res = http.get(`${TARGET}/${SHORT_CODE}`, { redirects: 0 });
  check(res, { 'is 302 (allowed) or 429 (rate-limited)': (r) => r.status === 302 || r.status === 429 });
  if (res.status === 429) {
    check(res, { '429 includes Retry-After header': (r) => r.headers['Retry-After'] !== undefined });
  }
}
```

The actual recorded result, not rounded: **19 allowed (302), 11 correctly
rejected (429)**, every single rejection carrying a `Retry-After` header.
That's checked explicitly against the anonymous rate-limit tier the
codebase guide documents — "anon 20/min per IP" — and the README states it
matches `backend/internal/middleware/ratelimit.go` "exactly." 19 allowed
plus 11 rejected out of 30 is consistent with a ~20/min cap under a tight
burst (the 20th request's exact pass/fail depends on window boundary
timing, which is expected sliding-window behavior, not a bug). 100% of k6
checks passed.

### `redirect-latency.js` — legitimate-traffic latency, deliberately under the cap

```js
export const options = {
  scenarios: {
    steady: { executor: 'constant-arrival-rate', rate: 10, timeUnit: '1m', duration: '1m', preAllocatedVUs: 2 },
  },
  thresholds: { http_req_duration: ['p(95)<500'] },
};
```

10 requests/minute — "well under the 20/min anonymous cap," per the
script's own comment — specifically so this run measures real latency
instead of triggering rejections. Recorded result: **0% errors, 100%
successful redirects, p95 969ms, median 240ms** over 11 requests in 60
seconds.

Worth noting directly: the script's own `thresholds` block asks for
`p(95)<500` (under 500ms), and the actual recorded p95 — 969ms — **fails**
that threshold. The README doesn't hide this; it states the number plainly
(969ms) rather than rounding down toward the threshold or omitting it.

### The honest caveat, stated as written

`load-tests/README.md`'s own words, quoted rather than paraphrased:

> the quick-tunnel path (browser → Cloudflare edge → `trycloudflare.com`
> tunnel → Colima VM → nginx → api) adds real latency that a production
> network path or a named tunnel would not have. These numbers measure
> *this* path on this date, not the backend's raw capacity.

That path — Cloudflare's free *quick* tunnel (not a named, stable tunnel),
terminating into a Colima VM (a lightweight Linux VM for running Docker on
macOS) — has several extra network hops baked into every one of these
numbers: the k6-to-Cloudflare-edge leg, the Cloudflare-to-tunnel leg, and
the tunnel-to-Colima-VM leg, before a request even reaches nginx. A p95
near a full second is believable for that path and says very little about
how fast the redirect handler itself is once a request actually reaches
it. This also tells you *why* the p95 misses the script's own 500ms
threshold: the threshold was written against application-latency
expectations, and what actually got measured includes a free public
tunnel's own overhead on top of that.

> **★ interview tip**
> **Q: The latency test's own threshold says p95<500ms, and the real
> result was 969ms. Doesn't that mean the test failed?**
> A: By k6's own pass/fail criterion, yes — that threshold wasn't met. But
> reading it as "the backend is slow" would be the wrong conclusion. The
> README's own caveat names the actual culprit: every one of these requests
> traveled browser → Cloudflare edge → a free quick tunnel → a Colima VM →
> nginx → the API, and back. That's several extra network hops that have
> nothing to do with how fast the Go redirect handler itself runs. The
> honest move here isn't to quietly raise the threshold or omit the
> number — it's exactly what the README did: report the real 969ms,
> and say plainly that it measures *this path on this date*, not the
> backend's raw capacity.

---

## 27. What's demo vs. real, stated plainly

This chapter exists so none of the above gets mistaken for more than it is.
Three specific places where this repo is honest about synthetic or
not-yet-permanent pieces, stated directly rather than left implicit:

**`scripts/seed-demo-data.sql` is synthetic, and says so in its own header
comment**, quoted directly:

> DEMO / LOCAL-ONLY SEED DATA. NOT FOR PRODUCTION. ... Every row this
> script inserts is clearly synthetic ... This does NOT simulate,
> impersonate, or claim to represent actual production traffic or real end
> users. There is no production deployment of this project serving real
> users today.

Concretely: 8 short links pointing at real public pages (github.com,
wikipedia.org, docs.python.org, etc. — real destinations, to make a demo
*look* plausible), with click counts, referrers, countries, and device
types all script-generated, not observed. `scripts/README.md` adds the
verification detail: this was tested against hand-built tables mirroring
`schema.prisma`, not an actual `prisma db push`'d database — "First run:
executed cleanly, inserted 8 links and 2,709 click events," idempotent on a
second run — but with an explicitly named residual unknown, that Prisma's
real generated DDL might differ in some way this script doesn't account
for. It's also not wired into anything automatic — "not referenced by
`docker-compose.yml`, CI, or the deploy workflow."

**MailHog stands in for a real SMTP relay in the current operating mode —
an admin relays verification links by hand.** `docker-compose.prod.yml`'s
own comment (`docker-compose.prod.yml:66-87`) is explicit that this is a
deliberate reversal of the usual rule ("never run a dev mail catcher in
prod"), made with the trade-off understood: no mail is ever delivered
externally; an admin reads the magic-link email out of MailHog's inbox over
an SSH tunnel (`ssh -L 8025:localhost:8025 <user>@<host>`) and passes the
code to the user by hand. That "does not scale" (the comment's own words),
and it's not meant to — it needs no external mail provider, which keeps the
whole stack at the project's ₹0/$0 constraint. The one real security
control that makes this acceptable at all is that MailHog's web UI is never
published (`ports: !reset []`) — if it were, anyone reaching the host could
read every user's verification link and claim any account. Setting real
`SMTP_HOST`/`SMTP_PORT`/`SMTP_USER`/`SMTP_PASSWORD` secrets switches
`internal/mail/mail.go` to an authenticated STARTTLS relay with no code
change — the capability exists, it's just not the currently exercised path.

**The Cloudflare quick-tunnel's URL is not permanent.** A `trycloudflare.com`
quick tunnel (`cloudflared tunnel --url http://localhost:80`, per
`DEPLOY.md`'s notes) mints a fresh, random subdomain every time it starts —
it is not a stable or reserved address. That's exactly why
`load-tests/README.md` frames its own results as measuring "this path on
this date," not a durable endpoint: the URL the load tests hit that day
will not be the URL a future run hits, and there is no standing guarantee
the tunnel (or the machine behind it) is up at any given moment someone
might try to reach it. `DEPLOY.md` itself recommends a *named* tunnel
(`cloudflared tunnel run`) as the more durable alternative, and is explicit
that the quick-tunnel approach is for exactly this kind of
demonstration-grade use, not a permanent address.

The throughline across all three: this project is honest, in its own
comments and READMEs, about exactly where the line sits between "real
mechanism, demonstrated" and "convenience that makes the demo possible."
None of the above should be read as this project claiming a production
deployment serving real users today — it isn't one, and it says so itself.

> **★ interview tip**
> **Q: If none of this is "real" production traffic, what's actually being
> demonstrated here?**
> A: A genuinely working pipeline and a genuinely working application,
> exercised end to end — CI testing and building a real multi-arch image,
> a real self-hosted runner pulling and deploying it, a real nginx+API+
> Postgres+Redis+Prometheus+Grafana stack coming up healthy, a real rate
> limiter rejecting real excess requests, real latency numbers off a real
> (if tunneled) network path. What's *not* real is the traffic volume and
> the SMTP delivery — there's no production user base and no outbound mail
> provider yet. Those are two specific, named gaps, not a blanket caveat
> over the whole project — and the repo's own comments draw that line
> explicitly rather than leaving a reader to guess which parts to trust.
