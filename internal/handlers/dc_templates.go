package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	htmxdctemplates "github.com/narendhupati/dc-management-tool/components/htmx/dc_templates"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	dctemplates "github.com/narendhupati/dc-management-tool/components/pages/dc_templates"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func ListTemplates(c echo.Context) error {
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

	templates, err := database.GetTemplatesByProjectID(projectID)
	if err != nil {
		slog.Error("Error fetching templates", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		templates = []*models.DCTemplate{}
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request())

	pageContent := dctemplates.List(
		user,
		project,
		allProjects,
		templates,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("DC Templates", sidebar, topbar, flashMessage, flashType, pageContent))
}

func ShowCreateTemplateForm(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	products, err := database.GetProductsByProjectID(projectID)
	if err != nil {
		slog.Error("Error fetching products", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		products = []*models.Product{}
	}

	return components.RenderOK(c, htmxdctemplates.DCTemplateForm(htmxdctemplates.DCTemplateFormProps{
		ProjectID:        projectID,
		Template:         models.DCTemplate{},
		IsEdit:           false,
		Errors:           map[string]string{},
		Products:         products,
		SelectedProducts: map[int]int{},
		CsrfToken:        csrf.Token(c.Request()),
	}))
}

func CreateTemplateHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	tmpl := &models.DCTemplate{
		ProjectID: projectID,
		Name:      strings.TrimSpace(c.FormValue("name")),
		Purpose:   strings.TrimSpace(c.FormValue("purpose")),
	}

	errors := helpers.ValidateStruct(tmpl)

	// Parse selected products
	if err := c.Request().ParseForm(); err != nil {
		slog.Error("Error parsing form", slog.String("error", err.Error()))
	}
	productIDs := c.Request().PostForm["product_ids"]
	var products []database.TemplateProductInput
	selectedProducts := make(map[int]int)

	for i, pidStr := range productIDs {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		qtyStr := c.FormValue(fmt.Sprintf("quantity_%d", pid))
		qty, _ := strconv.Atoi(qtyStr)
		if qty < 1 {
			qty = 1
		}
		products = append(products, database.TemplateProductInput{ProductID: pid, DefaultQuantity: qty, SortOrder: i})
		selectedProducts[pid] = qty
	}

	if len(products) == 0 {
		errors["products"] = "At least one product must be selected"
	}

	// Check uniqueness
	if _, ok := errors["name"]; !ok && tmpl.Name != "" {
		unique, err := database.CheckTemplateNameUnique(projectID, tmpl.Name, 0)
		if err != nil {
			slog.Error("Error checking template name uniqueness", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		} else if !unique {
			errors["name"] = "A template with this name already exists in this project"
		}
	}

	allProducts, _ := database.GetProductsByProjectID(projectID)

	if len(errors) > 0 {
		return components.RenderOK(c, htmxdctemplates.DCTemplateForm(htmxdctemplates.DCTemplateFormProps{
			ProjectID:        projectID,
			Template:         *tmpl,
			IsEdit:           false,
			Errors:           errors,
			Products:         allProducts,
			SelectedProducts: selectedProducts,
			CsrfToken:        csrf.Token(c.Request()),
		}))
	}

	if err := database.CreateTemplate(tmpl, products); err != nil {
		slog.Error("Error creating template", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		errors["general"] = "Failed to create template"
		return components.RenderOK(c, htmxdctemplates.DCTemplateForm(htmxdctemplates.DCTemplateFormProps{
			ProjectID:        projectID,
			Template:         *tmpl,
			IsEdit:           false,
			Errors:           errors,
			Products:         allProducts,
			SelectedProducts: selectedProducts,
			CsrfToken:        csrf.Token(c.Request()),
		}))
	}

	c.Response().Header().Set("HX-Trigger", "templateChanged")
	return components.RenderOK(c, htmxdctemplates.DCTemplateFormSuccess(htmxdctemplates.DCTemplateFormSuccessProps{
		Message: "Template created successfully",
	}))
}

func ShowTemplateDetail(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/templates", projectID))
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request(), "error", "Template not found")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/templates", projectID))
	}

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		slog.Error("Error fetching template products", slog.String("error", err.Error()), slog.Int("templateID", templateID))
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request())

	pageContent := dctemplates.Detail(
		user,
		project,
		allProjects,
		tmpl,
		products,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Template Details", sidebar, topbar, flashMessage, flashType, pageContent))
}

func ShowEditTemplateForm(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid template ID")
	}

	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		return c.String(http.StatusNotFound, "Template not found")
	}

	// Prevent editing templates that have been used in DCs
	if hasDCs, count, _ := database.CheckTemplateHasDCs(templateID); hasDCs {
		return c.String(http.StatusForbidden, fmt.Sprintf("Cannot edit template: %d DCs have been issued using this template", count))
	}

	products, err := database.GetProductsByProjectID(projectID)
	if err != nil {
		slog.Error("Error fetching products", slog.String("error", err.Error()), slog.Int("projectID", projectID))
		products = []*models.Product{}
	}

	selectedProducts, err := database.GetTemplateProductIDs(templateID)
	if err != nil {
		slog.Error("Error fetching template product IDs", slog.String("error", err.Error()), slog.Int("templateID", templateID))
		selectedProducts = map[int]int{}
	}

	return components.RenderOK(c, htmxdctemplates.DCTemplateForm(htmxdctemplates.DCTemplateFormProps{
		ProjectID:        projectID,
		Template:         *tmpl,
		IsEdit:           true,
		Errors:           map[string]string{},
		Products:         products,
		SelectedProducts: selectedProducts,
		CsrfToken:        csrf.Token(c.Request()),
	}))
}

func UpdateTemplateHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid project ID")
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid template ID")
	}

	existing, err := database.GetTemplateByID(templateID)
	if err != nil || existing.ProjectID != projectID {
		return c.String(http.StatusNotFound, "Template not found")
	}

	// Prevent updating templates that have been used in DCs
	if hasDCs, count, _ := database.CheckTemplateHasDCs(templateID); hasDCs {
		return c.String(http.StatusForbidden, fmt.Sprintf("Cannot edit template: %d DCs have been issued using this template", count))
	}

	tmpl := &models.DCTemplate{
		ID:        templateID,
		ProjectID: projectID,
		Name:      strings.TrimSpace(c.FormValue("name")),
		Purpose:   strings.TrimSpace(c.FormValue("purpose")),
	}

	errors := helpers.ValidateStruct(tmpl)

	if err := c.Request().ParseForm(); err != nil {
		slog.Error("Error parsing form", slog.String("error", err.Error()))
	}
	productIDs := c.Request().PostForm["product_ids"]
	var products []database.TemplateProductInput
	selectedProducts := make(map[int]int)

	for i, pidStr := range productIDs {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		qtyStr := c.FormValue(fmt.Sprintf("quantity_%d", pid))
		qty, _ := strconv.Atoi(qtyStr)
		if qty < 1 {
			qty = 1
		}
		products = append(products, database.TemplateProductInput{ProductID: pid, DefaultQuantity: qty, SortOrder: i})
		selectedProducts[pid] = qty
	}

	if len(products) == 0 {
		errors["products"] = "At least one product must be selected"
	}

	if _, ok := errors["name"]; !ok && tmpl.Name != "" {
		unique, err := database.CheckTemplateNameUnique(projectID, tmpl.Name, templateID)
		if err != nil {
			slog.Error("Error checking template name uniqueness", slog.String("error", err.Error()), slog.Int("projectID", projectID), slog.Int("templateID", templateID))
		} else if !unique {
			errors["name"] = "A template with this name already exists in this project"
		}
	}

	allProducts, _ := database.GetProductsByProjectID(projectID)

	if len(errors) > 0 {
		return components.RenderOK(c, htmxdctemplates.DCTemplateForm(htmxdctemplates.DCTemplateFormProps{
			ProjectID:        projectID,
			Template:         *tmpl,
			IsEdit:           true,
			Errors:           errors,
			Products:         allProducts,
			SelectedProducts: selectedProducts,
			CsrfToken:        csrf.Token(c.Request()),
		}))
	}

	if err := database.UpdateTemplate(tmpl, products); err != nil {
		slog.Error("Error updating template", slog.String("error", err.Error()), slog.Int("templateID", templateID), slog.Int("projectID", projectID))
		errors["general"] = "Failed to update template"
		return components.RenderOK(c, htmxdctemplates.DCTemplateForm(htmxdctemplates.DCTemplateFormProps{
			ProjectID:        projectID,
			Template:         *tmpl,
			IsEdit:           true,
			Errors:           errors,
			Products:         allProducts,
			SelectedProducts: selectedProducts,
			CsrfToken:        csrf.Token(c.Request()),
		}))
	}

	c.Response().Header().Set("HX-Trigger", "templateChanged")
	return components.RenderOK(c, htmxdctemplates.DCTemplateFormSuccess(htmxdctemplates.DCTemplateFormSuccessProps{
		Message: "Template updated successfully",
	}))
}

func DuplicateTemplateHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid template ID"})
	}

	newTemplate, err := database.DuplicateTemplate(templateID, projectID)
	if err != nil {
		slog.Error("Error duplicating template", slog.String("error", err.Error()), slog.Int("templateID", templateID), slog.Int("projectID", projectID))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	auth.SetFlash(c.Request(), "success", fmt.Sprintf("Template duplicated as '%s'", newTemplate.Name))
	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/projects/%d/templates/%d", projectID, newTemplate.ID))
	return c.JSON(http.StatusOK, map[string]interface{}{"success": true, "id": newTemplate.ID})
}

func DeleteTemplateHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid template ID"})
	}

	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "Template not found"})
	}

	if err := database.DeleteTemplate(templateID, projectID); err != nil {
		slog.Error("Error deleting template", slog.String("error", err.Error()), slog.Int("templateID", templateID), slog.Int("projectID", projectID))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	c.Response().Header().Set("HX-Trigger", "templateChanged")
	return c.JSON(http.StatusOK, map[string]interface{}{"success": true, "message": "Template deleted successfully"})
}
