# Phase 6: Split Operation Data Layer & Service

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 1 (Database Schema)
- Phase 2 (DC Numbering)
- Phase 3 (Data Access Layer)
- Phase 4 (Transfer Wizard) — Transfer DCs must exist
- Phase 5 (Transfer Detail) — Lifecycle transitions needed

## Overview

Implement the core business logic for splitting a Transfer DC into child shipment groups. This is the heart of the Split Shipment feature. When a user performs a split:

1. Select N destinations from the un-split pool
2. Provide transporter/vehicle details for the small vehicle
3. Enter serial numbers per product (validated against parent Transfer DC)
4. System creates a child shipment group (1 Transit DC + N Official DCs)
5. System marks those destinations as split in the Transfer DC

The split operation reuses the existing `CreateShipmentGroupDCs()` service with adaptations for inheriting data from the parent Transfer DC.

---

## New Files

| File | Purpose |
|------|---------|
| `internal/services/split_shipment.go` | Split operation service logic |
| `internal/services/split_shipment_test.go` | Tests for split service |

## Modified Files

| File | Changes |
|------|---------|
| `internal/services/dc_generation.go` | Minor: ensure `CreateShipmentGroupDCs` can accept `transfer_dc_id` |
| `internal/database/shipment_groups.go` | Accept `transfer_dc_id` in CreateShipmentGroup |
| `internal/database/transfer_dcs.go` | Add split-specific queries if not in Phase 3 |

---

## Tests to Write First

### Split Creation
- [ ] `TestCreateSplitShipment_HappyPath` — Split 5 destinations into one group, verify TDC + 5 ODCs created
- [ ] `TestCreateSplitShipment_InheritsRates` — Child TDC has same rates as parent Transfer DC
- [ ] `TestCreateSplitShipment_InheritsTaxType` — Child inherits tax_type and reverse_charge
- [ ] `TestCreateSplitShipment_InheritsAddresses` — Child inherits bill_from, dispatch_from, bill_to
- [ ] `TestCreateSplitShipment_CorrectQuantities` — Each ODC has correct per-destination quantities
- [ ] `TestCreateSplitShipment_TransitDCTotalQuantities` — Child TDC has sum of all destination quantities

### Serial Validation
- [ ] `TestSplitSerialValidation_MustBelongToParent` — Reject serials not in parent Transfer DC
- [ ] `TestSplitSerialValidation_ExactCount` — Serial count must match total quantity
- [ ] `TestSplitSerialValidation_NoDuplicates` — No duplicate serials within a split
- [ ] `TestSplitSerialValidation_NotAlreadyUsedInOtherSplit` — Serial can only be used in ONE split
- [ ] `TestSplitSerialValidation_PerProductCount` — Per-product serial count matches product total

### Destination Validation
- [ ] `TestSplitDestinationValidation_MustBeUnsplit` — Selected destinations must not already be split
- [ ] `TestSplitDestinationValidation_MustBelongToTransferDC` — Destinations must belong to this Transfer DC
- [ ] `TestSplitDestinationValidation_AtLeastOne` — At least one destination required

### Status Transitions
- [ ] `TestSplitUpdatesTransferDCStatus_IssuedToSplitting` — First split changes status
- [ ] `TestSplitUpdatesTransferDCStatus_SplittingToSplit` — Last split completes status
- [ ] `TestSplitUpdatesSplitProgress` — num_split counter updated correctly

### Edge Cases
- [ ] `TestSplitShipment_SingleDestination` — Split with just 1 destination (1 TDC + 1 ODC)
- [ ] `TestSplitShipment_AllRemainingDestinations` — Final split covers all remaining → status becomes "split"
- [ ] `TestSplitShipment_TransferDCMustBeIssued` — Cannot split draft Transfer DC
- [ ] `TestSplitShipment_TransferDCCannotBeSplit` — Cannot split already-fully-split Transfer DC

---

## Implementation Steps

### 1. Define SplitParams — `internal/services/split_shipment.go`

```go
type SplitShipmentParams struct {
    TransferDCID    int
    ProjectID       int
    DestinationIDs  []int    // IDs from transfer_dc_destinations table
    TransporterName string
    VehicleNumber   string
    EwayBillNumber  string
    DocketNumber    string
    Notes           string
    // Per-product serial assignments
    ProductSerials  []SplitProductSerials
    CreatedBy       int
}

type SplitProductSerials struct {
    ProductID     int
    SerialNumbers []string
    // Per-destination serial assignments (which serials go to which location)
    Assignments   map[int][]string  // map[shipToAddressID][]serials
}
```

### 2. Implement CreateSplitShipment — `internal/services/split_shipment.go`

