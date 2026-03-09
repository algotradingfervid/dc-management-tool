package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	pagetransfer "github.com/narendhupati/dc-management-tool/components/pages/transfer_dcs"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// ─── Transfer wizard form parsers ─────────────────────────────────────────────

// parseTransferStep1Form parses Step 1 form values carried as hidden fields.
func parseTransferStep1Form(c echo.Context) (templateID int, challanDate string, hubAddressID int, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge string) {
	templateID, _ = strconv.Atoi(c.FormValue("template_id"))
	challanDate = c.FormValue("challan_date")
	hubAddressID, _ = strconv.Atoi(c.FormValue("hub_address_id"))
	transporterName = c.FormValue("transporter_name")
	vehicleNumber = c.FormValue("vehicle_number")
	ewayBillNumber = c.FormValue("eway_bill_number")
	docketNumber = c.FormValue("docket_number")
	taxType = c.FormValue("tax_type")
	reverseCharge = c.FormValue("reverse_charge")
	return
}

// parseTransferStep2Form parses Step 2 address selections.
func parseTransferStep2Form(c echo.Context) (billFromAddrID, dispatchFromAddrID, billToAddrID int, shipToAddressIDs []int) {
	billFromAddrID, _ = strconv.Atoi(c.FormValue("bill_from_address_id"))
	dispatchFromAddrID, _ = strconv.Atoi(c.FormValue("dispatch_from_address_id"))
	billToAddrID, _ = strconv.Atoi(c.FormValue("bill_to_address_id"))
	if parseErr := c.Request().ParseMultipartForm(32 << 20); parseErr != nil {
		_ = c.Request().ParseForm()
	}
	for _, s := range c.Request().PostForm["ship_to_address_ids"] {
		id, idErr := strconv.Atoi(s)
		if idErr == nil && id > 0 {
			shipToAddressIDs = append(shipToAddressIDs, id)
		}
	}
	return
}

// parseTransferSerialForm parses bulk serial numbers per product (NO per-destination assignments).
func parseTransferSerialForm(c echo.Context, products []*models.TemplateProductRow) []pagetransfer.TransferSerialData {
	var result []pagetransfer.TransferSerialData
	for _, p := range products {
		sd := pagetransfer.TransferSerialData{ProductID: p.ID}
		serialsRaw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		if serialsRaw != "" {
			for _, sn := range strings.Split(serialsRaw, "\n") {
				sn = strings.TrimSpace(sn)
				if sn != "" {
					sd.AllSerials = append(sd.AllSerials, sn)
				}
			}
		}
		result = append(result, sd)
	}
	return result
}

// ─── Transfer wizard step handlers ────────────────────────────────────────────

// ShowCreateTransferWizard renders step 1 of the transfer DC wizard.
func ShowCreateTransferWizard(c echo.Context) error {
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

	templates, err := database.GetTemplatesByProjectID(projectID)
	if err != nil {
		slog.Error("Error fetching templates", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		templates = []*models.DCTemplate{}
	}

	transporters, err := database.GetTransportersByProjectID(projectID, true)
	if err != nil {
		slog.Error("Error fetching transporters", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		transporters = []*models.Transporter{}
	}

	// Load ship-to addresses for hub location dropdown
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var hubAddresses []*models.Address
	if shipToConfig != nil {
		hubAddresses, _ = database.GetAllAddressesByConfigID(shipToConfig.ID)
	}

	flashType, flashMessage := auth.PopFlash(c.Request())
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.WizardStep1(
		user,
		project,
		allProjects,
		templates,
		transporters,
		hubAddresses,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
		nil,
		0,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, flashMessage, flashType, pageContent))
}

// TransferWizardStep2 processes step 1 data and renders step 2 (address selection).
func TransferWizardStep2(c echo.Context) error {
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

	// Parse step 1 data
	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)

	// Validate
	validationErrors := make(map[string]string)
	if templateID == 0 {
		validationErrors["template_id"] = "Template is required"
	}
	if challanDate == "" {
		validationErrors["challan_date"] = "Challan date is required"
	}
	if hubAddressID == 0 {
		validationErrors["hub_address_id"] = "Hub / Transit location is required"
	}
	if taxType != "cgst_sgst" && taxType != "igst" {
		validationErrors["tax_type"] = "Tax type is required"
	}
	if reverseCharge == "" {
		reverseCharge = "N"
	}

	if len(validationErrors) > 0 {
		auth.SetFlash(c.Request(), "error", "Please fix the errors below")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	// Validate template belongs to project
	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Template not found or doesn't belong to this project")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	// Peek at STDC number
	transferDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransfer)

	// Load all 4 address types
	billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses := loadAllAddresses(projectID)

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.WizardStep2(
		user,
		project,
		allProjects,
		tmpl,
		challanDate,
		transferDCNumber,
		strconv.Itoa(templateID),
		strconv.Itoa(hubAddressID),
		transporterName,
		vehicleNumber,
		ewayBillNumber,
		docketNumber,
		taxType,
		reverseCharge,
		billFromAddresses,
		dispatchFromAddresses,
		billToAddresses,
		shipToAddresses,
		csrf.Token(c.Request()),
		nil,
		0, 0, 0,
		0,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "", "", pageContent))
}

