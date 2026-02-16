# Phase 17: PDF & Excel Export

## Overview
Implement PDF and Excel export functionality for both Transit and Official Delivery Challans. PDF generation will render the existing HTML view templates and convert them to PDF format, ensuring layout consistency. Excel exports will match the specific sheet layouts: FSS-Transit-DC for transit DCs and Fervid-DC-V1 for official DCs. Exports will be available via download endpoints with proper file naming and content headers.

**Tech Stack:**
- Go + Gin backend
- chromedp for PDF generation (headless Chrome)
- excelize library for Excel generation
- SQLite database

## Prerequisites
- Phase 15 (Transit DC View) completed
- Phase 16 (Official DC View) completed
- Both DC view templates rendering correctly
- Understanding of headless browser automation
- Familiarity with Excel file structure

## Goals
1. Generate PDF files from Transit DC HTML view
2. Generate PDF files from Official DC HTML view
3. Generate Excel files matching FSS-Transit-DC layout
4. Generate Excel files matching Fervid-DC-V1 layout
5. Implement download endpoints with proper headers
6. Handle special characters in DC numbers for filenames
7. Optimize PDF rendering for quality and file size
8. Ensure Excel sheets have proper formatting and formulas

## Detailed Implementation Steps

### Step 1: Install Required Libraries
Install chromedp for PDF generation and excelize for Excel:
```bash
go get github.com/chromedp/chromedp
go get github.com/xuri/excelize/v2
```

### Step 2: PDF Service - Setup Chromedp
Create PDF service with chromedp context and configuration:
- Initialize chromedp allocator
- Configure page settings (A4, margins)
- Set up print options
- Handle browser lifecycle

### Step 3: PDF Generation - Transit DC
Implement Transit DC PDF generation:
- Render Transit DC HTML view
- Convert to PDF using chromedp
- Apply print CSS
- Optimize for file size

### Step 4: PDF Generation - Official DC
Implement Official DC PDF generation:
- Render Official DC HTML view
- Convert to PDF using chromedp
- Ensure signature images are included
- Handle multi-page documents

### Step 5: Excel Service - Setup Excelize
Create Excel service with excelize:
- Initialize workbook
- Set up cell styles (fonts, borders, colors)
- Create reusable formatting functions
- Handle formula generation

### Step 6: Excel Generation - Transit DC
Implement Transit DC Excel export:
- Match FSS-Transit-DC sheet layout exactly
- Include company header
- Product table with all columns (including pricing)
- Tax summary section
- Signature blocks
- Apply proper column widths and row heights

### Step 7: Excel Generation - Official DC
Implement Official DC Excel export:
- Match Fervid-DC-V1 sheet layout exactly
- Company header with legal details
- Product table without pricing
- Acknowledgement section
- Dual signature blocks
- Apply proper formatting

### Step 8: Download Endpoints
Create download handlers:
- Generate file with temporary storage
- Set Content-Disposition header
- Set proper Content-Type
- Stream file to response
- Clean up temporary files

### Step 9: Filename Sanitization
Implement DC number sanitization for filenames:
- Replace `/` with `-`
- Remove special characters
- Handle spaces
- Ensure valid filename across OS

### Step 10: Testing & Optimization
Test export functionality:
- Verify PDF layout matches HTML view
- Validate Excel formulas
- Test file downloads
- Check file sizes
- Test with various DC data

## Files to Create/Modify

### Backend Files

