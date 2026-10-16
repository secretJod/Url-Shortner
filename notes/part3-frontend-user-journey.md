
# Part 3 — The Frontend and the User Journey

We've spent two parts underground, in Go handlers and Redis streams. Now we
surface. This part is about the thing a human actually looks at: a React SPA
that talks to all of that machinery through exactly one door. We'll walk the
component tree, find that one door, meet the two bits of global state that
hold the app together, and then tell the whole story end to end — a stranger
showing up with no account, and leaving with a working short link and a
dashboard full of stats.

---

## 16. The component tree

Everything starts at `frontend/src/main.jsx`. It's four lines of nesting
that matter:

```jsx
<BrowserRouter>
  <ToastProvider>
    <AuthProvider>
      <App />
    </AuthProvider>
  </ToastProvider>
</BrowserRouter>
```

(`frontend/src/main.jsx:11-17`). Order here isn't decorative — `ToastProvider`
wraps `AuthProvider`, which means anything auth does (including, as we'll see,
nothing on login failure) *could* reach for a toast, and does. `App` sits
innermost, so it's the one place that actually owns the page layout.

`App.jsx` itself is almost pure routing (`frontend/src/App.jsx:13-45`):

```
<div> (flex column, min-h-screen)
 ├─ <Navbar />
 ├─ <main>
 │    <Routes>
 │      "/"              → <LandingPage />
 │      "/login"         → <LoginPage />
 │      "/top"           → <TopLinksPage />
 │      "/dashboard"     → <ProtectedRoute><DashboardPage /></ProtectedRoute>
 │      "/stats/:shortCode" → <ProtectedRoute><StatsPage /></ProtectedRoute>
 │      "/admin"         → <ProtectedRoute><AdminPage /></ProtectedRoute>
 │    </Routes>
 ├─ <Footer />
 └─ <Toast />
```

Three public routes, three gated ones. `ProtectedRoute` (`frontend/src/components/ProtectedRoute.jsx:4-12`)
is deliberately tiny — it reads `isAuthenticated` off `useAuth()` and either
renders `children` or `<Navigate to="/login" replace />`. No loading state,
no role check — it's a boolean gate, nothing more. Worth flagging early: this
gate is just hiding UI. `/admin` is in that list, but nothing here or in the
Go backend ties "has an API key" to "is allowed to see system metrics" — any
authenticated user who manually fetches `/api/metrics` or visits `/admin`
gets the same system-wide dashboard. The route is "protected" from anonymous
visitors, not from ordinary users.

Here's the full tree, with what each leaf actually renders and calls:

```
App
├─ Navbar                         (frontend/src/components/Navbar.jsx)
│    state: mobileMenuOpen
│    reads: useAuth() → isAuthenticated, logout
│    no API calls — theme toggle just flips a DOM class + localStorage.theme
│
├─ Routes
│    ├─ LandingPage                (pages/LandingPage.jsx)
│    │    state: stats {totalLinks, totalClicks}, loadingStats
│    │    calls: GET /api/stats/top?limit=100  (on mount, derives totals client-side)
│    │    renders: ShortenForm, StatCard ×2, FeatureCard ×3 (local, non-exported)
│    │
│    ├─ LoginPage                  (pages/LoginPage.jsx)
│    │    state: email, loading, verificationSent, existingKey, emailFocused
│    │    reads: useAuth() → requestVerification, importKey; useToast()
│    │    calls: requestVerification() → POST /api/keys {email}  (indirect, via AuthContext)
│    │    no direct API call for the "import key" form — it's pure client-side
│    │    regex validation (isValidApiKey) then setApiKey
│    │
│    ├─ TopLinksPage                (pages/TopLinksPage.jsx)
│    │    state: links, loading
│    │    calls: GET /api/stats/top?limit=10   (public, unauthenticated)
│    │    renders: Spinner, plain <Link> to /stats/:shortCode per row
│    │
│    ├─ ProtectedRoute → DashboardPage  (pages/DashboardPage.jsx)
│    │    state: links, loading, searchTerm, showForm, searchFocused
│    │    reads: useAuth() → apiKey; useToast()
│    │    calls: GET /api/links  (on mount, re-run whenever apiKey changes)
│    │           DELETE /api/links/:shortCode  (on delete click)
│    │    renders: ShortenForm (conditionally, toggled by showForm),
│    │             LinkTable (filtered client-side by searchTerm), Spinner
│    │
│    ├─ ProtectedRoute → StatsPage      (pages/StatsPage.jsx)
│    │    params: :shortCode (from the URL)
│    │    state: stats, clicks, loading
│    │    calls (parallel, Promise.all): GET /api/stats/:shortCode
│    │                                    GET /api/links/:shortCode/clicks?limit=20
│    │    renders: StatCard ×2, DailyChart, plus an inline referrers list
│    │             and a recent-clicks table built directly in this file
│    │
│    └─ ProtectedRoute → AdminPage      (pages/AdminPage.jsx)
│         state: metrics, loading
│         calls: GET /api/metrics
│         renders: StatCard ×4, a Recharts PieChart (status codes),
│                  a hand-built cache hit-rate progress bar,
│                  two hit/miss stat tiles
│
├─ Footer                          (components/Footer.jsx) — static, no state, no calls
└─ Toast                           (components/Toast.jsx)
     reads: useToast() → toasts, removeToast
     renders one animated card per toast in a fixed bottom-right stack
```

