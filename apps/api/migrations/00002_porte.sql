-- +goose Up
-- porte/pg's Schema with porte_users left out: Sonde keeps its own users table
-- (is_admin and the owner_id references are app business) and repoints every
-- foreign key at it. The one UPDATE is porte v0.3.0's re-key, copied along so
-- a password identity is keyed on the account id rather than the address.
CREATE TABLE porte_identities (
    user_id         BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider        TEXT        NOT NULL,
    subject         TEXT        NOT NULL,
    password_hash   TEXT        NOT NULL DEFAULT '',
    access_token    TEXT        NOT NULL DEFAULT '',
    refresh_token   TEXT        NOT NULL DEFAULT '',
    token_expiry    TIMESTAMPTZ,
    roles           JSONB,
    roles_synced_at TIMESTAMPTZ,
    synced_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject)
);

CREATE INDEX porte_identities_user_idx ON porte_identities (user_id);

UPDATE porte_identities SET subject = user_id::text
 WHERE provider = 'local' AND subject <> user_id::text;

CREATE TABLE porte_sessions (
    id           BIGSERIAL PRIMARY KEY,
    token_hash   TEXT        NOT NULL UNIQUE,
    user_id      BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    label        TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ
);

CREATE INDEX porte_sessions_user_idx ON porte_sessions (user_id);
CREATE INDEX porte_sessions_expiry_idx ON porte_sessions (expires_at)
    WHERE expires_at IS NOT NULL;

CREATE TABLE porte_login_codes (
    code_hash  TEXT        PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS porte_login_codes;
DROP TABLE IF EXISTS porte_sessions;
DROP TABLE IF EXISTS porte_identities;
