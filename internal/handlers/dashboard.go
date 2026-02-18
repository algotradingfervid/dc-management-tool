package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/components/htmx"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	"github.com/narendhupati/dc-management-tool/components/pages/dashboard"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
	"github.com/narendhupati/dc-management-tool/internal/services"
)

// dashboardStatsToMap converts a *database.DashboardStats struct to the
// map[string]int64 that the dashboard templ component expects.
func dashboardStatsToMap(s *database.DashboardStats) map[string]int64 {
	if s == nil {
		return map[string]int64{}
	}
	return map[string]int64{
		"total_products":          int64(s.TotalProducts),
		"total_templates":         int64(s.TotalTemplates),
		"total_bill_to_addresses": int64(s.TotalBillToAddresses),
		"total_ship_to_addresses": int64(s.TotalShipToAddresses),
		"total_dcs":               int64(s.TotalDCs),
		"transit_dcs":             int64(s.TransitDCs),
		"official_dcs":            int64(s.OfficialDCs),
		"issued_dcs":              int64(s.IssuedDCs),
		"draft_dcs":               int64(s.DraftDCs),
		"transit_dcs_draft":       int64(s.TransitDCsDraft),
		"transit_dcs_issued":      int64(s.TransitDCsIssued),
		"official_dcs_draft":      int64(s.OfficialDCsDraft),
		"official_dcs_issued":     int64(s.OfficialDCsIssued),
		"total_serial_numbers":    int64(s.TotalSerialNumbers),
	}
}

// toActivityItems converts []database.RecentActivity to []dashboard.ActivityItem.
// The CreatedAt field is pre-formatted as a human-readable relative time string
// using helpers.TimeAgo, matching the dashboard component's expectation.
func toActivityItems(rows []database.RecentActivity) []dashboard.ActivityItem {
	items := make([]dashboard.ActivityItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, dashboard.ActivityItem{
			ProjectID:   r.ProjectID,
			EntityID:    r.EntityID,
			Status:      r.Status,
			Title:       r.Title,
			Description: r.Description,
			CreatedAt:   helpers.TimeAgo(r.CreatedAt),
		})
	}
	return items
}

// toBreadcrumbItems converts []helpers.Breadcrumb to []partials.BreadcrumbItem.
// Items without a URL are marked as active (the current/last page).
func toBreadcrumbItems(crumbs []helpers.Breadcrumb) []partials.BreadcrumbItem {
	items := make([]partials.BreadcrumbItem, len(crumbs))
	for i, b := range crumbs {
		items[i] = partials.BreadcrumbItem{
			Label:  b.Title,
			URL:    b.URL,
			Active: b.URL == "",
		}
	}
	return items
}

// wrapInLayout composes a templ component inside layouts.Main using
// templ.WithChildren. This is the standard Go-side way to inject child content
// into a templ layout that uses { children... }.
func wrapInLayout(title string, sidebar, topbar templ.Component, flashMessage, flashType string, content templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		ctx = templ.WithChildren(ctx, content)
		return layouts.Main(title, sidebar, topbar, flashMessage, flashType).Render(ctx, w)
	})
}

// ShowDashboard displays the project-scoped dashboard page with statistics.
func ShowDashboard(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	project := c.Get("currentProject").(*models.Project)

	// Get all user projects for the sidebar dropdown.
	allProjects, err := database.GetAccessibleProjects(user)
	if err != nil {
		slog.Error("error fetching user projects", slog.String("error", err.Error()), slog.Int("projectID", project.ID))
	}

	// Get flash messages.
	flashType, flashMessage := auth.PopFlash(c.Request())

	// Parse date range filter.
	dateRange := c.QueryParam("range")
	if dateRange == "" {
		dateRange = "all_time"
	}
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

	// Fetch statistics scoped to current project.
	stats, err := database.GetDashboardStats(project.ID, startDate, endDate)
	if err != nil {
		slog.Error("error fetching dashboard stats", slog.String("error", err.Error()), slog.Int("projectID", project.ID))
		stats = &database.DashboardStats{}
	}

	// Fetch recent DCs scoped to current project.
	recentDCs, err := database.GetRecentDCs(project.ID, 10)
	if err != nil {
		slog.Error("error fetching recent DCs", slog.String("error", err.Error()), slog.Int("projectID", project.ID))
	}

	// Fetch recent activity.
	recentActivity, err := database.GetRecentActivity(project.ID, 10)
	if err != nil {
		slog.Error("error fetching recent activity", slog.String("error", err.Error()), slog.Int("projectID", project.ID))
	}

	// Next DC number previews.
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

	// DC number format display.
	dcFormat := project.DCNumberFormat
	if dcFormat == "" {
		dcFormat = models.DefaultDCNumberFormat
	}

	// HTMX partial refresh — return only the stats + activity fragment.
	if c.Request().Header.Get("HX-Request") == "true" {
		props := htmx.DashboardStatsPartialProps{
			Stats:          stats,
			RecentDCs:      recentDCs,
			RecentActivity: recentActivity,
			CurrentProject: project,
			Range:          dateRange,
			NextTransitDC:  nextTransitDC,
			NextOfficialDC: nextOfficialDC,
			DCFormat:       dcFormat,
		}
		return components.RenderOK(c, htmx.DashboardStatsPartial(props))
	}

	// Build breadcrumbs.
	rawBreadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
		helpers.Breadcrumb{Title: "Dashboard"},
	)
	breadcrumbs := toBreadcrumbItems(rawBreadcrumbs)

	// Convert stats struct to the map[string]int64 form the component expects.
	statsMap := dashboardStatsToMap(stats)

	// Convert database.RecentActivity to dashboard.ActivityItem.
	activityItems := toActivityItems(recentActivity)

	// csrf.Token is available for forms; the dashboard component does not
	// currently need a CSRF token but we compute it to maintain consistency
	// with other handlers (suppressed with blank identifier to avoid unused
	// import errors if the component later needs it).
	_ = csrf.Token(c.Request())

	currentPath := c.Request().URL.Path

	// Compose the page content.
	pageContent := dashboard.Dashboard(
		user,
		project,
		allProjects,
		statsMap,
		breadcrumbs,
		flashType,
		flashMessage,
		dateRange,
		dcFormat,
		nextTransitDC,
		nextOfficialDC,
		activityItems,
	)

	// Wrap page content in the main layout.
	sidebar := partials.Sidebar(user, project, allProjects, currentPath)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	page := wrapInLayout("Dashboard", sidebar, topbar, flashMessage, flashType, pageContent)

	return components.RenderOK(c, page)
}
