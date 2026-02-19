package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/components/standalone"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
}

func HealthCheck(c echo.Context) error {
	dbStatus := "disconnected"
	if database.DB != nil {
		if err := database.DB.Ping(); err == nil {
			dbStatus = "connected"
		}
	}

	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Database:  dbStatus,
	}

	if c.Request().Header.Get("Accept") == "text/html" || c.QueryParam("format") == "html" {
		return components.RenderOK(c, standalone.Health(response.Status, response.Database, ""))
	}

	return c.JSON(http.StatusOK, response)
}

func ReadinessCheck(c echo.Context) error {
	if database.DB == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status": "not_ready",
			"error":  "database not initialized",
		})
	}
	if err := database.DB.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status": "not_ready",
			"error":  err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now(),
	})
}
