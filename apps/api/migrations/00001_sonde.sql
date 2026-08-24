-- +goose Up
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token      TEXT        PRIMARY KEY,
    user_id    BIGINT      NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_user ON sessions (user_id);

CREATE TABLE monitors (
    id               BIGSERIAL PRIMARY KEY,
    owner_id         BIGINT      NOT NULL,
    slug             TEXT        NOT NULL UNIQUE,
    name             TEXT        NOT NULL,
    type             TEXT        NOT NULL CHECK (type IN ('http', 'tcp', 'push')),
    target           TEXT        NOT NULL DEFAULT '',
    interval_seconds INTEGER     NOT NULL DEFAULT 60,
    timeout_seconds  INTEGER     NOT NULL DEFAULT 10,
    expected_status  INTEGER     NOT NULL DEFAULT 200,
    expected_keyword TEXT        NOT NULL DEFAULT '',
    enabled          BOOLEAN     NOT NULL DEFAULT true,
    push_token       TEXT        UNIQUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_monitors_owner ON monitors (owner_id);
CREATE INDEX idx_monitors_enabled ON monitors (enabled) WHERE enabled;

CREATE TABLE checks_raw (
    id             BIGSERIAL PRIMARY KEY,
    monitor_id     BIGINT      NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
    ok             BOOLEAN     NOT NULL,
    latency_ms     BIGINT      NOT NULL DEFAULT 0,
    error          TEXT        NOT NULL DEFAULT '',
    tls_expires_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_checks_raw_monitor_created ON checks_raw (monitor_id, created_at);
CREATE INDEX idx_checks_raw_created ON checks_raw (created_at);

CREATE TABLE checks_hourly (
    monitor_id     BIGINT      NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
    bucket_start   TIMESTAMPTZ NOT NULL,
    total          BIGINT      NOT NULL DEFAULT 0,
    failed         BIGINT      NOT NULL DEFAULT 0,
    avg_latency_ms BIGINT      NOT NULL DEFAULT 0,
    PRIMARY KEY (monitor_id, bucket_start)
);

CREATE TABLE checks_daily (
    monitor_id     BIGINT      NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
    bucket_start   TIMESTAMPTZ NOT NULL,
    total          BIGINT      NOT NULL DEFAULT 0,
    failed         BIGINT      NOT NULL DEFAULT 0,
    avg_latency_ms BIGINT      NOT NULL DEFAULT 0,
    PRIMARY KEY (monitor_id, bucket_start)
);

CREATE TABLE incidents (
    id          BIGSERIAL PRIMARY KEY,
    monitor_id  BIGINT      NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
    opened_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    cause       TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX idx_incidents_monitor ON incidents (monitor_id, resolved_at);

CREATE TABLE status_pages (
    id         BIGSERIAL PRIMARY KEY,
    slug       TEXT        NOT NULL UNIQUE,
    title      TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE status_page_monitors (
    id         BIGSERIAL PRIMARY KEY,
    page_id    BIGINT      NOT NULL REFERENCES status_pages (id) ON DELETE CASCADE,
    monitor_id BIGINT      NOT NULL REFERENCES monitors (id) ON DELETE CASCADE,
    position   INTEGER     NOT NULL DEFAULT 0,
    UNIQUE (page_id, monitor_id)
);

-- +goose Down
DROP TABLE IF EXISTS status_page_monitors;
DROP TABLE IF EXISTS status_pages;
DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS checks_daily;
DROP TABLE IF EXISTS checks_hourly;
DROP TABLE IF EXISTS checks_raw;
DROP TABLE IF EXISTS monitors;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
