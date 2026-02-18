package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// nullTimeFromDateStr parses an optional "YYYY-MM-DD" pointer into sql.NullTime.
func nullTimeFromDateStr(s *string) sql.NullTime {
	if s == nil || *s == "" {
		return sql.NullTime{}
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// mapGetDCRow maps a sqlc GetDeliveryChallanByIDRow to *models.DeliveryChallan.
func mapGetDCRow(r db.GetDeliveryChallanByIDRow) *models.DeliveryChallan {
	dc := &models.DeliveryChallan{
		ID:              int(r.ID),
		ProjectID:       int(r.ProjectID),
		DCNumber:        r.DcNumber,
		DCType:          r.DcType,
		Status:          r.Status,
		ShipToAddressID: int(r.ShipToAddressID),
		CreatedBy:       int(r.CreatedBy),
	}
	if r.CreatedAt.Valid {
		dc.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		dc.UpdatedAt = r.UpdatedAt.Time
	}
	if r.TemplateID.Valid {
		v := int(r.TemplateID.Int64)
		dc.TemplateID = &v
	}
	if r.BillToAddressID.Valid {
		v := int(r.BillToAddressID.Int64)
		dc.BillToAddressID = &v
	}
	if r.ChallanDate.Valid {
		s := r.ChallanDate.Time.Format("2006-01-02")
		dc.ChallanDate = &s
	}
	if r.IssuedAt.Valid {
		dc.IssuedAt = &r.IssuedAt.Time
	}
	if r.IssuedBy.Valid {
		v := int(r.IssuedBy.Int64)
		dc.IssuedBy = &v
	}
	if r.BundleID.Valid {
		v := int(r.BundleID.Int64)
		dc.BundleID = &v
	}
	if r.ShipmentGroupID.Valid {
		v := int(r.ShipmentGroupID.Int64)
		dc.ShipmentGroupID = &v
	}
	if r.BillFromAddressID.Valid {
		v := int(r.BillFromAddressID.Int64)
		dc.BillFromAddressID = &v
	}
	if r.DispatchFromAddressID.Valid {
		v := int(r.DispatchFromAddressID.Int64)
		dc.DispatchFromAddressID = &v
	}
	if r.TemplateName.Valid {
		dc.TemplateName = r.TemplateName.String
	}
	return dc
}

// mapDCListRow builds a *models.DeliveryChallan from the common list-query columns.
func mapDCListRow(id, projectID int64, dcNumber, dcType, status string, challanDate, createdAt, updatedAt sql.NullTime) *models.DeliveryChallan {
	dc := &models.DeliveryChallan{
		ID:        int(id),
		ProjectID: int(projectID),
		DCNumber:  dcNumber,
		DCType:    dcType,
		Status:    status,
	}
	if challanDate.Valid {
		s := challanDate.Time.Format("2006-01-02")
		dc.ChallanDate = &s
	}
	if createdAt.Valid {
		dc.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		dc.UpdatedAt = updatedAt.Time
	}
	return dc
}

// CreateDeliveryChallan creates a delivery challan with transit details, line items,
// and serial numbers inside a single transaction.
// sqlc-backed: InsertDeliveryChallan, InsertDCTransitDetails, InsertDCLineItem, InsertSerialNumber.
func CreateDeliveryChallan(dc *models.DeliveryChallan, transitDetails *models.DCTransitDetails, lineItems []models.DCLineItem, serialNumbersByLine [][]string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := db.New(tx)

	// Insert delivery challan.
	result, err := q.InsertDeliveryChallan(ctx(), db.InsertDeliveryChallanParams{
		ProjectID:             int64(dc.ProjectID),
		DcNumber:              dc.DCNumber,
		DcType:                dc.DCType,
		Status:                dc.Status,
		TemplateID:            nullInt64FromPtr(dc.TemplateID),
		BillToAddressID:       nullInt64FromPtr(dc.BillToAddressID),
		ShipToAddressID:       int64(dc.ShipToAddressID),
		ChallanDate:           nullTimeFromDateStr(dc.ChallanDate),
		CreatedBy:             int64(dc.CreatedBy),
		ShipmentGroupID:       nullInt64FromPtr(dc.ShipmentGroupID),
		BillFromAddressID:     nullInt64FromPtr(dc.BillFromAddressID),
		DispatchFromAddressID: nullInt64FromPtr(dc.DispatchFromAddressID),
	})
	if err != nil {
		return fmt.Errorf("failed to insert delivery challan: %w", err)
	}
	dcID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get insert ID for delivery challan: %w", err)
	}
	dc.ID = int(dcID)

	// Insert transit details if provided.
	if transitDetails != nil {
		transitDetails.DCID = dc.ID
		err = q.InsertDCTransitDetails(ctx(), db.InsertDCTransitDetailsParams{
			DcID:            int64(transitDetails.DCID),
			TransporterName: nullStringFromStr(transitDetails.TransporterName),
			VehicleNumber:   nullStringFromStr(transitDetails.VehicleNumber),
			EwayBillNumber:  nullStringFromStr(transitDetails.EwayBillNumber),
			Notes:           nullStringFromStr(transitDetails.Notes),
		})
		if err != nil {
			return fmt.Errorf("failed to insert transit details: %w", err)
		}
	}

	// Insert line items and their serial numbers.
	for i, item := range lineItems {
		item.DCID = dc.ID
		item.LineOrder = i + 1

		liResult, err := q.InsertDCLineItem(ctx(), db.InsertDCLineItemParams{
			DcID:          int64(item.DCID),
			ProductID:     int64(item.ProductID),
			Quantity:      int64(item.Quantity),
			Rate:          nullFloat64(item.Rate),
			TaxPercentage: nullFloat64(item.TaxPercentage),
			TaxableAmount: nullFloat64(item.TaxableAmount),
			TaxAmount:     nullFloat64(item.TaxAmount),
			TotalAmount:   nullFloat64(item.TotalAmount),
			LineOrder:     sql.NullInt64{Int64: int64(item.LineOrder), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
		liID, err := liResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("get insert ID for line item %d: %w", i+1, err)
		}
		lineItems[i].ID = int(liID)

		if i < len(serialNumbersByLine) {
			for _, sn := range serialNumbersByLine[i] {
				sn = strings.TrimSpace(sn)
				if sn == "" {
					continue
				}
				err = q.InsertSerialNumber(ctx(), db.InsertSerialNumberParams{
					ProjectID:    int64(dc.ProjectID),
					LineItemID:   liID,
					SerialNumber: sn,
					ProductID:    sql.NullInt64{Int64: int64(item.ProductID), Valid: true},
				})
				if err != nil {
					return fmt.Errorf("failed to insert serial number '%s': %w", sn, err)
				}
			}
		}
	}

	return tx.Commit()
}

// GetDeliveryChallanByID fetches a delivery challan by ID.
// sqlc-backed: GetDeliveryChallanByID.
func GetDeliveryChallanByID(id int) (*models.DeliveryChallan, error) {
	row, err := queries().GetDeliveryChallanByID(ctx(), int64(id))
	if err != nil {
		return nil, err
	}
	return mapGetDCRow(row), nil
}

// GetTransitDetailsByDCID fetches transit details for a DC.
// sqlc-backed: GetTransitDetailsByDCID.
func GetTransitDetailsByDCID(dcID int) (*models.DCTransitDetails, error) {
	row, err := queries().GetTransitDetailsByDCID(ctx(), int64(dcID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	td := &models.DCTransitDetails{
		ID:   int(row.ID),
		DCID: int(row.DcID),
	}
	if row.TransporterName.Valid {
		td.TransporterName = row.TransporterName.String
	}
	if row.VehicleNumber.Valid {
		td.VehicleNumber = row.VehicleNumber.String
	}
	if row.EwayBillNumber.Valid {
		td.EwayBillNumber = row.EwayBillNumber.String
	}
	if row.Notes.Valid {
		td.Notes = row.Notes.String
	}
	return td, nil
}

// GetLineItemsByDCID fetches all line items for a DC with product details joined.
// sqlc-backed: GetLineItemsByDCID.
func GetLineItemsByDCID(dcID int) ([]models.DCLineItem, error) {
	rows, err := queries().GetLineItemsByDCID(ctx(), int64(dcID))
	if err != nil {
		return nil, err
	}

	var items []models.DCLineItem
	for _, r := range rows {
		li := models.DCLineItem{
			ID:              int(r.ID),
			DCID:            int(r.DcID),
			ProductID:       int(r.ProductID),
			Quantity:        int(r.Quantity),
			ItemName:        r.ItemName,
			ItemDescription: r.ItemDescription,
			HSNCode:         r.HsnCode,
			BrandModel:      r.BrandModel,
		}
		if r.Rate.Valid {
			li.Rate = r.Rate.Float64
		}
		if r.TaxPercentage.Valid {
			li.TaxPercentage = r.TaxPercentage.Float64
		}
		if r.TaxableAmount.Valid {
			li.TaxableAmount = r.TaxableAmount.Float64
		}
		if r.TaxAmount.Valid {
			li.TaxAmount = r.TaxAmount.Float64
		}
		if r.TotalAmount.Valid {
			li.TotalAmount = r.TotalAmount.Float64
		}
		if r.LineOrder.Valid {
			li.LineOrder = int(r.LineOrder.Int64)
		}
		if r.CreatedAt.Valid {
			li.CreatedAt = r.CreatedAt.Time
		}
		if r.UpdatedAt.Valid {
			li.UpdatedAt = r.UpdatedAt.Time
		}
		if r.Uom.Valid {
			li.UoM = r.Uom.String
		}
		if r.GstPercentage.Valid {
			li.GSTPercentage = r.GstPercentage.Float64
		}
		items = append(items, li)
	}
	return items, nil
}

// GetSerialNumbersByLineItemID fetches all serial numbers for a line item.
// sqlc-backed: GetSerialNumbersByLineItemID.
func GetSerialNumbersByLineItemID(lineItemID int) ([]string, error) {
	return queries().GetSerialNumbersByLineItemID(ctx(), int64(lineItemID))
}

// SerialConflict represents a serial number that conflicts with an existing one in the project.
type SerialConflict struct {
	SerialNumber string
	ExistingDCID int
	DCNumber     string
	DCStatus     string
	ProductName  string
}

// CheckSerialsInProject checks which serial numbers already exist in a project.
// Returns conflicts with DC info. Optionally excludes a specific DC (for edit mode).
// Hand-written SQL: dynamic IN clause with variable-length serial list cannot be handled by sqlc.
func CheckSerialsInProject(projectID int, serials []string, excludeDCID *int) ([]SerialConflict, error) {
	if len(serials) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(serials))
	args := make([]interface{}, 0, len(serials)+2)
	args = append(args, projectID)
	for i, s := range serials {
		placeholders[i] = "?"
		args = append(args, s)
	}

	query := `
		SELECT sn.serial_number, sn.line_item_id, dc.id, dc.dc_number, dc.status, p.item_name
		FROM serial_numbers sn
		INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		INNER JOIN products p ON li.product_id = p.id
		WHERE sn.project_id = ?
		  AND sn.serial_number IN (` + strings.Join(placeholders, ",") + `)`

	if excludeDCID != nil {
		query += " AND dc.id != ?"
		args = append(args, *excludeDCID)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []SerialConflict
	for rows.Next() {
		var c SerialConflict
		var lineItemID int
		if err := rows.Scan(&c.SerialNumber, &lineItemID, &c.ExistingDCID, &c.DCNumber, &c.DCStatus, &c.ProductName); err != nil {
			return nil, err
		}
		conflicts = append(conflicts, c)
	}
	return conflicts, nil
}

// CheckSerialsInProjectByProduct checks serial uniqueness scoped to (project_id, product_id).
// Hand-written SQL: dynamic IN clause with variable-length serial list cannot be handled by sqlc.
func CheckSerialsInProjectByProduct(projectID, productID int, serials []string, excludeDCID *int) ([]SerialConflict, error) {
	if len(serials) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(serials))
	args := make([]interface{}, 0, len(serials)+3)
	args = append(args, projectID, productID)
	for i, s := range serials {
		placeholders[i] = "?"
		args = append(args, s)
	}

	query := `
		SELECT sn.serial_number, sn.line_item_id, dc.id, dc.dc_number, dc.status, p.item_name
		FROM serial_numbers sn
		INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		INNER JOIN products p ON li.product_id = p.id
		WHERE sn.project_id = ?
		  AND sn.product_id = ?
		  AND sn.serial_number IN (` + strings.Join(placeholders, ",") + `)`

	if excludeDCID != nil {
		query += " AND dc.id != ?"
		args = append(args, *excludeDCID)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []SerialConflict
	for rows.Next() {
		var c SerialConflict
		var lineItemID int
		if err := rows.Scan(&c.SerialNumber, &lineItemID, &c.ExistingDCID, &c.DCNumber, &c.DCStatus, &c.ProductName); err != nil {
			return nil, err
		}
		conflicts = append(conflicts, c)
	}
	return conflicts, nil
}

// DeleteDC deletes a DC and all associated data inside a single transaction.
// sqlc-backed: DeleteSerialNumbersByDCID, DeleteLineItemsByDCID, DeleteTransitDetailsByDCID, DeleteDeliveryChallan.
func DeleteDC(dcID int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := db.New(tx)

	if err := q.DeleteSerialNumbersByDCID(ctx(), int64(dcID)); err != nil {
		return fmt.Errorf("failed to delete serial numbers: %w", err)
	}
	if err := q.DeleteLineItemsByDCID(ctx(), int64(dcID)); err != nil {
		return fmt.Errorf("failed to delete line items: %w", err)
	}
	// Transit details may not exist for official DCs — ignore the error.
	_ = q.DeleteTransitDetailsByDCID(ctx(), int64(dcID))
	if err := q.DeleteDeliveryChallan(ctx(), int64(dcID)); err != nil {
		return fmt.Errorf("failed to delete DC: %w", err)
	}

	return tx.Commit()
}

// GetDCsByProjectID fetches all DCs for a project.
// sqlc-backed: GetDCsByProjectID (unfiltered) and GetDCsByProjectIDAndType (type-filtered).
func GetDCsByProjectID(projectID int, dcType string) ([]*models.DeliveryChallan, error) {
	var dcs []*models.DeliveryChallan

	if dcType == "" {
		rows, err := queries().GetDCsByProjectID(ctx(), int64(projectID))
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			dcs = append(dcs, mapDCListRow(r.ID, r.ProjectID, r.DcNumber, r.DcType, r.Status, r.ChallanDate, r.CreatedAt, r.UpdatedAt))
		}
	} else {
		rows, err := queries().GetDCsByProjectIDAndType(ctx(), db.GetDCsByProjectIDAndTypeParams{
			ProjectID: int64(projectID),
			DcType:    dcType,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			dcs = append(dcs, mapDCListRow(r.ID, r.ProjectID, r.DcNumber, r.DcType, r.Status, r.ChallanDate, r.CreatedAt, r.UpdatedAt))
		}
	}

	return dcs, nil
}

// IssueDC transitions a DC from draft to issued status.
// sqlc-backed: IssueDC.
func IssueDC(dcID int, userID int) error {
	now := time.Now()
	result, err := queries().IssueDC(ctx(), db.IssueDCParams{
		IssuedAt:  sql.NullTime{Time: now, Valid: true},
		IssuedBy:  sql.NullInt64{Int64: int64(userID), Valid: true},
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
		ID:        int64(dcID),
	})
	if err != nil {
		return fmt.Errorf("failed to issue DC: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected for issue DC: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("DC not found or already issued")
	}
	return nil
}

// GetDCsByShipmentGroup fetches all DCs belonging to a shipment group.
// sqlc-backed: GetDCsByShipmentGroup.
func GetDCsByShipmentGroup(groupID int) ([]*models.DeliveryChallan, error) {
	rows, err := queries().GetDCsByShipmentGroup(ctx(), sql.NullInt64{Int64: int64(groupID), Valid: true})
	if err != nil {
		return nil, err
	}

	var dcs []*models.DeliveryChallan
	for _, r := range rows {
		dcs = append(dcs, mapDCListRow(r.ID, r.ProjectID, r.DcNumber, r.DcType, r.Status, r.ChallanDate, r.CreatedAt, r.UpdatedAt))
	}
	return dcs, nil
}

// GetAllAddressesByConfigID returns all addresses for a config (no pagination, for dropdowns).
// Hand-written SQL: sqlc-generated SQL for this query is broken/truncated.
func GetAllAddressesByConfigID(configID int) ([]*models.Address, error) {
	rows, err := DB.Query(
		`SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
		 FROM addresses WHERE config_id = ? ORDER BY id`, configID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []*models.Address
	for rows.Next() {
		a := &models.Address{}
		if err := rows.Scan(&a.ID, &a.ConfigID, &a.DataJSON, &a.DistrictName, &a.MandalName, &a.MandalCode, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if err := a.ParseData(); err != nil {
			return nil, err
		}
		addresses = append(addresses, a)
	}
	return addresses, nil
}
