package services

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

//go:embed fonts/DejaVuSans.ttf
var dejaVuSansRegular []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var dejaVuSansBold []byte

// --- Color constants (Tailwind gray scale RGB equivalents) ---

type rgb struct{ R, G, B int }

var (
	colorGray900 = rgb{17, 24, 39}    // #111827
	colorGray800 = rgb{31, 41, 55}    // #1f2937
	colorGray700 = rgb{55, 65, 81}    // #374151
	colorGray600 = rgb{75, 85, 99}    // #4b5563
	colorGray500 = rgb{107, 114, 128} // #6b7280
	colorGray400 = rgb{156, 163, 175} // #9ca3af
	colorGray300 = rgb{209, 213, 219} // #d1d5db
	colorSlate300 = rgb{203, 213, 225} // #cbd5e1
	colorSlate100 = rgb{241, 245, 249} // #f1f5f9
	colorSlate50  = rgb{248, 250, 252} // #f8fafc
	colorBlack    = rgb{0, 0, 0}
	colorWhite    = rgb{255, 255, 255}
)

// --- Page constants ---
const (
	pageW       = 210.0 // A4 width mm
	pageH       = 297.0 // A4 height mm
	marginL     = 10.0
	marginR     = 10.0
	marginT     = 10.0
	marginB     = 10.0
	contentW    = pageW - marginL - marginR // 190mm
	lineH       = 4.5                       // default line height mm
	cellPad     = 1.5                       // cell padding mm
	tblMarginL  = 7.0                       // table uses tighter margins
	tblW        = pageW - 2*tblMarginL      // 196mm
)

// --- Data structs ---

// TransitDCPDFData holds all data needed to generate a Transit DC PDF.
type TransitDCPDFData struct {
	Project           *models.Project
	DC                *models.DeliveryChallan
	TransitDetails    *models.DCTransitDetails
	LineItems         []models.DCLineItem
	Company           *models.CompanySettings
	ShipToAddress        *models.Address
	BillToAddress        *models.Address
	BillFromAddress      *models.Address
	DispatchFromAddress  *models.Address
	ShipToConfig         *models.AddressListConfig // optional: for print column filtering
	BillToConfig         *models.AddressListConfig // optional: for print column filtering
	BillFromConfig       *models.AddressListConfig // optional: for print column filtering
	DispatchFromConfig   *models.AddressListConfig // optional: for print column filtering
	TotalTaxable         float64
	TotalTax          float64
	GrandTotal        float64
	RoundedTotal      float64
	RoundOff          float64
	HalfTax           float64
	TotalQty          int
	AmountInWords     string
}

// OfficialDCPDFData holds all data needed to generate an Official DC PDF.
type OfficialDCPDFData struct {
	Project         *models.Project
	DC              *models.DeliveryChallan
	LineItems       []models.DCLineItem
	Company         *models.CompanySettings
	ShipToAddress   *models.Address
	BillToAddress   *models.Address
	BillFromAddress *models.Address
	ShipToConfig    *models.AddressListConfig // optional: for print column filtering
	BillToConfig    *models.AddressListConfig // optional: for print column filtering
	TotalQty        int
}

// --- Table column definition ---

type tableCol struct {
	Header string
	Width  float64
	Align  string // "L", "C", "R"
}

// --- Public entry points ---

// GenerateTransitDCPDF produces a PDF for a Transit Delivery Challan.
func GenerateTransitDCPDF(data *TransitDCPDFData) ([]byte, error) {
	pdf := newPDF()

	drawCompanyHeader(pdf, data.Company, data.BillFromAddress, false, true)
	drawDCTitle(pdf, "Delivery Challan", false)
	drawDCAndPOGrid(pdf, data.DC, data.TransitDetails, data.Project)
	drawTransitAddressGrid(pdf, data.Company, data.BillFromAddress, data.DispatchFromAddress, data.BillToAddress, data.ShipToAddress, data.BillFromConfig, data.DispatchFromConfig, data.BillToConfig, data.ShipToConfig)
	drawTransitProductTable(pdf, data.LineItems, data.TotalQty, data.TotalTaxable, data.TotalTax, data.GrandTotal)
	drawTaxSummary(pdf, data.TotalTaxable, data.HalfTax, data.RoundOff, data.RoundedTotal)
	drawAmountInWords(pdf, data.AmountInWords)

	if data.TransitDetails != nil && data.TransitDetails.Notes != "" {
		drawNotes(pdf, data.TransitDetails.Notes)
	}

	drawTransitSignatures(pdf, data.Company, data.Project)

	return pdfToBytes(pdf)
}

// GenerateOfficialDCPDF produces a PDF for an Official Delivery Challan.
func GenerateOfficialDCPDF(data *OfficialDCPDFData) ([]byte, error) {
	pdf := newPDF()

	drawCompanyHeader(pdf, data.Company, data.BillFromAddress, true, true)
	drawDCTitle(pdf, "Delivery Challan", true)
	drawCopyIndicators(pdf)
	drawOfficialMetaGrid(pdf, data.DC, data.ShipToAddress)
	drawReferenceInfo(pdf, data.Project)

	if data.Project != nil && data.Project.PurposeText != "" {
		drawPurpose(pdf, data.Project.PurposeText)
	}

	drawIssuedTo(pdf, data.ShipToAddress)
	drawAddressPair(pdf, data.BillToAddress, data.ShipToAddress, data.BillToConfig, data.ShipToConfig)
	drawOfficialProductTable(pdf, data.LineItems)
	drawAcknowledgement(pdf)
	drawOfficialSignatures(pdf, data.Project)

	return pdfToBytes(pdf)
}

// --- PDF creation and output ---

func newPDF() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginL, marginT, marginR)
	pdf.SetAutoPageBreak(true, marginB)

	// Register embedded UTF-8 fonts
	pdf.AddUTF8FontFromBytes("dejavu", "", dejaVuSansRegular)
	pdf.AddUTF8FontFromBytes("dejavu", "B", dejaVuSansBold)

	pdf.AddPage()
	return pdf
}

func pdfToBytes(pdf *fpdf.Fpdf) ([]byte, error) {
	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("pdf generation error: %w", err)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output error: %w", err)
	}
	return buf.Bytes(), nil
}

