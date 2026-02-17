-- name: GetCompanySettings :one
SELECT id, name, address, city, state, state_code, pincode, gstin, signature_image,
       COALESCE(email, '') AS email, COALESCE(cin, '') AS cin
FROM company_settings
WHERE id = 1;

-- name: UpdateCompanySettings :exec
UPDATE company_settings SET
    name = ?, address = ?, city = ?, state = ?, state_code = ?,
    pincode = ?, gstin = ?, signature_image = ?, email = ?, cin = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- name: UpdateCompanySignature :exec
UPDATE company_settings SET signature_image = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- name: InitCompanySettings :exec
INSERT OR IGNORE INTO company_settings (id, name, address, city, state, state_code, pincode, gstin)
VALUES (1, ?, ?, ?, ?, ?, ?, ?);
