package handlers

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"

	"github.com/narendhupati/dc-management-tool/components/layouts"
	pagetransfer "github.com/narendhupati/dc-management-tool/components/pages/transfer_dcs"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// validateTransferDCIssue checks if a Transfer DC can be issued based on its current status.
func validateTransferDCIssue(currentStatus string) error {
	if currentStatus != models.DCStatusDraft {
		return fmt.Errorf("only draft Transfer DCs can be issued")
	}
	return nil
}

// validateTransferDCDelete checks if a Transfer DC can be deleted based on its current status.
func validateTransferDCDelete(currentStatus string) error {
	if currentStatus != models.DCStatusDraft {
		return fmt.Errorf("only draft Transfer DCs can be deleted")
	}
	return nil
}

// computeTransferDCStatus determines the correct status based on split progress.
// This is used after split creation/deletion to automatically transition between
// issued ↔ splitting ↔ split.
func computeTransferDCStatus(currentStatus string, numDestinations, numSplit int) string {
	// Draft status is never auto-transitioned
	if currentStatus == models.DCStatusDraft {
		return models.DCStatusDraft
	}

	if numSplit == 0 {
		return models.DCStatusIssued
	}
	if numSplit >= numDestinations {
		return models.DCStatusSplit
	}
	return models.DCStatusSplitting
}

// transferDCStatusBadgeClass returns Tailwind CSS classes for a status badge.
func transferDCStatusBadgeClass(status string) string {
	switch status {
	case models.DCStatusDraft:
		return "bg-gray-100 text-gray-800"
	case models.DCStatusIssued:
		return "bg-blue-100 text-blue-800"
	case models.DCStatusSplitting:
		return "bg-orange-100 text-orange-800"
	case models.DCStatusSplit:
		return "bg-green-100 text-green-800"
	default:
		return "bg-gray-100 text-gray-800"
	}
}

// ============================================================
// HTTP Handlers
// ============================================================

// ShowTransferDCDetail renders the full Transfer DC detail page.
func ShowTransferDCDetail(c echo.Context) error {
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

	// Get Transfer DC record
	tdc, err := database.GetTransferDCByDCID(dcID)
	if err != nil {
		slog.Error("Error fetching Transfer DC", slog.Int("dc_id", dcID), slog.String("error", err.Error()))
		auth.SetFlash(c.Request(), "error", "Transfer DC data not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	// Get destinations with quantities and full address objects
	destinations, _ := database.GetTransferDCDestinations(tdc.ID)
	destIDs := make([]int, len(destinations))
	for i, d := range destinations {
		destIDs[i] = d.ID
	}
	destQuantities, _ := database.GetQuantitiesForDestinations(destIDs)
	for _, d := range destinations {
		if qtys, ok := destQuantities[d.ID]; ok {
			d.Quantities = qtys
		}
		// Fetch full address object for proper formatting
		if d.ShipToAddressID > 0 {
			d.Address, _ = database.GetAddress(d.ShipToAddressID)
		}
	}

	// Get hub address
	var hubAddress *models.Address
	if tdc.HubAddressID > 0 {
		hubAddress, _ = database.GetAddress(tdc.HubAddressID)
	}

	// Get splits and populate CanDelete
	splits, _ := database.GetSplitsByTransferDCID(tdc.ID)
	for _, s := range splits {
		canDel, _ := database.CanDeleteSplit(s.ID)
		s.CanDelete = canDel
	}

	// Get summary
	summary, _ := database.GetTransferDCSummary(tdc.ID)

	flashType, flashMessage := auth.PopFlash(c.Request())
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.Detail(
		user,
		project,
		allProjects,
		dc,
		tdc,
		hubAddress,
		destinations,
		splits,
		summary,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Transfer DC", sidebar, topbar, flashMessage, flashType, pageContent))
}

// IssueTransferDC transitions a Transfer DC from draft → issued.
func IssueTransferDC(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid DC ID"})
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "DC not found"})
	}

	if err := validateTransferDCIssue(dc.Status); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	if err := database.IssueDC(dcID, user.ID); err != nil {
		slog.Error("Error issuing Transfer DC", slog.Int("dc_id", dcID), slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to issue Transfer DC"})
	}

	auth.SetFlash(c.Request(), "success", "Transfer DC issued successfully")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"message":  "Transfer DC issued successfully",
		"redirect": fmt.Sprintf("/projects/%d/dcs/%d", projectID, dcID),
	})
}

// DeleteTransferDCHandler deletes a draft Transfer DC and its parent delivery challan.
func DeleteTransferDCHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid DC ID"})
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "DC not found"})
	}

	if err := validateTransferDCDelete(dc.Status); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	// Delete the Transfer DC record (cascade handles destinations, quantities)
	tdc, err := database.GetTransferDCByDCID(dcID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Transfer DC data not found"})
	}

	if err := database.DeleteTransferDC(tdc.ID); err != nil {
		slog.Error("Error deleting Transfer DC", slog.Int("tdc_id", tdc.ID), slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to delete Transfer DC"})
	}

	// Delete the parent delivery challan
	if err := database.DeleteDC(dcID); err != nil {
		slog.Error("Error deleting parent DC", slog.Int("dc_id", dcID), slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to delete DC record"})
	}

	auth.SetFlash(c.Request(), "success", "Transfer DC deleted successfully")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"message":  "Transfer DC deleted successfully",
		"redirect": fmt.Sprintf("/projects/%d/transfer-dcs", projectID),
	})
}

