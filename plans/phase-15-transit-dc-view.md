# Phase 15: Transit DC View & Print Layout

## Overview

This phase implements a complete print-ready view for Transit Delivery Challans, matching the mockup in 14-transit-dc-detail.html. The view displays all DC information in a professional format suitable for printing, including company header, addresses, product details with serial numbers, tax calculations, and signature sections. Print CSS ensures proper formatting when using browser print functionality.

**Key Features:**
- Print-ready layout matching 14-transit-dc-detail.html
- Company header with GSTIN and address
- DC details (number, date, transport info, e-way bill)
- Bill To / Ship To / Dispatch From addresses in grid layout
- Product table with serial numbers embedded
- Tax summary (CGST+SGST or IGST)
- Amount in words (Indian format: Lakhs, Crores)
- Signature section with uploaded company signature
- Print CSS (@media print) for clean printing
- Browser print button integration

## Prerequisites

- Phase 14 (DC Lifecycle) completed
- Phase 11 (Transit DC Creation) completed
- Phase 13 (Serial Number Management) completed
- Mockup file: 14-transit-dc-detail.html
- Company settings with signature image upload
- Database with complete DC data

## Goals

1. Create print-ready Transit DC view page
2. Match mockup 14-transit-dc-detail.html exactly
3. Display all DC information (header, addresses, products, taxes)
4. Show serial numbers within product description
5. Calculate and display totals correctly
6. Convert amount to words (Indian format)
7. Include company signature image
8. Implement print CSS for clean printing
9. Hide navigation/UI elements when printing
10. Support browser print (Ctrl+P or File → Print)

## Detailed Implementation Steps

### Step 1: Backend Handler for Transit DC View

Update `handlers/transit_dc_handler.go`:

**ViewTransitDC:**
- Fetch DC by dc_id
- Fetch transit details
- Fetch line items with serial numbers
- Fetch Bill To and Ship To addresses
- Fetch company details and signature
- Calculate totals
- Convert amount to words
- Render view template

```go
func ViewTransitDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    projectID := c.Param("project_id")

    db := c.MustGet("db").(*sql.DB)

    // Fetch DC with all details
    dc, err := models.GetTransitDCWithDetails(db, dcID)
    if err != nil {
        c.HTML(http.StatusNotFound, "404.html", gin.H{"error": "DC not found"})
        return
    }

    // Verify DC belongs to project
    if dc.ProjectID != projectID {
        c.HTML(http.StatusForbidden, "403.html", gin.H{"error": "Access denied"})
        return
    }

    // Fetch serial numbers for each line item
    serialsByProduct, err := models.GetSerialNumbersByDC(db, dcID)
    if err != nil {
        c.HTML(http.StatusInternalServerError, "500.html", gin.H{"error": err.Error()})
        return
    }

    // Attach serial numbers to line items
    for i := range dc.LineItems {
        productID := dc.LineItems[i].ProductID
        if serials, ok := serialsByProduct[productID]; ok {
            dc.LineItems[i].SerialNumbers = serials
        }
    }

    // Fetch company details
    company, err := models.GetCompanySettings(db)
    if err != nil {
        c.HTML(http.StatusInternalServerError, "500.html", gin.H{"error": err.Error()})
        return
    }

    // Convert amount to words
    amountInWords := utils.NumberToIndianWords(dc.TransitDetails.TotalValue)

    // Render view
    c.HTML(http.StatusOK, "dc/transit_view.html", gin.H{
        "DC":             dc,
        "Company":        company,
        "AmountInWords":  amountInWords,
        "PageTitle":      "Transit DC - " + dc.DCNumber,
    })
}
```

### Step 2: Create Models for Complete DC Data

Add to `models/transit_dc.go`:

