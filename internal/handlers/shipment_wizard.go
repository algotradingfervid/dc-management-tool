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
	pageshipments "github.com/narendhupati/dc-management-tool/components/pages/shipments"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// ShowCreateShipmentWizard renders step 1 of the shipment wizard.
func ShowCreateShipmentWizard(c echo.Context) error {
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

	flashType, flashMessage := auth.PopFlash(c.Request())
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.WizardStep1(
		user,
		project,
		allProjects,
		templates,
		transporters,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, flashMessage, flashType, pageContent))
}

// ShipmentWizardStep2 processes step 1 data and renders step 2 (address selection).
func ShipmentWizardStep2(c echo.Context) error {
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
	templateID, _ := strconv.Atoi(c.FormValue("template_id"))
	numSets, _ := strconv.Atoi(c.FormValue("num_sets"))
	challanDate := c.FormValue("challan_date")
	transporterName := c.FormValue("transporter_name")
	vehicleNumber := c.FormValue("vehicle_number")
	ewayBillNumber := c.FormValue("eway_bill_number")
	docketNumber := c.FormValue("docket_number")
	taxType := c.FormValue("tax_type")
	reverseCharge := c.FormValue("reverse_charge")

	// Validate
	errors := make(map[string]string)
	if templateID == 0 {
		errors["template_id"] = "Template is required"
	}
	if numSets < 1 {
		errors["num_sets"] = "Number of sets must be at least 1"
	}
	if challanDate == "" {
		errors["challan_date"] = "Challan date is required"
	}
	if taxType != "cgst_sgst" && taxType != "igst" {
		errors["tax_type"] = "Tax type is required"
	}
	if reverseCharge == "" {
		reverseCharge = "N"
	}

	if len(errors) > 0 {
		auth.SetFlash(c.Request(), "error", "Please fix the errors below")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}
	_ = products

	// Validate template belongs to project
	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Template not found or doesn't belong to this project")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	// Peek at DC numbers
	transitDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransit)
	officialDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)

	// Load all 4 address types
	billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
	dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")
	billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")

	var billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses []*models.Address
	if billFromConfig != nil {
		billFromAddresses, _ = database.GetAllAddressesByConfigID(billFromConfig.ID)
	}
	if dispatchFromConfig != nil {
		dispatchFromAddresses, _ = database.GetAllAddressesByConfigID(dispatchFromConfig.ID)
	}
	if billToConfig != nil {
		billToAddresses, _ = database.GetAllAddressesByConfigID(billToConfig.ID)
	}
	if shipToConfig != nil {
		shipToAddresses, _ = database.GetAllAddressesByConfigID(shipToConfig.ID)
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.WizardStep2(
		user,
		project,
		allProjects,
		tmpl,
		numSets,
		challanDate,
		transitDCNumber,
		officialDCNumber,
		strconv.Itoa(templateID),
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
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, "", "", pageContent))
}