Shared leaf components, used by more than one page:
- `ShortenForm` (`components/ShortenForm.jsx`) — used by `LandingPage` and
  `DashboardPage`. Owns its own `url`/`customAlias`/`expiresAt`/`result`
  state; calls `POST /api/shorten` directly (not through a context).
- `LinkTable` (`components/LinkTable.jsx`) — pure presentational, takes
  `links` and `onDelete` as props, no API calls of its own.
- `StatCard` (`components/StatCard.jsx`) — pure presentational, a tilt-on-hover
  number tile; no API calls.
- `Spinner` (`components/Spinner.jsx`) — pure presentational.
- `DailyChart` (`components/DailyChart.jsx`) — pure presentational, wraps
  Recharts' `BarChart` around a `data` prop.

**★ interview tip**
> *"Why does `ProtectedRoute` only check `isAuthenticated`, and why can any
> logged-in user reach `/admin`?"*
> Because the frontend's auth model has exactly one tier below "has a valid
> key" — there's no role or scope encoded in the API key itself (see Part 2's
> auth chapter), so there's nothing finer-grained for the route guard to
> check. The fix isn't a frontend one: the backend would need a `role` or
> `admin` flag on the user record, returned from some "who am I" endpoint,
> before `ProtectedRoute` could meaningfully distinguish an admin from any
> other authenticated user.

---

## 17. How the UI and backend actually talk

Every single HTTP request this frontend makes — all ten-ish call sites you
saw in the tree above — goes through one file: `frontend/src/api/client.js`.
It's 48 lines, so let's take it in full.

```js
const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '',
  headers: { 'Content-Type': 'application/json' },
});
```
(`frontend/src/api/client.js:3-8`). `VITE_API_URL` is the only environment
knob. In dev it's typically unset, which means `baseURL` is `''` and every
call is a relative path — Vite's dev server proxies `/api`, `/health`, and
`/metrics` to `:8080` (per the codebase guide), so `apiClient.get('/api/links')`
just works without CORS ever entering the picture. In production the SPA is
served *by* the same Go binary it's calling, so relative paths work there too
— `VITE_API_URL` only matters if you ever split the frontend onto its own
origin.

The request interceptor is the entire authentication story on the client
side:

```js
apiClient.interceptors.request.use((config) => {
  const apiKey = localStorage.getItem('urlshortener_api_key');
  if (apiKey) {
    config.headers.Authorization = `Bearer ${apiKey}`;
  }
  return config;
});
```
(`frontend/src/api/client.js:10-19`). Every outgoing request — authenticated
or not — passes through here and gets the key attached *if one exists in
localStorage*. There's no in-memory source of truth checked first; the
interceptor reads directly from storage on every call. That's notable
because it means `AuthContext`'s `apiKey` state and `localStorage` have to
stay in sync by convention (they do — see Chapter 18), not because the
interceptor enforces it.

The response interceptor is where the app's error handling actually lives:

