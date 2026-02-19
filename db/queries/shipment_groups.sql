-- shipment_groups.sql
-- sqlc-annotated queries for shipment groups and their associated delivery challans.
--
-- Actual shipment_groups schema (from migrations 00024 + 00025):
--   id, project_id, template_id, num_sets, tax_type, reverse_charge,
--   status, created_by, created_at, updated_at
--
-- delivery_challans gained: shipment_group_id, bill_from_address_id,
--   dispatch_from_address_id (migration 00025).

-- =============================================================================
-- Create
-- =============================================================================

-- name: CreateShipmentGroup :execresult
INSERT INTO shipment_groups (
    project_id, template_id, num_sets, tax_type,
    reverse_charge, status, created_by
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- =============================================================================
-- Reads - single group
-- =============================================================================

-- name: GetShipmentGroup :one
-- Returns the group with joined template name, project name, transit DC info,
-- and a correlated sub-query for official DC count.
SELECT
    sg.id,
    sg.project_id,
    sg.template_id,
    sg.num_sets,
    sg.tax_type,
    sg.reverse_charge,
    sg.status,
    sg.created_by,
    sg.created_at,
    sg.updated_at,
    t.name                     AS template_name,
    p.name                     AS project_name,
    tdc.id                     AS transit_dc_id,
    tdc.dc_number              AS transit_dc_number,
    (
        SELECT COUNT(*)
        FROM delivery_challans dc2
        WHERE dc2.shipment_group_id = sg.id
          AND dc2.dc_type = 'official'
    )                          AS official_dc_count
FROM shipment_groups sg
LEFT JOIN dc_templates t      ON sg.template_id = t.id
LEFT JOIN projects p          ON sg.project_id  = p.id
LEFT JOIN delivery_challans tdc
       ON tdc.shipment_group_id = sg.id
      AND tdc.dc_type           = 'transit'
WHERE sg.id = ?;

-- name: GetShipmentGroupIDByDCID :one
-- Fetches the shipment_group_id for a delivery challan (used by GetShipmentGroupByDCID
-- in Go, which then calls GetShipmentGroup).
SELECT shipment_group_id
FROM delivery_challans
WHERE id                  = ?
  AND shipment_group_id IS NOT NULL;

-- =============================================================================
-- Reads - list
-- =============================================================================

-- name: GetShipmentGroupsByProjectID :many
-- Returns all shipment groups for a project with summary computed columns.
SELECT
    sg.id,
    sg.project_id,
    sg.template_id,
    sg.num_sets,
    sg.tax_type,
    sg.reverse_charge,
    sg.status,
    sg.created_by,
    sg.created_at,
    sg.updated_at,
    COALESCE(t.name, '')       AS template_name,
    (
        SELECT COUNT(*)
        FROM delivery_challans dc2
        WHERE dc2.shipment_group_id = sg.id
          AND dc2.dc_type = 'official'
    )                          AS official_dc_count,
    COALESCE(tdc.dc_number, '') AS transit_dc_number,
    tdc.id                      AS transit_dc_id
FROM shipment_groups sg
LEFT JOIN dc_templates t      ON sg.template_id = t.id
LEFT JOIN delivery_challans tdc
       ON tdc.shipment_group_id = sg.id
      AND tdc.dc_type           = 'transit'
WHERE sg.project_id = ?
ORDER BY sg.created_at DESC;

-- name: GetShipmentGroupDCs :many
-- Returns all delivery challans belonging to a shipment group.
-- (Identical query also used by GetDCsByShipmentGroup in delivery_challans.go.)
SELECT
    dc.id,
    dc.project_id,
    dc.dc_number,
    dc.dc_type,
    dc.status,
    dc.challan_date,
    dc.created_at,
    dc.updated_at
FROM delivery_challans dc
WHERE dc.shipment_group_id = ?
ORDER BY dc.dc_type DESC, dc.id;

-- =============================================================================
-- Updates
-- =============================================================================

-- name: UpdateShipmentGroupStatus :exec
UPDATE shipment_groups
SET status     = ?,
    updated_at = ?
WHERE id = ?;

-- name: IssueAllDCsInGroup :execresult
-- Issues every draft DC in a shipment group atomically.
UPDATE delivery_challans
SET status     = 'issued',
    issued_at  = ?,
    issued_by  = ?,
    updated_at = ?
WHERE shipment_group_id = ?
  AND status            = 'draft';
