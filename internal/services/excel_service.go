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
	DC             *models.DeliveryChallan
	LineItems      []models.DCLineItem
	Company        *models.CompanySettings
	Project        *models.Project
	ShipToAddress  *models.Address
	BillToAddress  *models.Address
	TotalTaxable   float64
	TotalTax       float64
	GrandTotal     float64
	RoundedTotal   float64
	RoundOff       float64
	CGST           float64
	SGST           float64
	AmountInWords  string
}

// OfficialDCExcelData holds data needed for Official DC Excel generation.
type OfficialDCExcelData struct {
	DC            *models.DeliveryChallan
	LineItems     []models.DCLineItem
	Company       *models.CompanySettings
	Project       *models.Project
	ShipToAddress *models.Address
	BillToAddress *models.Address
	TotalQty      int
}

// GenerateTransitDCExcel creates an Excel file matching the FSS-Transit-DC layout.
func GenerateTransitDCExcel(data *TransitDCExcelData) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Transit DC"
	f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")

	// Column widths
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 25)
	f.SetColWidth(sheet, "D", "D", 12)
	f.SetColWidth(sheet, "E", "E", 10)
	f.SetColWidth(sheet, "F", "F", 6)
	f.SetColWidth(sheet, "G", "G", 14)
	f.SetColWidth(sheet, "H", "H", 14)
	f.SetColWidth(sheet, "I", "I", 8)
	f.SetColWidth(sheet, "J", "J", 12)
	f.SetColWidth(sheet, "K", "K", 14)

	// Styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	subHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
	})
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	tableHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Border:    createBorder(),
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	cellStyle, _ := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	numStyle, _ := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Size: 9},
		NumFmt:    4,
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	totalLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	totalValueStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		NumFmt:    4,
		Alignment: &excelize.Alignment{Horizontal: "right"},
		Border:    createBorder(),
	})

	row := 1

	// Company header
	if data.Company != nil {
		f.MergeCell(sheet, "A1", "K1")
		f.SetCellValue(sheet, "A1", data.Company.Name)
		f.SetCellStyle(sheet, "A1", "K1", headerStyle)
		f.SetRowHeight(sheet, 1, 28)

		row = 2
		f.MergeCell(sheet, "A2", "K2")
		addr := fmt.Sprintf("%s, %s, %s %s", data.Company.Address, data.Company.City, data.Company.State, data.Company.Pincode)
		f.SetCellValue(sheet, "A2", addr)
		f.SetCellStyle(sheet, "A2", "K2", subHeaderStyle)

		row = 3
		info := fmt.Sprintf("GSTIN: %s", data.Company.GSTIN)
		if data.Company.Email != "" {
			info = fmt.Sprintf("Email: %s | %s", data.Company.Email, info)
		}
		if data.Company.CIN != "" {
			info += fmt.Sprintf(" | CIN: %s", data.Company.CIN)
		}
		f.MergeCell(sheet, "A3", "K3")
		f.SetCellValue(sheet, "A3", info)
		f.SetCellStyle(sheet, "A3", "K3", subHeaderStyle)
		row = 4
	}

	// Title
	row++
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("K%d", row))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DELIVERY CHALLAN")
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("K%d", row), titleStyle)
	f.SetRowHeight(sheet, row, 22)

	// DC details
	row += 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DC Number:")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.DC.DCNumber)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "DC Date:")
	if data.DC.ChallanDate != nil {
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), *data.DC.ChallanDate)
	}
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)

	row++
	if data.Project != nil {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Project:")
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.Project.Name)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)

		if data.Project.POReference != "" {
			row++
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "PO Number:")
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.Project.POReference)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
			if data.Project.PODate != nil {
				f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "PO Date:")
				f.SetCellValue(sheet, fmt.Sprintf("H%d", row), *data.Project.PODate)
				f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)
			}
		}
	}

	// Addresses
	row += 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Bill To:")
	f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "Ship To:")
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)

	row++
	if data.BillToAddress != nil {
		billTo := formatAddressData(data.BillToAddress.Data)
		f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("F%d", row+1))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), billTo)
	}
	if data.ShipToAddress != nil {
		shipTo := formatAddressData(data.ShipToAddress.Data)
		f.MergeCell(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("K%d", row+1))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), shipTo)
	}

	row += 3

	// Product table header
	headers := []string{"S.No", "Item Name", "Description", "Brand/Model", "HSN", "Qty", "Per Unit Price", "Taxable Value", "GST %", "GST Amount", "Total Value"}
	for i, h := range headers {
		col := string(rune('A' + i))
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), h)
		f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), tableHeaderStyle)
	}
	f.SetRowHeight(sheet, row, 28)

	// Product rows
	row++
	startRow := row
	for i, li := range data.LineItems {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), li.ItemName)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), li.ItemDescription)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), li.BrandModel)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), li.HSNCode)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), li.Quantity)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), li.Rate)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), li.TaxableAmount)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%.0f%%", li.TaxPercentage))
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), li.TaxAmount)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), li.TotalAmount)

		for col := 'A'; col <= 'K'; col++ {
			s := cellStyle
			if col >= 'G' && col <= 'K' && col != 'I' {
				s = numStyle
			}
			f.SetCellStyle(sheet, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), s)
		}
		row++
	}
	_ = startRow

	// Tax summary
	row++
	summaryItems := []struct {
		label string
		value float64
	}{
		{"Taxable Value:", data.TotalTaxable},
		{"CGST:", data.CGST},
		{"SGST:", data.SGST},
		{"Round Off:", data.RoundOff},
		{"Invoice Value:", data.RoundedTotal},
	}
	for _, item := range summaryItems {
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), item.label)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), item.value)
		f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), totalLabelStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), totalValueStyle)
		row++
	}

	// Amount in words
	row++
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("K%d", row))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Amount in Words: "+data.AmountInWords)
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("K%d", row), boldStyle)

	// Signature section
	row += 3
	if data.Company != nil {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Receiver's Signature")
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("For %s", data.Company.Name))
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), boldStyle)

		row += 4
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), "Authorised Signatory")
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), boldStyle)
	}

	return f, nil
}

