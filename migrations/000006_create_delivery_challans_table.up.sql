-- Delivery Challans table (main DC entity)
CREATE TABLE delivery_challans (
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
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (template_id) REFERENCES dc_templates(id) ON DELETE SET NULL,
    FOREIGN KEY (bill_to_address_id) REFERENCES addresses(id),
    FOREIGN KEY (ship_to_address_id) REFERENCES addresses(id),
    FOREIGN KEY (issued_by) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE(project_id, dc_number)
);

-- DC Transit Details table (one-to-one with delivery_challans for transit DCs)
CREATE TABLE dc_transit_details (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dc_id INTEGER NOT NULL UNIQUE,
    transporter_name TEXT,
    vehicle_number TEXT,
    eway_bill_number TEXT,
    notes TEXT,
    FOREIGN KEY (dc_id) REFERENCES delivery_challans(id) ON DELETE CASCADE
);

-- Indexes for delivery challans
CREATE INDEX idx_delivery_challans_project_id ON delivery_challans(project_id);
CREATE INDEX idx_delivery_challans_dc_number ON delivery_challans(dc_number);
CREATE INDEX idx_delivery_challans_status ON delivery_challans(status);
CREATE INDEX idx_delivery_challans_dc_type ON delivery_challans(dc_type);
CREATE INDEX idx_delivery_challans_created_by ON delivery_challans(created_by);
CREATE INDEX idx_dc_transit_details_dc_id ON dc_transit_details(dc_id);
