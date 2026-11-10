-- +goose Up
CREATE TABLE fleet_edges (
    id         BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES fleet_projects (id) ON DELETE CASCADE,
    to_name    TEXT   NOT NULL,
    kind       TEXT   NOT NULL CHECK (kind IN ('lib', 'call', 'event', 'identity', 'logs')),
    auth       TEXT   NOT NULL DEFAULT '' CHECK (auth IN ('', 'audience', 'static_key', 'webhook', 'url')),
    evidence   TEXT   NOT NULL DEFAULT '',
    UNIQUE (project_id, to_name, kind, auth, evidence)
);

CREATE INDEX idx_fleet_edges_to ON fleet_edges (to_name);

CREATE TABLE fleet_checks (
    id         BIGSERIAL PRIMARY KEY,
    project_id BIGINT  NOT NULL REFERENCES fleet_projects (id) ON DELETE CASCADE,
    check_id   TEXT    NOT NULL,
    ok         BOOLEAN NOT NULL,
    detail     TEXT    NOT NULL DEFAULT '',
    UNIQUE (project_id, check_id)
);

-- +goose Down
DROP TABLE IF EXISTS fleet_checks;
DROP TABLE IF EXISTS fleet_edges;
