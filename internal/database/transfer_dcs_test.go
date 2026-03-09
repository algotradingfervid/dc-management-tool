package database

import (
	"database/sql"
	"testing"

	"github.com/narendhupati/dc-management-tool/internal/models"
	_ "modernc.org/sqlite"
)

func setupTransferDCTestDB(t *testing.T) func() {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&mode=memory")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	db.Exec("PRAGMA foreign_keys = ON")

	stmts := []string{
		// --- Core tables ---
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
		`INSERT OR IGNORE INTO dc_templates (id, name, project_id) VALUES (1, 'Default Template', 1)`,

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
		`INSERT OR IGNORE INTO addresses (id, config_id, address_data, district_name, mandal_name, mandal_code)
			VALUES (1, NULL, '{"name":"Hub Warehouse"}', '', '', '')`,
		`INSERT OR IGNORE INTO addresses (id, config_id, address_data, district_name, mandal_name, mandal_code)
			VALUES (2, NULL, '{"name":"Dest Chennai"}', '', '', '')`,
		`INSERT OR IGNORE INTO addresses (id, config_id, address_data, district_name, mandal_name, mandal_code)
			VALUES (3, NULL, '{"name":"Dest Mumbai"}', '', '', '')`,

		`CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_name TEXT NOT NULL,
			item_description TEXT DEFAULT '',
			hsn_code TEXT DEFAULT '',
			uom TEXT DEFAULT 'nos',
			brand_model TEXT DEFAULT '',
			gst_percentage REAL DEFAULT 0
		)`,
		`INSERT OR IGNORE INTO products (id, item_name) VALUES (1, 'Solar Panel')`,
		`INSERT OR IGNORE INTO products (id, item_name) VALUES (2, 'Inverter')`,

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
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			transfer_dc_id INTEGER,
			split_id INTEGER
		)`,

		`CREATE TABLE IF NOT EXISTS delivery_challans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			dc_number TEXT NOT NULL,
			dc_type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			template_id INTEGER,
			bill_to_address_id INTEGER,
			ship_to_address_id INTEGER,
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
			transfer_dc_id INTEGER,
			UNIQUE(project_id, dc_number)
		)`,

		// --- Transfer DC tables (migration 00034) ---
		`CREATE TABLE IF NOT EXISTS transfer_dcs (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			dc_id           INTEGER NOT NULL UNIQUE REFERENCES delivery_challans(id) ON DELETE CASCADE,
			hub_address_id  INTEGER NOT NULL REFERENCES addresses(id),
			template_id     INTEGER REFERENCES dc_templates(id) ON DELETE SET NULL,
			tax_type        TEXT NOT NULL DEFAULT 'cgst_sgst' CHECK (tax_type IN ('cgst_sgst', 'igst')),
			reverse_charge  TEXT NOT NULL DEFAULT 'N' CHECK (reverse_charge IN ('Y', 'N')),
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
			created_by          INTEGER REFERENCES users(id),
			created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS transfer_dc_destinations (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			transfer_dc_id      INTEGER NOT NULL REFERENCES transfer_dcs(id) ON DELETE CASCADE,
			ship_to_address_id  INTEGER NOT NULL REFERENCES addresses(id),
			split_group_id      INTEGER REFERENCES transfer_dc_splits(id) ON DELETE SET NULL,
			is_split            INTEGER NOT NULL DEFAULT 0,
			created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS transfer_dc_destination_quantities (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			destination_id  INTEGER NOT NULL REFERENCES transfer_dc_destinations(id) ON DELETE CASCADE,
			product_id      INTEGER NOT NULL REFERENCES products(id),
			quantity        INTEGER NOT NULL DEFAULT 0,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(destination_id, product_id)
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

// insertTransferTestDC inserts a minimal delivery challan of type "transfer" and returns its ID.
func insertTransferTestDC(t *testing.T, projectID int, dcNumber string, shipToAddrID int) int {
	t.Helper()
	res, err := DB.Exec(
		`INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, ship_to_address_id, created_by)
		 VALUES (?, ?, 'transfer', 'draft', ?, 1)`,
		projectID, dcNumber, shipToAddrID,
	)
	if err != nil {
		t.Fatalf("insertTransferTestDC failed: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// insertTransferDCRow inserts a transfer_dcs record directly and returns its ID.
func insertTransferDCRow(t *testing.T, dcID, hubAddrID int, templateID *int) int {
	t.Helper()
	var res sql.Result
	var err error
	if templateID != nil {
		res, err = DB.Exec(
			`INSERT INTO transfer_dcs (dc_id, hub_address_id, template_id) VALUES (?, ?, ?)`,
			dcID, hubAddrID, *templateID,
		)
	} else {
		res, err = DB.Exec(
			`INSERT INTO transfer_dcs (dc_id, hub_address_id) VALUES (?, ?)`,
			dcID, hubAddrID,
		)
	}
	if err != nil {
		t.Fatalf("insertTransferDCRow failed: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// insertDestination inserts a transfer_dc_destinations row and returns its ID.
func insertDestination(t *testing.T, transferDCID, shipToAddrID int) int {
	t.Helper()
	res, err := DB.Exec(
		`INSERT INTO transfer_dc_destinations (transfer_dc_id, ship_to_address_id) VALUES (?, ?)`,
		transferDCID, shipToAddrID,
	)
	if err != nil {
		t.Fatalf("insertDestination failed: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// insertShipmentGroup inserts a shipment_groups row and returns its ID.
func insertShipmentGroup(t *testing.T, projectID int) int {
	t.Helper()
	res, err := DB.Exec(
		`INSERT INTO shipment_groups (project_id, num_sets, tax_type, reverse_charge, status, created_by)
		 VALUES (?, 1, 'cgst_sgst', 'N', 'draft', 1)`,
		projectID,
	)
	if err != nil {
		t.Fatalf("insertShipmentGroup failed: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// ---------------------------------------------------------------------------
// Transfer DC Core CRUD
// ---------------------------------------------------------------------------

func TestCreateTransferDC(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	// Create a parent delivery challan first
	dcID := insertTransferTestDC(t, 1, "XFER-2026-001", 1)

	tdc := &models.TransferDC{
		DCID:            dcID,
		HubAddressID:    1,
		TemplateID:      intPtr(1),
		TaxType:         "cgst_sgst",
		ReverseCharge:   "N",
		TransporterName: "FastShip Co",
		VehicleNumber:   "TN-01-AB-1234",
		Notes:           "Test transfer",
	}

	id, err := CreateTransferDC(tdc)
	if err != nil {
		t.Fatalf("CreateTransferDC failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID from CreateTransferDC")
	}

	// Verify the record exists in the database
	var gotDCID, gotHubAddr int
	var gotTaxType, gotRC string
	var gotTransporter, gotVehicle sql.NullString
	err = DB.QueryRow(
		`SELECT dc_id, hub_address_id, tax_type, reverse_charge, transporter_name, vehicle_number
		 FROM transfer_dcs WHERE id = ?`, id,
	).Scan(&gotDCID, &gotHubAddr, &gotTaxType, &gotRC, &gotTransporter, &gotVehicle)
	if err != nil {
		t.Fatalf("failed to read back transfer_dcs row: %v", err)
	}
	if gotDCID != dcID {
		t.Errorf("dc_id: want %d, got %d", dcID, gotDCID)
	}
	if gotHubAddr != 1 {
		t.Errorf("hub_address_id: want 1, got %d", gotHubAddr)
	}
	if gotTaxType != "cgst_sgst" {
		t.Errorf("tax_type: want cgst_sgst, got %q", gotTaxType)
	}
	if gotRC != "N" {
		t.Errorf("reverse_charge: want N, got %q", gotRC)
	}
	if gotTransporter.String != "FastShip Co" {
		t.Errorf("transporter_name: want 'FastShip Co', got %q", gotTransporter.String)
	}
	if gotVehicle.String != "TN-01-AB-1234" {
		t.Errorf("vehicle_number: want 'TN-01-AB-1234', got %q", gotVehicle.String)
	}
}

func TestGetTransferDC(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-002", 1)
	tmplID := 1
	tdcID := insertTransferDCRow(t, dcID, 1, &tmplID)

	got, err := GetTransferDC(tdcID)
	if err != nil {
		t.Fatalf("GetTransferDC failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetTransferDC returned nil")
	}

	// Verify core fields
	if got.ID != tdcID {
		t.Errorf("ID: want %d, got %d", tdcID, got.ID)
	}
	if got.DCID != dcID {
		t.Errorf("DCID: want %d, got %d", dcID, got.DCID)
	}
	if got.HubAddressID != 1 {
		t.Errorf("HubAddressID: want 1, got %d", got.HubAddressID)
	}

	// Verify joined fields
	if got.DCNumber != "XFER-2026-002" {
		t.Errorf("DCNumber: want 'XFER-2026-002', got %q", got.DCNumber)
	}
	if got.DCStatus != "draft" {
		t.Errorf("DCStatus: want 'draft', got %q", got.DCStatus)
	}
	if got.TemplateName != "Default Template" {
		t.Errorf("TemplateName: want 'Default Template', got %q", got.TemplateName)
	}
}

func TestGetTransferDCByDCID(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-003", 1)
	tmplID := 1
	tdcID := insertTransferDCRow(t, dcID, 1, &tmplID)

	got, err := GetTransferDCByDCID(dcID)
	if err != nil {
		t.Fatalf("GetTransferDCByDCID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetTransferDCByDCID returned nil")
	}
	if got.ID != tdcID {
		t.Errorf("ID: want %d, got %d", tdcID, got.ID)
	}
	if got.DCID != dcID {
		t.Errorf("DCID: want %d, got %d", dcID, got.DCID)
	}
	if got.DCNumber != "XFER-2026-003" {
		t.Errorf("DCNumber: want 'XFER-2026-003', got %q", got.DCNumber)
	}
}

func TestUpdateTransferDC(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-004", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)

	tdc := &models.TransferDC{
		ID:              tdcID,
		DCID:            dcID,
		HubAddressID:    1,
		TransporterName: "Updated Transport",
		VehicleNumber:   "KA-05-XY-9999",
		EwayBillNumber:  "EWB-123456",
		DocketNumber:    "DOC-789",
		Notes:           "Updated notes",
		TaxType:         "igst",
		ReverseCharge:   "Y",
	}

	if err := UpdateTransferDC(tdc); err != nil {
		t.Fatalf("UpdateTransferDC failed: %v", err)
	}

	// Verify the updates
	var gotTransporter, gotVehicle, gotEway, gotDocket, gotNotes sql.NullString
	var gotTaxType, gotRC string
	err := DB.QueryRow(
		`SELECT transporter_name, vehicle_number, eway_bill_number, docket_number, notes, tax_type, reverse_charge
		 FROM transfer_dcs WHERE id = ?`, tdcID,
	).Scan(&gotTransporter, &gotVehicle, &gotEway, &gotDocket, &gotNotes, &gotTaxType, &gotRC)
	if err != nil {
		t.Fatalf("failed to read updated transfer_dcs row: %v", err)
	}
	if gotTransporter.String != "Updated Transport" {
		t.Errorf("transporter_name: want 'Updated Transport', got %q", gotTransporter.String)
	}
	if gotVehicle.String != "KA-05-XY-9999" {
		t.Errorf("vehicle_number: want 'KA-05-XY-9999', got %q", gotVehicle.String)
	}
	if gotEway.String != "EWB-123456" {
		t.Errorf("eway_bill_number: want 'EWB-123456', got %q", gotEway.String)
	}
	if gotDocket.String != "DOC-789" {
		t.Errorf("docket_number: want 'DOC-789', got %q", gotDocket.String)
	}
	if gotNotes.String != "Updated notes" {
		t.Errorf("notes: want 'Updated notes', got %q", gotNotes.String)
	}
	if gotTaxType != "igst" {
		t.Errorf("tax_type: want 'igst', got %q", gotTaxType)
	}
	if gotRC != "Y" {
		t.Errorf("reverse_charge: want 'Y', got %q", gotRC)
	}
}

func TestDeleteTransferDC(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-005", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)

	// Add destinations so we can verify cascade
	dest1ID := insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)

	// Set some quantities too
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 1, 10)`, dest1ID)
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 2, 20)`, dest2ID)

	if err := DeleteTransferDC(tdcID); err != nil {
		t.Fatalf("DeleteTransferDC failed: %v", err)
	}

	// Transfer DC should be gone
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&count)
	if count != 0 {
		t.Error("transfer_dcs row should have been deleted")
	}

	// Destinations should be cascade-deleted
	DB.QueryRow(`SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ?`, tdcID).Scan(&count)
	if count != 0 {
		t.Errorf("destinations should have been cascade-deleted, got %d", count)
	}

	// Quantities should be cascade-deleted
	DB.QueryRow(`SELECT COUNT(*) FROM transfer_dc_destination_quantities WHERE destination_id IN (?, ?)`, dest1ID, dest2ID).Scan(&count)
	if count != 0 {
		t.Errorf("destination quantities should have been cascade-deleted, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Destination Management
// ---------------------------------------------------------------------------

func TestAddTransferDCDestinations(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-006", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)

	// Add two destinations
	err := AddTransferDCDestinations(tdcID, []int{2, 3})
	if err != nil {
		t.Fatalf("AddTransferDCDestinations failed: %v", err)
	}

	// Verify both destinations exist
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ?`, tdcID).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 destinations, got %d", count)
	}

	// Verify num_destinations counter was updated on transfer_dcs
	var numDest int
	DB.QueryRow(`SELECT num_destinations FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numDest)
	if numDest != 2 {
		t.Errorf("num_destinations: want 2, got %d", numDest)
	}
}

func TestGetTransferDCDestinations(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-007", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	insertDestination(t, tdcID, 2)
	insertDestination(t, tdcID, 3)

	dests, err := GetTransferDCDestinations(tdcID)
	if err != nil {
		t.Fatalf("GetTransferDCDestinations failed: %v", err)
	}
	if len(dests) != 2 {
		t.Fatalf("expected 2 destinations, got %d", len(dests))
	}

	// Verify destination fields
	found2, found3 := false, false
	for _, d := range dests {
		if d.TransferDCID != tdcID {
			t.Errorf("destination transfer_dc_id: want %d, got %d", tdcID, d.TransferDCID)
		}
		if d.ShipToAddressID == 2 {
			found2 = true
		}
		if d.ShipToAddressID == 3 {
			found3 = true
		}
		if d.IsSplit {
			t.Error("newly created destination should not be split")
		}
	}
	if !found2 || !found3 {
		t.Error("expected destinations for address IDs 2 and 3")
	}
}

func TestGetUnsplitDestinations(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-008", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)

	// Mark one destination as split
	DB.Exec(`UPDATE transfer_dc_destinations SET is_split = 1 WHERE id = ?`, dest2ID)

	unsplit, err := GetUnsplitDestinations(tdcID)
	if err != nil {
		t.Fatalf("GetUnsplitDestinations failed: %v", err)
	}
	if len(unsplit) != 1 {
		t.Fatalf("expected 1 unsplit destination, got %d", len(unsplit))
	}
	if unsplit[0].ShipToAddressID != 2 {
		t.Errorf("unsplit destination address: want 2, got %d", unsplit[0].ShipToAddressID)
	}
}

func TestUpdateDestinationSplitStatus(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-009", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	dest1ID := insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)

	// Create a real split record so FK constraint is satisfied
	sgID := insertShipmentGroup(t, 1)
	splitRes, splitErr := DB.Exec(
		`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (?, ?, 1, 1)`,
		tdcID, sgID,
	)
	if splitErr != nil {
		t.Fatalf("insert split record: %v", splitErr)
	}
	splitID64, _ := splitRes.LastInsertId()
	splitGroupID := intPtr(int(splitID64))

	// Mark both as split
	err := UpdateDestinationSplitStatus([]int{dest1ID, dest2ID}, splitGroupID, true)
	if err != nil {
		t.Fatalf("UpdateDestinationSplitStatus failed: %v", err)
	}

	// Verify both are split
	var isSplit1, isSplit2 int
	DB.QueryRow(`SELECT is_split FROM transfer_dc_destinations WHERE id = ?`, dest1ID).Scan(&isSplit1)
	DB.QueryRow(`SELECT is_split FROM transfer_dc_destinations WHERE id = ?`, dest2ID).Scan(&isSplit2)
	if isSplit1 != 1 {
		t.Errorf("dest1 is_split: want 1, got %d", isSplit1)
	}
	if isSplit2 != 1 {
		t.Errorf("dest2 is_split: want 1, got %d", isSplit2)
	}

	// Now reset them
	err = UpdateDestinationSplitStatus([]int{dest1ID, dest2ID}, nil, false)
	if err != nil {
		t.Fatalf("UpdateDestinationSplitStatus (reset) failed: %v", err)
	}
	DB.QueryRow(`SELECT is_split FROM transfer_dc_destinations WHERE id = ?`, dest1ID).Scan(&isSplit1)
	DB.QueryRow(`SELECT is_split FROM transfer_dc_destinations WHERE id = ?`, dest2ID).Scan(&isSplit2)
	if isSplit1 != 0 {
		t.Errorf("dest1 is_split after reset: want 0, got %d", isSplit1)
	}
	if isSplit2 != 0 {
		t.Errorf("dest2 is_split after reset: want 0, got %d", isSplit2)
	}
}

// ---------------------------------------------------------------------------
// Quantity Grid
// ---------------------------------------------------------------------------

func TestSetDestinationQuantities(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-010", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	destID := insertDestination(t, tdcID, 2)

	quantities := []models.TransferDCDestinationQty{
		{DestinationID: destID, ProductID: 1, Quantity: 50},
		{DestinationID: destID, ProductID: 2, Quantity: 30},
	}

	err := SetDestinationQuantities(destID, quantities)
	if err != nil {
		t.Fatalf("SetDestinationQuantities failed: %v", err)
	}

	// Verify via GetDestinationQuantities
	got, err := GetDestinationQuantities(destID)
	if err != nil {
		t.Fatalf("GetDestinationQuantities failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 quantity rows, got %d", len(got))
	}

	qtyMap := make(map[int]int)
	for _, q := range got {
		qtyMap[q.ProductID] = q.Quantity
	}
	if qtyMap[1] != 50 {
		t.Errorf("product 1 quantity: want 50, got %d", qtyMap[1])
	}
	if qtyMap[2] != 30 {
		t.Errorf("product 2 quantity: want 30, got %d", qtyMap[2])
	}
}

func TestGetQuantityGrid(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-011", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	dest1ID := insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)

	// Set quantities for destination 1
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 1, 100)`, dest1ID)
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 2, 200)`, dest1ID)

	// Set quantities for destination 2
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 1, 150)`, dest2ID)
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 2, 250)`, dest2ID)

	grid, err := GetQuantityGrid(tdcID)
	if err != nil {
		t.Fatalf("GetQuantityGrid failed: %v", err)
	}

	// grid should be map[destinationID]map[productID]quantity
	if len(grid) != 2 {
		t.Fatalf("expected grid with 2 destinations, got %d", len(grid))
	}

	// Check destination 1
	if grid[dest1ID] == nil {
		t.Fatalf("grid missing dest1ID %d", dest1ID)
	}
	if grid[dest1ID][1] != 100 {
		t.Errorf("grid[dest1][product1]: want 100, got %d", grid[dest1ID][1])
	}
	if grid[dest1ID][2] != 200 {
		t.Errorf("grid[dest1][product2]: want 200, got %d", grid[dest1ID][2])
	}

	// Check destination 2
	if grid[dest2ID] == nil {
		t.Fatalf("grid missing dest2ID %d", dest2ID)
	}
	if grid[dest2ID][1] != 150 {
		t.Errorf("grid[dest2][product1]: want 150, got %d", grid[dest2ID][1])
	}
	if grid[dest2ID][2] != 250 {
		t.Errorf("grid[dest2][product2]: want 250, got %d", grid[dest2ID][2])
	}
}

func TestGetQuantitiesForDestinations(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-012", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	dest1ID := insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)

	// Set quantities
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 1, 10)`, dest1ID)
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 2, 20)`, dest1ID)
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 1, 30)`, dest2ID)

	result, err := GetQuantitiesForDestinations([]int{dest1ID, dest2ID})
	if err != nil {
		t.Fatalf("GetQuantitiesForDestinations failed: %v", err)
	}

	// result should be map[destinationID][]TransferDCDestinationQty
	if len(result) != 2 {
		t.Fatalf("expected results for 2 destinations, got %d", len(result))
	}

	// Destination 1 should have 2 qty rows
	if len(result[dest1ID]) != 2 {
		t.Errorf("dest1 quantities: want 2, got %d", len(result[dest1ID]))
	}

	// Destination 2 should have 1 qty row
	if len(result[dest2ID]) != 1 {
		t.Errorf("dest2 quantities: want 1, got %d", len(result[dest2ID]))
	}

	// Verify specific quantity
	if result[dest2ID][0].Quantity != 30 {
		t.Errorf("dest2 product1 quantity: want 30, got %d", result[dest2ID][0].Quantity)
	}
}

// ---------------------------------------------------------------------------
// Split Tracking
// ---------------------------------------------------------------------------

func TestCreateSplit(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-013", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	dest1ID := insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)

	// Update num_destinations so counters are correct
	DB.Exec(`UPDATE transfer_dcs SET num_destinations = 2 WHERE id = ?`, tdcID)

	// Create a shipment group for the split
	sgID := insertShipmentGroup(t, 1)

	split, err := CreateSplit(tdcID, sgID, []int{dest1ID, dest2ID}, 1)
	if err != nil {
		t.Fatalf("CreateSplit failed: %v", err)
	}
	if split == nil {
		t.Fatal("CreateSplit returned nil split")
	}
	if split.ID == 0 {
		t.Error("expected non-zero split ID")
	}
	if split.TransferDCID != tdcID {
		t.Errorf("split TransferDCID: want %d, got %d", tdcID, split.TransferDCID)
	}
	if split.ShipmentGroupID != sgID {
		t.Errorf("split ShipmentGroupID: want %d, got %d", sgID, split.ShipmentGroupID)
	}
	if split.SplitNumber != 1 {
		t.Errorf("split SplitNumber: want 1, got %d", split.SplitNumber)
	}

	// Verify destinations are marked as split
	var isSplit1, isSplit2 int
	DB.QueryRow(`SELECT is_split FROM transfer_dc_destinations WHERE id = ?`, dest1ID).Scan(&isSplit1)
	DB.QueryRow(`SELECT is_split FROM transfer_dc_destinations WHERE id = ?`, dest2ID).Scan(&isSplit2)
	if isSplit1 != 1 {
		t.Error("dest1 should be marked as split")
	}
	if isSplit2 != 1 {
		t.Error("dest2 should be marked as split")
	}

	// Verify num_split counter was updated
	var numSplit int
	DB.QueryRow(`SELECT num_split FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numSplit)
	if numSplit != 2 {
		t.Errorf("num_split: want 2, got %d", numSplit)
	}
}

func TestGetSplitsByTransferDCID(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-014", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)

	// Create two shipment groups
	sg1ID := insertShipmentGroup(t, 1)
	sg2ID := insertShipmentGroup(t, 1)

	// Insert splits directly
	DB.Exec(`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (?, ?, 1, 1)`, tdcID, sg1ID)
	DB.Exec(`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (?, ?, 2, 1)`, tdcID, sg2ID)

	splits, err := GetSplitsByTransferDCID(tdcID)
	if err != nil {
		t.Fatalf("GetSplitsByTransferDCID failed: %v", err)
	}
	if len(splits) != 2 {
		t.Fatalf("expected 2 splits, got %d", len(splits))
	}

	// Should be ordered by split_number
	if splits[0].SplitNumber != 1 {
		t.Errorf("first split number: want 1, got %d", splits[0].SplitNumber)
	}
	if splits[1].SplitNumber != 2 {
		t.Errorf("second split number: want 2, got %d", splits[1].SplitNumber)
	}
	if splits[0].ShipmentGroupID != sg1ID {
		t.Errorf("first split shipment group: want %d, got %d", sg1ID, splits[0].ShipmentGroupID)
	}
}

func TestGetSplitByShipmentGroupID(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-015", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	sgID := insertShipmentGroup(t, 1)

	DB.Exec(`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (?, ?, 1, 1)`, tdcID, sgID)

	split, err := GetSplitByShipmentGroupID(sgID)
	if err != nil {
		t.Fatalf("GetSplitByShipmentGroupID failed: %v", err)
	}
	if split == nil {
		t.Fatal("GetSplitByShipmentGroupID returned nil")
	}
	if split.TransferDCID != tdcID {
		t.Errorf("TransferDCID: want %d, got %d", tdcID, split.TransferDCID)
	}
	if split.ShipmentGroupID != sgID {
		t.Errorf("ShipmentGroupID: want %d, got %d", sgID, split.ShipmentGroupID)
	}
	if split.SplitNumber != 1 {
		t.Errorf("SplitNumber: want 1, got %d", split.SplitNumber)
	}
}

func TestDeleteSplit(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-016", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	dest1ID := insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)
	sgID := insertShipmentGroup(t, 1)

	// Insert the split
	res, _ := DB.Exec(
		`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (?, ?, 1, 1)`,
		tdcID, sgID,
	)
	splitID64, _ := res.LastInsertId()
	splitID := int(splitID64)

	// Mark destinations as split with the split_group_id
	DB.Exec(`UPDATE transfer_dc_destinations SET is_split = 1, split_group_id = ? WHERE id IN (?, ?)`, splitID, dest1ID, dest2ID)
	DB.Exec(`UPDATE transfer_dcs SET num_destinations = 2, num_split = 2 WHERE id = ?`, tdcID)

	// Delete the split
	if err := DeleteSplit(splitID); err != nil {
		t.Fatalf("DeleteSplit failed: %v", err)
	}

	// Verify split is gone
	var splitCount int
	DB.QueryRow(`SELECT COUNT(*) FROM transfer_dc_splits WHERE id = ?`, splitID).Scan(&splitCount)
	if splitCount != 0 {
		t.Error("split should have been deleted")
	}

	// Verify destinations are reset to unsplit
	var isSplit1, isSplit2 int
	DB.QueryRow(`SELECT is_split FROM transfer_dc_destinations WHERE id = ?`, dest1ID).Scan(&isSplit1)
	DB.QueryRow(`SELECT is_split FROM transfer_dc_destinations WHERE id = ?`, dest2ID).Scan(&isSplit2)
	if isSplit1 != 0 {
		t.Error("dest1 should have is_split reset to 0")
	}
	if isSplit2 != 0 {
		t.Error("dest2 should have is_split reset to 0")
	}

	// Verify num_split counter was decremented
	var numSplit int
	DB.QueryRow(`SELECT num_split FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numSplit)
	if numSplit != 0 {
		t.Errorf("num_split: want 0, got %d", numSplit)
	}
}

func TestGetNextSplitNumber(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-017", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)

	// No splits yet — should return 1
	num, err := GetNextSplitNumber(tdcID)
	if err != nil {
		t.Fatalf("GetNextSplitNumber failed: %v", err)
	}
	if num != 1 {
		t.Errorf("first split number: want 1, got %d", num)
	}

	// Add a split, then check again
	sgID := insertShipmentGroup(t, 1)
	DB.Exec(`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (?, ?, 1, 1)`, tdcID, sgID)

	num, err = GetNextSplitNumber(tdcID)
	if err != nil {
		t.Fatalf("GetNextSplitNumber after one split failed: %v", err)
	}
	if num != 2 {
		t.Errorf("second split number: want 2, got %d", num)
	}
}

// ---------------------------------------------------------------------------
// Status & Progress
// ---------------------------------------------------------------------------

func TestRecalculateSplitProgress(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-018", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)

	// Manually mark one destination as split (simulating out-of-sync state)
	DB.Exec(`UPDATE transfer_dc_destinations SET is_split = 1 WHERE id = ?`, dest2ID)
	// Set counters to wrong values
	DB.Exec(`UPDATE transfer_dcs SET num_destinations = 0, num_split = 0 WHERE id = ?`, tdcID)

	if err := RecalculateSplitProgress(tdcID); err != nil {
		t.Fatalf("RecalculateSplitProgress failed: %v", err)
	}

	var numDest, numSplit int
	DB.QueryRow(`SELECT num_destinations, num_split FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numDest, &numSplit)
	if numDest != 2 {
		t.Errorf("num_destinations: want 2, got %d", numDest)
	}
	if numSplit != 1 {
		t.Errorf("num_split: want 1, got %d", numSplit)
	}
}

// ---------------------------------------------------------------------------
// Listing & Filtering
// ---------------------------------------------------------------------------

func TestListTransferDCsByProject(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	// Create two Transfer DCs for project 1
	dc1ID := insertTransferTestDC(t, 1, "XFER-2026-019", 1)
	insertTransferDCRow(t, dc1ID, 1, nil)

	dc2ID := insertTransferTestDC(t, 1, "XFER-2026-020", 1)
	insertTransferDCRow(t, dc2ID, 1, nil)

	tdcs, total, err := ListTransferDCsByProject(1, "", 1, 10)
	if err != nil {
		t.Fatalf("ListTransferDCsByProject failed: %v", err)
	}
	if total != 2 {
		t.Errorf("total: want 2, got %d", total)
	}
	if len(tdcs) != 2 {
		t.Errorf("returned items: want 2, got %d", len(tdcs))
	}

	// Test pagination: page size 1
	tdcsPage1, totalPaged, err := ListTransferDCsByProject(1, "", 1, 1)
	if err != nil {
		t.Fatalf("ListTransferDCsByProject (page 1) failed: %v", err)
	}
	if totalPaged != 2 {
		t.Errorf("total (paged): want 2, got %d", totalPaged)
	}
	if len(tdcsPage1) != 1 {
		t.Errorf("page 1 items: want 1, got %d", len(tdcsPage1))
	}

	tdcsPage2, _, err := ListTransferDCsByProject(1, "", 2, 1)
	if err != nil {
		t.Fatalf("ListTransferDCsByProject (page 2) failed: %v", err)
	}
	if len(tdcsPage2) != 1 {
		t.Errorf("page 2 items: want 1, got %d", len(tdcsPage2))
	}
}

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

func TestGetTransferDCSummary(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-021", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)

	// Add 2 destinations
	dest1ID := insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)
	DB.Exec(`UPDATE transfer_dcs SET num_destinations = 2 WHERE id = ?`, tdcID)

	// Set quantities: dest1 gets 2 products, dest2 gets 1 product
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 1, 50)`, dest1ID)
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 2, 30)`, dest1ID)
	DB.Exec(`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity) VALUES (?, 1, 40)`, dest2ID)

	// Create a split (mark dest1 as split)
	sgID := insertShipmentGroup(t, 1)
	res, _ := DB.Exec(
		`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by) VALUES (?, ?, 1, 1)`,
		tdcID, sgID,
	)
	splitID64, _ := res.LastInsertId()
	splitID := int(splitID64)
	DB.Exec(`UPDATE transfer_dc_destinations SET is_split = 1, split_group_id = ? WHERE id = ?`, splitID, dest1ID)
	DB.Exec(`UPDATE transfer_dcs SET num_split = 1 WHERE id = ?`, tdcID)

	summary, err := GetTransferDCSummary(tdcID)
	if err != nil {
		t.Fatalf("GetTransferDCSummary failed: %v", err)
	}
	if summary == nil {
		t.Fatal("GetTransferDCSummary returned nil")
	}

	if summary.TotalDestinations != 2 {
		t.Errorf("TotalDestinations: want 2, got %d", summary.TotalDestinations)
	}
	if summary.SplitDestinations != 1 {
		t.Errorf("SplitDestinations: want 1, got %d", summary.SplitDestinations)
	}
	if summary.PendingDestinations != 1 {
		t.Errorf("PendingDestinations: want 1, got %d", summary.PendingDestinations)
	}
	if summary.TotalProducts != 2 {
		t.Errorf("TotalProducts: want 2, got %d", summary.TotalProducts)
	}
	// Total quantity = 50 + 30 + 40 = 120
	if summary.TotalQuantity != 120 {
		t.Errorf("TotalQuantity: want 120, got %d", summary.TotalQuantity)
	}
	if summary.SplitCount != 1 {
		t.Errorf("SplitCount: want 1, got %d", summary.SplitCount)
	}
}

// ---------------------------------------------------------------------------
// Lifecycle Integration Tests
// ---------------------------------------------------------------------------

func TestUpdateTransferDCStatus_DraftToIssued(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-LC1", 1)
	insertTransferDCRow(t, dcID, 1, nil)

	// Issue the DC
	if err := UpdateTransferDCStatus(dcID, "issued"); err != nil {
		t.Fatalf("UpdateTransferDCStatus failed: %v", err)
	}

	// Verify status
	var status string
	DB.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&status)
	if status != "issued" {
		t.Errorf("status: want 'issued', got %q", status)
	}
}

func TestUpdateTransferDCStatus_IssuedToSplitting(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-LC2", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)

	// Issue the DC first
	DB.Exec(`UPDATE delivery_challans SET status = 'issued' WHERE id = ?`, dcID)

	// Add destinations
	dest1ID := insertDestination(t, tdcID, 2)
	insertDestination(t, tdcID, 3)
	DB.Exec(`UPDATE transfer_dcs SET num_destinations = 2 WHERE id = ?`, tdcID)

	// Create a split for one destination
	sgID := insertShipmentGroup(t, 1)
	_, err := CreateSplit(tdcID, sgID, []int{dest1ID}, 1)
	if err != nil {
		t.Fatalf("CreateSplit failed: %v", err)
	}

	// Simulate auto-transition: update status to splitting
	if err := UpdateTransferDCStatus(dcID, "splitting"); err != nil {
		t.Fatalf("UpdateTransferDCStatus to splitting failed: %v", err)
	}

	var status string
	DB.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&status)
	if status != "splitting" {
		t.Errorf("status: want 'splitting', got %q", status)
	}
}