```js
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      const { status, data } = error.response;
      if (status === 401) {
        localStorage.removeItem('urlshortener_api_key');
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }
      }
      let errorMessage = data?.error || data?.message || 'An unexpected error occurred';
      if (status === 429) errorMessage = 'Too many requests. Please wait a moment and try again.';
      else if (status === 500) errorMessage = 'Server error. Please try again later.';
      return Promise.reject({ status, message: errorMessage, data });
    }
    return Promise.reject({ status: 0, message: 'Network error. Please check your connection.' });
  }
);
```
(`frontend/src/api/client.js:21-46`). Two things worth calling out:

1. **The 401 handler is a hard `window.location.href` redirect**, not a
   React Router navigation. That means a full page reload on every
   session-expiry — simple, reliable, and it blows away any in-flight app
   state, which is arguably fine for an app this size.
2. **Every call site gets a normalized error shape**: `{status, message, data}`,
   never a raw axios error. That's why you see the same pattern everywhere —
   `catch (error) { showToast(error.message || '...', 'error'); }` — in
   `LoginPage`, `DashboardPage`, `StatsPage`, `TopLinksPage`, `AdminPage`.
   The interceptor did the translation once so every page could skip it.

So: `client.js` is simultaneously the auth-header injector, the session-expiry
watchdog, and the error-message translator. Nothing downstream — not a single
page or component — talks to `fetch` or raw `axios` directly. If you wanted
to add request tracing, a loading-progress bar, or retry-on-5xx, this is the
one file you'd touch.

**★ interview tip**
> *"What happens if the API key in localStorage is stale (revoked
> server-side) and the user is sitting on the dashboard?"*
> The next request that hits `/api/links` or similar gets a 401 back. The
> response interceptor clears `urlshortener_api_key` from storage and does a
> full `window.location.href = '/login'`. `AuthContext`'s `apiKey` state
> doesn't get cleared through its own `logout()` here — it's bypassed
   entirely because the redirect reloads the page, remounting
   `AuthProvider` from scratch, which re-reads `localStorage` (now empty) in
   its `useState` initializer (`context/AuthContext.jsx:7`). So the state
   ends up correct, just via a full remount rather than a state update.

---

## 18. AuthContext and ToastContext

These are the only two pieces of state shared across more than one branch of
the tree — everything else (`links`, `stats`, `metrics`, form inputs) is
local to the page that fetched it.

**AuthContext** (`frontend/src/context/AuthContext.jsx`) holds exactly two
pieces of state:

```js
const [apiKey, setApiKey] = useState(localStorage.getItem('urlshortener_api_key') || null);
const [isAuthenticated, setIsAuthenticated] = useState(!!apiKey);
```
(lines 7-8). `isAuthenticated` is derivable from `apiKey` and is kept in sync
by an effect rather than computed inline:

```js
useEffect(() => {
  if (apiKey) {
    localStorage.setItem('urlshortener_api_key', apiKey);
    setIsAuthenticated(true);
  } else {
    localStorage.removeItem('urlshortener_api_key');
    setIsAuthenticated(false);
  }
}, [apiKey]);
```
(lines 10-18). This is the *other* half of the localStorage sync mentioned
in Chapter 17 — the effect writes to storage whenever `apiKey` changes in
React state, while the axios interceptor reads from storage independently.
Two different code paths agreeing on the same key name (`urlshortener_api_key`)
is doing the work that a single shared accessor would normally do.

Three functions are exposed through the context value
(`context/AuthContext.jsx:39`):
- `requestVerification(email)` — `POST /api/keys {email}`, returns the
  backend's message. This is the "send me a magic link" call. Note the
  comment right above it in the source (lines 20-24): the backend used to
  return a key directly from this endpoint; now it only sends an email, and
  the key only becomes available once the link is clicked.
- `importKey(key)` — just `setApiKey(key)`. This is how a freshly-obtained
  key (pasted in after clicking the magic link, or typed in directly) enters
  the app's state.
- `logout()` — `setApiKey(null)`, which cascades through the effect above to
  clear storage and flip `isAuthenticated`.

Consumers: `Navbar` (`isAuthenticated`, `logout`), `ProtectedRoute`
(`isAuthenticated`), `LoginPage` (`requestVerification`, `importKey`),
`DashboardPage`, `StatsPage`, `AdminPage` (all just read `apiKey`, to key
their `useEffect` fetches).

