package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/internal/auth"
)

// RequestLoggingMiddleware logs all HTTP requests with timing and status.
func RequestLoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			statusCode := c.Response().Status

			attrs := []any{
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", statusCode),
				slog.Duration("duration", duration),
				slog.String("client_ip", c.RealIP()),
			}

			if user := auth.GetCurrentUser(c); user != nil {
				attrs = append(attrs, slog.Int("user_id", user.ID))
			}

			switch {
			case statusCode >= 500:
				slog.Error("Request failed", attrs...)
			case statusCode >= 400:
				slog.Warn("Client error", attrs...)
			default:
				slog.Info("Request completed", attrs...)
			}

			return err
		}
	}
}