```go
type TransitDCWithDetails struct {
    // DC Basic Info
    ID          int64
    ProjectID   int64
    DCNumber    string
    DCDate      time.Time
    Purpose     string
    Notes       string
    Status      string
    CreatedAt   time.Time
    IssuedAt    *time.Time

    // Addresses
    BillTo      Address
    ShipTo      Address
    DispatchFrom Address

    // Transit Details
    TransitDetails TransitDetails

    // Line Items
    LineItems []DCLineItemWithSerials

    // Project Details
    Project Project
}

type DCLineItemWithSerials struct {
    ID            int64
    LineNumber    int
    ItemName      string
    Description   string
    BrandModel    string
    UOM           string
    HSNCode       string
    Price         float64
    GSTPercentage float64
    Quantity      int
    TaxableValue  float64
    GSTAmount     float64
    TotalAmount   float64
    SerialNumbers []string
}

type Address struct {
    ID          int64
    CompanyName string
    Address     string
    City        string
    State       string
    StateCode   string
    Pincode     string
    GSTIN       string
}

type CompanySettings struct {
    Name           string
    Address        string
    City           string
    State          string
    StateCode      string
    Pincode        string
    GSTIN          string
    SignatureImage string // Path to signature image
}

func GetTransitDCWithDetails(db *sql.DB, dcID string) (*TransitDCWithDetails, error) {
    dc := &TransitDCWithDetails{}

    // Main query joining all necessary tables
    query := `
        SELECT
            dc.id, dc.project_id, dc.dc_number, dc.dc_date,
            dc.purpose, dc.notes, dc.status, dc.created_at, dc.issued_at,

            td.mode_of_transport, td.driver_name, td.vehicle_number,
            td.docket_number, td.eway_bill_number, td.reverse_charge,
            td.tax_type, td.taxable_value, td.cgst_amount, td.sgst_amount,
            td.igst_amount, td.round_off, td.total_value,

            bt.id, bt.company_name, bt.address, bt.city, bt.state,
            bt.state_code, bt.pincode, bt.gstin,

            st.id, st.company_name, st.address, st.city, st.state,
            st.state_code, st.pincode, st.gstin,

            p.id, p.name, p.description, p.po_number, p.tender_reference

        FROM delivery_challans dc
        INNER JOIN dc_transit_details td ON dc.id = td.dc_id
        LEFT JOIN project_addresses bt ON dc.bill_to_id = bt.id
        LEFT JOIN project_addresses st ON dc.ship_to_id = st.id
        LEFT JOIN projects p ON dc.project_id = p.id
        WHERE dc.id = ? AND dc.dc_type = 'transit'
    `

    var issuedAt sql.NullTime
    var docketNumber, ewayBillNumber sql.NullString
    var cgst, sgst, igst sql.NullFloat64

    err := db.QueryRow(query, dcID).Scan(
        &dc.ID, &dc.ProjectID, &dc.DCNumber, &dc.DCDate,
        &dc.Purpose, &dc.Notes, &dc.Status, &dc.CreatedAt, &issuedAt,

        &dc.TransitDetails.ModeOfTransport, &dc.TransitDetails.DriverName,
        &dc.TransitDetails.VehicleNumber, &docketNumber, &ewayBillNumber,
        &dc.TransitDetails.ReverseCharge, &dc.TransitDetails.TaxType,
        &dc.TransitDetails.TaxableValue, &cgst, &sgst, &igst,
        &dc.TransitDetails.RoundOff, &dc.TransitDetails.TotalValue,

        &dc.BillTo.ID, &dc.BillTo.CompanyName, &dc.BillTo.Address,
        &dc.BillTo.City, &dc.BillTo.State, &dc.BillTo.StateCode,
        &dc.BillTo.Pincode, &dc.BillTo.GSTIN,

        &dc.ShipTo.ID, &dc.ShipTo.CompanyName, &dc.ShipTo.Address,
        &dc.ShipTo.City, &dc.ShipTo.State, &dc.ShipTo.StateCode,
        &dc.ShipTo.Pincode, &dc.ShipTo.GSTIN,

        &dc.Project.ID, &dc.Project.Name, &dc.Project.Description,
        &dc.Project.PONumber, &dc.Project.TenderReference,
    )

    if err != nil {
        return nil, err
    }

    // Handle nullable fields
    if issuedAt.Valid {
        dc.IssuedAt = &issuedAt.Time
    }
    if docketNumber.Valid {
        dc.TransitDetails.DocketNumber = &docketNumber.String
    }
    if ewayBillNumber.Valid {
        dc.TransitDetails.EWayBillNumber = &ewayBillNumber.String
    }
    if cgst.Valid {
        dc.TransitDetails.CGSTAmount = &cgst.Float64
    }
    if sgst.Valid {
        dc.TransitDetails.SGSTAmount = &sgst.Float64
    }
    if igst.Valid {
        dc.TransitDetails.IGSTAmount = &igst.Float64
    }

    // Fetch line items
    lineItems, err := getLineItemsForDC(db, dcID)
    if err != nil {
        return nil, err
    }
    dc.LineItems = lineItems

    // Get dispatch from address (company address)
    dispatchFrom, err := getCompanyAddress(db)
    if err != nil {
        return nil, err
    }
    dc.DispatchFrom = dispatchFrom

    return dc, nil
}

func getLineItemsForDC(db *sql.DB, dcID string) ([]DCLineItemWithSerials, error) {
    query := `
        SELECT
            id, line_number, item_name, description, brand_model,
            uom, hsn_code, price, gst_percentage, quantity,
            taxable_value, gst_amount, total_amount
        FROM dc_line_items
        WHERE dc_id = ?
        ORDER BY line_number
    `

    rows, err := db.Query(query, dcID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    lineItems := []DCLineItemWithSerials{}
    for rows.Next() {
        var item DCLineItemWithSerials
        err := rows.Scan(
            &item.ID, &item.LineNumber, &item.ItemName, &item.Description,
            &item.BrandModel, &item.UOM, &item.HSNCode, &item.Price,
            &item.GSTPercentage, &item.Quantity, &item.TaxableValue,
            &item.GSTAmount, &item.TotalAmount,
        )
        if err != nil {
            return nil, err
        }
        lineItems = append(lineItems, item)
    }

    return lineItems, nil
}
```

### Step 3: Number to Words Utility (Indian Format)

Create `utils/number_to_words.go`:

