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

	htmxdc "github.com/narendhupati/dc-management-tool/components/htmx/delivery_challans"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	deliverychallan "github.com/narendhupati/dc-management-tool/components/pages/delivery_challans"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ShowDCDetail dispatches to the correct detail view based on dc_type.
func ShowDCDetail(c echo.Context) error {
	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		projectID, _ := strconv.Atoi(c.Param("id"))
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil {
		projectID, _ := strconv.Atoi(c.Param("id"))
		auth.SetFlash(c.Request(), "error", "DC not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	if dc.DCType == "official" {
		return ShowOfficialDCDetail(c)
	}
	return showTransitDCDetail(c)
}

// showTransitDCDetail shows a Transit DC's details.
func showTransitDCDetail(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "DC not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	transitDetails, _ := database.GetTransitDetailsByDCID(dcID)
	lineItems, _ := database.GetLineItemsByDCID(dcID)

	// Load serial numbers for each line item
	for i := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItems[i].ID)
		lineItems[i].SerialNumbers = serials
	}

	// Calculate totals
	var totalTaxable, totalTax, grandTotal float64
	for _, li := range lineItems {
		totalTaxable += li.TaxableAmount
		totalTax += li.TaxAmount
		grandTotal += li.TotalAmount
	}
	roundedTotal := math.Round(grandTotal)
	roundOff := roundedTotal - grandTotal

	// Get ship-to address data
	var shipToAddress *models.Address
	if dc.ShipToAddressID > 0 {
		shipToAddress, _ = database.GetAddress(dc.ShipToAddressID)
	}
	var billToAddress *models.Address
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}

	// Get Bill From and Dispatch From addresses
	var billFromAddress, dispatchFromAddress *models.Address
	if dc.BillFromAddressID != nil && *dc.BillFromAddressID > 0 {
		billFromAddress, _ = database.GetAddress(*dc.BillFromAddressID)
	}
	if dc.DispatchFromAddressID != nil && *dc.DispatchFromAddressID > 0 {
		dispatchFromAddress, _ = database.GetAddress(*dc.DispatchFromAddressID)
	}

	// Get shipment group info if DC belongs to one
	var shipmentGroup *models.ShipmentGroup
	var siblingDCs []*models.DeliveryChallan
	if dc.ShipmentGroupID != nil {
		shipmentGroup, _ = database.GetShipmentGroup(*dc.ShipmentGroupID)
		siblingDCs, _ = database.GetShipmentGroupDCs(*dc.ShipmentGroupID)
	}

	flashType, flashMessage := auth.PopFlash(c.Request())

	var breadcrumbItems []helpers.Breadcrumb
	breadcrumbItems = append(breadcrumbItems,
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
	)
	if dc.ShipmentGroupID != nil {
		breadcrumbItems = append(breadcrumbItems,
			helpers.Breadcrumb{Title: "Shipments", URL: fmt.Sprintf("/projects/%d/shipments", project.ID)},
			helpers.Breadcrumb{Title: fmt.Sprintf("Group #%d", *dc.ShipmentGroupID), URL: fmt.Sprintf("/projects/%d/shipments/%d", project.ID, *dc.ShipmentGroupID)},
		)
	} else {
		breadcrumbItems = append(breadcrumbItems,
			helpers.Breadcrumb{Title: "All DCs", URL: fmt.Sprintf("/projects/%d/dcs-list", project.ID)},
		)
	}
	breadcrumbItems = append(breadcrumbItems, helpers.Breadcrumb{Title: dc.DCNumber, URL: ""})
	_ = helpers.BuildBreadcrumbs(breadcrumbItems...)

	// Suppress unused variable warnings — these are computed but passed via templ now.
	_ = totalTaxable
	_ = totalTax
	_ = grandTotal
	_ = roundedTotal
	_ = roundOff
	_ = shipToAddress
	_ = billToAddress
	_ = billFromAddress
	_ = dispatchFromAddress
	_ = shipmentGroup
	_ = siblingDCs
	_ = transitDetails

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := deliverychallan.Detail(
		user,
		project,
		allProjects,
		dc,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Transit DC", sidebar, topbar, flashMessage, flashType, pageContent))
}

// LoadTemplateProducts is an HTMX endpoint that returns product line items for a template.
func LoadTemplateProducts(c echo.Context) error {
	templateIDStr := c.Param("tid")
	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid template ID")
	}

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		slog.Error("Error fetching template products", slog.Int("template_id", templateID), slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Failed to load products")
	}

	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil {
		return c.String(http.StatusNotFound, "Template not found")
	}

	_ = tmpl.Purpose // purpose is embedded in the templ component via product data
	return components.RenderOK(c, htmxdc.ProductLines(htmxdc.ProductLinesProps{
		Products: products,
	}))
}

// ShowTransitDCPrintView renders a print-ready view for a Transit DC.
func ShowTransitDCPrintView(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "DC not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	return components.RenderOK(c, deliverychallan.TransitPrint(project, dc))
}

// parseSerialNumbers splits newline-separated serial numbers, trimming whitespace and removing empty lines.
func parseSerialNumbers(raw string) []string {
	lines := strings.Split(raw, "\n")
	var serials []string
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s != "" {
			serials = append(serials, s)
		}
	}
	return serials
}
