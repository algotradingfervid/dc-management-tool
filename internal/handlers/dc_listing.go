package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
)

// ListAllDeliveryChallans shows all DCs across all projects with filters, sorting, and pagination.
func ListAllDeliveryChallans(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Parse filters
	filters := database.DCListFilters{
		ProjectID: c.DefaultQuery("project", "all"),
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
			"user":        user,
			"currentPath": c.Request.URL.Path,
			"error":       "Failed to load delivery challans",
		})
		return
	}

	// Fetch projects for dropdown
	projects, err := database.GetAllProjectOptions()
	if err != nil {
		log.Printf("Error fetching projects: %v", err)
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "All DCs", URL: ""},
	)

	c.HTML(http.StatusOK, "delivery_challans/list.html", gin.H{
		"user":         user,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  breadcrumbs,
		"result":       result,
		"filters":      filters,
		"projects":     projects,
		"flashType":    flashType,
		"flashMessage": flashMessage,
	})

}
