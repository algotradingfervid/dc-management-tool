-- +goose Up
CREATE TABLE IF NOT EXISTS shipment_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL REFERENCES projects(id),
    template_id INTEGER REFERENCES dc_templates(id),
    num_sets INTEGER NOT NULL DEFAULT 1,
    tax_type TEXT NOT NULL DEFAULT 'cgst_sgst',
    reverse_charge TEXT NOT NULL DEFAULT 'N',
    status TEXT NOT NULL DEFAULT 'draft',
    created_by INTEGER REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_shipment_groups_project_id ON shipment_groups(project_id);
CREATE INDEX idx_shipment_groups_status ON shipment_groups(status);

-- +goose Down
DROP TABLE IF EXISTS shipment_groups;
