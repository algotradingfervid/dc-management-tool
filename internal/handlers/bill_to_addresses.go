package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// ShowBillToPage renders the bill-to addresses page.
func ShowBillToPage(c *gin.Context) {
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

	config, err := database.GetOrCreateAddressConfig(projectID, "bill_to")
	if err != nil {
		log.Printf("Error getting address config: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to load address configuration")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	search := c.Query("search")

	addressPage, err := database.ListAddresses(config.ID, page, 50, search)
	if err != nil {
		log.Printf("Error listing addresses: %v", err)
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Bill To Addresses", URL: ""},
	)

	c.HTML(http.StatusOK, "addresses/bill-to.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"project":      project,
		"config":       config,
		"columns":      config.ColumnDefinitions,
		"addressPage":  addressPage,
		"search":       search,
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"csrfToken":    csrf.Token(c.Request),
		"csrfField":    csrf.TemplateField(c.Request),
	})
}

// UpdateColumnConfig handles POST to update column definitions.
func UpdateColumnConfig(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "bill_to")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

	// Parse column definitions from form
	columnsJSON := c.PostForm("columns_json")
	if columnsJSON == "" {
		// Build from form fields
		var columns []models.ColumnDefinition
		names := c.PostFormArray("col_name[]")
		requireds := c.PostFormArray("col_required[]")

		for i, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			req := false
			if i < len(requireds) && requireds[i] == "true" {
				req = true
			}
			columns = append(columns, models.ColumnDefinition{
				Name:     name,
				Required: req,
				Type:     "text",
			})
		}
		config.ColumnDefinitions = columns
	} else {
		if err := json.Unmarshal([]byte(columnsJSON), &config.ColumnDefinitions); err != nil {
			auth.SetFlash(c.Request, "error", "Invalid column configuration")
			c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
			return
		}
	}

	if errs := config.ValidateColumns(); len(errs) > 0 {
		for _, msg := range errs {
			auth.SetFlash(c.Request, "error", msg)
			break
		}
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	colJSON, err := config.ColumnsToJSON()
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to save configuration")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	if err := database.UpdateAddressConfig(config.ID, colJSON); err != nil {
		log.Printf("Error updating config: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to save configuration")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	auth.SetFlash(c.Request, "success", "Column configuration updated successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
}

// UploadAddresses handles CSV/Excel file upload.
func UploadAddresses(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "bill_to")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Please select a file to upload")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}
	defer file.Close()

	// Check file size (10MB)
	if header.Size > 10*1024*1024 {
		auth.SetFlash(c.Request, "error", "File size must be less than 10MB")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	mode := c.PostForm("mode")
	if mode != "append" {
		mode = "replace"
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))

	var rows []map[string]string
	var parseErr error

	switch ext {
	case ".csv":
		rows, parseErr = parseCSV(file, config.ColumnDefinitions)
	case ".xlsx", ".xls":
		rows, parseErr = parseExcel(file, header, config.ColumnDefinitions)
	default:
		auth.SetFlash(c.Request, "error", "Only CSV and Excel (.xlsx) files are supported")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	if parseErr != nil {
		auth.SetFlash(c.Request, "error", parseErr.Error())
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	if len(rows) == 0 {
		auth.SetFlash(c.Request, "error", "File contains no data rows")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	if len(rows) > 10000 {
		auth.SetFlash(c.Request, "error", "Maximum 10,000 rows per upload")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	// Validate rows and build addresses
	result := &models.UploadResult{TotalRows: len(rows), Mode: mode}
	var validAddresses []*models.Address

	for i, row := range rows {
		errs := database.ValidateAddressData(row, config.ColumnDefinitions)
		if len(errs) > 0 {
			result.Failed++
			for _, e := range errs {
				result.Errors = append(result.Errors, models.UploadError{
					Row:   i + 2, // +2 for header row + 0-index
					Field: "",
					Error: e,
				})
			}
		} else {
			result.Successful++
			validAddresses = append(validAddresses, &models.Address{Data: row})
		}
	}

	// Execute upload
	if len(validAddresses) > 0 {
		if mode == "replace" {
			if err := database.DeleteAllAddresses(config.ID); err != nil {
				log.Printf("Error deleting addresses: %v", err)
				auth.SetFlash(c.Request, "error", "Failed to replace addresses")
				c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
				return
			}
		}
		if err := database.BulkInsertAddresses(config.ID, validAddresses); err != nil {
			log.Printf("Error inserting addresses: %v", err)
			auth.SetFlash(c.Request, "error", "Failed to save addresses")
			c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
			return
		}
	}

	msg := fmt.Sprintf("Upload complete: %d of %d addresses imported (%s mode)", result.Successful, result.TotalRows, mode)
	if result.Failed > 0 {
		msg += fmt.Sprintf(". %d rows failed validation.", result.Failed)
	}
	auth.SetFlash(c.Request, "success", msg)
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
}

// CreateAddressHandler handles adding a single address.
func CreateAddressHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "bill_to")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	data := make(map[string]string)
	for _, col := range config.ColumnDefinitions {
		data[col.Name] = strings.TrimSpace(c.PostForm("field_" + sanitizeFieldName(col.Name)))
	}

	errs := database.ValidateAddressData(data, config.ColumnDefinitions)
	if len(errs) > 0 {
		auth.SetFlash(c.Request, "error", strings.Join(errs, "; "))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	if _, err := database.CreateAddress(config.ID, data); err != nil {
		log.Printf("Error creating address: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to create address")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	auth.SetFlash(c.Request, "success", "Address added successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
}

// UpdateAddressHandler handles editing a single address.
func UpdateAddressHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	addressID, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "bill_to")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	data := make(map[string]string)
	for _, col := range config.ColumnDefinitions {
		data[col.Name] = strings.TrimSpace(c.PostForm("field_" + sanitizeFieldName(col.Name)))
	}

	errs := database.ValidateAddressData(data, config.ColumnDefinitions)
	if len(errs) > 0 {
		auth.SetFlash(c.Request, "error", strings.Join(errs, "; "))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	if err := database.UpdateAddress(addressID, data); err != nil {
		log.Printf("Error updating address: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to update address")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
		return
	}

	auth.SetFlash(c.Request, "success", "Address updated successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/bill-to", projectID))
}

// DeleteAddressHandler handles DELETE for a single address.
func DeleteAddressHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	addressID, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "bill_to")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

	if err := database.DeleteAddress(addressID, config.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteAllAddressesHandler handles DELETE for all addresses.
func DeleteAllAddressesHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "bill_to")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

	if err := database.DeleteAllAddresses(config.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete addresses"})
		return
	}

	auth.SetFlash(c.Request, "success", "All addresses deleted")
	c.JSON(http.StatusOK, gin.H{"success": true, "redirect": fmt.Sprintf("/projects/%d/bill-to", projectID)})
}

// GetAddressJSON returns a single address as JSON (for edit form).
func GetAddressJSON(c *gin.Context) {
	addressID, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	addr, err := database.GetAddress(addressID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": addr.ID, "data": addr.Data})
}

// parseCSV parses a CSV file and maps headers to column definitions.
func parseCSV(file io.Reader, columns []models.ColumnDefinition) ([]map[string]string, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Trim headers
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	// Build header index
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[h] = i
	}

	// Verify required columns exist
	for _, col := range columns {
		if col.Required {
			if _, ok := headerMap[col.Name]; !ok {
				return nil, fmt.Errorf("missing required column: %s", col.Name)
			}
		}
	}

	var rows []map[string]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV row: %w", err)
		}

		row := make(map[string]string)
		for _, col := range columns {
			if idx, ok := headerMap[col.Name]; ok && idx < len(record) {
				row[col.Name] = strings.TrimSpace(record[idx])
			}
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// parseExcel parses an Excel file and maps headers to column definitions.
func parseExcel(file io.Reader, header interface{}, columns []models.ColumnDefinition) ([]map[string]string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	excelRows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel sheet: %w", err)
	}

	if len(excelRows) < 2 {
		return nil, fmt.Errorf("Excel file has no data rows (only header or empty)")
	}

	// Build header map from first row
	headers := excelRows[0]
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

	// Verify required columns
	for _, col := range columns {
		if col.Required {
			if _, ok := headerMap[col.Name]; !ok {
				return nil, fmt.Errorf("missing required column: %s", col.Name)
			}
		}
	}

	var rows []map[string]string
	for _, excelRow := range excelRows[1:] {
		row := make(map[string]string)
		for _, col := range columns {
			if idx, ok := headerMap[col.Name]; ok && idx < len(excelRow) {
				row[col.Name] = strings.TrimSpace(excelRow[idx])
			}
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// sanitizeFieldName converts a column name to a safe form field name.
func sanitizeFieldName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ToLower(name)
	return name
}
