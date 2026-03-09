# Phase 4: Transfer DC Creation Wizard (Handler + Templates)

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 1 (Database Schema)
- Phase 2 (DC Numbering)
- Phase 3 (Data Access Layer)

## Overview

Build the 5-step Transfer DC creation wizard — analogous to the existing shipment creation wizard but with key differences:

1. **Step 1**: Template + Hub location + Transporter/Vehicle (large truck) + Tax config
2. **Step 2**: Select all ~25 final ship-to destinations + bill-to + bill-from + dispatch-from
3. **Step 3**: Quantity grid (product × destination)
4. **Step 4**: Bulk serial entry (all serials per product, NOT per-destination)
5. **Step 5**: Review & submit

Key difference from shipment wizard: serials are bulk (not assigned to destinations). Destination-level serial assignment happens during the split operation.

---

## New Files

| File | Purpose |
|------|---------|
| `internal/handlers/transfer_wizard.go` | All Transfer DC wizard handler functions |
| `components/pages/transfer_dcs/wizard_step1.templ` | Step 1: Config + transporter |
| `components/pages/transfer_dcs/wizard_step2.templ` | Step 2: Address selection |
| `components/pages/transfer_dcs/wizard_step3.templ` | Step 3: Quantity grid |
| `components/pages/transfer_dcs/wizard_step4.templ` | Step 4: Bulk serial entry |
| `components/pages/transfer_dcs/wizard_step5.templ` | Step 5: Review & submit |
| `components/pages/transfer_dcs/wizard_steps.templ` | Wizard progress indicator (5 steps) |

## Modified Files

| File | Changes |
|------|---------|
| `cmd/server/main.go` | Add Transfer DC wizard routes |
| `internal/services/dc_generation.go` | Add `CreateTransferDC()` service function |

---

## Tests to Write First

### Service Layer
- [ ] `TestCreateTransferDC_HappyPath` — Create with template, 5 destinations, quantities, serials
- [ ] `TestCreateTransferDC_ValidationErrors` — Missing required fields
- [ ] `TestCreateTransferDC_DCNumberGenerated` — Verify STDC number format
- [ ] `TestCreateTransferDC_BulkSerials` — Serials stored at DC level (not per-destination)
- [ ] `TestCreateTransferDC_QuantityGrid` — Verify per-destination quantities stored correctly
- [ ] `TestCreateTransferDC_ProjectWideSerialUniqueness` — No duplicate serials across project

### Handler Layer
- [ ] `TestShowTransferWizardStep1` — Renders with templates, transporters
- [ ] `TestTransferWizardStep2_ValidatesStep1` — Step 1 validation before showing step 2
- [ ] `TestTransferWizardStep3_ValidatesStep2` — Address validation before quantity grid
- [ ] `TestTransferWizardStep4_ValidatesQuantities` — Quantity validation before serial entry
- [ ] `TestTransferWizardStep5_ValidatesSerials` — Serial validation before review
- [ ] `TestCreateTransferDC_Handler` — Full form submission creates Transfer DC

---

## Implementation Steps

### 1. Define TransferDCParams — `internal/services/dc_generation.go`

```go
type TransferDCParams struct {
    ProjectID           int
    TemplateID          int
    HubAddressID        int      // The hub/transit location (existing ship-to address)
    BillFromAddressID   int
    DispatchFromAddressID int
    BillToAddressID     int
    ShipToAddressIDs    []int    // All ~25 final destinations
    ChallanDate         string
    TaxType             string   // cgst_sgst or igst
    ReverseCharge       string   // Y or N
    TransporterName     string   // Large truck transporter
    VehicleNumber       string
    EwayBillNumber      string
    DocketNumber        string
    Notes               string
    LineItems           []TransferDCLineItem
    CreatedBy           int
}

type TransferDCLineItem struct {
    ProductID       int
    QtyByDestination map[int]int  // map[shipToAddressID] → qty
    Rate            float64
    TaxPercentage   float64
    AllSerials      []string     // Bulk serials (NOT per-destination)
}
```

### 2. Implement CreateTransferDC service — `internal/services/dc_generation.go`

```go
func CreateTransferDC(params TransferDCParams) (int, error) {
    // 1. Begin transaction
    // 2. Generate STDC number: services.GenerateDCNumber(projectID, "transfer")
    // 3. Insert delivery_challans record (dc_type="transfer", status="draft")
    //    - ship_to_address_id = params.HubAddressID (the hub is the DC's ship-to)
    //    - bill_to_address_id, bill_from_address_id, dispatch_from_address_id from params
    // 4. Insert dc_line_items with TOTAL quantities (sum across all destinations)
    //    - rate, tax calculations same as transit DC
    // 5. Insert serial_numbers (all serials linked to line items, like transit DC)
    // 6. Insert transfer_dcs record (hub, template, transporter, tax config)
    // 7. Insert transfer_dc_destinations (one per shipToAddressID)
    // 8. Insert transfer_dc_destination_quantities (per destination × product)
    // 9. Update num_destinations counter on transfer_dcs
    // 10. Commit transaction
    // 11. Return transfer_dcs.id
}
```

### 3. Create handler functions — `internal/handlers/transfer_wizard.go`

