-- name: GetAddressConfig :one
SELECT id, project_id, address_type, column_definitions, created_at, updated_at
FROM address_list_configs
WHERE project_id = ? AND address_type = ?;

-- name: CreateAddressConfig :execresult
INSERT INTO address_list_configs (project_id, address_type, column_definitions)
VALUES (?, ?, ?);

-- name: UpdateAddressConfig :exec
UPDATE address_list_configs SET column_definitions = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: InsertAddress :execresult
INSERT INTO addresses (config_id, address_data, district_name, mandal_name, mandal_code)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteAllAddresses :exec
DELETE FROM addresses WHERE config_id = ?;

-- name: DeleteAddress :exec
DELETE FROM addresses WHERE id = ? AND config_id = ?;

-- name: CountAddresses :one
SELECT COUNT(*) FROM addresses WHERE config_id = ?;

-- name: CountAddressesWithSearch :one
SELECT COUNT(*) FROM addresses
WHERE config_id = ?
  AND (address_data LIKE ? OR district_name LIKE ? OR mandal_name LIKE ? OR mandal_code LIKE ?);

-- name: GetAddress :one
SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
FROM addresses
WHERE id = ?;

-- name: ListAddresses :many
SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
FROM addresses
WHERE config_id = ?
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: ListAddressesWithSearch :many
SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
FROM addresses
WHERE config_id = ?
  AND (address_data LIKE ? OR district_name LIKE ? OR mandal_name LIKE ? OR mandal_code LIKE ?)
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: UpdateAddress :exec
UPDATE addresses SET address_data = ?, district_name = ?, mandal_name = ?, mandal_code = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: SearchAddressesForSelector :many
SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
FROM addresses
WHERE config_id = ?
  AND (address_data LIKE ? OR district_name LIKE ? OR mandal_name LIKE ? OR mandal_code LIKE ?)
ORDER BY district_name, mandal_name
LIMIT ?;

-- name: SearchAddressesForSelectorSimple :many
SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
FROM addresses
WHERE config_id = ? AND address_data LIKE ?
ORDER BY id
LIMIT ?;

-- name: SearchAddressesNoFilter :many
SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
FROM addresses
WHERE config_id = ?
ORDER BY id
LIMIT ?;