```go
package utils

import (
    "fmt"
    "math"
    "strings"
)

// NumberToIndianWords converts a number to words in Indian numbering system
func NumberToIndianWords(num float64) string {
    if num == 0 {
        return "Zero Rupees Only"
    }

    // Split into rupees and paise
    rupees := int64(num)
    paise := int64(math.Round((num - float64(rupees)) * 100))

    words := ""

    // Convert rupees
    if rupees > 0 {
        words = convertToIndianWords(rupees) + " Rupees"
    }

    // Convert paise
    if paise > 0 {
        if words != "" {
            words += " and "
        }
        words += convertToIndianWords(paise) + " Paise"
    }

    return words + " Only"
}

func convertToIndianWords(num int64) string {
    if num == 0 {
        return ""
    }

    ones := []string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine"}
    tens := []string{"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"}
    teens := []string{"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"}

    convertTwoDigits := func(n int64) string {
        if n < 10 {
            return ones[n]
        }
        if n >= 10 && n < 20 {
            return teens[n-10]
        }
        return strings.TrimSpace(tens[n/10] + " " + ones[n%10])
    }

    convertThreeDigits := func(n int64) string {
        if n < 100 {
            return convertTwoDigits(n)
        }
        hundred := ones[n/100] + " Hundred"
        remainder := n % 100
        if remainder > 0 {
            return hundred + " " + convertTwoDigits(remainder)
        }
        return hundred
    }

    // Indian numbering: Crores, Lakhs, Thousands, Hundreds
    crore := num / 10000000
    num = num % 10000000

    lakh := num / 100000
    num = num % 100000

    thousand := num / 1000
    num = num % 1000

    words := []string{}

    if crore > 0 {
        words = append(words, convertTwoDigits(crore)+" Crore")
    }

    if lakh > 0 {
        words = append(words, convertTwoDigits(lakh)+" Lakh")
    }

    if thousand > 0 {
        words = append(words, convertTwoDigits(thousand)+" Thousand")
    }

    if num > 0 {
        words = append(words, convertThreeDigits(num))
    }

    return strings.TrimSpace(strings.Join(words, " "))
}
```

### Step 4: Create Transit DC View Template

