-- reports.sql
-- sqlc-annotated queries for the reports module.
--
-- NOTE: The Go helper dateFilterSQL() builds WHERE clauses dynamically based on
-- whether startDate and/or endDate are non-nil. sqlc requires static SQL.
-- Strategy:
--   • Unfiltered variant  — no date clause (suffix-less name)
--   • Date-filtered variant — both bounds present (suffix "Filtered")
-- For single-bound date ranges (start-only or end-only), the hand-written Go
-- queries in internal/database/reports.go remain appropriate.
--
-- NOTE: GetSerialReport also accepts a variable-length comma-separated search
-- string that builds a dynamic LIKE OR clause. That function stays in Go.
-- A single-term LIKE variant is provided here for reference.

-- =============================================================================
-- DC Summary Report
-- =============================================================================

-- name: GetDCSummaryTransitDraft :one
SELECT COUNT(*) AS transit_draft_dcs
FROM delivery_challans dc
WHERE dc.project_id = ?
  AND dc.dc_type    = 'transit'
  AND dc.status     = 'draft';

-- name: GetDCSummaryTransitDraftFiltered :one
SELECT COUNT(*) AS transit_draft_dcs
FROM delivery_challans dc
WHERE dc.project_id    = ?
  AND dc.dc_type       = 'transit'
  AND dc.status        = 'draft'
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?;

-- name: GetDCSummaryTransitIssued :one
SELECT COUNT(*) AS transit_issued_dcs
FROM delivery_challans dc
WHERE dc.project_id = ?
  AND dc.dc_type    = 'transit'
  AND dc.status     = 'issued';

-- name: GetDCSummaryTransitIssuedFiltered :one
SELECT COUNT(*) AS transit_issued_dcs
FROM delivery_challans dc
WHERE dc.project_id    = ?
  AND dc.dc_type       = 'transit'
  AND dc.status        = 'issued'
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?;

-- name: GetDCSummaryOfficialDraft :one
SELECT COUNT(*) AS official_draft_dcs
FROM delivery_challans dc
WHERE dc.project_id = ?
  AND dc.dc_type    = 'official'
  AND dc.status     = 'draft';

-- name: GetDCSummaryOfficialDraftFiltered :one
SELECT COUNT(*) AS official_draft_dcs
FROM delivery_challans dc
WHERE dc.project_id    = ?
  AND dc.dc_type       = 'official'
  AND dc.status        = 'draft'
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?;

-- name: GetDCSummaryOfficialIssued :one
SELECT COUNT(*) AS official_issued_dcs
FROM delivery_challans dc
WHERE dc.project_id = ?
  AND dc.dc_type    = 'official'
  AND dc.status     = 'issued';

-- name: GetDCSummaryOfficialIssuedFiltered :one
SELECT COUNT(*) AS official_issued_dcs
FROM delivery_challans dc
WHERE dc.project_id    = ?
  AND dc.dc_type       = 'official'
  AND dc.status        = 'issued'
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?;

-- name: GetDCSummaryTotalItemsDispatched :one
SELECT COALESCE(SUM(li.quantity), 0) AS total_items_dispatched
FROM dc_line_items li
INNER JOIN delivery_challans dc ON li.dc_id = dc.id
WHERE dc.project_id = ?
  AND dc.status     = 'issued';

-- name: GetDCSummaryTotalItemsDispatchedFiltered :one
SELECT COALESCE(SUM(li.quantity), 0) AS total_items_dispatched
FROM dc_line_items li
INNER JOIN delivery_challans dc ON li.dc_id = dc.id
WHERE dc.project_id    = ?
  AND dc.status        = 'issued'
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?;

-- name: GetDCSummaryTotalSerialsUsed :one
SELECT COUNT(*) AS total_serials_used
FROM serial_numbers sn
INNER JOIN dc_line_items li ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc ON li.dc_id = dc.id
WHERE dc.project_id = ?;

-- name: GetDCSummaryTotalSerialsUsedFiltered :one
SELECT COUNT(*) AS total_serials_used
FROM serial_numbers sn
INNER JOIN dc_line_items li ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc ON li.dc_id = dc.id
WHERE dc.project_id    = ?
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?;

-- =============================================================================
-- Destination Report
-- =============================================================================

