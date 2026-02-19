# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DC Management Tool — an internal Go web application for creating and managing delivery challans across multiple projects. Built with Go/Echo v4 + HTMX + Alpine.js + Tailwind CSS, using SQLite. Templates are compiled via a-h/templ (type-safe Go templating).

## Common Commands

Uses [go-task](https://taskfile.dev) (Taskfile.yml). Legacy Makefile also available.

```bash
task dev          # Generate templ files, then run dev server with hot reload (Air)
task build        # Build binary to bin/dc-management-tool
task run          # Build and run production binary
task test         # Run all Go tests (verbose)
task fmt          # Format Go code
task lint         # Run golangci-lint
task migrate      # Run database migrations (goose)
task seed         # Seed test data
task setup        # Install deps and initialize directories
task clean        # Clean build artifacts
task tailwind     # Build Tailwind CSS
task templ:gen    # Generate Go code from .templ files (templ generate ./components)
task templ:fmt    # Format .templ files
```

Run a single test: `task test:one -- -run TestName ./internal/services/`

## Architecture

**Entry point:** `cmd/server/main.go` — initializes SQLite, SCS sessions, Echo router, CSRF middleware (gorilla/csrf wrapped as Echo middleware), and defines all routes.

**Key directories:**
- `internal/database/` — Data access layer (one file per entity, global `DB` instance)
- `internal/handlers/` — HTTP handlers (one file per feature area, use templ components)
- `internal/models/` — Structs with `Validate() map[string]string` methods
- `internal/services/` — Business logic (DC numbering, PDF/Excel export, financial year)
- `internal/auth/` — Session management, password hashing, user context helpers
- `internal/middleware/` — Auth middleware (`RequireAuth`, `ProjectContext`, `RequireRole`)
- `internal/migrations/` — Numbered SQL migration files embedded via `migrations.FS`, run automatically on startup
- `internal/components/` — `render.go` with `components.RenderOK(c, component)` and `components.Render(c, status, component)` helpers; also defines `PageProps` and `BreadcrumbItem`
- `components/` — All a-h/templ source files (`.templ`) and generated Go code (`.templ.go`)
  - `components/layouts/` — Base and main layout components
  - `components/pages/` — Full page components
  - `components/htmx/` — Partial components for HTMX responses (no layout)
  - `components/standalone/` — Login/health (no layout)
  - `components/partials/` — Shared components (sidebar, topbar, breadcrumb)

**Template system:** a-h/templ v0.3.977 — type-safe Go templates compiled to Go code.
- The `templates/` HTML directory has been removed. All templates are now `.templ` files under `components/`.
- Run `task templ:gen` after modifying any `.templ` file to regenerate the Go code.
- Handlers call `components.RenderOK(c, someComponent(...))` instead of `c.Render()`.
- `internal/helpers/templates.go` is kept only for `TemplateFuncs` usage in `export_html_builder.go`.

**Alpine.js** (v3.14.9 via CDN):
- Loaded in `components/layouts/base.templ` with `alpine:init` store registration.
- Global store: `Alpine.store('sidebar', { open, collapsed, toggle, close, openSidebar, toggleCollapse })`.
- `[x-cloak] { display: none !important }` in `static/css/design-system.css`.
- `static/js/sidebar.js` and `static/js/dc_lifecycle.js` have been deleted; logic is now inline Alpine.js.
- Go-to-Alpine data passing: use `data-*` attributes, read via `$el.dataset.*` in Alpine expressions.

## Key Patterns

- **Handler pattern:** `auth.GetCurrentUser(c)` for user context; pass `CSRFToken`/`CSRFField` via `components.PageProps` to templ components
- **Rendering:** `components.RenderOK(c, component)` for 200 responses; `components.Render(c, status, component)` for other status codes
- **Flash messages:** `auth.SetFlash(r, "success"|"error", msg)` / `auth.PopFlash(r)`
- **Database:** Direct SQL queries via `database/sql`, no ORM. Functions follow `Get*`, `Create*`, `Update*`, `Delete*` naming
- **DC numbering:** `PREFIX-TDC/ODC-YYYY-SEQUENCE` format using financial year (April–March)
- **Sessions:** SCS with SQLite store, 7-day lifetime, 24h idle timeout
- **CSRF:** gorilla/csrf wrapped as Echo middleware; token passed as string to templ components

## templ v0.3.977 Gotchas

- No `templ.SafeJS` or `templ.SafeScript` — use `templ.ComponentScript{Call: "..."}` or data attributes
- No explicit `import "github.com/a-h/templ"` needed when `templ.Component` is used as a param (auto-imported)
- `csrf.Token(c.Request())` returns a string; use this, not `csrf.TemplateField()`, in templ components
- HTMX subdirectories can share the `package htmx` name — alias on import in handlers if needed

## Environment Variables

```
APP_ENV=development
SERVER_ADDRESS=:8080
DATABASE_PATH=./data/dc_management.db
SESSION_SECRET=dev-secret-change-in-production
UPLOAD_PATH=./static/uploads
```

## Dependencies

Go 1.25.5+, labstack/echo v4, a-h/templ v0.3.977, alexedwards/scs (sessions), gorilla/csrf, modernc.org/sqlite (pure Go, no CGO), golang.org/x/crypto (bcrypt), pressly/goose v3 (migrations), xuri/excelize v2 (Excel export), chromedp (PDF export).
