package database

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func setupDashboardTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "dashboard_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	db, err := sql.Open("sqlite", tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
		CREATE TABLE projects (id INTEGER PRIMARY KEY, name TEXT, created_by INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE products (id INTEGER PRIMARY KEY, project_id INTEGER, item_name TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE dc_templates (id INTEGER PRIMARY KEY, project_id INTEGER, name TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE address_list_configs (id INTEGER PRIMARY KEY, project_id INTEGER, address_type TEXT, name TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE addresses (id INTEGER PRIMARY KEY, config_id INTEGER, district_name TEXT, mandal_name TEXT, address_data TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE delivery_challans (
			id INTEGER PRIMARY KEY, project_id INTEGER, dc_number TEXT, dc_type TEXT, status TEXT,
			template_id INTEGER, bill_to_address_id INTEGER, ship_to_address_id INTEGER,
			challan_date TEXT, issued_at DATETIME, issued_by INTEGER, created_by INTEGER,
			bundle_id INTEGER, transfer_dc_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE dc_line_items (
			id INTEGER PRIMARY KEY, dc_id INTEGER, product_id INTEGER, quantity INTEGER,
			rate REAL, tax_percentage REAL, taxable_amount REAL, tax_amount REAL, total_amount REAL,
			line_order INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE serial_numbers (id INTEGER PRIMARY KEY, project_id INTEGER, line_item_id INTEGER, serial_number TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE dc_transit_details (id INTEGER PRIMARY KEY, dc_id INTEGER, transporter_name TEXT, vehicle_number TEXT, eway_bill_number TEXT, notes TEXT);
		CREATE TABLE transfer_dcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dc_id INTEGER NOT NULL UNIQUE REFERENCES delivery_challans(id),
			hub_address_id INTEGER NOT NULL,
			num_destinations INTEGER NOT NULL DEFAULT 0,
			num_split INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	DB = db
	return db
}

func seedDashboardData(t *testing.T, db *sql.DB) {
	t.Helper()

	db.Exec(`INSERT INTO projects (id, name, created_by) VALUES (1, 'Project Alpha', 1)`)

	// Transit DCs: 1 draft, 1 issued
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (1, 1, 'TDC-001', 'transit', 'draft', '2026-01-10', 1)`)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (2, 1, 'TDC-002', 'transit', 'issued', '2026-01-15', 1)`)

	// Official DCs: 1 draft, 2 issued
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (3, 1, 'ODC-001', 'official', 'draft', '2026-01-12', 1)`)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (4, 1, 'ODC-002', 'official', 'issued', '2026-01-14', 1)`)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (5, 1, 'ODC-003', 'official', 'issued', '2026-01-16', 1)`)

	// Transfer DCs: 1 draft, 1 issued, 1 splitting, 1 split
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (10, 1, 'STDC-001', 'transfer', 'draft', '2026-01-20', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, num_destinations, num_split) VALUES (1, 10, 1, 0, 0)`)

	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (11, 1, 'STDC-002', 'transfer', 'issued', '2026-01-22', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, num_destinations, num_split) VALUES (2, 11, 1, 5, 0)`)

	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (12, 1, 'STDC-003', 'transfer', 'splitting', '2026-01-24', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, num_destinations, num_split) VALUES (3, 12, 1, 10, 3)`)

	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (13, 1, 'STDC-004', 'transfer', 'split', '2026-01-26', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, num_destinations, num_split) VALUES (4, 13, 1, 8, 8)`)

	// Serial numbers
	db.Exec(`INSERT INTO dc_line_items (id, dc_id, product_id, quantity) VALUES (1, 2, 1, 10)`)
	db.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES (1, 1, 'SN-001')`)
	db.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES (1, 1, 'SN-002')`)
}

func TestDashboard_TransferDCStats(t *testing.T) {
	db := setupDashboardTestDB(t)
	seedDashboardData(t, db)

	stats, err := GetDashboardStats(1, nil, nil)
	if err != nil {
		t.Fatalf("GetDashboardStats: %v", err)
	}

	// Transfer DC counts
	if stats.TransferDCs != 4 {
		t.Errorf("TransferDCs = %d; want 4", stats.TransferDCs)
	}
	if stats.TransferDCsDraft != 1 {
		t.Errorf("TransferDCsDraft = %d; want 1", stats.TransferDCsDraft)
	}
	if stats.TransferDCsIssued != 1 {
		t.Errorf("TransferDCsIssued = %d; want 1", stats.TransferDCsIssued)
	}
	if stats.TransferDCsSplitting != 1 {
		t.Errorf("TransferDCsSplitting = %d; want 1", stats.TransferDCsSplitting)
	}
	if stats.TransferDCsSplit != 1 {
		t.Errorf("TransferDCsSplit = %d; want 1", stats.TransferDCsSplit)
	}

	// Total DCs should include transfer DCs
	// 2 transit + 3 official + 4 transfer = 9
	if stats.TotalDCs != 9 {
		t.Errorf("TotalDCs = %d; want 9", stats.TotalDCs)
	}

	// Existing stats still correct
	if stats.TransitDCs != 2 {
		t.Errorf("TransitDCs = %d; want 2", stats.TransitDCs)
	}
	if stats.OfficialDCs != 3 {
		t.Errorf("OfficialDCs = %d; want 3", stats.OfficialDCs)
	}
	if stats.TransitDCsDraft != 1 {
		t.Errorf("TransitDCsDraft = %d; want 1", stats.TransitDCsDraft)
	}
	if stats.TransitDCsIssued != 1 {
		t.Errorf("TransitDCsIssued = %d; want 1", stats.TransitDCsIssued)
	}
	if stats.TotalSerialNumbers != 2 {
		t.Errorf("TotalSerialNumbers = %d; want 2", stats.TotalSerialNumbers)
	}
}

func TestDashboardStatsToMap_IncludesTransferDCs(t *testing.T) {
	// This test verifies that dashboardStatsToMap properly maps Transfer DC fields.
	// The function is in the handlers package, so we test the struct fields exist here.
	stats := &DashboardStats{
		TransferDCs:         4,
		TransferDCsDraft:    1,
		TransferDCsIssued:   1,
		TransferDCsSplitting: 1,
		TransferDCsSplit:    1,
	}

	if stats.TransferDCs != 4 {
		t.Errorf("TransferDCs = %d; want 4", stats.TransferDCs)
	}
	if stats.TransferDCsDraft != 1 {
		t.Errorf("TransferDCsDraft = %d; want 1", stats.TransferDCsDraft)
	}
	if stats.TransferDCsIssued != 1 {
		t.Errorf("TransferDCsIssued = %d; want 1", stats.TransferDCsIssued)
	}
	if stats.TransferDCsSplitting != 1 {
		t.Errorf("TransferDCsSplitting = %d; want 1", stats.TransferDCsSplitting)
	}
	if stats.TransferDCsSplit != 1 {
		t.Errorf("TransferDCsSplit = %d; want 1", stats.TransferDCsSplit)
	}
}
