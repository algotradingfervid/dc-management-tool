package database

import (
	"context"
	"database/sql"
	"fmt"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// mapUserRow converts a sqlc GetUserByIDRow to a models.User.
func mapUserByIDRow(r db.GetUserByIDRow) *models.User {
	u := &models.User{
		ID:           int(r.ID),
		Username:     r.Username,
		PasswordHash: r.PasswordHash,
		FullName:     r.FullName,
		Email:        r.Email.String,
		Role:         r.Role,
		IsActive:     r.IsActive != 0,
	}
	if r.CreatedAt.Valid {
		u.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		u.UpdatedAt = r.UpdatedAt.Time
	}
	if r.LastProjectID.Valid {
		id := int(r.LastProjectID.Int64)
		u.LastProjectID = &id
	}
	return u
}

// mapUserByUsernameRow converts a sqlc GetUserByUsernameRow to a models.User.
func mapUserByUsernameRow(r db.GetUserByUsernameRow) *models.User {
	u := &models.User{
		ID:           int(r.ID),
		Username:     r.Username,
		PasswordHash: r.PasswordHash,
		FullName:     r.FullName,
		Email:        r.Email.String,
		Role:         r.Role,
		IsActive:     r.IsActive != 0,
	}
	if r.CreatedAt.Valid {
		u.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		u.UpdatedAt = r.UpdatedAt.Time
	}
	if r.LastProjectID.Valid {
		id := int(r.LastProjectID.Int64)
		u.LastProjectID = &id
	}
	return u
}

// mapGetAllUsersRow converts a sqlc GetAllUsersRow to a models.User.
func mapGetAllUsersRow(r db.GetAllUsersRow) *models.User {
	u := &models.User{
		ID:           int(r.ID),
		Username:     r.Username,
		PasswordHash: r.PasswordHash,
		FullName:     r.FullName,
		Email:        r.Email.String,
		Role:         r.Role,
		IsActive:     r.IsActive != 0,
	}
	if r.CreatedAt.Valid {
		u.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		u.UpdatedAt = r.UpdatedAt.Time
	}
	if r.LastProjectID.Valid {
		id := int(r.LastProjectID.Int64)
		u.LastProjectID = &id
	}
	return u
}

// mapProjectAssignedUserRow converts a sqlc GetProjectAssignedUsersRow to a models.User.
func mapProjectAssignedUserRow(r db.GetProjectAssignedUsersRow) *models.User {
	u := &models.User{
		ID:           int(r.ID),
		Username:     r.Username,
		PasswordHash: r.PasswordHash,
		FullName:     r.FullName,
		Email:        r.Email.String,
		Role:         r.Role,
		IsActive:     r.IsActive != 0,
	}
	if r.CreatedAt.Valid {
		u.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		u.UpdatedAt = r.UpdatedAt.Time
	}
	if r.LastProjectID.Valid {
		id := int(r.LastProjectID.Int64)
		u.LastProjectID = &id
	}
	return u
}

func queries() *db.Queries {
	return db.New(DB)
}

func GetUserByUsername(username string) (*models.User, error) {
	row, err := queries().GetUserByUsername(context.Background(), username)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	return mapUserByUsernameRow(row), nil
}

func GetUserByID(id int) (*models.User, error) {
	row, err := queries().GetUserByID(context.Background(), int64(id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	return mapUserByIDRow(row), nil
}

func CreateUser(user *models.User) error {
	result, err := queries().CreateUser(context.Background(), db.CreateUserParams{
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		FullName:     user.FullName,
		Email:        sql.NullString{String: user.Email, Valid: user.Email != ""},
		Role:         user.Role,
		IsActive:     boolToInt64(user.IsActive),
	})
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
	return queries().UpdateUser(context.Background(), db.UpdateUserParams{
		FullName: user.FullName,
		Email:    sql.NullString{String: user.Email, Valid: user.Email != ""},
		Role:     user.Role,
		ID:       int64(user.ID),
	})
}

func UpdateUserPassword(userID int, passwordHash string) error {
	return queries().UpdateUserPassword(context.Background(), db.UpdateUserPasswordParams{
		PasswordHash: passwordHash,
		ID:           int64(userID),
	})
}

func DeactivateUser(userID int) error {
	return queries().DeactivateUser(context.Background(), int64(userID))
}

func ActivateUser(userID int) error {
	return queries().ActivateUser(context.Background(), int64(userID))
}

func GetAllUsers() ([]*models.User, error) {
	rows, err := queries().GetAllUsers(context.Background())
	if err != nil {
		return nil, err
	}
	users := make([]*models.User, 0, len(rows))
	for _, r := range rows {
		users = append(users, mapGetAllUsersRow(r))
	}
	return users, nil
}

func UpdateLastProjectID(userID, projectID int) error {
	return queries().UpdateLastProjectID(context.Background(), db.UpdateLastProjectIDParams{
		LastProjectID: sql.NullInt64{Int64: int64(projectID), Valid: true},
		ID:            int64(userID),
	})
}

func GetUserProjectCount(userID int) (int, error) {
	count, err := queries().GetUserProjectCount(context.Background(), int64(userID))
	return int(count), err
}

func GetUserProjects(userID int) ([]*models.Project, error) {
	rows, err := queries().GetUserProjects(context.Background(), int64(userID))
	if err != nil {
		return nil, err
	}
	projects := make([]*models.Project, 0, len(rows))
	for _, r := range rows {
		p := &models.Project{
			ID:       int(r.ID),
			Name:     r.Name,
			DCPrefix: r.DcPrefix,
		}
		if r.CreatedAt.Valid {
			p.CreatedAt = r.CreatedAt.Time
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// User-Project assignment functions

func AssignUserToProject(userID, projectID int) error {
	return queries().AssignUserToProject(context.Background(), db.AssignUserToProjectParams{
		UserID:    int64(userID),
		ProjectID: int64(projectID),
	})
}

func RemoveUserFromProject(userID, projectID int) error {
	return queries().RemoveUserFromProject(context.Background(), db.RemoveUserFromProjectParams{
		UserID:    int64(userID),
		ProjectID: int64(projectID),
	})
}

func GetUserAssignedProjects(userID int) ([]*models.Project, error) {
	rows, err := queries().GetUserAssignedProjects(context.Background(), int64(userID))
	if err != nil {
		return nil, err
	}
	projects := make([]*models.Project, 0, len(rows))
	for _, r := range rows {
		p := &models.Project{
			ID:       int(r.ID),
			Name:     r.Name,
			DCPrefix: r.DcPrefix,
		}
		if r.CreatedAt.Valid {
			p.CreatedAt = r.CreatedAt.Time
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func GetAssignedProjectIDs(userID int) ([]int, error) {
	ids64, err := queries().GetAssignedProjectIDs(context.Background(), int64(userID))
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(ids64))
	for _, id := range ids64 {
		ids = append(ids, int(id))
	}
	return ids, nil
}

func IsUserAssignedToProject(userID, projectID int) (bool, error) {
	count, err := queries().IsUserAssignedToProject(context.Background(), db.IsUserAssignedToProjectParams{
		UserID:    int64(userID),
		ProjectID: int64(projectID),
	})
	return count > 0, err
}

func GetProjectAssignedUsers(projectID int) ([]*models.User, error) {
	rows, err := queries().GetProjectAssignedUsers(context.Background(), int64(projectID))
	if err != nil {
		return nil, err
	}
	users := make([]*models.User, 0, len(rows))
	for _, r := range rows {
		users = append(users, mapProjectAssignedUserRow(r))
	}
	return users, nil
}

// GetAccessibleProjects returns all projects for admins, or only assigned projects for regular users.
func GetAccessibleProjects(user *models.User) ([]*models.Project, error) {
	if user.IsAdmin() {
		return GetAllProjects()
	}
	return GetUserAssignedProjects(user.ID)
}

// boolToInt64 converts a bool to int64 (1 or 0) for SQLite storage.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
