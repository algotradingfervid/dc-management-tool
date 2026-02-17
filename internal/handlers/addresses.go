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

// validAddressTab returns the tab value if valid, otherwise "bill_to".
func validAddressTab(tab string) string {
	switch tab {
	case "bill_to", "ship_to", "bill_from", "dispatch_from":
		return tab
	default:
		return "bill_to"
	}
}

// ShowAddressesPage renders the unified addresses page with 4 address type tabs.
func ShowAddressesPage(c *gin.Context) {
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

	// Determine active tab
	tab := validAddressTab(c.DefaultQuery("tab", "bill_to"))

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
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
		helpers.Breadcrumb{Title: "Addresses", URL: ""},
	)

	// Get fixed columns for ship-to
	fixedColumns := models.FixedShipToColumns()

	c.HTML(http.StatusOK, "addresses/index.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"project":      project,
		"currentProject":  project,
		"config":       config,
		"columns":      config.ColumnDefinitions,
		"fixedColumns": fixedColumns,
		"addressPage":  addressPage,
		"search":       search,
		"tab":          tab,
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"csrfToken":    csrf.Token(c.Request),
		"csrfField":    csrf.TemplateField(c.Request),
	})
}

// UpdateAddressColumnConfig handles POST to update column definitions for either address type.
func UpdateAddressColumnConfig(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	tab := validAddressTab(c.DefaultQuery("tab", "bill_to"))

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

	// Parse column definitions from form
	columnsJSON := c.PostForm("columns_json")
	if columnsJSON == "" {
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
			c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
			return
		}
	}

	if errs := config.ValidateColumns(); len(errs) > 0 {
		for _, msg := range errs {
			auth.SetFlash(c.Request, "error", msg)
			break
		}
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	colJSON, err := config.ColumnsToJSON()
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to save configuration")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	if err := database.UpdateAddressConfig(config.ID, colJSON); err != nil {
		log.Printf("Error updating config: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to save configuration")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	auth.SetFlash(c.Request, "success", "Column configuration updated successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
}

// DownloadAddressImportTemplate generates a CSV template matching the current column config.
func DownloadAddressImportTemplate(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	tab := c.DefaultQuery("tab", "bill_to")
	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load column config"})
		return
	}

	var headers []string
	if tab == "ship_to" {
		for _, col := range models.FixedShipToColumns() {
			headers = append(headers, col.Name)
		}
	}
	for _, col := range config.ColumnDefinitions {
		headers = append(headers, col.Name)
	}

	filename := tab + "_address_template.csv"
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename="+filename)

	writer := csv.NewWriter(c.Writer)
	writer.Write(headers)
	writer.Flush()
}

