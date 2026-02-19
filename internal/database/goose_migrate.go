package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"
)

// RunMigrationsWithGoose executes all pending migrations using goose
func RunMigrationsWithGoose(db *sql.DB, migrationsFS fs.FS) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to run goose migrations: %w", err)
	}

	slog.Info("All migrations completed successfully")
	return nil
}
