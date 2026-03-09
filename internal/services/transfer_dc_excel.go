package services

import (
	"fmt"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/xuri/excelize/v2"
)

// --- Transfer DC Excel Data Structs ---

// TransferDCExcelData holds all data needed for Transfer DC Excel generation.
type TransferDCExcelData struct {
	DC         *models.DeliveryChallan
	TransferDC *models.TransferDC
	Company    *models.CompanySettings
	Project    *models.Project
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

	// Destinations
	Destinations []TransferDCExcelDestination
	Products     []TransferDCExcelProduct

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

// TransferDCExcelDestination represents a destination row.
type TransferDCExcelDestination struct {
	Name       string
	Quantities map[int]int // productID → qty
}

// TransferDCExcelProduct identifies a product column.
type TransferDCExcelProduct struct {
	ID   int
	Name string
}

// GenerateTransferDCExcel creates an Excel file for a Transfer DC with three sheets.
func GenerateTransferDCExcel(data *TransferDCExcelData) (*excelize.File, error) {
	f := excelize.NewFile()

	// --- Sheet 1: Transfer DC ---
	sheet := "Transfer DC"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	if err = f.DeleteSheet("Sheet1"); err != nil {
		return nil, err
	}

	// Column widths (A-K: 11 columns matching PDF)
	_ = f.SetColWidth(sheet, "A", "A", 6)
	_ = f.SetColWidth(sheet, "B", "B", 40)
	_ = f.SetColWidth(sheet, "C", "C", 25)
	_ = f.SetColWidth(sheet, "D", "D", 8)
	_ = f.SetColWidth(sheet, "E", "E", 10)
	_ = f.SetColWidth(sheet, "F", "F", 8)
	_ = f.SetColWidth(sheet, "G", "G", 14)
	_ = f.SetColWidth(sheet, "H", "H", 14)
	_ = f.SetColWidth(sheet, "I", "I", 8)
	_ = f.SetColWidth(sheet, "J", "J", 12)
	_ = f.SetColWidth(sheet, "K", "K", 14)

	lastCol := "K"

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
	totalRowStyle, _ := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Bold: true, Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F1F5F9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	totalRowNumStyle, _ := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Bold: true, Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F1F5F9"}, Pattern: 1},
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
	addrLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 7},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})
	addrStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})

	row := 1

	// --- Company header ---
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
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "SPLIT TRANSFER DELIVERY CHALLAN")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastCol, row), titleStyle)
	_ = f.SetRowHeight(sheet, row, 22)

	// --- DC details + PO details ---
	row += 2
	dcNumber := ""
	if data.DC != nil {
		dcNumber = data.DC.DCNumber
	}
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DC No:")
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), dcNumber)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	if data.DC != nil && data.DC.ChallanDate != nil {
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "DC Date:")
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), *data.DC.ChallanDate)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)
	}

	if data.TransferDC != nil {
		if data.TransferDC.TransporterName != "" {
			row++
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Transporter:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.TransferDC.TransporterName)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		}
		if data.TransferDC.VehicleNumber != "" {
			_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "Vehicle:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), data.TransferDC.VehicleNumber)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)
		}
		if data.TransferDC.EwayBillNumber != "" {
			row++
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "E-Way Bill:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.TransferDC.EwayBillNumber)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
		}
		if data.TransferDC.DocketNumber != "" {
			_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "Docket:")
			_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), data.TransferDC.DocketNumber)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), boldStyle)
		}
	}

	row++
	reverseCharge := "No"
	if data.TransferDC != nil && data.TransferDC.ReverseCharge != "" {
		reverseCharge = data.TransferDC.ReverseCharge
	}
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Reverse Charge:")
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), reverseCharge)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)

	// Hub location
	if data.TransferDC != nil && data.TransferDC.HubAddressName != "" {
		row++
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Hub Location:")
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), data.TransferDC.HubAddressName)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	}

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

	// --- Address Grid ---
	row += 2
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
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "DISPATCH FROM")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), addrLabelStyle)
	_ = f.MergeCell(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "HUB LOCATION")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row), addrLabelStyle)

	row++
	dispatchFromLines := addressLinesFiltered(data.DispatchFromAddress, data.DispatchFromConfig)
	if len(dispatchFromLines) == 0 {
		dispatchFromLines = companyAddressLines(data.Company)
	}
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), strings.Join(dispatchFromLines, "\n"))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row+1), addrStyle)

	hubLines := addressLinesFiltered(data.HubAddress, data.HubConfig)
	_ = f.MergeCell(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row+1))
	_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), strings.Join(hubLines, "\n"))
	_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%s%d", lastCol, row+1), addrStyle)
	_ = f.SetRowHeight(sheet, row, 30)

	row += 3

	// --- Product table ---
	headers := []string{"S.No", "Description", "Serials", "UoM", "HSN", "Qty", "Rate", "Taxable", "GST %", "GST", "Total"}
	for i, h := range headers {
		col := string(rune('A' + i))
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), h)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), tableHeaderStyle)
	}
	_ = f.SetRowHeight(sheet, row, 28)

	row++
	for i, li := range data.LineItems {
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

	// Totals row
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

	// --- Tax summary ---
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

	// --- Signature section ---
	row += 3
	companyName := ""
	if data.Company != nil {
		companyName = data.Company.Name
	}
	if data.Project != nil && data.Project.CompanyName != "" {
		companyName = data.Project.CompanyName
	}
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Receiver's Signature")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), boldStyle)
	_ = f.MergeCell(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), "For "+companyName)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row), boldStyle)

	row += 4
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Name: _________________________")
	_ = f.MergeCell(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row))
	_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), "Authorized Signatory")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%s%d", lastCol, row), boldStyle)

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

	// --- Sheet 2: Destinations ---
	if err := writeDestinationsSheet(f, data); err != nil {
		return nil, err
	}

	// --- Sheet 3: Serial Numbers ---
	if err := writeSerialsSheet(f, data); err != nil {
		return nil, err
	}

	return f, nil
}

