-- SQLite doesn't support DROP COLUMN in older versions,
-- but Go's migrate library handles this via table recreation if needed.
-- For SQLite 3.35.0+ (2021-03-12), DROP COLUMN is supported.
DROP INDEX IF EXISTS idx_addresses_district;
DROP INDEX IF EXISTS idx_addresses_mandal;
ALTER TABLE addresses DROP COLUMN district_name;
ALTER TABLE addresses DROP COLUMN mandal_name;
ALTER TABLE addresses DROP COLUMN mandal_code;
