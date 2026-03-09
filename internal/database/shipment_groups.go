package database

import (
	"database/sql"
	"fmt"
	"time"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// CreateShipmentGroup inserts a new shipment group and returns its ID.
func CreateShipmentGroup(group *models.ShipmentGroup) (int, error) {
	result, err := queries().CreateShipmentGroup(ctx(), db.CreateShipmentGroupParams{
		ProjectID:     int64(group.ProjectID),
		TemplateID:    nullInt64FromPtr(group.TemplateID),
		NumSets:       int64(group.NumLocations),
		TaxType:       group.TaxType,
		ReverseCharge: group.ReverseCharge,
		Status:        group.Status,
		CreatedBy:     nullInt64FromInt(group.CreatedBy),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create shipment group: %w", err)
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

// GetShipmentGroup fetches a shipment group by ID with computed fields.
func GetShipmentGroup(id int) (*models.ShipmentGroup, error) {
	r, err := queries().GetShipmentGroup(ctx(), int64(id))
	if err != nil {
		return nil, err
	}
	sg := &models.ShipmentGroup{
		ID:              int(r.ID),
		ProjectID:       int(r.ProjectID),
		NumLocations:    int(r.NumSets),
		TaxType:         r.TaxType,
		ReverseCharge:   r.ReverseCharge,
		Status:          r.Status,
		OfficialDCCount: int(r.OfficialDcCount),
		ProjectName:     r.ProjectName.String,
		TemplateName:    r.TemplateName.String,
		TransitDCNumber: r.TransitDcNumber.String,
	}
	if r.CreatedBy.Valid {
		sg.CreatedBy = int(r.CreatedBy.Int64)
	}
	if r.CreatedAt.Valid {
		sg.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		sg.UpdatedAt = r.UpdatedAt.Time
	}
	if r.TemplateID.Valid {
		v := int(r.TemplateID.Int64)
		sg.TemplateID = &v
	}
	if r.TransitDcID.Valid {
		v := int(r.TransitDcID.Int64)
		sg.TransitDCID = &v
	}
	return sg, nil
}

// GetShipmentGroupByDCID finds the shipment group that contains a given DC.
func GetShipmentGroupByDCID(dcID int) (*models.ShipmentGroup, error) {
	groupIDNull, err := queries().GetShipmentGroupIDByDCID(ctx(), int64(dcID))
	if err != nil {
		return nil, err
	}
	if !groupIDNull.Valid {
		return nil, sql.ErrNoRows
	}
	return GetShipmentGroup(int(groupIDNull.Int64))
}

// GetShipmentGroupDCs returns all delivery challans in a shipment group.
func GetShipmentGroupDCs(groupID int) ([]*models.DeliveryChallan, error) {
	rows, err := queries().GetShipmentGroupDCs(ctx(), sql.NullInt64{Int64: int64(groupID), Valid: true})
	if err != nil {
		return nil, err
	}
	dcs := make([]*models.DeliveryChallan, 0, len(rows))
	for _, r := range rows {
		dc := &models.DeliveryChallan{
			ID:        int(r.ID),
			ProjectID: int(r.ProjectID),
			DCNumber:  r.DcNumber,
			DCType:    r.DcType,
			Status:    r.Status,
		}
		if r.ChallanDate.Valid {
			s := r.ChallanDate.Time.Format("2006-01-02")
			dc.ChallanDate = &s
		}
		if r.CreatedAt.Valid {
			dc.CreatedAt = r.CreatedAt.Time
		}
		if r.UpdatedAt.Valid {
			dc.UpdatedAt = r.UpdatedAt.Time
		}
		dcs = append(dcs, dc)
	}
	return dcs, nil
}

// UpdateShipmentGroupStatus updates the status of a shipment group.
func UpdateShipmentGroupStatus(id int, status string) error {
	now := time.Now()
	err := queries().UpdateShipmentGroupStatus(ctx(), db.UpdateShipmentGroupStatusParams{
		Status:    status,
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
		ID:        int64(id),
	})
	if err != nil {
		return fmt.Errorf("failed to update shipment group status: %w", err)
	}
	return nil
}

// GetShipmentGroupsByProjectID returns all shipment groups for a project.
func GetShipmentGroupsByProjectID(projectID int) ([]*models.ShipmentGroup, error) {
	rows, err := queries().GetShipmentGroupsByProjectID(ctx(), int64(projectID))
	if err != nil {
		return nil, err
	}
	groups := make([]*models.ShipmentGroup, 0, len(rows))
	for _, r := range rows {
		sg := &models.ShipmentGroup{
			ID:              int(r.ID),
			ProjectID:       int(r.ProjectID),
			NumLocations:    int(r.NumSets),
			TaxType:         r.TaxType,
			ReverseCharge:   r.ReverseCharge,
			Status:          r.Status,
			TemplateName:    r.TemplateName,
			OfficialDCCount: int(r.OfficialDcCount),
			TransitDCNumber: r.TransitDcNumber,
		}
		if r.CreatedBy.Valid {
			sg.CreatedBy = int(r.CreatedBy.Int64)
		}
		if r.CreatedAt.Valid {
			sg.CreatedAt = r.CreatedAt.Time
		}
		if r.UpdatedAt.Valid {
			sg.UpdatedAt = r.UpdatedAt.Time
		}
		if r.TemplateID.Valid {
			v := int(r.TemplateID.Int64)
			sg.TemplateID = &v
		}
		if r.TransitDcID.Valid {
			v := int(r.TransitDcID.Int64)
			sg.TransitDCID = &v
		}
		groups = append(groups, sg)
	}
	return groups, nil
}

// IssueAllDCsInGroup issues all draft DCs in a shipment group.
func IssueAllDCsInGroup(groupID int, issuedBy int) (int, error) {
	now := time.Now()
	result, err := queries().IssueAllDCsInGroup(ctx(), db.IssueAllDCsInGroupParams{
		IssuedAt:        sql.NullTime{Time: now, Valid: true},
		IssuedBy:        sql.NullInt64{Int64: int64(issuedBy), Valid: true},
		UpdatedAt:       sql.NullTime{Time: now, Valid: true},
		ShipmentGroupID: sql.NullInt64{Int64: int64(groupID), Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to issue DCs in group: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// UpdateShipmentGroup updates mutable fields on an existing draft shipment group.
// Only call this while the group is still in "draft" status.
func UpdateShipmentGroup(groupID int, templateID *int, numLocations int, taxType, reverseCharge string) error {
	_, err := DB.ExecContext(ctx(),
		`UPDATE shipment_groups
		    SET template_id = ?, num_sets = ?, tax_type = ?, reverse_charge = ?, updated_at = CURRENT_TIMESTAMP
		  WHERE id = ? AND status = 'draft'`,
		nullInt64FromPtr(templateID), numLocations, taxType, reverseCharge, groupID,
	)
	if err != nil {
		return fmt.Errorf("UpdateShipmentGroup: %w", err)
	}
	return nil
}

// DeleteShipmentGroup deletes a shipment group and all its DCs, line items, and serial numbers.
func DeleteShipmentGroup(groupID int) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("DeleteShipmentGroup: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Delete serial numbers for all DCs in the group
	if _, err := tx.ExecContext(ctx(),
		`DELETE FROM serial_numbers WHERE line_item_id IN (
			SELECT li.id FROM dc_line_items li
			INNER JOIN delivery_challans dc ON li.dc_id = dc.id
			WHERE dc.shipment_group_id = ?
		)`, groupID); err != nil {
		return fmt.Errorf("DeleteShipmentGroup: delete serials: %w", err)
	}

	// 2. Delete line items for all DCs in the group
	if _, err := tx.ExecContext(ctx(),
		`DELETE FROM dc_line_items WHERE dc_id IN (
			SELECT id FROM delivery_challans WHERE shipment_group_id = ?
		)`, groupID); err != nil {
		return fmt.Errorf("DeleteShipmentGroup: delete line items: %w", err)
	}

	// 3. Delete transit details for all DCs in the group
	if _, err := tx.ExecContext(ctx(),
		`DELETE FROM dc_transit_details WHERE dc_id IN (
			SELECT id FROM delivery_challans WHERE shipment_group_id = ?
		)`, groupID); err != nil {
		return fmt.Errorf("DeleteShipmentGroup: delete transit details: %w", err)
	}

	// 4. Delete all delivery challans in the group
	if _, err := tx.ExecContext(ctx(),
		`DELETE FROM delivery_challans WHERE shipment_group_id = ?`, groupID); err != nil {
		return fmt.Errorf("DeleteShipmentGroup: delete DCs: %w", err)
	}

	// 5. Delete the shipment group itself
	if _, err := tx.ExecContext(ctx(),
		`DELETE FROM shipment_groups WHERE id = ?`, groupID); err != nil {
		return fmt.Errorf("DeleteShipmentGroup: delete group: %w", err)
	}

	return tx.Commit()
}

// GetIssuedShipToAddressIDs returns ship-to address IDs used in issued shipment groups for a project.
func GetIssuedShipToAddressIDs(projectID int) ([]int, error) {
	rows, err := DB.QueryContext(ctx(),
		`SELECT DISTINCT dc.ship_to_address_id
		 FROM delivery_challans dc
		 INNER JOIN shipment_groups sg ON dc.shipment_group_id = sg.id
		 WHERE sg.project_id = ? AND sg.status = 'issued'
		   AND dc.dc_type = 'official' AND dc.ship_to_address_id IS NOT NULL`,
		projectID)
	if err != nil {
		return nil, fmt.Errorf("GetIssuedShipToAddressIDs: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetShipToAddressIDsByGroup returns ship-to address IDs for official DCs in a shipment group.
func GetShipToAddressIDsByGroup(groupID int) ([]int, error) {
	rows, err := DB.QueryContext(ctx(),
		`SELECT ship_to_address_id FROM delivery_challans
		 WHERE shipment_group_id = ? AND dc_type = 'official' AND ship_to_address_id IS NOT NULL`,
		groupID)
	if err != nil {
		return nil, fmt.Errorf("GetShipToAddressIDsByGroup: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
