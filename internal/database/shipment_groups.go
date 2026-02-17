package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// CreateShipmentGroup inserts a new shipment group and returns its ID.
func CreateShipmentGroup(group *models.ShipmentGroup) (int, error) {
	result, err := DB.Exec(
		`INSERT INTO shipment_groups (project_id, template_id, num_sets, tax_type, reverse_charge, status, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		group.ProjectID, group.TemplateID, group.NumSets,
		group.TaxType, group.ReverseCharge, group.Status, group.CreatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create shipment group: %w", err)
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

// GetShipmentGroup fetches a shipment group by ID with computed fields.
func GetShipmentGroup(id int) (*models.ShipmentGroup, error) {
	sg := &models.ShipmentGroup{}
	var templateID sql.NullInt64
	var templateName sql.NullString
	var transitDCID sql.NullInt64
	var transitDCNumber sql.NullString

	err := DB.QueryRow(
		`SELECT sg.id, sg.project_id, sg.template_id, sg.num_sets, sg.tax_type,
		        sg.reverse_charge, sg.status, sg.created_by, sg.created_at, sg.updated_at,
		        t.name,
		        p.name,
		        tdc.id, tdc.dc_number,
		        (SELECT COUNT(*) FROM delivery_challans dc2 WHERE dc2.shipment_group_id = sg.id AND dc2.dc_type = 'official')
		 FROM shipment_groups sg
		 LEFT JOIN dc_templates t ON sg.template_id = t.id
		 LEFT JOIN projects p ON sg.project_id = p.id
		 LEFT JOIN delivery_challans tdc ON tdc.shipment_group_id = sg.id AND tdc.dc_type = 'transit'
		 WHERE sg.id = ?`, id,
	).Scan(
		&sg.ID, &sg.ProjectID, &templateID, &sg.NumSets, &sg.TaxType,
		&sg.ReverseCharge, &sg.Status, &sg.CreatedBy, &sg.CreatedAt, &sg.UpdatedAt,
		&templateName,
		&sg.ProjectName,
		&transitDCID, &transitDCNumber,
		&sg.OfficialDCCount,
	)
	if err != nil {
		return nil, err
	}

	if templateID.Valid {
		v := int(templateID.Int64)
		sg.TemplateID = &v
	}
	if templateName.Valid {
		sg.TemplateName = templateName.String
	}
	if transitDCID.Valid {
		v := int(transitDCID.Int64)
		sg.TransitDCID = &v
	}
	if transitDCNumber.Valid {
		sg.TransitDCNumber = transitDCNumber.String
	}

	return sg, nil
}

// GetShipmentGroupByDCID finds the shipment group that contains a given DC.
func GetShipmentGroupByDCID(dcID int) (*models.ShipmentGroup, error) {
	var groupID int
	err := DB.QueryRow(
		`SELECT shipment_group_id FROM delivery_challans WHERE id = ? AND shipment_group_id IS NOT NULL`, dcID,
	).Scan(&groupID)
	if err != nil {
		return nil, err
	}
	return GetShipmentGroup(groupID)
}

// GetShipmentGroupDCs returns all delivery challans in a shipment group.
func GetShipmentGroupDCs(groupID int) ([]*models.DeliveryChallan, error) {
	rows, err := DB.Query(
		`SELECT dc.id, dc.project_id, dc.dc_number, dc.dc_type, dc.status,
		        dc.challan_date, dc.created_at, dc.updated_at
		 FROM delivery_challans dc
		 WHERE dc.shipment_group_id = ?
		 ORDER BY dc.dc_type DESC, dc.id`, groupID,
	)
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

// UpdateShipmentGroupStatus updates the status of a shipment group.
func UpdateShipmentGroupStatus(id int, status string) error {
	_, err := DB.Exec(
		`UPDATE shipment_groups SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to update shipment group status: %w", err)
	}
	return nil
}

// GetShipmentGroupsByProjectID returns all shipment groups for a project.
func GetShipmentGroupsByProjectID(projectID int) ([]*models.ShipmentGroup, error) {
	rows, err := DB.Query(
		`SELECT sg.id, sg.project_id, sg.template_id, sg.num_sets, sg.tax_type,
		        sg.reverse_charge, sg.status, sg.created_by, sg.created_at, sg.updated_at,
		        COALESCE(t.name, ''),
		        (SELECT COUNT(*) FROM delivery_challans dc2 WHERE dc2.shipment_group_id = sg.id AND dc2.dc_type = 'official'),
		        COALESCE(tdc.dc_number, ''),
		        tdc.id
		 FROM shipment_groups sg
		 LEFT JOIN dc_templates t ON sg.template_id = t.id
		 LEFT JOIN delivery_challans tdc ON tdc.shipment_group_id = sg.id AND tdc.dc_type = 'transit'
		 WHERE sg.project_id = ?
		 ORDER BY sg.created_at DESC`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*models.ShipmentGroup
	for rows.Next() {
		sg := &models.ShipmentGroup{}
		var templateID sql.NullInt64
		var transitDCID sql.NullInt64
		err := rows.Scan(
			&sg.ID, &sg.ProjectID, &templateID, &sg.NumSets, &sg.TaxType,
			&sg.ReverseCharge, &sg.Status, &sg.CreatedBy, &sg.CreatedAt, &sg.UpdatedAt,
			&sg.TemplateName,
			&sg.OfficialDCCount,
			&sg.TransitDCNumber,
			&transitDCID,
		)
		if err != nil {
			return nil, err
		}
		if templateID.Valid {
			v := int(templateID.Int64)
			sg.TemplateID = &v
		}
		if transitDCID.Valid {
			v := int(transitDCID.Int64)
			sg.TransitDCID = &v
		}
		groups = append(groups, sg)
	}
	return groups, nil
}

// IssueAllDCsInGroup issues all draft DCs in a shipment group.
func IssueAllDCsInGroup(groupID int, issuedBy int) (int, error) {
	now := time.Now()
	result, err := DB.Exec(
		`UPDATE delivery_challans SET status = 'issued', issued_at = ?, issued_by = ?, updated_at = ?
		 WHERE shipment_group_id = ? AND status = 'draft'`,
		now, issuedBy, now, groupID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to issue DCs in group: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}