// --- Shared drawing helpers ---

func setFont(pdf *fpdf.Fpdf, style string, size float64) {
	pdf.SetFont("dejavu", style, size)
}

func setColor(pdf *fpdf.Fpdf, c rgb) {
	pdf.SetTextColor(c.R, c.G, c.B)
}

func setDrawColor(pdf *fpdf.Fpdf, c rgb) {
	pdf.SetDrawColor(c.R, c.G, c.B)
}

func setFillColor(pdf *fpdf.Fpdf, c rgb) {
	pdf.SetFillColor(c.R, c.G, c.B)
}

func drawHLine(pdf *fpdf.Fpdf, y float64, c rgb, thickness float64) {
	setDrawColor(pdf, c)
	pdf.SetLineWidth(thickness)
	pdf.Line(marginL, y, pageW-marginR, y)
}

func spacer(pdf *fpdf.Fpdf, h float64) {
	pdf.Ln(h)
}

// drawBorderedRect draws a rectangle with a light gray border.
func drawBorderedRect(pdf *fpdf.Fpdf, x, y, w, h float64) {
	setDrawColor(pdf, colorSlate300)
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, w, h, "D")
}

// drawFilledRect draws a filled rectangle.
func drawFilledRect(pdf *fpdf.Fpdf, x, y, w, h float64, fill rgb) {
	setFillColor(pdf, fill)
	setDrawColor(pdf, colorSlate300)
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, w, h, "FD")
}

// kvRow draws a label: value pair on the current line within a given total width.
func kvRow(pdf *fpdf.Fpdf, label, value string, labelW, totalW float64) {
	y := pdf.GetY()
	x := pdf.GetX()

	setFont(pdf, "B", 8)
	setColor(pdf, colorGray500)
	pdf.SetXY(x, y)
	pdf.CellFormat(labelW, lineH, label, "", 0, "L", false, 0, "")

	setFont(pdf, "", 8)
	setColor(pdf, colorGray800)
	valueW := totalW - labelW
	if valueW < 1 {
		valueW = 1
	}
	pdf.CellFormat(valueW, lineH, value, "", 1, "R", false, 0, "")
}

// kvRowLeft draws a label: value pair left-aligned.
func kvRowLeft(pdf *fpdf.Fpdf, label, value string) {
	setFont(pdf, "B", 8)
	setColor(pdf, colorGray500)
	pdf.CellFormat(0, lineH, label+" "+value, "", 1, "L", false, 0, "")
}

// --- Formatting helpers ---

func fmtINR(amount float64) string {
	isNeg := amount < 0
	amount = math.Abs(amount)
	whole := int64(amount)
	decimal := int64(math.Round((amount - float64(whole)) * 100))

	s := fmt.Sprintf("%d", whole)
	// Indian grouping: last 3 digits, then groups of 2
	if len(s) > 3 {
		result := s[len(s)-3:]
		s = s[:len(s)-3]
		for len(s) > 2 {
			result = s[len(s)-2:] + "," + result
			s = s[:len(s)-2]
		}
		if len(s) > 0 {
			s = s + "," + result
		} else {
			s = result
		}
	}

	formatted := fmt.Sprintf("%s.%02d", s, decimal)
	if isNeg {
		return "-" + formatted
	}
	return formatted
}

// fixedFieldMap maps fixed column names to their Address struct field accessors.
var fixedFieldMap = map[string]func(*models.Address) string{
	"District Name":   func(a *models.Address) string { return a.DistrictName },
	"Mandal/ULB Name": func(a *models.Address) string { return a.MandalName },
	"Mandal Code":     func(a *models.Address) string { return a.MandalCode },
}

func addressLines(addr *models.Address) []string {
	if addr == nil {
		return nil
	}
	var lines []string
	// Include fixed fields first
	for _, getter := range []func(*models.Address) string{
		fixedFieldMap["District Name"],
		fixedFieldMap["Mandal/ULB Name"],
		fixedFieldMap["Mandal Code"],
	} {
		v := strings.TrimSpace(getter(addr))
		if v != "" {
			lines = append(lines, v)
		}
	}
	// Then dynamic data
	for _, v := range addr.Data {
		v = strings.TrimSpace(v)
		if v != "" {
			lines = append(lines, v)
		}
	}
	return lines
}

// addressLinesFiltered returns address data values respecting the config's print
// visibility and sort order. Handles both fixed columns (stored as Address struct
// fields) and dynamic columns (stored in addr.Data). Falls back to addressLines
// if config is nil.
func addressLinesFiltered(addr *models.Address, config *models.AddressListConfig) []string {
	if addr == nil {
		return nil
	}
	if config == nil {
		return addressLines(addr)
	}

	printCols := config.PrintVisibleColumns()
	var lines []string
	for _, col := range printCols {
		var v string
		if getter, ok := fixedFieldMap[col.Name]; ok {
			// Fixed column — read from Address struct field
			v = strings.TrimSpace(getter(addr))
		} else {
			// Dynamic column — read from Data map
			v = strings.TrimSpace(addr.Data[col.Name])
		}
		if v != "" {
			lines = append(lines, v)
		}
	}
	return lines
}

// calcMultiCellHeight estimates the height a MultiCell would require.
func calcMultiCellHeight(pdf *fpdf.Fpdf, w float64, txt string, fontSize float64) float64 {
	if txt == "" {
		return lineH
	}
	setFont(pdf, "", fontSize)
	availW := w - 2*cellPad
	if availW < 1 {
		availW = 1
	}
	lines := pdf.SplitText(txt, availW)
	if len(lines) == 0 {
		return lineH
	}
	return float64(len(lines)) * lineH
}

// ensureSpace checks remaining page space and adds a new page if insufficient.
func ensureSpace(pdf *fpdf.Fpdf, needed float64) {
	if pdf.GetY()+needed > pageH-marginB {
		pdf.AddPage()
	}
}

// --- Section: Company Header ---

