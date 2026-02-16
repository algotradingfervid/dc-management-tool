package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// ExportDCPDF generates and serves a PDF for a DC by navigating to its print view with headless Chrome.
func ExportDCPDF(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DC ID"})
		return
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
		return
	}

	// Render print template to HTML and convert to PDF via headless Chrome
	pdfData, err := generatePDFForDC(projectID, dcID, dc)
	if err != nil {
		log.Printf("Error generating PDF for DC %d: %v", dcID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	filename := services.SanitizeDCFilename(dc.DCNumber) + ".pdf"

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/pdf", pdfData)
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
	for _, li := range lineItems {
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
	for _, li := range lineItems {
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
func ExportDCExcel(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DC ID"})
		return
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
		return
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
		for _, li := range lineItems {
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
			log.Printf("Error generating Excel for DC %d: %v", dcID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel"})
			return
		}

		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		if err := excelFile.Write(c.Writer); err != nil {
			log.Printf("Error writing Excel: %v", err)
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
			log.Printf("Error generating Excel for DC %d: %v", dcID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel"})
			return
		}

		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		if err := excelFile.Write(c.Writer); err != nil {
			log.Printf("Error writing Excel: %v", err)
		}
	}
}
