# Phase 7: Split Wizard (Handler + Templates)

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 5 (Transfer Detail) — Split is initiated from the Transfer DC detail page
- Phase 6 (Split Service) — Core split logic must be implemented

## Overview

Build the 4-step split wizard UI that allows users to split a Transfer DC's remaining destinations into a new vehicle group. This wizard is launched from the Transfer DC detail page's "Create Split" button.

**Wizard Steps:**
1. **Select Destinations**: Choose which un-split destinations go to this vehicle
2. **Transporter/Vehicle**: Enter small vehicle details
3. **Serial Numbers**: Enter/scan serials per product for this vehicle (validated against parent)
4. **Review & Confirm**: Review all data, submit to create child shipment group

---

## New Files

| File | Purpose |
|------|---------|
| `internal/handlers/split_wizard.go` | Split wizard handler functions |
| `components/pages/transfer_dcs/split_step1.templ` | Step 1: Select destinations |
| `components/pages/transfer_dcs/split_step2.templ` | Step 2: Transporter/vehicle |
| `components/pages/transfer_dcs/split_step3.templ` | Step 3: Serial numbers |
| `components/pages/transfer_dcs/split_step4.templ` | Step 4: Review & confirm |
| `components/pages/transfer_dcs/split_steps.templ` | Split wizard progress indicator |

## Modified Files

| File | Changes |
|------|---------|
| `cmd/server/main.go` | Add split wizard routes |

---

## Tests to Write First

### Handler Layer
- [ ] `TestShowSplitWizardStep1` — Renders with unsplit destinations, product quantities
- [ ] `TestShowSplitWizardStep1_TransferDCMustBeIssuedOrSplitting` — Error for draft/split
- [ ] `TestSplitWizardStep2_ValidatesDestinations` — At least 1 destination required
- [ ] `TestSplitWizardStep3_ValidatesTransporter` — Transporter name required
- [ ] `TestSplitWizardStep4_ValidatesSerials` — Serial validation before review
- [ ] `TestCreateSplitShipment_Handler` — Full form submission creates split
- [ ] `TestSplitWizardBackNavigation` — Back buttons preserve data

---

## Implementation Steps

### 1. Create handlers — `internal/handlers/split_wizard.go`

```go
// Step 1: Show destination selection
func ShowSplitWizardStep1(c echo.Context) error {
    // 1. Get Transfer DC by ID from URL (:tdcid)
    // 2. Verify status is "issued" or "splitting"
    // 3. Get un-split destinations with quantities
    // 4. Get product list from template
    // 5. Render split_step1 with:
    //    - Transfer DC info (DC number, hub, date)
    //    - Un-split destinations with per-product quantities
    //    - Checkbox per destination
}

// Process Step 1 → Show Step 2
func SplitWizardStep2(c echo.Context) error {
    // 1. Parse selected destination IDs (checkbox values)
    // 2. Validate: at least 1 selected, all must be un-split
    // 3. Calculate total quantities per product for selected destinations
    // 4. Load transporters for project
    // 5. Render split_step2 with:
    //    - Selected destinations summary
    //    - Product quantities for this split
    //    - Transporter/vehicle form fields
}

// Process Step 2 → Show Step 3 (serials)
func SplitWizardStep3(c echo.Context) error {
    // 1. Parse transporter, vehicle, eway bill, docket, notes
    // 2. Validate: transporter name required
    // 3. Get available serials per product (parent serials − already-used serials)
    // 4. Calculate expected serial counts per product
    // 5. Render split_step3 with:
    //    - One textarea per product
    //    - Expected count per product
    //    - Available serials as reference/hint
    //    - Per-destination serial assignment fields
}

// Process Step 3 → Show Step 4 (review)
func SplitWizardStep4(c echo.Context) error {
    // 1. Parse serials per product + per-destination assignments
    // 2. Validate:
    //    a) All serials exist in parent Transfer DC
    //    b) No serial already used in another split
    //    c) Per-product count matches expected
    //    d) Per-destination assignments match destination quantities
    //    e) No duplicates
    // 3. Render split_step4 with:
    //    - Full review: destinations, transporter, products, serials
    //    - Hidden form with all data
    //    - Confirm button
}

// Process Step 4 → Create split
func CreateSplitShipmentHandler(c echo.Context) error {
    // 1. Parse all form data from hidden fields
    // 2. Re-validate everything
    // 3. Call services.CreateSplitShipment(params)
    // 4. Flash success message
    // 5. Redirect to Transfer DC detail page (shows updated split progress)
}
```

