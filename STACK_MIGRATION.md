# DC Management Tool — Stack Migration Guide

## Overview

This document details the migration from the current Go/Gin + html/template + raw SQL stack to an improved stack optimized for developer productivity and long-term maintainability.

**Current Stack:** Go + Gin + html/template + database/sql + mattn/go-sqlite3 + Makefile
**Target Stack:** Go + Echo + Templ + sqlc + goose + modernc/sqlite + Alpine.js + Taskfile

---

## Complete Stack Reference

### Core

| Layer | Package | Version | Purpose |
|-------|---------|---------|---------|
| Language | Go | 1.23+ | Runtime |
| Router | labstack/echo/v4 | v4.12+ | HTTP routing, middleware, request binding |
| Database Driver | modernc.org/sqlite | latest | Pure Go SQLite, no CGO |
| Sessions | alexedwards/scs/v2 | v2 | Keep existing session management |
| CSRF | gorilla/csrf | v1 | Keep existing CSRF protection |

### Data Layer

| Layer | Package | Purpose |
|-------|---------|---------|
| SQL Codegen | sqlc/sqlc | Generates type-safe Go from SQL queries |
| Migrations | pressly/goose/v3 | Embedded SQL migrations with up/down |
| Validation | go-playground/validator/v10 | Struct tag-based input validation |

### Frontend

| Layer | Tool | Purpose |
|-------|------|---------|
| Templates | a-h/templ | Type-safe Go template components |
| Interactivity | htmx.org | Server-driven UI updates (keep as-is) |
| Client Logic | Alpine.js 3.x | Lightweight JS for dropdowns, modals, toggles |
| Styling | Tailwind CSS v4 | Utility-first CSS (keep as-is, use standalone CLI) |

### Dev Tooling

| Tool | Purpose |
|------|---------|
| air | Hot reload during development |
| go-task/task | YAML-based task runner (replaces Makefile) |
| golangci-lint | Static analysis and linting |
| templ generate --watch | Auto-regenerate Go from .templ on save |
| sqlc generate | Regenerate Go from SQL queries |

### Testing

| Tool | Purpose |
|------|---------|
| stretchr/testify | Readable assertions: assert.Equal, require.NoError |
| Real SQLite in tests | sqlc-generated code tested against real SQLite |
| net/http/httptest | HTTP handler testing (stdlib) |

### Observability

| Tool | Purpose |
|------|---------|
| log/slog (stdlib) | Structured JSON logging (Go 1.21+) |
| otelecho | OpenTelemetry middleware for request tracing (add later) |

### Deployment

| Tool | Purpose |
|------|---------|
| Docker | Single-stage build (no CGO = no gcc needed) |
| Litestream | Continuous SQLite replication to S3 for backup |
| Docker Compose | Container orchestration (keep existing) |

---

## Target Project Structure

