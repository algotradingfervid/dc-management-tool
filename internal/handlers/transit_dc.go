package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// ShowCreateTransitDC shows the Transit DC creation form, pre-populated from a template.
func ShowCreateTransitDC(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Project not found")
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	// Get template ID from query param
	templateIDStr := c.Query("template_id")
	var tmpl *models.DCTemplate
	var products []*models.TemplateProductRow

	if templateIDStr != "" {
		templateID, err := strconv.Atoi(templateIDStr)
		if err == nil {
			tmpl, err = database.GetTemplateByID(templateID)
			if err != nil {
				log.Printf("Error fetching template %d: %v", templateID, err)
			} else {
				products, err = database.GetTemplateProducts(templateID)
				if err != nil {
					log.Printf("Error fetching template products: %v", err)
				}
			}
		}
	}

	// Get all templates for the project (for dropdown)
	templates, err := database.GetTemplatesByProjectID(projectID)
	if err != nil {
		log.Printf("Error fetching templates: %v", err)
		templates = []*models.DCTemplate{}
	}

	// Generate DC number
	dcNumber, err := services.GenerateDCNumber(database.DB, projectID, services.DCTypeTransit)
	if err != nil {
		log.Printf("Error generating DC number: %v", err)
		dcNumber = "Error generating number"
	}

	// Get ship-to and bill-to addresses
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")

	var shipToAddresses, billToAddresses []*models.Address
	if shipToConfig != nil {
		shipToAddresses, _ = database.GetAllAddressesByConfigID(shipToConfig.ID)
	}
	if billToConfig != nil {
		billToAddresses, _ = database.GetAllAddressesByConfigID(billToConfig.ID)
	}

	// Default date to today
	today := time.Now().Format("2006-01-02")

	purpose := ""
	if tmpl != nil {
		purpose = tmpl.Purpose
	}

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Create Transit DC", URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/create.html", gin.H{
		"user":            user,
		"currentPath":     c.Request.URL.Path,
		"breadcrumbs":     breadcrumbs,
		"project":         project,
		"template":        tmpl,
		"templates":       templates,
		"products":        products,
		"shipToAddresses": shipToAddresses,
		"billToAddresses": billToAddresses,
		"dcNumber":        dcNumber,
		"challanDate":     today,
		"purpose":         purpose,
		"activeTab":       "templates",
		"errors":          map[string]string{},
		"csrfToken":       csrf.Token(c.Request),
		"csrfField":       csrf.TemplateField(c.Request),
	})
}

// CreateTransitDC handles the Transit DC form submission.
func CreateTransitDC(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Project not found")
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	// Parse form
	challanDate := strings.TrimSpace(c.PostForm("challan_date"))
	shipToIDStr := c.PostForm("ship_to_address_id")
	billToIDStr := c.PostForm("bill_to_address_id")
	transporterName := strings.TrimSpace(c.PostForm("transporter_name"))
	vehicleNumber := strings.TrimSpace(c.PostForm("vehicle_number"))
	ewayBillNumber := strings.TrimSpace(c.PostForm("eway_bill_number"))
	notes := strings.TrimSpace(c.PostForm("notes"))
	taxType := c.PostForm("tax_type")
	dcNumber := strings.TrimSpace(c.PostForm("dc_number"))
	purpose := strings.TrimSpace(c.PostForm("purpose"))
	templateIDStr := c.PostForm("template_id")

	_ = purpose // purpose is informational, not stored separately

	errors := make(map[string]string)

	// Validate required fields
	if challanDate == "" {
		errors["challan_date"] = "DC date is required"
	}
	shipToID, err := strconv.Atoi(shipToIDStr)
	if err != nil || shipToID == 0 {
		errors["ship_to_address_id"] = "Ship To address is required"
	}

	var billToID *int
	if billToIDStr != "" {
		v, err := strconv.Atoi(billToIDStr)
		if err == nil && v > 0 {
			billToID = &v
		}
	}

	var templateID *int
	if templateIDStr != "" {
		v, err := strconv.Atoi(templateIDStr)
		if err == nil && v > 0 {
			templateID = &v
		}
	}

	if taxType == "" {
		taxType = "cgst_sgst"
	}

	// Parse line items
	var lineItems []models.DCLineItem
	var serialNumbersByLine [][]string

	// Count line items by checking for product_id fields
	for i := 0; ; i++ {
		pidStr := c.PostForm(fmt.Sprintf("line_items[%d].product_id", i))
		if pidStr == "" {
			break
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		rateStr := c.PostForm(fmt.Sprintf("line_items[%d].rate", i))
		rate, _ := strconv.ParseFloat(rateStr, 64)

		taxPctStr := c.PostForm(fmt.Sprintf("line_items[%d].tax_percentage", i))
		taxPct, _ := strconv.ParseFloat(taxPctStr, 64)

		serialsRaw := c.PostForm(fmt.Sprintf("line_items[%d].serial_numbers", i))
		serials := parseSerialNumbers(serialsRaw)

		quantity := len(serials)
		taxableAmount := rate * float64(quantity)
		taxAmount := taxableAmount * taxPct / 100.0
		totalAmount := taxableAmount + taxAmount

		lineItems = append(lineItems, models.DCLineItem{
			ProductID:     pid,
			Quantity:      quantity,
			Rate:          rate,
			TaxPercentage: taxPct,
			TaxableAmount: math.Round(taxableAmount*100) / 100,
			TaxAmount:     math.Round(taxAmount*100) / 100,
			TotalAmount:   math.Round(totalAmount*100) / 100,
		})
		serialNumbersByLine = append(serialNumbersByLine, serials)
	}

	if len(lineItems) == 0 {
		errors["line_items"] = "At least one product line with serial numbers is required"
	}

	// Check all line items have at least one serial
	for i, li := range lineItems {
		if li.Quantity == 0 {
			errors[fmt.Sprintf("line_items[%d]", i)] = fmt.Sprintf("Product line %d must have at least one serial number", i+1)
		}
	}

	if len(errors) > 0 {
		// Re-render form with errors
		renderCreateFormWithErrors(c, project, templateID, dcNumber, challanDate, errors)
		return
	}

	// Create the DC
	dc := &models.DeliveryChallan{
		ProjectID:       projectID,
		DCNumber:        dcNumber,
		DCType:          "transit",
		Status:          "draft",
		TemplateID:      templateID,
		BillToAddressID: billToID,
		ShipToAddressID: shipToID,
		ChallanDate:     &challanDate,
		CreatedBy:       user.ID,
	}

	transitDetails := &models.DCTransitDetails{
		TransporterName: transporterName,
		VehicleNumber:   vehicleNumber,
		EwayBillNumber:  ewayBillNumber,
		Notes:           notes,
	}

	if err := database.CreateDeliveryChallan(dc, transitDetails, lineItems, serialNumbersByLine); err != nil {
		log.Printf("Error creating transit DC: %v", err)

		if strings.Contains(err.Error(), "UNIQUE constraint failed") && strings.Contains(err.Error(), "serial_number") {
			errors["serial_numbers"] = "One or more serial numbers are already in use"
		} else if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			errors["dc_number"] = "DC number already exists"
		} else {
			errors["general"] = "Failed to create Transit DC: " + err.Error()
		}
		renderCreateFormWithErrors(c, project, templateID, dcNumber, challanDate, errors)
		return
	}

	auth.SetFlash(c.Request, "success", fmt.Sprintf("Transit DC %s created as draft", dc.DCNumber))
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dcs/%d", projectID, dc.ID))
}

