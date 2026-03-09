package services

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

// SplitShipmentParams holds all parameters needed to create a split from a Transfer DC.
type SplitShipmentParams struct {
	TransferDCID    int
	ParentDCID      int // delivery_challans.id of the Transfer DC
	ProjectID       int
	DestinationIDs  []int // IDs from transfer_dc_destinations table
	TransporterName string
	VehicleNumber   string
	EwayBillNumber  string
	DocketNumber    string
	Notes           string
	ProductSerials  []SplitProductSerials
	CreatedBy       int
}

// SplitProductSerials holds serial numbers for a single product in a split.
type SplitProductSerials struct {
	ProductID     int
	SerialNumbers []string
}

// SplitResult holds the result of a split operation.
type SplitResult struct {
	GroupID int
	SplitID int
}

// validateSplitDestinations checks that all selected destination IDs are valid and unsplit.
func validateSplitDestinations(selectedIDs []int, unsplitDestIDs map[int]bool) error {
	if len(selectedIDs) == 0 {
		return fmt.Errorf("at least one destination must be selected")
	}
	for _, id := range selectedIDs {
		if !unsplitDestIDs[id] {
			return fmt.Errorf("destination %d is not available for splitting (already split or does not belong to this Transfer DC)", id)
		}
	}
	return nil
}

// validateSplitSerials validates serial numbers for a split operation.
// parentSerials: map[productID]map[serialNumber]bool — all serials from parent DC
// usedSerials: map[serialNumber]bool — serials already consumed by previous splits
// expectedQty: map[productID]int — expected total quantity per product for this split
func validateSplitSerials(
	productSerials []SplitProductSerials,
	expectedQty map[int]int,
	parentSerials map[int]map[string]bool,
	usedSerials map[string]bool,
) map[string]string {
	errs := make(map[string]string)

	for _, ps := range productSerials {
		productKey := fmt.Sprintf("product_%d", ps.ProductID)

		// Check count matches expected quantity
		expected := expectedQty[ps.ProductID]
		if len(ps.SerialNumbers) != expected {
			errs[productKey+"_count"] = fmt.Sprintf("expected %d serials for product %d, got %d", expected, ps.ProductID, len(ps.SerialNumbers))
		}

		// Check each serial
		seen := make(map[string]bool)
		for _, sn := range ps.SerialNumbers {
			// Check for duplicates within this split
			if seen[sn] {
				errs[productKey+"_serials"] = fmt.Sprintf("duplicate serial number %q for product %d", sn, ps.ProductID)
				break
			}
			seen[sn] = true

			// Check serial belongs to parent
			productParentSerials := parentSerials[ps.ProductID]
			if productParentSerials == nil || !productParentSerials[sn] {
				errs[productKey+"_serials"] = fmt.Sprintf("serial %q does not belong to parent Transfer DC for product %d", sn, ps.ProductID)
				break
			}

			// Check serial not already used in another split
			if usedSerials[sn] {
				errs[productKey+"_serials"] = fmt.Sprintf("serial %q is already used in another split", sn)
				break
			}
		}
	}

	return errs
}

// validateSplitStatus checks that a Transfer DC is in a valid status for splitting.
func validateSplitStatus(status string) error {
	if status != "issued" && status != "splitting" {
		return fmt.Errorf("Transfer DC must be in 'issued' or 'splitting' status to create a split (current: %q)", status)
	}
	return nil
}

