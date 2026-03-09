package handlers

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// ExportDCPDF generates and serves a PDF for a DC using the native Go PDF builder.
func ExportDCPDF(c echo.Context) error {
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

	// Render print template to HTML and convert to PDF via headless Chrome
	pdfData, err := generatePDFForDC(projectID, dcID, dc)
	if err != nil {
		slog.Error("error generating PDF for DC", slog.String("error", err.Error()), slog.Int("dcID", dcID), slog.Int("projectID", projectID))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to generate PDF"})
	}

	filename := services.SanitizeDCFilename(dc.DCNumber) + ".pdf"

	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.Blob(http.StatusOK, "application/pdf", pdfData)
}

// generatePDFForDC fetches DC data and generates a PDF using the native Go PDF builder.
func generatePDFForDC(projectID, dcID int, dc *models.DeliveryChallan) ([]byte, error) {
	switch dc.DCType {
	case "official":
		return buildOfficialPDF(projectID, dcID, dc)
	case "transfer":
		return buildTransferPDF(projectID, dcID, dc)
	default:
		return buildTransitPDF(projectID, dcID, dc)
	}
}

func buildTransitPDF(projectID, dcID int, dc *models.DeliveryChallan) ([]byte, error) {
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return nil, err
	}

	transitDetails, _ := database.GetTransitDetailsByDCID(dcID)
	lineItems, _ := database.GetLineItemsByDCID(dcID)
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
	roundOff := roundedTotal - grandTotal
	halfTax := totalTax / 2.0

	var shipToAddress, billToAddress *models.Address
	if dc.ShipToAddressID > 0 {
		shipToAddress, _ = database.GetAddress(dc.ShipToAddressID)
	}
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}

	// Fetch Bill From and Dispatch From addresses selected during DC creation
	var billFromAddress, dispatchFromAddress *models.Address
	if dc.BillFromAddressID != nil && *dc.BillFromAddressID > 0 {
		billFromAddress, _ = database.GetAddress(*dc.BillFromAddressID)
	}
	if dc.DispatchFromAddressID != nil && *dc.DispatchFromAddressID > 0 {
		dispatchFromAddress, _ = database.GetAddress(*dc.DispatchFromAddressID)
	}

	company, _ := database.GetCompanySettings()
	amountInWords := helpers.NumberToIndianWords(roundedTotal)

	// Look up parent Transfer DC number if this transit DC came from a split
	var transferDCNumber string
	if dc.ShipmentGroupID != nil {
		var tdcNum sql.NullString
		_ = database.DB.QueryRow(
			`SELECT dc2.dc_number FROM shipment_groups sg
			 INNER JOIN transfer_dcs t ON sg.transfer_dc_id = t.id
			 INNER JOIN delivery_challans dc2 ON t.dc_id = dc2.id
			 WHERE sg.id = ?`, *dc.ShipmentGroupID,
		).Scan(&tdcNum)
		if tdcNum.Valid {
			transferDCNumber = tdcNum.String
		}
	}

	// Fetch address configs for print column filtering
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")
	billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
	dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")

	return services.GenerateTransitDCPDF(&services.TransitDCPDFData{
		Project:            project,
		DC:                 dc,
		TransitDetails:     transitDetails,
		LineItems:          lineItems,
		Company:            company,
		ShipToAddress:      shipToAddress,
		BillToAddress:      billToAddress,
		BillFromAddress:    billFromAddress,
		DispatchFromAddress: dispatchFromAddress,
		ShipToConfig:       shipToConfig,
		BillToConfig:       billToConfig,
		BillFromConfig:     billFromConfig,
		DispatchFromConfig: dispatchFromConfig,
		TotalTaxable:       totalTaxable,
		TotalTax:           totalTax,
		GrandTotal:         grandTotal,
		RoundedTotal:       roundedTotal,
		RoundOff:           roundOff,
		HalfTax:            halfTax,
		TotalQty:           totalQty,
		AmountInWords:      amountInWords,
		TransferDCNumber:   transferDCNumber,
	})
}

