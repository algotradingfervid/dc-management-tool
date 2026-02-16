# Phase 5: Project CRUD

## Overview

Build complete project management functionality including list view with card grid layout, create/edit forms with all fields from PRD, project detail view with tabs for products/templates/addresses/DCs, delete functionality with confirmation, image upload for company signature, and server-side validation. Implement HTMX interactions for smooth user experience.

## Prerequisites

- Phase 1 completed (project scaffolding)
- Phase 2 completed (database schema with projects table)
- Phase 3 completed (authentication)
- Phase 4 completed (UI layout with sidebar navigation)

## Goals

- Display all projects in card grid layout matching mockup
- Create new project form with all fields (PO info, billing details, signature upload)
- Edit existing projects
- Delete projects with confirmation dialog
- Project detail view with tabbed interface
- Image upload and storage for company signature
- Server-side form validation
- HTMX-powered smooth interactions
- Flash messages for success/error feedback
- Breadcrumb navigation for all project pages
- Responsive design for mobile/tablet/desktop

## Detailed Implementation Steps

### 1. Create Project Model and Repository

1.1. Create project model in `internal/models/project.go`
- Project struct with all fields
- Validation methods

1.2. Create project repository in `internal/database/projects.go`
- GetAllProjects (with stats: DC counts, template counts)
- GetProjectByID
- CreateProject
- UpdateProject
- DeleteProject
- Check if project can be deleted (has issued DCs)

### 2. Create Project List Page

2.1. Create projects list handler in `internal/handlers/projects.go`
- GET /projects - list all projects
- Display in card grid (matching mockup 03-projects-list.html)
- Show project name, PO number, template count, DC count
- Quick action buttons (View, Edit, New DC)

2.2. Create projects list template in `templates/projects/list.html`
- Card grid layout (responsive: 1 col mobile, 2 col tablet, 3 col desktop)
- Each card shows project summary
- "Create New Project" button at top
- Empty state message if no projects

### 3. Create Project Form (New/Edit)

3.1. Create project form handler
- GET /projects/new - show create form
- POST /projects - create new project
- GET /projects/:id/edit - show edit form
- PUT /projects/:id - update project

3.2. Create project form template in `templates/projects/form.html`
- Match mockup 04-project-form.html design
- Fields:
  - Project Name (required)
  - PO Number
  - PO Date (date picker)
  - Billing Name
  - Billing Address (textarea)
  - Billing GSTIN
  - Company Signature (file upload)
- Submit button
- Cancel button (back to list)
- Show existing signature image in edit mode

3.3. Implement form validation
- Required field validation
- Date format validation
- GSTIN format validation (optional)
- File type validation (images only)
- File size validation (max 2MB)

### 4. Implement Image Upload

4.1. Create upload handler in `internal/handlers/upload.go`
- Handle multipart form data
- Validate file type (jpg, png, gif)
- Validate file size
- Generate unique filename
- Save to uploads directory
- Return file path

4.2. Create upload utilities
- File type checker
- File size validator
- Filename sanitizer
- Thumbnail generator (optional)

### 5. Create Project Detail View

5.1. Create project detail handler
- GET /projects/:id - show project details
- Tabs: Products, Templates, Bill To, Ship To, Issued DCs
- Default to Products tab

5.2. Create project detail template in `templates/projects/detail.html`
- Match mockup 05-project-detail.html design
- Header with project info
- Action buttons (Edit, Delete, New DC)
- Tabbed interface
- Products tab (placeholder for Phase 6)
- Templates tab (placeholder for Phase 9)
- Bill To tab (placeholder for Phase 7)
- Ship To tab (placeholder for Phase 8)
- Issued DCs tab (list issued DCs)

### 6. Implement Delete Functionality

6.1. Create delete handler
- DELETE /projects/:id - delete project
- Check if project can be deleted (no issued DCs)
- Show confirmation dialog before delete
- Redirect to list after delete

6.2. Create confirmation modal component
- Reusable modal for confirmations
- JavaScript for showing/hiding
- HTMX trigger for delete action

### 7. Add HTMX Interactions

7.1. Implement HTMX for smoother UX
- hx-boost for navigation links
- hx-delete for delete action
- hx-target for partial updates
- Loading states during requests

### 8. Create Helper Functions

8.1. Create form helpers
- Form field components (input, textarea, file)
- Error message display
- Date formatting

8.2. Create validation helpers
- GSTIN validator
- Date validator
- Required field validator

## Files to Create/Modify

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/models/project.go`
```go
package models

import (
	"time"
)

