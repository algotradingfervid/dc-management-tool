package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/components/htmx"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	addressespkg "github.com/narendhupati/dc-management-tool/components/pages/addresses"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
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
func ShowAddressesPage(c echo.Context) error {
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

	// Determine active tab
	tab := validAddressTab(func() string {
		if v := c.QueryParam("tab"); v != "" {
			return v
		}
		return "bill_to"
	}())

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		slog.Error("error getting address config", slog.String("error", err.Error()), slog.Int("projectID", projectID), slog.String("tab", tab))
		auth.SetFlash(c.Request(), "error", "Failed to load address configuration")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	pageParam := c.QueryParam("page")
	if pageParam == "" {
		pageParam = "1"
	}
	page, _ := strconv.Atoi(pageParam)
	search := c.QueryParam("search")

	addressPage, err := database.ListAddresses(config.ID, page, 50, search)
	if err != nil {
		slog.Error("error listing addresses", slog.String("error", err.Error()), slog.Int("configID", config.ID))
	}

	flashType, flashMessage := auth.PopFlash(c.Request())

	helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Addresses", URL: ""},
	)

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := addressespkg.Index(
		user,
		project,
		allProjects,
		addressPage,
		search,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
		tab,
		config,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Addresses", sidebar, topbar, flashMessage, flashType, pageContent))
}

// UpdateAddressColumnConfig handles POST to update column definitions for either address type.
func UpdateAddressColumnConfig(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	tab := validAddressTab(func() string {
		if v := c.QueryParam("tab"); v != "" {
			return v
		}
		return "bill_to"
	}())

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to load config"})
	}

	// Parse column definitions from form
	columnsJSON := c.FormValue("columns_json")
	if columnsJSON == "" {
		var columns []models.ColumnDefinition

		// Parse fixed columns — name/required are locked, visibility/order are editable
		if len(models.FixedColumnsForType(tab)) > 0 {
			fixedNames := c.Request().Form["fixed_col_name[]"]
			fixedShowTables := c.Request().Form["fixed_show_table[]"]
			fixedTableOrders := c.Request().Form["fixed_table_order[]"]
			fixedShowPrints := c.Request().Form["fixed_show_print[]"]
			fixedPrintOrders := c.Request().Form["fixed_print_order[]"]

			for i, name := range fixedNames {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				col := models.ColumnDefinition{
					Name:     name,
					Required: true,
					Type:     "text",
					Fixed:    true,
				}
				if i < len(fixedShowTables) {
					v := fixedShowTables[i] == "true"
					col.ShowInTable = &v
				}
				if i < len(fixedTableOrders) {
					if order, parseErr := strconv.Atoi(fixedTableOrders[i]); parseErr == nil && order >= 0 {
						col.TableSortOrder = order
					}
				}
				if i < len(fixedShowPrints) {
					v := fixedShowPrints[i] == "true"
					col.ShowInPrint = &v
				}
				if i < len(fixedPrintOrders) {
					if order, parseErr := strconv.Atoi(fixedPrintOrders[i]); parseErr == nil && order >= 0 {
						col.PrintSortOrder = order
					}
				}
				columns = append(columns, col)
			}
		}

		// Build set of fixed column names to skip duplicates in dynamic section
		fixedNameSet := make(map[string]bool)
		for _, fc := range models.FixedColumnsForType(tab) {
			fixedNameSet[fc.Name] = true
		}

		// Parse dynamic columns
		names := c.Request().Form["col_name[]"]
		requireds := c.Request().Form["col_required[]"]
		showTables := c.Request().Form["col_show_table[]"]
		tableOrders := c.Request().Form["col_table_order[]"]
		showPrints := c.Request().Form["col_show_print[]"]
		printOrders := c.Request().Form["col_print_order[]"]

		for i, name := range names {
			name = strings.TrimSpace(name)
			if name == "" || fixedNameSet[name] {
				continue
			}
			req := false
			if i < len(requireds) && requireds[i] == "true" {
				req = true
			}

			col := models.ColumnDefinition{
				Name:     name,
				Required: req,
				Type:     "text",
			}

			// Parse table visibility (default: true)
			if i < len(showTables) {
				v := showTables[i] == "true"
				col.ShowInTable = &v
			}

			// Parse table sort order
			if i < len(tableOrders) {
				if order, err := strconv.Atoi(tableOrders[i]); err == nil && order >= 0 {
					col.TableSortOrder = order
				}
			}

			// Parse print visibility (default: true)
			if i < len(showPrints) {
				v := showPrints[i] == "true"
				col.ShowInPrint = &v
			}

			// Parse print sort order
			if i < len(printOrders) {
				if order, err := strconv.Atoi(printOrders[i]); err == nil && order >= 0 {
					col.PrintSortOrder = order
				}
			}

			columns = append(columns, col)
		}
		config.ColumnDefinitions = columns
	} else {
		if unmarshalErr := json.Unmarshal([]byte(columnsJSON), &config.ColumnDefinitions); unmarshalErr != nil {
			auth.SetFlash(c.Request(), "error", "Invalid column configuration")
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		}
	}

	if errs := config.ValidateColumns(); len(errs) > 0 {
		for _, msg := range errs {
			auth.SetFlash(c.Request(), "error", msg)
			break
		}
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	colJSON, err := config.ColumnsToJSON()
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to save configuration")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	if err := database.UpdateAddressConfig(config.ID, colJSON); err != nil {
		slog.Error("error updating address config", slog.String("error", err.Error()), slog.Int("configID", config.ID))
		auth.SetFlash(c.Request(), "error", "Failed to save configuration")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	auth.SetFlash(c.Request(), "success", "Column configuration updated successfully")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
}

