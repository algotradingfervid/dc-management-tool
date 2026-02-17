package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// ShowDashboard displays the project-scoped dashboard page with statistics
func ShowDashboard(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	project := c.MustGet("currentProject").(*models.Project)

	// Get all user projects for the dropdown
	allProjects, err := database.GetAccessibleProjects(user)
	if err != nil {
		log.Printf("Error fetching user projects: %v", err)
	}

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

	// Fetch statistics scoped to current project
	stats, err := database.GetDashboardStats(project.ID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching dashboard stats: %v", err)
		stats = &database.DashboardStats{}
	}

	// Fetch recent DCs scoped to current project
	recentDCs, err := database.GetRecentDCs(project.ID, 10)
	if err != nil {
		log.Printf("Error fetching recent DCs: %v", err)
	}

	// Fetch recent activity
	recentActivity, err := database.GetRecentActivity(project.ID, 10)
	if err != nil {
		log.Printf("Error fetching recent activity: %v", err)
	}

	// Next DC number previews
	nextTransitDC := ""
	nextOfficialDC := ""
	if project.DCPrefix != "" {
		if num, err := services.PeekNextDCNumber(database.DB, project.ID, "transit"); err == nil {
			nextTransitDC = num
		}
		if num, err := services.PeekNextDCNumber(database.DB, project.ID, "official"); err == nil {
			nextOfficialDC = num
		}
	}

	// DC number format display
	dcFormat := project.DCNumberFormat
	if dcFormat == "" {
		dcFormat = models.DefaultDCNumberFormat
	}

	// Build breadcrumbs
	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
		helpers.Breadcrumb{Title: "Dashboard"},
	)

	data := gin.H{
		"user":           user,
		"currentProject": project,
		"allProjects":    allProjects,
		"currentPath":    c.Request.URL.Path,
		"breadcrumbs":    breadcrumbs,
		"flashMessage":   flashMessage,
		"flashType":      flashType,
		"csrfToken":      csrf.Token(c.Request),
		"Stats":          stats,
		"RecentDCs":      recentDCs,
		"RecentActivity": recentActivity,
		"Range":          dateRange,
		"NextTransitDC":  nextTransitDC,
		"NextOfficialDC": nextOfficialDC,
		"DCFormat":       dcFormat,
	}

	// HTMX partial refresh
	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "htmx/dashboard_stats_partial.html", data)
		return
	}

	c.HTML(http.StatusOK, "dashboard.html", data)
}
