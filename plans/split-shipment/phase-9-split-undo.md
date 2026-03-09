# Phase 9: Split Undo & Child Group Deletion

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 6 (Split Service) — Split creation must exist to undo
- Phase 7 (Split Wizard) — Child groups must exist

## Overview

Allow users to undo a split operation by deleting a child shipment group, returning its destinations and serials to the Transfer DC's un-split pool.

**Rules:**
- Can only delete a child shipment group if its Transit DC has NOT been issued
- Once a child TDC is issued, that split is permanent
- Deleting a child group:
  1. Deletes all DCs (Transit + Officials) in the child shipment group
  2. Frees serial numbers (removes from child DC line items)
  3. Marks destinations as un-split in the parent Transfer DC
  4. Deletes the split record
  5. Updates Transfer DC split progress counters
  6. May transition Transfer DC status back (split → splitting, or splitting → issued)

---

## Modified Files

| File | Changes |
|------|---------|
| `internal/services/split_shipment.go` | Add `DeleteSplitShipment()` function |
| `internal/services/split_shipment_test.go` | Add undo tests |
| `internal/handlers/transfer_dc.go` | Add delete-split handler |
| `internal/database/transfer_dcs.go` | Add `ResetDestinationsForSplit()` function |
| `internal/database/shipment_groups.go` | Add `DeleteShipmentGroupWithDCs()` function |
| `components/pages/transfer_dcs/detail.templ` | Add delete button per split (conditional) |
| `cmd/server/main.go` | Add delete-split route |

---

## Tests to Write First

### Deletion
- [ ] `TestDeleteSplitShipment_HappyPath` — Delete draft child group, destinations return to pool
- [ ] `TestDeleteSplitShipment_SerialsFreed` — Serials available again after deletion
- [ ] `TestDeleteSplitShipment_DestinationsReset` — `is_split` set to 0, `split_group_id` cleared
- [ ] `TestDeleteSplitShipment_SplitRecordDeleted` — `transfer_dc_splits` record removed
- [ ] `TestDeleteSplitShipment_ProgressUpdated` — `num_split` counter decremented
- [ ] `TestDeleteSplitShipment_AllDCsDeleted` — Child TDC + all ODCs removed

### Status Transitions
- [ ] `TestDeleteSplit_SplitToSplitting` — If was fully split, status reverts to splitting
- [ ] `TestDeleteSplit_SplittingToIssued` — If last split deleted, status reverts to issued
- [ ] `TestDeleteSplit_SplittingStaysSplitting` — If other splits remain, stays splitting

### Validation
- [ ] `TestDeleteSplit_BlockedIfChildIssued` — Error if child TDC has been issued
- [ ] `TestDeleteSplit_BlockedIfChildPartiallyIssued` — Error if any ODC in child group is issued
- [ ] `TestDeleteSplit_TransferDCMustExist` — Error if Transfer DC not found

### Edge Cases
- [ ] `TestDeleteAllSplits_ReturnsToIssued` — Delete all splits, Transfer DC back to "issued"
- [ ] `TestDeleteSplit_ConcurrentAccess` — Two users try to delete same split
- [ ] `TestDeleteSplit_ChildGroupAlreadyDeleted` — Idempotent handling

---

## Implementation Steps

### 1. Implement DeleteSplitShipment — `internal/services/split_shipment.go`

```go
func DeleteSplitShipment(splitID int, userID int) error {
    // 1. Get split record
    split, err := database.GetSplitByID(splitID)
    if err != nil {
        return fmt.Errorf("split not found: %w", err)
    }

    // 2. Get child shipment group
    group, err := database.GetShipmentGroup(split.ShipmentGroupID)
    if err != nil {
        return fmt.Errorf("child group not found: %w", err)
    }

    // 3. Check if child group can be deleted (no issued DCs)
    childDCs, err := database.GetShipmentGroupDCs(split.ShipmentGroupID)
    for _, dc := range childDCs {
        if dc.Status == "issued" {
            return fmt.Errorf("cannot delete split: child DC %s has been issued", dc.DCNumber)
        }
    }

    // 4. Begin transaction
    tx, err := database.DB.Begin()

    // 5. Delete all DCs in child shipment group
    //    - Delete serial_numbers for each DC's line items
    //    - Delete dc_line_items for each DC
    //    - Delete dc_transit_details (for child TDC)
    //    - Delete delivery_challans records
    for _, dc := range childDCs {
        deleteAllDCDataInTx(tx, dc.ID)
    }

    // 6. Delete shipment group record
    deleteShipmentGroupInTx(tx, split.ShipmentGroupID)

    // 7. Get destinations that were in this split
    destinations := getDestinationsForSplit(tx, split.ID)

    // 8. Reset destination split status
    resetDestinationSplitStatusInTx(tx, destinations, split.ID)

    // 9. Delete split record
    deleteSplitRecordInTx(tx, split.ID)

    // 10. Recalculate split progress
    recalculateSplitProgressInTx(tx, split.TransferDCID)

    // 11. Update Transfer DC status
    //     - If num_split == 0: status = "issued"
    //     - If num_split > 0 && num_split < num_destinations: status = "splitting"
    //     - If num_split == num_destinations: status = "split" (shouldn't happen after delete)
    updateTransferDCStatusAfterDelete(tx, split.TransferDCID)

    // 12. Commit
    tx.Commit()
    return nil
}
```

