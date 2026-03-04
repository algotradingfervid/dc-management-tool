-- +goose Up
ALTER TABLE projects ADD COLUMN company_name TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN company_pan TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projects DROP COLUMN company_name;
ALTER TABLE projects DROP COLUMN company_pan;