**ToastContext** (`frontend/src/context/ToastContext.jsx`) holds one array:

```js
const [toasts, setToasts] = useState([]);
```
with `showToast(message, type = 'info', duration = 4000)` appending a
`{id, message, type}` object and scheduling its own removal via
`setTimeout` (lines 8-15), and `removeToast(id)` for the manual dismiss (the
`X` button in `Toast.jsx`). Nothing else holds a reference to a toast after
it's created — no queue limit, no dedup. Every page that can fail a network
call calls `showToast` from its `catch` block; `Toast.jsx` is the only
consumer that reads the `toasts` array itself, rendering one animated card
per entry.

What actually triggers a change in each context:
- **AuthContext** changes on: a successful `importKey` call (either from
  pasting a magic-link key, or — see Chapter 19 — any flow that produces a
  key), and on `logout()` (Navbar's logout button) or the implicit logout
  caused by a 401 reload (Chapter 17).
- **ToastContext** changes on: any `catch` block across five pages/components,
  three explicit success toasts (`ShortenForm` on shorten, `LoginPage` on
  verification sent, `LinkTable`/`ShortenForm` on copy-to-clipboard), and the
  self-expiring `setTimeout` inside `showToast` itself.

**★ interview tip**
> *"Why does `AuthContext` need both `apiKey` and `isAuthenticated` if one is
> derivable from the other?"*
> It doesn't strictly need both — `isAuthenticated` is just `!!apiKey` kept
> as a mirrored boolean via `useEffect` rather than computed at each call
> site. It's a readability/consistency choice (every consumer that only
> cares about "logged in or not" doesn't have to know the field is a key),
> not a correctness requirement. It does cost one extra render per `apiKey`
> change, since the effect fires after the state update.

---

## 19. The full user journey

Picture someone — call her Priya — who has never seen this app before. She
has a link she wants to shrink and hand to a client. Here's what happens,
file by file.

She opens the root URL. React mounts `main.jsx`'s provider stack
(`frontend/src/main.jsx:11-17`): `ToastProvider` first, `AuthProvider`
second — and `AuthProvider`'s very first act, before anything renders, is to
check `localStorage.getItem('urlshortener_api_key')` (`context/AuthContext.jsx:7`).
Priya's never been here, so that's `null`, and `isAuthenticated` starts
`false`. `App.jsx` routes `"/"` to `LandingPage` (`App.jsx:19`). On mount,
`LandingPage` fires `GET /api/stats/top?limit=100` through `apiClient`
(`pages/LandingPage.jsx:24-25`) — note this call carries no `Authorization`
header, because the request interceptor only *adds* one if `localStorage`
has a key (`api/client.js:12-14`), and public aggregate stats don't need
one. The response feeds two `StatCard`s showing "Links Shortened" and "Total
Clicks Tracked" (`pages/LandingPage.jsx:149-167`) — social proof, not
anything Priya needs yet.

What Priya actually wants is a login. She clicks "Login" in the `Navbar`
(rendered because `isAuthenticated` is false, so the nav shows the
"Login" link rather than a logout button — `components/Navbar.jsx:128-141`),
landing on `LoginPage`. She types her email and submits the first form. That
calls `handleRequestVerification` (`pages/LoginPage.jsx:21-35`), which
awaits `requestVerification(email)` from `AuthContext` — which is just
`apiClient.post('/api/keys', { email })` (`context/AuthContext.jsx:26`). The
backend's response here is deliberately just a message, not a key (per the
comment at `AuthContext.jsx:20-24`) — it has gone off to send her a magic
link instead. `LoginPage` flips `verificationSent` to `true`
(`pages/LoginPage.jsx:28`), which swaps the form out for a "Check your
email" card (`pages/LoginPage.jsx:64-108`), and a success toast fires
(`'Verification email sent!'`, line 29).

