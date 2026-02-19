package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	pageprojects "github.com/narendhupati/dc-management-tool/components/pages/projects"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func ShowProjectSelector(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projects, err := database.GetAccessibleProjects(user)
	if err != nil {
		slog.Error("Error fetching user projects", slog.String("error", err.Error()), slog.Int("userID", user.ID))
		projects = []*models.Project{}
	}

	if len(projects) == 0 {
		if user.IsAdmin() {
			return c.Redirect(http.StatusFound, "/projects/new")
		}
		return components.RenderOK(c, pageprojects.Select(user, projects))
	}

	return components.RenderOK(c, pageprojects.Select(user, projects))
}

func ListProjects(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	var projects []*models.Project
	var err error
	if user.IsAdmin() {
		projects, err = database.GetAllProjects()
	} else {
		projects, err = database.GetAccessibleProjects(user)
	}
	if err != nil {
		slog.Error("Error fetching projects", slog.String("error", err.Error()), slog.Int("userID", user.ID))
		return c.Redirect(http.StatusFound, "/projects/select")
	}

	// Filter by search query if provided
	q := strings.TrimSpace(c.QueryParam("q"))
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

	flashType, flashMessage := auth.PopFlash(c.Request())

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageprojects.List(
		user,
		projects,
		nil, // currentProject — not project-scoped
		allProjects,
		flashType,
		flashMessage,
	)
	sidebar := partials.Sidebar(user, nil, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, nil, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Projects", sidebar, topbar, flashMessage, flashType, pageContent))
}

func ShowProjectForm(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	allProjects, _ := database.GetAccessibleProjects(user)

	formData := map[string]string{
		"company_gstin": "36AACCF9742K1Z8",
	}

	pageContent := pageprojects.CreateWizard(
		user,
		allProjects,
		1, // currentStep
		map[string]string{},
		csrf.Token(c.Request()),
		formData,
	)
	sidebar := partials.Sidebar(user, nil, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, nil, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Create Project", sidebar, topbar, "", "", pageContent))
}

func CreateProject(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	project := buildProjectFromForm(c)
	project.CreatedBy = user.ID

	errors := validateProject(project)

	// Handle file uploads
	handleProjectFileUploads(c, project, errors)

	allProjects, _ := database.GetAccessibleProjects(user)

	if len(errors) > 0 {
		formData := projectToFormData(project)
		pageContent := pageprojects.CreateWizard(
			user,
			allProjects,
			1,
			errors,
			csrf.Token(c.Request()),
			formData,
		)
		sidebar := partials.Sidebar(user, nil, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, nil, allProjects, "", "")
		return components.RenderOK(c, layouts.MainWithContent("Create Project", sidebar, topbar, "", "", pageContent))
	}

	if err := database.CreateProject(project); err != nil {
		slog.Error("Error creating project", slog.String("error", err.Error()))
		errors["general"] = "Failed to create project"
		formData := projectToFormData(project)
		pageContent := pageprojects.CreateWizard(
			user,
			allProjects,
			1,
			errors,
			csrf.Token(c.Request()),
			formData,
		)
		sidebar := partials.Sidebar(user, nil, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, nil, allProjects, "", "")
		return components.RenderOK(c, layouts.MainWithContent("Create Project", sidebar, topbar, "", "", pageContent))
	}

	auth.SetFlash(c.Request(), "success", "Project created successfully")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", project.ID))
}

func ShowEditProjectForm(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	project, err := database.GetProjectByID(id)
	if err != nil {
		slog.Error("Error fetching project", slog.String("error", err.Error()), slog.Int("projectID", id))
		auth.SetFlash(c.Request(), "error", "Project not found")
		return c.Redirect(http.StatusFound, "/projects")
	}

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageprojects.Form(
		user,
		project,
		allProjects,
		project,
		map[string]string{},
		csrf.Token(c.Request()),
		"",
		"",
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, "", "")
	return components.RenderOK(c, layouts.MainWithContent("Edit Project", sidebar, topbar, "", "", pageContent))
}

func UpdateProject(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	existing, err := database.GetProjectByID(id)
	if err != nil {
		auth.SetFlash(c.Request(), "error", "Project not found")
		return c.Redirect(http.StatusFound, "/projects")
	}

	project := buildProjectFromForm(c)
	project.ID = id
	project.CompanySignaturePath = existing.CompanySignaturePath
	project.CompanySealPath = existing.CompanySealPath
	project.CreatedBy = existing.CreatedBy

	errors := validateProject(project)

	// Handle file uploads
	handleProjectFileUploads(c, project, errors)

	allProjects, _ := database.GetAccessibleProjects(user)

	if len(errors) > 0 {
		pageContent := pageprojects.Form(
			user,
			project,
			allProjects,
			project,
			errors,
			csrf.Token(c.Request()),
			"",
			"",
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "", "")
		return components.RenderOK(c, layouts.MainWithContent("Edit Project", sidebar, topbar, "", "", pageContent))
	}

	if err := database.UpdateProject(project); err != nil {
		slog.Error("Error updating project", slog.String("error", err.Error()), slog.Int("projectID", id))
		errors["general"] = "Failed to update project"
		pageContent := pageprojects.Form(
			user,
			project,
			allProjects,
			project,
			errors,
			csrf.Token(c.Request()),
			"",
			"",
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "", "")
		return components.RenderOK(c, layouts.MainWithContent("Edit Project", sidebar, topbar, "", "", pageContent))
	}

	auth.SetFlash(c.Request(), "success", "Project updated successfully")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", project.ID))
}

func ShowProject(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	project, err := database.GetProjectByID(id)
	if err != nil {
		slog.Error("Error fetching project", slog.String("error", err.Error()), slog.Int("projectID", id))
		auth.SetFlash(c.Request(), "error", "Project not found")
		return c.Redirect(http.StatusFound, "/projects")
	}

	flashType, flashMessage := auth.PopFlash(c.Request())

	allProjects, _ := database.GetAccessibleProjects(user)

	pageContent := pageprojects.Detail(
		user,
		project,
		allProjects,
		map[string]int64{},
		flashType,
		flashMessage,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Project Details", sidebar, topbar, flashMessage, flashType, pageContent))
}

func DeleteProject(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	canDelete, err := database.CanDeleteProject(id)
	if err != nil {
		slog.Error("Error checking delete eligibility", slog.String("error", err.Error()), slog.Int("projectID", id))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to check project"})
	}

	if !canDelete {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Cannot delete project with issued delivery challans"})
	}

	if err := database.DeleteProject(id); err != nil {
		slog.Error("Error deleting project", slog.String("error", err.Error()), slog.Int("projectID", id))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to delete project"})
	}

	auth.SetFlash(c.Request(), "success", "Project deleted successfully")
	return c.JSON(http.StatusOK, map[string]interface{}{"success": true, "redirect": "/projects"})
}

// validateProject runs validation on a project and returns a map of errors.
func validateProject(project *models.Project) map[string]string {
	errors := make(map[string]string)
	if strings.TrimSpace(project.Name) == "" {
		errors["name"] = "Project name is required"
	}
	if strings.TrimSpace(project.DCPrefix) == "" {
		errors["dc_prefix"] = "DC prefix is required"
	}
	if project.SeqPadding != 0 && (project.SeqPadding < 2 || project.SeqPadding > 6) {
		errors["seq_padding"] = "Sequence padding must be between 2 and 6"
	}
	return errors
}

// projectToFormData converts a project struct into the formData map expected by CreateWizard.
func projectToFormData(p *models.Project) map[string]string {
	fd := map[string]string{
		"name":                  p.Name,
		"description":           p.Description,
		"dc_prefix":             p.DCPrefix,
		"bill_from_address":     p.BillFromAddress,
		"dispatch_from_address": p.DispatchFromAddress,
		"company_gstin":         p.CompanyGSTIN,
		"company_email":         p.CompanyEmail,
		"company_cin":           p.CompanyCIN,
		"dc_number_format":      p.DCNumberFormat,
		"dc_number_separator":   p.DCNumberSeparator,
		"purpose_text":          p.PurposeText,
		"tender_ref_number":     p.TenderRefNumber,
		"tender_ref_details":    p.TenderRefDetails,
		"po_reference":          p.POReference,
	}
	if p.PODate != nil {
		fd["po_date"] = *p.PODate
	}
	if p.SeqPadding != 0 {
		fd["seq_padding"] = strconv.Itoa(p.SeqPadding)
	}
	return fd
}

func buildProjectFromForm(c echo.Context) *models.Project {
	project := &models.Project{
		Name:                c.FormValue("name"),
		Description:         c.FormValue("description"),
		DCPrefix:            strings.ToUpper(strings.TrimSpace(c.FormValue("dc_prefix"))),
		TenderRefNumber:     c.FormValue("tender_ref_number"),
		TenderRefDetails:    c.FormValue("tender_ref_details"),
		POReference:         c.FormValue("po_reference"),
		BillFromAddress:     c.FormValue("bill_from_address"),
		DispatchFromAddress: c.FormValue("dispatch_from_address"),
		CompanyGSTIN:        strings.ToUpper(strings.TrimSpace(c.FormValue("company_gstin"))),
		CompanyEmail:        strings.TrimSpace(c.FormValue("company_email")),
		CompanyCIN:          strings.TrimSpace(c.FormValue("company_cin")),
		DCNumberFormat:      c.FormValue("dc_number_format"),
		DCNumberSeparator:   c.FormValue("dc_number_separator"),
		PurposeText:         c.FormValue("purpose_text"),
	}

	poDate := c.FormValue("po_date")
	if poDate != "" {
		project.PODate = &poDate
	}

	if padding := c.FormValue("seq_padding"); padding != "" {
		if p, err := strconv.Atoi(padding); err == nil {
			project.SeqPadding = p
		}
	}

	return project
}

func handleProjectFileUploads(c echo.Context, project *models.Project, errors map[string]string) {
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
	if mkdirErr := os.MkdirAll(uploadDir, 0o755); mkdirErr != nil {
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
