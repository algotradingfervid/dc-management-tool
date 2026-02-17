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

func ShowProjectSelector(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projects, err := database.GetAccessibleProjects(user)
	if err != nil {
		log.Printf("Error fetching user projects: %v", err)
		projects = []*models.Project{}
	}

	if len(projects) == 0 {
		if user.IsAdmin() {
			c.Redirect(http.StatusFound, "/projects/new")
		} else {
			c.HTML(http.StatusOK, "projects/select.html", gin.H{
				"user":     user,
				"projects": projects,
			})
		}
		return
	}

	c.HTML(http.StatusOK, "projects/select.html", gin.H{
		"user":     user,
		"projects": projects,
	})
}

func ListProjects(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	var projects []*models.Project
	var err error
	if user.IsAdmin() {
		projects, err = database.GetAllProjects()
	} else {
		projects, err = database.GetAccessibleProjects(user)
	}
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

	c.HTML(http.StatusOK, "projects/create-wizard.html", gin.H{
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

	project := buildProjectFromForm(c)
	project.CreatedBy = user.ID

	errors := project.Validate()

	// Handle file uploads
	handleProjectFileUploads(c, project, errors)

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "projects/create-wizard.html", gin.H{
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
		c.HTML(http.StatusOK, "projects/create-wizard.html", gin.H{
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

	project := buildProjectFromForm(c)
	project.ID = id
	project.CompanySignaturePath = existing.CompanySignaturePath
	project.CompanySealPath = existing.CompanySealPath
	project.CreatedBy = existing.CreatedBy

	errors := project.Validate()

	// Handle file uploads
	handleProjectFileUploads(c, project, errors)

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

func buildProjectFromForm(c *gin.Context) *models.Project {
	project := &models.Project{
		Name:                c.PostForm("name"),
		Description:         c.PostForm("description"),
		DCPrefix:            strings.ToUpper(strings.TrimSpace(c.PostForm("dc_prefix"))),
		TenderRefNumber:     c.PostForm("tender_ref_number"),
		TenderRefDetails:    c.PostForm("tender_ref_details"),
		POReference:         c.PostForm("po_reference"),
		BillFromAddress:     c.PostForm("bill_from_address"),
		DispatchFromAddress: c.PostForm("dispatch_from_address"),
		CompanyGSTIN:        strings.ToUpper(strings.TrimSpace(c.PostForm("company_gstin"))),
		CompanyEmail:        strings.TrimSpace(c.PostForm("company_email")),
		CompanyCIN:          strings.TrimSpace(c.PostForm("company_cin")),
		DCNumberFormat:      c.PostForm("dc_number_format"),
		DCNumberSeparator:   c.PostForm("dc_number_separator"),
		PurposeText:         c.PostForm("purpose_text"),
	}

	poDate := c.PostForm("po_date")
	if poDate != "" {
		project.PODate = &poDate
	}

	if padding := c.PostForm("seq_padding"); padding != "" {
		if p, err := strconv.Atoi(padding); err == nil {
			project.SeqPadding = p
		}
	}

	return project
}

func handleProjectFileUploads(c *gin.Context, project *models.Project, errors map[string]string) {
	// Handle signature upload
	if file, err := c.FormFile("company_signature"); err == nil {
		path, uploadErr := handleImageUpload(file, "sig")
		if uploadErr != nil {
			errors["company_signature"] = uploadErr.Error()
		} else {
			project.CompanySignaturePath = path
		}
	}

	// Handle seal upload
	if file, err := c.FormFile("company_seal"); err == nil {
		path, uploadErr := handleImageUpload(file, "seal")
		if uploadErr != nil {
			errors["company_seal"] = uploadErr.Error()
		} else {
			project.CompanySealPath = path
		}
	}
}

func handleImageUpload(file *multipart.FileHeader, prefix string) (string, error) {
	return handleSignatureUpload(file, prefix)
}

func handleSignatureUpload(file *multipart.FileHeader, prefixes ...string) (string, error) {
	prefix := "sig"
	if len(prefixes) > 0 {
		prefix = prefixes[0]
	}
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
	filename := fmt.Sprintf("%s_%d%s", prefix, time.Now().UnixNano(), ext)

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