func buildOfficialPDF(projectID, dcID int, dc *models.DeliveryChallan) ([]byte, error) {
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return nil, err
	}

	lineItems, _ := database.GetLineItemsByDCID(dcID)
	for i := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItems[i].ID)
		lineItems[i].SerialNumbers = serials
	}

	var totalQty int
	for _, li := range lineItems { //nolint:gocritic
		totalQty += li.Quantity
	}

	var shipToAddress, billToAddress, billFromAddress, dispatchFromAddress *models.Address
	if dc.ShipToAddressID > 0 {
		shipToAddress, _ = database.GetAddress(dc.ShipToAddressID)
	}
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}
	if dc.BillFromAddressID != nil && *dc.BillFromAddressID > 0 {
		billFromAddress, _ = database.GetAddress(*dc.BillFromAddressID)
	}
	if dc.DispatchFromAddressID != nil && *dc.DispatchFromAddressID > 0 {
		dispatchFromAddress, _ = database.GetAddress(*dc.DispatchFromAddressID)
	}
	company, _ := database.GetCompanySettings()

	// Fetch transit details and inherit addresses from parent TDC in the same shipment group
	var transitDetails *models.DCTransitDetails
	if dc.ShipmentGroupID != nil && *dc.ShipmentGroupID > 0 {
		groupDCs, _ := database.GetDCsByShipmentGroup(*dc.ShipmentGroupID)
		for _, groupDC := range groupDCs {
			if groupDC.DCType == "transit" {
				transitDetails, _ = database.GetTransitDetailsByDCID(groupDC.ID)
				// Fall back to TDC's addresses for ODCs created before address inheritance was added
				if billFromAddress == nil && groupDC.BillFromAddressID != nil && *groupDC.BillFromAddressID > 0 {
					billFromAddress, _ = database.GetAddress(*groupDC.BillFromAddressID)
				}
				if dispatchFromAddress == nil && groupDC.DispatchFromAddressID != nil && *groupDC.DispatchFromAddressID > 0 {
					dispatchFromAddress, _ = database.GetAddress(*groupDC.DispatchFromAddressID)
				}
				break
			}
		}
	}

	// Fetch address configs for print column filtering
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
	billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")
	billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
	dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")

	return services.GenerateOfficialDCPDF(&services.OfficialDCPDFData{
		Project:             project,
		DC:                  dc,
		TransitDetails:      transitDetails,
		LineItems:           lineItems,
		Company:             company,
		ShipToAddress:       shipToAddress,
		BillToAddress:       billToAddress,
		BillFromAddress:     billFromAddress,
		DispatchFromAddress: dispatchFromAddress,
		ShipToConfig:        shipToConfig,
		BillToConfig:        billToConfig,
		BillFromConfig:      billFromConfig,
		DispatchFromConfig:  dispatchFromConfig,
		TotalQty:            totalQty,
	})
}

