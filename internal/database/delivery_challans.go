package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// CreateDeliveryChallan creates a delivery challan with transit details, line items, and serial numbers in a transaction.
func CreateDeliveryChallan(dc *models.DeliveryChallan, transitDetails *models.DCTransitDetails, lineItems []models.DCLineItem, serialNumbersByLine [][]string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert delivery challan
	result, err := tx.Exec(
		`INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, template_id, bill_to_address_id, ship_to_address_id, challan_date, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		dc.ProjectID, dc.DCNumber, dc.DCType, dc.Status,
		dc.TemplateID, dc.BillToAddressID, dc.ShipToAddressID,
		dc.ChallanDate, dc.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to insert delivery challan: %w", err)
	}
	dcID, _ := result.LastInsertId()
	dc.ID = int(dcID)

	// Insert transit details if provided
	if transitDetails != nil {
		transitDetails.DCID = dc.ID
		_, err = tx.Exec(
			`INSERT INTO dc_transit_details (dc_id, transporter_name, vehicle_number, eway_bill_number, notes)
			 VALUES (?, ?, ?, ?, ?)`,
			transitDetails.DCID, transitDetails.TransporterName, transitDetails.VehicleNumber,
			transitDetails.EwayBillNumber, transitDetails.Notes,
		)
		if err != nil {
			return fmt.Errorf("failed to insert transit details: %w", err)
		}
	}

	// Insert line items and serial numbers
	for i, item := range lineItems {
		item.DCID = dc.ID
		item.LineOrder = i + 1

		liResult, err := tx.Exec(
			`INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.DCID, item.ProductID, item.Quantity, item.Rate,
			item.TaxPercentage, item.TaxableAmount, item.TaxAmount, item.TotalAmount, item.LineOrder,
		)
		if err != nil {
			return fmt.Errorf("failed to insert line item %d: %w", i+1, err)
		}
		liID, _ := liResult.LastInsertId()
		lineItems[i].ID = int(liID)

		// Insert serial numbers for this line item
		if i < len(serialNumbersByLine) {
			for _, sn := range serialNumbersByLine[i] {
				sn = strings.TrimSpace(sn)
				if sn == "" {
					continue
				}
				_, err = tx.Exec(
					`INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES (?, ?, ?)`,
					dc.ProjectID, int(liID), sn,
				)
				if err != nil {
					return fmt.Errorf("failed to insert serial number '%s': %w", sn, err)
				}
			}
		}
	}

	return tx.Commit()
}

