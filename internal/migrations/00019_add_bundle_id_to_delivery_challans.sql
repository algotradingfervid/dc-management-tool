-- +goose Up
-- Add bundle_id foreign key to delivery_challans for linking generated DCs back to their source bundle
ALTER TABLE delivery_challans ADD COLUMN bundle_id INTEGER REFERENCES dc_bundles(id);

-- Index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_delivery_challans_bundle_id ON delivery_challans(bundle_id);

-- +goose Down
-- SQLite doesn't support DROP COLUMN directly in older versions.
-- This migration would need to recreate the table. For simplicity, we create a new table without bundle_id.
-- In practice, this is rarely needed.

DROP INDEX IF EXISTS idx_delivery_challans_bundle_id;

-- SQLite 3.35+ supports ALTER TABLE DROP COLUMN
ALTER TABLE delivery_challans DROP COLUMN bundle_id;
