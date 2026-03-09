package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ============================================================
// Transfer DC Core CRUD
// ============================================================

// CreateTransferDC creates a transfer_dcs record. The parent delivery_challan must already exist.
func CreateTransferDC(tdc *models.TransferDC) (int, error) {
	result, err := DB.ExecContext(ctx(),
		`INSERT INTO transfer_dcs (dc_id, hub_address_id, template_id, tax_type, reverse_charge,
            transporter_name, vehicle_number, eway_bill_number, docket_number, notes,
            num_destinations, num_split)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tdc.DCID, tdc.HubAddressID, nullInt64FromPtr(tdc.TemplateID),
		tdc.TaxType, tdc.ReverseCharge,
		nullStringFromStr(tdc.TransporterName), nullStringFromStr(tdc.VehicleNumber),
		nullStringFromStr(tdc.EwayBillNumber), nullStringFromStr(tdc.DocketNumber),
		nullStringFromStr(tdc.Notes),
		tdc.NumDestinations, tdc.NumSplit,
	)
	if err != nil {
		return 0, fmt.Errorf("CreateTransferDC: %w", err)
	}
	id, _ := result.LastInsertId()
	tdc.ID = int(id)
	return int(id), nil
}

// GetTransferDC retrieves a transfer DC by its ID with joined fields.
func GetTransferDC(id int) (*models.TransferDC, error) {
	row := DB.QueryRowContext(ctx(),
		`SELECT t.id, t.dc_id, t.hub_address_id, t.template_id, t.tax_type, t.reverse_charge,
            t.transporter_name, t.vehicle_number, t.eway_bill_number, t.docket_number, t.notes,
            t.num_destinations, t.num_split, t.created_at, t.updated_at,
            dc.dc_number, dc.status, dc.challan_date, dc.project_id,
            COALESCE(a.address_data, '{}') AS hub_address_name,
            COALESCE(tmpl.name, '') AS template_name
         FROM transfer_dcs t
         INNER JOIN delivery_challans dc ON t.dc_id = dc.id
         LEFT JOIN addresses a ON t.hub_address_id = a.id
         LEFT JOIN dc_templates tmpl ON t.template_id = tmpl.id
         WHERE t.id = ?`, id)
	return scanTransferDC(row)
}

// GetTransferDCByDCID retrieves a transfer DC by its parent delivery_challans.id.
func GetTransferDCByDCID(dcID int) (*models.TransferDC, error) {
	row := DB.QueryRowContext(ctx(),
		`SELECT t.id, t.dc_id, t.hub_address_id, t.template_id, t.tax_type, t.reverse_charge,
            t.transporter_name, t.vehicle_number, t.eway_bill_number, t.docket_number, t.notes,
            t.num_destinations, t.num_split, t.created_at, t.updated_at,
            dc.dc_number, dc.status, dc.challan_date, dc.project_id,
            COALESCE(a.address_data, '{}') AS hub_address_name,
            COALESCE(tmpl.name, '') AS template_name
         FROM transfer_dcs t
         INNER JOIN delivery_challans dc ON t.dc_id = dc.id
         LEFT JOIN addresses a ON t.hub_address_id = a.id
         LEFT JOIN dc_templates tmpl ON t.template_id = tmpl.id
         WHERE t.dc_id = ?`, dcID)
	return scanTransferDC(row)
}

// scanTransferDC scans a row into a TransferDC model.
func scanTransferDC(row *sql.Row) (*models.TransferDC, error) {
	var tdc models.TransferDC
	var templateID sql.NullInt64
	var transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes sql.NullString
	var createdAt, updatedAt sql.NullTime
	var challanDate sql.NullString

	err := row.Scan(
		&tdc.ID, &tdc.DCID, &tdc.HubAddressID, &templateID,
		&tdc.TaxType, &tdc.ReverseCharge,
		&transporterName, &vehicleNumber, &ewayBillNumber, &docketNumber, &notes,
		&tdc.NumDestinations, &tdc.NumSplit, &createdAt, &updatedAt,
		&tdc.DCNumber, &tdc.DCStatus, &challanDate, &tdc.ProjectID,
		&tdc.HubAddressName, &tdc.TemplateName,
	)
	if err != nil {
		return nil, err
	}
	if templateID.Valid {
		v := int(templateID.Int64)
		tdc.TemplateID = &v
	}
	if transporterName.Valid {
		tdc.TransporterName = transporterName.String
	}
	if vehicleNumber.Valid {
		tdc.VehicleNumber = vehicleNumber.String
	}
	if ewayBillNumber.Valid {
		tdc.EwayBillNumber = ewayBillNumber.String
	}
	if docketNumber.Valid {
		tdc.DocketNumber = docketNumber.String
	}
	if notes.Valid {
		tdc.Notes = notes.String
	}
	if createdAt.Valid {
		tdc.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		tdc.UpdatedAt = updatedAt.Time
	}
	if challanDate.Valid {
		tdc.ChallanDate = &challanDate.String
	}
	tdc.HubAddressName = models.FormatAddressJSON(tdc.HubAddressName)
	return &tdc, nil
}

// UpdateTransferDC updates mutable fields on a transfer DC.
func UpdateTransferDC(tdc *models.TransferDC) error {
	_, err := DB.ExecContext(ctx(),
		`UPDATE transfer_dcs SET
            hub_address_id = ?, template_id = ?, tax_type = ?, reverse_charge = ?,
            transporter_name = ?, vehicle_number = ?, eway_bill_number = ?,
            docket_number = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
         WHERE id = ?`,
		tdc.HubAddressID, nullInt64FromPtr(tdc.TemplateID),
		tdc.TaxType, tdc.ReverseCharge,
		nullStringFromStr(tdc.TransporterName), nullStringFromStr(tdc.VehicleNumber),
		nullStringFromStr(tdc.EwayBillNumber), nullStringFromStr(tdc.DocketNumber),
		nullStringFromStr(tdc.Notes),
		tdc.ID,
	)
	if err != nil {
		return fmt.Errorf("UpdateTransferDC: %w", err)
	}
	return nil
}

// DeleteTransferDC deletes a transfer DC. CASCADE handles destinations, quantities, and splits.
func DeleteTransferDC(id int) error {
	_, err := DB.ExecContext(ctx(), `DELETE FROM transfer_dcs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("DeleteTransferDC: %w", err)
	}
	return nil
}

