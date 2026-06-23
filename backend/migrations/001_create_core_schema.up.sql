-- ============================================================
-- Marshal: Core schema — requests, groups, drivers, jobs, chat
-- ============================================================

-- Drivers (registered via Telegram bot /start)
CREATE TABLE drivers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    telegram_id     BIGINT UNIQUE NOT NULL,
    telegram_chat   BIGINT,
    status          TEXT NOT NULL DEFAULT 'offline',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ride requests (student intake)
CREATE TABLE ride_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_name  TEXT NOT NULL,
    pickup_lat      DOUBLE PRECISION NOT NULL,
    pickup_lng      DOUBLE PRECISION NOT NULL,
    dropoff_lat     DOUBLE PRECISION NOT NULL,
    dropoff_lng     DOUBLE PRECISION NOT NULL,
    pickup_h3       BIGINT,
    dropoff_h3      BIGINT,
    arrive_by       TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ride groups (formed by grouper, priority-queued by route_score)
CREATE TABLE ride_groups (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status              TEXT NOT NULL DEFAULT 'grouped',
    route_score    DOUBLE PRECISION NOT NULL DEFAULT 0,
    arrive_by           TIMESTAMPTZ NOT NULL,
    expected_departure  TIMESTAMPTZ,
    driver_id           UUID REFERENCES drivers(id),
    dispatch_attempts   INT NOT NULL DEFAULT 0,
    telegram_msg_id     INT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Group membership (many-to-many: requests <-> groups)
CREATE TABLE group_members (
    request_id  UUID REFERENCES ride_requests(id),
    group_id    UUID REFERENCES ride_groups(id),
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (request_id, group_id)
);

-- Job queue (workers pull from here via FOR UPDATE SKIP LOCKED)
CREATE TABLE jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type        TEXT NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'queued',
    attempts        INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 3,
    run_after       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Chat messages (Telegram <-> WebSocket relay bridge)
CREATE TABLE chat_messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID REFERENCES ride_groups(id) NOT NULL,
    sender_type TEXT NOT NULL,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Indexes: Worker hot paths
-- ============================================================

-- Grouper: WHERE status = 'pending'
CREATE INDEX idx_requests_status ON ride_requests(status);

-- Grouper: H3 binning
CREATE INDEX idx_requests_status_h3 ON ride_requests(status, pickup_h3);

-- Assigner: priority queue (THE max-heap)
CREATE INDEX idx_groups_priority_queue
    ON ride_groups(route_score DESC)
    WHERE status = 'grouped' AND driver_id IS NULL;

-- Driver assignment lookups
CREATE INDEX idx_groups_driver_status ON ride_groups(driver_id, status);

-- Group member joins
CREATE INDEX idx_members_group_id ON group_members(group_id);

-- Job queue: pull next available job
CREATE INDEX idx_jobs_queue ON jobs(run_after ASC)
    WHERE status = 'queued';

-- Chat: messages by group
CREATE INDEX idx_chat_group_id ON chat_messages(group_id);