// TransferWizardQuantityStep processes step 2 (addresses) and renders step 3 (quantity grid).
func TransferWizardQuantityStep(c echo.Context) error {
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

	// Parse step 1 and step 2 data
	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	// Validate
	validationErrors := make(map[string]string)
	if len(shipToAddressIDs) == 0 {
		validationErrors["ship_to"] = "Please select at least one ship-to address"
	}

	if len(validationErrors) > 0 {
		// Build combined error message
		var msgs []string
		for _, msg := range validationErrors {
			msgs = append(msgs, msg)
		}
		auth.SetFlash(c.Request(), "error", strings.Join(msgs, ". "))

		// Re-render step 2 with error
		tmpl, _ := database.GetTemplateByID(templateID)
		transferDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransfer)
		billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses := loadAllAddresses(projectID)
		flashType, flashMessage := auth.PopFlash(c.Request())
		allProjects, _ := database.GetAccessibleProjects(user)

		pageContent := pagetransfer.WizardStep2(
			user, project, allProjects, tmpl,
			challanDate, transferDCNumber,
			strconv.Itoa(templateID), strconv.Itoa(hubAddressID),
			transporterName, vehicleNumber, ewayBillNumber, docketNumber,
			taxType, reverseCharge,
			billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses,
			csrf.Token(c.Request()),
			shipToAddressIDs,
			billFromAddrID, dispatchFromAddrID, billToAddrID,
			0,
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
		return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, flashMessage, flashType, pageContent))
	}

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	// Build QuantityAddress list from selected ship-to IDs
	quantityAddresses := loadTransferQuantityAddresses(projectID, shipToAddressIDs)

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.WizardStep3(
		user,
		project,
		allProjects,
		products,
		quantityAddresses,
		templateID,
		challanDate,
		hubAddressID,
		transporterName,
		vehicleNumber,
		ewayBillNumber,
		docketNumber,
		taxType,
		reverseCharge,
		billFromAddrID,
		dispatchFromAddrID,
		billToAddrID,
		shipToAddressIDs,
		csrf.Token(c.Request()),
		"", "",
		nil, nil,
		0,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "", "", pageContent))
}

