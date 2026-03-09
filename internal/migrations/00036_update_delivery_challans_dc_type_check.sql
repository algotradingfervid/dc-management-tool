-- +goose Up
-- +goose NO TRANSACTION
-- Update delivery_challans CHECK constraint to allow 'transfer' dc_type
-- SQLite requires table recreation to modify CHECK constraints
-- Must disable FK checks because other tables reference delivery_challans

PRAGMA foreign_keys = OFF;

CREATE TABLE delivery_challans_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    dc_number TEXT NOT NULL,
    dc_type TEXT NOT NULL CHECK(dc_type IN ('transit', 'official', 'transfer')),
    status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'issued', 'splitting', 'split')),
    template_id INTEGER,
    bill_to_address_id INTEGER,
    ship_to_address_id INTEGER NOT NULL,
    challan_date DATE,
    issued_at DATETIME,
    issued_by INTEGER,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    bundle_id INTEGER REFERENCES dc_bundles(id),
    shipment_group_id INTEGER REFERENCES shipment_groups(id),
    bill_from_address_id INTEGER,
    dispatch_from_address_id INTEGER,
    transfer_dc_id INTEGER REFERENCES transfer_dcs(id),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (template_id) REFERENCES dc_templates(id) ON DELETE SET NULL,
    FOREIGN KEY (bill_to_address_id) REFERENCES addresses(id),
    FOREIGN KEY (ship_to_address_id) REFERENCES addresses(id),
    FOREIGN KEY (issued_by) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE(project_id, dc_number)
);

INSERT INTO delivery_challans_new SELECT * FROM delivery_challans;
DROP TABLE delivery_challans;
ALTER TABLE delivery_challans_new RENAME TO delivery_challans;

-- Recreate indexes
CREATE INDEX idx_delivery_challans_project_id ON delivery_challans(project_id);
CREATE INDEX idx_delivery_challans_dc_number ON delivery_challans(dc_number);
CREATE INDEX idx_delivery_challans_status ON delivery_challans(status);
CREATE INDEX idx_delivery_challans_dc_type ON delivery_challans(dc_type);
CREATE INDEX idx_delivery_challans_created_by ON delivery_challans(created_by);
CREATE INDEX IF NOT EXISTS idx_delivery_challans_bundle_id ON delivery_challans(bundle_id);

PRAGMA foreign_keys = ON;

-- +goose Down
-- +goose NO TRANSACTION

PRAGMA foreign_keys = OFF;

CREATE TABLE delivery_challans_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    dc_number TEXT NOT NULL,
    dc_type TEXT NOT NULL CHECK(dc_type IN ('transit', 'official')),
    status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'issued')),
    template_id INTEGER,
    bill_to_address_id INTEGER,
    ship_to_address_id INTEGER NOT NULL,
    challan_date DATE,
    issued_at DATETIME,
    issued_by INTEGER,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    bundle_id INTEGER REFERENCES dc_bundles(id),
    shipment_group_id INTEGER REFERENCES shipment_groups(id),
    bill_from_address_id INTEGER,
    dispatch_from_address_id INTEGER,
    transfer_dc_id INTEGER REFERENCES transfer_dcs(id),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (template_id) REFERENCES dc_templates(id) ON DELETE SET NULL,
    FOREIGN KEY (bill_to_address_id) REFERENCES addresses(id),
    FOREIGN KEY (ship_to_address_id) REFERENCES addresses(id),
    FOREIGN KEY (issued_by) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE(project_id, dc_number)
);

INSERT INTO delivery_challans_old SELECT * FROM delivery_challans WHERE dc_type IN ('transit', 'official');
DROP TABLE delivery_challans;
ALTER TABLE delivery_challans_old RENAME TO delivery_challans;

CREATE INDEX idx_delivery_challans_project_id ON delivery_challans(project_id);
CREATE INDEX idx_delivery_challans_dc_number ON delivery_challans(dc_number);
CREATE INDEX idx_delivery_challans_status ON delivery_challans(status);
CREATE INDEX idx_delivery_challans_dc_type ON delivery_challans(dc_type);
CREATE INDEX idx_delivery_challans_created_by ON delivery_challans(created_by);
CREATE INDEX IF NOT EXISTS idx_delivery_challans_bundle_id ON delivery_challans(bundle_id);

PRAGMA foreign_keys = ON;