```
ProjectManagementTool/
├── cmd/
│   └── server/
│       └── main.go                    # Echo setup, goose migrations, routes
│
├── internal/
│   ├── handlers/                      # HTTP handlers (one file per feature)
│   │   ├── dashboard.go
│   │   ├── delivery_challans.go
│   │   ├── products.go
│   │   ├── projects.go
│   │   ├── dc_templates.go
│   │   ├── addresses.go
│   │   ├── transporters.go
│   │   ├── dc_bundles.go
│   │   ├── reports.go
│   │   ├── shipment_wizard.go
│   │   ├── serial_search.go
│   │   ├── serial_validation.go
│   │   ├── export_handler.go
│   │   ├── auth.go
│   │   └── user_management.go
│   │
│   ├── services/                      # Business logic
│   │   ├── dc_numbering.go
│   │   ├── dc_generation.go
│   │   ├── excel_service.go
│   │   └── export_html_builder.go
│   │
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── project_context.go
│   │   └── role_auth.go
│   │
│   └── auth/                          # Session helpers, password hashing
│       └── auth.go
│
├── db/
│   ├── sqlc.yaml                      # sqlc configuration
│   ├── queries/                       # SQL query files for sqlc
│   │   ├── users.sql
│   │   ├── projects.sql
│   │   ├── products.sql
│   │   ├── addresses.sql
│   │   ├── delivery_challans.sql
│   │   ├── dc_templates.sql
│   │   ├── dc_bundles.sql
│   │   ├── transporters.sql
│   │   ├── shipment_groups.sql
│   │   ├── reports.sql
│   │   └── dashboard.sql
│   │
│   ├── migrations/                    # goose SQL migration files
│   │   ├── 00001_initial_schema.sql
│   │   ├── 00002_user_roles.sql
│   │   └── ...
│   │
│   └── generated/                     # sqlc output (auto-generated, do not edit)
│       ├── db.go
│       ├── models.go
│       ├── users.sql.go
│       ├── projects.sql.go
│       └── ...
│
├── components/                        # Templ template files
│   ├── layouts/
│   │   ├── base.templ                 # HTML skeleton: head, body, scripts
│   │   └── main.templ                 # Authenticated layout: sidebar + topbar + content
│   │
│   ├── pages/
│   │   ├── dashboard.templ
│   │   ├── delivery_challans/
│   │   │   ├── list.templ
│   │   │   ├── detail.templ
│   │   │   ├── official_detail.templ
│   │   │   └── official_print.templ
│   │   ├── products/
│   │   │   └── list.templ
│   │   ├── projects/
│   │   │   ├── list.templ
│   │   │   ├── select.templ
│   │   │   ├── settings.templ
│   │   │   └── create_wizard.templ
│   │   ├── dc_templates/
│   │   │   ├── list.templ
│   │   │   └── detail.templ
│   │   ├── addresses/
│   │   │   └── index.templ
│   │   ├── transporters/
│   │   │   └── list.templ (and partials)
│   │   ├── dc_bundles/
│   │   │   └── ...
│   │   ├── reports/
│   │   │   └── ...
│   │   ├── shipments/
│   │   │   └── ...
│   │   ├── admin/
│   │   │   └── ...
│   │   ├── serial_search.templ
│   │   └── error.templ
│   │
│   ├── partials/
│   │   ├── sidebar.templ
│   │   ├── topbar.templ
│   │   ├── breadcrumb.templ
│   │   └── flash.templ
│   │
│   ├── htmx/                          # Partial responses for HTMX swaps
│   │   ├── dashboard_stats.templ
│   │   ├── dc_templates_form.templ
│   │   ├── products_form.templ
│   │   ├── products_table.templ
│   │   ├── address_selector.templ
│   │   ├── reports/
│   │   │   └── ...
│   │   ├── transporters/
│   │   │   └── ...
│   │   └── admin/
│   │       └── ...
│   │
│   └── standalone/
│       └── login.templ
│
├── static/
│   ├── css/
│   │   ├── tailwind-input.css
│   │   ├── tailwind-output.css
│   │   └── design-system.css
│   └── js/
│       ├── sidebar.js
│       └── quantity-matrix.js
│
├── Taskfile.yml                       # Replaces Makefile
├── sqlc.yaml                          # Can also live at root
├── .golangci.yml                      # Linter configuration
├── .air.toml                          # Hot reload config
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── CLAUDE.md
```

---

## Migration Phases

### Phase 1: modernc/sqlite (30 minutes)

**Goal:** Remove CGO dependency. Zero behavior change.

**Steps:**

1. Replace the import in all files that reference the SQLite driver:
   ```go
   // Before
   import _ "github.com/mattn/go-sqlite3"

   // After
   import _ "modernc.org/sqlite"
   ```

2. Update the driver name in database open calls:
   ```go
   // Before
   db, err := sql.Open("sqlite3", dbPath)

   // After
   db, err := sql.Open("sqlite", dbPath)
   ```

3. Update go.mod:
   ```bash
   go get modernc.org/sqlite
   go mod tidy
   ```

4. Update Dockerfile — remove gcc/build-essential dependencies:
   ```dockerfile
   # Before
   RUN apk add --no-cache gcc musl-dev

   # After — not needed anymore
   ```