**services/pdf_service.go** (create new)
```go
package services

import (
    "context"
    "fmt"
    "io/ioutil"
    "os"
    "time"

    "github.com/chromedp/chromedp"
    "github.com/chromedp/cdproto/page"
)

type PDFService struct {
    tempDir string
}

func NewPDFService(tempDir string) *PDFService {
    return &PDFService{tempDir: tempDir}
}

// Generate PDF from HTML content
func (s *PDFService) GenerateFromHTML(html string, landscape bool) ([]byte, error) {
    // Create context
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()

    // Set timeout
    ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    var buf []byte

    // Define print parameters
    printParams := page.PrintToPDF()
    printParams.PrintBackground = true
    printParams.PaperWidth = 8.27   // A4 width in inches
    printParams.PaperHeight = 11.69 // A4 height in inches
    printParams.MarginTop = 0.4
    printParams.MarginBottom = 0.4
    printParams.MarginLeft = 0.4
    printParams.MarginRight = 0.4
    if landscape {
        printParams.Landscape = true
    }

    // Navigate and print
    err := chromedp.Run(ctx,
        chromedp.Navigate("data:text/html,"+html),
        chromedp.WaitReady("body"),
        chromedp.ActionFunc(func(ctx context.Context) error {
            var err error
            buf, _, err = printParams.Do(ctx)
            return err
        }),
    )

    if err != nil {
        return nil, fmt.Errorf("failed to generate PDF: %w", err)
    }

    return buf, nil
}

// Generate Transit DC PDF
func (s *PDFService) GenerateTransitDCPDF(dcID string, renderFunc func(string) (string, error)) ([]byte, error) {
    // Render HTML using the provided render function
    html, err := renderFunc(dcID)
    if err != nil {
        return nil, err
    }

    return s.GenerateFromHTML(html, false)
}

// Generate Official DC PDF
func (s *PDFService) GenerateOfficialDCPDF(dcID string, renderFunc func(string) (string, error)) ([]byte, error) {
    // Render HTML using the provided render function
    html, err := renderFunc(dcID)
    if err != nil {
        return nil, err
    }

    return s.GenerateFromHTML(html, false)
}

// Save PDF to temporary file
func (s *PDFService) SaveToTempFile(pdfData []byte) (string, error) {
    tmpFile, err := ioutil.TempFile(s.tempDir, "dc-*.pdf")
    if err != nil {
        return "", err
    }
    defer tmpFile.Close()

    _, err = tmpFile.Write(pdfData)
    if err != nil {
        return "", err
    }

    return tmpFile.Name(), nil
}

// Clean up temporary file
func (s *PDFService) CleanupTempFile(filepath string) error {
    return os.Remove(filepath)
}
```

