-- delivery_challans.sql
-- sqlc-annotated queries for delivery challans, transit details, line items, and serial numbers.
--
-- NOTE: CreateDeliveryChallan, DeleteDC are transactions in Go code that wrap
-- multiple INSERT/DELETE statements. The individual statements are extracted here
-- as standalone sqlc queries. The transaction orchestration stays in Go.
--
-- NOTE: CheckSerialsInProject and CheckSerialsInProjectByProduct accept a
-- variable-length list of serial numbers via an IN clause built dynamically in Go.
-- sqlc cannot statically analyse dynamic IN clauses. These functions must remain
-- in hand-written Go. Representative single-serial variants are provided below for
-- reference/documentation only (marked with a comment).

-- =============================================================================
-- Delivery Challans
-- =============================================================================

-- name: InsertDeliveryChallan :execresult
INSERT INTO delivery_challans (
    project_id, dc_number, dc_type, status, template_id,
    bill_to_address_id, ship_to_address_id, challan_date,
    created_by, shipment_group_id, bill_from_address_id, dispatch_from_address_id
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?
);

-- name: GetDeliveryChallanByID :one
SELECT
    d.id, d.project_id, d.dc_number, d.dc_type, d.status,
    d.template_id, d.bill_to_address_id, d.ship_to_address_id,
    d.challan_date, d.issued_at, d.issued_by, d.created_by,
    d.created_at, d.updated_at,
    d.bundle_id, d.shipment_group_id, d.bill_from_address_id,
    d.dispatch_from_address_id,
    t.name AS template_name
FROM delivery_challans d
LEFT JOIN dc_templates t ON d.template_id = t.id
WHERE d.id = ?;

-- name: GetDCsByProjectID :many
-- NOTE: dc_type filter is optional; Go builds this conditionally.
-- Two variants are provided: unfiltered and type-filtered.
SELECT
    dc.id, dc.project_id, dc.dc_number, dc.dc_type, dc.status,
    dc.challan_date, dc.created_at, dc.updated_at
FROM delivery_challans dc
WHERE dc.project_id = ?
ORDER BY dc.created_at DESC;

-- name: GetDCsByProjectIDAndType :many
SELECT
    dc.id, dc.project_id, dc.dc_number, dc.dc_type, dc.status,
    dc.challan_date, dc.created_at, dc.updated_at
FROM delivery_challans dc
WHERE dc.project_id = ?
  AND dc.dc_type = ?
ORDER BY dc.created_at DESC;

-- name: GetDCsByShipmentGroup :many
SELECT
    dc.id, dc.project_id, dc.dc_number, dc.dc_type, dc.status,
    dc.challan_date, dc.created_at, dc.updated_at
FROM delivery_challans dc
WHERE dc.shipment_group_id = ?
ORDER BY dc.dc_type DESC, dc.id;

-- name: IssueDC :execresult
UPDATE delivery_challans
SET status     = 'issued',
    issued_at  = ?,
    issued_by  = ?,
    updated_at = ?
WHERE id     = ?
  AND status = 'draft';

-- name: DeleteSerialNumbersByDCID :exec
-- Step 1 of DeleteDC transaction: remove serial numbers for all line items of a DC.
DELETE FROM serial_numbers
WHERE line_item_id IN (
    SELECT id FROM dc_line_items WHERE dc_id = ?
);

-- name: DeleteLineItemsByDCID :exec
-- Step 2 of DeleteDC transaction: remove line items for a DC.
DELETE FROM dc_line_items WHERE dc_id = ?;

-- name: DeleteTransitDetailsByDCID :exec
-- Step 3 of DeleteDC transaction: remove transit details (may not exist for official DCs).
DELETE FROM dc_transit_details WHERE dc_id = ?;

-- name: DeleteDeliveryChallan :exec
-- Step 4 of DeleteDC transaction: remove the DC record itself.
DELETE FROM delivery_challans WHERE id = ?;

-- =============================================================================
-- DC Transit Details
-- =============================================================================

-- name: InsertDCTransitDetails :exec
INSERT INTO dc_transit_details (dc_id, transporter_name, vehicle_number, eway_bill_number, notes)
VALUES (?, ?, ?, ?, ?);

-- name: GetTransitDetailsByDCID :one
SELECT id, dc_id, transporter_name, vehicle_number, eway_bill_number, notes
FROM dc_transit_details
WHERE dc_id = ?;

-- =============================================================================
-- DC Line Items
-- =============================================================================

-- name: InsertDCLineItem :execresult
INSERT INTO dc_line_items (
    dc_id, product_id, quantity, rate, tax_percentage,
    taxable_amount, tax_amount, total_amount, line_order
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetLineItemsByDCID :many
SELECT
    li.id, li.dc_id, li.product_id, li.quantity, li.rate, li.tax_percentage,
    li.taxable_amount, li.tax_amount, li.total_amount, li.line_order,
    li.created_at, li.updated_at,
    p.item_name, p.item_description, COALESCE(p.hsn_code, '') AS hsn_code,
    p.uom, COALESCE(p.brand_model, '') AS brand_model, p.gst_percentage
FROM dc_line_items li
INNER JOIN products p ON li.product_id = p.id
WHERE li.dc_id = ?
ORDER BY li.line_order;

-- =============================================================================
-- Serial Numbers
-- =============================================================================

-- name: InsertSerialNumber :exec
INSERT INTO serial_numbers (project_id, line_item_id, serial_number, product_id)
VALUES (?, ?, ?, ?);

-- name: GetSerialNumbersByLineItemID :many
SELECT serial_number
FROM serial_numbers
WHERE line_item_id = ?
ORDER BY id;

-- name: CheckSingleSerialInProject :many
-- NOTE: Single-serial variant of CheckSerialsInProject (variable IN list is not
-- statically analysable by sqlc — the multi-serial version stays in hand-written Go).
-- This variant is useful for single-serial validation calls.
SELECT
    sn.serial_number,
    li.id AS line_item_id,
    dc.id AS dc_id,
    dc.dc_number,
    dc.status,
    p.item_name AS product_name
FROM serial_numbers sn
INNER JOIN dc_line_items li ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc ON li.dc_id = dc.id
INNER JOIN products p ON li.product_id = p.id
WHERE sn.project_id    = ?
  AND sn.serial_number = ?;

-- name: CheckSingleSerialInProjectByProduct :many
-- NOTE: Single-serial variant of CheckSerialsInProjectByProduct (variable IN list
-- stays in hand-written Go for the multi-serial version).
SELECT
    sn.serial_number,
    li.id AS line_item_id,
    dc.id AS dc_id,
    dc.dc_number,
    dc.status,
    p.item_name AS product_name
FROM serial_numbers sn
INNER JOIN dc_line_items li ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc ON li.dc_id = dc.id
INNER JOIN products p ON li.product_id = p.id
WHERE sn.project_id    = ?
  AND sn.product_id    = ?
  AND sn.serial_number = ?;

-- =============================================================================
-- Addresses (dropdown helper — lives in delivery_challans.go in Go code)
-- =============================================================================

-- name: GetAllAddressesByConfigID :many
SELECT
    id, config_id, address_data, district_name, mandal_name, mandal_code,
    created_at, updated_at
FROM addresses
WHERE config_id = ?
ORDER BY id;
