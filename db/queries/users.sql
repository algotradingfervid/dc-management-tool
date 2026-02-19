-- name: GetUserByUsername :one
SELECT id, username, password_hash, full_name, email, role, is_active, last_project_id, created_at, updated_at
FROM users
WHERE username = ?;

-- name: GetUserByID :one
SELECT id, username, password_hash, full_name, email, role, is_active, last_project_id, created_at, updated_at
FROM users
WHERE id = ?;

-- name: CreateUser :execresult
INSERT INTO users (username, password_hash, full_name, email, role, is_active)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateUser :exec
UPDATE users
SET full_name = ?, email = ?, role = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeactivateUser :exec
UPDATE users
SET is_active = 0, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ActivateUser :exec
UPDATE users
SET is_active = 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetAllUsers :many
SELECT id, username, password_hash, full_name, email, role, is_active, last_project_id, created_at, updated_at
FROM users
ORDER BY full_name ASC;

-- name: UpdateLastProjectID :exec
UPDATE users
SET last_project_id = ?
WHERE id = ?;

-- name: GetUserProjectCount :one
SELECT COUNT(*)
FROM projects
WHERE created_by = ?;

-- name: GetUserProjects :many
SELECT p.id, p.name, p.dc_prefix, p.created_at
FROM projects p
WHERE p.created_by = ?
ORDER BY p.name ASC;

-- name: AssignUserToProject :exec
INSERT OR IGNORE INTO user_projects (user_id, project_id)
VALUES (?, ?);

-- name: RemoveUserFromProject :exec
DELETE FROM user_projects
WHERE user_id = ? AND project_id = ?;

-- name: GetUserAssignedProjects :many
SELECT p.id, p.name, p.dc_prefix, p.created_at
FROM projects p
INNER JOIN user_projects up ON up.project_id = p.id
WHERE up.user_id = ?
ORDER BY p.name ASC;

-- name: GetAssignedProjectIDs :many
SELECT project_id
FROM user_projects
WHERE user_id = ?;

-- name: IsUserAssignedToProject :one
SELECT COUNT(*)
FROM user_projects
WHERE user_id = ? AND project_id = ?;

-- name: GetProjectAssignedUsers :many
SELECT u.id, u.username, u.password_hash, u.full_name, u.email, u.role, u.is_active, u.last_project_id, u.created_at, u.updated_at
FROM users u
INNER JOIN user_projects up ON up.user_id = u.id
WHERE up.project_id = ?
ORDER BY u.full_name ASC;
