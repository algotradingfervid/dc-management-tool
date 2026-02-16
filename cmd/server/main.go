package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/config"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/handlers"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/middleware"
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

	// Initialize SCS session manager with SQLite store
	isSecure := cfg.Environment == "production"
	auth.InitSessionManager(db, isSecure)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Set up custom template renderer with composition support
	renderer, err := helpers.NewTemplateRenderer("./templates", helpers.TemplateFuncs())
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}
	router.HTMLRender = renderer

	router.Static("/static", "./static")

	// Public routes
	router.GET("/login", handlers.ShowLogin)
	router.POST("/login", handlers.ProcessLogin)
	router.GET("/logout", handlers.Logout)
	router.GET("/health", handlers.HealthCheck)

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.RequireAuth())
	{
		protected.GET("/", handlers.ShowDashboard)

		// Project routes
		protected.GET("/projects", handlers.ListProjects)
		protected.GET("/projects/new", handlers.ShowProjectForm)
		protected.POST("/projects", handlers.CreateProject)
		protected.GET("/projects/:id", handlers.ShowProject)
		protected.GET("/projects/:id/edit", handlers.ShowEditProjectForm)
		protected.POST("/projects/:id", handlers.UpdateProject)
		protected.DELETE("/projects/:id", handlers.DeleteProject)
	}

	// Wrap with SCS session middleware + CSRF middleware
	csrfOpts := []csrf.Option{
		csrf.Secure(isSecure),
		csrf.SameSite(csrf.SameSiteLaxMode),
	}
	if !isSecure {
		csrfOpts = append(csrfOpts, csrf.TrustedOrigins([]string{"localhost:" + cfg.ServerAddress[1:]}))
	}
	csrfMiddleware := csrf.Protect(
		[]byte(cfg.SessionSecret),
		csrfOpts...,
	)

	// Stack: CSRF wraps SCS wraps Gin router
	handler := csrfMiddleware(auth.SessionManager.LoadAndSave(router))

	log.Printf("Starting server on %s in %s mode", cfg.ServerAddress, cfg.Environment)
	if err := http.ListenAndServe(cfg.ServerAddress, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
