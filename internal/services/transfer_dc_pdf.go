package services

import (
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// --- Transfer DC PDF Data Structs ---

// TransferDCPDFData holds all data needed to generate a Transfer DC PDF.
type TransferDCPDFData struct {
	Project    *models.Project
	DC         *models.DeliveryChallan
	TransferDC *models.TransferDC
	Company    *models.CompanySettings
	LineItems  []models.DCLineItem

	// Addresses
	HubAddress          *models.Address
	BillFromAddress     *models.Address
	DispatchFromAddress *models.Address
	BillToAddress       *models.Address
	HubConfig           *models.AddressListConfig
	BillFromConfig      *models.AddressListConfig
	DispatchFromConfig  *models.AddressListConfig
	BillToConfig        *models.AddressListConfig
	ShipToConfig        *models.AddressListConfig

	// Destinations (for quantity breakdown table)
	Destinations []TransferDCPDFDestination
	Products     []TransferDCPDFProduct

	// Financial totals
	TotalTaxable  float64
	TotalTax      float64
	GrandTotal    float64
	RoundedTotal  float64
	RoundOff      float64
	HalfTax       float64
	TotalQty      int
	AmountInWords string
}

// TransferDCPDFDestination represents a destination row in the breakdown table.
type TransferDCPDFDestination struct {
	Name       string
	Address    string
	FullAddr   *models.Address          // full address for filtered rendering
	Quantities map[int]int              // productID → qty
}

// TransferDCPDFProduct identifies a product column in the destination grid.
type TransferDCPDFProduct struct {
	ID   int
	Name string
}

// GenerateTransferDCPDF produces a PDF for a Split Transfer Delivery Challan.
func GenerateTransferDCPDF(data *TransferDCPDFData) ([]byte, error) {
	pdf := newPDF(&pdfHeaderConfig{Project: data.Project, Company: data.Company})

	drawDCTitle(pdf, "Split Transfer Delivery Challan", false)
	drawTransferDCDetailsGrid(pdf, data)
	drawTransferAddressGrid(pdf, data)
	drawTransferProductTable(pdf, data.LineItems, data.TotalQty, data.TotalTaxable, data.TotalTax, data.GrandTotal)
	drawTaxSummary(pdf, data.TotalTaxable, data.HalfTax, data.RoundOff, data.RoundedTotal)
	drawAmountInWords(pdf, data.AmountInWords)

	if data.Project != nil && data.Project.Notes != "" {
		drawNotes(pdf, data.Project.Notes)
	}

	drawTransitSignatures(pdf, data.Company, data.Project)

	if len(data.Destinations) > 0 && len(data.Products) > 0 {
		pdf.AddPage()
		drawAnnexureDestinationBreakdown(pdf, data.Destinations, data.Products, data.ShipToConfig)
	}

	drawSerialNumbersAnnexure(pdf, data.LineItems, 2)

	return pdfToBytes(pdf)
}

// --- Transfer DC specific drawing functions ---

func drawTransferDCDetailsGrid(pdf *fpdf.Fpdf, data *TransferDCPDFData) {
	y := pdf.GetY()
	colW := contentW/2 - 2
	gap := 4.0
	innerW := colW - 2*cellPad

	// Count lines for left box (DC details)
	leftLines := 1 // DC No (always)
	if data.DC != nil && data.DC.ChallanDate != nil {
		leftLines++
	}
	if data.TransferDC != nil {
		if data.TransferDC.TransporterName != "" {
			leftLines++
		}
		if data.TransferDC.VehicleNumber != "" {
			leftLines++
		}
		if data.TransferDC.EwayBillNumber != "" {
			leftLines++
		}
		if data.TransferDC.DocketNumber != "" {
			leftLines++
		}
	}
	leftLines++ // Reverse Charge

	// Count lines for right box (PO details)
	rightLines := 0
	if data.Project != nil {
		if data.Project.POReference != "" {
			rightLines++
		}
		if data.Project.PODate != nil {
			rightLines++
		}
		rightLines++ // Project name
	}

	maxLines := leftLines
	if rightLines > maxLines {
		maxLines = rightLines
	}
	boxH := float64(maxLines)*lineH + 2*cellPad

	// Left box: DC details
	dcNumber := ""
	if data.DC != nil {
		dcNumber = data.DC.DCNumber
	}
	drawBorderedRect(pdf, marginL, y, colW, boxH)
	pdf.SetXY(marginL+cellPad, y+cellPad)
	kvRow(pdf, "DC No:", dcNumber, 24, innerW)

	if data.DC != nil && data.DC.ChallanDate != nil {
		pdf.SetX(marginL + cellPad)
		kvRow(pdf, "Date:", *data.DC.ChallanDate, 24, innerW)
	}
	if data.TransferDC != nil {
		if data.TransferDC.TransporterName != "" {
			pdf.SetX(marginL + cellPad)
			kvRow(pdf, "Transporter:", data.TransferDC.TransporterName, 24, innerW)
		}
		if data.TransferDC.VehicleNumber != "" {
			pdf.SetX(marginL + cellPad)
			kvRow(pdf, "Vehicle:", data.TransferDC.VehicleNumber, 24, innerW)
		}
		if data.TransferDC.EwayBillNumber != "" {
			pdf.SetX(marginL + cellPad)
			kvRow(pdf, "E-Way Bill:", data.TransferDC.EwayBillNumber, 24, innerW)
		}
		if data.TransferDC.DocketNumber != "" {
			pdf.SetX(marginL + cellPad)
			kvRow(pdf, "Docket:", data.TransferDC.DocketNumber, 24, innerW)
		}
	}
	reverseCharge := "No"
	if data.TransferDC != nil && data.TransferDC.ReverseCharge != "" {
		reverseCharge = data.TransferDC.ReverseCharge
	}
	pdf.SetX(marginL + cellPad)
	kvRow(pdf, "Reverse Charge:", reverseCharge, 30, innerW)

	// Right box: PO details
	rightX := marginL + colW + gap
	drawBorderedRect(pdf, rightX, y, colW, boxH)
	pdf.SetXY(rightX+cellPad, y+cellPad)

	if data.Project != nil {
		if data.Project.POReference != "" {
			pdf.SetX(rightX + cellPad)
			kvRow(pdf, "PO Number:", data.Project.POReference, 22, innerW)
		}
		if data.Project.PODate != nil {
			pdf.SetX(rightX + cellPad)
			kvRow(pdf, "PO Date:", *data.Project.PODate, 22, innerW)
		}
		pdf.SetX(rightX + cellPad)
		kvRow(pdf, "Project:", data.Project.Name, 22, innerW)
	}

	pdf.SetY(y + boxH + 2)
}

func drawTransferAddressGrid(pdf *fpdf.Fpdf, data *TransferDCPDFData) {
	colW := contentW/2 - 2
	gap := 4.0
	y := pdf.GetY()

	// Row 1: Bill From / Bill To
	billFromLines := addressLinesFiltered(data.BillFromAddress, data.BillFromConfig)
	if len(billFromLines) == 0 {
		billFromLines = companyAddressLines(data.Company)
	}
	h1 := drawAddressBox(pdf, marginL, y, colW, "Bill From", billFromLines)
	h2 := drawAddressBox(pdf, marginL+colW+gap, y, colW, "Bill To", addressLinesFiltered(data.BillToAddress, data.BillToConfig))
	rowH := h1
	if h2 > rowH {
		rowH = h2
	}
	pdf.SetY(y + rowH + 1)

	y = pdf.GetY()
	// Row 2: Dispatch From / Hub Location
	dispatchFromLines := addressLinesFiltered(data.DispatchFromAddress, data.DispatchFromConfig)
	if len(dispatchFromLines) == 0 {
		dispatchFromLines = companyAddressLines(data.Company)
	}
	h1 = drawAddressBox(pdf, marginL, y, colW, "Dispatch From", dispatchFromLines)
	hubLines := addressLinesFiltered(data.HubAddress, data.HubConfig)
	h2 = drawAddressBox(pdf, marginL+colW+gap, y, colW, "Ship To", hubLines)
	rowH = h1
	if h2 > rowH {
		rowH = h2
	}
	pdf.SetY(y + rowH + 2)
}

func drawTransferProductTable(pdf *fpdf.Fpdf, items []models.DCLineItem, totalQty int, totalTaxable, totalTax, grandTotal float64) {
	cols := []tableCol{
		{"S.No", 9, "C"},
		{"Description", 70, "L"},
		{"UoM", 9, "C"},
		{"HSN", 14, "C"},
		{"Qty", 9, "C"},
		{"Rate", 19, "R"},
		{"Taxable", 20, "R"},
		{"GST %", 10, "C"},
		{"GST", 18, "R"},
		{"Total", 18, "R"},
	}

	drawTableHeaderRow(pdf, cols)

	for i, li := range items {
		desc := li.ItemName
		if li.BrandModel != "" {
			desc += "\n" + li.BrandModel
		}
		if li.ItemDescription != "" {
			desc += "\n" + li.ItemDescription
		}

		values := []string{
			fmt.Sprintf("%d", i+1),
			desc,
			li.UoM,
			li.HSNCode,
			fmt.Sprintf("%d", li.Quantity),
			"Rs." + fmtINR(li.Rate),
			"Rs." + fmtINR(li.TaxableAmount),
			fmt.Sprintf("%.0f%%", li.TaxPercentage),
			"Rs." + fmtINR(li.TaxAmount),
			"Rs." + fmtINR(li.TotalAmount),
		}

		drawTableDataRow(pdf, cols, values, false)
	}

	// Totals row
	totals := []string{
		"", "Total", "", "",
		fmt.Sprintf("%d", totalQty),
		"", "Rs." + fmtINR(totalTaxable), "",
		"Rs." + fmtINR(totalTax),
		"Rs." + fmtINR(grandTotal),
	}
	drawTableDataRow(pdf, cols, totals, true)
	spacer(pdf, 4)
}

func drawAnnexureDestinationBreakdown(pdf *fpdf.Fpdf, destinations []TransferDCPDFDestination, products []TransferDCPDFProduct, shipToConfig *models.AddressListConfig) {
	// Annexure title — centered, larger font
	setFont(pdf, "B", 11)
	setColor(pdf, colorBlack)
	pdf.CellFormat(contentW, 6, "Annexure - 1", "", 1, "C", false, 0, "")
	spacer(pdf, 3)

	// Subtitle
	setFont(pdf, "B", 9)
	setColor(pdf, colorBlack)
	pdf.CellFormat(contentW, lineH+1, "DESTINATION BREAKDOWN", "", 1, "C", false, 0, "")
	spacer(pdf, 3)

	// Build columns: #, Destination, then one column per product
	snoW := 8.0
	productColW := 25.0
	if len(products) > 0 {
		productColW = (tblW - snoW - 60) / float64(len(products))
		if productColW > 30 {
			productColW = 30
		}
	}
	// Expand Destination column to fill remaining width so table uses full tblW
	totalProductW := productColW * float64(len(products))
	destW := tblW - snoW - totalProductW
	if destW < 40 {
		destW = 40
	}

	cols := []tableCol{
		{"#", snoW, "C"},
		{"Destination", destW, "L"},
	}
	for _, p := range products {
		cols = append(cols, tableCol{p.Name, productColW, "C"})
	}

	// Draw header with text wrapping for product name columns
	drawWrappingHeaderRow(pdf, cols)

	// Destination rows
	totals := make(map[int]int)
	for i, dest := range destinations {
		destLabel := dest.Name
		if dest.FullAddr != nil {
			lines := addressLinesFiltered(dest.FullAddr, shipToConfig)
			if len(lines) > 0 {
				destLabel = strings.Join(lines, ", ")
			}
		}
		values := []string{
			fmt.Sprintf("%d", i+1),
			destLabel,
		}
		for _, p := range products {
			qty := dest.Quantities[p.ID]
			totals[p.ID] += qty
			values = append(values, fmt.Sprintf("%d", qty))
		}
		drawTableDataRow(pdf, cols, values, false)
	}

	// Totals row
	totalValues := []string{"", "TOTAL"}
	for _, p := range products {
		totalValues = append(totalValues, fmt.Sprintf("%d", totals[p.ID]))
	}
	drawTableDataRow(pdf, cols, totalValues, true)
	spacer(pdf, 4)
}

// drawWrappingHeaderRow draws a table header where text wraps within each column.
// Unlike drawTableHeaderRow which uses single-line CellFormat, this uses MultiCell
// measurement to handle long product names that overflow narrow columns.
func drawWrappingHeaderRow(pdf *fpdf.Fpdf, cols []tableCol) {
	headerFontSize := 6.0
	headerLineH := 3.5
	setFont(pdf, "B", headerFontSize)
	setColor(pdf, colorBlack)
	setFillColor(pdf, colorWhite)
	setDrawColor(pdf, colorBlack)
	pdf.SetLineWidth(0.3)

	// Measure the max number of wrapped lines across all columns
	maxLines := 1
	wrappedTexts := make([][]string, len(cols))
	for i, col := range cols {
		txt := strings.ToUpper(col.Header)
		availW := col.Width - 2*cellPad
		if availW < 1 {
			availW = 1
		}
		lines := pdf.SplitText(txt, availW)
		if len(lines) == 0 {
			lines = []string{txt}
		}
		wrappedTexts[i] = lines
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}

	h := float64(maxLines)*headerLineH + 2 // padding top+bottom
	y := pdf.GetY()
	x := tblMarginL

	for i, col := range cols {
		// Draw filled cell border
		pdf.Rect(x, y, col.Width, h, "FD")

		availW := col.Width - 2*cellPad
		if availW < 1 {
			availW = 1
		}

		lines := wrappedTexts[i]
		// Vertically center the block of lines within the header cell
		blockH := float64(len(lines)) * headerLineH
		startY := y + (h-blockH)/2

		for j, line := range lines {
			pdf.SetXY(x+cellPad, startY+float64(j)*headerLineH)
			pdf.CellFormat(availW, headerLineH, line, "", 0, "C", false, 0, "")
		}

		x += col.Width
	}
	pdf.SetY(y + h)
}

