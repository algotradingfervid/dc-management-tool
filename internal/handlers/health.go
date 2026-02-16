package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
}

func HealthCheck(c *gin.Context) {
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

	if c.GetHeader("Accept") == "text/html" || c.Request.URL.Query().Get("format") == "html" {
		c.HTML(http.StatusOK, "health.html", response)
		return
	}

	c.JSON(http.StatusOK, response)
}
