package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
)

// ShowDashboard displays the dashboard page with statistics
func ShowDashboard(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Get flash messages
	flashType, flashMessage := auth.PopFlash(c.Request)

	// Parse date range filter
	dateRange := c.DefaultQuery("range", "all_time")
	var startDate, endDate *time.Time

	now := time.Now()
	switch dateRange {
	case "last_7_days":
		s := now.AddDate(0, 0, -7)
		startDate = &s
	case "last_30_days":
		s := now.AddDate(0, 0, -30)
		startDate = &s
	case "last_3_months":
		s := now.AddDate(0, -3, 0)
		startDate = &s
	case "current_month":
		s := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		e := s.AddDate(0, 1, -1)
		startDate = &s
		endDate = &e
	case "all_time":
		// no filter
	}

	// Fetch statistics
	stats, err := database.GetDashboardStats(startDate, endDate)
	if err != nil {
		log.Printf("Error fetching dashboard stats: %v", err)
		stats = &database.DashboardStats{}
	}

	// Fetch recent DCs
	recentDCs, err := database.GetRecentDCs(10)
	if err != nil {
		log.Printf("Error fetching recent DCs: %v", err)
	}

	// Fetch project DC counts
	projectCounts, err := database.GetProjectDCCounts()
	if err != nil {
		log.Printf("Error fetching project DC counts: %v", err)
	}

	// Build breadcrumbs
	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Dashboard"},
	)

	data := gin.H{
		"user":          user,
		"currentPath":   c.Request.URL.Path,
		"breadcrumbs":   breadcrumbs,
		"flashMessage":  flashMessage,
		"flashType":     flashType,
		"csrfToken":     csrf.Token(c.Request),
		"Stats":         stats,
		"RecentDCs":     recentDCs,
		"ProjectCounts": projectCounts,
		"Range":         dateRange,
	}

	// HTMX partial refresh
	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "htmx/dashboard_stats_partial.html", data)
		return
	}

	c.HTML(http.StatusOK, "dashboard.html", data)
}
