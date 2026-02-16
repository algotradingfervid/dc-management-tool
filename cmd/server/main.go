package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/config"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/handlers"
)

func main() {
	cfg := config.Load()

	db, err := database.Init(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db, "./migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")
	router.GET("/health", handlers.HealthCheck)

	log.Printf("Starting server on %s in %s mode", cfg.ServerAddress, cfg.Environment)
	if err := router.Run(cfg.ServerAddress); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
