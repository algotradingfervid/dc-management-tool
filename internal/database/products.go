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
