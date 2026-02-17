-- serial_search.sql
-- sqlc-annotated queries for the global serial number search feature.
--
-- NOTE: SearchSerialNumbers in Go parses a comma/newline-separated query string
-- into individual tokens and builds a dynamic WHERE clause with one LIKE condition
-- per token joined with OR:
--
--     WHERE (sn.serial_number LIKE ? OR sn.serial_number LIKE ? OR ...)
--       AND dc.project_id = ?        -- optional, only when projectID != "" && != "all"
--
-- sqlc cannot statically analyse this pattern. The full hand-written implementation
-- in internal/database/serial_search.go must remain unchanged.
--
-- The queries below cover the statically-expressible cases:
--   1. Single search token, all projects
--   2. Single search token, specific project
--   3. Two search tokens, all projects     (demonstrates :many with fixed arity)
--   4. Two search tokens, specific project
--
-- For production use the Go function is preferred; these variants exist so that
-- sqlc can generate useful helper types (SerialSearchResult row struct) and so
-- that the canonical SQL is documented and version-controlled.

-- =============================================================================
-- Single token — all projects
-- =============================================================================

-- name: SearchSerialsSingleTermAllProjects :many
SELECT
    sn.serial_number,
    dc.dc_number,
    dc.id                                    AS dc_id,
    dc.dc_type,
    dc.project_id,
    COALESCE(p.name,       'Unknown')        AS project_name,
    COALESCE(pr.item_name, 'Unknown')        AS product_name,
    COALESCE(dc.challan_date, '')            AS challan_date,
    COALESCE(
        (
            SELECT GROUP_CONCAT(val, ', ')
            FROM (
                SELECT json_each.value AS val
                FROM addresses a2, json_each(a2.address_data)
                WHERE a2.id = dc.ship_to_address_id
                LIMIT 2
            )
        ),
        'N/A'
    )                                        AS ship_to_summary,
    dc.status
FROM serial_numbers sn
INNER JOIN dc_line_items li        ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc    ON li.dc_id        = dc.id
LEFT JOIN products pr              ON li.product_id   = pr.id
LEFT JOIN projects p               ON dc.project_id   = p.id
WHERE sn.serial_number LIKE ?
ORDER BY dc.challan_date DESC, sn.serial_number ASC
LIMIT 200;

-- =============================================================================
-- Single token — scoped to one project
-- =============================================================================

-- name: SearchSerialsSingleTermByProject :many
SELECT
    sn.serial_number,
    dc.dc_number,
    dc.id                                    AS dc_id,
    dc.dc_type,
    dc.project_id,
    COALESCE(p.name,       'Unknown')        AS project_name,
    COALESCE(pr.item_name, 'Unknown')        AS product_name,
    COALESCE(dc.challan_date, '')            AS challan_date,
    COALESCE(
        (
            SELECT GROUP_CONCAT(val, ', ')
            FROM (
                SELECT json_each.value AS val
                FROM addresses a2, json_each(a2.address_data)
                WHERE a2.id = dc.ship_to_address_id
                LIMIT 2
            )
        ),
        'N/A'
    )                                        AS ship_to_summary,
    dc.status
FROM serial_numbers sn
INNER JOIN dc_line_items li        ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc    ON li.dc_id        = dc.id
LEFT JOIN products pr              ON li.product_id   = pr.id
LEFT JOIN projects p               ON dc.project_id   = p.id
WHERE sn.serial_number LIKE ?
  AND dc.project_id = ?
ORDER BY dc.challan_date DESC, sn.serial_number ASC
LIMIT 200;

-- =============================================================================
-- Two tokens — all projects
-- (Illustrates fixed-arity OR expansion; Go handles arbitrary arity.)
-- =============================================================================

-- name: SearchSerialsTwoTermsAllProjects :many
SELECT
    sn.serial_number,
    dc.dc_number,
    dc.id                                    AS dc_id,
    dc.dc_type,
    dc.project_id,
    COALESCE(p.name,       'Unknown')        AS project_name,
    COALESCE(pr.item_name, 'Unknown')        AS product_name,
    COALESCE(dc.challan_date, '')            AS challan_date,
    COALESCE(
        (
            SELECT GROUP_CONCAT(val, ', ')
            FROM (
                SELECT json_each.value AS val
                FROM addresses a2, json_each(a2.address_data)
                WHERE a2.id = dc.ship_to_address_id
                LIMIT 2
            )
        ),
        'N/A'
    )                                        AS ship_to_summary,
    dc.status
FROM serial_numbers sn
INNER JOIN dc_line_items li        ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc    ON li.dc_id        = dc.id
LEFT JOIN products pr              ON li.product_id   = pr.id
LEFT JOIN projects p               ON dc.project_id   = p.id
WHERE (sn.serial_number LIKE ? OR sn.serial_number LIKE ?)
ORDER BY dc.challan_date DESC, sn.serial_number ASC
LIMIT 200;

-- =============================================================================
-- Two tokens — scoped to one project
-- =============================================================================

-- name: SearchSerialsTwoTermsByProject :many
SELECT
    sn.serial_number,
    dc.dc_number,
    dc.id                                    AS dc_id,
    dc.dc_type,
    dc.project_id,
    COALESCE(p.name,       'Unknown')        AS project_name,
    COALESCE(pr.item_name, 'Unknown')        AS product_name,
    COALESCE(dc.challan_date, '')            AS challan_date,
    COALESCE(
        (
            SELECT GROUP_CONCAT(val, ', ')
            FROM (
                SELECT json_each.value AS val
                FROM addresses a2, json_each(a2.address_data)
                WHERE a2.id = dc.ship_to_address_id
                LIMIT 2
            )
        ),
        'N/A'
    )                                        AS ship_to_summary,
    dc.status
FROM serial_numbers sn
INNER JOIN dc_line_items li        ON sn.line_item_id = li.id
INNER JOIN delivery_challans dc    ON li.dc_id        = dc.id
LEFT JOIN products pr              ON li.product_id   = pr.id
LEFT JOIN projects p               ON dc.project_id   = p.id
WHERE (sn.serial_number LIKE ? OR sn.serial_number LIKE ?)
  AND dc.project_id = ?
ORDER BY dc.challan_date DESC, sn.serial_number ASC
LIMIT 200;