**services/excel_service.go** (create new)
```go
package services

import (
    "fmt"
    "strings"
    "time"

    "github.com/xuri/excelize/v2"
)

type ExcelService struct{}

func NewExcelService() *ExcelService {
    return &ExcelService{}
}

// Generate Transit DC Excel (FSS-Transit-DC layout)
func (s *ExcelService) GenerateTransitDC(dc *DCDetail) (*excelize.File, error) {
    f := excelize.NewFile()
    sheet := "Transit DC"
    f.NewSheet(sheet)
    f.DeleteSheet("Sheet1")

    // Set column widths
    f.SetColWidth(sheet, "A", "A", 5)
    f.SetColWidth(sheet, "B", "B", 25)
    f.SetColWidth(sheet, "C", "C", 35)
    f.SetColWidth(sheet, "D", "D", 20)
    f.SetColWidth(sheet, "E", "F", 12)
    f.SetColWidth(sheet, "G", "H", 15)
    f.SetColWidth(sheet, "I", "J", 15)

    // Styles
    headerStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Size: 16},
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
    })

    titleStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Size: 14},
        Alignment: &excelize.Alignment{Horizontal: "center"},
    })

    boldStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true},
    })

    tableHeaderStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
        Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
        Border: s.createBorder(),
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
    })

    tableCellStyle, _ := f.NewStyle(&excelize.Style{
        Border: s.createBorder(),
        Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
    })

    currencyStyle, _ := f.NewStyle(&excelize.Style{
        Border: s.createBorder(),
        NumFmt: 4, // Currency format with 2 decimals
    })

    // Row 1-2: Company Header
    f.MergeCell(sheet, "A1", "J2")
    f.SetCellValue(sheet, "A1", "FERVID SMART SOLUTIONS PRIVATE LIMITED")
    f.SetCellStyle(sheet, "A1", "J2", headerStyle)
    f.SetRowHeight(sheet, 1, 30)

    // Row 3: Address
    f.MergeCell(sheet, "A3", "J3")
    f.SetCellValue(sheet, "A3", "Plot No 14/2, Dwaraka Park View, 1st Floor, Sector-1, HUDA Techno Enclave, Madhapur, Hyderabad, Telangana 500081")
    f.SetCellStyle(sheet, "A3", "J3", titleStyle)

    // Row 4: Contact Info
    f.MergeCell(sheet, "A4", "J4")
    f.SetCellValue(sheet, "A4", "Email: odishaprojects@fervidsmart.com | GSTIN: 36AACCF9742K1Z8 | CIN: U45100TG2016PTC113752")
    f.SetCellStyle(sheet, "A4", "J4", titleStyle)

    // Row 5: Empty
    f.SetRowHeight(sheet, 5, 10)

    // Row 6: DC Title
    f.MergeCell(sheet, "A6", "J6")
    f.SetCellValue(sheet, "A6", "TAX INVOICE / DELIVERY CHALLAN")
    f.SetCellStyle(sheet, "A6", "J6", titleStyle)
    f.SetRowHeight(sheet, 6, 25)

    // Row 7: Empty
    f.SetRowHeight(sheet, 7, 10)

    // Row 8-11: DC Details in 2 columns
    row := 8
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "DC Number:")
    f.SetCellValue(sheet, "B"+fmt.Sprint(row), dc.DCNumber)
    f.SetCellValue(sheet, "F"+fmt.Sprint(row), "DC Date:")
    f.SetCellValue(sheet, "G"+fmt.Sprint(row), dc.DCDate.Format("02/01/2006"))
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), boldStyle)
    f.SetCellStyle(sheet, "F"+fmt.Sprint(row), "F"+fmt.Sprint(row), boldStyle)

    row++
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Project:")
    f.SetCellValue(sheet, "B"+fmt.Sprint(row), dc.ProjectName)
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), boldStyle)

    row++
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "PO Number:")
    f.SetCellValue(sheet, "B"+fmt.Sprint(row), dc.PONumber)
    f.SetCellValue(sheet, "F"+fmt.Sprint(row), "PO Date:")
    f.SetCellValue(sheet, "G"+fmt.Sprint(row), dc.PODate.Format("02/01/2006"))
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), boldStyle)
    f.SetCellStyle(sheet, "F"+fmt.Sprint(row), "F"+fmt.Sprint(row), boldStyle)

    row += 2 // Skip to row 12

    // Bill To and Ship To
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "E"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Bill To:")
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "E"+fmt.Sprint(row), boldStyle)

    f.MergeCell(sheet, "F"+fmt.Sprint(row), "J"+fmt.Sprint(row))
    f.SetCellValue(sheet, "F"+fmt.Sprint(row), "Ship To:")
    f.SetCellStyle(sheet, "F"+fmt.Sprint(row), "J"+fmt.Sprint(row), boldStyle)

    row++
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "E"+fmt.Sprint(row+3))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), dc.BillToAddress)

    f.MergeCell(sheet, "F"+fmt.Sprint(row), "J"+fmt.Sprint(row+3))
    f.SetCellValue(sheet, "F"+fmt.Sprint(row), dc.ShipToAddress)

    row += 5 // Skip to row 18

    // Product Table Header
    headers := []string{"S.No", "Item Name", "Description", "Brand/Model", "HSN", "Qty", "Unit Price", "Taxable Value", "Tax Rate", "Total"}
    for i, header := range headers {
        col := string(rune('A' + i))
        f.SetCellValue(sheet, col+fmt.Sprint(row), header)
        f.SetCellStyle(sheet, col+fmt.Sprint(row), col+fmt.Sprint(row), tableHeaderStyle)
    }
    f.SetRowHeight(sheet, row, 30)

    row++ // Start products from row 19
    startRow := row

    // Product Rows
    for i, product := range dc.Products {
        f.SetCellValue(sheet, "A"+fmt.Sprint(row), i+1)
        f.SetCellValue(sheet, "B"+fmt.Sprint(row), product.ItemName)
        f.SetCellValue(sheet, "C"+fmt.Sprint(row), product.Description)
        f.SetCellValue(sheet, "D"+fmt.Sprint(row), product.BrandModel)
        f.SetCellValue(sheet, "E"+fmt.Sprint(row), product.HSNCode)
        f.SetCellValue(sheet, "F"+fmt.Sprint(row), product.Quantity)
        f.SetCellValue(sheet, "G"+fmt.Sprint(row), product.UnitPrice)

        // Formula for Taxable Value = Qty * Unit Price
        f.SetCellFormula(sheet, "H"+fmt.Sprint(row), fmt.Sprintf("F%d*G%d", row, row))

        f.SetCellValue(sheet, "I"+fmt.Sprint(row), product.TaxRate)

        // Formula for Total = Taxable Value * (1 + Tax Rate / 100)
        f.SetCellFormula(sheet, "J"+fmt.Sprint(row), fmt.Sprintf("H%d*(1+I%d/100)", row, row))

        // Apply styles
        for col := 'A'; col <= 'J'; col++ {
            cellStyle := tableCellStyle
            if col >= 'G' && col <= 'J' {
                cellStyle = currencyStyle
            }
            f.SetCellStyle(sheet, string(col)+fmt.Sprint(row), string(col)+fmt.Sprint(row), cellStyle)
        }

        row++
    }
    endRow := row - 1

    row++ // Empty row

    // Tax Summary Section
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "G"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Tax Summary:")
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "G"+fmt.Sprint(row), boldStyle)

    row++
    f.SetCellValue(sheet, "H"+fmt.Sprint(row), "Subtotal:")
    f.SetCellFormula(sheet, "J"+fmt.Sprint(row), fmt.Sprintf("SUM(H%d:H%d)", startRow, endRow))
    f.SetCellStyle(sheet, "H"+fmt.Sprint(row), "H"+fmt.Sprint(row), boldStyle)
    f.SetCellStyle(sheet, "J"+fmt.Sprint(row), "J"+fmt.Sprint(row), currencyStyle)

    row++
    f.SetCellValue(sheet, "H"+fmt.Sprint(row), "CGST:")
    f.SetCellFormula(sheet, "J"+fmt.Sprint(row), fmt.Sprintf("SUM(H%d:H%d)*I%d/200", startRow, endRow, startRow))
    f.SetCellStyle(sheet, "H"+fmt.Sprint(row), "H"+fmt.Sprint(row), boldStyle)
    f.SetCellStyle(sheet, "J"+fmt.Sprint(row), "J"+fmt.Sprint(row), currencyStyle)

    row++
    f.SetCellValue(sheet, "H"+fmt.Sprint(row), "SGST:")
    f.SetCellFormula(sheet, "J"+fmt.Sprint(row), fmt.Sprintf("SUM(H%d:H%d)*I%d/200", startRow, endRow, startRow))
    f.SetCellStyle(sheet, "H"+fmt.Sprint(row), "H"+fmt.Sprint(row), boldStyle)
    f.SetCellStyle(sheet, "J"+fmt.Sprint(row), "J"+fmt.Sprint(row), currencyStyle)

    row++
    f.SetCellValue(sheet, "H"+fmt.Sprint(row), "Grand Total:")
    f.SetCellFormula(sheet, "J"+fmt.Sprint(row), fmt.Sprintf("SUM(J%d:J%d)", startRow, endRow))
    f.SetCellStyle(sheet, "H"+fmt.Sprint(row), "H"+fmt.Sprint(row), boldStyle)
    f.SetCellStyle(sheet, "J"+fmt.Sprint(row), "J"+fmt.Sprint(row), currencyStyle)

    row += 2 // Empty rows

    // Signature Section
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "E"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "For FERVID SMART SOLUTIONS PRIVATE LIMITED")
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "E"+fmt.Sprint(row), boldStyle)

    row += 3
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "E"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Authorized Signatory")

    return f, nil
}

// Generate Official DC Excel (Fervid-DC-V1 layout)
func (s *ExcelService) GenerateOfficialDC(dc *DCDetail) (*excelize.File, error) {
    f := excelize.NewFile()
    sheet := "Official DC"
    f.NewSheet(sheet)
    f.DeleteSheet("Sheet1")

    // Set column widths
    f.SetColWidth(sheet, "A", "A", 5)
    f.SetColWidth(sheet, "B", "B", 20)
    f.SetColWidth(sheet, "C", "C", 30)
    f.SetColWidth(sheet, "D", "D", 20)
    f.SetColWidth(sheet, "E", "E", 10)
    f.SetColWidth(sheet, "F", "F", 25)
    f.SetColWidth(sheet, "G", "G", 20)

    // Styles (similar to transit, but adjusted)
    headerStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Size: 16},
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
        Border: s.createBorder(),
    })

    titleStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Size: 14},
        Alignment: &excelize.Alignment{Horizontal: "center"},
    })

    boldStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true},
    })

    tableHeaderStyle, _ := f.NewStyle(&excelize.Style{
        Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
        Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
        Border: s.createBorder(),
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
    })

    tableCellStyle, _ := f.NewStyle(&excelize.Style{
        Border: s.createBorder(),
        Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
    })

    // Row 1-2: Company Header with Border
    f.MergeCell(sheet, "A1", "G2")
    f.SetCellValue(sheet, "A1", "FERVID SMART SOLUTIONS PRIVATE LIMITED")
    f.SetCellStyle(sheet, "A1", "G2", headerStyle)
    f.SetRowHeight(sheet, 1, 30)

    // Row 3: Address
    f.MergeCell(sheet, "A3", "G3")
    f.SetCellValue(sheet, "A3", "Plot No 14/2, Dwaraka Park View, 1st Floor, Sector-1, HUDA Techno Enclave, Madhapur, Hyderabad, Telangana 500081")
    f.SetCellStyle(sheet, "A3", "G3", titleStyle)

    // Row 4: Contact Info
    f.MergeCell(sheet, "A4", "G4")
    f.SetCellValue(sheet, "A4", "Email: odishaprojects@fervidsmart.com | GSTIN: 36AACCF9742K1Z8 | CIN: U45100TG2016PTC113752")
    f.SetCellStyle(sheet, "A4", "G4", titleStyle)

    row := 6 // Start at row 6

    // DC Title
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "G"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "DELIVERY CHALLAN")
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "G"+fmt.Sprint(row), titleStyle)
    f.SetRowHeight(sheet, row, 25)

    row += 2

    // DC Details
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "DC Number:")
    f.SetCellValue(sheet, "B"+fmt.Sprint(row), dc.DCNumber)
    f.SetCellValue(sheet, "D"+fmt.Sprint(row), "DC Date:")
    f.SetCellValue(sheet, "E"+fmt.Sprint(row), dc.DCDate.Format("02/01/2006"))
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), boldStyle)
    f.SetCellStyle(sheet, "D"+fmt.Sprint(row), "D"+fmt.Sprint(row), boldStyle)

    row++
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Mandal/ULB:")
    f.SetCellValue(sheet, "B"+fmt.Sprint(row), dc.MandalName)
    f.SetCellValue(sheet, "D"+fmt.Sprint(row), "Mandal Code:")
    f.SetCellValue(sheet, "E"+fmt.Sprint(row), dc.MandalCode)
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), boldStyle)
    f.SetCellStyle(sheet, "D"+fmt.Sprint(row), "D"+fmt.Sprint(row), boldStyle)

    row += 2

    // Project Details
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "G"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Project Details")
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "G"+fmt.Sprint(row), boldStyle)

    row++
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Project:")
    f.MergeCell(sheet, "B"+fmt.Sprint(row), "G"+fmt.Sprint(row))
    f.SetCellValue(sheet, "B"+fmt.Sprint(row), dc.ProjectName)

    row++
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "PO Number:")
    f.SetCellValue(sheet, "B"+fmt.Sprint(row), dc.PONumber)
    f.SetCellValue(sheet, "D"+fmt.Sprint(row), "PO Date:")
    f.SetCellValue(sheet, "E"+fmt.Sprint(row), dc.PODate.Format("02/01/2006"))

    row += 2

    // Issued To
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Issued To:")
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "A"+fmt.Sprint(row), boldStyle)
    row++
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "G"+fmt.Sprint(row+2))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), dc.ShipToAddress)

    row += 4

    // Product Table Header (NO PRICING)
    headers := []string{"S.No", "Item Name", "Description", "Brand/Model No", "Quantity", "Serial Number", "Remarks"}
    for i, header := range headers {
        col := string(rune('A' + i))
        f.SetCellValue(sheet, col+fmt.Sprint(row), header)
        f.SetCellStyle(sheet, col+fmt.Sprint(row), col+fmt.Sprint(row), tableHeaderStyle)
    }
    f.SetRowHeight(sheet, row, 30)

    row++

    // Product Rows
    for i, product := range dc.Products {
        f.SetCellValue(sheet, "A"+fmt.Sprint(row), i+1)
        f.SetCellValue(sheet, "B"+fmt.Sprint(row), product.ItemName)
        f.SetCellValue(sheet, "C"+fmt.Sprint(row), product.Description)
        f.SetCellValue(sheet, "D"+fmt.Sprint(row), product.BrandModel)
        f.SetCellValue(sheet, "E"+fmt.Sprint(row), product.Quantity)
        f.SetCellValue(sheet, "F"+fmt.Sprint(row), product.SerialNumbers)
        f.SetCellValue(sheet, "G"+fmt.Sprint(row), product.Remarks)

        // Apply styles
        for col := 'A'; col <= 'G'; col++ {
            f.SetCellStyle(sheet, string(col)+fmt.Sprint(row), string(col)+fmt.Sprint(row), tableCellStyle)
        }

        row++
    }

    row += 2

    // Acknowledgement
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "G"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "It is certified that the material is received in good condition.")

    row += 2

    // Receipt Date
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Date of Receipt: _______________________")

    row += 3

    // Dual Signature Blocks
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "C"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "FSSPL Representative")
    f.SetCellStyle(sheet, "A"+fmt.Sprint(row), "C"+fmt.Sprint(row), boldStyle)

    f.MergeCell(sheet, "E"+fmt.Sprint(row), "G"+fmt.Sprint(row))
    f.SetCellValue(sheet, "E"+fmt.Sprint(row), "Department Official")
    f.SetCellStyle(sheet, "E"+fmt.Sprint(row), "G"+fmt.Sprint(row), boldStyle)

    row += 3 // Space for signature

    f.MergeCell(sheet, "A"+fmt.Sprint(row), "C"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Name: "+dc.CompanyRepName)
    row++
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "C"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Designation: "+dc.CompanyRepDesignation)
    row++
    f.MergeCell(sheet, "A"+fmt.Sprint(row), "C"+fmt.Sprint(row))
    f.SetCellValue(sheet, "A"+fmt.Sprint(row), "Mobile: "+dc.CompanyRepMobile)

    return f, nil
}

// Helper to create border style
func (s *ExcelService) createBorder() []excelize.Border {
    return []excelize.Border{
        {Type: "left", Color: "000000", Style: 1},
        {Type: "right", Color: "000000", Style: 1},
        {Type: "top", Color: "000000", Style: 1},
        {Type: "bottom", Color: "000000", Style: 1},
    }
}
```

