package handlers

import (
	"fmt"
	"log/slog"
	"math"
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

// ─── Types ───────────────────────────────────────────────────────────────────

// transferEditSerialData holds serial numbers for one product during transfer DC edit.
type transferEditSerialData struct {
	ProductID  int
	AllSerials []string
}

// ─── Pure logic (tested) ─────────────────────────────────────────────────────

// validateTransferDCEdit checks if a Transfer DC can be edited based on its current status.
func validateTransferDCEdit(currentStatus string) error {
	if currentStatus != models.DCStatusDraft {
		return fmt.Errorf("only draft Transfer DCs can be edited")
	}
	return nil
}

// buildTransferEditLineItems constructs line items (with pricing + amounts) and their
// serial number slices for the parent delivery_challans record of a Transfer DC.
func buildTransferEditLineItems(products []*models.TemplateProductRow, quantities map[int]map[int]int, serials []transferEditSerialData) ([]models.DCLineItem, [][]string) {
	items := make([]models.DCLineItem, 0, len(products))
	serialsByLine := make([][]string, 0, len(products))

	for _, p := range products {
		qty := 0
		if qMap, ok := quantities[p.ID]; ok {
			for _, q := range qMap {
				qty += q
			}
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

		var sn []string
		for _, sd := range serials {
			if sd.ProductID == p.ID {
				sn = sd.AllSerials
				break
			}
		}
		serialsByLine = append(serialsByLine, sn)
	}

	return items, serialsByLine
}

// ─── Error helper ────────────────────────────────────────────────────────────

func handleTransferEditError(c echo.Context, projectID, tdcID int, err error) error {
	auth.SetFlash(c.Request(), "error", "Failed to save changes: "+err.Error())
	return c.Redirect(http.StatusSeeOther,
		fmt.Sprintf("/projects/%d/transfer-dcs/%d/edit", projectID, tdcID))
}

// ─── HTTP Handlers ───────────────────────────────────────────────────────────

// ShowEditTransferWizard renders Step 1 of the edit wizard pre-filled with existing data.
func ShowEditTransferWizard(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project id")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "project not found")
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transfer DC id")
	}

	// Load Transfer DC; must be draft and belong to this project.
	tdc, err := database.GetTransferDC(tdcID)
	if err != nil || tdc.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "transfer DC not found")
	}
	if err := validateTransferDCEdit(tdc.DCStatus); err != nil {
		auth.SetFlash(c.Request(), "error", err.Error())
		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d/transfer-dcs/%d", projectID, tdc.DCID))
	}

	// Load wizard dropdown dependencies.
	templates, err := database.GetTemplatesByProjectID(project.ID)
	if err != nil {
		slog.Error("Error fetching templates", slog.String("error", err.Error()))
		templates = []*models.DCTemplate{}
	}
	transporters, err := database.GetTransportersByProjectID(project.ID, true)
	if err != nil {
		slog.Error("Error fetching transporters", slog.String("error", err.Error()))
		transporters = []*models.Transporter{}
	}

	// Load hub addresses (ship-to config).
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	var hubAddresses []*models.Address
	if shipToConfig != nil {
		hubAddresses, _ = database.GetAllAddressesByConfigID(shipToConfig.ID)
	}

	// Build challan date string.
	challanDateStr := ""
	if tdc.ChallanDate != nil {
		challanDateStr = *tdc.ChallanDate
	}

	// Look up transporter ID by name.
	transporterID := 0
	for _, t := range transporters {
		if t.CompanyName == tdc.TransporterName {
			transporterID = t.ID
			break
		}
	}

	// Build Step 1 prefill.
	prefill := &pagetransfer.TransferStep1Prefill{
		TemplateID:      tdc.TemplateID,
		ChallanDate:     challanDateStr,
		HubAddressID:    tdc.HubAddressID,
		TransporterID:   transporterID,
		TransporterName: tdc.TransporterName,
		VehicleNumber:   tdc.VehicleNumber,
		EwayBillNumber:  tdc.EwayBillNumber,
		DocketNumber:    tdc.DocketNumber,
		TaxType:         tdc.TaxType,
		ReverseCharge:   tdc.ReverseCharge,
	}

	flashType, flashMessage := auth.PopFlash(c.Request())
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.WizardStep1(
		user, project, allProjects,
		templates, transporters, hubAddresses,
		flashType, flashMessage, csrf.Token(c.Request()),
		prefill,
		tdcID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c,
		layouts.MainWithContent("Edit Transfer DC", sidebar, topbar, flashMessage, flashType, pageContent),
	)
}

