-- +goose Up
CREATE TABLE webhooks (
    id            BIGSERIAL PRIMARY KEY,
    owner_id      BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name          TEXT        NOT NULL DEFAULT '',
    url           TEXT        NOT NULL,
    secret        TEXT        NOT NULL DEFAULT '',
    secret_header TEXT        NOT NULL DEFAULT 'x-sonde-signature-256',
    enabled       BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_webhooks_owner ON webhooks (owner_id);
CREATE INDEX idx_webhooks_enabled ON webhooks (enabled) WHERE enabled;

-- +goose Down
DROP TABLE IF EXISTS webhooks;
