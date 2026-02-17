package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ShowDCDetail dispatches to the correct detail view based on dc_type.
func ShowDCDetail(c *gin.Context) {
	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		projectID, _ := strconv.Atoi(c.Param("id"))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
		return
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil {
		projectID, _ := strconv.Atoi(c.Param("id"))
		auth.SetFlash(c.Request, "error", "DC not found")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
		return
	}

	if dc.DCType == "official" {
		ShowOfficialDCDetail(c)
		return
	}
	showTransitDCDetail(c)
}

// showTransitDCDetail shows a Transit DC's details.
func showTransitDCDetail(c *gin.Context) {
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

	c.HTML(http.StatusOK, "delivery_challans/detail.html", gin.H{
		"user":                user,
		"currentPath":         c.Request.URL.Path,
		"breadcrumbs":         breadcrumbs,
		"project":             project,
		"currentProject":      project,
		"dc":                  dc,
		"transitDetails":      transitDetails,
		"lineItems":           lineItems,
		"totalTaxable":        totalTaxable,
		"totalTax":            totalTax,
		"grandTotal":          grandTotal,
		"roundedTotal":        roundedTotal,
		"roundOff":            roundOff,
		"shipToAddress":       shipToAddress,
		"billToAddress":       billToAddress,
		"billFromAddress":     billFromAddress,
		"dispatchFromAddress": dispatchFromAddress,
		"shipmentGroup":       shipmentGroup,
		"siblingDCs":          siblingDCs,
		"activeTab":           "templates",
		"flashType":           flashType,
		"flashMessage":        flashMessage,
		"csrfToken":           csrf.Token(c.Request),
		"csrfField":           csrf.TemplateField(c.Request),
	})
}

// LoadTemplateProducts is an HTMX endpoint that returns product line items for a template.
func LoadTemplateProducts(c *gin.Context) {
	templateIDStr := c.Param("tid")
	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid template ID")
		return
	}

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		log.Printf("Error fetching template products: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load products")
		return
	}

	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil {
		c.String(http.StatusNotFound, "Template not found")
		return
	}

	c.HTML(http.StatusOK, "htmx/delivery_challans/product-lines.html", gin.H{
		"products": products,
		"purpose":  tmpl.Purpose,
	})
}

// ShowTransitDCPrintView renders a print-ready view for a Transit DC.
func ShowTransitDCPrintView(c *gin.Context) {
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

	transitDetails, _ := database.GetTransitDetailsByDCID(dcID)
	lineItems, _ := database.GetLineItemsByDCID(dcID)

	// Load serial numbers for each line item
	for i := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItems[i].ID)
		lineItems[i].SerialNumbers = serials
	}

	// Calculate totals
	var totalTaxable, totalTax, grandTotal float64
	var totalQty int
	for _, li := range lineItems {
		totalTaxable += li.TaxableAmount
		totalTax += li.TaxAmount
		grandTotal += li.TotalAmount
		totalQty += li.Quantity
	}
	roundedTotal := math.Round(grandTotal)
	roundOff := roundedTotal - grandTotal

	// Determine tax split (CGST/SGST vs IGST)
	// For now we assume CGST/SGST (same state), can be enhanced later
	halfTax := totalTax / 2.0

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

	// Amount in words
	amountInWords := helpers.NumberToIndianWords(roundedTotal)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: dc.DCNumber, URL: fmt.Sprintf("/projects/%d/dcs/%d", projectID, dcID)},
		helpers.Breadcrumb{Title: "Print View", URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/transit_print.html", gin.H{
		"user":           user,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"project":        project,
		"currentProject":  project,
		"dc":             dc,
		"transitDetails": transitDetails,
		"lineItems":      lineItems,
		"totalTaxable":   totalTaxable,
		"totalTax":       totalTax,
		"grandTotal":     grandTotal,
		"roundedTotal":   roundedTotal,
		"roundOff":       roundOff,
		"totalQty":       totalQty,
		"cgst":           math.Round(halfTax*100) / 100,
		"sgst":           math.Round(halfTax*100) / 100,
		"shipToAddress":  shipToAddress,
		"billToAddress":  billToAddress,
		"company":        company,
		"amountInWords":  amountInWords,
		"activeTab":      "templates",
	})
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
