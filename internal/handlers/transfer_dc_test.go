package handlers

import (
	"testing"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ---------------------------------------------------------------------------
// Lifecycle Logic Tests
// ---------------------------------------------------------------------------

func TestIssueTransferDC_Validation(t *testing.T) {
	tests := []struct {
		name      string
		dcStatus  string
		wantError bool
		errMsg    string
	}{
		{
			name:      "draft can be issued",
			dcStatus:  models.DCStatusDraft,
			wantError: false,
		},
		{
			name:      "issued cannot be re-issued",
			dcStatus:  models.DCStatusIssued,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be issued",
		},
		{
			name:      "splitting cannot be issued",
			dcStatus:  models.DCStatusSplitting,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be issued",
		},
		{
			name:      "split cannot be issued",
			dcStatus:  models.DCStatusSplit,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be issued",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTransferDCIssue(tt.dcStatus)
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

func TestDeleteTransferDC_Validation(t *testing.T) {
	tests := []struct {
		name      string
		dcStatus  string
		wantError bool
		errMsg    string
	}{
		{
			name:      "draft can be deleted",
			dcStatus:  models.DCStatusDraft,
			wantError: false,
		},
		{
			name:      "issued cannot be deleted",
			dcStatus:  models.DCStatusIssued,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be deleted",
		},
		{
			name:      "splitting cannot be deleted",
			dcStatus:  models.DCStatusSplitting,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be deleted",
		},
		{
			name:      "split cannot be deleted",
			dcStatus:  models.DCStatusSplit,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTransferDCDelete(tt.dcStatus)
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

func TestComputeTransferDCStatus(t *testing.T) {
	tests := []struct {
		name            string
		numDestinations int
		numSplit        int
		currentStatus   string
		wantStatus      string
	}{
		{
			name:            "all split → split status",
			numDestinations: 25,
			numSplit:        25,
			currentStatus:   models.DCStatusSplitting,
			wantStatus:      models.DCStatusSplit,
		},
		{
			name:            "some split → splitting status",
			numDestinations: 25,
			numSplit:        10,
			currentStatus:   models.DCStatusIssued,
			wantStatus:      models.DCStatusSplitting,
		},
		{
			name:            "none split returns to issued",
			numDestinations: 25,
			numSplit:        0,
			currentStatus:   models.DCStatusSplitting,
			wantStatus:      models.DCStatusIssued,
		},
		{
			name:            "draft stays draft regardless of counts",
			numDestinations: 25,
			numSplit:        0,
			currentStatus:   models.DCStatusDraft,
			wantStatus:      models.DCStatusDraft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeTransferDCStatus(tt.currentStatus, tt.numDestinations, tt.numSplit)
			if got != tt.wantStatus {
				t.Errorf("computeTransferDCStatus(%q, %d, %d): want %q, got %q",
					tt.currentStatus, tt.numDestinations, tt.numSplit, tt.wantStatus, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Edit Validation Tests
// ---------------------------------------------------------------------------

func TestEditTransferDC_OnlyDraft(t *testing.T) {
	tests := []struct {
		name      string
		dcStatus  string
		wantError bool
		errMsg    string
	}{
		{
			name:      "draft can be edited",
			dcStatus:  models.DCStatusDraft,
			wantError: false,
		},
		{
			name:      "issued cannot be edited",
			dcStatus:  models.DCStatusIssued,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be edited",
		},
		{
			name:      "splitting cannot be edited",
			dcStatus:  models.DCStatusSplitting,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be edited",
		},
		{
			name:      "split cannot be edited",
			dcStatus:  models.DCStatusSplit,
			wantError: true,
			errMsg:    "only draft Transfer DCs can be edited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTransferDCEdit(tt.dcStatus)
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
// Destination Reconciliation Tests
// ---------------------------------------------------------------------------

func TestTransferEditReconcileDestinations(t *testing.T) {
	tests := []struct {
		name       string
		oldDestIDs []int
		newDestIDs []int
		wantKeep   []int
		wantAdd    []int
		wantDelete []int
	}{
		{
			name:       "no changes",
			oldDestIDs: []int{1, 2, 3},
			newDestIDs: []int{1, 2, 3},
			wantKeep:   []int{1, 2, 3},
			wantAdd:    nil,
			wantDelete: nil,
		},
		{
			name:       "add new destinations",
			oldDestIDs: []int{1, 2},
			newDestIDs: []int{1, 2, 3, 4},
			wantKeep:   []int{1, 2},
			wantAdd:    []int{3, 4},
			wantDelete: nil,
		},
		{
			name:       "remove destinations",
			oldDestIDs: []int{1, 2, 3},
			newDestIDs: []int{1},
			wantKeep:   []int{1},
			wantAdd:    nil,
			wantDelete: []int{2, 3},
		},
		{
			name:       "add and remove destinations",
			oldDestIDs: []int{1, 2, 3},
			newDestIDs: []int{2, 4, 5},
			wantKeep:   []int{2},
			wantAdd:    []int{4, 5},
			wantDelete: []int{1, 3},
		},
		{
			name:       "complete replacement",
			oldDestIDs: []int{1, 2},
			newDestIDs: []int{3, 4},
			wantKeep:   nil,
			wantAdd:    []int{3, 4},
			wantDelete: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSet := toIntSet(tt.oldDestIDs)
			newSet := toIntSet(tt.newDestIDs)
			keep := intersectSets(oldSet, newSet)
			add := subtractSets(newSet, oldSet)
			del := subtractSets(oldSet, newSet)

			assertIntSliceEqual(t, "keep", tt.wantKeep, keep)
			assertIntSliceEqual(t, "add", tt.wantAdd, add)
			assertIntSliceEqual(t, "delete", tt.wantDelete, del)
		})
	}
}

// ---------------------------------------------------------------------------
// Transfer Line Item Builder Tests
// ---------------------------------------------------------------------------

func TestBuildTransferLineItems(t *testing.T) {
	products := []*models.TemplateProductRow{
		{Product: models.Product{ID: 10, ItemName: "Widget A", PerUnitPrice: 100.0, GSTPercentage: 18.0}, DefaultQuantity: 5},
		{Product: models.Product{ID: 20, ItemName: "Widget B", PerUnitPrice: 200.0, GSTPercentage: 12.0}, DefaultQuantity: 3},
	}
	quantities := map[int]map[int]int{
		10: {100: 10, 200: 20},
		20: {100: 5, 200: 15},
	}
	serials := []transferEditSerialData{
		{ProductID: 10, AllSerials: []string{"SN1", "SN2", "SN3"}},
		{ProductID: 20, AllSerials: []string{"SN4", "SN5"}},
	}

	lineItems, serialsByLine := buildTransferEditLineItems(products, quantities, serials)

	if len(lineItems) != 2 {
		t.Fatalf("expected 2 line items, got %d", len(lineItems))
	}

	// Widget A: qty = 10+20 = 30, rate = 100, tax = 18%
	if lineItems[0].ProductID != 10 {
		t.Errorf("lineItems[0].ProductID: want 10, got %d", lineItems[0].ProductID)
	}
	if lineItems[0].Quantity != 30 {
		t.Errorf("lineItems[0].Quantity: want 30, got %d", lineItems[0].Quantity)
	}
	if lineItems[0].Rate != 100.0 {
		t.Errorf("lineItems[0].Rate: want 100.0, got %f", lineItems[0].Rate)
	}

	// Widget B: qty = 5+15 = 20, rate = 200, tax = 12%
	if lineItems[1].ProductID != 20 {
		t.Errorf("lineItems[1].ProductID: want 20, got %d", lineItems[1].ProductID)
	}
	if lineItems[1].Quantity != 20 {
		t.Errorf("lineItems[1].Quantity: want 20, got %d", lineItems[1].Quantity)
	}

	// Serials
	if len(serialsByLine) != 2 {
		t.Fatalf("expected 2 serial groups, got %d", len(serialsByLine))
	}
	if len(serialsByLine[0]) != 3 {
		t.Errorf("serialsByLine[0] len: want 3, got %d", len(serialsByLine[0]))
	}
	if len(serialsByLine[1]) != 2 {
		t.Errorf("serialsByLine[1] len: want 2, got %d", len(serialsByLine[1]))
	}
}

func TestBuildTransferLineItems_EmptySerials(t *testing.T) {
	products := []*models.TemplateProductRow{
		{Product: models.Product{ID: 10, ItemName: "Widget A", PerUnitPrice: 100.0, GSTPercentage: 18.0}, DefaultQuantity: 5},
	}
	quantities := map[int]map[int]int{
		10: {100: 10},
	}
	serials := []transferEditSerialData{
		{ProductID: 10, AllSerials: nil},
	}

	lineItems, serialsByLine := buildTransferEditLineItems(products, quantities, serials)

	if len(lineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(lineItems))
	}
	if lineItems[0].Quantity != 10 {
		t.Errorf("lineItems[0].Quantity: want 10, got %d", lineItems[0].Quantity)
	}
	if len(serialsByLine[0]) != 0 {
		t.Errorf("serialsByLine[0] should be empty, got %d", len(serialsByLine[0]))
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func assertIntSliceEqual(t *testing.T, label string, want, got []int) {
	t.Helper()
	wantSet := make(map[int]bool)
	for _, v := range want {
		wantSet[v] = true
	}
	gotSet := make(map[int]bool)
	for _, v := range got {
		gotSet[v] = true
	}
	if len(wantSet) != len(gotSet) {
		t.Errorf("%s: want %v, got %v", label, want, got)
		return
	}
	for k := range wantSet {
		if !gotSet[k] {
			t.Errorf("%s: missing %d; want %v, got %v", label, k, want, got)
		}
	}
}

func TestStatusBadgeClass(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{models.DCStatusDraft, "bg-gray-100 text-gray-800"},
		{models.DCStatusIssued, "bg-blue-100 text-blue-800"},
		{models.DCStatusSplitting, "bg-orange-100 text-orange-800"},
		{models.DCStatusSplit, "bg-green-100 text-green-800"},
		{"unknown", "bg-gray-100 text-gray-800"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := transferDCStatusBadgeClass(tt.status)
			if got != tt.want {
				t.Errorf("transferDCStatusBadgeClass(%q): want %q, got %q", tt.status, tt.want, got)
			}
		})
	}
}