type Project struct {
	ID                     int       `json:"id"`
	Name                   string    `json:"name"`
	PONumber               string    `json:"po_number"`
	PODate                 *string   `json:"po_date"` // Nullable date
	BillingName            string    `json:"billing_name"`
	BillingAddress         string    `json:"billing_address"`
	BillingGSTIN           string    `json:"billing_gstin"`
	CompanySignaturePath   string    `json:"company_signature_path"`
	LastTransitDCNumber    int       `json:"last_transit_dc_number"`
	LastOfficialDCNumber   int       `json:"last_official_dc_number"`
	CreatedBy              int       `json:"created_by"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`

	// Computed fields (not in database)
	TransitDCCount  int `json:"transit_dc_count"`
	OfficialDCCount int `json:"official_dc_count"`
	TemplateCount   int `json:"template_count"`
	ProductCount    int `json:"product_count"`
}

// Validate checks if project data is valid
func (p *Project) Validate() map[string]string {
	errors := make(map[string]string)

	if p.Name == "" {
		errors["name"] = "Project name is required"
	}

	// Add more validation as needed
	return errors
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/database/projects.go`
```go
package database

import (
	"database/sql"
	"fmt"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// GetAllProjects retrieves all projects with counts
func GetAllProjects() ([]*models.Project, error) {
	query := `
		SELECT
			p.id,
			p.name,
			p.po_number,
			p.po_date,
			p.billing_name,
			p.billing_address,
			p.billing_gstin,
			p.company_signature_path,
			p.last_transit_dc_number,
			p.last_official_dc_number,
			p.created_by,
			p.created_at,
			p.updated_at,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_dc_count,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_dc_count,
			COUNT(DISTINCT t.id) as template_count,
			COUNT(DISTINCT pr.id) as product_count
		FROM projects p
		LEFT JOIN delivery_challans dc ON p.id = dc.project_id
		LEFT JOIN dc_templates t ON p.id = t.project_id
		LEFT JOIN products pr ON p.id = pr.project_id
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.PONumber,
			&p.PODate,
			&p.BillingName,
			&p.BillingAddress,
			&p.BillingGSTIN,
			&p.CompanySignaturePath,
			&p.LastTransitDCNumber,
			&p.LastOfficialDCNumber,
			&p.CreatedBy,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.TransitDCCount,
			&p.OfficialDCCount,
			&p.TemplateCount,
			&p.ProductCount,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

// GetProjectByID retrieves a single project by ID with counts
func GetProjectByID(id int) (*models.Project, error) {
	query := `
		SELECT
			p.id,
			p.name,
			p.po_number,
			p.po_date,
			p.billing_name,
			p.billing_address,
			p.billing_gstin,
			p.company_signature_path,
			p.last_transit_dc_number,
			p.last_official_dc_number,
			p.created_by,
			p.created_at,
			p.updated_at,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_dc_count,
			COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_dc_count,
			COUNT(DISTINCT t.id) as template_count,
			COUNT(DISTINCT pr.id) as product_count
		FROM projects p
		LEFT JOIN delivery_challans dc ON p.id = dc.project_id
		LEFT JOIN dc_templates t ON p.id = t.project_id
		LEFT JOIN products pr ON p.id = pr.project_id
		WHERE p.id = ?
		GROUP BY p.id
	`

	p := &models.Project{}
	err := DB.QueryRow(query, id).Scan(
		&p.ID,
		&p.Name,
		&p.PONumber,
		&p.PODate,
		&p.BillingName,
		&p.BillingAddress,
		&p.BillingGSTIN,
		&p.CompanySignaturePath,
		&p.LastTransitDCNumber,
		&p.LastOfficialDCNumber,
		&p.CreatedBy,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.TransitDCCount,
		&p.OfficialDCCount,
		&p.TemplateCount,
		&p.ProductCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found")
	}
	if err != nil {
		return nil, err
	}

	return p, nil
}

// CreateProject creates a new project
func CreateProject(p *models.Project) error {
	query := `
		INSERT INTO projects (
			name, po_number, po_date, billing_name, billing_address,
			billing_gstin, company_signature_path, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := DB.Exec(
		query,
		p.Name,
		p.PONumber,
		p.PODate,
		p.BillingName,
		p.BillingAddress,
		p.BillingGSTIN,
		p.CompanySignaturePath,
		p.CreatedBy,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	p.ID = int(id)
	return nil
}

// UpdateProject updates an existing project
func UpdateProject(p *models.Project) error {
	query := `
		UPDATE projects SET
			name = ?,
			po_number = ?,
			po_date = ?,
			billing_name = ?,
			billing_address = ?,
			billing_gstin = ?,
			company_signature_path = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := DB.Exec(
		query,
		p.Name,
		p.PONumber,
		p.PODate,
		p.BillingName,
		p.BillingAddress,
		p.BillingGSTIN,
		p.CompanySignaturePath,
		p.ID,
	)

	return err
}

// DeleteProject deletes a project
func DeleteProject(id int) error {
	// Check if project has issued DCs
	var issuedCount int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND status = 'issued'",
		id,
	).Scan(&issuedCount)
	if err != nil {
		return err
	}

	if issuedCount > 0 {
		return fmt.Errorf("cannot delete project with issued delivery challans")
	}

	// Delete project (cascade will delete related records)
	_, err = DB.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

// CanDeleteProject checks if a project can be deleted
func CanDeleteProject(id int) (bool, error) {
	var issuedCount int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND status = 'issued'",
		id,
	).Scan(&issuedCount)
	if err != nil {
		return false, err
	}

	return issuedCount == 0, nil
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/handlers/projects.go`
```go
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ListProjects displays all projects
func ListProjects(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Get all projects
	projects, err := database.GetAllProjects()
	if err != nil {
		log.Printf("Error fetching projects: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load projects",
		})
		return
	}

	// Get flash messages
	session, _ := auth.GetStore().Get(c.Request, "dc_management_session")
	flashes := auth.GetFlash(session)
	session.Save(c.Request, c.Writer)

	// Build breadcrumbs
	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: ""},
	)

	c.HTML(http.StatusOK, "projects/list.html", gin.H{
		"user":        user,
		"currentPath": c.Request.URL.Path,
		"breadcrumbs": breadcrumbs,
		"projects":    projects,
		"flashes":     flashes,
	})
}

// ShowProjectForm displays the project creation form
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
		"project":     &models.Project{},
		"errors":      map[string]string{},
		"isEdit":      false,
	})
}

// CreateProject handles project creation
func CreateProject(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Parse form
	project := &models.Project{
		Name:           c.PostForm("name"),
		PONumber:       c.PostForm("po_number"),
		BillingName:    c.PostForm("billing_name"),
		BillingAddress: c.PostForm("billing_address"),
		BillingGSTIN:   c.PostForm("billing_gstin"),
		CreatedBy:      user.ID,
	}

	// Handle optional PO date
	poDate := c.PostForm("po_date")
	if poDate != "" {
		project.PODate = &poDate
	}

	// Validate
	errors := project.Validate()
	if len(errors) > 0 {
		c.HTML(http.StatusOK, "projects/form.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": helpers.BuildBreadcrumbs(
				helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
				helpers.Breadcrumb{Title: "New Project", URL: ""},
			),
			"project": project,
			"errors":  errors,
			"isEdit":  false,
		})
		return
	}

	// Handle file upload
	file, err := c.FormFile("company_signature")
	if err == nil {
		// File was uploaded
		filepath, err := handleSignatureUpload(c, file)
		if err != nil {
			errors["company_signature"] = err.Error()
			c.HTML(http.StatusOK, "projects/form.html", gin.H{
				"user":        user,
				"currentPath": c.Request.URL.Path,
				"breadcrumbs": helpers.BuildBreadcrumbs(
					helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
					helpers.Breadcrumb{Title: "New Project", URL: ""},
				),
				"project": project,
				"errors":  errors,
				"isEdit":  false,
			})
			return
		}
		project.CompanySignaturePath = filepath
	}

	// Create project
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
			"project": project,
			"errors":  errors,
			"isEdit":  false,
		})
		return
	}

	// Success - redirect to project detail
	session, _ := auth.GetStore().Get(c.Request, "dc_management_session")
	auth.SetFlash(session, "success", "Project created successfully")
	session.Save(c.Request, c.Writer)

	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", project.ID))
}

// ShowEditProjectForm displays the project edit form
func ShowEditProjectForm(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Get project ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid project ID"})
		return
	}

	// Get project
	project, err := database.GetProjectByID(id)
	if err != nil {
		log.Printf("Error fetching project: %v", err)
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Project not found"})
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
	})
}

// UpdateProject handles project updates
func UpdateProject(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Get project ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid project ID"})
		return
	}

	// Get existing project
	existingProject, err := database.GetProjectByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Project not found"})
		return
	}

	// Parse form
	project := &models.Project{
		ID:                   id,
		Name:                 c.PostForm("name"),
		PONumber:             c.PostForm("po_number"),
		BillingName:          c.PostForm("billing_name"),
		BillingAddress:       c.PostForm("billing_address"),
		BillingGSTIN:         c.PostForm("billing_gstin"),
		CompanySignaturePath: existingProject.CompanySignaturePath,
		CreatedBy:            existingProject.CreatedBy,
	}

	// Handle optional PO date
	poDate := c.PostForm("po_date")
	if poDate != "" {
		project.PODate = &poDate
	}

	// Validate
	errors := project.Validate()
	if len(errors) > 0 {
		c.HTML(http.StatusOK, "projects/form.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": helpers.BuildBreadcrumbs(
				helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
				helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
				helpers.Breadcrumb{Title: "Edit", URL: ""},
			),
			"project": project,
			"errors":  errors,
			"isEdit":  true,
		})
		return
	}

	// Handle file upload (optional in edit)
	file, err := c.FormFile("company_signature")
	if err == nil {
		// New file was uploaded
		filepath, err := handleSignatureUpload(c, file)
		if err != nil {
			errors["company_signature"] = err.Error()
			c.HTML(http.StatusOK, "projects/form.html", gin.H{
				"user":        user,
				"currentPath": c.Request.URL.Path,
				"breadcrumbs": helpers.BuildBreadcrumbs(
					helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
					helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d", project.ID)},
					helpers.Breadcrumb{Title: "Edit", URL: ""},
				),
				"project": project,
				"errors":  errors,
				"isEdit":  true,
			})
			return
		}
		project.CompanySignaturePath = filepath
	}

	// Update project
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
			"project": project,
			"errors":  errors,
			"isEdit":  true,
		})
		return
	}

	// Success - redirect to project detail
	session, _ := auth.GetStore().Get(c.Request, "dc_management_session")
	auth.SetFlash(session, "success", "Project updated successfully")
	session.Save(c.Request, c.Writer)

	c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d", project.ID))
}

// ShowProject displays project details
func ShowProject(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Get project ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid project ID"})
		return
	}

	// Get project
	project, err := database.GetProjectByID(id)
	if err != nil {
		log.Printf("Error fetching project: %v", err)
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Project not found"})
		return
	}

	// Get flash messages
	session, _ := auth.GetStore().Get(c.Request, "dc_management_session")
	flashes := auth.GetFlash(session)
	session.Save(c.Request, c.Writer)

	// Get active tab (default to products)
	activeTab := c.DefaultQuery("tab", "products")

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Projects", URL: "/projects"},
		helpers.Breadcrumb{Title: project.Name, URL: ""},
	)

	c.HTML(http.StatusOK, "projects/detail.html", gin.H{
		"user":        user,
		"currentPath": c.Request.URL.Path,
		"breadcrumbs": breadcrumbs,
		"project":     project,
		"activeTab":   activeTab,
		"flashes":     flashes,
	})
}

// DeleteProject handles project deletion
func DeleteProject(c *gin.Context) {
	// Get project ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Check if can delete
	canDelete, err := database.CanDeleteProject(id)
	if err != nil {
		log.Printf("Error checking if project can be deleted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	if !canDelete {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete project with issued delivery challans"})
		return
	}

	// Delete project
	if err := database.DeleteProject(id); err != nil {
		log.Printf("Error deleting project: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	// Success
	session, _ := auth.GetStore().Get(c.Request, "dc_management_session")
	auth.SetFlash(session, "success", "Project deleted successfully")
	session.Save(c.Request, c.Writer)

	c.JSON(http.StatusOK, gin.H{"success": true, "redirect": "/projects"})
}
```

### Continue in next response due to length...

## Database Queries

```sql
-- List all projects with counts
SELECT
    p.*,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_dc_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_dc_count,
    COUNT(DISTINCT t.id) as template_count,
    COUNT(DISTINCT pr.id) as product_count
FROM projects p
LEFT JOIN delivery_challans dc ON p.id = dc.project_id
LEFT JOIN dc_templates t ON p.id = t.project_id
LEFT JOIN products pr ON p.id = pr.project_id
GROUP BY p.id
ORDER BY p.created_at DESC;

-- Get single project by ID with counts
SELECT
    p.*,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_dc_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_dc_count,
    COUNT(DISTINCT t.id) as template_count,
    COUNT(DISTINCT pr.id) as product_count
FROM projects p
LEFT JOIN delivery_challans dc ON p.id = dc.project_id
LEFT JOIN dc_templates t ON p.id = t.project_id
LEFT JOIN products pr ON p.id = pr.project_id
WHERE p.id = ?
GROUP BY p.id;

-- Create project
INSERT INTO projects (
    name, po_number, po_date, billing_name, billing_address,
    billing_gstin, company_signature_path, created_by
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- Update project
UPDATE projects SET
    name = ?,
    po_number = ?,
    po_date = ?,
    billing_name = ?,
    billing_address = ?,
    billing_gstin = ?,
    company_signature_path = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- Delete project
DELETE FROM projects WHERE id = ?;

-- Check if project can be deleted
SELECT COUNT(*) FROM delivery_challans
WHERE project_id = ? AND status = 'issued';
```

## Testing Checklist

### Manual Testing (Browser-tested with Playwright)

- [x] Access /projects shows list of all projects in card grid
- [x] Empty state message shows when no projects exist
- [x] Click "New Project" navigates to /projects/new
- [x] Fill and submit new project form creates project
- [x] Required field validation works (name, dc_prefix)
- [x] Date picker works for PO Date
- [x] File upload works for company signature
- [x] File validation rejects non-image files (code validated)
- [x] File validation rejects files >2MB (code validated)
- [x] Success message shows after creating project (toast notification)
- [x] Redirect to project detail after creation
- [x] Click "View" on project card navigates to detail
- [x] Project detail shows all tabs (Overview, Products, Templates, Addresses, DCs)
- [x] Click "Edit" navigates to edit form
- [x] Edit form pre-fills all project data
- [x] Updating project saves changes
- [x] File upload in edit replaces signature
- [x] Click "Delete" shows confirmation modal
- [x] Confirming delete removes project
- [x] Cannot delete project with issued DCs (CSRF + DB check)
- [x] Error message shows when deletion fails
- [x] Breadcrumbs show correct path on all pages
- [x] Flash messages appear as toasts
- [x] HTMX search filtering works via GET /projects?q=
- [x] Responsive layout with Tailwind (1/2/3 col grid)

## Acceptance Criteria

- [x] Project list page with card grid layout
- [x] Each card shows project name, PO number, counts
- [x] New project form with all fields
- [x] Edit project form with pre-filled data
- [x] Project detail view with tabbed interface
- [x] Delete functionality with confirmation
- [x] Image upload for company signature
- [x] Server-side validation for all fields
- [x] HTMX interactions for smooth UX
- [x] Flash messages for success/error
- [x] Breadcrumb navigation
- [x] Responsive design

## Implementation Summary

### Files Created
- `internal/models/project.go` - Project struct matching actual DB schema (name, description, dc_prefix, tender_ref_number, tender_ref_details, po_reference, po_date, bill_from_address, company_gstin, company_signature_path) with Validate() method
- `internal/database/projects.go` - Full CRUD repository: GetAllProjects (with JOIN counts), GetProjectByID, CreateProject, UpdateProject, DeleteProject, CanDeleteProject
- `internal/handlers/projects.go` - All handlers: ListProjects (with search), ShowProjectForm, CreateProject, ShowEditProjectForm, UpdateProject, ShowProject, DeleteProject, handleSignatureUpload
- `templates/pages/projects/list.html` - Card grid layout with HTMX search, responsive 1/2/3 column grid, empty state
- `templates/pages/projects/form.html` - Create/Edit form with sections (Project Details, Tender & PO, Billing, Signature Upload), server-side validation display
- `templates/pages/projects/detail.html` - Tabbed detail view (Overview, Products, Templates, Addresses, DCs), stats cards, delete confirmation modal with CSRF

### Files Modified
- `cmd/server/main.go` - Added project routes (GET/POST /projects, GET/POST /projects/:id, GET /projects/:id/edit, DELETE /projects/:id)
- `internal/helpers/templates.go` - Extended template renderer to support subdirectory pages (pages/projects/*.html)
- `internal/helpers/template.go` - Added `derefStr` (with datetime handling) and `add` template functions

### Key Design Decisions
- Used actual DB schema fields (dc_prefix, tender_ref_number, etc.) instead of plan's originally proposed fields (po_number, billing_name)
- sql.NullString for nullable fields (company_signature_path, po_date)
- POST for update (HTML forms don't support PUT), DELETE via fetch with CSRF token
- Server-side search filtering (case-insensitive match on name, dc_prefix, po_reference)
- Image upload saves to static/uploads/ with unique timestamped filename
- Delete protection: checks for issued DCs before allowing deletion

## Next Steps

After completing Phase 5, proceed to:
- **Phase 6**: Product Management - build product catalog management within projects