// ============================================================
// Destination Management
// ============================================================

// DeleteTransferDCDestination deletes a single destination and its quantities.
func DeleteTransferDCDestination(destID int) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("DeleteTransferDCDestination begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Delete quantities first (may not cascade automatically depending on schema)
	if _, err := tx.ExecContext(ctx(),
		`DELETE FROM transfer_dc_destination_quantities WHERE destination_id = ?`, destID,
	); err != nil {
		return fmt.Errorf("DeleteTransferDCDestination delete quantities: %w", err)
	}

	if _, err := tx.ExecContext(ctx(),
		`DELETE FROM transfer_dc_destinations WHERE id = ?`, destID,
	); err != nil {
		return fmt.Errorf("DeleteTransferDCDestination delete destination: %w", err)
	}

	return tx.Commit()
}

// AddTransferDCDestinations inserts multiple destinations in a batch.
func AddTransferDCDestinations(transferDCID int, shipToAddressIDs []int) error {
	if len(shipToAddressIDs) == 0 {
		return nil
	}
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("AddTransferDCDestinations begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(
		`INSERT INTO transfer_dc_destinations (transfer_dc_id, ship_to_address_id) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("AddTransferDCDestinations prepare: %w", err)
	}
	defer stmt.Close()

	for _, addrID := range shipToAddressIDs {
		if _, err := stmt.Exec(transferDCID, addrID); err != nil {
			return fmt.Errorf("AddTransferDCDestinations insert addr %d: %w", addrID, err)
		}
	}

	// Update num_destinations counter
	if _, err := tx.Exec(
		`UPDATE transfer_dcs SET num_destinations = (
            SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ?
        ), updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		transferDCID, transferDCID,
	); err != nil {
		return fmt.Errorf("AddTransferDCDestinations update count: %w", err)
	}

	return tx.Commit()
}

// GetTransferDCDestinations retrieves all destinations for a transfer DC.
func GetTransferDCDestinations(transferDCID int) ([]*models.TransferDCDestination, error) {
	return getDestinations(transferDCID, "")
}

// GetUnsplitDestinations retrieves destinations not yet assigned to a split group.
func GetUnsplitDestinations(transferDCID int) ([]*models.TransferDCDestination, error) {
	return getDestinations(transferDCID, "AND d.is_split = 0")
}

