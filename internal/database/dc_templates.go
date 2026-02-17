package database

import (
	"database/sql"
	"fmt"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

func GetTemplatesByProjectID(projectID int) ([]*models.DCTemplate, error) {
	query := `
		SELECT
			t.id, t.project_id, t.name, t.purpose, t.created_at, t.updated_at,
			COUNT(DISTINCT tp.product_id) as product_count,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_count,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_count,
			COUNT(DISTINCT dc.id) as usage_count
		FROM dc_templates t
		LEFT JOIN dc_template_products tp ON t.id = tp.template_id
		LEFT JOIN delivery_challans dc ON t.id = dc.template_id
		WHERE t.project_id = ?
		GROUP BY t.id
		ORDER BY t.created_at DESC
	`

	rows, err := DB.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.DCTemplate
	for rows.Next() {
		t := &models.DCTemplate{}
		err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Name, &t.Purpose, &t.CreatedAt, &t.UpdatedAt,
			&t.ProductCount, &t.TransitDCCount, &t.OfficialDCCount, &t.UsageCount,
		)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}

	return templates, nil
}

func GetTemplateByID(id int) (*models.DCTemplate, error) {
	query := `
		SELECT
			t.id, t.project_id, t.name, t.purpose, t.created_at, t.updated_at,
			COUNT(DISTINCT tp.product_id) as product_count,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_count,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_count,
			COUNT(DISTINCT dc.id) as usage_count
		FROM dc_templates t
		LEFT JOIN dc_template_products tp ON t.id = tp.template_id
		LEFT JOIN delivery_challans dc ON t.id = dc.template_id
		WHERE t.id = ?
		GROUP BY t.id
	`

	t := &models.DCTemplate{}
	err := DB.QueryRow(query, id).Scan(
		&t.ID, &t.ProjectID, &t.Name, &t.Purpose, &t.CreatedAt, &t.UpdatedAt,
		&t.ProductCount, &t.TransitDCCount, &t.OfficialDCCount, &t.UsageCount,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, err
	}

	return t, nil
}

func GetTemplateProducts(templateID int) ([]*models.TemplateProductRow, error) {
	query := `
		SELECT p.id, p.project_id, p.item_name, p.item_description, p.hsn_code, p.uom,
		       p.brand_model, p.per_unit_price, p.gst_percentage, p.created_at, p.updated_at,
		       tp.default_quantity
		FROM products p
		INNER JOIN dc_template_products tp ON p.id = tp.product_id
		WHERE tp.template_id = ?
		ORDER BY tp.sort_order, p.item_name
	`

	rows, err := DB.Query(query, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*models.TemplateProductRow
	for rows.Next() {
		p := &models.TemplateProductRow{}
		var hsnCode, brandModel sql.NullString
		err := rows.Scan(
			&p.ID, &p.ProjectID, &p.ItemName, &p.ItemDescription,
			&hsnCode, &p.UoM, &brandModel,
			&p.PerUnitPrice, &p.GSTPercentage, &p.CreatedAt, &p.UpdatedAt,
			&p.DefaultQuantity,
		)
		if err != nil {
			return nil, err
		}
		if hsnCode.Valid {
			p.HSNCode = hsnCode.String
		}
		if brandModel.Valid {
			p.BrandModel = brandModel.String
		}
		products = append(products, p)
	}

	return products, nil
}

type TemplateProductInput struct {
	ProductID       int
	DefaultQuantity int
	SortOrder       int
}

func CreateTemplate(t *models.DCTemplate, products []TemplateProductInput) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"INSERT INTO dc_templates (project_id, name, purpose) VALUES (?, ?, ?)",
		t.ProjectID, t.Name, t.Purpose,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = int(id)

	for _, p := range products {
		_, err := tx.Exec(
			"INSERT INTO dc_template_products (template_id, product_id, default_quantity, sort_order) VALUES (?, ?, ?, ?)",
			t.ID, p.ProductID, p.DefaultQuantity, p.SortOrder,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func UpdateTemplate(t *models.DCTemplate, products []TemplateProductInput) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"UPDATE dc_templates SET name = ?, purpose = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND project_id = ?",
		t.Name, t.Purpose, t.ID, t.ProjectID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM dc_template_products WHERE template_id = ?", t.ID)
	if err != nil {
		return err
	}

	for _, p := range products {
		_, err := tx.Exec(
			"INSERT INTO dc_template_products (template_id, product_id, default_quantity, sort_order) VALUES (?, ?, ?, ?)",
			t.ID, p.ProductID, p.DefaultQuantity, p.SortOrder,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func DeleteTemplate(id, projectID int) error {
	hasDCs, count, err := CheckTemplateHasDCs(id)
	if err != nil {
		return err
	}
	if hasDCs {
		return fmt.Errorf("cannot delete template: %d DCs have been issued using this template", count)
	}

	_, err = DB.Exec("DELETE FROM dc_templates WHERE id = ? AND project_id = ?", id, projectID)
	return err
}

func CheckTemplateHasDCs(templateID int) (bool, int, error) {
	// Check if delivery_challans table exists
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

	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE template_id = ?", templateID).Scan(&count)
	if err != nil {
		return false, 0, err
	}
	return count > 0, count, nil
}

func CheckTemplateNameUnique(projectID int, name string, excludeID int) (bool, error) {
	query := "SELECT COUNT(*) FROM dc_templates WHERE project_id = ? AND name = ?"
	args := []interface{}{projectID, name}

	if excludeID > 0 {
		query += " AND id != ?"
		args = append(args, excludeID)
	}

	var count int
	err := DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// GetTemplateProductIDs returns the product IDs for a template (for pre-selecting in edit form)
func GetTemplateProductIDs(templateID int) (map[int]int, error) {
	query := "SELECT product_id, default_quantity FROM dc_template_products WHERE template_id = ?"
	rows, err := DB.Query(query, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]int)
	for rows.Next() {
		var productID, qty int
		if err := rows.Scan(&productID, &qty); err != nil {
			return nil, err
		}
		result[productID] = qty
	}
	return result, nil
}

// DuplicateTemplate creates a copy of a template with "Copy of " prefix and all product links
func DuplicateTemplate(templateID, projectID int) (*models.DCTemplate, error) {
	original, err := GetTemplateByID(templateID)
	if err != nil {
		return nil, err
	}
	if original.ProjectID != projectID {
		return nil, fmt.Errorf("template not found in project")
	}

	// Get original products with sort order
	rows, err := DB.Query("SELECT product_id, default_quantity, sort_order FROM dc_template_products WHERE template_id = ? ORDER BY sort_order", templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []TemplateProductInput
	for rows.Next() {
		var p TemplateProductInput
		if err := rows.Scan(&p.ProductID, &p.DefaultQuantity, &p.SortOrder); err != nil {
			return nil, err
		}
		products = append(products, p)
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
