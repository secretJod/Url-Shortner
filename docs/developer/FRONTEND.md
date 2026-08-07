# 🎨 Frontend Developer Guide

> **Purpose:** Technical reference for the React/Vite frontend (LinkSnip). Covers pages, components, API client, auth flow, and how it connects to the backend.

---

## Stack

| Layer | Tech |
|-------|------|
| Framework | React 18 |
| Build Tool | Vite 5 |
| Styling | Tailwind CSS |
| Routing | React Router |
| HTTP | Axios |
| Charts | Recharts |
| Icons | Lucide |
| State | React Context (Auth + Toast) |

---

## Pages (6)

| Route | Page | Auth Required | What It Does |
|-------|------|---------------|--------------|
| `/` | LandingPage | No | Hero + URL shortening form |
| `/login` | LoginPage | No | Get API key with email |
| `/dashboard` | DashboardPage | Yes | User's links table |
| `/stats/:shortCode` | StatsPage | Yes | Link analytics + charts |
| `/top` | TopLinksPage | No | Most-clicked links |
| `/admin` | AdminPage | No | App metrics dashboard |

---

## Components

| Component | Purpose |
|-----------|---------|
| `Navbar` | Top navigation (logo, links, dark mode toggle, logout) |
| `Footer` | Bottom footer |
| `ShortenForm` | URL input + shorten button + result display |
| `LinkTable` | Table of user's links (short URL, original, created, actions) |
| `StatCard` | Big number display (total clicks, unique visitors) |
| `DailyChart` | Bar chart of daily clicks (Recharts) |
| `Spinner` | Loading indicator |
| `Toast` | Success/error notifications |
| `ProtectedRoute` | Redirects to /login if no API key |

---

## API Client (`src/api/client.js`)

- Base URL: `import.meta.env.VITE_API_URL || ''` (empty = same origin, proxied by Vite)
- **Request interceptor**: adds `Authorization: Bearer <api_key>` from localStorage
- **Response interceptor**: 
  - 401 → clears key, redirects to /login
  - 429 → shows "Too many requests" toast
  - 500 → shows error toast

---

## Auth Flow

1. User enters email on `/login`
2. Frontend calls `POST /api/keys` with `{"email": "..."}`
3. Backend returns `{"api_key": "usk_..."}` — shown ONCE
4. Frontend stores key in `localStorage` under `urlshortener_api_key`
5. All subsequent API calls include `Authorization: Bearer usk_...`
6. `useAuth` hook checks localStorage for key
7. `ProtectedRoute` redirects to /login if no key

---

## How Frontend Connects to Backend

### Dev Mode (Two Servers)

```
Browser → http://localhost:5173 (Vite dev server)
              ↓ proxy /api → http://localhost:8080 (Go API)
```

Vite proxy config (`vite.config.js`):
```js
server: {
  proxy: {
    '/api': { target: 'http://localhost:8080', changeOrigin: true },
    '/health': { target: 'http://localhost:8080', changeOrigin: true },
    '/metrics': { target: 'http://localhost:8080', changeOrigin: true }
  }
}
```

### Prod Mode (One Server)

```
Browser → http://localhost:8080 (Go API serves frontend/dist)
              ↓ /api/* → Go API handlers
              ↓ /* → frontend/dist/index.html (SPA fallback)
```

---

## Short Link URLs

**Important:** Short links must point to the **backend** (`:8080`), NOT the frontend (`:5173`).

```js
// Correct — points to backend redirector
const shortUrl = `${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/${link.short_code}`;
```

This is used in `LinkTable.jsx` and `TopLinksPage.jsx`.

---

## Key Files

| File | Purpose |
|------|---------|
| `src/main.jsx` | Entry point — mounts React, wraps with Router + Providers |
| `src/App.jsx` | Route definitions |
| `src/api/client.js` | Axios instance with auth interceptors |
| `src/context/AuthContext.jsx` | Auth state (apiKey, login, logout) |
| `src/context/ToastContext.jsx` | Toast notification state |
| `src/hooks/useAuth.js` | Hook to access auth context |
| `src/hooks/useToast.js` | Hook to show toasts |
| `src/components/*.jsx` | Reusable UI components |
| `src/pages/*.jsx` | Page components |
| `src/utils/format.js` | Date/URL formatting helpers |

---

## How to Run

```bash
cd frontend
npm install
npm run dev
```

Dev server runs at `http://localhost:5173`

---

## Build for Production

```bash
cd frontend
npm run build
```

Output goes to `frontend/dist/` — served by the Go backend at `http://localhost:8080`.