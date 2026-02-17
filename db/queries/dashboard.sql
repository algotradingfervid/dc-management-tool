-- dashboard.sql
-- sqlc-annotated queries for the project dashboard.
--
-- NOTE: GetDashboardStats in Go builds optional date filters (challan_date >= ?,
-- challan_date <= ?) dynamically. sqlc requires static SQL.
-- Two variants are provided for each DC count query:
--   *          — no date filter
--   *Filtered  — both start and end date bound (most common production case)
-- For single-bound date ranges, hand-written Go queries remain appropriate.
--
-- GetDashboardStats also issues many small COUNT queries sequentially; those are
-- all represented here. The Go wrapper function continues to call them individually
-- and compose the DashboardStats struct.

-- =============================================================================
-- Entity counts (no date filter — these never filter by date in the Go code)
-- =============================================================================

-- name: CountProductsByProject :one
SELECT COUNT(*) AS total_products
FROM products
WHERE project_id = ?;

-- name: CountTemplatesByProject :one
SELECT COUNT(*) AS total_templates
FROM dc_templates
WHERE project_id = ?;

-- name: CountBillToAddressesByProject :one
SELECT COUNT(*) AS total_bill_to_addresses
FROM addresses a
JOIN address_list_configs c ON a.config_id = c.id
WHERE c.project_id   = ?
  AND c.address_type = 'bill_to';

-- name: CountShipToAddressesByProject :one
SELECT COUNT(*) AS total_ship_to_addresses
FROM addresses a
JOIN address_list_configs c ON a.config_id = c.id
WHERE c.project_id   = ?
  AND c.address_type = 'ship_to';

-- =============================================================================
-- DC counts — no date filter
-- =============================================================================

-- name: CountDCsByProject :one
SELECT COUNT(*) AS total_dcs
FROM delivery_challans
WHERE project_id = ?;

-- name: CountTransitDCsByProject :one
SELECT COUNT(*) AS transit_dcs
FROM delivery_challans
WHERE project_id = ?
  AND dc_type    = 'transit';

-- name: CountOfficialDCsByProject :one
SELECT COUNT(*) AS official_dcs
FROM delivery_challans
WHERE project_id = ?
  AND dc_type    = 'official';

-- name: CountIssuedDCsByProject :one
SELECT COUNT(*) AS issued_dcs
FROM delivery_challans
WHERE project_id = ?
  AND status     = 'issued';

-- name: CountDraftDCsByProject :one
-- DraftDCs is intentionally not date-filtered in the Go code.
SELECT COUNT(*) AS draft_dcs
FROM delivery_challans
WHERE project_id = ?
  AND status     = 'draft';

-- name: CountTransitDraftDCsByProject :one
SELECT COUNT(*) AS transit_draft_dcs
FROM delivery_challans
WHERE project_id = ?
  AND dc_type    = 'transit'
  AND status     = 'draft';

-- name: CountTransitIssuedDCsByProject :one
SELECT COUNT(*) AS transit_issued_dcs
FROM delivery_challans
WHERE project_id = ?
  AND dc_type    = 'transit'
  AND status     = 'issued';

-- name: CountOfficialDraftDCsByProject :one
SELECT COUNT(*) AS official_draft_dcs
FROM delivery_challans
WHERE project_id = ?
  AND dc_type    = 'official'
  AND status     = 'draft';

-- name: CountOfficialIssuedDCsByProject :one
SELECT COUNT(*) AS official_issued_dcs
FROM delivery_challans
WHERE project_id = ?
  AND dc_type    = 'official'
  AND status     = 'issued';

-- =============================================================================
-- DC counts — date-filtered variants (challan_date range)
-- =============================================================================

-- name: CountDCsByProjectFiltered :one
SELECT COUNT(*) AS total_dcs
FROM delivery_challans
WHERE project_id  = ?
  AND challan_date >= ?
  AND challan_date <= ?;

-- name: CountTransitDCsByProjectFiltered :one
SELECT COUNT(*) AS transit_dcs
FROM delivery_challans
WHERE project_id  = ?
  AND dc_type     = 'transit'
  AND challan_date >= ?
  AND challan_date <= ?;

-- name: CountOfficialDCsByProjectFiltered :one
SELECT COUNT(*) AS official_dcs
FROM delivery_challans
WHERE project_id  = ?
  AND dc_type     = 'official'
  AND challan_date >= ?
  AND challan_date <= ?;

-- name: CountIssuedDCsByProjectFiltered :one
SELECT COUNT(*) AS issued_dcs
FROM delivery_challans
WHERE project_id  = ?
  AND status      = 'issued'
  AND challan_date >= ?
  AND challan_date <= ?;

-- name: CountTransitDraftDCsByProjectFiltered :one
SELECT COUNT(*) AS transit_draft_dcs
FROM delivery_challans
WHERE project_id  = ?
  AND dc_type     = 'transit'
  AND status      = 'draft'
  AND challan_date >= ?
  AND challan_date <= ?;

-- name: CountTransitIssuedDCsByProjectFiltered :one
SELECT COUNT(*) AS transit_issued_dcs
FROM delivery_challans
WHERE project_id  = ?
  AND dc_type     = 'transit'
  AND status      = 'issued'
  AND challan_date >= ?
  AND challan_date <= ?;

-- name: CountOfficialDraftDCsByProjectFiltered :one
SELECT COUNT(*) AS official_draft_dcs
FROM delivery_challans
WHERE project_id  = ?
  AND dc_type     = 'official'
  AND status      = 'draft'
  AND challan_date >= ?
  AND challan_date <= ?;

-- name: CountOfficialIssuedDCsByProjectFiltered :one
SELECT COUNT(*) AS official_issued_dcs
FROM delivery_challans
WHERE project_id  = ?
  AND dc_type     = 'official'
  AND status      = 'issued'
  AND challan_date >= ?
  AND challan_date <= ?;

-- =============================================================================
-- DCs this month (fixed two-bound date filter; bounds computed in Go)
-- =============================================================================

-- name: CountDCsThisMonth :one
SELECT COUNT(*) AS dcs_this_month
FROM delivery_challans
WHERE project_id   = ?
  AND challan_date >= ?
  AND challan_date <= ?;

-- =============================================================================
-- Serial numbers
-- =============================================================================

-- name: CountSerialNumbersByProject :one
SELECT COUNT(*) AS total_serial_numbers
FROM serial_numbers
WHERE project_id = ?;

-- =============================================================================
-- Recent DCs feed
-- =============================================================================

-- name: GetRecentDCs :many
SELECT
    dc.id,
    dc.dc_number,
    dc.dc_type,
    p.name                         AS project_name,
    dc.project_id,
    COALESCE(dc.challan_date, '')  AS challan_date,
    dc.status,
    COALESCE(dc.created_at, '')    AS created_at
FROM delivery_challans dc
LEFT JOIN projects p ON dc.project_id = p.id
WHERE dc.project_id = ?
ORDER BY dc.created_at DESC
LIMIT ?;

-- =============================================================================
-- Recent activity feed
-- =============================================================================

-- name: GetRecentActivity :many
SELECT
    id,
    'dc'                                    AS entity_type,
    id                                      AS entity_id,
    dc_number                               AS title,
    dc_type || ' DC ' || status            AS description,
    status,
    created_at,
    project_id
FROM delivery_challans
WHERE project_id = ?
ORDER BY created_at DESC
LIMIT ?;
