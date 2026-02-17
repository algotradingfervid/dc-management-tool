package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
	"github.com/xuri/excelize/v2"
)

// parseDateRange parses the date range from query params and returns start/end times.
func parseDateRange(c *gin.Context) (dateRange string, startDate, endDate *time.Time) {
	dateRange = c.DefaultQuery("range", "this_fy")
	now := time.Now()

	switch dateRange {
	case "this_month":
		s := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		e := s.AddDate(0, 1, -1)
		startDate = &s
		endDate = &e
	case "this_fy":
		year := now.Year()
		if now.Month() < time.April {
			year--
		}
		s := services.GetFinancialYearStart(year)
		e := services.GetFinancialYearEnd(year)
		startDate = &s
		endDate = &e
	case "last_fy":
		year := now.Year()
		if now.Month() < time.April {
			year--
		}
		year--
		s := services.GetFinancialYearStart(year)
		e := services.GetFinancialYearEnd(year)
		startDate = &s
		endDate = &e
	case "custom":
		if from := c.Query("from"); from != "" {
			if t, err := time.Parse("2006-01-02", from); err == nil {
				startDate = &t
			}
		}
		if to := c.Query("to"); to != "" {
			if t, err := time.Parse("2006-01-02", to); err == nil {
				endDate = &t
			}
		}
	case "all_time":
		// no filter
	}
	return
}

// commonReportData builds the base data map for all report pages.
func commonReportData(c *gin.Context, reportType string) gin.H {
	user := auth.GetCurrentUser(c)
	project := c.MustGet("currentProject").(*models.Project)
	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
		helpers.Breadcrumb{Title: "Reports", URL: fmt.Sprintf("/projects/%d/reports", project.ID)},
	)
	if reportType != "" {
		breadcrumbs = helpers.BuildBreadcrumbs(
			helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
			helpers.Breadcrumb{Title: "Reports", URL: fmt.Sprintf("/projects/%d/reports", project.ID)},
			helpers.Breadcrumb{Title: reportType},
		)
	}

	dateRange, startDate, endDate := parseDateRange(c)

	return gin.H{
		"user":           user,
		"currentProject": project,
		"allProjects":    allProjects,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"flashMessage":   flashMessage,
		"flashType":      flashType,
		"csrfToken":      csrf.Token(c.Request),
		"Range":          dateRange,
		"StartDate":      startDate,
		"EndDate":        endDate,
		"FromDate":       c.Query("from"),
		"ToDate":         c.Query("to"),
		"basePath":       fmt.Sprintf("/projects/%d/reports", project.ID),
	}
}

// ShowReportsIndex shows the report type selector page.
func ShowReportsIndex(c *gin.Context) {
	data := commonReportData(c, "")
	c.HTML(http.StatusOK, "reports/index.html", data)
}

// ShowDCSummaryReport shows the DC summary report.
func ShowDCSummaryReport(c *gin.Context) {
	project := c.MustGet("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	report, err := database.GetDCSummaryReport(project.ID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching DC summary report: %v", err)
		report = &database.DCSummaryReport{}
	}

	data := commonReportData(c, "DC Summary")
	data["Report"] = report

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "htmx/reports/dc_summary_partial.html", data)
		return
	}
	c.HTML(http.StatusOK, "reports/dc_summary.html", data)
}

// ShowDestinationReport shows the destination-wise report.
func ShowDestinationReport(c *gin.Context) {
	project := c.MustGet("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	// Check for drill-down
	district := c.Query("district")
	mandal := c.Query("mandal")
	if district != "" && mandal != "" {
		dcs, err := database.GetDestinationDCs(project.ID, district, mandal, startDate, endDate)
		if err != nil {
			log.Printf("Error fetching destination DCs: %v", err)
		}
		data := commonReportData(c, fmt.Sprintf("Destination: %s / %s", district, mandal))
		data["DCs"] = dcs
		data["District"] = district
		data["Mandal"] = mandal
		data["DrillDown"] = true

		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "htmx/reports/destination_drilldown_partial.html", data)
			return
		}
		c.HTML(http.StatusOK, "reports/destination.html", data)
		return
	}

	rows, err := database.GetDestinationReport(project.ID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching destination report: %v", err)
	}

	data := commonReportData(c, "Destination Report")
	data["Rows"] = rows

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "htmx/reports/destination_partial.html", data)
		return
	}
	c.HTML(http.StatusOK, "reports/destination.html", data)
}