// ShipmentWizardStep3 processes step 2 (addresses) and renders step 3 (serial entry).
func ShipmentWizardStep3(c echo.Context) error {
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

	// Parse step 1 carry-forward data
	templateID, _ := strconv.Atoi(c.FormValue("template_id"))
	numSets, _ := strconv.Atoi(c.FormValue("num_sets"))
	challanDate := c.FormValue("challan_date")
	transporterName := c.FormValue("transporter_name")
	vehicleNumber := c.FormValue("vehicle_number")
	ewayBillNumber := c.FormValue("eway_bill_number")
	docketNumber := c.FormValue("docket_number")
	taxType := c.FormValue("tax_type")
	reverseCharge := c.FormValue("reverse_charge")

	// Parse step 2 data (addresses)
	billFromAddressID, _ := strconv.Atoi(c.FormValue("bill_from_address_id"))
	dispatchFromAddressID, _ := strconv.Atoi(c.FormValue("dispatch_from_address_id"))
	billToAddressID, _ := strconv.Atoi(c.FormValue("bill_to_address_id"))
	transitShipToAddrID, _ := strconv.Atoi(c.FormValue("transit_ship_to_address_id"))

	// Parse multiple ship-to address selections
	if parseErr := c.Request().ParseMultipartForm(32 << 20); parseErr != nil {
		// Fall back to regular form parsing if multipart fails
		_ = c.Request().ParseForm()
	}
	shipToIDStrs := c.Request().PostForm["ship_to_address_ids"]
	var shipToAddressIDs []int
	for _, s := range shipToIDStrs {
		id, idErr := strconv.Atoi(s)
		if idErr == nil && id > 0 {
			shipToAddressIDs = append(shipToAddressIDs, id)
		}
	}

	// Validate
	validationErrors := make(map[string]string)
	if len(shipToAddressIDs) == 0 {
		validationErrors["ship_to"] = "Please select at least one ship-to address"
	} else if len(shipToAddressIDs) != numSets {
		validationErrors["ship_to"] = fmt.Sprintf("Please select exactly %d ship-to addresses to match the number of sets (got %d)", numSets, len(shipToAddressIDs))
	}

	// Validate transit ship-to is among selected
	if transitShipToAddrID > 0 {
		found := false
		for _, id := range shipToAddressIDs {
			if id == transitShipToAddrID {
				found = true
				break
			}
		}
		if !found {
			validationErrors["transit_ship_to"] = "Transit ship-to must be one of the selected ship-to addresses"
		}
	} else {
		validationErrors["transit_ship_to"] = "Transit ship-to address is required"
	}

	if len(validationErrors) > 0 {
		// Build combined error message
		var msgs []string
		for _, msg := range validationErrors {
			msgs = append(msgs, msg)
		}
		auth.SetFlash(c.Request(), "error", strings.Join(msgs, ". "))

		// Re-render step 2 instead of redirecting to step 1
		billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
		dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")
		billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")
		shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")

		var billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses []*models.Address
		if billFromConfig != nil {
			billFromAddresses, _ = database.GetAllAddressesByConfigID(billFromConfig.ID)
		}
		if dispatchFromConfig != nil {
			dispatchFromAddresses, _ = database.GetAllAddressesByConfigID(dispatchFromConfig.ID)
		}
		if billToConfig != nil {
			billToAddresses, _ = database.GetAllAddressesByConfigID(billToConfig.ID)
		}
		if shipToConfig != nil {
			shipToAddresses, _ = database.GetAllAddressesByConfigID(shipToConfig.ID)
		}

		tmpl, _ := database.GetTemplateByID(templateID)
		transitDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransit)
		officialDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)
		flashType, flashMessage := auth.PopFlash(c.Request())
		allProjects, _ := database.GetAccessibleProjects(user)
		_ = flashType
		_ = flashMessage

		pageContent := pageshipments.WizardStep2(
			user,
			project,
			allProjects,
			tmpl,
			numSets,
			challanDate,
			transitDCNumber,
			officialDCNumber,
			strconv.Itoa(templateID),
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
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "", "")
		return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, "", "", pageContent))
	}

	// Load template products with quantities
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	// Load ship-to address details for serial assignment UI
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var shipToAddresses []*models.Address
	if shipToConfig != nil {
		allShipTo, _ := database.GetAllAddressesByConfigID(shipToConfig.ID)
		// Filter to only selected ones
		selectedSet := make(map[int]bool)
		for _, id := range shipToAddressIDs {
			selectedSet[id] = true
		}
		for _, a := range allShipTo {
			if selectedSet[a.ID] {
				shipToAddresses = append(shipToAddresses, a)
			}
		}
	}

	// Build ship-to IDs string for hidden field
	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.WizardStep3(
		user,
		project,
		allProjects,
		products,
		numSets,
		challanDate,
		strconv.Itoa(templateID),
		transporterName,
		vehicleNumber,
		ewayBillNumber,
		docketNumber,
		taxType,
		reverseCharge,
		strconv.Itoa(billFromAddressID),
		strconv.Itoa(dispatchFromAddressID),
		strconv.Itoa(billToAddressID),
		strconv.Itoa(transitShipToAddrID),
		shipToIDStrings,
		shipToAddresses,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, "", "", pageContent))
}