5. Test: `go test ./...` — everything should pass unchanged.

**Verification:** Application starts, all existing features work, tests pass.

---

### Phase 2: goose Migrations (1-2 hours)

**Goal:** Proper migration tooling with embedded SQL.

**Steps:**

1. Install goose:
   ```bash
   go install github.com/pressly/goose/v3/cmd/goose@latest
   ```

2. Add goose to go.mod:
   ```bash
   go get github.com/pressly/goose/v3
   ```

3. Move existing migration files to `db/migrations/`. Rename to goose format:
   ```
   migrations/000013_user_last_project.up.sql
   →  db/migrations/00013_user_last_project.sql
   ```

4. Convert each migration pair (up + down) into a single goose file:
   ```sql
   -- db/migrations/00013_user_last_project.sql

   -- +goose Up
   ALTER TABLE users ADD COLUMN last_project_id INTEGER;

   -- +goose Down
   ALTER TABLE users DROP COLUMN last_project_id;
   ```

5. Embed and run migrations in main.go:
   ```go
   import (
       "embed"
       "github.com/pressly/goose/v3"
   )

   //go:embed db/migrations/*.sql
   var migrations embed.FS

   func runMigrations(db *sql.DB) error {
       goose.SetBaseFS(migrations)
       goose.SetDialect("sqlite")
       return goose.Up(db, "db/migrations")
   }
   ```

6. Remove the old custom migration runner code from main.go.

**Verification:** Fresh database initializes correctly. Existing database with data still works.

---

### Phase 3: sqlc — Incremental Entity Migration (1-2 days)

**Goal:** Replace hand-written database layer with generated code. Migrate one entity at a time.

**Steps:**

1. Install sqlc:
   ```bash
   go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
   ```

2. Create sqlc configuration at `db/sqlc.yaml`:
   ```yaml
   version: "2"
   sql:
     - engine: "sqlite"
       queries: "queries"
       schema: "migrations"
       gen:
         go:
           package: "generated"
           out: "generated"
           emit_json_tags: true
           emit_empty_slices: true
   ```

3. Migrate entities one at a time, starting with the simplest. For each entity:

   a. **Write the query file** (`db/queries/products.sql`):
   ```sql
   -- name: GetProduct :one
   SELECT * FROM products WHERE id = ? AND project_id = ?;

   -- name: ListProducts :many
   SELECT * FROM products
   WHERE project_id = ?
   ORDER BY name ASC
   LIMIT ? OFFSET ?;

   -- name: CreateProduct :one
   INSERT INTO products (project_id, name, unit, hsn_code, created_by)
   VALUES (?, ?, ?, ?, ?)
   RETURNING *;

   -- name: UpdateProduct :one
   UPDATE products
   SET name = ?, unit = ?, hsn_code = ?, updated_at = CURRENT_TIMESTAMP
   WHERE id = ? AND project_id = ?
   RETURNING *;

   -- name: DeleteProduct :exec
   DELETE FROM products WHERE id = ? AND project_id = ?;

   -- name: CountProducts :one
   SELECT COUNT(*) FROM products WHERE project_id = ?;
   ```

   b. **Generate Go code:**
   ```bash
   cd db && sqlc generate
   ```

   c. **Update handlers** to use generated code instead of old database functions:
   ```go
   // Before
   product, err := database.GetProduct(id, projectID)

   // After
   queries := generated.New(db)
   product, err := queries.GetProduct(ctx, generated.GetProductParams{
       ID:        id,
       ProjectID: projectID,
   })
   ```

   d. **Delete the old** `internal/database/<entity>.go` file once all callers are migrated.

4. **Migration order** (simplest to most complex):
   1. `products` — simple CRUD
   2. `transporters` — simple CRUD
   3. `addresses` — simple CRUD
   4. `projects` — CRUD + settings
   5. `users` — CRUD + auth queries
   6. `dc_templates` — CRUD + sort order
   7. `dc_bundles` — CRUD + relations
   8. `delivery_challans` — most complex, many joins and filters
   9. `shipment_groups` — relations
   10. `dashboard` — aggregation queries
   11. `reports` — complex queries

