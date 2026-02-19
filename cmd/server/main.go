package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/csrf"
	echov4 "github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/config"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/handlers"
	appmiddleware "github.com/narendhupati/dc-management-tool/internal/middleware"
	"github.com/narendhupati/dc-management-tool/internal/migrations"
	staticfiles "github.com/narendhupati/dc-management-tool/static"
)

func initLogger(env string) {
	var handler slog.Handler
	if env == "development" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	slog.SetDefault(slog.New(handler))
}

func main() {
	cfg := config.Load()
	initLogger(cfg.Environment)

	// Ensure upload directory exists (not embedded, lives on disk)
	if err := os.MkdirAll(cfg.UploadPath, 0o755); err != nil {
		slog.Error("Failed to create upload directory", slog.String("path", cfg.UploadPath), slog.String("error", err.Error()))
		os.Exit(1)
	}

	db, err := database.Init(cfg.DatabasePath)
	if err != nil {
		slog.Error("Failed to initialize database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrationsWithGoose(db, migrations.FS); err != nil {
		slog.Error("Failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1) //nolint:gocritic
	}

	// Initialize SCS session manager with SQLite store
	isSecure := cfg.Environment == "production"
	auth.InitSessionManager(db, isSecure)

	e := echov4.New()
	e.HideBanner = true

	// Embedded CSS/JS served from binary
	e.StaticFS("/static", staticfiles.FS)
	// User uploads served from disk (more specific path, takes priority)
	e.Static("/static/uploads", cfg.UploadPath)

	// CSRF middleware setup
	csrfOpts := []csrf.Option{
		csrf.Secure(isSecure),
		csrf.SameSite(csrf.SameSiteLaxMode),
	}
	if !isSecure {
		csrfOpts = append(csrfOpts, csrf.TrustedOrigins([]string{"localhost:" + cfg.ServerAddress[1:]}))
	}
	csrfProtect := csrf.Protect([]byte(cfg.SessionSecret), csrfOpts...)

	// Wrap SCS session manager as Echo middleware
	sessionMiddleware := func(next echov4.HandlerFunc) echov4.HandlerFunc {
		return func(c echov4.Context) error {
			var handlerErr error
			auth.SessionManager.LoadAndSave(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Update both request and response writer so SCS can inject the
					// session cookie before WriteHeader is committed on the wire.
					c.SetRequest(r)
					c.Response().Writer = w
					handlerErr = next(c)
				}),
			).ServeHTTP(c.Response().Writer, c.Request())
			return handlerErr
		}
	}

	// Wrap gorilla/csrf as Echo middleware
	csrfMiddleware := func(next echov4.HandlerFunc) echov4.HandlerFunc {
		return func(c echov4.Context) error {
			var handlerErr error
			var innerHandler http.Handler = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				c.SetRequest(r)
				handlerErr = next(c)
			})
			if !isSecure {
				innerHandler = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					c.SetRequest(csrf.PlaintextHTTPRequest(r))
					handlerErr = next(c)
				})
			}
			csrfProtect(innerHandler).ServeHTTP(c.Response().Writer, c.Request())
			return handlerErr
		}
	}

	// Global middleware: recover, logging, session, CSRF
	e.Use(echomiddleware.Recover())
	e.Use(appmiddleware.RequestLoggingMiddleware())
	e.Use(sessionMiddleware)
	e.Use(csrfMiddleware)

	// Public routes
	e.GET("/login", handlers.ShowLogin)
	e.POST("/login", handlers.ProcessLogin)
	e.GET("/logout", handlers.Logout)
	e.GET("/health", handlers.HealthCheck)
	e.GET("/ready", handlers.ReadinessCheck)

	// Protected routes
	protected := e.Group("")
	protected.Use(appmiddleware.RequireAuth())
	{
		// Root redirect — sends user to their last project or project selector
		protected.GET("/", func(c echov4.Context) error {
			user := auth.GetCurrentUser(c)
			if user != nil {
				return handlers.RedirectToProject(c, user.ID)
			}
			return c.Redirect(http.StatusFound, "/login")
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
	adminRoutes := e.Group("/admin")
	adminRoutes.Use(appmiddleware.RequireAuth(), appmiddleware.RequireRole("admin"))
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
	projectRoutes := e.Group("/projects/:id")
	projectRoutes.Use(appmiddleware.RequireAuth(), appmiddleware.ProjectContext())
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
		projectRoutes.GET("/bill-to", func(c echov4.Context) error {
			return c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/projects/%s/addresses?tab=bill_to", c.Param("id")))
		})
		projectRoutes.GET("/ship-to", func(c echov4.Context) error {
			return c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/projects/%s/addresses?tab=ship_to", c.Param("id")))
		})
	}

	slog.Info("Starting server",
		slog.String("address", cfg.ServerAddress),
		slog.String("environment", cfg.Environment),
	)
	if err := e.Start(cfg.ServerAddress); err != nil && err != http.ErrServerClosed {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
