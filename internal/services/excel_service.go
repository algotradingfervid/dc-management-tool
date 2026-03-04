package services

import (
	"fmt"
	"math"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/xuri/excelize/v2"
)

func createBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
	}
}

// TransitDCExcelData holds data needed for Transit DC Excel generation.
type TransitDCExcelData struct {
	DC                  *models.DeliveryChallan
	LineItems           []models.DCLineItem
	Company             *models.CompanySettings
	Project             *models.Project
	ShipToAddress       *models.Address
	BillToAddress       *models.Address
	BillFromAddress     *models.Address
	DispatchFromAddress *models.Address
	TransitDetails      *models.DCTransitDetails
	ShipToConfig        *models.AddressListConfig
	BillToConfig        *models.AddressListConfig
	BillFromConfig      *models.AddressListConfig
	DispatchFromConfig  *models.AddressListConfig
	TotalTaxable        float64
	TotalTax            float64
	GrandTotal          float64
	RoundedTotal        float64
	RoundOff            float64
	HalfTax             float64
	TotalQty            int
	AmountInWords       string
}

// OfficialDCExcelData holds data needed for Official DC Excel generation.
type OfficialDCExcelData struct {
	DC                  *models.DeliveryChallan
	LineItems           []models.DCLineItem
	Company             *models.CompanySettings
	Project             *models.Project
	ShipToAddress       *models.Address
	BillToAddress       *models.Address
	BillFromAddress     *models.Address
	DispatchFromAddress *models.Address
	TransitDetails      *models.DCTransitDetails
	ShipToConfig        *models.AddressListConfig
	BillToConfig        *models.AddressListConfig
	BillFromConfig      *models.AddressListConfig
	DispatchFromConfig  *models.AddressListConfig
	TotalQty            int
}

