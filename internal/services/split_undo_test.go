package services

import (
	"database/sql"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper: performSplit creates a split via CreateSplitShipment and returns
// the SplitResult. This simplifies test setup for delete/undo tests.
// ---------------------------------------------------------------------------

func performSplit(t *testing.T, db *sql.DB, tdcID, dcID, projectID int, destIDs []int, p1Serials, p2Serials []string) *SplitResult {
	t.Helper()

	params := SplitShipmentParams{
		TransferDCID:    tdcID,
		ParentDCID:      dcID,
		ProjectID:       projectID,
		DestinationIDs:  destIDs,
		TransporterName: "Test Transport",
		VehicleNumber:   "TS01-XX-1234",
		ProductSerials: []SplitProductSerials{
			{ProductID: 1, SerialNumbers: p1Serials},
			{ProductID: 2, SerialNumbers: p2Serials},
		},
		CreatedBy: 1,
	}

	result, err := CreateSplitShipment(db, params)
	if err != nil {
		t.Fatalf("performSplit failed: %v", err)
	}
	return result
}

// ---------------------------------------------------------------------------
// Deletion Tests
// ---------------------------------------------------------------------------

func TestDeleteSplitShipment_HappyPath(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Create a split with destinations 0,1 (addr 100,200: P1=10, P2=5)
	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)

	// Delete the split
	err := DeleteSplitShipment(db, result.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	// Verify destinations returned to unsplit pool
	var unsplitCount int
	db.QueryRow(`SELECT COUNT(*) FROM transfer_dc_destinations WHERE id IN (?, ?) AND is_split = 0`,
		destIDs[0], destIDs[1]).Scan(&unsplitCount)
	if unsplitCount != 2 {
		t.Errorf("expected 2 destinations returned to unsplit, got %d", unsplitCount)
	}

	// Verify child shipment group was deleted
	var sgCount int
	db.QueryRow(`SELECT COUNT(*) FROM shipment_groups WHERE id = ?`, result.GroupID).Scan(&sgCount)
	if sgCount != 0 {
		t.Errorf("expected child shipment group to be deleted, got %d", sgCount)
	}

	// Verify Transfer DC status reverted to issued (0 of 4 destinations split)
	var dcStatus string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&dcStatus)
	if dcStatus != "issued" {
		t.Errorf("Transfer DC status: want 'issued', got %q", dcStatus)
	}
}

func TestDeleteSplitShipment_SerialsFreed(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split dests 0,1 using serials SN-P1-001..010 and SN-P2-001..005
	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)

	// Delete the split
	err := DeleteSplitShipment(db, result.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	// Verify child serial_numbers were deleted (via cascade from deleted DCs)
	var childSerialCount int
	db.QueryRow(`SELECT COUNT(*) FROM serial_numbers sn
		INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		WHERE dc.shipment_group_id = ?`, result.GroupID).Scan(&childSerialCount)
	if childSerialCount != 0 {
		t.Errorf("expected 0 child serials after undo, got %d", childSerialCount)
	}

	// Parent serials still exist
	var parentSerialCount int
	db.QueryRow(`SELECT COUNT(*) FROM serial_numbers sn
		INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		WHERE li.dc_id = ?`, dcID).Scan(&parentSerialCount)
	if parentSerialCount != 30 { // 20 P1 + 10 P2
		t.Errorf("expected 30 parent serials intact, got %d", parentSerialCount)
	}

	// Freed serials can be used in a new split
	result2 := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10), // same serials as before — should work now
		serialRange("SN-P2-", 1, 5),
	)
	if result2 == nil {
		t.Fatal("expected to re-split with freed serials, got nil")
	}
}

func TestDeleteSplitShipment_DestinationsReset(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)

	// Verify destinations are split before undo
	var splitBefore int
	db.QueryRow(`SELECT COUNT(*) FROM transfer_dc_destinations WHERE id IN (?, ?) AND is_split = 1`,
		destIDs[0], destIDs[1]).Scan(&splitBefore)
	if splitBefore != 2 {
		t.Fatalf("pre-condition: expected 2 split destinations, got %d", splitBefore)
	}

	err := DeleteSplitShipment(db, result.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	// Verify is_split = 0 and split_group_id = NULL
	var isSplit int
	var splitGroupID sql.NullInt64
	db.QueryRow(`SELECT is_split, split_group_id FROM transfer_dc_destinations WHERE id = ?`, destIDs[0]).Scan(&isSplit, &splitGroupID)
	if isSplit != 0 {
		t.Errorf("destination %d: is_split should be 0, got %d", destIDs[0], isSplit)
	}
	if splitGroupID.Valid {
		t.Errorf("destination %d: split_group_id should be NULL, got %d", destIDs[0], splitGroupID.Int64)
	}
}