// ShowTransitDCDetail shows a Transit DC's details.
func ShowTransitDCDetail(c *gin.Context) {
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

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: dc.DCNumber, URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/detail.html", gin.H{
		"user":           user,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"project":        project,
		"dc":             dc,
		"transitDetails": transitDetails,
		"lineItems":      lineItems,
		"totalTaxable":   totalTaxable,
		"totalTax":       totalTax,
		"grandTotal":     grandTotal,
		"roundedTotal":   roundedTotal,
		"roundOff":       roundOff,
		"shipToAddress":  shipToAddress,
		"billToAddress":  billToAddress,
		"activeTab":      "templates",
		"flashType":      flashType,
		"flashMessage":   flashMessage,
		"csrfToken":      csrf.Token(c.Request),
		"csrfField":      csrf.TemplateField(c.Request),
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

func renderCreateFormWithErrors(c *gin.Context, project *models.Project, templateID *int, dcNumber, challanDate string, errors map[string]string) {
	user := auth.GetCurrentUser(c)

	templates, _ := database.GetTemplatesByProjectID(project.ID)

	var tmpl *models.DCTemplate
	var products []*models.TemplateProductRow
	if templateID != nil {
		tmpl, _ = database.GetTemplateByID(*templateID)
		if tmpl != nil {
			products, _ = database.GetTemplateProducts(tmpl.ID)
		}
	}

	shipToConfig, _ := database.GetOrCreateAddressConfig(project.ID, "ship_to")
	billToConfig, _ := database.GetOrCreateAddressConfig(project.ID, "bill_to")

	var shipToAddresses, billToAddresses []*models.Address
	if shipToConfig != nil {
		shipToAddresses, _ = database.GetAllAddressesByConfigID(shipToConfig.ID)
	}
	if billToConfig != nil {
		billToAddresses, _ = database.GetAllAddressesByConfigID(billToConfig.ID)
	}

	purpose := ""
	if tmpl != nil {
		purpose = tmpl.Purpose
	}

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Create Transit DC", URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/create.html", gin.H{
		"user":            user,
		"currentPath":     c.Request.URL.Path,
		"breadcrumbs":     breadcrumbs,
		"project":         project,
		"template":        tmpl,
		"templates":       templates,
		"products":        products,
		"shipToAddresses": shipToAddresses,
		"billToAddresses": billToAddresses,
		"dcNumber":        dcNumber,
		"challanDate":     challanDate,
		"purpose":         purpose,
		"activeTab":       "templates",
		"errors":          errors,
		"csrfToken":       csrf.Token(c.Request),
		"csrfField":       csrf.TemplateField(c.Request),
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