// DownloadAddressImportTemplate generates a CSV template matching the current column config.
func DownloadAddressImportTemplate(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	tab := c.QueryParam("tab")
	if tab == "" {
		tab = "bill_to"
	}
	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to load column config"})
	}

	var headers []string
	for _, col := range models.FixedColumnsForType(tab) {
		headers = append(headers, col.Name)
	}
	for _, col := range config.DynamicColumns() {
		headers = append(headers, col.Name)
	}

	filename := tab + "_address_template.csv"
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename="+filename)

	writer := csv.NewWriter(c.Response().Writer)
	_ = writer.Write(headers)
	writer.Flush()
	return nil
}

// UploadAddressesHandler handles CSV/Excel file upload for either address type.
func UploadAddressesHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	tab := validAddressTab(func() string {
		if v := c.QueryParam("tab"); v != "" {
			return v
		}
		return "bill_to"
	}())

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load config")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Please select a file to upload")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		auth.SetFlash(c.Request(), "error", "File size must be less than 10MB")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	mode := c.FormValue("mode")
	if mode != "append" {
		mode = "replace"
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))

	// config.ColumnDefinitions already includes fixed columns (via ensureFixedColumns)
	allColumns := config.ColumnDefinitions

	var rows []map[string]string
	var parseErr error

	switch ext {
	case ".csv":
		rows, parseErr = parseCSVFile(file, allColumns)
	case ".xlsx", ".xls":
		rows, parseErr = parseExcelFile(file, header, allColumns)
	default:
		auth.SetFlash(c.Request(), "error", "Only CSV and Excel (.xlsx) files are supported")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	if parseErr != nil {
		auth.SetFlash(c.Request(), "error", parseErr.Error())
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	if len(rows) == 0 {
		auth.SetFlash(c.Request(), "error", "File contains no data rows")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	if len(rows) > 100000 {
		auth.SetFlash(c.Request(), "error", "Maximum 100,000 rows per upload")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	// Validate rows
	result := &models.UploadResult{TotalRows: len(rows)}
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
				slog.Error("error deleting addresses", slog.String("error", err.Error()), slog.Int("configID", config.ID))
				auth.SetFlash(c.Request(), "error", "Failed to replace addresses")
				return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
			}
		}
		if err := database.BulkInsertAddresses(config.ID, validAddresses); err != nil {
			slog.Error("error inserting addresses", slog.String("error", err.Error()), slog.Int("configID", config.ID))
			auth.SetFlash(c.Request(), "error", "Failed to save addresses")
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		}
	}

	msg := fmt.Sprintf("Upload complete: %d of %d addresses imported (%s mode)", result.Successful, result.TotalRows, mode)
	if result.Failed > 0 {
		msg += fmt.Sprintf(". %d rows failed validation.", result.Failed)
	}
	auth.SetFlash(c.Request(), "success", msg)
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
}