**handlers/export_handler.go** (create new)
```go
package handlers

import (
    "fmt"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

type ExportHandler struct {
    pdfService   *PDFService
    excelService *ExcelService
    dcService    *DCService
}

func NewExportHandler(pdfService *PDFService, excelService *ExcelService, dcService *DCService) *ExportHandler {
    return &ExportHandler{
        pdfService:   pdfService,
        excelService: excelService,
        dcService:    dcService,
    }
}

// Export Transit DC as PDF
func (h *ExportHandler) ExportTransitDCPDF(c *gin.Context) {
    dcID := c.Param("id")

    // Get DC details
    dc, err := h.dcService.GetDCWithDetails(dcID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }

    if dc.Type != "transit" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Not a transit DC"})
        return
    }

    // Render function that returns HTML
    renderFunc := func(id string) (string, error) {
        // This would use your template rendering logic
        // For now, returning placeholder
        return h.renderTransitDCHTML(dc)
    }

    // Generate PDF
    pdfData, err := h.pdfService.GenerateTransitDCPDF(dcID, renderFunc)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
        return
    }

    // Sanitize filename
    filename := sanitizeFilename(dc.DCNumber) + ".pdf"

    // Set headers
    c.Header("Content-Type", "application/pdf")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
    c.Data(http.StatusOK, "application/pdf", pdfData)
}

// Export Official DC as PDF
func (h *ExportHandler) ExportOfficialDCPDF(c *gin.Context) {
    dcID := c.Param("id")

    // Get DC details
    dc, err := h.dcService.GetDCWithDetails(dcID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }

    if dc.Type != "official" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Not an official DC"})
        return
    }

    // Render function
    renderFunc := func(id string) (string, error) {
        return h.renderOfficialDCHTML(dc)
    }

    // Generate PDF
    pdfData, err := h.pdfService.GenerateOfficialDCPDF(dcID, renderFunc)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
        return
    }

    // Sanitize filename
    filename := sanitizeFilename(dc.DCNumber) + ".pdf"

    // Set headers
    c.Header("Content-Type", "application/pdf")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
    c.Data(http.StatusOK, "application/pdf", pdfData)
}

// Export Transit DC as Excel
func (h *ExportHandler) ExportTransitDCExcel(c *gin.Context) {
    dcID := c.Param("id")

    // Get DC details
    dc, err := h.dcService.GetDCWithDetails(dcID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }

    if dc.Type != "transit" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Not a transit DC"})
        return
    }

    // Generate Excel
    excelFile, err := h.excelService.GenerateTransitDC(dc)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel"})
        return
    }

    // Sanitize filename
    filename := sanitizeFilename(dc.DCNumber) + ".xlsx"

    // Set headers
    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

    // Write to response
    if err := excelFile.Write(c.Writer); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write Excel"})
        return
    }
}

// Export Official DC as Excel
func (h *ExportHandler) ExportOfficialDCExcel(c *gin.Context) {
    dcID := c.Param("id")

    // Get DC details
    dc, err := h.dcService.GetDCWithDetails(dcID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }

    if dc.Type != "official" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Not an official DC"})
        return
    }

    // Generate Excel
    excelFile, err := h.excelService.GenerateOfficialDC(dc)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel"})
        return
    }

    // Sanitize filename
    filename := sanitizeFilename(dc.DCNumber) + ".xlsx"

    // Set headers
    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

    // Write to response
    if err := excelFile.Write(c.Writer); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write Excel"})
        return
    }
}

// Sanitize DC number for filename
func sanitizeFilename(dcNumber string) string {
    // Replace / with -
    filename := strings.ReplaceAll(dcNumber, "/", "-")

    // Remove other special characters
    filename = strings.ReplaceAll(filename, " ", "_")
    filename = strings.ReplaceAll(filename, "\\", "-")

    // Prefix with DC_
    return "DC_" + filename
}

// Helper methods to render HTML (implement based on your template engine)
func (h *ExportHandler) renderTransitDCHTML(dc *DCDetail) (string, error) {
    // Use your template engine to render transit-dc-detail.html
    // Return the HTML string
    return "", nil
}

func (h *ExportHandler) renderOfficialDCHTML(dc *DCDetail) (string, error) {
    // Use your template engine to render official-dc-detail.html
    // Return the HTML string
    return "", nil
}
```

