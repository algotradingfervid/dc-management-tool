-- +goose Up
-- Address List Configurations table (metadata for dynamic columns)
CREATE TABLE address_list_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    address_type TEXT NOT NULL CHECK(address_type IN ('bill_to', 'ship_to')),
    column_definitions TEXT NOT NULL, -- JSON: [{"name": "Legal Name", "required": true}, ...]
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE(project_id, address_type)
);

-- Addresses table (stores actual addresses with dynamic fields)
CREATE TABLE addresses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_id INTEGER NOT NULL,
    address_data TEXT NOT NULL, -- JSON: {"Legal Name": "ABC Corp", "GSTIN": "...", ...}
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (config_id) REFERENCES address_list_configs(id) ON DELETE CASCADE
);

-- Indexes for addresses
CREATE INDEX idx_address_list_configs_project_id ON address_list_configs(project_id);
CREATE INDEX idx_addresses_config_id ON addresses(config_id);

-- +goose Down
DROP INDEX IF EXISTS idx_addresses_config_id;
DROP INDEX IF EXISTS idx_address_list_configs_project_id;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS address_list_configs;
