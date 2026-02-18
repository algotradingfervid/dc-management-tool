package components

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// PageProps holds common data passed to all page components.
type PageProps struct {
	User           *models.User
	CurrentProject *models.Project
	AllProjects    []*models.Project
	Breadcrumbs    []BreadcrumbItem
	FlashType      string
	FlashMessage   string
	CSRFToken      string
	CSRFField      string // raw HTML field
	CurrentPath    string
}

// BreadcrumbItem represents a single breadcrumb navigation item.
// This is distinct from helpers.Breadcrumb: it adds an Active flag and
// renames Title -> Label to better match templ component conventions.
type BreadcrumbItem struct {
	Label  string
	URL    string
	Active bool
}

// Render writes a templ component to the Echo response writer with the given
// HTTP status code.
func Render(c echo.Context, status int, component templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
	c.Response().WriteHeader(status)
	return component.Render(c.Request().Context(), c.Response())
}

// RenderOK is shorthand for Render with 200 OK.
func RenderOK(c echo.Context, component templ.Component) error {
	return Render(c, http.StatusOK, component)
}