```go
// Step 1: Show wizard start page
func ShowCreateTransferWizard(c echo.Context) error
    // Load: templates, transporters with vehicles, addresses (for hub selection)
    // Render: transfer_dcs.WizardStep1(...)

// Process Step 1 → Show Step 2
func TransferWizardStep2(c echo.Context) error
    // Parse: templateID, hubAddressID, transporterName, vehicleNumber, etc.
    // Validate: required fields
    // Load: bill-from, dispatch-from, bill-to, ship-to addresses
    // Render: transfer_dcs.WizardStep2(...)

// Process Step 2 → Show Step 3 (quantity grid)
func TransferWizardQuantityStep(c echo.Context) error
    // Parse: all address selections
    // Validate: at least 1 ship-to, hub != any ship-to
    // Load: template products
    // Render: transfer_dcs.WizardStep3(...) (quantity grid: products × destinations)

// Process Step 3 → Show Step 4 (bulk serials)
func TransferWizardStep4(c echo.Context) error
    // Parse: quantity grid (qty_{productID}_{shipToAddrID})
    // Validate: non-negative, each product has >0 total, grand total > 0
    // Render: transfer_dcs.WizardStep4(...) (serial entry: one textarea per product)

// Process Step 4 → Show Step 5 (review)
func TransferWizardStep5(c echo.Context) error
    // Parse: bulk serials per product
    // Validate: serial count == total quantity per product, no duplicates, project-wide uniqueness
    // Render: transfer_dcs.WizardStep5(...) (review page)

// Process Step 5 → Create Transfer DC
func CreateTransferDC(c echo.Context) error
    // Re-validate all data
    // Call services.CreateTransferDC(params)
    // Redirect to Transfer DC detail page
```

### 4. Create templ components — `components/pages/transfer_dcs/`

**wizard_steps.templ**: Progress indicator showing 5 steps:
1. Configuration
2. Addresses
3. Quantities
4. Serial Numbers
5. Review

**wizard_step1.templ**:
- Reuse pattern from `components/pages/shipments/wizard_step1.templ`
- Add "Hub Location" dropdown (ship-to addresses) — this is the destination for the large truck
- Template, transporter/vehicle, tax config fields (same as shipment step 1)
- Remove "Number of Locations" field (determined by step 2 selection)

**wizard_step2.templ**:
- Reuse pattern from `components/pages/shipments/wizard_step2.templ`
- Bill-from, dispatch-from, bill-to dropdowns
- Ship-to address tag picker (multi-select, no limit on count)
- NO transit ship-to selector (the hub IS the transfer DC's ship-to)

**wizard_step3.templ**:
- Quantity grid: rows = products (from template), columns = destinations
- Each cell: number input `qty_{productID}_{shipToAddrID}`
- Row totals, column totals, grand total
- Responsive: horizontal scroll for many destinations

**wizard_step4.templ**:
- One textarea per product for bulk serial entry
- Show expected count (total across all destinations)
- Real-time count feedback (green/red/gray)
- NO per-destination assignment (that happens at split time)

**wizard_step5.templ**:
- Summary of all data: config, addresses, quantities, serials
- Products table with totals
- All destinations listed
- Hidden form with all data
- Submit button: "Create Transfer DC"

### 5. Add back navigation handlers

```go
func TransferWizardBackToStep1(c echo.Context) error
func TransferWizardBackToStep2(c echo.Context) error
func TransferWizardBackToStep3(c echo.Context) error
func TransferWizardBackToStep4(c echo.Context) error
```

### 6. Register routes — `cmd/server/main.go`

```go
// Transfer DC wizard (under project-scoped routes)
projectRoutes.GET("/transfer-dcs/new", handlers.ShowCreateTransferWizard)
projectRoutes.POST("/transfer-dcs/new/step2", handlers.TransferWizardStep2)
projectRoutes.POST("/transfer-dcs/new/step3", handlers.TransferWizardQuantityStep)
projectRoutes.POST("/transfer-dcs/new/step4", handlers.TransferWizardStep4)
projectRoutes.POST("/transfer-dcs/new/step5", handlers.TransferWizardStep5)
projectRoutes.POST("/transfer-dcs", handlers.CreateTransferDC)
projectRoutes.POST("/transfer-dcs/new/back-to-step1", handlers.TransferWizardBackToStep1)
projectRoutes.POST("/transfer-dcs/new/back-to-step2", handlers.TransferWizardBackToStep2)
projectRoutes.POST("/transfer-dcs/new/back-to-step3", handlers.TransferWizardBackToStep3)
projectRoutes.POST("/transfer-dcs/new/back-to-step4", handlers.TransferWizardBackToStep4)
```

### 7. Generate templ code — `task templ:gen`

---

## Key Design Decisions

1. **Hub as ship-to**: The Transfer DC's `ship_to_address_id` = hub address. This is where the large truck goes. The final destinations are stored in `transfer_dc_destinations`.

2. **Serials as bulk**: Unlike the shipment wizard (which assigns serials per-destination), the transfer wizard stores ALL serials at the DC line-item level. Per-destination serial assignment happens during the split operation.

3. **Quantity grid stores per-destination breakdown**: Even though serials are bulk, quantities are per-destination. This is needed so the split operation knows how much of each product goes to each destination.

4. **Line items store TOTALS**: The `dc_line_items` table stores the total quantity per product (sum across all destinations). The per-destination breakdown is in `transfer_dc_destination_quantities`.

---

## Acceptance Criteria

- [ ] Transfer DC creation wizard renders all 5 steps correctly
- [ ] Step 1: Template, hub location, transporter, vehicle, tax config all captured
- [ ] Step 2: Multi-select ship-to addresses with tag picker UI
- [ ] Step 3: Quantity grid with per-destination inputs and running totals
- [ ] Step 4: Bulk serial entry with count validation
- [ ] Step 5: Complete review of all data with submit
- [ ] Back navigation works between all steps (preserves data)
- [ ] Form submission creates Transfer DC with status="draft"
- [ ] DC number generated in STDC format
- [ ] Line items, serials, destinations, and quantities all stored correctly
- [ ] Validation errors shown inline for each step
- [ ] Flash messages on success/failure
- [ ] All routes registered and working
- [ ] `task templ:gen` runs clean
- [ ] All tests pass