// TransferWizardStep4 processes step 3 (quantities) and renders step 4 (serial entry).
func TransferWizardStep4(c echo.Context) error {
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

	// Parse all carry-forward data
	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	// Parse quantities from step 3
	quantities := parseQuantityForm(c, products, shipToAddressIDs)

	// Validate quantities
	qtyErrors := validateQuantities(quantities, products, shipToAddressIDs)
	if len(qtyErrors) > 0 {
		globalErr := qtyErrors["global"]
		if globalErr == "" {
			globalErr = "Please fix the quantity errors below"
		}

		// Re-render quantity grid with errors
		quantityAddresses := loadTransferQuantityAddresses(projectID, shipToAddressIDs)
		allProjects, _ := database.GetAccessibleProjects(user)

		pageContent := pagetransfer.WizardStep3(
			user, project, allProjects, products, quantityAddresses,
			templateID, challanDate,
			hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber,
			taxType, reverseCharge,
			billFromAddrID, dispatchFromAddrID, billToAddrID,
			shipToAddressIDs, csrf.Token(c.Request()),
			globalErr, "error",
			quantities, qtyErrors,
			0,
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", globalErr)
		return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, globalErr, "error", pageContent))
	}

	// Build quantity hidden field values for carry-forward
	var quantityHiddenFields []pagetransfer.QuantityHiddenField
	for _, p := range products {
		for _, addrID := range shipToAddressIDs {
			qty := 0
			if qMap, ok := quantities[p.ID]; ok {
				qty = qMap[addrID]
			}
			quantityHiddenFields = append(quantityHiddenFields, pagetransfer.QuantityHiddenField{
				Name:  fmt.Sprintf("qty_%d_%d", p.ID, addrID),
				Value: strconv.Itoa(qty),
			})
		}
	}

	// Build ship-to IDs string for hidden field
	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.WizardStep4(
		user,
		project,
		allProjects,
		products,
		challanDate,
		strconv.Itoa(templateID),
		strconv.Itoa(hubAddressID),
		transporterName,
		vehicleNumber,
		ewayBillNumber,
		docketNumber,
		taxType,
		reverseCharge,
		strconv.Itoa(billFromAddrID),
		strconv.Itoa(dispatchFromAddrID),
		strconv.Itoa(billToAddrID),
		shipToIDStrings,
		csrf.Token(c.Request()),
		quantityHiddenFields,
		computeProductQuantityTotals(quantities),
		nil, nil,
		0,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "", "", pageContent))
}