### 2. Create templ components

#### `split_steps.templ` — Progress indicator
```
4 steps: Destinations → Vehicle → Serials → Review
```

#### `split_step1.templ` — Select Destinations

```
┌─────────────────────────────────────────────────────┐
│ Split Transfer DC: PRJ-STDC-2526-001                │
│ Hub: District Warehouse, Hyderabad                   │
│ Split Progress ─ ────────────────────── (2 steps)    │
├─────────────────────────────────────────────────────┤
│ Select destinations for this vehicle:                │
│                                                      │
│ ☐ Select All (15 remaining)                          │
│                                                      │
│ ┌─────────────────────────────────────────────┐     │
│ │ ☑ │ Location Name    │ Prod A │ Prod B │     │     │
│ │ ☑ │ Mandal X, Dist Y │ 4      │ 2      │     │     │
│ │ ☐ │ Mandal Z, Dist W │ 4      │ 2      │     │     │
│ │ ☑ │ Mandal P, Dist Q │ 6      │ 3      │     │     │
│ │ ...                                      │     │
│ └─────────────────────────────────────────────┘     │
│                                                      │
│ Selected: 3 destinations                             │
│ Total Qty: Product A = 14, Product B = 7            │
│                                                      │
│                              [Next: Vehicle Details] │
└─────────────────────────────────────────────────────┘
```

JavaScript features:
- Select All checkbox toggles all
- Running count of selected destinations
- Running total quantities per product (updates as checkboxes toggle)

#### `split_step2.templ` — Transporter/Vehicle

```
┌─────────────────────────────────────────────────────┐
│ Selected: 3 destinations (14 × Prod A, 7 × Prod B) │
├─────────────────────────────────────────────────────┤
│ Transporter:  [Dropdown: project transporters    ]  │
│ Vehicle:      [Dropdown: vehicles for transporter]  │
│ E-Way Bill:   [____________]                        │
│ Docket No:    [____________]                        │
│ Notes:        [________________________]            │
│                                                      │
│ [← Back]                    [Next: Serial Numbers]  │
└─────────────────────────────────────────────────────┘
```

Reuse transporter/vehicle dropdown pattern from shipment wizard step 1.

#### `split_step3.templ` — Serial Numbers

```
┌─────────────────────────────────────────────────────┐
│ Enter serial numbers for this vehicle split          │
├─────────────────────────────────────────────────────┤
│ Product A (Expected: 14 serials)                     │
│ Available: SN001-SN100 (86 remaining from parent)   │
│ ┌──────────────────────────────────────┐            │
│ │ Enter serials (one per line):        │            │
│ │ SN001                                │            │
│ │ SN002                                │            │
│ │ ...                                  │ 14/14 ✓   │
│ └──────────────────────────────────────┘            │
│                                                      │
│ Per-destination assignment:                          │
│ ┌──────────────────────────────┐                    │
│ │ Mandal X (expects 4):        │                    │
│ │ SN001, SN002, SN003, SN004  │ 4/4 ✓             │
│ │                              │                    │
│ │ Mandal P (expects 6):        │                    │
│ │ SN005-SN010                  │ 6/6 ✓             │
│ │                              │                    │
│ │ Mandal Z (expects 4):        │                    │
│ │ SN011-SN014                  │ 4/4 ✓             │
│ └──────────────────────────────┘                    │
│                                                      │
│ Product B (Expected: 7 serials)                      │
│ [Similar layout]                                     │
│                                                      │
│ [← Back]                           [Next: Review]   │
└─────────────────────────────────────────────────────┘
```

JavaScript features:
- Real-time serial count feedback (green/red/gray)
- Available serials display (from parent minus already-used)
- Per-destination assignment textareas
- Validation before proceeding

