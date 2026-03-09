# Plan: Split Shipment Feature

## Context
- **Goal**: Enable a two-tier shipment workflow where material is loaded onto a large truck (Transfer DC), transported to a hub location, then split into smaller vehicles — each creating its own shipment group (1 Transit DC + N Official DCs). This supports scenarios where ~25 delivery locations are served from a single hub via multiple smaller vehicles.
- **Affected Areas**: Database schema, DC numbering, DC generation service, new Transfer DC wizard (5 steps), new Split wizard (4 steps), Transfer DC detail page with split progress, PDF/Excel export, reports, sidebar navigation, DC listing filters
- **Dependencies**: Existing shipment group infrastructure, DC numbering service, address system, template system
- **Created**: 2026-03-09
- **Last Updated**: 2026-03-09

## Interview Summary (Requirements)

### Transfer DC Creation
- **New DC type**: `transfer` with its own numbering sequence (STDC prefix, e.g., `PRJ-STDC-2526-001`)
- **5-step wizard** (mirrors existing shipment wizard):
  - Step 1: Template + Hub location (existing ship-to address) + Transporter/Vehicle (large truck) + Tax config
  - Step 2: Select all ~25 final destinations (ship-to) + bill-to + bill-from + dispatch-from addresses
  - Step 3: Quantity grid (product x destination)
  - Step 4: Bulk serial entry (all serials per product, NOT per-destination — destination assignment happens at split time)
  - Step 5: Review & submit
- **Lifecycle**: `draft` → `issued` → `splitting` → `split`
- **Editable** in draft status (full edit like current shipments)

### Split Operation
- **Partial splits allowed**: Not all destinations need to be split at once
- **Multi-step split wizard** (per vehicle group):
  - Step 1: Select destinations from remaining un-split pool
  - Step 2: Enter transporter/vehicle details for the small vehicle
  - Step 3: Enter/scan serial numbers per product (validated against parent Transfer DC's master serial list)
  - Step 4: Review & confirm
- **Strict validation**: Serials MUST exist in parent Transfer DC; quantities MUST exactly match planned amounts per destination
- **Pricing**: Child Transit DCs inherit rates/tax from parent Transfer DC
- **Undo**: Deleting a draft child shipment group returns destinations + serials to un-split pool. Once child TDC is issued, deletion is blocked.

### Transfer DC Detail Page
- Full detail: DC number, status, transporter, hub location, all destinations with quantities, all serials
- Split progress panel: which destinations are split vs pending, links to child shipment groups
- "Create Split" action button (available when status is `issued` or `splitting`)

### Export & Reporting
- Full PDF/Excel export and browser print view for Transfer DCs
- New "Transfer DCs" section in reports and DC listing filter
- Transfer DC report shows split progress and child group hierarchy

### Navigation
- Under existing "Delivery Challans" sidebar section: add "Split Shipments" and "New Split Shipment" links

## Phases
1. [Database Schema & Migrations](./phase-1-database-schema.md) — ⬜ Not Started
2. [DC Numbering & Constants](./phase-2-dc-numbering.md) — ⬜ Not Started
3. [Transfer DC Data Access Layer](./phase-3-data-access.md) — ⬜ Not Started
4. [Transfer DC Creation Wizard (Handler + Templates)](./phase-4-transfer-wizard.md) — ⬜ Not Started
5. [Transfer DC Detail Page & Lifecycle](./phase-5-transfer-detail.md) — ⬜ Not Started
6. [Split Operation Data Layer & Service](./phase-6-split-service.md) — ⬜ Not Started
7. [Split Wizard (Handler + Templates)](./phase-7-split-wizard.md) — ⬜ Not Started
8. [Transfer DC Edit Wizard](./phase-8-transfer-edit.md) — ⬜ Not Started
9. [Split Undo & Child Group Deletion](./phase-9-split-undo.md) — ⬜ Not Started
10. [PDF/Excel Export & Print Views](./phase-10-export.md) — ⬜ Not Started
11. [Reports & DC Listing Integration](./phase-11-reports.md) — ⬜ Not Started
12. [Sidebar Navigation & UI Polish](./phase-12-navigation-polish.md) — ⬜ Not Started

## Verification Checklist
- [ ] All tests pass (`task test`)
- [ ] `templ generate` clean (`task templ:gen`)
- [ ] `go vet ./...` clean
- [ ] `go build ./...` clean
- [ ] Manual smoke test: create Transfer DC → issue → split into 2 groups → verify child TDC/ODCs
- [ ] Manual smoke test: partial split → verify remaining destinations shown
- [ ] Manual smoke test: delete draft child group → verify destinations return to pool
- [ ] Manual smoke test: PDF/Excel export for Transfer DC
- [ ] No regressions in existing shipment/DC features
- [ ] Existing tests still pass