Create `views/dc/transit_view.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.PageTitle}}</title>
    <link href="/static/css/tailwind.css" rel="stylesheet">
    <link href="/static/css/print.css" rel="stylesheet">
</head>
<body class="bg-gray-100">
    <!-- Print Button (Hidden on Print) -->
    <div class="no-print fixed top-4 right-4 z-50">
        <button onclick="window.print()" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 shadow-lg">
            Print DC
        </button>
        <a href="/projects/{{.DC.ProjectID}}/dc" class="ml-2 px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 shadow-lg">
            Back to List
        </a>
    </div>

    <!-- Print-Ready Container -->
    <div class="max-w-5xl mx-auto bg-white shadow-lg my-8 p-8 print:shadow-none print:m-0 print:p-6">

        <!-- Company Header -->
        <div class="text-center border-b-2 border-gray-800 pb-4 mb-6">
            <h1 class="text-2xl font-bold">{{.Company.Name}}</h1>
            <p class="text-sm mt-1">{{.Company.Address}}, {{.Company.City}}, {{.Company.State}} - {{.Company.Pincode}}</p>
            <p class="text-sm">GSTIN: <strong>{{.Company.GSTIN}}</strong></p>
        </div>

        <!-- DC Title -->
        <div class="text-center mb-6">
            <h2 class="text-xl font-bold uppercase">Delivery Challan</h2>
        </div>

        <!-- DC Details Section -->
        <div class="grid grid-cols-2 gap-6 mb-6">
            <!-- Left Column -->
            <div class="space-y-2">
                <div class="flex">
                    <span class="font-semibold w-40">DC Number:</span>
                    <span>{{.DC.DCNumber}}</span>
                </div>
                <div class="flex">
                    <span class="font-semibold w-40">DC Date:</span>
                    <span>{{.DC.DCDate.Format "02-Jan-2006"}}</span>
                </div>
                <div class="flex">
                    <span class="font-semibold w-40">Mode of Transport:</span>
                    <span>{{.DC.TransitDetails.ModeOfTransport}}</span>
                </div>
                <div class="flex">
                    <span class="font-semibold w-40">Vehicle Number:</span>
                    <span>{{.DC.TransitDetails.VehicleNumber}}</span>
                </div>
                <div class="flex">
                    <span class="font-semibold w-40">Driver Name:</span>
                    <span>{{.DC.TransitDetails.DriverName}}</span>
                </div>
                {{if .DC.TransitDetails.DocketNumber}}
                <div class="flex">
                    <span class="font-semibold w-40">Docket Number:</span>
                    <span>{{.DC.TransitDetails.DocketNumber}}</span>
                </div>
                {{end}}
                {{if .DC.TransitDetails.EWayBillNumber}}
                <div class="flex">
                    <span class="font-semibold w-40">E-Way Bill No:</span>
                    <span>{{.DC.TransitDetails.EWayBillNumber}}</span>
                </div>
                {{end}}
            </div>

            <!-- Right Column -->
            <div class="space-y-2">
                <div class="flex">
                    <span class="font-semibold w-40">State:</span>
                    <span>{{.DC.ShipTo.State}}</span>
                </div>
                <div class="flex">
                    <span class="font-semibold w-40">State Code:</span>
                    <span>{{.DC.ShipTo.StateCode}}</span>
                </div>
                <div class="flex">
                    <span class="font-semibold w-40">Reverse Charge:</span>
                    <span>{{if .DC.TransitDetails.ReverseCharge}}Yes{{else}}No{{end}}</span>
                </div>
            </div>
        </div>

        <!-- Project Details -->
        <div class="mb-6 p-4 bg-gray-50 rounded border">
            <div class="grid grid-cols-2 gap-4">
                <div>
                    <span class="font-semibold">Project:</span>
                    <span>{{.DC.Project.Name}}</span>
                </div>
                <div>
                    <span class="font-semibold">PO Number:</span>
                    <span>{{.DC.Project.PONumber}}</span>
                </div>
                {{if .DC.Project.TenderReference}}
                <div class="col-span-2">
                    <span class="font-semibold">Tender Reference:</span>
                    <span>{{.DC.Project.TenderReference}}</span>
                </div>
                {{end}}
            </div>
        </div>

        <!-- Addresses Grid (2x2) -->
        <div class="grid grid-cols-2 gap-4 mb-6 text-sm">
            <!-- Bill From (Dispatch From) -->
            <div class="border border-gray-300 p-3">
                <h3 class="font-bold mb-2">Bill From:</h3>
                <p class="font-semibold">{{.DC.DispatchFrom.CompanyName}}</p>
                <p>{{.DC.DispatchFrom.Address}}</p>
                <p>{{.DC.DispatchFrom.City}}, {{.DC.DispatchFrom.State}} - {{.DC.DispatchFrom.Pincode}}</p>
                <p><strong>GSTIN:</strong> {{.DC.DispatchFrom.GSTIN}}</p>
                <p><strong>State Code:</strong> {{.DC.DispatchFrom.StateCode}}</p>
            </div>

            <!-- Bill To -->
            <div class="border border-gray-300 p-3">
                <h3 class="font-bold mb-2">Bill To:</h3>
                <p class="font-semibold">{{.DC.BillTo.CompanyName}}</p>
                <p>{{.DC.BillTo.Address}}</p>
                <p>{{.DC.BillTo.City}}, {{.DC.BillTo.State}} - {{.DC.BillTo.Pincode}}</p>
                <p><strong>GSTIN:</strong> {{.DC.BillTo.GSTIN}}</p>
                <p><strong>State Code:</strong> {{.DC.BillTo.StateCode}}</p>
            </div>

            <!-- Dispatch From (same as Bill From typically) -->
            <div class="border border-gray-300 p-3">
                <h3 class="font-bold mb-2">Dispatch From:</h3>
                <p class="font-semibold">{{.DC.DispatchFrom.CompanyName}}</p>
                <p>{{.DC.DispatchFrom.Address}}</p>
                <p>{{.DC.DispatchFrom.City}}, {{.DC.DispatchFrom.State}} - {{.DC.DispatchFrom.Pincode}}</p>
            </div>

            <!-- Ship To -->
            <div class="border border-gray-300 p-3">
                <h3 class="font-bold mb-2">Ship To:</h3>
                <p class="font-semibold">{{.DC.ShipTo.CompanyName}}</p>
                <p>{{.DC.ShipTo.Address}}</p>
                <p>{{.DC.ShipTo.City}}, {{.DC.ShipTo.State}} - {{.DC.ShipTo.Pincode}}</p>
                <p><strong>GSTIN:</strong> {{.DC.ShipTo.GSTIN}}</p>
            </div>
        </div>

        <!-- Product Table -->
        <table class="w-full border-collapse border border-gray-300 mb-6 text-sm">
            <thead class="bg-gray-100">
                <tr>
                    <th class="border border-gray-300 px-2 py-2 text-left w-12">S.No</th>
                    <th class="border border-gray-300 px-2 py-2 text-left">Description of Goods</th>
                    <th class="border border-gray-300 px-2 py-2 text-center w-16">UoM</th>
                    <th class="border border-gray-300 px-2 py-2 text-center w-20">HSN</th>
                    <th class="border border-gray-300 px-2 py-2 text-center w-16">Qty</th>
                    <th class="border border-gray-300 px-2 py-2 text-right w-24">Per Unit</th>
                    <th class="border border-gray-300 px-2 py-2 text-right w-28">Taxable Value</th>
                    <th class="border border-gray-300 px-2 py-2 text-center w-16">GST%</th>
                    <th class="border border-gray-300 px-2 py-2 text-right w-24">GST Amt</th>
                    <th class="border border-gray-300 px-2 py-2 text-right w-28">Total</th>
                </tr>
            </thead>
            <tbody>
                {{range $index, $item := .DC.LineItems}}
                <tr>
                    <td class="border border-gray-300 px-2 py-2 text-center">{{add $index 1}}</td>
                    <td class="border border-gray-300 px-2 py-2">
                        <div class="font-semibold">{{$item.ItemName}}</div>
                        <div class="text-xs text-gray-600">{{$item.Description}}</div>
                        {{if $item.BrandModel}}
                        <div class="text-xs text-gray-600">Model: {{$item.BrandModel}}</div>
                        {{end}}
                        {{if $item.SerialNumbers}}
                        <div class="text-xs mt-1">
                            <strong>Serial Numbers:</strong>
                            <div class="font-mono">{{join $item.SerialNumbers ", "}}</div>
                        </div>
                        {{end}}
                    </td>
                    <td class="border border-gray-300 px-2 py-2 text-center">{{$item.UOM}}</td>
                    <td class="border border-gray-300 px-2 py-2 text-center">{{$item.HSNCode}}</td>
                    <td class="border border-gray-300 px-2 py-2 text-center">{{$item.Quantity}}</td>
                    <td class="border border-gray-300 px-2 py-2 text-right">₹{{printf "%.2f" $item.Price}}</td>
                    <td class="border border-gray-300 px-2 py-2 text-right">₹{{printf "%.2f" $item.TaxableValue}}</td>
                    <td class="border border-gray-300 px-2 py-2 text-center">{{printf "%.0f" $item.GSTPercentage}}%</td>
                    <td class="border border-gray-300 px-2 py-2 text-right">₹{{printf "%.2f" $item.GSTAmount}}</td>
                    <td class="border border-gray-300 px-2 py-2 text-right font-semibold">₹{{printf "%.2f" $item.TotalAmount}}</td>
                </tr>
                {{end}}

                <!-- Totals Row -->
                <tr class="bg-gray-50 font-bold">
                    <td colspan="6" class="border border-gray-300 px-2 py-2 text-right">Total:</td>
                    <td class="border border-gray-300 px-2 py-2 text-right">₹{{printf "%.2f" .DC.TransitDetails.TaxableValue}}</td>
                    <td class="border border-gray-300 px-2 py-2"></td>
                    <td class="border border-gray-300 px-2 py-2 text-right">
                        {{if eq .DC.TransitDetails.TaxType "cgst_sgst"}}
                            ₹{{printf "%.2f" (add .DC.TransitDetails.CGSTAmount .DC.TransitDetails.SGSTAmount)}}
                        {{else}}
                            ₹{{printf "%.2f" .DC.TransitDetails.IGSTAmount}}
                        {{end}}
                    </td>
                    <td class="border border-gray-300 px-2 py-2 text-right">₹{{printf "%.2f" .DC.TransitDetails.TotalValue}}</td>
                </tr>
            </tbody>
        </table>

        <!-- Tax Summary Box -->
        <div class="border border-gray-300 p-4 mb-6 max-w-md ml-auto">
            <div class="grid grid-cols-2 gap-2 text-sm">
                <div class="font-semibold">Taxable Value:</div>
                <div class="text-right">₹{{printf "%.2f" .DC.TransitDetails.TaxableValue}}</div>

                {{if eq .DC.TransitDetails.TaxType "cgst_sgst"}}
                <div class="font-semibold">CGST:</div>
                <div class="text-right">₹{{printf "%.2f" .DC.TransitDetails.CGSTAmount}}</div>

                <div class="font-semibold">SGST:</div>
                <div class="text-right">₹{{printf "%.2f" .DC.TransitDetails.SGSTAmount}}</div>
                {{else}}
                <div class="font-semibold">IGST:</div>
                <div class="text-right">₹{{printf "%.2f" .DC.TransitDetails.IGSTAmount}}</div>
                {{end}}

                <div class="font-semibold">Round Off:</div>
                <div class="text-right">₹{{printf "%.2f" .DC.TransitDetails.RoundOff}}</div>

                <div class="font-bold text-base border-t pt-2">Invoice/DC Value:</div>
                <div class="text-right font-bold text-base border-t pt-2">₹{{printf "%.2f" .DC.TransitDetails.TotalValue}}</div>
            </div>
        </div>

        <!-- Amount in Words -->
        <div class="mb-6 p-3 bg-gray-50 border border-gray-300">
            <strong>Amount in Words:</strong>
            <div class="mt-1 italic">{{.AmountInWords}}</div>
        </div>

        <!-- Notes -->
        {{if .DC.Notes}}
        <div class="mb-6">
            <strong>Notes:</strong>
            <p class="mt-1 text-sm">{{.DC.Notes}}</p>
        </div>
        {{end}}

        <!-- Signature Section -->
        <div class="grid grid-cols-2 gap-8 mt-12 pt-6 border-t-2 border-gray-300">
            <!-- Receiver's Signature -->
            <div>
                <p class="font-semibold mb-16">Receiver's Signature</p>
                <div class="border-t border-gray-400 pt-2">
                    <p class="text-xs text-gray-600">Name & Signature</p>
                </div>
            </div>

            <!-- Company Signature -->
            <div class="text-right">
                <p class="font-semibold mb-4">For {{.Company.Name}}</p>
                {{if .Company.SignatureImage}}
                <img src="{{.Company.SignatureImage}}" alt="Signature" class="ml-auto h-16 mb-2">
                {{else}}
                <div class="h-16 mb-2"></div>
                {{end}}
                <div class="border-t border-gray-400 pt-2 inline-block min-w-[200px]">
                    <p class="text-xs text-gray-600">Authorized Signatory</p>
                </div>
            </div>
        </div>

        <!-- Footer (Print Only) -->
        <div class="print-only text-center text-xs text-gray-500 mt-8 pt-4 border-t">
            <p>This is a computer-generated delivery challan and does not require a signature.</p>
            <p>Generated on {{now.Format "02-Jan-2006 15:04:05"}}</p>
        </div>
    </div>
</body>
</html>
```