func TestDeleteSplitShipment_SplitRecordDeleted(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0]},
		serialRange("SN-P1-", 1, 5),
		serialRange("SN-P2-", 1, 3),
	)

	err := DeleteSplitShipment(db, result.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	var splitRecordCount int
	db.QueryRow(`SELECT COUNT(*) FROM transfer_dc_splits WHERE id = ?`, result.SplitID).Scan(&splitRecordCount)
	if splitRecordCount != 0 {
		t.Errorf("expected split record deleted, got count=%d", splitRecordCount)
	}
}

func TestDeleteSplitShipment_ProgressUpdated(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split 2 of 4 destinations → num_split = 2
	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)

	var numSplitBefore int
	db.QueryRow(`SELECT num_split FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numSplitBefore)
	if numSplitBefore != 2 {
		t.Fatalf("pre-condition: expected num_split=2, got %d", numSplitBefore)
	}

	err := DeleteSplitShipment(db, result.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	var numSplitAfter int
	db.QueryRow(`SELECT num_split FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numSplitAfter)
	if numSplitAfter != 0 {
		t.Errorf("num_split: want 0, got %d", numSplitAfter)
	}
}

func TestDeleteSplitShipment_AllDCsDeleted(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split 2 destinations → 1 transit DC + 2 official DCs
	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)

	// Verify child DCs exist before undo
	var dcCountBefore int
	db.QueryRow(`SELECT COUNT(*) FROM delivery_challans WHERE shipment_group_id = ?`, result.GroupID).Scan(&dcCountBefore)
	if dcCountBefore != 3 { // 1 transit + 2 official
		t.Fatalf("pre-condition: expected 3 child DCs, got %d", dcCountBefore)
	}

	err := DeleteSplitShipment(db, result.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	// All child DCs deleted
	var dcCountAfter int
	db.QueryRow(`SELECT COUNT(*) FROM delivery_challans WHERE shipment_group_id = ?`, result.GroupID).Scan(&dcCountAfter)
	if dcCountAfter != 0 {
		t.Errorf("expected 0 child DCs after undo, got %d", dcCountAfter)
	}

	// Line items deleted
	var lineItemCount int
	db.QueryRow(`SELECT COUNT(*) FROM dc_line_items li
		INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		WHERE dc.shipment_group_id = ?`, result.GroupID).Scan(&lineItemCount)
	if lineItemCount != 0 {
		t.Errorf("expected 0 child line items after undo, got %d", lineItemCount)
	}
}

// ---------------------------------------------------------------------------
// Status Transition Tests
// ---------------------------------------------------------------------------

func TestDeleteSplit_SplitToSplitting(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split all 4 destinations in 2 batches → status = "split"
	performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)
	result2 := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[2], destIDs[3]},
		serialRange("SN-P1-", 11, 20),
		serialRange("SN-P2-", 6, 10),
	)

	// Verify status is "split"
	var statusBefore string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&statusBefore)
	if statusBefore != "split" {
		t.Fatalf("pre-condition: expected status 'split', got %q", statusBefore)
	}

	// Delete second split → 2 of 4 split → status should be "splitting"
	err := DeleteSplitShipment(db, result2.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	var statusAfter string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&statusAfter)
	if statusAfter != "splitting" {
		t.Errorf("Transfer DC status after undo: want 'splitting', got %q", statusAfter)
	}
}

func TestDeleteSplit_SplittingToIssued(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split 1 of 4 → status = "splitting"
	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0]},
		serialRange("SN-P1-", 1, 5),
		serialRange("SN-P2-", 1, 3),
	)

	var statusBefore string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&statusBefore)
	if statusBefore != "splitting" {
		t.Fatalf("pre-condition: expected status 'splitting', got %q", statusBefore)
	}

	// Delete the only split → 0 of 4 → status = "issued"
	err := DeleteSplitShipment(db, result.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	var statusAfter string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&statusAfter)
	if statusAfter != "issued" {
		t.Errorf("Transfer DC status after undo: want 'issued', got %q", statusAfter)
	}
}

