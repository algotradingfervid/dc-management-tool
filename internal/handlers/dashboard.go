package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
)

// ShowDashboard displays the dashboard page
func ShowDashboard(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Get flash messages
	flashType, flashMessage := auth.PopFlash(c.Request)

	// Build breadcrumbs
	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Dashboard"},
	)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"flashMessage": flashMessage,
		"flashType":    flashType,
	})
}
