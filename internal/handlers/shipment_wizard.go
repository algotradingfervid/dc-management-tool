package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// ShowCreateShipmentWizard renders step 1 of the shipment wizard.
func ShowCreateShipmentWizard(c *gin.Context) {
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

	templates, err := database.GetTemplatesByProjectID(projectID)
	if err != nil {
		log.Printf("Error fetching templates: %v", err)
		templates = []*models.DCTemplate{}
	}

	transporters, err := database.GetTransportersByProjectID(projectID, true)
	if err != nil {
		log.Printf("Error fetching transporters: %v", err)
		transporters = []*models.Transporter{}
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "New Shipment", URL: ""},
	)

	c.HTML(http.StatusOK, "shipments/wizard_step1.html", gin.H{
		"user":           user,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"project":        project,
		"currentProject": project,
		"templates":      templates,
		"transporters":   transporters,
		"flashType":      flashType,
		"flashMessage":   flashMessage,
		"csrfField":      csrf.TemplateField(c.Request),
	})
}

// ShipmentWizardStep2 processes step 1 data and renders step 2 (address selection).
func ShipmentWizardStep2(c *gin.Context) {
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

	// Parse step 1 data
	templateID, _ := strconv.Atoi(c.PostForm("template_id"))
	numSets, _ := strconv.Atoi(c.PostForm("num_sets"))
	challanDate := c.PostForm("challan_date")
	transporterName := c.PostForm("transporter_name")
	vehicleNumber := c.PostForm("vehicle_number")
	ewayBillNumber := c.PostForm("eway_bill_number")
	docketNumber := c.PostForm("docket_number")
	taxType := c.PostForm("tax_type")
	reverseCharge := c.PostForm("reverse_charge")

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
		auth.SetFlash(c.Request, "error", "Please fix the errors below")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
	}

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load template products")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
	}

	// Validate template belongs to project
	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request, "error", "Template not found or doesn't belong to this project")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
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

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "New Shipment - Addresses", URL: ""},
	)

	c.HTML(http.StatusOK, "shipments/wizard_step2.html", gin.H{
		"user":                  user,
		"currentPath":           c.Request.URL.Path,
		"breadcrumbs":           breadcrumbs,
		"project":               project,
		"currentProject":        project,
		"products":              products,
		"template":              tmpl,
		"numSets":               numSets,
		"transitDCNumber":       transitDCNumber,
		"officialDCNumber":      officialDCNumber,
		"billFromAddresses":     billFromAddresses,
		"dispatchFromAddresses": dispatchFromAddresses,
		"billToAddresses":       billToAddresses,
		"shipToAddresses":       shipToAddresses,
		// Carry forward step 1 data
		"templateID":      templateID,
		"challanDate":     challanDate,
		"transporterName": transporterName,
		"vehicleNumber":   vehicleNumber,
		"ewayBillNumber":  ewayBillNumber,
		"docketNumber":    docketNumber,
		"taxType":         taxType,
		"reverseCharge":   reverseCharge,
		"csrfField":       csrf.TemplateField(c.Request),
	})
}

