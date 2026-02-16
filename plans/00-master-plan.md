# DC Management Tool - Master Implementation Plan

## Project Overview

Internal web application for creating and managing Delivery Challans (DCs) across multiple projects. Built with Go + Gin backend, HTMX + Tailwind frontend, and SQLite database.

## Known Library Versions

This project uses the following pinned library versions:

**Backend Libraries (Go 1.26):**
- `github.com/gin-gonic/gin v1.11.0` - Web framework
- `github.com/mattn/go-sqlite3 v1.14.24` - SQLite driver
- `github.com/alexedwards/scs/v2 v2.8.0` - Session management
- `github.com/alexedwards/scs/sqlite3store` - SQLite session store
- `github.com/gorilla/csrf v1.7.2` - CSRF protection
- `golang.org/x/crypto v0.48.0` - Cryptography (bcrypt)
- `github.com/air-verse/air` - Hot reload for development

**Frontend Libraries (CDN):**
- **HTMX**: `https://unpkg.com/htmx.org@2.0.8` - Latest stable HTMX 2.x
- **Tailwind CSS**: `https://cdn.tailwindcss.com` - Latest version (development only)

**Future Libraries (to be added):**
- PDF Generation: `chromedp` or `wkhtmltopdf` (Phase 17)
- Excel Export: `github.com/xuri/excelize` (Phase 17)

## Technology Stack

- **Backend**: Go 1.26, Gin v1.11.0 web framework
- **Frontend**: HTMX 2.0.8, Tailwind CSS
- **Database**: SQLite with go-sqlite3 v1.14.24
- **Session Management**: alexedwards/scs/v2 v2.8.0 with sqlite3store
- **CSRF Protection**: gorilla/csrf v1.7.2
- **Password Hashing**: golang.org/x/crypto v0.48.0 (bcrypt)
- **Hot Reload**: Air (github.com/air-verse/air) - development
- **PDF Generation**: chromedp or wkhtmltopdf (Phase 17)
- **Excel Export**: excelize v2.10.0 (Phase 17)

## Implementation Phases

### Phase 1: Project Scaffolding & Dev Environment
**Complexity**: Small (S)
**Dependencies**: None
**Description**: Initialize Go project structure, configure Gin framework, set up SQLite, integrate HTMX and Tailwind CSS, create hot-reload development environment with Air, establish directory structure and Makefile for build automation.

### Phase 2: Database Schema & Migrations
**Complexity**: Medium (M)
**Dependencies**: Phase 1
**Description**: Design and implement complete SQLite schema for all 12 tables (users, projects, products, addresses, DC templates, delivery challans, line items, serial numbers), create migration system, establish indexes and foreign key constraints, prepare seed data for testing.

### Phase 3: User Authentication
**Complexity**: Medium (M)
**Dependencies**: Phase 1, Phase 2
**Description**: Implement username/password authentication with bcrypt hashing, server-side session management using alexedwards/scs with SQLite-backed session store, CSRF protection via gorilla/csrf, auth middleware for route protection, login/logout handlers, session timeout handling. Simple authentication without role-based access control.

### Phase 4: Shared UI Layout & Navigation Shell
**Complexity**: Small (S)
**Dependencies**: Phase 3
**Description**: Create base template with responsive sidebar navigation, top bar with user info, HTMX configuration for smooth page transitions, Tailwind theme configuration, toast notification component, breadcrumb system, template inheritance structure.

### Phase 5: Project CRUD
**Complexity**: Large (L)
**Dependencies**: Phase 4
**Description**: Build complete project management interface - list view (card grid), create/edit forms with all fields (PO number, dates, billing info, company signature upload), detail view with tabs, delete with confirmation, server-side validation, HTMX interactions for smooth UX.

### Phase 6: Product Management
**Complexity**: Medium (M)
**Dependencies**: Phase 5
**Description**: Product catalog management within projects - add/edit/delete products with fields (name, SAC/HSN, unit, rate, GST %), inline editing on project detail page, product list view, validation for numeric fields, reusable product across multiple DCs.

### Phase 7: Bill To Address Management
**Complexity**: Medium (M)
**Dependencies**: Phase 5
**Description**: Dynamic address list configuration - define custom column names (Legal Name, GSTIN, Billing Address, etc.), add multiple bill-to addresses per project, JSON storage for flexible columns, validation for required fields, address selection UI with search/filter.

