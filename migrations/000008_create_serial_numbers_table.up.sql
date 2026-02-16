-- Serial Numbers table (tracking per line item)
CREATE TABLE serial_numbers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    line_item_id INTEGER NOT NULL,
    serial_number TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (line_item_id) REFERENCES dc_line_items(id) ON DELETE CASCADE,
    UNIQUE(project_id, serial_number)
);

-- Indexes for serial numbers
CREATE INDEX idx_serial_numbers_line_item_id ON serial_numbers(line_item_id);
CREATE INDEX idx_serial_numbers_serial_number ON serial_numbers(serial_number);
