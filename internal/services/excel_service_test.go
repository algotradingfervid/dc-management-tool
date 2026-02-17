package services

import (
	"math"
	"testing"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

func TestCalcTransitTotals(t *testing.T) {
	lineItems := []models.DCLineItem{
		{TaxableAmount: 10000, TaxAmount: 1800, TotalAmount: 11800},
		{TaxableAmount: 5000, TaxAmount: 900, TotalAmount: 5900},
	}

	totalTaxable, totalTax, grandTotal, roundedTotal, roundOff, cgst, sgst := CalcTransitTotals(lineItems)

	if totalTaxable != 15000 {
		t.Errorf("totalTaxable = %v, want 15000", totalTaxable)
	}
	if totalTax != 2700 {
		t.Errorf("totalTax = %v, want 2700", totalTax)
	}
	if grandTotal != 17700 {
		t.Errorf("grandTotal = %v, want 17700", grandTotal)
	}
	if roundedTotal != 17700 {
		t.Errorf("roundedTotal = %v, want 17700", roundedTotal)
	}
	if roundOff != 0 {
		t.Errorf("roundOff = %v, want 0", roundOff)
	}
	if cgst != 1350 {
		t.Errorf("cgst = %v, want 1350", cgst)
	}
	if sgst != 1350 {
		t.Errorf("sgst = %v, want 1350", sgst)
	}
}

func TestCalcTransitTotals_WithRounding(t *testing.T) {
	lineItems := []models.DCLineItem{
		{TaxableAmount: 100, TaxAmount: 18.33, TotalAmount: 118.33},
		{TaxableAmount: 200, TaxAmount: 36.67, TotalAmount: 236.67},
	}

	_, _, grandTotal, roundedTotal, roundOff, _, _ := CalcTransitTotals(lineItems)

	if grandTotal != 355.0 {
		t.Errorf("grandTotal = %v, want 355.0", grandTotal)
	}
	if roundedTotal != 355 {
		t.Errorf("roundedTotal = %v, want 355", roundedTotal)
	}
	if math.Abs(roundOff) > 0.5 {
		t.Errorf("roundOff = %v, expected within 0.5", roundOff)
	}
}

func TestCalcTransitTotals_EmptyItems(t *testing.T) {
	totalTaxable, totalTax, grandTotal, roundedTotal, roundOff, cgst, sgst := CalcTransitTotals(nil)

	if totalTaxable != 0 || totalTax != 0 || grandTotal != 0 || roundedTotal != 0 || roundOff != 0 || cgst != 0 || sgst != 0 {
		t.Error("Expected all zeros for empty line items")
	}
}

func TestCalcTransitTotals_SingleItem(t *testing.T) {
	lineItems := []models.DCLineItem{
		{TaxableAmount: 50000, TaxAmount: 9000, TotalAmount: 59000},
	}

	totalTaxable, totalTax, grandTotal, _, _, cgst, sgst := CalcTransitTotals(lineItems)

	if totalTaxable != 50000 {
		t.Errorf("totalTaxable = %v, want 50000", totalTaxable)
	}
	if totalTax != 9000 {
		t.Errorf("totalTax = %v, want 9000", totalTax)
	}
	if grandTotal != 59000 {
		t.Errorf("grandTotal = %v, want 59000", grandTotal)
	}
	if cgst != 4500 {
		t.Errorf("cgst = %v, want 4500", cgst)
	}
	if sgst != 4500 {
		t.Errorf("sgst = %v, want 4500", sgst)
	}
}

func TestGenerateTransitDCExcel(t *testing.T) {
	data := &TransitDCExcelData{
		DC: &models.DeliveryChallan{
			ID:       1,
			DCNumber: "FSS-TDC-2526-001",
			DCType:   "transit",
			Status:   "draft",
		},
		LineItems: []models.DCLineItem{
			{
				ItemName:      "UPS 1KVA",
				Quantity:      2,
				Rate:          10000,
				TaxPercentage: 18,
				TaxableAmount: 20000,
				TaxAmount:     3600,
				TotalAmount:   23600,
				HSNCode:       "850440",
				UoM:           "Nos",
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
			Name: "Test Project",
		},
		TotalTaxable:  20000,
		TotalTax:      3600,
		GrandTotal:    23600,
		RoundedTotal:  23600,
		RoundOff:      0,
		CGST:          1800,
		SGST:          1800,
		AmountInWords: "Twenty Three Thousand Six Hundred Rupees Only",
	}

	f, err := GenerateTransitDCExcel(data)
	if err != nil {
		t.Fatalf("GenerateTransitDCExcel() error: %v", err)
	}

	// Verify sheet exists
	sheets := f.GetSheetList()
	if len(sheets) == 0 || sheets[0] != "Transit DC" {
		t.Errorf("Expected sheet 'Transit DC', got %v", sheets)
	}

	// Verify company name in header
	val, _ := f.GetCellValue("Transit DC", "A1")
	if val != "Test Company" {
		t.Errorf("A1 = %q, want 'Test Company'", val)
	}
}

func TestSanitizeDCFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"FSS-TDC-2526-001", "DC_FSS-TDC-2526-001"},
		{"FS/GS/25-26/001", "DC_FS-GS-25-26-001"},
		{"DC 001", "DC_DC_001"},
	}

	for _, tt := range tests {
		result := SanitizeDCFilename(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeDCFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