// ShowProductReport shows the product-wise report.
func ShowProductReport(c *gin.Context) {
	project := c.MustGet("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	rows, err := database.GetProductReport(project.ID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching product report: %v", err)
	}

	data := commonReportData(c, "Product Report")
	data["Rows"] = rows

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "htmx/reports/product_partial.html", data)
		return
	}
	c.HTML(http.StatusOK, "reports/product.html", data)
}

// ShowSerialReport shows the serial number report.
func ShowSerialReport(c *gin.Context) {
	project := c.MustGet("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)
	search := c.Query("q")

	rows, err := database.GetSerialReport(project.ID, search, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching serial report: %v", err)
	}

	data := commonReportData(c, "Serial Number Report")
	data["Rows"] = rows
	data["Search"] = search

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "htmx/reports/serial_partial.html", data)
		return
	}
	c.HTML(http.StatusOK, "reports/serial.html", data)
}

// ExportDCSummaryExcel exports the DC summary report as Excel.
func ExportDCSummaryExcel(c *gin.Context) {
	project := c.MustGet("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	report, err := database.GetDCSummaryReport(project.ID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	f := excelize.NewFile()
	sheet := "DC Summary"
	f.SetSheetName("Sheet1", sheet)

	// Headers
	headers := []string{"Metric", "Count"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	data := [][]interface{}{
		{"Transit DCs (Draft)", report.TransitDraftDCs},
		{"Transit DCs (Issued)", report.TransitIssuedDCs},
		{"Official DCs (Draft)", report.OfficialDraftDCs},
		{"Official DCs (Issued)", report.OfficialIssuedDCs},
		{"Total Items Dispatched", report.TotalItemsDispatched},
		{"Total Serial Numbers Used", report.TotalSerialsUsed},
	}
	for i, row := range data {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, val)
		}
	}

	filename := fmt.Sprintf("dc-summary-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	if err := f.Write(c.Writer); err != nil {
		log.Printf("Error writing Excel: %v", err)
	}
}

// ExportDestinationExcel exports the destination report as Excel.
func ExportDestinationExcel(c *gin.Context) {
	project := c.MustGet("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	rows, err := database.GetDestinationReport(project.ID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	f := excelize.NewFile()
	sheet := "Destination Report"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"District", "Mandal", "Official DCs", "Total Items", "Draft", "Issued"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, r := range rows {
		row := i + 2
		f.SetCellValue(sheet, cellName(1, row), r.District)
		f.SetCellValue(sheet, cellName(2, row), r.Mandal)
		f.SetCellValue(sheet, cellName(3, row), r.OfficialDCs)
		f.SetCellValue(sheet, cellName(4, row), r.TotalItems)
		f.SetCellValue(sheet, cellName(5, row), r.DraftCount)
		f.SetCellValue(sheet, cellName(6, row), r.IssuedCount)
	}

	filename := fmt.Sprintf("destination-report-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	f.Write(c.Writer)
}

// ExportProductExcel exports the product report as Excel.
func ExportProductExcel(c *gin.Context) {
	project := c.MustGet("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	rows, err := database.GetProductReport(project.ID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	f := excelize.NewFile()
	sheet := "Product Report"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Product Name", "Total Qty Dispatched", "# DCs", "# Destinations"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, r := range rows {
		row := i + 2
		f.SetCellValue(sheet, cellName(1, row), r.ProductName)
		f.SetCellValue(sheet, cellName(2, row), r.TotalQty)
		f.SetCellValue(sheet, cellName(3, row), r.DCCount)
		f.SetCellValue(sheet, cellName(4, row), r.DestinationCount)
	}

	filename := fmt.Sprintf("product-report-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	f.Write(c.Writer)
}

// ExportSerialExcel exports the serial number report as Excel.
func ExportSerialExcel(c *gin.Context) {
	project := c.MustGet("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)
	search := c.Query("q")

	rows, err := database.GetSerialReport(project.ID, search, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	f := excelize.NewFile()
	sheet := "Serial Numbers"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Serial Number", "Product", "DC Number", "Date", "Vehicle"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, r := range rows {
		row := i + 2
		f.SetCellValue(sheet, cellName(1, row), r.SerialNumber)
		f.SetCellValue(sheet, cellName(2, row), r.ProductName)
		f.SetCellValue(sheet, cellName(3, row), r.TransitDCNumber)
		f.SetCellValue(sheet, cellName(4, row), r.ChallanDate)
		f.SetCellValue(sheet, cellName(5, row), r.VehicleNumber)
	}

	filename := fmt.Sprintf("serial-report-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	f.Write(c.Writer)
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
