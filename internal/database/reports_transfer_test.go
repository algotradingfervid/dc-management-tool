package database

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTransferReportsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "transfer_reports_test_*.db")
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
		CREATE TABLE addresses (id INTEGER PRIMARY KEY, project_id INTEGER, address_type TEXT, district_name TEXT, mandal_name TEXT, mandal_code TEXT, data TEXT, address_data TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE products (id INTEGER PRIMARY KEY, project_id INTEGER, item_name TEXT, item_description TEXT, hsn_code TEXT, uom TEXT, gst_percentage REAL, brand_model TEXT, per_unit_price REAL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
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
		CREATE TABLE shipment_groups (
			id INTEGER PRIMARY KEY, project_id INTEGER, name TEXT, status TEXT,
			transfer_dc_id INTEGER, split_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE transfer_dcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dc_id INTEGER NOT NULL UNIQUE REFERENCES delivery_challans(id),
			hub_address_id INTEGER NOT NULL,
			template_id INTEGER,
			tax_type TEXT NOT NULL DEFAULT 'cgst_sgst',
			reverse_charge TEXT NOT NULL DEFAULT 'N',
			transporter_name TEXT, vehicle_number TEXT, eway_bill_number TEXT, docket_number TEXT, notes TEXT,
			num_destinations INTEGER NOT NULL DEFAULT 0,
			num_split INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE transfer_dc_splits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			transfer_dc_id INTEGER NOT NULL REFERENCES transfer_dcs(id),
			shipment_group_id INTEGER NOT NULL UNIQUE,
			split_number INTEGER NOT NULL,
			created_by INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE transfer_dc_destinations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			transfer_dc_id INTEGER NOT NULL REFERENCES transfer_dcs(id),
			ship_to_address_id INTEGER NOT NULL,
			split_group_id INTEGER,
			is_split INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE transfer_dc_destination_quantities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			destination_id INTEGER NOT NULL REFERENCES transfer_dc_destinations(id),
			product_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	DB = db
	return db
}

func seedTransferReportsData(t *testing.T, db *sql.DB) {
	t.Helper()

	// Projects
	db.Exec(`INSERT INTO projects (id, name, created_by) VALUES (1, 'Project Alpha', 1)`)
	db.Exec(`INSERT INTO projects (id, name, created_by) VALUES (2, 'Project Beta', 1)`)

	// Addresses
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (1, 1, 'hub', 'Hyderabad', 'Secunderabad')`)
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (2, 1, 'ship_to', 'Warangal', 'Hanamkonda')`)
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (3, 1, 'ship_to', 'Karimnagar', 'Karimnagar')`)
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (4, 1, 'ship_to', 'Nizamabad', 'Nizamabad')`)
	db.Exec(`INSERT INTO addresses (id, project_id, address_type, district_name, mandal_name) VALUES (5, 2, 'hub', 'Chennai', 'Central')`)

	// Products
	db.Exec(`INSERT INTO products (id, project_id, item_name) VALUES (1, 1, 'Solar Panel 250W')`)
	db.Exec(`INSERT INTO products (id, project_id, item_name) VALUES (2, 1, 'Inverter 5kW')`)

	// --- Existing transit/official DCs (for summary report baseline) ---
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, ship_to_address_id, challan_date, created_by) VALUES (1, 1, 'TDC-2526-001', 'transit', 'issued', 2, '2026-01-10', 1)`)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, ship_to_address_id, challan_date, created_by) VALUES (2, 1, 'TDC-2526-002', 'transit', 'draft', 3, '2026-01-15', 1)`)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, ship_to_address_id, challan_date, created_by) VALUES (3, 1, 'ODC-2526-001', 'official', 'issued', 2, '2026-01-12', 1)`)

	// --- Transfer DCs ---
	// Transfer DC 1: status=split, 3 destinations, 2 splits done (challan_date in Jan)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (10, 1, 'STDC-2526-001', 'transfer', 'split', '2026-01-20', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, transporter_name, vehicle_number, num_destinations, num_split) VALUES (1, 10, 1, 'ABC Transport', 'TS09AB1234', 3, 2)`)

	// Transfer DC 2: status=splitting, 2 destinations, 1 split done (challan_date in Feb)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (11, 1, 'STDC-2526-002', 'transfer', 'splitting', '2026-02-05', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, transporter_name, vehicle_number, num_destinations, num_split) VALUES (2, 11, 1, 'XYZ Logistics', 'TS10CD5678', 2, 1)`)

	// Transfer DC 3: status=issued, 4 destinations, 0 splits (challan_date in Jan)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (12, 1, 'STDC-2526-003', 'transfer', 'issued', '2026-01-25', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, num_destinations, num_split) VALUES (3, 12, 1, 4, 0)`)

	// Transfer DC 4: status=draft, 0 destinations (challan_date in Jan)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (13, 1, 'STDC-2526-004', 'transfer', 'draft', '2026-01-28', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, num_destinations, num_split) VALUES (4, 13, 1, 0, 0)`)

	// Transfer DC for project 2 (should NOT appear in project 1 reports)
	db.Exec(`INSERT INTO delivery_challans (id, project_id, dc_number, dc_type, status, challan_date, created_by) VALUES (20, 2, 'STDC-2526-001', 'transfer', 'issued', '2026-01-15', 1)`)
	db.Exec(`INSERT INTO transfer_dcs (id, dc_id, hub_address_id, num_destinations, num_split) VALUES (5, 20, 5, 5, 0)`)

	// --- Transfer DC Splits (for DC 1) ---
	db.Exec(`INSERT INTO shipment_groups (id, project_id, name, status, transfer_dc_id, split_id) VALUES (100, 1, 'Split Group 1', 'active', 1, 1)`)
	db.Exec(`INSERT INTO shipment_groups (id, project_id, name, status, transfer_dc_id, split_id) VALUES (101, 1, 'Split Group 2', 'active', 1, 2)`)
	db.Exec(`INSERT INTO transfer_dc_splits (id, transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (1, 1, 100, 1, 1)`)
	db.Exec(`INSERT INTO transfer_dc_splits (id, transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (2, 1, 101, 2, 1)`)

	// --- Transfer DC Splits (for DC 2 — 1 split) ---
	db.Exec(`INSERT INTO shipment_groups (id, project_id, name, status, transfer_dc_id, split_id) VALUES (102, 1, 'Split Group 3', 'active', 2, 3)`)
	db.Exec(`INSERT INTO transfer_dc_splits (id, transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (3, 2, 102, 1, 1)`)

	// --- Transfer DC Destinations ---
	// DC 1: 3 destinations (2 split, 1 unsplit)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, split_group_id, is_split) VALUES (1, 1, 2, 1, 1)`)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, split_group_id, is_split) VALUES (2, 1, 3, 2, 1)`)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, split_group_id, is_split) VALUES (3, 1, 4, NULL, 0)`)

	// DC 2: 2 destinations (1 split, 1 unsplit)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, split_group_id, is_split) VALUES (4, 2, 2, 3, 1)`)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, split_group_id, is_split) VALUES (5, 2, 3, NULL, 0)`)

	// DC 3: 4 destinations (none split)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, is_split) VALUES (6, 3, 2, 0)`)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, is_split) VALUES (7, 3, 3, 0)`)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, is_split) VALUES (8, 3, 4, 0)`)
	db.Exec(`INSERT INTO transfer_dc_destinations (id, transfer_dc_id, ship_to_address_id, is_split) VALUES (9, 3, 2, 0)`) // duplicate address is fine

	// --- Destination Quantities ---
	db.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (1, 1, 10)`)
	db.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (1, 2, 5)`)
	db.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (2, 1, 8)`)
	db.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (3, 1, 12)`)
	db.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (4, 1, 15)`)
	db.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (5, 2, 7)`)
}

