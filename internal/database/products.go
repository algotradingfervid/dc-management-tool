package database

import (
	"context"
	"database/sql"
	"fmt"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// productFromRow converts a sqlc db.Product row into a models.Product.
func productFromRow(p db.Product) *models.Product {
	return &models.Product{
		ID:              int(p.ID),
		ProjectID:       int(p.ProjectID),
		ItemName:        p.ItemName,
		ItemDescription: p.ItemDescription,
		HSNCode:         p.HsnCode.String,
		UoM:             p.Uom.String,
		BrandModel:      p.BrandModel,
		PerUnitPrice:    p.PerUnitPrice.Float64,
		GSTPercentage:   p.GstPercentage.Float64,
		CreatedAt:       p.CreatedAt.Time,
		UpdatedAt:       p.UpdatedAt.Time,
	}
}

func GetProductsByProjectID(projectID int) ([]*models.Product, error) {
	q := db.New(DB)
	rows, err := q.GetProductsByProjectID(context.Background(), int64(projectID))
	if err != nil {
		return nil, err
	}

	products := make([]*models.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, productFromRow(row))
	}
	return products, nil
}

func GetProductByID(id int) (*models.Product, error) {
	q := db.New(DB)
	row, err := q.GetProductByID(context.Background(), int64(id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found")
	}
	if err != nil {
		return nil, err
	}
	return productFromRow(row), nil
}

func CreateProductRecord(p *models.Product) error {
	q := db.New(DB)
	result, err := q.CreateProduct(context.Background(), db.CreateProductParams{
		ProjectID:       int64(p.ProjectID),
		ItemName:        p.ItemName,
		ItemDescription: p.ItemDescription,
		HsnCode:         sql.NullString{String: p.HSNCode, Valid: p.HSNCode != ""},
		Uom:             sql.NullString{String: p.UoM, Valid: p.UoM != ""},
		BrandModel:      p.BrandModel,
		PerUnitPrice:    sql.NullFloat64{Float64: p.PerUnitPrice, Valid: true},
		GstPercentage:   sql.NullFloat64{Float64: p.GSTPercentage, Valid: true},
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

func UpdateProductRecord(p *models.Product) error {
	q := db.New(DB)
	return q.UpdateProduct(context.Background(), db.UpdateProductParams{
		ItemName:        p.ItemName,
		ItemDescription: p.ItemDescription,
		HsnCode:         sql.NullString{String: p.HSNCode, Valid: p.HSNCode != ""},
		Uom:             sql.NullString{String: p.UoM, Valid: p.UoM != ""},
		BrandModel:      p.BrandModel,
		PerUnitPrice:    sql.NullFloat64{Float64: p.PerUnitPrice, Valid: true},
		GstPercentage:   sql.NullFloat64{Float64: p.GSTPercentage, Valid: true},
		ID:              int64(p.ID),
		ProjectID:       int64(p.ProjectID),
	})
}

func DeleteProductRecord(id, projectID int) error {
	used, err := CheckProductUsageInTemplates(id)
	if err != nil {
		return err
	}
	if used {
		return fmt.Errorf("cannot delete product: it is used in DC templates")
	}

	q := db.New(DB)
	return q.DeleteProduct(context.Background(), db.DeleteProductParams{
		ID:        int64(id),
		ProjectID: int64(projectID),
	})
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

	q := db.New(DB)
	n, err := q.CheckProductUsageInTemplates(context.Background(), int64(productID))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func CheckProductNameUnique(projectID int, itemName string, excludeID int) (bool, error) {
	q := db.New(DB)
	if excludeID > 0 {
		count, err := q.CheckProductNameUniqueExcludeID(context.Background(), db.CheckProductNameUniqueExcludeIDParams{
			ProjectID: int64(projectID),
			ItemName:  itemName,
			ID:        int64(excludeID),
		})
		if err != nil {
			return false, err
		}
		return count == 0, nil
	}

	count, err := q.CheckProductNameUnique(context.Background(), db.CheckProductNameUniqueParams{
		ProjectID: int64(projectID),
		ItemName:  itemName,
	})
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// GetProductCount returns the total number of products for a project.
func GetProductCount(projectID int) (int, error) {
	q := db.New(DB)
	count, err := q.GetProductCount(context.Background(), int64(projectID))
	return int(count), err
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

	q := db.New(DB)
	ctx := context.Background()

	var total int64
	var rows []db.Product
	var err error

	if search != "" {
		like := "%" + search + "%"
		likeNull := sql.NullString{String: like, Valid: true}

		// Get total count with filter
		total, err = q.SearchProductsCount(ctx, db.SearchProductsCountParams{
			ProjectID:       int64(projectID),
			ItemName:        like,
			HsnCode:         likeNull,
			BrandModel:      like,
			ItemDescription: like,
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Get total count without filter
		total, err = q.SearchProductsCountNoFilter(ctx, int64(projectID))
		if err != nil {
			return nil, err
		}
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * perPage

	// For non-default sort orders, fall back to hand-written SQL since sqlc
	// uses fixed ORDER BY item_name ASC. Only use sqlc for the default sort.
	if sortBy == "item_name" && sortDir == "asc" {
		if search != "" {
			like := "%" + search + "%"
			likeNull := sql.NullString{String: like, Valid: true}
			rows, err = q.SearchProducts(ctx, db.SearchProductsParams{
				ProjectID:       int64(projectID),
				ItemName:        like,
				HsnCode:         likeNull,
				BrandModel:      like,
				ItemDescription: like,
				Limit:           int64(perPage),
				Offset:          int64(offset),
			})
		} else {
			rows, err = q.SearchProductsNoFilter(ctx, db.SearchProductsNoFilterParams{
				ProjectID: int64(projectID),
				Limit:     int64(perPage),
				Offset:    int64(offset),
			})
		}
		if err != nil {
			return nil, err
		}
	} else {
		// Dynamic ORDER BY — keep hand-written SQL
		var queryArgs []interface{}
		where := "WHERE project_id = ?"
		queryArgs = append(queryArgs, projectID)

		if search != "" {
			where += " AND (item_name LIKE ? OR hsn_code LIKE ? OR brand_model LIKE ? OR item_description LIKE ?)"
			like := "%" + search + "%"
			queryArgs = append(queryArgs, like, like, like, like)
		}

		query := fmt.Sprintf(`
			SELECT id, project_id, item_name, item_description, hsn_code, uom,
			       brand_model, per_unit_price, gst_percentage, created_at, updated_at
			FROM products %s
			ORDER BY %s %s
			LIMIT ? OFFSET ?
		`, where, sortBy, sortDir)

		queryArgs = append(queryArgs, perPage, offset)
		dbRows, qErr := DB.Query(query, queryArgs...)
		if qErr != nil {
			return nil, qErr
		}
		defer dbRows.Close()

		var products []*models.Product
		for dbRows.Next() {
			p := &models.Product{}
			var hsnCode, uom sql.NullString
			var perUnitPrice, gstPct sql.NullFloat64
			var createdAt, updatedAt sql.NullTime
			scanErr := dbRows.Scan(
				&p.ID, &p.ProjectID, &p.ItemName, &p.ItemDescription,
				&hsnCode, &uom, &p.BrandModel,
				&perUnitPrice, &gstPct, &createdAt, &updatedAt,
			)
			if scanErr != nil {
				return nil, scanErr
			}
			p.HSNCode = hsnCode.String
			p.UoM = uom.String
			p.PerUnitPrice = perUnitPrice.Float64
			p.GSTPercentage = gstPct.Float64
			p.CreatedAt = createdAt.Time
			p.UpdatedAt = updatedAt.Time
			products = append(products, p)
		}

		return &models.ProductPage{
			Products:    products,
			CurrentPage: page,
			PerPage:     perPage,
			TotalCount:  int(total),
			TotalPages:  totalPages,
			Search:      search,
			SortBy:      sortBy,
			SortDir:     sortDir,
		}, nil
	}

	products := make([]*models.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, productFromRow(row))
	}

	return &models.ProductPage{
		Products:    products,
		CurrentPage: page,
		PerPage:     perPage,
		TotalCount:  int(total),
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
	q := db.New(DB)
	ctx := context.Background()

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
		err = q.DeleteProduct(ctx, db.DeleteProductParams{
			ID:        int64(id),
			ProjectID: int64(projectID),
		})
		if err != nil {
			errors = append(errors, fmt.Sprintf("Product %d: delete failed", id))
			continue
		}
		deleted++
	}
	return deleted, errors
}
