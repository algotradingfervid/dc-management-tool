package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	htmxproducts "github.com/narendhupati/dc-management-tool/components/htmx/products"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	productspage "github.com/narendhupati/dc-management-tool/components/pages/products"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/xuri/excelize/v2"
)

func ListProducts(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Project not found")
		return c.Redirect(http.StatusFound, "/projects")
	}

	search := c.QueryParam("search")
	sortBy := c.QueryParam("sort")
	if sortBy == "" {
		sortBy = "item_name"
	}
	sortDir := c.QueryParam("dir")
	if sortDir == "" {
		sortDir = "asc"
	}
	pageStr := c.QueryParam("page")
	if pageStr == "" {
		pageStr = "1"
	}
	page, _ := strconv.Atoi(pageStr)

	productPage, err := database.SearchProducts(projectID, search, sortBy, sortDir, page, 20)
	if err != nil {
		slog.Error("error fetching products", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		productPage = &models.ProductPage{Products: []*models.Product{}, CurrentPage: 1, TotalPages: 1, PerPage: 20}
	}

	flashType, flashMessage := auth.PopFlash(c.Request())

	// If HTMX request for table only, return partial
	if c.Request().Header.Get("HX-Request") == "true" && c.QueryParam("partial") == "true" {
		return components.RenderOK(c, htmxproducts.ProductTable(htmxproducts.ProductTableProps{
			Products:    productPage.Products,
			ProductPage: productPage,
			ProjectID:   projectID,
			SortBy:      sortBy,
			SortDir:     sortDir,
			Search:      search,
		}))
	}

	allProjects, err := database.GetAccessibleProjects(user)
	if err != nil {
		slog.Warn("error fetching all projects for products page", slog.String("error", err.Error()))
		allProjects = []*models.Project{}
	}

	pageContent := productspage.List(
		user,
		project,
		allProjects,
		productPage,
		search,
		sortBy,
		sortDir,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Products", sidebar, topbar, flashMessage, flashType, pageContent))
}

func ShowAddProductForm(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	return components.RenderOK(c, htmxproducts.ProductForm(htmxproducts.ProductFormProps{
		ProjectID: projectID,
		Product:   models.Product{UoM: "Nos"},
		IsEdit:    false,
		Errors:    map[string]string{},
		CsrfToken: csrf.Token(c.Request()),
	}))
}

func CreateProductHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	price, _ := strconv.ParseFloat(c.FormValue("per_unit_price"), 64)
	gst, _ := strconv.ParseFloat(c.FormValue("gst_percentage"), 64)

	product := &models.Product{
		ProjectID:       projectID,
		ItemName:        strings.TrimSpace(c.FormValue("item_name")),
		ItemDescription: strings.TrimSpace(c.FormValue("item_description")),
		HSNCode:         strings.TrimSpace(c.FormValue("hsn_code")),
		UoM:             strings.TrimSpace(c.FormValue("uom")),
		BrandModel:      strings.TrimSpace(c.FormValue("brand_model")),
		PerUnitPrice:    price,
		GSTPercentage:   gst,
	}

	errors := helpers.ValidateStruct(product)
	if product.HSNCode != "" {
		hsnRegex := regexp.MustCompile(`^\d{6,8}$`)
		if !hsnRegex.MatchString(strings.TrimSpace(product.HSNCode)) {
			errors["hsn_code"] = "HSN code must be 6-8 digits"
		}
	}

	// Check uniqueness
	if _, ok := errors["item_name"]; !ok && product.ItemName != "" {
		unique, err := database.CheckProductNameUnique(projectID, product.ItemName, 0)
		if err != nil {
			slog.Warn("error checking product name uniqueness", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		} else if !unique {
			errors["item_name"] = "A product with this name already exists in this project"
		}
	}

	csrfToken := csrf.Token(c.Request())

	if len(errors) > 0 {
		return components.RenderOK(c, htmxproducts.ProductForm(htmxproducts.ProductFormProps{
			ProjectID: projectID,
			Product:   *product,
			IsEdit:    false,
			Errors:    errors,
			CsrfToken: csrfToken,
		}))
	}

	if err := database.CreateProductRecord(product); err != nil {
		slog.Error("error creating product", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		errors["general"] = "Failed to create product"
		return components.RenderOK(c, htmxproducts.ProductForm(htmxproducts.ProductFormProps{
			ProjectID: projectID,
			Product:   *product,
			IsEdit:    false,
			Errors:    errors,
			CsrfToken: csrfToken,
		}))
	}

	saveAndAdd := c.FormValue("save_and_add") == "true"

	if saveAndAdd {
		// Return a fresh form for adding another product
		c.Response().Header().Set("HX-Trigger", "productChanged")
		return components.RenderOK(c, htmxproducts.ProductForm(htmxproducts.ProductFormProps{
			ProjectID:      projectID,
			Product:        models.Product{UoM: "Nos"},
			IsEdit:         false,
			Errors:         map[string]string{},
			CsrfToken:      csrfToken,
			SuccessMessage: "Product added! Add another below.",
		}))
	}

	// Return HX-Trigger to close slide-over and refresh product list
	c.Response().Header().Set("HX-Trigger", "productChanged")
	return components.RenderOK(c, htmxproducts.ProductFormSuccess(htmxproducts.ProductFormSuccessProps{
		Message: "Product added successfully",
	}))
}

func ShowEditProductForm(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid product ID")
	}

	product, err := database.GetProductByID(productID)
	if err != nil || product.ProjectID != projectID {
		return c.String(http.StatusNotFound, "Product not found")
	}

	return components.RenderOK(c, htmxproducts.ProductForm(htmxproducts.ProductFormProps{
		ProjectID: projectID,
		Product:   *product,
		IsEdit:    true,
		Errors:    map[string]string{},
		CsrfToken: csrf.Token(c.Request()),
	}))
}

func UpdateProductHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid product ID")
	}

	existing, err := database.GetProductByID(productID)
	if err != nil || existing.ProjectID != projectID {
		return c.String(http.StatusNotFound, "Product not found")
	}

	price, _ := strconv.ParseFloat(c.FormValue("per_unit_price"), 64)
	gst, _ := strconv.ParseFloat(c.FormValue("gst_percentage"), 64)

	product := &models.Product{
		ID:              productID,
		ProjectID:       projectID,
		ItemName:        strings.TrimSpace(c.FormValue("item_name")),
		ItemDescription: strings.TrimSpace(c.FormValue("item_description")),
		HSNCode:         strings.TrimSpace(c.FormValue("hsn_code")),
		UoM:             strings.TrimSpace(c.FormValue("uom")),
		BrandModel:      strings.TrimSpace(c.FormValue("brand_model")),
		PerUnitPrice:    price,
		GSTPercentage:   gst,
	}

	errors := helpers.ValidateStruct(product)
	if product.HSNCode != "" {
		hsnRegex := regexp.MustCompile(`^\d{6,8}$`)
		if !hsnRegex.MatchString(strings.TrimSpace(product.HSNCode)) {
			errors["hsn_code"] = "HSN code must be 6-8 digits"
		}
	}

	// Check uniqueness excluding current product
	if _, ok := errors["item_name"]; !ok && product.ItemName != "" {
		unique, err := database.CheckProductNameUnique(projectID, product.ItemName, productID)
		if err != nil {
			slog.Warn("error checking product name uniqueness", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		} else if !unique {
			errors["item_name"] = "A product with this name already exists in this project"
		}
	}

	csrfToken := csrf.Token(c.Request())

	if len(errors) > 0 {
		return components.RenderOK(c, htmxproducts.ProductForm(htmxproducts.ProductFormProps{
			ProjectID: projectID,
			Product:   *product,
			IsEdit:    true,
			Errors:    errors,
			CsrfToken: csrfToken,
		}))
	}

	if err := database.UpdateProductRecord(product); err != nil {
		slog.Error("error updating product", slog.String("error", err.Error()), slog.Int("productID", productID), slog.Int("projectID", projectID))
		errors["general"] = "Failed to update product"
		return components.RenderOK(c, htmxproducts.ProductForm(htmxproducts.ProductFormProps{
			ProjectID: projectID,
			Product:   *product,
			IsEdit:    true,
			Errors:    errors,
			CsrfToken: csrfToken,
		}))
	}

	c.Response().Header().Set("HX-Trigger", "productChanged")
	return components.RenderOK(c, htmxproducts.ProductFormSuccess(htmxproducts.ProductFormSuccessProps{
		Message: "Product updated successfully",
	}))
}

func DeleteProductHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid product ID"})
	}

	product, err := database.GetProductByID(productID)
	if err != nil || product.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Product not found"})
	}

	if err := database.DeleteProductRecord(productID, projectID); err != nil {
		slog.Error("error deleting product", slog.String("error", err.Error()), slog.Int("productID", productID), slog.Int("projectID", projectID))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	c.Response().Header().Set("HX-Trigger", "productChanged")
	return c.String(http.StatusOK, "")
}

func BulkDeleteProductsHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	idsStr := c.FormValue("ids")
	if idsStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "No products selected"})
	}

	var ids []int
	for _, s := range strings.Split(idsStr, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "No valid product IDs"})
	}

	deleted, errs := database.BulkDeleteProducts(ids, projectID)

	c.Response().Header().Set("HX-Trigger", "productChanged")
	if len(errs) > 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"deleted": deleted,
			"errors":  errs,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"deleted": deleted})
}

func ImportProductsHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		c.Response().Header().Set("HX-Trigger", "importError")
		return components.RenderOK(c, htmxproducts.ProductImportResult(htmxproducts.ProductImportResultProps{
			Error: "Please select a file to upload",
		}))
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		c.Response().Header().Set("HX-Trigger", "importError")
		return components.RenderOK(c, htmxproducts.ProductImportResult(htmxproducts.ProductImportResultProps{
			Error: "File size must be less than 10MB",
		}))
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))

	var rows [][]string
	var headers []string

	switch ext {
	case ".csv":
		rows, headers, err = parseProductCSV(file)
	case ".xlsx", ".xls":
		rows, headers, err = parseProductExcel(file, header)
	default:
		return components.RenderOK(c, htmxproducts.ProductImportResult(htmxproducts.ProductImportResultProps{
			Error: "Only CSV and Excel (.xlsx) files are supported",
		}))
	}

	if err != nil {
		return components.RenderOK(c, htmxproducts.ProductImportResult(htmxproducts.ProductImportResultProps{
			Error: err.Error(),
		}))
	}

	if len(rows) == 0 {
		return components.RenderOK(c, htmxproducts.ProductImportResult(htmxproducts.ProductImportResultProps{
			Error: "File contains no data rows",
		}))
	}

	// Auto-map columns
	colMap := autoMapProductColumns(headers)

	result := &models.ProductImportResult{TotalRows: len(rows)}

	for i, row := range rows {
		product := mapRowToProduct(row, colMap, projectID)
		errs := helpers.ValidateStruct(product)
		if product.HSNCode != "" {
			hsnRegex := regexp.MustCompile(`^\d{6,8}$`)
			if !hsnRegex.MatchString(strings.TrimSpace(product.HSNCode)) {
				errs["hsn_code"] = "HSN code must be 6-8 digits"
			}
		}

		if _, ok := errs["item_name"]; !ok && product.ItemName != "" {
			unique, _ := database.CheckProductNameUnique(projectID, product.ItemName, 0)
			if !unique {
				errs["item_name"] = "Duplicate name"
			}
		}

		if len(errs) > 0 {
			result.Failed++
			for field, msg := range errs {
				result.Errors = append(result.Errors, models.ProductImportError{
					Row:   i + 2,
					Field: field,
					Error: msg,
				})
			}
			continue
		}

		if err := database.CreateProductRecord(product); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, models.ProductImportError{
				Row:   i + 2,
				Error: "Failed to save: " + err.Error(),
			})
			continue
		}
		result.Successful++
	}

	c.Response().Header().Set("HX-Trigger", "productChanged")
	return components.RenderOK(c, htmxproducts.ProductImportResult(htmxproducts.ProductImportResultProps{
		Result: result,
	}))
}

