package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ShowOfficialDCDetail shows an Official DC's details.
func ShowOfficialDCDetail(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
		return
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		auth.SetFlash(c.Request, "error", "DC not found")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
		return
	}

	lineItems, _ := database.GetLineItemsByDCID(dcID)

	// Load serial numbers for each line item
	for i := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItems[i].ID)
		lineItems[i].SerialNumbers = serials
	}

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

	flashType, flashMessage := auth.PopFlash(c.Request)

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
	breadcrumbs := helpers.BuildBreadcrumbs(breadcrumbItems...)

	c.HTML(http.StatusOK, "delivery_challans/official_detail.html", gin.H{
		"user":           user,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"project":        project,
		"currentProject": project,
		"dc":             dc,
		"lineItems":      lineItems,
		"shipToAddress":  shipToAddress,
		"billToAddress":  billToAddress,
		"shipmentGroup":  shipmentGroup,
		"siblingDCs":     siblingDCs,
		"dcPosition":     dcPosition,
		"officialCount":  officialCount,
		"activeTab":      "templates",
		"flashType":      flashType,
		"flashMessage":   flashMessage,
		"csrfToken":      csrf.Token(c.Request),
		"csrfField":      csrf.TemplateField(c.Request),
	})
}

// ShowOfficialDCPrintView renders a print-ready view for an Official DC.
func ShowOfficialDCPrintView(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
		return
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		auth.SetFlash(c.Request, "error", "DC not found")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
		return
	}

	lineItems, _ := database.GetLineItemsByDCID(dcID)

	// Load serial numbers for each line item
	for i := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItems[i].ID)
		lineItems[i].SerialNumbers = serials
	}

	// Calculate total quantity
	var totalQty int
	for _, li := range lineItems {
		totalQty += li.Quantity
	}

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

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: dc.DCNumber, URL: fmt.Sprintf("/projects/%d/dcs/%d", projectID, dcID)},
		helpers.Breadcrumb{Title: "Official Print View", URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/official_print.html", gin.H{
		"user":          user,
		"currentPath":   c.Request.URL.Path,
		"breadcrumbs":   breadcrumbs,
		"project":       project,
		"currentProject":  project,
		"dc":            dc,
		"lineItems":     lineItems,
		"totalQty":      totalQty,
		"shipToAddress": shipToAddress,
		"billToAddress": billToAddress,
		"company":       company,
		"activeTab":     "templates",
	})
}

