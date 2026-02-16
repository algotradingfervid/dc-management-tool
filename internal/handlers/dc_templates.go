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

func ListTemplates(c *gin.Context) {
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

	templates, err := database.GetTemplatesByProjectID(projectID)
	if err != nil {
		log.Printf("Error fetching templates: %v", err)
		templates = []*models.DCTemplate{}
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Templates", URL: ""},
	)

	c.HTML(http.StatusOK, "dc_templates/list.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"project":      project,
		"templates":    templates,
		"activeTab":    "templates",
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"csrfToken":    csrf.Token(c.Request),
		"csrfField":    csrf.TemplateField(c.Request),
	})
}

func ShowCreateTemplateForm(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	products, err := database.GetProductsByProjectID(projectID)
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		products = []*models.Product{}
	}

	c.HTML(http.StatusOK, "htmx/dc_templates/form.html", gin.H{
		"projectID":          projectID,
		"template":           &models.DCTemplate{},
		"products":           products,
		"selectedProducts":   map[int]int{},
		"errors":             map[string]string{},
		"isEdit":             false,
		"csrfField":          csrf.TemplateField(c.Request),
	})
}

func CreateTemplateHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	tmpl := &models.DCTemplate{
		ProjectID: projectID,
		Name:      strings.TrimSpace(c.PostForm("name")),
		Purpose:   strings.TrimSpace(c.PostForm("purpose")),
	}

	errors := tmpl.Validate()

	// Parse selected products
	productIDs := c.PostFormArray("product_ids")
	var products []database.TemplateProductInput
	selectedProducts := make(map[int]int)

	for _, pidStr := range productIDs {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		qtyStr := c.PostForm(fmt.Sprintf("quantity_%d", pid))
		qty, _ := strconv.Atoi(qtyStr)
		if qty < 1 {
			qty = 1
		}
		products = append(products, database.TemplateProductInput{ProductID: pid, DefaultQuantity: qty})
		selectedProducts[pid] = qty
	}

	if len(products) == 0 {
		errors["products"] = "At least one product must be selected"
	}

	// Check uniqueness
	if _, ok := errors["name"]; !ok && tmpl.Name != "" {
		unique, err := database.CheckTemplateNameUnique(projectID, tmpl.Name, 0)
		if err != nil {
			log.Printf("Error checking uniqueness: %v", err)
		} else if !unique {
			errors["name"] = "A template with this name already exists in this project"
		}
	}

	allProducts, _ := database.GetProductsByProjectID(projectID)

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/dc_templates/form.html", gin.H{
			"projectID":        projectID,
			"template":         tmpl,
			"products":         allProducts,
			"selectedProducts": selectedProducts,
			"errors":           errors,
			"isEdit":           false,
			"csrfField":        csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.CreateTemplate(tmpl, products); err != nil {
		log.Printf("Error creating template: %v", err)
		errors["general"] = "Failed to create template"
		c.HTML(http.StatusOK, "htmx/dc_templates/form.html", gin.H{
			"projectID":        projectID,
			"template":         tmpl,
			"products":         allProducts,
			"selectedProducts": selectedProducts,
			"errors":           errors,
			"isEdit":           false,
			"csrfField":        csrf.TemplateField(c.Request),
		})
		return
	}

	c.Header("HX-Trigger", "templateChanged")
	c.HTML(http.StatusOK, "htmx/dc_templates/form-success.html", gin.H{
		"message": "Template created successfully",
	})
}

func ShowTemplateDetail(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/templates", projectID))
		return
	}

	project, err := database.GetProjectByID(projectID)
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		auth.SetFlash(c.Request, "error", "Template not found")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/templates", projectID))
		return
	}

	products, err := database.GetTemplateProducts(templateID)
	if err != nil {
		log.Printf("Error fetching template products: %v", err)
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Templates", URL: fmt.Sprintf("/projects/%d/templates", project.ID)},
		helpers.Breadcrumb{Title: tmpl.Name, URL: ""},
	)

	c.HTML(http.StatusOK, "dc_templates/detail.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"project":      project,
		"template":     tmpl,
		"products":     products,
		"activeTab":    "templates",
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"csrfToken":    csrf.Token(c.Request),
		"csrfField":    csrf.TemplateField(c.Request),
	})
}

