# Execution Prompts for All Phases

Copy-paste these prompts to execute each phase with Claude Code. Each prompt creates a detailed task list with dependencies, uses multiple specialized agents, runs all tests, and updates the phase document with a summary.

---

## Phase 1: Project Scaffolding & Dev Environment

```
@plans/phase-01-project-scaffolding.md implement phase 1 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Set up Go module, install all dependencies (Gin, SQLite, scs, bcrypt, HTMX, Tailwind CSS)
2. Create directory structure (cmd/, internal/, templates/, static/, migrations/, data/)
3. Create initial main.go with Gin server, config loader, database init
4. Set up Air for hot reload with .air.toml config
5. Create Makefile with all targets (dev, build, run, test, clean, fmt, lint)
6. Create .gitignore for Go, data/, tmp/, bin/
7. Set up Tailwind CSS with CDN or local build
8. Create base HTML template with HTMX script tags
9. Create health check handler and verify server starts
10. Run `go build ./...` to verify compilation, test `make dev` works
Make sure you check if everything is done and do all the tests and update the phase 01 document appropriately with summary.
```

---

## Phase 2: Database Schema & Migrations

```
@plans/phase-02-database-schema.md implement phase 2 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create migration runner in internal/database/migrate.go
2. Create all 8 migration pairs (up/down SQL files) for: users, projects, products, dc_templates + dc_template_products, address_list_configs + addresses, delivery_challans + dc_transit_details, dc_line_items, serial_numbers
3. Add proper foreign keys, indexes, CHECK constraints, UNIQUE constraints, CASCADE deletes
4. Create seed_data.sql with test users, projects, products, addresses, templates, DCs, line items, serial numbers
5. Update main.go to run migrations on startup
6. Update Makefile with migrate and seed targets
7. Run migrations and verify all tables created with `sqlite3 data/dc_management.db ".tables"`
8. Run seed data and verify with sample queries
9. Test foreign key constraints, unique constraints, and cascade deletes
Make sure you check if everything is done and do all the tests and update the phase 02 document appropriately with summary. Use chrome extension for browser checks.
```

---

## Phase 3: User Authentication

```
@plans/phase-03-authentication.md implement phase 3 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create user model and repository in internal/models/ and internal/repository/
2. Set up scs/v2 session manager with SQLite store
3. Implement bcrypt password hashing utilities
4. Create login page template with Tailwind CSS styling
5. Create auth handlers (GET /login, POST /login, POST /logout)
6. Implement auth middleware to protect routes
7. Add CSRF protection middleware
8. Create session-based user context for templates
9. Test login with seed data users, verify session persistence
10. Test protected routes redirect to login, test logout clears session
11. Test CSRF token validation on forms
Make sure you check if everything is done and do all the tests and update the phase 03 document appropriately with summary. Use chrome extension for browser checks (login page rendering, form submission, redirect behavior).
```

---

## Phase 4: Shared UI Layout & Navigation Shell

```
@plans/phase-04-ui-layout.md implement phase 4 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create base layout template with Tailwind CSS (sidebar + main content area)
2. Build responsive sidebar with navigation links (Dashboard, Projects, DCs, Serial Search)
3. Create top bar with user info and logout button
4. Set up HTMX configuration (hx-boost, progress indicators, error handling)
5. Implement toast notification system (success/error/warning)
6. Create reusable partial templates (pagination, empty states, loading spinners)
7. Add Alpine.js for sidebar toggle and dropdown interactions
8. Create consistent form styling components
9. Test responsive layout at different screen sizes
10. Verify HTMX navigation works without full page reloads
Make sure you check if everything is done and do all the tests and update the phase 04 document appropriately with summary. Use chrome extension for browser checks (layout rendering, responsive behavior, sidebar toggle, HTMX navigation).
```

---

## Phase 5: Project CRUD

```
@plans/phase-05-project-crud.md implement phase 5 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create project model and repository with all CRUD operations
2. Build project list page with card/table view, search, and pagination
3. Create project creation form with all fields (name, description, DC prefix, tender ref, PO details, bill-from address, GSTIN)
4. Implement project edit form with pre-populated fields
5. Build project detail view with tabbed interface (Overview, Products, Templates, Addresses, DCs)
6. Implement company signature image upload and storage
7. Create project delete with confirmation dialog
8. Add HTMX-powered search and filtering
9. Wire up all routes (GET /projects, GET /projects/new, POST /projects, GET /projects/:id, etc.)
10. Test all CRUD operations, verify form validation, test image upload
Make sure you check if everything is done and do all the tests and update the phase 05 document appropriately with summary. Use chrome extension for browser checks (project list, create form, edit form, detail view with tabs, image upload).
```

