package middleware

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

func RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := auth.GetUserID(c.Request())
			if userID == 0 {
				return c.Redirect(http.StatusFound, "/login")
			}

			user, err := database.GetUserByID(userID)
			if err != nil {
				slog.Error("Failed to load user from session",
					slog.Int("user_id", userID),
					slog.String("error", err.Error()),
				)
				_ = auth.DestroySession(c.Request())
				return c.Redirect(http.StatusFound, "/login")
			}

			if !user.IsActive {
				_ = auth.DestroySession(c.Request())
				return c.Redirect(http.StatusFound, "/login")
			}

			auth.SetCurrentUser(c, user)
			return next(c)
		}
	}
}