// EditTransferWizardStep2 processes edit Step 1 and renders Step 2 (addresses) with prefill.
func EditTransferWizardStep2(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transfer DC id")
	}
	tdc, err := database.GetTransferDC(tdcID)
	if err != nil || tdc.DCStatus != models.DCStatusDraft || tdc.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft transfer DC not found")
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
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/%d/edit", projectID, tdcID))
	}

	// Validate template belongs to project
	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Template not found or doesn't belong to this project")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/%d/edit", projectID, tdcID))
	}

	// Load preselected addresses from existing Transfer DC destinations.
	dests, _ := database.GetTransferDCDestinations(tdcID)
	var preselectedIDs []int
	for _, d := range dests {
		preselectedIDs = append(preselectedIDs, d.ShipToAddressID)
	}

	// Load preselected bill-from/dispatch-from/bill-to from parent DC
	dc, _ := database.GetDeliveryChallanByID(tdc.DCID)
	preselectedBillFromID := 0
	preselectedDispatchFromID := 0
	preselectedBillToID := 0
	if dc != nil {
		if dc.BillFromAddressID != nil {
			preselectedBillFromID = *dc.BillFromAddressID
		}
		if dc.DispatchFromAddressID != nil {
			preselectedDispatchFromID = *dc.DispatchFromAddressID
		}
		if dc.BillToAddressID != nil {
			preselectedBillToID = *dc.BillToAddressID
		}
	}

	transferDCNumber, _ := peekTransferDCNumber(projectID)
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
		preselectedIDs,
		preselectedBillFromID, preselectedDispatchFromID, preselectedBillToID,
		tdcID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 2", sidebar, topbar, "", "", pageContent))
}

// EditTransferWizardQuantityStep processes edit Step 2 (addresses) and renders Step 3 (quantities) with prefill.
func EditTransferWizardQuantityStep(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transfer DC id")
	}
	tdc, err := database.GetTransferDC(tdcID)
	if err != nil || tdc.DCStatus != models.DCStatusDraft || tdc.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft transfer DC not found")
	}

	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	// Validate
	if len(shipToAddressIDs) == 0 {
		auth.SetFlash(c.Request(), "error", "Please select at least one ship-to address")

		tmpl, _ := database.GetTemplateByID(templateID)
		transferDCNumber, _ := peekTransferDCNumber(projectID)
		billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses := loadAllAddresses(projectID)
		allProjects, _ := database.GetAccessibleProjects(user)
		flashType, flashMessage := auth.PopFlash(c.Request())

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
			tdcID,
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
		return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 2", sidebar, topbar, flashMessage, flashType, pageContent))
	}

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/%d/edit", projectID, tdcID))
	}

	quantityAddresses := loadTransferQuantityAddresses(projectID, shipToAddressIDs)

	// Load existing quantity grid from Transfer DC for prefill
	prefillQuantities := loadTransferEditQuantities(tdcID, shipToAddressIDs)

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.WizardStep3(
		user, project, allProjects, products, quantityAddresses,
		templateID, challanDate,
		hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		billFromAddrID, dispatchFromAddrID, billToAddrID,
		shipToAddressIDs, csrf.Token(c.Request()),
		"", "",
		prefillQuantities, nil,
		tdcID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 3: Quantities", sidebar, topbar, "", "", pageContent))
}

// EditTransferWizardStep4 processes edit Step 3 (quantities) and renders Step 4 (serials) with prefill.
func EditTransferWizardStep4(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transfer DC id")
	}
	tdc, err := database.GetTransferDC(tdcID)
	if err != nil || tdc.DCStatus != models.DCStatusDraft || tdc.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft transfer DC not found")
	}

	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/%d/edit", projectID, tdcID))
	}

	quantities := parseQuantityForm(c, products, shipToAddressIDs)

	// Validate quantities
	qtyErrors := validateQuantities(quantities, products, shipToAddressIDs)
	if len(qtyErrors) > 0 {
		globalErr := qtyErrors["global"]
		if globalErr == "" {
			globalErr = "Please fix the quantity errors below"
		}

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
			tdcID,
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", globalErr)
		return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 3: Quantities", sidebar, topbar, globalErr, "error", pageContent))
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

	// Load existing serials from parent DC for prefill
	prefillSerials := loadTransferEditSerials(tdc.DCID)

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
		tdcID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 4: Serials", sidebar, topbar, "", "", pageContent))
}