-- name: GetDestinationReport :many
SELECT
    COALESCE(a.district_name, 'Unknown')  AS district,
    COALESCE(a.mandal_name,   'Unknown')  AS mandal,
    COUNT(CASE WHEN dc.dc_type = 'official' THEN 1 END)                       AS official_dcs,
    COALESCE(SUM(li_counts.total_qty), 0)                                     AS total_items,
    COUNT(CASE WHEN dc.dc_type = 'official' AND dc.status = 'draft'  THEN 1 END) AS draft_count,
    COUNT(CASE WHEN dc.dc_type = 'official' AND dc.status = 'issued' THEN 1 END) AS issued_count
FROM delivery_challans dc
LEFT JOIN addresses a ON dc.ship_to_address_id = a.id
LEFT JOIN (
    SELECT dc_id, SUM(quantity) AS total_qty
    FROM dc_line_items
    GROUP BY dc_id
) li_counts ON li_counts.dc_id = dc.id
WHERE dc.project_id = ?
GROUP BY COALESCE(a.district_name, 'Unknown'), COALESCE(a.mandal_name, 'Unknown')
ORDER BY district, mandal;

-- name: GetDestinationReportFiltered :many
SELECT
    COALESCE(a.district_name, 'Unknown')  AS district,
    COALESCE(a.mandal_name,   'Unknown')  AS mandal,
    COUNT(CASE WHEN dc.dc_type = 'official' THEN 1 END)                       AS official_dcs,
    COALESCE(SUM(li_counts.total_qty), 0)                                     AS total_items,
    COUNT(CASE WHEN dc.dc_type = 'official' AND dc.status = 'draft'  THEN 1 END) AS draft_count,
    COUNT(CASE WHEN dc.dc_type = 'official' AND dc.status = 'issued' THEN 1 END) AS issued_count
FROM delivery_challans dc
LEFT JOIN addresses a ON dc.ship_to_address_id = a.id
LEFT JOIN (
    SELECT dc_id, SUM(quantity) AS total_qty
    FROM dc_line_items
    GROUP BY dc_id
) li_counts ON li_counts.dc_id = dc.id
WHERE dc.project_id    = ?
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?
GROUP BY COALESCE(a.district_name, 'Unknown'), COALESCE(a.mandal_name, 'Unknown')
ORDER BY district, mandal;

-- =============================================================================
-- Destination Drill-Down
-- =============================================================================

-- name: GetDestinationDCs :many
SELECT
    dc.id,
    dc.dc_number,
    COALESCE(dc.challan_date, '') AS challan_date,
    dc.status,
    COALESCE(SUM(li.quantity), 0) AS total_items,
    dc.project_id
FROM delivery_challans dc
LEFT JOIN addresses a     ON dc.ship_to_address_id = a.id
LEFT JOIN dc_line_items li ON li.dc_id = dc.id
WHERE dc.project_id = ?
  AND dc.dc_type    = 'official'
  AND COALESCE(a.district_name, 'Unknown') = ?
  AND COALESCE(a.mandal_name,   'Unknown') = ?
GROUP BY dc.id
ORDER BY dc.challan_date DESC;

-- name: GetDestinationDCsFiltered :many
SELECT
    dc.id,
    dc.dc_number,
    COALESCE(dc.challan_date, '') AS challan_date,
    dc.status,
    COALESCE(SUM(li.quantity), 0) AS total_items,
    dc.project_id
FROM delivery_challans dc
LEFT JOIN addresses a     ON dc.ship_to_address_id = a.id
LEFT JOIN dc_line_items li ON li.dc_id = dc.id
WHERE dc.project_id    = ?
  AND dc.dc_type       = 'official'
  AND COALESCE(a.district_name, 'Unknown') = ?
  AND COALESCE(a.mandal_name,   'Unknown') = ?
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?
GROUP BY dc.id
ORDER BY dc.challan_date DESC;

-- =============================================================================
-- Product Report
-- =============================================================================

-- name: GetProductReport :many
SELECT
    COALESCE(p.item_name, 'Unknown')                                          AS product_name,
    COALESCE(SUM(li.quantity), 0)                                             AS total_qty,
    COUNT(DISTINCT dc.id)                                                     AS dc_count,
    COUNT(DISTINCT COALESCE(a.district_name, '') || '|' || COALESCE(a.mandal_name, '')) AS destination_count
FROM dc_line_items li
INNER JOIN delivery_challans dc ON li.dc_id = dc.id
LEFT JOIN products p            ON li.product_id = p.id
LEFT JOIN addresses a           ON dc.ship_to_address_id = a.id
WHERE dc.project_id = ?
GROUP BY li.product_id
ORDER BY total_qty DESC;

