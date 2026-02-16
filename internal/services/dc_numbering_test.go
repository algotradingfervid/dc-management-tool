package services

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with the required schema.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Create projects table
	_, err = db.Exec(`
		CREATE TABLE projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			dc_prefix TEXT NOT NULL,
			tender_ref_number TEXT NOT NULL DEFAULT '',
			tender_ref_details TEXT NOT NULL DEFAULT '',
			po_reference TEXT NOT NULL DEFAULT '',
			bill_from_address TEXT NOT NULL DEFAULT '',
			company_gstin TEXT NOT NULL DEFAULT '',
			last_transit_dc_number INTEGER DEFAULT 0,
			last_official_dc_number INTEGER DEFAULT 0,
			created_by INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create projects table: %v", err)
	}

	// Create dc_number_sequences table
	_, err = db.Exec(`
		CREATE TABLE dc_number_sequences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			dc_type TEXT NOT NULL CHECK(dc_type IN ('transit', 'official')),
			financial_year TEXT NOT NULL,
			next_sequence INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
			UNIQUE (project_id, dc_type, financial_year)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create dc_number_sequences table: %v", err)
	}

	return db
}

func insertTestProject(t *testing.T, db *sql.DB, name, prefix string) int {
	t.Helper()
	result, err := db.Exec("INSERT INTO projects (name, dc_prefix) VALUES (?, ?)", name, prefix)
	if err != nil {
		t.Fatalf("failed to insert project: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

func TestFormatDCNumber(t *testing.T) {
	tests := []struct {
		prefix   string
		fy       string
		dcType   string
		seq      int
		expected string
	}{
		{"SCP", "2425", DCTypeTransit, 1, "SCP-TDC-2425-001"},
		{"SCP", "2425", DCTypeOfficial, 12, "SCP-ODC-2425-012"},
		{"PWD/AP", "2526", DCTypeTransit, 5, "PWD/AP-TDC-2526-005"},
		{"TEST", "2526", DCTypeOfficial, 999, "TEST-ODC-2526-999"},
		{"X", "2627", DCTypeTransit, 1000, "X-TDC-2627-1000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDCNumber(tt.prefix, tt.fy, tt.dcType, tt.seq)
			if result != tt.expected {
				t.Errorf("FormatDCNumber(%s, %s, %s, %d) = %s; want %s",
					tt.prefix, tt.fy, tt.dcType, tt.seq, result, tt.expected)
			}
		})
	}
}

func TestParseDCNumber(t *testing.T) {
	tests := []struct {
		dcNumber    string
		wantPrefix  string
		wantFY      string
		wantType    string
		wantSeq     int
		expectError bool
	}{
		{"SCP-TDC-2425-001", "SCP", "2425", DCTypeTransit, 1, false},
		{"SCP-ODC-2425-012", "SCP", "2425", DCTypeOfficial, 12, false},
		{"PWD/AP-TDC-2526-005", "PWD/AP", "2526", DCTypeTransit, 5, false},
		{"X-TDC-2627-1000", "X", "2627", DCTypeTransit, 1000, false},
		{"INVALID", "", "", "", 0, true},
		{"", "", "", "", 0, true},
		{"SCP-XDC-2425-001", "", "", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.dcNumber, func(t *testing.T) {
			parts, err := ParseDCNumber(tt.dcNumber)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if parts.Prefix != tt.wantPrefix || parts.FinancialYear != tt.wantFY ||
				parts.DCType != tt.wantType || parts.SequenceNumber != tt.wantSeq {
				t.Errorf("ParseDCNumber(%s) = %+v; want prefix=%s fy=%s type=%s seq=%d",
					tt.dcNumber, parts, tt.wantPrefix, tt.wantFY, tt.wantType, tt.wantSeq)
			}
		})
	}
}

func TestIsValidDCNumber(t *testing.T) {
	valid := []string{"SCP-TDC-2425-001", "PWD/AP-ODC-2526-100", "X-TDC-2627-1000"}
	invalid := []string{"", "INVALID", "SCP-XDC-2425-001", "SCP-TDC-25-001"}

	for _, dc := range valid {
		if !IsValidDCNumber(dc) {
			t.Errorf("IsValidDCNumber(%s) = false; want true", dc)
		}
	}
	for _, dc := range invalid {
		if IsValidDCNumber(dc) {
			t.Errorf("IsValidDCNumber(%s) = true; want false", dc)
		}
	}
}

func TestGenerateDCNumber_Basic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "Test Project", "SCP")
	date := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC) // FY 2526

	dc1, err := GenerateDCNumberForDate(db, projectID, DCTypeTransit, date)
	if err != nil {
		t.Fatalf("failed to generate DC number: %v", err)
	}
	if dc1 != "SCP-TDC-2526-001" {
		t.Errorf("first DC = %s; want SCP-TDC-2526-001", dc1)
	}

	dc2, err := GenerateDCNumberForDate(db, projectID, DCTypeTransit, date)
	if err != nil {
		t.Fatalf("failed to generate second DC number: %v", err)
	}
	if dc2 != "SCP-TDC-2526-002" {
		t.Errorf("second DC = %s; want SCP-TDC-2526-002", dc2)
	}
}

func TestGenerateDCNumber_DifferentTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "Test Project", "SCP")
	date := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC)

	transit, err := GenerateDCNumberForDate(db, projectID, DCTypeTransit, date)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	official, err := GenerateDCNumberForDate(db, projectID, DCTypeOfficial, date)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	// Different types should have independent sequences
	if transit != "SCP-TDC-2526-001" {
		t.Errorf("transit = %s; want SCP-TDC-2526-001", transit)
	}
	if official != "SCP-ODC-2526-001" {
		t.Errorf("official = %s; want SCP-ODC-2526-001", official)
	}
}