In dev, that email landed in MailHog, and the UI is honest about it — the
confirmation card literally links to `localhost:8025`
(`pages/LoginPage.jsx:88-93`) so Priya doesn't have to go hunting for it. She
opens MailHog, finds the magic link, and clicks it. That GET request is
outside the SPA entirely — it hits the backend directly (`GET
/api/keys/verify?...` per the `AuthContext` comment), which validates the
15-minute token, generates an API key of the form `usk_<64 hex>`, and shows
it to her — once, as the codebase guide puts it; only its SHA-256 hash
persists server-side. That key never lives in the React app yet; Priya has
to carry it back herself.

She goes back to the login tab and pastes the key into the second form —
"Already have a key?" (`pages/LoginPage.jsx:183-205`). Submitting calls
`handleImportKey` (lines 37-46), which runs the key through
`isValidApiKey` — a plain regex, `/^usk_[a-f0-9]{64}$/`
(`utils/format.js:40-42`) — purely client-side, no network round trip to
check it's real. If it matches the shape, `importKey(existingKey)` is
called, which is `AuthContext`'s `setApiKey` (`context/AuthContext.jsx:30-32`).
That state change triggers the `useEffect` at `AuthContext.jsx:10-18`, which
writes the key into `localStorage` under `urlshortener_api_key` and flips
`isAuthenticated` to `true`. `LoginPage` then calls `navigate('/dashboard')`
(`pages/LoginPage.jsx:45`). Notice: the app never actually confirms the key
is *valid* against the server at this moment — the first real test of the
key is whatever `DashboardPage` fetches next.

`DashboardPage` mounts, sees a non-null `apiKey` from `useAuth()`, and its
effect fires `fetchLinks()` → `GET /api/links`
(`pages/DashboardPage.jsx:30-46`) — this request does carry
`Authorization: Bearer usk_...` now, attached invisibly by the request
interceptor. If the key were bogus, this is where a 401 would come back and
the response interceptor would bounce her straight to `/login`
(`api/client.js:27-31`). It isn't bogus, so she gets an (empty, for now) list
back, and `LinkTable` renders its empty state: "No links found. Shorten your
first URL!" (`components/LinkTable.jsx:19-29`).

She clicks "Shorten New URL" (`pages/DashboardPage.jsx:77-87`), which toggles
`showForm` and animates in a `ShortenForm`
(`pages/DashboardPage.jsx:90-105`). She pastes her long URL, optionally opens
"Advanced Options" to set a custom alias or expiry
(`components/ShortenForm.jsx:107-154`), and submits. `handleSubmit`
(`components/ShortenForm.jsx:19-44`) builds a body — `{url, custom_alias?,
expires_at?}` — and calls `apiClient.post('/api/shorten', body)`. The
response becomes `result`, rendered as a green "Your shortened URL" card with
a copy button (`components/ShortenForm.jsx:157-189`); `onSuccess` is also
invoked, which on the dashboard means `fetchLinks()` runs again and the form
collapses (`pages/DashboardPage.jsx:99-102`) — so the new link shows up in
the table underneath, not just in the ephemeral result card.

Now she wants to see how it's doing. Back on the dashboard, her new row in
`LinkTable` has a chart icon linking to `/stats/:shortCode`
(`components/LinkTable.jsx:91-99`). Following it mounts `StatsPage`, whose
effect fires two requests in parallel via `Promise.all`
(`pages/StatsPage.jsx:35-38`): `GET /api/stats/:shortCode` for aggregate
numbers, and `GET /api/links/:shortCode/clicks?limit=20` for the raw recent
hits. The aggregate response populates two `StatCard`s (total clicks, unique
visitors — `pages/StatsPage.jsx:102-103`), a `DailyChart` bar chart fed by
`stats.daily_clicks` (`pages/StatsPage.jsx:106-109`, chart itself in
`components/DailyChart.jsx`), and a referrers list built inline right there
in `StatsPage` (lines 117-138). The per-click response fills the "Recent
Clicks" table, also inline in the same file (lines 141-183), each row
showing a formatted timestamp (`formatDate`, `utils/format.js:1-11`),
country, and a device icon swapped by type
(`pages/StatsPage.jsx:50-56`). Right now, with zero clicks, both are empty —
"No referrer data available," "No recent clicks."

