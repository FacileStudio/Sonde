-- +goose Up
CREATE TABLE fleet_projects (
    id               BIGSERIAL PRIMARY KEY,
    github_repo_id   BIGINT      NOT NULL UNIQUE,
    name             TEXT        NOT NULL,
    default_branch   TEXT        NOT NULL DEFAULT '',
    tier             TEXT        NOT NULL DEFAULT 'unclassified'
        CHECK (tier IN ('unclassified', 'deployed', 'library', 'cli', 'client', 'experiment', 'archive')),
    version_url      TEXT        NOT NULL DEFAULT '',
    journal_app      TEXT        NOT NULL DEFAULT '',
    covered          BOOLEAN     NOT NULL DEFAULT false,
    not_covered      TEXT        NOT NULL DEFAULT '',
    last_scanned_at  TIMESTAMPTZ,
    ci_conclusion    TEXT        NOT NULL DEFAULT '',
    filet_wired      BOOLEAN     NOT NULL DEFAULT false,
    filet_pin        TEXT        NOT NULL DEFAULT '',
    deployed_version TEXT        NOT NULL DEFAULT '',
    go_directive     TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE fleet_runs (
    id             BIGSERIAL PRIMARY KEY,
    started_at     TIMESTAMPTZ NOT NULL,
    finished_at    TIMESTAMPTZ NOT NULL,
    douane_version TEXT        NOT NULL DEFAULT '',
    repos_expected INTEGER     NOT NULL,
    repos_scanned  INTEGER     NOT NULL,
    complete       BOOLEAN     NOT NULL,
    findings_open  INTEGER     NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fleet_runs_created ON fleet_runs (created_at DESC);

CREATE TABLE fleet_findings (
    id          BIGSERIAL PRIMARY KEY,
    project_id  BIGINT           NOT NULL REFERENCES fleet_projects (id) ON DELETE CASCADE,
    key         TEXT             NOT NULL,
    advisory    TEXT             NOT NULL,
    aliases     TEXT             NOT NULL DEFAULT '',
    summary     TEXT             NOT NULL DEFAULT '',
    severity    TEXT             NOT NULL DEFAULT 'UNKNOWN',
    package     TEXT             NOT NULL,
    ecosystem   TEXT             NOT NULL,
    installed   TEXT             NOT NULL DEFAULT '',
    fixed_in    TEXT             NOT NULL DEFAULT '',
    kev         BOOLEAN          NOT NULL DEFAULT false,
    epss        DOUBLE PRECISION NOT NULL DEFAULT 0,
    scope       TEXT             NOT NULL DEFAULT '',
    target      TEXT             NOT NULL DEFAULT '',
    first_seen  TIMESTAMPTZ      NOT NULL,
    last_seen   TIMESTAMPTZ      NOT NULL,
    resolved_at TIMESTAMPTZ,
    UNIQUE (project_id, key)
);

CREATE INDEX idx_fleet_findings_open ON fleet_findings (project_id) WHERE resolved_at IS NULL;
CREATE INDEX idx_fleet_findings_resolved ON fleet_findings (resolved_at) WHERE resolved_at IS NOT NULL;

CREATE TABLE fleet_pins (
    id         BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES fleet_projects (id) ON DELETE CASCADE,
    path       TEXT   NOT NULL,
    package    TEXT   NOT NULL,
    spec       TEXT   NOT NULL DEFAULT '',
    resolved   TEXT   NOT NULL DEFAULT '',
    UNIQUE (project_id, path, package)
);

CREATE INDEX idx_fleet_pins_package ON fleet_pins (package);

CREATE TABLE fleet_latest (
    package    TEXT        PRIMARY KEY,
    version    TEXT        NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE fleet_policies (
    id           BIGSERIAL PRIMARY KEY,
    package      TEXT        NOT NULL,
    project_id   BIGINT      REFERENCES fleet_projects (id) ON DELETE CASCADE,
    held_version TEXT        NOT NULL,
    reason       TEXT        NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_by   TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fleet_policies_package ON fleet_policies (package);

-- +goose Down
DROP TABLE IF EXISTS fleet_policies;
DROP TABLE IF EXISTS fleet_latest;
DROP TABLE IF EXISTS fleet_pins;
DROP TABLE IF EXISTS fleet_findings;
DROP TABLE IF EXISTS fleet_runs;
DROP TABLE IF EXISTS fleet_projects;
