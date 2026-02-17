package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ProjectExistsWithName checks if a project with the given name already exists,
// optionally excluding a specific project ID (for updates).
func ProjectExistsWithName(name string, excludeID int) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM projects WHERE LOWER(name) = LOWER(?) AND id != ?"
	err := DB.QueryRow(query, strings.TrimSpace(name), excludeID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ProjectExistsWithPrefix checks if a project with the given DC prefix already exists,
// optionally excluding a specific project ID (for updates).
func ProjectExistsWithPrefix(prefix string, excludeID int) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM projects WHERE LOWER(dc_prefix) = LOWER(?) AND id != ?"
	err := DB.QueryRow(query, strings.TrimSpace(prefix), excludeID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// projectColumns is the standard column list for project queries.
const projectColumns = `
	p.id, p.name, p.description, p.dc_prefix,
	p.tender_ref_number, p.tender_ref_details,
	p.po_reference, p.po_date,
	p.bill_from_address, p.dispatch_from_address,
	p.company_gstin, p.company_email, p.company_cin,
	p.company_signature_path, p.company_seal_path,
	p.dc_number_format, p.dc_number_separator,
	p.purpose_text, p.seq_padding,
	p.last_transit_dc_number, p.last_official_dc_number,
	p.created_by, p.created_at, p.updated_at`

func scanProjectWithCounts(scanner interface {
	Scan(dest ...interface{}) error
}, p *models.Project) error {
	var sigPath, sealPath, poDate sql.NullString
	err := scanner.Scan(
		&p.ID, &p.Name, &p.Description, &p.DCPrefix,
		&p.TenderRefNumber, &p.TenderRefDetails,
		&p.POReference, &poDate,
		&p.BillFromAddress, &p.DispatchFromAddress,
		&p.CompanyGSTIN, &p.CompanyEmail, &p.CompanyCIN,
		&sigPath, &sealPath,
		&p.DCNumberFormat, &p.DCNumberSeparator,
		&p.PurposeText, &p.SeqPadding,
		&p.LastTransitDCNumber, &p.LastOfficialDCNumber,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		&p.TransitDCCount, &p.OfficialDCCount,
		&p.TemplateCount, &p.ProductCount,
	)
	if err != nil {
		return err
	}
	if sigPath.Valid {
		p.CompanySignaturePath = sigPath.String
	}
	if sealPath.Valid {
		p.CompanySealPath = sealPath.String
	}
	if poDate.Valid {
		p.PODate = &poDate.String
	}
	return nil
}

func GetAllProjects() ([]*models.Project, error) {
	query := `
		SELECT ` + projectColumns + `,
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
		if err := scanProjectWithCounts(rows, p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

func GetProjectByID(id int) (*models.Project, error) {
	query := `
		SELECT ` + projectColumns + `,
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
	if err := scanProjectWithCounts(DB.QueryRow(query, id), p); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, err
	}

	return p, nil
}

func CreateProject(p *models.Project) error {
	exists, err := ProjectExistsWithName(p.Name, 0)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("a project with the name '%s' already exists", p.Name)
	}

	prefixExists, err := ProjectExistsWithPrefix(p.DCPrefix, 0)
	if err != nil {
		return err
	}
	if prefixExists {
		return fmt.Errorf("a project with the DC prefix '%s' already exists", p.DCPrefix)
	}

	query := `
		INSERT INTO projects (
			name, description, dc_prefix, tender_ref_number, tender_ref_details,
			po_reference, po_date, bill_from_address, dispatch_from_address,
			company_gstin, company_email, company_cin,
			company_signature_path, company_seal_path,
			dc_number_format, dc_number_separator,
			purpose_text, seq_padding, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var poDate interface{}
	if p.PODate != nil && *p.PODate != "" {
		poDate = *p.PODate
	}

	// Set defaults
	if p.DCNumberFormat == "" {
		p.DCNumberFormat = models.DefaultDCNumberFormat
	}
	if p.DCNumberSeparator == "" {
		p.DCNumberSeparator = "-"
	}
	if p.PurposeText == "" {
		p.PurposeText = "DELIVERED AS PART OF PROJECT EXECUTION"
	}
	if p.SeqPadding == 0 {
		p.SeqPadding = 3
	}

	var sigPath, sealPath interface{}
	if p.CompanySignaturePath != "" {
		sigPath = p.CompanySignaturePath
	}
	if p.CompanySealPath != "" {
		sealPath = p.CompanySealPath
	}

	result, err := DB.Exec(
		query,
		p.Name, p.Description, p.DCPrefix,
		p.TenderRefNumber, p.TenderRefDetails,
		p.POReference, poDate,
		p.BillFromAddress, p.DispatchFromAddress,
		p.CompanyGSTIN, p.CompanyEmail, p.CompanyCIN,
		sigPath, sealPath,
		p.DCNumberFormat, p.DCNumberSeparator,
		p.PurposeText, p.SeqPadding, p.CreatedBy,
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
	exists, err := ProjectExistsWithName(p.Name, p.ID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("a project with the name '%s' already exists", p.Name)
	}

	prefixExists, err := ProjectExistsWithPrefix(p.DCPrefix, p.ID)
	if err != nil {
		return err
	}
	if prefixExists {
		return fmt.Errorf("a project with the DC prefix '%s' already exists", p.DCPrefix)
	}

	query := `
		UPDATE projects SET
			name = ?, description = ?, dc_prefix = ?,
			tender_ref_number = ?, tender_ref_details = ?,
			po_reference = ?, po_date = ?,
			bill_from_address = ?, dispatch_from_address = ?,
			company_gstin = ?, company_email = ?, company_cin = ?,
			company_signature_path = ?, company_seal_path = ?,
			dc_number_format = ?, dc_number_separator = ?,
			purpose_text = ?, seq_padding = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	var poDate interface{}
	if p.PODate != nil && *p.PODate != "" {
		poDate = *p.PODate
	}

	var sigPath, sealPath interface{}
	if p.CompanySignaturePath != "" {
		sigPath = p.CompanySignaturePath
	}
	if p.CompanySealPath != "" {
		sealPath = p.CompanySealPath
	}

	_, err = DB.Exec(
		query,
		p.Name, p.Description, p.DCPrefix,
		p.TenderRefNumber, p.TenderRefDetails,
		p.POReference, poDate,
		p.BillFromAddress, p.DispatchFromAddress,
		p.CompanyGSTIN, p.CompanyEmail, p.CompanyCIN,
		sigPath, sealPath,
		p.DCNumberFormat, p.DCNumberSeparator,
		p.PurposeText, p.SeqPadding,
		p.ID,
	)

	return err
}