func drawCompanyHeader(pdf *fpdf.Fpdf, company *models.CompanySettings, billFromAddr *models.Address, showEmail, showCIN bool) {
	// Extract header fields from Bill From address when available, fall back to CompanySettings.
	var name, addr, email, gstin, cin string

	if billFromAddr != nil && billFromAddr.Data != nil {
		name = strings.TrimSpace(billFromAddr.Data["Company Name"])
		// Build address line from address fields
		var addrParts []string
		for _, key := range []string{"Address Line 1", "Address Line 2", "City", "State", "PIN Code"} {
			v := strings.TrimSpace(billFromAddr.Data[key])
			if v != "" {
				addrParts = append(addrParts, v)
			}
		}
		addr = strings.Join(addrParts, ", ")
		email = strings.TrimSpace(billFromAddr.Data["Email"])
		gstin = strings.TrimSpace(billFromAddr.Data["GSTIN"])
		cin = strings.TrimSpace(billFromAddr.Data["CIN No."])
	}

	// Fall back to CompanySettings for any missing fields
	if name == "" && company != nil {
		name = company.Name
	}
	if addr == "" && company != nil {
		addr = fmt.Sprintf("%s, %s, %s %s", company.Address, company.City, company.State, company.Pincode)
	}
	if email == "" && company != nil {
		email = company.Email
	}
	if gstin == "" && company != nil {
		gstin = company.GSTIN
	}
	if cin == "" && company != nil {
		cin = company.CIN
	}

	if name == "" {
		return
	}

	setFont(pdf, "B", 14)
	setColor(pdf, colorGray900)
	pdf.CellFormat(contentW, 6, strings.ToUpper(name), "", 1, "C", false, 0, "")

	if addr != "" {
		setFont(pdf, "", 8)
		setColor(pdf, colorGray600)
		pdf.CellFormat(contentW, 4, addr, "", 1, "C", false, 0, "")
	}

	if showEmail && email != "" {
		setFont(pdf, "", 8)
		setColor(pdf, colorGray500)
		pdf.CellFormat(contentW, 4, "Email: "+email, "", 1, "C", false, 0, "")
	}

	// GSTIN (and optionally CIN)
	if gstin != "" {
		setFont(pdf, "B", 8)
		setColor(pdf, colorGray700)
		gstinLine := "GSTIN: " + gstin
		if showCIN && cin != "" {
			gstinLine += "    CIN: " + cin
		}
		pdf.CellFormat(contentW, 4, gstinLine, "", 1, "C", false, 0, "")
	}

	spacer(pdf, 2)
	drawHLine(pdf, pdf.GetY(), colorGray800, 0.5)
	spacer(pdf, 4)
}

// --- Section: DC Title ---

func drawDCTitle(pdf *fpdf.Fpdf, title string, doubleBorder bool) {
	y := pdf.GetY()

	if doubleBorder {
		drawHLine(pdf, y, colorGray800, 0.5)
		spacer(pdf, 2)
	}

	setFont(pdf, "B", 12)
	setColor(pdf, colorGray900)
	titleW := pdf.GetStringWidth(title) + 20
	x := marginL + (contentW-titleW)/2

	if doubleBorder {
		pdf.CellFormat(contentW, 6, strings.ToUpper(title), "", 1, "C", false, 0, "")
		spacer(pdf, 1)
		drawHLine(pdf, pdf.GetY(), colorGray800, 0.5)
	} else {
		setDrawColor(pdf, colorSlate300)
		pdf.SetLineWidth(0.3)
		pdf.SetXY(x, pdf.GetY())
		pdf.CellFormat(titleW, 7, strings.ToUpper(title), "1", 1, "C", false, 0, "")
	}

	spacer(pdf, 4)
	_ = y
}

// --- Section: DC Details + PO Details (side by side) ---

func drawDCAndPOGrid(pdf *fpdf.Fpdf, dc *models.DeliveryChallan, td *models.DCTransitDetails, project *models.Project) {
	y := pdf.GetY()
	colW := contentW/2 - 2
	gap := 4.0
	innerW := colW - 2*cellPad

	// Count lines for left box (DC details)
	leftLines := 1 // DC No (always)
	if dc.ChallanDate != nil {
		leftLines++
	}
	if td != nil {
		if td.TransporterName != "" {
			leftLines++
		}
		if td.VehicleNumber != "" {
			leftLines++
		}
		if td.EwayBillNumber != "" {
			leftLines++
		}
	}
	leftLines++ // Reverse Charge (always)

	// Count lines for right box (PO details)
	rightLines := 0
	if project != nil {
		if project.POReference != "" {
			rightLines++
		}
		if project.PODate != nil {
			rightLines++
		}
		rightLines++ // Project name (always)
	}

	maxLines := leftLines
	if rightLines > maxLines {
		maxLines = rightLines
	}
	boxH := float64(maxLines)*lineH + 2*cellPad

	// Left box: DC details
	drawBorderedRect(pdf, marginL, y, colW, boxH)
	pdf.SetXY(marginL+cellPad, y+cellPad)
	kvRow(pdf, "DC No:", dc.DCNumber, 24, innerW)
	if dc.ChallanDate != nil {
		pdf.SetX(marginL + cellPad)
		kvRow(pdf, "Date:", *dc.ChallanDate, 24, innerW)
	}
	if td != nil {
		if td.TransporterName != "" {
			pdf.SetX(marginL + cellPad)
			kvRow(pdf, "Transporter:", td.TransporterName, 24, innerW)
		}
		if td.VehicleNumber != "" {
			pdf.SetX(marginL + cellPad)
			kvRow(pdf, "Vehicle:", td.VehicleNumber, 24, innerW)
		}
		if td.EwayBillNumber != "" {
			pdf.SetX(marginL + cellPad)
			kvRow(pdf, "E-Way Bill:", td.EwayBillNumber, 24, innerW)
		}
	}
	pdf.SetX(marginL + cellPad)
	kvRow(pdf, "Reverse Charge:", "No", 30, innerW)

	// Right box: PO details
	rightX := marginL + colW + gap
	drawBorderedRect(pdf, rightX, y, colW, boxH)
	pdf.SetXY(rightX+cellPad, y+cellPad)

	if project != nil {
		if project.POReference != "" {
			pdf.SetX(rightX + cellPad)
			kvRow(pdf, "PO Number:", project.POReference, 22, innerW)
		}
		if project.PODate != nil {
			pdf.SetX(rightX + cellPad)
			kvRow(pdf, "PO Date:", *project.PODate, 22, innerW)
		}
		pdf.SetX(rightX + cellPad)
		kvRow(pdf, "Project:", project.Name, 22, innerW)
	}

	pdf.SetY(y + boxH + 4)
}

