-- Recreate serial_numbers table with product_id and new unique constraint
-- SQLite doesn't support dropping constraints, so we use the rename-recreate pattern

-- 1. Rename old table
ALTER TABLE serial_numbers RENAME TO serial_numbers_old;

-- 2. Create new table with product_id and updated unique constraint
CREATE TABLE serial_numbers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    line_item_id INTEGER NOT NULL,
    product_id INTEGER,
    serial_number TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (line_item_id) REFERENCES dc_line_items(id) ON DELETE CASCADE
);

-- 3. Copy data, backfilling product_id from dc_line_items
INSERT INTO serial_numbers (id, project_id, line_item_id, product_id, serial_number, created_at)
SELECT s.id, s.project_id, s.line_item_id,
       (SELECT li.product_id FROM dc_line_items li WHERE li.id = s.line_item_id),
       s.serial_number, s.created_at
FROM serial_numbers_old s;

-- 4. Create new unique index per (project_id, product_id, serial_number)
CREATE UNIQUE INDEX idx_serial_numbers_project_product_serial
    ON serial_numbers(project_id, product_id, serial_number);

-- 5. Drop old table
DROP TABLE serial_numbers_old;