### Frontend Files

**templates/transit-dc-detail.html** (modify - add export buttons)
```html
<!-- Add export buttons section after back button -->
<div class="max-w-4xl mx-auto px-4 py-4 print:hidden flex gap-4">
    <a href="/dcs/{{ .DC.ID }}/pdf"
       class="bg-red-600 text-white px-6 py-2 rounded-lg hover:bg-red-700">
        📄 Download PDF
    </a>
    <a href="/dcs/{{ .DC.ID }}/excel"
       class="bg-green-600 text-white px-6 py-2 rounded-lg hover:bg-green-700">
        📊 Download Excel
    </a>
    <button onclick="window.print()"
            class="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700">
        🖨️ Print DC
    </button>
</div>
```

**templates/official-dc-detail.html** (modify - add export buttons)
```html
<!-- Add export buttons section after back button -->
<div class="max-w-4xl mx-auto px-4 py-4 print:hidden flex gap-4">
    <a href="/dcs/{{ .DC.ID }}/pdf"
       class="bg-red-600 text-white px-6 py-2 rounded-lg hover:bg-red-700">
        📄 Download PDF
    </a>
    <a href="/dcs/{{ .DC.ID }}/excel"
       class="bg-green-600 text-white px-6 py-2 rounded-lg hover:bg-green-700">
        📊 Download Excel
    </a>
    <button onclick="window.print()"
            class="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700">
        🖨️ Print DC
    </button>
</div>
```

