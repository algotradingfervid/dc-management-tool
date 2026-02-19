-- +goose Up
-- DC Templates table
CREATE TABLE dc_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    purpose TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- DC Template Products junction table (many-to-many)
CREATE TABLE dc_template_products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    default_quantity INTEGER DEFAULT 1,
    FOREIGN KEY (template_id) REFERENCES dc_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE(template_id, product_id)
);

-- Indexes for templates
CREATE INDEX idx_dc_templates_project_id ON dc_templates(project_id);
CREATE INDEX idx_dc_template_products_template_id ON dc_template_products(template_id);
CREATE INDEX idx_dc_template_products_product_id ON dc_template_products(product_id);

-- +goose Down
DROP INDEX IF EXISTS idx_dc_template_products_product_id;
DROP INDEX IF EXISTS idx_dc_template_products_template_id;
DROP INDEX IF EXISTS idx_dc_templates_project_id;
DROP TABLE IF EXISTS dc_template_products;
DROP TABLE IF EXISTS dc_templates;