// GetDeliveryChallanByID fetches a delivery challan by ID.
func GetDeliveryChallanByID(id int) (*models.DeliveryChallan, error) {
	dc := &models.DeliveryChallan{}
	var templateID sql.NullInt64
	var billToID sql.NullInt64
	var challanDate sql.NullString
	var issuedAt sql.NullTime
	var issuedBy sql.NullInt64

	err := DB.QueryRow(
		`SELECT id, project_id, dc_number, dc_type, status, template_id, bill_to_address_id,
		        ship_to_address_id, challan_date, issued_at, issued_by, created_by, created_at, updated_at
		 FROM delivery_challans WHERE id = ?`, id,
	).Scan(
		&dc.ID, &dc.ProjectID, &dc.DCNumber, &dc.DCType, &dc.Status,
		&templateID, &billToID, &dc.ShipToAddressID,
		&challanDate, &issuedAt, &issuedBy, &dc.CreatedBy,
		&dc.CreatedAt, &dc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if templateID.Valid {
		v := int(templateID.Int64)
		dc.TemplateID = &v
	}
	if billToID.Valid {
		v := int(billToID.Int64)
		dc.BillToAddressID = &v
	}
	if challanDate.Valid {
		dc.ChallanDate = &challanDate.String
	}
	if issuedAt.Valid {
		dc.IssuedAt = &issuedAt.Time
	}
	if issuedBy.Valid {
		v := int(issuedBy.Int64)
		dc.IssuedBy = &v
	}

	return dc, nil
}

// GetTransitDetailsByDCID fetches transit details for a DC.
func GetTransitDetailsByDCID(dcID int) (*models.DCTransitDetails, error) {
	td := &models.DCTransitDetails{}
	var transporterName, vehicleNumber, ewayBill, notes sql.NullString

	err := DB.QueryRow(
		`SELECT id, dc_id, transporter_name, vehicle_number, eway_bill_number, notes
		 FROM dc_transit_details WHERE dc_id = ?`, dcID,
	).Scan(&td.ID, &td.DCID, &transporterName, &vehicleNumber, &ewayBill, &notes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if transporterName.Valid {
		td.TransporterName = transporterName.String
	}
	if vehicleNumber.Valid {
		td.VehicleNumber = vehicleNumber.String
	}
	if ewayBill.Valid {
		td.EwayBillNumber = ewayBill.String
	}
	if notes.Valid {
		td.Notes = notes.String
	}

	return td, nil
}

// GetLineItemsByDCID fetches all line items for a DC with product details joined.
func GetLineItemsByDCID(dcID int) ([]models.DCLineItem, error) {
	query := `
		SELECT li.id, li.dc_id, li.product_id, li.quantity, li.rate, li.tax_percentage,
		       li.taxable_amount, li.tax_amount, li.total_amount, li.line_order,
		       li.created_at, li.updated_at,
		       p.item_name, p.item_description, COALESCE(p.hsn_code, ''), p.uom,
		       COALESCE(p.brand_model, ''), p.gst_percentage
		FROM dc_line_items li
		INNER JOIN products p ON li.product_id = p.id
		WHERE li.dc_id = ?
		ORDER BY li.line_order
	`

	rows, err := DB.Query(query, dcID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.DCLineItem
	for rows.Next() {
		var li models.DCLineItem
		err := rows.Scan(
			&li.ID, &li.DCID, &li.ProductID, &li.Quantity, &li.Rate, &li.TaxPercentage,
			&li.TaxableAmount, &li.TaxAmount, &li.TotalAmount, &li.LineOrder,
			&li.CreatedAt, &li.UpdatedAt,
			&li.ItemName, &li.ItemDescription, &li.HSNCode, &li.UoM,
			&li.BrandModel, &li.GSTPercentage,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, li)
	}

	return items, nil
}

// GetSerialNumbersByLineItemID fetches all serial numbers for a line item.
func GetSerialNumbersByLineItemID(lineItemID int) ([]string, error) {
	rows, err := DB.Query(
		`SELECT serial_number FROM serial_numbers WHERE line_item_id = ? ORDER BY id`, lineItemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var serials []string
	for rows.Next() {
		var sn string
		if err := rows.Scan(&sn); err != nil {
			return nil, err
		}
		serials = append(serials, sn)
	}
	return serials, nil
}

// GetDCsByProjectID fetches all DCs for a project.
func GetDCsByProjectID(projectID int, dcType string) ([]*models.DeliveryChallan, error) {
	query := `
		SELECT dc.id, dc.project_id, dc.dc_number, dc.dc_type, dc.status,
		       dc.challan_date, dc.created_at, dc.updated_at
		FROM delivery_challans dc
		WHERE dc.project_id = ?
	`
	args := []interface{}{projectID}

	if dcType != "" {
		query += " AND dc.dc_type = ?"
		args = append(args, dcType)
	}

	query += " ORDER BY dc.created_at DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dcs []*models.DeliveryChallan
	for rows.Next() {
		dc := &models.DeliveryChallan{}
		var challanDate sql.NullString
		err := rows.Scan(
			&dc.ID, &dc.ProjectID, &dc.DCNumber, &dc.DCType, &dc.Status,
			&challanDate, &dc.CreatedAt, &dc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if challanDate.Valid {
			dc.ChallanDate = &challanDate.String
		}
		dcs = append(dcs, dc)
	}

	return dcs, nil
}

// GetAllAddressesByConfigID returns all addresses for a config (no pagination, for dropdowns).
func GetAllAddressesByConfigID(configID int) ([]*models.Address, error) {
	rows, err := DB.Query(
		`SELECT id, config_id, address_data, created_at, updated_at
		 FROM addresses WHERE config_id = ? ORDER BY id`, configID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []*models.Address
	for rows.Next() {
		a := &models.Address{}
		if err := rows.Scan(&a.ID, &a.ConfigID, &a.DataJSON, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if err := a.ParseData(); err != nil {
			return nil, err
		}
		addresses = append(addresses, a)
	}
	return addresses, nil
}
