package middleware

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	errorpage "github.com/narendhupati/dc-management-tool/components/pages/error"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

func ProjectContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			idStr := c.Param("id")
			projectID, err := strconv.Atoi(idStr)
			if err != nil {
				return components.Render(c, http.StatusNotFound,
					errorpage.ErrorPage(http.StatusNotFound, "Invalid project ID", ""))
			}

			project, err := database.GetProjectByID(projectID)
			if err != nil {
				return components.Render(c, http.StatusNotFound,
					errorpage.ErrorPage(http.StatusNotFound, "Project not found", ""))
			}

			user := auth.GetCurrentUser(c)
			if user == nil {
				return c.Redirect(http.StatusFound, "/login")
			}

			// Admin can access all projects; regular users need assignment or ownership
			if !user.IsAdmin() {
				assigned, err := database.IsUserAssignedToProject(user.ID, project.ID)
				if err != nil || (!assigned && project.CreatedBy != user.ID) {
					return components.Render(c, http.StatusForbidden,
						errorpage.ErrorPage(http.StatusForbidden, "You don't have access to this project", ""))
				}
			}

			c.Set("currentProject", project)

			if count, err := database.GetProductCount(project.ID); err == nil {
				c.Set("productCount", count)
			}

			go func() {
				if err := database.UpdateLastProjectID(user.ID, project.ID); err != nil {
					slog.Warn("Failed to update last_project_id",
						slog.Int("user_id", user.ID),
						slog.String("error", err.Error()),
					)
				}
			}()

			return next(c)
		}
	}
}
