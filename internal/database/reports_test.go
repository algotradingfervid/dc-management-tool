package database

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupReportsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "reports_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	db, err := sql.Open("sqlite", tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// Create tables
	schema := `
		CREATE TABLE projects (id INTEGER PRIMARY KEY, name TEXT, created_by INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE addresses (id INTEGER PRIMARY KEY, project_id INTEGER, address_type TEXT, district_name TEXT, mandal_name TEXT, mandal_code TEXT, data TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE products (id INTEGER PRIMARY KEY, project_id INTEGER, item_name TEXT, item_description TEXT, hsn_code TEXT, uom TEXT, gst_percentage REAL, brand_model TEXT, per_unit_price REAL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE delivery_challans (
			id INTEGER PRIMARY KEY, project_id INTEGER, dc_number TEXT, dc_type TEXT, status TEXT,
			template_id INTEGER, bill_to_address_id INTEGER, ship_to_address_id INTEGER,
			challan_date TEXT, issued_at DATETIME, issued_by INTEGER, created_by INTEGER,
			bundle_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE dc_line_items (
			id INTEGER PRIMARY KEY, dc_id INTEGER, product_id INTEGER, quantity INTEGER,
			rate REAL, tax_percentage REAL, taxable_amount REAL, tax_amount REAL, total_amount REAL,
			line_order INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE serial_numbers (id INTEGER PRIMARY KEY, project_id INTEGER, line_item_id INTEGER, serial_number TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE dc_transit_details (id INTEGER PRIMARY KEY, dc_id INTEGER, transporter_name TEXT, vehicle_number TEXT, eway_bill_number TEXT, notes TEXT);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Set global DB
	DB = db
	return db
}

func seedReportsData(t *testing.T, db *sql.DB) {
	t.Helper()

	// Project 1
	db.Exec(`INSERT INTO projects (id, name, created_by) VALUES (1, 'Project Alpha', 1)`)
	// Project 2 (for scoping tests)
	db.Exec(`INSERT INTO projects (id, name, created_by) VALUES (2, 'Project Beta', 1)`)

	// Addresses
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (1, 1, 'ship_to', 'Hyderabad', 'Secunderabad')`)
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (2, 1, 'ship_to', 'Hyderabad', 'Malkajgiri')`)
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (3, 1, 'ship_to', 'Warangal', 'Hanamkonda')`)
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (4, 2, 'ship_to', 'Hyderabad', 'Secunderabad')`)

	// Products
	db.Exec(`INSERT INTO products (id, project_id, item_name) VALUES (1, 1, 'Solar Panel 250W')`)
	db.Exec(`INSERT INTO products (id, project_id, item_name) VALUES (2, 1, 'Inverter 5kW')`)

	// DCs for project 1
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, ship_to_address_id, challan_date, created_by) VALUES (1, 1, 'FSS-TDC-2526-001', 'transit', 'issued', 1, '2026-01-10', 1)`)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, ship_to_address_id, challan_date, created_by) VALUES (2, 1, 'FSS-TDC-2526-002', 'transit', 'draft', 2, '2026-01-15', 1)`)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, ship_to_address_id, challan_date, created_by) VALUES (3, 1, 'FSS-ODC-2526-001', 'official', 'issued', 1, '2026-01-12', 1)`)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, ship_to_address_id, challan_date, created_by) VALUES (4, 1, 'FSS-ODC-2526-002', 'official', 'draft', 3, '2026-02-01', 1)`)

	// DC for project 2 (should not appear in project 1 reports)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, ship_to_address_id, challan_date, created_by) VALUES (5, 2, 'FSS-TDC-2526-001', 'transit', 'issued', 4, '2026-01-10', 1)`)

	// Line items
	db.Exec(`INSERT INTO dc_line_items (id, dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order) VALUES (1, 1, 1, 10, 100, 18, 1000, 180, 1180, 1)`)
	db.Exec(`INSERT INTO dc_line_items (id, dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order) VALUES (2, 1, 2, 5, 200, 18, 1000, 180, 1180, 2)`)
	db.Exec(`INSERT INTO dc_line_items (id, dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order) VALUES (3, 3, 1, 20, 100, 18, 2000, 360, 2360, 1)`)
	db.Exec(`INSERT INTO dc_line_items (id, dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order) VALUES (4, 4, 2, 8, 200, 18, 1600, 288, 1888, 1)`)
	db.Exec(`INSERT INTO dc_line_items (id, dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order) VALUES (5, 5, 1, 15, 100, 18, 1500, 270, 1770, 1)`) // project 2

	// Serial numbers
	db.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES (1, 1, 'SN-001')`)
	db.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES (1, 1, 'SN-002')`)
	db.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES (1, 3, 'SN-003')`)
	db.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES (2, 5, 'SN-100')`) // project 2

	// Transit details
	db.Exec(`INSERT INTO dc_transit_details (dc_id, transporter_name, vehicle_number) VALUES (1, 'ABC Transport', 'TS09AB1234')`)
}