// CreateAddressUnified handles adding a single address for either type.
func CreateAddressUnified(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	tab := validAddressTab(func() string {
		if v := c.QueryParam("tab"); v != "" {
			return v
		}
		return "bill_to"
	}())

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load config")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	// Collect dynamic column data (fixed columns are collected separately below)
	dynamicCols := config.DynamicColumns()
	data := make(map[string]string)
	for _, col := range dynamicCols {
		data[col.Name] = strings.TrimSpace(c.FormValue("field_" + sanitizeFieldName(col.Name)))
	}

	// Collect address code
	addressCode := strings.TrimSpace(c.FormValue("address_code"))

	// Collect fixed fields for ship-to
	var districtName, mandalName, mandalCode string
	if tab == "ship_to" {
		districtName = strings.TrimSpace(c.FormValue("district_name"))
		mandalName = strings.TrimSpace(c.FormValue("mandal_name"))
		mandalCode = strings.TrimSpace(c.FormValue("mandal_code"))

		// Validate fixed fields
		if districtName == "" || mandalName == "" || mandalCode == "" {
			auth.SetFlash(c.Request(), "error", "District Name, Mandal/ULB Name, and Mandal Code are required")
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		}
	}

	errs := database.ValidateAddressData(data, dynamicCols)
	if len(errs) > 0 {
		auth.SetFlash(c.Request(), "error", strings.Join(errs, "; "))
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	// Validate address_code uniqueness
	if addressCode != "" {
		unique, err := database.CheckAddressCodeUnique(addressCode, 0)
		if err != nil {
			slog.Warn("error checking address code uniqueness", slog.String("error", err.Error()))
		} else if !unique {
			auth.SetFlash(c.Request(), "error", "This address code is already in use")
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		}
	}

	if _, err := database.CreateAddress(config.ID, data, districtName, mandalName, mandalCode, addressCode); err != nil {
		slog.Error("error creating address", slog.String("error", err.Error()), slog.Int("projectID", projectID), slog.String("tab", tab))
		auth.SetFlash(c.Request(), "error", "Failed to create address")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	auth.SetFlash(c.Request(), "success", "Address added successfully")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
}

// UpdateAddressUnified handles editing a single address.
func UpdateAddressUnified(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	addressID, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid address ID"})
	}

	tab := validAddressTab(func() string {
		if v := c.QueryParam("tab"); v != "" {
			return v
		}
		return "bill_to"
	}())

	// Check if address is used in an issued DC before allowing edit
	usedInIssued, checkErr := database.IsAddressUsedInIssuedDC(addressID)
	if checkErr != nil {
		slog.Error("error checking issued DC usage", slog.String("error", checkErr.Error()), slog.Int("addressID", addressID))
	}
	if usedInIssued {
		auth.SetFlash(c.Request(), "error", "Cannot edit this address because it is used in an issued delivery challan")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load config")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	dynamicCols := config.DynamicColumns()
	data := make(map[string]string)
	for _, col := range dynamicCols {
		data[col.Name] = strings.TrimSpace(c.FormValue("field_" + sanitizeFieldName(col.Name)))
	}

	// Collect address code
	addressCode := strings.TrimSpace(c.FormValue("address_code"))

	var districtName, mandalName, mandalCode string
	if tab == "ship_to" {
		districtName = strings.TrimSpace(c.FormValue("district_name"))
		mandalName = strings.TrimSpace(c.FormValue("mandal_name"))
		mandalCode = strings.TrimSpace(c.FormValue("mandal_code"))

		if districtName == "" || mandalName == "" || mandalCode == "" {
			auth.SetFlash(c.Request(), "error", "District Name, Mandal/ULB Name, and Mandal Code are required")
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		}
	}

	errs := database.ValidateAddressData(data, dynamicCols)
	if len(errs) > 0 {
		auth.SetFlash(c.Request(), "error", strings.Join(errs, "; "))
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	// Validate address_code uniqueness excluding current address
	if addressCode != "" {
		unique, err := database.CheckAddressCodeUnique(addressCode, addressID)
		if err != nil {
			slog.Warn("error checking address code uniqueness", slog.String("error", err.Error()))
		} else if !unique {
			auth.SetFlash(c.Request(), "error", "This address code is already in use")
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
		}
	}

	if err := database.UpdateAddress(addressID, data, districtName, mandalName, mandalCode, addressCode); err != nil {
		slog.Error("error updating address", slog.String("error", err.Error()), slog.Int("addressID", addressID), slog.Int("projectID", projectID))
		auth.SetFlash(c.Request(), "error", "Failed to update address")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
	}

	auth.SetFlash(c.Request(), "success", "Address updated successfully")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab))
}

// DeleteAddressUnified handles DELETE for a single address.
func DeleteAddressUnified(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	addressID, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid address ID"})
	}

	tab := validAddressTab(func() string {
		if v := c.QueryParam("tab"); v != "" {
			return v
		}
		return "bill_to"
	}())

	// Check if address is used in an issued DC before allowing delete
	usedInIssued, checkErr := database.IsAddressUsedInIssuedDC(addressID)
	if checkErr != nil {
		slog.Error("delete address: error checking issued DC usage", slog.String("error", checkErr.Error()), slog.Int("addressID", addressID))
	}
	if usedInIssued {
		return c.JSON(http.StatusConflict, map[string]interface{}{"error": "Cannot delete this address because it is used in an issued delivery challan"})
	}

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		slog.Error("delete address: failed to load config", slog.String("error", err.Error()), slog.Int("projectID", projectID), slog.String("tab", tab))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("Failed to load config: %v", err)})
	}

	if err := database.DeleteAddress(addressID, config.ID); err != nil {
		slog.Error("delete address: failed to delete", slog.String("error", err.Error()), slog.Int("addressID", addressID), slog.Int("configID", config.ID), slog.Int("projectID", projectID), slog.String("tab", tab))
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return c.JSON(http.StatusConflict, map[string]interface{}{"error": "Cannot delete this address because it is used in one or more delivery challans"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("Failed to delete address: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"success": true})
}

// DeleteAllAddressesUnified handles DELETE for all addresses.
func DeleteAllAddressesUnified(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	tab := validAddressTab(func() string {
		if v := c.QueryParam("tab"); v != "" {
			return v
		}
		return "bill_to"
	}())

	config, err := database.GetOrCreateAddressConfig(projectID, tab)
	if err != nil {
		slog.Error("delete all addresses: failed to load config", slog.String("error", err.Error()), slog.Int("projectID", projectID), slog.String("tab", tab))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("Failed to load config: %v", err)})
	}

	// Check if any address in this config is used in an issued DC
	allAddresses, listErr := database.ListAddresses(config.ID, 1, 100000, "")
	if listErr != nil {
		slog.Error("delete all addresses: failed to list addresses", slog.String("error", listErr.Error()), slog.Int("configID", config.ID))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to check address usage"})
	}
	for _, addr := range allAddresses.Addresses {
		usedInIssued, checkErr := database.IsAddressUsedInIssuedDC(addr.ID)
		if checkErr != nil {
			slog.Error("delete all addresses: error checking issued DC usage", slog.String("error", checkErr.Error()), slog.Int("addressID", addr.ID))
			continue
		}
		if usedInIssued {
			return c.JSON(http.StatusConflict, map[string]interface{}{"error": "Cannot delete all addresses because some are used in issued delivery challans"})
		}
	}

	if err := database.DeleteAllAddresses(config.ID); err != nil {
		slog.Error("delete all addresses: failed to delete", slog.String("error", err.Error()), slog.Int("configID", config.ID), slog.Int("projectID", projectID), slog.String("tab", tab))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("Failed to delete addresses: %v", err)})
	}

	auth.SetFlash(c.Request(), "success", "All addresses deleted")
	return c.JSON(http.StatusOK, map[string]interface{}{"success": true, "redirect": fmt.Sprintf("/projects/%d/addresses?tab=%s", projectID, tab)})
}

