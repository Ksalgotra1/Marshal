# Marshal — ER Diagram

Entity-relationship diagram derived directly from `backend/migrations/001_create_core_schema.up.sql`.

```mermaid
erDiagram
    drivers {
        UUID id PK
        TEXT name
        BIGINT telegram_id UK
        BIGINT telegram_chat
        TEXT status
        TIMESTAMPTZ last_seen_at
        TIMESTAMPTZ created_at
    }

    ride_requests {
        UUID id PK
        TEXT requester_name
        DOUBLE_PRECISION pickup_lat
        DOUBLE_PRECISION pickup_lng
        DOUBLE_PRECISION dropoff_lat
        DOUBLE_PRECISION dropoff_lng
        BIGINT pickup_h3
        BIGINT dropoff_h3
        TIMESTAMPTZ arrive_by
        TEXT status
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    ride_groups {
        UUID id PK
        TEXT status
        TEXT priority
        DOUBLE_PRECISION route_score
        TIMESTAMPTZ arrive_by
        TIMESTAMPTZ expected_departure
        UUID driver_id FK
        INT dispatch_attempts
        INT telegram_msg_id
        TIMESTAMPTZ dispatch_timeout_at
        TIMESTAMPTZ completed_at
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    group_members {
        UUID request_id FK
        UUID group_id FK
        TIMESTAMPTZ joined_at
    }

    jobs {
        UUID id PK
        TEXT job_type
        JSONB payload
        TEXT status
        TEXT priority
        INT attempts
        INT max_attempts
        TIMESTAMPTZ run_after
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    chat_messages {
        UUID id PK
        UUID group_id FK
        TEXT sender_type
        TEXT sender_name
        TEXT content
        TIMESTAMPTZ created_at
    }

    drivers         ||--o{ ride_groups    : "assigned to"
    ride_requests   ||--o{ group_members  : "member of"
    ride_groups     ||--o{ group_members  : "contains"
    ride_groups     ||--o{ chat_messages  : "has"
```

---

### Key Indexes

| Index | Purpose |
|---|---|
| `idx_requests_status` | Grouper scans `WHERE status = 'pending'` |
| `idx_requests_status_h3` | Grouper H3 cell bucketing |
| `idx_groups_priority_queue` | Assigner max-heap: `(priority_rank, route_score DESC)` WHERE unassigned |
| `idx_jobs_queue` | Worker pull: `run_after ASC` WHERE queued |
| `idx_chat_group_id` | Chat message fetch per group |

---

<!--
EXCALIDRAW FALLBACK
===================
To convert this to an Excalidraw diagram:

1. Open https://app.excalidraw.com
2. Create a new drawing
3. Recreate the tables as rectangles with the fields listed above
4. Draw relationship lines between:
   - drivers.id → ride_groups.driver_id (1:many)
   - ride_requests.id → group_members.request_id (1:many)
   - ride_groups.id → group_members.group_id (1:many)
   - ride_groups.id → chat_messages.group_id (1:many)
5. Use colour #001E2B for table backgrounds, #00ED64 for PK fields
6. Export as PNG → save to docs/diagrams/er-diagram.png
7. Export as .excalidraw → save to docs/diagrams/er-diagram.excalidraw

Then replace the Mermaid block above with:
![ER Diagram](diagrams/er-diagram.png)
-->
