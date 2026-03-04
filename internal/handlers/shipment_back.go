package handlers

import (
	"fmt"
	"net/http"
	"strconv"

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

// ─── Create flow back handlers ────────────────────────────────────────────────

// BackToStep1 re-renders Step 1 with carry-forward data from a later step.
func BackToStep1(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)

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

	tid := templateID
	prefill := &pageshipments.ShipStep1Prefill{
		TemplateID:      &tid,
		NumLocations:    numLocations,
		ChallanDate:     challanDate,
		TransporterID:   transporterID,
		TransporterName: transporterName,
		VehicleNumber:   vehicleNumber,
		EwayBillNumber:  ewayBillNumber,
		DocketNumber:    docketNumber,
		TaxType:         taxType,
		ReverseCharge:   reverseCharge,
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pageshipments.WizardStep1(user, project, allProjects, templates, transporters, "", "", csrf.Token(c.Request()), 0, prefill)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, "", "", pageContent))
}

// BackToStep2 re-renders Step 2 with carry-forward data.
func BackToStep2(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, _, shipToAddressIDs := parseStep3Form(c)

	tmpl, _ := database.GetTemplateByID(templateID)
	transitDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransit)
	officialDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)

	billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses := loadAllAddresses(projectID)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pageshipments.WizardStep2(
		user, project, allProjects, tmpl,
		numLocations, challanDate, transitDCNumber, officialDCNumber,
		strconv.Itoa(templateID), transporterName, vehicleNumber,
		ewayBillNumber, docketNumber, taxType, reverseCharge,
		billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses,
		csrf.Token(c.Request()), 0, shipToAddressIDs,
		billFromAddrID, dispatchFromAddrID, billToAddrID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, "", "", pageContent))
}

// BackToStep3 re-renders Step 3 (quantity grid) with carry-forward data.
func BackToStep3(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, transitShipToAddrID, shipToAddressIDs := parseStep3Form(c)

	products, _ := database.GetTemplateProducts(templateID)
	quantities := parseQuantityForm(c, products, shipToAddressIDs)
	quantityAddresses := loadQuantityAddresses(projectID, shipToAddressIDs)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pageshipments.WizardStep3Quantities(
		user, project, allProjects, products, quantityAddresses,
		templateID, numLocations, challanDate,
		transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		billFromAddrID, dispatchFromAddrID, billToAddrID, transitShipToAddrID,
		shipToAddressIDs, csrf.Token(c.Request()), 0,
		"", "",
		quantities, nil,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, "", "", pageContent))
}

// BackToStep4 re-renders Step 4 (serials) with carry-forward data.
func BackToStep4(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, transitShipToAddrID, shipToAddressIDs := parseStep3Form(c)
	_ = billFromAddrID
	_ = dispatchFromAddrID
	_ = billToAddrID
	_ = transitShipToAddrID

	products, _ := database.GetTemplateProducts(templateID)
	quantities := parseQuantityForm(c, products, shipToAddressIDs)
	serialData := parseStep4Form(c, products, shipToAddressIDs)

	// Build prefill maps from serial data
	prefillSerials := make(map[int][]string)
	prefillAssignments := make(map[string][]string)
	for _, pd := range serialData {
		prefillSerials[pd.ProductID] = pd.AllSerials
		for shipToID, assigned := range pd.Assignments {
			key := fmt.Sprintf("%d_%d", pd.ProductID, shipToID)
			prefillAssignments[key] = assigned
		}
	}

	// Build quantity hidden fields
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

	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}
	shipToAddresses := loadShipToAddresses(projectID, shipToAddressIDs)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pageshipments.WizardStep3(
		user, project, allProjects, products, numLocations, challanDate,
		strconv.Itoa(templateID), transporterName, vehicleNumber,
		ewayBillNumber, docketNumber, taxType, reverseCharge,
		strconv.Itoa(billFromAddrID), strconv.Itoa(dispatchFromAddrID),
		strconv.Itoa(billToAddrID), strconv.Itoa(transitShipToAddrID),
		shipToIDStrings, shipToAddresses,
		csrf.Token(c.Request()), 0,
		prefillSerials, prefillAssignments, nil,
		quantityHiddenFields,
		computeProductQuantityTotals(quantities),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Shipment Wizard", sidebar, topbar, "", "", pageContent))
}

// ─── Edit flow back handlers ──────────────────────────────────────────────────

// EditBackToStep1 re-renders Step 1 for the edit flow.
func EditBackToStep1(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}

	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)

	templates, _ := database.GetTemplatesByProjectID(projectID)
	transporters, _ := database.GetTransportersByProjectID(projectID, true)

	transporterID := 0
	for _, t := range transporters {
		if t.CompanyName == transporterName {
			transporterID = t.ID
			break
		}
	}

	tid := templateID
	prefill := &pageshipments.ShipStep1Prefill{
		TemplateID:      &tid,
		NumLocations:    numLocations,
		ChallanDate:     challanDate,
		TransporterID:   transporterID,
		TransporterName: transporterName,
		VehicleNumber:   vehicleNumber,
		EwayBillNumber:  ewayBillNumber,
		DocketNumber:    docketNumber,
		TaxType:         taxType,
		ReverseCharge:   reverseCharge,
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pageshipments.WizardStep1(user, project, allProjects, templates, transporters, "", "", csrf.Token(c.Request()), gid, prefill)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Shipment", sidebar, topbar, "", "", pageContent))
}