**Verification:** After each entity migration, run `go test ./...` and manually test the feature.

---

### Phase 4: go-playground/validator (2-3 hours)

**Goal:** Replace hand-written `Validate()` methods with struct tags.

**Steps:**

1. Add dependency:
   ```bash
   go get github.com/go-playground/validator/v10
   ```

2. Create a shared validator instance in `internal/helpers/validator.go`:
   ```go
   package helpers

   import (
       "github.com/go-playground/validator/v10"
   )

   var Validate *validator.Validate

   func init() {
       Validate = validator.New()
       // Register custom validations if needed
       Validate.RegisterValidation("dc_number", validateDCNumber)
   }

   // Convert validator errors to map[string]string for templates
   func ValidationErrors(err error) map[string]string {
       errors := make(map[string]string)
       if validationErrors, ok := err.(validator.ValidationErrors); ok {
           for _, e := range validationErrors {
               field := e.Field()
               switch e.Tag() {
               case "required":
                   errors[field] = field + " is required"
               case "min":
                   errors[field] = field + " must be at least " + e.Param() + " characters"
               case "max":
                   errors[field] = field + " must be at most " + e.Param() + " characters"
               case "email":
                   errors[field] = "Invalid email address"
               default:
                   errors[field] = field + " is invalid"
               }
           }
       }
       return errors
   }
   ```

3. Update models with struct tags. Example for Product:
   ```go
   // Before
   type Product struct {
       ID        int64
       ProjectID int64
       Name      string
       Unit      string
       HSNCode   string
   }

   func (p *Product) Validate() map[string]string {
       errors := make(map[string]string)
       if p.Name == "" {
           errors["Name"] = "Product name is required"
       }
       if p.Unit == "" {
           errors["Unit"] = "Unit is required"
       }
       return errors
   }

   // After
   type Product struct {
       ID        int64  `json:"id"`
       ProjectID int64  `json:"project_id" validate:"required"`
       Name      string `json:"name" validate:"required,min=2,max=100"`
       Unit      string `json:"unit" validate:"required"`
       HSNCode   string `json:"hsn_code" validate:"max=20"`
   }
   ```

4. Update handlers to use the new validator:
   ```go
   // Before
   errors := product.Validate()
   if len(errors) > 0 { ... }

   // After
   err := helpers.Validate.Struct(product)
   if err != nil {
       errors := helpers.ValidationErrors(err)
       ...
   }
   ```

5. Migrate one model at a time. Delete the old `Validate()` method after migration.

**Verification:** Submit forms with invalid data — error messages appear correctly.

---

### Phase 5: Echo Router Migration (3-4 hours)

**Goal:** Replace Gin with Echo for better middleware and binding.

**Steps:**

1. Add dependency:
   ```bash
   go get github.com/labstack/echo/v4
   go get github.com/labstack/echo/v4/middleware
   ```

2. Update `cmd/server/main.go`:
   ```go
   // Before (Gin)
   r := gin.Default()
   r.GET("/dashboard", handlers.Dashboard)

   // After (Echo)
   e := echo.New()
   e.Use(middleware.Logger())
   e.Use(middleware.Recover())
   e.GET("/dashboard", handlers.Dashboard)
   ```

3. Update all handler signatures:
   ```go
   // Before (Gin)
   func Dashboard(c *gin.Context) {
       projectID := c.Param("id")
       c.HTML(200, "dashboard.html", data)
   }

   // After (Echo)
   func Dashboard(c echo.Context) error {
       projectID := c.Param("id")
       return c.Render(200, "dashboard", data)
       // Or with Templ (Phase 6):
       return components.Dashboard(data).Render(c.Request().Context(), c.Response())
   }
   ```