// GenerateOfficialDCExcel creates an Excel file matching the Fervid-DC-V1 layout.
func GenerateOfficialDCExcel(data *OfficialDCExcelData) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Official DC"
	f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")

	// Column widths
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 25)
	f.SetColWidth(sheet, "D", "D", 18)
	f.SetColWidth(sheet, "E", "E", 8)
	f.SetColWidth(sheet, "F", "F", 30)
	f.SetColWidth(sheet, "G", "G", 15)

	// Styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	subHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
	})
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	tableHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Border:    createBorder(),
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	cellStyle, _ := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})

	row := 1

	// Company header
	if data.Company != nil {
		f.MergeCell(sheet, "A1", "G1")
		f.SetCellValue(sheet, "A1", data.Company.Name)
		f.SetCellStyle(sheet, "A1", "G1", headerStyle)
		f.SetRowHeight(sheet, 1, 28)

		row = 2
		f.MergeCell(sheet, "A2", "G2")
		addr := fmt.Sprintf("%s, %s, %s %s", data.Company.Address, data.Company.City, data.Company.State, data.Company.Pincode)
		f.SetCellValue(sheet, "A2", addr)
		f.SetCellStyle(sheet, "A2", "G2", subHeaderStyle)

		row = 3
		info := fmt.Sprintf("GSTIN: %s", data.Company.GSTIN)
		if data.Company.Email != "" {
			info = fmt.Sprintf("Email: %s | %s", data.Company.Email, info)
		}
		if data.Company.CIN != "" {
			info += fmt.Sprintf(" | CIN: %s", data.Company.CIN)
		}
		f.MergeCell(sheet, "A3", "G3")
		f.SetCellValue(sheet, "A3", info)
		f.SetCellStyle(sheet, "A3", "G3", subHeaderStyle)
		row = 4
	}

	// Title
	row++
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DELIVERY CHALLAN")
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), titleStyle)
	f.SetRowHeight(sheet, row, 22)

	// DC details
	row += 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DC Number:")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.DC.DCNumber)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "DC Date:")
	if data.DC.ChallanDate != nil {
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), *data.DC.ChallanDate)
	}
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), boldStyle)

	// Mandal info from ship-to address
	if data.ShipToAddress != nil {
		if mandal, ok := data.ShipToAddress.Data["mandal"]; ok && mandal != "" {
			row++
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Mandal/ULB:")
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), mandal)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
			if code, ok := data.ShipToAddress.Data["mandal_code"]; ok && code != "" {
				f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Mandal Code:")
				f.SetCellValue(sheet, fmt.Sprintf("F%d", row), code)
				f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), boldStyle)
			}
		}
	}

	// Project details
	if data.Project != nil {
		row += 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Project:")
		f.MergeCell(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("G%d", row))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.Project.Name)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)

		if data.Project.POReference != "" {
			row++
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "PO Number:")
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.Project.POReference)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		}
	}

	// Issued To
	if data.ShipToAddress != nil {
		row += 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Issued To:")
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		row++
		shipTo := formatAddressData(data.ShipToAddress.Data)
		f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row+1))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), shipTo)
		row += 2
	}

	row++

	// Product table (NO PRICING)
	headers := []string{"S.No", "Item Name", "Description", "Brand/Model No", "Qty", "Serial Number", "Remarks"}
	for i, h := range headers {
		col := string(rune('A' + i))
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), h)
		f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), tableHeaderStyle)
	}
	f.SetRowHeight(sheet, row, 28)

	row++
	for i, li := range data.LineItems {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), li.ItemName)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), li.ItemDescription)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), li.BrandModel)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), li.Quantity)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), strings.Join(li.SerialNumbers, "\n"))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "-")

		for col := 'A'; col <= 'G'; col++ {
			f.SetCellStyle(sheet, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), cellStyle)
		}
		row++
	}

	// Acknowledgement
	row += 2
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "It is certified that the material is received in good condition.")

	row += 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Date of Receipt: _______________________")

	// Dual signature blocks
	row += 3
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "FSSPL Representative")
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), boldStyle)

	f.MergeCell(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row))
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Department Official")
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row), boldStyle)

	row += 4
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Name: ___________________")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Name: ___________________")
	row++
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Designation: ____________")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Designation: ____________")
	row++
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Mobile: _________________")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Mobile: _________________")

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
