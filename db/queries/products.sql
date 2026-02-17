-- name: GetProductsByProjectID :many
SELECT id, project_id, item_name, item_description, hsn_code, uom,
       brand_model, per_unit_price, gst_percentage, created_at, updated_at
FROM products
WHERE project_id = ?
ORDER BY item_name ASC;

-- name: GetProductByID :one
SELECT id, project_id, item_name, item_description, hsn_code, uom,
       brand_model, per_unit_price, gst_percentage, created_at, updated_at
FROM products
WHERE id = ?;

-- name: CreateProduct :execresult
INSERT INTO products (
    project_id, item_name, item_description, hsn_code, uom,
    brand_model, per_unit_price, gst_percentage
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProduct :exec
UPDATE products SET
    item_name = ?, item_description = ?, hsn_code = ?, uom = ?,
    brand_model = ?, per_unit_price = ?, gst_percentage = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND project_id = ?;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = ? AND project_id = ?;

-- name: CheckProductUsageInTemplates :one
SELECT COUNT(*) FROM dc_template_products WHERE product_id = ?;

-- name: CheckProductNameUnique :one
SELECT COUNT(*) FROM products WHERE project_id = ? AND item_name = ?;

-- name: CheckProductNameUniqueExcludeID :one
SELECT COUNT(*) FROM products WHERE project_id = ? AND item_name = ? AND id != ?;

-- name: GetProductCount :one
SELECT COUNT(*) FROM products WHERE project_id = ?;

-- name: SearchProductsCount :one
SELECT COUNT(*) FROM products
WHERE project_id = ?
  AND (item_name LIKE ? OR hsn_code LIKE ? OR brand_model LIKE ? OR item_description LIKE ?);

-- name: SearchProductsCountNoFilter :one
SELECT COUNT(*) FROM products WHERE project_id = ?;

-- name: SearchProducts :many
SELECT id, project_id, item_name, item_description, hsn_code, uom,
       brand_model, per_unit_price, gst_percentage, created_at, updated_at
FROM products
WHERE project_id = ?
  AND (item_name LIKE ? OR hsn_code LIKE ? OR brand_model LIKE ? OR item_description LIKE ?)
ORDER BY item_name ASC
LIMIT ? OFFSET ?;

-- name: SearchProductsNoFilter :many
SELECT id, project_id, item_name, item_description, hsn_code, uom,
       brand_model, per_unit_price, gst_percentage, created_at, updated_at
FROM products
WHERE project_id = ?
ORDER BY item_name ASC
LIMIT ? OFFSET ?;
