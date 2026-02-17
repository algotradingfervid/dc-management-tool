-- SQLite doesn't support DROP COLUMN easily, recreate table without sort_order
CREATE TABLE dc_template_products_backup AS SELECT id, template_id, product_id, default_quantity FROM dc_template_products;
DROP TABLE dc_template_products;
CREATE TABLE dc_template_products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    default_quantity INTEGER DEFAULT 1,
    FOREIGN KEY (template_id) REFERENCES dc_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE(template_id, product_id)
);
INSERT INTO dc_template_products (id, template_id, product_id, default_quantity) SELECT id, template_id, product_id, default_quantity FROM dc_template_products_backup;
DROP TABLE dc_template_products_backup;