-- name: GetProductReportFiltered :many
SELECT
    COALESCE(p.item_name, 'Unknown')                                          AS product_name,
    COALESCE(SUM(li.quantity), 0)                                             AS total_qty,
    COUNT(DISTINCT dc.id)                                                     AS dc_count,
    COUNT(DISTINCT COALESCE(a.district_name, '') || '|' || COALESCE(a.mandal_name, '')) AS destination_count
FROM dc_line_items li
INNER JOIN delivery_challans dc ON li.dc_id = dc.id
LEFT JOIN products p            ON li.product_id = p.id
LEFT JOIN addresses a           ON dc.ship_to_address_id = a.id
WHERE dc.project_id    = ?
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?
GROUP BY li.product_id
ORDER BY total_qty DESC;

-- =============================================================================
-- Serial Report
-- NOTE: GetSerialReport in Go builds a dynamic LIKE OR clause for comma-separated
-- search terms. That function stays in hand-written Go. The single-term variant
-- below covers the common case of one search token and is usable via sqlc.
-- =============================================================================

-- name: GetSerialReportNoSearch :many
SELECT
    sn.serial_number,
    COALESCE(p.item_name, 'Unknown')          AS product_name,
    dc.dc_number                              AS transit_dc_number,
    dc.id                                     AS transit_dc_id,
    COALESCE(dc.challan_date, '')             AS challan_date,
    COALESCE(td.vehicle_number, '')           AS vehicle_number,
    dc.project_id
FROM serial_numbers sn
INNER JOIN dc_line_items li        ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc    ON li.dc_id        = dc.id
LEFT JOIN products p               ON li.product_id   = p.id
LEFT JOIN dc_transit_details td    ON td.dc_id        = dc.id
WHERE dc.project_id = ?
ORDER BY sn.serial_number
LIMIT 500;

-- name: GetSerialReportNoSearchFiltered :many
SELECT
    sn.serial_number,
    COALESCE(p.item_name, 'Unknown')          AS product_name,
    dc.dc_number                              AS transit_dc_number,
    dc.id                                     AS transit_dc_id,
    COALESCE(dc.challan_date, '')             AS challan_date,
    COALESCE(td.vehicle_number, '')           AS vehicle_number,
    dc.project_id
FROM serial_numbers sn
INNER JOIN dc_line_items li        ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc    ON li.dc_id        = dc.id
LEFT JOIN products p               ON li.product_id   = p.id
LEFT JOIN dc_transit_details td    ON td.dc_id        = dc.id
WHERE dc.project_id    = ?
  AND dc.challan_date >= ?
  AND dc.challan_date <= ?
ORDER BY sn.serial_number
LIMIT 500;

-- name: GetSerialReportSingleSearch :many
-- Single search term variant (LIKE). Multi-term variant stays in hand-written Go.
SELECT
    sn.serial_number,
    COALESCE(p.item_name, 'Unknown')          AS product_name,
    dc.dc_number                              AS transit_dc_number,
    dc.id                                     AS transit_dc_id,
    COALESCE(dc.challan_date, '')             AS challan_date,
    COALESCE(td.vehicle_number, '')           AS vehicle_number,
    dc.project_id
FROM serial_numbers sn
INNER JOIN dc_line_items li        ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc    ON li.dc_id        = dc.id
LEFT JOIN products p               ON li.product_id   = p.id
LEFT JOIN dc_transit_details td    ON td.dc_id        = dc.id
WHERE dc.project_id          = ?
  AND sn.serial_number LIKE  ?
ORDER BY sn.serial_number
LIMIT 500;

-- name: GetSerialReportSingleSearchFiltered :many
-- Single search term + date range variant.
SELECT
    sn.serial_number,
    COALESCE(p.item_name, 'Unknown')          AS product_name,
    dc.dc_number                              AS transit_dc_number,
    dc.id                                     AS transit_dc_id,
    COALESCE(dc.challan_date, '')             AS challan_date,
    COALESCE(td.vehicle_number, '')           AS vehicle_number,
    dc.project_id
FROM serial_numbers sn
INNER JOIN dc_line_items li        ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc    ON li.dc_id        = dc.id
LEFT JOIN products p               ON li.product_id   = p.id
LEFT JOIN dc_transit_details td    ON td.dc_id        = dc.id
WHERE dc.project_id          = ?
  AND dc.challan_date        >= ?
  AND dc.challan_date        <= ?
  AND sn.serial_number LIKE  ?
ORDER BY sn.serial_number
LIMIT 500;
