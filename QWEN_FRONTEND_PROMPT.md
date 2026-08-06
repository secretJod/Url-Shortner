# 🎨 QWEN FRONTEND PROMPT — URL Shortener Enterprise Frontend

> **Copy this entire file and paste it to Qwen to generate the frontend.**

---

## Your Task

Build a **beautiful, enterprise-level frontend** for a URL Shortener application. The backend is a Go API (Fiber framework) with PostgreSQL + Redis. You need to create a complete React (or Next.js) frontend that connects to the backend API.

**IMPORTANT:** Create the frontend in a folder called `frontend/` so it can be dropped into the project root. Also create a `FRONTEND_README.md` file documenting every page, component, API call, and how the frontend connects to the backend.

---

## Backend API Endpoints (Use These)

The backend runs at `http://localhost:8080`. All API endpoints return JSON.

### 1. Health Check
```
GET /health
```
**Response:**
```json
{"redis":"ok","status":"ok"}
```

### 2. Create API Key (Auth)
```
POST /api/keys
Content-Type: application/json
Body: {"email": "user@example.com"}
```
**Response:**
```json
{
  "api_key": "usk_f82f9f017b04a174c4e94e1bb7513b9b7a4da233387f238a2039bcbc1f372cf9",
  "email": "user@example.com"
}
```
> ⚠️ The API key is shown **ONLY ONCE**. Store it in localStorage. All subsequent requests use it in the `Authorization: Bearer <api_key>` header.

### 3. Shorten a URL
```
POST /api/shorten
Content-Type: application/json
Authorization: Bearer <api_key> (optional — works without auth too)
Body: {"url": "https://www.example.com/very/long/path", "custom_alias": "mylink", "expires_at": "2026-12-31T23:59:59Z"}
```
**Response:**
```json
{
  "short_url": "http://localhost:8080/mylink",
  "short_code": "mylink",
  "long_url": "https://www.example.com/very/long/path"
}
```
> `custom_alias` and `expires_at` are optional. If no custom_alias, a random short code is generated.

### 4. Redirect (Visit Short Link)
```
GET /:shortCode
```
> This redirects the browser to the long URL. Not used by the frontend directly, but users will click short links.

### 5. Get Link Stats
```
GET /api/stats/:shortCode
```
**Response:**
```json
{
  "short_code": "mylink",
  "long_url": "https://www.example.com/very/long/path",
  "total_clicks": 42,
  "unique_ips": 35,
  "top_referrers": ["google.com", "twitter.com", "facebook.com"],
  "daily_clicks": [
    {"date": "2026-08-01", "count": 5},
    {"date": "2026-08-02", "count": 12},
    {"date": "2026-08-03", "count": 25}
  ]
}
```

### 6. Get Top Links
```
GET /api/stats/top?limit=10
```
**Response:**
```json
{
  "top_links": [
    {"short_code": "abc", "long_url": "https://example.com", "total_clicks": 100},
    {"short_code": "def", "long_url": "https://google.com", "total_clicks": 50}
  ]
}
```

### 7. Get User's Links (Requires Auth)
```
GET /api/links
Authorization: Bearer <api_key>
```
**Response:**
```json
{
  "links": [
    {
      "short_code": "abc",
      "long_url": "https://example.com",
      "custom_alias": true,
      "created_at": "2026-08-01T10:00:00Z"
    }
  ]
}
```

### 8. Get Recent Clicks for a Link
```
GET /api/links/:shortCode/clicks?limit=20
```
**Response:**
```json
{
  "clicks": [
    {
      "timestamp": "2026-08-04T16:02:29.037Z",
      "referrer": "https://google.com",
      "country": "US",
      "device_type": "mobile",
      "ip_hash": "12ca17b49af22894"
    }
  ]
}
```

### 9. Metrics (Admin)
```
GET /metrics
```
**Response:**
```json
{
  "uptime_seconds": 3600,
  "total_requests": 12345,
  "status_codes": {"200": 12000, "404": 300, "429": 45},
  "avg_latency_ms": 4.59,
  "max_latency_ms": 22.78,
  "cache_hits": 10000,
  "cache_misses": 2345,
  "cache_hit_rate": 81.0,
  "events_processed": 5000,
  "events_failed": 2
}
```

---

## Pages to Build

### 1. Landing Page (`/`)
- Hero section with app name and tagline
- URL shortening form (the main feature)
- Shows the shortened link with a "Copy" button
- Clean, modern design with gradient backgrounds
- Stats preview (total links shortened, total clicks)

### 2. Login / Signup Page (`/login`)
- Email input + "Get API Key" button
- On success, shows the API key **ONCE** with a warning "Save this key — you won't see it again"
- Stores the key in localStorage
- "Already have a key?" section to enter an existing key

### 3. Dashboard (`/dashboard`)
- **Requires auth** (redirect to /login if no API key)
- Shows user's links in a table:
  - Short URL (clickable)
  - Long URL (truncated)
  - Created date
  - Total clicks
  - Actions: View Stats, Copy, Delete
- "Shorten New URL" button
- Search/filter links

### 4. Link Stats Page (`/stats/:shortCode`)
- **Requires auth**
- Shows for a specific link:
  - Total clicks (big number)
  - Unique visitors (big number)
  - Daily clicks **bar chart** (use Chart.js or Recharts)
  - Top referrers (list with counts)
  - Recent clicks table (timestamp, referrer, country, device)
- Back to dashboard button

### 5. Top Links Page (`/top`)
- Shows the most-clicked links
- Ranked list with click counts
- Click to view stats

