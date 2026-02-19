-- name: ProjectExistsWithName :one
SELECT COUNT(*)
FROM projects
WHERE LOWER(name) = LOWER(?) AND id != ?;

-- name: ProjectExistsWithPrefix :one
SELECT COUNT(*)
FROM projects
WHERE LOWER(dc_prefix) = LOWER(?) AND id != ?;

-- name: GetAllProjects :many
SELECT
    p.id, p.name, p.description, p.dc_prefix,
    p.tender_ref_number, p.tender_ref_details,
    p.po_reference, p.po_date,
    p.bill_from_address, p.dispatch_from_address,
    p.company_gstin, p.company_email, p.company_cin,
    p.company_signature_path, p.company_seal_path,
    p.dc_number_format, p.dc_number_separator,
    p.purpose_text, p.seq_padding,
    p.last_transit_dc_number, p.last_official_dc_number,
    p.created_by, p.created_at, p.updated_at,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) AS transit_dc_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) AS official_dc_count,
    COUNT(DISTINCT t.id) AS template_count,
    COUNT(DISTINCT pr.id) AS product_count
FROM projects p
LEFT JOIN delivery_challans dc ON p.id = dc.project_id
LEFT JOIN dc_templates t ON p.id = t.project_id
LEFT JOIN products pr ON p.id = pr.project_id
GROUP BY p.id
ORDER BY p.created_at DESC;

-- name: GetProjectByID :one
SELECT
    p.id, p.name, p.description, p.dc_prefix,
    p.tender_ref_number, p.tender_ref_details,
    p.po_reference, p.po_date,
    p.bill_from_address, p.dispatch_from_address,
    p.company_gstin, p.company_email, p.company_cin,
    p.company_signature_path, p.company_seal_path,
    p.dc_number_format, p.dc_number_separator,
    p.purpose_text, p.seq_padding,
    p.last_transit_dc_number, p.last_official_dc_number,
    p.created_by, p.created_at, p.updated_at,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) AS transit_dc_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) AS official_dc_count,
    COUNT(DISTINCT t.id) AS template_count,
    COUNT(DISTINCT pr.id) AS product_count
FROM projects p
LEFT JOIN delivery_challans dc ON p.id = dc.project_id
LEFT JOIN dc_templates t ON p.id = t.project_id
LEFT JOIN products pr ON p.id = pr.project_id
WHERE p.id = ?
GROUP BY p.id;

-- name: CreateProject :execresult
INSERT INTO projects (
    name, description, dc_prefix, tender_ref_number, tender_ref_details,
    po_reference, po_date, bill_from_address, dispatch_from_address,
    company_gstin, company_email, company_cin,
    company_signature_path, company_seal_path,
    dc_number_format, dc_number_separator,
    purpose_text, seq_padding, created_by
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProject :exec
UPDATE projects SET
    name = ?, description = ?, dc_prefix = ?,
    tender_ref_number = ?, tender_ref_details = ?,
    po_reference = ?, po_date = ?,
    bill_from_address = ?, dispatch_from_address = ?,
    company_gstin = ?, company_email = ?, company_cin = ?,
    company_signature_path = ?, company_seal_path = ?,
    dc_number_format = ?, dc_number_separator = ?,
    purpose_text = ?, seq_padding = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateProjectSettingsGeneral :exec
UPDATE projects
SET name = ?, description = ?, dc_prefix = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateProjectSettingsCompany :exec
UPDATE projects
SET bill_from_address = ?, dispatch_from_address = ?, company_gstin = ?,
    company_email = ?, company_cin = ?, company_signature_path = ?,
    company_seal_path = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateProjectSettingsDCConfig :exec
UPDATE projects
SET dc_number_format = ?, dc_number_separator = ?, purpose_text = ?,
    seq_padding = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateProjectSettingsTender :exec
UPDATE projects
SET tender_ref_number = ?, tender_ref_details = ?, po_reference = ?,
    po_date = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CountIssuedDCsForProject :one
SELECT COUNT(*)
FROM delivery_challans
WHERE project_id = ? AND status = 'issued';

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = ?;

-- name: GetAllProjectOptions :many
SELECT id, name
FROM projects
ORDER BY name;
