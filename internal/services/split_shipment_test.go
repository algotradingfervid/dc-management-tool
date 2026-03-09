package services

import (
	"database/sql"
	"testing"
)

// setupSplitTestDB creates an in-memory SQLite database with all tables needed
// for split shipment testing (extends setupDCGenTestDB with transfer DC tables).
func setupSplitTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db := setupDCGenTestDB(t) // reuses dc_generation_test.go helper

	// Add transfer DC tables
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS transfer_dcs (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			dc_id           INTEGER NOT NULL UNIQUE REFERENCES delivery_challans(id) ON DELETE CASCADE,
			hub_address_id  INTEGER NOT NULL,
			template_id     INTEGER,
			tax_type        TEXT NOT NULL DEFAULT 'cgst_sgst',
			reverse_charge  TEXT NOT NULL DEFAULT 'N',
			transporter_name TEXT,
			vehicle_number  TEXT,
			eway_bill_number TEXT,
			docket_number   TEXT,
			notes           TEXT,
			num_destinations INTEGER NOT NULL DEFAULT 0,
			num_split       INTEGER NOT NULL DEFAULT 0,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS transfer_dc_splits (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			transfer_dc_id      INTEGER NOT NULL REFERENCES transfer_dcs(id) ON DELETE CASCADE,
			shipment_group_id   INTEGER NOT NULL UNIQUE REFERENCES shipment_groups(id) ON DELETE CASCADE,
			split_number        INTEGER NOT NULL,
			created_by          INTEGER,
			created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS transfer_dc_destinations (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			transfer_dc_id      INTEGER NOT NULL REFERENCES transfer_dcs(id) ON DELETE CASCADE,
			ship_to_address_id  INTEGER NOT NULL,
			split_group_id      INTEGER REFERENCES transfer_dc_splits(id) ON DELETE SET NULL,
			is_split            INTEGER NOT NULL DEFAULT 0,
			created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS transfer_dc_destination_quantities (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			destination_id  INTEGER NOT NULL REFERENCES transfer_dc_destinations(id) ON DELETE CASCADE,
			product_id      INTEGER NOT NULL,
			quantity        INTEGER NOT NULL DEFAULT 0,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(destination_id, product_id)
		)`,
		// Add updated_at column to shipment_groups for status updates
		`ALTER TABLE shipment_groups ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP`,
		// Add transfer_dc_id and split_id to shipment_groups
		`ALTER TABLE shipment_groups ADD COLUMN transfer_dc_id INTEGER`,
		`ALTER TABLE shipment_groups ADD COLUMN split_id INTEGER`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("setup split tables failed:\n%s\nerr: %v", s, err)
		}
	}

	return db
}

// createTestTransferDC creates a complete Transfer DC with destinations, quantities, and serials for testing.
// Returns (transferDCID, dcID, destinationIDs).
func createTestTransferDC(t *testing.T, db *sql.DB, projectID int) (int, int, []int) {
	t.Helper()

	// Create parent delivery challan
	dcResult, err := db.Exec(
		`INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, template_id, bill_to_address_id, ship_to_address_id, challan_date, created_by, bill_from_address_id, dispatch_from_address_id)
		 VALUES (?, 'TST-STDC-2526-001', 'transfer', 'issued', 1, 1, 1, '2026-01-15', 1, 1, 1)`,
		projectID,
	)
	if err != nil {
		t.Fatalf("createTestTransferDC: insert DC failed: %v", err)
	}
	dcID64, _ := dcResult.LastInsertId()
	dcID := int(dcID64)

	// Insert line items with rates (Product 1: rate=500, tax=18%; Product 2: rate=200, tax=18%)
	li1Result, err := db.Exec(
		`INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order)
		 VALUES (?, 1, 20, 500.00, 18.00, 10000.00, 1800.00, 11800.00, 1)`, dcID,
	)
	if err != nil {
		t.Fatalf("createTestTransferDC: insert line item 1 failed: %v", err)
	}
	li1ID64, _ := li1Result.LastInsertId()
	li1ID := int(li1ID64)

	li2Result, err := db.Exec(
		`INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order)
		 VALUES (?, 2, 10, 200.00, 18.00, 2000.00, 360.00, 2360.00, 2)`, dcID,
	)
	if err != nil {
		t.Fatalf("createTestTransferDC: insert line item 2 failed: %v", err)
	}
	li2ID64, _ := li2Result.LastInsertId()
	li2ID := int(li2ID64)

	// Insert serial numbers for product 1 (20 serials)
	for i := 1; i <= 20; i++ {
		sn := "SN-P1-" + padInt(i, 3)
		_, err := db.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number, product_id) VALUES (?, ?, ?, 1)`, projectID, li1ID, sn)
		if err != nil {
			t.Fatalf("createTestTransferDC: insert serial %s: %v", sn, err)
		}
	}
	// Insert serial numbers for product 2 (10 serials)
	for i := 1; i <= 10; i++ {
		sn := "SN-P2-" + padInt(i, 3)
		_, err := db.Exec(`INSERT INTO serial_numbers (project_id, line_item_id, serial_number, product_id) VALUES (?, ?, ?, 2)`, projectID, li2ID, sn)
		if err != nil {
			t.Fatalf("createTestTransferDC: insert serial %s: %v", sn, err)
		}
	}

	// Create transfer_dcs record
	tdcResult, err := db.Exec(
		`INSERT INTO transfer_dcs (dc_id, hub_address_id, template_id, tax_type, reverse_charge, num_destinations, num_split)
		 VALUES (?, 1, 1, 'cgst_sgst', 'N', 4, 0)`, dcID,
	)
	if err != nil {
		t.Fatalf("createTestTransferDC: insert transfer_dcs failed: %v", err)
	}
	tdcID64, _ := tdcResult.LastInsertId()
	tdcID := int(tdcID64)

	// Create 4 destinations with quantities
	// dest 1 (addr 100): P1=5, P2=3
	// dest 2 (addr 200): P1=5, P2=2
	// dest 3 (addr 300): P1=5, P2=3
	// dest 4 (addr 400): P1=5, P2=2
	// Total: P1=20, P2=10
	destAddrs := []int{100, 200, 300, 400}
	p1Qtys := []int{5, 5, 5, 5}
	p2Qtys := []int{3, 2, 3, 2}
	var destIDs []int

	for i, addr := range destAddrs {
		destResult, err := db.Exec(
			`INSERT INTO transfer_dc_destinations (transfer_dc_id, ship_to_address_id) VALUES (?, ?)`,
			tdcID, addr,
		)
		if err != nil {
			t.Fatalf("createTestTransferDC: insert dest %d: %v", addr, err)
		}
		destID64, _ := destResult.LastInsertId()
		destID := int(destID64)
		destIDs = append(destIDs, destID)

		// Insert quantities
		db.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 1, ?)`, destID, p1Qtys[i])
		db.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 2, ?)`, destID, p2Qtys[i])
	}

	return tdcID, dcID, destIDs
}