func TestUpdateTransferDCStatus_SplittingToSplit(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-LC3", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)

	DB.Exec(`UPDATE delivery_challans SET status = 'issued' WHERE id = ?`, dcID)

	// Add destinations and split ALL of them
	dest1ID := insertDestination(t, tdcID, 2)
	dest2ID := insertDestination(t, tdcID, 3)
	DB.Exec(`UPDATE transfer_dcs SET num_destinations = 2 WHERE id = ?`, tdcID)

	sgID := insertShipmentGroup(t, 1)
	_, err := CreateSplit(tdcID, sgID, []int{dest1ID, dest2ID}, 1)
	if err != nil {
		t.Fatalf("CreateSplit failed: %v", err)
	}

	// All destinations are split → status should become "split"
	if err := UpdateTransferDCStatus(dcID, "split"); err != nil {
		t.Fatalf("UpdateTransferDCStatus to split failed: %v", err)
	}

	var status string
	DB.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&status)
	if status != "split" {
		t.Errorf("status: want 'split', got %q", status)
	}
}

func TestDeleteTransferDC_DraftOnly(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	dcID := insertTransferTestDC(t, 1, "XFER-2026-LC4", 1)
	tdcID := insertTransferDCRow(t, dcID, 1, nil)
	insertDestination(t, tdcID, 2)

	// Delete should work for draft
	err := DeleteTransferDC(tdcID)
	if err != nil {
		t.Fatalf("DeleteTransferDC on draft failed: %v", err)
	}

	// Verify it's gone
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&count)
	if count != 0 {
		t.Error("transfer_dcs row should have been deleted")
	}
}

