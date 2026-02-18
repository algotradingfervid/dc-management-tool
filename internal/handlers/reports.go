package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	htmxreports "github.com/narendhupati/dc-management-tool/components/htmx/reports"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	pagesreports "github.com/narendhupati/dc-management-tool/components/pages/reports"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
	"github.com/xuri/excelize/v2"
)

// parseDateRange parses the date range from query params and returns start/end times.
func parseDateRange(c echo.Context) (dateRange string, startDate, endDate *time.Time) {
	dateRange = c.QueryParam("range")
	if dateRange == "" {
		dateRange = "this_fy"
	}
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
		if from := c.QueryParam("from"); from != "" {
			if t, err := time.Parse("2006-01-02", from); err == nil {
				startDate = &t
			}
		}
		if to := c.QueryParam("to"); to != "" {
			if t, err := time.Parse("2006-01-02", to); err == nil {
				endDate = &t
			}
		}
	case "all_time":
		// no filter
	}
	return
}

// reportFields holds common values shared across all report handlers.
type reportFields struct {
	user           *models.User
	currentProject *models.Project
	allProjects    []*models.Project
	flashType      string
	flashMessage   string
	dateRange      string
	startDate      *time.Time
	endDate        *time.Time
	fromDate       string
	toDate         string
}

// getReportFields extracts common values used by every report handler.
func getReportFields(c echo.Context, reportType string) reportFields {
	user := auth.GetCurrentUser(c)
	project, _ := c.Get("currentProject").(*models.Project)
	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request())

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
	c.Set("breadcrumbs", breadcrumbs)
	c.Set("csrfToken", csrf.Token(c.Request()))

	dateRange, startDate, endDate := parseDateRange(c)

	return reportFields{
		user:           user,
		currentProject: project,
		allProjects:    allProjects,
		flashType:      flashType,
		flashMessage:   flashMessage,
		dateRange:      dateRange,
		startDate:      startDate,
		endDate:        endDate,
		fromDate:       c.QueryParam("from"),
		toDate:         c.QueryParam("to"),
	}
}