## API Routes/Endpoints

### Route Definitions

**main.go** (modify)
```go
// Export routes
exportHandler := handlers.NewExportHandler(pdfService, excelService, dcService)

dcGroup := r.Group("/dcs")
{
    // PDF exports
    dcGroup.GET("/:id/pdf", exportHandler.ExportPDF) // Auto-detect type

    // Excel exports
    dcGroup.GET("/:id/excel", exportHandler.ExportExcel) // Auto-detect type
}
```

### Endpoint Details

| Method | Endpoint | Description | Response Type | Filename Pattern |
|--------|----------|-------------|---------------|------------------|
| GET | `/dcs/:id/pdf` | Export DC as PDF (auto-detect type) | application/pdf | DC_FSS-24-25-001.pdf |
| GET | `/dcs/:id/excel` | Export DC as Excel (auto-detect type) | application/vnd...sheet | DC_FSS-24-25-001.xlsx |

### Response Headers
```
Content-Type: application/pdf OR application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename=DC_FSS-24-25-001.pdf
```

## Database Queries

No new database queries required. Uses existing DC retrieval queries from Phase 15 and 16.

## UI Components

### Export Button Group Component
Located on both Transit and Official DC detail pages:
- PDF Download button (red)
- Excel Download button (green)
- Print button (blue)
- Positioned above the DC document
- Hidden on print

