package models

import (
	"time"
)

type User struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	PasswordHash  string    `json:"-"`
	FullName      string    `json:"full_name"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	IsActive      bool      `json:"is_active"`
	LastProjectID *int      `json:"last_project_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

func (u *User) ValidateUser() map[string]string {
	errors := make(map[string]string)
	if u.Username == "" {
		errors["username"] = "Username is required"
	}
	if u.FullName == "" {
		errors["full_name"] = "Full name is required"
	}
	if u.Email == "" {
		errors["email"] = "Email is required"
	}
	if u.Role != "admin" && u.Role != "user" {
		errors["role"] = "Role must be admin or user"
	}
	return errors
}