// ShipmentWizardStep4 processes step 3 (serials) and renders review page.
func ShipmentWizardStep4(c echo.Context) error {
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
	templateID, _ := strconv.Atoi(c.FormValue("template_id"))
	numSets, _ := strconv.Atoi(c.FormValue("num_sets"))
	challanDate := c.FormValue("challan_date")
	transporterName := c.FormValue("transporter_name")
	vehicleNumber := c.FormValue("vehicle_number")
	ewayBillNumber := c.FormValue("eway_bill_number")
	docketNumber := c.FormValue("docket_number")
	taxType := c.FormValue("tax_type")
	reverseCharge := c.FormValue("reverse_charge")
	billFromAddressID, _ := strconv.Atoi(c.FormValue("bill_from_address_id"))
	dispatchFromAddressID, _ := strconv.Atoi(c.FormValue("dispatch_from_address_id"))
	billToAddressID, _ := strconv.Atoi(c.FormValue("bill_to_address_id"))
	transitShipToAddrID, _ := strconv.Atoi(c.FormValue("transit_ship_to_address_id"))

	if parseErr := c.Request().ParseMultipartForm(32 << 20); parseErr != nil {
		_ = c.Request().ParseForm()
	}
	shipToIDStrs := c.Request().PostForm["ship_to_address_ids"]
	var shipToAddressIDs []int
	for _, s := range shipToIDStrs {
		id, _ := strconv.Atoi(s)
		if id > 0 {
			shipToAddressIDs = append(shipToAddressIDs, id)
		}
	}

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	// Parse serial numbers per product
	var serialData []pageshipments.WizardSerialData

	for _, p := range products {
		pd := pageshipments.WizardSerialData{
			ProductID:   p.ID,
			Assignments: make(map[int][]string),
		}

		// All serials for this product
		serialsRaw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		if serialsRaw != "" {
			for _, sn := range strings.Split(serialsRaw, "\n") {
				sn = strings.TrimSpace(sn)
				if sn != "" {
					pd.AllSerials = append(pd.AllSerials, sn)
				}
			}
		}

		// Serial assignments per ship-to address
		for _, shipToID := range shipToAddressIDs {
			assignRaw := c.FormValue(fmt.Sprintf("assign_%d_%d", p.ID, shipToID))
			if assignRaw != "" {
				for _, sn := range strings.Split(assignRaw, "\n") {
					sn = strings.TrimSpace(sn)
					if sn != "" {
						pd.Assignments[shipToID] = append(pd.Assignments[shipToID], sn)
					}
				}
			}
		}

		serialData = append(serialData, pd)
	}

	// Validate serial counts
	validationErrors := make(map[string]string)
	for i, pd := range serialData {
		expectedTotal := products[i].DefaultQuantity * numSets
		if len(pd.AllSerials) > 0 && len(pd.AllSerials) != expectedTotal {
			validationErrors[fmt.Sprintf("serials_%d", pd.ProductID)] = fmt.Sprintf("Expected %d serials, got %d", expectedTotal, len(pd.AllSerials))
		}

		// Check for duplicates within same product
		seen := make(map[string]bool)
		for _, sn := range pd.AllSerials {
			if seen[sn] {
				validationErrors[fmt.Sprintf("serials_%d", pd.ProductID)] = fmt.Sprintf("Duplicate serial: %s", sn)
				break
			}
			seen[sn] = true
		}

		// Check assignment counts don't exceed qty_per_set
		for shipToID, assigned := range pd.Assignments {
			if len(assigned) > products[i].DefaultQuantity {
				validationErrors[fmt.Sprintf("assign_%d_%d", pd.ProductID, shipToID)] = fmt.Sprintf("Too many serials assigned (max %d)", products[i].DefaultQuantity)
			}
		}
	}

	if len(validationErrors) > 0 {
		auth.SetFlash(c.Request(), "error", "Serial number validation failed")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	// Load template for display
	tmpl, _ := database.GetTemplateByID(templateID)

	// Load address details for review display
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var shipToAddresses []*models.Address
	if shipToConfig != nil {
		allShipTo, _ := database.GetAllAddressesByConfigID(shipToConfig.ID)
		selectedSet := make(map[int]bool)
		for _, id := range shipToAddressIDs {
			selectedSet[id] = true
		}
		for _, a := range allShipTo {
			if selectedSet[a.ID] {
				shipToAddresses = append(shipToAddresses, a)
			}
		}
	}

	// Build serialized ship-to IDs for hidden fields
	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.WizardStep4(
		user,
		project,
		allProjects,
		tmpl,
		products,
		numSets,
		challanDate,
		transporterName,
		vehicleNumber,
		ewayBillNumber,
		docketNumber,
		taxType,
		reverseCharge,
		strconv.Itoa(templateID),
		strconv.Itoa(billFromAddressID),
		strconv.Itoa(dispatchFromAddressID),
		strconv.Itoa(billToAddressID),
		strconv.Itoa(transitShipToAddrID),
		shipToIDStrings,
		shipToAddresses,
		serialData,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, "", "", pageContent))
}

