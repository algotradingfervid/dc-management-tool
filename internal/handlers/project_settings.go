package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	pageprojects "github.com/narendhupati/dc-management-tool/components/pages/projects"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

func ShowProjectSettings(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects")
	}

	project, err := database.GetProjectByID(id)
	if err != nil {
		slog.Error("Error fetching project", slog.Int("project_id", id), slog.String("error", err.Error()))
		auth.SetFlash(c.Request(), "error", "Project not found")
		return c.Redirect(http.StatusFound, "/projects")
	}

	flashType, flashMessage := auth.PopFlash(c.Request())

	allProjects, _ := database.GetAccessibleProjects(user)

	// Generate DC number preview (kept for reference; Settings templ handles its own preview)
	_ = services.PreviewDCNumber(project.DCNumberFormat, project.DCPrefix, project.DCPrefix, project.SeqPadding)

	pageContent := pageprojects.Settings(
		user,
		project,
		allProjects,
		map[string]string{},
		csrf.Token(c.Request()),
		flashType,
		flashMessage,
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Project Settings", sidebar, topbar, flashMessage, flashType, pageContent))
}

func UpdateProjectSettings(c echo.Context) error {
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

	tab := c.FormValue("tab")
	if tab == "" {
		tab = "general"
	}

	// Start with existing project and update only the relevant tab fields
	project := existing

	errors := make(map[string]string)

	switch tab {
	case "general":
		project.Name = c.FormValue("name")
		project.Description = c.FormValue("description")
		project.DCPrefix = strings.ToUpper(strings.TrimSpace(c.FormValue("dc_prefix")))
		if strings.TrimSpace(project.Name) == "" {
			errors["name"] = "Project name is required"
		}
		if strings.TrimSpace(project.DCPrefix) == "" {
			errors["dc_prefix"] = "DC prefix is required"
		}

	case "company":
		project.BillFromAddress = c.FormValue("bill_from_address")
		project.DispatchFromAddress = c.FormValue("dispatch_from_address")
		project.CompanyGSTIN = strings.ToUpper(strings.TrimSpace(c.FormValue("company_gstin")))
		project.CompanyEmail = strings.TrimSpace(c.FormValue("company_email"))
		project.CompanyCIN = strings.TrimSpace(c.FormValue("company_cin"))

		if project.CompanyGSTIN != "" && len(project.CompanyGSTIN) != 15 {
			errors["company_gstin"] = "GSTIN must be exactly 15 characters"
		}
		if project.CompanyEmail != "" && !strings.Contains(project.CompanyEmail, "@") {
			errors["company_email"] = "Invalid email address"
		}

		// Handle file uploads
		if file, fileErr := c.FormFile("company_signature"); fileErr == nil {
			path, uploadErr := handleImageUpload(file, "sig")
			if uploadErr != nil {
				errors["company_signature"] = uploadErr.Error()
			} else {
				project.CompanySignaturePath = path
			}
		}
		if file, fileErr := c.FormFile("company_seal"); fileErr == nil {
			path, uploadErr := handleImageUpload(file, "seal")
			if uploadErr != nil {
				errors["company_seal"] = uploadErr.Error()
			} else {
				project.CompanySealPath = path
			}
		}

	case "dc_config":
		project.DCNumberFormat = c.FormValue("dc_number_format")
		project.DCNumberSeparator = c.FormValue("dc_number_separator")
		project.PurposeText = c.FormValue("purpose_text")
		if padding := c.FormValue("seq_padding"); padding != "" {
			if p, convErr := strconv.Atoi(padding); convErr == nil {
				project.SeqPadding = p
			}
		}
		if project.SeqPadding < 2 || project.SeqPadding > 6 {
			errors["seq_padding"] = "Sequence padding must be between 2 and 6"
		}

	case "tender":
		project.TenderRefNumber = c.FormValue("tender_ref_number")
		project.TenderRefDetails = c.FormValue("tender_ref_details")
		project.POReference = c.FormValue("po_reference")
		poDate := c.FormValue("po_date")
		if poDate != "" {
			project.PODate = &poDate
		} else {
			project.PODate = nil
		}
	}

	if len(errors) > 0 {
		allProjects, _ := database.GetAccessibleProjects(user)
		pageContent := pageprojects.Settings(
			user,
			project,
			allProjects,
			errors,
			csrf.Token(c.Request()),
			"",
			"",
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "", "")
		return components.RenderOK(c, layouts.MainWithContent("Project Settings", sidebar, topbar, "", "", pageContent))
	}

	if err := database.UpdateProjectSettings(project, tab); err != nil {
		slog.Error("Error updating project settings", slog.Int("project_id", id), slog.String("tab", tab), slog.String("error", err.Error()))
		auth.SetFlash(c.Request(), "error", "Failed to save settings")
		return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/settings?tab=%s", id, tab))
	}

	auth.SetFlash(c.Request(), "success", "Settings saved successfully")
	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/settings?tab=%s", id, tab))
}

// PreviewDCNumberAPI returns a JSON preview of the DC number format.
func PreviewDCNumberAPI(c echo.Context) error {
	format := c.QueryParam("format")
	prefix := c.QueryParam("prefix")
	paddingStr := c.QueryParam("padding")
	if paddingStr == "" {
		paddingStr = "3"
	}

	padding, err := strconv.Atoi(paddingStr)
	if err != nil {
		padding = 3
	}

	if prefix == "" {
		prefix = "XXX"
	}

	preview := services.PreviewDCNumber(format, prefix, prefix, padding)
	return c.JSON(http.StatusOK, map[string]interface{}{"preview": preview})
}
