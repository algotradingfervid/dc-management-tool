# Phase 11: Reports & DC Listing Integration

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 5 (Transfer Detail) — Transfer DCs must be viewable
- Phase 7 (Split Wizard) — Child groups must exist for split progress reporting

## Overview

Integrate Transfer DCs into the existing reporting and DC listing infrastructure:

1. **DC Listing Page**: Add "Transfer" as a DC type filter option
2. **DC Summary Report**: Add Transfer DC counts (draft, issued, splitting, split)
3. **New Transfer DC Report**: Dedicated report showing Transfer DC → split hierarchy with progress
4. **Existing Reports**: Ensure destination, product, and serial reports include Transfer DC data

---

## Modified Files

| File | Changes |
|------|---------|
| `internal/handlers/dc_listing.go` | Add "transfer" to valid dc_type filters |
| `internal/handlers/reports.go` | Add Transfer DC section in summary, new transfer report handler |
| `internal/database/reports.go` | Add Transfer DC report queries |
| `internal/database/dc_listing.go` | Ensure "transfer" type works in filter |
| `components/pages/delivery_challans/list.templ` | Add "Transfer" option in type filter dropdown |
| `components/pages/reports/index.templ` | Add Transfer DC report link |
| `components/pages/reports/summary.templ` | Add Transfer DC stats section |
| `components/pages/reports/transfer.templ` | NEW: Transfer DC report page |
| `cmd/server/main.go` | Add transfer report route |

---

## Tests to Write First

### DC Listing
- [ ] `TestDCListFilter_TransferType` — Filter by dc_type="transfer" returns only Transfer DCs
- [ ] `TestDCListFilter_AllTypes` — "all" filter includes Transfer DCs
- [ ] `TestDCList_TransferDCDisplay` — Transfer DCs show correct status badges (including splitting/split)

### Reports
- [ ] `TestDCSummaryReport_IncludesTransferDCs` — Transfer DC counts in summary
- [ ] `TestDCSummaryReport_TransferStatuses` — Shows draft/issued/splitting/split breakdown
- [ ] `TestTransferDCReport_HappyPath` — Lists Transfer DCs with split progress
- [ ] `TestTransferDCReport_DateFilter` — Filter by date range
- [ ] `TestDestinationReport_IncludesTransferDCDestinations` — Destinations from Transfer DCs appear
- [ ] `TestProductReport_IncludesTransferDCProducts` — Products from Transfer DCs counted
- [ ] `TestSerialReport_IncludesTransferDCSerials` — Serials from Transfer DCs searchable

---

## Implementation Steps

### 1. Update DC Listing — `components/pages/delivery_challans/list.templ`

Add Transfer option to the DC type filter dropdown:
```templ
<select name="type" ...>
    <option value="all">All Types</option>
    <option value="transit">Transit DC</option>
    <option value="official">Official DC</option>
    <option value="transfer">Transfer DC</option>  <!-- NEW -->
</select>
```

Add status filter options for new statuses:
```templ
<select name="status" ...>
    <option value="all">All Statuses</option>
    <option value="draft">Draft</option>
    <option value="issued">Issued</option>
    <option value="splitting">Splitting</option>  <!-- NEW -->
    <option value="split">Split</option>           <!-- NEW -->
</select>
```

Update DC type badge rendering:
```templ
if dc.DCType == "transfer" {
    <span class="badge badge-purple">Transfer</span>
}
```

Update status badge rendering:
```templ
if dc.Status == "splitting" {
    <span class="badge badge-orange">Splitting</span>
}
if dc.Status == "split" {
    <span class="badge badge-green">Split</span>
}
```

### 2. Update DC Summary Report — `internal/handlers/reports.go`

Add Transfer DC stats alongside existing Transit/Official counts:

```go
type DCSummaryStats struct {
    // Existing
    TransitDrafts   int
    TransitIssued   int
    OfficialDrafts  int
    OfficialIssued  int
    // NEW
    TransferDrafts    int
    TransferIssued    int
    TransferSplitting int
    TransferSplit     int
    TransferTotal     int
}
```

### 3. Add Transfer DC Report queries — `internal/database/reports.go`