### Step 5: Print CSS Stylesheet

Create `static/css/print.css`:

```css
/* Print-specific styles */
@media print {
    /* Remove all margins and backgrounds */
    body {
        margin: 0;
        padding: 0;
        background: white;
    }

    /* Hide non-printable elements */
    .no-print {
        display: none !important;
    }

    /* Show print-only elements */
    .print-only {
        display: block !important;
    }

    /* Reset container for full-page print */
    .max-w-5xl {
        max-width: 100%;
        margin: 0;
        padding: 1cm;
    }

    /* Remove shadows and borders that don't print well */
    .shadow-lg,
    .shadow {
        box-shadow: none !important;
    }

    /* Page break controls */
    .page-break-before {
        page-break-before: always;
    }

    .page-break-after {
        page-break-after: always;
    }

    .page-break-avoid {
        page-break-inside: avoid;
    }

    /* Ensure tables don't break awkwardly */
    table {
        page-break-inside: avoid;
    }

    thead {
        display: table-header-group;
    }

    tr {
        page-break-inside: avoid;
    }

    /* Signature section should stay together */
    .grid.grid-cols-2:has(.font-semibold) {
        page-break-inside: avoid;
    }

    /* Optimize font sizes for print */
    body {
        font-size: 12pt;
    }

    h1 {
        font-size: 20pt;
    }

    h2 {
        font-size: 16pt;
    }

    h3 {
        font-size: 14pt;
    }

    .text-xs {
        font-size: 9pt;
    }

    .text-sm {
        font-size: 10pt;
    }

    /* Ensure proper page margins */
    @page {
        margin: 1cm;
        size: A4 portrait;
    }

    /* Print links without underlines */
    a {
        text-decoration: none;
        color: inherit;
    }

    /* Ensure borders print correctly */
    .border,
    .border-gray-300,
    .border-gray-800 {
        border-color: #000 !important;
    }

    /* Background colors should print */
    .bg-gray-50,
    .bg-gray-100 {
        background-color: #f9f9f9 !important;
        -webkit-print-color-adjust: exact;
        print-color-adjust: exact;
    }
}

/* Screen-only styles */
@media screen {
    .print-only {
        display: none;
    }
}
```

