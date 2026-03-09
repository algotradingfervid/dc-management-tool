package handlers

import (
	"fmt"
	"log/slog"
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

// ─── Split wizard data carrier ──────────────────────────────────────────────

// splitWizardData carries form data across split wizard steps via hidden fields.
type splitWizardData struct {
	TransferDCID    int
	ParentDCID      int
	DestinationIDs  []string // string form values from checkboxes
	TransporterName string
	VehicleNumber   string
	EwayBillNumber  string
	DocketNumber    string
	Notes           string
	// Per-product serials: map[productID][]serialNumbers
	ProductSerials map[int][]string
}

// destinationIDsAsInts converts string destination IDs to ints, skipping invalid values.
func (d *splitWizardData) destinationIDsAsInts() []int {
	var result []int
	for _, s := range d.DestinationIDs {
		id, err := strconv.Atoi(s)
		if err == nil {
			result = append(result, id)
		}
	}
	return result
}

// ─── Validation helpers ─────────────────────────────────────────────────────

// validateSplitWizardAccess checks that a Transfer DC is in a valid status for the split wizard.
func validateSplitWizardAccess(dcStatus string) error {
	if dcStatus != "issued" && dcStatus != "splitting" {
		return fmt.Errorf("Transfer DC must be in 'issued' or 'splitting' status to create a split")
	}
	return nil
}

// validateSplitDestinationSelection checks that at least one destination is selected.
func validateSplitDestinationSelection(selectedIDs []string) error {
	if len(selectedIDs) == 0 {
		return fmt.Errorf("at least one destination must be selected")
	}
	return nil
}

// validateSplitTransporter checks that a transporter name is provided.
func validateSplitTransporter(transporterName string) error {
	if strings.TrimSpace(transporterName) == "" {
		return fmt.Errorf("transporter name is required")
	}
	return nil
}

// parseSplitSerials parses newline-separated serial numbers, trimming whitespace and skipping empty lines.
func parseSplitSerials(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var result []string
	for _, line := range strings.Split(raw, "\n") {
		sn := strings.TrimSpace(line)
		if sn != "" {
			result = append(result, sn)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ─── Form parsers ───────────────────────────────────────────────────────────

// parseSplitFormBase parses common hidden fields carried across split wizard steps.
func parseSplitFormBase(c echo.Context) (transferDCID, parentDCID int, destinationIDs []string) {
	transferDCID, _ = strconv.Atoi(c.FormValue("transfer_dc_id"))
	parentDCID, _ = strconv.Atoi(c.FormValue("parent_dc_id"))
	if parseErr := c.Request().ParseMultipartForm(32 << 20); parseErr != nil {
		_ = c.Request().ParseForm()
	}
	destinationIDs = c.Request().PostForm["destination_ids"]
	return
}

// parseSplitTransportForm parses transporter/vehicle fields from step 2.
func parseSplitTransportForm(c echo.Context) (transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes string) {
	transporterName = c.FormValue("transporter_name")
	vehicleNumber = c.FormValue("vehicle_number")
	ewayBillNumber = c.FormValue("eway_bill_number")
	docketNumber = c.FormValue("docket_number")
	notes = c.FormValue("notes")
	return
}

// ─── Shared data loaders ────────────────────────────────────────────────────

// loadSplitContext fetches the Transfer DC and validates access. Returns nil tdc + redirect on error.
func loadSplitContext(c echo.Context) (*models.TransferDC, *models.Project, *models.User, error) {
	user := auth.GetCurrentUser(c)
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return nil, nil, nil, c.Redirect(http.StatusFound, "/projects")
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return nil, nil, nil, c.Redirect(http.StatusFound, "/projects")
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Invalid Transfer DC ID")
		return nil, nil, nil, c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs", projectID))
	}

	tdc, err := database.GetTransferDC(tdcID)
	if err != nil || tdc.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Transfer DC not found")
		return nil, nil, nil, c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs", projectID))
	}

	return tdc, project, user, nil
}

// buildSplitProducts extracts unique product info from destination quantities.
func buildSplitProducts(destinations []*models.TransferDCDestination) []pagetransfer.SplitProductInfo {
	seen := make(map[int]bool)
	var products []pagetransfer.SplitProductInfo
	for _, dest := range destinations {
		for _, q := range dest.Quantities {
			if !seen[q.ProductID] {
				seen[q.ProductID] = true
				products = append(products, pagetransfer.SplitProductInfo{
					ID:   q.ProductID,
					Name: q.ProductName,
				})
			}
		}
	}
	return products
}

// computeSelectedQty calculates total quantity per product for selected destination IDs.
func computeSelectedQty(destIDs []int, destQuantities map[int][]models.TransferDCDestinationQty) map[int]int {
	result := make(map[int]int)
	for _, destID := range destIDs {
		for _, q := range destQuantities[destID] {
			result[q.ProductID] += q.Quantity
		}
	}
	return result
}

// buildSelectedSummary builds a text summary like "3 destinations (14 × Product A, 7 × Product B)".
func buildSelectedSummary(numDests int, expectedQty map[int]int, products []pagetransfer.SplitProductInfo) string {
	parts := []string{fmt.Sprintf("%d destination(s)", numDests)}
	for _, p := range products {
		if qty, ok := expectedQty[p.ID]; ok && qty > 0 {
			parts = append(parts, fmt.Sprintf("%d × %s", qty, p.Name))
		}
	}
	return strings.Join(parts, ", ")
}

// getAvailableSerials fetches parent serials per product, minus those already used in splits.
func getAvailableSerials(parentDCID int, transferDCID int) map[int][]string {
	lineItems, err := database.GetLineItemsByDCID(parentDCID)
	if err != nil {
		return nil
	}

	// Get parent serials per product
	parentSerials := make(map[int][]string)
	for _, li := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(li.ID)
		parentSerials[li.ProductID] = serials
	}

	// Get used serials from existing splits (via the split service's query pattern)
	usedSerials := make(map[string]bool)
	splits, _ := database.GetSplitsByTransferDCID(transferDCID)
	for _, s := range splits {
		// Get transit DC from the split's shipment group
		groupDCs, _ := database.GetDCsByShipmentGroup(s.ShipmentGroupID)
		for _, dc := range groupDCs {
			if dc.DCType == "transit" {
				splitLineItems, _ := database.GetLineItemsByDCID(dc.ID)
				for _, sli := range splitLineItems {
					splitSerials, _ := database.GetSerialNumbersByLineItemID(sli.ID)
					for _, sn := range splitSerials {
						usedSerials[sn] = true
					}
				}
			}
		}
	}

	// Filter available
	available := make(map[int][]string)
	for productID, serials := range parentSerials {
		for _, sn := range serials {
			if !usedSerials[sn] {
				available[productID] = append(available[productID], sn)
			}
		}
	}
	return available
}

// ─── Step 1: Select Destinations ────────────────────────────────────────────

// ShowSplitWizardStep1 renders the destination selection step.
func ShowSplitWizardStep1(c echo.Context) error {
	tdc, project, user, err := loadSplitContext(c)
	if err != nil {
		return err
	}

	// Validate status
	if err := validateSplitWizardAccess(tdc.DCStatus); err != nil {
		auth.SetFlash(c.Request(), "error", err.Error())
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dcs/%d", project.ID, tdc.DCID))
	}

	// Get unsplit destinations
	destinations, _ := database.GetUnsplitDestinations(tdc.ID)

	// Load quantities for these destinations
	destIDs := make([]int, len(destinations))
	for i, d := range destinations {
		destIDs[i] = d.ID
	}
	destQuantities, _ := database.GetQuantitiesForDestinations(destIDs)
	for _, d := range destinations {
		if qtys, ok := destQuantities[d.ID]; ok {
			d.Quantities = qtys
		}
	}

	products := buildSplitProducts(destinations)

	flashType, flashMessage := auth.PopFlash(c.Request())
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.SplitStep1(user, project, allProjects, tdc, destinations, products, flashType, flashMessage, csrf.Token(c.Request()))
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Split Wizard", sidebar, topbar, flashMessage, flashType, pageContent))
}

// ─── Step 2: Vehicle Details ────────────────────────────────────────────────

// SplitWizardStep2 processes step 1 (destinations) and renders step 2 (transporter/vehicle).
func SplitWizardStep2(c echo.Context) error {
	tdc, project, user, err := loadSplitContext(c)
	if err != nil {
		return err
	}

	_, _, destinationIDs := parseSplitFormBase(c)

	// Validate destination selection
	if err := validateSplitDestinationSelection(destinationIDs); err != nil {
		auth.SetFlash(c.Request(), "error", err.Error())
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/%d/split", project.ID, tdc.ID))
	}

	// Compute selected quantities for summary
	destIDInts := (&splitWizardData{DestinationIDs: destinationIDs}).destinationIDsAsInts()
	destQuantities, _ := database.GetQuantitiesForDestinations(destIDInts)

	// Build products from all unsplit destinations (for naming)
	unsplitDests, _ := database.GetUnsplitDestinations(tdc.ID)
	allDestIDs := make([]int, len(unsplitDests))
	for i, d := range unsplitDests {
		allDestIDs[i] = d.ID
	}
	allDestQtys, _ := database.GetQuantitiesForDestinations(allDestIDs)
	for _, d := range unsplitDests {
		if qtys, ok := allDestQtys[d.ID]; ok {
			d.Quantities = qtys
		}
	}
	products := buildSplitProducts(unsplitDests)
	expectedQty := computeSelectedQty(destIDInts, destQuantities)
	selectedSummary := buildSelectedSummary(len(destinationIDs), expectedQty, products)

	// Load transporters
	transporters, _ := database.GetTransportersByProjectID(project.ID, true)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pagetransfer.SplitStep2(user, project, allProjects, tdc, destinationIDs, selectedSummary, transporters, csrf.Token(c.Request()), nil, "", "", "", "", "")
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Split Wizard", sidebar, topbar, "", "", pageContent))
}

// ─── Step 3: Serial Numbers ─────────────────────────────────────────────────

// SplitWizardStep3 processes step 2 (transporter) and renders step 3 (serial entry).
func SplitWizardStep3(c echo.Context) error {
	tdc, project, user, err := loadSplitContext(c)
	if err != nil {
		return err
	}

	_, _, destinationIDs := parseSplitFormBase(c)
	transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes := parseSplitTransportForm(c)

	// Validate transporter
	if err := validateSplitTransporter(transporterName); err != nil {
		auth.SetFlash(c.Request(), "error", err.Error())
		// Re-render step 2 with error
		transporters, _ := database.GetTransportersByProjectID(project.ID, true)
		allProjects, _ := database.GetAccessibleProjects(user)
		valErrs := map[string]string{"transporter_name": err.Error()}
		pageContent := pagetransfer.SplitStep2(user, project, allProjects, tdc, destinationIDs, "", transporters, csrf.Token(c.Request()), valErrs, transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", err.Error())
		return components.RenderOK(c, layouts.MainWithContent("Split Wizard", sidebar, topbar, err.Error(), "error", pageContent))
	}

	// Build context for step 3
	destIDInts := (&splitWizardData{DestinationIDs: destinationIDs}).destinationIDsAsInts()
	destQuantities, _ := database.GetQuantitiesForDestinations(destIDInts)

	unsplitDests, _ := database.GetUnsplitDestinations(tdc.ID)
	allDestIDs := make([]int, len(unsplitDests))
	for i, d := range unsplitDests {
		allDestIDs[i] = d.ID
	}
	allDestQtys, _ := database.GetQuantitiesForDestinations(allDestIDs)
	for _, d := range unsplitDests {
		if qtys, ok := allDestQtys[d.ID]; ok {
			d.Quantities = qtys
		}
	}
	products := buildSplitProducts(unsplitDests)
	expectedQty := computeSelectedQty(destIDInts, destQuantities)
	selectedSummary := buildSelectedSummary(len(destinationIDs), expectedQty, products)

	availableSerials := getAvailableSerials(tdc.DCID, tdc.ID)

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pagetransfer.SplitStep3(user, project, allProjects, tdc, destinationIDs, selectedSummary, products, expectedQty, availableSerials, transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes, csrf.Token(c.Request()), nil, nil)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Split Wizard", sidebar, topbar, "", "", pageContent))
}

// ─── Step 4: Review ─────────────────────────────────────────────────────────

// SplitWizardStep4 processes step 3 (serials) and renders step 4 (review).
func SplitWizardStep4(c echo.Context) error {
	tdc, project, user, err := loadSplitContext(c)
	if err != nil {
		return err
	}

	_, _, destinationIDs := parseSplitFormBase(c)
	transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes := parseSplitTransportForm(c)

	// Build context
	destIDInts := (&splitWizardData{DestinationIDs: destinationIDs}).destinationIDsAsInts()
	destQuantities, _ := database.GetQuantitiesForDestinations(destIDInts)

	unsplitDests, _ := database.GetUnsplitDestinations(tdc.ID)
	allDestIDs := make([]int, len(unsplitDests))
	for i, d := range unsplitDests {
		allDestIDs[i] = d.ID
	}
	allDestQtys, _ := database.GetQuantitiesForDestinations(allDestIDs)
	for _, d := range unsplitDests {
		if qtys, ok := allDestQtys[d.ID]; ok {
			d.Quantities = qtys
		}
	}
	products := buildSplitProducts(unsplitDests)
	expectedQty := computeSelectedQty(destIDInts, destQuantities)

	// Parse serials per product
	productSerials := make(map[int][]string)
	serialErrors := make(map[int]string)
	for _, p := range products {
		raw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		serials := parseSplitSerials(raw)
		productSerials[p.ID] = serials

		// Validate serial count
		expected := expectedQty[p.ID]
		if len(serials) != expected {
			serialErrors[p.ID] = fmt.Sprintf("Expected %d serials, got %d", expected, len(serials))
			continue
		}

		// Check duplicates
		seen := make(map[string]bool)
		for _, sn := range serials {
			if seen[sn] {
				serialErrors[p.ID] = fmt.Sprintf("Duplicate serial: %s", sn)
				break
			}
			seen[sn] = true
		}
	}

	// If serial errors, re-render step 3
	if len(serialErrors) > 0 {
		selectedSummary := buildSelectedSummary(len(destinationIDs), expectedQty, products)
		availableSerials := getAvailableSerials(tdc.DCID, tdc.ID)

		allProjects, _ := database.GetAccessibleProjects(user)
		pageContent := pagetransfer.SplitStep3(user, project, allProjects, tdc, destinationIDs, selectedSummary, products, expectedQty, availableSerials, transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes, csrf.Token(c.Request()), serialErrors, productSerials)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", "Please fix the serial number errors below")
		return components.RenderOK(c, layouts.MainWithContent("Split Wizard", sidebar, topbar, "Please fix the serial number errors below", "error", pageContent))
	}

	// Load selected destinations for review display
	var selectedDests []*models.TransferDCDestination
	for _, dest := range unsplitDests {
		for _, did := range destIDInts {
			if dest.ID == did {
				selectedDests = append(selectedDests, dest)
				break
			}
		}
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pagetransfer.SplitStep4(user, project, allProjects, tdc, destinationIDs, selectedDests, products, expectedQty, transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes, productSerials, csrf.Token(c.Request()))
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Split Wizard", sidebar, topbar, "", "", pageContent))
}

// ─── Final submission ───────────────────────────────────────────────────────

// CreateSplitShipmentHandler processes the final split form and creates the child shipment.
func CreateSplitShipmentHandler(c echo.Context) error {
	tdc, project, user, err := loadSplitContext(c)
	if err != nil {
		return err
	}

	_, _, destinationIDs := parseSplitFormBase(c)
	transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes := parseSplitTransportForm(c)

	destIDInts := (&splitWizardData{DestinationIDs: destinationIDs}).destinationIDsAsInts()

	// Get products for serial parsing
	unsplitDests, _ := database.GetUnsplitDestinations(tdc.ID)
	allDestIDs := make([]int, len(unsplitDests))
	for i, d := range unsplitDests {
		allDestIDs[i] = d.ID
	}
	allDestQtys, _ := database.GetQuantitiesForDestinations(allDestIDs)
	for _, d := range unsplitDests {
		if qtys, ok := allDestQtys[d.ID]; ok {
			d.Quantities = qtys
		}
	}
	products := buildSplitProducts(unsplitDests)

	// Parse serials
	var productSerials []services.SplitProductSerials
	for _, p := range products {
		raw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		serials := parseSplitSerials(raw)
		productSerials = append(productSerials, services.SplitProductSerials{
			ProductID:     p.ID,
			SerialNumbers: serials,
		})
	}

	params := services.SplitShipmentParams{
		TransferDCID:    tdc.ID,
		ParentDCID:      tdc.DCID,
		ProjectID:       project.ID,
		DestinationIDs:  destIDInts,
		TransporterName: transporterName,
		VehicleNumber:   vehicleNumber,
		EwayBillNumber:  ewayBillNumber,
		DocketNumber:    docketNumber,
		Notes:           notes,
		ProductSerials:  productSerials,
		CreatedBy:       user.ID,
	}

	_, err = services.CreateSplitShipment(database.DB, params)
	if err != nil {
		slog.Error("Error creating split shipment", slog.String("error", err.Error()), slog.Int("transferDCID", tdc.ID))
		auth.SetFlash(c.Request(), "error", fmt.Sprintf("Failed to create split: %v", err))
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs/%d/split", project.ID, tdc.ID))
	}

	auth.SetFlash(c.Request(), "success", "Split created successfully")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dcs/%d", project.ID, tdc.DCID))
}

// ─── Back navigation handlers ───────────────────────────────────────────────

// SplitWizardBackToStep1 re-renders step 1 (ignoring carried forward data beyond destinations).
func SplitWizardBackToStep1(c echo.Context) error {
	return ShowSplitWizardStep1(c)
}

// SplitWizardBackToStep2 re-renders step 2 with previously entered data.
func SplitWizardBackToStep2(c echo.Context) error {
	tdc, project, user, err := loadSplitContext(c)
	if err != nil {
		return err
	}

	_, _, destinationIDs := parseSplitFormBase(c)
	transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes := parseSplitTransportForm(c)

	// Build summary
	destIDInts := (&splitWizardData{DestinationIDs: destinationIDs}).destinationIDsAsInts()
	destQuantities, _ := database.GetQuantitiesForDestinations(destIDInts)
	unsplitDests, _ := database.GetUnsplitDestinations(tdc.ID)
	allDestIDs := make([]int, len(unsplitDests))
	for i, d := range unsplitDests {
		allDestIDs[i] = d.ID
	}
	allDestQtys, _ := database.GetQuantitiesForDestinations(allDestIDs)
	for _, d := range unsplitDests {
		if qtys, ok := allDestQtys[d.ID]; ok {
			d.Quantities = qtys
		}
	}
	products := buildSplitProducts(unsplitDests)
	expectedQty := computeSelectedQty(destIDInts, destQuantities)
	selectedSummary := buildSelectedSummary(len(destinationIDs), expectedQty, products)

	transporters, _ := database.GetTransportersByProjectID(project.ID, true)
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.SplitStep2(user, project, allProjects, tdc, destinationIDs, selectedSummary, transporters, csrf.Token(c.Request()), nil, transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Split Wizard", sidebar, topbar, "", "", pageContent))
}

// SplitWizardBackToStep3 re-renders step 3 with previously entered data.
func SplitWizardBackToStep3(c echo.Context) error {
	tdc, project, user, err := loadSplitContext(c)
	if err != nil {
		return err
	}

	_, _, destinationIDs := parseSplitFormBase(c)
	transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes := parseSplitTransportForm(c)

	// Build context
	destIDInts := (&splitWizardData{DestinationIDs: destinationIDs}).destinationIDsAsInts()
	destQuantities, _ := database.GetQuantitiesForDestinations(destIDInts)
	unsplitDests, _ := database.GetUnsplitDestinations(tdc.ID)
	allDestIDs := make([]int, len(unsplitDests))
	for i, d := range unsplitDests {
		allDestIDs[i] = d.ID
	}
	allDestQtys, _ := database.GetQuantitiesForDestinations(allDestIDs)
	for _, d := range unsplitDests {
		if qtys, ok := allDestQtys[d.ID]; ok {
			d.Quantities = qtys
		}
	}
	products := buildSplitProducts(unsplitDests)
	expectedQty := computeSelectedQty(destIDInts, destQuantities)
	selectedSummary := buildSelectedSummary(len(destinationIDs), expectedQty, products)
	availableSerials := getAvailableSerials(tdc.DCID, tdc.ID)

	// Parse previously entered serials for prefill
	prefillSerials := make(map[int][]string)
	for _, p := range products {
		raw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		serials := parseSplitSerials(raw)
		if len(serials) > 0 {
			prefillSerials[p.ID] = serials
		}
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	pageContent := pagetransfer.SplitStep3(user, project, allProjects, tdc, destinationIDs, selectedSummary, products, expectedQty, availableSerials, transporterName, vehicleNumber, ewayBillNumber, docketNumber, notes, csrf.Token(c.Request()), nil, prefillSerials)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Split Wizard", sidebar, topbar, "", "", pageContent))
}