### 2. Add database helper functions

```go
// ResetDestinationsForSplit marks destinations as un-split for a given split.
func ResetDestinationsForSplit(tx *sql.Tx, splitID int) error {
    _, err := tx.Exec(`
        UPDATE transfer_dc_destinations
        SET is_split = 0, split_group_id = NULL
        WHERE split_group_id = ?`, splitID)
    return err
}

// DeleteShipmentGroupWithDCs deletes a shipment group and all its DCs in a transaction.
func DeleteShipmentGroupWithDCs(tx *sql.Tx, groupID int) error {
    // Delete serial_numbers → dc_line_items → dc_transit_details → delivery_challans → shipment_groups
    // CASCADE handles most of this, but we need to ensure proper ordering
}

// CanDeleteSplit checks if a split can be deleted (no issued child DCs).
func CanDeleteSplit(splitID int) (bool, string, error) {
    // Returns (canDelete, reason, error)
}
```

### 3. Add handler — `internal/handlers/transfer_dc.go`

```go
// DeleteSplitHandler handles deletion of a split (undo).
func DeleteSplitHandler(c echo.Context) error {
    splitID := parseIntParam(c, "splitid")

    // Check permissions
    canDelete, reason, err := database.CanDeleteSplit(splitID)
    if !canDelete {
        auth.SetFlash(c.Request(), "error", reason)
        return redirectToTransferDCDetail(c)
    }

    err = services.DeleteSplitShipment(splitID, userID)
    if err != nil {
        auth.SetFlash(c.Request(), "error", "Failed to undo split: " + err.Error())
    } else {
        auth.SetFlash(c.Request(), "success", "Split undone. Destinations returned to pool.")
    }

    return redirectToTransferDCDetail(c)
}
```

### 4. Update detail page template — `components/pages/transfer_dcs/detail.templ`

In the Split Progress section, add a delete button per split:

```templ
for _, split := range splits {
    <div class="split-card">
        // ... split info ...
        if split.CanDelete {
            <form method="POST" action={...delete URL...}>
                <input type="hidden" name="gorilla.csrf.Token" value={csrfToken}/>
                <button type="submit" class="text-red-600 hover:text-red-800"
                    onclick="return confirm('Undo this split? All child DCs will be deleted.')">
                    Undo Split
                </button>
            </form>
        } else {
            <span class="text-gray-400 text-sm">Issued (locked)</span>
        }
    </div>
}
```

### 5. Register route — `cmd/server/main.go`

```go
projectRoutes.DELETE("/transfer-dcs/:tdcid/splits/:splitid", handlers.DeleteSplitHandler)
// Or POST with _method override:
projectRoutes.POST("/transfer-dcs/:tdcid/splits/:splitid/delete", handlers.DeleteSplitHandler)
```

---

## Serial Tracking on Undo

When a split is undone:
- Child DC serial_numbers records are deleted (CASCADE from dc_line_items)
- The serials are NOT deleted from the parent Transfer DC — they remain in the master list
- The serials become "available" again for future splits
- Available serials = Parent serials − Serials in remaining (non-deleted) splits

---

## Acceptance Criteria

- [ ] Draft child shipment groups can be deleted, returning destinations to un-split pool
- [ ] Issued child groups CANNOT be deleted (error message shown)
- [ ] All child DCs (TDC + ODCs) are deleted when split is undone
- [ ] Serials become available again for future splits
- [ ] Destinations reset: `is_split = 0`, `split_group_id = NULL`
- [ ] Split progress counters updated correctly
- [ ] Transfer DC status transitions correctly on undo
- [ ] Confirmation dialog before undo action
- [ ] Flash messages on success/failure
- [ ] All operations are transactional
- [ ] All tests pass