### Step 6: Template Helper Functions

Add template helpers in `main.go` or template registration:

```go
// Template functions
funcMap := template.FuncMap{
    "add": func(a, b int) int {
        return a + b
    },
    "join": func(strs []string, sep string) string {
        return strings.Join(strs, sep)
    },
    "now": func() time.Time {
        return time.Now()
    },
}

// Load templates with functions
tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("views/**/*.html"))
```

### Step 7: Add Route for View

Update `routes/dc_routes.go`:

```go
// View routes
dcRoutes.GET("/transit/:dc_id/view", handlers.ViewTransitDC)
dcRoutes.GET("/transit/:dc_id/print", handlers.ViewTransitDC) // Same as view, browser print handles it
```

### Step 8: Company Settings for Signature

Add company settings table if not exists:

```sql
CREATE TABLE IF NOT EXISTS company_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- Only one row allowed
    name VARCHAR(255) NOT NULL,
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(100),
    state_code VARCHAR(10),
    pincode VARCHAR(10),
    gstin VARCHAR(20),
    signature_image VARCHAR(500), -- Path to uploaded signature
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default company settings
INSERT INTO company_settings (id, name, address, city, state, state_code, pincode, gstin)
VALUES (
    1,
    'FERVID SMART SOLUTIONS PVT. LTD',
    '123 Business Park, Tech Road',
    'Bangalore',
    'Karnataka',
    '29',
    '560001',
    '29AAACF1234M1Z5'
) ON CONFLICT(id) DO NOTHING;
```

### Step 9: Signature Upload Handler (Optional)

Create handler to upload company signature:

```go
func UploadCompanySignature(c *gin.Context) {
    file, err := c.FormFile("signature")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
        return
    }

    // Validate file type
    if !strings.HasSuffix(file.Filename, ".png") && !strings.HasSuffix(file.Filename, ".jpg") {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Only PNG and JPG files allowed"})
        return
    }

    // Save file
    filepath := "static/uploads/signature.png"
    if err := c.SaveUploadedFile(file, filepath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
        return
    }

    // Update database
    db := c.MustGet("db").(*sql.DB)
    _, err = db.Exec("UPDATE company_settings SET signature_image = ? WHERE id = 1", "/"+filepath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "path": "/" + filepath})
}
```

## Files to Create/Modify

### New Files

1. **views/dc/transit_view.html**
   - Complete print-ready DC view
   - Matches 14-transit-dc-detail.html mockup
   - Includes all sections and styling

2. **static/css/print.css**
   - Print-specific styles
   - @media print rules
   - Page break controls

3. **utils/number_to_words.go**
   - NumberToIndianWords function
   - Indian numbering system (Lakhs, Crores)

