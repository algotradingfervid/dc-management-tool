package services

import (
	"database/sql"
	"testing"
)

// setupDCGenTestDB creates an in-memory SQLite database with the tables
// required by CreateShipmentGroupDCs.
func setupDCGenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db := setupTestDB(t) // reuses the helper from dc_numbering_test.go

	// Additional tables needed for DC generation
	stmts := []string{
		`CREATE TABLE shipment_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			template_id INTEGER,
			num_sets INTEGER NOT NULL DEFAULT 1,
			tax_type TEXT NOT NULL DEFAULT 'igst',
			reverse_charge TEXT NOT NULL DEFAULT 'N',
			status TEXT NOT NULL DEFAULT 'draft',
			created_by INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id)
		)`,
		`CREATE TABLE delivery_challans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			dc_number TEXT NOT NULL,
			dc_type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			template_id INTEGER,
			bill_to_address_id INTEGER,
			ship_to_address_id INTEGER NOT NULL,
			challan_date TEXT,
			created_by INTEGER NOT NULL DEFAULT 1,
			shipment_group_id INTEGER,
			bill_from_address_id INTEGER,
			dispatch_from_address_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id)
		)`,
		`CREATE TABLE dc_transit_details (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dc_id INTEGER NOT NULL,
			transporter_name TEXT,
			vehicle_number TEXT,
			eway_bill_number TEXT,
			notes TEXT,
			FOREIGN KEY (dc_id) REFERENCES delivery_challans(id)
		)`,
		`CREATE TABLE dc_line_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dc_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL,
			rate REAL NOT NULL DEFAULT 0,
			tax_percentage REAL NOT NULL DEFAULT 0,
			taxable_amount REAL NOT NULL DEFAULT 0,
			tax_amount REAL NOT NULL DEFAULT 0,
			total_amount REAL NOT NULL DEFAULT 0,
			line_order INTEGER NOT NULL DEFAULT 1,
			FOREIGN KEY (dc_id) REFERENCES delivery_challans(id)
		)`,
		`CREATE TABLE serial_numbers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			line_item_id INTEGER NOT NULL,
			serial_number TEXT NOT NULL,
			product_id INTEGER NOT NULL,
			FOREIGN KEY (line_item_id) REFERENCES dc_line_items(id)
		)`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("failed to create table: %v\nSQL: %s", err, s)
		}
	}

	return db
}

// getLineItemQty fetches the quantity for a specific dc_id + product_id.
func getLineItemQty(t *testing.T, db *sql.DB, dcID, productID int) int {
	t.Helper()
	var qty int
	err := db.QueryRow("SELECT quantity FROM dc_line_items WHERE dc_id = ? AND product_id = ?", dcID, productID).Scan(&qty)
	if err != nil {
		t.Fatalf("getLineItemQty(dc=%d, product=%d): %v", dcID, productID, err)
	}
	return qty
}

func TestCreateShipmentGroupDCs_CustomQuantities(t *testing.T) {
	db := setupDCGenTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "TestProject", "TST")

	// 2 ship-to addresses (IDs 100, 200)
	// Product A (ID=1): 5 units to addr 100, 3 units to addr 200 → transit total = 8
	// Product B (ID=2): 10 units to addr 100, 0 units to addr 200 → transit total = 10
	params := ShipmentParams{
		ProjectID:             projectID,
		TemplateID:            1,
		NumLocations:          2,
		ChallanDate:           "2026-01-15",
		TaxType:               "igst",
		ReverseCharge:         "N",
		TransporterName:       "Test Transport",
		VehicleNumber:         "KA01AB1234",
		BillFromAddressID:     1,
		DispatchFromAddressID: 1,
		BillToAddressID:       1,
		ShipToAddressIDs:      []int{100, 200},
		TransitShipToAddrID:   100,
		LineItems: []ShipmentLineItem{
			{
				ProductID:     1,
				QtyByLocation: map[int]int{100: 5, 200: 3},
				Rate:          100.0,
				TaxPercentage: 18.0,
			},
			{
				ProductID:     2,
				QtyByLocation: map[int]int{100: 10, 200: 0},
				Rate:          50.0,
				TaxPercentage: 18.0,
			},
		},
		CreatedBy: 1,
	}

	result, err := CreateShipmentGroupDCs(db, params)
	if err != nil {
		t.Fatalf("CreateShipmentGroupDCs failed: %v", err)
	}

	// Transit DC: product 1 qty=8, product 2 qty=10
	transitQty1 := getLineItemQty(t, db, result.TransitDC.ID, 1)
	if transitQty1 != 8 {
		t.Errorf("transit DC product 1: got qty %d, want 8", transitQty1)
	}
	transitQty2 := getLineItemQty(t, db, result.TransitDC.ID, 2)
	if transitQty2 != 10 {
		t.Errorf("transit DC product 2: got qty %d, want 10", transitQty2)
	}

	// Should have 1 official DC (addr 200 has product 2 qty=0, but product 1 qty=3 > 0)
	// Actually addr 200 has product 1 qty=3, product 2 qty=0 — hasQty is true because product 1 > 0
	// addr 100: product 1 qty=5, product 2 qty=10 — hasQty is true
	// So 2 official DCs
	if len(result.OfficialDCs) != 2 {
		t.Fatalf("expected 2 official DCs, got %d", len(result.OfficialDCs))
	}

	// Find official DC for addr 100
	var offDC100, offDC200 int
	for _, dc := range result.OfficialDCs {
		if dc.ShipToAddressID == 100 {
			offDC100 = dc.ID
		} else if dc.ShipToAddressID == 200 {
			offDC200 = dc.ID
		}
	}

	// Official DC for addr 100: product 1 qty=5, product 2 qty=10
	if qty := getLineItemQty(t, db, offDC100, 1); qty != 5 {
		t.Errorf("official DC addr=100, product 1: got qty %d, want 5", qty)
	}
	if qty := getLineItemQty(t, db, offDC100, 2); qty != 10 {
		t.Errorf("official DC addr=100, product 2: got qty %d, want 10", qty)
	}

	// Official DC for addr 200: product 1 qty=3, product 2 qty=0
	if qty := getLineItemQty(t, db, offDC200, 1); qty != 3 {
		t.Errorf("official DC addr=200, product 1: got qty %d, want 3", qty)
	}
	if qty := getLineItemQty(t, db, offDC200, 2); qty != 0 {
		t.Errorf("official DC addr=200, product 2: got qty %d, want 0", qty)
	}
}

