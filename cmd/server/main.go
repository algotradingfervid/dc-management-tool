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

		// Product routes
		protected.GET("/projects/:id/products", handlers.ListProducts)
		protected.GET("/projects/:id/products/new", handlers.ShowAddProductForm)
		protected.POST("/projects/:id/products", handlers.CreateProductHandler)
		protected.GET("/projects/:id/products/:pid/edit", handlers.ShowEditProductForm)
		protected.POST("/projects/:id/products/:pid", handlers.UpdateProductHandler)
		protected.DELETE("/projects/:id/products/:pid", handlers.DeleteProductHandler)

		// DC Template routes
		protected.GET("/projects/:id/templates", handlers.ListTemplates)
		protected.GET("/projects/:id/templates/new", handlers.ShowCreateTemplateForm)
		protected.POST("/projects/:id/templates", handlers.CreateTemplateHandler)
		protected.GET("/projects/:id/templates/:tid", handlers.ShowTemplateDetail)
		protected.GET("/projects/:id/templates/:tid/edit", handlers.ShowEditTemplateForm)
		protected.POST("/projects/:id/templates/:tid", handlers.UpdateTemplateHandler)
		protected.DELETE("/projects/:id/templates/:tid", handlers.DeleteTemplateHandler)

		// Bill-to address routes
		protected.GET("/projects/:id/bill-to", handlers.ShowBillToPage)
		protected.POST("/projects/:id/bill-to/config", handlers.UpdateColumnConfig)
		protected.POST("/projects/:id/bill-to/upload", handlers.UploadAddresses)
		protected.POST("/projects/:id/bill-to/addresses", handlers.CreateAddressHandler)
		protected.POST("/projects/:id/bill-to/addresses/:aid", handlers.UpdateAddressHandler)
		protected.DELETE("/projects/:id/bill-to/addresses/:aid", handlers.DeleteAddressHandler)
		protected.DELETE("/projects/:id/bill-to/addresses", handlers.DeleteAllAddressesHandler)
		protected.GET("/projects/:id/bill-to/addresses/:aid", handlers.GetAddressJSON)

		// Transit DC routes
		protected.GET("/projects/:id/dcs/transit/new", handlers.ShowCreateTransitDC)
		protected.POST("/projects/:id/dcs/transit", handlers.CreateTransitDC)
		protected.GET("/projects/:id/templates/:tid/products", handlers.LoadTemplateProducts)

		// Official DC routes
		protected.GET("/projects/:id/dcs/official/new", handlers.ShowCreateOfficialDC)
		protected.POST("/projects/:id/dcs/official", handlers.CreateOfficialDC)

		// DC detail and lifecycle
		protected.GET("/projects/:id/dcs/:dcid", handlers.ShowDCDetail)
		protected.GET("/projects/:id/dcs/:dcid/print", handlers.ShowTransitDCPrintView)
		protected.GET("/projects/:id/dcs/:dcid/official-print", handlers.ShowOfficialDCPrintView)
		protected.POST("/projects/:id/dcs/:dcid/issue", handlers.IssueDCHandler)
		protected.DELETE("/projects/:id/dcs/:dcid", handlers.DeleteDCHandler)

		// Serial number validation API
		protected.POST("/api/serial-numbers/validate", handlers.ValidateSerialNumbers)

		// Ship-to address routes
		protected.GET("/projects/:id/ship-to", handlers.ShowShipToPage)
		protected.POST("/projects/:id/ship-to/config", handlers.UpdateShipToColumnConfig)
		protected.POST("/projects/:id/ship-to/upload", handlers.UploadShipToAddresses)
		protected.POST("/projects/:id/ship-to/addresses", handlers.CreateShipToAddressHandler)
		protected.POST("/projects/:id/ship-to/addresses/:aid", handlers.UpdateShipToAddressHandler)
		protected.DELETE("/projects/:id/ship-to/addresses/:aid", handlers.DeleteShipToAddressHandler)
		protected.DELETE("/projects/:id/ship-to/addresses", handlers.DeleteAllShipToAddressesHandler)
		protected.GET("/projects/:id/ship-to/addresses/:aid", handlers.GetShipToAddressJSON)
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