// --- DC Summary Report Tests ---

func TestDCSummaryReport_IncludesTransferDCs(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	report, err := GetDCSummaryReport(1, nil, nil)
	if err != nil {
		t.Fatalf("GetDCSummaryReport: %v", err)
	}

	// Existing transit/official DCs should still be counted
	if report.TransitDraftDCs != 1 {
		t.Errorf("TransitDraftDCs = %d; want 1", report.TransitDraftDCs)
	}
	if report.TransitIssuedDCs != 1 {
		t.Errorf("TransitIssuedDCs = %d; want 1", report.TransitIssuedDCs)
	}
	if report.OfficialIssuedDCs != 1 {
		t.Errorf("OfficialIssuedDCs = %d; want 1", report.OfficialIssuedDCs)
	}

	// Transfer DC counts
	if report.TransferDraftDCs != 1 {
		t.Errorf("TransferDraftDCs = %d; want 1", report.TransferDraftDCs)
	}
	if report.TransferIssuedDCs != 1 {
		t.Errorf("TransferIssuedDCs = %d; want 1", report.TransferIssuedDCs)
	}
	if report.TransferSplittingDCs != 1 {
		t.Errorf("TransferSplittingDCs = %d; want 1", report.TransferSplittingDCs)
	}
	if report.TransferSplitDCs != 1 {
		t.Errorf("TransferSplitDCs = %d; want 1", report.TransferSplitDCs)
	}
}