// GenerateTransitDCExcel creates an Excel file matching the Transit DC PDF layout.
func GenerateTransitDCExcel(data *TransitDCExcelData) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Transit DC"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	if err = f.DeleteSheet("Sheet1"); err != nil {
		return nil, err
	}

	// Column widths (A-K: 11 columns matching PDF)
	_ = f.SetColWidth(sheet, "A", "A", 6)   // S.No
	_ = f.SetColWidth(sheet, "B", "B", 40)  // Description (merged: item name + brand/model + description)
	_ = f.SetColWidth(sheet, "C", "C", 25)  // Serials
	_ = f.SetColWidth(sheet, "D", "D", 8)   // UoM
	_ = f.SetColWidth(sheet, "E", "E", 10)  // HSN
	_ = f.SetColWidth(sheet, "F", "F", 8)   // Qty
	_ = f.SetColWidth(sheet, "G", "G", 14)  // Rate
	_ = f.SetColWidth(sheet, "H", "H", 14)  // Taxable
	_ = f.SetColWidth(sheet, "I", "I", 8)   // GST %
	_ = f.SetColWidth(sheet, "J", "J", 12)  // GST
	_ = f.SetColWidth(sheet, "K", "K", 14)  // Total

	lastCol := "K" // last column letter for merges

	// Styles
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	subHeaderStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	boldStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	if err != nil {
		return nil, err
	}
	tableHeaderStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Border:    createBorder(),
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	cellStyle, err := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	numStyle, err := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Size: 9},
		NumFmt:    4,
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	if err != nil {
		return nil, err
	}
	totalRowStyle, err := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Bold: true, Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F1F5F9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	totalRowNumStyle, err := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Bold: true, Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F1F5F9"}, Pattern: 1},
		NumFmt:    4,
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	if err != nil {
		return nil, err
	}
	totalLabelStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	if err != nil {
		return nil, err
	}
	totalValueStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		NumFmt:    4,
		Alignment: &excelize.Alignment{Horizontal: "right"},
		Border:    createBorder(),
	})
	if err != nil {
		return nil, err
	}
	addrLabelStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 7},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})
	if err != nil {
		return nil, err
	}
	addrStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		return nil, err
	}

	row := 1

	// --- Company header (matches PDF drawCompanyHeader with showEmail=false) ---
	name, addr, gstin, cin, pan := resolveCompanyHeader(data.Project, data.Company)

	if name != "" {
		_ = f.MergeCell(sheet, "A1", lastCol+"1")
		_ = f.SetCellValue(sheet, "A1", strings.ToUpper(name))
		_ = f.SetCellStyle(sheet, "A1", lastCol+"1", headerStyle)
		_ = f.SetRowHeight(sheet, 1, 28)

		if addr != "" {
			_ = f.MergeCell(sheet, "A2", lastCol+"2")
			_ = f.SetCellValue(sheet, "A2", addr)
			_ = f.SetCellStyle(sheet, "A2", lastCol+"2", subHeaderStyle)
		}

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
			_ = f.MergeCell(sheet, "A3", lastCol+"3")
			_ = f.SetCellValue(sheet, "A3", strings.Join(regParts, "    "))
			_ = f.SetCellStyle(sheet, "A3", lastCol+"3", subHeaderStyle)
		}
		row = 4
	}

	// --- Title ---
	row++
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DELIVERY CHALLAN")
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return nil, err
	}
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastCol, row), titleStyle)
	_ = f.SetRowHeight(sheet, row, 22)

	// --- DC details + PO details (matches PDF drawDCAndPOGrid) ---
	row += 2
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DC No:")
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.DC.DCNumber)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	if data.DC.ChallanDate != nil {
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "DC Date:")
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), *data.DC.ChallanDate)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)
	}

	if data.TransitDetails != nil {
		if data.TransitDetails.TransporterName != "" {
			row++
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Transporter:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.TransitDetails.TransporterName)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		}
		if data.TransitDetails.VehicleNumber != "" {
			_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "Vehicle:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), data.TransitDetails.VehicleNumber)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)
		}
		if data.TransitDetails.EwayBillNumber != "" {
			row++
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "E-Way Bill:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.TransitDetails.EwayBillNumber)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		}
	}

	row++
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Reverse Charge:")
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "No")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)

	// PO details (right side)
	poRow := row - 1
	if data.Project != nil {
		if data.Project.POReference != "" {
			_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", poRow), "PO Number:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", poRow), data.Project.POReference)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", poRow), fmt.Sprintf("G%d", poRow), boldStyle)
		}
		if data.Project.PODate != nil {
			poRow++
			_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", poRow), "PO Date:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", poRow), *data.Project.PODate)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", poRow), fmt.Sprintf("G%d", poRow), boldStyle)
		}
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "Project:")
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), data.Project.Name)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)
	}

	// --- Address Grid (2x2 matching PDF drawTransitAddressGrid) ---
	row += 2

	// Row 1: Bill From (A-E) / Bill To (G-K)
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "BILL FROM")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), addrLabelStyle)
	_ = f.MergeCell(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "BILL TO")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row), addrLabelStyle)

	row++
	billFromLines := addressLinesFiltered(data.BillFromAddress, data.BillFromConfig)
	if len(billFromLines) == 0 {
		billFromLines = companyAddressLines(data.Company)
	}
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), strings.Join(billFromLines, "\n"))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row+1), addrStyle)

	billToLines := addressLinesFiltered(data.BillToAddress, data.BillToConfig)
	_ = f.MergeCell(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), strings.Join(billToLines, "\n"))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row+1), addrStyle)
	_ = f.SetRowHeight(sheet, row, 30)

	row += 2

	// Row 2: Dispatch From (A-E) / Ship To (G-K)
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DISPATCH FROM")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), addrLabelStyle)
	_ = f.MergeCell(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "SHIP TO")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row), addrLabelStyle)

	row++
	dispatchFromLines := addressLinesFiltered(data.DispatchFromAddress, data.DispatchFromConfig)
	if len(dispatchFromLines) == 0 {
		dispatchFromLines = companyAddressLines(data.Company)
	}
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), strings.Join(dispatchFromLines, "\n"))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row+1), addrStyle)

	shipToLines := addressLinesFiltered(data.ShipToAddress, data.ShipToConfig)
	_ = f.MergeCell(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), strings.Join(shipToLines, "\n"))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row+1), addrStyle)
	_ = f.SetRowHeight(sheet, row, 30)

	row += 3

	// --- Product table (11 columns matching PDF) ---
	headers := []string{"S.No", "Description", "Serials", "UoM", "HSN", "Qty", "Rate", "Taxable", "GST %", "GST", "Total"}
	for i, h := range headers {
		col := string(rune('A' + i))
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), h)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), tableHeaderStyle)
	}
	_ = f.SetRowHeight(sheet, row, 28)

	// Product rows
	row++
	for i, li := range data.LineItems {
		// Merge description: item name + brand/model + description (matching PDF)
		desc := li.ItemName
		if li.BrandModel != "" {
			desc += "\n" + li.BrandModel
		}
		if li.ItemDescription != "" {
			desc += "\n" + li.ItemDescription
		}
		serials := strings.Join(li.SerialNumbers, "\n")

		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), desc)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), serials)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), li.UoM)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), li.HSNCode)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), li.Quantity)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), li.Rate)
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), li.TaxableAmount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%.0f%%", li.TaxPercentage))
		_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", row), li.TaxAmount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", row), li.TotalAmount)

		for col := 'A'; col <= 'K'; col++ {
			s := cellStyle
			if col == 'G' || col == 'H' || col == 'J' || col == 'K' {
				s = numStyle
			}
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), s)
		}
		row++
	}

	// Totals row (matching PDF)
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "Total")
	_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), data.TotalQty)
	_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), data.TotalTaxable)
	_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", row), data.TotalTax)
	_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", row), data.GrandTotal)
	for col := 'A'; col <= 'K'; col++ {
		s := totalRowStyle
		if col == 'H' || col == 'J' || col == 'K' {
			s = totalRowNumStyle
		}
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), s)
	}

	// --- Tax summary (matching PDF drawTaxSummary) ---
	row += 2
	summaryItems := []struct {
		label string
		value float64
	}{
		{"Taxable Value:", data.TotalTaxable},
		{"CGST:", data.HalfTax},
		{"SGST:", data.HalfTax},
		{"Round Off:", data.RoundOff},
		{"Invoice Value:", data.RoundedTotal},
	}
	for _, item := range summaryItems {
		_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", row), item.label)
		_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", row), item.value)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), totalLabelStyle)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), totalValueStyle)
		row++
	}

	// --- Amount in words ---
	row++
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Amount in Words: "+data.AmountInWords)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastCol, row), boldStyle)

	// --- Notes (matching PDF drawNotes) ---
	if data.TransitDetails != nil && data.TransitDetails.Notes != "" {
		row += 2
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Notes:")
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		row++
		_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastCol, row))
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), data.TransitDetails.Notes)
	}

	// --- Signature section (matching PDF drawTransitSignatures) ---
	row += 3

	// Left: Receiver's Signature
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Receiver's Signature")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)

	// Right: For [Company]
	companyName := ""
	if data.Company != nil {
		companyName = data.Company.Name
	}
	if data.Project != nil && data.Project.CompanyName != "" {
		companyName = data.Project.CompanyName
	}
	_ = f.MergeCell(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), "For "+companyName)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row), boldStyle)

	row += 4
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Name: _________________________")

	_ = f.MergeCell(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), "Authorized Signatory")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row), boldStyle)

	// Signatory details from project settings (matching PDF)
	if data.Project != nil {
		if data.Project.SignatoryName != "" {
			row++
			_ = f.MergeCell(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row))
			_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), data.Project.SignatoryName)
		}
		if data.Project.SignatoryDesignation != "" {
			row++
			_ = f.MergeCell(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row))
			_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), data.Project.SignatoryDesignation)
		}
		if data.Project.SignatoryMobile != "" {
			row++
			_ = f.MergeCell(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row))
			_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), "Ph: "+data.Project.SignatoryMobile)
		}
	}

	return f, nil
}

