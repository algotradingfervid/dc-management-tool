package handlers

import (
	"fmt"
	"math"
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

// ShowEditShipmentWizard renders Step 1 of the shipment wizard pre-filled
// with the existing draft group data. It is the entry point for the edit flow.
func ShowEditShipmentWizard(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "project not found")
	}

	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}

	// 1. Load the group; must exist, be draft, and belong to this project.
	group, err := database.GetShipmentGroup(gid)
	if err != nil || group.Status != "draft" || group.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft shipment group not found")
	}

	// 2. Load all DCs in the group (transit + official).
	dcs, err := database.GetShipmentGroupDCs(gid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// 3. Find the transit DC.
	var transitDC *models.DeliveryChallan
	for _, dc := range dcs {
		if dc.DCType == "transit" {
			transitDC = dc
			break
		}
	}
	if transitDC == nil {
		return echo.NewHTTPError(http.StatusNotFound, "transit DC not found in group")
	}

	// 4. Load transit details (transporter, vehicle, eway bill, notes).
	transitDetails, err := database.GetTransitDetailsByDCID(transitDC.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// 5. Load line items for the transit DC; collect serials per product.
	lineItems, err := database.GetLineItemsByDCID(transitDC.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	prefillSerials := map[int][]string{}
	for _, item := range lineItems {
		serials, serErr := database.GetSerialNumbersByLineItemID(item.ID)
		if serErr != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, serErr.Error())
		}
		prefillSerials[item.ProductID] = serials
	}

	// 6. Collect preselected ship-to address IDs from official DCs.
	// GetShipmentGroupDCs does not populate ShipToAddressID, so we fetch each full DC.
	preselectedIDs := []int{}
	for _, dc := range dcs {
		if dc.DCType == "official" {
			fullDC, fullErr := database.GetDeliveryChallanByID(dc.ID)
			if fullErr == nil && fullDC.ShipToAddressID > 0 {
				preselectedIDs = append(preselectedIDs, fullDC.ShipToAddressID)
			}
		}
	}

	// 7. Load wizard dropdown dependencies.
	templates, err := database.GetTemplatesByProjectID(project.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	transporters, err := database.GetTransportersByProjectID(project.ID, true)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	allProjects, _ := database.GetAccessibleProjects(user)

	// 8. Build challan date string for Step 1 prefill.
	// ChallanDate is *string on the model.
	challanDateStr := ""
	if transitDC.ChallanDate != nil {
		challanDateStr = *transitDC.ChallanDate
	}

	// Look up transporter ID by name from the transporters list.
	transporterID := 0
	for _, t := range transporters {
		if t.CompanyName == transitDetails.TransporterName {
			transporterID = t.ID
			break
		}
	}

	// Build Step 1 prefill from loaded group + transit DC data.
	prefill := &pageshipments.ShipStep1Prefill{
		TemplateID:      group.TemplateID,
		NumLocations:    group.NumLocations,
		ChallanDate:     challanDateStr,
		TransporterID:   transporterID,
		TransporterName: transitDetails.TransporterName,
		VehicleNumber:   transitDetails.VehicleNumber,
		EwayBillNumber:  transitDetails.EwayBillNumber,
		DocketNumber:    transitDetails.Notes,
		TaxType:         group.TaxType,
		ReverseCharge:   group.ReverseCharge,
	}
	_ = prefillSerials
	_ = preselectedIDs

	flashType, flashMessage := auth.PopFlash(c.Request())
	csrfToken := csrf.Token(c.Request())

	pageContent := pageshipments.WizardStep1(
		user,
		project,
		allProjects,
		templates,
		transporters,
		flashType,
		flashMessage,
		csrfToken,
		gid,
		prefill,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)

	return components.RenderOK(c,
		layouts.MainWithContent("Edit Shipment", sidebar, topbar, flashMessage, flashType, pageContent),
	)
}

// safeStr dereferences a *string safely, returning "" for nil.
func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// EditWizardStep2 processes Step 1 of the edit wizard and renders Step 2 (address selection)
// with the group's current official-DC ship-to addresses pre-selected.
func EditWizardStep2(c echo.Context) error {
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

	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}
	group, err := database.GetShipmentGroup(gid)
	if err != nil || group.Status != "draft" || group.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft shipment group not found")
	}

	// Parse step 1 data (submitted by the edit step 1 form)
	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)

	// Validate template belongs to project
	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Template not found or doesn't belong to this project")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/%d/edit", projectID, gid))
	}

	// Peek DC numbers for the summary header
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

	// Load preselected address IDs from the group's existing DCs.
	dcs, _ := database.GetShipmentGroupDCs(gid)
	var preselectedIDs []int
	preselectedBillFromID := 0
	preselectedDispatchFromID := 0
	preselectedBillToID := 0
	for _, dc := range dcs {
		if dc.DCType == "official" {
			fullDC, fullErr := database.GetDeliveryChallanByID(dc.ID)
			if fullErr == nil && fullDC.ShipToAddressID > 0 {
				preselectedIDs = append(preselectedIDs, fullDC.ShipToAddressID)
			}
			// Use first official DC for bill-to pre-selection
			if fullErr == nil && preselectedBillToID == 0 && fullDC.BillToAddressID != nil && *fullDC.BillToAddressID > 0 {
				preselectedBillToID = *fullDC.BillToAddressID
			}
		} else if dc.DCType == "transit" {
			fullDC, fullErr := database.GetDeliveryChallanByID(dc.ID)
			if fullErr == nil {
				if fullDC.BillFromAddressID != nil {
					preselectedBillFromID = *fullDC.BillFromAddressID
				}
				if fullDC.DispatchFromAddressID != nil {
					preselectedDispatchFromID = *fullDC.DispatchFromAddressID
				}
			}
		}
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request())

	pageContent := pageshipments.WizardStep2(
		user,
		project,
		allProjects,
		tmpl,
		numLocations,
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
		gid,
		preselectedIDs,
		preselectedBillFromID,
		preselectedDispatchFromID,
		preselectedBillToID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 2", sidebar, topbar, flashMessage, flashType, pageContent))
}

