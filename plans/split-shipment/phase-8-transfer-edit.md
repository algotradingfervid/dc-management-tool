# Phase 8: Transfer DC Edit Wizard

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 4 (Transfer Wizard) — Creation wizard must exist to base edit on
- Phase 5 (Transfer Detail) — Detail page with Edit button

## Overview

Allow editing of draft Transfer DCs. The edit wizard mirrors the creation wizard (5 steps) but pre-populates all fields with existing data. This is similar to how `internal/handlers/shipment_edit.go` handles editing draft shipments.

**Rules:**
- Only `draft` Transfer DCs can be edited
- Once issued, the Transfer DC is locked (no edits)
- Edit can change: destinations, quantities, serials, transporter, template, hub, addresses
- Edit is a full reconciliation: compare old vs new data, apply changes

---

## New Files

| File | Purpose |
|------|---------|
| `internal/handlers/transfer_edit.go` | Transfer DC edit wizard handlers |
| `components/pages/transfer_dcs/edit_step1.templ` | Edit step 1 (or reuse creation templates with edit mode) |

## Modified Files

| File | Changes |
|------|---------|
| `components/pages/transfer_dcs/wizard_step1.templ` | Add prefill support (edit mode) |
| `components/pages/transfer_dcs/wizard_step2.templ` | Add prefill support |
| `components/pages/transfer_dcs/wizard_step3.templ` | Add prefill support |
| `components/pages/transfer_dcs/wizard_step4.templ` | Add prefill support |
| `components/pages/transfer_dcs/wizard_step5.templ` | Add prefill support + edit mode action URL |
| `cmd/server/main.go` | Add edit routes |

---

## Tests to Write First

- [ ] `TestShowEditTransferWizard_PrePopulated` — All fields filled from existing data
- [ ] `TestEditTransferDC_ChangeDestinations` — Add/remove destinations, quantities updated
- [ ] `TestEditTransferDC_ChangeSerials` — Replace serials, old ones freed
- [ ] `TestEditTransferDC_ChangeTransporter` — Update transporter/vehicle
- [ ] `TestEditTransferDC_OnlyDraft` — Error if Transfer DC is not draft
- [ ] `TestEditTransferDC_ReconcileQuantities` — Added destinations get quantities, removed ones are deleted
- [ ] `TestEditTransferDC_DCNumberUnchanged` — DC number stays the same after edit

---

## Implementation Steps

### 1. Create edit handlers — `internal/handlers/transfer_edit.go`

```go
// ShowEditTransferWizard loads existing Transfer DC data and renders Step 1 with prefill.
func ShowEditTransferWizard(c echo.Context) error {
    // 1. Get Transfer DC by ID from URL
    // 2. Verify status == "draft"
    // 3. Load all existing data: destinations, quantities, serials, transporter
    // 4. Build prefill struct
    // 5. Render wizard_step1 with prefill
}

// EditTransferWizardStep2-5: Same flow as creation but with prefill
func EditTransferWizardStep2(c echo.Context) error
func EditTransferWizardQuantityStep(c echo.Context) error
func EditTransferWizardStep4(c echo.Context) error
func EditTransferWizardStep5(c echo.Context) error

// SaveTransferEdit applies all changes to the draft Transfer DC.
func SaveTransferEdit(c echo.Context) error {
    // 1. Parse all form data
    // 2. Verify Transfer DC still in draft
    // 3. Begin transaction
    // 4. Reconcile destinations:
    //    - toKeep: destinations in both old & new → update quantities
    //    - toAdd: new destinations → insert
    //    - toRemove: old destinations not in new → delete
    // 5. Update line items (totals from new quantities)
    // 6. Reconcile serials:
    //    - Delete old serials
    //    - Insert new serials
    // 7. Update transfer_dcs record (transporter, hub, template, tax)
    // 8. Update delivery_challans record (addresses, date)
    // 9. Commit transaction
    // 10. Flash success, redirect to detail page
}
```

### 2. Add prefill support to creation templates

Templates should accept an optional `prefill` struct:

```go
type TransferWizardPrefill struct {
    TemplateID        int
    HubAddressID      int
    ChallanDate       string
    TransporterName   string
    VehicleNumber     string
    EwayBillNumber    string
    DocketNumber      string
    TaxType           string
    ReverseCharge     string
    Notes             string
    BillFromAddressID int
    DispatchFromAddressID int
    BillToAddressID   int
    ShipToAddressIDs  []int
    Quantities        map[int]map[int]int  // product → destination → qty
    Serials           map[int][]string     // product → serials
}
```

Use `if prefill != nil { ... }` pattern to conditionally set values.

### 3. Add back navigation for edit mode

```go
func EditTransferWizardBackToStep1(c echo.Context) error
func EditTransferWizardBackToStep2(c echo.Context) error
func EditTransferWizardBackToStep3(c echo.Context) error
func EditTransferWizardBackToStep4(c echo.Context) error
```

### 4. Register routes — `cmd/server/main.go`

```go
// Transfer DC edit wizard (under project-scoped routes)
projectRoutes.GET("/transfer-dcs/:tdcid/edit", handlers.ShowEditTransferWizard)
projectRoutes.POST("/transfer-dcs/:tdcid/edit/step2", handlers.EditTransferWizardStep2)
projectRoutes.POST("/transfer-dcs/:tdcid/edit/step3", handlers.EditTransferWizardQuantityStep)
projectRoutes.POST("/transfer-dcs/:tdcid/edit/step4", handlers.EditTransferWizardStep4)
projectRoutes.POST("/transfer-dcs/:tdcid/edit/step5", handlers.EditTransferWizardStep5)
projectRoutes.POST("/transfer-dcs/:tdcid/edit", handlers.SaveTransferEdit)
projectRoutes.POST("/transfer-dcs/:tdcid/edit/back-to-step1", handlers.EditTransferWizardBackToStep1)
projectRoutes.POST("/transfer-dcs/:tdcid/edit/back-to-step2", handlers.EditTransferWizardBackToStep2)
projectRoutes.POST("/transfer-dcs/:tdcid/edit/back-to-step3", handlers.EditTransferWizardBackToStep3)
projectRoutes.POST("/transfer-dcs/:tdcid/edit/back-to-step4", handlers.EditTransferWizardBackToStep4)
```

---

## Reconciliation Logic

### Destination Reconciliation
```
oldDests = set of ship_to_address_ids in existing Transfer DC
newDests = set of ship_to_address_ids from edit form

toKeep = oldDests ∩ newDests  → update quantities
toAdd  = newDests − oldDests  → insert new destinations + quantities
toDelete = oldDests − newDests → delete destinations + quantities
```

### Serial Reconciliation
Since serials are bulk (not per-destination), the simplest approach is:
1. Delete all existing serial_numbers for the Transfer DC's line items
2. Delete all existing dc_line_items for the Transfer DC
3. Re-insert line items with new totals
4. Re-insert serials

This avoids complex diff logic and is safe since draft DCs have no dependencies.

---

## Acceptance Criteria

- [ ] Edit wizard shows pre-populated data for all 5 steps
- [ ] Can change destinations (add/remove), quantities, serials, transporter, hub
- [ ] Save correctly reconciles: old destinations removed, new ones added, quantities updated
- [ ] Line items and serials fully re-created from new data
- [ ] DC number does NOT change after edit
- [ ] Only draft Transfer DCs can be edited
- [ ] Back navigation preserves edit data
- [ ] Flash messages on success/failure
- [ ] All routes registered
- [ ] All tests pass
