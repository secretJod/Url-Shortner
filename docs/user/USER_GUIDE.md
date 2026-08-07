# 🧑‍💼 LinkSnip User Guide

> **Purpose:** How to use the LinkSnip URL shortener app — for non-technical users.

---

## What is LinkSnip?

LinkSnip is a **URL shortener** — it takes a long, ugly web address and turns it into a short, clean one. Like Bitly or TinyURL.

---

## Getting Started

### 1. Open the App

Go to: **http://localhost:5173**

### 2. Shorten a URL

1. Paste a long URL into the input box (e.g., `https://www.example.com/very/long/path`)
2. Click the **Shorten** button
3. You'll see your short link (e.g., `http://localhost:8080/abc123`)
4. Click **Copy** to copy it to your clipboard

### 3. Use the Short Link

- Paste the short link in a browser → it redirects to the original long URL
- Share it in emails, texts, tweets, etc.

---

## Getting an API Key (Optional)

An API key lets you:
- See your links in the Dashboard
- View click statistics
- Get higher rate limits

### How to Get a Key

1. Go to **http://localhost:5173/login**
2. Enter your email
3. Click **Get API Key**
4. **IMPORTANT:** Copy the key immediately — it's shown only ONCE!
5. The key is saved in your browser automatically

---

## Dashboard

Go to **http://localhost:5173/dashboard** (requires API key)

Shows:
- All your shortened links
- Original URLs
- When they were created
- Actions: View Stats, Copy, Delete

---

## Viewing Stats

Click the **chart icon** next to any link to see:
- **Total clicks** — how many times the link was visited
- **Unique visitors** — how many different people clicked
- **Daily clicks chart** — clicks per day (bar chart)
- **Top referrers** — where visitors came from (Google, Twitter, etc.)
- **Recent clicks** — list of recent visits with device type

---

## Top Links

Go to **http://localhost:5173/top**

Shows the most-clicked links across the platform, ranked with medals 🥇🥈🥉

---

## Admin Metrics

Go to **http://localhost:5173/admin**

Shows app health:
- Uptime
- Total requests
- Average/max response time
- Cache hit rate
- Events processed

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| Page won't load | Make sure backend is running: `cd backend && go run ./cmd/api` |
| Short link shows blank page | Make sure you're on `:8080` not `:5173` — short links point to the backend |
| "Too many requests" | You've hit the rate limit (20/min anonymous, 60/min with key). Wait a minute. |
| Can't see dashboard | You need an API key — go to /login first |
| Google Drive error | `localhost` only works on YOUR computer. Short links won't work on other websites. |

---

## Rate Limits

| Who | Limit |
|-----|-------|
| Anonymous (no key) | 20 requests per minute |
| Standard API key | 60 requests per minute |
| Pro API key | 600 requests per minute |