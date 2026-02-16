package database

import (
	"database/sql"
	"fmt"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

func GetAllProjects() ([]*models.Project, error) {
	query := `
		SELECT
			p.id, p.name, p.description, p.dc_prefix,
			p.tender_ref_number, p.tender_ref_details,
			p.po_reference, p.po_date,
			p.bill_from_address, p.company_gstin,
			p.company_signature_path,
			p.last_transit_dc_number, p.last_official_dc_number,
			p.created_by, p.created_at, p.updated_at,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_dc_count,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_dc_count,
			COUNT(DISTINCT t.id) as template_count,
			COUNT(DISTINCT pr.id) as product_count
		FROM projects p
		LEFT JOIN delivery_challans dc ON p.id = dc.project_id
		LEFT JOIN dc_templates t ON p.id = t.project_id
		LEFT JOIN products pr ON p.id = pr.project_id
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		var sigPath sql.NullString
		var poDate sql.NullString
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.DCPrefix,
			&p.TenderRefNumber, &p.TenderRefDetails,
			&p.POReference, &poDate,
			&p.BillFromAddress, &p.CompanyGSTIN,
			&sigPath,
			&p.LastTransitDCNumber, &p.LastOfficialDCNumber,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
			&p.TransitDCCount, &p.OfficialDCCount,
			&p.TemplateCount, &p.ProductCount,
		)
		if err != nil {
			return nil, err
		}
		if sigPath.Valid {
			p.CompanySignaturePath = sigPath.String
		}
		if poDate.Valid {
			p.PODate = &poDate.String
		}
		projects = append(projects, p)
	}

	return projects, nil
}

func GetProjectByID(id int) (*models.Project, error) {
	query := `
		SELECT
			p.id, p.name, p.description, p.dc_prefix,
			p.tender_ref_number, p.tender_ref_details,
			p.po_reference, p.po_date,
			p.bill_from_address, p.company_gstin,
			p.company_signature_path,
			p.last_transit_dc_number, p.last_official_dc_number,
			p.created_by, p.created_at, p.updated_at,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_dc_count,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_dc_count,
			COUNT(DISTINCT t.id) as template_count,
			COUNT(DISTINCT pr.id) as product_count
		FROM projects p
		LEFT JOIN delivery_challans dc ON p.id = dc.project_id
		LEFT JOIN dc_templates t ON p.id = t.project_id
		LEFT JOIN products pr ON p.id = pr.project_id
		WHERE p.id = ?
		GROUP BY p.id
	`

	p := &models.Project{}
	var sigPath sql.NullString
	var poDate sql.NullString
	err := DB.QueryRow(query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.DCPrefix,
		&p.TenderRefNumber, &p.TenderRefDetails,
		&p.POReference, &poDate,
		&p.BillFromAddress, &p.CompanyGSTIN,
		&sigPath,
		&p.LastTransitDCNumber, &p.LastOfficialDCNumber,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		&p.TransitDCCount, &p.OfficialDCCount,
		&p.TemplateCount, &p.ProductCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found")
	}
	if err != nil {
		return nil, err
	}

	if sigPath.Valid {
		p.CompanySignaturePath = sigPath.String
	}
	if poDate.Valid {
		p.PODate = &poDate.String
	}

	return p, nil
}

func CreateProject(p *models.Project) error {
	query := `
		INSERT INTO projects (
			name, description, dc_prefix, tender_ref_number, tender_ref_details,
			po_reference, po_date, bill_from_address, company_gstin,
			company_signature_path, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var poDate interface{}
	if p.PODate != nil && *p.PODate != "" {
		poDate = *p.PODate
	}

	result, err := DB.Exec(
		query,
		p.Name, p.Description, p.DCPrefix,
		p.TenderRefNumber, p.TenderRefDetails,
		p.POReference, poDate,
		p.BillFromAddress, p.CompanyGSTIN,
		p.CompanySignaturePath, p.CreatedBy,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	p.ID = int(id)
	return nil
}

func UpdateProject(p *models.Project) error {
	query := `
		UPDATE projects SET
			name = ?, description = ?, dc_prefix = ?,
			tender_ref_number = ?, tender_ref_details = ?,
			po_reference = ?, po_date = ?,
			bill_from_address = ?, company_gstin = ?,
			company_signature_path = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	var poDate interface{}
	if p.PODate != nil && *p.PODate != "" {
		poDate = *p.PODate
	}

	_, err := DB.Exec(
		query,
		p.Name, p.Description, p.DCPrefix,
		p.TenderRefNumber, p.TenderRefDetails,
		p.POReference, poDate,
		p.BillFromAddress, p.CompanyGSTIN,
		p.CompanySignaturePath,
		p.ID,
	)

	return err
}

func DeleteProject(id int) error {
	var issuedCount int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND status = 'issued'",
		id,
	).Scan(&issuedCount)
	if err != nil {
		return err
	}

	if issuedCount > 0 {
		return fmt.Errorf("cannot delete project with issued delivery challans")
	}

	_, err = DB.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

func CanDeleteProject(id int) (bool, error) {
	var issuedCount int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND status = 'issued'",
		id,
	).Scan(&issuedCount)
	if err != nil {
		return false, err
	}

	return issuedCount == 0, nil
}