// TransferWizardStep5 processes step 4 (serials) and renders step 5 (review page).
func TransferWizardStep5(c echo.Context) error {
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

	// Parse all carry-forward data
	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	// Parse serial data
	serialData := parseTransferSerialForm(c, products)

	// Parse quantities for validation
	quantities := parseQuantityForm(c, products, shipToAddressIDs)

	// Validate serial counts and collect per-product errors
	serialErrors := make(map[int]string)
	for i, sd := range serialData {
		// Sum quantities across all destinations for this product
		expectedTotal := 0
		for _, addrID := range shipToAddressIDs {
			expectedTotal += quantities[products[i].ID][addrID]
		}
		if len(sd.AllSerials) > 0 && len(sd.AllSerials) != expectedTotal {
			serialErrors[sd.ProductID] = fmt.Sprintf("Expected %d serials, got %d", expectedTotal, len(sd.AllSerials))
			continue
		}

		// Check for duplicates within same product
		seen := make(map[string]bool)
		for _, sn := range sd.AllSerials {
			if seen[sn] {
				serialErrors[sd.ProductID] = fmt.Sprintf("Duplicate serial within this product: %s", sn)
				break
			}
			seen[sn] = true
		}

		// Project-wide duplicate check
		if _, alreadyHasError := serialErrors[sd.ProductID]; !alreadyHasError && len(sd.AllSerials) > 0 {
			conflicts, conflictsErr := database.CheckSerialsInProject(projectID, sd.AllSerials, nil)
			if conflictsErr != nil {
				slog.Error("Error checking serials", slog.String("error", conflictsErr.Error()), slog.Int("projectID", projectID))
			}
			if len(conflicts) > 0 {
				serialErrors[sd.ProductID] = fmt.Sprintf("Serial %s already exists in DC %s", conflicts[0].SerialNumber, conflicts[0].DCNumber)
			}
		}
	}

	// Build quantity hidden fields from carry-forward data
	var quantityHiddenFields []pagetransfer.QuantityHiddenField
	for _, p := range products {
		for _, addrID := range shipToAddressIDs {
			qty := 0
			if qMap, ok := quantities[p.ID]; ok {
				qty = qMap[addrID]
			}
			quantityHiddenFields = append(quantityHiddenFields, pagetransfer.QuantityHiddenField{
				Name:  fmt.Sprintf("qty_%d_%d", p.ID, addrID),
				Value: strconv.Itoa(qty),
			})
		}
	}

	// If any errors, re-render step 4 (serials) with pre-filled data and inline errors
	if len(serialErrors) > 0 {
		prefillSerials := make(map[int][]string)
		for _, sd := range serialData {
			prefillSerials[sd.ProductID] = sd.AllSerials
		}

		shipToIDStrings := make([]string, len(shipToAddressIDs))
		for i, id := range shipToAddressIDs {
			shipToIDStrings[i] = strconv.Itoa(id)
		}

		allProjects, _ := database.GetAccessibleProjects(user)

		pageContent := pagetransfer.WizardStep4(
			user, project, allProjects, products, challanDate,
			strconv.Itoa(templateID), strconv.Itoa(hubAddressID),
			transporterName, vehicleNumber, ewayBillNumber, docketNumber,
			taxType, reverseCharge,
			strconv.Itoa(billFromAddrID), strconv.Itoa(dispatchFromAddrID),
			strconv.Itoa(billToAddrID),
			shipToIDStrings,
			csrf.Token(c.Request()),
			quantityHiddenFields,
			computeProductQuantityTotals(quantities),
			prefillSerials, serialErrors,
			0,
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", "Please fix the serial number errors below")
		return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "Please fix the serial number errors below", "error", pageContent))
	}

	// Load template for display
	tmpl, _ := database.GetTemplateByID(templateID)

	// Load ship-to addresses for review display
	shipToAddresses := loadShipToAddresses(projectID, shipToAddressIDs)

	// Build serialized ship-to IDs for hidden fields
	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}

	// Load hub address name
	hubAddressName := ""
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	if shipToConfig != nil {
		allShipTo, _ := database.GetAllAddressesByConfigID(shipToConfig.ID)
		for _, a := range allShipTo {
			if a.ID == hubAddressID {
				hubAddressName = a.DisplayName()
				break
			}
		}
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.WizardStep5(
		user,
		project,
		allProjects,
		tmpl,
		products,
		challanDate,
		hubAddressName,
		transporterName,
		vehicleNumber,
		ewayBillNumber,
		docketNumber,
		taxType,
		reverseCharge,
		strconv.Itoa(templateID),
		strconv.Itoa(hubAddressID),
		strconv.Itoa(billFromAddrID),
		strconv.Itoa(dispatchFromAddrID),
		strconv.Itoa(billToAddrID),
		shipToIDStrings,
		shipToAddresses,
		serialData,
		csrf.Token(c.Request()),
		quantityHiddenFields,
		computeProductQuantityTotals(quantities),
		0,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "", "", pageContent))
}

