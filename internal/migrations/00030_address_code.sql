-- +goose Up
ALTER TABLE addresses ADD COLUMN address_code TEXT DEFAULT NULL;
CREATE UNIQUE INDEX idx_addresses_address_code ON addresses(address_code) WHERE address_code IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_addresses_address_code;
ALTER TABLE addresses DROP COLUMN address_code;