// CreateShipment processes the final submission and creates all DCs.
func CreateShipment(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	// Re-parse all data from hidden fields
	templateID, _ := strconv.Atoi(c.FormValue("template_id"))
	numSets, _ := strconv.Atoi(c.FormValue("num_sets"))
	challanDate := c.FormValue("challan_date")
	transporterName := c.FormValue("transporter_name")
	vehicleNumber := c.FormValue("vehicle_number")
	ewayBillNumber := c.FormValue("eway_bill_number")
	docketNumber := c.FormValue("docket_number")
	taxType := c.FormValue("tax_type")
	reverseCharge := c.FormValue("reverse_charge")
	billFromAddressID, _ := strconv.Atoi(c.FormValue("bill_from_address_id"))
	dispatchFromAddressID, _ := strconv.Atoi(c.FormValue("dispatch_from_address_id"))
	billToAddressID, _ := strconv.Atoi(c.FormValue("bill_to_address_id"))
	transitShipToAddrID, _ := strconv.Atoi(c.FormValue("transit_ship_to_address_id"))

	if parseErr := c.Request().ParseMultipartForm(32 << 20); parseErr != nil {
		_ = c.Request().ParseForm()
	}
	shipToIDStrs := c.Request().PostForm["ship_to_address_ids"]
	var shipToAddressIDs []int
	for _, s := range shipToIDStrs {
		id, _ := strconv.Atoi(s)
		if id > 0 {
			shipToAddressIDs = append(shipToAddressIDs, id)
		}
	}

	// Re-validate
	if templateID == 0 || numSets < 1 || challanDate == "" {
		auth.SetFlash(c.Request(), "error", "Invalid shipment data")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}
	if len(shipToAddressIDs) != numSets {
		auth.SetFlash(c.Request(), "error", fmt.Sprintf("Expected %d ship-to addresses, got %d", numSets, len(shipToAddressIDs)))
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	// Validate template belongs to project
	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Template not found or doesn't belong to this project")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	// Build line items with serials
	var lineItems []services.ShipmentLineItem
	for _, p := range products {
		item := services.ShipmentLineItem{
			ProductID:     p.ID,
			QtyPerSet:     p.DefaultQuantity,
			Rate:          p.PerUnitPrice,
			TaxPercentage: p.GSTPercentage,
			Assignments:   make(map[int][]string),
		}

		// Parse all serials
		serialsRaw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		if serialsRaw != "" {
			for _, sn := range strings.Split(serialsRaw, "\n") {
				sn = strings.TrimSpace(sn)
				if sn != "" {
					item.AllSerials = append(item.AllSerials, sn)
				}
			}
		}

		// Parse serial assignments
		for _, shipToID := range shipToAddressIDs {
			assignRaw := c.FormValue(fmt.Sprintf("assign_%d_%d", p.ID, shipToID))
			if assignRaw != "" {
				for _, sn := range strings.Split(assignRaw, "\n") {
					sn = strings.TrimSpace(sn)
					if sn != "" {
						item.Assignments[shipToID] = append(item.Assignments[shipToID], sn)
					}
				}
			}
		}

		lineItems = append(lineItems, item)
	}

	// Validate serial counts
	for i, item := range lineItems {
		expectedTotal := products[i].DefaultQuantity * numSets
		if len(item.AllSerials) > 0 && len(item.AllSerials) != expectedTotal {
			auth.SetFlash(c.Request(), "error", fmt.Sprintf("Product %s: expected %d serials, got %d", products[i].ItemName, expectedTotal, len(item.AllSerials)))
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		}

		// Check duplicates within same product
		seen := make(map[string]bool)
		for _, sn := range item.AllSerials {
			if seen[sn] {
				auth.SetFlash(c.Request(), "error", fmt.Sprintf("Duplicate serial %s for product %s", sn, products[i].ItemName))
				return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
			}
			seen[sn] = true
		}

		// Check assignments don't exceed qty_per_set
		for shipToID, assigned := range item.Assignments {
			if len(assigned) > products[i].DefaultQuantity {
				auth.SetFlash(c.Request(), "error", fmt.Sprintf("Too many serials assigned for product %s to destination %d", products[i].ItemName, shipToID))
				return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
			}
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
				return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
			}
		}
	}

	params := services.ShipmentParams{
		ProjectID:             projectID,
		TemplateID:            templateID,
		NumSets:               numSets,
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
		TransitShipToAddrID:   transitShipToAddrID,
		LineItems:             lineItems,
		CreatedBy:             user.ID,
	}

	result, err := services.CreateShipmentGroupDCs(database.DB, params)
	if err != nil {
		slog.Error("Error creating shipment", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		auth.SetFlash(c.Request(), "error", fmt.Sprintf("Failed to create shipment: %v", err))
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
	}

	auth.SetFlash(c.Request(), "success", fmt.Sprintf("Shipment created successfully with %d DCs", 1+len(result.OfficialDCs)))
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/%d", projectID, result.GroupID))
}

// ShowShipmentGroup displays a shipment group detail page.
func ShowShipmentGroup(c echo.Context) error {
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

	groupID, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Invalid shipment group ID")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dashboard", projectID))
	}

	group, err := database.GetShipmentGroup(groupID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Shipment group not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dashboard", projectID))
	}

	// Get all DCs in this group
	dcPtrs, err := database.GetShipmentGroupDCs(groupID)
	if err != nil {
		slog.Error("Error fetching shipment group DCs", slog.String("error", err.Error()), slog.Int("groupID", groupID))
		dcPtrs = []*models.DeliveryChallan{}
	}

	// Convert []*models.DeliveryChallan to []models.DeliveryChallan
	dcs := make([]models.DeliveryChallan, len(dcPtrs))
	for i, dc := range dcPtrs {
		dcs[i] = *dc
	}

	flashType, flashMessage := auth.PopFlash(c.Request())
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.GroupDetail(
		user,
		project,
		allProjects,
		group,
		dcs,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, flashMessage, flashType, pageContent))
}