// CreateTransferDC processes the final submission and creates the Transfer DC.
func CreateTransferDC(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	// Re-parse ALL data from hidden fields
	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddressID, dispatchFromAddressID, billToAddressID, shipToAddressIDs := parseTransferStep2Form(c)

	// Re-validate
	if templateID == 0 || challanDate == "" || hubAddressID == 0 {
		auth.SetFlash(c.Request(), "error", "Invalid transfer DC data")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}
	if len(shipToAddressIDs) == 0 {
		auth.SetFlash(c.Request(), "error", "At least one ship-to address is required")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	// Validate template belongs to project
	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Template not found or doesn't belong to this project")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	// Parse quantities
	quantities := parseQuantityForm(c, products, shipToAddressIDs)

	// Build line items
	var lineItems []services.TransferDCLineItem
	for _, p := range products {
		qtyByDest := make(map[int]int)
		for _, addrID := range shipToAddressIDs {
			qtyByDest[addrID] = quantities[p.ID][addrID]
		}
		item := services.TransferDCLineItem{
			ProductID:        p.ID,
			QtyByDestination: qtyByDest,
			Rate:             p.PerUnitPrice,
			TaxPercentage:    p.GSTPercentage,
		}

		// Parse serials
		serialsRaw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		if serialsRaw != "" {
			for _, sn := range strings.Split(serialsRaw, "\n") {
				sn = strings.TrimSpace(sn)
				if sn != "" {
					item.AllSerials = append(item.AllSerials, sn)
				}
			}
		}

		lineItems = append(lineItems, item)
	}

	// Validate serial counts
	for i, item := range lineItems {
		expectedTotal := 0
		for _, addrID := range shipToAddressIDs {
			expectedTotal += quantities[products[i].ID][addrID]
		}
		if len(item.AllSerials) > 0 && len(item.AllSerials) != expectedTotal {
			auth.SetFlash(c.Request(), "error", fmt.Sprintf("Product %s: expected %d serials, got %d", products[i].ItemName, expectedTotal, len(item.AllSerials)))
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
		}

		// Check duplicates within same product
		seen := make(map[string]bool)
		for _, sn := range item.AllSerials {
			if seen[sn] {
				auth.SetFlash(c.Request(), "error", fmt.Sprintf("Duplicate serial %s for product %s", sn, products[i].ItemName))
				return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
			}
			seen[sn] = true
		}
	}

	// Check for duplicate serials in project
	for _, item := range lineItems {
		if len(item.AllSerials) > 0 {
			conflicts, conflictsErr := database.CheckSerialsInProject(projectID, item.AllSerials, nil)
			if conflictsErr != nil {
				slog.Error("Error checking serials", slog.String("error", conflictsErr.Error()), slog.Int("projectID", projectID))
			}
			if len(conflicts) > 0 {
				auth.SetFlash(c.Request(), "error", fmt.Sprintf("Serial %s already exists in DC %s", conflicts[0].SerialNumber, conflicts[0].DCNumber))
				return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
			}
		}
	}

	params := services.TransferDCParams{
		ProjectID:             projectID,
		TemplateID:            templateID,
		HubAddressID:          hubAddressID,
		ChallanDate:           challanDate,
		TaxType:               taxType,
		ReverseCharge:         reverseCharge,
		TransporterName:       transporterName,
		VehicleNumber:         vehicleNumber,
		EwayBillNumber:        ewayBillNumber,
		DocketNumber:          docketNumber,
		BillFromAddressID:     billFromAddressID,
		DispatchFromAddressID: dispatchFromAddressID,
		BillToAddressID:       billToAddressID,
		ShipToAddressIDs:      shipToAddressIDs,
		LineItems:             lineItems,
		CreatedBy:             user.ID,
	}

	_, err = services.CreateTransferDC(database.DB, params)
	if err != nil {
		slog.Error("Error creating transfer DC", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		auth.SetFlash(c.Request(), "error", fmt.Sprintf("Failed to create transfer DC: %v", err))
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/new", projectID))
	}

	auth.SetFlash(c.Request(), "success", "Transfer DC created successfully")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dashboard", projectID))
}

// ─── Transfer wizard back navigation handlers ─────────────────────────────────

// TransferBackToStep1 re-renders Step 1 with carry-forward data from a later step.
func TransferBackToStep1(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)

	templates, _ := database.GetTemplatesByProjectID(projectID)
	transporters, _ := database.GetTransportersByProjectID(projectID, true)

	// Look up transporter ID by name
	transporterID := 0
	for _, t := range transporters {
		if t.CompanyName == transporterName {
			transporterID = t.ID
			break
		}
	}

	// Load hub addresses
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var hubAddresses []*models.Address
	if shipToConfig != nil {
		hubAddresses, _ = database.GetAllAddressesByConfigID(shipToConfig.ID)
	}

	tid := templateID
	prefill := &pagetransfer.TransferStep1Prefill{
		TemplateID:      &tid,
		ChallanDate:     challanDate,
		HubAddressID:    hubAddressID,
		TransporterID:   transporterID,
		TransporterName: transporterName,
		VehicleNumber:   vehicleNumber,
		EwayBillNumber:  ewayBillNumber,
		DocketNumber:    docketNumber,
		TaxType:         taxType,
		ReverseCharge:   reverseCharge,
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pagetransfer.WizardStep1(user, project, allProjects, templates, transporters, hubAddresses, "", "", csrf.Token(c.Request()), prefill, 0)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "", "", pageContent))
}

