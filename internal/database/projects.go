package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ProjectExistsWithName checks if a project with the given name already exists,
// optionally excluding a specific project ID (for updates).
func ProjectExistsWithName(name string, excludeID int) (bool, error) {
	count, err := queries().ProjectExistsWithName(context.Background(), db.ProjectExistsWithNameParams{
		LOWER: strings.TrimSpace(name),
		ID:    int64(excludeID),
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ProjectExistsWithPrefix checks if a project with the given DC prefix already exists,
// optionally excluding a specific project ID (for updates).
func ProjectExistsWithPrefix(prefix string, excludeID int) (bool, error) {
	count, err := queries().ProjectExistsWithPrefix(context.Background(), db.ProjectExistsWithPrefixParams{
		LOWER: strings.TrimSpace(prefix),
		ID:    int64(excludeID),
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// mapGetAllProjectsRow converts a sqlc GetAllProjectsRow to a models.Project.
func mapGetAllProjectsRow(r db.GetAllProjectsRow) *models.Project {
	p := &models.Project{
		ID:                   int(r.ID),
		Name:                 r.Name,
		Description:          r.Description,
		DCPrefix:             r.DcPrefix,
		TenderRefNumber:      r.TenderRefNumber,
		TenderRefDetails:     r.TenderRefDetails,
		POReference:          r.PoReference,
		BillFromAddress:      r.BillFromAddress,
		DispatchFromAddress:  r.DispatchFromAddress,
		CompanyGSTIN:         r.CompanyGstin,
		CompanyEmail:         r.CompanyEmail,
		CompanyCIN:           r.CompanyCin,
		CompanySignaturePath: r.CompanySignaturePath.String,
		CompanySealPath:      r.CompanySealPath.String,
		DCNumberFormat:       r.DcNumberFormat,
		DCNumberSeparator:    r.DcNumberSeparator,
		PurposeText:          r.PurposeText,
		SeqPadding:           int(r.SeqPadding),
		LastTransitDCNumber:  int(r.LastTransitDcNumber.Int64),
		LastOfficialDCNumber: int(r.LastOfficialDcNumber.Int64),
		CreatedBy:            int(r.CreatedBy),
		// Computed counts
		TransitDCCount:  int(r.TransitDcCount),
		OfficialDCCount: int(r.OfficialDcCount),
		TemplateCount:   int(r.TemplateCount),
		ProductCount:    int(r.ProductCount),
	}
	if r.PoDate.Valid {
		s := r.PoDate.Time.Format("2006-01-02")
		p.PODate = &s
	}
	if r.CreatedAt.Valid {
		p.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		p.UpdatedAt = r.UpdatedAt.Time
	}
	return p
}

// mapGetProjectByIDRow converts a sqlc GetProjectByIDRow to a models.Project.
func mapGetProjectByIDRow(r db.GetProjectByIDRow) *models.Project {
	p := &models.Project{
		ID:                   int(r.ID),
		Name:                 r.Name,
		Description:          r.Description,
		DCPrefix:             r.DcPrefix,
		TenderRefNumber:      r.TenderRefNumber,
		TenderRefDetails:     r.TenderRefDetails,
		POReference:          r.PoReference,
		BillFromAddress:      r.BillFromAddress,
		DispatchFromAddress:  r.DispatchFromAddress,
		CompanyGSTIN:         r.CompanyGstin,
		CompanyEmail:         r.CompanyEmail,
		CompanyCIN:           r.CompanyCin,
		CompanySignaturePath: r.CompanySignaturePath.String,
		CompanySealPath:      r.CompanySealPath.String,
		DCNumberFormat:       r.DcNumberFormat,
		DCNumberSeparator:    r.DcNumberSeparator,
		PurposeText:          r.PurposeText,
		SeqPadding:           int(r.SeqPadding),
		LastTransitDCNumber:  int(r.LastTransitDcNumber.Int64),
		LastOfficialDCNumber: int(r.LastOfficialDcNumber.Int64),
		CreatedBy:            int(r.CreatedBy),
		// Computed counts
		TransitDCCount:  int(r.TransitDcCount),
		OfficialDCCount: int(r.OfficialDcCount),
		TemplateCount:   int(r.TemplateCount),
		ProductCount:    int(r.ProductCount),
	}
	if r.PoDate.Valid {
		s := r.PoDate.Time.Format("2006-01-02")
		p.PODate = &s
	}
	if r.CreatedAt.Valid {
		p.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		p.UpdatedAt = r.UpdatedAt.Time
	}
	return p
}

// poDateToNullTime converts an optional PO date string (*string) to sql.NullTime.
func poDateToNullTime(poDate *string) sql.NullTime {
	if poDate == nil || *poDate == "" {
		return sql.NullTime{}
	}
	t, err := time.Parse("2006-01-02", *poDate)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func GetAllProjects() ([]*models.Project, error) {
	rows, err := queries().GetAllProjects(context.Background())
	if err != nil {
		return nil, err
	}
	projects := make([]*models.Project, 0, len(rows))
	for _, r := range rows {
		projects = append(projects, mapGetAllProjectsRow(r))
	}
	return projects, nil
}

func GetProjectByID(id int) (*models.Project, error) {
	row, err := queries().GetProjectByID(context.Background(), int64(id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found")
	}
	if err != nil {
		return nil, err
	}
	return mapGetProjectByIDRow(row), nil
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

	result, err := queries().CreateProject(context.Background(), db.CreateProjectParams{
		Name:                 p.Name,
		Description:          p.Description,
		DcPrefix:             p.DCPrefix,
		TenderRefNumber:      p.TenderRefNumber,
		TenderRefDetails:     p.TenderRefDetails,
		PoReference:          p.POReference,
		PoDate:               poDateToNullTime(p.PODate),
		BillFromAddress:      p.BillFromAddress,
		DispatchFromAddress:  p.DispatchFromAddress,
		CompanyGstin:         p.CompanyGSTIN,
		CompanyEmail:         p.CompanyEmail,
		CompanyCin:           p.CompanyCIN,
		CompanySignaturePath: nullStringFromStr(p.CompanySignaturePath),
		CompanySealPath:      nullStringFromStr(p.CompanySealPath),
		DcNumberFormat:       p.DCNumberFormat,
		DcNumberSeparator:    p.DCNumberSeparator,
		PurposeText:          p.PurposeText,
		SeqPadding:           int64(p.SeqPadding),
		CreatedBy:            int64(p.CreatedBy),
	})
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

	return queries().UpdateProject(context.Background(), db.UpdateProjectParams{
		Name:                 p.Name,
		Description:          p.Description,
		DcPrefix:             p.DCPrefix,
		TenderRefNumber:      p.TenderRefNumber,
		TenderRefDetails:     p.TenderRefDetails,
		PoReference:          p.POReference,
		PoDate:               poDateToNullTime(p.PODate),
		BillFromAddress:      p.BillFromAddress,
		DispatchFromAddress:  p.DispatchFromAddress,
		CompanyGstin:         p.CompanyGSTIN,
		CompanyEmail:         p.CompanyEmail,
		CompanyCin:           p.CompanyCIN,
		CompanySignaturePath: nullStringFromStr(p.CompanySignaturePath),
		CompanySealPath:      nullStringFromStr(p.CompanySealPath),
		DcNumberFormat:       p.DCNumberFormat,
		DcNumberSeparator:    p.DCNumberSeparator,
		PurposeText:          p.PurposeText,
		SeqPadding:           int64(p.SeqPadding),
		ID:                   int64(p.ID),
	})
}

// UpdateProjectSettings updates only the settings-related fields of a project.
func UpdateProjectSettings(p *models.Project, tab string) error {
	ctx := context.Background()

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
		return queries().UpdateProjectSettingsGeneral(ctx, db.UpdateProjectSettingsGeneralParams{
			Name:        p.Name,
			Description: p.Description,
			DcPrefix:    p.DCPrefix,
			ID:          int64(p.ID),
		})

	case "company":
		return queries().UpdateProjectSettingsCompany(ctx, db.UpdateProjectSettingsCompanyParams{
			BillFromAddress:      p.BillFromAddress,
			DispatchFromAddress:  p.DispatchFromAddress,
			CompanyGstin:         p.CompanyGSTIN,
			CompanyEmail:         p.CompanyEmail,
			CompanyCin:           p.CompanyCIN,
			CompanySignaturePath: nullStringFromStr(p.CompanySignaturePath),
			CompanySealPath:      nullStringFromStr(p.CompanySealPath),
			ID:                   int64(p.ID),
		})

	case "dc_config":
		return queries().UpdateProjectSettingsDCConfig(ctx, db.UpdateProjectSettingsDCConfigParams{
			DcNumberFormat:    p.DCNumberFormat,
			DcNumberSeparator: p.DCNumberSeparator,
			PurposeText:       p.PurposeText,
			SeqPadding:        int64(p.SeqPadding),
			ID:                int64(p.ID),
		})

	case "tender":
		return queries().UpdateProjectSettingsTender(ctx, db.UpdateProjectSettingsTenderParams{
			TenderRefNumber:  p.TenderRefNumber,
			TenderRefDetails: p.TenderRefDetails,
			PoReference:      p.POReference,
			PoDate:           poDateToNullTime(p.PODate),
			ID:               int64(p.ID),
		})

	default:
		return fmt.Errorf("unknown settings tab: %s", tab)
	}
}

func DeleteProject(id int) error {
	issuedCount, err := queries().CountIssuedDCsForProject(context.Background(), int64(id))
	if err != nil {
		return err
	}
	if issuedCount > 0 {
		return fmt.Errorf("cannot delete project with issued delivery challans")
	}
	return queries().DeleteProject(context.Background(), int64(id))
}

func CanDeleteProject(id int) (bool, error) {
	issuedCount, err := queries().CountIssuedDCsForProject(context.Background(), int64(id))
	if err != nil {
		return false, err
	}
	return issuedCount == 0, nil
}
