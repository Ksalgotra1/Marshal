# Marshal — Backend

Go HTTP server that powers the Marshal ride-grouping and dispatch engine. No framework — stdlib `net/http` only.

**Live:** `https://marshal-api.onrender.com` (Render, Docker)

---

## Architecture

```
                      ┌─────────────────────────────────────────────┐
                      │                Go HTTP Server                 │
                      │                                               │
  Student/Admin ──►   │  Handlers  ──►  Store (pgx)  ──►  Postgres  │
  (REST + WS + SSE)   │                                               │
                      │  Grouper Worker                               │
                      │    ├── pending requests → H3 cells (res 9)   │
                      │    ├── 3-pass spatial match                   │
                      │    ├── score() → rerank()                    │
                      │    └── INSERT ride_groups                     │
                      │                                               │
                      │  Assigner Worker                              │
                      │    ├── SELECT ... FOR UPDATE SKIP LOCKED      │
                      │    ├── LISTEN/NOTIFY wakeup                   │
                      │    ├── Telegram dispatch (inline keyboard)    │
                      │    └── 2-min pooling-delay re-enqueue         │
                      │                                               │
                      │  Realtime Hub                                 │
                      │    ├── Rooms (global + per-group)             │
                      │    ├── WebSocket clients (gorilla/websocket)  │
                      │    └── SSE clients                            │
                      │                                               │
                      │  Telegram Bot                                 │
                      │    ├── Webhook (prod) / polling (dev)         │
                      │    ├── Inline Accept/Pass keyboard            │
                      │    └── Chat relay (student ↔ driver)         │
                      └─────────────────────────────────────────────┘
```


### Grouping Algorithm

```
pending ride_requests
        │
        ▼
  H3 bucketing (resolution 9, ~0.1 km² cells)
        │
        ├── Pass 1: exact H3 cell match
        ├── Pass 2: k-ring 1 neighbour cells
        └── Pass 3: relaxed (wider neighbourhood)
        │
        ▼
  score() per candidate group
    ├── pickup spread    (mean pairwise km, std dev)
    ├── dropoff spread   (mean pairwise km, std dev)
    └── time-window overlap (std dev of arrive_by minutes)
        │
        ▼
  rerank() — composite score → route_score column
        │
        ▼
  INSERT INTO ride_groups + NOTIFY assigner_wakeup
```


### Job Queue / Worker Architecture

```
  ┌──────────┐   Enqueue()    ┌────────────┐
  │ Grouper  │ ─────────────► │  jobs      │  status: queued
  │ Assigner │               │  table     │  FOR UPDATE SKIP LOCKED
  └──────────┘               └─────┬──────┘
                                   │
                    LISTEN/NOTIFY  │  or 30s ticker fallback
                                   ▼
                            ┌────────────┐
                            │  Worker    │  ProcessFunc(ctx, pool, payload)
                            │  goroutine │
                            └────────────┘
```


### Realtime Fan-out

```
                      ┌──────────────────────┐
                      │     Realtime Hub      │
                      │                       │
  backend event  ──►  │  Rooms                │
                      │    ├── "global"       │──► Admin SSE clients
                      │    └── "group:{id}"   │──► Student WS clients
                      └──────────────────────┘
```

Events: `group:formed`, `group:assigned`, `request:updated`, `chat:message`


---

## Tech Stack

| | |
|---|---|
| Language | Go 1.25 |
| HTTP | stdlib `net/http` |
| Database driver | `jackc/pgx/v5` |
| WebSocket | `gorilla/websocket` |
| Spatial indexing | `uber/h3-go/v4` (resolution 9) |
| Rate limiting | `golang.org/x/time/rate` (token bucket) |
| UUIDs | `google/uuid` |
| Migrations | `golang-migrate` (run at startup) |
| Containerisation | Docker (Render) |

---

## API Reference

Base URL: `https://marshal-api.onrender.com`

All JSON endpoints require `Content-Type: application/json` on POST bodies. Every response includes `X-Request-ID` for tracing.