### Component Styling
```html
<div class="flex gap-4 print:hidden">
    <a href="/dcs/{{ .DC.ID }}/pdf"
       class="bg-red-600 text-white px-6 py-2 rounded-lg hover:bg-red-700 flex items-center gap-2">
        <svg>...</svg> Download PDF
    </a>
    <a href="/dcs/{{ .DC.ID }}/excel"
       class="bg-green-600 text-white px-6 py-2 rounded-lg hover:bg-green-700 flex items-center gap-2">
        <svg>...</svg> Download Excel
    </a>
    <button onclick="window.print()"
            class="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 flex items-center gap-2">
        <svg>...</svg> Print
    </button>
</div>
```

## Testing Checklist

### PDF Generation Testing
- [ ] Transit DC PDF generation works
- [ ] Official DC PDF generation works
- [ ] PDF layout matches HTML view
- [ ] PDF is A4 size with correct margins
- [ ] All text is readable in PDF
- [ ] Images (signatures) appear in PDF
- [ ] Tables render correctly in PDF
- [ ] Page breaks are appropriate
- [ ] PDF file size is reasonable (< 1MB for typical DC)
- [ ] PDF downloads with correct filename
- [ ] Special characters in DC number handled in filename

### Excel Generation Testing
- [ ] Transit DC Excel generation works
- [ ] Official DC Excel generation works
- [ ] Transit Excel matches FSS-Transit-DC layout
- [ ] Official Excel matches Fervid-DC-V1 layout
- [ ] All columns have correct headers
- [ ] Data populates correctly in cells
- [ ] Formulas calculate correctly (Transit DC)
- [ ] Tax summary calculations are accurate
- [ ] Column widths are appropriate
- [ ] Cell borders and styling appear correctly
- [ ] Header styling (bold, colors) applied
- [ ] Excel file opens without errors
- [ ] Excel downloads with correct filename