```go
func CreateSplitShipment(params SplitShipmentParams) (*models.TransferDCSplit, error) {
    // === VALIDATION ===
    // 1. Get Transfer DC, verify status is "issued" or "splitting"
    transferDC, err := database.GetTransferDCByDCID(params.TransferDCID)

    // 2. Get parent DC for pricing and address data
    parentDC, err := database.GetDeliveryChallanByID(transferDC.DCID)

    // 3. Validate all destination IDs belong to this Transfer DC and are un-split
    unsplitDests, err := database.GetUnsplitDestinations(transferDC.ID)
    // ... validate params.DestinationIDs ⊆ unsplitDests

    // 4. Get quantities for selected destinations
    destQuantities, err := database.GetQuantitiesForDestinations(params.DestinationIDs)

    // 5. Validate serials:
    //    a) All serials exist in parent Transfer DC's serial list
    //    b) No serial already used in another split's child group
    //    c) Per-product count matches sum of destination quantities
    //    d) No duplicates
    parentSerials := getParentDCSerials(parentDC.ID)
    usedSerials := getSerialsUsedInExistingSplits(transferDC.ID)
    validateSplitSerials(params.ProductSerials, destQuantities, parentSerials, usedSerials)

    // === CREATION (in transaction) ===
    tx, err := database.DB.Begin()

    // 6. Get next split number
    splitNum, err := database.GetNextSplitNumber(transferDC.ID)

    // 7. Build ShipmentParams from Transfer DC data + split params
    shipmentParams := buildShipmentParamsFromSplit(parentDC, transferDC, params, destQuantities)
    // Key: inherits bill_from, dispatch_from, bill_to, rates, tax_type from parent
    // Key: transit ship-to = first destination (or user-chosen)

    // 8. Create child shipment group via existing CreateShipmentGroupDCs
    //    - Pass transfer_dc_id so child group links back to parent
    groupID, err := createShipmentGroupDCsInTx(tx, shipmentParams)

    // 9. Create split record
    split := database.CreateSplitInTx(tx, transferDC.ID, groupID, splitNum, params.CreatedBy)

    // 10. Mark destinations as split
    database.UpdateDestinationSplitStatusInTx(tx, params.DestinationIDs, split.ID, true)

    // 11. Update Transfer DC split progress counters
    database.RecalculateSplitProgressInTx(tx, transferDC.ID)

    // 12. Update Transfer DC status if needed
    //     - If first split: issued → splitting
    //     - If all destinations now split: splitting → split
    newStatus := determineTransferDCStatus(transferDC.ID, tx)
    database.UpdateTransferDCStatusInTx(tx, parentDC.ID, newStatus)

    // 13. Commit transaction
    tx.Commit()

    return split, nil
}
```

### 3. Implement helper functions

```go
// buildShipmentParamsFromSplit converts Transfer DC data + split params into ShipmentParams
func buildShipmentParamsFromSplit(
    parentDC *models.DeliveryChallan,
    transferDC *models.TransferDC,
    splitParams SplitShipmentParams,
    destQuantities map[int][]models.TransferDCDestinationQty,
) services.ShipmentParams {
    // Map destination quantities to ShipmentLineItems
    // Inherit rates from parent DC line items
    // Set transfer_dc_id on the shipment group
}

// validateSplitSerials performs all serial validation for a split operation
func validateSplitSerials(
    productSerials []SplitProductSerials,
    destQuantities map[int][]models.TransferDCDestinationQty,
    parentSerials map[int][]string,  // map[productID][]serials from parent
    usedSerials map[string]bool,     // serials already used in other splits
) map[string]string  // returns field → error message map

// determineTransferDCStatus checks split progress and returns the correct status
func determineTransferDCStatus(transferDCID int, tx *sql.Tx) string

// getAvailableSerials returns serials from parent that are not yet used in splits
func getAvailableSerials(transferDCID int, productID int) ([]string, error)
```

### 4. Modify CreateShipmentGroupDCs — `internal/services/dc_generation.go`

Add `TransferDCID` field to `ShipmentParams`:
```go
type ShipmentParams struct {
    // ... existing fields ...
    TransferDCID int  // NEW: 0 for standalone shipments, >0 for split child groups
}
```

In `CreateShipmentGroupDCs()`, pass `transfer_dc_id` to `CreateShipmentGroup()` when non-zero.

### 5. Update shipment_groups database — `internal/database/shipment_groups.go`

```go
// CreateShipmentGroup: add transfer_dc_id and split_id columns to INSERT
func CreateShipmentGroup(group *models.ShipmentGroup) (int, error) {
    // Updated INSERT to include transfer_dc_id, split_id
}
```

---

## Serial Tracking Design

### Available Serials Calculation
For each product in a Transfer DC:
```
Available Serials = Parent DC Serials − Serials Used In Existing Splits
```

The split wizard UI will show available serials as a reference. The user enters/scans serials for the current split, and validation ensures they come from the available pool.

### Serial Flow
```
Transfer DC (parent)
  └─ dc_line_items → serial_numbers (ALL serials, bulk)
       │
       ├─ Split #1 → Shipment Group #42
       │    └─ Transit DC → dc_line_items → serial_numbers (subset)
       │    └─ Official DCs (no serials)
       │
       ├─ Split #2 → Shipment Group #43
       │    └─ Transit DC → dc_line_items → serial_numbers (subset)
       │    └─ Official DCs (no serials)
       │
       └─ (Remaining serials available for future splits)
```

---

## Acceptance Criteria

- [ ] Split operation creates child shipment group (1 TDC + N ODCs) correctly
- [ ] Child TDC inherits rates, tax type, reverse charge, bill_from, dispatch_from, bill_to from parent
- [ ] Child TDC has correct total quantities (sum of selected destinations)
- [ ] Each child ODC has correct per-destination quantities
- [ ] Serial validation: must belong to parent, not already used in another split, count matches
- [ ] Destination validation: must be un-split, must belong to Transfer DC
- [ ] Transfer DC status transitions automatically: issued→splitting→split
- [ ] Split progress counters (num_split/num_destinations) stay accurate
- [ ] Split number increments sequentially per Transfer DC
- [ ] Child shipment group links back to parent Transfer DC (transfer_dc_id)
- [ ] All operations are transactional (atomic commit/rollback)
- [ ] Existing standalone shipment creation is NOT affected
- [ ] All tests pass
