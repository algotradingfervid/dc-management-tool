package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"

	"github.com/narendhupati/dc-management-tool/components/layouts"
	deliverychallan "github.com/narendhupati/dc-management-tool/components/pages/delivery_challans"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ListAllDeliveryChallans shows all DCs for the current project with filters, sorting, and pagination.
func ListAllDeliveryChallans(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	project := c.Get("currentProject").(*models.Project)

	projectIDStr := strconv.Itoa(project.ID)

	dcType := c.QueryParam("type")
	if dcType == "" {
		dcType = "all"
	}
	status := c.QueryParam("status")
	if status == "" {
		status = "all"
	}
	sortBy := c.QueryParam("sort_by")
	if sortBy == "" {
		sortBy = "challan_date"
	}
	sortOrder := c.QueryParam("sort_order")
	if sortOrder == "" {
		sortOrder = "desc"
	}
	pageStr := c.QueryParam("page")
	if pageStr == "" {
		pageStr = "1"
	}

	// Parse filters — force project filter to current project
	filters := database.DCListFilters{
		ProjectID: projectIDStr,
		DCType:    dcType,
		Status:    status,
		DateFrom:  c.QueryParam("date_from"),
		DateTo:    c.QueryParam("date_to"),
		Search:    c.QueryParam("search"),
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	filters.Page = page
	filters.PageSize = 25

	// Fetch DCs
	result, err := database.GetAllDCsFiltered(filters)
	if err != nil {
		slog.Error("Error fetching DCs", slog.Int("project_id", project.ID), slog.String("error", err.Error()))
		// Build a minimal error page using the list component with empty data.
		allProjects, _ := database.GetAccessibleProjects(user)
		filtersMap := dcFiltersToMap(filters)
		pageContent := deliverychallan.List(
			user,
			project,
			allProjects,
			nil,
			filtersMap,
			"error",
			"Failed to load delivery challans",
			csrf.Token(c.Request()),
		)
		sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
		topbar := partials.Topbar(user, project, allProjects, "error", "Failed to load delivery challans")
		return components.Render(c, http.StatusInternalServerError, layouts.MainWithContent("Delivery Challans", sidebar, topbar, "Failed to load delivery challans", "error", pageContent))
	}

	flashType, flashMessage := auth.PopFlash(c.Request())

	_ = fmt.Sprintf("/projects/%d/dcs-list", project.ID) // basePath kept for reference
	_ = helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
		helpers.Breadcrumb{Title: "All DCs"},
	)

	allProjects, _ := database.GetAccessibleProjects(user)

	// Convert DCListItem slice to []models.DeliveryChallan for the templ component.
	dcs := dcListItemsToModels(result.DCs)

	filtersMap := dcFiltersToMap(filters)

	pageContent := deliverychallan.List(
		user,
		project,
		allProjects,
		dcs,
		filtersMap,
		flashType,
		flashMessage,
		csrf.Token(c.Request()),
	)
	sidebar := partials.Sidebar(user, project, allProjects, c.Request().URL.Path)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	return components.RenderOK(c, layouts.MainWithContent("Delivery Challans", sidebar, topbar, flashMessage, flashType, pageContent))
}

// dcFiltersToMap converts DCListFilters to the map[string]string expected by the List templ component.
func dcFiltersToMap(f database.DCListFilters) map[string]string {
	return map[string]string{
		"type":       f.DCType,
		"status":     f.Status,
		"date_from":  f.DateFrom,
		"date_to":    f.DateTo,
		"search":     f.Search,
		"sort_by":    f.SortBy,
		"sort_order": f.SortOrder,
		"page":       strconv.Itoa(f.Page),
	}
}

// dcListItemsToModels converts database.DCListItem slice to []models.DeliveryChallan.
// Only the fields used by the List templ component are populated.
func dcListItemsToModels(items []database.DCListItem) []models.DeliveryChallan {
	out := make([]models.DeliveryChallan, len(items))
	for i, item := range items { //nolint:gocritic
		cd := item.ChallanDate
		out[i] = models.DeliveryChallan{
			ID:          item.ID,
			DCNumber:    item.DCNumber,
			DCType:      item.DCType,
			ChallanDate: &cd,
			ProjectID:   item.ProjectID,
			ProjectName: item.ProjectName,
			Status:      item.Status,
		}
	}
	return out
}