### Filename Sanitization Testing
- [ ] DC number with `/` replaced with `-`
- [ ] DC number with spaces replaced with `_`
- [ ] DC number with special characters removed
- [ ] Filename is valid across Windows, Mac, Linux
- [ ] Filename pattern: DC_<sanitized_number>.pdf/xlsx

### Download Testing
- [ ] PDF downloads trigger browser download
- [ ] Excel downloads trigger browser download
- [ ] Content-Disposition header is correct
- [ ] Content-Type header is correct
- [ ] Files open correctly after download
- [ ] No corruption in downloaded files

### Performance Testing
- [ ] PDF generation completes in < 5 seconds
- [ ] Excel generation completes in < 2 seconds
- [ ] Large DCs (20+ products) generate successfully
- [ ] Multiple concurrent exports don't crash server
- [ ] Temporary files are cleaned up properly

### Cross-Browser Testing
- [ ] PDF download works in Chrome
- [ ] PDF download works in Firefox
- [ ] PDF download works in Safari
- [ ] PDF download works in Edge
- [ ] Excel download works across browsers

### Error Handling
- [ ] Invalid DC ID returns 404
- [ ] Wrong DC type handled gracefully
- [ ] PDF generation errors handled
- [ ] Excel generation errors handled
- [ ] Network errors handled
- [ ] User-friendly error messages

## Acceptance Criteria

### Must Have
1. ✅ PDF generation for Transit DC matching HTML view layout
2. ✅ PDF generation for Official DC matching HTML view layout
3. ✅ Excel generation for Transit DC matching FSS-Transit-DC sheet
4. ✅ Excel generation for Official DC matching Fervid-DC-V1 sheet
5. ✅ PDF download endpoint: GET /dcs/:id/pdf
6. ✅ Excel download endpoint: GET /dcs/:id/excel
7. ✅ Proper Content-Disposition headers for downloads
8. ✅ Proper Content-Type headers (application/pdf and application/vnd...sheet)
9. ✅ Filename sanitization: replace `/` with `-` in DC numbers
10. ✅ Filename pattern: DC_<sanitized_number>.pdf or .xlsx
11. ✅ Transit Excel includes: company header, product table with pricing, tax summary, signature block
12. ✅ Official Excel includes: company header, product table without pricing, acknowledgement, dual signatures
13. ✅ Excel formulas calculate correctly (Transit DC: taxable value, total, tax summary)
14. ✅ Export buttons visible on DC detail pages
15. ✅ Export buttons hidden on print

### Nice to Have
1. ⭐ Batch export (multiple DCs at once)
2. ⭐ Email DC as attachment
3. ⭐ Custom watermarks for draft DCs
4. ⭐ PDF/Excel templates customization
5. ⭐ Export history/log
6. ⭐ Progress indicator for large exports
7. ⭐ Zip multiple exports together
8. ⭐ Export with QR code for verification

### Performance Criteria
- PDF generation < 5 seconds
- Excel generation < 2 seconds
- File size: PDF < 1MB, Excel < 500KB (typical DC)
- Handle DCs with up to 50 products
- Support concurrent exports (5+ users)

### Quality Criteria
- PDF text is selectable (not images)
- Excel cells are editable
- Formulas in Excel are preserved
- Consistent styling across exports
- No data loss or corruption

### Compatibility Criteria
- PDF opens in Adobe Reader, Preview, Chrome
- Excel opens in Microsoft Excel, Google Sheets, LibreOffice
- Filenames valid on Windows, Mac, Linux
- Works across modern browsers

---

## Notes
- chromedp recommended over wkhtmltopdf (pure Go, no external dependencies)
- excelize is the most popular Go library for Excel generation
- Consider caching rendered HTML for faster PDF generation
- Temporary files should be cleaned up after download
- For very large DCs, consider async generation with download link
- Future: store generated PDFs for faster re-download