// ShowReportsIndex shows the report type selector page.
func ShowReportsIndex(c echo.Context) error {
	f := getReportFields(c, "")
	pageContent := pagesreports.Index(
		f.user,
		f.currentProject,
		f.allProjects,
		f.flashType,
		f.flashMessage,
	)
	sidebar := partials.Sidebar(f.user, f.currentProject, f.allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(f.user, f.currentProject, f.allProjects, f.flashType, f.flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Reports", sidebar, topbar, f.flashMessage, f.flashType, pageContent))
}

// ShowDCSummaryReport shows the DC summary report.
func ShowDCSummaryReport(c echo.Context) error {
	f := getReportFields(c, "DC Summary")

	report, err := database.GetDCSummaryReport(f.currentProject.ID, f.startDate, f.endDate)
	if err != nil {
		slog.Error("error fetching DC summary report", slog.String("error", err.Error()), slog.Int("projectID", f.currentProject.ID))
		report = &database.DCSummaryReport{}
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return components.RenderOK(c, htmxreports.DCSummaryPartial(htmxreports.DCSummaryPartialProps{
			Report: report,
		}))
	}
	pageContent := pagesreports.DCSummary(
		f.user,
		f.currentProject,
		f.allProjects,
		report,
		f.dateRange,
		f.fromDate,
		f.toDate,
		f.flashType,
		f.flashMessage,
	)
	sidebar := partials.Sidebar(f.user, f.currentProject, f.allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(f.user, f.currentProject, f.allProjects, f.flashType, f.flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Reports", sidebar, topbar, f.flashMessage, f.flashType, pageContent))
}

// ShowDestinationReport shows the destination-wise report.
func ShowDestinationReport(c echo.Context) error {
	f := getReportFields(c, "Destination Report")

	district := c.QueryParam("district")
	mandal := c.QueryParam("mandal")

	if district != "" && mandal != "" {
		dcs, err := database.GetDestinationDCs(f.currentProject.ID, district, mandal, f.startDate, f.endDate)
		if err != nil {
			slog.Error("error fetching destination DCs",
				slog.String("error", err.Error()),
				slog.Int("projectID", f.currentProject.ID),
				slog.String("district", district),
				slog.String("mandal", mandal),
			)
			dcs = nil
		}

		if c.Request().Header.Get("HX-Request") == "true" {
			return components.RenderOK(c, htmxreports.DestinationDrilldownPartial(htmxreports.DestinationDrilldownPartialProps{
				DCs: dcs,
			}))
		}

		// Convert value slice to pointer slice for the page component.
		dcPtrs := make([]*database.DestinationDCRow, len(dcs))
		for i := range dcs {
			dcPtrs[i] = &dcs[i]
		}
		pageContent := pagesreports.Destination(
			f.user,
			f.currentProject,
			f.allProjects,
			nil,    // rows (summary) not used in drill-down
			dcPtrs, // dcs
			true,   // drillDown
			district,
			mandal,
			f.dateRange,
			f.fromDate,
			f.toDate,
			f.flashType,
			f.flashMessage,
		)
		sidebar := partials.Sidebar(f.user, f.currentProject, f.allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(f.user, f.currentProject, f.allProjects, f.flashType, f.flashMessage)
		return components.RenderOK(c, layouts.MainWithContent("Reports", sidebar, topbar, f.flashMessage, f.flashType, pageContent))
	}

	rows, err := database.GetDestinationReport(f.currentProject.ID, f.startDate, f.endDate)
	if err != nil {
		slog.Error("error fetching destination report", slog.String("error", err.Error()), slog.Int("projectID", f.currentProject.ID))
		rows = nil
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return components.RenderOK(c, htmxreports.DestinationPartial(htmxreports.DestinationPartialProps{
			Rows:     rows,
			BasePath: fmt.Sprintf("/projects/%d/reports", f.currentProject.ID),
			Range:    f.dateRange,
			FromDate: f.fromDate,
			ToDate:   f.toDate,
		}))
	}

	// Convert value slice to pointer slice for the page component.
	rowPtrs := make([]*database.DestinationRow, len(rows))
	for i := range rows {
		rowPtrs[i] = &rows[i]
	}
	pageContent := pagesreports.Destination(
		f.user,
		f.currentProject,
		f.allProjects,
		rowPtrs, // rows
		nil,     // dcs (drill-down) not used in summary view
		false,   // drillDown
		"",
		"",
		f.dateRange,
		f.fromDate,
		f.toDate,
		f.flashType,
		f.flashMessage,
	)
	sidebar := partials.Sidebar(f.user, f.currentProject, f.allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(f.user, f.currentProject, f.allProjects, f.flashType, f.flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Reports", sidebar, topbar, f.flashMessage, f.flashType, pageContent))
}

// ShowProductReport shows the product-wise report.
func ShowProductReport(c echo.Context) error {
	f := getReportFields(c, "Product Report")

	rows, err := database.GetProductReport(f.currentProject.ID, f.startDate, f.endDate)
	if err != nil {
		slog.Error("error fetching product report", slog.String("error", err.Error()), slog.Int("projectID", f.currentProject.ID))
		rows = nil
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return components.RenderOK(c, htmxreports.ProductPartial(htmxreports.ProductPartialProps{
			Rows: rows,
		}))
	}

	// Convert value slice to pointer slice for the page component.
	rowPtrs := make([]*database.ProductReportRow, len(rows))
	for i := range rows {
		rowPtrs[i] = &rows[i]
	}
	pageContent := pagesreports.Product(
		f.user,
		f.currentProject,
		f.allProjects,
		rowPtrs,
		f.dateRange,
		f.fromDate,
		f.toDate,
		f.flashType,
		f.flashMessage,
	)
	sidebar := partials.Sidebar(f.user, f.currentProject, f.allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(f.user, f.currentProject, f.allProjects, f.flashType, f.flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Reports", sidebar, topbar, f.flashMessage, f.flashType, pageContent))
}

// ShowSerialReport shows the serial number report.
func ShowSerialReport(c echo.Context) error {
	f := getReportFields(c, "Serial Number Report")
	search := c.QueryParam("q")

	rows, err := database.GetSerialReport(f.currentProject.ID, search, f.startDate, f.endDate)
	if err != nil {
		slog.Error("error fetching serial report", slog.String("error", err.Error()), slog.Int("projectID", f.currentProject.ID))
		rows = nil
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return components.RenderOK(c, htmxreports.SerialPartial(htmxreports.SerialPartialProps{
			Rows: rows,
		}))
	}

	// Convert value slice to pointer slice for the page component.
	rowPtrs := make([]*database.SerialReportRow, len(rows))
	for i := range rows {
		rowPtrs[i] = &rows[i]
	}
	pageContent := pagesreports.Serial(
		f.user,
		f.currentProject,
		f.allProjects,
		rowPtrs,
		search,
		f.dateRange,
		f.fromDate,
		f.toDate,
		f.flashType,
		f.flashMessage,
	)
	sidebar := partials.Sidebar(f.user, f.currentProject, f.allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(f.user, f.currentProject, f.allProjects, f.flashType, f.flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Reports", sidebar, topbar, f.flashMessage, f.flashType, pageContent))
}

// ExportDCSummaryExcel exports the DC summary report as Excel.
func ExportDCSummaryExcel(c echo.Context) error {
	project, _ := c.Get("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	report, err := database.GetDCSummaryReport(project.ID, startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to generate report"})
	}

	f := excelize.NewFile()
	sheet := "DC Summary"
	_ = f.SetSheetName("Sheet1", sheet)

	headers := []string{"Metric", "Count"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
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
			_ = f.SetCellValue(sheet, cell, val)
		}
	}

	filename := fmt.Sprintf("dc-summary-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	if err := f.Write(c.Response().Writer); err != nil {
		slog.Error("error writing Excel response", slog.String("error", err.Error()), slog.Int("projectID", project.ID))
	}
	return nil
}

// ExportDestinationExcel exports the destination report as Excel.
func ExportDestinationExcel(c echo.Context) error {
	project, _ := c.Get("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	rows, err := database.GetDestinationReport(project.ID, startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to generate report"})
	}

	f := excelize.NewFile()
	sheet := "Destination Report"
	_ = f.SetSheetName("Sheet1", sheet)

	headers := []string{"District", "Mandal", "Official DCs", "Total Items", "Draft", "Issued"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, r := range rows {
		row := i + 2
		_ = f.SetCellValue(sheet, cellName(1, row), r.District)
		_ = f.SetCellValue(sheet, cellName(2, row), r.Mandal)
		_ = f.SetCellValue(sheet, cellName(3, row), r.OfficialDCs)
		_ = f.SetCellValue(sheet, cellName(4, row), r.TotalItems)
		_ = f.SetCellValue(sheet, cellName(5, row), r.DraftCount)
		_ = f.SetCellValue(sheet, cellName(6, row), r.IssuedCount)
	}

	filename := fmt.Sprintf("destination-report-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	_ = f.Write(c.Response().Writer)
	return nil
}

// ExportProductExcel exports the product report as Excel.
func ExportProductExcel(c echo.Context) error {
	project, _ := c.Get("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)

	rows, err := database.GetProductReport(project.ID, startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to generate report"})
	}

	f := excelize.NewFile()
	sheet := "Product Report"
	_ = f.SetSheetName("Sheet1", sheet)

	headers := []string{"Product Name", "Total Qty Dispatched", "# DCs", "# Destinations"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, r := range rows {
		row := i + 2
		_ = f.SetCellValue(sheet, cellName(1, row), r.ProductName)
		_ = f.SetCellValue(sheet, cellName(2, row), r.TotalQty)
		_ = f.SetCellValue(sheet, cellName(3, row), r.DCCount)
		_ = f.SetCellValue(sheet, cellName(4, row), r.DestinationCount)
	}

	filename := fmt.Sprintf("product-report-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	_ = f.Write(c.Response().Writer)
	return nil
}

// ExportSerialExcel exports the serial number report as Excel.
func ExportSerialExcel(c echo.Context) error {
	project, _ := c.Get("currentProject").(*models.Project)
	_, startDate, endDate := parseDateRange(c)
	search := c.QueryParam("q")

	rows, err := database.GetSerialReport(project.ID, search, startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to generate report"})
	}

	f := excelize.NewFile()
	sheet := "Serial Numbers"
	_ = f.SetSheetName("Sheet1", sheet)

	headers := []string{"Serial Number", "Product", "DC Number", "Date", "Vehicle"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, r := range rows {
		row := i + 2
		_ = f.SetCellValue(sheet, cellName(1, row), r.SerialNumber)
		_ = f.SetCellValue(sheet, cellName(2, row), r.ProductName)
		_ = f.SetCellValue(sheet, cellName(3, row), r.TransitDCNumber)
		_ = f.SetCellValue(sheet, cellName(4, row), r.ChallanDate)
		_ = f.SetCellValue(sheet, cellName(5, row), r.VehicleNumber)
	}

	filename := fmt.Sprintf("serial-report-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	_ = f.Write(c.Response().Writer)
	return nil
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