// TransferBackToStep2 re-renders Step 2 with carry-forward data.
func TransferBackToStep2(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	tmpl, _ := database.GetTemplateByID(templateID)
	transferDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransfer)

	billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses := loadAllAddresses(projectID)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pagetransfer.WizardStep2(
		user, project, allProjects, tmpl,
		challanDate, transferDCNumber,
		strconv.Itoa(templateID), strconv.Itoa(hubAddressID),
		transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses,
		csrf.Token(c.Request()),
		shipToAddressIDs,
		billFromAddrID, dispatchFromAddrID, billToAddrID,
		0,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "", "", pageContent))
}

// TransferBackToStep3 re-renders Step 3 (quantity grid) with carry-forward data.
func TransferBackToStep3(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	products, _ := database.GetTemplateProducts(templateID)
	quantities := parseQuantityForm(c, products, shipToAddressIDs)
	quantityAddresses := loadTransferQuantityAddresses(projectID, shipToAddressIDs)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pagetransfer.WizardStep3(
		user, project, allProjects, products, quantityAddresses,
		templateID, challanDate,
		hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		billFromAddrID, dispatchFromAddrID, billToAddrID,
		shipToAddressIDs, csrf.Token(c.Request()),
		"", "",
		quantities, nil,
		0,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "", "", pageContent))
}

// TransferBackToStep4 re-renders Step 4 (serials) with carry-forward data.
func TransferBackToStep4(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	products, _ := database.GetTemplateProducts(templateID)
	quantities := parseQuantityForm(c, products, shipToAddressIDs)
	serialData := parseTransferSerialForm(c, products)

	// Build prefill map from serial data
	prefillSerials := make(map[int][]string)
	for _, sd := range serialData {
		prefillSerials[sd.ProductID] = sd.AllSerials
	}

	// Build quantity hidden fields
	var quantityHiddenFields []pagetransfer.QuantityHiddenField
	for _, p := range products {
		for _, addrID := range shipToAddressIDs {
			qty := 0
			if qMap, ok := quantities[p.ID]; ok {
				qty = qMap[addrID]
			}
			quantityHiddenFields = append(quantityHiddenFields, pagetransfer.QuantityHiddenField{
				Name:  fmt.Sprintf("qty_%d_%d", p.ID, addrID),
				Value: strconv.Itoa(qty),
			})
		}
	}

	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pagetransfer.WizardStep4(
		user, project, allProjects, products, challanDate,
		strconv.Itoa(templateID), strconv.Itoa(hubAddressID),
		transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		strconv.Itoa(billFromAddrID), strconv.Itoa(dispatchFromAddrID),
		strconv.Itoa(billToAddrID),
		shipToIDStrings,
		csrf.Token(c.Request()),
		quantityHiddenFields,
		computeProductQuantityTotals(quantities),
		prefillSerials, nil,
		0,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC Wizard", sidebar, topbar, "", "", pageContent))
}

// ─── Transfer wizard shared helpers ───────────────────────────────────────────

// loadTransferQuantityAddresses loads QuantityAddress structs (transfer_dcs package type)
// for the selected ship-to IDs.
func loadTransferQuantityAddresses(projectID int, shipToAddressIDs []int) []pagetransfer.QuantityAddress {
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var result []pagetransfer.QuantityAddress
	if shipToConfig != nil {
		allShipTo, _ := database.GetAllAddressesByConfigID(shipToConfig.ID)
		selectedSet := make(map[int]bool)
		for _, id := range shipToAddressIDs {
			selectedSet[id] = true
		}
		for _, a := range allShipTo {
			if selectedSet[a.ID] {
				result = append(result, pagetransfer.QuantityAddress{
					ID:   a.ID,
					Name: a.DisplayName(),
				})
			}
		}
	}
	return result
}