// ExportDCExcel generates and serves an Excel file for a DC.
func ExportDCExcel(c echo.Context) error {
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

	project, _ := database.GetProjectByID(projectID)
	lineItems, _ := database.GetLineItemsByDCID(dcID)
	for i := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(lineItems[i].ID)
		lineItems[i].SerialNumbers = serials
	}

	var shipToAddress, billToAddress *models.Address
	if dc.ShipToAddressID > 0 {
		shipToAddress, _ = database.GetAddress(dc.ShipToAddressID)
	}
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}

	company, _ := database.GetCompanySettings()

	filename := services.SanitizeDCFilename(dc.DCNumber) + ".xlsx"

	if dc.DCType == "transfer" {
		return buildTransferExcel(c, projectID, dc, lineItems, project, company, shipToAddress, billToAddress, filename)
	}

	if dc.DCType == "official" {
		var totalQty int
		for _, li := range lineItems { //nolint:gocritic
			totalQty += li.Quantity
		}

		// Fetch Bill From and Dispatch From addresses
		var billFromAddress, dispatchFromAddress *models.Address
		if dc.BillFromAddressID != nil && *dc.BillFromAddressID > 0 {
			billFromAddress, _ = database.GetAddress(*dc.BillFromAddressID)
		}
		if dc.DispatchFromAddressID != nil && *dc.DispatchFromAddressID > 0 {
			dispatchFromAddress, _ = database.GetAddress(*dc.DispatchFromAddressID)
		}

		// Fetch transit details from parent TDC in shipment group
		var transitDetails *models.DCTransitDetails
		if dc.ShipmentGroupID != nil && *dc.ShipmentGroupID > 0 {
			groupDCs, _ := database.GetDCsByShipmentGroup(*dc.ShipmentGroupID)
			for _, groupDC := range groupDCs {
				if groupDC.DCType == "transit" {
					transitDetails, _ = database.GetTransitDetailsByDCID(groupDC.ID)
					if billFromAddress == nil && groupDC.BillFromAddressID != nil && *groupDC.BillFromAddressID > 0 {
						billFromAddress, _ = database.GetAddress(*groupDC.BillFromAddressID)
					}
					if dispatchFromAddress == nil && groupDC.DispatchFromAddressID != nil && *groupDC.DispatchFromAddressID > 0 {
						dispatchFromAddress, _ = database.GetAddress(*groupDC.DispatchFromAddressID)
					}
					break
				}
			}
		}

		// Fetch address configs for print column filtering
		shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
		billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")
		billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
		dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")

		excelFile, err := services.GenerateOfficialDCExcel(&services.OfficialDCExcelData{
			DC:                  dc,
			LineItems:           lineItems,
			Company:             company,
			Project:             project,
			ShipToAddress:       shipToAddress,
			BillToAddress:       billToAddress,
			BillFromAddress:     billFromAddress,
			DispatchFromAddress: dispatchFromAddress,
			TransitDetails:      transitDetails,
			ShipToConfig:        shipToConfig,
			BillToConfig:        billToConfig,
			BillFromConfig:      billFromConfig,
			DispatchFromConfig:  dispatchFromConfig,
			TotalQty:            totalQty,
		})
		if err != nil {
			slog.Error("error generating official DC Excel", slog.String("error", err.Error()), slog.Int("dcID", dcID), slog.Int("projectID", projectID))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to generate Excel"})
		}

		c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		if err := excelFile.Write(c.Response().Writer); err != nil {
			slog.Error("error writing official DC Excel response", slog.String("error", err.Error()), slog.Int("dcID", dcID))
		}
	} else {
		transitDetails, _ := database.GetTransitDetailsByDCID(dcID)
		totalTaxable, totalTax, grandTotal, roundedTotal, roundOff, _, _ := services.CalcTransitTotals(lineItems)
		halfTax := totalTax / 2.0
		amountInWords := helpers.NumberToIndianWords(roundedTotal)

		var totalQty int
		for _, li := range lineItems { //nolint:gocritic
			totalQty += li.Quantity
		}

		// Fetch Bill From and Dispatch From addresses
		var billFromAddress, dispatchFromAddress *models.Address
		if dc.BillFromAddressID != nil && *dc.BillFromAddressID > 0 {
			billFromAddress, _ = database.GetAddress(*dc.BillFromAddressID)
		}
		if dc.DispatchFromAddressID != nil && *dc.DispatchFromAddressID > 0 {
			dispatchFromAddress, _ = database.GetAddress(*dc.DispatchFromAddressID)
		}

		// Fetch address configs for print column filtering
		shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")
		billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")
		billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
		dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")

		excelFile, err := services.GenerateTransitDCExcel(&services.TransitDCExcelData{
			DC:                  dc,
			LineItems:           lineItems,
			Company:             company,
			Project:             project,
			ShipToAddress:       shipToAddress,
			BillToAddress:       billToAddress,
			BillFromAddress:     billFromAddress,
			DispatchFromAddress: dispatchFromAddress,
			TransitDetails:      transitDetails,
			ShipToConfig:        shipToConfig,
			BillToConfig:        billToConfig,
			BillFromConfig:      billFromConfig,
			DispatchFromConfig:  dispatchFromConfig,
			TotalTaxable:        totalTaxable,
			TotalTax:            totalTax,
			GrandTotal:          grandTotal,
			RoundedTotal:        roundedTotal,
			RoundOff:            roundOff,
			HalfTax:             halfTax,
			TotalQty:            totalQty,
			AmountInWords:       amountInWords,
		})
		if err != nil {
			slog.Error("error generating transit DC Excel", slog.String("error", err.Error()), slog.Int("dcID", dcID), slog.Int("projectID", projectID))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to generate Excel"})
		}

		c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		if err := excelFile.Write(c.Response().Writer); err != nil {
			slog.Error("error writing transit DC Excel response", slog.String("error", err.Error()), slog.Int("dcID", dcID))
		}
	}

	return nil
}

