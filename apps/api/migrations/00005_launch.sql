-- +goose Up
-- porte v0.10.0: CLI and mobile login codes are bound to the identity that
-- vouched for them. Without these two columns the device exchange fails.
ALTER TABLE porte_login_codes ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT '',
                              ADD COLUMN IF NOT EXISTS subject  TEXT NOT NULL DEFAULT '';

-- porte serves no /auth/me. The locale claim of the last Facile sign-in lives
-- here and reaches the client through GET /api/auth/me.
ALTER TABLE users ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'en';

-- porte/directory v0.3.0: the GDPR deletion receiver persists its cursor and the
-- subjects it already handled as one opaque blob.
CREATE TABLE IF NOT EXISTS directory_state (
    key        TEXT        PRIMARY KEY,
    state      BYTEA       NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS directory_state;
ALTER TABLE users DROP COLUMN IF EXISTS locale;
ALTER TABLE porte_login_codes DROP COLUMN IF EXISTS subject, DROP COLUMN IF EXISTS provider;
