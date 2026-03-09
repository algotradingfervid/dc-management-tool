package services

import (
	"fmt"
	"testing"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// --- PDF Tests ---

func newTestTransferDCPDFData() *TransferDCPDFData {
	challanDate := "2026-03-09"
	poDate := "2026-01-01"
	return &TransferDCPDFData{
		Project: &models.Project{
			ID:                   1,
			Name:                 "Test Project",
			CompanyName:          "Test Company Ltd",
			CompanyGSTIN:         "36AABCT1234F1Z5",
			CompanyPAN:           "AABCT1234F",
			CompanyCIN:           "U12345TG2020PLC123456",
			CompanyEmail:         "info@test.com",
			BillFromAddress:      "123 Company St, Hyderabad",
			POReference:          "PO/2026/001",
			PODate:               &poDate,
			SignatoryName:        "John Doe",
			SignatoryDesignation: "Manager",
			SignatoryMobile:      "9876543210",
		},
		DC: &models.DeliveryChallan{
			ID:        1,
			ProjectID: 1,
			DCNumber:  "TST-STDC-2526-001",
			DCType:    "transfer",
			Status:    "issued",
			ChallanDate: &challanDate,
		},
		TransferDC: &models.TransferDC{
			ID:              1,
			DCID:            1,
			TaxType:         "cgst_sgst",
			ReverseCharge:   "No",
			TransporterName: "ABC Logistics",
			VehicleNumber:   "TS09AB9999",
			EwayBillNumber:  "EWB123456",
			DocketNumber:    "DK789",
			HubAddressName:  "District Warehouse",
		},
		HubAddress: &models.Address{
			ID:           10,
			DistrictName: "Hyderabad",
			MandalName:   "Secunderabad",
			Data:         map[string]string{"Address": "Warehouse Rd"},
		},
		BillFromAddress: &models.Address{
			ID:           20,
			DistrictName: "Hyderabad",
			Data:         map[string]string{"Address": "Company HQ"},
		},
		DispatchFromAddress: &models.Address{
			ID:           21,
			DistrictName: "Hyderabad",
			Data:         map[string]string{"Address": "Dispatch Center"},
		},
		BillToAddress: &models.Address{
			ID:           22,
			DistrictName: "Warangal",
			Data:         map[string]string{"Address": "Bill To Office"},
		},
		LineItems: []models.DCLineItem{
			{
				ID:            1,
				ItemName:      "UPS 1KVA",
				ItemDescription: "Online UPS",
				HSNCode:       "850440",
				UoM:           "Nos",
				Quantity:      100,
				Rate:          500,
				TaxPercentage: 18,
				TaxableAmount: 50000,
				TaxAmount:     9000,
				TotalAmount:   59000,
				SerialNumbers: []string{"SN001", "SN002", "SN003"},
			},
			{
				ID:            2,
				ItemName:      "Battery",
				HSNCode:       "850720",
				UoM:           "Nos",
				Quantity:      50,
				Rate:          300,
				TaxPercentage: 18,
				TaxableAmount: 15000,
				TaxAmount:     2700,
				TotalAmount:   17700,
				SerialNumbers: []string{"BN001", "BN002"},
			},
		},
		Destinations: []TransferDCPDFDestination{
			{
				Name:       "Mandal X, District Y",
				Address:    "School Building, Mandal X",
				Quantities: map[int]int{1: 40, 2: 20},
			},
			{
				Name:       "Mandal Z, District W",
				Address:    "Govt Office, Mandal Z",
				Quantities: map[int]int{1: 60, 2: 30},
			},
		},
		Products: []TransferDCPDFProduct{
			{ID: 1, Name: "UPS 1KVA"},
			{ID: 2, Name: "Battery"},
		},
		TotalTaxable: 65000,
		TotalTax:     11700,
		GrandTotal:   76700,
		RoundedTotal: 76700,
		RoundOff:     0,
		HalfTax:      5850,
		TotalQty:     150,
		AmountInWords: "Seventy Six Thousand Seven Hundred Rupees Only",
	}
}

func TestGenerateTransferDCPDF_HappyPath(t *testing.T) {
	data := newTestTransferDCPDFData()
	pdfBytes, err := GenerateTransferDCPDF(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCPDF() error: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("GenerateTransferDCPDF() returned empty PDF")
	}
	// Verify PDF header magic bytes
	if string(pdfBytes[:4]) != "%PDF" {
		t.Error("Output does not start with %PDF header")
	}
}

func TestGenerateTransferDCPDF_MinimalData(t *testing.T) {
	// Minimal data — no optional fields, no destinations, no serials
	data := &TransferDCPDFData{
		DC: &models.DeliveryChallan{
			DCNumber: "TST-STDC-2526-002",
			DCType:   "transfer",
			Status:   "draft",
		},
		TransferDC: &models.TransferDC{
			TaxType:       "igst",
			ReverseCharge: "No",
		},
	}

	pdfBytes, err := GenerateTransferDCPDF(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCPDF() with minimal data error: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("GenerateTransferDCPDF() returned empty PDF with minimal data")
	}
}

func TestGenerateTransferDCPDF_NilDestinations(t *testing.T) {
	data := newTestTransferDCPDFData()
	data.Destinations = nil
	data.Products = nil

	pdfBytes, err := GenerateTransferDCPDF(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCPDF() with nil destinations error: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("GenerateTransferDCPDF() returned empty PDF with nil destinations")
	}
}

func TestGenerateTransferDCPDF_ManyDestinations(t *testing.T) {
	data := newTestTransferDCPDFData()

	// 25 destinations to test page-break behavior
	data.Destinations = make([]TransferDCPDFDestination, 25)
	for i := range data.Destinations {
		data.Destinations[i] = TransferDCPDFDestination{
			Name:       fmt.Sprintf("Mandal %d, District %d", i+1, i+1),
			Address:    fmt.Sprintf("Location %d", i+1),
			Quantities: map[int]int{1: 4, 2: 2},
		}
	}

	pdfBytes, err := GenerateTransferDCPDF(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCPDF() with 25 destinations error: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("GenerateTransferDCPDF() returned empty PDF with 25 destinations")
	}
}

// --- Excel Tests ---

func newTestTransferDCExcelData() *TransferDCExcelData {
	challanDate := "2026-03-09"
	poDate := "2026-01-01"
	return &TransferDCExcelData{
		DC: &models.DeliveryChallan{
			ID:          1,
			DCNumber:    "TST-STDC-2526-001",
			DCType:      "transfer",
			Status:      "issued",
			ChallanDate: &challanDate,
		},
		TransferDC: &models.TransferDC{
			ID:              1,
			TaxType:         "cgst_sgst",
			ReverseCharge:   "No",
			TransporterName: "ABC Logistics",
			VehicleNumber:   "TS09AB9999",
			EwayBillNumber:  "EWB123456",
			DocketNumber:    "DK789",
			HubAddressName:  "District Warehouse",
		},
		HubAddress: &models.Address{
			ID:           10,
			DistrictName: "Hyderabad",
			MandalName:   "Secunderabad",
			Data:         map[string]string{"Address": "Warehouse Rd"},
		},
		BillFromAddress: &models.Address{
			ID:           20,
			DistrictName: "Hyderabad",
			Data:         map[string]string{"Address": "Company HQ"},
		},
		LineItems: []models.DCLineItem{
			{
				ItemName:      "UPS 1KVA",
				HSNCode:       "850440",
				UoM:           "Nos",
				Quantity:      100,
				Rate:          500,
				TaxPercentage: 18,
				TaxableAmount: 50000,
				TaxAmount:     9000,
				TotalAmount:   59000,
				SerialNumbers: []string{"SN001", "SN002"},
			},
			{
				ItemName:      "Battery",
				HSNCode:       "850720",
				UoM:           "Nos",
				Quantity:      50,
				Rate:          300,
				TaxPercentage: 18,
				TaxableAmount: 15000,
				TaxAmount:     2700,
				TotalAmount:   17700,
			},
		},
		Company: &models.CompanySettings{
			Name:    "Test Company",
			Address: "123 Test St",
			City:    "Hyderabad",
			State:   "Telangana",
			Pincode: "500001",
			GSTIN:   "36AABCT1234F1Z5",
		},
		Project: &models.Project{
			ID:           1,
			Name:         "Test Project",
			CompanyName:  "Test Company Ltd",
			CompanyGSTIN: "36AABCT1234F1Z5",
			POReference:  "PO/2026/001",
			PODate:       &poDate,
		},
		Destinations: []TransferDCExcelDestination{
			{
				Name:       "Mandal X",
				Quantities: map[int]int{1: 40, 2: 20},
			},
			{
				Name:       "Mandal Z",
				Quantities: map[int]int{1: 60, 2: 30},
			},
		},
		Products: []TransferDCExcelProduct{
			{ID: 1, Name: "UPS 1KVA"},
			{ID: 2, Name: "Battery"},
		},
		TotalTaxable:  65000,
		TotalTax:      11700,
		GrandTotal:    76700,
		RoundedTotal:  76700,
		RoundOff:      0,
		HalfTax:       5850,
		TotalQty:      150,
		AmountInWords: "Seventy Six Thousand Seven Hundred Rupees Only",
	}
}

func TestGenerateTransferDCExcel_HappyPath(t *testing.T) {
	data := newTestTransferDCExcelData()
	f, err := GenerateTransferDCExcel(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCExcel() error: %v", err)
	}

	// Verify main sheet exists
	sheets := f.GetSheetList()
	foundMain := false
	for _, s := range sheets {
		if s == "Transfer DC" {
			foundMain = true
		}
	}
	if !foundMain {
		t.Errorf("Expected sheet 'Transfer DC', got %v", sheets)
	}
}

func TestGenerateTransferDCExcel_CompanyHeader(t *testing.T) {
	data := newTestTransferDCExcelData()
	f, err := GenerateTransferDCExcel(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCExcel() error: %v", err)
	}

	val, _ := f.GetCellValue("Transfer DC", "A1")
	if val != "TEST COMPANY LTD" {
		t.Errorf("A1 = %q, want 'TEST COMPANY LTD'", val)
	}
}

func TestGenerateTransferDCExcel_LineItems(t *testing.T) {
	data := newTestTransferDCExcelData()
	f, err := GenerateTransferDCExcel(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCExcel() error: %v", err)
	}

	// Verify the Excel file has content by checking sheets aren't empty
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("Excel file has no sheets")
	}
}