4. **handlers/company_settings_handler.go**
   - UploadCompanySignature
   - GetCompanySettings

5. **database/migrations/005_company_settings.sql**
   - CREATE TABLE company_settings
   - Default company data

### Modified Files

1. **handlers/transit_dc_handler.go**
   - Add ViewTransitDC handler
   - Fetch complete DC data with serials

2. **models/transit_dc.go**
   - Add TransitDCWithDetails struct
   - Add GetTransitDCWithDetails method
   - Add Address, CompanySettings structs

3. **routes/dc_routes.go**
   - Add /transit/:dc_id/view route
   - Add /transit/:dc_id/print route

4. **main.go**
   - Add template helper functions (add, join, now)

## API Routes/Endpoints

### View Routes

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects/:project_id/dc/transit/:dc_id/view` | View Transit DC (print-ready) |
| GET | `/projects/:project_id/dc/transit/:dc_id/print` | Same as view (browser print) |

### Company Settings (Optional)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/admin/company/signature` | Upload company signature image |
| GET | `/admin/company/settings` | Get company settings |

## Database Queries

### Fetch Complete Transit DC

```sql
SELECT
    dc.id, dc.project_id, dc.dc_number, dc.dc_date,
    dc.purpose, dc.notes, dc.status, dc.created_at, dc.issued_at,

    td.mode_of_transport, td.driver_name, td.vehicle_number,
    td.docket_number, td.eway_bill_number, td.reverse_charge,
    td.tax_type, td.taxable_value, td.cgst_amount, td.sgst_amount,
    td.igst_amount, td.round_off, td.total_value,

    bt.company_name, bt.address, bt.city, bt.state,
    bt.state_code, bt.pincode, bt.gstin,

    st.company_name, st.address, st.city, st.state,
    st.state_code, st.pincode, st.gstin

FROM delivery_challans dc
INNER JOIN dc_transit_details td ON dc.id = td.dc_id
LEFT JOIN project_addresses bt ON dc.bill_to_id = bt.id
LEFT JOIN project_addresses st ON dc.ship_to_id = st.id
WHERE dc.id = ? AND dc.dc_type = 'transit';
```

### Fetch Line Items with Serial Numbers

```sql
-- Line items
SELECT
    li.id, li.line_number, li.item_name, li.description,
    li.brand_model, li.uom, li.hsn_code, li.price,
    li.gst_percentage, li.quantity, li.taxable_value,
    li.gst_amount, li.total_amount
FROM dc_line_items li
WHERE li.dc_id = ?
ORDER BY li.line_number;

-- Serial numbers
SELECT product_id, serial_number
FROM serial_numbers
WHERE dc_id = ?
ORDER BY id;
```

### Fetch Company Settings

```sql
SELECT
    name, address, city, state, state_code,
    pincode, gstin, signature_image
FROM company_settings
WHERE id = 1;
```

## UI Components

### 1. Print Button

```html
<button onclick="window.print()" class="no-print px-4 py-2 bg-blue-600 text-white rounded-md">
    Print DC
</button>
```

### 2. Company Header

```html
<div class="text-center border-b-2 border-gray-800 pb-4">
    <h1 class="text-2xl font-bold">FERVID SMART SOLUTIONS PVT. LTD</h1>
    <p class="text-sm">123 Business Park, Tech Road, Bangalore - 560001</p>
    <p class="text-sm">GSTIN: <strong>29AAACF1234M1Z5</strong></p>
</div>
```

### 3. Address Grid (2x2)

```html
<div class="grid grid-cols-2 gap-4">
    <div class="border p-3">
        <h3 class="font-bold">Bill From:</h3>
        <!-- Address details -->
    </div>
    <div class="border p-3">
        <h3 class="font-bold">Bill To:</h3>
        <!-- Address details -->
    </div>
    <div class="border p-3">
        <h3 class="font-bold">Dispatch From:</h3>
        <!-- Address details -->
    </div>
    <div class="border p-3">
        <h3 class="font-bold">Ship To:</h3>
        <!-- Address details -->
    </div>
</div>
```

### 4. Product Table with Serial Numbers

```html
<td class="border px-2 py-2">
    <div class="font-semibold">Smart Lock Pro</div>
    <div class="text-xs text-gray-600">WiFi enabled smart door lock</div>
    <div class="text-xs text-gray-600">Model: SL-2000X</div>
    <div class="text-xs mt-1">
        <strong>Serial Numbers:</strong>
        <div class="font-mono">SN001, SN002, SN003</div>
    </div>
</td>
```

### 5. Signature Section

```html
<div class="grid grid-cols-2 gap-8">
    <div>
        <p class="font-semibold">Receiver's Signature</p>
        <div class="border-t pt-2">Name & Signature</div>
    </div>
    <div class="text-right">
        <p class="font-semibold">For Company Name</p>
        <img src="/static/uploads/signature.png" class="h-16">
        <div class="border-t pt-2">Authorized Signatory</div>
    </div>
</div>
```

## Testing Checklist

### Functional Testing