### 6. Admin Metrics Page (`/admin`)
- Shows the /metrics data:
  - Uptime
  - Total requests
  - Status code breakdown (pie chart)
  - Average/max latency
  - Cache hit rate (progress bar)
  - Events processed/failed

---

## Design Requirements

- **Modern, beautiful, enterprise-grade** UI
- Use **Tailwind CSS** (or styled-components if you prefer)
- **Dark mode** support (toggle)
- **Responsive** — works on mobile, tablet, desktop
- Smooth animations and transitions
- Loading spinners/skeletons while fetching data
- Toast notifications for success/error
- Clean error states (404, rate limit, invalid URL)
- Use **React Router** for navigation
- Use **Axios** or **fetch** for API calls
- Use **Chart.js** or **Recharts** for charts
- Icons: use **Lucide** or **Heroicons**

---

## Data Flow / State Management

- Use **React Context** or **Zustand** for auth state
- Store API key in localStorage under `urlshortener_api_key`
- Create an `api.js` utility that:
  - Has base URL `http://localhost:8080`
  - Automatically adds `Authorization: Bearer <key>` header if key exists
  - Handles 401 (redirect to login), 429 (show rate limit message), 500 (show error)
- Create a `useAuth` hook that:
  - Checks localStorage for API key
  - Provides `login(email)`, `logout()`, `isAuthenticated`

---

## Folder Structure (Create This)

```
frontend/
├── public/
│   └── index.html
├── src/
│   ├── App.jsx              (main app with routes)
│   ├── main.jsx             (entry point)
│   ├── index.css            (global styles)
│   ├── api/
│   │   └── client.js        (API utility with auth header)
│   ├── context/
│   │   └── AuthContext.jsx  (auth state management)
│   ├── hooks/
│   │   └── useAuth.js       (auth hook)
│   ├── components/
│   │   ├── Navbar.jsx       (top navigation)
│   │   ├── Footer.jsx
│   │   ├── ShortenForm.jsx  (URL input + button)
│   │   ├── LinkTable.jsx    (table of links)
│   │   ├── StatCard.jsx     (big number display)
│   │   ├── DailyChart.jsx   (bar chart)
│   │   ├── Toast.jsx        (notifications)
│   │   └── Spinner.jsx      (loading)
│   ├── pages/
│   │   ├── LandingPage.jsx  (hero + shorten form)
│   │   ├── LoginPage.jsx    (get API key)
│   │   ├── DashboardPage.jsx (user's links)
│   │   ├── StatsPage.jsx    (link analytics)
│   │   ├── TopLinksPage.jsx (most clicked)
│   │   └── AdminPage.jsx    (metrics)
│   └── utils/
│       └── format.js        (date formatting, URL helpers)
├── package.json
├── vite.config.js           (with proxy to localhost:8080)
└── FRONTEND_README.md       (document everything!)
```

---

## FRONTEND_README.md (MUST CREATE)

Create a `FRONTEND_README.md` file that documents:

1. **How to run the frontend** (npm install, npm run dev)
2. **Every page** and what it does
3. **Every API call** the frontend makes (endpoint, method, body, response)
4. **How auth works** (API key in localStorage, Authorization header)
5. **Environment variables** (VITE_API_URL)
6. **How to connect to the backend** (proxy config, CORS)
7. **Component tree** showing how components relate
8. **State management** explanation
9. **Any assumptions or decisions** you made

---

## Important Notes

- The backend does NOT have CORS enabled yet — use Vite's proxy in `vite.config.js`:
  ```js
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/metrics': 'http://localhost:8080'
    }
  }
  ```
- Use `VITE_API_URL` env variable (default: `http://localhost:8080`)
- The API key format is `usk_` followed by 64 hex characters
- Rate limits: anonymous = 20 req/min, standard = 60 req/min, pro = 600 req/min
- Handle 429 responses with a friendly "Too many requests, please wait" message

---

## Deliverables

1. Complete `frontend/` folder with all source code
2. `FRONTEND_README.md` documenting everything
3. Working `npm run dev` that starts the frontend
4. All pages functional and connected to the backend API

**Build it beautifully. This is an enterprise-grade URL shortener frontend.**

---

## How the Backend Connects (Already Prepped ✔)

The backend has ALREADY been wired up to serve the frontend. Here's what's ready:

### CORS Middleware (Added)
- Backend allows requests from `http://localhost:5173`, `http://localhost:3000`, `http://localhost:8080`
- Headers allowed: Origin, Content-Type, Accept, Authorization
- Methods allowed: GET, POST, PUT, DELETE, OPTIONS

### Static File Serving (Added)
- Backend serves `./frontend/dist` as static files
- When you run `npm run build`, the dist folder is served by the Go server at `http://localhost:8080`

### SPA Routes (Added)
- GET / → index.html
- GET /login → index.html
- GET /dashboard → index.html
- GET /stats/* → index.html
- GET /top → index.html
- GET /admin → index.html
- Unknown short codes → index.html (SPA fallback)

### Dockerfile (Updated)
- Added node:20-alpine frontend build stage
- Runs `npm run build` and copies dist to the runner container

### How to Test
1. Build frontend: `npm run build`
2. Start backend: `go run ./cmd/api`
3. Open `http://localhost:8080` → frontend loads

OR for dev mode (two servers):
1. Start backend: `go run ./cmd/api`  (port 8080)
2. Start frontend: `npm run dev`  (port 5173)
3. Vite proxies /api, /health, /metrics to localhost:8080