func TestGenerateTransferDCExcel_DestinationsSheet(t *testing.T) {
	data := newTestTransferDCExcelData()
	f, err := GenerateTransferDCExcel(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCExcel() error: %v", err)
	}

	// Verify destinations sheet exists
	sheets := f.GetSheetList()
	foundDest := false
	for _, s := range sheets {
		if s == "Destinations" {
			foundDest = true
		}
	}
	if !foundDest {
		t.Errorf("Expected sheet 'Destinations', got %v", sheets)
	}
}

func TestGenerateTransferDCExcel_SerialsSheet(t *testing.T) {
	data := newTestTransferDCExcelData()
	f, err := GenerateTransferDCExcel(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCExcel() error: %v", err)
	}

	// Verify serials sheet exists
	sheets := f.GetSheetList()
	foundSerials := false
	for _, s := range sheets {
		if s == "Serial Numbers" {
			foundSerials = true
		}
	}
	if !foundSerials {
		t.Errorf("Expected sheet 'Serial Numbers', got %v", sheets)
	}
}

func TestGenerateTransferDCExcel_NoDestinations(t *testing.T) {
	data := newTestTransferDCExcelData()
	data.Destinations = nil
	data.Products = nil

	f, err := GenerateTransferDCExcel(data)
	if err != nil {
		t.Fatalf("GenerateTransferDCExcel() with no destinations error: %v", err)
	}

	// Should still create file without error
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("Excel file has no sheets")
	}
}