// resolveCompanyHeader returns company header fields from project settings with CompanySettings fallback.
func resolveCompanyHeader(project *models.Project, company *models.CompanySettings) (name, addr, gstin, cin, pan string) {
	if project != nil {
		name = project.CompanyName
		addr = project.BillFromAddress
		gstin = project.CompanyGSTIN
		cin = project.CompanyCIN
		pan = project.CompanyPAN
	}
	if name == "" && company != nil {
		name = company.Name
	}
	if addr == "" && company != nil {
		addr = fmt.Sprintf("%s, %s, %s %s", company.Address, company.City, company.State, company.Pincode)
	}
	if gstin == "" && company != nil {
		gstin = company.GSTIN
	}
	if cin == "" && company != nil {
		cin = company.CIN
	}
	return
}

// GenerateOfficialDCExcel creates an Excel file matching the Fervid-DC-V1 layout.
func GenerateOfficialDCExcel(data *OfficialDCExcelData) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Official DC"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	if err = f.DeleteSheet("Sheet1"); err != nil {
		return nil, err
	}

	// Column widths
	_ = f.SetColWidth(sheet, "A", "A", 6)
	_ = f.SetColWidth(sheet, "B", "B", 20)
	_ = f.SetColWidth(sheet, "C", "C", 25)
	_ = f.SetColWidth(sheet, "D", "D", 18)
	_ = f.SetColWidth(sheet, "E", "E", 8)
	_ = f.SetColWidth(sheet, "F", "F", 30)
	_ = f.SetColWidth(sheet, "G", "G", 15)

	// Styles
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	subHeaderStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	boldStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	if err != nil {
		return nil, err
	}
	tableHeaderStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Border:    createBorder(),
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	cellStyle, err := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		return nil, err
	}

	row := 1

	// --- Company header (project settings with CompanySettings fallback, includes email) ---
	name, addr, gstin, cin, pan := resolveCompanyHeader(data.Project, data.Company)
	var email string
	if data.Project != nil {
		email = data.Project.CompanyEmail
	}
	if email == "" && data.Company != nil {
		email = data.Company.Email
	}

	if name != "" {
		_ = f.MergeCell(sheet, "A1", "G1")
		_ = f.SetCellValue(sheet, "A1", name)
		_ = f.SetCellStyle(sheet, "A1", "G1", headerStyle)
		_ = f.SetRowHeight(sheet, 1, 28)

		if addr != "" {
			_ = f.MergeCell(sheet, "A2", "G2")
			_ = f.SetCellValue(sheet, "A2", addr)
			_ = f.SetCellStyle(sheet, "A2", "G2", subHeaderStyle)
		}

		// Email + registration line
		var regParts []string
		if email != "" {
			regParts = append(regParts, "Email: "+email)
		}
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
			_ = f.MergeCell(sheet, "A3", "G3")
			_ = f.SetCellValue(sheet, "A3", strings.Join(regParts, " | "))
			_ = f.SetCellStyle(sheet, "A3", "G3", subHeaderStyle)
		}
		row = 4
	}

	// Title
	row++
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DELIVERY CHALLAN")
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return nil, err
	}
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), titleStyle)
	_ = f.SetRowHeight(sheet, row, 22)

	// --- DC details (matching PDF drawDCAndPOGrid) ---
	row += 2
	// Left column: DC No, Date, Transporter, Vehicle, E-Way Bill, Reverse Charge
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DC Number:")
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.DC.DCNumber)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	if data.DC.ChallanDate != nil {
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "DC Date:")
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), *data.DC.ChallanDate)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), boldStyle)
	}

	if data.TransitDetails != nil {
		if data.TransitDetails.TransporterName != "" {
			row++
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Transporter:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.TransitDetails.TransporterName)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		}
		if data.TransitDetails.VehicleNumber != "" {
			_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Vehicle No:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), data.TransitDetails.VehicleNumber)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), boldStyle)
		}
		if data.TransitDetails.EwayBillNumber != "" {
			row++
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "E-Way Bill:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.TransitDetails.EwayBillNumber)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		}
	}

	row++
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Reverse Charge:")
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "No")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)

	// Right column: PO Number, PO Date, Project
	if data.Project != nil {
		if data.Project.POReference != "" {
			row++
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "PO Number:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.Project.POReference)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
			if data.Project.PODate != nil {
				_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "PO Date:")
				_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), *data.Project.PODate)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), boldStyle)
			}
		}
		row++
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Project:")
		_ = f.MergeCell(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("G%d", row))
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.Project.Name)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	}

	// Mandal info from ship-to address
	if data.ShipToAddress != nil {
		if mandal, ok := data.ShipToAddress.Data["mandal"]; ok && mandal != "" {
			row++
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Mandal/ULB:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), mandal)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
			if code, ok := data.ShipToAddress.Data["mandal_code"]; ok && code != "" {
				_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Mandal Code:")
				_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), code)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), boldStyle)
			}
		}
	}

	// --- 2x2 Address grid (Bill From, Bill To, Dispatch From, Ship To) ---
	addressWrapStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"},
	})

	// Row 1: Bill From / Bill To
	row += 2
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Bill From:")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Bill To:")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), boldStyle)
	row++

	billFromLines := addressLinesFiltered(data.BillFromAddress, data.BillFromConfig)
	if len(billFromLines) == 0 {
		billFromLines = companyAddressLines(data.Company)
	}
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), strings.Join(billFromLines, ", "))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row+1), addressWrapStyle)

	billToLines := addressLinesFiltered(data.BillToAddress, data.BillToConfig)
	_ = f.MergeCell(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), strings.Join(billToLines, ", "))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row+1), addressWrapStyle)
	row += 2

	// Row 2: Dispatch From / Ship To
	row++
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Dispatch From:")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Ship To:")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), boldStyle)
	row++

	dispatchFromLines := addressLinesFiltered(data.DispatchFromAddress, data.DispatchFromConfig)
	if len(dispatchFromLines) == 0 {
		dispatchFromLines = companyAddressLines(data.Company)
	}
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), strings.Join(dispatchFromLines, ", "))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row+1), addressWrapStyle)

	shipToLines := addressLinesFiltered(data.ShipToAddress, data.ShipToConfig)
	_ = f.MergeCell(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), strings.Join(shipToLines, ", "))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row+1), addressWrapStyle)
	row += 3

	// --- Product table (NO PRICING) ---
	headers := []string{"S.No", "Item Name", "Description", "Brand/Model No", "Qty", "Serial Number", "Remarks"}
	for i, h := range headers {
		col := string(rune('A' + i))
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), h)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), tableHeaderStyle)
	}
	_ = f.SetRowHeight(sheet, row, 28)

	row++
	for i, li := range data.LineItems {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), li.ItemName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), li.ItemDescription)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), li.BrandModel)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), li.Quantity)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), strings.Join(li.SerialNumbers, "\n"))
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "-")

		for col := 'A'; col <= 'G'; col++ {
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), cellStyle)
		}
		row++
	}

	// Acknowledgement
	row += 2
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "It is certified that the material is received in good condition.")

	row += 2
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Date of Receipt: _______________________")

	// --- Dual signature blocks ---
	row += 3
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "FSSPL Representative")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), boldStyle)

	_ = f.MergeCell(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Department Official")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row), boldStyle)

	// FSSPL Representative: fill signatory details from project settings
	row += 4
	if data.Project != nil && data.Project.SignatoryName != "" {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Name: "+data.Project.SignatoryName)
	} else {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Name: ___________________")
	}
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Name: ___________________")

	row++
	if data.Project != nil && data.Project.SignatoryDesignation != "" {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Designation: "+data.Project.SignatoryDesignation)
	} else {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Designation: ____________")
	}
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Designation: ____________")

	row++
	if data.Project != nil && data.Project.SignatoryMobile != "" {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Mobile: "+data.Project.SignatoryMobile)
	} else {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Mobile: _________________")
	}
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Mobile: _________________")

	return f, nil
}

// SanitizeDCFilename sanitizes a DC number for use in filenames.
func SanitizeDCFilename(dcNumber string) string {
	filename := strings.ReplaceAll(dcNumber, "/", "-")
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = strings.ReplaceAll(filename, "\\", "-")
	return "DC_" + filename
}

// formatAddressData formats address data map into a readable string.
func formatAddressData(data map[string]string) string {
	var parts []string
	for _, v := range data {
		v = strings.TrimSpace(v)
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ", ")
}

// CalcTransitTotals calculates tax totals for transit DC line items.
func CalcTransitTotals(lineItems []models.DCLineItem) (totalTaxable, totalTax, grandTotal, roundedTotal, roundOff, cgst, sgst float64) {
	for _, li := range lineItems {
		totalTaxable += li.TaxableAmount
		totalTax += li.TaxAmount
		grandTotal += li.TotalAmount
	}
	roundedTotal = math.Round(grandTotal)
	roundOff = roundedTotal - grandTotal
	halfTax := totalTax / 2.0
	cgst = math.Round(halfTax*100) / 100
	sgst = math.Round(halfTax*100) / 100
	return
}
