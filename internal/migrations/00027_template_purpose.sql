-- +goose Up
-- purpose column already exists on dc_templates (added in earlier migration)
-- This migration is intentionally a no-op.
SELECT 1;

-- +goose Down
ALTER TABLE dc_templates DROP COLUMN purpose;
