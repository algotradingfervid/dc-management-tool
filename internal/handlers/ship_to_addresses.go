package handlers

import (
	"fmt"
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
)

// ShowShipToPage renders the ship-to addresses page with table or grid view.
func ShowShipToPage(c *gin.Context) {
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

	config, err := database.GetOrCreateAddressConfig(projectID, "ship_to")
	if err != nil {
		log.Printf("Error getting ship-to config: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to load address configuration")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	if perPage < 1 || perPage > 200 {
		perPage = 50
	}
	search := c.Query("search")
	viewMode := c.DefaultQuery("view", "table")
	if viewMode != "grid" {
		viewMode = "table"
	}

	addressPage, err := database.ListAddresses(config.ID, page, perPage, search)
	if err != nil {
		log.Printf("Error listing ship-to addresses: %v", err)
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Ship To Addresses", URL: ""},
	)

	c.HTML(http.StatusOK, "addresses/ship-to.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"project":      project,
		"config":       config,
		"columns":      config.ColumnDefinitions,
		"addressPage":  addressPage,
		"search":       search,
		"viewMode":     viewMode,
		"perPage":      perPage,
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"csrfToken":    csrf.Token(c.Request),
		"csrfField":    csrf.TemplateField(c.Request),
	})
}

// UpdateShipToColumnConfig handles POST to update ship-to column definitions.
func UpdateShipToColumnConfig(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "ship_to")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

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

	if errs := config.ValidateColumns(); len(errs) > 0 {
		for _, msg := range errs {
			auth.SetFlash(c.Request, "error", msg)
			break
		}
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	colJSON, err := config.ColumnsToJSON()
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to save configuration")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	if err := database.UpdateAddressConfig(config.ID, colJSON); err != nil {
		log.Printf("Error updating ship-to config: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to save configuration")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	auth.SetFlash(c.Request, "success", "Column configuration updated successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
}

// UploadShipToAddresses handles CSV/Excel file upload for ship-to addresses.
func UploadShipToAddresses(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "ship_to")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Please select a file to upload")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		auth.SetFlash(c.Request, "error", "File size must be less than 10MB")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
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
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	if parseErr != nil {
		auth.SetFlash(c.Request, "error", parseErr.Error())
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	if len(rows) == 0 {
		auth.SetFlash(c.Request, "error", "File contains no data rows")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	if len(rows) > 10000 {
		auth.SetFlash(c.Request, "error", "Maximum 10,000 rows per upload")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	result := &models.UploadResult{TotalRows: len(rows), Mode: mode}
	var validAddresses []*models.Address

	for i, row := range rows {
		errs := database.ValidateAddressData(row, config.ColumnDefinitions)
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
			validAddresses = append(validAddresses, &models.Address{Data: row})
		}
	}

	if len(validAddresses) > 0 {
		if mode == "replace" {
			if err := database.DeleteAllAddresses(config.ID); err != nil {
				log.Printf("Error deleting ship-to addresses: %v", err)
				auth.SetFlash(c.Request, "error", "Failed to replace addresses")
				c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
				return
			}
		}
		if err := database.BulkInsertAddresses(config.ID, validAddresses); err != nil {
			log.Printf("Error inserting ship-to addresses: %v", err)
			auth.SetFlash(c.Request, "error", "Failed to save addresses")
			c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
			return
		}
	}

	msg := fmt.Sprintf("Upload complete: %d of %d addresses imported (%s mode)", result.Successful, result.TotalRows, mode)
	if result.Failed > 0 {
		msg += fmt.Sprintf(". %d rows failed validation.", result.Failed)
	}
	auth.SetFlash(c.Request, "success", msg)
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
}

// CreateShipToAddressHandler handles adding a single ship-to address.
func CreateShipToAddressHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "ship_to")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	data := make(map[string]string)
	for _, col := range config.ColumnDefinitions {
		data[col.Name] = strings.TrimSpace(c.PostForm("field_" + sanitizeFieldName(col.Name)))
	}

	errs := database.ValidateAddressData(data, config.ColumnDefinitions)
	if len(errs) > 0 {
		auth.SetFlash(c.Request, "error", strings.Join(errs, "; "))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	if _, err := database.CreateAddress(config.ID, data); err != nil {
		log.Printf("Error creating ship-to address: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to create address")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	auth.SetFlash(c.Request, "success", "Address added successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
}

// UpdateShipToAddressHandler handles editing a single ship-to address.
func UpdateShipToAddressHandler(c *gin.Context) {
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

	config, err := database.GetOrCreateAddressConfig(projectID, "ship_to")
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load config")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	data := make(map[string]string)
	for _, col := range config.ColumnDefinitions {
		data[col.Name] = strings.TrimSpace(c.PostForm("field_" + sanitizeFieldName(col.Name)))
	}

	errs := database.ValidateAddressData(data, config.ColumnDefinitions)
	if len(errs) > 0 {
		auth.SetFlash(c.Request, "error", strings.Join(errs, "; "))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	if err := database.UpdateAddress(addressID, data); err != nil {
		log.Printf("Error updating ship-to address: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to update address")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
		return
	}

	auth.SetFlash(c.Request, "success", "Address updated successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/ship-to", projectID))
}

// DeleteShipToAddressHandler handles DELETE for a single ship-to address.
func DeleteShipToAddressHandler(c *gin.Context) {
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

	config, err := database.GetOrCreateAddressConfig(projectID, "ship_to")
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

// DeleteAllShipToAddressesHandler handles DELETE for all ship-to addresses.
func DeleteAllShipToAddressesHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	config, err := database.GetOrCreateAddressConfig(projectID, "ship_to")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config"})
		return
	}

	if err := database.DeleteAllAddresses(config.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete addresses"})
		return
	}

	auth.SetFlash(c.Request, "success", "All addresses deleted")
	c.JSON(http.StatusOK, gin.H{"success": true, "redirect": fmt.Sprintf("/projects/%d/ship-to", projectID)})
}

// GetShipToAddressJSON returns a single ship-to address as JSON (for edit form).
func GetShipToAddressJSON(c *gin.Context) {
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
