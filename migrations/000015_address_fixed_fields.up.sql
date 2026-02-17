-- Add fixed columns for ship-to addresses (District, Mandal, Mandal Code)
-- These are used in Official DC "Issued To" and need to be queryable
ALTER TABLE addresses ADD COLUMN district_name TEXT NOT NULL DEFAULT '';
ALTER TABLE addresses ADD COLUMN mandal_name TEXT NOT NULL DEFAULT '';
ALTER TABLE addresses ADD COLUMN mandal_code TEXT NOT NULL DEFAULT '';

-- Index for searching by district/mandal
CREATE INDEX idx_addresses_district ON addresses(district_name);
CREATE INDEX idx_addresses_mandal ON addresses(mandal_name);
