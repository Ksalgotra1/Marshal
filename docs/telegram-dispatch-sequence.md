# Marshal — Telegram Dispatch Sequence

The flow from a formed group reaching the Telegram driver channel to the student seeing "Assigned" live.

```mermaid
sequenceDiagram
    participant A as Assigner Worker
    participant DB as PostgreSQL
    participant Bot as Telegram Bot
    participant TG as Telegram API
    participant DG as Driver Group Chat
    participant WH as Webhook Handler
    participant Hub as Realtime Hub
    participant SA as Student App (WS)
    participant AD as Admin Dashboard (SSE)

    A->>DB: SELECT ride_groups FOR UPDATE SKIP LOCKED<br/>(ordered by priority, route_score DESC)
    DB-->>A: grouped, unassigned group

    A->>Bot: SendGroupCard(groupID, members, pickup, dropoff)
    Bot->>TG: sendMessage(driver_group_chat_id)<br/>with InlineKeyboard [✅ Accept] [⏭ Pass]
    TG-->>DG: Group dispatch card appears

    Note over DG: Driver reviews route & taps Accept

    DG->>TG: callbackQuery {data: "accept:{groupID}"}
    TG->>WH: POST /telegram/webhook
    WH->>Bot: HandleCallback(accept, groupID, driverID)
    Bot->>DB: UPDATE ride_groups SET status='assigned', driver_id=?
    Bot->>DB: UPDATE drivers SET status='online', last_seen_at=now()

    Bot->>Hub: Publish(group:{groupID}, event:group:assigned)
    Hub->>SA: WS push → status stepper flips to Assigned
    Hub->>AD: SSE push → Groups Board card updates badge

    Bot->>TG: answerCallbackQuery("Assigned!")
    Bot->>TG: editMessageText (update card to show driver name)

    Note over SA: Student sees driver info, chat unlocks

    SA->>Bot: POST /api/groups/{id}/messages {content: "Hi!"}
    Bot->>TG: sendMessage(driver_group_chat_id, "Student: Hi!")
    TG-->>DG: Message appears for driver

    DG->>TG: Driver replies in group chat
    TG->>WH: POST /telegram/webhook (message update)
    WH->>DB: INSERT chat_messages
    WH->>Hub: Publish(group:{groupID}, event:chat:message)
    Hub->>SA: WS push → message appears in chat tab
```

### Telegram Driver Interface

These screenshots demonstrate the driver UI in Telegram for the dispatch sequence described above.

| Telegram Group Chat | Telegram Personal Chat |
|---|---|
| ![Telegram Group Chat](screenshots/telegram-group-chat.png) | ![Telegram Personal Chat](screenshots/telegram-personal-chat.png) |

---

<!-- EXCALIDRAW FALLBACK
===================
To convert this to an Excalidraw sequence diagram:

1. Open https://app.excalidraw.com
2. Create participant swimlanes (vertical columns) for:
   Assigner Worker | PostgreSQL | Telegram Bot | Telegram API | Driver Group | Webhook | Realtime Hub | Student App | Admin Dashboard
3. Draw arrows between them following the sequence above
4. Use dashed arrows for async events (SSE/WS pushes)
5. Add a dashed box labelled "Driver taps Accept" around the TG→DG interaction
6. Colour the realtime fan-out arrows (Hub→SA, Hub→AD) in #00ED64 to highlight the live update moment
7. Export as PNG → docs/diagrams/telegram-dispatch.png
8. Export as .excalidraw → docs/diagrams/telegram-dispatch.excalidraw

Then replace this file's Mermaid block with:
![Telegram Dispatch Sequence](diagrams/telegram-dispatch.png)


-->

---

### Pass Flow

If the driver taps **Pass** instead of Accept:

```mermaid
sequenceDiagram
    participant DG as Driver Group Chat
    participant TG as Telegram API
    participant WH as Webhook Handler
    participant DB as PostgreSQL
    participant A as Assigner Worker

    DG->>TG: callbackQuery {data: "pass:{groupID}"}
    TG->>WH: POST /telegram/webhook
    WH->>DB: UPDATE ride_groups SET dispatch_attempts = dispatch_attempts + 1
    WH->>DB: INSERT jobs (assign_group, run_after = now() + backoff)
    Note over A: Worker picks up job after backoff<br/>and re-dispatches to same or next driver
```

Backoff increases with `dispatch_attempts` to prevent flooding the driver group.