// --- Section: Transit Address Grid (2x2) ---

func drawTransitAddressGrid(pdf *fpdf.Fpdf, company *models.CompanySettings, billFrom, dispatchFrom, billTo, shipTo *models.Address, billFromConfig, dispatchFromConfig, billToConfig, shipToConfig *models.AddressListConfig) {
	colW := contentW/2 - 2
	gap := 4.0
	y := pdf.GetY()

	// Row 1: Bill From / Bill To
	// Use DC-selected bill-from address; fall back to company address for older DCs
	billFromLines := addressLinesFiltered(billFrom, billFromConfig)
	if len(billFromLines) == 0 {
		billFromLines = companyAddressLines(company)
	}
	h1 := drawAddressBox(pdf, marginL, y, colW, "Bill From", billFromLines)
	h2 := drawAddressBox(pdf, marginL+colW+gap, y, colW, "Bill To", addressLinesFiltered(billTo, billToConfig))
	rowH := math.Max(h1, h2)
	pdf.SetY(y + rowH + 2)

	y = pdf.GetY()
	// Row 2: Dispatch From / Ship To
	// Use DC-selected dispatch-from address; fall back to company address for older DCs
	dispatchFromLines := addressLinesFiltered(dispatchFrom, dispatchFromConfig)
	if len(dispatchFromLines) == 0 {
		dispatchFromLines = companyAddressLines(company)
	}
	h1 = drawAddressBox(pdf, marginL, y, colW, "Dispatch From", dispatchFromLines)
	h2 = drawAddressBox(pdf, marginL+colW+gap, y, colW, "Ship To", addressLinesFiltered(shipTo, shipToConfig))
	rowH = math.Max(h1, h2)
	pdf.SetY(y + rowH + 4)
}

func companyAddressLines(company *models.CompanySettings) []string {
	if company == nil {
		return nil
	}
	return []string{
		company.Name,
		fmt.Sprintf("%s, %s, %s %s", company.Address, company.City, company.State, company.Pincode),
		"GSTIN: " + company.GSTIN,
	}
}

// drawAddressBox draws a titled address block with text wrapping and returns its height.
func drawAddressBox(pdf *fpdf.Fpdf, x, y, w float64, title string, lines []string) float64 {
	minH := 20.0
	lh := 3.5
	innerW := w - 2*cellPad

	// Pre-calculate required height by measuring wrapped lines
	h := cellPad + 4 // top padding + title height

	if len(lines) == 0 {
		h += lh // "Not specified"
	} else {
		for i, line := range lines {
			if i == 0 {
				setFont(pdf, "B", 8)
			} else {
				setFont(pdf, "", 8)
			}
			if line == "" {
				h += lh
				continue
			}
			wrapped := pdf.SplitText(line, innerW)
			if len(wrapped) == 0 {
				h += lh
			} else {
				h += float64(len(wrapped)) * lh
			}
		}
	}
	h += cellPad // bottom padding
	if h < minH {
		h = minH
	}

	// Draw the border at the computed height
	drawBorderedRect(pdf, x, y, w, h)

	// Title
	pdf.SetXY(x+cellPad, y+cellPad)
	setFont(pdf, "B", 7)
	setColor(pdf, colorGray400)
	pdf.CellFormat(innerW, 3.5, strings.ToUpper(title), "", 1, "L", false, 0, "")

	if len(lines) == 0 {
		setFont(pdf, "", 8)
		setColor(pdf, colorGray400)
		pdf.SetX(x + cellPad)
		pdf.CellFormat(innerW, lh, "Not specified", "", 1, "L", false, 0, "")
	} else {
		for i, line := range lines {
			if i == 0 {
				setFont(pdf, "B", 8)
				setColor(pdf, colorGray800)
			} else {
				setFont(pdf, "", 8)
				setColor(pdf, colorGray600)
			}
			pdf.SetX(x + cellPad)
			// Use MultiCell for text wrapping within the box width
			pdf.MultiCell(innerW, lh, line, "", "L", false)
		}
	}

	return h
}

// --- Section: Transit Product Table ---

