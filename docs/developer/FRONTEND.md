# 🎨 Frontend Developer Guide

> **Purpose:** Technical reference for the React/Vite frontend (LinkSnip). Covers pages, components, API client, auth flow, and how it connects to the backend.

---

## Stack

| Layer | Tech |
|-------|------|
| Framework | React 18 |
| Build Tool | Vite 5 |
| Styling | Tailwind CSS 3 (custom design system) |
| Animation | Framer Motion (3D transforms, spring physics, AnimatePresence) |
| Routing | React Router |
| HTTP | Axios |
| Charts | Recharts |
| Icons | Lucide |
| State | React Context (Auth + Toast) |

---

## Design System (UI Overhaul)

The frontend uses a **glassmorphism + 3D animation** design system built on Tailwind CSS and Framer Motion, with **Google Material Design** inputs and buttons for a clean, professional look.

### Core Visual Language
- **Google Material inputs/buttons**: Solid flat fills, outlined borders (`1px solid #80868b`), focus state goes to solid `2px solid #1a73e8` (Google blue). No gradients on interactive elements.
- **Glassmorphism cards**: `backdrop-blur-xl` + translucent backgrounds (`rgba(255,255,255,0.7)`) — see `.glass`, `.card` classes in `src/index.css`
- **Animated gradient text**: `.text-gradient` / `.text-gradient-animated` — sky → indigo → purple gradient clipped to text, with animated background position on the hero headline only
- **3D perspective hover**: `StatCard` and feature cards use `rotateX`/`rotateY` via Framer Motion's `useMotionValue` + `useSpring` for realistic tilt-on-cursor
- **Glow effects**: Custom `boxShadow` tokens (`shadow-glow-brand`, `shadow-glow-purple`, `shadow-glow-green`) defined in `tailwind.config.js`
- **Floating background orbs**: Animated blurred radial gradients that drift on the landing and login pages

### Input & Button Design (Google Material Style)
- **Inputs**: Outlined by default — thin `1px solid #80868b` border, solid white/dark background. On focus: border becomes `2px solid #1a73e8` (Google blue), padding compensates for thicker border. Leading icon transitions color on focus. No glow, no gradient, no rainbow border.
- **Primary buttons**: Solid `#1a73e8` fill (no gradient), subtle elevation shadow on hover, darker on active (`#1557b0`), no shimmer sweep
- **Secondary buttons**: Tinted background (`rgba(26,115,232,0.08)`), no border, subtle elevation on hover, darker tint on active

### Color Palette
- **Primary action**: `#1a73e8` (Google blue) for buttons, input focus borders, active states
- **Brand gradient**: `#0ea5e9 → #8b5cf6` (sky → purple) — used only for text gradient accents and decorative elements (NOT on inputs/buttons)
- **Glass surfaces**: `rgba(255,255,255,0.7)` light, `rgba(31,41,55,0.6)` dark
- **Status colors**: green (success), red (error), blue (info), orange (warning)

### Component Classes (`src/index.css`)
| Class | Purpose |
|-------|---------|
| `.btn-primary` | Google Material solid fill button (`#1a73e8`), elevation shadow on hover, no gradient |
| `.btn-secondary` | Tinted background button (`rgba(26,115,232,0.08)`), no border, subtle elevation |
| `.input-field` | Google Material outlined input (`1px solid #80868b` → `2px solid #1a73e8` on focus), solid background |
| `.card` | Glassmorphic surface with hover lift + shadow bloom |
| `.card-flush` | Card variant with no padding (tables) |
| `.text-gradient-animated` | Animated gradient clipped to text (hero headline only) |
| `.glass` | Pure frosted glass utility |
| `.shimmer-surface` | Sliding highlight sweep for badges/medals |
| `.perspective` / `.preserve-3d` | 3D transform helpers for Framer Motion tilt |

### Custom Animations (`tailwind.config.js`)
17 keyframe animations including: `fade-in`, `slide-up`, `scale-in`, `float`, `pulse-slow`, `gradient-text`, `glow-pulse`, `shimmer`, `gradient-border`, `tilt-in`, `card-entrance`, `heartbeat`.

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

| Component | Purpose | Animation Notes |
|-----------|---------|------------------|
| `Navbar` | Top navigation (logo, links, dark mode toggle, logout) | Glassmorphic, spring slide-in, animated active pill (`layoutId`), rotating theme toggle, `AnimatePresence` mobile menu with staggered items |
| `Footer` | Bottom footer | Gradient top border, pulsing heart icon, spring hover lift on social icons |
| `ShortenForm` | URL input + shorten button + result display | Google Material outlined input with focus color shift, `AnimatePresence` advanced options + result card |
| `LinkTable` | Table of user's links (short URL, original, created, actions) | Staggered row entrance (`delay: index * 0.05`), hover highlight, spring scale on action buttons |
| `StatCard` | Big number display (total clicks, unique visitors) | **3D tilt-on-cursor** (`useMotionValue` + `rotateX/rotateY`), glowing icon container, shimmer sweep on hover |
| `DailyChart` | Bar chart of daily clicks (Recharts) | Gradient-filled bars with SVG glow filter, animated entrance, glass tooltip |
| `Spinner` | Loading indicator | Rotating gradient ring with pulsing glow halo |
| `Toast` | Success/error notifications | `AnimatePresence` slide+scale enter/exit, glassmorphic, auto-dismiss progress bar |
| `ProtectedRoute` | Redirects to /login if no API key | (Logic only — no visual layer) |

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