-- +goose Up
ALTER TABLE fleet_projects RENAME TO projects;

ALTER TABLE monitors ADD COLUMN project_id BIGINT REFERENCES projects (id) ON DELETE SET NULL;

CREATE INDEX idx_monitors_project ON monitors (project_id) WHERE project_id IS NOT NULL;

UPDATE monitors m SET project_id = p.id
FROM projects p
WHERE lower(m.name) = lower(p.name);

ALTER TABLE users ADD COLUMN github_id TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS github_id;
DROP INDEX IF EXISTS idx_monitors_project;
ALTER TABLE monitors DROP COLUMN IF EXISTS project_id;
ALTER TABLE projects RENAME TO fleet_projects;