### Phase 8: Ship To Address Management
**Complexity**: Medium (M)
**Dependencies**: Phase 5
**Description**: Ship-to address configuration (separate from bill-to) - custom columns (Site Name, Site Address, Contact Person, etc.), add multiple ship-to addresses per project, same JSON storage pattern as bill-to, validation, address picker component.

### Phase 9: DC Template Management
**Complexity**: Large (L)
**Dependencies**: Phase 6
**Description**: Create DC templates within projects - name template, select products with default quantities, template list view, clone template feature, template assignment to transit/official DCs, edit/delete templates, validation ensuring at least one product per template.

### Phase 10: DC Number Generation Logic
**Complexity**: Small (S)
**Dependencies**: Phase 5
**Description**: Implement DC numbering system - auto-generate sequential numbers per project (001, 002, etc.), separate sequences for Transit and Official DCs, handle concurrent creation safely, reset sequence per project, store last number in projects table or separate sequence table.

### Phase 11: Transit DC Creation (Draft)
**Complexity**: Extra Large (XL)
**Dependencies**: Phase 7, Phase 8, Phase 9, Phase 10
**Description**: Create Transit DC in draft state - select ship-to address, choose template (optional), add/edit line items (product, quantity, rate, tax, amount), enter transit details (challan date, transporter, vehicle, e-way bill), auto-calculate totals, save as draft, validation for required fields.

### Phase 12: Official DC Creation (Draft)
**Complexity**: Extra Large (XL)
**Dependencies**: Phase 7, Phase 8, Phase 9, Phase 10
**Description**: Create Official DC in draft state - select both bill-to and ship-to addresses, choose template (optional), add/edit line items with full pricing, billing info fields, save as draft, similar UX to Transit DC but with more fields, validation.

### Phase 13: Serial Number Management & Validation
**Complexity**: Large (L)
**Dependencies**: Phase 11, Phase 12
**Description**: Serial number entry and validation system - add serials per line item, ensure quantity matches serial count, prevent duplicate serials across all DCs, bulk paste from Excel/CSV, serial number validation before DC issuance, display serial list in DC view.

### Phase 14: DC Lifecycle (Issue & Lock)
**Complexity**: Medium (M)
**Dependencies**: Phase 13
**Description**: DC state management - "Issue DC" action to transition from draft to issued (locked), validation before issuance (all serials entered, required fields filled), lock editing after issuance, show issued date and timestamp, prevent deletion of issued DCs.

### Phase 15: Transit DC View & Print Layout
**Complexity**: Large (L)
**Dependencies**: Phase 14
**Description**: Transit DC view page - display all DC details (header, line items, transit info, serials), print-friendly layout matching mockup, responsive design, show status badge (draft/issued), print button triggering browser print dialog, proper page breaks for multi-page DCs.

### Phase 16: Official DC View & Print Layout
**Complexity**: Large (L)
**Dependencies**: Phase 14
**Description**: Official DC view page - display bill-to, ship-to, line items with pricing, tax breakdown, totals, billing info, company signature, print-friendly layout, status badge, print button, professional invoice-style layout.

### Phase 17: PDF & Excel Export
**Complexity**: Medium (M)
**Dependencies**: Phase 15, Phase 16
**Description**: Export functionality - generate PDF from DC view (using chromedp or wkhtmltopdf), Excel export of DC data (line items, serials), download buttons on DC view page, proper formatting in exports, file naming convention (DC-number-date.pdf).

### Phase 18: Dashboard & Statistics
**Complexity**: Medium (M)
**Dependencies**: Phase 14
**Description**: Dashboard homepage - summary cards (total projects, total DCs, issued vs draft), recent DCs list, quick action buttons, statistics by project, charts (optional - DC creation over time), responsive grid layout, links to filtered DC lists.

### Phase 19: Global DC Listing & Filters
**Complexity**: Large (L)
**Dependencies**: Phase 14
**Description**: All DCs list page - table view with columns (DC number, type, project, date, status), filter by project/status/type/date range, search by DC number, pagination, sort by columns, export filtered results, responsive table design, HTMX-powered filtering without page reload.