So she copies the short link (the copy button in `ShortenForm`'s result card,
or from `LinkTable` — both just call `navigator.clipboard.writeText` and fire
a success toast, `components/ShortenForm.jsx:46-51` /
`components/LinkTable.jsx:12-17`) and sends it to her client. The client — a
third party who never touches this React app at all — clicks the link in
their browser. That request goes straight to the Go backend's redirect
handler; nothing in the SPA is involved (see Chapter 20, and Part 2's
Chapter 9 for the server-side mechanics of that hit).

Some time later, Priya comes back, revisits `/stats/that-code`, and
`StatsPage`'s effect runs again (it re-fires whenever `shortCode` or `apiKey`
change — `pages/StatsPage.jsx:48` — and a fresh mount counts as that). The
same two requests go out, but now `total_clicks` is 1, `unique_ips` is 1, the
`DailyChart` has a bar, and the "Recent Clicks" table has her client's hit —
country and device type included, assuming the backend's click pipeline
(Part 2) finished processing it by the time she checked. Nothing pushed that
update to her proactively — there's no polling or websocket here, just a
fresh fetch on each visit.

**★ interview tip**
> *"Where exactly does the frontend ever see or persist the raw API key,
> beyond pasting it into the import form?"*
> Exactly once per login: as the string the user manually copies from the
> magic-link landing page and pastes into `LoginPage`'s second form
> (`pages/LoginPage.jsx:188-195`). From there it's `importKey` →
   `setApiKey` → the `AuthContext` effect writes it to
   `localStorage.urlshortener_api_key`
   (`context/AuthContext.jsx:12`), and the axios request interceptor reads
   it back out of storage on every subsequent call
   (`api/client.js:12`). It is never kept purely in memory-only state in a
   way that would survive a key *not* being in localStorage — refreshing the
   page re-derives `isAuthenticated` from storage, not from any server
   session.

---

## 20. What happens when the third party clicks the link — from the browser's side

Zoom out from the React app entirely for a moment, because the person who
clicks Priya's short link never loads this SPA at all. From their browser's
point of view, the sequence is almost comically plain:

1. **DNS/TLS** — the browser resolves whatever domain the short link is
   hosted on and negotiates TLS if it's HTTPS. This is entirely outside this
   application's control; the Go binary doesn't participate in either step
   beyond possibly sitting behind a reverse proxy that handles TLS
   termination (see the deployment docs for how nginx fits into that, where
   applicable).
2. **The GET request** — the browser sends a plain `GET /{shortCode}` to
   whatever host answers. No cookies, no special headers, nothing the SPA
   set up — this is just a normal top-level navigation, indistinguishable
   from typing a URL directly.
3. **The 302 response** — the server replies with an HTTP redirect
   (`Location: <long url>`) rather than any HTML page. The browser never
   renders anything from this app's frontend bundle for this request —
   there's no `index.html`, no JS, no React involved at any point in a short
   link visit.
4. **The browser's own redirect behavior** — on receiving the 3xx, the
   browser transparently issues a second request to the `Location` target
   and navigates there. The user sees only the destination site load; the
   short-link hop is invisible unless they're watching the network tab or
   the URL bar flickers momentarily.

What the user never sees behind that redirect is everything Part 2's
Chapter 9 already walked through in detail: the Redis cache lookup keyed on
the short code (cache hit returning the `{url, id}` JSON immediately, or a
Postgres fallback on miss with a cache backfill), and the fire-and-forget
push of a click event onto the Redis Stream for the analytics worker to pick
up later. None of that touches the response the browser gets — the 302 goes
out either way, and the click-event write happens asynchronously, off the
hot path. That asynchronous write is exactly what turns into the "Recent
Clicks" row and the incremented `total_clicks` that Priya saw when she
checked back in Chapter 19 — the browser's redirect and the dashboard's
updated numbers are two completely decoupled consequences of the same single
GET request, joined only by the click-event stream in between.

**★ interview tip**
> *"If the click-event write to the Redis Stream fails, does the visitor's
> redirect fail too?"*
> No — by construction they can't be coupled. The redirect response is
> already determined by the cache/DB lookup before the click event is
> pushed, and the push itself is fire-and-forget. A failed or delayed event
> write only affects whether that click ever shows up in `StatsPage`'s
> numbers; it has zero effect on whether the third party's browser gets
> redirected. That's the whole point of decoupling analytics from the
   redirect hot path (Part 2, Chapter 9).