---

## Phase 6: Product Management

```
@plans/phase-06-product-management.md implement phase 6 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create product model and repository with CRUD operations
2. Build product list view within project detail page (Products tab)
3. Create product add/edit form (item name, description, HSN code, UOM, brand/model, price, GST%)
4. Implement inline editing or modal-based product editing
5. Add product delete with dependency checking (prevent delete if used in DCs)
6. Implement bulk product import via CSV if specified in plan
7. Wire up routes nested under projects (GET /projects/:id/products, POST, PUT, DELETE)
8. Test CRUD operations, GST calculation preview, validation
Make sure you check if everything is done and do all the tests and update the phase 06 document appropriately with summary. Use chrome extension for browser checks (product list within project, add/edit forms, delete confirmation).
```

---

## Phase 7: Bill-To Address Management

```
@plans/phase-07-bill-to-addresses.md implement phase 7 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create address list config model and repository
2. Create address model and repository with JSON data handling
3. Build bill-to address config UI (define dynamic columns with name/required fields)
4. Implement address list view with dynamic columns rendered from JSON schema
5. Create address add/edit form that dynamically generates fields from column config
6. Implement CSV/Excel file upload for bulk address import
7. Add address search and filtering
8. Wire up routes under projects (GET /projects/:id/bill-to-addresses, POST config, CRUD addresses)
9. Test dynamic column definition, address CRUD, CSV import, JSON storage/retrieval
Make sure you check if everything is done and do all the tests and update the phase 07 document appropriately with summary. Use chrome extension for browser checks (dynamic column config, address list, add/edit forms, CSV upload).
```

---

## Phase 8: Ship-To Address Management

```
@plans/phase-08-ship-to-addresses.md implement phase 8 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Reuse address infrastructure from Phase 7 for ship-to type
2. Build ship-to address config UI with dynamic column definitions
3. Implement ship-to address list with table and card views
4. Create address add/edit form with dynamic fields from config
5. Implement CSV/Excel upload for bulk ship-to address import
6. Add search, filtering, and pagination
7. Wire up routes (GET /projects/:id/ship-to-addresses, POST config, CRUD addresses)
8. Test address CRUD, view switching, search, CSV import
Make sure you check if everything is done and do all the tests and update the phase 08 document appropriately with summary. Use chrome extension for browser checks (ship-to address list, table/card views, search, forms).
```

---

## Phase 9: DC Template Management

```
@plans/phase-09-dc-template-management.md implement phase 9 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create DC template model and repository
2. Create dc_template_products junction table operations
3. Build template list view within project detail page (Templates tab)
4. Create template creation form with name, purpose, and product selection
5. Implement multi-select product picker with default quantities
6. Build template edit and delete functionality
7. Wire up routes (GET /projects/:id/templates, POST, PUT, DELETE)
8. Test template CRUD, product association, default quantities
Make sure you check if everything is done and do all the tests and update the phase 09 document appropriately with summary. Use chrome extension for browser checks (template list, create/edit forms, product selection).
```

---

## Phase 10: DC Number Generation Logic

```
@plans/phase-10-dc-numbering.md implement phase 10 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Implement Indian financial year calculation (April-March)
2. Create DC number format: {prefix}-{T/O}DC-{FY}-{sequential} (e.g., SCP-TDC-2425-001)
3. Implement thread-safe sequential number generation with database transactions
4. Create number generation service in internal/services/
5. Handle financial year rollover (reset counter at April 1st)
6. Update project model to track last_transit_dc_number and last_official_dc_number per FY
7. Write unit tests for number generation, FY calculation, edge cases
8. Test concurrent number generation doesn't produce duplicates
Make sure you check if everything is done and do all the tests and update the phase 10 document appropriately with summary.
```

---

## Phase 11: Transit DC Creation (Draft)

