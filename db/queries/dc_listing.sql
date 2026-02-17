-- name: CountAllDCsFiltered :one
-- Dynamic filters are applied in Go; this base query is used when no filters are set.
-- The actual runtime query is built dynamically in GetAllDCsFiltered.
SELECT COUNT(*)
FROM delivery_challans dc;

-- name: CountAllDCsFilteredByProject :one
SELECT COUNT(*)
FROM delivery_challans dc
WHERE dc.project_id = ?;

-- name: CountAllDCsFilteredByType :one
SELECT COUNT(*)
FROM delivery_challans dc
WHERE dc.dc_type = ?;

-- name: CountAllDCsFilteredByStatus :one
SELECT COUNT(*)
FROM delivery_challans dc
WHERE dc.status = ?;

-- name: CountAllDCsFilteredByDateRange :one
SELECT COUNT(*)
FROM delivery_challans dc
WHERE dc.challan_date >= ? AND dc.challan_date <= ?;

-- name: CountAllDCsFilteredBySearch :one
SELECT COUNT(*)
FROM delivery_challans dc
WHERE dc.dc_number LIKE ?;

-- name: GetAllDCsPaginated :many
-- Base paginated listing with project join; dynamic filters applied in Go.
-- The actual runtime query is built dynamically in GetAllDCsFiltered.
SELECT
    dc.id,
    dc.dc_number,
    dc.dc_type,
    COALESCE(dc.challan_date, '') AS challan_date,
    dc.project_id,
    COALESCE(p.name, 'Unknown') AS project_name,
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
    ) AS ship_to_summary,
    dc.status,
    (SELECT SUM(li.total_amount) FROM dc_line_items li WHERE li.dc_id = dc.id) AS total_value,
    (SELECT COUNT(*) FROM dc_line_items li WHERE li.dc_id = dc.id) AS line_item_count,
    (SELECT COALESCE(SUM(li.quantity), 0) FROM dc_line_items li WHERE li.dc_id = dc.id) AS total_quantity
FROM delivery_challans dc
LEFT JOIN projects p ON dc.project_id = p.id
ORDER BY dc.challan_date DESC
LIMIT ? OFFSET ?;

