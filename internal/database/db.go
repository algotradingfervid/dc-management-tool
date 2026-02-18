package database

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // register SQLite driver
)

var DB *sql.DB

func Init(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// SQLite is not designed for high write concurrency.
	// A single connection prevents SQLITE_BUSY "database is locked" errors.
	// WAL mode allows concurrent reads alongside the single writer connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=1000",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, err
		}
	}

	DB = db
	return db, nil
}
