-- +goose Up
-- Recreate dc_number_sequences with updated CHECK constraint to include 'transfer'
CREATE TABLE dc_number_sequences_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    dc_type TEXT NOT NULL CHECK(dc_type IN ('transit', 'official', 'transfer')),
    financial_year TEXT NOT NULL,
    next_sequence INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE (project_id, dc_type, financial_year)
);
INSERT INTO dc_number_sequences_new SELECT * FROM dc_number_sequences;
DROP TABLE dc_number_sequences;
ALTER TABLE dc_number_sequences_new RENAME TO dc_number_sequences;

-- +goose Down
-- Revert to original CHECK constraint (without 'transfer')
CREATE TABLE dc_number_sequences_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    dc_type TEXT NOT NULL CHECK(dc_type IN ('transit', 'official')),
    financial_year TEXT NOT NULL,
    next_sequence INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE (project_id, dc_type, financial_year)
);
INSERT INTO dc_number_sequences_old SELECT * FROM dc_number_sequences WHERE dc_type IN ('transit', 'official');
DROP TABLE dc_number_sequences;
ALTER TABLE dc_number_sequences_old RENAME TO dc_number_sequences;
