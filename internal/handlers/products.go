package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/xuri/excelize/v2"
)

func ListProducts(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Project not found")
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "item_name")
	sortDir := c.DefaultQuery("dir", "asc")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	productPage, err := database.SearchProducts(projectID, search, sortBy, sortDir, page, 20)
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		productPage = &models.ProductPage{Products: []*models.Product{}, CurrentPage: 1, TotalPages: 1, PerPage: 20}
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Products", URL: ""},
	)

	// Get product count for sidebar badge
	productCount, _ := c.Get("productCount")

	data := gin.H{
		"user":           user,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"project":        project,
		"currentProject": project,
		"productCount":   productCount,
		"productPage":    productPage,
		"products":       productPage.Products,
		"search":         search,
		"sortBy":         sortBy,
		"sortDir":        sortDir,
		"activeTab":      "products",
		"flashType":      flashType,
		"flashMessage":   flashMessage,
		"csrfToken":      csrf.Token(c.Request),
		"csrfField":      csrf.TemplateField(c.Request),
	}

	// If HTMX request for table only, return partial
	if c.GetHeader("HX-Request") == "true" && c.Query("partial") == "true" {
		c.HTML(http.StatusOK, "htmx/products/table.html", data)
		return
	}

	c.HTML(http.StatusOK, "products/list.html", data)
}

func ShowAddProductForm(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
		"projectID": projectID,
		"product":   &models.Product{UoM: "Nos"},
		"errors":    map[string]string{},
		"isEdit":    false,
		"csrfField": csrf.TemplateField(c.Request),
	})
}

func CreateProductHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	price, _ := strconv.ParseFloat(c.PostForm("per_unit_price"), 64)
	gst, _ := strconv.ParseFloat(c.PostForm("gst_percentage"), 64)

	product := &models.Product{
		ProjectID:       projectID,
		ItemName:        strings.TrimSpace(c.PostForm("item_name")),
		ItemDescription: strings.TrimSpace(c.PostForm("item_description")),
		HSNCode:         strings.TrimSpace(c.PostForm("hsn_code")),
		UoM:             strings.TrimSpace(c.PostForm("uom")),
		BrandModel:      strings.TrimSpace(c.PostForm("brand_model")),
		PerUnitPrice:    price,
		GSTPercentage:   gst,
	}

	errors := product.Validate()

	// Check uniqueness
	if _, ok := errors["item_name"]; !ok && product.ItemName != "" {
		unique, err := database.CheckProductNameUnique(projectID, product.ItemName, 0)
		if err != nil {
			log.Printf("Error checking uniqueness: %v", err)
		} else if !unique {
			errors["item_name"] = "A product with this name already exists in this project"
		}
	}

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID": projectID,
			"product":   product,
			"errors":    errors,
			"isEdit":    false,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.CreateProductRecord(product); err != nil {
		log.Printf("Error creating product: %v", err)
		errors["general"] = "Failed to create product"
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID": projectID,
			"product":   product,
			"errors":    errors,
			"isEdit":    false,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	saveAndAdd := c.PostForm("save_and_add") == "true"

	if saveAndAdd {
		// Return a fresh form for adding another product
		c.Header("HX-Trigger", "productChanged")
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID":    projectID,
			"product":      &models.Product{UoM: "Nos"},
			"errors":       map[string]string{},
			"isEdit":       false,
			"csrfField":    csrf.TemplateField(c.Request),
			"successMessage": "Product added! Add another below.",
		})
		return
	}

	// Return HX-Trigger to close slide-over and refresh product list
	c.Header("HX-Trigger", "productChanged")
	c.HTML(http.StatusOK, "htmx/products/form-success.html", gin.H{
		"message": "Product added successfully",
	})
}

func ShowEditProductForm(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid product ID")
		return
	}

	product, err := database.GetProductByID(productID)
	if err != nil || product.ProjectID != projectID {
		c.String(http.StatusNotFound, "Product not found")
		return
	}

	c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
		"projectID": projectID,
		"product":   product,
		"errors":    map[string]string{},
		"isEdit":    true,
		"csrfField": csrf.TemplateField(c.Request),
	})
}

func UpdateProductHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid product ID")
		return
	}

	existing, err := database.GetProductByID(productID)
	if err != nil || existing.ProjectID != projectID {
		c.String(http.StatusNotFound, "Product not found")
		return
	}

	price, _ := strconv.ParseFloat(c.PostForm("per_unit_price"), 64)
	gst, _ := strconv.ParseFloat(c.PostForm("gst_percentage"), 64)

	product := &models.Product{
		ID:              productID,
		ProjectID:       projectID,
		ItemName:        strings.TrimSpace(c.PostForm("item_name")),
		ItemDescription: strings.TrimSpace(c.PostForm("item_description")),
		HSNCode:         strings.TrimSpace(c.PostForm("hsn_code")),
		UoM:             strings.TrimSpace(c.PostForm("uom")),
		BrandModel:      strings.TrimSpace(c.PostForm("brand_model")),
		PerUnitPrice:    price,
		GSTPercentage:   gst,
	}

	errors := product.Validate()

	// Check uniqueness excluding current product
	if _, ok := errors["item_name"]; !ok && product.ItemName != "" {
		unique, err := database.CheckProductNameUnique(projectID, product.ItemName, productID)
		if err != nil {
			log.Printf("Error checking uniqueness: %v", err)
		} else if !unique {
			errors["item_name"] = "A product with this name already exists in this project"
		}
	}

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID": projectID,
			"product":   product,
			"errors":    errors,
			"isEdit":    true,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.UpdateProductRecord(product); err != nil {
		log.Printf("Error updating product: %v", err)
		errors["general"] = "Failed to update product"
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID": projectID,
			"product":   product,
			"errors":    errors,
			"isEdit":    true,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	c.Header("HX-Trigger", "productChanged")
	c.HTML(http.StatusOK, "htmx/products/form-success.html", gin.H{
		"message": "Product updated successfully",
	})
}

func DeleteProductHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := database.GetProductByID(productID)
	if err != nil || product.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if err := database.DeleteProductRecord(productID, projectID); err != nil {
		log.Printf("Error deleting product: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Trigger", "productChanged")
	c.String(http.StatusOK, "")
}

func BulkDeleteProductsHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	idsStr := c.PostForm("ids")
	if idsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No products selected"})
		return
	}

	var ids []int
	for _, s := range strings.Split(idsStr, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid product IDs"})
		return
	}

	deleted, errs := database.BulkDeleteProducts(ids, projectID)

	c.Header("HX-Trigger", "productChanged")
	if len(errs) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"deleted": deleted,
			"errors":  errs,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func ImportProductsHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.Header("HX-Trigger", "importError")
		c.HTML(http.StatusOK, "htmx/products/import-result.html", gin.H{
			"error": "Please select a file to upload",
		})
		return
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		c.Header("HX-Trigger", "importError")
		c.HTML(http.StatusOK, "htmx/products/import-result.html", gin.H{
			"error": "File size must be less than 10MB",
		})
		return
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
		c.HTML(http.StatusOK, "htmx/products/import-result.html", gin.H{
			"error": "Only CSV and Excel (.xlsx) files are supported",
		})
		return
	}

	if err != nil {
		c.HTML(http.StatusOK, "htmx/products/import-result.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	if len(rows) == 0 {
		c.HTML(http.StatusOK, "htmx/products/import-result.html", gin.H{
			"error": "File contains no data rows",
		})
		return
	}

	// Auto-map columns
	colMap := autoMapProductColumns(headers)

	result := &models.ProductImportResult{TotalRows: len(rows)}

	for i, row := range rows {
		product := mapRowToProduct(row, colMap, projectID)
		errs := product.Validate()

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

	c.Header("HX-Trigger", "productChanged")
	c.HTML(http.StatusOK, "htmx/products/import-result.html", gin.H{
		"result": result,
	})
}

func DownloadProductImportTemplate(c *gin.Context) {
	header := []string{"Item Name", "Description", "HSN Code", "UoM", "Brand/Model", "Per Unit Price", "GST %"}
	example := []string{"Solar Panel 400W", "Monocrystalline 400W solar panel", "85414011", "Nos", "Tata Power Solar", "10000.00", "18"}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=product_import_template.csv")

	writer := csv.NewWriter(c.Writer)
	writer.Write(header)
	writer.Write(example)
	writer.Flush()
}

func parseProductCSV(file io.Reader) ([][]string, []string, error) {
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

func parseProductExcel(file io.Reader, header *multipart.FileHeader) ([][]string, []string, error) {
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
