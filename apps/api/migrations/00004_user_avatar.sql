-- +goose Up
-- The picture Registre publishes at <issuer>/avatars/<subject>. Sonde stores the
-- URL and never the bytes, so a change in Registre shows on the next page load.
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
