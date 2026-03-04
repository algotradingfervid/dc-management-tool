-- +goose Up
ALTER TABLE projects ADD COLUMN notes TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projects DROP COLUMN notes;
