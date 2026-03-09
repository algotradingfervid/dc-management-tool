# Phase 3: Transfer DC Data Access Layer

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 1 (Database Schema) — tables must exist
- Phase 2 (DC Numbering) — constants and type codes needed

## Overview

Create the database access functions for Transfer DCs — CRUD operations for the `transfer_dcs`, `transfer_dc_destinations`, `transfer_dc_destination_quantities`, and `transfer_dc_splits` tables. This mirrors the existing pattern in `internal/database/` with one file per entity.

---

## New Files

| File | Purpose |
|------|---------|
| `internal/database/transfer_dcs.go` | All Transfer DC database operations |
| `internal/database/transfer_dcs_test.go` | Tests for Transfer DC database operations |

## Modified Files

| File | Changes |
|------|---------|
| `internal/database/delivery_challans.go` | Minor: ensure `insertDCWithLineItemsAndSerials` works for transfer type |
| `internal/database/shipment_groups.go` | Add `transfer_dc_id` handling in queries |

---

## Tests to Write First

### Transfer DC CRUD
- [ ] `TestCreateTransferDC` — Create a Transfer DC record with hub address, template, transporter details
- [ ] `TestGetTransferDC` — Retrieve by ID with joined fields (hub address name, template name, DC number)
- [ ] `TestGetTransferDCByDCID` — Retrieve by parent delivery_challans.id
- [ ] `TestUpdateTransferDC` — Update transporter, vehicle, notes fields
- [ ] `TestDeleteTransferDC` — Delete and verify cascade to destinations and splits

### Destination Management
- [ ] `TestAddTransferDCDestinations` — Add multiple destinations in batch
- [ ] `TestGetTransferDCDestinations` — List destinations with quantities and split status
- [ ] `TestGetUnsplitDestinations` — List only destinations where `is_split = 0`
- [ ] `TestGetSplitDestinations` — List only destinations where `is_split = 1`
- [ ] `TestUpdateDestinationSplitStatus` — Mark destinations as split
- [ ] `TestResetDestinationSplitStatus` — Unmark destinations (for undo)

### Quantity Grid
- [ ] `TestSetDestinationQuantities` — Set product quantities for a destination
- [ ] `TestGetDestinationQuantities` — Get quantities for a destination with product info
- [ ] `TestGetQuantityGrid` — Get full grid (all destinations x all products) for a Transfer DC
- [ ] `TestUpdateDestinationQuantities` — Update quantities (for edit wizard)

### Split Tracking
- [ ] `TestCreateSplit` — Create a split record linking Transfer DC to shipment group
- [ ] `TestGetSplitsByTransferDCID` — List all splits for a Transfer DC
- [ ] `TestGetSplitByShipmentGroupID` — Get split record by child shipment group
- [ ] `TestDeleteSplit` — Delete split record and verify destination status resets
- [ ] `TestGetNextSplitNumber` — Get next sequential split number for a Transfer DC

### List & Filter
- [ ] `TestListTransferDCsByProject` — List all Transfer DCs for a project with pagination
- [ ] `TestListTransferDCsByStatus` — Filter by status (draft, issued, splitting, split)
- [ ] `TestGetTransferDCSplitProgress` — Get split progress (N/M destinations split)

---

## Implementation Steps

### 1. Create `internal/database/transfer_dcs.go`

