package services

import (
	"database/sql"
	"fmt"
)

// DeleteSplitShipment undoes a split operation: deletes the child shipment group
// and all its DCs, frees serial numbers, resets destination split status, and
// recalculates Transfer DC status.
func DeleteSplitShipment(db *sql.DB, splitID int) error {
	// 1. Get split record
	var transferDCID, shipmentGroupID int
	err := db.QueryRow(
		`SELECT transfer_dc_id, shipment_group_id FROM transfer_dc_splits WHERE id = ?`, splitID,
	).Scan(&transferDCID, &shipmentGroupID)
	if err != nil {
		return fmt.Errorf("split not found: %w", err)
	}

	// 2. Get parent DC ID for status updates
	var parentDCID int
	err = db.QueryRow(`SELECT dc_id FROM transfer_dcs WHERE id = ?`, transferDCID).Scan(&parentDCID)
	if err != nil {
		return fmt.Errorf("transfer DC not found: %w", err)
	}

	// 3. Check if any child DCs have been issued — block deletion if so
	rows, err := db.Query(
		`SELECT dc_number, status FROM delivery_challans WHERE shipment_group_id = ?`, shipmentGroupID,
	)
	if err != nil {
		return fmt.Errorf("failed to get child DCs: %w", err)
	}
	defer rows.Close()

	var childDCIDs []int
	for rows.Next() {
		var dcNumber, status string
		rows.Scan(&dcNumber, &status)
		if status == "issued" {
			return fmt.Errorf("cannot delete split: child DC %s has been issued", dcNumber)
		}
	}
	rows.Close()

	// Re-query to get the IDs we need for deletion
	idRows, err := db.Query(
		`SELECT id FROM delivery_challans WHERE shipment_group_id = ?`, shipmentGroupID,
	)
	if err != nil {
		return fmt.Errorf("failed to get child DC IDs: %w", err)
	}
	for idRows.Next() {
		var id int
		idRows.Scan(&id)
		childDCIDs = append(childDCIDs, id)
	}
	idRows.Close()

	// 4. Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 5. Reset destination split status FIRST (before cascade deletes null out split_group_id)
	_, err = tx.Exec(
		`UPDATE transfer_dc_destinations SET is_split = 0, split_group_id = NULL
		 WHERE split_group_id = ?`, splitID,
	)
	if err != nil {
		return fmt.Errorf("failed to reset destinations: %w", err)
	}

	// 6. Reassign serials back to parent DC, then delete child DC data
	// First, get the parent DC's line items keyed by product_id
	parentLineItems := make(map[int]int) // product_id → parent line_item_id
	pliRows, err := tx.Query(
		`SELECT id, product_id FROM dc_line_items WHERE dc_id = ?`, parentDCID,
	)
	if err != nil {
		return fmt.Errorf("failed to get parent line items: %w", err)
	}
	for pliRows.Next() {
		var liID, productID int
		pliRows.Scan(&liID, &productID)
		parentLineItems[productID] = liID
	}
	pliRows.Close()

	for _, dcID := range childDCIDs {
		// Move serials back to parent DC line items
		snRows, err := tx.Query(
			`SELECT sn.id, sn.product_id FROM serial_numbers sn
			 INNER JOIN dc_line_items li ON sn.line_item_id = li.id
			 WHERE li.dc_id = ?`, dcID,
		)
		if err != nil {
			return fmt.Errorf("failed to get serials for DC %d: %w", dcID, err)
		}
		type snInfo struct{ id, productID int }
		var serials []snInfo
		for snRows.Next() {
			var s snInfo
			snRows.Scan(&s.id, &s.productID)
			serials = append(serials, s)
		}
		snRows.Close()

		for _, s := range serials {
			parentLIID, ok := parentLineItems[s.productID]
			if !ok {
				return fmt.Errorf("no parent line item found for product %d when reassigning serial %d", s.productID, s.id)
			}
			_, err = tx.Exec(`UPDATE serial_numbers SET line_item_id = ? WHERE id = ?`, parentLIID, s.id)
			if err != nil {
				return fmt.Errorf("failed to reassign serial %d back to parent: %w", s.id, err)
			}
		}

		_, err = tx.Exec(`DELETE FROM dc_line_items WHERE dc_id = ?`, dcID)
		if err != nil {
			return fmt.Errorf("failed to delete line items for DC %d: %w", dcID, err)
		}

		_, err = tx.Exec(`DELETE FROM dc_transit_details WHERE dc_id = ?`, dcID)
		if err != nil {
			return fmt.Errorf("failed to delete transit details for DC %d: %w", dcID, err)
		}

		_, err = tx.Exec(`DELETE FROM delivery_challans WHERE id = ?`, dcID)
		if err != nil {
			return fmt.Errorf("failed to delete DC %d: %w", dcID, err)
		}
	}

	// 7. Delete the split record
	_, err = tx.Exec(`DELETE FROM transfer_dc_splits WHERE id = ?`, splitID)
	if err != nil {
		return fmt.Errorf("failed to delete split record: %w", err)
	}

	// 8. Delete the shipment group
	_, err = tx.Exec(`DELETE FROM shipment_groups WHERE id = ?`, shipmentGroupID)
	if err != nil {
		return fmt.Errorf("failed to delete shipment group: %w", err)
	}

	// 9. Recalculate split progress counters
	_, err = tx.Exec(
		`UPDATE transfer_dcs SET
			num_split = (SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ? AND is_split = 1),
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		transferDCID, transferDCID,
	)
	if err != nil {
		return fmt.Errorf("failed to update split progress: %w", err)
	}

	// 10. Determine and update Transfer DC status
	var numDest, numSplit int
	tx.QueryRow(`SELECT num_destinations, num_split FROM transfer_dcs WHERE id = ?`, transferDCID).Scan(&numDest, &numSplit)

	var newStatus string
	if numSplit >= numDest {
		newStatus = "split"
	} else if numSplit > 0 {
		newStatus = "splitting"
	} else {
		newStatus = "issued"
	}

	_, err = tx.Exec(
		`UPDATE delivery_challans SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		newStatus, parentDCID,
	)
	if err != nil {
		return fmt.Errorf("failed to update Transfer DC status: %w", err)
	}

	return tx.Commit()
}
