package handlers

import (
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

// ExportDCPDF generates and serves a PDF for a DC by navigating to its print view with headless Chrome.
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

// generatePDFForDC renders the print template to HTML and converts to PDF.
func generatePDFForDC(projectID, dcID int, dc *models.DeliveryChallan) ([]byte, error) {
	// Build a full HTML page from the print template data
	var htmlContent string
	var err error

	if dc.DCType == "official" {
		htmlContent, err = renderOfficialPrintHTML(projectID, dcID, dc)
	} else {
		htmlContent, err = renderTransitPrintHTML(projectID, dcID, dc)
	}

	if err != nil {
		return nil, err
	}

	return services.GeneratePDFFromHTML(htmlContent)
}

func renderTransitPrintHTML(projectID, dcID int, dc *models.DeliveryChallan) (string, error) {
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return "", err
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
	company, _ := database.GetCompanySettings()
	amountInWords := helpers.NumberToIndianWords(roundedTotal)

	return buildTransitPrintHTML(project, dc, transitDetails, lineItems, company,
		shipToAddress, billToAddress, totalTaxable, totalTax, grandTotal,
		roundedTotal, roundOff, totalQty, halfTax, amountInWords), nil
}

func renderOfficialPrintHTML(projectID, dcID int, dc *models.DeliveryChallan) (string, error) {
	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return "", err
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

	var shipToAddress, billToAddress *models.Address
	if dc.ShipToAddressID > 0 {
		shipToAddress, _ = database.GetAddress(dc.ShipToAddressID)
	}
	if dc.BillToAddressID != nil && *dc.BillToAddressID > 0 {
		billToAddress, _ = database.GetAddress(*dc.BillToAddressID)
	}
	company, _ := database.GetCompanySettings()

	return buildOfficialPrintHTML(project, dc, lineItems, company,
		shipToAddress, billToAddress, totalQty), nil
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

	if dc.DCType == "official" {
		var totalQty int
		for _, li := range lineItems { //nolint:gocritic
			totalQty += li.Quantity
		}

		excelFile, err := services.GenerateOfficialDCExcel(&services.OfficialDCExcelData{
			DC:            dc,
			LineItems:     lineItems,
			Company:       company,
			Project:       project,
			ShipToAddress: shipToAddress,
			BillToAddress: billToAddress,
			TotalQty:      totalQty,
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
		totalTaxable, totalTax, grandTotal, roundedTotal, roundOff, cgst, sgst := services.CalcTransitTotals(lineItems)
		amountInWords := helpers.NumberToIndianWords(roundedTotal)

		excelFile, err := services.GenerateTransitDCExcel(&services.TransitDCExcelData{
			DC:            dc,
			LineItems:     lineItems,
			Company:       company,
			Project:       project,
			ShipToAddress: shipToAddress,
			BillToAddress: billToAddress,
			TotalTaxable:  totalTaxable,
			TotalTax:      totalTax,
			GrandTotal:    grandTotal,
			RoundedTotal:  roundedTotal,
			RoundOff:      roundOff,
			CGST:          cgst,
			SGST:          sgst,
			AmountInWords: amountInWords,
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