// ShipmentWizardStep3 processes step 2 (addresses) and renders step 3 (serial entry).
func ShipmentWizardStep3(c *gin.Context) {
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

	// Parse step 1 carry-forward data
	templateID, _ := strconv.Atoi(c.PostForm("template_id"))
	numSets, _ := strconv.Atoi(c.PostForm("num_sets"))
	challanDate := c.PostForm("challan_date")
	transporterName := c.PostForm("transporter_name")
	vehicleNumber := c.PostForm("vehicle_number")
	ewayBillNumber := c.PostForm("eway_bill_number")
	docketNumber := c.PostForm("docket_number")
	taxType := c.PostForm("tax_type")
	reverseCharge := c.PostForm("reverse_charge")

	// Parse step 2 data (addresses)
	billFromAddressID, _ := strconv.Atoi(c.PostForm("bill_from_address_id"))
	dispatchFromAddressID, _ := strconv.Atoi(c.PostForm("dispatch_from_address_id"))
	billToAddressID, _ := strconv.Atoi(c.PostForm("bill_to_address_id"))
	transitShipToAddrID, _ := strconv.Atoi(c.PostForm("transit_ship_to_address_id"))

	// Parse multiple ship-to address selections
	shipToIDStrs := c.PostFormArray("ship_to_address_ids")
	var shipToAddressIDs []int
	for _, s := range shipToIDStrs {
		id, err := strconv.Atoi(s)
		if err == nil && id > 0 {
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
		auth.SetFlash(c.Request, "error", strings.Join(msgs, ". "))

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
		flashType, flashMessage := auth.PopFlash(c.Request)

		breadcrumbs := helpers.BuildBreadcrumbs(
			helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
			helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
			helpers.Breadcrumb{Title: "New Shipment - Addresses", URL: ""},
		)

		c.HTML(http.StatusOK, "shipments/wizard_step2.html", gin.H{
			"user":                  user,
			"currentPath":           c.Request.URL.Path,
			"breadcrumbs":           breadcrumbs,
			"project":               project,
			"currentProject":        project,
			"template":              tmpl,
			"numSets":               numSets,
			"transitDCNumber":       transitDCNumber,
			"officialDCNumber":      officialDCNumber,
			"billFromAddresses":     billFromAddresses,
			"dispatchFromAddresses": dispatchFromAddresses,
			"billToAddresses":       billToAddresses,
			"shipToAddresses":       shipToAddresses,
			"templateID":            templateID,
			"challanDate":           challanDate,
			"transporterName":       transporterName,
			"vehicleNumber":         vehicleNumber,
			"ewayBillNumber":        ewayBillNumber,
			"docketNumber":          docketNumber,
			"taxType":               taxType,
			"reverseCharge":         reverseCharge,
			"flashType":             flashType,
			"flashMessage":          flashMessage,
			"csrfField":             csrf.TemplateField(c.Request),
		})
		return
	}

	// Load template products with quantities
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load template products")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
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

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "New Shipment - Serials", URL: ""},
	)

	c.HTML(http.StatusOK, "shipments/wizard_step3.html", gin.H{
		"user":                  user,
		"currentPath":           c.Request.URL.Path,
		"breadcrumbs":           breadcrumbs,
		"project":               project,
		"currentProject":        project,
		"products":              products,
		"numSets":               numSets,
		"shipToAddresses":       shipToAddresses,
		// Carry forward all previous data
		"templateID":            templateID,
		"challanDate":           challanDate,
		"transporterName":       transporterName,
		"vehicleNumber":         vehicleNumber,
		"ewayBillNumber":        ewayBillNumber,
		"docketNumber":          docketNumber,
		"taxType":               taxType,
		"reverseCharge":         reverseCharge,
		"billFromAddressID":     billFromAddressID,
		"dispatchFromAddressID": dispatchFromAddressID,
		"billToAddressID":       billToAddressID,
		"transitShipToAddrID":   transitShipToAddrID,
		"shipToAddressIDs":      shipToIDStrings,
		"csrfField":             csrf.TemplateField(c.Request),
	})
}

