-- SQLite does not support DROP COLUMN before 3.35.0
-- Rebuild table without the new columns
CREATE TABLE delivery_challans_backup AS SELECT
    id, project_id, dc_number, dc_type, status, template_id,
    bill_to_address_id, ship_to_address_id, challan_date,
    issued_at, issued_by, created_by, created_at, updated_at, bundle_id
FROM delivery_challans;

DROP TABLE delivery_challans;

ALTER TABLE delivery_challans_backup RENAME TO delivery_challans;
