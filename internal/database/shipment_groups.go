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
		NumSets:       int64(group.NumSets),
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
		NumSets:         int(r.NumSets),
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
			NumSets:         int(r.NumSets),
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