// ShipmentWizardStep4 processes step 3 (serials) and renders review page.
func ShipmentWizardStep4(c *gin.Context) {
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

	// Parse all carry-forward data
	templateID, _ := strconv.Atoi(c.PostForm("template_id"))
	numSets, _ := strconv.Atoi(c.PostForm("num_sets"))
	challanDate := c.PostForm("challan_date")
	transporterName := c.PostForm("transporter_name")
	vehicleNumber := c.PostForm("vehicle_number")
	ewayBillNumber := c.PostForm("eway_bill_number")
	docketNumber := c.PostForm("docket_number")
	taxType := c.PostForm("tax_type")
	reverseCharge := c.PostForm("reverse_charge")
	billFromAddressID, _ := strconv.Atoi(c.PostForm("bill_from_address_id"))
	dispatchFromAddressID, _ := strconv.Atoi(c.PostForm("dispatch_from_address_id"))
	billToAddressID, _ := strconv.Atoi(c.PostForm("bill_to_address_id"))
	transitShipToAddrID, _ := strconv.Atoi(c.PostForm("transit_ship_to_address_id"))

	shipToIDStrs := c.PostFormArray("ship_to_address_ids")
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
		auth.SetFlash(c.Request, "error", "Failed to load template products")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
	}

	// Parse serial numbers per product
	type productSerialData struct {
		ProductID   int
		AllSerials  []string
		Assignments map[int][]string // shipToAddrID -> serials
	}
	var serialData []productSerialData

	for _, p := range products {
		pd := productSerialData{
			ProductID:   p.ID,
			Assignments: make(map[int][]string),
		}

		// All serials for this product
		serialsRaw := c.PostForm(fmt.Sprintf("serials_%d", p.ID))
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
			assignRaw := c.PostForm(fmt.Sprintf("assign_%d_%d", p.ID, shipToID))
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
		auth.SetFlash(c.Request, "error", "Serial number validation failed")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
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

	// Build serialized serial data for hidden fields
	shipToIDStrings := make([]string, len(shipToAddressIDs))
	for i, id := range shipToAddressIDs {
		shipToIDStrings[i] = strconv.Itoa(id)
	}

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "New Shipment - Review", URL: ""},
	)

	c.HTML(http.StatusOK, "shipments/wizard_step4.html", gin.H{
		"user":                  user,
		"currentPath":           c.Request.URL.Path,
		"breadcrumbs":           breadcrumbs,
		"project":               project,
		"currentProject":        project,
		"products":              products,
		"template":              tmpl,
		"numSets":               numSets,
		"serialData":            serialData,
		"shipToAddresses":       shipToAddresses,
		// Carry forward all data
		"templateID":            templateID,
		"challanDate":           challanDate,
		"transporterName":       transporterName,
		"vehicleNumber":         vehicleNumber,
		"ewayBillNumber":        ewayBillNumber,
		"docketNumber":          docketNumber,
		"taxType":               taxType,
		"reverseCharge":         reverseCharge,
		"billFromAddressID":     billFromAddressID,
		"dispatchFromAddressID": dispatchFromAddressID,
		"billToAddressID":       billToAddressID,
		"transitShipToAddrID":   transitShipToAddrID,
		"shipToAddressIDs":      shipToIDStrings,
		"csrfField":             csrf.TemplateField(c.Request),
	})
}