func TestDeleteSplit_SplittingStaysSplitting(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Split 3 of 4 in 2 batches → status = "splitting"
	performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)
	result2 := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[2]},
		serialRange("SN-P1-", 11, 15),
		serialRange("SN-P2-", 6, 8),
	)

	// Delete second split → 2 of 4 still split → stays "splitting"
	err := DeleteSplitShipment(db, result2.SplitID)
	if err != nil {
		t.Fatalf("DeleteSplitShipment failed: %v", err)
	}

	var statusAfter string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&statusAfter)
	if statusAfter != "splitting" {
		t.Errorf("Transfer DC status after undo: want 'splitting', got %q", statusAfter)
	}
}

// ---------------------------------------------------------------------------
// Validation Tests
// ---------------------------------------------------------------------------

func TestDeleteSplit_BlockedIfChildIssued(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0]},
		serialRange("SN-P1-", 1, 5),
		serialRange("SN-P2-", 1, 3),
	)

	// Issue the child transit DC
	db.Exec(`UPDATE delivery_challans SET status = 'issued' WHERE shipment_group_id = ? AND dc_type = 'transit'`, result.GroupID)

	err := DeleteSplitShipment(db, result.SplitID)
	if err == nil {
		t.Fatal("expected error when deleting split with issued child DC")
	}
}

func TestDeleteSplit_BlockedIfChildPartiallyIssued(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	result := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)

	// Issue one of the official DCs (not the transit)
	var oneOfficialID int
	db.QueryRow(`SELECT id FROM delivery_challans WHERE shipment_group_id = ? AND dc_type = 'official' LIMIT 1`, result.GroupID).Scan(&oneOfficialID)
	db.Exec(`UPDATE delivery_challans SET status = 'issued' WHERE id = ?`, oneOfficialID)

	err := DeleteSplitShipment(db, result.SplitID)
	if err == nil {
		t.Fatal("expected error when deleting split with partially issued child DCs")
	}
}

func TestDeleteSplit_TransferDCMustExist(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	// Try to delete a non-existent split
	err := DeleteSplitShipment(db, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent split")
	}
}

// ---------------------------------------------------------------------------
// Edge Case Tests
// ---------------------------------------------------------------------------

func TestDeleteAllSplits_ReturnsToIssued(t *testing.T) {
	db := setupSplitTestDB(t)
	defer db.Close()

	projectID := insertTestProject(t, db, "UndoProject", "UND")
	tdcID, dcID, destIDs := createTestTransferDC(t, db, projectID)

	// Create 2 splits
	result1 := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[0], destIDs[1]},
		serialRange("SN-P1-", 1, 10),
		serialRange("SN-P2-", 1, 5),
	)
	result2 := performSplit(t, db, tdcID, dcID, projectID,
		[]int{destIDs[2], destIDs[3]},
		serialRange("SN-P1-", 11, 20),
		serialRange("SN-P2-", 6, 10),
	)

	// Status should be "split" (all 4 destinations split)
	var statusBefore string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&statusBefore)
	if statusBefore != "split" {
		t.Fatalf("pre-condition: expected 'split', got %q", statusBefore)
	}

	// Delete both splits
	if err := DeleteSplitShipment(db, result2.SplitID); err != nil {
		t.Fatalf("DeleteSplitShipment(2) failed: %v", err)
	}
	if err := DeleteSplitShipment(db, result1.SplitID); err != nil {
		t.Fatalf("DeleteSplitShipment(1) failed: %v", err)
	}

	// Transfer DC should be back to "issued"
	var statusAfter string
	db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, dcID).Scan(&statusAfter)
	if statusAfter != "issued" {
		t.Errorf("Transfer DC status after deleting all splits: want 'issued', got %q", statusAfter)
	}

	// All destinations are unsplit
	var unsplitCount int
	db.QueryRow(`SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ? AND is_split = 0`, tdcID).Scan(&unsplitCount)
	if unsplitCount != 4 {
		t.Errorf("expected 4 unsplit destinations, got %d", unsplitCount)
	}

	// num_split = 0
	var numSplit int
	db.QueryRow(`SELECT num_split FROM transfer_dcs WHERE id = ?`, tdcID).Scan(&numSplit)
	if numSplit != 0 {
		t.Errorf("num_split: want 0, got %d", numSplit)
	}
}
