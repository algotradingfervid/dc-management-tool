-- +goose Up
ALTER TABLE transporter_vehicles ADD COLUMN rc_image_path TEXT NOT NULL DEFAULT '';
ALTER TABLE transporter_vehicles ADD COLUMN driver_license_path TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN on older versions; no-op
SELECT 1;
