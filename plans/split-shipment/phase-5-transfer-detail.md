# Phase 5: Transfer DC Detail Page & Lifecycle

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 1 (Database Schema)
- Phase 2 (DC Numbering)
- Phase 3 (Data Access Layer)
- Phase 4 (Transfer Wizard) — Transfer DCs must exist to view them

## Overview

Build the Transfer DC detail page — the central hub for viewing a Transfer DC's full information and managing its lifecycle. This page shows:

1. **Header**: DC number, status badge, challan date, actions
2. **Transfer Info**: Hub location, transporter/vehicle, template, tax config
3. **Addresses**: Bill-from, dispatch-from, bill-to
4. **Line Items**: All products with total quantities, rates, amounts
5. **Serial Numbers**: All bulk serials grouped by product
6. **Destinations**: Full list of all ~25 destinations with per-destination quantities
7. **Split Progress Panel**: Which destinations are split vs pending, links to child shipment groups
8. **Actions**: Issue, Create Split, Edit (draft only), Delete (draft only)

Also implements the lifecycle transitions: `draft → issued → splitting → split`.

---

## New Files

| File | Purpose |
|------|---------|
| `internal/handlers/transfer_dc.go` | Transfer DC detail, lifecycle, and list handlers |
| `components/pages/transfer_dcs/detail.templ` | Full Transfer DC detail page |
| `components/pages/transfer_dcs/list.templ` | Transfer DC list page |
| `components/pages/transfer_dcs/print.templ` | Browser print view |

## Modified Files

| File | Changes |
|------|---------|
| `internal/handlers/transit_dc.go` | Update `ShowDCDetail()` dispatcher to handle transfer type |
| `cmd/server/main.go` | Add Transfer DC detail/list/lifecycle routes |

---

## Tests to Write First

### Lifecycle
- [ ] `TestIssueTransferDC_HappyPath` — Draft → Issued
- [ ] `TestIssueTransferDC_NotDraft` — Error if already issued/splitting/split
- [ ] `TestTransferDCStatusTransition_IssuedToSplitting` — Automatic on first split
- [ ] `TestTransferDCStatusTransition_SplittingToSplit` — Automatic when all destinations split
- [ ] `TestDeleteTransferDC_DraftOnly` — Can delete draft, error if issued

### Detail Page
- [ ] `TestShowTransferDCDetail_RendersAllSections` — All data shown
- [ ] `TestShowTransferDCDetail_SplitProgress` — Shows N/M destinations split
- [ ] `TestShowTransferDCDetail_ActionButtons` — Issue shown for draft, Split shown for issued/splitting

### List Page
- [ ] `TestListTransferDCs_Pagination` — Paginated list
- [ ] `TestListTransferDCs_StatusFilter` — Filter by draft/issued/splitting/split

---

## Implementation Steps

### 1. Create handler — `internal/handlers/transfer_dc.go`

```go
// ShowTransferDCDetail renders the full Transfer DC detail page.
func ShowTransferDCDetail(c echo.Context) error {
    // 1. Get DC by ID from URL param
    // 2. Get TransferDC record (hub, transporter, template)
    // 3. Get line items with serials
    // 4. Get all destinations with quantities
    // 5. Get split records with child shipment groups
    // 6. Calculate split progress
    // 7. Build page with sidebar/topbar layout
    // 8. Render transfer_dcs.Detail(...)
}

// IssueTransferDC transitions status from draft → issued.
func IssueTransferDC(c echo.Context) error {
    // 1. Get DC, verify status == "draft"
    // 2. Update status to "issued"
    // 3. Set issued_at, issued_by
    // 4. Flash success message
    // 5. Redirect to detail page
}

// DeleteTransferDC deletes a draft Transfer DC and all related data.
func DeleteTransferDC(c echo.Context) error {
    // 1. Get DC, verify status == "draft"
    // 2. Verify no child shipment groups exist
    // 3. Delete Transfer DC (cascades to destinations, quantities)
    // 4. Delete parent delivery_challans record
    // 5. Flash success message
    // 6. Redirect to Transfer DC list
}

// ListTransferDCs shows all Transfer DCs for the current project.
func ListTransferDCs(c echo.Context) error {
    // 1. Parse filters: status, page, search
    // 2. Query Transfer DCs with pagination
    // 3. Render transfer_dcs.List(...)
}

// ShowTransferDCPrintView renders the browser print view.
func ShowTransferDCPrintView(c echo.Context) error
```

### 2. Update DC detail dispatcher — `internal/handlers/transit_dc.go`

```go
// In ShowDCDetail() function (line ~25):
func ShowDCDetail(c echo.Context) error {
    dc, err := database.GetDeliveryChallanByID(dcID)
    // ...
    switch dc.DCType {
    case "official":
        return ShowOfficialDCDetail(c)
    case "transfer":
        return ShowTransferDCDetail(c)  // NEW
    default:
        return showTransitDCDetail(c)
    }
}
```

### 3. Create detail templ component — `components/pages/transfer_dcs/detail.templ`

