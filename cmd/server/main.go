package main

import (
	"fmt"
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
		// Root redirect — sends user to their last project or project selector
		protected.GET("/", func(c *gin.Context) {
			user := auth.GetCurrentUser(c)
			if user != nil {
				handlers.RedirectToProject(c, user.ID)
			} else {
				c.Redirect(http.StatusFound, "/login")
			}
		})

		// Project selector page
		protected.GET("/projects/select", handlers.ShowProjectSelector)

		// Project routes (not project-scoped)
		protected.GET("/projects", handlers.ListProjects)
		protected.GET("/projects/new", handlers.ShowProjectForm)
		protected.POST("/projects", handlers.CreateProject)

		// Serial number validation API
		protected.POST("/api/serial-numbers/validate", handlers.ValidateSerialNumbers)
	}

	// Admin routes (requires admin role)
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(middleware.RequireAuth(), middleware.RequireRole("admin"))
	{
		adminRoutes.GET("/users", handlers.ListUsers)
		adminRoutes.GET("/users/new", handlers.ShowCreateUserForm)
		adminRoutes.POST("/users", handlers.CreateUserHandler)
		adminRoutes.GET("/users/:uid/edit", handlers.ShowEditUserForm)
		adminRoutes.POST("/users/:uid", handlers.UpdateUserHandler)
		adminRoutes.POST("/users/:uid/toggle-status", handlers.ToggleUserStatusHandler)
		adminRoutes.POST("/users/:uid/reset-password", handlers.ResetUserPasswordHandler)
	}

	// Project-scoped routes (with project context middleware)
	projectRoutes := router.Group("/projects/:id")
	projectRoutes.Use(middleware.RequireAuth(), middleware.ProjectContext())
	{
		// Dashboard
		projectRoutes.GET("/dashboard", handlers.ShowDashboard)

		// Project-scoped DC listing and serial search
		projectRoutes.GET("/dcs-list", handlers.ListAllDeliveryChallans)
		projectRoutes.GET("/serial-search", handlers.ShowSerialSearch)

		// Project detail/settings
		projectRoutes.GET("", handlers.ShowProject)
		projectRoutes.GET("/edit", handlers.ShowEditProjectForm)
		projectRoutes.POST("", handlers.UpdateProject)
		projectRoutes.DELETE("", handlers.DeleteProject)

		// Project settings
		projectRoutes.GET("/settings", handlers.ShowProjectSettings)
		projectRoutes.POST("/settings", handlers.UpdateProjectSettings)
		projectRoutes.GET("/settings/dc-preview", handlers.PreviewDCNumberAPI)

		// Product routes
		projectRoutes.GET("/products", handlers.ListProducts)
		projectRoutes.GET("/products/new", handlers.ShowAddProductForm)
		projectRoutes.POST("/products", handlers.CreateProductHandler)
		projectRoutes.GET("/products/:pid/edit", handlers.ShowEditProductForm)
		projectRoutes.POST("/products/:pid", handlers.UpdateProductHandler)
		projectRoutes.DELETE("/products/:pid", handlers.DeleteProductHandler)
		projectRoutes.POST("/products/bulk-delete", handlers.BulkDeleteProductsHandler)
		projectRoutes.POST("/products/import", handlers.ImportProductsHandler)
		projectRoutes.GET("/products/import-template", handlers.DownloadProductImportTemplate)

		// DC Template routes
		projectRoutes.GET("/templates", handlers.ListTemplates)
		projectRoutes.GET("/templates/new", handlers.ShowCreateTemplateForm)
		projectRoutes.POST("/templates", handlers.CreateTemplateHandler)
		projectRoutes.GET("/templates/:tid", handlers.ShowTemplateDetail)
		projectRoutes.GET("/templates/:tid/edit", handlers.ShowEditTemplateForm)
		projectRoutes.POST("/templates/:tid", handlers.UpdateTemplateHandler)
		projectRoutes.DELETE("/templates/:tid", handlers.DeleteTemplateHandler)
		projectRoutes.POST("/templates/:tid/duplicate", handlers.DuplicateTemplateHandler)

		// Unified address routes
		projectRoutes.GET("/addresses", handlers.ShowAddressesPage)
		projectRoutes.POST("/addresses/config", handlers.UpdateAddressColumnConfig)
		projectRoutes.POST("/addresses/upload", handlers.UploadAddressesHandler)
		projectRoutes.GET("/addresses/import-template", handlers.DownloadAddressImportTemplate)
		projectRoutes.POST("/addresses/create", handlers.CreateAddressUnified)
		projectRoutes.POST("/addresses/:aid", handlers.UpdateAddressUnified)
		projectRoutes.DELETE("/addresses/:aid", handlers.DeleteAddressUnified)
		projectRoutes.DELETE("/addresses", handlers.DeleteAllAddressesUnified)
		projectRoutes.GET("/addresses/:aid", handlers.GetAddressJSONUnified)
		projectRoutes.GET("/addresses/search", handlers.SearchAddressSelector)

		// Transporter routes
		projectRoutes.GET("/transporters", handlers.ListTransporters)
		projectRoutes.GET("/transporters/new", handlers.ShowAddTransporterForm)
		projectRoutes.POST("/transporters", handlers.CreateTransporterHandler)
		projectRoutes.GET("/transporters/:tid", handlers.ShowTransporterDetail)
		projectRoutes.GET("/transporters/:tid/edit", handlers.ShowEditTransporterForm)
		projectRoutes.POST("/transporters/:tid", handlers.UpdateTransporterHandler)
		projectRoutes.POST("/transporters/:tid/toggle-status", handlers.ToggleTransporterStatus)
		projectRoutes.POST("/transporters/:tid/vehicles", handlers.AddVehicleHandler)
		projectRoutes.DELETE("/transporters/:tid/vehicles/:vid", handlers.RemoveVehicleHandler)
		projectRoutes.GET("/api/transporters", handlers.GetTransportersJSON)

		// Shipment Wizard routes
		projectRoutes.GET("/shipments/new", handlers.ShowCreateShipmentWizard)
		projectRoutes.POST("/shipments/new/step2", handlers.ShipmentWizardStep2)
		projectRoutes.POST("/shipments/new/step3", handlers.ShipmentWizardStep3)
		projectRoutes.POST("/shipments/new/step4", handlers.ShipmentWizardStep4)
		projectRoutes.POST("/shipments", handlers.CreateShipment)
		projectRoutes.GET("/shipments/:gid", handlers.ShowShipmentGroup)
		projectRoutes.GET("/shipments", handlers.ListShipmentGroups)
		projectRoutes.POST("/shipments/:gid/issue", handlers.IssueShipmentGroup)

		// Template product loading (used by shipment wizard)
		projectRoutes.GET("/templates/:tid/products", handlers.LoadTemplateProducts)

		// DC detail and lifecycle
		projectRoutes.GET("/dcs/:dcid", handlers.ShowDCDetail)
		projectRoutes.GET("/dcs/:dcid/print", handlers.ShowTransitDCPrintView)
		projectRoutes.GET("/dcs/:dcid/official-print", handlers.ShowOfficialDCPrintView)
		projectRoutes.POST("/dcs/:dcid/issue", handlers.IssueDCHandler)
		projectRoutes.DELETE("/dcs/:dcid", handlers.DeleteDCHandler)

		// DC Export routes (PDF & Excel)
		projectRoutes.GET("/dcs/:dcid/export/pdf", handlers.ExportDCPDF)
		projectRoutes.GET("/dcs/:dcid/export/excel", handlers.ExportDCExcel)

		// Reports
		projectRoutes.GET("/reports", handlers.ShowReportsIndex)
		projectRoutes.GET("/reports/dc-summary", handlers.ShowDCSummaryReport)
		projectRoutes.GET("/reports/dc-summary/export", handlers.ExportDCSummaryExcel)
		projectRoutes.GET("/reports/destination", handlers.ShowDestinationReport)
		projectRoutes.GET("/reports/destination/export", handlers.ExportDestinationExcel)
		projectRoutes.GET("/reports/product", handlers.ShowProductReport)
		projectRoutes.GET("/reports/product/export", handlers.ExportProductExcel)
		projectRoutes.GET("/reports/serial", handlers.ShowSerialReport)
		projectRoutes.GET("/reports/serial/export", handlers.ExportSerialExcel)

		// Legacy address redirects (for backward compatibility)
		projectRoutes.GET("/bill-to", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/projects/%s/addresses?tab=bill_to", c.Param("id")))
		})
		projectRoutes.GET("/ship-to", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/projects/%s/addresses?tab=ship_to", c.Param("id")))
		})
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
	innerHandler := csrfMiddleware(auth.SessionManager.LoadAndSave(router))

	// Wrap with plaintext HTTP context for non-TLS (development) environments
	// gorilla/csrf v1.7.3 defaults to TLS mode unless PlaintextHTTPContextKey is set
	var handler http.Handler
	if !isSecure {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			innerHandler.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
		})
	} else {
		handler = innerHandler
	}

	log.Printf("Starting server on %s in %s mode", cfg.ServerAddress, cfg.Environment)
	if err := http.ListenAndServe(cfg.ServerAddress, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