func ShowEditTemplateForm(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid template ID")
		return
	}

	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		c.String(http.StatusNotFound, "Template not found")
		return
	}

	products, err := database.GetProductsByProjectID(projectID)
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		products = []*models.Product{}
	}

	selectedProducts, err := database.GetTemplateProductIDs(templateID)
	if err != nil {
		log.Printf("Error fetching template product IDs: %v", err)
		selectedProducts = map[int]int{}
	}

	c.HTML(http.StatusOK, "htmx/dc_templates/form.html", gin.H{
		"projectID":        projectID,
		"template":         tmpl,
		"products":         products,
		"selectedProducts": selectedProducts,
		"errors":           map[string]string{},
		"isEdit":           true,
		"csrfField":        csrf.TemplateField(c.Request),
	})
}

func UpdateTemplateHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid template ID")
		return
	}

	existing, err := database.GetTemplateByID(templateID)
	if err != nil || existing.ProjectID != projectID {
		c.String(http.StatusNotFound, "Template not found")
		return
	}

	tmpl := &models.DCTemplate{
		ID:        templateID,
		ProjectID: projectID,
		Name:      strings.TrimSpace(c.PostForm("name")),
		Purpose:   strings.TrimSpace(c.PostForm("purpose")),
	}

	errors := tmpl.Validate()

	productIDs := c.PostFormArray("product_ids")
	var products []database.TemplateProductInput
	selectedProducts := make(map[int]int)

	for _, pidStr := range productIDs {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		qtyStr := c.PostForm(fmt.Sprintf("quantity_%d", pid))
		qty, _ := strconv.Atoi(qtyStr)
		if qty < 1 {
			qty = 1
		}
		products = append(products, database.TemplateProductInput{ProductID: pid, DefaultQuantity: qty})
		selectedProducts[pid] = qty
	}

	if len(products) == 0 {
		errors["products"] = "At least one product must be selected"
	}

	if _, ok := errors["name"]; !ok && tmpl.Name != "" {
		unique, err := database.CheckTemplateNameUnique(projectID, tmpl.Name, templateID)
		if err != nil {
			log.Printf("Error checking uniqueness: %v", err)
		} else if !unique {
			errors["name"] = "A template with this name already exists in this project"
		}
	}

	allProducts, _ := database.GetProductsByProjectID(projectID)

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/dc_templates/form.html", gin.H{
			"projectID":        projectID,
			"template":         tmpl,
			"products":         allProducts,
			"selectedProducts": selectedProducts,
			"errors":           errors,
			"isEdit":           true,
			"csrfField":        csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.UpdateTemplate(tmpl, products); err != nil {
		log.Printf("Error updating template: %v", err)
		errors["general"] = "Failed to update template"
		c.HTML(http.StatusOK, "htmx/dc_templates/form.html", gin.H{
			"projectID":        projectID,
			"template":         tmpl,
			"products":         allProducts,
			"selectedProducts": selectedProducts,
			"errors":           errors,
			"isEdit":           true,
			"csrfField":        csrf.TemplateField(c.Request),
		})
		return
	}

	c.Header("HX-Trigger", "templateChanged")
	c.HTML(http.StatusOK, "htmx/dc_templates/form-success.html", gin.H{
		"message": "Template updated successfully",
	})
}

func DeleteTemplateHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	templateID, err := strconv.Atoi(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	tmpl, err := database.GetTemplateByID(templateID)
	if err != nil || tmpl.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	if err := database.DeleteTemplate(templateID, projectID); err != nil {
		log.Printf("Error deleting template: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Trigger", "templateChanged")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Template deleted successfully"})
}
