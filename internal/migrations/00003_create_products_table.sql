-- +goose Up
-- Products table (catalog per project)
CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    item_name TEXT NOT NULL,
    item_description TEXT NOT NULL,
    hsn_code TEXT,
    uom TEXT DEFAULT 'Nos',
    brand_model TEXT NOT NULL,
    per_unit_price DECIMAL(10, 2),
    gst_percentage DECIMAL(5, 2) DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Indexes for products
CREATE INDEX idx_products_project_id ON products(project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_products_project_id;
DROP TABLE IF EXISTS products;
