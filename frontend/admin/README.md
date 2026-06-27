# Marshal — Admin Dispatch Console

Real-time dispatch dashboard for campus ride coordinators. Shows the live request queue, formed groups with route-score rankings, and driver availability — all updating instantly over SSE without a page refresh.

**Live:** `https://marshal-admin.vercel.app` (Vercel)

---

## Screenshot

![Admin Dispatch Console](../../docs/screenshots/admin-dashboard.png)
*Empty-state dashboard — 4 stat tiles, Live Queue, Groups Board (ordered by route score), Drivers panel*


---

## What This Portal Does

The Dispatch Console is a read-mostly admin view with three columns:

**Live Queue** — every pending `ride_request` that hasn't yet been grouped. New rows appear the moment a student submits.

**Groups Board** — formed `ride_groups` ordered by `route_score` descending. Each card shows member count, confidence score bar, and current status badge (`grouped` / `dispatching` / `assigned`). Updates live as the assigner advances groups through states.

**Drivers** — registered drivers and their online/offline status. The `Drivers online` stat tile at the top reflects this count.

Stats bar: groups formed today, drivers online, average route score, live WebSocket/SSE connections.

---

## Stack

- **Vite 6** + **React 19** + **Tailwind CSS 4**
- SSE via `useEventSource` hook (`/events` endpoint) — live event stream from the backend hub
- REST via `useApi` hook — polling on mount + triggered refetches on SSE events
- React Router v7 (SPA, single route)
- Lucide React icons

---

## Environment Variables

| Variable | Default (dev) | Description |
|---|---|---|
| `VITE_API_URL` | `http://localhost:8080` | Backend API base URL |
| `PORT` | `5174` | Dev server port (avoids conflict with student app on 5173) |

Copy `.env.example` to `.env.local` for local development. In Vercel, set `VITE_API_URL` to the production Render URL.

---

## Local Setup

```bash
cd frontend/admin
cp .env.example .env.local
npm install
npm run dev     # → http://localhost:5174
```

The backend must be running on `VITE_API_URL` (default `localhost:8080`). See the [backend README](../../backend/README.md) for how to start it.

```bash
npm run build   # production build → dist/
npm run lint    # ESLint
```

---

## Deployment

Deployed to Vercel via Git push. Build settings in Vercel dashboard:

- **Framework preset:** Vite
- **Build command:** `npm run build`
- **Output directory:** `dist`
- **Root directory:** `frontend/admin`
- **Environment variable:** `VITE_API_URL=https://marshal-api.onrender.com`

---

## Architecture Notes

The admin app uses **SSE** (Server-Sent Events) rather than WebSocket because it is read-only — it only needs to receive pushes, never send. This keeps the connection lightweight and works seamlessly behind Vercel's edge network.

The `useEventSource` hook reconnects automatically on drop. The `useApi` hook wraps `fetch` with base URL injection and JSON parsing. Both are in `src/hooks/`.