// UploadAddressesHandler handles CSV/Excel file upload for either address type.
func UploadAddressesHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	tab := validAddressTab(c.DefaultQuery("tab", "bill_to"))

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Please select a file to upload")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		auth.SetFlash(c.Request, "error", "File size must be less than 10MB")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	mode := c.PostForm("mode")
	if mode != "append" {
		mode = "replace"
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))

	// For ship-to, we need to parse fixed columns too
	allColumns := config.ColumnDefinitions
	if tab == "ship_to" {
		allColumns = append(models.FixedShipToColumns(), config.ColumnDefinitions...)
	}

	var rows []map[string]string
	var parseErr error

	switch ext {
	case ".csv":
		rows, parseErr = parseCSVFile(file, allColumns)
	case ".xlsx", ".xls":
		rows, parseErr = parseExcelFile(file, header, allColumns)
	default:
		auth.SetFlash(c.Request, "error", "Only CSV and Excel (.xlsx) files are supported")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	if parseErr != nil {
		auth.SetFlash(c.Request, "error", parseErr.Error())
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	if len(rows) == 0 {
		auth.SetFlash(c.Request, "error", "File contains no data rows")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	if len(rows) > 10000 {
		auth.SetFlash(c.Request, "error", "Maximum 10,000 rows per upload")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	// Validate rows
	result := &models.UploadResult{TotalRows: len(rows), Mode: mode}
	var validAddresses []*models.Address

	fixedCols := models.FixedShipToColumns()

	for i, row := range rows {
		errs := database.ValidateAddressData(row, allColumns)
		if len(errs) > 0 {
			result.Failed++
			for _, e := range errs {
				result.Errors = append(result.Errors, models.UploadError{
					Row:   i + 2,
					Field: "",
					Error: e,
				})
			}
		} else {
			result.Successful++
			addr := &models.Address{}

			if tab == "ship_to" {
				// Extract fixed fields from row data
				addr.DistrictName = row[fixedCols[0].Name]
				addr.MandalName = row[fixedCols[1].Name]
				addr.MandalCode = row[fixedCols[2].Name]
				// Remove fixed fields from dynamic data
				dynamicData := make(map[string]string)
				for k, v := range row {
					isFixed := false
					for _, fc := range fixedCols {
						if k == fc.Name {
							isFixed = true
							break
						}
					}
					if !isFixed {
						dynamicData[k] = v
					}
				}
				addr.Data = dynamicData
			} else {
				addr.Data = row
			}

			validAddresses = append(validAddresses, addr)
		}
	}

	if len(validAddresses) > 0 {
		if mode == "replace" {
			if err := database.DeleteAllAddresses(config.ID); err != nil {
				log.Printf("Error deleting addresses: %v", err)
				auth.SetFlash(c.Request, "error", "Failed to replace addresses")
				c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
				return
			}
		}
		if err := database.BulkInsertAddresses(config.ID, validAddresses); err != nil {
			log.Printf("Error inserting addresses: %v", err)
			auth.SetFlash(c.Request, "error", "Failed to save addresses")
			c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
			return
		}
	}

	msg := fmt.Sprintf("Upload complete: %d of %d addresses imported (%s mode)", result.Successful, result.TotalRows, mode)
	if result.Failed > 0 {
		msg += fmt.Sprintf(". %d rows failed validation.", result.Failed)
	}
	auth.SetFlash(c.Request, "success", msg)
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
}

// CreateAddressUnified handles adding a single address for either type.
func CreateAddressUnified(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	tab := validAddressTab(c.DefaultQuery("tab", "bill_to"))

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	// Collect dynamic column data
	data := make(map[string]string)
	for _, col := range config.ColumnDefinitions {
		data[col.Name] = strings.TrimSpace(c.PostForm("field_" + sanitizeFieldName(col.Name)))
	}

	// Collect fixed fields for ship-to
	var districtName, mandalName, mandalCode string
	if tab == "ship_to" {
		districtName = strings.TrimSpace(c.PostForm("district_name"))
		mandalName = strings.TrimSpace(c.PostForm("mandal_name"))
		mandalCode = strings.TrimSpace(c.PostForm("mandal_code"))

		// Validate fixed fields
		if districtName == "" || mandalName == "" || mandalCode == "" {
			auth.SetFlash(c.Request, "error", "District Name, Mandal/ULB Name, and Mandal Code are required")
			c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
			return
		}
	}

	errs := database.ValidateAddressData(data, config.ColumnDefinitions)
	if len(errs) > 0 {
		auth.SetFlash(c.Request, "error", strings.Join(errs, "; "))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	if _, err := database.CreateAddress(config.ID, data, districtName, mandalName, mandalCode); err != nil {
		log.Printf("Error creating address: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to create address")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	auth.SetFlash(c.Request, "success", "Address added successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
}

// UpdateAddressUnified handles editing a single address.
func UpdateAddressUnified(c *gin.Context) {
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

	tab := validAddressTab(c.DefaultQuery("tab", "bill_to"))

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	data := make(map[string]string)
	for _, col := range config.ColumnDefinitions {
		data[col.Name] = strings.TrimSpace(c.PostForm("field_" + sanitizeFieldName(col.Name)))
	}

	var districtName, mandalName, mandalCode string
	if tab == "ship_to" {
		districtName = strings.TrimSpace(c.PostForm("district_name"))
		mandalName = strings.TrimSpace(c.PostForm("mandal_name"))
		mandalCode = strings.TrimSpace(c.PostForm("mandal_code"))

		if districtName == "" || mandalName == "" || mandalCode == "" {
			auth.SetFlash(c.Request, "error", "District Name, Mandal/ULB Name, and Mandal Code are required")
			c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
			return
		}
	}

	errs := database.ValidateAddressData(data, config.ColumnDefinitions)
	if len(errs) > 0 {
		auth.SetFlash(c.Request, "error", strings.Join(errs, "; "))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	if err := database.UpdateAddress(addressID, data, districtName, mandalName, mandalCode); err != nil {
		log.Printf("Error updating address: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to update address")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		return
	}

	auth.SetFlash(c.Request, "success", "Address updated successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
}

// DeleteAddressUnified handles DELETE for a single address.
func DeleteAddressUnified(c *gin.Context) {
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

	tab := validAddressTab(c.DefaultQuery("tab", "bill_to"))

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
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

// DeleteAllAddressesUnified handles DELETE for all addresses.
func DeleteAllAddressesUnified(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	tab := validAddressTab(c.DefaultQuery("tab", "bill_to"))

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

	if err := database.DeleteAllAddresses(config.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete addresses"})
		return
	}

	auth.SetFlash(c.Request, "success", "All addresses deleted")
	c.JSON(http.StatusOK, gin.H{"success": true, "redirect": fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab)})
}

// GetAddressJSONUnified returns a single address as JSON (for edit form).
func GetAddressJSONUnified(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{
		"id":            addr.ID,
		"data":          addr.Data,
		"district_name": addr.DistrictName,
		"mandal_name":   addr.MandalName,
		"mandal_code":   addr.MandalCode,
	})
}

// SearchAddressSelector handles the HTMX-powered address selector search.
func SearchAddressSelector(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	addressType := validAddressTab(c.DefaultQuery("type", "ship_to"))

	search := c.Query("q")

	config, err := database.GetOrCreateAddressConfig(projectID, addressType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

	addresses, err := database.SearchAddressesForSelector(config.ID, search, addressType, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search addresses"})
		return
	}

	c.HTML(http.StatusOK, "htmx/address-selector-results.html", gin.H{
		"addresses":   addresses,
		"addressType": addressType,
		"columns":     config.ColumnDefinitions,
	})
}

// parseCSVFile parses a CSV file and maps headers to column definitions.
func parseCSVFile(file io.Reader, columns []models.ColumnDefinition) ([]map[string]string, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[h] = i
	}

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

// parseExcelFile parses an Excel file and maps headers to column definitions.
func parseExcelFile(file io.Reader, header interface{}, columns []models.ColumnDefinition) ([]map[string]string, error) {
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

	headers := excelRows[0]
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

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
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ToLower(name)
	return name
}