### Ride Requests

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/requests` | — | List all ride requests |
| `POST` | `/api/requests` | — | Create a new request (rate-limited) |
| `GET` | `/api/requests/{id}` | — | Get a single request by UUID |

**POST /api/requests body:**
```json
{
  "requester_name": "Alice",
  "pickup_lat": 30.7046,
  "pickup_lng": 76.7179,
  "dropoff_lat": 30.7333,
  "dropoff_lng": 76.7794,
  "arrive_by": "2026-06-27T15:30:00Z"
}
```

### Ride Groups

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/groups` | — | List all groups (admin view) |
| `GET` | `/api/groups/open` | — | List groups open for student join |
| `GET` | `/api/groups/{id}` | — | Get group detail + members |
| `POST` | `/api/groups/{id}/join` | — | Student joins an open group (rate-limited) |
| `POST` | `/api/groups/{id}/claim` | — | Internal: mark group claimed (rate-limited) |
| `GET` | `/api/groups/{id}/messages` | — | List chat messages for a group |
| `POST` | `/api/groups/{id}/messages` | — | Send a chat message (rate-limited) |

### Drivers

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/drivers` | `X-Admin-Key` | Register a driver (admin only) |
| `GET` | `/api/drivers` | — | List drivers + online status |

### Realtime

| Method | Path | Description |
|---|---|---|
| `GET` | `/ws` | WebSocket upgrade (student live updates) |
| `GET` | `/events` | SSE stream (admin dashboard) |

### Telegram

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/telegram/webhook` | `X-Telegram-Bot-Api-Secret-Token` | Telegram webhook receiver |

### Health

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Returns `{"status":"ok","connections":N}` |

---

## Environment Variables

Copy `.env.example` to `.env`:

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | ✓ | — | Postgres connection string |
| `PORT` | — | `8080` | HTTP listen port |
| `ALLOWED_ORIGIN` | — | `http://localhost:5173` | CORS allowlist (comma-separated) |
| `LOG_FORMAT` | — | `text` | `text` or `json` |
| `TELEGRAM_BOT_TOKEN` | ✓ (for Telegram) | — | Bot token from @BotFather |
| `TELEGRAM_DRIVER_GROUP_ID` | ✓ (for Telegram) | — | Telegram group chat ID for drivers |
| `TELEGRAM_WEBHOOK_URL` | — | — | Public HTTPS URL; leave blank for polling |
| `TELEGRAM_WEBHOOK_SECRET` | — | — | Shared secret for webhook validation |
| `ADMIN_API_KEY` | ✓ (for driver registration) | — | Header key for `POST /api/drivers`. Unset = endpoint disabled (fail-closed) |
| `DRIVER_PRESENCE_TTL` | — | `300` | Seconds before a driver goes offline |

---

## Local Setup

**Prerequisites:** Go 1.25, Docker, `golang-migrate`

```bash
# 1. Start Postgres
docker-compose -f ../docker-compose.yaml up -d

# 2. Configure
cp .env.example .env
# edit .env — fill TELEGRAM_BOT_TOKEN, TELEGRAM_DRIVER_GROUP_ID, ADMIN_API_KEY

# 3. Run migrations
make migrate-up

# 4. Start the server
make run
# → http://localhost:8080
```

For local Telegram testing, leave `TELEGRAM_WEBHOOK_URL` blank — the bot will fall back to long-polling automatically.

To expose a local webhook: `ngrok http 8080`, then set `TELEGRAM_WEBHOOK_URL=https://<id>.ngrok.io/telegram/webhook`.

---

## Testing

```bash
make test-unit          # pure unit tests — no DB, no network
make test-integration   # integration tests (Postgres required)
make test-all           # both
make check-coverage     # filtered coverage gate (threshold: 40%)
make cover              # generate HTML coverage report → coverage.html
```

The test suite is split into unit (`_unit_test.go`) and integration (`_integration_test.go`) files. Integration tests spin up a real Postgres connection via `internal/testdb`. The coverage gate excludes `cmd/`, `internal/api/`, and `internal/config/` (scaffolding) to keep the metric meaningful.

---

## Security Notes

- **`ADMIN_API_KEY`** — constant-time comparison (`crypto/subtle`) prevents timing attacks. If the env var is unset, the endpoint is `503 Service Unavailable` (fail-closed).
- **Telegram webhook secret** — every incoming update is validated against `X-Telegram-Bot-Api-Secret-Token` before processing.
- **Rate limiting** — per-IP token bucket on all mutation endpoints: 10 requests per 3 seconds burst, cleaned up every 5 minutes.
- **CORS** — strict origin allowlist via `ALLOWED_ORIGIN`; multiple origins via comma-separated values.
- **Security headers** — `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` applied globally.

---

## Deployment

Deployed on Render using Docker. The `Dockerfile` multi-stage build produces a minimal `gcr.io/distroless/base` image. `render.yaml` configures the service.

Migrations run automatically at startup via `golang-migrate` embedded against the managed Postgres instance.