4. Update middleware signatures:
   ```go
   // Before (Gin)
   func RequireAuth() gin.HandlerFunc {
       return func(c *gin.Context) {
           // ...
           c.Next()
       }
   }

   // After (Echo)
   func RequireAuth() echo.MiddlewareFunc {
       return func(next echo.HandlerFunc) echo.HandlerFunc {
           return func(c echo.Context) error {
               // ...
               return next(c)
           }
       }
   }
   ```

5. Update route groups:
   ```go
   // Echo route groups
   api := e.Group("/projects/:projectId")
   api.Use(middleware.RequireAuth(), middleware.ProjectContext())

   api.GET("/dashboard", handlers.Dashboard)
   api.GET("/products", handlers.ListProducts)
   api.POST("/products", handlers.CreateProduct)
   ```

6. Update session and CSRF integration to work with Echo.

7. Remove Gin from go.mod:
   ```bash
   go mod tidy
   ```

**Verification:** All routes work. Auth flow works. CSRF protection works. Flash messages work.

---

### Phase 6: Templ Templates (2-3 days)

**Goal:** Replace html/template files with type-safe Templ components.

**Steps:**

1. Install templ:
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   ```

2. Start with the layout components. Create `components/layouts/base.templ`:
   ```go
   package layouts

   templ Base(title string) {
       <!DOCTYPE html>
       <html lang="en">
       <head>
           <meta charset="UTF-8"/>
           <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
           <title>{ title } | DC Management</title>
           <link rel="stylesheet" href="/static/css/tailwind-output.css"/>
           <script src="/static/js/htmx.min.js"></script>
           <script src="/static/js/alpine.min.js" defer></script>
       </head>
       <body class="bg-gray-50">
           { children... }
       </body>
       </html>
   }
   ```

3. Create the authenticated layout `components/layouts/main.templ`:
   ```go
   package layouts

   import "ProjectManagementTool/components/partials"

   type PageData struct {
       Title      string
       User       User
       Project    Project
       CSRFField  string
       Flash      *Flash
       Breadcrumbs []Breadcrumb
   }

   templ Main(data PageData) {
       @Base(data.Title) {
           <div class="flex h-screen">
               @partials.Sidebar(data.User, data.Project)
               <div class="flex-1 flex flex-col overflow-hidden">
                   @partials.Topbar(data.User, data.Breadcrumbs)
                   <main class="flex-1 overflow-y-auto p-6">
                       if data.Flash != nil {
                           @partials.Flash(data.Flash)
                       }
                       { children... }
                   </main>
               </div>
           </div>
       }
   }
   ```

4. Convert pages one at a time. Example for products list `components/pages/products/list.templ`:
   ```go
   package products

   import (
       "ProjectManagementTool/components/layouts"
       "ProjectManagementTool/db/generated"
   )

   templ List(page layouts.PageData, products []generated.Product, count int64) {
       @layouts.Main(page) {
           <div class="flex justify-between items-center mb-6">
               <h1 class="text-2xl font-bold">Products</h1>
               <button
                   hx-get="/products/new"
                   hx-target="#modal"
                   class="btn btn-primary"
               >
                   Add Product
               </button>
           </div>
           <div class="bg-white rounded-lg shadow">
               <table class="min-w-full">
                   <thead>
                       <tr>
                           <th class="px-4 py-3 text-left">Name</th>
                           <th class="px-4 py-3 text-left">Unit</th>
                           <th class="px-4 py-3 text-left">HSN Code</th>
                           <th class="px-4 py-3 text-right">Actions</th>
                       </tr>
                   </thead>
                   <tbody>
                       for _, p := range products {
                           @ProductRow(p)
                       }
                   </tbody>
               </table>
           </div>
       }
   }

   templ ProductRow(p generated.Product) {
       <tr class="border-t">
           <td class="px-4 py-3">{ p.Name }</td>
           <td class="px-4 py-3">{ p.Unit }</td>
           <td class="px-4 py-3">{ p.HsnCode }</td>
           <td class="px-4 py-3 text-right">
               <button
                   hx-get={ "/products/" + fmt.Sprint(p.ID) + "/edit" }
                   hx-target="#modal"
                   class="text-blue-600 hover:underline"
               >
                   Edit
               </button>
           </td>
       </tr>
   }
   ```

5. Update handlers to render Templ components:
   ```go
   func ListProducts(c echo.Context) error {
       // ... fetch data ...
       component := products.List(pageData, productList, count)
       return component.Render(c.Request().Context(), c.Response())
   }
   ```

6. For HTMX partials, create small Templ components that render without layout:
   ```go
   // components/htmx/products_table.templ
   templ ProductsTable(products []generated.Product) {
       for _, p := range products {
           @ProductRow(p)
       }
   }
   ```

7. **Migration order:**
   1. Layouts (base, main) — foundation for everything else
   2. Partials (sidebar, topbar, breadcrumb, flash) — used everywhere
   3. Standalone (login) — simple, good practice
   4. Simplest pages (products, transporters)
   5. Medium complexity (projects, addresses, dc_templates)
   6. Complex pages (delivery_challans, reports, shipments)
   7. HTMX partials last (they depend on page components)

8. Delete `internal/helpers/templates.go` and `internal/helpers/template.go` after all templates are migrated.

9. Delete the `templates/` directory.

**Verification:** Every page renders correctly. HTMX swaps work. Flash messages display.

---

### Phase 7: Alpine.js (1-2 hours)

**Goal:** Replace custom JavaScript with declarative Alpine.js.

**Steps:**

1. Add Alpine.js to the base layout (already done in Phase 6 base.templ).

2. Replace custom JS patterns:

   **Sidebar toggle:**
   ```html
   <!-- Before: custom JS in sidebar.js -->
   <div id="sidebar" class="...">...</div>
   <script src="/static/js/sidebar.js"></script>

   <!-- After: Alpine.js inline -->
   <div x-data="{ open: true }" class="flex">
       <aside x-show="open" class="w-64 ...">...</aside>
       <button @click="open = !open">Toggle</button>
   </div>
   ```

   **Dropdown menus:**
   ```html
   <div x-data="{ open: false }" class="relative">
       <button @click="open = !open">Menu</button>
       <div x-show="open" @click.outside="open = false" class="absolute ...">
           <!-- menu items -->
       </div>
   </div>
   ```

   **Confirm delete:**
   ```html
   <button
       x-data
       @click="if(confirm('Delete this product?')) $el.closest('form').submit()"
   >
       Delete
   </button>
   ```

   **Form conditional fields:**
   ```html
   <div x-data="{ dcType: 'transit' }">
       <select x-model="dcType">
           <option value="transit">Transit</option>
           <option value="official">Official</option>
       </select>
       <div x-show="dcType === 'official'">
           <!-- official-only fields -->
       </div>
   </div>
   ```

3. Remove custom JS files that Alpine.js replaces. Keep `quantity-matrix.js` if it has complex logic that doesn't fit Alpine.

**Verification:** All interactive UI elements work — dropdowns, modals, toggles, conditional forms.

---

### Phase 8: Taskfile + Linting (1 hour)

**Goal:** Replace Makefile with Taskfile. Add linting.

**Steps:**

1. Install Task:
   ```bash
   brew install go-task
   ```

2. Install golangci-lint:
   ```bash
   brew install golangci-lint
   ```

3. Create `Taskfile.yml`:
   ```yaml
   version: '3'

   tasks:
     dev:
       desc: Run development server with hot reload
       deps: [generate]
       cmds:
         - air

     generate:
       desc: Generate all code (sqlc + templ)
       cmds:
         - sqlc generate -f db/sqlc.yaml
         - templ generate

     build:
       desc: Build production binary
       deps: [generate]
       cmds:
         - go build -o bin/dc-management-tool cmd/server/main.go

     run:
       desc: Build and run production binary
       deps: [build]
       cmds:
         - ./bin/dc-management-tool

     test:
       desc: Run all tests
       cmds:
         - go test -v ./...

     test:one:
       desc: Run a single test
       cmds:
         - go test -v -run {{.CLI_ARGS}} ./...

     lint:
       desc: Run linter
       cmds:
         - golangci-lint run ./...

     fmt:
       desc: Format all code
       cmds:
         - go fmt ./...
         - templ fmt .

     migrate:
       desc: Run database migrations
       cmds:
         - go run cmd/server/main.go -migrate

     seed:
       desc: Seed test data
       cmds:
         - sqlite3 data/dc_management.db < migrations/seed_data.sql

     tailwind:
       desc: Build Tailwind CSS
       cmds:
         - tailwindcss -i static/css/tailwind-input.css -o static/css/tailwind-output.css --minify

     tailwind:watch:
       desc: Watch Tailwind CSS for changes
       cmds:
         - tailwindcss -i static/css/tailwind-input.css -o static/css/tailwind-output.css --watch

     clean:
       desc: Remove build artifacts
       cmds:
         - rm -rf bin/
         - rm -rf db/generated/

     setup:
       desc: Install all dependencies and tools
       cmds:
         - go mod download
         - go install github.com/a-h/templ/cmd/templ@latest
         - go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
         - go install github.com/pressly/goose/v3/cmd/goose@latest
         - go install github.com/air-verse/air@latest
         - mkdir -p data static/uploads
   ```

4. Create `.golangci.yml`:
   ```yaml
   linters:
     enable:
       - errcheck
       - govet
       - staticcheck
       - unused
       - gosimple
       - ineffassign
     disable:
       - typecheck  # can conflict with generated code

   linters-settings:
     errcheck:
       check-blank: false

   issues:
     exclude-dirs:
       - db/generated
   ```

5. Delete the old `Makefile` (or keep it as an alias that calls `task`).

**Verification:** `task dev`, `task build`, `task test`, `task lint` all work.

---

### Phase 9: Structured Logging with slog (1-2 hours)

**Goal:** Replace `log.Println` with structured logging.

**Steps:**

1. Set up slog in main.go:
   ```go
   import "log/slog"

   func main() {
       // Development: human-readable
       if os.Getenv("APP_ENV") == "development" {
           slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
               Level: slog.LevelDebug,
           })))
       } else {
           // Production: JSON
           slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
               Level: slog.LevelInfo,
           })))
       }
   }
   ```

2. Replace log calls throughout the codebase:
   ```go
   // Before
   log.Printf("Created DC %s for project %d", dcNumber, projectID)
   log.Printf("Error: %v", err)

   // After
   slog.Info("dc.created", "number", dcNumber, "project_id", projectID)
   slog.Error("dc.creation_failed", "error", err, "project_id", projectID)
   ```

3. Add request logging middleware:
   ```go
   func RequestLogger() echo.MiddlewareFunc {
       return func(next echo.HandlerFunc) echo.HandlerFunc {
           return func(c echo.Context) error {
               start := time.Now()
               err := next(c)
               slog.Info("http.request",
                   "method", c.Request().Method,
                   "path", c.Request().URL.Path,
                   "status", c.Response().Status,
                   "duration_ms", time.Since(start).Milliseconds(),
               )
               return err
           }
       }
   }
   ```

**Verification:** Logs are structured. Dev shows text, production shows JSON.

---

### Phase 10: Litestream Backup (30 minutes)

**Goal:** Continuous SQLite backup to S3.

**Steps:**

1. Add Litestream to docker-compose.yml:
   ```yaml
   services:
     app:
       build: .
       volumes:
         - ./data:/app/data

     litestream:
       image: litestream/litestream:latest
       volumes:
         - ./data:/data
         - ./litestream.yml:/etc/litestream.yml
       command: replicate
       depends_on:
         - app
   ```

2. Create `litestream.yml`:
   ```yaml
   dbs:
     - path: /data/dc_management.db
       replicas:
         - type: s3
           bucket: your-backup-bucket
           path: dc-management
           region: ap-south-1
   ```

3. Alternatively, embed Litestream in the Dockerfile for single-container deployment:
   ```dockerfile
   FROM golang:1.23-alpine AS builder
   RUN go build -o /app/server cmd/server/main.go

   FROM alpine:latest
   COPY --from=builder /app/server /app/server
   COPY --from=litestream/litestream:latest /usr/local/bin/litestream /usr/local/bin/litestream
   COPY litestream.yml /etc/litestream.yml
   CMD ["litestream", "replicate", "-exec", "/app/server"]
   ```

**Verification:** Database changes replicate to S3. Restore from backup works.

---

## Migration Checklist

```
Phase 1: modernc/sqlite
  [ ] Replace import and driver name
  [ ] Update go.mod
  [ ] Remove CGO from Dockerfile
  [ ] Run tests

