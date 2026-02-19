-- +goose Up
ALTER TABLE users ADD COLUMN last_project_id INTEGER REFERENCES projects(id);

-- +goose Down
-- SQLite doesn't support DROP COLUMN directly in older versions
-- This is handled by recreating the table without the column
CREATE TABLE users_backup AS SELECT id, username, password_hash, full_name, email, created_at, updated_at FROM users;
DROP TABLE users;
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO users SELECT * FROM users_backup;
DROP TABLE users_backup;
