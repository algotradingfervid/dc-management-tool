# Phase 10: PDF/Excel Export & Print Views

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 5 (Transfer Detail) — Transfer DC detail page must exist
- Phase 7 (Split Wizard) — Child groups may be referenced in exports

## Overview

Add full PDF and Excel export support for Transfer DCs, plus a browser print view. The Transfer DC export is similar to Transit DC export but includes additional sections for hub location, all destination addresses, and per-destination quantity breakdown.

---

## New Files

| File | Purpose |
|------|---------|
| `components/pages/transfer_dcs/print.templ` | Browser print view layout |

## Modified Files

| File | Changes |
|------|---------|
| `internal/services/pdf_builder.go` | Add `TransferDCPDFData` struct + `GenerateTransferDCPDF()` |
| `internal/services/excel_service.go` | Add `TransferDCExcelData` struct + `GenerateTransferDCExcel()` |
| `internal/handlers/export_handler.go` | Add transfer DC case in `ExportDCPDF()` and `ExportDCExcel()` |
| `internal/handlers/transfer_dc.go` | Add `ShowTransferDCPrintView()` handler |
| `cmd/server/main.go` | Add export routes |

---

## Tests to Write First

### PDF
- [ ] `TestGenerateTransferDCPDF_HappyPath` — PDF generates without error
- [ ] `TestGenerateTransferDCPDF_AllSections` — Contains header, addresses, line items, serials, destinations
- [ ] `TestGenerateTransferDCPDF_HubAddress` — Hub location prominently displayed
- [ ] `TestGenerateTransferDCPDF_DestinationGrid` — Per-destination quantities in table format

### Excel
- [ ] `TestGenerateTransferDCExcel_HappyPath` — Excel file generates without error
- [ ] `TestGenerateTransferDCExcel_LineItems` — Correct product rows with totals
- [ ] `TestGenerateTransferDCExcel_Destinations` — Destination breakdown sheet/section
- [ ] `TestGenerateTransferDCExcel_Serials` — All serials included

### Handler
- [ ] `TestExportTransferDCPDF_Handler` — HTTP 200 with PDF content-type
- [ ] `TestExportTransferDCExcel_Handler` — HTTP 200 with Excel content-type
- [ ] `TestExportTransferDC_NotFound` — HTTP 404 for invalid DC

---

## Implementation Steps

### 1. Define Transfer DC PDF data struct — `internal/services/pdf_builder.go`

```go
type TransferDCPDFData struct {
    // Project & Company Info
    ProjectName          string
    CompanyName          string
    CompanyGSTIN         string
    CompanyPAN           string
    CompanyEmail         string
    CompanyCIN           string
    SignatoryName        string
    SignatoryDesignation string
    SignatoryMobile      string
    CompanySignaturePath string
    CompanySealPath      string

    // DC Info
    DCNumber        string
    ChallanDate     string
    Status          string
    TaxType         string
    ReverseCharge   string

    // Hub & Transport
    HubAddress      map[string]string  // Hub address fields
    HubAddressName  string
    TransporterName string
    VehicleNumber   string
    EwayBillNumber  string
    DocketNumber    string

    // Addresses
    BillFromAddress    map[string]string
    DispatchFromAddress map[string]string
    BillToAddress      map[string]string

    // Line Items
    LineItems       []TransferDCPDFLineItem
    TotalTaxable    float64
    TotalCGST       float64
    TotalSGST       float64
    TotalIGST       float64
    GrandTotal      float64
    RoundOff        float64
    AmountInWords   string

    // Destinations (for quantity breakdown table)
    Destinations    []TransferDCPDFDestination
    Products        []TransferDCPDFProduct  // Column headers for destination grid

    // Tender/PO
    TenderRefNumber  string
    TenderRefDetails string
    POReference      string
    PODate           string
    PurposeText      string
}

type TransferDCPDFLineItem struct {
    SNo             int
    ItemName        string
    ItemDescription string
    HSNCode         string
    UoM             string
    Quantity        int
    Rate            float64
    TaxPercentage   float64
    TaxableAmount   float64
    TaxAmount       float64
    TotalAmount     float64
    SerialNumbers   []string
}

type TransferDCPDFDestination struct {
    Name          string
    Address       string
    Quantities    map[int]int  // productID → qty
    IsSplit       bool
    SplitGroupNum *int
}

type TransferDCPDFProduct struct {
    ID   int
    Name string
}
```

### 2. Implement GenerateTransferDCPDF — `internal/services/pdf_builder.go`

The Transfer DC PDF layout:

```
┌─────────────────────────────────────────────────────┐
│ COMPANY NAME                         DC No: STDC-001│
│ GSTIN: 36AACCF9742K1Z8              Date: 09-03-2026│
│ CIN: xxxxxxxx                                        │
├─────────────────────────────────────────────────────┤
│ SPLIT TRANSFER DELIVERY CHALLAN                      │
├──────────┬────────────┬──────────────────────────────┤
│ Bill From│ Dispatch   │ Bill To                      │
│          │ From       │                              │
├──────────┴────────────┴──────────────────────────────┤
│ Hub Location: District Warehouse, Hyderabad          │
│ Transporter: ABC Logistics | Vehicle: TS09-9999      │
│ E-Way Bill: EWB123 | Docket: DK456                   │
├─────────────────────────────────────────────────────┤
│ Tender Ref: xxx | PO Ref: xxx | PO Date: xxx        │
│ Purpose: Supply of materials as per PO               │
├─────────────────────────────────────────────────────┤
│ LINE ITEMS                                           │
│ # │Product│HSN │Qty│Rate │Tax%│Taxable│Tax  │Total  │
│ 1 │Prod A │8542│100│₹500│18% │₹50000│₹9000│₹59000 │
│ 2 │Prod B │    │ 50│₹300│18% │₹15000│₹2700│₹17700 │
│───┼───────┼────┼───┼────┼────┼──────┼─────┼───────│
│   │       │    │   │    │    │₹65000│₹5850│₹76700 │
│   │       │    │   │    │    │      │₹5850│(CGST) │
├─────────────────────────────────────────────────────┤
│ Amount in Words: Seventy-six thousand seven hundred  │
├─────────────────────────────────────────────────────┤
│ DESTINATION BREAKDOWN                                │
│ # │ Destination         │ Prod A │ Prod B │          │
│ 1 │ Mandal X, District Y│ 4      │ 2      │          │
│ 2 │ Mandal Z, District W│ 4      │ 2      │          │
│ ... (all 25 destinations)                            │
│ ──┼─────────────────────┼────────┼────────┤          │
│   │ TOTAL               │ 100    │ 50     │          │
├─────────────────────────────────────────────────────┤
│ SERIAL NUMBERS                                       │
│ Product A: SN001, SN002, SN003, ... SN100            │
│ Product B: SN101, SN102, ... SN150                   │
├─────────────────────────────────────────────────────┤
│ Tax Type: CGST + SGST | Reverse Charge: No          │
├─────────────────────────────────────────────────────┤
│ Authorized Signatory                                 │
│ [Signature]      [Seal]                              │
│ Name: John Doe                                       │
│ Designation: Manager                                 │
└─────────────────────────────────────────────────────┘
```

### 3. Implement GenerateTransferDCExcel — `internal/services/excel_service.go`

```go
type TransferDCExcelData struct {
    // Same fields as PDF data struct
    // ...
}

func GenerateTransferDCExcel(data TransferDCExcelData) (*bytes.Buffer, error) {
    // Sheet 1: "Transfer DC" — Header info + line items + totals
    // Sheet 2: "Destinations" — Destination × product quantity grid
    // Sheet 3: "Serial Numbers" — Product → serial list
}
```

### 4. Update export handler — `internal/handlers/export_handler.go`

In `ExportDCPDF()`:
```go
switch dc.DCType {
case "official":
    return buildOfficialPDF(c, dc, project)
case "transfer":
    return buildTransferPDF(c, dc, project)  // NEW
default:
    return buildTransitPDF(c, dc, project)
}
```

Same pattern for `ExportDCExcel()`.

### 5. Build Transfer PDF data helper

```go
func buildTransferPDF(c echo.Context, dc *models.DeliveryChallan, project *models.Project) error {
    // 1. Get Transfer DC record (hub, transporter)
    // 2. Get line items with serials
    // 3. Get all destinations with quantities
    // 4. Get addresses (bill_from, dispatch_from, bill_to, hub)
    // 5. Calculate totals, tax, amount in words
    // 6. Build TransferDCPDFData struct
    // 7. Call services.GenerateTransferDCPDF(data)
    // 8. Return PDF response
}
```

### 6. Create print view template — `components/pages/transfer_dcs/print.templ`

Browser print view with:
- All sections from PDF layout
- Print-optimized CSS (page breaks, margins)
- No sidebar, no topbar
- Print button that triggers `window.print()`

### 7. Register export routes — `cmd/server/main.go`

```go
// Already covered by existing generic routes:
// projectRoutes.GET("/dcs/:dcid/export/pdf", handlers.ExportDCPDF)
// projectRoutes.GET("/dcs/:dcid/export/excel", handlers.ExportDCExcel)
// These already dispatch by dc_type — just need the transfer case added

// Transfer DC-specific print view:
projectRoutes.GET("/transfer-dcs/:tdcid/print", handlers.ShowTransferDCPrintView)
```

---

## Acceptance Criteria

- [ ] PDF export generates valid PDF file with all Transfer DC sections
- [ ] PDF includes: company info, DC number, hub location, addresses, line items, amounts, destinations, serials
- [ ] Destination breakdown table shows all destinations with per-product quantities
- [ ] Excel export generates valid .xlsx with multiple sheets
- [ ] Browser print view renders correctly with print-optimized CSS
- [ ] Export handler correctly dispatches to Transfer DC builder
- [ ] Content-Type headers correct (application/pdf, application/vnd.openxmlformats-officedocument.spreadsheetml.sheet)
- [ ] Filename includes DC number (e.g., PRJ-STDC-2526-001.pdf)
- [ ] All tests pass
