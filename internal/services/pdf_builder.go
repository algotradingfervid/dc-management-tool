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
	qrcode "github.com/skip2/go-qrcode"
)

//go:embed fonts/DejaVuSans.ttf
var dejaVuSansRegular []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var dejaVuSansBold []byte

// --- Color constants (Tailwind gray scale RGB equivalents) ---

type rgb struct{ R, G, B int }

var (
	colorBlack = rgb{0, 0, 0}
	colorWhite = rgb{255, 255, 255}
)

// --- Page constants ---
const (
	pageW       = 210.0 // A4 width mm
	pageH       = 297.0 // A4 height mm
	marginL     = 10.0
	marginR     = 10.0
	marginT     = 10.0
	marginB     = 14.0
	contentW    = pageW - marginL - marginR // 190mm
	lineH       = 4.0                       // default line height mm
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
	TransferDCNumber  string // parent Transfer DC number (for split transit DCs)
}

// OfficialDCPDFData holds all data needed to generate an Official DC PDF.
type OfficialDCPDFData struct {
	Project             *models.Project
	DC                  *models.DeliveryChallan
	TransitDetails      *models.DCTransitDetails
	LineItems           []models.DCLineItem
	Company             *models.CompanySettings
	ShipToAddress       *models.Address
	BillToAddress       *models.Address
	BillFromAddress     *models.Address
	DispatchFromAddress *models.Address
	ShipToConfig        *models.AddressListConfig // optional: for print column filtering
	BillToConfig        *models.AddressListConfig // optional: for print column filtering
	BillFromConfig      *models.AddressListConfig // optional: for print column filtering
	DispatchFromConfig  *models.AddressListConfig // optional: for print column filtering
	TotalQty            int
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
	pdf := newPDF(&pdfHeaderConfig{Project: data.Project, Company: data.Company})

	drawCompanyHeader(pdf, data.Project, data.Company, false, 0)
	drawDCTitle(pdf, "Delivery Challan", false)
	drawDCAndPOGrid(pdf, data.DC, data.TransitDetails, data.Project, data.TransferDCNumber)
	drawTransitAddressGrid(pdf, data.Company, data.BillFromAddress, data.DispatchFromAddress, data.BillToAddress, data.ShipToAddress, data.BillFromConfig, data.DispatchFromConfig, data.BillToConfig, data.ShipToConfig)
	drawTransitProductTable(pdf, data.LineItems, data.TotalQty, data.TotalTaxable, data.TotalTax, data.GrandTotal)
	drawTaxSummary(pdf, data.TotalTaxable, data.HalfTax, data.RoundOff, data.RoundedTotal)
	drawAmountInWords(pdf, data.AmountInWords)

	if data.Project != nil && data.Project.Notes != "" {
		drawNotes(pdf, data.Project.Notes)
	}

	drawTransitSignatures(pdf, data.Company, data.Project)
	drawSerialNumbersAnnexure(pdf, data.LineItems, 1)

	return pdfToBytes(pdf)
}

// GenerateOfficialDCPDF produces a PDF for an Official Delivery Challan.
func GenerateOfficialDCPDF(data *OfficialDCPDFData) ([]byte, error) {
	pdf := newPDF(&pdfHeaderConfig{
		Project:    data.Project,
		Company:    data.Company,
		ShowEmail:  true,
		QRReserved: 25,
		QRDCNumber: data.DC.DCNumber,
	})

	drawCompanyHeader(pdf, data.Project, data.Company, true, 25)
	drawQRCode(pdf, data.DC.DCNumber)
	drawDCTitle(pdf, "Official Delivery Challan", false)
	drawDCAndPOGrid(pdf, data.DC, data.TransitDetails, data.Project)
	drawTransitAddressGrid(pdf, data.Company, data.BillFromAddress, data.DispatchFromAddress, data.BillToAddress, data.ShipToAddress, data.BillFromConfig, data.DispatchFromConfig, data.BillToConfig, data.ShipToConfig)
	drawOfficialProductTable(pdf, data.LineItems)

	// Ensure acknowledgement + notes + signatures all land on the same page
	ackH := 15.0 // acknowledgement height + gap
	notesH := 0.0
	if data.Project != nil && data.Project.Notes != "" {
		notesH = estimateNotesHeight(pdf, data.Project.Notes)
	}
	sigH := 65.0 // signature section reserved height
	ensureSpace(pdf, ackH+notesH+sigH)

	drawAcknowledgement(pdf)

	if data.Project != nil && data.Project.Notes != "" {
		drawNotes(pdf, data.Project.Notes)
	}

	drawOfficialSignatures(pdf, data.Project)

	return pdfToBytes(pdf)
}

