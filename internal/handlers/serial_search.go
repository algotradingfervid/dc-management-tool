package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
)

// ShowSerialSearch handles GET /serial-search
func ShowSerialSearch(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	query := strings.TrimSpace(c.Query("q"))
	projectID := c.DefaultQuery("project", "all")

	projects, _ := database.GetAllProjectOptions()

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Serial Search", URL: ""},
	)

	// No query yet — show initial state
	if query == "" {
		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "htmx/serial_search_results.html", gin.H{
				"Initial": true,
			})
			return
		}
		c.HTML(http.StatusOK, "serial_search.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": breadcrumbs,
			"projects":    projects,
			"query":       query,
			"projectID":   projectID,
			"Initial":     true,
		})
		return
	}

	results, notFound, err := database.SearchSerialNumbers(query, projectID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "serial_search.html", gin.H{
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"breadcrumbs": breadcrumbs,
			"projects":    projects,
			"query":       query,
			"projectID":   projectID,
			"error":       "Search failed: " + err.Error(),
		})
		return
	}

	data := gin.H{
		"user":        user,
		"currentPath": c.Request.URL.Path,
		"breadcrumbs": breadcrumbs,
		"projects":    projects,
		"query":       query,
		"projectID":   projectID,
		"results":     results,
		"notFound":    notFound,
		"resultCount": len(results),
		"Initial":     false,
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "htmx/serial_search_results.html", data)
		return
	}

	c.HTML(http.StatusOK, "serial_search.html", data)
}