// GetSplitDestinations retrieves destinations that have been split.
func GetSplitDestinations(transferDCID int) ([]*models.TransferDCDestination, error) {
	return getDestinations(transferDCID, "AND d.is_split = 1")
}

func getDestinations(transferDCID int, extraWhere string) ([]*models.TransferDCDestination, error) {
	query := fmt.Sprintf(
		`SELECT d.id, d.transfer_dc_id, d.ship_to_address_id, d.split_group_id, d.is_split, d.created_at,
            COALESCE(a.address_data, '{}') AS address_name
         FROM transfer_dc_destinations d
         LEFT JOIN addresses a ON d.ship_to_address_id = a.id
         WHERE d.transfer_dc_id = ? %s
         ORDER BY d.id`, extraWhere)

	rows, err := DB.QueryContext(ctx(), query, transferDCID)
	if err != nil {
		return nil, fmt.Errorf("getDestinations: %w", err)
	}
	defer rows.Close()

	var dests []*models.TransferDCDestination
	for rows.Next() {
		d := &models.TransferDCDestination{}
		var splitGroupID sql.NullInt64
		var isSplit int
		var createdAt sql.NullTime
		if err := rows.Scan(&d.ID, &d.TransferDCID, &d.ShipToAddressID, &splitGroupID, &isSplit, &createdAt, &d.AddressName); err != nil {
			return nil, fmt.Errorf("getDestinations scan: %w", err)
		}
		d.IsSplit = isSplit == 1
		if splitGroupID.Valid {
			v := int(splitGroupID.Int64)
			d.SplitGroupID = &v
		}
		if createdAt.Valid {
			d.CreatedAt = createdAt.Time
		}
		d.AddressName = models.FormatAddressJSON(d.AddressName)
		dests = append(dests, d)
	}

	// Batch-load quantities for all destinations
	if len(dests) > 0 {
		ids := make([]int, len(dests))
		for i, d := range dests {
			ids[i] = d.ID
		}
		qtyMap, err := GetQuantitiesForDestinations(ids)
		if err == nil {
			for _, d := range dests {
				if qtys, ok := qtyMap[d.ID]; ok {
					d.Quantities = qtys
				}
			}
		}
	}

	return dests, nil
}

