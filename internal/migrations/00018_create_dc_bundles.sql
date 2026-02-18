-- +goose Up
-- Draft Bundles: planning documents for generating Transit + Official DCs
CREATE TABLE IF NOT EXISTS dc_bundles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    template_id INTEGER,
    status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'generated')),
    bill_to_address_id INTEGER,
    transit_ship_to_address_id INTEGER,
    bill_from_address TEXT DEFAULT '',
    dispatch_from_address TEXT DEFAULT '',
    transporter_id INTEGER,
    transporter_name TEXT DEFAULT '',
    vehicle_number TEXT DEFAULT '',
    mode_of_transport TEXT DEFAULT '',
    docket_number TEXT DEFAULT '',
    eway_bill_number TEXT DEFAULT '',
    dc_date TEXT,
    reverse_charge INTEGER NOT NULL DEFAULT 0,
    notes TEXT DEFAULT '',
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (template_id) REFERENCES dc_templates(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_dc_bundles_project ON dc_bundles(project_id);
CREATE INDEX IF NOT EXISTS idx_dc_bundles_status ON dc_bundles(status);

-- Products in a bundle with total quantities
CREATE TABLE IF NOT EXISTS dc_bundle_products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bundle_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    total_quantity INTEGER NOT NULL DEFAULT 0,
    line_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (bundle_id) REFERENCES dc_bundles(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE INDEX IF NOT EXISTS idx_dc_bundle_products_bundle ON dc_bundle_products(bundle_id);

-- Quantity allocation: how many of each product goes to each destination
CREATE TABLE IF NOT EXISTS dc_bundle_allocations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bundle_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    ship_to_address_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (bundle_id) REFERENCES dc_bundles(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE INDEX IF NOT EXISTS idx_dc_bundle_allocations_bundle ON dc_bundle_allocations(bundle_id);

-- Serial numbers per product in a bundle
CREATE TABLE IF NOT EXISTS dc_bundle_serials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bundle_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    serial_number TEXT NOT NULL,
    FOREIGN KEY (bundle_id) REFERENCES dc_bundles(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE INDEX IF NOT EXISTS idx_dc_bundle_serials_bundle ON dc_bundle_serials(bundle_id);
CREATE INDEX IF NOT EXISTS idx_dc_bundle_serials_serial ON dc_bundle_serials(serial_number);

-- +goose Down
DROP TABLE IF EXISTS dc_bundle_serials;
DROP TABLE IF EXISTS dc_bundle_allocations;
DROP TABLE IF EXISTS dc_bundle_products;
DROP TABLE IF EXISTS dc_bundles;