// EditTransferWizardStep5 processes edit Step 4 (serials) and renders Step 5 (review).
func EditTransferWizardStep5(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transfer DC id")
	}
	tdc, err := database.GetTransferDC(tdcID)
	if err != nil || tdc.DCStatus != models.DCStatusDraft || tdc.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "draft transfer DC not found")
	}

	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddrID, dispatchFromAddrID, billToAddrID, shipToAddressIDs := parseTransferStep2Form(c)

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Failed to load template products")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/%d/edit", projectID, tdcID))
	}

	serialData := parseTransferSerialForm(c, products)
	quantities := parseQuantityForm(c, products, shipToAddressIDs)

	// Validate serial counts
	serialErrors := make(map[int]string)
	for i, sd := range serialData {
		expectedTotal := 0
		for _, addrID := range shipToAddressIDs {
			expectedTotal += quantities[products[i].ID][addrID]
		}
		if len(sd.AllSerials) > 0 && len(sd.AllSerials) != expectedTotal {
			serialErrors[sd.ProductID] = fmt.Sprintf("Expected %d serials, got %d", expectedTotal, len(sd.AllSerials))
			continue
		}
		seen := make(map[string]bool)
		for _, sn := range sd.AllSerials {
			if seen[sn] {
				serialErrors[sd.ProductID] = fmt.Sprintf("Duplicate serial within this product: %s", sn)
				break
			}
			seen[sn] = true
		}
		// Project-wide duplicate check (exclude current DC's serials)
		if _, alreadyHasError := serialErrors[sd.ProductID]; !alreadyHasError && len(sd.AllSerials) > 0 {
			excludeDCID := tdc.DCID
			conflicts, conflictsErr := database.CheckSerialsInProject(projectID, sd.AllSerials, &excludeDCID)
			if conflictsErr != nil {
				slog.Error("Error checking serials", slog.String("error", conflictsErr.Error()))
			}
			if len(conflicts) > 0 {
				serialErrors[sd.ProductID] = fmt.Sprintf("Serial %s already exists in DC %s", conflicts[0].SerialNumber, conflicts[0].DCNumber)
			}
		}
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
			tdcID,
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", "Please fix the serial number errors below")
		return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 4: Serials", sidebar, topbar, "Please fix the serial number errors below", "error", pageContent))
	}

	// Load template for display
	tmpl, _ := database.GetTemplateByID(templateID)

	shipToAddresses := loadShipToAddresses(projectID, shipToAddressIDs)
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
		user, project, allProjects, tmpl, products,
		challanDate, hubAddressName,
		transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		strconv.Itoa(templateID), strconv.Itoa(hubAddressID),
		strconv.Itoa(billFromAddrID), strconv.Itoa(dispatchFromAddrID),
		strconv.Itoa(billToAddrID),
		shipToIDStrings, shipToAddresses, serialData,
		csrf.Token(c.Request()),
		quantityHiddenFields,
		computeProductQuantityTotals(quantities),
		tdcID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 5: Review", sidebar, topbar, "", "", pageContent))
}

