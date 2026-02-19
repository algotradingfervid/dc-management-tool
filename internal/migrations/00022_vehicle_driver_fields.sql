-- +goose Up
ALTER TABLE transporter_vehicles ADD COLUMN driver_name TEXT DEFAULT '';
ALTER TABLE transporter_vehicles ADD COLUMN driver_phone1 TEXT DEFAULT '';
ALTER TABLE transporter_vehicles ADD COLUMN driver_phone2 TEXT DEFAULT '';

-- +goose Down
-- SQLite doesn't support DROP COLUMN before 3.35.0, recreate table
CREATE TABLE transporter_vehicles_backup AS SELECT id, transporter_id, vehicle_number, vehicle_type, created_at FROM transporter_vehicles;
DROP TABLE transporter_vehicles;
ALTER TABLE transporter_vehicles_backup RENAME TO transporter_vehicles;
