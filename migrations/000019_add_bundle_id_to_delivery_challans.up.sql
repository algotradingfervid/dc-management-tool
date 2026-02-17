-- Add bundle_id foreign key to delivery_challans for linking generated DCs back to their source bundle
ALTER TABLE delivery_challans ADD COLUMN bundle_id INTEGER REFERENCES dc_bundles(id);

-- Index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_delivery_challans_bundle_id ON delivery_challans(bundle_id);