Phase 2: goose
  [ ] Install goose
  [ ] Convert migration files to goose format
  [ ] Embed migrations in main.go
  [ ] Remove old migration runner
  [ ] Test fresh DB and existing DB

Phase 3: sqlc (per entity)
  [ ] products
  [ ] transporters
  [ ] addresses
  [ ] projects
  [ ] users
  [ ] dc_templates
  [ ] dc_bundles
  [ ] delivery_challans
  [ ] shipment_groups
  [ ] dashboard
  [ ] reports
  [ ] Remove old internal/database/ files

Phase 4: validator
  [ ] Create shared validator
  [ ] Migrate product model
  [ ] Migrate project model
  [ ] Migrate user model
  [ ] Migrate address model
  [ ] Migrate dc_template model
  [ ] Migrate delivery_challan model
  [ ] Migrate transporter model
  [ ] Remove old Validate() methods

Phase 5: Echo
  [ ] Replace Gin with Echo in main.go
  [ ] Update all handler signatures
  [ ] Update all middleware
  [ ] Update route groups
  [ ] Update session integration
  [ ] Update CSRF integration
  [ ] Remove Gin from go.mod

Phase 6: Templ
  [ ] base layout
  [ ] main layout
  [ ] sidebar partial
  [ ] topbar partial
  [ ] breadcrumb partial
  [ ] flash partial
  [ ] login page
  [ ] dashboard page
  [ ] products pages
  [ ] projects pages
  [ ] dc_templates pages
  [ ] addresses pages
  [ ] transporters pages
  [ ] delivery_challans pages
  [ ] dc_bundles pages
  [ ] reports pages
  [ ] shipments pages
  [ ] admin pages
  [ ] serial_search page
  [ ] error page
  [ ] All HTMX partials
  [ ] Remove templates/ directory
  [ ] Remove template helpers