func TestListTransferDCsByProject_StatusFilter(t *testing.T) {
	cleanup := setupTransferDCTestDB(t)
	defer cleanup()

	// Create a draft TDC
	dc1ID := insertTransferTestDC(t, 1, "XFER-2026-LC5", 1)
	insertTransferDCRow(t, dc1ID, 1, nil)

	// Create an issued TDC
	dc2ID := insertTransferTestDC(t, 1, "XFER-2026-LC6", 1)
	insertTransferDCRow(t, dc2ID, 1, nil)
	DB.Exec(`UPDATE delivery_challans SET status = 'issued' WHERE id = ?`, dc2ID)

	// Filter by draft
	draftTDCs, draftTotal, err := ListTransferDCsByProject(1, "draft", 1, 20)
	if err != nil {
		t.Fatalf("ListTransferDCsByProject(draft) failed: %v", err)
	}
	if draftTotal != 1 {
		t.Errorf("draft total: want 1, got %d", draftTotal)
	}
	if len(draftTDCs) != 1 {
		t.Errorf("draft items: want 1, got %d", len(draftTDCs))
	}

	// Filter by issued
	issuedTDCs, issuedTotal, err := ListTransferDCsByProject(1, "issued", 1, 20)
	if err != nil {
		t.Fatalf("ListTransferDCsByProject(issued) failed: %v", err)
	}
	if issuedTotal != 1 {
		t.Errorf("issued total: want 1, got %d", issuedTotal)
	}
	if len(issuedTDCs) != 1 {
		t.Errorf("issued items: want 1, got %d", len(issuedTDCs))
	}

	// No filter → all
	allTDCs, allTotal, err := ListTransferDCsByProject(1, "", 1, 20)
	if err != nil {
		t.Fatalf("ListTransferDCsByProject(all) failed: %v", err)
	}
	if allTotal != 2 {
		t.Errorf("all total: want 2, got %d", allTotal)
	}
	if len(allTDCs) != 2 {
		t.Errorf("all items: want 2, got %d", len(allTDCs))
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func intPtr(v int) *int {
	return &v
}
