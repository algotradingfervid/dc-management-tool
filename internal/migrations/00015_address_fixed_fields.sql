-- +goose Up
-- Add fixed columns for ship-to addresses (District, Mandal, Mandal Code)
-- These are used in Official DC "Issued To" and need to be queryable
ALTER TABLE addresses ADD COLUMN district_name TEXT NOT NULL DEFAULT '';
ALTER TABLE addresses ADD COLUMN mandal_name TEXT NOT NULL DEFAULT '';
ALTER TABLE addresses ADD COLUMN mandal_code TEXT NOT NULL DEFAULT '';

-- Index for searching by district/mandal
CREATE INDEX idx_addresses_district ON addresses(district_name);
CREATE INDEX idx_addresses_mandal ON addresses(mandal_name);

-- +goose Down
-- SQLite doesn't support DROP COLUMN in older versions,
-- but Go's migrate library handles this via table recreation if needed.
-- For SQLite 3.35.0+ (2021-03-12), DROP COLUMN is supported.
DROP INDEX IF EXISTS idx_addresses_district;
DROP INDEX IF EXISTS idx_addresses_mandal;
ALTER TABLE addresses DROP COLUMN district_name;
ALTER TABLE addresses DROP COLUMN mandal_name;
ALTER TABLE addresses DROP COLUMN mandal_code;