func buildTransferPDF(projectID, dcID int, dc *models.DeliveryChallan) ([]byte, error) {
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return nil, err
	}

	tdc, _ := database.GetTransferDCByDCID(dcID)

	lineItems, _ := database.GetLineItemsByDCID(dcID)
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
	roundOff := roundedTotal - grandTotal
	halfTax := totalTax / 2.0

	// Fetch addresses
	var billToAddress, billFromAddress, dispatchFromAddress, hubAddress *models.Address
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}
	if dc.BillFromAddressID != nil && *dc.BillFromAddressID > 0 {
		billFromAddress, _ = database.GetAddress(*dc.BillFromAddressID)
	}
	if dc.DispatchFromAddressID != nil && *dc.DispatchFromAddressID > 0 {
		dispatchFromAddress, _ = database.GetAddress(*dc.DispatchFromAddressID)
	}
	if tdc != nil && tdc.HubAddressID > 0 {
		hubAddress, _ = database.GetAddress(tdc.HubAddressID)
	}

	company, _ := database.GetCompanySettings()
	amountInWords := helpers.NumberToIndianWords(roundedTotal)

	// Build destinations and products for the breakdown table
	var destinations []services.TransferDCPDFDestination
	var products []services.TransferDCPDFProduct
	if tdc != nil {
		dests, _ := database.GetTransferDCDestinations(tdc.ID)
		for _, d := range dests {
			qtyMap := make(map[int]int)
			for _, q := range d.Quantities {
				qtyMap[q.ProductID] = q.Quantity
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
		// Build product list from line items
		for _, li := range lineItems {
			products = append(products, services.TransferDCPDFProduct{
				ID:   li.ProductID,
				Name: li.ItemName,
			})
		}
	}

	// Fetch address configs for print column filtering
	billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
	dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")
	billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")
	shipToConfig, _ := database.GetOrCreateAddressConfig(projectID, "ship_to")

	return services.GenerateTransferDCPDF(&services.TransferDCPDFData{
		Project:             project,
		DC:                  dc,
		TransferDC:          tdc,
		Company:             company,
		LineItems:           lineItems,
		HubAddress:          hubAddress,
		BillFromAddress:     billFromAddress,
		DispatchFromAddress: dispatchFromAddress,
		BillToAddress:       billToAddress,
		BillFromConfig:      billFromConfig,
		DispatchFromConfig:  dispatchFromConfig,
		BillToConfig:        billToConfig,
		ShipToConfig:        shipToConfig,
		Destinations:        destinations,
		Products:            products,
		TotalTaxable:        totalTaxable,
		TotalTax:            totalTax,
		GrandTotal:          grandTotal,
		RoundedTotal:        roundedTotal,
		RoundOff:            roundOff,
		HalfTax:             halfTax,
		TotalQty:            totalQty,
		AmountInWords:       amountInWords,
	})
}

func buildTransferExcel(c echo.Context, projectID int, dc *models.DeliveryChallan, lineItems []models.DCLineItem, project *models.Project, company *models.CompanySettings, shipToAddress, billToAddress *models.Address, filename string) error {
	tdc, _ := database.GetTransferDCByDCID(dc.ID)

	totalTaxable, totalTax, grandTotal, roundedTotal, roundOff, _, _ := services.CalcTransitTotals(lineItems)
	halfTax := totalTax / 2.0
	amountInWords := helpers.NumberToIndianWords(roundedTotal)

	var totalQty int
	for _, li := range lineItems { //nolint:gocritic
		totalQty += li.Quantity
	}

	// Fetch addresses
	var billFromAddress, dispatchFromAddress, hubAddress *models.Address
	if dc.BillFromAddressID != nil && *dc.BillFromAddressID > 0 {
		billFromAddress, _ = database.GetAddress(*dc.BillFromAddressID)
	}
	if dc.DispatchFromAddressID != nil && *dc.DispatchFromAddressID > 0 {
		dispatchFromAddress, _ = database.GetAddress(*dc.DispatchFromAddressID)
	}
	if tdc != nil && tdc.HubAddressID > 0 {
		hubAddress, _ = database.GetAddress(tdc.HubAddressID)
	}

	// Build destinations and products
	var destinations []services.TransferDCExcelDestination
	var products []services.TransferDCExcelProduct
	if tdc != nil {
		dests, _ := database.GetTransferDCDestinations(tdc.ID)
		for _, d := range dests {
			qtyMap := make(map[int]int)
			for _, q := range d.Quantities {
				qtyMap[q.ProductID] = q.Quantity
			}
			destinations = append(destinations, services.TransferDCExcelDestination{
				Name:       d.AddressName,
				Quantities: qtyMap,
			})
		}
		for _, li := range lineItems {
			products = append(products, services.TransferDCExcelProduct{
				ID:   li.ProductID,
				Name: li.ItemName,
			})
		}
	}

	// Fetch address configs
	billFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_from")
	dispatchFromConfig, _ := database.GetOrCreateAddressConfig(projectID, "dispatch_from")
	billToConfig, _ := database.GetOrCreateAddressConfig(projectID, "bill_to")

	excelFile, err := services.GenerateTransferDCExcel(&services.TransferDCExcelData{
		DC:                  dc,
		TransferDC:          tdc,
		Company:             company,
		Project:             project,
		LineItems:           lineItems,
		HubAddress:          hubAddress,
		BillFromAddress:     billFromAddress,
		DispatchFromAddress: dispatchFromAddress,
		BillToAddress:       billToAddress,
		BillFromConfig:      billFromConfig,
		DispatchFromConfig:  dispatchFromConfig,
		BillToConfig:        billToConfig,
		Destinations:        destinations,
		Products:            products,
		TotalTaxable:        totalTaxable,
		TotalTax:            totalTax,
		GrandTotal:          grandTotal,
		RoundedTotal:        roundedTotal,
		RoundOff:            roundOff,
		HalfTax:             halfTax,
		TotalQty:            totalQty,
		AmountInWords:       amountInWords,
	})
	if err != nil {
		slog.Error("error generating transfer DC Excel", slog.String("error", err.Error()), slog.Int("dcID", dc.ID))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to generate Excel"})
	}

	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	if err := excelFile.Write(c.Response().Writer); err != nil {
		slog.Error("error writing transfer DC Excel response", slog.String("error", err.Error()), slog.Int("dcID", dc.ID))
	}
	return nil
}
