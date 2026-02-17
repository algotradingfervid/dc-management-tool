package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ShowSerialSearch handles GET /projects/:id/serial-search
func ShowSerialSearch(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	project := c.MustGet("currentProject").(*models.Project)
	query := strings.TrimSpace(c.Query("q"))
	projectIDStr := strconv.Itoa(project.ID)

	basePath := fmt.Sprintf("/projects/%d/serial-search", project.ID)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
		helpers.Breadcrumb{Title: "Serial Search"},
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
			"user":           user,
			"currentProject": project,
			"currentPath":    c.Request.URL.Path,
			"breadcrumbs":    breadcrumbs,
			"basePath":       basePath,
			"query":          query,
			"Initial":        true,
		})
		return
	}

	results, notFound, err := database.SearchSerialNumbers(query, projectIDStr)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "serial_search.html", gin.H{
			"user":           user,
			"currentProject": project,
			"currentPath":    c.Request.URL.Path,
			"breadcrumbs":    breadcrumbs,
			"basePath":       basePath,
			"query":          query,
			"error":          "Search failed: " + err.Error(),
		})
		return
	}

	data := gin.H{
		"user":           user,
		"currentProject": project,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"basePath":       basePath,
		"query":          query,
		"results":        results,
		"notFound":       notFound,
		"resultCount":    len(results),
		"Initial":        false,
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "htmx/serial_search_results.html", data)
		return
	}

	c.HTML(http.StatusOK, "serial_search.html", data)
}