// UpdateProjectSettings updates only the settings-related fields of a project.
func UpdateProjectSettings(p *models.Project, tab string) error {
	var query string
	var args []interface{}

	switch tab {
	case "general":
		exists, err := ProjectExistsWithName(p.Name, p.ID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("a project with the name '%s' already exists", p.Name)
		}
		prefixExists, err := ProjectExistsWithPrefix(p.DCPrefix, p.ID)
		if err != nil {
			return err
		}
		if prefixExists {
			return fmt.Errorf("a project with the DC prefix '%s' already exists", p.DCPrefix)
		}
		query = `UPDATE projects SET name = ?, description = ?, dc_prefix = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{p.Name, p.Description, p.DCPrefix, p.ID}
	case "company":
		var sigPath, sealPath interface{}
		if p.CompanySignaturePath != "" {
			sigPath = p.CompanySignaturePath
		}
		if p.CompanySealPath != "" {
			sealPath = p.CompanySealPath
		}
		query = `UPDATE projects SET bill_from_address = ?, dispatch_from_address = ?, company_gstin = ?, company_email = ?, company_cin = ?, company_signature_path = ?, company_seal_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{p.BillFromAddress, p.DispatchFromAddress, p.CompanyGSTIN, p.CompanyEmail, p.CompanyCIN, sigPath, sealPath, p.ID}
	case "dc_config":
		query = `UPDATE projects SET dc_number_format = ?, dc_number_separator = ?, purpose_text = ?, seq_padding = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{p.DCNumberFormat, p.DCNumberSeparator, p.PurposeText, p.SeqPadding, p.ID}
	case "tender":
		var poDate interface{}
		if p.PODate != nil && *p.PODate != "" {
			poDate = *p.PODate
		}
		query = `UPDATE projects SET tender_ref_number = ?, tender_ref_details = ?, po_reference = ?, po_date = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{p.TenderRefNumber, p.TenderRefDetails, p.POReference, poDate, p.ID}
	default:
		return fmt.Errorf("unknown settings tab: %s", tab)
	}

	_, err := DB.Exec(query, args...)
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
