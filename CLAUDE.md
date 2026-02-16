# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DC Management Tool — an internal Go web application for creating and managing delivery challans across multiple projects. Built with Go/Gin + HTMX + Tailwind CSS, using SQLite.

## Common Commands

```bash
make dev          # Run dev server with hot reload (Air)
make build        # Build binary to bin/dc-management-tool
make run          # Build and run production binary
make test         # Run all Go tests (verbose)
make fmt          # Format Go code
make migrate      # Run database migrations
make seed         # Seed test data
make setup        # Install deps and initialize directories
```

Run a single test: `go test -v -run TestName ./internal/services/`

## Architecture

**Entry point:** `cmd/server/main.go` — initializes SQLite, SCS sessions, Gin router, CSRF middleware, and defines all routes.

**Key directories:**
- `internal/database/` — Data access layer (one file per entity, global `DB` instance)
- `internal/handlers/` — HTTP handlers (one file per feature area)
- `internal/models/` — Structs with `Validate() map[string]string` methods
- `internal/services/` — Business logic (DC numbering, PDF/Excel export, financial year)
- `internal/auth/` — Session management, password hashing, user context helpers
- `internal/middleware/` — Auth middleware (`RequireAuth`)
- `migrations/` — Numbered SQL migration files, run automatically on startup

**Template system** (`templates/`):
- `pages/*.html` — Full pages using base+main layout
- `htmx/*.html` — Partial templates for HTMX responses (no layout)
- `standalone/*.html` — Login/health (no layout)
- `partials/` — Shared components (sidebar, topbar, breadcrumb)
- Custom renderer in `internal/helpers/templates.go` with 15+ template functions

## Key Patterns

- **Handler pattern:** `auth.GetCurrentUser(c)` for user context, always pass `csrfField` to templates
- **Flash messages:** `auth.SetFlash(r, "success"|"error", msg)` / `auth.PopFlash(r)`
- **Database:** Direct SQL queries via `database/sql`, no ORM. Functions follow `Get*`, `Create*`, `Update*`, `Delete*` naming
- **DC numbering:** `PREFIX-TDC/ODC-YYYY-SEQUENCE` format using financial year (April–March)
- **Sessions:** SCS with SQLite store, 7-day lifetime, 24h idle timeout
- **CSRF:** gorilla/csrf wrapping the Gin router

## Environment Variables

```
APP_ENV=development
SERVER_ADDRESS=:8080
DATABASE_PATH=./data/dc_management.db
SESSION_SECRET=dev-secret-change-in-production
UPLOAD_PATH=./static/uploads
```

## Dependencies

Go 1.25.5+, gin-gonic/gin, alexedwards/scs (sessions), gorilla/csrf, mattn/go-sqlite3 (requires CGO), golang.org/x/crypto (bcrypt).