func TestDCSummaryReport_TransferStatusBreakdown(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	// Date filter: only January (should include 3 of 4 Transfer DCs)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	report, err := GetDCSummaryReport(1, &start, &end)
	if err != nil {
		t.Fatalf("GetDCSummaryReport with date range: %v", err)
	}

	// Jan Transfer DCs: STDC-001 (split), STDC-003 (issued), STDC-004 (draft)
	// STDC-002 (splitting) is Feb, excluded
	if report.TransferDraftDCs != 1 {
		t.Errorf("TransferDraftDCs (Jan) = %d; want 1", report.TransferDraftDCs)
	}
	if report.TransferIssuedDCs != 1 {
		t.Errorf("TransferIssuedDCs (Jan) = %d; want 1", report.TransferIssuedDCs)
	}
	if report.TransferSplittingDCs != 0 {
		t.Errorf("TransferSplittingDCs (Jan) = %d; want 0", report.TransferSplittingDCs)
	}
	if report.TransferSplitDCs != 1 {
		t.Errorf("TransferSplitDCs (Jan) = %d; want 1", report.TransferSplitDCs)
	}
}

func TestDCSummaryReport_TransferProjectScoped(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	report, err := GetDCSummaryReport(2, nil, nil)
	if err != nil {
		t.Fatalf("GetDCSummaryReport project 2: %v", err)
	}

	// Project 2 has 1 issued Transfer DC
	if report.TransferIssuedDCs != 1 {
		t.Errorf("Project 2 TransferIssuedDCs = %d; want 1", report.TransferIssuedDCs)
	}
	if report.TransferDraftDCs != 0 {
		t.Errorf("Project 2 TransferDraftDCs = %d; want 0", report.TransferDraftDCs)
	}
	if report.TransferSplittingDCs != 0 {
		t.Errorf("Project 2 TransferSplittingDCs = %d; want 0", report.TransferSplittingDCs)
	}
	if report.TransferSplitDCs != 0 {
		t.Errorf("Project 2 TransferSplitDCs = %d; want 0", report.TransferSplitDCs)
	}
}

// --- Transfer DC Report Tests ---

func TestTransferDCReport_HappyPath(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	rows, err := GetTransferDCReport(1, nil, nil)
	if err != nil {
		t.Fatalf("GetTransferDCReport: %v", err)
	}

	// Project 1 has 4 Transfer DCs
	if len(rows) != 4 {
		t.Fatalf("Expected 4 Transfer DC rows, got %d", len(rows))
	}

	// Verify ordering: most recent challan_date first
	// STDC-002 (Feb 05), STDC-004 (Jan 28), STDC-003 (Jan 25), STDC-001 (Jan 20)
	if rows[0].DCNumber != "STDC-2526-002" {
		t.Errorf("First row DC number = %s; want STDC-2526-002", rows[0].DCNumber)
	}
	if rows[0].Status != "splitting" {
		t.Errorf("First row status = %s; want splitting", rows[0].Status)
	}
	if rows[0].NumDestinations != 2 {
		t.Errorf("STDC-002 NumDestinations = %d; want 2", rows[0].NumDestinations)
	}
	if rows[0].SplitCount != 1 {
		t.Errorf("STDC-002 SplitCount = %d; want 1", rows[0].SplitCount)
	}

	// Check STDC-001 (split, 3 destinations, 2 splits)
	var stdc001 *TransferDCReportRow
	for i := range rows {
		if rows[i].DCNumber == "STDC-2526-001" {
			stdc001 = &rows[i]
			break
		}
	}
	if stdc001 == nil {
		t.Fatal("STDC-2526-001 not found in report")
	}
	if stdc001.Status != "split" {
		t.Errorf("STDC-001 Status = %s; want split", stdc001.Status)
	}
	if stdc001.NumDestinations != 3 {
		t.Errorf("STDC-001 NumDestinations = %d; want 3", stdc001.NumDestinations)
	}
	if stdc001.SplitCount != 2 {
		t.Errorf("STDC-001 SplitCount = %d; want 2", stdc001.SplitCount)
	}
	if stdc001.TransporterName != "ABC Transport" {
		t.Errorf("STDC-001 TransporterName = %s; want ABC Transport", stdc001.TransporterName)
	}
	if stdc001.VehicleNumber != "TS09AB1234" {
		t.Errorf("STDC-001 VehicleNumber = %s; want TS09AB1234", stdc001.VehicleNumber)
	}
}

