-- name: GetTemplatesByProjectID :many
SELECT
    t.id, t.project_id, t.name, t.purpose, t.created_at, t.updated_at,
    COUNT(DISTINCT tp.product_id) AS product_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) AS transit_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) AS official_count,
    COUNT(DISTINCT dc.id) AS usage_count
FROM dc_templates t
LEFT JOIN dc_template_products tp ON t.id = tp.template_id
LEFT JOIN delivery_challans dc ON t.id = dc.template_id
WHERE t.project_id = ?
GROUP BY t.id
ORDER BY t.created_at DESC;

-- name: GetTemplateByID :one
SELECT
    t.id, t.project_id, t.name, t.purpose, t.created_at, t.updated_at,
    COUNT(DISTINCT tp.product_id) AS product_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) AS transit_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) AS official_count,
    COUNT(DISTINCT dc.id) AS usage_count
FROM dc_templates t
LEFT JOIN dc_template_products tp ON t.id = tp.template_id
LEFT JOIN delivery_challans dc ON t.id = dc.template_id
WHERE t.id = ?
GROUP BY t.id;

-- name: GetTemplateProducts :many
SELECT
    p.id, p.project_id, p.item_name, p.item_description, p.hsn_code, p.uom,
    p.brand_model, p.per_unit_price, p.gst_percentage, p.created_at, p.updated_at,
    tp.default_quantity
FROM products p
INNER JOIN dc_template_products tp ON p.id = tp.product_id
WHERE tp.template_id = ?
ORDER BY tp.sort_order, p.item_name;

-- name: CreateTemplate :execresult
INSERT INTO dc_templates (project_id, name, purpose)
VALUES (?, ?, ?);

-- name: InsertTemplateProduct :exec
INSERT INTO dc_template_products (template_id, product_id, default_quantity, sort_order)
VALUES (?, ?, ?, ?);

-- name: UpdateTemplate :exec
UPDATE dc_templates
SET name = ?, purpose = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND project_id = ?;

-- name: DeleteTemplateProducts :exec
DELETE FROM dc_template_products
WHERE template_id = ?;

-- name: DeleteTemplate :exec
DELETE FROM dc_templates
WHERE id = ? AND project_id = ?;

-- name: CheckTemplateHasDCs :one
SELECT COUNT(*)
FROM delivery_challans
WHERE template_id = ?;

-- name: CheckTemplateNameUnique :one
SELECT COUNT(*)
FROM dc_templates
WHERE project_id = ? AND name = ?;

-- name: CheckTemplateNameUniqueExcludeID :one
SELECT COUNT(*)
FROM dc_templates
WHERE project_id = ? AND name = ? AND id != ?;

-- name: GetTemplateProductIDs :many
SELECT product_id, default_quantity
FROM dc_template_products
WHERE template_id = ?;

-- name: GetTemplateDuplicateSource :many
SELECT product_id, default_quantity, sort_order
FROM dc_template_products
WHERE template_id = ?
ORDER BY sort_order;
