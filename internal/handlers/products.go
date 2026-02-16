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

func ListProducts(c *gin.Context) {
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

	products, err := database.GetProductsByProjectID(projectID)
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		products = []*models.Product{}
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Products", URL: ""},
	)

	c.HTML(http.StatusOK, "products/list.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"project":      project,
		"products":     products,
		"activeTab":    "products",
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"csrfToken":    csrf.Token(c.Request),
		"csrfField":    csrf.TemplateField(c.Request),
	})
}

func ShowAddProductForm(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
		"projectID": projectID,
		"product":   &models.Product{UoM: "Nos"},
		"errors":    map[string]string{},
		"isEdit":    false,
		"csrfField": csrf.TemplateField(c.Request),
	})
}

func CreateProductHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	price, _ := strconv.ParseFloat(c.PostForm("per_unit_price"), 64)
	gst, _ := strconv.ParseFloat(c.PostForm("gst_percentage"), 64)

	product := &models.Product{
		ProjectID:       projectID,
		ItemName:        strings.TrimSpace(c.PostForm("item_name")),
		ItemDescription: strings.TrimSpace(c.PostForm("item_description")),
		HSNCode:         strings.TrimSpace(c.PostForm("hsn_code")),
		UoM:             strings.TrimSpace(c.PostForm("uom")),
		BrandModel:      strings.TrimSpace(c.PostForm("brand_model")),
		PerUnitPrice:    price,
		GSTPercentage:   gst,
	}

	errors := product.Validate()

	// Check uniqueness
	if _, ok := errors["item_name"]; !ok && product.ItemName != "" {
		unique, err := database.CheckProductNameUnique(projectID, product.ItemName, 0)
		if err != nil {
			log.Printf("Error checking uniqueness: %v", err)
		} else if !unique {
			errors["item_name"] = "A product with this name already exists in this project"
		}
	}

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID": projectID,
			"product":   product,
			"errors":    errors,
			"isEdit":    false,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.CreateProductRecord(product); err != nil {
		log.Printf("Error creating product: %v", err)
		errors["general"] = "Failed to create product"
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID": projectID,
			"product":   product,
			"errors":    errors,
			"isEdit":    false,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	// Return HX-Trigger to close slide-over and refresh product list
	c.Header("HX-Trigger", "productChanged")
	c.HTML(http.StatusOK, "htmx/products/form-success.html", gin.H{
		"message": "Product added successfully",
	})
}

func ShowEditProductForm(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid product ID")
		return
	}

	product, err := database.GetProductByID(productID)
	if err != nil || product.ProjectID != projectID {
		c.String(http.StatusNotFound, "Product not found")
		return
	}

	c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
		"projectID": projectID,
		"product":   product,
		"errors":    map[string]string{},
		"isEdit":    true,
		"csrfField": csrf.TemplateField(c.Request),
	})
}

func UpdateProductHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid project ID")
		return
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid product ID")
		return
	}

	existing, err := database.GetProductByID(productID)
	if err != nil || existing.ProjectID != projectID {
		c.String(http.StatusNotFound, "Product not found")
		return
	}

	price, _ := strconv.ParseFloat(c.PostForm("per_unit_price"), 64)
	gst, _ := strconv.ParseFloat(c.PostForm("gst_percentage"), 64)

	product := &models.Product{
		ID:              productID,
		ProjectID:       projectID,
		ItemName:        strings.TrimSpace(c.PostForm("item_name")),
		ItemDescription: strings.TrimSpace(c.PostForm("item_description")),
		HSNCode:         strings.TrimSpace(c.PostForm("hsn_code")),
		UoM:             strings.TrimSpace(c.PostForm("uom")),
		BrandModel:      strings.TrimSpace(c.PostForm("brand_model")),
		PerUnitPrice:    price,
		GSTPercentage:   gst,
	}

	errors := product.Validate()

	// Check uniqueness excluding current product
	if _, ok := errors["item_name"]; !ok && product.ItemName != "" {
		unique, err := database.CheckProductNameUnique(projectID, product.ItemName, productID)
		if err != nil {
			log.Printf("Error checking uniqueness: %v", err)
		} else if !unique {
			errors["item_name"] = "A product with this name already exists in this project"
		}
	}

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID": projectID,
			"product":   product,
			"errors":    errors,
			"isEdit":    true,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.UpdateProductRecord(product); err != nil {
		log.Printf("Error updating product: %v", err)
		errors["general"] = "Failed to update product"
		c.HTML(http.StatusOK, "htmx/products/form.html", gin.H{
			"projectID": projectID,
			"product":   product,
			"errors":    errors,
			"isEdit":    true,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	c.Header("HX-Trigger", "productChanged")
	c.HTML(http.StatusOK, "htmx/products/form-success.html", gin.H{
		"message": "Product updated successfully",
	})
}

func DeleteProductHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	productID, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := database.GetProductByID(productID)
	if err != nil || product.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if err := database.DeleteProductRecord(productID, projectID); err != nil {
		log.Printf("Error deleting product: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Trigger", "productChanged")
	c.String(http.StatusOK, "")
}
