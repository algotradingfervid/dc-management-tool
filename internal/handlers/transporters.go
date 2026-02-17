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
)

func ListTransporters(c *gin.Context) {
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

	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	transporterPage, err := database.SearchTransporters(projectID, search, page, 20)
	if err != nil {
		log.Printf("Error fetching transporters: %v", err)
		transporterPage = &models.TransporterPage{Transporters: []*models.Transporter{}, CurrentPage: 1, TotalPages: 1, PerPage: 20}
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Transporters", URL: ""},
	)

	data := gin.H{
		"user":            user,
		"currentPath":     c.Request.URL.Path,
		"currentProject":  project,
		"breadcrumbs":     breadcrumbs,
		"project":         project,
		"transporterPage": transporterPage,
		"transporters":    transporterPage.Transporters,
		"search":          search,
		"activeTab":       "transporters",
		"flashType":       flashType,
		"flashMessage":    flashMessage,
		"csrfToken":       csrf.Token(c.Request),
		"csrfField":       csrf.TemplateField(c.Request),
	}

	c.HTML(http.StatusOK, "transporters/list.html", data)
}

func ShowAddTransporterForm(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	c.HTML(http.StatusOK, "htmx/transporters/form.html", gin.H{
		"projectID":   projectID,
		"transporter": &models.Transporter{IsActive: true},
		"errors":      map[string]string{},
		"isEdit":      false,
		"csrfField":   csrf.TemplateField(c.Request),
	})
}

func CreateTransporterHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	transporter := &models.Transporter{
		ProjectID:     projectID,
		CompanyName:   strings.TrimSpace(c.PostForm("company_name")),
		ContactPerson: strings.TrimSpace(c.PostForm("contact_person")),
		Phone:         strings.TrimSpace(c.PostForm("phone")),
		GSTNumber:     strings.TrimSpace(c.PostForm("gst_number")),
		IsActive:      true,
	}

	errors := transporter.Validate()

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/transporters/form.html", gin.H{
			"projectID":   projectID,
			"transporter": transporter,
			"errors":      errors,
			"isEdit":      false,
			"csrfField":   csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.CreateTransporter(transporter); err != nil {
		log.Printf("Error creating transporter: %v", err)
		errors["general"] = "Failed to create transporter"
		c.HTML(http.StatusOK, "htmx/transporters/form.html", gin.H{
			"projectID":   projectID,
			"transporter": transporter,
			"errors":      errors,
			"isEdit":      false,
			"csrfField":   csrf.TemplateField(c.Request),
		})
		return
	}

	c.Header("HX-Trigger", "transporterChanged")
	c.HTML(http.StatusOK, "htmx/transporters/form-success.html", gin.H{
		"message": "Transporter added successfully",
	})
}

func ShowEditTransporterForm(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid transporter ID")
		return
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		c.String(http.StatusNotFound, "Transporter not found")
		return
	}

	c.HTML(http.StatusOK, "htmx/transporters/form.html", gin.H{
		"projectID":   projectID,
		"transporter": transporter,
		"errors":      map[string]string{},
		"isEdit":      true,
		"csrfField":   csrf.TemplateField(c.Request),
	})
}

func UpdateTransporterHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid transporter ID")
		return
	}

	existing, err := database.GetTransporterByID(transporterID)
	if err != nil || existing.ProjectID != projectID {
		c.String(http.StatusNotFound, "Transporter not found")
		return
	}

	transporter := &models.Transporter{
		ID:            transporterID,
		ProjectID:     projectID,
		CompanyName:   strings.TrimSpace(c.PostForm("company_name")),
		ContactPerson: strings.TrimSpace(c.PostForm("contact_person")),
		Phone:         strings.TrimSpace(c.PostForm("phone")),
		GSTNumber:     strings.TrimSpace(c.PostForm("gst_number")),
		IsActive:      existing.IsActive,
	}

	errors := transporter.Validate()

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/transporters/form.html", gin.H{
			"projectID":   projectID,
			"transporter": transporter,
			"errors":      errors,
			"isEdit":      true,
			"csrfField":   csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.UpdateTransporter(transporter); err != nil {
		log.Printf("Error updating transporter: %v", err)
		errors["general"] = "Failed to update transporter"
		c.HTML(http.StatusOK, "htmx/transporters/form.html", gin.H{
			"projectID":   projectID,
			"transporter": transporter,
			"errors":      errors,
			"isEdit":      true,
			"csrfField":   csrf.TemplateField(c.Request),
		})
		return
	}

	c.Header("HX-Trigger", "transporterChanged")
	c.HTML(http.StatusOK, "htmx/transporters/form-success.html", gin.H{
		"message": "Transporter updated successfully",
	})
}