func TestCreateShipmentGroupDCs_ZeroQtyLocationSkipped(t *testing.T) {
	db := setupDCGenTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "TestProject", "TST")

	// 3 ship-to addresses: 100, 200, 300
	// Product A: addr 100=5, addr 200=0, addr 300=3
	// Product B: addr 100=2, addr 200=0, addr 300=1
	// addr 200 has ALL zero → should NOT generate an official DC
	params := ShipmentParams{
		ProjectID:             projectID,
		TemplateID:            1,
		NumLocations:          3,
		ChallanDate:           "2026-01-15",
		TaxType:               "igst",
		ReverseCharge:         "N",
		TransporterName:       "Test Transport",
		VehicleNumber:         "KA01AB1234",
		BillFromAddressID:     1,
		DispatchFromAddressID: 1,
		BillToAddressID:       1,
		ShipToAddressIDs:      []int{100, 200, 300},
		TransitShipToAddrID:   100,
		LineItems: []ShipmentLineItem{
			{
				ProductID:     1,
				QtyByLocation: map[int]int{100: 5, 200: 0, 300: 3},
				Rate:          100.0,
				TaxPercentage: 18.0,
			},
			{
				ProductID:     2,
				QtyByLocation: map[int]int{100: 2, 200: 0, 300: 1},
				Rate:          50.0,
				TaxPercentage: 18.0,
			},
		},
		CreatedBy: 1,
	}

	result, err := CreateShipmentGroupDCs(db, params)
	if err != nil {
		t.Fatalf("CreateShipmentGroupDCs failed: %v", err)
	}

	// Should have 2 official DCs (addr 200 skipped)
	if len(result.OfficialDCs) != 2 {
		t.Fatalf("expected 2 official DCs (addr 200 skipped), got %d", len(result.OfficialDCs))
	}

	// Verify that none of the official DCs are for addr 200
	for _, dc := range result.OfficialDCs {
		if dc.ShipToAddressID == 200 {
			t.Error("official DC created for addr 200 which has all-zero quantities — should have been skipped")
		}
	}

	// Transit DC should still have total qty: product 1=8, product 2=3
	transitQty1 := getLineItemQty(t, db, result.TransitDC.ID, 1)
	if transitQty1 != 8 {
		t.Errorf("transit DC product 1: got qty %d, want 8", transitQty1)
	}
	transitQty2 := getLineItemQty(t, db, result.TransitDC.ID, 2)
	if transitQty2 != 3 {
		t.Errorf("transit DC product 2: got qty %d, want 3", transitQty2)
	}
}

func TestShipmentLineItem_TotalQty(t *testing.T) {
	// Test with QtyByLocation
	item := ShipmentLineItem{
		QtyByLocation: map[int]int{100: 5, 200: 3, 300: 7},
	}
	if got := item.TotalQty(); got != 15 {
		t.Errorf("TotalQty() with QtyByLocation = %d, want 15", got)
	}

	// Test fallback to QtyPerSet
	item2 := ShipmentLineItem{
		QtyPerSet: 10,
	}
	if got := item2.TotalQty(); got != 10 {
		t.Errorf("TotalQty() with QtyPerSet fallback = %d, want 10", got)
	}
}

func TestShipmentLineItem_QtyForLocation(t *testing.T) {
	item := ShipmentLineItem{
		QtyByLocation: map[int]int{100: 5, 200: 3},
		QtyPerSet:     99, // should not be used
	}
	if got := item.QtyForLocation(100); got != 5 {
		t.Errorf("QtyForLocation(100) = %d, want 5", got)
	}
	if got := item.QtyForLocation(200); got != 3 {
		t.Errorf("QtyForLocation(200) = %d, want 3", got)
	}
	// Missing location returns 0 from the map
	if got := item.QtyForLocation(999); got != 0 {
		t.Errorf("QtyForLocation(999) = %d, want 0", got)
	}

	// Fallback to QtyPerSet when QtyByLocation is empty
	item2 := ShipmentLineItem{QtyPerSet: 7}
	if got := item2.QtyForLocation(100); got != 7 {
		t.Errorf("QtyForLocation fallback = %d, want 7", got)
	}
}
