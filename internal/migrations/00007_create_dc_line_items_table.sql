-- +goose Up
-- DC Line Items table (products in a DC)
CREATE TABLE dc_line_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dc_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    rate DECIMAL(10, 2),
    tax_percentage DECIMAL(5, 2),
    taxable_amount DECIMAL(12, 2),
    tax_amount DECIMAL(12, 2),
    total_amount DECIMAL(12, 2),
    line_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (dc_id) REFERENCES delivery_challans(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Indexes for line items
CREATE INDEX idx_dc_line_items_dc_id ON dc_line_items(dc_id);
CREATE INDEX idx_dc_line_items_product_id ON dc_line_items(product_id);

-- +goose Down
DROP INDEX IF EXISTS idx_dc_line_items_product_id;
DROP INDEX IF EXISTS idx_dc_line_items_dc_id;
DROP TABLE IF EXISTS dc_line_items;
