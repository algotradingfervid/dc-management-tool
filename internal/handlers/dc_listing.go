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

// ListAllDeliveryChallans shows all DCs for the current project with filters, sorting, and pagination.
func ListAllDeliveryChallans(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	project := c.MustGet("currentProject").(*models.Project)

	projectIDStr := strconv.Itoa(project.ID)

	// Parse filters — force project filter to current project
	filters := database.DCListFilters{
		ProjectID: projectIDStr,
		DCType:    c.DefaultQuery("type", "all"),
		Status:    c.DefaultQuery("status", "all"),
		DateFrom:  c.Query("date_from"),
		DateTo:    c.Query("date_to"),
		Search:    c.Query("search"),
		SortBy:    c.DefaultQuery("sort_by", "challan_date"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	filters.Page = page
	filters.PageSize = 25

	// Fetch DCs
	result, err := database.GetAllDCsFiltered(filters)
	if err != nil {
		log.Printf("Error fetching DCs: %v", err)
		c.HTML(http.StatusInternalServerError, "delivery_challans/list.html", gin.H{
			"user":           user,
			"currentProject": project,
			"currentPath":    c.Request.URL.Path,
			"error":          "Failed to load delivery challans",
		})
		return
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	basePath := fmt.Sprintf("/projects/%d/dcs-list", project.ID)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
		helpers.Breadcrumb{Title: "All DCs"},
	)

	c.HTML(http.StatusOK, "delivery_challans/list.html", gin.H{
		"user":           user,
		"currentProject": project,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"result":         result,
		"filters":        filters,
		"basePath":       basePath,
		"flashType":      flashType,
		"flashMessage":   flashMessage,
	})
}
