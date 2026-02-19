-- +goose Up
-- Users table for authentication
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL,
    email TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Index for login lookups
CREATE INDEX idx_users_username ON users(username);

-- +goose Down
DROP INDEX IF EXISTS idx_users_username;
DROP TABLE IF EXISTS users;