// UpdateDestinationSplitStatus marks destinations as split or un-split.
func UpdateDestinationSplitStatus(destinationIDs []int, splitGroupID *int, isSplit bool) error {
	if len(destinationIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(destinationIDs))
	args := make([]any, 0, len(destinationIDs)+2)

	isSplitVal := 0
	if isSplit {
		isSplitVal = 1
	}
	args = append(args, isSplitVal, nullInt64FromPtr(splitGroupID))
	for i, id := range destinationIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	_, err := DB.ExecContext(ctx(),
		`UPDATE transfer_dc_destinations SET is_split = ?, split_group_id = ?
         WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return fmt.Errorf("UpdateDestinationSplitStatus: %w", err)
	}
	return nil
}

// ============================================================
// Quantity Grid
// ============================================================

// SetDestinationQuantities upserts product quantities for a destination.
func SetDestinationQuantities(destinationID int, quantities []models.TransferDCDestinationQty) error {
	if len(quantities) == 0 {
		return nil
	}
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("SetDestinationQuantities begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(
		`INSERT INTO transfer_dc_destination_quantities (destination_id, product_id, quantity)
         VALUES (?, ?, ?)
         ON CONFLICT (destination_id, product_id) DO UPDATE SET quantity = excluded.quantity`)
	if err != nil {
		return fmt.Errorf("SetDestinationQuantities prepare: %w", err)
	}
	defer stmt.Close()

	for _, q := range quantities {
		if _, err := stmt.Exec(destinationID, q.ProductID, q.Quantity); err != nil {
			return fmt.Errorf("SetDestinationQuantities upsert product %d: %w", q.ProductID, err)
		}
	}
	return tx.Commit()
}

// GetDestinationQuantities retrieves quantities for a single destination with product info.
func GetDestinationQuantities(destinationID int) ([]models.TransferDCDestinationQty, error) {
	rows, err := DB.QueryContext(ctx(),
		`SELECT dq.id, dq.destination_id, dq.product_id, dq.quantity,
            COALESCE(p.item_name, '') AS product_name
         FROM transfer_dc_destination_quantities dq
         LEFT JOIN products p ON dq.product_id = p.id
         WHERE dq.destination_id = ?
         ORDER BY dq.product_id`, destinationID)
	if err != nil {
		return nil, fmt.Errorf("GetDestinationQuantities: %w", err)
	}
	defer rows.Close()

	var qtys []models.TransferDCDestinationQty
	for rows.Next() {
		var q models.TransferDCDestinationQty
		if err := rows.Scan(&q.ID, &q.DestinationID, &q.ProductID, &q.Quantity, &q.ProductName); err != nil {
			return nil, fmt.Errorf("GetDestinationQuantities scan: %w", err)
		}
		qtys = append(qtys, q)
	}
	return qtys, nil
}

// GetQuantityGrid retrieves the full quantity grid for a Transfer DC.
// Returns map[destinationID]map[productID]quantity.
func GetQuantityGrid(transferDCID int) (map[int]map[int]int, error) {
	rows, err := DB.QueryContext(ctx(),
		`SELECT dq.destination_id, dq.product_id, dq.quantity
         FROM transfer_dc_destination_quantities dq
         INNER JOIN transfer_dc_destinations d ON dq.destination_id = d.id
         WHERE d.transfer_dc_id = ?`, transferDCID)
	if err != nil {
		return nil, fmt.Errorf("GetQuantityGrid: %w", err)
	}
	defer rows.Close()

	grid := make(map[int]map[int]int)
	for rows.Next() {
		var destID, productID, qty int
		if err := rows.Scan(&destID, &productID, &qty); err != nil {
			return nil, fmt.Errorf("GetQuantityGrid scan: %w", err)
		}
		if grid[destID] == nil {
			grid[destID] = make(map[int]int)
		}
		grid[destID][productID] = qty
	}
	return grid, nil
}

// GetQuantitiesForDestinations retrieves quantities grouped by destination ID.
func GetQuantitiesForDestinations(destinationIDs []int) (map[int][]models.TransferDCDestinationQty, error) {
	if len(destinationIDs) == 0 {
		return make(map[int][]models.TransferDCDestinationQty), nil
	}
	placeholders := make([]string, len(destinationIDs))
	args := make([]any, len(destinationIDs))
	for i, id := range destinationIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	rows, err := DB.QueryContext(ctx(),
		`SELECT dq.id, dq.destination_id, dq.product_id, dq.quantity,
            COALESCE(p.item_name, '') AS product_name
         FROM transfer_dc_destination_quantities dq
         LEFT JOIN products p ON dq.product_id = p.id
         WHERE dq.destination_id IN (`+strings.Join(placeholders, ",")+`)
         ORDER BY dq.destination_id, dq.product_id`, args...)
	if err != nil {
		return nil, fmt.Errorf("GetQuantitiesForDestinations: %w", err)
	}
	defer rows.Close()

	result := make(map[int][]models.TransferDCDestinationQty)
	for rows.Next() {
		var q models.TransferDCDestinationQty
		if err := rows.Scan(&q.ID, &q.DestinationID, &q.ProductID, &q.Quantity, &q.ProductName); err != nil {
			return nil, fmt.Errorf("GetQuantitiesForDestinations scan: %w", err)
		}
		result[q.DestinationID] = append(result[q.DestinationID], q)
	}
	return result, nil
}

// ============================================================
// Split Tracking
// ============================================================

// CreateSplit creates a split record and marks destinations as split.
func CreateSplit(transferDCID int, shipmentGroupID int, destinationIDs []int, createdBy int) (*models.TransferDCSplit, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("CreateSplit begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get next split number
	var maxSplit sql.NullInt64
	err = tx.QueryRowContext(ctx(),
		`SELECT MAX(split_number) FROM transfer_dc_splits WHERE transfer_dc_id = ?`, transferDCID,
	).Scan(&maxSplit)
	if err != nil {
		return nil, fmt.Errorf("CreateSplit get max split: %w", err)
	}
	nextSplit := 1
	if maxSplit.Valid {
		nextSplit = int(maxSplit.Int64) + 1
	}

	// Insert split record
	result, err := tx.ExecContext(ctx(),
		`INSERT INTO transfer_dc_splits (transfer_dc_id, shipment_group_id, split_number, created_by)
         VALUES (?, ?, ?, ?)`,
		transferDCID, shipmentGroupID, nextSplit, createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateSplit insert: %w", err)
	}
	splitID, _ := result.LastInsertId()

	// Mark destinations as split
	if len(destinationIDs) > 0 {
		placeholders := make([]string, len(destinationIDs))
		args := make([]any, 0, len(destinationIDs)+1)
		args = append(args, splitID)
		for i, id := range destinationIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		_, err = tx.ExecContext(ctx(),
			`UPDATE transfer_dc_destinations SET is_split = 1, split_group_id = ?
             WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
		if err != nil {
			return nil, fmt.Errorf("CreateSplit update destinations: %w", err)
		}
	}

	// Update num_split counter
	if _, err := tx.ExecContext(ctx(),
		`UPDATE transfer_dcs SET num_split = (
            SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ? AND is_split = 1
        ), updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		transferDCID, transferDCID,
	); err != nil {
		return nil, fmt.Errorf("CreateSplit update counter: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("CreateSplit commit: %w", err)
	}

	return &models.TransferDCSplit{
		ID:              int(splitID),
		TransferDCID:    transferDCID,
		ShipmentGroupID: shipmentGroupID,
		SplitNumber:     nextSplit,
		CreatedBy:       createdBy,
	}, nil
}

// GetSplitsByTransferDCID retrieves all split records for a Transfer DC.
func GetSplitsByTransferDCID(transferDCID int) ([]*models.TransferDCSplit, error) {
	rows, err := DB.QueryContext(ctx(),
		`SELECT id, transfer_dc_id, shipment_group_id, split_number, created_by, created_at
         FROM transfer_dc_splits
         WHERE transfer_dc_id = ?
         ORDER BY split_number`, transferDCID)
	if err != nil {
		return nil, fmt.Errorf("GetSplitsByTransferDCID: %w", err)
	}
	defer rows.Close()

	var splits []*models.TransferDCSplit
	for rows.Next() {
		s := &models.TransferDCSplit{}
		var createdBy sql.NullInt64
		var createdAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.TransferDCID, &s.ShipmentGroupID, &s.SplitNumber, &createdBy, &createdAt); err != nil {
			return nil, fmt.Errorf("GetSplitsByTransferDCID scan: %w", err)
		}
		if createdBy.Valid {
			s.CreatedBy = int(createdBy.Int64)
		}
		if createdAt.Valid {
			s.CreatedAt = createdAt.Time
		}
		splits = append(splits, s)
	}
	return splits, nil
}

// GetSplitByShipmentGroupID retrieves a split record by child shipment group ID.
func GetSplitByShipmentGroupID(shipmentGroupID int) (*models.TransferDCSplit, error) {
	s := &models.TransferDCSplit{}
	var createdBy sql.NullInt64
	var createdAt sql.NullTime
	err := DB.QueryRowContext(ctx(),
		`SELECT id, transfer_dc_id, shipment_group_id, split_number, created_by, created_at
         FROM transfer_dc_splits
         WHERE shipment_group_id = ?`, shipmentGroupID,
	).Scan(&s.ID, &s.TransferDCID, &s.ShipmentGroupID, &s.SplitNumber, &createdBy, &createdAt)
	if err != nil {
		return nil, err
	}
	if createdBy.Valid {
		s.CreatedBy = int(createdBy.Int64)
	}
	if createdAt.Valid {
		s.CreatedAt = createdAt.Time
	}
	return s, nil
}

// DeleteSplit deletes a split record and resets destination split status.
func DeleteSplit(splitID int) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("DeleteSplit begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get transfer_dc_id before deleting
	var transferDCID int
	err = tx.QueryRowContext(ctx(),
		`SELECT transfer_dc_id FROM transfer_dc_splits WHERE id = ?`, splitID,
	).Scan(&transferDCID)
	if err != nil {
		return fmt.Errorf("DeleteSplit get transfer_dc_id: %w", err)
	}

	// Reset destination split status for this split group
	if _, err := tx.ExecContext(ctx(),
		`UPDATE transfer_dc_destinations SET is_split = 0, split_group_id = NULL
         WHERE split_group_id = ?`, splitID,
	); err != nil {
		return fmt.Errorf("DeleteSplit reset destinations: %w", err)
	}

	// Delete the split record
	if _, err := tx.ExecContext(ctx(), `DELETE FROM transfer_dc_splits WHERE id = ?`, splitID); err != nil {
		return fmt.Errorf("DeleteSplit delete: %w", err)
	}

	// Update num_split counter
	if _, err := tx.ExecContext(ctx(),
		`UPDATE transfer_dcs SET num_split = (
            SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ? AND is_split = 1
        ), updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		transferDCID, transferDCID,
	); err != nil {
		return fmt.Errorf("DeleteSplit update counter: %w", err)
	}

	return tx.Commit()
}