// DeleteSplitHandler undoes a split by deleting the child shipment group and returning
// destinations to the unsplit pool.
func DeleteSplitHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid Transfer DC ID"})
	}

	splitID, err := strconv.Atoi(c.Param("splitid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid split ID"})
	}

	// Get the Transfer DC to find parent DC ID for redirect
	tdc, err := database.GetTransferDC(tdcID)
	if err != nil || tdc.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Transfer DC not found"})
	}

	err = services.DeleteSplitShipment(database.DB, splitID)
	if err != nil {
		slog.Error("Error deleting split", slog.Int("split_id", splitID), slog.String("error", err.Error()))
		auth.SetFlash(c.Request(), "error", "Failed to undo split: "+err.Error())
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dcs/%d", projectID, tdc.DCID))
	}

	auth.SetFlash(c.Request(), "success", "Split undone. Destinations returned to pool.")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dcs/%d", projectID, tdc.DCID))
}

// ListTransferDCs shows all Transfer DCs for the current project.
func ListTransferDCs(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	// Parse filters
	status := c.QueryParam("status")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 20

	tdcs, total, err := database.ListTransferDCsByProject(projectID, status, page, pageSize)
	if err != nil {
		slog.Error("Error listing Transfer DCs", slog.Int("project_id", projectID), slog.String("error", err.Error()))
	}

	totalPages := (total + pageSize - 1) / pageSize

	flashType, flashMessage := auth.PopFlash(c.Request())
	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pagetransfer.List(
		user,
		project,
		allProjects,
		tdcs,
		status,
		page,
		totalPages,
		total,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Transfer DCs", sidebar, topbar, flashMessage, flashType, pageContent))
}

// ShowTransferDCPrintView renders a browser print view for a Transfer DC.
func ShowTransferDCPrintView(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	tdcID, err := strconv.Atoi(c.Param("tdcid"))
	if err != nil {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", projectID))
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	tdc, err := database.GetTransferDC(tdcID)
	if err != nil {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs", projectID))
	}

	dc, err := database.GetDeliveryChallanByID(tdc.DCID)
	if err != nil || dc.ProjectID != projectID {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transfer-dcs", projectID))
	}

	lineItems, _ := database.GetLineItemsByDCID(tdc.DCID)
	for i := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItems[i].ID)
		lineItems[i].SerialNumbers = serials
	}

	var totalTaxable, totalTax, grandTotal float64
	var totalQty int
	for _, li := range lineItems { //nolint:gocritic
		totalTaxable += li.TaxableAmount
		totalTax += li.TaxAmount
		grandTotal += li.TotalAmount
		totalQty += li.Quantity
	}
	roundedTotal := math.Round(grandTotal)
	halfTax := totalTax / 2.0

	// Fetch addresses
	var hubAddress, billFromAddress, dispatchFromAddress, billToAddress *models.Address
	if tdc.HubAddressID > 0 {
		hubAddress, _ = database.GetAddress(tdc.HubAddressID)
	}
	if dc.BillFromAddressID != nil && *dc.BillFromAddressID > 0 {
		billFromAddress, _ = database.GetAddress(*dc.BillFromAddressID)
	}
	if dc.DispatchFromAddressID != nil && *dc.DispatchFromAddressID > 0 {
		dispatchFromAddress, _ = database.GetAddress(*dc.DispatchFromAddressID)
	}
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}

	// Build destinations
	var destinations []services.TransferDCPDFDestination
	var products []services.TransferDCPDFProduct
	dests, _ := database.GetTransferDCDestinations(tdc.ID)
	destIDs := make([]int, len(dests))
	for i, d := range dests {
		destIDs[i] = d.ID
	}
	destQuantities, _ := database.GetQuantitiesForDestinations(destIDs)
	for _, d := range dests {
		qtyMap := make(map[int]int)
		if qtys, ok := destQuantities[d.ID]; ok {
			for _, q := range qtys {
				qtyMap[q.ProductID] = q.Quantity
			}
		}
		// Fetch full address for proper display name and PDF filtering
		destName := d.AddressName
		var fullAddr *models.Address
		if addr, err := database.GetAddress(d.ShipToAddressID); err == nil && addr != nil {
			destName = addr.DisplayName()
			fullAddr = addr
		}
		destinations = append(destinations, services.TransferDCPDFDestination{
			Name:       destName,
			FullAddr:   fullAddr,
			Quantities: qtyMap,
		})
	}
	for _, li := range lineItems {
		products = append(products, services.TransferDCPDFProduct{
			ID:   li.ProductID,
			Name: li.ItemName,
		})
	}

	amountInWords := helpers.NumberToIndianWords(roundedTotal)

	// Fetch address configs
	billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
	dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")
	billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")

	printData := pagetransfer.PrintData{
		Project:             project,
		DC:                  dc,
		TransferDC:          tdc,
		LineItems:           lineItems,
		HubAddress:          hubAddress,
		BillFromAddress:     billFromAddress,
		DispatchFromAddress: dispatchFromAddress,
		BillToAddress:       billToAddress,
		Destinations:        destinations,
		Products:            products,
		TotalTaxable:        totalTaxable,
		TotalTax:            totalTax,
		GrandTotal:          grandTotal,
		RoundedTotal:        roundedTotal,
		HalfTax:             halfTax,
		TotalQty:            totalQty,
		AmountInWords:       amountInWords,
		BillFromConfig:      billFromConfig,
		DispatchFromConfig:  dispatchFromConfig,
		BillToConfig:        billToConfig,
	}

	return components.RenderOK(c, pagetransfer.TransferPrint(printData))
}
