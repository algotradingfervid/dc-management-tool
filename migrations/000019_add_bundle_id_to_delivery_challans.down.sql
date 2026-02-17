-- SQLite doesn't support DROP COLUMN directly in older versions.
-- This migration would need to recreate the table. For simplicity, we create a new table without bundle_id.
-- In practice, this is rarely needed.

DROP INDEX IF EXISTS idx_delivery_challans_bundle_id;

-- SQLite 3.35+ supports ALTER TABLE DROP COLUMN
ALTER TABLE delivery_challans DROP COLUMN bundle_id;