func TestDCSummaryReport(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	report, err := GetDCSummaryReport(1, nil, nil)
	if err != nil {
		t.Fatalf("GetDCSummaryReport: %v", err)
	}

	if report.TransitDraftDCs != 1 {
		t.Errorf("TransitDraftDCs = %d; want 1", report.TransitDraftDCs)
	}
	if report.TransitIssuedDCs != 1 {
		t.Errorf("TransitIssuedDCs = %d; want 1", report.TransitIssuedDCs)
	}
	if report.OfficialDraftDCs != 1 {
		t.Errorf("OfficialDraftDCs = %d; want 1", report.OfficialDraftDCs)
	}
	if report.OfficialIssuedDCs != 1 {
		t.Errorf("OfficialIssuedDCs = %d; want 1", report.OfficialIssuedDCs)
	}
	// Items dispatched: only issued DCs (DC 1: 10+5=15, DC 3: 20 = 35)
	if report.TotalItemsDispatched != 35 {
		t.Errorf("TotalItemsDispatched = %d; want 35", report.TotalItemsDispatched)
	}
	// Serial numbers: SN-001, SN-002, SN-003 = 3
	if report.TotalSerialsUsed != 3 {
		t.Errorf("TotalSerialsUsed = %d; want 3", report.TotalSerialsUsed)
	}
}

func TestDCSummaryReportWithDateRange(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	report, err := GetDCSummaryReport(1, &start, &end)
	if err != nil {
		t.Fatalf("GetDCSummaryReport with date range: %v", err)
	}

	// Only Jan DCs: DC 1 (transit issued), DC 2 (transit draft), DC 3 (official issued)
	if report.TransitIssuedDCs != 1 {
		t.Errorf("TransitIssuedDCs = %d; want 1", report.TransitIssuedDCs)
	}
	if report.OfficialDraftDCs != 0 {
		t.Errorf("OfficialDraftDCs = %d; want 0 (Feb DC excluded)", report.OfficialDraftDCs)
	}
}

func TestDCSummaryReportProjectScoped(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	report, err := GetDCSummaryReport(2, nil, nil)
	if err != nil {
		t.Fatalf("GetDCSummaryReport project 2: %v", err)
	}

	if report.TransitIssuedDCs != 1 {
		t.Errorf("Project 2 TransitIssuedDCs = %d; want 1", report.TransitIssuedDCs)
	}
	if report.OfficialDraftDCs != 0 {
		t.Errorf("Project 2 OfficialDraftDCs = %d; want 0", report.OfficialDraftDCs)
	}
	if report.TotalSerialsUsed != 1 {
		t.Errorf("Project 2 TotalSerialsUsed = %d; want 1", report.TotalSerialsUsed)
	}
}