### Phase 20: Serial Number Search
**Complexity**: Medium (M)
**Dependencies**: Phase 13
**Description**: Global serial number search - search form accepting single or multiple serials, display results showing which DC contains each serial, highlight duplicate serials (if any), link to DC detail view, search history (optional), export search results.

## Dependency Graph

```
Phase 1 (Scaffolding)
  └─> Phase 2 (Database)
        └─> Phase 3 (Auth)
              └─> Phase 4 (UI Layout)
                    └─> Phase 5 (Projects)
                          ├─> Phase 6 (Products)
                          ├─> Phase 7 (Bill To)
                          │     └─> Phase 11 (Transit DC) ──┐
                          │           └─> Phase 13 (Serials) ─> Phase 14 (Lifecycle)
                          │                                           ├─> Phase 15 (Transit View)
                          │                                           │     └─> Phase 17 (Export)
                          │                                           ├─> Phase 16 (Official View)
                          │                                           │     └─> Phase 17 (Export)
                          │                                           ├─> Phase 18 (Dashboard)
                          │                                           └─> Phase 19 (DC List)
                          ├─> Phase 8 (Ship To)
                          │     └─> Phase 11 (Transit DC) ──┘
                          │           └─> Phase 12 (Official DC) ─> Phase 13 (Serials)
                          └─> Phase 9 (Templates)
                          │     └─> Phase 11, Phase 12
                          └─> Phase 10 (DC Numbers)
                                └─> Phase 11, Phase 12

Phase 13 (Serials)
  └─> Phase 20 (Serial Search)
```

## Recommended Implementation Order

1. **Foundation (Weeks 1-2)**: Phases 1-4
2. **Project & Product Setup (Week 3)**: Phases 5-6
3. **Address & Template Management (Week 4)**: Phases 7-9
4. **DC Creation Core (Weeks 5-6)**: Phases 10-12
5. **Serial Management & Lifecycle (Week 7)**: Phases 13-14
6. **View & Export (Week 8)**: Phases 15-17
7. **Dashboard & Search (Week 9)**: Phases 18-20

## Critical Success Factors

1. **Data Integrity**: Serial number uniqueness validation is critical
2. **State Management**: Draft vs Issued state must be immutable once issued
3. **Concurrent Safety**: DC number generation must handle concurrent requests
4. **User Experience**: HTMX interactions should feel seamless, not janky
5. **Print Quality**: Print layouts must be production-ready (proper margins, page breaks)
6. **Validation**: Server-side validation on all forms before saving
7. **Backup Strategy**: SQLite database must be backed up regularly (not in app scope, but document for ops)

## Testing Strategy

- **Unit Tests**: Critical business logic (DC number generation, serial validation)
- **Integration Tests**: Database operations, authentication flow
- **Manual Testing**: UI workflows, print layouts, HTMX interactions
- **Test Data**: Comprehensive seed data covering edge cases

## Future Enhancements (Post-MVP)

- Role-based access control (admin vs user)
- Audit log for DC changes
- Email notifications for DC issuance
- Barcode/QR code generation for serials
- Mobile-responsive improvements
- Advanced reporting and analytics
- Multi-tenancy support
- API for external integrations

## Development Environment Setup

1. Install Go 1.26+
2. Install SQLite3
3. Install Air for hot reload: `go install github.com/air-verse/air@latest`
4. Clone repository and run `make setup`
5. Run `make dev` to start development server
6. Access at http://localhost:8080

## Production Deployment Checklist

- [ ] Environment variables for sensitive config
- [ ] HTTPS with valid SSL certificate
- [ ] Database backup automation
- [ ] Session secret from environment
- [ ] File upload size limits configured
- [ ] Error logging to file
- [ ] Health check endpoint for monitoring
- [ ] Graceful shutdown handling
- [ ] Static asset optimization (minify CSS/JS)
- [ ] Database connection pooling configured

## Estimated Timeline

- **Total Duration**: 9 weeks (assuming 1 developer, full-time)
- **MVP Launch**: End of Week 7 (through Phase 17)
- **Full Feature Set**: End of Week 9 (all 20 phases)

## Document Conventions

- All file paths are absolute starting from project root
- SQL examples use SQLite syntax
- Go code follows standard Go conventions (gofmt)
- Template files use Go html/template syntax
- HTMX attributes prefixed with `hx-`
- Tailwind classes use utility-first approach