```go
// GetTransferDCReport returns Transfer DCs with split progress for reporting.
func GetTransferDCReport(projectID int, dateFrom, dateTo string) ([]TransferDCReportRow, error) {
    // Query:
    // SELECT dc.dc_number, dc.status, dc.challan_date,
    //        tdc.num_destinations, tdc.num_split,
    //        tdc.transporter_name, tdc.vehicle_number,
    //        (SELECT COUNT(*) FROM transfer_dc_splits WHERE transfer_dc_id = tdc.id) as split_count
    // FROM delivery_challans dc
    // JOIN transfer_dcs tdc ON tdc.dc_id = dc.id
    // WHERE dc.project_id = ? AND dc.dc_type = 'transfer'
    // AND dc.challan_date BETWEEN ? AND ?
    // ORDER BY dc.challan_date DESC
}

type TransferDCReportRow struct {
    DCNumber        string
    Status          string
    ChallanDate     string
    NumDestinations int
    NumSplit        int
    SplitCount      int
    TransporterName string
    VehicleNumber   string
    ChildGroups     []TransferDCReportChildGroup
}

type TransferDCReportChildGroup struct {
    SplitNumber     int
    ShipmentGroupID int
    TransitDCNumber string
    NumOfficialDCs  int
    Status          string
}
```

### 4. Create Transfer DC report handler — `internal/handlers/reports.go`

```go
func ShowTransferDCReport(c echo.Context) error {
    // 1. Parse date range filter
    // 2. Query Transfer DC report data
    // 3. For each Transfer DC, get child split groups
    // 4. Render reports/transfer.templ
}

func ExportTransferDCReportExcel(c echo.Context) error {
    // Export report data as Excel
}
```

### 5. Create Transfer DC report template — `components/pages/reports/transfer.templ`

```
┌─────────────────────────────────────────────────────┐
│ Transfer DC Report                     [Date Filter] │
├─────────────────────────────────────────────────────┤
│ Summary Cards:                                       │
│ ┌──────┐ ┌──────┐ ┌──────────┐ ┌──────┐            │
│ │Draft │ │Issued│ │Splitting │ │Split │              │
│ │  2   │ │  3   │ │    5     │ │  10  │              │
│ └──────┘ └──────┘ └──────────┘ └──────┘              │
├─────────────────────────────────────────────────────┤
│ DC Number   │Status   │Date    │Dest│Split│Progress  │
│─────────────┼─────────┼────────┼────┼─────┼──────────│
│STDC-2526-001│Split    │01-03-26│ 25 │  5  │████████  │
│  ├─Split#1 → TDC-001 (5 ODCs) — Issued              │
│  ├─Split#2 → TDC-002 (5 ODCs) — Issued              │
│  ├─Split#3 → TDC-003 (5 ODCs) — Draft               │
│  ├─Split#4 → TDC-004 (5 ODCs) — Draft               │
│  └─Split#5 → TDC-005 (5 ODCs) — Draft               │
│─────────────┼─────────┼────────┼────┼─────┼──────────│
│STDC-2526-002│Splitting│05-03-26│ 15 │  2  │████░░░░  │
│  ├─Split#1 → TDC-006 (8 ODCs) — Issued              │
│  └─Split#2 → TDC-007 (3 ODCs) — Draft               │
│             │         │        │    │     │ 4 pending │
└─────────────────────────────────────────────────────┘
```

### 6. Update report index page — `components/pages/reports/index.templ`

Add Transfer DC report card:
```templ
<a href={reportURL("transfer")}>
    <div class="report-card">
        <h3>Transfer DC Report</h3>
        <p>View Transfer DCs with split progress and child group details</p>
    </div>
</a>
```

### 7. Ensure existing reports include Transfer DC data

**Destination Report**: Transfer DC destinations should appear in destination aggregation. Query should include destinations from `transfer_dc_destinations` alongside regular official DCs.

**Product Report**: Products in Transfer DCs should be counted. Line items from Transfer DCs contribute to product dispatch totals.

**Serial Report**: Serials from Transfer DCs are searchable. They exist in `serial_numbers` table linked to Transfer DC line items.

### 8. Register routes — `cmd/server/main.go`

```go
projectRoutes.GET("/reports/transfer", handlers.ShowTransferDCReport)
projectRoutes.GET("/reports/transfer/export", handlers.ExportTransferDCReportExcel)
```

---

## Acceptance Criteria

- [ ] DC listing page shows Transfer DCs with "transfer" type badge
- [ ] DC listing filter has "Transfer DC" option that works correctly
- [ ] Status filter includes "Splitting" and "Split" options
- [ ] DC Summary report includes Transfer DC counts with all 4 statuses
- [ ] Transfer DC report page shows hierarchical view (Transfer DC → splits → child groups)
- [ ] Transfer DC report has date range filtering
- [ ] Transfer DC report export to Excel works
- [ ] Destination report includes Transfer DC destinations
- [ ] Product report includes Transfer DC products
- [ ] Serial report includes Transfer DC serials
- [ ] All new status badges render correctly (splitting = orange, split = green)
- [ ] All tests pass