func DownloadProductImportTemplate(c echo.Context) error {
	header := []string{"Item Name", "Description", "HSN Code", "UoM", "Brand/Model", "Per Unit Price", "GST %"}
	example := []string{"Solar Panel 400W", "Monocrystalline 400W solar panel", "85414011", "Nos", "Tata Power Solar", "10000.00", "18"}

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=product_import_template.csv")

	writer := csv.NewWriter(c.Response().Writer)
	_ = writer.Write(header)
	_ = writer.Write(example)
	writer.Flush()
	return nil
}

func parseProductCSV(file io.Reader) (rows [][]string, errors []string, err error) {
	reader := csv.NewReader(file)
	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CSV: %v", err)
	}
	if len(allRows) < 2 {
		return nil, nil, nil
	}
	return allRows[1:], allRows[0], nil
}

func parseProductExcel(file io.Reader, _ *multipart.FileHeader) ([][]string, []string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("Excel file has no sheets")
	}

	allRows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read sheet: %v", err)
	}
	if len(allRows) < 2 {
		return nil, nil, nil
	}
	return allRows[1:], allRows[0], nil
}

func autoMapProductColumns(headers []string) map[string]int {
	colMap := make(map[string]int)
	for i, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		switch {
		case strings.Contains(h, "item") && strings.Contains(h, "name"):
			colMap["item_name"] = i
		case strings.Contains(h, "description") || strings.Contains(h, "desc"):
			colMap["item_description"] = i
		case strings.Contains(h, "hsn"):
			colMap["hsn_code"] = i
		case strings.Contains(h, "uom") || strings.Contains(h, "unit"):
			colMap["uom"] = i
		case strings.Contains(h, "brand") || strings.Contains(h, "model"):
			colMap["brand_model"] = i
		case strings.Contains(h, "price") || strings.Contains(h, "rate"):
			colMap["per_unit_price"] = i
		case strings.Contains(h, "gst"):
			colMap["gst_percentage"] = i
		}
	}
	return colMap
}

func mapRowToProduct(row []string, colMap map[string]int, projectID int) *models.Product {
	getVal := func(key string) string {
		if idx, ok := colMap[key]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	price, _ := strconv.ParseFloat(getVal("per_unit_price"), 64)
	gst, _ := strconv.ParseFloat(getVal("gst_percentage"), 64)

	uom := getVal("uom")
	if uom == "" {
		uom = "Nos"
	}

	return &models.Product{
		ProjectID:       projectID,
		ItemName:        getVal("item_name"),
		ItemDescription: getVal("item_description"),
		HSNCode:         getVal("hsn_code"),
		UoM:             uom,
		BrandModel:      getVal("brand_model"),
		PerUnitPrice:    price,
		GSTPercentage:   gst,
	}
}
