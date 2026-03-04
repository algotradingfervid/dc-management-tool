package database

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/narendhupati/dc-management-tool/internal/models"
	_ "modernc.org/sqlite"
)

func setupDCTestDB(t *testing.T) func() {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&mode=memory")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	db.Exec("PRAGMA foreign_keys = ON")

	// Create all required tables
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS projects (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            created_by INTEGER DEFAULT 1,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
		`INSERT OR IGNORE INTO projects (id, name) VALUES (1, 'Test Project')`,
		`CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL,
            password_hash TEXT NOT NULL,
            role TEXT DEFAULT 'user'
        )`,
		`INSERT OR IGNORE INTO users (id, username, password_hash) VALUES (1, 'testuser', 'hash')`,
		`CREATE TABLE IF NOT EXISTS dc_templates (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            project_id INTEGER
        )`,
		`CREATE TABLE IF NOT EXISTS addresses (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            config_id INTEGER,
            address_data TEXT DEFAULT '{}',
            district_name TEXT DEFAULT '',
            mandal_name TEXT DEFAULT '',
            mandal_code TEXT DEFAULT '',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
		`INSERT OR IGNORE INTO addresses (id, address_data) VALUES (1, '{}'), (2, '{}')`,
		`CREATE TABLE IF NOT EXISTS shipment_groups (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            project_id INTEGER NOT NULL REFERENCES projects(id),
            template_id INTEGER REFERENCES dc_templates(id),
            num_sets INTEGER NOT NULL DEFAULT 1,
            tax_type TEXT NOT NULL DEFAULT 'cgst_sgst',
            reverse_charge TEXT NOT NULL DEFAULT 'N',
            status TEXT NOT NULL DEFAULT 'draft',
            created_by INTEGER REFERENCES users(id),
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS delivery_challans (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            project_id INTEGER NOT NULL,
            dc_number TEXT NOT NULL,
            dc_type TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'draft',
            template_id INTEGER,
            bill_to_address_id INTEGER,
            ship_to_address_id INTEGER NOT NULL,
            challan_date DATE,
            issued_at DATETIME,
            issued_by INTEGER,
            created_by INTEGER NOT NULL DEFAULT 1,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            bundle_id INTEGER,
            shipment_group_id INTEGER,
            bill_from_address_id INTEGER,
            dispatch_from_address_id INTEGER,
            UNIQUE(project_id, dc_number)
        )`,
		`CREATE TABLE IF NOT EXISTS dc_transit_details (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            dc_id INTEGER NOT NULL UNIQUE,
            transporter_name TEXT,
            vehicle_number TEXT,
            eway_bill_number TEXT,
            notes TEXT,
            FOREIGN KEY (dc_id) REFERENCES delivery_challans(id) ON DELETE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS products (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            item_name TEXT NOT NULL,
            item_description TEXT DEFAULT '',
            hsn_code TEXT DEFAULT '',
            uom TEXT DEFAULT 'nos',
            brand_model TEXT DEFAULT '',
            gst_percentage REAL DEFAULT 0
        )`,
		`INSERT OR IGNORE INTO products (id, item_name) VALUES (1, 'Test Product')`,
		`CREATE TABLE IF NOT EXISTS dc_line_items (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            dc_id INTEGER NOT NULL,
            product_id INTEGER NOT NULL,
            quantity INTEGER NOT NULL DEFAULT 1,
            rate REAL,
            tax_percentage REAL,
            taxable_amount REAL,
            tax_amount REAL,
            total_amount REAL,
            line_order INTEGER,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (dc_id) REFERENCES delivery_challans(id) ON DELETE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS serial_numbers (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            project_id INTEGER NOT NULL,
            line_item_id INTEGER NOT NULL,
            serial_number TEXT NOT NULL,
            product_id INTEGER,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (line_item_id) REFERENCES dc_line_items(id) ON DELETE CASCADE,
            UNIQUE(project_id, serial_number)
        )`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("setup stmt failed:\n%s\nerr: %v", s, err)
		}
	}

	DB = db
	return func() { db.Close() }
}

// helper: insert a minimal delivery challan and return its ID
func insertTestDC(t *testing.T, projectID int, dcNumber, dcType string, shipToAddrID int) int {
	t.Helper()
	res, err := DB.Exec(
		`INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, ship_to_address_id, created_by)
         VALUES (?, ?, ?, 'draft', ?, 1)`,
		projectID, dcNumber, dcType, shipToAddrID,
	)
	if err != nil {
		t.Fatalf("insertTestDC failed: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func TestUpdateShipmentGroup(t *testing.T) {
	cleanup := setupDCTestDB(t)
	defer cleanup()

	// Insert a draft shipment group
	res, err := DB.Exec(
		`INSERT INTO shipment_groups (project_id, num_sets, tax_type, reverse_charge, status) VALUES (1, 2, 'cgst_sgst', 'N', 'draft')`,
	)
	if err != nil {
		t.Fatalf("insert shipment group: %v", err)
	}
	groupID64, _ := res.LastInsertId()
	groupID := int(groupID64)

	// Update it
	numSets := 5
	taxType := "igst"
	reverseCharge := "Y"
	if err := UpdateShipmentGroup(groupID, nil, numSets, taxType, reverseCharge); err != nil {
		t.Fatalf("UpdateShipmentGroup failed: %v", err)
	}

	// Verify
	var gotNumSets int
	var gotTaxType, gotReverseCharge string
	row := DB.QueryRow(`SELECT num_sets, tax_type, reverse_charge FROM shipment_groups WHERE id = ?`, groupID)
	if err := row.Scan(&gotNumSets, &gotTaxType, &gotReverseCharge); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if gotNumSets != numSets {
		t.Errorf("num_sets: want %d, got %d", numSets, gotNumSets)
	}
	if gotTaxType != taxType {
		t.Errorf("tax_type: want %q, got %q", taxType, gotTaxType)
	}
	if gotReverseCharge != reverseCharge {
		t.Errorf("reverse_charge: want %q, got %q", reverseCharge, gotReverseCharge)
	}
}

func TestUpdateShipmentGroup_IgnoresNonDraft(t *testing.T) {
	cleanup := setupDCTestDB(t)
	defer cleanup()

	// Insert an ISSUED shipment group
	res, _ := DB.Exec(
		`INSERT INTO shipment_groups (project_id, num_sets, tax_type, reverse_charge, status) VALUES (1, 2, 'cgst_sgst', 'N', 'issued')`,
	)
	groupID64, _ := res.LastInsertId()
	groupID := int(groupID64)

	// UpdateShipmentGroup should silently no-op (WHERE status='draft' won't match)
	if err := UpdateShipmentGroup(groupID, nil, 9, "igst", "Y"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var gotNumSets int
	DB.QueryRow(`SELECT num_sets FROM shipment_groups WHERE id = ?`, groupID).Scan(&gotNumSets)
	if gotNumSets != 2 {
		t.Errorf("issued group should not be updated; want 2, got %d", gotNumSets)
	}
}

func TestUpdateTransitDC(t *testing.T) {
	cleanup := setupDCTestDB(t)
	defer cleanup()

	dcID := insertTestDC(t, 1, "TDC-001", "transit", 1)

	// Insert transit details row
	if _, err := DB.Exec(
		`INSERT INTO dc_transit_details (dc_id, transporter_name, vehicle_number, eway_bill_number, notes) VALUES (?, 'OldCo', 'OLD-001', 'EW-001', 'old note')`,
		dcID,
	); err != nil {
		t.Fatalf("insert transit details: %v", err)
	}

	date := "2025-04-01"
	if err := UpdateTransitDC(dcID, &date, "NewCo", "NEW-001", "EW-999", "new note"); err != nil {
		t.Fatalf("UpdateTransitDC failed: %v", err)
	}

	var transporter, vehicle, eway, notes string
	var challanDate sql.NullString
	DB.QueryRow(`SELECT challan_date FROM delivery_challans WHERE id = ?`, dcID).Scan(&challanDate)
	DB.QueryRow(`SELECT transporter_name, vehicle_number, eway_bill_number, notes FROM dc_transit_details WHERE dc_id = ?`, dcID).
		Scan(&transporter, &vehicle, &eway, &notes)

	if !strings.Contains(challanDate.String, "2025-04-01") {
		t.Errorf("challan_date: want 2025-04-01, got %q", challanDate.String)
	}
	if transporter != "NewCo" {
		t.Errorf("transporter_name: want NewCo, got %q", transporter)
	}
	if vehicle != "NEW-001" {
		t.Errorf("vehicle_number: want NEW-001, got %q", vehicle)
	}
	if notes != "new note" {
		t.Errorf("notes: want 'new note', got %q", notes)
	}
}

func TestReplaceLineItemsAndSerials(t *testing.T) {
	cleanup := setupDCTestDB(t)
	defer cleanup()

	dcID := insertTestDC(t, 1, "ODC-001", "official", 1)

	// Insert an old line item + serial
	liRes, _ := DB.Exec(`INSERT INTO dc_line_items (dc_id, product_id, quantity, line_order) VALUES (?, 1, 3, 1)`, dcID)
	oldLIID, _ := liRes.LastInsertId()
	DB.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number, product_id) VALUES (1, ?, 'OLD-SN-001', 1)`, oldLIID)

	// Replace with new items
	newItems := []models.DCLineItem{
		{ProductID: 1, Quantity: 5, Rate: 100.0},
	}
	newSerials := [][]string{{"SN-001", "SN-002"}}

	if err := ReplaceLineItemsAndSerials(dcID, 1, newItems, newSerials); err != nil {
		t.Fatalf("ReplaceLineItemsAndSerials failed: %v", err)
	}

	// Old line item should be gone
	var oldCount int
	DB.QueryRow(`SELECT COUNT(*) FROM dc_line_items WHERE id = ?`, oldLIID).Scan(&oldCount)
	if oldCount != 0 {
		t.Error("old line item should have been deleted")
	}

	// Old serial should be gone
	var oldSNCount int
	DB.QueryRow(`SELECT COUNT(*) FROM serial_numbers WHERE serial_number = 'OLD-SN-001'`).Scan(&oldSNCount)
	if oldSNCount != 0 {
		t.Error("old serial should have been deleted")
	}

	// New items and serials should exist
	var newLICount int
	DB.QueryRow(`SELECT COUNT(*) FROM dc_line_items WHERE dc_id = ?`, dcID).Scan(&newLICount)
	if newLICount != 1 {
		t.Errorf("want 1 new line item, got %d", newLICount)
	}

	var newSNCount int
	DB.QueryRow(`SELECT COUNT(*) FROM serial_numbers WHERE serial_number IN ('SN-001','SN-002')`).Scan(&newSNCount)
	if newSNCount != 2 {
		t.Errorf("want 2 new serials, got %d", newSNCount)
	}
}

func TestDeleteOfficialDC(t *testing.T) {
	cleanup := setupDCTestDB(t)
	defer cleanup()

	dcID := insertTestDC(t, 1, "ODC-DEL-001", "official", 1)

	// Insert a line item + serial
	liRes, _ := DB.Exec(`INSERT INTO dc_line_items (dc_id, product_id, quantity, line_order) VALUES (?, 1, 2, 1)`, dcID)
	liID, _ := liRes.LastInsertId()
	DB.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number, product_id) VALUES (1, ?, 'DEL-SN-001', 1)`, liID)

	if err := DeleteOfficialDC(dcID); err != nil {
		t.Fatalf("DeleteOfficialDC failed: %v", err)
	}

	// DC should be gone
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM delivery_challans WHERE id = ?`, dcID).Scan(&count)
	if count != 0 {
		t.Error("DC should have been deleted")
	}

	// Line item should be gone (cascade or explicit delete in DeleteDC)
	var liCount int
	DB.QueryRow(`SELECT COUNT(*) FROM dc_line_items WHERE id = ?`, liID).Scan(&liCount)
	if liCount != 0 {
		t.Error("line item should have been deleted")
	}
}
