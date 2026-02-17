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
	"github.com/narendhupati/dc-management-tool/internal/services"
)

func ShowProjectSettings(c *gin.Context) {
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

	activeTab := c.DefaultQuery("tab", "general")
	flashType, flashMessage := auth.PopFlash(c.Request)

	// Generate DC number preview
	dcPreview := services.PreviewDCNumber(project.DCNumberFormat, project.DCPrefix, project.DCPrefix, project.SeqPadding)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
		helpers.Breadcrumb{Title: "Settings", URL: ""},
	)

	c.HTML(http.StatusOK, "projects/settings.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"project":      project,
		"currentProject":  project,
		"activeTab":    activeTab,
		"dcPreview":    dcPreview,
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"errors":       map[string]string{},
		"csrfField":    csrf.TemplateField(c.Request),
	})
}

func UpdateProjectSettings(c *gin.Context) {
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

	tab := c.PostForm("tab")
	if tab == "" {
		tab = "general"
	}

	// Start with existing project and update only the relevant tab fields
	project := existing

	errors := make(map[string]string)

	switch tab {
	case "general":
		project.Name = c.PostForm("name")
		project.Description = c.PostForm("description")
		project.DCPrefix = strings.ToUpper(strings.TrimSpace(c.PostForm("dc_prefix")))
		if strings.TrimSpace(project.Name) == "" {
			errors["name"] = "Project name is required"
		}
		if strings.TrimSpace(project.DCPrefix) == "" {
			errors["dc_prefix"] = "DC prefix is required"
		}

	case "company":
		project.BillFromAddress = c.PostForm("bill_from_address")
		project.DispatchFromAddress = c.PostForm("dispatch_from_address")
		project.CompanyGSTIN = strings.ToUpper(strings.TrimSpace(c.PostForm("company_gstin")))
		project.CompanyEmail = strings.TrimSpace(c.PostForm("company_email"))
		project.CompanyCIN = strings.TrimSpace(c.PostForm("company_cin"))

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
		project.DCNumberFormat = c.PostForm("dc_number_format")
		project.DCNumberSeparator = c.PostForm("dc_number_separator")
		project.PurposeText = c.PostForm("purpose_text")
		if padding := c.PostForm("seq_padding"); padding != "" {
			if p, convErr := strconv.Atoi(padding); convErr == nil {
				project.SeqPadding = p
			}
		}
		if project.SeqPadding < 2 || project.SeqPadding > 6 {
			errors["seq_padding"] = "Sequence padding must be between 2 and 6"
		}

	case "tender":
		project.TenderRefNumber = c.PostForm("tender_ref_number")
		project.TenderRefDetails = c.PostForm("tender_ref_details")
		project.POReference = c.PostForm("po_reference")
		poDate := c.PostForm("po_date")
		if poDate != "" {
			project.PODate = &poDate
		} else {
			project.PODate = nil
		}
	}

	if len(errors) > 0 {
		dcPreview := services.PreviewDCNumber(project.DCNumberFormat, project.DCPrefix, project.DCPrefix, project.SeqPadding)
		breadcrumbs := helpers.BuildBreadcrumbs(
			helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
			helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
			helpers.Breadcrumb{Title: "Settings", URL: ""},
		)
		c.HTML(http.StatusOK, "projects/settings.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": breadcrumbs,
			"project":     project,
		"currentProject":  project,
			"activeTab":   tab,
			"dcPreview":   dcPreview,
			"errors":      errors,
			"csrfField":   csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.UpdateProjectSettings(project, tab); err != nil {
		log.Printf("Error updating project settings: %v", err)
		auth.SetFlash(c.Request, "error", "Failed to save settings")
		c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/settings?tab=%s", id, tab))
		return
	}

	auth.SetFlash(c.Request, "success", "Settings saved successfully")
	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/settings?tab=%s", id, tab))
}

// PreviewDCNumberAPI returns a JSON preview of the DC number format.
func PreviewDCNumberAPI(c *gin.Context) {
	format := c.Query("format")
	prefix := c.Query("prefix")
	paddingStr := c.DefaultQuery("padding", "3")

	padding, err := strconv.Atoi(paddingStr)
	if err != nil {
		padding = 3
	}

	if prefix == "" {
		prefix = "XXX"
	}

	preview := services.PreviewDCNumber(format, prefix, prefix, padding)
	c.JSON(http.StatusOK, gin.H{"preview": preview})
}