// CreateSplitShipment performs the split operation: creates a child shipment group
// (1 transit DC + N official DCs) from selected Transfer DC destinations.
func CreateSplitShipment(db *sql.DB, params SplitShipmentParams) (*SplitResult, error) {
	// === VALIDATION ===

	// 1. Get parent DC status and validate
	var parentStatus string
	err := db.QueryRow(`SELECT status FROM delivery_challans WHERE id = ?`, params.ParentDCID).Scan(&parentStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent DC: %w", err)
	}
	if err := validateSplitStatus(parentStatus); err != nil {
		return nil, err
	}

	// 2. Get unsplit destinations for this Transfer DC
	rows, err := db.Query(
		`SELECT id FROM transfer_dc_destinations WHERE transfer_dc_id = ? AND is_split = 0`,
		params.TransferDCID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get unsplit destinations: %w", err)
	}
	unsplitIDs := make(map[int]bool)
	for rows.Next() {
		var id int
		rows.Scan(&id)
		unsplitIDs[id] = true
	}
	rows.Close()

	if err := validateSplitDestinations(params.DestinationIDs, unsplitIDs); err != nil {
		return nil, err
	}

	// 3. Get quantities for selected destinations
	// Build map[productID] → totalQty and per-destination quantities
	type destQty struct {
		shipToAddrID int
		productID    int
		quantity     int
	}
	var allDestQtys []destQty
	productTotalQty := make(map[int]int) // total qty per product across selected dests

	// Also build map from destID → shipToAddressID
	destToAddr := make(map[int]int)

	placeholders := make([]string, len(params.DestinationIDs))
	args := make([]any, len(params.DestinationIDs))
	for i, id := range params.DestinationIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	qRows, err := db.Query(
		`SELECT d.id, d.ship_to_address_id, dq.product_id, dq.quantity
		 FROM transfer_dc_destinations d
		 INNER JOIN transfer_dc_destination_quantities dq ON d.id = dq.destination_id
		 WHERE d.id IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get destination quantities: %w", err)
	}
	for qRows.Next() {
		var destID, shipToAddr, productID, qty int
		qRows.Scan(&destID, &shipToAddr, &productID, &qty)
		allDestQtys = append(allDestQtys, destQty{shipToAddrID: shipToAddr, productID: productID, quantity: qty})
		productTotalQty[productID] += qty
		destToAddr[destID] = shipToAddr
	}
	qRows.Close()

	// 4. Get parent DC line items for rates and serial validation
	type parentLineItem struct {
		productID     int
		rate          float64
		taxPercentage float64
	}
	var parentItems []parentLineItem
	liRows, err := db.Query(
		`SELECT product_id, rate, tax_percentage FROM dc_line_items WHERE dc_id = ? ORDER BY line_order`,
		params.ParentDCID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent line items: %w", err)
	}
	for liRows.Next() {
		var pli parentLineItem
		liRows.Scan(&pli.productID, &pli.rate, &pli.taxPercentage)
		parentItems = append(parentItems, pli)
	}
	liRows.Close()

	// Build rate/tax lookup by product
	rateByProduct := make(map[int]float64)
	taxByProduct := make(map[int]float64)
	for _, pli := range parentItems {
		rateByProduct[pli.productID] = pli.rate
		taxByProduct[pli.productID] = pli.taxPercentage
	}

	// 5. Get all serials that originally belong to this Transfer DC for validation.
	//    This includes serials still on the parent DC AND serials already moved to child splits.
	parentSerials := make(map[int]map[string]bool)
	snRows, err := db.Query(
		`SELECT sn.product_id, sn.serial_number FROM serial_numbers sn
		 INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		 WHERE li.dc_id = ?
		 UNION
		 SELECT sn.product_id, sn.serial_number FROM serial_numbers sn
		 INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		 INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		 INNER JOIN shipment_groups sg ON dc.shipment_group_id = sg.id
		 INNER JOIN transfer_dc_splits ts ON sg.id = ts.shipment_group_id
		 WHERE ts.transfer_dc_id = ? AND dc.dc_type = 'transit'`,
		params.ParentDCID, params.TransferDCID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent serials: %w", err)
	}
	for snRows.Next() {
		var productID int
		var sn string
		snRows.Scan(&productID, &sn)
		if parentSerials[productID] == nil {
			parentSerials[productID] = make(map[string]bool)
		}
		parentSerials[productID][sn] = true
	}
	snRows.Close()

	// 6. Get serials already used in existing splits
	usedSerials := make(map[string]bool)
	usedRows, err := db.Query(
		`SELECT sn.serial_number FROM serial_numbers sn
		 INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		 INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		 INNER JOIN shipment_groups sg ON dc.shipment_group_id = sg.id
		 INNER JOIN transfer_dc_splits ts ON sg.id = ts.shipment_group_id
		 WHERE ts.transfer_dc_id = ? AND dc.dc_type = 'transit'`,
		params.TransferDCID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get used serials: %w", err)
	}
	for usedRows.Next() {
		var sn string
		usedRows.Scan(&sn)
		usedSerials[sn] = true
	}
	usedRows.Close()

	// 7. Validate serials
	serialErrs := validateSplitSerials(params.ProductSerials, productTotalQty, parentSerials, usedSerials)
	if len(serialErrs) > 0 {
		// Collect first error
		for _, msg := range serialErrs {
			return nil, fmt.Errorf("serial validation failed: %s", msg)
		}
	}

	// === CREATION (in transaction) ===
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 8. Get parent DC data for inheritance
	var challanDate, taxType, reverseCharge string
	var templateID int
	var billFromID, dispatchFromID, billToID sql.NullInt64
	err = tx.QueryRow(
		`SELECT COALESCE(dc.challan_date, ''), t.tax_type, t.reverse_charge, COALESCE(t.template_id, 0),
		        dc.bill_from_address_id, dc.dispatch_from_address_id, dc.bill_to_address_id
		 FROM delivery_challans dc
		 INNER JOIN transfer_dcs t ON dc.id = t.dc_id
		 WHERE dc.id = ?`, params.ParentDCID,
	).Scan(&challanDate, &taxType, &reverseCharge, &templateID, &billFromID, &dispatchFromID, &billToID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent DC data: %w", err)
	}

	// Parse DC date for financial year
	dcDate, err := time.Parse("2006-01-02", challanDate)
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

	// 9. Create child shipment group
	sgResult, err := tx.Exec(
		`INSERT INTO shipment_groups (project_id, template_id, num_sets, tax_type, reverse_charge, status, created_by, transfer_dc_id)
		 VALUES (?, ?, ?, ?, ?, 'draft', ?, ?)`,
		params.ProjectID, templateID, len(params.DestinationIDs), taxType, reverseCharge, params.CreatedBy, params.TransferDCID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create child shipment group: %w", err)
	}
	groupID64, _ := sgResult.LastInsertId()
	groupID := int(groupID64)

	// Build ship-to address list from selected destinations (in order)
	var shipToAddrs []int
	for _, destID := range params.DestinationIDs {
		shipToAddrs = append(shipToAddrs, destToAddr[destID])
	}

	// Use first destination address as transit ship-to
	transitShipToAddr := shipToAddrs[0]

	// Build per-product qty-by-location maps
	qtyByLocation := make(map[int]map[int]int) // map[productID]map[shipToAddr]qty
	for _, dq := range allDestQtys {
		if qtyByLocation[dq.productID] == nil {
			qtyByLocation[dq.productID] = make(map[int]int)
		}
		qtyByLocation[dq.productID][dq.shipToAddrID] += dq.quantity
	}

	// Build serial lookup by product
	serialByProduct := make(map[int][]string)
	for _, ps := range params.ProductSerials {
		serialByProduct[ps.ProductID] = ps.SerialNumbers
	}

	// 10. Create Transit DC
	transitSeq, err := getNextSequence(tx, params.ProjectID, DCTypeTransit, fy)
	if err != nil {
		return nil, fmt.Errorf("failed to get transit sequence: %w", err)
	}
	transitDCNumber := formatNumber(dcNumberFormat, dcPrefix, fy, DCTypeTransit, transitSeq, seqPadding)

	transitResult, err := tx.Exec(
		`INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, template_id, bill_to_address_id, ship_to_address_id, challan_date, created_by, shipment_group_id, bill_from_address_id, dispatch_from_address_id)
		 VALUES (?, ?, 'transit', 'draft', ?, ?, ?, ?, ?, ?, ?, ?)`,
		params.ProjectID, transitDCNumber, templateID,
		billToID, transitShipToAddr, challanDate,
		params.CreatedBy, groupID, billFromID, dispatchFromID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert child transit DC: %w", err)
	}
	transitDCID64, _ := transitResult.LastInsertId()
	transitDCID := int(transitDCID64)

	// Insert transit details
	_, err = tx.Exec(
		`INSERT INTO dc_transit_details (dc_id, transporter_name, vehicle_number, eway_bill_number, notes)
		 VALUES (?, ?, ?, ?, ?)`,
		transitDCID, params.TransporterName, params.VehicleNumber, params.EwayBillNumber, params.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert transit details: %w", err)
	}

	// Insert transit DC line items (total quantities across selected destinations)
	for lineOrder, pli := range parentItems {
		productID := pli.productID
		totalQty := productTotalQty[productID]
		if totalQty == 0 {
			continue
		}

		taxableAmount := pli.rate * float64(totalQty)
		taxAmount := taxableAmount * pli.taxPercentage / 100.0
		totalAmount := taxableAmount + taxAmount

		liResult, err := tx.Exec(
			`INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			transitDCID, productID, totalQty, pli.rate, pli.taxPercentage,
			math.Round(taxableAmount*100)/100,
			math.Round(taxAmount*100)/100,
			math.Round(totalAmount*100)/100,
			lineOrder+1,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert transit line item: %w", err)
		}
		liID64, _ := liResult.LastInsertId()
		liID := int(liID64)

		// Reassign serials from parent DC to this child transit DC line item
		for _, sn := range serialByProduct[productID] {
			res, err := tx.Exec(
				`UPDATE serial_numbers SET line_item_id = ? WHERE project_id = ? AND product_id = ? AND serial_number = ?`,
				liID, params.ProjectID, productID, sn,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to reassign serial number '%s': %w", sn, err)
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return nil, fmt.Errorf("serial number '%s' not found in project for product %d", sn, productID)
			}
		}
	}

	// 11. Create Official DCs (one per destination)
	for _, shipToID := range shipToAddrs {
		// Check if this location has any qty
		hasQty := false
		for _, pli := range parentItems {
			if qtyByLocation[pli.productID] != nil && qtyByLocation[pli.productID][shipToID] > 0 {
				hasQty = true
				break
			}
		}
		if !hasQty {
			continue
		}

		offSeq, err := getNextSequence(tx, params.ProjectID, DCTypeOfficial, fy)
		if err != nil {
			return nil, fmt.Errorf("failed to get official sequence: %w", err)
		}
		offDCNumber := formatNumber(dcNumberFormat, dcPrefix, fy, DCTypeOfficial, offSeq, seqPadding)

		offResult, err := tx.Exec(
			`INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, template_id, bill_to_address_id, ship_to_address_id, challan_date, created_by, shipment_group_id, bill_from_address_id, dispatch_from_address_id)
			 VALUES (?, ?, 'official', 'draft', ?, ?, ?, ?, ?, ?, ?, ?)`,
			params.ProjectID, offDCNumber, templateID,
			billToID, shipToID, challanDate,
			params.CreatedBy, groupID, billFromID, dispatchFromID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert official DC: %w", err)
		}
		offDCID64, _ := offResult.LastInsertId()
		offDCID := int(offDCID64)

		// Insert line items (per-location qty, no pricing, no serials)
		for lineOrder, pli := range parentItems {
			qty := 0
			if qtyByLocation[pli.productID] != nil {
				qty = qtyByLocation[pli.productID][shipToID]
			}
			_, err := tx.Exec(
				`INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order)
				 VALUES (?, ?, ?, 0, 0, 0, 0, 0, ?)`,
				offDCID, pli.productID, qty, lineOrder+1,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to insert official line item: %w", err)
			}
		}
	}

	// 12. Create split record
	var maxSplit sql.NullInt64
	tx.QueryRow(`SELECT MAX(split_number) FROM transfer_dc_splits WHERE transfer_dc_id = ?`, params.TransferDCID).Scan(&maxSplit)
	nextSplit := 1
	if maxSplit.Valid {
		nextSplit = int(maxSplit.Int64) + 1
	}

	splitResult, err := tx.Exec(
		`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by)
		 VALUES (?, ?, ?, ?)`,
		params.TransferDCID, groupID, nextSplit, params.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create split record: %w", err)
	}
	splitID64, _ := splitResult.LastInsertId()
	splitID := int(splitID64)

	// 13. Mark destinations as split
	destPlaceholders := make([]string, len(params.DestinationIDs))
	destArgs := []any{splitID}
	for i, id := range params.DestinationIDs {
		destPlaceholders[i] = "?"
		destArgs = append(destArgs, id)
	}
	_, err = tx.Exec(
		`UPDATE transfer_dc_destinations SET is_split = 1, split_group_id = ?
		 WHERE id IN (`+strings.Join(destPlaceholders, ",")+`)`,
		destArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mark destinations as split: %w", err)
	}

	// 14. Update split progress counters
	_, err = tx.Exec(
		`UPDATE transfer_dcs SET
			num_split = (SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ? AND is_split = 1),
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		params.TransferDCID, params.TransferDCID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update split progress: %w", err)
	}

	// 15. Determine and update Transfer DC status
	var numDest, numSplit int
	tx.QueryRow(`SELECT num_destinations, num_split FROM transfer_dcs WHERE id = ?`, params.TransferDCID).Scan(&numDest, &numSplit)

	var newStatus string
	if numSplit >= numDest {
		newStatus = "split"
	} else if numSplit > 0 {
		newStatus = "splitting"
	} else {
		newStatus = "issued"
	}

	_, err = tx.Exec(`UPDATE delivery_challans SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, newStatus, params.ParentDCID)
	if err != nil {
		return nil, fmt.Errorf("failed to update Transfer DC status: %w", err)
	}

	// Update shipment group with split_id reference
	_, err = tx.Exec(`UPDATE shipment_groups SET split_id = ? WHERE id = ?`, splitID, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to update shipment group split_id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit split transaction: %w", err)
	}

	return &SplitResult{
		GroupID: groupID,
		SplitID: splitID,
	}, nil
}
