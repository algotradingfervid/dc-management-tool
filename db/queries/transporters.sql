-- name: GetTransportersByProjectID :many
SELECT id, project_id, company_name, contact_person, phone, gst_number,
       is_active, created_at, updated_at
FROM transporters
WHERE project_id = ?
ORDER BY company_name ASC;

-- name: GetActiveTransportersByProjectID :many
SELECT id, project_id, company_name, contact_person, phone, gst_number,
       is_active, created_at, updated_at
FROM transporters
WHERE project_id = ? AND is_active = 1
ORDER BY company_name ASC;

-- name: GetTransporterByID :one
SELECT id, project_id, company_name, contact_person, phone, gst_number,
       is_active, created_at, updated_at
FROM transporters
WHERE id = ?;

-- name: CreateTransporter :execresult
INSERT INTO transporters (project_id, company_name, contact_person, phone, gst_number, is_active)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateTransporter :exec
UPDATE transporters SET
    company_name = ?, contact_person = ?, phone = ?, gst_number = ?,
    is_active = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND project_id = ?;

-- name: DeactivateTransporter :exec
UPDATE transporters SET is_active = 0, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND project_id = ?;

-- name: ActivateTransporter :exec
UPDATE transporters SET is_active = 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND project_id = ?;

-- name: GetTransporterCount :one
SELECT COUNT(*) FROM transporters WHERE project_id = ? AND is_active = 1;

-- name: SearchTransportersCount :one
SELECT COUNT(*) FROM transporters
WHERE project_id = ?
  AND (company_name LIKE ? OR contact_person LIKE ? OR phone LIKE ? OR gst_number LIKE ?);

-- name: SearchTransportersCountNoFilter :one
SELECT COUNT(*) FROM transporters WHERE project_id = ?;

-- name: SearchTransporters :many
SELECT id, project_id, company_name, contact_person, phone, gst_number,
       is_active, created_at, updated_at
FROM transporters
WHERE project_id = ?
  AND (company_name LIKE ? OR contact_person LIKE ? OR phone LIKE ? OR gst_number LIKE ?)
ORDER BY company_name ASC
LIMIT ? OFFSET ?;

-- name: SearchTransportersNoFilter :many
SELECT id, project_id, company_name, contact_person, phone, gst_number,
       is_active, created_at, updated_at
FROM transporters
WHERE project_id = ?
ORDER BY company_name ASC
LIMIT ? OFFSET ?;

-- name: GetVehiclesByTransporterID :many
SELECT id, transporter_id, vehicle_number, vehicle_type,
       driver_name, driver_phone1, driver_phone2, created_at
FROM transporter_vehicles
WHERE transporter_id = ?
ORDER BY vehicle_number ASC;

-- name: GetVehicleByID :one
SELECT id, transporter_id, vehicle_number, vehicle_type,
       driver_name, driver_phone1, driver_phone2, created_at
FROM transporter_vehicles
WHERE id = ?;

-- name: CreateVehicle :execresult
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
VALUES (?, ?, ?, ?, ?, ?);

-- name: IsVehicleUsedInDC :one
SELECT COUNT(*) FROM dc_transit_details WHERE vehicle_number = ?;

-- name: DeleteVehicle :exec
DELETE FROM transporter_vehicles WHERE id = ? AND transporter_id = ?;
