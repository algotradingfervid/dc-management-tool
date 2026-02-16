package database

import (
	"database/sql"
	"fmt"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

func GetUserByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, full_name, email, created_at, updated_at
		FROM users
		WHERE username = ?
	`

	user := &models.User{}
	err := DB.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.FullName,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func GetUserByID(id int) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, full_name, email, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	user := &models.User{}
	err := DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.FullName,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (username, password_hash, full_name, email)
		VALUES (?, ?, ?, ?)
	`

	result, err := DB.Exec(query, user.Username, user.PasswordHash, user.FullName, user.Email)
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