func padInt(n, width int) string {
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	for len(s) < width {
		s = "0" + s
	}
	return s
}

// ---------------------------------------------------------------------------
// Destination Validation Tests
// ---------------------------------------------------------------------------

func TestValidateSplitDestinations_MustBeUnsplit(t *testing.T) {
	// If a destination is already split, it cannot be selected again
	unsplitDestIDs := map[int]bool{1: true, 2: true} // only 1 and 2 are unsplit
	selectedIDs := []int{1, 3}                        // 3 is NOT unsplit

	err := validateSplitDestinations(selectedIDs, unsplitDestIDs)
	if err == nil {
		t.Fatal("expected error for selecting already-split destination")
	}
}

func TestValidateSplitDestinations_MustBelongToTransferDC(t *testing.T) {
	// All unsplit destinations for this TDC are {1, 2}
	unsplitDestIDs := map[int]bool{1: true, 2: true}
	selectedIDs := []int{1, 999} // 999 doesn't belong

	err := validateSplitDestinations(selectedIDs, unsplitDestIDs)
	if err == nil {
		t.Fatal("expected error for destination not belonging to Transfer DC")
	}
}

func TestValidateSplitDestinations_AtLeastOne(t *testing.T) {
	unsplitDestIDs := map[int]bool{1: true}
	selectedIDs := []int{}

	err := validateSplitDestinations(selectedIDs, unsplitDestIDs)
	if err == nil {
		t.Fatal("expected error for empty destination selection")
	}
}

