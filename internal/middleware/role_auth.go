package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/internal/auth"
)

// RequireRole checks that the current user has the specified role.
func RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := auth.GetCurrentUser(c)
			if user == nil {
				return c.Redirect(http.StatusFound, "/login")
			}

			if user.Role != role {
				return c.Render(http.StatusForbidden, "error.html", map[string]interface{}{
					"error":  "You don't have permission to access this page",
					"status": 403,
					"user":   user,
				})
			}
			return next(c)
		}
	}
}