- [ ] Access Transit DC view page
- [ ] Verify all DC details display correctly
- [ ] Verify company header shows correct info
- [ ] Verify addresses display in 2x2 grid
- [ ] Verify product table shows all line items
- [ ] Verify serial numbers appear within product description
- [ ] Verify tax calculations display correctly (CGST+SGST or IGST)
- [ ] Verify amount in words is accurate
- [ ] Verify company signature image displays (if uploaded)
- [ ] Click Print button and verify browser print dialog opens
- [ ] Verify print preview matches expected layout
- [ ] Test actual printing to PDF
- [ ] Test actual printing to paper

### Layout Testing

- [ ] Verify layout matches 14-transit-dc-detail.html mockup
- [ ] Test on different screen sizes (desktop, tablet)
- [ ] Verify all sections are properly aligned
- [ ] Verify table borders are consistent
- [ ] Verify signature section spacing
- [ ] Test with very long serial number lists (wrapping)
- [ ] Test with many line items (pagination)

### Print Testing

- [ ] Verify Print button is hidden when printing
- [ ] Verify navigation/UI elements are hidden when printing
- [ ] Verify page margins are correct (1cm)
- [ ] Verify table doesn't break across pages awkwardly
- [ ] Verify signature section stays together
- [ ] Verify colors print correctly (if color printer)
- [ ] Verify black & white print is readable
- [ ] Test print preview in Chrome, Firefox, Safari
- [ ] Verify A4 page size is respected

### Data Testing

- [ ] Test with Transit DC (CGST+SGST)
- [ ] Test with Transit DC (IGST)
- [ ] Test with optional fields empty (Docket, E-Way Bill)
- [ ] Test with very long product descriptions
- [ ] Test with 0 serial numbers (should handle gracefully)
- [ ] Test with 100+ serial numbers (formatting)
- [ ] Verify amount in words for various amounts:
  - [ ] ₹0
  - [ ] ₹1
  - [ ] ₹99
  - [ ] ₹1,000
  - [ ] ₹1,00,000 (1 Lakh)
  - [ ] ₹10,00,000 (10 Lakhs)
  - [ ] ₹1,00,00,000 (1 Crore)

### Integration Testing

- [ ] Verify view is accessible from DC detail page
- [ ] Verify view shows data from Phase 11 (Transit DC)
- [ ] Verify serial numbers from Phase 13 are displayed
- [ ] Verify status from Phase 14 doesn't affect view
- [ ] Test with different company settings

## Acceptance Criteria

### Must Have

1. **Layout Compliance**
   - ✅ View matches mockup 14-transit-dc-detail.html
   - ✅ All sections present and correctly positioned
   - ✅ Professional appearance suitable for business use

2. **Data Display**
   - ✅ All DC information displayed accurately
   - ✅ Company header with GSTIN
   - ✅ DC details (number, date, transport info)
   - ✅ Addresses in 2x2 grid layout
   - ✅ Product table with serial numbers
   - ✅ Tax calculations (CGST+SGST or IGST)
   - ✅ Amount in words (Indian format)

3. **Print Functionality**
   - ✅ Browser print button works
   - ✅ Print CSS hides UI elements
   - ✅ Print layout is clean and professional
   - ✅ Page margins are appropriate
   - ✅ No awkward page breaks

4. **Serial Numbers**
   - ✅ Serial numbers embedded in product description
   - ✅ Comma-separated format
   - ✅ Handles long lists gracefully

5. **Signature Section**
   - ✅ Receiver's signature placeholder
   - ✅ Company signature image (if uploaded)
   - ✅ Authorized signatory text

### Should Have

1. **Responsive Design**
   - ✅ Readable on desktop screens
   - ✅ Print layout optimized for A4

2. **Amount in Words**
   - ✅ Accurate conversion for all amounts
   - ✅ Indian numbering (Lakhs, Crores)
   - ✅ Handles decimals (paise)

3. **Professional Appearance**
   - ✅ Clean typography
   - ✅ Consistent spacing
   - ✅ Proper alignment

### Nice to Have

1. **Advanced Features**
   - ⭕ Download as PDF button
   - ⭕ Email DC directly from view
   - ⭕ Print multiple DCs at once
   - ⭕ Customizable company logo

2. **Print Enhancements**
   - ⭕ Watermark for Draft status
   - ⭕ Footer with page numbers
   - ⭕ Print settings customization

---

## Notes

- Print functionality relies on browser's native print feature
- Use Ctrl+P or Cmd+P or Print button to open print dialog
- Print to PDF recommended for archiving
- Serial numbers displayed comma-separated for readability
- Amount in words follows Indian convention (Rupees and Paise Only)
- Company signature should be PNG with transparent background
- Recommended signature size: 150x60 pixels

## Dependencies

- **Phase 11:** Transit DC data structure
- **Phase 13:** Serial numbers to display
- **Phase 14:** Access control (view any status)
- **Mockup:** 14-transit-dc-detail.html for layout reference

## Next Steps

After Phase 15 completion:
1. Implement Official DC view (similar layout, no pricing)
2. Add PDF generation using library (e.g., wkhtmltopdf, pdfkit)
3. Add Excel export functionality
4. Implement email sending with DC as attachment
5. Add bulk print for multiple DCs