#### `split_step4.templ` — Review & Confirm

```
┌─────────────────────────────────────────────────────┐
│ Review Split #3 for PRJ-STDC-2526-001               │
├─────────────────────────────────────────────────────┤
│ Vehicle: TS09-1234 (Transporter: XYZ Logistics)     │
│ E-Way Bill: EWB123456 | Docket: DK789              │
├─────────────────────────────────────────────────────┤
│ Destinations (3):                                    │
│ • Mandal X, District Y                              │
│ • Mandal Z, District W                              │
│ • Mandal P, District Q                              │
├─────────────────────────────────────────────────────┤
│ Products:                                            │
│ Product A: 14 units (₹500/unit, 18% GST)           │
│ Product B: 7 units (₹300/unit, 18% GST)            │
├─────────────────────────────────────────────────────┤
│ Serial Numbers: 21 total                             │
│ Product A: SN001-SN014                              │
│ Product B: SN101-SN107                              │
├─────────────────────────────────────────────────────┤
│ This will create:                                    │
│ • 1 Transit DC (TDC) for vehicle TS09-1234          │
│ • 3 Official DCs (ODC) — one per destination        │
│                                                      │
│ [← Back]                      [Confirm & Create]    │
└─────────────────────────────────────────────────────┘
```

### 3. Add back navigation handlers

```go
func SplitWizardBackToStep1(c echo.Context) error
func SplitWizardBackToStep2(c echo.Context) error
func SplitWizardBackToStep3(c echo.Context) error
```

### 4. Register routes — `cmd/server/main.go`

```go
// Split wizard (under project-scoped routes)
projectRoutes.GET("/transfer-dcs/:tdcid/split", handlers.ShowSplitWizardStep1)
projectRoutes.POST("/transfer-dcs/:tdcid/split/step2", handlers.SplitWizardStep2)
projectRoutes.POST("/transfer-dcs/:tdcid/split/step3", handlers.SplitWizardStep3)
projectRoutes.POST("/transfer-dcs/:tdcid/split/step4", handlers.SplitWizardStep4)
projectRoutes.POST("/transfer-dcs/:tdcid/split", handlers.CreateSplitShipmentHandler)
projectRoutes.POST("/transfer-dcs/:tdcid/split/back-to-step1", handlers.SplitWizardBackToStep1)
projectRoutes.POST("/transfer-dcs/:tdcid/split/back-to-step2", handlers.SplitWizardBackToStep2)
projectRoutes.POST("/transfer-dcs/:tdcid/split/back-to-step3", handlers.SplitWizardBackToStep3)
```

### 5. Generate templ code — `task templ:gen`

---

## Key UX Decisions

1. **Available serials hint**: Step 3 shows which serials from the parent are still available (not used in prior splits). This helps the user select the right serials.

2. **Per-destination serial assignment**: Unlike the Transfer DC creation (bulk serials only), the split wizard DOES require per-destination serial assignment. This is because the child shipment group needs to know which serials go to which Official DC.

3. **Quantity pre-filled**: The quantity per product per destination is pre-determined by the Transfer DC's quantity grid. The user cannot change quantities during split — they must match exactly.

4. **Transporter dropdown**: Reuses the project's existing transporter list with vehicle auto-population, same as the shipment wizard.

---

## Acceptance Criteria

- [ ] Split wizard accessible from Transfer DC detail page (status must be issued/splitting)
- [ ] Step 1: Shows un-split destinations with quantities, checkbox selection, running totals
- [ ] Step 2: Transporter/vehicle form with dropdown auto-population
- [ ] Step 3: Serial entry with available serial hints, per-destination assignment, count validation
- [ ] Step 4: Complete review with summary of what will be created
- [ ] Form submission creates child shipment group (1 TDC + N ODCs)
- [ ] Transfer DC split progress updates after successful split
- [ ] Back navigation preserves data across all steps
- [ ] Validation errors shown inline at each step
- [ ] Redirect to Transfer DC detail page after successful split
- [ ] Flash messages on success/failure
- [ ] All routes registered
- [ ] `task templ:gen` runs clean
- [ ] All tests pass
