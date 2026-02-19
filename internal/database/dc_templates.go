package database

import (
	"context"
	"database/sql"
	"fmt"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// GetTemplatesByProjectID returns all templates for a project with aggregate counts.
// sqlc-backed via GetTemplatesByProjectID query.
func GetTemplatesByProjectID(projectID int) ([]*models.DCTemplate, error) {
	q := db.New(DB)
	rows, err := q.GetTemplatesByProjectID(context.Background(), int64(projectID))
	if err != nil {
		return nil, err
	}

	templates := make([]*models.DCTemplate, 0, len(rows))
	for _, r := range rows {
		t := &models.DCTemplate{
			ID:              int(r.ID),
			ProjectID:       int(r.ProjectID),
			Name:            r.Name,
			Purpose:         r.Purpose,
			ProductCount:    int(r.ProductCount),
			TransitDCCount:  int(r.TransitCount),
			OfficialDCCount: int(r.OfficialCount),
			UsageCount:      int(r.UsageCount),
		}
		if r.CreatedAt.Valid {
			t.CreatedAt = r.CreatedAt.Time
		}
		if r.UpdatedAt.Valid {
			t.UpdatedAt = r.UpdatedAt.Time
		}
		templates = append(templates, t)
	}
	return templates, nil
}

// GetTemplateByID returns a single template by ID with aggregate counts.
// sqlc-backed via GetTemplateByID query.
func GetTemplateByID(id int) (*models.DCTemplate, error) {
	q := db.New(DB)
	r, err := q.GetTemplateByID(context.Background(), int64(id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, err
	}

	t := &models.DCTemplate{
		ID:              int(r.ID),
		ProjectID:       int(r.ProjectID),
		Name:            r.Name,
		Purpose:         r.Purpose,
		ProductCount:    int(r.ProductCount),
		TransitDCCount:  int(r.TransitCount),
		OfficialDCCount: int(r.OfficialCount),
		UsageCount:      int(r.UsageCount),
	}
	if r.CreatedAt.Valid {
		t.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		t.UpdatedAt = r.UpdatedAt.Time
	}
	return t, nil
}

// GetTemplateProducts returns all products for a template with their default quantities.
// sqlc-backed via GetTemplateProducts query.
func GetTemplateProducts(templateID int) ([]*models.TemplateProductRow, error) {
	q := db.New(DB)
	rows, err := q.GetTemplateProducts(context.Background(), int64(templateID))
	if err != nil {
		return nil, err
	}

	products := make([]*models.TemplateProductRow, 0, len(rows))
	for _, r := range rows {
		p := &models.TemplateProductRow{
			DefaultQuantity: int(r.DefaultQuantity.Int64),
		}
		p.ID = int(r.ID)
		p.ProjectID = int(r.ProjectID)
		p.ItemName = r.ItemName
		p.ItemDescription = r.ItemDescription
		p.HSNCode = r.HsnCode.String
		p.UoM = r.Uom.String
		p.BrandModel = r.BrandModel
		p.PerUnitPrice = r.PerUnitPrice.Float64
		p.GSTPercentage = r.GstPercentage.Float64
		if r.CreatedAt.Valid {
			p.CreatedAt = r.CreatedAt.Time
		}
		if r.UpdatedAt.Valid {
			p.UpdatedAt = r.UpdatedAt.Time
		}
		products = append(products, p)
	}
	return products, nil
}

// TemplateProductInput carries product data for template creation/update.
type TemplateProductInput struct {
	ProductID       int
	DefaultQuantity int
	SortOrder       int
}

// CreateTemplate inserts a new template and its product associations in a transaction.
// Transaction uses DB.Begin(); sqlc queries run within that transaction via db.New(tx).
func CreateTemplate(t *models.DCTemplate, products []TemplateProductInput) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	qtx := db.New(tx)

	result, err := qtx.CreateTemplate(context.Background(), db.CreateTemplateParams{
		ProjectID: int64(t.ProjectID),
		Name:      t.Name,
		Purpose:   t.Purpose,
	})
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = int(id)

	for _, p := range products {
		err := qtx.InsertTemplateProduct(context.Background(), db.InsertTemplateProductParams{
			TemplateID:      int64(t.ID),
			ProductID:       int64(p.ProductID),
			DefaultQuantity: sql.NullInt64{Int64: int64(p.DefaultQuantity), Valid: true},
			SortOrder:       sql.NullInt64{Int64: int64(p.SortOrder), Valid: true},
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateTemplate updates template fields and replaces all product associations in a transaction.
// Transaction uses DB.Begin(); sqlc queries run within that transaction via db.New(tx).
func UpdateTemplate(t *models.DCTemplate, products []TemplateProductInput) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	qtx := db.New(tx)

	err = qtx.UpdateTemplate(context.Background(), db.UpdateTemplateParams{
		Name:      t.Name,
		Purpose:   t.Purpose,
		ID:        int64(t.ID),
		ProjectID: int64(t.ProjectID),
	})
	if err != nil {
		return err
	}

	if err = qtx.DeleteTemplateProducts(context.Background(), int64(t.ID)); err != nil {
		return err
	}

	for _, p := range products {
		err := qtx.InsertTemplateProduct(context.Background(), db.InsertTemplateProductParams{
			TemplateID:      int64(t.ID),
			ProductID:       int64(p.ProductID),
			DefaultQuantity: sql.NullInt64{Int64: int64(p.DefaultQuantity), Valid: true},
			SortOrder:       sql.NullInt64{Int64: int64(p.SortOrder), Valid: true},
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteTemplate deletes a template if no DCs have been issued against it.
// sqlc-backed for the DC count check and deletion.
func DeleteTemplate(id, projectID int) error {
	hasDCs, count, err := CheckTemplateHasDCs(id)
	if err != nil {
		return err
	}
	if hasDCs {
		return fmt.Errorf("cannot delete template: %d DCs have been issued using this template", count)
	}

	q := db.New(DB)
	return q.DeleteTemplate(context.Background(), db.DeleteTemplateParams{
		ID:        int64(id),
		ProjectID: int64(projectID),
	})
}

// CheckTemplateHasDCs returns whether any DCs reference this template.
// The sqlite_master table-existence check is kept as hand-written SQL because
// sqlc does not support querying sqlite_master. The DC count query is sqlc-backed.
func CheckTemplateHasDCs(templateID int) (bool, int, error) {
	// Hand-written: sqlite_master is not expressible in sqlc.
	var tableCount int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='delivery_challans'",
	).Scan(&tableCount)
	if err != nil {
		return false, 0, err
	}
	if tableCount == 0 {
		return false, 0, nil
	}

	q := db.New(DB)
	count, err := q.CheckTemplateHasDCs(context.Background(), sql.NullInt64{Int64: int64(templateID), Valid: true})
	if err != nil {
		return false, 0, err
	}
	return count > 0, int(count), nil
}

// CheckTemplateNameUnique returns true if the given name is not already used
// within the project (optionally excluding a template ID for edit scenarios).
// sqlc-backed via CheckTemplateNameUnique / CheckTemplateNameUniqueExcludeID.
func CheckTemplateNameUnique(projectID int, name string, excludeID int) (bool, error) {
	q := db.New(DB)

	if excludeID > 0 {
		count, err := q.CheckTemplateNameUniqueExcludeID(context.Background(), db.CheckTemplateNameUniqueExcludeIDParams{
			ProjectID: int64(projectID),
			Name:      name,
			ID:        int64(excludeID),
		})
		if err != nil {
			return false, err
		}
		return count == 0, nil
	}

	count, err := q.CheckTemplateNameUnique(context.Background(), db.CheckTemplateNameUniqueParams{
		ProjectID: int64(projectID),
		Name:      name,
	})
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// GetTemplateProductIDs returns a map of productID → defaultQuantity for a template.
// Used to pre-select products in edit forms.
// sqlc-backed via GetTemplateProductIDs query.
func GetTemplateProductIDs(templateID int) (map[int]int, error) {
	q := db.New(DB)
	rows, err := q.GetTemplateProductIDs(context.Background(), int64(templateID))
	if err != nil {
		return nil, err
	}

	result := make(map[int]int, len(rows))
	for _, r := range rows {
		result[int(r.ProductID)] = int(r.DefaultQuantity.Int64)
	}
	return result, nil
}

// DuplicateTemplate creates a copy of a template (prefixed "Copy of ") with all product links.
// sqlc-backed for source product retrieval and the underlying CreateTemplate call.
func DuplicateTemplate(templateID, projectID int) (*models.DCTemplate, error) {
	original, err := GetTemplateByID(templateID)
	if err != nil {
		return nil, err
	}
	if original.ProjectID != projectID {
		return nil, fmt.Errorf("template not found in project")
	}

	q := db.New(DB)
	srcRows, err := q.GetTemplateDuplicateSource(context.Background(), int64(templateID))
	if err != nil {
		return nil, err
	}

	products := make([]TemplateProductInput, 0, len(srcRows))
	for _, r := range srcRows {
		products = append(products, TemplateProductInput{
			ProductID:       int(r.ProductID),
			DefaultQuantity: int(r.DefaultQuantity.Int64),
			SortOrder:       int(r.SortOrder.Int64),
		})
	}

	newTemplate := &models.DCTemplate{
		ProjectID: projectID,
		Name:      "Copy of " + original.Name,
		Purpose:   original.Purpose,
	}

	if err := CreateTemplate(newTemplate, products); err != nil {
		return nil, err
	}

	return newTemplate, nil
}
