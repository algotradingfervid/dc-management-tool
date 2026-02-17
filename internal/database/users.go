package database

import (
	"database/sql"
	"fmt"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

func scanUser(row interface{ Scan(...any) error }) (*models.User, error) {
	user := &models.User{}
	var lastProjectID sql.NullInt64
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.FullName,
		&user.Email,
		&user.Role,
		&user.IsActive,
		&lastProjectID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lastProjectID.Valid {
		id := int(lastProjectID.Int64)
		user.LastProjectID = &id
	}
	return user, nil
}

const userColumns = `id, username, password_hash, full_name, email, role, is_active, last_project_id, created_at, updated_at`

func GetUserByUsername(username string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE username = ?`, userColumns)
	user, err := scanUser(DB.QueryRow(query, username))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return user, err
}

func GetUserByID(id int) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE id = ?`, userColumns)
	user, err := scanUser(DB.QueryRow(query, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return user, err
}

func CreateUser(user *models.User) error {
	query := `INSERT INTO users (username, password_hash, full_name, email, role, is_active) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := DB.Exec(query, user.Username, user.PasswordHash, user.FullName, user.Email, user.Role, user.IsActive)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)
	return nil
}

func UpdateUser(user *models.User) error {
	query := `UPDATE users SET full_name = ?, email = ?, role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := DB.Exec(query, user.FullName, user.Email, user.Role, user.ID)
	return err
}

func UpdateUserPassword(userID int, passwordHash string) error {
	_, err := DB.Exec("UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", passwordHash, userID)
	return err
}

func DeactivateUser(userID int) error {
	_, err := DB.Exec("UPDATE users SET is_active = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?", userID)
	return err
}

func ActivateUser(userID int) error {
	_, err := DB.Exec("UPDATE users SET is_active = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", userID)
	return err
}

func GetAllUsers() ([]*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users ORDER BY full_name ASC`, userColumns)
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func UpdateLastProjectID(userID, projectID int) error {
	_, err := DB.Exec("UPDATE users SET last_project_id = ? WHERE id = ?", projectID, userID)
	return err
}

func GetUserProjectCount(userID int) (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM projects WHERE created_by = ?", userID).Scan(&count)
	return count, err
}

func GetUserProjects(userID int) ([]*models.Project, error) {
	query := `
		SELECT p.id, p.name, p.dc_prefix, p.created_at
		FROM projects p
		WHERE p.created_by = ?
		ORDER BY p.name ASC
	`
	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		err := rows.Scan(&p.ID, &p.Name, &p.DCPrefix, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// User-Project assignment functions

func AssignUserToProject(userID, projectID int) error {
	query := `INSERT OR IGNORE INTO user_projects (user_id, project_id) VALUES (?, ?)`
	_, err := DB.Exec(query, userID, projectID)
	return err
}

func RemoveUserFromProject(userID, projectID int) error {
	_, err := DB.Exec("DELETE FROM user_projects WHERE user_id = ? AND project_id = ?", userID, projectID)
	return err
}

func GetUserAssignedProjects(userID int) ([]*models.Project, error) {
	query := `
		SELECT p.id, p.name, p.dc_prefix, p.created_at
		FROM projects p
		INNER JOIN user_projects up ON up.project_id = p.id
		WHERE up.user_id = ?
		ORDER BY p.name ASC
	`
	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		err := rows.Scan(&p.ID, &p.Name, &p.DCPrefix, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func GetAssignedProjectIDs(userID int) ([]int, error) {
	rows, err := DB.Query("SELECT project_id FROM user_projects WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func IsUserAssignedToProject(userID, projectID int) (bool, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM user_projects WHERE user_id = ? AND project_id = ?", userID, projectID).Scan(&count)
	return count > 0, err
}

func GetProjectAssignedUsers(projectID int) ([]*models.User, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM users u
		INNER JOIN user_projects up ON up.user_id = u.id
		WHERE up.project_id = ?
		ORDER BY u.full_name ASC
	`, userColumns)
	rows, err := DB.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// GetAccessibleProjects returns all projects for admins, or only assigned projects for regular users
func GetAccessibleProjects(user *models.User) ([]*models.Project, error) {
	if user.IsAdmin() {
		return GetAllProjects()
	}
	return GetUserAssignedProjects(user.ID)
}
