-- Transporters table (transport companies per project)
CREATE TABLE IF NOT EXISTS transporters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    company_name TEXT NOT NULL,
    contact_person TEXT DEFAULT '',
    phone TEXT DEFAULT '',
    gst_number TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_transporters_project_id ON transporters(project_id);
CREATE INDEX IF NOT EXISTS idx_transporters_is_active ON transporters(is_active);

-- Transporter vehicles table
CREATE TABLE IF NOT EXISTS transporter_vehicles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transporter_id INTEGER NOT NULL,
    vehicle_number TEXT NOT NULL,
    vehicle_type TEXT DEFAULT 'truck',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (transporter_id) REFERENCES transporters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_transporter_vehicles_transporter_id ON transporter_vehicles(transporter_id);
