package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

// RunMigrations executes all pending migrations
func RunMigrations(db *sql.DB, migrationsPath string) error {
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	migrations, err := loadMigrations(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	for _, migration := range migrations {
		if migration.Version > currentVersion {
			log.Printf("Running migration %d: %s", migration.Version, migration.Name)
			if err := runMigration(db, migration); err != nil {
				return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
			}
		}
	}

	log.Println("All migrations completed successfully")
	return nil
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

func getCurrentVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func loadMigrations(path string) ([]Migration, error) {
	var migrations []Migration
	migrationFiles := make(map[int]Migration)

	err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		fileName := d.Name()
		if !strings.HasSuffix(fileName, ".sql") {
			return nil
		}

		// Parse filename: 000001_create_users_table.up.sql
		parts := strings.Split(fileName, "_")
		if len(parts) < 2 {
			return nil
		}

		var version int
		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			return nil
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		migration, exists := migrationFiles[version]
		if !exists {
			// Extract name: remove version prefix and .up.sql/.down.sql suffix
			namePart := strings.TrimPrefix(fileName, parts[0]+"_")
			namePart = strings.TrimSuffix(namePart, ".up.sql")
			namePart = strings.TrimSuffix(namePart, ".down.sql")
			migration = Migration{
				Version: version,
				Name:    namePart,
			}
		}

		if strings.HasSuffix(fileName, ".up.sql") {
			migration.UpSQL = string(content)
		} else if strings.HasSuffix(fileName, ".down.sql") {
			migration.DownSQL = string(content)
		}

		migrationFiles[version] = migration
		return nil
	})

	if err != nil {
		return nil, err
	}

	for _, migration := range migrationFiles {
		migrations = append(migrations, migration)
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func runMigration(db *sql.DB, migration Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(migration.UpSQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
		migration.Version, migration.Name,
	); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}