// GetAddressJSONUnified returns a single address as JSON (for edit form).
func GetAddressJSONUnified(c echo.Context) error {
	addressID, err := strconv.Atoi(c.Param("aid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid address ID"})
	}

	addr, err := database.GetAddress(addressID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Address not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":            addr.ID,
		"address_code":  addr.AddressCode,
		"data":          addr.Data,
		"district_name": addr.DistrictName,
		"mandal_name":   addr.MandalName,
		"mandal_code":   addr.MandalCode,
	})
}

// SearchAddressSelector handles the HTMX-powered address selector search.
func SearchAddressSelector(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	addressType := validAddressTab(func() string {
		if v := c.QueryParam("type"); v != "" {
			return v
		}
		return "ship_to"
	}())

	search := c.QueryParam("q")

	config, err := database.GetOrCreateAddressConfig(projectID, addressType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to load config"})
	}

	addresses, err := database.SearchAddressesForSelector(config.ID, search, addressType, 20)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to search addresses"})
	}

	return components.RenderOK(c, htmx.AddressSelectorResults(htmx.AddressSelectorResultsProps{
		Addresses:   addresses,
		AddressType: addressType,
		Columns:     config.ColumnDefinitions,
	}))
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

		// Skip entirely empty rows
		allEmpty := true
		for _, v := range row {
			if v != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// parseExcelFile parses an Excel file and maps headers to column definitions.
func parseExcelFile(file io.Reader, _ interface{}, columns []models.ColumnDefinition) ([]map[string]string, error) {
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

		// Skip entirely empty rows
		allEmpty := true
		for _, v := range row {
			if v != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
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