// --- PDF creation and output ---

// pdfHeaderConfig bundles header/footer settings for newPDF.
type pdfHeaderConfig struct {
	Project    *models.Project
	Company    *models.CompanySettings
	ShowEmail  bool
	QRReserved float64
	QRDCNumber string // non-empty → draw QR code on page 1 only (Official DC)
}

func newPDF(hdr *pdfHeaderConfig) *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginL, marginT, marginR)
	pdf.SetAutoPageBreak(true, marginB)

	// Register embedded UTF-8 fonts
	pdf.AddUTF8FontFromBytes("dejavu", "", dejaVuSansRegular)
	pdf.AddUTF8FontFromBytes("dejavu", "B", dejaVuSansBold)

	// Automatic company header on every page
	if hdr != nil {
		pdf.SetHeaderFunc(func() {
			drawCompanyHeader(pdf, hdr.Project, hdr.Company, hdr.ShowEmail, hdr.QRReserved)
			if hdr.QRDCNumber != "" && pdf.PageNo() == 1 {
				drawQRCode(pdf, hdr.QRDCNumber)
			}
		})
	}

	// Page number footer on every page
	pdf.AliasNbPages("{totalpages}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-marginB)
		setFont(pdf, "", 7)
		setColor(pdf, colorBlack)
		pdf.CellFormat(contentW, lineH, fmt.Sprintf("Page %d of {totalpages}", pdf.PageNo()), "", 0, "C", false, 0, "")
	})

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
	setDrawColor(pdf, colorBlack)
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, w, h, "D")
}

// drawFilledRect draws a filled rectangle.
func drawFilledRect(pdf *fpdf.Fpdf, x, y, w, h float64, fill rgb) {
	setFillColor(pdf, fill)
	setDrawColor(pdf, colorBlack)
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, w, h, "FD")
}