// ListShipmentGroups shows all shipment groups for a project.
func ListShipmentGroups(c echo.Context) error {
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

	groupPtrs, err := database.GetShipmentGroupsByProjectID(projectID)
	if err != nil {
		slog.Error("Error fetching shipment groups", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		groupPtrs = []*models.ShipmentGroup{}
	}

	// Convert []*models.ShipmentGroup to []models.ShipmentGroup
	groups := make([]models.ShipmentGroup, len(groupPtrs))
	for i, g := range groupPtrs {
		groups[i] = *g
	}

	flashType, flashMessage := auth.PopFlash(c.Request())
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.List(
		user,
		project,
		allProjects,
		groups,
		flashType,
		flashMessage,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, flashMessage, flashType, pageContent))
}

// IssueShipmentGroup issues all draft DCs in a shipment group.
func IssueShipmentGroup(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	groupID, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid group ID"})
	}

	group, err := database.GetShipmentGroup(groupID)
	if err != nil || group.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Shipment group not found"})
	}

	if group.Status == "issued" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Shipment group is already issued"})
	}

	count, err := database.IssueAllDCsInGroup(groupID, user.ID)
	if err != nil {
		slog.Error("Error issuing DCs in group", slog.String("error", err.Error()), slog.Int("groupID", groupID))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to issue DCs"})
	}

	// Update group status
	if err := database.UpdateShipmentGroupStatus(groupID, "issued"); err != nil {
		slog.Error("Error updating group status", slog.String("error", err.Error()), slog.Int("groupID", groupID))
	}

	auth.SetFlash(c.Request(), "success", fmt.Sprintf("Successfully issued %d DCs", count))
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Successfully issued %d DCs", count),
		"redirect": fmt.Sprintf("/projects/%d/shipments/%d", projectID, groupID),
	})
}
