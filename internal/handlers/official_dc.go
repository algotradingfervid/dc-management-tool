package handlers

import (
	"fmt"
	"log"
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

// ShowCreateOfficialDC shows the Official DC creation form, pre-populated from a template.
func ShowCreateOfficialDC(c *gin.Context) {
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

	// Peek at next DC number without incrementing (only increment on actual creation)
	dcNumber, err := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)
	if err != nil {
		log.Printf("Error peeking DC number: %v", err)
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

	today := time.Now().Format("2006-01-02")

	purpose := ""
	if tmpl != nil {
		purpose = tmpl.Purpose
	}

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Create Official DC", URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/official_create.html", gin.H{
		"user":            user,
		"currentPath":     c.Request.URL.Path,
		"breadcrumbs":     breadcrumbs,
		"project":         project,
		"template":        tmpl,
		"templates":       templates,
		"products":        products,
		"shipToAddresses": shipToAddresses,
		"billToAddresses": billToAddresses,
		"billToColumns":   getColumnNames(billToConfig),
		"shipToColumns":   getColumnNames(shipToConfig),
		"dcNumber":        dcNumber,
		"challanDate":     today,
		"purpose":         purpose,
		"activeTab":       "templates",
		"errors":          map[string]string{},
		"csrfToken":       csrf.Token(c.Request),
		"csrfField":       csrf.TemplateField(c.Request),
	})
}

// CreateOfficialDC handles the Official DC form submission.
func CreateOfficialDC(c *gin.Context) {
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
	_ = strings.TrimSpace(c.PostForm("notes")) // Notes not stored for Official DCs (no transit_details)
	purpose := strings.TrimSpace(c.PostForm("purpose"))
	templateIDStr := c.PostForm("template_id")

	_ = purpose

	errors := make(map[string]string)

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

	// Parse line items (no pricing for Official DCs)
	var lineItems []models.DCLineItem
	var serialNumbersByLine [][]string

	for i := 0; ; i++ {
		pidStr := c.PostForm(fmt.Sprintf("line_items[%d].product_id", i))
		if pidStr == "" {
			break
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		serialsRaw := c.PostForm(fmt.Sprintf("line_items[%d].serial_numbers", i))
		serials := parseSerialNumbers(serialsRaw)

		quantity := len(serials)

		lineItems = append(lineItems, models.DCLineItem{
			ProductID:     pid,
			Quantity:      quantity,
			Rate:          0,
			TaxPercentage: 0,
			TaxableAmount: 0,
			TaxAmount:     0,
			TotalAmount:   0,
		})
		serialNumbersByLine = append(serialNumbersByLine, serials)
	}

	if len(lineItems) == 0 {
		errors["line_items"] = "At least one product line with serial numbers is required"
	}

	for i, li := range lineItems {
		if li.Quantity == 0 {
			errors[fmt.Sprintf("line_items[%d]", i)] = fmt.Sprintf("Product line %d must have at least one serial number", i+1)
		}
	}

	if len(errors) > 0 {
		peekNum, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)
		renderOfficialCreateFormWithErrors(c, project, templateID, peekNum, challanDate, errors)
		return
	}

	// Generate the real DC number (increments sequence) only at actual creation time
	dcNumber, err := services.GenerateDCNumber(database.DB, projectID, services.DCTypeOfficial)
	if err != nil {
		log.Printf("Error generating DC number: %v", err)
		errors["general"] = "Failed to generate DC number"
		peekNum, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)
		renderOfficialCreateFormWithErrors(c, project, templateID, peekNum, challanDate, errors)
		return
	}

	dc := &models.DeliveryChallan{
		ProjectID:       projectID,
		DCNumber:        dcNumber,
		DCType:          "official",
		Status:          "draft",
		TemplateID:      templateID,
		BillToAddressID: billToID,
		ShipToAddressID: shipToID,
		ChallanDate:     &challanDate,
		CreatedBy:       user.ID,
	}

	// No transit details for Official DCs
	if err := database.CreateDeliveryChallan(dc, nil, lineItems, serialNumbersByLine); err != nil {
		log.Printf("Error creating official DC: %v", err)

		if strings.Contains(err.Error(), "UNIQUE constraint failed") && strings.Contains(err.Error(), "serial_number") {
			errors["serial_numbers"] = "One or more serial numbers are already in use"
		} else if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			errors["dc_number"] = "DC number already exists"
		} else {
			errors["general"] = "Failed to create Official DC: " + err.Error()
		}
		peekNum, _ := services.PeekNextDCNumber(database.DB, projectID, services.DCTypeOfficial)
		renderOfficialCreateFormWithErrors(c, project, templateID, peekNum, challanDate, errors)
		return
	}

	auth.SetFlash(c.Request, "success", fmt.Sprintf("Official DC %s created as draft", dc.DCNumber))
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dcs/%d", projectID, dc.ID))
}

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

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: dc.DCNumber, URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/official_detail.html", gin.H{
		"user":          user,
		"currentPath":   c.Request.URL.Path,
		"breadcrumbs":   breadcrumbs,
		"project":       project,
		"dc":            dc,
		"lineItems":     lineItems,
		"shipToAddress": shipToAddress,
		"billToAddress": billToAddress,
		"activeTab":     "templates",
		"flashType":     flashType,
		"flashMessage":  flashMessage,
		"csrfToken":     csrf.Token(c.Request),
		"csrfField":     csrf.TemplateField(c.Request),
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
		"dc":            dc,
		"lineItems":     lineItems,
		"totalQty":      totalQty,
		"shipToAddress": shipToAddress,
		"billToAddress": billToAddress,
		"company":       company,
		"activeTab":     "templates",
	})
}

func renderOfficialCreateFormWithErrors(c *gin.Context, project *models.Project, templateID *int, dcNumber, challanDate string, errors map[string]string) {
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
		helpers.Breadcrumb{Title: "Create Official DC", URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/official_create.html", gin.H{
		"user":            user,
		"currentPath":     c.Request.URL.Path,
		"breadcrumbs":     breadcrumbs,
		"project":         project,
		"template":        tmpl,
		"templates":       templates,
		"products":        products,
		"shipToAddresses": shipToAddresses,
		"billToAddresses": billToAddresses,
		"billToColumns":   getColumnNames(billToConfig),
		"shipToColumns":   getColumnNames(shipToConfig),
		"dcNumber":        dcNumber,
		"challanDate":     challanDate,
		"purpose":         purpose,
		"activeTab":       "templates",
		"errors":          errors,
		"csrfToken":       csrf.Token(c.Request),
		"csrfField":       csrf.TemplateField(c.Request),
	})
}