func drawTransitProductTable(pdf *fpdf.Fpdf, items []models.DCLineItem, totalQty int, totalTaxable, totalTax, grandTotal float64) {
	cols := []tableCol{
		{"S.No", 9, "C"},
		{"Description", 48, "L"},
		{"Serials", 22, "L"},
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
		serials := strings.Join(li.SerialNumbers, "\n")

		values := []string{
			fmt.Sprintf("%d", i+1),
			desc,
			serials,
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
		"", "", "Total", "", "",
		fmt.Sprintf("%d", totalQty),
		"", "Rs." + fmtINR(totalTaxable), "",
		"Rs." + fmtINR(totalTax),
		"Rs." + fmtINR(grandTotal),
	}
	drawTableDataRow(pdf, cols, totals, true)
	spacer(pdf, 4)
}

// --- Section: Tax Summary ---

func drawTaxSummary(pdf *fpdf.Fpdf, totalTaxable, halfTax, roundOff, roundedTotal float64) {
	boxW := 80.0
	x := marginL + contentW - boxW
	y := pdf.GetY()

	setDrawColor(pdf, colorSlate300)
	pdf.SetLineWidth(0.3)

	rows := []struct {
		label string
		value string
		bold  bool
	}{
		{"Taxable Value", "Rs." + fmtINR(totalTaxable), false},
		{"CGST", "Rs." + fmtINR(halfTax), false},
		{"SGST", "Rs." + fmtINR(halfTax), false},
		{"Round Off", fmt.Sprintf("Rs.%.2f", roundOff), false},
		{"Invoice Value", "Rs." + fmtINR(roundedTotal), true},
	}

	rowH := 5.0
	totalH := float64(len(rows)) * rowH
	ensureSpace(pdf, totalH+4)
	y = pdf.GetY()

	drawBorderedRect(pdf, x, y, boxW, totalH)

	for i, row := range rows {
		ry := y + float64(i)*rowH
		if row.bold {
			drawFilledRect(pdf, x, ry, boxW, rowH, colorSlate50)
		}
		if i < len(rows)-1 {
			setDrawColor(pdf, colorSlate100)
			pdf.SetLineWidth(0.1)
			pdf.Line(x, ry+rowH, x+boxW, ry+rowH)
		}

		if row.bold {
			setFont(pdf, "B", 8)
		} else {
			setFont(pdf, "", 8)
		}
		setColor(pdf, colorGray500)
		pdf.SetXY(x+cellPad, ry+0.5)
		pdf.CellFormat(boxW/2-cellPad, rowH-1, row.label, "", 0, "L", false, 0, "")

		if row.bold {
			setFont(pdf, "B", 8)
			setColor(pdf, colorGray900)
		} else {
			setFont(pdf, "B", 8)
			setColor(pdf, colorGray800)
		}
		pdf.CellFormat(boxW/2-cellPad, rowH-1, row.value, "", 0, "R", false, 0, "")
	}

	pdf.SetY(y + totalH + 4)
}

// --- Section: Amount in Words ---

func drawAmountInWords(pdf *fpdf.Fpdf, words string) {
	ensureSpace(pdf, 14)
	y := pdf.GetY()
	h := 12.0
	drawBorderedRect(pdf, marginL, y, contentW, h)

	pdf.SetXY(marginL+cellPad, y+cellPad)
	setFont(pdf, "B", 7)
	setColor(pdf, colorGray400)
	pdf.CellFormat(contentW-2*cellPad, 3.5, "AMOUNT IN WORDS", "", 1, "L", false, 0, "")

	pdf.SetX(marginL + cellPad)
	setFont(pdf, "B", 8)
	setColor(pdf, colorGray800)
	pdf.CellFormat(contentW-2*cellPad, 4, words, "", 1, "L", false, 0, "")

	pdf.SetY(y + h + 4)
}

// --- Section: Notes ---

func drawNotes(pdf *fpdf.Fpdf, notes string) {
	ensureSpace(pdf, 14)
	y := pdf.GetY()

	// Calculate height needed
	setFont(pdf, "", 8)
	noteLines := pdf.SplitText(notes, contentW-2*cellPad)
	h := 3.5 + float64(len(noteLines))*lineH + 2*cellPad + 2
	if h < 12 {
		h = 12
	}

	drawBorderedRect(pdf, marginL, y, contentW, h)

	pdf.SetXY(marginL+cellPad, y+cellPad)
	setFont(pdf, "B", 7)
	setColor(pdf, colorGray400)
	pdf.CellFormat(contentW-2*cellPad, 3.5, "NOTES", "", 1, "L", false, 0, "")

	pdf.SetX(marginL + cellPad)
	setFont(pdf, "", 8)
	setColor(pdf, colorGray700)
	pdf.MultiCell(contentW-2*cellPad, lineH, notes, "", "L", false)

	pdf.SetY(y + h + 6)
}

// --- Section: Transit Signatures ---

func drawTransitSignatures(pdf *fpdf.Fpdf, company *models.CompanySettings, project *models.Project) {
	ensureSpace(pdf, 45)

	drawHLine(pdf, pdf.GetY(), colorSlate300, 0.3)
	spacer(pdf, 4)

	y := pdf.GetY()
	colW := contentW / 2

	// Left: Receiver
	pdf.SetXY(marginL, y)
	setFont(pdf, "B", 8)
	setColor(pdf, colorGray700)
	pdf.CellFormat(colW, lineH, "Receiver's Signature", "", 1, "L", false, 0, "")
	spacer(pdf, 16)
	pdf.SetX(marginL)
	setDrawColor(pdf, colorGray400)
	pdf.SetLineWidth(0.3)
	pdf.Line(marginL, pdf.GetY(), marginL+60, pdf.GetY())
	spacer(pdf, 2)
	pdf.SetX(marginL)
	setFont(pdf, "", 8)
	setColor(pdf, colorGray400)
	pdf.CellFormat(colW, lineH, "Name: _________________________", "", 1, "L", false, 0, "")

	// Right: Authorized Signatory
	rightX := marginL + colW
	pdf.SetXY(rightX, y)
	if company != nil {
		setFont(pdf, "B", 8)
		setColor(pdf, colorGray700)
		pdf.CellFormat(colW, lineH, "For "+company.Name, "", 1, "R", false, 0, "")
	}

	// Company signature image (between company name and "Authorized Signatory")
	sigBottomY := y + 25 // default gap when no image
	if project != nil && project.CompanySignaturePath != "" {
		uploadRoot := os.Getenv("UPLOAD_PATH")
		if uploadRoot == "" {
			uploadRoot = "./static/uploads"
		}
		sigPath := filepath.Join(uploadRoot, project.CompanySignaturePath)
		if _, err := os.Stat(sigPath); err == nil {
			// Width only; height=0 preserves aspect ratio
			sigImgW := 29.0
			imgX := rightX + colW - sigImgW
			imgY := y + lineH + 2
			pdf.ImageOptions(sigPath, imgX, imgY, sigImgW, 0, false, fpdf.ImageOptions{ReadDpi: true}, 0, "")
			// Query the registered image to compute rendered height
			info := pdf.GetImageInfo(sigPath)
			if info != nil && info.Width() > 0 {
				sigRenderedH := sigImgW * info.Height() / info.Width()
				sigBottomY = imgY + sigRenderedH + 3
			}
		}
	}

	pdf.SetXY(rightX, sigBottomY)
	setFont(pdf, "B", 8)
	setColor(pdf, colorGray600)
	pdf.CellFormat(colW, lineH, "Authorized Signatory", "", 1, "R", false, 0, "")

	// Print signatory details if configured
	if project != nil && project.SignatoryName != "" {
		setFont(pdf, "", 7)
		setColor(pdf, colorGray700)
		spacer(pdf, 1)
		if project.SignatoryName != "" {
			pdf.SetX(rightX)
			pdf.CellFormat(colW, lineH, project.SignatoryName, "", 1, "R", false, 0, "")
		}
		if project.SignatoryDesignation != "" {
			pdf.SetX(rightX)
			pdf.CellFormat(colW, lineH, project.SignatoryDesignation, "", 1, "R", false, 0, "")
		}
		if project.SignatoryMobile != "" {
			pdf.SetX(rightX)
			pdf.CellFormat(colW, lineH, "Ph: "+project.SignatoryMobile, "", 1, "R", false, 0, "")
		}
	}
}

// --- Section: Copy Indicators ---

func drawCopyIndicators(pdf *fpdf.Fpdf) {
	setFont(pdf, "", 8)
	setColor(pdf, colorGray600)
	labels := []string{"Original", "Duplicate", "Triplicate"}
	totalW := float64(0)
	for _, l := range labels {
		totalW += pdf.GetStringWidth(l) + 8
	}
	x := marginL + contentW - totalW
	for _, l := range labels {
		// checkbox (checked)
		pdf.SetXY(x, pdf.GetY())
		pdf.CellFormat(4, lineH, "[x]", "", 0, "C", false, 0, "")
		x += 4
		w := pdf.GetStringWidth(l) + 4
		pdf.CellFormat(w, lineH, l, "", 0, "L", false, 0, "")
		x += w
	}
	spacer(pdf, 6)
}

// --- Section: Official Meta Grid ---

func drawOfficialMetaGrid(pdf *fpdf.Fpdf, dc *models.DeliveryChallan, shipTo *models.Address) {
	colW := contentW / 2

	setFont(pdf, "B", 8)
	setColor(pdf, colorGray500)
	pdf.CellFormat(colW, lineH, "DC No: ", "", 0, "L", false, 0, "")

	setFont(pdf, "B", 9)
	setColor(pdf, colorGray900)
	y := pdf.GetY()
	pdf.SetXY(marginL+18, y)
	pdf.CellFormat(colW-18, lineH, dc.DCNumber, "", 0, "L", false, 0, "")

	if dc.ChallanDate != nil {
		pdf.SetXY(marginL+colW, y)
		setFont(pdf, "B", 8)
		setColor(pdf, colorGray500)
		pdf.CellFormat(14, lineH, "Date: ", "", 0, "L", false, 0, "")
		setFont(pdf, "B", 9)
		setColor(pdf, colorGray900)
		pdf.CellFormat(colW-14, lineH, *dc.ChallanDate, "", 0, "L", false, 0, "")
	}
	pdf.Ln(lineH + 1)

	if shipTo != nil {
		if mandal, ok := shipTo.Data["mandal"]; ok && mandal != "" {
			setFont(pdf, "B", 8)
			setColor(pdf, colorGray500)
			pdf.CellFormat(25, lineH, "Mandal/ULB: ", "", 0, "L", false, 0, "")
			setFont(pdf, "B", 9)
			setColor(pdf, colorGray900)
			pdf.CellFormat(colW-25, lineH, mandal, "", 0, "L", false, 0, "")
		}
		if code, ok := shipTo.Data["mandal_code"]; ok && code != "" {
			setFont(pdf, "B", 8)
			setColor(pdf, colorGray500)
			pdf.CellFormat(25, lineH, "Mandal Code: ", "", 0, "L", false, 0, "")
			setFont(pdf, "B", 9)
			setColor(pdf, colorGray900)
			pdf.CellFormat(colW-25, lineH, code, "", 0, "L", false, 0, "")
		}
		pdf.Ln(lineH + 1)
	}

	spacer(pdf, 4)
}

// --- Section: Reference Info ---

func drawReferenceInfo(pdf *fpdf.Fpdf, project *models.Project) {
	if project == nil {
		return
	}

	if project.TenderRefNumber != "" {
		setFont(pdf, "B", 8)
		setColor(pdf, colorGray500)
		pdf.CellFormat(22, lineH, "Tender Ref: ", "", 0, "L", false, 0, "")
		setFont(pdf, "", 9)
		setColor(pdf, colorGray800)
		pdf.CellFormat(0, lineH, project.TenderRefNumber, "", 1, "L", false, 0, "")
	}
	if project.POReference != "" {
		setFont(pdf, "B", 8)
		setColor(pdf, colorGray500)
		pdf.CellFormat(22, lineH, "PO Ref: ", "", 0, "L", false, 0, "")
		setFont(pdf, "", 9)
		setColor(pdf, colorGray800)
		po := project.POReference
		if project.PODate != nil {
			po += " (" + *project.PODate + ")"
		}
		pdf.CellFormat(0, lineH, po, "", 1, "L", false, 0, "")
	}

	setFont(pdf, "B", 8)
	setColor(pdf, colorGray500)
	pdf.CellFormat(22, lineH, "Project: ", "", 0, "L", false, 0, "")
	setFont(pdf, "", 9)
	setColor(pdf, colorGray800)
	pdf.CellFormat(0, lineH, project.Name, "", 1, "L", false, 0, "")

	spacer(pdf, 4)
}

// --- Section: Purpose ---

func drawPurpose(pdf *fpdf.Fpdf, purpose string) {
	setFont(pdf, "B", 8)
	setColor(pdf, colorGray500)
	pdf.CellFormat(18, lineH, "Purpose: ", "", 0, "L", false, 0, "")
	setFont(pdf, "B", 9)
	setColor(pdf, colorGray900)
	pdf.CellFormat(0, lineH, strings.ToUpper(purpose), "", 1, "L", false, 0, "")
	spacer(pdf, 3)
}

// --- Section: Issued To ---

func drawIssuedTo(pdf *fpdf.Fpdf, shipTo *models.Address) {
	if shipTo == nil {
		return
	}
	issuedTo := ""
	if d, ok := shipTo.Data["district"]; ok && d != "" {
		issuedTo = d + " District"
	}
	if m, ok := shipTo.Data["mandal"]; ok && m != "" {
		if issuedTo != "" {
			issuedTo += ", "
		}
		issuedTo += m + " Mandal/ULB"
	}
	if issuedTo == "" {
		return
	}

	setFont(pdf, "B", 8)
	setColor(pdf, colorGray500)
	pdf.CellFormat(20, lineH, "Issued To: ", "", 0, "L", false, 0, "")
	setFont(pdf, "B", 9)
	setColor(pdf, colorGray900)
	pdf.CellFormat(0, lineH, issuedTo, "", 1, "L", false, 0, "")
	spacer(pdf, 3)
}

// --- Section: Address Pair (Bill To / Ship To) ---

func drawAddressPair(pdf *fpdf.Fpdf, billTo, shipTo *models.Address, billToConfig, shipToConfig *models.AddressListConfig) {
	colW := contentW/2 - 2
	gap := 4.0
	y := pdf.GetY()

	h1 := 0.0
	if billTo != nil {
		h1 = drawAddressBox(pdf, marginL, y, colW, "Bill To", addressLinesFiltered(billTo, billToConfig))
	}
	h2 := 0.0
	if shipTo != nil {
		h2 = drawAddressBox(pdf, marginL+colW+gap, y, colW, "Ship To", addressLinesFiltered(shipTo, shipToConfig))
	}
	rowH := math.Max(h1, h2)
	if rowH < 1 {
		rowH = 0
	}
	pdf.SetY(y + rowH + 4)
}

// --- Section: Official Product Table ---

func drawOfficialProductTable(pdf *fpdf.Fpdf, items []models.DCLineItem) {
	cols := []tableCol{
		{"S.No", 12, "C"},
		{"Item Name", 35, "L"},
		{"Description", 38, "L"},
		{"Brand / Model No", 30, "L"},
		{"Qty", 13, "C"},
		{"Serial Number", 40, "L"},
		{"Remarks", 22, "L"},
	}

	drawTableHeaderRow(pdf, cols)

	for i, li := range items {
		serials := strings.Join(li.SerialNumbers, "\n")
		values := []string{
			fmt.Sprintf("%d", i+1),
			li.ItemName,
			li.ItemDescription,
			li.BrandModel,
			fmt.Sprintf("%d", li.Quantity),
			serials,
			"-",
		}
		drawTableDataRow(pdf, cols, values, false)
	}

	spacer(pdf, 6)
}

// --- Section: Acknowledgement ---

func drawAcknowledgement(pdf *fpdf.Fpdf) {
	ensureSpace(pdf, 18)
	y := pdf.GetY()
	h := 16.0

	drawFilledRect(pdf, marginL, y, contentW, h, colorSlate50)

	pdf.SetXY(marginL+cellPad, y+cellPad)
	setFont(pdf, "B", 8)
	setColor(pdf, colorGray800)
	pdf.CellFormat(contentW-2*cellPad, 5, "\"It is certified that the material is received in good condition.\"", "", 1, "L", false, 0, "")

	pdf.SetX(marginL + cellPad)
	setFont(pdf, "B", 8)
	setColor(pdf, colorGray500)
	pdf.CellFormat(30, 5, "Date of Receipt: ", "", 0, "L", false, 0, "")
	setDrawColor(pdf, colorGray400)
	pdf.SetLineWidth(0.3)
	lineY := pdf.GetY() + 4.5
	pdf.Line(marginL+cellPad+30, lineY, marginL+cellPad+100, lineY)

	pdf.SetY(y + h + 6)
}

// --- Section: Official Signatures ---

func drawOfficialSignatures(pdf *fpdf.Fpdf, project *models.Project) {
	ensureSpace(pdf, 65)

	drawHLine(pdf, pdf.GetY(), colorSlate300, 0.3)
	spacer(pdf, 4)

	y := pdf.GetY()
	colW := contentW / 2

	// Resolve signature image path once
	var sigPath string
	if project != nil && project.CompanySignaturePath != "" {
		uploadRoot := os.Getenv("UPLOAD_PATH")
		if uploadRoot == "" {
			uploadRoot = "./static/uploads"
		}
		candidate := filepath.Join(uploadRoot, project.CompanySignaturePath)
		if _, err := os.Stat(candidate); err == nil {
			sigPath = candidate
		}
	}

	// Extract signatory details from project
	var sigName, sigDesignation, sigMobile string
	if project != nil {
		sigName = project.SignatoryName
		sigDesignation = project.SignatoryDesignation
		sigMobile = project.SignatoryMobile
	}

	// Helper to draw one signature block
	drawSigBlock := func(x, y float64, title string, showSigImage bool) {
		pdf.SetXY(x, y)
		setFont(pdf, "B", 8)
		setColor(pdf, colorGray800)
		w := colW - 10
		pdf.CellFormat(w, lineH, strings.ToUpper(title), "", 1, "C", false, 0, "")
		spacer(pdf, 2)

		boxY := pdf.GetY()
		fieldsY := boxY + 22 // default offset for dashed-box case

		if showSigImage && sigPath != "" {
			// Render the company signature image centered; height=0 preserves aspect ratio
			sigImgW := 29.0
			imgX := x + (w-sigImgW)/2
			imgY := boxY + 2
			pdf.ImageOptions(sigPath, imgX, imgY, sigImgW, 0, false, fpdf.ImageOptions{ReadDpi: true}, 0, "")
			// Compute actual rendered height for proper spacing
			info := pdf.GetImageInfo(sigPath)
			if info != nil && info.Width() > 0 {
				sigRenderedH := sigImgW * info.Height() / info.Width()
				fieldsY = imgY + sigRenderedH + 3
			}
		} else {
			// Signature box (dashed)
			boxX := x + (w-55)/2
			setDrawColor(pdf, colorGray300)
			pdf.SetLineWidth(0.3)
			pdf.SetDashPattern([]float64{2, 1}, 0)
			pdf.Rect(boxX, boxY, 55, 18, "D")
			pdf.SetDashPattern([]float64{}, 0) // reset dash

			setFont(pdf, "", 7)
			setColor(pdf, colorGray400)
			pdf.SetXY(boxX, boxY+6)
			pdf.CellFormat(55, 5, "Signature", "", 0, "C", false, 0, "")
		}

		pdf.SetXY(x, fieldsY)
		setFont(pdf, "", 8)
		setColor(pdf, colorGray800)

		type labelVal struct{ Label, Value string }
		fields := []labelVal{
			{"Name:", sigName},
			{"Designation:", sigDesignation},
			{"Mobile:", sigMobile},
		}
		for _, f := range fields {
			pdf.SetX(x)
			pdf.CellFormat(22, 5, f.Label, "", 0, "L", false, 0, "")
			if showSigImage && f.Value != "" {
				pdf.CellFormat(45, 5, f.Value, "", 0, "L", false, 0, "")
			} else {
				setDrawColor(pdf, colorGray400)
				pdf.SetLineWidth(0.2)
				lx := x + 22
				ly := pdf.GetY() + 4.5
				pdf.Line(lx, ly, lx+45, ly)
			}
			pdf.Ln(5)
		}
	}

	drawSigBlock(marginL, y, "FSSPL Representative", true)
	drawSigBlock(marginL+colW, y, "Department Official", false)
}

// --- Table drawing helpers ---

func drawTableHeaderRow(pdf *fpdf.Fpdf, cols []tableCol) {
	headerFontSize := 6.0
	headerLineH := 3.5
	setFont(pdf, "B", headerFontSize)
	setColor(pdf, colorGray700)
	setFillColor(pdf, colorSlate100)
	setDrawColor(pdf, colorSlate300)
	pdf.SetLineWidth(0.3)

	// Single-line header — fixed height
	h := headerLineH + 3 // padding top+bottom

	y := pdf.GetY()
	x := tblMarginL

	for _, col := range cols {
		txt := strings.ToUpper(col.Header)
		// Draw filled cell border
		pdf.Rect(x, y, col.Width, h, "FD")
		// Draw single-line text centered vertically
		pdf.SetXY(x+cellPad, y+(h-headerLineH)/2)
		availW := col.Width - 2*cellPad
		if availW < 1 {
			availW = 1
		}
		pdf.CellFormat(availW, headerLineH, txt, "", 0, "C", false, 0, "")
		x += col.Width
	}
	pdf.SetY(y + h)
}

func drawTableDataRow(pdf *fpdf.Fpdf, cols []tableCol, values []string, isTotalRow bool) {
	dataFontSize := 7.0
	padding := 2.0
	minRowH := lineH + padding

	// Pre-split all cell text into lines
	setFont(pdf, "", dataFontSize)
	cellLines := make([][]string, len(cols))
	for i, col := range cols {
		val := ""
		if i < len(values) {
			val = values[i]
		}
		if val == "" {
			cellLines[i] = []string{""}
		} else {
			availW := col.Width - 2*cellPad
			if availW < 1 {
				availW = 1
			}
			cellLines[i] = pdf.SplitText(val, availW)
			if len(cellLines[i]) == 0 {
				cellLines[i] = []string{""}
			}
		}
	}

	// Calculate full row height
	maxLines := 1
	for _, lines := range cellLines {
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	maxH := float64(maxLines)*lineH + padding

	remainingSpace := pageH - marginB - pdf.GetY()

	// Estimate header height for page break decisions
	headerH := 8.0 // approximate; header recalculated on draw

	if maxH <= remainingSpace {
		// Fits on current page — draw normally
		drawDataRowSlice(pdf, cols, cellLines, 0, maxLines, maxH, isTotalRow)
	} else if remainingSpace >= headerH+minRowH && maxH > remainingSpace {
		// Split the row across pages
		linesPerPage := int((remainingSpace - padding) / lineH)
		if linesPerPage < 1 {
			linesPerPage = 1
		}

		// Draw first portion
		firstH := float64(linesPerPage)*lineH + padding
		drawDataRowSlice(pdf, cols, cellLines, 0, linesPerPage, firstH, isTotalRow)

		// Continue on next page(s)
		drawn := linesPerPage
		for drawn < maxLines {
			pdf.AddPage()
			drawTableHeaderRow(pdf, cols)

			remaining := maxLines - drawn
			pageLinesAvail := int((pageH - marginB - marginT - headerH - padding) / lineH)
			if pageLinesAvail < 1 {
				pageLinesAvail = 1
			}
			chunk := remaining
			if chunk > pageLinesAvail {
				chunk = pageLinesAvail
			}
			chunkH := float64(chunk)*lineH + padding
			drawDataRowSlice(pdf, cols, cellLines, drawn, drawn+chunk, chunkH, isTotalRow)
			drawn += chunk
		}
	} else {
		// Not enough space for even a partial row — push to next page
		pdf.AddPage()
		drawTableHeaderRow(pdf, cols)
		drawDataRowSlice(pdf, cols, cellLines, 0, maxLines, maxH, isTotalRow)
	}
}

// drawDataRowSlice draws a portion of a data row (lines fromLine..toLine) with the given height.
func drawDataRowSlice(pdf *fpdf.Fpdf, cols []tableCol, cellLines [][]string, fromLine, toLine int, rowH float64, isTotalRow bool) {
	y := pdf.GetY()
	x := tblMarginL

	if isTotalRow {
		setFillColor(pdf, colorSlate50)
	}

	setDrawColor(pdf, colorSlate300)
	pdf.SetLineWidth(0.3)

	for i, col := range cols {
		// Draw cell border
		if isTotalRow {
			pdf.Rect(x, y, col.Width, rowH, "FD")
		} else {
			pdf.Rect(x, y, col.Width, rowH, "D")
		}

		// Set font
		if isTotalRow {
			setFont(pdf, "B", 7)
			setColor(pdf, colorGray900)
		} else {
			setFont(pdf, "", 7)
			setColor(pdf, colorGray800)
		}

		// Extract lines for this slice
		lines := cellLines[i]
		start := fromLine
		end := toLine
		if start >= len(lines) {
			start = len(lines)
		}
		if end > len(lines) {
			end = len(lines)
		}
		sliceLines := lines[start:end]

		availW := col.Width - 2*cellPad
		if availW < 1 {
			availW = 1
		}

		if len(sliceLines) <= 1 {
			// Single line — vertically center
			txt := ""
			if len(sliceLines) == 1 {
				txt = sliceLines[0]
			}
			pdf.SetXY(x+cellPad, y+(rowH-lineH)/2)
			pdf.CellFormat(availW, lineH, txt, "", 0, col.Align, false, 0, "")
		} else {
			// Multi-line — draw from top with padding
			for j, line := range sliceLines {
				pdf.SetXY(x+cellPad, y+cellPad+float64(j)*lineH)
				pdf.CellFormat(availW, lineH, line, "", 0, col.Align, false, 0, "")
			}
		}

		x += col.Width
	}

	pdf.SetY(y + rowH)
}
