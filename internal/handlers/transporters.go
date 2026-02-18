package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	htmxtransporters "github.com/narendhupati/dc-management-tool/components/htmx/transporters"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	pagetransporters "github.com/narendhupati/dc-management-tool/components/pages/transporters"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func ListTransporters(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Project not found")
		return c.Redirect(http.StatusFound, "/projects")
	}

	search := c.QueryParam("search")
	pageStr := c.QueryParam("page")
	if pageStr == "" {
		pageStr = "1"
	}
	page, _ := strconv.Atoi(pageStr)

	transporterPage, err := database.SearchTransporters(projectID, search, page, 20)
	if err != nil {
		slog.Error("Error fetching transporters", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		transporterPage = &models.TransporterPage{Transporters: []*models.Transporter{}, CurrentPage: 1, TotalPages: 1, PerPage: 20}
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request())

	_ = helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Transporters", URL: ""},
	)

	pageContent := pagetransporters.List(
		user,
		project,
		allProjects,
		transporterPage,
		search,
		"", // sortBy
		"", // sortDir
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Transporters", sidebar, topbar, flashMessage, flashType, pageContent))
}

func ShowAddTransporterForm(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	return components.RenderOK(c, htmxtransporters.TransporterForm(htmxtransporters.TransporterFormProps{
		ProjectID:   projectID,
		Transporter: models.Transporter{IsActive: true},
		IsEdit:      false,
		Errors:      map[string]string{},
		CsrfToken:   csrf.Token(c.Request()),
	}))
}

func CreateTransporterHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	transporter := &models.Transporter{
		ProjectID:     projectID,
		CompanyName:   strings.TrimSpace(c.FormValue("company_name")),
		ContactPerson: strings.TrimSpace(c.FormValue("contact_person")),
		Phone:         strings.TrimSpace(c.FormValue("phone")),
		GSTNumber:     strings.TrimSpace(c.FormValue("gst_number")),
		IsActive:      true,
	}

	errors := helpers.ValidateStruct(&transporter)

	if len(errors) > 0 {
		return components.RenderOK(c, htmxtransporters.TransporterForm(htmxtransporters.TransporterFormProps{
			ProjectID:   projectID,
			Transporter: *transporter,
			IsEdit:      false,
			Errors:      errors,
			CsrfToken:   csrf.Token(c.Request()),
		}))
	}

	if err := database.CreateTransporter(transporter); err != nil {
		slog.Error("Error creating transporter", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		errors["general"] = "Failed to create transporter"
		return components.RenderOK(c, htmxtransporters.TransporterForm(htmxtransporters.TransporterFormProps{
			ProjectID:   projectID,
			Transporter: *transporter,
			IsEdit:      false,
			Errors:      errors,
			CsrfToken:   csrf.Token(c.Request()),
		}))
	}

	c.Response().Header().Set("HX-Trigger", "transporterChanged")
	return components.RenderOK(c, htmxtransporters.TransporterFormSuccess(htmxtransporters.TransporterFormSuccessProps{
		Message: "Transporter added successfully",
	}))
}

func ShowEditTransporterForm(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid transporter ID")
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		return c.String(http.StatusNotFound, "Transporter not found")
	}

	return components.RenderOK(c, htmxtransporters.TransporterForm(htmxtransporters.TransporterFormProps{
		ProjectID:   projectID,
		Transporter: *transporter,
		IsEdit:      true,
		Errors:      map[string]string{},
		CsrfToken:   csrf.Token(c.Request()),
	}))
}

func UpdateTransporterHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid transporter ID")
	}

	existing, err := database.GetTransporterByID(transporterID)
	if err != nil || existing.ProjectID != projectID {
		return c.String(http.StatusNotFound, "Transporter not found")
	}

	transporter := &models.Transporter{
		ID:            transporterID,
		ProjectID:     projectID,
		CompanyName:   strings.TrimSpace(c.FormValue("company_name")),
		ContactPerson: strings.TrimSpace(c.FormValue("contact_person")),
		Phone:         strings.TrimSpace(c.FormValue("phone")),
		GSTNumber:     strings.TrimSpace(c.FormValue("gst_number")),
		IsActive:      existing.IsActive,
	}

	errors := helpers.ValidateStruct(&transporter)

	if len(errors) > 0 {
		return components.RenderOK(c, htmxtransporters.TransporterForm(htmxtransporters.TransporterFormProps{
			ProjectID:   projectID,
			Transporter: *transporter,
			IsEdit:      true,
			Errors:      errors,
			CsrfToken:   csrf.Token(c.Request()),
		}))
	}

	if err := database.UpdateTransporter(transporter); err != nil {
		slog.Error("Error updating transporter", slog.String("error", err.Error()), slog.Int("transporterID", transporterID))
		errors["general"] = "Failed to update transporter"
		return components.RenderOK(c, htmxtransporters.TransporterForm(htmxtransporters.TransporterFormProps{
			ProjectID:   projectID,
			Transporter: *transporter,
			IsEdit:      true,
			Errors:      errors,
			CsrfToken:   csrf.Token(c.Request()),
		}))
	}

	c.Response().Header().Set("HX-Trigger", "transporterChanged")
	return components.RenderOK(c, htmxtransporters.TransporterFormSuccess(htmxtransporters.TransporterFormSuccessProps{
		Message: "Transporter updated successfully",
	}))
}