```
@plans/phase-11-transit-dc-creation.md implement phase 11 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create delivery challan model and repository
2. Create dc_line_items and dc_transit_details repositories
3. Build Transit DC creation form with template selection (auto-populates products)
4. Implement ship-to address selector (no bill-to for transit)
5. Build line items editor with add/remove products, quantity, pricing
6. Implement auto-calculation of taxable amount, tax, and totals
7. Add transit-specific fields (transporter name, vehicle number, e-way bill, notes)
8. Implement serial number entry per line item
9. Wire up routes (GET /projects/:id/dcs/new?type=transit, POST)
10. Test DC creation with template, manual product addition, calculations, serial numbers
Make sure you check if everything is done and do all the tests and update the phase 11 document appropriately with summary. Use chrome extension for browser checks (DC creation form, template selection, line items editor, calculations).
```

---

## Phase 12: Official DC Creation (Draft)

```
@plans/phase-12-official-dc-creation.md implement phase 12 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Build Official DC creation form (simpler than transit - no pricing/tax)
2. Implement bill-to AND ship-to address selectors
3. Build line items editor with products and quantities (no pricing columns)
4. Add serial number entry per line item
5. Implement template selection for auto-populating products
6. Wire up routes (GET /projects/:id/dcs/new?type=official, POST)
7. Test DC creation, address selection, line items without pricing
Make sure you check if everything is done and do all the tests and update the phase 12 document appropriately with summary. Use chrome extension for browser checks (official DC form, address selectors, line items editor).
```

---

## Phase 13: Serial Number Management & Validation

```
@plans/phase-13-serial-number-management.md implement phase 13 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create serial number repository with validation logic
2. Implement project-scoped uniqueness validation (no duplicate serial within a project)
3. Build serial number entry UI with real-time duplicate checking via HTMX
4. Add bulk serial number paste/import functionality
5. Implement serial number count validation (must match line item quantity for issued DCs)
6. Create serial number edit/delete on draft DCs
7. Wire up HTMX validation endpoints (POST /api/serial-numbers/validate)
8. Test uniqueness constraints, bulk import, validation feedback
Make sure you check if everything is done and do all the tests and update the phase 13 document appropriately with summary. Use chrome extension for browser checks (serial number entry, duplicate detection, validation messages).
```

---

## Phase 14: DC Lifecycle (Issue & Lock)

```
@plans/phase-14-dc-lifecycle.md implement phase 14 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Implement DC status transitions (draft -> issued)
2. Add "Issue DC" action with validation (all required fields filled, serial numbers match quantities)
3. Implement field locking on issued DCs (prevent editing)
4. Create DC edit functionality for draft DCs
5. Implement DC delete for draft DCs only (with confirmation)
6. Add issued_at timestamp and issued_by user tracking
7. Update DC list views to show status badges
8. Wire up routes (POST /dcs/:id/issue, PUT /dcs/:id, DELETE /dcs/:id)
9. Test issue flow, validation errors, field locking, edit/delete restrictions
Make sure you check if everything is done and do all the tests and update the phase 14 document appropriately with summary. Use chrome extension for browser checks (issue button, validation errors, locked fields, status badges).
```

---

## Phase 15: Transit DC View & Print Layout

```
@plans/phase-15-transit-dc-view.md implement phase 15 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create Transit DC detail view page with all information
2. Build print-ready layout with company header, logo, and address
3. Display ship-to address, transporter details, vehicle number, e-way bill
4. Create product table with HSN, quantity, rate, tax breakdown, totals
5. Add serial numbers section per line item
6. Implement grand total with tax summary (CGST/SGST or IGST)
7. Add signature blocks and terms/conditions
8. Create print CSS for clean A4 printing
9. Add "Print" button triggering browser print dialog
10. Test print preview layout, page breaks, alignment
Make sure you check if everything is done and do all the tests and update the phase 15 document appropriately with summary. Use chrome extension for browser checks (DC view layout, print preview, table formatting, totals).
```

---

## Phase 16: Official DC View & Print Layout

```
@plans/phase-16-official-dc-view.md implement phase 16 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create Official DC detail view page
2. Build print-ready layout with company header and logo
3. Display bill-to and ship-to addresses side by side
4. Create product table with quantities (no pricing columns)
5. Add serial numbers section per line item
6. Implement dual signature blocks (sender and receiver)
7. Add terms/conditions section
8. Create print CSS for A4 layout
9. Add "Print" button
10. Test print preview, layout differences from transit DC
Make sure you check if everything is done and do all the tests and update the phase 16 document appropriately with summary. Use chrome extension for browser checks (official DC view, print layout, dual signatures, no pricing columns).
```