// CreateShipment processes the final submission and creates all DCs.
func CreateShipment(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	// Re-parse all data from hidden fields
	templateID, _ := strconv.Atoi(c.PostForm("template_id"))
	numSets, _ := strconv.Atoi(c.PostForm("num_sets"))
	challanDate := c.PostForm("challan_date")
	transporterName := c.PostForm("transporter_name")
	vehicleNumber := c.PostForm("vehicle_number")
	ewayBillNumber := c.PostForm("eway_bill_number")
	docketNumber := c.PostForm("docket_number")
	taxType := c.PostForm("tax_type")
	reverseCharge := c.PostForm("reverse_charge")
	billFromAddressID, _ := strconv.Atoi(c.PostForm("bill_from_address_id"))
	dispatchFromAddressID, _ := strconv.Atoi(c.PostForm("dispatch_from_address_id"))
	billToAddressID, _ := strconv.Atoi(c.PostForm("bill_to_address_id"))
	transitShipToAddrID, _ := strconv.Atoi(c.PostForm("transit_ship_to_address_id"))

	shipToIDStrs := c.PostFormArray("ship_to_address_ids")
	var shipToAddressIDs []int
	for _, s := range shipToIDStrs {
		id, _ := strconv.Atoi(s)
		if id > 0 {
			shipToAddressIDs = append(shipToAddressIDs, id)
		}
	}

	// Re-validate
	if templateID == 0 || numSets < 1 || challanDate == "" {
		auth.SetFlash(c.Request, "error", "Invalid shipment data")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
	}
	if len(shipToAddressIDs) != numSets {
		auth.SetFlash(c.Request, "error", fmt.Sprintf("Expected %d ship-to addresses, got %d", numSets, len(shipToAddressIDs)))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
	}

	// Validate template belongs to project
	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request, "error", "Template not found or doesn't belong to this project")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
	}

	// Load template products
	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to load template products")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
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
		serialsRaw := c.PostForm(fmt.Sprintf("serials_%d", p.ID))
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
			assignRaw := c.PostForm(fmt.Sprintf("assign_%d_%d", p.ID, shipToID))
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
			auth.SetFlash(c.Request, "error", fmt.Sprintf("Product %s: expected %d serials, got %d", products[i].ItemName, expectedTotal, len(item.AllSerials)))
			c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
			return
		}

		// Check duplicates within same product
		seen := make(map[string]bool)
		for _, sn := range item.AllSerials {
			if seen[sn] {
				auth.SetFlash(c.Request, "error", fmt.Sprintf("Duplicate serial %s for product %s", sn, products[i].ItemName))
				c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
				return
			}
			seen[sn] = true
		}

		// Check assignments don't exceed qty_per_set
		for shipToID, assigned := range item.Assignments {
			if len(assigned) > products[i].DefaultQuantity {
				auth.SetFlash(c.Request, "error", fmt.Sprintf("Too many serials assigned for product %s to destination %d", products[i].ItemName, shipToID))
				c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
				return
			}
		}
	}

	// Check for duplicate serials in project
	for _, item := range lineItems {
		if len(item.AllSerials) > 0 {
			conflicts, err := database.CheckSerialsInProject(projectID, item.AllSerials, nil)
			if err != nil {
				log.Printf("Error checking serials: %v", err)
			}
			if len(conflicts) > 0 {
				auth.SetFlash(c.Request, "error", fmt.Sprintf("Serial %s already exists in DC %s", conflicts[0].SerialNumber, conflicts[0].DCNumber))
				c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
				return
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
		log.Printf("Error creating shipment: %v", err)
		auth.SetFlash(c.Request, "error", fmt.Sprintf("Failed to create shipment: %v", err))
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/new", projectID))
		return
	}

	auth.SetFlash(c.Request, "success", fmt.Sprintf("Shipment created successfully with %d DCs", 1+len(result.OfficialDCs)))
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/shipments/%d", projectID, result.GroupID))
}

// ShowShipmentGroup displays a shipment group detail page.
func ShowShipmentGroup(c *gin.Context) {
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

	groupID, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		auth.SetFlash(c.Request, "error", "Invalid shipment group ID")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dashboard", projectID))
		return
	}

	group, err := database.GetShipmentGroup(groupID)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Shipment group not found")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dashboard", projectID))
		return
	}

	// Get all DCs in this group
	dcs, err := database.GetShipmentGroupDCs(groupID)
	if err != nil {
		log.Printf("Error fetching shipment group DCs: %v", err)
		dcs = []*models.DeliveryChallan{}
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Shipment Group", URL: ""},
	)

	c.HTML(http.StatusOK, "shipments/group_detail.html", gin.H{
		"user":           user,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"project":        project,
		"currentProject": project,
		"group":          group,
		"dcs":            dcs,
		"flashType":      flashType,
		"flashMessage":   flashMessage,
		"csrfField":      csrf.TemplateField(c.Request),
	})
}

// ListShipmentGroups shows all shipment groups for a project.
func ListShipmentGroups(c *gin.Context) {
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

	groups, err := database.GetShipmentGroupsByProjectID(projectID)
	if err != nil {
		log.Printf("Error fetching shipment groups: %v", err)
		groups = []*models.ShipmentGroup{}
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
		helpers.Breadcrumb{Title: "Shipment Groups"},
	)

	c.HTML(http.StatusOK, "shipments/list.html", gin.H{
		"user":           user,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"project":        project,
		"currentProject": project,
		"groups":         groups,
		"flashType":      flashType,
		"flashMessage":   flashMessage,
		"csrfField":      csrf.TemplateField(c.Request),
	})
}

// IssueShipmentGroup issues all draft DCs in a shipment group.
func IssueShipmentGroup(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	groupID, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	group, err := database.GetShipmentGroup(groupID)
	if err != nil || group.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shipment group not found"})
		return
	}

	if group.Status == "issued" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Shipment group is already issued"})
		return
	}

	count, err := database.IssueAllDCsInGroup(groupID, user.ID)
	if err != nil {
		log.Printf("Error issuing DCs in group %d: %v", groupID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue DCs"})
		return
	}

	// Update group status
	if err := database.UpdateShipmentGroupStatus(groupID, "issued"); err != nil {
		log.Printf("Error updating group status: %v", err)
	}

	auth.SetFlash(c.Request, "success", fmt.Sprintf("Successfully issued %d DCs", count))
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  fmt.Sprintf("Successfully issued %d DCs", count),
		"redirect": fmt.Sprintf("/projects/%d/shipments/%d", projectID, groupID),
	})
}