```
Layout:
┌─────────────────────────────────────────────────────┐
│ Header: STDC Number | Status Badge | Actions        │
│ (Issue / Edit / Delete / Print / Export buttons)     │
├─────────────────────────────────────────────────────┤
│ Transfer Info Card                                   │
│ ┌──────────────┬────────────────┐                   │
│ │ Hub Location  │ Template       │                   │
│ │ Challan Date  │ Tax Type       │                   │
│ │ Transporter   │ Reverse Charge │                   │
│ │ Vehicle       │ E-Way Bill     │                   │
│ └──────────────┴────────────────┘                   │
├─────────────────────────────────────────────────────┤
│ Addresses Row                                        │
│ ┌─────────┬──────────┬─────────┐                    │
│ │Bill From│Dispatch  │ Bill To │                    │
│ │         │  From    │         │                    │
│ └─────────┴──────────┴─────────┘                    │
├─────────────────────────────────────────────────────┤
│ Line Items Table                                     │
│ Product | HSN | Qty | Rate | Tax% | Tax | Total     │
│ ──────────────────────────────────────────────────   │
│ Product A | 8542 | 100 | ₹500 | 18% | ₹9000 | ... │
│ Product B | ...                                      │
│ ──────────────────────────────────────────────────   │
│ Totals: Taxable | CGST | SGST | Grand Total         │
├─────────────────────────────────────────────────────┤
│ Serial Numbers (collapsible per product)             │
│ ▼ Product A (100 serials)                            │
│   SN001, SN002, SN003, ...                          │
│ ▼ Product B (50 serials)                             │
│   SN101, SN102, ...                                  │
├─────────────────────────────────────────────────────┤
│ Destinations & Quantities                            │
│ ┌───────────────────────────────────────┐           │
│ │ # │ Destination │ Prod A │ Prod B │ Status │      │
│ │ 1 │ Location X  │ 4      │ 2      │ ✓ Split│      │
│ │ 2 │ Location Y  │ 4      │ 2      │ ✓ Split│      │
│ │ 3 │ Location Z  │ 4      │ 2      │ Pending│      │
│ │ ...                                           │    │
│ └───────────────────────────────────────┘           │
├─────────────────────────────────────────────────────┤
│ Split Progress                                       │
│ ┌───────────────────────────────────────┐           │
│ │ Progress: 10/25 destinations split    │ ████░░░   │
│ │                                        │           │
│ │ Split #1 (5 destinations) → Group #42  │ View →   │
│ │   Vehicle: TS09-1234 | Transporter: X  │           │
│ │   Locations: A, B, C, D, E             │           │
│ │                                        │           │
│ │ Split #2 (5 destinations) → Group #43  │ View →   │
│ │   Vehicle: TS09-5678 | Transporter: Y  │           │
│ │   Locations: F, G, H, I, J             │           │
│ │                                        │           │
│ │ [+ Create New Split] (15 remaining)    │           │
│ └───────────────────────────────────────┘           │
└─────────────────────────────────────────────────────┘
```

### 4. Create list templ component — `components/pages/transfer_dcs/list.templ`

- Table columns: DC Number, Status, Hub Location, Date, Destinations (N split / M total), Actions
- Status filter: All, Draft, Issued, Splitting, Split
- Search by DC number
- Pagination
- Row click → detail page

### 5. Create print view — `components/pages/transfer_dcs/print.templ`

- Similar to Transit DC print view but includes:
  - Hub location prominently
  - All destination addresses listed
  - Per-destination quantity breakdown table
  - All serial numbers

### 6. Register routes — `cmd/server/main.go`

```go
// Transfer DC detail & lifecycle (under project-scoped routes)
projectRoutes.GET("/transfer-dcs", handlers.ListTransferDCs)
projectRoutes.GET("/transfer-dcs/:tdcid", handlers.ShowTransferDCDetail)
projectRoutes.POST("/transfer-dcs/:tdcid/issue", handlers.IssueTransferDC)
projectRoutes.DELETE("/transfer-dcs/:tdcid", handlers.DeleteTransferDC)
projectRoutes.GET("/transfer-dcs/:tdcid/print", handlers.ShowTransferDCPrintView)
```

### 7. Lifecycle state machine logic

```
draft → issued:
  - Manual action (Issue button)
  - Sets issued_at, issued_by
  - Transfer DC is now locked (no edits)
  - Split operations become available

issued → splitting:
  - Automatic on first split creation
  - At least 1 destination is now split, but not all

splitting → split:
  - Automatic when ALL destinations are assigned to split groups
  - num_split == num_destinations

splitting → issued:
  - Automatic if all splits are undone (all child groups deleted)
  - num_split returns to 0
```

---

## Acceptance Criteria

- [ ] Transfer DC detail page renders with all 7 sections
- [ ] Status badges show correct colors: draft (gray), issued (blue), splitting (orange), split (green)
- [ ] Issue button only appears for draft Transfer DCs
- [ ] Create Split button only appears for issued/splitting Transfer DCs
- [ ] Edit/Delete only available for draft Transfer DCs
- [ ] Lifecycle transitions work: draft→issued, issued→splitting (automatic), splitting→split (automatic)
- [ ] Split progress bar shows accurate N/M count
- [ ] Each split record links to its child shipment group
- [ ] Destinations table shows per-product quantities with split status
- [ ] List page with pagination, filtering, and search works
- [ ] Print view renders correctly in browser
- [ ] DC detail dispatcher correctly routes to Transfer DC detail
- [ ] All routes registered and accessible
- [ ] `task templ:gen` runs clean
- [ ] All tests pass
