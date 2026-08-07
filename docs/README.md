# 📚 Documentation Index

> **Purpose:** This is the master index for ALL documentation in this project. Every doc is organized by audience so you always know where to look.

---

## 📖 For Non-Technical People (You)

| Doc | What It Covers |
|-----|----------------|
| [`KNOWLEDGE.md`](../KNOWLEDGE.md) | **The complete project explained in plain English** — what it is, how it works, every file, every feature. Read this if you've never coded before. |

---

## 🤖 For AI Agents (Claude, Qwen, GPT, etc.)

| Doc | What It Covers |
|-----|----------------|
| [`.ai/PROJECT_CONTEXT.md`](../.ai/PROJECT_CONTEXT.md) | **Read first** — full project context in 2 minutes |
| [`.ai/CODE_MAP.md`](../.ai/CODE_MAP.md) | Every source file explained |
| [`.ai/CONVENTIONS.md`](../.ai/CONVENTIONS.md) | Coding patterns & how to add features |
| [`.ai/GOTCHAS.md`](../.ai/GOTCHAS.md) | All known bugs & solutions |
| [`.ai/QUICKSTART.md`](../.ai/QUICKSTART.md) | 3-command setup guide |
| [`.ai/AGENT_INSTRUCTIONS.md`](../.ai/AGENT_INSTRUCTIONS.md) | Rules & restrictions for agents |
| [`.ai/NEXT_AGENT_PROMPT.md`](../.ai/NEXT_AGENT_PROMPT.md) | Copy-paste prompt for next agent |

---

## 👨‍💻 For Developers

| Doc | What It Covers |
|-----|----------------|
| [`developer/BACKEND.md`](developer/BACKEND.md) | Backend architecture, API endpoints, database, Redis, worker |
| [`developer/FRONTEND.md`](developer/FRONTEND.md) | Frontend pages, components, API client, auth flow |
| [`PROJECT_OVERVIEW.md`](../PROJECT_OVERVIEW.md) | Session log, phase tracking, architecture decisions |

---

## 🧑‍💼 For Users of the App

| Doc | What It Covers |
|-----|----------------|
| [`user/USER_GUIDE.md`](user/USER_GUIDE.md) | How to use the LinkSnip app (shorten links, view stats, get API key) |

---

## 📁 Project Structure (Quick Map)

```
Url-Shortner/
├── backend/          ← Go API (Fiber + Prisma + PostgreSQL + Redis)
├── frontend/         ← React/Vite UI (LinkSnip)
├── docs/             ← ← YOU ARE HERE — all documentation
├── .ai/              ← AI agent context files
├── KNOWLEDGE.md      ← Non-technical guide (plain English)
├── PROJECT_OVERVIEW.md ← Developer session log
├── QWEN_FRONTEND_PROMPT.md ← Prompt used to generate the frontend
└── docker-compose.yml
```

---

## 🚀 Quick Start (3 Commands)

```bash
# 1. Start backend
cd backend && go run ./cmd/api

# 2. Start frontend (in another terminal)
cd frontend && npm install && npm run dev

# 3. Open the app
open http://localhost:5173
```

---

## 📊 Current Phase Status

| Phase | Status | What |
|-------|--------|------|
| 0 | ✅ | Scaffold, Docker, Fiber, Redis |
| 1 | ✅ | Core shorten + redirect API |
| 2 | ✅ | API key auth + rate limiting |
| 3 | ✅ | Async analytics pipeline |
| 4 | ✅ | Admin/Stats API |
| 5 | ✅ | Observability (metrics) |
| 6 | ✅ | Frontend UI (LinkSnip) + backend wiring |
| **7** | **⬜ NEXT** | **Deployment (go live on the internet)** |