// SaveTransferEdit processes the review form and reconciles the DB state.
func SaveTransferEdit(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transfer DC id")
	}

	// 1. Load existing state; draft guard.
	tdc, err := database.GetTransferDC(tdcID)
	if err != nil || tdc.ProjectID != project.ID {
		return echo.NewHTTPError(http.StatusNotFound, "transfer DC not found")
	}
	if err := validateTransferDCEdit(tdc.DCStatus); err != nil {
		auth.SetFlash(c.Request(), "error", "This transfer DC has already been issued and cannot be edited.")
		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d/transfer-dcs/%d", project.ID, tdc.DCID))
	}

	// 2. Parse all form data.
	templateID, challanDate, hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge := parseTransferStep1Form(c)
	billFromAddressID, dispatchFromAddressID, billToAddressID, shipToAddressIDs := parseTransferStep2Form(c)

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		return handleTransferEditError(c, project.ID, tdcID, fmt.Errorf("failed to load template products: %w", err))
	}

	quantities := parseQuantityForm(c, products, shipToAddressIDs)

	// Parse serials
	var serials []transferEditSerialData
	for _, p := range products {
		sd := transferEditSerialData{ProductID: p.ID}
		serialsRaw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		if serialsRaw != "" {
			for _, sn := range strings.Split(serialsRaw, "\n") {
				sn = strings.TrimSpace(sn)
				if sn != "" {
					sd.AllSerials = append(sd.AllSerials, sn)
				}
			}
		}
		serials = append(serials, sd)
	}

	// 3. Load current destinations; compute sets.
	oldDests, err := database.GetTransferDCDestinations(tdcID)
	if err != nil {
		return handleTransferEditError(c, project.ID, tdcID, err)
	}
	oldDestMap := make(map[int]*models.TransferDCDestination) // shipToAddressID → destination
	oldAddressIDs := make([]int, 0, len(oldDests))
	for _, d := range oldDests {
		oldDestMap[d.ShipToAddressID] = d
		oldAddressIDs = append(oldAddressIDs, d.ShipToAddressID)
	}

	newAddressSet := toIntSet(shipToAddressIDs)
	oldAddressSet := toIntSet(oldAddressIDs)
	toKeep := intersectSets(oldAddressSet, newAddressSet)
	toAdd := subtractSets(newAddressSet, oldAddressSet)
	toDelete := subtractSets(oldAddressSet, newAddressSet)

	// 4. Build line items and serials.
	lineItems, serialsByLine := buildTransferEditLineItems(products, quantities, serials)

	var challanDatePtr *string
	if challanDate != "" {
		challanDatePtr = &challanDate
	}

	// 5. Update transfer_dcs record (metadata).
	tdc.HubAddressID = hubAddressID
	tdc.TemplateID = &templateID
	tdc.TaxType = taxType
	tdc.ReverseCharge = reverseCharge
	tdc.TransporterName = transporterName
	tdc.VehicleNumber = vehicleNumber
	tdc.EwayBillNumber = ewayBillNumber
	tdc.DocketNumber = docketNumber
	if err := database.UpdateTransferDC(tdc); err != nil {
		return handleTransferEditError(c, project.ID, tdcID, err)
	}

	// 6. Update parent delivery_challans record (addresses, date).
	if err := database.UpdateDeliveryChallanAddressesAndDate(
		tdc.DCID, challanDatePtr,
		billFromAddressID, dispatchFromAddressID, billToAddressID, hubAddressID,
	); err != nil {
		return handleTransferEditError(c, project.ID, tdcID, err)
	}

	// 7. Replace line items and serials on parent DC.
	if err := database.ReplaceLineItemsAndSerials(tdc.DCID, project.ID, lineItems, serialsByLine); err != nil {
		return handleTransferEditError(c, project.ID, tdcID, err)
	}

	// 8. Reconcile destinations.
	// Update existing destinations' quantities.
	for _, addrID := range toKeep {
		dest := oldDestMap[addrID]
		var destQtys []models.TransferDCDestinationQty
		for _, p := range products {
			qty := 0
			if qMap, ok := quantities[p.ID]; ok {
				qty = qMap[addrID]
			}
			destQtys = append(destQtys, models.TransferDCDestinationQty{
				ProductID: p.ID,
				Quantity:  qty,
			})
		}
		if err := database.SetDestinationQuantities(dest.ID, destQtys); err != nil {
			return handleTransferEditError(c, project.ID, tdcID, err)
		}
	}

	// Delete removed destinations.
	for _, addrID := range toDelete {
		dest := oldDestMap[addrID]
		if err := database.DeleteTransferDCDestination(dest.ID); err != nil {
			return handleTransferEditError(c, project.ID, tdcID, err)
		}
	}

	// Add new destinations.
	if len(toAdd) > 0 {
		if err := database.AddTransferDCDestinations(tdcID, toAdd); err != nil {
			return handleTransferEditError(c, project.ID, tdcID, err)
		}
		// Set quantities for newly added destinations
		newDests, _ := database.GetTransferDCDestinations(tdcID)
		for _, d := range newDests {
			if newAddressSet[d.ShipToAddressID] && !oldAddressSet[d.ShipToAddressID] {
				var destQtys []models.TransferDCDestinationQty
				for _, p := range products {
					qty := 0
					if qMap, ok := quantities[p.ID]; ok {
						qty = qMap[d.ShipToAddressID]
					}
					destQtys = append(destQtys, models.TransferDCDestinationQty{
						ProductID: p.ID,
						Quantity:  qty,
					})
				}
				if err := database.SetDestinationQuantities(d.ID, destQtys); err != nil {
					return handleTransferEditError(c, project.ID, tdcID, err)
				}
			}
		}
	}

	// 9. Recalculate destination count.
	if err := database.RecalculateSplitProgress(tdcID); err != nil {
		slog.Error("Error recalculating split progress", slog.String("error", err.Error()))
	}

	_ = user // used for context
	auth.SetFlash(c.Request(), "success", "Transfer DC updated successfully.")
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d/transfer-dcs/%d", project.ID, tdc.DCID))
}