func writeDestinationsSheet(f *excelize.File, data *TransferDCExcelData) error {
	sheet := "Destinations"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
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
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Border: createBorder(),
		Font:   &excelize.Font{Bold: true, Size: 9},
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"F1F5F9"}, Pattern: 1},
	})

	_ = f.SetColWidth(sheet, "A", "A", 6)
	_ = f.SetColWidth(sheet, "B", "B", 40)

	row := 1
	// DC number reference
	dcNumber := ""
	if data.DC != nil {
		dcNumber = data.DC.DCNumber
	}
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	_ = f.SetCellValue(sheet, "A1", "DC Number:")
	_ = f.SetCellValue(sheet, "B1", dcNumber)
	_ = f.SetCellStyle(sheet, "A1", "A1", boldStyle)
	row = 3

	// Headers: #, Destination, then product names
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "#")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), headerStyle)
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "Destination")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), headerStyle)

	for i, p := range data.Products {
		col := string(rune('C' + i))
		_ = f.SetColWidth(sheet, col, col, 15)
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), p.Name)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), headerStyle)
	}

	// Destination rows
	totals := make(map[int]int)
	for i, dest := range data.Destinations {
		row++
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), cellStyle)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), dest.Name)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), cellStyle)

		for j, p := range data.Products {
			col := string(rune('C' + j))
			qty := dest.Quantities[p.ID]
			totals[p.ID] += qty
			_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), qty)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), cellStyle)
		}
	}

	// Totals row
	row++
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), totalStyle)
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "TOTAL")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), totalStyle)
	for j, p := range data.Products {
		col := string(rune('C' + j))
		_ = f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), totals[p.ID])
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), totalStyle)
	}

	return nil
}

func writeSerialsSheet(f *excelize.File, data *TransferDCExcelData) error {
	sheet := "Serial Numbers"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Border:    createBorder(),
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	cellStyle, _ := f.NewStyle(&excelize.Style{
		Border:    createBorder(),
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})

	_ = f.SetColWidth(sheet, "A", "A", 6)
	_ = f.SetColWidth(sheet, "B", "B", 30)
	_ = f.SetColWidth(sheet, "C", "C", 60)

	// DC number reference
	dcNumber := ""
	if data.DC != nil {
		dcNumber = data.DC.DCNumber
	}
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	_ = f.SetCellValue(sheet, "A1", "DC Number:")
	_ = f.SetCellValue(sheet, "B1", dcNumber)
	_ = f.SetCellStyle(sheet, "A1", "A1", boldStyle)

	row := 3
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "#")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), headerStyle)
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "Product")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), headerStyle)
	_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "Serial Numbers")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), headerStyle)

	sno := 0
	for _, li := range data.LineItems {
		if len(li.SerialNumbers) == 0 {
			continue
		}
		sno++
		row++
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), sno)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), cellStyle)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), li.ItemName)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), cellStyle)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), strings.Join(li.SerialNumbers, ", "))
		_ = f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), cellStyle)
	}

	return nil
}
