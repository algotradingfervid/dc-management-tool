-- Remove the per-product unique index
DROP INDEX IF EXISTS idx_serial_numbers_project_product_serial;

-- Restore original unique constraint
CREATE UNIQUE INDEX idx_serial_numbers_project_serial
    ON serial_numbers(project_id, serial_number);

-- Note: Cannot easily DROP COLUMN product_id in older SQLite versions
-- The column will remain but be unused