// CanDeleteSplit checks if a split can be deleted (no issued child DCs).
func CanDeleteSplit(splitID int) (bool, error) {
	var shipmentGroupID int
	err := DB.QueryRowContext(ctx(),
		`SELECT shipment_group_id FROM transfer_dc_splits WHERE id = ?`, splitID,
	).Scan(&shipmentGroupID)
	if err != nil {
		return false, fmt.Errorf("CanDeleteSplit get group: %w", err)
	}

	var issuedCount int
	err = DB.QueryRowContext(ctx(),
		`SELECT COUNT(*) FROM delivery_challans WHERE shipment_group_id = ? AND status = 'issued'`,
		shipmentGroupID,
	).Scan(&issuedCount)
	if err != nil {
		return false, fmt.Errorf("CanDeleteSplit count issued: %w", err)
	}

	return issuedCount == 0, nil
}

// GetNextSplitNumber returns the next sequential split number for a Transfer DC.
func GetNextSplitNumber(transferDCID int) (int, error) {
	var maxSplit sql.NullInt64
	err := DB.QueryRowContext(ctx(),
		`SELECT MAX(split_number) FROM transfer_dc_splits WHERE transfer_dc_id = ?`, transferDCID,
	).Scan(&maxSplit)
	if err != nil {
		return 0, fmt.Errorf("GetNextSplitNumber: %w", err)
	}
	if maxSplit.Valid {
		return int(maxSplit.Int64) + 1, nil
	}
	return 1, nil
}