func TestValidateSplitDestinations_HappyPath(t *testing.T) {
	unsplitDestIDs := map[int]bool{1: true, 2: true, 3: true}
	selectedIDs := []int{1, 2}

	err := validateSplitDestinations(selectedIDs, unsplitDestIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Serial Validation Tests
// ---------------------------------------------------------------------------

func TestValidateSplitSerials_MustBelongToParent(t *testing.T) {
	parentSerials := map[int]map[string]bool{
		1: {"SN-001": true, "SN-002": true},
	}
	usedSerials := map[string]bool{} // none used yet

	productSerials := []SplitProductSerials{
		{ProductID: 1, SerialNumbers: []string{"SN-001", "SN-FAKE"}},
	}
	expectedQty := map[int]int{1: 2}

	errs := validateSplitSerials(productSerials, expectedQty, parentSerials, usedSerials)
	if errs["product_1_serials"] == "" {
		t.Fatal("expected error for serial not belonging to parent")
	}
}

func TestValidateSplitSerials_ExactCount(t *testing.T) {
	parentSerials := map[int]map[string]bool{
		1: {"SN-001": true, "SN-002": true, "SN-003": true},
	}
	usedSerials := map[string]bool{}

	// Providing 2 serials when expected qty is 3
	productSerials := []SplitProductSerials{
		{ProductID: 1, SerialNumbers: []string{"SN-001", "SN-002"}},
	}
	expectedQty := map[int]int{1: 3}

	errs := validateSplitSerials(productSerials, expectedQty, parentSerials, usedSerials)
	if errs["product_1_count"] == "" {
		t.Fatal("expected error for serial count mismatch")
	}
}

func TestValidateSplitSerials_NoDuplicates(t *testing.T) {
	parentSerials := map[int]map[string]bool{
		1: {"SN-001": true, "SN-002": true},
	}
	usedSerials := map[string]bool{}

	productSerials := []SplitProductSerials{
		{ProductID: 1, SerialNumbers: []string{"SN-001", "SN-001"}}, // duplicate
	}
	expectedQty := map[int]int{1: 2}

	errs := validateSplitSerials(productSerials, expectedQty, parentSerials, usedSerials)
	if errs["product_1_serials"] == "" {
		t.Fatal("expected error for duplicate serial")
	}
}

func TestValidateSplitSerials_NotAlreadyUsedInOtherSplit(t *testing.T) {
	parentSerials := map[int]map[string]bool{
		1: {"SN-001": true, "SN-002": true},
	}
	usedSerials := map[string]bool{"SN-001": true} // SN-001 already used in another split

	productSerials := []SplitProductSerials{
		{ProductID: 1, SerialNumbers: []string{"SN-001", "SN-002"}},
	}
	expectedQty := map[int]int{1: 2}

	errs := validateSplitSerials(productSerials, expectedQty, parentSerials, usedSerials)
	if errs["product_1_serials"] == "" {
		t.Fatal("expected error for serial already used in another split")
	}
}

func TestValidateSplitSerials_HappyPath(t *testing.T) {
	parentSerials := map[int]map[string]bool{
		1: {"SN-001": true, "SN-002": true, "SN-003": true},
		2: {"SN-A01": true, "SN-A02": true},
	}
	usedSerials := map[string]bool{}

	productSerials := []SplitProductSerials{
		{ProductID: 1, SerialNumbers: []string{"SN-001", "SN-002"}},
		{ProductID: 2, SerialNumbers: []string{"SN-A01"}},
	}
	expectedQty := map[int]int{1: 2, 2: 1}

	errs := validateSplitSerials(productSerials, expectedQty, parentSerials, usedSerials)
	if len(errs) != 0 {
		t.Fatalf("unexpected validation errors: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// Status Transition Tests
// ---------------------------------------------------------------------------

func TestValidateSplitStatus_MustBeIssuedOrSplitting(t *testing.T) {
	tests := []struct {
		status    string
		wantError bool
	}{
		{"draft", true},
		{"issued", false},
		{"splitting", false},
		{"split", true},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			err := validateSplitStatus(tt.status)
			if tt.wantError && err == nil {
				t.Fatalf("expected error for status %q", tt.status)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error for status %q: %v", tt.status, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: CreateSplitShipment
// ---------------------------------------------------------------------------

func TestCreateSplitShipment_HappyPath(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split destinations 1 and 2 (addr 100: P1=5,P2=3; addr 200: P1=5,P2=2)
	// Total: P1=10, P2=5
	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: []int{destIDs[0], destIDs[1]},
		TransporterName: "Small Vehicle Co",
		VehicleNumber:   "TS01-AB-1234",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 10)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 5)},
		},
		CreatedBy: 1,
	}

	result, err := CreateSplitShipment(db, params)
	if err != nil {
		t.Fatalf("CreateSplitShipment failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.GroupID == 0 {
		t.Error("expected non-zero GroupID")
	}
	if result.SplitID == 0 {
		t.Error("expected non-zero SplitID")
	}

	// Verify child shipment group was created
	var sgCount int
	db.QueryRow(`SELECT COUNT(*) FROM shipment_groups WHERE id = ?`, result.GroupID).Scan(&sgCount)
	if sgCount != 1 {
		t.Errorf("expected 1 shipment group, got %d", sgCount)
	}

	// Verify 1 transit DC + 2 official DCs were created
	var transitCount, officialCount int
	db.QueryRow(`SELECT COUNT(*) FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'transit'`, result.GroupID).Scan(&transitCount)
	db.QueryRow(`SELECT COUNT(*) FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'official'`, result.GroupID).Scan(&officialCount)
	if transitCount != 1 {
		t.Errorf("expected 1 transit DC, got %d", transitCount)
	}
	if officialCount != 2 {
		t.Errorf("expected 2 official DCs, got %d", officialCount)
	}

	// Verify transit DC has correct total quantities
	var transitDCID int
	db.QueryRow(`SELECT id FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'transit'`, result.GroupID).Scan(&transitDCID)
	p1Qty := getLineItemQty(t, db, transitDCID, 1)
	p2Qty := getLineItemQty(t, db, transitDCID, 2)
	if p1Qty != 10 {
		t.Errorf("transit DC product 1 qty: want 10, got %d", p1Qty)
	}
	if p2Qty != 5 {
		t.Errorf("transit DC product 2 qty: want 5, got %d", p2Qty)
	}

	// Verify destinations are marked as split
	var splitCount int
	db.QueryRow(`SELECT COUNT(*) FROM transfer_dc_destinations WHERE id IN (?, ?) AND is_split = 1`, destIDs[0], destIDs[1]).Scan(&splitCount)
	if splitCount != 2 {
		t.Errorf("expected 2 destinations marked as split, got %d", splitCount)
	}

	// Verify Transfer DC status changed to splitting (2 of 4 destinations split)
	var dcStatus string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&dcStatus)
	if dcStatus != "splitting" {
		t.Errorf("Transfer DC status: want 'splitting', got %q", dcStatus)
	}

	// Verify num_split counter updated
	var numSplit int
	db.QueryRow(`SELECT num_split FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numSplit)
	if numSplit != 2 {
		t.Errorf("num_split: want 2, got %d", numSplit)
	}
}

func TestCreateSplitShipment_InheritsRatesFromParent(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: []int{destIDs[0]}, // addr 100: P1=5, P2=3
		TransporterName: "Test",
		VehicleNumber:   "TS01",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 5)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 3)},
		},
		CreatedBy: 1,
	}

	result, err := CreateSplitShipment(db, params)
	if err != nil {
		t.Fatalf("CreateSplitShipment failed: %v", err)
	}

	// Get child transit DC and verify rates match parent
	var transitDCID int
	db.QueryRow(`SELECT id FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'transit'`, result.GroupID).Scan(&transitDCID)

	var rate1, taxPct1 float64
	db.QueryRow(`SELECT rate, tax_percentage FROM dc_line_items WHERE dc_id = ? AND product_id = 1`, transitDCID).Scan(&rate1, &taxPct1)
	if rate1 != 500.0 {
		t.Errorf("child transit DC product 1 rate: want 500.0, got %f", rate1)
	}
	if taxPct1 != 18.0 {
		t.Errorf("child transit DC product 1 tax%%: want 18.0, got %f", taxPct1)
	}

	var rate2, taxPct2 float64
	db.QueryRow(`SELECT rate, tax_percentage FROM dc_line_items WHERE dc_id = ? AND product_id = 2`, transitDCID).Scan(&rate2, &taxPct2)
	if rate2 != 200.0 {
		t.Errorf("child transit DC product 2 rate: want 200.0, got %f", rate2)
	}
}

func TestCreateSplitShipment_InheritsTaxType(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: []int{destIDs[0]},
		TransporterName: "Test",
		VehicleNumber:   "TS01",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 5)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 3)},
		},
		CreatedBy: 1,
	}

	result, err := CreateSplitShipment(db, params)
	if err != nil {
		t.Fatalf("CreateSplitShipment failed: %v", err)
	}

	// Parent Transfer DC has tax_type='cgst_sgst', reverse_charge='N'
	var taxType, reverseCharge string
	db.QueryRow(`SELECT tax_type, reverse_charge FROM shipment_groups WHERE id = ?`, result.GroupID).Scan(&taxType, &reverseCharge)
	if taxType != "cgst_sgst" {
		t.Errorf("child shipment group tax_type: want 'cgst_sgst', got %q", taxType)
	}
	if reverseCharge != "N" {
		t.Errorf("child shipment group reverse_charge: want 'N', got %q", reverseCharge)
	}
}

func TestCreateSplitShipment_InheritsAddresses(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: []int{destIDs[0]},
		TransporterName: "Test",
		VehicleNumber:   "TS01",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 5)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 3)},
		},
		CreatedBy: 1,
	}

	result, err := CreateSplitShipment(db, params)
	if err != nil {
		t.Fatalf("CreateSplitShipment failed: %v", err)
	}

	// Verify child transit DC inherits addresses from parent DC
	var billFromID, dispatchFromID, billToID sql.NullInt64
	db.QueryRow(
		`SELECT bill_from_address_id, dispatch_from_address_id, bill_to_address_id FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'transit'`,
		result.GroupID,
	).Scan(&billFromID, &dispatchFromID, &billToID)

	if !billFromID.Valid || int(billFromID.Int64) != 1 {
		t.Errorf("child transit DC bill_from_address_id: want 1, got %v", billFromID)
	}
	if !dispatchFromID.Valid || int(dispatchFromID.Int64) != 1 {
		t.Errorf("child transit DC dispatch_from_address_id: want 1, got %v", dispatchFromID)
	}
	if !billToID.Valid || int(billToID.Int64) != 1 {
		t.Errorf("child transit DC bill_to_address_id: want 1, got %v", billToID)
	}
}

func TestCreateSplitShipment_CorrectOfficialDCQuantities(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split 2 destinations: addr 100 (P1=5, P2=3) and addr 200 (P1=5, P2=2)
	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: []int{destIDs[0], destIDs[1]},
		TransporterName: "Test",
		VehicleNumber:   "TS01",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 10)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 5)},
		},
		CreatedBy: 1,
	}

	result, err := CreateSplitShipment(db, params)
	if err != nil {
		t.Fatalf("CreateSplitShipment failed: %v", err)
	}

	// Get ship-to addresses of the dest IDs
	var addr1, addr2 int
	db.QueryRow(`SELECT ship_to_address_id FROM transfer_dc_destinations WHERE id = ?`, destIDs[0]).Scan(&addr1)
	db.QueryRow(`SELECT ship_to_address_id FROM transfer_dc_destinations WHERE id = ?`, destIDs[1]).Scan(&addr2)

	// Find official DCs by ship_to_address_id
	var offDC1ID, offDC2ID int
	db.QueryRow(`SELECT id FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'official' AND ship_to_address_id = ?`, result.GroupID, addr1).Scan(&offDC1ID)
	db.QueryRow(`SELECT id FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'official' AND ship_to_address_id = ?`, result.GroupID, addr2).Scan(&offDC2ID)

	// Official DC for addr 100: P1=5, P2=3
	if qty := getLineItemQty(t, db, offDC1ID, 1); qty != 5 {
		t.Errorf("official DC addr=100, product 1: want 5, got %d", qty)
	}
	if qty := getLineItemQty(t, db, offDC1ID, 2); qty != 3 {
		t.Errorf("official DC addr=100, product 2: want 3, got %d", qty)
	}
	// Official DC for addr 200: P1=5, P2=2
	if qty := getLineItemQty(t, db, offDC2ID, 1); qty != 5 {
		t.Errorf("official DC addr=200, product 1: want 5, got %d", qty)
	}
	if qty := getLineItemQty(t, db, offDC2ID, 2); qty != 2 {
		t.Errorf("official DC addr=200, product 2: want 2, got %d", qty)
	}
}

func TestCreateSplitShipment_SingleDestination(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split just 1 destination: addr 100 (P1=5, P2=3) → 1 TDC + 1 ODC
	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: []int{destIDs[0]},
		TransporterName: "Test",
		VehicleNumber:   "TS01",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 5)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 3)},
		},
		CreatedBy: 1,
	}

	result, err := CreateSplitShipment(db, params)
	if err != nil {
		t.Fatalf("CreateSplitShipment failed: %v", err)
	}

	var officialCount int
	db.QueryRow(`SELECT COUNT(*) FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'official'`, result.GroupID).Scan(&officialCount)
	if officialCount != 1 {
		t.Errorf("expected 1 official DC for single destination, got %d", officialCount)
	}
}

func TestCreateSplitShipment_AllRemainingDestinations(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split ALL 4 destinations → status should become "split"
	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: destIDs, // all 4
		TransporterName: "Test",
		VehicleNumber:   "TS01",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 20)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 10)},
		},
		CreatedBy: 1,
	}

	_, err := CreateSplitShipment(db, params)
	if err != nil {
		t.Fatalf("CreateSplitShipment failed: %v", err)
	}

	// All destinations split → status should be "split"
	var dcStatus string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&dcStatus)
	if dcStatus != "split" {
		t.Errorf("Transfer DC status: want 'split', got %q", dcStatus)
	}

	var numSplit int
	db.QueryRow(`SELECT num_split FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numSplit)
	if numSplit != 4 {
		t.Errorf("num_split: want 4, got %d", numSplit)
	}
}

func TestCreateSplitShipment_TransferDCMustBeIssued(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Set status back to draft
	db.Exec(`UPDATE delivery_challans SET status = 'draft' WHERE id = ?`, dcID)

	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: []int{destIDs[0]},
		TransporterName: "Test",
		VehicleNumber:   "TS01",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 5)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 3)},
		},
		CreatedBy: 1,
	}

	_, err := CreateSplitShipment(db, params)
	if err == nil {
		t.Fatal("expected error when splitting a draft Transfer DC")
	}
}

func TestCreateSplitShipment_TransferDCCannotBeSplit(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "SplitProject", "SPL")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Set status to "split" (fully split)
	db.Exec(`UPDATE delivery_challans SET status = 'split' WHERE id = ?`, dcID)

	params := SplitShipmentParams{
		TransferDCID:   tdcID,
		ParentDCID:     dcID,
		ProjectID:      projectID,
		DestinationIDs: []int{destIDs[0]},
		TransporterName: "Test",
		VehicleNumber:   "TS01",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: serialRange("SN-P1-", 1, 5)},
			{ProductID: 2, SerialNumbers: serialRange("SN-P2-", 1, 3)},
		},
		CreatedBy: 1,
	}

	_, err := CreateSplitShipment(db, params)
	if err == nil {
		t.Fatal("expected error when splitting a fully-split Transfer DC")
	}
}

// serialRange generates serial numbers like "SN-P1-001", "SN-P1-002", ...
func serialRange(prefix string, from, to int) []string {
	var serials []string
	for i := from; i <= to; i++ {
		serials = append(serials, prefix+padInt(i, 3))
	}
	return serials
}
