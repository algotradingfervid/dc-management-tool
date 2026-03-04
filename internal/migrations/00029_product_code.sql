-- +goose Up
ALTER TABLE products ADD COLUMN product_code TEXT DEFAULT NULL;
CREATE UNIQUE INDEX idx_products_product_code ON products(product_code) WHERE product_code IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_products_product_code;
ALTER TABLE products DROP COLUMN product_code;
