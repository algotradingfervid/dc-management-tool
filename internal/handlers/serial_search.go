package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/components/htmx"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	errorpage "github.com/narendhupati/dc-management-tool/components/pages/error"
	serialsearchpage "github.com/narendhupati/dc-management-tool/components/pages/serial_search"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ShowSerialSearch handles GET /projects/:id/serial-search
func ShowSerialSearch(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	project := c.Get("currentProject").(*models.Project)
	query := strings.TrimSpace(c.QueryParam("q"))
	currentPath := c.Request().URL.Path

	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request())

	rawBreadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: project.Name, URL: fmt.Sprintf("/projects/%d/dashboard", project.ID)},
		helpers.Breadcrumb{Title: "Serial Search"},
	)
	breadcrumbs := toBreadcrumbItems(rawBreadcrumbs)
	_ = breadcrumbs // available for future use

	isHTMX := c.Request().Header.Get("HX-Request") == "true"

	// No query yet — show initial state.
	if query == "" {
		if isHTMX {
			return components.RenderOK(c, htmx.SerialSearchResults(htmx.SerialSearchResultsProps{
				Initial:     true,
				Results:     nil,
				NotFound:    nil,
				ResultCount: 0,
			}))
		}

		sidebar := partials.Sidebar(user, project, allProjects, currentPath)
		topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
		pageContent := serialsearchpage.SerialSearch(
			user, project, allProjects,
			query, true, nil, nil,
			flashType, flashMessage,
		)
		return components.RenderOK(c,
			layouts.MainWithContent("Serial Search", sidebar, topbar, flashMessage, flashType, pageContent))
	}

	projectIDStr := fmt.Sprintf("%d", project.ID)
	results, notFound, err := database.SearchSerialNumbers(query, projectIDStr)
	if err != nil {
		slog.Error("Serial search failed", slog.Int("project_id", project.ID), slog.String("query", query), slog.String("error", err.Error()))
		if isHTMX {
			// Return an HTMX-friendly error fragment.
			return components.Render(c, http.StatusInternalServerError,
				htmx.SerialSearchResults(htmx.SerialSearchResultsProps{
					Initial:     false,
					Results:     nil,
					NotFound:    nil,
					ResultCount: 0,
				}))
		}
		return components.Render(c, http.StatusInternalServerError,
			errorpage.ErrorPage(http.StatusInternalServerError, "Search failed", err.Error()))
	}

	if isHTMX {
		return components.RenderOK(c, htmx.SerialSearchResults(htmx.SerialSearchResultsProps{
			Initial:     false,
			Results:     results,
			NotFound:    notFound,
			ResultCount: len(results),
		}))
	}

	sidebar := partials.Sidebar(user, project, allProjects, currentPath)
	topbar := partials.Topbar(user, project, allProjects, flashType, flashMessage)
	pageContent := serialsearchpage.SerialSearch(
		user, project, allProjects,
		query, false, results, notFound,
		flashType, flashMessage,
	)
	return components.RenderOK(c,
		layouts.MainWithContent("Serial Search", sidebar, topbar, flashMessage, flashType, pageContent))
}
