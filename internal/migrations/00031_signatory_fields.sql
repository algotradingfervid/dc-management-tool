-- +goose Up
ALTER TABLE projects ADD COLUMN signatory_name TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN signatory_designation TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN signatory_mobile TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projects DROP COLUMN signatory_name;
ALTER TABLE projects DROP COLUMN signatory_designation;
ALTER TABLE projects DROP COLUMN signatory_mobile;
