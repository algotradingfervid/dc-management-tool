-- Projects table
CREATE TABLE projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    dc_prefix TEXT NOT NULL,
    tender_ref_number TEXT NOT NULL,
    tender_ref_details TEXT NOT NULL,
    po_reference TEXT NOT NULL,
    po_date DATE,
    bill_from_address TEXT NOT NULL,
    company_gstin TEXT NOT NULL DEFAULT '36AACCF9742K1Z8',
    company_signature_path TEXT,
    last_transit_dc_number INTEGER DEFAULT 0,
    last_official_dc_number INTEGER DEFAULT 0,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Indexes for projects
CREATE INDEX idx_projects_created_by ON projects(created_by);
CREATE INDEX idx_projects_dc_prefix ON projects(dc_prefix);