// EditBackToStep2 re-renders Step 2 for the edit flow.
func EditBackToStep2(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}

	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, _, shipToAddressIDs := parseStep3Form(c)

	tmpl, _ := database.GetTemplateByID(templateID)
	transitDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransit)
	officialDCNumber, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)

	billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses := loadAllAddresses(projectID)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pageshipments.WizardStep2(
		user, project, allProjects, tmpl,
		numLocations, challanDate, transitDCNumber, officialDCNumber,
		strconv.Itoa(templateID), transporterName, vehicleNumber,
		ewayBillNumber, docketNumber, taxType, reverseCharge,
		billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses,
		csrf.Token(c.Request()), gid, shipToAddressIDs,
		billFromAddrID, dispatchFromAddrID, billToAddrID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 2", sidebar, topbar, "", "", pageContent))
}

// EditBackToStep3 re-renders Step 3 (quantity grid) for the edit flow.
func EditBackToStep3(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}

	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, transitShipToAddrID, shipToAddressIDs := parseStep3Form(c)

	products, _ := database.GetTemplateProducts(templateID)
	quantities := parseQuantityForm(c, products, shipToAddressIDs)
	quantityAddresses := loadQuantityAddresses(projectID, shipToAddressIDs)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pageshipments.WizardStep3Quantities(
		user, project, allProjects, products, quantityAddresses,
		templateID, numLocations, challanDate,
		transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		billFromAddrID, dispatchFromAddrID, billToAddrID, transitShipToAddrID,
		shipToAddressIDs, csrf.Token(c.Request()), gid,
		"", "",
		quantities, nil,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 3: Quantities", sidebar, topbar, "", "", pageContent))
}

// EditBackToStep4 re-renders Step 4 (serials) for the edit flow.
func EditBackToStep4(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	gid, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
	}

	templateID, numLocations, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseStep2Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, transitShipToAddrID, shipToAddressIDs := parseStep3Form(c)
	_ = billFromAddrID
	_ = dispatchFromAddrID
	_ = billToAddrID
	_ = transitShipToAddrID

	products, _ := database.GetTemplateProducts(templateID)
	quantities := parseQuantityForm(c, products, shipToAddressIDs)
	serialData := parseStep4Form(c, products, shipToAddressIDs)

	prefillSerials := make(map[int][]string)
	prefillAssignments := make(map[string][]string)
	for _, pd := range serialData {
		prefillSerials[pd.ProductID] = pd.AllSerials
		for shipToID, assigned := range pd.Assignments {
			key := fmt.Sprintf("%d_%d", pd.ProductID, shipToID)
			prefillAssignments[key] = assigned
		}
	}

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

	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}
	shipToAddresses := loadShipToAddresses(projectID, shipToAddressIDs)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pageshipments.WizardStep3(
		user, project, allProjects, products, numLocations, challanDate,
		strconv.Itoa(templateID), transporterName, vehicleNumber,
		ewayBillNumber, docketNumber, taxType, reverseCharge,
		strconv.Itoa(billFromAddrID), strconv.Itoa(dispatchFromAddrID),
		strconv.Itoa(billToAddrID), strconv.Itoa(transitShipToAddrID),
		shipToIDStrings, shipToAddresses,
		csrf.Token(c.Request()), gid,
		prefillSerials, prefillAssignments, nil,
		quantityHiddenFields,
		computeProductQuantityTotals(quantities),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Shipment — Step 4: Serials", sidebar, topbar, "", "", pageContent))
}

// ─── Shared helper functions ──────────────────────────────────────────────────

// loadAllAddresses loads all 4 address types for a project.
func loadAllAddresses(projectID int) (billFrom, dispatchFrom, billTo, shipTo []*models.Address) {
	billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
	dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")
	billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	if billFromConfig != nil {
		billFrom, _ = database.GetAllAddressesByConfigID(billFromConfig.ID)
	}
	if dispatchFromConfig != nil {
		dispatchFrom, _ = database.GetAllAddressesByConfigID(dispatchFromConfig.ID)
	}
	if billToConfig != nil {
		billTo, _ = database.GetAllAddressesByConfigID(billToConfig.ID)
	}
	if shipToConfig != nil {
		shipTo, _ = database.GetAllAddressesByConfigID(shipToConfig.ID)
	}
	return
}

// loadQuantityAddresses loads QuantityAddress structs for the selected ship-to IDs.
func loadQuantityAddresses(projectID int, shipToAddressIDs []int) []pageshipments.QuantityAddress {
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var result []pageshipments.QuantityAddress
	if shipToConfig != nil {
		allShipTo, _ := database.GetAllAddressesByConfigID(shipToConfig.ID)
		selectedSet := make(map[int]bool)
		for _, id := range shipToAddressIDs {
			selectedSet[id] = true
		}
		for _, a := range allShipTo {
			if selectedSet[a.ID] {
				result = append(result, pageshipments.QuantityAddress{
					ID:   a.ID,
					Name: a.DisplayName(),
				})
			}
		}
	}
	return result
}

// loadShipToAddresses loads Address models for the selected ship-to IDs.
func loadShipToAddresses(projectID int, shipToAddressIDs []int) []*models.Address {
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var result []*models.Address
	if shipToConfig != nil {
		allShipTo, _ := database.GetAllAddressesByConfigID(shipToConfig.ID)
		selectedSet := make(map[int]bool)
		for _, id := range shipToAddressIDs {
			selectedSet[id] = true
		}
		for _, a := range allShipTo {
			if selectedSet[a.ID] {
				result = append(result, a)
			}
		}
	}
	return result
}

