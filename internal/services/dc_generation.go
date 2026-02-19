package services

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ShipmentParams holds all parameters needed to create a shipment group with DCs.
type ShipmentParams struct {
	ProjectID             int
	TemplateID            int
	NumSets               int
	ChallanDate           string
	TaxType               string
	ReverseCharge         string
	TransporterName       string
	VehicleNumber         string
	EwayBillNumber        string
	DocketNumber          string
	BillFromAddressID     int
	DispatchFromAddressID int
	BillToAddressID       int
	ShipToAddressIDs      []int // N addresses, one per set
	TransitShipToAddrID   int   // which one for transit DC
	LineItems             []ShipmentLineItem
	CreatedBy             int
}

// ShipmentLineItem holds product info and serial assignments for a shipment.
type ShipmentLineItem struct {
	ProductID     int
	QtyPerSet     int
	Rate          float64
	TaxPercentage float64
	AllSerials    []string
	Assignments   map[int][]string // map[shipToAddressID][]serialNumbers
}

// ShipmentResult holds the result of creating a shipment group.
type ShipmentResult struct {
	GroupID     int
	TransitDC   *models.DeliveryChallan
	OfficialDCs []*models.DeliveryChallan
}

// CreateShipmentGroupDCs creates a shipment group with 1 transit DC + N official DCs in a transaction.
func CreateShipmentGroupDCs(db *sql.DB, params ShipmentParams) (*ShipmentResult, error) {
	if params.NumSets < 1 {
		return nil, fmt.Errorf("num_sets must be at least 1")
	}
	if params.ChallanDate == "" {
		return nil, fmt.Errorf("challan date is required")
	}
	if len(params.ShipToAddressIDs) != params.NumSets {
		return nil, fmt.Errorf("expected %d ship-to addresses, got %d", params.NumSets, len(params.ShipToAddressIDs))
	}
	if len(params.LineItems) == 0 {
		return nil, fmt.Errorf("at least one line item is required")
	}

	// Validate transit ship-to is one of the selected ship-to addresses
	validTransit := false
	for _, id := range params.ShipToAddressIDs {
		if id == params.TransitShipToAddrID {
			validTransit = true
			break
		}
	}
	if !validTransit {
		return nil, fmt.Errorf("transit ship-to address must be one of the selected ship-to addresses")
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Parse DC date for financial year
	dcDate, err := time.Parse("2006-01-02", params.ChallanDate)
	if err != nil {
		dcDate = time.Now()
	}

	// Get project settings
	var dcPrefix, dcNumberFormat string
	var seqPadding int
	err = tx.QueryRow("SELECT dc_prefix, dc_number_format, seq_padding FROM projects WHERE id = ?", params.ProjectID).
		Scan(&dcPrefix, &dcNumberFormat, &seqPadding)
	if err != nil {
		return nil, fmt.Errorf("failed to get project settings: %w", err)
	}
	if dcPrefix == "" {
		return nil, fmt.Errorf("project has no DC prefix set")
	}

	fy := GetFinancialYear(dcDate)

	// Create shipment group
	sgResult, err := tx.Exec(
		`INSERT INTO shipment_groups (project_id, template_id, num_sets, tax_type, reverse_charge, status, created_by)
		 VALUES (?, ?, ?, ?, ?, 'draft', ?)`,
		params.ProjectID, params.TemplateID, params.NumSets, params.TaxType, params.ReverseCharge, params.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create shipment group: %w", err)
	}
	groupID, err := sgResult.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment group ID: %w", err)
	}

	// --- Create Transit DC ---
	transitSeq, err := getNextSequence(tx, params.ProjectID, DCTypeTransit, fy)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit sequence: %w", err)
	}
	transitDCNumber := formatNumber(dcNumberFormat, dcPrefix, fy, DCTypeTransit, transitSeq, seqPadding)

	billToPtr := &params.BillToAddressID
	billFromPtr := &params.BillFromAddressID
	dispatchFromPtr := &params.DispatchFromAddressID
	shipmentGroupIDPtr := int(groupID)

	transitDC := &models.DeliveryChallan{
		ProjectID:             params.ProjectID,
		DCNumber:              transitDCNumber,
		DCType:                "transit",
		Status:                "draft",
		TemplateID:            &params.TemplateID,
		BillToAddressID:       billToPtr,
		ShipToAddressID:       params.TransitShipToAddrID,
		ChallanDate:           &params.ChallanDate,
		CreatedBy:             params.CreatedBy,
		ShipmentGroupID:       &shipmentGroupIDPtr,
		BillFromAddressID:     billFromPtr,
		DispatchFromAddressID: dispatchFromPtr,
	}

	result, err := tx.Exec(
		`INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, template_id, bill_to_address_id, ship_to_address_id, challan_date, created_by, shipment_group_id, bill_from_address_id, dispatch_from_address_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		transitDC.ProjectID, transitDC.DCNumber, transitDC.DCType, transitDC.Status,
		transitDC.TemplateID, transitDC.BillToAddressID, transitDC.ShipToAddressID,
		transitDC.ChallanDate, transitDC.CreatedBy, groupID,
		transitDC.BillFromAddressID, transitDC.DispatchFromAddressID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert transit DC: %w", err)
	}
	transitDCID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get transit DC ID: %w", err)
	}
	transitDC.ID = int(transitDCID)

	// Insert transit details
	_, err = tx.Exec(
		`INSERT INTO dc_transit_details (dc_id, transporter_name, vehicle_number, eway_bill_number, notes)
		 VALUES (?, ?, ?, ?, ?)`,
		transitDC.ID, params.TransporterName, params.VehicleNumber, params.EwayBillNumber, params.DocketNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert transit details: %w", err)
	}

	// Insert transit DC line items (all products with total quantities and all serials)
	for i, item := range params.LineItems {
		totalQty := item.QtyPerSet * params.NumSets
		taxableAmount := item.Rate * float64(totalQty)
		taxAmount := taxableAmount * item.TaxPercentage / 100.0
		totalAmount := taxableAmount + taxAmount

		liResult, err := tx.Exec(
			`INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			transitDC.ID, item.ProductID, totalQty, item.Rate, item.TaxPercentage,
			math.Round(taxableAmount*100)/100,
			math.Round(taxAmount*100)/100,
			math.Round(totalAmount*100)/100,
			i+1,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert transit line item: %w", err)
		}
		liID, err := liResult.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get line item ID: %w", err)
		}

		// Insert all serial numbers for this product
		for _, sn := range item.AllSerials {
			_, err = tx.Exec(
				`INSERT INTO serial_numbers (project_id, line_item_id, serial_number, product_id) VALUES (?, ?, ?, ?)`,
				params.ProjectID, int(liID), sn, item.ProductID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert serial number '%s': %w", sn, err)
			}
		}
	}

	// --- Create Official DCs (one per ship-to address) ---
	var officialDCs []*models.DeliveryChallan

	for _, shipToID := range params.ShipToAddressIDs {
		offSeq, err := getNextSequence(tx, params.ProjectID, DCTypeOfficial, fy)
		if err != nil {
			return nil, fmt.Errorf("failed to get official sequence: %w", err)
		}
		offDCNumber := formatNumber(dcNumberFormat, dcPrefix, fy, DCTypeOfficial, offSeq, seqPadding)

		offDC := &models.DeliveryChallan{
			ProjectID:       params.ProjectID,
			DCNumber:        offDCNumber,
			DCType:          "official",
			Status:          "draft",
			TemplateID:      &params.TemplateID,
			BillToAddressID: billToPtr,
			ShipToAddressID: shipToID,
			ChallanDate:     &params.ChallanDate,
			CreatedBy:       params.CreatedBy,
			ShipmentGroupID: &shipmentGroupIDPtr,
		}

		result, err := tx.Exec(
			`INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, template_id, bill_to_address_id, ship_to_address_id, challan_date, created_by, shipment_group_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			offDC.ProjectID, offDC.DCNumber, offDC.DCType, offDC.Status,
			offDC.TemplateID, offDC.BillToAddressID, offDC.ShipToAddressID,
			offDC.ChallanDate, offDC.CreatedBy, groupID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert official DC: %w", err)
		}
		offDCID, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get official DC ID: %w", err)
		}
		offDC.ID = int(offDCID)

		// Insert line items for this official DC (no pricing, no serials - serials tracked on transit DC only)
		for lineOrder, item := range params.LineItems {
			_, err := tx.Exec(
				`INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order)
				 VALUES (?, ?, ?, 0, 0, 0, 0, 0, ?)`,
				offDC.ID, item.ProductID, item.QtyPerSet, lineOrder+1,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert official line item: %w", err)
			}
			_ = item.Assignments[shipToID] // assignments tracked but not inserted as serial_numbers (those are on transit DC)
		}

		officialDCs = append(officialDCs, offDC)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &ShipmentResult{
		GroupID:     int(groupID),
		TransitDC:   transitDC,
		OfficialDCs: officialDCs,
	}, nil
}

// formatNumber formats a DC number using the project's configured format or legacy default.
func formatNumber(dcNumberFormat, prefix, fy, dcType string, seq, padding int) string {
	if dcNumberFormat != "" && dcNumberFormat != "{PREFIX}-{TYPE}-{FY}-{SEQ}" {
		return FormatDCNumberConfigurable(dcNumberFormat, prefix, prefix, fy, dcType, seq, padding)
	}
	return FormatDCNumber(prefix, fy, dcType, seq)
}
