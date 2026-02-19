-- +goose Up
-- Add unique constraints on project name and dc_prefix
CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_name_unique ON projects(name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_dc_prefix_unique ON projects(dc_prefix);

-- +goose Down
DROP INDEX IF EXISTS idx_projects_name_unique;
DROP INDEX IF EXISTS idx_projects_dc_prefix_unique;