```go
package database

// === Transfer DC Core CRUD ===

// CreateTransferDC creates the transfer_dcs record (NOT the parent delivery_challan).
// The parent DC should be created first via insertDCWithLineItemsAndSerials.
func CreateTransferDC(transferDC *models.TransferDC) (int, error)

// GetTransferDC retrieves a transfer DC by its ID with joined fields.
func GetTransferDC(id int) (*models.TransferDC, error)

// GetTransferDCByDCID retrieves a transfer DC by its parent delivery_challans.id.
func GetTransferDCByDCID(dcID int) (*models.TransferDC, error)

// UpdateTransferDC updates mutable fields (transporter, vehicle, notes, etc.).
func UpdateTransferDC(transferDC *models.TransferDC) error

// DeleteTransferDC deletes a transfer DC and cascades to destinations/splits.
func DeleteTransferDC(id int) error

// === Destination Management ===

// AddTransferDCDestinations inserts multiple destinations in a batch.
// Each destination includes a ship_to_address_id.
func AddTransferDCDestinations(transferDCID int, shipToAddressIDs []int) error

// GetTransferDCDestinations retrieves all destinations for a transfer DC.
// Includes joined address names and split status.
func GetTransferDCDestinations(transferDCID int) ([]*models.TransferDCDestination, error)

// GetUnsplitDestinations retrieves destinations not yet assigned to a split group.
func GetUnsplitDestinations(transferDCID int) ([]*models.TransferDCDestination, error)

// GetSplitDestinations retrieves destinations that have been split.
func GetSplitDestinations(transferDCID int) ([]*models.TransferDCDestination, error)

// UpdateDestinationSplitStatus marks destinations as split (or un-split for undo).
func UpdateDestinationSplitStatus(destinationIDs []int, splitGroupID *int, isSplit bool) error

// === Quantity Grid ===

// SetDestinationQuantities sets product quantities for a destination (upsert).
func SetDestinationQuantities(destinationID int, quantities []models.TransferDCDestinationQty) error

// GetDestinationQuantities retrieves quantities for a single destination.
func GetDestinationQuantities(destinationID int) ([]models.TransferDCDestinationQty, error)

// GetQuantityGrid retrieves the full quantity grid for a Transfer DC.
// Returns map[destinationID]map[productID]quantity
func GetQuantityGrid(transferDCID int) (map[int]map[int]int, error)

// GetQuantitiesForDestinations retrieves quantities for specific destination IDs.
// Used during split to know how much of each product goes to selected destinations.
func GetQuantitiesForDestinations(destinationIDs []int) (map[int][]models.TransferDCDestinationQty, error)

// === Split Tracking ===

// CreateSplit creates a split record and updates destination split status.
func CreateSplit(transferDCID int, shipmentGroupID int, destinationIDs []int, createdBy int) (*models.TransferDCSplit, error)

// GetSplitsByTransferDCID retrieves all split records for a Transfer DC.
func GetSplitsByTransferDCID(transferDCID int) ([]*models.TransferDCSplit, error)

// GetSplitByShipmentGroupID retrieves a split record by child shipment group ID.
func GetSplitByShipmentGroupID(shipmentGroupID int) (*models.TransferDCSplit, error)

// DeleteSplit deletes a split record and resets destination split status.
// Returns error if child shipment group has issued DCs.
func DeleteSplit(splitID int) error

// GetNextSplitNumber returns the next sequential split number for a Transfer DC.
func GetNextSplitNumber(transferDCID int) (int, error)

// === Transfer DC Status Helpers ===

// UpdateTransferDCStatus updates the parent DC status and transfer_dcs counters.
func UpdateTransferDCStatus(dcID int, status string) error

// RecalculateSplitProgress recounts split vs total destinations and updates transfer_dcs counters.
func RecalculateSplitProgress(transferDCID int) error

// === Listing & Filtering ===

// ListTransferDCsByProject lists Transfer DCs for a project with pagination and filtering.
func ListTransferDCsByProject(projectID int, status string, page, pageSize int) ([]*models.TransferDC, int, error)

// GetTransferDCSummary returns aggregate stats for a Transfer DC (destinations, products, serials, splits).
func GetTransferDCSummary(transferDCID int) (*models.TransferDCSummary, error)
```

### 2. Update `internal/database/delivery_challans.go`
- Verify `insertDCWithLineItemsAndSerials` works with `dc_type = "transfer"`
- No changes expected (function is already generic on dc_type)

### 3. Update `internal/database/shipment_groups.go`
- Update `GetShipmentGroup` query to include `transfer_dc_id` and `split_id` columns
- Update `CreateShipmentGroup` to accept `transfer_dc_id` parameter
- Add `GetShipmentGroupsByTransferDC(transferDCID int)` function

### 4. Add summary model — `internal/models/transfer_dc.go`
```go
type TransferDCSummary struct {
    TotalDestinations  int
    SplitDestinations  int
    PendingDestinations int
    TotalProducts      int
    TotalQuantity      int
    TotalSerials       int
    SplitCount         int
}
```

---

## Acceptance Criteria

- [ ] All CRUD operations work: Create, Get, Update, Delete for transfer_dcs
- [ ] Destination management: add, list, filter by split status, update split status
- [ ] Quantity grid: set, get per-destination, get full grid
- [ ] Split tracking: create, list, delete with cascade status reset
- [ ] List/filter Transfer DCs by project and status with pagination
- [ ] Split progress tracking (num_split / num_destinations) stays in sync
- [ ] Shipment groups correctly reference parent Transfer DC when created via split
- [ ] All database functions follow existing patterns (global DB instance, error handling)
- [ ] All tests pass
- [ ] No regression in existing delivery challan or shipment group operations
