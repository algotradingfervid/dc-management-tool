package models

import (
	"time"
)

type User struct {
	ID            int       `json:"id"`
	Username      string    `json:"username" validate:"required,max=50"`
	PasswordHash  string    `json:"-"`
	FullName      string    `json:"full_name" validate:"required,max=255"`
	Email         string    `json:"email" validate:"required,email"`
	Role          string    `json:"role" validate:"required,oneof=admin user"`
	IsActive      bool      `json:"is_active"`
	LastProjectID *int      `json:"last_project_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}