func TestTransferDCReport_DateFilter(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	// Only January
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	rows, err := GetTransferDCReport(1, &start, &end)
	if err != nil {
		t.Fatalf("GetTransferDCReport with date filter: %v", err)
	}

	// Jan Transfer DCs: STDC-001, STDC-003, STDC-004
	if len(rows) != 3 {
		t.Fatalf("Expected 3 Transfer DC rows for Jan, got %d", len(rows))
	}

	// Verify STDC-002 (Feb) is excluded
	for _, r := range rows {
		if r.DCNumber == "STDC-2526-002" {
			t.Error("STDC-2526-002 (Feb) should be excluded from Jan date range")
		}
	}
}

func TestTransferDCReport_ProjectScoped(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	rows, err := GetTransferDCReport(2, nil, nil)
	if err != nil {
		t.Fatalf("GetTransferDCReport project 2: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("Expected 1 Transfer DC row for project 2, got %d", len(rows))
	}
	if rows[0].DCNumber != "STDC-2526-001" {
		t.Errorf("Project 2 DC number = %s; want STDC-2526-001", rows[0].DCNumber)
	}
}

// --- DC Listing Filter Tests ---

func TestDCListFilter_TransferType(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	result, err := GetAllDCsFiltered(DCListFilters{
		ProjectID: "1",
		DCType:    "transfer",
		Status:    "all",
		Page:      1,
		PageSize:  25,
	})
	if err != nil {
		t.Fatalf("GetAllDCsFiltered(transfer): %v", err)
	}

	// Project 1 has 4 Transfer DCs
	if result.TotalCount != 4 {
		t.Errorf("Transfer type filter: TotalCount = %d; want 4", result.TotalCount)
	}
	for _, dc := range result.DCs {
		if dc.DCType != "transfer" {
			t.Errorf("Expected all DCs to be type 'transfer', got '%s' for %s", dc.DCType, dc.DCNumber)
		}
	}
}

func TestDCListFilter_AllTypes_IncludesTransfer(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	result, err := GetAllDCsFiltered(DCListFilters{
		ProjectID: "1",
		DCType:    "all",
		Status:    "all",
		Page:      1,
		PageSize:  25,
	})
	if err != nil {
		t.Fatalf("GetAllDCsFiltered(all): %v", err)
	}

	// Project 1: 3 transit/official + 4 transfer = 7 DCs total
	if result.TotalCount != 7 {
		t.Errorf("All types filter: TotalCount = %d; want 7", result.TotalCount)
	}

	// Verify Transfer DCs are included
	transferCount := 0
	for _, dc := range result.DCs {
		if dc.DCType == "transfer" {
			transferCount++
		}
	}
	if transferCount != 4 {
		t.Errorf("All types filter: transfer DC count = %d; want 4", transferCount)
	}
}

func TestDCListFilter_SplittingStatus(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	result, err := GetAllDCsFiltered(DCListFilters{
		ProjectID: "1",
		DCType:    "all",
		Status:    "splitting",
		Page:      1,
		PageSize:  25,
	})
	if err != nil {
		t.Fatalf("GetAllDCsFiltered(splitting): %v", err)
	}

	if result.TotalCount != 1 {
		t.Errorf("Splitting status filter: TotalCount = %d; want 1", result.TotalCount)
	}
	if len(result.DCs) > 0 && result.DCs[0].DCNumber != "STDC-2526-002" {
		t.Errorf("Splitting status: got %s; want STDC-2526-002", result.DCs[0].DCNumber)
	}
}

func TestDCListFilter_SplitStatus(t *testing.T) {
	db := setupTransferReportsTestDB(t)
	seedTransferReportsData(t, db)

	result, err := GetAllDCsFiltered(DCListFilters{
		ProjectID: "1",
		DCType:    "all",
		Status:    "split",
		Page:      1,
		PageSize:  25,
	})
	if err != nil {
		t.Fatalf("GetAllDCsFiltered(split): %v", err)
	}

	if result.TotalCount != 1 {
		t.Errorf("Split status filter: TotalCount = %d; want 1", result.TotalCount)
	}
	if len(result.DCs) > 0 && result.DCs[0].DCNumber != "STDC-2526-001" {
		t.Errorf("Split status: got %s; want STDC-2526-001", result.DCs[0].DCNumber)
	}
}