// kvRow draws a label: value pair on the current line within a given total width.
func kvRow(pdf *fpdf.Fpdf, label, value string, labelW, totalW float64) {
	y := pdf.GetY()
	x := pdf.GetX()

	setFont(pdf, "B", 8)
	setColor(pdf, colorBlack)
	pdf.SetXY(x, y)
	pdf.CellFormat(labelW, lineH, label, "", 0, "L", false, 0, "")

	setFont(pdf, "", 8)
	setColor(pdf, colorBlack)
	valueW := totalW - labelW
	if valueW < 1 {
		valueW = 1
	}
	pdf.CellFormat(valueW, lineH, value, "", 1, "R", false, 0, "")
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

// ensureSpace checks remaining page space and adds a new page if insufficient.
func ensureSpace(pdf *fpdf.Fpdf, needed float64) {
	if pdf.GetY()+needed > pageH-marginB {
		pdf.AddPage()
	}
}

// --- Section: Company Header ---

// drawQRCode generates a QR code containing the DC number and places it in the
// top-right corner of the page. Uses absolute positioning so it doesn't affect
// the Y cursor.
func drawQRCode(pdf *fpdf.Fpdf, dcNumber string) {
	if dcNumber == "" {
		return
	}
	png, err := qrcode.Encode(dcNumber, qrcode.Medium, 256)
	if err != nil {
		return // silently skip QR on error
	}
	const qrSize = 20.0
	x := pageW - marginR - qrSize
	y := marginT

	opts := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader("qr_"+dcNumber, opts, bytes.NewReader(png))

	savedY := pdf.GetY()
	pdf.ImageOptions("qr_"+dcNumber, x, y, qrSize, qrSize, false, opts, 0, "")
	pdf.SetY(savedY)
}

func drawCompanyHeader(pdf *fpdf.Fpdf, project *models.Project, company *models.CompanySettings, showEmail bool, qrReserved float64) {
	// Pull header fields from project settings; fall back to CompanySettings for legacy data.
	var name, addr, email, gstin, cin, pan string

	if project != nil {
		name = project.CompanyName
		addr = project.BillFromAddress
		email = project.CompanyEmail
		gstin = project.CompanyGSTIN
		cin = project.CompanyCIN
		pan = project.CompanyPAN
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

	hdrW := contentW - qrReserved

	setFont(pdf, "B", 14)
	setColor(pdf, colorBlack)
	pdf.CellFormat(hdrW, 6, strings.ToUpper(name), "", 1, "C", false, 0, "")

	// Strip any existing email line from the address so we can place it inline.
	addrLine := addr
	if email != "" {
		// Remove lines like "Email: x@y.com" or bare "x@y.com" from the address
		var cleaned []string
		for _, line := range strings.Split(addrLine, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if strings.Contains(trimmed, email) {
				continue
			}
			cleaned = append(cleaned, trimmed)
		}
		addrLine = strings.Join(cleaned, "\n")
	}
	// Append email inline
	if showEmail && email != "" {
		if addrLine != "" {
			addrLine += ",  Email: " + email
		} else {
			addrLine = "Email: " + email
		}
	}
	if addrLine != "" {
		setFont(pdf, "", 8)
		setColor(pdf, colorBlack)
		pdf.MultiCell(hdrW, 4, addrLine, "", "C", false)
	}

	// GSTIN, CIN, PAN line
	var regParts []string
	if gstin != "" {
		regParts = append(regParts, "GSTIN: "+gstin)
	}
	if cin != "" {
		regParts = append(regParts, "CIN: "+cin)
	}
	if pan != "" {
		regParts = append(regParts, "PAN: "+pan)
	}
	if len(regParts) > 0 {
		setFont(pdf, "B", 8)
		setColor(pdf, colorBlack)
		pdf.MultiCell(hdrW, 4, strings.Join(regParts, "    "), "", "C", false)
	}

	spacer(pdf, 1)
	drawHLine(pdf, pdf.GetY(), colorBlack, 0.5)
	spacer(pdf, 2)
}

// --- Section: DC Title ---

func drawDCTitle(pdf *fpdf.Fpdf, title string, doubleBorder bool) {
	y := pdf.GetY()

	if doubleBorder {
		drawHLine(pdf, y, colorBlack, 0.5)
		spacer(pdf, 2)
	}

	setFont(pdf, "B", 12)
	setColor(pdf, colorBlack)

	if doubleBorder {
		pdf.CellFormat(contentW, 6, strings.ToUpper(title), "", 1, "C", false, 0, "")
		spacer(pdf, 1)
		drawHLine(pdf, pdf.GetY(), colorBlack, 0.5)
	} else {
		setFont(pdf, "B", 10)
		pdf.CellFormat(contentW, 6, strings.ToUpper(title), "", 1, "C", false, 0, "")
	}

	spacer(pdf, 2)
	_ = y
}

// --- Section: DC Details + PO Details (side by side) ---

func drawDCAndPOGrid(pdf *fpdf.Fpdf, dc *models.DeliveryChallan, td *models.DCTransitDetails, project *models.Project, transferDCNumber ...string) {
	y := pdf.GetY()
	colW := contentW/2 - 2
	gap := 4.0
	innerW := colW - 2*cellPad

	// Resolve optional Transfer DC number
	refDCNumber := ""
	if len(transferDCNumber) > 0 {
		refDCNumber = transferDCNumber[0]
	}

	// Count lines for left box (DC details)
	leftLines := 1 // DC No (always)
	if refDCNumber != "" {
		leftLines++
	}
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
	if refDCNumber != "" {
		pdf.SetX(marginL + cellPad)
		kvRow(pdf, "Ref DC:", refDCNumber, 24, innerW)
	}
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

	pdf.SetY(y + boxH + 2)
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
	pdf.SetY(y + rowH + 1)

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
	pdf.SetY(y + rowH + 2)
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
	minH := 16.0
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
	setColor(pdf, colorBlack)
	pdf.CellFormat(innerW, 3.5, strings.ToUpper(title), "", 1, "L", false, 0, "")

	if len(lines) == 0 {
		setFont(pdf, "", 8)
		setColor(pdf, colorBlack)
		pdf.SetX(x + cellPad)
		pdf.CellFormat(innerW, lh, "Not specified", "", 1, "L", false, 0, "")
	} else {
		for i, line := range lines {
			if i == 0 {
				setFont(pdf, "B", 8)
				setColor(pdf, colorBlack)
			} else {
				setFont(pdf, "", 8)
				setColor(pdf, colorBlack)
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

// --- Section: Tax Summary ---

func drawTaxSummary(pdf *fpdf.Fpdf, totalTaxable, halfTax, roundOff, roundedTotal float64) {
	boxW := 80.0
	x := marginL + contentW - boxW
	y := pdf.GetY()

	setDrawColor(pdf, colorBlack)
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
			drawFilledRect(pdf, x, ry, boxW, rowH, colorWhite)
		}
		if i < len(rows)-1 {
			setDrawColor(pdf, colorBlack)
			pdf.SetLineWidth(0.1)
			pdf.Line(x, ry+rowH, x+boxW, ry+rowH)
		}

		if row.bold {
			setFont(pdf, "B", 8)
		} else {
			setFont(pdf, "", 8)
		}
		setColor(pdf, colorBlack)
		pdf.SetXY(x+cellPad, ry+0.5)
		pdf.CellFormat(boxW/2-cellPad, rowH-1, row.label, "", 0, "L", false, 0, "")

		if row.bold {
			setFont(pdf, "B", 8)
			setColor(pdf, colorBlack)
		} else {
			setFont(pdf, "B", 8)
			setColor(pdf, colorBlack)
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
	setColor(pdf, colorBlack)
	pdf.CellFormat(contentW-2*cellPad, 3.5, "AMOUNT IN WORDS", "", 1, "L", false, 0, "")

	pdf.SetX(marginL + cellPad)
	setFont(pdf, "B", 8)
	setColor(pdf, colorBlack)
	pdf.CellFormat(contentW-2*cellPad, 4, words, "", 1, "L", false, 0, "")

	pdf.SetY(y + h + 4)
}

// --- Section: Notes ---

func drawNotes(pdf *fpdf.Fpdf, notes string) {
	pdf.SetX(marginL)
	setFont(pdf, "B", 8)
	setColor(pdf, colorBlack)
	labelW := pdf.GetStringWidth("Note: ") + 1
	pdf.CellFormat(labelW, lineH, "Note: ", "", 0, "L", false, 0, "")

	setFont(pdf, "", 8)
	setColor(pdf, colorBlack)
	pdf.MultiCell(contentW-labelW, lineH, notes, "", "L", false)

	spacer(pdf, 3)
}

// estimateNotesHeight returns the approximate height the notes section will occupy.
func estimateNotesHeight(pdf *fpdf.Fpdf, notes string) float64 {
	if notes == "" {
		return 0
	}
	setFont(pdf, "", 8)
	labelW := 15.0 // approximate "Note: " width
	noteLines := pdf.SplitText(notes, contentW-labelW)
	h := float64(len(noteLines))*lineH + 3 // lines + spacer
	return h
}

// --- Section: Transit Signatures ---

func drawTransitSignatures(pdf *fpdf.Fpdf, company *models.CompanySettings, project *models.Project) {
	ensureSpace(pdf, 45)

	drawHLine(pdf, pdf.GetY(), colorBlack, 0.3)
	spacer(pdf, 4)

	y := pdf.GetY()
	colW := contentW / 2

	// Left: Receiver
	pdf.SetXY(marginL, y)
	setFont(pdf, "B", 8)
	setColor(pdf, colorBlack)
	pdf.CellFormat(colW, lineH, "Receiver's Signature", "", 1, "L", false, 0, "")
	spacer(pdf, 16)
	pdf.SetX(marginL)
	setDrawColor(pdf, colorBlack)
	pdf.SetLineWidth(0.3)
	pdf.Line(marginL, pdf.GetY(), marginL+60, pdf.GetY())
	spacer(pdf, 2)
	pdf.SetX(marginL)
	setFont(pdf, "", 8)
	setColor(pdf, colorBlack)
	pdf.CellFormat(colW, lineH, "Name: _________________________", "", 1, "L", false, 0, "")

	// Right: Authorized Signatory
	rightX := marginL + colW
	pdf.SetXY(rightX, y)
	if company != nil {
		setFont(pdf, "B", 8)
		setColor(pdf, colorBlack)
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
	setColor(pdf, colorBlack)
	pdf.CellFormat(colW, lineH, "Authorized Signatory", "", 1, "R", false, 0, "")

	// Print signatory details if configured
	if project != nil && project.SignatoryName != "" {
		setFont(pdf, "", 7)
		setColor(pdf, colorBlack)
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

// --- Section: Serial Numbers Annexure ---

func drawSerialNumbersAnnexure(pdf *fpdf.Fpdf, items []models.DCLineItem, annexureNumber int) {
	// Check if any line item has serial numbers
	hasSerials := false
	for _, li := range items {
		if len(li.SerialNumbers) > 0 {
			hasSerials = true
			break
		}
	}
	if !hasSerials {
		return
	}

	pdf.AddPage()

	// Annexure title — centered, matching Annexure-1 style
	setFont(pdf, "B", 11)
	setColor(pdf, colorBlack)
	pdf.CellFormat(contentW, 6, fmt.Sprintf("Annexure - %d", annexureNumber), "", 1, "C", false, 0, "")
	spacer(pdf, 3)

	// Subtitle
	setFont(pdf, "B", 9)
	setColor(pdf, colorBlack)
	pdf.CellFormat(contentW, lineH+1, "SERIAL NUMBERS", "", 1, "C", false, 0, "")
	spacer(pdf, 3)

	// Table: #, Product, Serial Numbers
	snoW := 8.0
	productW := 60.0
	serialsW := tblW - snoW - productW

	cols := []tableCol{
		{"#", snoW, "C"},
		{"Product", productW, "L"},
		{"Serial Numbers", serialsW, "L"},
	}

	drawTableHeaderRow(pdf, cols)

	row := 1
	for _, li := range items {
		if len(li.SerialNumbers) == 0 {
			continue
		}
		serialsText := strings.Join(li.SerialNumbers, ", ")
		values := []string{
			fmt.Sprintf("%d", row),
			li.ItemName,
			serialsText,
		}
		drawTableDataRow(pdf, cols, values, false)
		row++
	}
	spacer(pdf, 4)
}

// --- Section: Official Product Table ---

func drawOfficialProductTable(pdf *fpdf.Fpdf, items []models.DCLineItem) {
	cols := []tableCol{
		{"S.No", 12, "C"},
		{"Item Details", 103, "L"},
		{"Qty", 13, "C"},
		{"Serial Number", 40, "L"},
		{"Remarks", 22, "L"},
	}

	drawTableHeaderRow(pdf, cols)

	for i, li := range items {
		serials := strings.Join(li.SerialNumbers, "\n")

		// Combine item name, description, and brand/model into one stacked cell
		itemDetails := li.ItemName
		if li.ItemDescription != "" {
			itemDetails += "\n" + li.ItemDescription
		}
		if li.BrandModel != "" {
			itemDetails += "\nMake & Model: " + li.BrandModel
		}

		values := []string{
			fmt.Sprintf("%d", i+1),
			itemDetails,
			fmt.Sprintf("%d", li.Quantity),
			serials,
			"-",
		}
		drawTableDataRow(pdf, cols, values, false)
	}

	spacer(pdf, 3)
}

// --- Section: Acknowledgement ---

func drawAcknowledgement(pdf *fpdf.Fpdf) {
	y := pdf.GetY()

	pdf.SetXY(marginL, y)
	setFont(pdf, "B", 8)
	setColor(pdf, colorBlack)
	pdf.CellFormat(contentW, 5, "\"It is certified that the material is received in good condition.\"", "", 1, "L", false, 0, "")

	pdf.SetX(marginL)
	setFont(pdf, "B", 8)
	setColor(pdf, colorBlack)
	pdf.CellFormat(30, 5, "Date of Receipt: ", "", 0, "L", false, 0, "")
	setDrawColor(pdf, colorBlack)
	pdf.SetLineWidth(0.3)
	lineY := pdf.GetY() + 4.5
	pdf.Line(marginL+30, lineY, marginL+100, lineY)

	pdf.SetY(y + 12 + 3)
}

// --- Section: Official Signatures ---

func drawOfficialSignatures(pdf *fpdf.Fpdf, project *models.Project) {
	ensureSpace(pdf, 65)

	drawHLine(pdf, pdf.GetY(), colorBlack, 0.3)
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
		setColor(pdf, colorBlack)
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
			setDrawColor(pdf, colorBlack)
			pdf.SetLineWidth(0.3)
			pdf.SetDashPattern([]float64{2, 1}, 0)
			pdf.Rect(boxX, boxY, 55, 18, "D")
			pdf.SetDashPattern([]float64{}, 0) // reset dash

			setFont(pdf, "", 7)
			setColor(pdf, colorBlack)
			pdf.SetXY(boxX, boxY+6)
			pdf.CellFormat(55, 5, "Signature", "", 0, "C", false, 0, "")
		}

		pdf.SetXY(x, fieldsY)
		setFont(pdf, "", 8)
		setColor(pdf, colorBlack)

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
				setDrawColor(pdf, colorBlack)
				pdf.SetLineWidth(0.2)
				lx := x + 22
				ly := pdf.GetY() + 4.5
				pdf.Line(lx, ly, lx+45, ly)
			}
			pdf.Ln(5)
		}
	}

	drawSigBlock(marginL, y, "Department Official", false)
	drawSigBlock(marginL+colW, y, "FSSPL Representative", true)
}

// --- Table drawing helpers ---

func drawTableHeaderRow(pdf *fpdf.Fpdf, cols []tableCol) {
	headerFontSize := 6.0
	headerLineH := 3.5
	setFont(pdf, "B", headerFontSize)
	setColor(pdf, colorBlack)
	setFillColor(pdf, colorWhite)
	setDrawColor(pdf, colorBlack)
	pdf.SetLineWidth(0.3)

	// Single-line header — fixed height
	h := headerLineH + 2 // padding top+bottom

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
	padding := 1.5
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
		setFillColor(pdf, colorWhite)
	}

	setDrawColor(pdf, colorBlack)
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
			setColor(pdf, colorBlack)
		} else {
			setFont(pdf, "", 7)
			setColor(pdf, colorBlack)
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