func ToggleTransporterStatus(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid transporter ID"})
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Transporter not found"})
	}

	if transporter.IsActive {
		err = database.DeactivateTransporter(transporterID, projectID)
	} else {
		err = database.ActivateTransporter(transporterID, projectID)
	}

	if err != nil {
		slog.Error("Error toggling transporter status", slog.String("error", err.Error()), slog.Int("transporterID", transporterID))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to update status"})
	}

	c.Response().Header().Set("HX-Trigger", "transporterChanged")
	return c.String(http.StatusOK, "")
}

func ShowTransporterDetail(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transporters", projectID))
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Transporter not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transporters", projectID))
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request())

	pageContent := pagetransporters.Detail(
		user,
		project,
		allProjects,
		transporter,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Transporter Details", sidebar, topbar, flashMessage, flashType, pageContent))
}

func AddVehicleHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid transporter ID"})
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Transporter not found"})
	}

	vehicle := &models.TransporterVehicle{
		TransporterID: transporterID,
		VehicleNumber: strings.TrimSpace(c.FormValue("vehicle_number")),
		VehicleType:   strings.TrimSpace(c.FormValue("vehicle_type")),
		DriverName:    strings.TrimSpace(c.FormValue("driver_name")),
		DriverPhone1:  strings.TrimSpace(c.FormValue("driver_phone1")),
		DriverPhone2:  strings.TrimSpace(c.FormValue("driver_phone2")),
	}

	if vehicle.VehicleType == "" {
		vehicle.VehicleType = "truck"
	}

	errors := helpers.ValidateStruct(&vehicle)
	if len(errors) > 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": errors["vehicle_number"]})
	}

	if err := database.CreateVehicle(vehicle); err != nil {
		slog.Error("Error adding vehicle", slog.String("error", err.Error()), slog.Int("transporterID", transporterID))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to add vehicle"})
	}

	vehicles, _ := database.GetVehiclesByTransporterID(transporterID)
	return components.RenderOK(c, htmxtransporters.HTMXVehicleList(htmxtransporters.VehicleListProps{
		Vehicles:      vehicles,
		ProjectID:     projectID,
		TransporterID: transporterID,
		CsrfToken:     csrf.Token(c.Request()),
	}))
}

func RemoveVehicleHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid transporter ID"})
	}

	vehicleID, err := strconv.Atoi(c.Param("vid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid vehicle ID"})
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Transporter not found"})
	}

	vehicle, err := database.GetVehicleByID(vehicleID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Vehicle not found"})
	}

	used, err := database.IsVehicleUsedInDC(vehicle.VehicleNumber)
	if err != nil {
		slog.Error("Error checking vehicle usage", slog.String("error", err.Error()), slog.Int("vehicleID", vehicleID))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to check vehicle usage"})
	}
	if used {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Cannot delete vehicle — it is used in one or more Delivery Challans"})
	}

	if err := database.DeleteVehicle(vehicleID, transporterID); err != nil {
		slog.Error("Error removing vehicle", slog.String("error", err.Error()), slog.Int("vehicleID", vehicleID))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to remove vehicle"})
	}

	vehicles, _ := database.GetVehiclesByTransporterID(transporterID)
	return components.RenderOK(c, htmxtransporters.HTMXVehicleList(htmxtransporters.VehicleListProps{
		Vehicles:      vehicles,
		ProjectID:     projectID,
		TransporterID: transporterID,
		CsrfToken:     csrf.Token(c.Request()),
	}))
}

// GetTransportersJSON is an API endpoint for DC form integration.
func GetTransportersJSON(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	transporters, err := database.GetTransportersByProjectID(projectID, true)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to load transporters"})
	}

	for _, t := range transporters {
		vehicles, _ := database.GetVehiclesByTransporterID(t.ID)
		t.Vehicles = vehicles
	}

	return c.JSON(http.StatusOK, transporters)
}