---

## Phase 17: PDF & Excel Export

```
@plans/phase-17-pdf-excel-export.md implement phase 17 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Set up chromedp for HTML-to-PDF conversion
2. Implement PDF export for Transit DC (reuse print layout template)
3. Implement PDF export for Official DC
4. Set up excelize library for Excel export
5. Create Excel export for DC line items with formatting
6. Create Excel export for DC summary/listing
7. Add download buttons to DC view pages
8. Wire up routes (GET /dcs/:id/export/pdf, GET /dcs/:id/export/excel)
9. Test PDF output matches print layout, Excel has correct data and formatting
Make sure you check if everything is done and do all the tests and update the phase 17 document appropriately with summary. Use chrome extension for browser checks (download buttons, verify PDF opens correctly).
```

---

## Phase 18: Dashboard & Statistics

```
@plans/phase-18-dashboard.md implement phase 18 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create dashboard handler with aggregated statistics queries
2. Build summary cards (total projects, total DCs, draft DCs, issued DCs)
3. Add project-wise DC count breakdown
4. Create recent activity feed (last 10 DCs created/issued)
5. Build quick action buttons (New Project, New Transit DC, New Official DC)
6. Implement dashboard as the authenticated home page (GET /)
7. Add HTMX-powered refresh for statistics
8. Test dashboard data accuracy with seed data
Make sure you check if everything is done and do all the tests and update the phase 18 document appropriately with summary. Use chrome extension for browser checks (dashboard layout, statistics cards, recent activity, quick actions).
```

---

## Phase 19: Global DC Listing & Filters

```
@plans/phase-19-global-dc-listing.md implement phase 19 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create global DC listing handler with cross-project query
2. Build DC list page with table view (DC number, type, status, project, date, addresses)
3. Implement advanced filters (by project, DC type, status, date range)
4. Add sorting by multiple columns (date, DC number, status)
5. Implement pagination with HTMX
6. Add search by DC number
7. Create filter persistence (remember last applied filters)
8. Wire up routes (GET /delivery-challans with query params)
9. Test filtering, sorting, pagination, search across multiple projects
Make sure you check if everything is done and do all the tests and update the phase 19 document appropriately with summary. Use chrome extension for browser checks (DC listing, filters, sorting, pagination, search).
```

---

## Phase 20: Serial Number Search

```
@plans/phase-20-serial-search.md implement phase 20 by creating a detailed tasklist with dependencies and multiple specialized agents/skills. Steps:
1. Create serial search handler with cross-project query
2. Build search page with text input for serial number(s)
3. Implement single and bulk serial number search (paste multiple, comma/newline separated)
4. Display results showing: serial number, product name, DC number, DC type, project, status
5. Add links to parent DC from search results
6. Handle not-found serials with clear messaging
7. Implement HTMX-powered live search
8. Wire up routes (GET /serial-search, POST /serial-search)
9. Test single search, bulk search, not-found handling, cross-project results
Make sure you check if everything is done and do all the tests and update the phase 20 document appropriately with summary. Use chrome extension for browser checks (search page, results display, bulk search, links to DCs).
```

---

## Full Sequential Execution (All Phases)

To execute all phases in order, run each prompt above sequentially, waiting for each phase to complete before starting the next. Each phase depends on the previous ones being fully implemented and tested.

**Dependency Chain:**
```
Phase 1 (Scaffolding)
  -> Phase 2 (Database)
    -> Phase 3 (Auth)
      -> Phase 4 (UI Layout)
        -> Phase 5 (Projects)
          -> Phase 6 (Products)
            -> Phase 7 (Bill-To Addresses)
            -> Phase 8 (Ship-To Addresses)
            -> Phase 9 (DC Templates)
              -> Phase 10 (DC Numbering)
                -> Phase 11 (Transit DC Creation)
                -> Phase 12 (Official DC Creation)
                  -> Phase 13 (Serial Numbers)
                    -> Phase 14 (DC Lifecycle)
                      -> Phase 15 (Transit DC View)
                      -> Phase 16 (Official DC View)
                        -> Phase 17 (PDF/Excel Export)
                          -> Phase 18 (Dashboard)
                          -> Phase 19 (Global DC Listing)
                          -> Phase 20 (Serial Search)
```