func TestGenerateDCNumber_DifferentProjects(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	p1 := insertTestProject(t, db, "Project 1", "AAA")
	p2 := insertTestProject(t, db, "Project 2", "BBB")
	date := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC)

	dc1, _ := GenerateDCNumberForDate(db, p1, DCTypeTransit, date)
	dc2, _ := GenerateDCNumberForDate(db, p2, DCTypeTransit, date)

	if dc1 != "AAA-TDC-2526-001" {
		t.Errorf("project 1 DC = %s; want AAA-TDC-2526-001", dc1)
	}
	if dc2 != "BBB-TDC-2526-001" {
		t.Errorf("project 2 DC = %s; want BBB-TDC-2526-001", dc2)
	}
}

func TestGenerateDCNumber_FYRollover(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "Test", "SCP")

	// Generate in FY 2425
	march := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC) // FY 2425
	dc1, _ := GenerateDCNumberForDate(db, projectID, DCTypeTransit, march)
	if dc1 != "SCP-TDC-2425-001" {
		t.Errorf("FY 2425 DC = %s; want SCP-TDC-2425-001", dc1)
	}

	// Generate in FY 2526 - sequence should reset
	april := time.Date(2025, time.April, 1, 0, 0, 0, 0, time.UTC) // FY 2526
	dc2, _ := GenerateDCNumberForDate(db, projectID, DCTypeTransit, april)
	if dc2 != "SCP-TDC-2526-001" {
		t.Errorf("FY 2526 DC = %s; want SCP-TDC-2526-001", dc2)
	}
}

func TestGenerateDCNumber_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "Test", "SCP")
	_, err := GenerateDCNumber(db, projectID, "invalid")
	if err == nil {
		t.Error("expected error for invalid DC type")
	}
}

func TestGenerateDCNumber_ProjectNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := GenerateDCNumber(db, 999, DCTypeTransit)
	if err == nil {
		t.Error("expected error for non-existent project")
	}
}

func TestGenerateDCNumber_EmptyPrefix(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "Test", "")
	_, err := GenerateDCNumber(db, projectID, DCTypeTransit)
	if err == nil {
		t.Error("expected error for empty DC prefix")
	}
}

func TestGenerateDCNumber_SequentialIntegrity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "Test", "SCP")
	date := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC)

	for i := 1; i <= 20; i++ {
		dc, err := GenerateDCNumberForDate(db, projectID, DCTypeTransit, date)
		if err != nil {
			t.Fatalf("failed at sequence %d: %v", i, err)
		}
		expected := fmt.Sprintf("SCP-TDC-2526-%03d", i)
		if dc != expected {
			t.Errorf("sequence %d: got %s; want %s", i, dc, expected)
		}
	}
}

func TestGenerateDCNumber_Concurrent(t *testing.T) {
	// Use file-based DB for concurrency (in-memory doesn't share across connections)
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1) // SQLite works best with single writer

	// Enable foreign keys and WAL mode
	db.Exec("PRAGMA foreign_keys = ON")
	db.Exec("PRAGMA journal_mode = WAL")

	// Create tables
	db.Exec(`CREATE TABLE projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL, dc_prefix TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		tender_ref_number TEXT NOT NULL DEFAULT '',
		tender_ref_details TEXT NOT NULL DEFAULT '',
		po_reference TEXT NOT NULL DEFAULT '',
		bill_from_address TEXT NOT NULL DEFAULT '',
		company_gstin TEXT NOT NULL DEFAULT '',
		created_by INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE dc_number_sequences (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		dc_type TEXT NOT NULL CHECK(dc_type IN ('transit', 'official')),
		financial_year TEXT NOT NULL,
		next_sequence INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
		UNIQUE (project_id, dc_type, financial_year)
	)`)

	result, _ := db.Exec("INSERT INTO projects (name, dc_prefix) VALUES ('Test', 'SCP')")
	projectID64, _ := result.LastInsertId()
	projectID := int(projectID64)

	date := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC)
	numGoroutines := 50

	var wg sync.WaitGroup
	results := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			dc, err := GenerateDCNumberForDate(db, projectID, DCTypeTransit, date)
			results[idx] = dc
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Check no errors
	for i, err := range errors {
		if err != nil {
			t.Errorf("goroutine %d failed: %v", i, err)
		}
	}

	// Check all unique
	seen := make(map[string]bool)
	for i, dc := range results {
		if dc == "" {
			continue
		}
		if seen[dc] {
			t.Errorf("duplicate DC number at goroutine %d: %s", i, dc)
		}
		seen[dc] = true
	}

	// Check we got all expected sequence numbers
	if len(seen) != numGoroutines {
		t.Errorf("expected %d unique DC numbers, got %d", numGoroutines, len(seen))
	}

	// Verify sequential - all numbers from 1 to numGoroutines should exist
	for i := 1; i <= numGoroutines; i++ {
		expected := fmt.Sprintf("SCP-TDC-2526-%03d", i)
		if !seen[expected] {
			t.Errorf("missing expected DC number: %s", expected)
		}
	}
}

func TestGenerateDCNumber_SequenceExceeds999(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "Test", "SCP")

	// Manually set sequence to 999
	db.Exec(`INSERT INTO dc_number_sequences (project_id, dc_type, financial_year, next_sequence)
		VALUES (?, 'transit', '2526', 1000)`, projectID)

	date := time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC)
	dc, err := GenerateDCNumberForDate(db, projectID, DCTypeTransit, date)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	// Should still work, just wider number
	if dc != "SCP-TDC-2526-1000" {
		t.Errorf("got %s; want SCP-TDC-2526-1000", dc)
	}
}
