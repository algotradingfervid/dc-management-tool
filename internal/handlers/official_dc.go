package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"

	"github.com/narendhupati/dc-management-tool/components/layouts"
	deliverychallan "github.com/narendhupati/dc-management-tool/components/pages/delivery_challans"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// lineItemsToPointers converts a []models.DCLineItem slice to []*models.DCLineItem.
func lineItemsToPointers(items []models.DCLineItem) []*models.DCLineItem {
	out := make([]*models.DCLineItem, len(items))
	for i := range items {
		out[i] = &items[i]
	}
	return out
}

// ShowOfficialDCDetail shows an Official DC's details.
func ShowOfficialDCDetail(c echo.Context) error {
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
		slog.Error("Error fetching project", slog.Int("project_id", projectID), slog.String("error", err.Error()))
		return c.Redirect(http.StatusFound, "/projects")
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		if err != nil {
			slog.Error("Error fetching DC", slog.Int("dc_id", dcID), slog.String("error", err.Error()))
		}
		auth.SetFlash(c.Request(), "error", "DC not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	lineItemsVal, _ := database.GetLineItemsByDCID(dcID)

	// Load serial numbers for each line item
	for i := range lineItemsVal {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItemsVal[i].ID)
		lineItemsVal[i].SerialNumbers = serials
	}

	lineItems := lineItemsToPointers(lineItemsVal)

	// Get addresses
	var shipToAddress *models.Address
	if dc.ShipToAddressID > 0 {
		shipToAddress, _ = database.GetAddress(dc.ShipToAddressID)
	}
	var billToAddress *models.Address
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}

	// Get shipment group info if DC belongs to one
	var shipmentGroup *models.ShipmentGroup
	var siblingDCs []*models.DeliveryChallan
	var dcPosition int
	var officialCount int
	if dc.ShipmentGroupID != nil {
		shipmentGroup, _ = database.GetShipmentGroup(*dc.ShipmentGroupID)
		siblingDCs, _ = database.GetShipmentGroupDCs(*dc.ShipmentGroupID)
		for _, sdc := range siblingDCs {
			if sdc.DCType == "official" {
				officialCount++
				if sdc.ID == dc.ID {
					dcPosition = officialCount
				}
			}
		}
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

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := deliverychallan.OfficialDetail(
		user,
		project,
		allProjects,
		dc,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
		lineItems,
		shipToAddress,
		billToAddress,
		shipmentGroup,
		siblingDCs,
		dcPosition,
		officialCount,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Official DC", sidebar, topbar, flashMessage, flashType, pageContent))
}

// ShowOfficialDCPrintView renders a print-ready view for an Official DC.
func ShowOfficialDCPrintView(c echo.Context) error {
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
		slog.Error("Error fetching project for print view", slog.Int("project_id", projectID), slog.String("error", err.Error()))
		return c.Redirect(http.StatusFound, "/projects")
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		if err != nil {
			slog.Error("Error fetching DC for print view", slog.Int("dc_id", dcID), slog.String("error", err.Error()))
		}
		auth.SetFlash(c.Request(), "error", "DC not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	lineItemsVal, _ := database.GetLineItemsByDCID(dcID)

	// Load serial numbers for each line item
	for i := range lineItemsVal {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItemsVal[i].ID)
		lineItemsVal[i].SerialNumbers = serials
	}

	lineItems := lineItemsToPointers(lineItemsVal)

	// Get addresses
	var shipToAddress *models.Address
	if dc.ShipToAddressID > 0 {
		shipToAddress, _ = database.GetAddress(dc.ShipToAddressID)
	}
	var billToAddress *models.Address
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}

	// Get company settings
	company, _ := database.GetCompanySettings()

	return components.RenderOK(c, deliverychallan.OfficialPrint(
		project,
		dc,
		lineItems,
		shipToAddress,
		billToAddress,
		company,
	))
}