// ─── Edit Back Navigation ────────────────────────────────────────────────────

// EditTransferBackToStep1 re-renders Step 1 with carry-forward data.
func EditTransferBackToStep1(c echo.Context) error {
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

	transporterID := 0
	for _, t := range transporters {
		if t.CompanyName == transporterName {
			transporterID = t.ID
			break
		}
	}

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
	tdcID, _ := strconv.Atoi(c.Param("tdcid"))
	pageContent := pagetransfer.WizardStep1(user, project, allProjects, templates, transporters, hubAddresses, "", "", csrf.Token(c.Request()), prefill, tdcID)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC", sidebar, topbar, "", "", pageContent))
}

// EditTransferBackToStep2 re-renders Step 2 with carry-forward data.
func EditTransferBackToStep2(c echo.Context) error {
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
	transferDCNumber, _ := peekTransferDCNumber(projectID)

	billFromAddresses, dispatchFromAddresses, billToAddresses, shipToAddresses := loadAllAddresses(projectID)

	allProjects, _ := database.GetAccessibleProjects(user)
	tdcID, _ := strconv.Atoi(c.Param("tdcid"))
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
		tdcID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 2", sidebar, topbar, "", "", pageContent))
}

// EditTransferBackToStep3 re-renders Step 3 with carry-forward data.
func EditTransferBackToStep3(c echo.Context) error {
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
	tdcID, _ := strconv.Atoi(c.Param("tdcid"))
	pageContent := pagetransfer.WizardStep3(
		user, project, allProjects, products, quantityAddresses,
		templateID, challanDate,
		hubAddressID, transporterName, vehicleNumber, ewayBillNumber, docketNumber,
		taxType, reverseCharge,
		billFromAddrID, dispatchFromAddrID, billToAddrID,
		shipToAddressIDs, csrf.Token(c.Request()),
		"", "",
		quantities, nil,
		tdcID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 3: Quantities", sidebar, topbar, "", "", pageContent))
}

// EditTransferBackToStep4 re-renders Step 4 with carry-forward data.
func EditTransferBackToStep4(c echo.Context) error {
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

	prefillSerials := make(map[int][]string)
	for _, sd := range serialData {
		prefillSerials[sd.ProductID] = sd.AllSerials
	}

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
	tdcID, _ := strconv.Atoi(c.Param("tdcid"))
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
		tdcID,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Transfer DC — Step 4: Serials", sidebar, topbar, "", "", pageContent))
}

// ─── Edit helper functions ───────────────────────────────────────────────────

// peekTransferDCNumber wraps the services.PeekNextDCNumber call for transfer DCs.
func peekTransferDCNumber(projectID int) (string, error) {
	return services.PeekNextDCNumber(database.DB, projectID, services.DCTypeTransfer)
}

// loadTransferEditQuantities loads existing per-destination quantities from a Transfer DC
// and remaps them into a product→address→qty map suitable for quantity grid prefill.
func loadTransferEditQuantities(tdcID int, shipToAddressIDs []int) map[int]map[int]int {
	dests, err := database.GetTransferDCDestinations(tdcID)
	if err != nil {
		return nil
	}

	// Build destination ID → ship-to address ID map
	destIDToAddrID := make(map[int]int)
	for _, d := range dests {
		destIDToAddrID[d.ID] = d.ShipToAddressID
	}

	// Collect all destination IDs
	destIDs := make([]int, 0, len(dests))
	for _, d := range dests {
		destIDs = append(destIDs, d.ID)
	}

	qtyByDest, err := database.GetQuantitiesForDestinations(destIDs)
	if err != nil {
		return nil
	}

	// Remap: product → shipToAddressID → qty
	result := make(map[int]map[int]int)
	for destID, qtys := range qtyByDest {
		addrID := destIDToAddrID[destID]
		for _, q := range qtys {
			if result[q.ProductID] == nil {
				result[q.ProductID] = make(map[int]int)
			}
			result[q.ProductID][addrID] = q.Quantity
		}
	}
	return result
}

// loadTransferEditSerials loads existing serial numbers from the parent DC's line items.
func loadTransferEditSerials(dcID int) map[int][]string {
	lineItems, err := database.GetLineItemsByDCID(dcID)
	if err != nil {
		return nil
	}
	result := make(map[int][]string)
	for _, item := range lineItems {
		serials, err := database.GetSerialNumbersByLineItemID(item.ID)
		if err != nil {
			continue
		}
		result[item.ProductID] = serials
	}
	return result
}
