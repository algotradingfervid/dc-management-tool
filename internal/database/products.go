package database

import (
	"database/sql"
	"fmt"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

func GetProductsByProjectID(projectID int) ([]*models.Product, error) {
	query := `
		SELECT id, project_id, item_name, item_description, hsn_code, uom,
		       brand_model, per_unit_price, gst_percentage, created_at, updated_at
		FROM products
		WHERE project_id = ?
		ORDER BY item_name ASC
	`

	rows, err := DB.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*models.Product
	for rows.Next() {
		p := &models.Product{}
		var hsnCode, brandModel sql.NullString
		err := rows.Scan(
			&p.ID, &p.ProjectID, &p.ItemName, &p.ItemDescription,
			&hsnCode, &p.UoM, &brandModel,
			&p.PerUnitPrice, &p.GSTPercentage, &p.CreatedAt, &p.UpdatedAt,
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

func GetProductByID(id int) (*models.Product, error) {
	query := `
		SELECT id, project_id, item_name, item_description, hsn_code, uom,
		       brand_model, per_unit_price, gst_percentage, created_at, updated_at
		FROM products
		WHERE id = ?
	`

	p := &models.Product{}
	var hsnCode, brandModel sql.NullString
	err := DB.QueryRow(query, id).Scan(
		&p.ID, &p.ProjectID, &p.ItemName, &p.ItemDescription,
		&hsnCode, &p.UoM, &brandModel,
		&p.PerUnitPrice, &p.GSTPercentage, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found")
	}
	if err != nil {
		return nil, err
	}
	if hsnCode.Valid {
		p.HSNCode = hsnCode.String
	}
	if brandModel.Valid {
		p.BrandModel = brandModel.String
	}

	return p, nil
}

func CreateProductRecord(p *models.Product) error {
	query := `
		INSERT INTO products (
			project_id, item_name, item_description, hsn_code, uom,
			brand_model, per_unit_price, gst_percentage
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := DB.Exec(query,
		p.ProjectID, p.ItemName, p.ItemDescription, p.HSNCode, p.UoM,
		p.BrandModel, p.PerUnitPrice, p.GSTPercentage,
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

func UpdateProductRecord(p *models.Product) error {
	query := `
		UPDATE products SET
			item_name = ?, item_description = ?, hsn_code = ?, uom = ?,
			brand_model = ?, per_unit_price = ?, gst_percentage = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND project_id = ?
	`

	_, err := DB.Exec(query,
		p.ItemName, p.ItemDescription, p.HSNCode, p.UoM,
		p.BrandModel, p.PerUnitPrice, p.GSTPercentage,
		p.ID, p.ProjectID,
	)
	return err
}

func DeleteProductRecord(id, projectID int) error {
	used, err := CheckProductUsageInTemplates(id)
	if err != nil {
		return err
	}
	if used {
		return fmt.Errorf("cannot delete product: it is used in DC templates")
	}

	_, err = DB.Exec("DELETE FROM products WHERE id = ? AND project_id = ?", id, projectID)
	return err
}

func CheckProductUsageInTemplates(productID int) (bool, error) {
	// Check dc_template_products table if it exists
	var count int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='dc_template_products'",
	).Scan(&count)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}

	err = DB.QueryRow("SELECT COUNT(*) FROM dc_template_products WHERE product_id = ?", productID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func CheckProductNameUnique(projectID int, itemName string, excludeID int) (bool, error) {
	query := "SELECT COUNT(*) FROM products WHERE project_id = ? AND item_name = ?"
	args := []interface{}{projectID, itemName}

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

// GetProductCount returns the total number of products for a project.
func GetProductCount(projectID int) (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM products WHERE project_id = ?", projectID).Scan(&count)
	return count, err
}

// SearchProducts returns paginated, searchable, sortable product results.
func SearchProducts(projectID int, search string, sortBy string, sortDir string, page int, perPage int) (*models.ProductPage, error) {
	// Validate sort params
	allowedSorts := map[string]bool{
		"item_name": true, "hsn_code": true, "uom": true,
		"brand_model": true, "per_unit_price": true, "gst_percentage": true, "created_at": true,
	}
	if !allowedSorts[sortBy] {
		sortBy = "item_name"
	}
	if sortDir != "desc" {
		sortDir = "asc"
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Build WHERE clause
	where := "WHERE project_id = ?"
	args := []interface{}{projectID}
	if search != "" {
		where += " AND (item_name LIKE ? OR hsn_code LIKE ? OR brand_model LIKE ? OR item_description LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM products " + where
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * perPage

	// Query products
	query := fmt.Sprintf(`
		SELECT id, project_id, item_name, item_description, hsn_code, uom,
		       brand_model, per_unit_price, gst_percentage, created_at, updated_at
		FROM products %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, where, sortBy, sortDir)

	queryArgs := append(args, perPage, offset)
	rows, err := DB.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*models.Product
	for rows.Next() {
		p := &models.Product{}
		var hsnCode, brandModel sql.NullString
		err := rows.Scan(
			&p.ID, &p.ProjectID, &p.ItemName, &p.ItemDescription,
			&hsnCode, &p.UoM, &brandModel,
			&p.PerUnitPrice, &p.GSTPercentage, &p.CreatedAt, &p.UpdatedAt,
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

	return &models.ProductPage{
		Products:    products,
		CurrentPage: page,
		PerPage:     perPage,
		TotalCount:  total,
		TotalPages:  totalPages,
		Search:      search,
		SortBy:      sortBy,
		SortDir:     sortDir,
	}, nil
}

// BulkDeleteProducts deletes multiple products by IDs, checking usage first.
func BulkDeleteProducts(ids []int, projectID int) (int, []string) {
	var deleted int
	var errors []string
	for _, id := range ids {
		used, err := CheckProductUsageInTemplates(id)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Product %d: check failed", id))
			continue
		}
		if used {
			errors = append(errors, fmt.Sprintf("Product %d: used in DC templates", id))
			continue
		}
		_, err = DB.Exec("DELETE FROM products WHERE id = ? AND project_id = ?", id, projectID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Product %d: delete failed", id))
			continue
		}
		deleted++
	}
	return deleted, errors
}