// ============================================================
// Status Helpers
// ============================================================

// UpdateTransferDCStatus updates the parent DC status.
func UpdateTransferDCStatus(dcID int, status string) error {
	_, err := DB.ExecContext(ctx(),
		`UPDATE delivery_challans SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, dcID,
	)
	if err != nil {
		return fmt.Errorf("UpdateTransferDCStatus: %w", err)
	}
	return nil
}

// RecalculateSplitProgress recounts split vs total destinations and updates transfer_dcs counters.
func RecalculateSplitProgress(transferDCID int) error {
	_, err := DB.ExecContext(ctx(),
		`UPDATE transfer_dcs SET
            num_destinations = (SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ?),
            num_split = (SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ? AND is_split = 1),
            updated_at = CURRENT_TIMESTAMP
         WHERE id = ?`,
		transferDCID, transferDCID, transferDCID,
	)
	if err != nil {
		return fmt.Errorf("RecalculateSplitProgress: %w", err)
	}
	return nil
}

// ============================================================
// Listing & Filtering
// ============================================================

// ListTransferDCsByProject lists Transfer DCs for a project with optional status filter and pagination.
func ListTransferDCsByProject(projectID int, status string, page, pageSize int) ([]*models.TransferDC, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Count query
	countQuery := `SELECT COUNT(*) FROM transfer_dcs t
        INNER JOIN delivery_challans dc ON t.dc_id = dc.id
        WHERE dc.project_id = ?`
	countArgs := []any{projectID}
	if status != "" {
		countQuery += " AND dc.status = ?"
		countArgs = append(countArgs, status)
	}

	var total int
	if err := DB.QueryRowContext(ctx(), countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ListTransferDCsByProject count: %w", err)
	}

	// Data query
	dataQuery := `SELECT t.id, t.dc_id, t.hub_address_id, t.template_id, t.tax_type, t.reverse_charge,
        t.transporter_name, t.vehicle_number, t.eway_bill_number, t.docket_number, t.notes,
        t.num_destinations, t.num_split, t.created_at, t.updated_at,
        dc.dc_number, dc.status, dc.challan_date, dc.project_id,
        COALESCE(a.address_data, '{}') AS hub_address_name,
        COALESCE(tmpl.name, '') AS template_name
     FROM transfer_dcs t
     INNER JOIN delivery_challans dc ON t.dc_id = dc.id
     LEFT JOIN addresses a ON t.hub_address_id = a.id
     LEFT JOIN dc_templates tmpl ON t.template_id = tmpl.id
     WHERE dc.project_id = ?`
	dataArgs := []any{projectID}
	if status != "" {
		dataQuery += " AND dc.status = ?"
		dataArgs = append(dataArgs, status)
	}
	dataQuery += " ORDER BY t.created_at DESC LIMIT ? OFFSET ?"
	dataArgs = append(dataArgs, pageSize, offset)

	rows, err := DB.QueryContext(ctx(), dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListTransferDCsByProject query: %w", err)
	}
	defer rows.Close()

	var tdcs []*models.TransferDC
	for rows.Next() {
		tdc := &models.TransferDC{}
		var templateID sql.NullInt64
		var transporterName, vehicleNumber, ewayBillNumber, docketNumber, notesVal sql.NullString
		var createdAt, updatedAt sql.NullTime
		var challanDate sql.NullString

		if err := rows.Scan(
			&tdc.ID, &tdc.DCID, &tdc.HubAddressID, &templateID,
			&tdc.TaxType, &tdc.ReverseCharge,
			&transporterName, &vehicleNumber, &ewayBillNumber, &docketNumber, &notesVal,
			&tdc.NumDestinations, &tdc.NumSplit, &createdAt, &updatedAt,
			&tdc.DCNumber, &tdc.DCStatus, &challanDate, &tdc.ProjectID,
			&tdc.HubAddressName, &tdc.TemplateName,
		); err != nil {
			return nil, 0, fmt.Errorf("ListTransferDCsByProject scan: %w", err)
		}
		if templateID.Valid {
			v := int(templateID.Int64)
			tdc.TemplateID = &v
		}
		if transporterName.Valid {
			tdc.TransporterName = transporterName.String
		}
		if vehicleNumber.Valid {
			tdc.VehicleNumber = vehicleNumber.String
		}
		if ewayBillNumber.Valid {
			tdc.EwayBillNumber = ewayBillNumber.String
		}
		if docketNumber.Valid {
			tdc.DocketNumber = docketNumber.String
		}
		if notesVal.Valid {
			tdc.Notes = notesVal.String
		}
		if createdAt.Valid {
			tdc.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			tdc.UpdatedAt = updatedAt.Time
		}
		if challanDate.Valid {
			tdc.ChallanDate = &challanDate.String
		}
		tdc.HubAddressName = models.FormatAddressJSON(tdc.HubAddressName)
		tdcs = append(tdcs, tdc)
	}
	return tdcs, total, nil
}

// GetTransferDCSummary returns aggregate stats for a Transfer DC.
func GetTransferDCSummary(transferDCID int) (*models.TransferDCSummary, error) {
	var s models.TransferDCSummary
	err := DB.QueryRowContext(ctx(),
		`SELECT
            (SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ?) AS total_dest,
            (SELECT COUNT(*) FROM transfer_dc_destinations WHERE transfer_dc_id = ? AND is_split = 1) AS split_dest,
            (SELECT COUNT(DISTINCT dq.product_id) FROM transfer_dc_destination_quantities dq
             INNER JOIN transfer_dc_destinations d ON dq.destination_id = d.id WHERE d.transfer_dc_id = ?) AS total_products,
            (SELECT COALESCE(SUM(dq.quantity), 0) FROM transfer_dc_destination_quantities dq
             INNER JOIN transfer_dc_destinations d ON dq.destination_id = d.id WHERE d.transfer_dc_id = ?) AS total_qty,
            (SELECT COUNT(*) FROM transfer_dc_splits WHERE transfer_dc_id = ?) AS split_count`,
		transferDCID, transferDCID, transferDCID, transferDCID, transferDCID,
	).Scan(&s.TotalDestinations, &s.SplitDestinations, &s.TotalProducts, &s.TotalQuantity, &s.SplitCount)
	if err != nil {
		return nil, fmt.Errorf("GetTransferDCSummary: %w", err)
	}
	s.PendingDestinations = s.TotalDestinations - s.SplitDestinations
	return &s, nil
}
