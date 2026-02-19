-- +goose Up
-- DC Number Sequences table
-- Tracks sequential DC numbers per project, per DC type, per financial year
CREATE TABLE IF NOT EXISTS dc_number_sequences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    dc_type TEXT NOT NULL CHECK(dc_type IN ('transit', 'official')),
    financial_year TEXT NOT NULL, -- Format: YYYYYY (e.g., '2425' for FY 2024-25)
    next_sequence INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE (project_id, dc_type, financial_year)
);

CREATE INDEX IF NOT EXISTS idx_dc_sequences_lookup ON dc_number_sequences(project_id, dc_type, financial_year);

-- +goose Down
DROP TABLE IF EXISTS dc_number_sequences;
