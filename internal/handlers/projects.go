package handlers

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func ListProjects(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projects, err := database.GetAllProjects()
	if err != nil {
		log.Printf("Error fetching projects: %v", err)
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"user":  user,
			"error": "Failed to load projects",
		})
		return
	}

	// Filter by search query if provided
	q := strings.TrimSpace(c.Query("q"))
	if q != "" {
		q = strings.ToLower(q)
		var filtered []*models.Project
		for _, p := range projects {
			if strings.Contains(strings.ToLower(p.Name), q) ||
				strings.Contains(strings.ToLower(p.DCPrefix), q) ||
				strings.Contains(strings.ToLower(p.POReference), q) {
				filtered = append(filtered, p)
			}
		}
		projects = filtered
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: ""},
	)

	c.HTML(http.StatusOK, "projects/list.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"projects":     projects,
		"flashType":    flashType,
		"flashMessage": flashMessage,
	})
}

func ShowProjectForm(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: "New Project", URL: ""},
	)

	c.HTML(http.StatusOK, "projects/form.html", gin.H{
		"user":        user,
		"currentPath": c.Request.URL.Path,
		"breadcrumbs": breadcrumbs,
		"project":     &models.Project{CompanyGSTIN: "36AACCF9742K1Z8"},
		"errors":      map[string]string{},
		"isEdit":      false,
		"csrfField":   csrf.TemplateField(c.Request),
	})
}

func CreateProject(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	project := &models.Project{
		Name:             c.PostForm("name"),
		Description:      c.PostForm("description"),
		DCPrefix:         strings.ToUpper(strings.TrimSpace(c.PostForm("dc_prefix"))),
		TenderRefNumber:  c.PostForm("tender_ref_number"),
		TenderRefDetails: c.PostForm("tender_ref_details"),
		POReference:      c.PostForm("po_reference"),
		BillFromAddress:  c.PostForm("bill_from_address"),
		CompanyGSTIN:     strings.ToUpper(strings.TrimSpace(c.PostForm("company_gstin"))),
		CreatedBy:        user.ID,
	}

	poDate := c.PostForm("po_date")
	if poDate != "" {
		project.PODate = &poDate
	}

	errors := project.Validate()

	// Handle file upload
	file, err := c.FormFile("company_signature")
	if err == nil {
		path, uploadErr := handleSignatureUpload(file)
		if uploadErr != nil {
			errors["company_signature"] = uploadErr.Error()
		} else {
			project.CompanySignaturePath = path
		}
	}

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "projects/form.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": helpers.BuildBreadcrumbs(
				helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
				helpers.Breadcrumb{Title: "New Project", URL: ""},
			),
			"project":   project,
			"errors":    errors,
			"isEdit":    false,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.CreateProject(project); err != nil {
		log.Printf("Error creating project: %v", err)
		errors["general"] = "Failed to create project"
		c.HTML(http.StatusOK, "projects/form.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": helpers.BuildBreadcrumbs(
				helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
				helpers.Breadcrumb{Title: "New Project", URL: ""},
			),
			"project":   project,
			"errors":    errors,
			"isEdit":    false,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	auth.SetFlash(c.Request, "success", "Project created successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", project.ID))
}

func ShowEditProjectForm(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	project, err := database.GetProjectByID(id)
	if err != nil {
		log.Printf("Error fetching project: %v", err)
		auth.SetFlash(c.Request, "error", "Project not found")
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Edit", URL: ""},
	)

	c.HTML(http.StatusOK, "projects/form.html", gin.H{
		"user":        user,
		"currentPath": c.Request.URL.Path,
		"breadcrumbs": breadcrumbs,
		"project":     project,
		"errors":      map[string]string{},
		"isEdit":      true,
		"csrfField":   csrf.TemplateField(c.Request),
	})
}

func UpdateProject(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	existing, err := database.GetProjectByID(id)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Project not found")
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	project := &models.Project{
		ID:                   id,
		Name:                 c.PostForm("name"),
		Description:          c.PostForm("description"),
		DCPrefix:             strings.ToUpper(strings.TrimSpace(c.PostForm("dc_prefix"))),
		TenderRefNumber:      c.PostForm("tender_ref_number"),
		TenderRefDetails:     c.PostForm("tender_ref_details"),
		POReference:          c.PostForm("po_reference"),
		BillFromAddress:      c.PostForm("bill_from_address"),
		CompanyGSTIN:         strings.ToUpper(strings.TrimSpace(c.PostForm("company_gstin"))),
		CompanySignaturePath: existing.CompanySignaturePath,
		CreatedBy:            existing.CreatedBy,
	}

	poDate := c.PostForm("po_date")
	if poDate != "" {
		project.PODate = &poDate
	}

	errors := project.Validate()

	// Handle file upload
	file, uploadErr := c.FormFile("company_signature")
	if uploadErr == nil {
		path, err := handleSignatureUpload(file)
		if err != nil {
			errors["company_signature"] = err.Error()
		} else {
			project.CompanySignaturePath = path
		}
	}

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "projects/form.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": helpers.BuildBreadcrumbs(
				helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
				helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
				helpers.Breadcrumb{Title: "Edit", URL: ""},
			),
			"project":   project,
			"errors":    errors,
			"isEdit":    true,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.UpdateProject(project); err != nil {
		log.Printf("Error updating project: %v", err)
		errors["general"] = "Failed to update project"
		c.HTML(http.StatusOK, "projects/form.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": helpers.BuildBreadcrumbs(
				helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
				helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
				helpers.Breadcrumb{Title: "Edit", URL: ""},
			),
			"project":   project,
			"errors":    errors,
			"isEdit":    true,
			"csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	auth.SetFlash(c.Request, "success", "Project updated successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", project.ID))
}

func ShowProject(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	project, err := database.GetProjectByID(id)
	if err != nil {
		log.Printf("Error fetching project: %v", err)
		auth.SetFlash(c.Request, "error", "Project not found")
		c.Redirect(http.StatusFound, "/projects")
		return
	}

	flashType, flashMessage := auth.PopFlash(c.Request)
	activeTab := c.DefaultQuery("tab", "overview")

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: ""},
	)

	c.HTML(http.StatusOK, "projects/detail.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"project":      project,
		"activeTab":    activeTab,
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"csrfToken":    csrf.Token(c.Request),
	})
}

func DeleteProject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	canDelete, err := database.CanDeleteProject(id)
	if err != nil {
		log.Printf("Error checking delete: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project"})
		return
	}

	if !canDelete {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete project with issued delivery challans"})
		return
	}

	if err := database.DeleteProject(id); err != nil {
		log.Printf("Error deleting project: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	auth.SetFlash(c.Request, "success", "Project deleted successfully")
	c.JSON(http.StatusOK, gin.H{"success": true, "redirect": "/projects"})
}

func handleSignatureUpload(file *multipart.FileHeader) (string, error) {
	// Validate file size (2MB max)
	if file.Size > 2*1024*1024 {
		return "", fmt.Errorf("file size must be less than 2MB")
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		return "", fmt.Errorf("only JPG, PNG, and GIF files are allowed")
	}

	// Open source file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to read file")
	}
	defer src.Close()

	// Generate unique filename
	filename := fmt.Sprintf("sig_%d%s", time.Now().UnixNano(), ext)

	// Ensure uploads directory exists
	uploadDir := "./static/uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory")
	}

	// Create destination file
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		return "", fmt.Errorf("failed to save file")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file")
	}

	return filename, nil
}