func TestDestinationReport(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	rows, err := GetDestinationReport(1, nil, nil)
	if err != nil {
		t.Fatalf("GetDestinationReport: %v", err)
	}

	if len(rows) < 2 {
		t.Fatalf("Expected at least 2 destination rows, got %d", len(rows))
	}

	// Check that Hyderabad entries exist
	found := false
	for _, r := range rows {
		if r.District == "Hyderabad" && r.Mandal == "Secunderabad" {
			found = true
			if r.OfficialDCs != 1 {
				t.Errorf("Secunderabad OfficialDCs = %d; want 1", r.OfficialDCs)
			}
		}
	}
	if !found {
		t.Error("Expected Hyderabad/Secunderabad row not found")
	}
}

func TestDestinationDrillDown(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	dcs, err := GetDestinationDCs(1, "Hyderabad", "Secunderabad", nil, nil)
	if err != nil {
		t.Fatalf("GetDestinationDCs: %v", err)
	}

	if len(dcs) != 1 {
		t.Errorf("Expected 1 DC for Hyderabad/Secunderabad, got %d", len(dcs))
	}
	if len(dcs) > 0 && dcs[0].DCNumber != "FSS-ODC-2526-001" {
		t.Errorf("Expected FSS-ODC-2526-001, got %s", dcs[0].DCNumber)
	}
}

func TestProductReport(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	rows, err := GetProductReport(1, nil, nil)
	if err != nil {
		t.Fatalf("GetProductReport: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("Expected 2 product rows, got %d", len(rows))
	}

	// Solar Panel: DC1 (10) + DC3 (20) = 30
	for _, r := range rows {
		if r.ProductName == "Solar Panel 250W" {
			if r.TotalQty != 30 {
				t.Errorf("Solar Panel TotalQty = %d; want 30", r.TotalQty)
			}
			if r.DCCount != 2 {
				t.Errorf("Solar Panel DCCount = %d; want 2", r.DCCount)
			}
		}
	}
}

func TestSerialReport(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	// All serials for project 1
	rows, err := GetSerialReport(1, "", nil, nil)
	if err != nil {
		t.Fatalf("GetSerialReport: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("Expected 3 serials for project 1, got %d", len(rows))
	}

	// Search for specific serial
	rows, err = GetSerialReport(1, "SN-001", nil, nil)
	if err != nil {
		t.Fatalf("GetSerialReport search: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("Expected 1 serial for SN-001 search, got %d", len(rows))
	}
	if len(rows) > 0 {
		if rows[0].VehicleNumber != "TS09AB1234" {
			t.Errorf("VehicleNumber = %s; want TS09AB1234", rows[0].VehicleNumber)
		}
	}
}

func TestSerialReportProjectScoped(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	rows, err := GetSerialReport(2, "", nil, nil)
	if err != nil {
		t.Fatalf("GetSerialReport project 2: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("Expected 1 serial for project 2, got %d", len(rows))
	}
}

func TestDateRangeEdgeCases(t *testing.T) {
	db := setupReportsTestDB(t)
	seedReportsData(t, db)

	// Empty range (future dates)
	start := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC)

	report, err := GetDCSummaryReport(1, &start, &end)
	if err != nil {
		t.Fatalf("GetDCSummaryReport future: %v", err)
	}
	if report.TransitDraftDCs != 0 || report.TransitIssuedDCs != 0 {
		t.Errorf("Expected 0 DCs for future dates, got transit draft=%d, transit issued=%d",
			report.TransitDraftDCs, report.TransitIssuedDCs)
	}

	// Start only, no end
	startOnly := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	report2, err := GetDCSummaryReport(1, &startOnly, nil)
	if err != nil {
		t.Fatalf("GetDCSummaryReport start-only: %v", err)
	}
	// Only Feb DC (official draft)
	if report2.OfficialDraftDCs != 1 {
		t.Errorf("OfficialDraftDCs with Feb start = %d; want 1", report2.OfficialDraftDCs)
	}
}