func ToggleTransporterStatus(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transporter ID"})
		return
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transporter not found"})
		return
	}

	if transporter.IsActive {
		err = database.DeactivateTransporter(transporterID, projectID)
	} else {
		err = database.ActivateTransporter(transporterID, projectID)
	}

	if err != nil {
		log.Printf("Error toggling transporter status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.Header("HX-Trigger", "transporterChanged")
	c.String(http.StatusOK, "")
}

func ShowTransporterDetail(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transporters", projectID))
		return
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		auth.SetFlash(c.Request, "error", "Transporter not found")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/transporters", projectID))
		return
	}

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Transporters", URL: fmt.Sprintf("/projects/%d/transporters", project.ID)},
		helpers.Breadcrumb{Title: transporter.CompanyName, URL: ""},
	)

	flashType, flashMessage := auth.PopFlash(c.Request)

	c.HTML(http.StatusOK, "transporters/detail.html", gin.H{
		"user":           user,
		"currentPath":    c.Request.URL.Path,
		"currentProject": project,
		"breadcrumbs":    breadcrumbs,
		"project":        project,
		"transporter":    transporter,
		"vehicles":       transporter.Vehicles,
		"flashType":      flashType,
		"flashMessage":   flashMessage,
		"csrfToken":      csrf.Token(c.Request),
		"csrfField":      csrf.TemplateField(c.Request),
	})
}

func AddVehicleHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transporter ID"})
		return
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transporter not found"})
		return
	}

	vehicle := &models.TransporterVehicle{
		TransporterID: transporterID,
		VehicleNumber: strings.TrimSpace(c.PostForm("vehicle_number")),
		VehicleType:   strings.TrimSpace(c.PostForm("vehicle_type")),
		DriverName:    strings.TrimSpace(c.PostForm("driver_name")),
		DriverPhone1:  strings.TrimSpace(c.PostForm("driver_phone1")),
		DriverPhone2:  strings.TrimSpace(c.PostForm("driver_phone2")),
	}

	if vehicle.VehicleType == "" {
		vehicle.VehicleType = "truck"
	}

	errors := vehicle.Validate()
	if len(errors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors["vehicle_number"]})
		return
	}

	if err := database.CreateVehicle(vehicle); err != nil {
		log.Printf("Error adding vehicle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add vehicle"})
		return
	}

	// Return updated vehicle list partial
	vehicles, _ := database.GetVehiclesByTransporterID(transporterID)
	c.HTML(http.StatusOK, "htmx/transporters/vehicle-list.html", gin.H{
		"vehicles":    vehicles,
		"projectID":   projectID,
		"transporterID": transporterID,
		"csrfToken":   csrf.Token(c.Request),
		"csrfField":   csrf.TemplateField(c.Request),
	})
}

func RemoveVehicleHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	transporterID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transporter ID"})
		return
	}

	vehicleID, err := strconv.Atoi(c.Param("vid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vehicle ID"})
		return
	}

	transporter, err := database.GetTransporterByID(transporterID)
	if err != nil || transporter.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transporter not found"})
		return
	}

	vehicle, err := database.GetVehicleByID(vehicleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	}

	used, err := database.IsVehicleUsedInDC(vehicle.VehicleNumber)
	if err != nil {
		log.Printf("Error checking vehicle usage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check vehicle usage"})
		return
	}
	if used {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete vehicle — it is used in one or more Delivery Challans"})
		return
	}

	if err := database.DeleteVehicle(vehicleID, transporterID); err != nil {
		log.Printf("Error removing vehicle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove vehicle"})
		return
	}

	// Return updated vehicle list partial
	vehicles, _ := database.GetVehiclesByTransporterID(transporterID)
	c.HTML(http.StatusOK, "htmx/transporters/vehicle-list.html", gin.H{
		"vehicles":    vehicles,
		"projectID":   projectID,
		"transporterID": transporterID,
		"csrfToken":   csrf.Token(c.Request),
		"csrfField":   csrf.TemplateField(c.Request),
	})
}

// API endpoint for DC form integration
func GetTransportersJSON(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	transporters, err := database.GetTransportersByProjectID(projectID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load transporters"})
		return
	}

	// Load vehicles for each
	for _, t := range transporters {
		vehicles, _ := database.GetVehiclesByTransporterID(t.ID)
		t.Vehicles = vehicles
	}

	c.JSON(http.StatusOK, transporters)
}