// EditWizardStep3 processes Step 2 of the edit wizard (addresses) and renders
// Step 3 (quantity grid) with existing per-location quantities pre-filled.
func EditWizardStep3(c echo.Context) error {
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

	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}
	group, err := database.GetShipmentGroup(gid)
	if err != nil || group.Status != "draft" || group.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft shipment group not found")
	}

	// Parse step 1 and step 2 data
	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddressID, dispatchFromAddressID, billToAddressID, transitShipToAddrID, shipToAddressIDs := parseStep3Form(c)

	// Validate address selections
	validationErrors := make(map[string]string)
	if len(shipToAddressIDs) == 0 {
		validationErrors["ship_to"] = "Please select at least one ship-to address"
	} else if len(shipToAddressIDs) != numLocations {
		validationErrors["ship_to"] = fmt.Sprintf("Please select exactly %d ship-to addresses to match the number of locations (got %d)", numLocations, len(shipToAddressIDs))
	}
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
		var msgs []string
		for _, msg := range validationErrors {
			msgs = append(msgs, msg)
		}
		errMsg := strings.Join(msgs, ". ")

		// Re-render Step 2 in edit mode; preselect whatever the user just submitted
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

		tmplForRerender, _ := database.GetTemplateByID(templateID)
		transitDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransit)
		officialDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)
		allProjects, _ := database.GetAccessibleProjects(user)

		pageContent := pageshipments.WizardStep2(
			user, project, allProjects,
			tmplForRerender,
			numLocations, challanDate,
			transitDCNumber, officialDCNumber,
			strconv.Itoa(templateID),
			transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge,
			billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses,
			csrf.Token(c.Request()),
			gid,
			shipToAddressIDs,
			billFromAddressID, dispatchFromAddressID, billToAddressID,
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", errMsg)
		return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 2", sidebar, topbar, errMsg, "error", pageContent))
	}

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/%d/edit", projectID, gid))
	}

	// Load ship-to address details for quantity grid column headers
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var quantityAddresses []pageshipments.QuantityAddress
	if shipToConfig != nil {
		allShipTo, _ := database.GetAllAddressesByConfigID(shipToConfig.ID)
		selectedSet := make(map[int]bool)
		for _, id := range shipToAddressIDs {
			selectedSet[id] = true
		}
		for _, a := range allShipTo {
			if selectedSet[a.ID] {
				quantityAddresses = append(quantityAddresses, pageshipments.QuantityAddress{
					ID:   a.ID,
					Name: a.DisplayName(),
				})
			}
		}
	}

	// Load existing per-location quantities from official DCs to pre-fill grid
	dcs, _ := database.GetShipmentGroupDCs(gid)
	prefillQuantities := make(map[int]map[int]int)
	for _, dc := range dcs {
		if dc.DCType == "official" {
			fullDC, fullErr := database.GetDeliveryChallanByID(dc.ID)
			if fullErr != nil {
				continue
			}
			officialLineItems, _ := database.GetLineItemsByDCID(dc.ID)
			for _, li := range officialLineItems {
				if prefillQuantities[li.ProductID] == nil {
					prefillQuantities[li.ProductID] = make(map[int]int)
				}
				prefillQuantities[li.ProductID][fullDC.ShipToAddressID] = li.Quantity
			}
		}
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.WizardStep3Quantities(
		user, project, allProjects, products, quantityAddresses,
		templateID, numLocations, challanDate,
		transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		billFromAddressID, dispatchFromAddressID, billToAddressID, transitShipToAddrID,
		shipToAddressIDs, csrf.Token(c.Request()), gid,
		"", "",
		prefillQuantities, nil,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 3: Quantities", sidebar, topbar, "", "", pageContent))
}

// EditWizardStep4 processes Step 3 of the edit wizard (quantities) and renders
// Step 4 (serial entry) with existing serial numbers pre-filled.
func EditWizardStep4(c echo.Context) error {
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

	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}
	group, err := database.GetShipmentGroup(gid)
	if err != nil || group.Status != "draft" || group.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft shipment group not found")
	}

	// Parse carry-forward data
	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddressID, dispatchFromAddressID, billToAddressID, transitShipToAddrID, shipToAddressIDs := parseStep3Form(c)

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/%d/edit", projectID, gid))
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
		shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
		var quantityAddresses []pageshipments.QuantityAddress
		if shipToConfig != nil {
			allShipTo, _ := database.GetAllAddressesByConfigID(shipToConfig.ID)
			selectedSet := make(map[int]bool)
			for _, id := range shipToAddressIDs {
				selectedSet[id] = true
			}
			for _, a := range allShipTo {
				if selectedSet[a.ID] {
					quantityAddresses = append(quantityAddresses, pageshipments.QuantityAddress{
						ID:   a.ID,
						Name: a.DisplayName(),
					})
				}
			}
		}

		allProjects, _ := database.GetAccessibleProjects(user)
		pageContent := pageshipments.WizardStep3Quantities(
			user, project, allProjects, products, quantityAddresses,
			templateID, numLocations, challanDate,
			transporterName, vehicleNumber, ewayBillNumber, docketNumber,
			taxType, reverseCharge,
			billFromAddressID, dispatchFromAddressID, billToAddressID, transitShipToAddrID,
			shipToAddressIDs, csrf.Token(c.Request()), gid,
			globalErr, "error",
			quantities, qtyErrors,
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", globalErr)
		return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 3: Quantities", sidebar, topbar, globalErr, "error", pageContent))
	}

	// Load ship-to address details for serial assignment UI
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

	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}

	// Build quantity hidden field values for carry-forward
	var quantityHiddenFields []pageshipments.QuantityHiddenField
	for _, p := range products {
		for _, addrID := range shipToAddressIDs {
			qty := 0
			if qMap, ok := quantities[p.ID]; ok {
				qty = qMap[addrID]
			}
			quantityHiddenFields = append(quantityHiddenFields, pageshipments.QuantityHiddenField{
				Name:  fmt.Sprintf("qty_%d_%d", p.ID, addrID),
				Value: strconv.Itoa(qty),
			})
		}
	}

	// Load existing serial numbers from the transit DC to pre-fill serial textareas
	dcs, _ := database.GetShipmentGroupDCs(gid)
	var transitDC *models.DeliveryChallan
	for _, dc := range dcs {
		if dc.DCType == "transit" {
			transitDC = dc
			break
		}
	}
	prefillSerials := map[int][]string{}
	if transitDC != nil {
		lineItems, _ := database.GetLineItemsByDCID(transitDC.ID)
		for _, item := range lineItems {
			serials, _ := database.GetSerialNumbersByLineItemID(item.ID)
			prefillSerials[item.ProductID] = serials
		}
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.WizardStep3(
		user, project, allProjects, products, numLocations, challanDate,
		strconv.Itoa(templateID), transporterName, vehicleNumber,
		ewayBillNumber, docketNumber, taxType, reverseCharge,
		strconv.Itoa(billFromAddressID), strconv.Itoa(dispatchFromAddressID),
		strconv.Itoa(billToAddressID), strconv.Itoa(transitShipToAddrID),
		shipToIDStrings, shipToAddresses,
		csrf.Token(c.Request()), gid,
		prefillSerials, nil, nil,
		quantityHiddenFields,
		computeProductQuantityTotals(quantities),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 4: Serials", sidebar, topbar, "", "", pageContent))
}

// EditWizardStep5 processes Step 4 of the edit wizard (serials) and renders Step 5 (review).
func EditWizardStep5(c echo.Context) error {
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

	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}
	group, err := database.GetShipmentGroup(gid)
	if err != nil || group.Status != "draft" || group.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft shipment group not found")
	}

	// Parse all carry-forward data
	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddressID, dispatchFromAddressID, billToAddressID, transitShipToAddrID, shipToAddressIDs := parseStep3Form(c)

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/%d/edit", projectID, gid))
	}

	// Parse serial numbers per product
	serialData := parseStep4Form(c, products, shipToAddressIDs)

	// Parse quantities from carry-forward data
	quantities := parseQuantityForm(c, products, shipToAddressIDs)

	// Validate serial counts using per-location quantities
	validationErrors := make(map[string]string)
	for i, pd := range serialData {
		// Sum quantities across all locations for this product
		expectedTotal := 0
		for _, addrID := range shipToAddressIDs {
			expectedTotal += quantities[products[i].ID][addrID]
		}
		if len(pd.AllSerials) > 0 && len(pd.AllSerials) != expectedTotal {
			validationErrors[fmt.Sprintf("serials_%d", pd.ProductID)] = fmt.Sprintf("Expected %d serials, got %d", expectedTotal, len(pd.AllSerials))
		}
		seen := make(map[string]bool)
		for _, sn := range pd.AllSerials {
			if seen[sn] {
				validationErrors[fmt.Sprintf("serials_%d", pd.ProductID)] = fmt.Sprintf("Duplicate serial: %s", sn)
				break
			}
			seen[sn] = true
		}
		for shipToID, assigned := range pd.Assignments {
			locationQty := quantities[products[i].ID][shipToID]
			if len(assigned) > locationQty {
				validationErrors[fmt.Sprintf("assign_%d_%d", pd.ProductID, shipToID)] = fmt.Sprintf("Too many serials assigned (max %d)", locationQty)
			}
		}
	}

	if len(validationErrors) > 0 {
		auth.SetFlash(c.Request(), "error", "Serial number validation failed")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/%d/edit", projectID, gid))
	}

	// Load template for display
	tmpl, _ := database.GetTemplateByID(templateID)

	// Load ship-to address details for review display
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

	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}

	// Build quantity hidden fields from carry-forward data
	var quantityHiddenFields []pageshipments.QuantityHiddenField
	for _, p := range products {
		for _, addrID := range shipToAddressIDs {
			qty := quantities[p.ID][addrID]
			quantityHiddenFields = append(quantityHiddenFields, pageshipments.QuantityHiddenField{
				Name:  fmt.Sprintf("qty_%d_%d", p.ID, addrID),
				Value: strconv.Itoa(qty),
			})
		}
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageshipments.WizardStep4(
		user,
		project,
		allProjects,
		tmpl,
		products,
		numLocations,
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
		gid,
		quantityHiddenFields,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 5: Review", sidebar, topbar, "", "", pageContent))
}

// ─── Set utility helpers (used by SaveShipmentEdit reconciliation) ───────────

// toIntSet converts a []int into a map[int]bool for O(1) membership tests.
func toIntSet(ids []int) map[int]bool {
	s := make(map[int]bool, len(ids))
	for _, id := range ids {
		s[id] = true
	}
	return s
}

// intersectSets returns the elements present in both sets a and b.
func intersectSets(a, b map[int]bool) []int {
	var result []int
	for k := range a {
		if b[k] {
			result = append(result, k)
		}
	}
	return result
}

// subtractSets returns elements in a that are not in b (a minus b).
func subtractSets(a, b map[int]bool) []int {
	var result []int
	for k := range a {
		if !b[k] {
			result = append(result, k)
		}
	}
	return result
}

// ─── Line-item builders (used by SaveShipmentEdit) ────────────────────────────

// buildTransitLineItems constructs line items (with pricing + amounts) and their
// serial number slices for the transit DC. Uses per-location quantities when available.
func buildTransitLineItems(products []*models.TemplateProductRow, serialData []pageshipments.WizardSerialData, numLocations int, quantities map[int]map[int]int) ([]models.DCLineItem, [][]string) {
	items := make([]models.DCLineItem, 0, len(products))
	serialsByLine := make([][]string, 0, len(products))
	for _, p := range products {
		// Calculate total qty: sum of per-location quantities, or fallback to default * numLocations
		qty := 0
		if qMap, ok := quantities[p.ID]; ok && len(qMap) > 0 {
			for _, q := range qMap {
				qty += q
			}
		} else {
			qty = p.DefaultQuantity * numLocations
		}
		taxable := math.Round(p.PerUnitPrice*float64(qty)*100) / 100
		tax := math.Round(taxable*p.GSTPercentage/100*100) / 100
		total := math.Round((taxable+tax)*100) / 100
		items = append(items, models.DCLineItem{
			ProductID:     p.ID,
			Quantity:      qty,
			Rate:          p.PerUnitPrice,
			TaxPercentage: p.GSTPercentage,
			TaxableAmount: taxable,
			TaxAmount:     tax,
			TotalAmount:   total,
		})
		var serials []string
		for _, sd := range serialData {
			if sd.ProductID == p.ID {
				serials = sd.AllSerials
				break
			}
		}
		serialsByLine = append(serialsByLine, serials)
	}
	return items, serialsByLine
}

// buildOfficialLineItems constructs line items (no pricing, no serials) for an official DC
// with per-location quantities.
func buildOfficialLineItems(products []*models.TemplateProductRow, quantities map[int]map[int]int, shipToAddrID int) []models.DCLineItem {
	items := make([]models.DCLineItem, 0, len(products))
	for _, p := range products {
		qty := p.DefaultQuantity // fallback
		if qMap, ok := quantities[p.ID]; ok {
			if q, ok := qMap[shipToAddrID]; ok {
				qty = q
			}
		}
		items = append(items, models.DCLineItem{
			ProductID: p.ID,
			Quantity:  qty,
		})
	}
	return items
}

// ─── Error helper ─────────────────────────────────────────────────────────────

// handleEditError sets an error flash and redirects back to Step 1 of the edit wizard.
func handleEditError(c echo.Context, projectID, gid int, err error) error {
	auth.SetFlash(c.Request(), "error", "Failed to save changes: "+err.Error())
	return c.Redirect(http.StatusSeeOther,
		fmt.Sprintf("/projects/%d/shipments/%d/edit", projectID, gid))
}

// ─── SaveShipmentEdit ─────────────────────────────────────────────────────────

// SaveShipmentEdit processes the Step 4 review form and reconciles the DB state to
// match the new desired state. Each DB function manages its own transaction; on any
// error the handler redirects (soft rollback — no atomic guarantee across operations).
func SaveShipmentEdit(c echo.Context) error {
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

	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}

	// 1. Load existing state. Draft guard must run before any DB writes.
	group, err := database.GetShipmentGroup(gid)
	if err != nil || group.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "shipment group not found")
	}
	if group.Status != "draft" {
		auth.SetFlash(c.Request(), "error", "This shipment has already been issued and cannot be edited.")
		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d/shipments/%d", project.ID, gid))
	}

	// 2. Parse all form data (same hidden fields carried through Steps 1–4).
	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	_, _, _, _, shipToAddressIDs := parseStep3Form(c)

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		return handleEditError(c, project.ID, gid, fmt.Errorf("failed to load template products: %w", err))
	}

	serialData := parseStep4Form(c, products, shipToAddressIDs)

	// 3. Load current DCs; build officialDCMap keyed by ship_to_address_id.
	// GetShipmentGroupDCs does not populate ShipToAddressID, so fetch each full DC.
	allDCs, err := database.GetShipmentGroupDCs(gid)
	if err != nil {
		return handleEditError(c, project.ID, gid, err)
	}
	var transitDC *models.DeliveryChallan
	officialDCMap := map[int]*models.DeliveryChallan{}
	for _, dc := range allDCs {
		if dc.DCType == "transit" {
			transitDC = dc
		} else {
			fullDC, fullErr := database.GetDeliveryChallanByID(dc.ID)
			if fullErr == nil && fullDC.ShipToAddressID > 0 {
				officialDCMap[fullDC.ShipToAddressID] = fullDC
			}
		}
	}
	if transitDC == nil {
		return handleEditError(c, project.ID, gid, fmt.Errorf("transit DC not found in group"))
	}

	// 4. Compute address sets.
	oldAddressIDs := make([]int, 0, len(officialDCMap))
	for k := range officialDCMap {
		oldAddressIDs = append(oldAddressIDs, k)
	}
	newAddressSet := toIntSet(shipToAddressIDs)
	oldAddressSet := toIntSet(oldAddressIDs)
	toUpdate := intersectSets(oldAddressSet, newAddressSet)
	toAdd := subtractSets(newAddressSet, oldAddressSet)
	toDelete := subtractSets(oldAddressSet, newAddressSet)

	// 5. Build line items and serials.
	quantities := parseQuantityForm(c, products, shipToAddressIDs)
	transitLineItems, transitSerialsByLine := buildTransitLineItems(products, serialData, numLocations, quantities)

	var challanDatePtr *string
	if challanDate != "" {
		challanDatePtr = &challanDate
	}

	// 6–10. Execute reconciliation.
	// Note: soft-rollback design — each DB call manages its own tx. A mid-flight failure
	// leaves partial writes; redirect prevents further damage.
	templateIDCopy := templateID
	if err := database.UpdateShipmentGroup(gid, &templateIDCopy, numLocations, taxType, reverseCharge); err != nil {
		return handleEditError(c, project.ID, gid, err)
	}
	if err := database.UpdateTransitDC(transitDC.ID, challanDatePtr, transporterName, vehicleNumber, ewayBillNumber, docketNumber); err != nil {
		return handleEditError(c, project.ID, gid, err)
	}
	if err := database.ReplaceLineItemsAndSerials(transitDC.ID, project.ID, transitLineItems, transitSerialsByLine); err != nil {
		return handleEditError(c, project.ID, gid, err)
	}

	for _, addressID := range toUpdate {
		dc := officialDCMap[addressID]
		if err := database.UpdateOfficialDC(dc.ID, addressID, challanDatePtr); err != nil {
			return handleEditError(c, project.ID, gid, err)
		}
		officialLI := buildOfficialLineItems(products, quantities, addressID)
		if err := database.ReplaceLineItemsAndSerials(dc.ID, project.ID, officialLI, nil); err != nil {
			return handleEditError(c, project.ID, gid, err)
		}
	}

	for _, addressID := range toDelete {
		dc := officialDCMap[addressID]
		if err := database.DeleteOfficialDC(dc.ID); err != nil {
			return handleEditError(c, project.ID, gid, err)
		}
	}

	for _, addressID := range toAdd {
		officialLI := buildOfficialLineItems(products, quantities, addressID)
		if _, err := database.CreateOfficialDCInGroup(
			project.ID, gid, addressID, challanDatePtr,
			taxType, reverseCharge,
			officialLI, nil,
			user.ID,
		); err != nil {
			return handleEditError(c, project.ID, gid, err)
		}
	}

	// 11. Success — redirect to group detail.
	auth.SetFlash(c.Request(), "success", "Shipment updated successfully.")
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d/shipments/%d", project.ID, gid))
}
