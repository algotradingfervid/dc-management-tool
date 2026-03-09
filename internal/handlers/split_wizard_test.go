package handlers

import (
	"testing"

	pagetransfer "github.com/narendhupati/dc-management-tool/components/pages/transfer_dcs"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ---------------------------------------------------------------------------
// Split Wizard Status Validation Tests
// ---------------------------------------------------------------------------

func TestValidateSplitWizardAccess(t *testing.T) {
	tests := []struct {
		name      string
		dcStatus  string
		wantError bool
		errMsg    string
	}{
		{
			name:      "issued status allows split wizard",
			dcStatus:  "issued",
			wantError: false,
		},
		{
			name:      "splitting status allows split wizard",
			dcStatus:  "splitting",
			wantError: false,
		},
		{
			name:      "draft status blocks split wizard",
			dcStatus:  "draft",
			wantError: true,
			errMsg:    "Transfer DC must be in 'issued' or 'splitting' status to create a split",
		},
		{
			name:      "split status blocks split wizard",
			dcStatus:  "split",
			wantError: true,
			errMsg:    "Transfer DC must be in 'issued' or 'splitting' status to create a split",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSplitWizardAccess(tt.dcStatus)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error message: want %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Destination Selection Validation Tests
// ---------------------------------------------------------------------------

func TestValidateSplitDestinationSelection(t *testing.T) {
	tests := []struct {
		name        string
		selectedIDs []string
		wantError   bool
		errMsg      string
	}{
		{
			name:        "at least one destination selected",
			selectedIDs: []string{"1", "2"},
			wantError:   false,
		},
		{
			name:        "single destination selected",
			selectedIDs: []string{"5"},
			wantError:   false,
		},
		{
			name:        "no destinations selected",
			selectedIDs: []string{},
			wantError:   true,
			errMsg:      "at least one destination must be selected",
		},
		{
			name:        "nil destinations",
			selectedIDs: nil,
			wantError:   true,
			errMsg:      "at least one destination must be selected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSplitDestinationSelection(tt.selectedIDs)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error message: want %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Transporter Validation Tests
// ---------------------------------------------------------------------------

func TestValidateSplitTransporter(t *testing.T) {
	tests := []struct {
		name            string
		transporterName string
		wantError       bool
		errMsg          string
	}{
		{
			name:            "transporter name provided",
			transporterName: "XYZ Logistics",
			wantError:       false,
		},
		{
			name:            "empty transporter name",
			transporterName: "",
			wantError:       true,
			errMsg:          "transporter name is required",
		},
		{
			name:            "whitespace-only transporter name",
			transporterName: "   ",
			wantError:       true,
			errMsg:          "transporter name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSplitTransporter(tt.transporterName)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error message: want %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Serial Parsing Tests
// ---------------------------------------------------------------------------

func TestParseSplitSerials(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected []string
	}{
		{
			name:     "newline separated serials",
			raw:      "SN001\nSN002\nSN003",
			expected: []string{"SN001", "SN002", "SN003"},
		},
		{
			name:     "trims whitespace",
			raw:      "  SN001 \n SN002  \n  SN003  ",
			expected: []string{"SN001", "SN002", "SN003"},
		},
		{
			name:     "skips empty lines",
			raw:      "SN001\n\nSN002\n\n\nSN003\n",
			expected: []string{"SN001", "SN002", "SN003"},
		},
		{
			name:     "empty string returns nil",
			raw:      "",
			expected: nil,
		},
		{
			name:     "only whitespace returns nil",
			raw:      "  \n  \n  ",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSplitSerials(tt.raw)
			if len(got) != len(tt.expected) {
				t.Fatalf("length: want %d, got %d", len(tt.expected), len(got))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("serial[%d]: want %q, got %q", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Split Wizard Data Carrier Tests
// ---------------------------------------------------------------------------

func TestSplitWizardData_DestinationIDsAsInts(t *testing.T) {
	data := &splitWizardData{
		DestinationIDs: []string{"1", "5", "12"},
	}

	got := data.destinationIDsAsInts()

	if len(got) != 3 {
		t.Fatalf("length: want 3, got %d", len(got))
	}
	expected := []int{1, 5, 12}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("id[%d]: want %d, got %d", i, expected[i], got[i])
		}
	}
}

func TestSplitWizardData_DestinationIDsAsInts_SkipsInvalid(t *testing.T) {
	data := &splitWizardData{
		DestinationIDs: []string{"1", "abc", "5"},
	}

	got := data.destinationIDsAsInts()

	if len(got) != 2 {
		t.Fatalf("length: want 2, got %d", len(got))
	}
	if got[0] != 1 || got[1] != 5 {
		t.Errorf("got %v, expected [1, 5]", got)
	}
}

// ---------------------------------------------------------------------------
// buildSplitProducts Tests
// ---------------------------------------------------------------------------

func TestBuildSplitProducts(t *testing.T) {
	destinations := []*models.TransferDCDestination{
		{
			ID: 1,
			Quantities: []models.TransferDCDestinationQty{
				{ProductID: 10, ProductName: "Product A"},
				{ProductID: 20, ProductName: "Product B"},
			},
		},
		{
			ID: 2,
			Quantities: []models.TransferDCDestinationQty{
				{ProductID: 10, ProductName: "Product A"},
				{ProductID: 20, ProductName: "Product B"},
			},
		},
	}

	products := buildSplitProducts(destinations)

	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	// Check deduplication
	ids := make(map[int]bool)
	for _, p := range products {
		ids[p.ID] = true
	}
	if !ids[10] || !ids[20] {
		t.Errorf("expected products 10 and 20, got %v", products)
	}
}

func TestBuildSplitProducts_Empty(t *testing.T) {
	products := buildSplitProducts(nil)
	if len(products) != 0 {
		t.Errorf("expected 0 products, got %d", len(products))
	}
}

// ---------------------------------------------------------------------------
// computeSelectedQty Tests
// ---------------------------------------------------------------------------

func TestComputeSelectedQty(t *testing.T) {
	destQuantities := map[int][]models.TransferDCDestinationQty{
		1: {
			{ProductID: 10, Quantity: 4},
			{ProductID: 20, Quantity: 2},
		},
		2: {
			{ProductID: 10, Quantity: 6},
			{ProductID: 20, Quantity: 3},
		},
		3: {
			{ProductID: 10, Quantity: 2},
			{ProductID: 20, Quantity: 1},
		},
	}

	// Select destinations 1 and 2
	result := computeSelectedQty([]int{1, 2}, destQuantities)

	if result[10] != 10 {
		t.Errorf("product 10: want 10, got %d", result[10])
	}
	if result[20] != 5 {
		t.Errorf("product 20: want 5, got %d", result[20])
	}
}

// ---------------------------------------------------------------------------
// buildSelectedSummary Tests
// ---------------------------------------------------------------------------

func TestBuildSelectedSummary(t *testing.T) {
	products := []pagetransfer.SplitProductInfo{
		{ID: 10, Name: "Product A"},
		{ID: 20, Name: "Product B"},
	}
	expectedQty := map[int]int{10: 14, 20: 7}

	summary := buildSelectedSummary(3, expectedQty, products)

	if summary != "3 destination(s), 14 × Product A, 7 × Product B" {
		t.Errorf("unexpected summary: %q", summary)
	}
}

func TestBuildSelectedSummary_SingleProduct(t *testing.T) {
	products := []pagetransfer.SplitProductInfo{
		{ID: 10, Name: "Product A"},
	}
	expectedQty := map[int]int{10: 5}

	summary := buildSelectedSummary(1, expectedQty, products)

	if summary != "1 destination(s), 5 × Product A" {
		t.Errorf("unexpected summary: %q", summary)
	}
}