Phase 7: Alpine.js
  [ ] Add Alpine.js to base layout
  [ ] Migrate sidebar toggle
  [ ] Migrate dropdown menus
  [ ] Migrate modal interactions
  [ ] Migrate conditional form fields
  [ ] Remove replaced JS files

Phase 8: Taskfile + Linting
  [ ] Create Taskfile.yml
  [ ] Create .golangci.yml
  [ ] Verify all tasks work
  [ ] Remove Makefile

Phase 9: slog
  [ ] Set up slog in main.go
  [ ] Replace log.Printf calls
  [ ] Add request logging middleware

Phase 10: Litestream
  [ ] Create litestream.yml
  [ ] Update docker-compose.yml
  [ ] Test backup and restore
```

---

## References

| Tool | Documentation |
|------|--------------|
| Templ | https://templ.guide |
| sqlc | https://docs.sqlc.dev |
| goose | https://github.com/pressly/goose |
| Echo | https://echo.labstack.com |
| go-playground/validator | https://github.com/go-playground/validator |
| modernc/sqlite | https://pkg.go.dev/modernc.org/sqlite |
| Alpine.js | https://alpinejs.dev |
| go-task | https://taskfile.dev |
| golangci-lint | https://golangci-lint.run |
| slog | https://pkg.go.dev/log/slog |
| Litestream | https://litestream.io |
| HTMX | https://htmx.org |
| Tailwind CSS | https://tailwindcss.com |
