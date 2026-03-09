# Phase 12: Sidebar Navigation & UI Polish

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- All prior phases (this is the final polish phase)

## Overview

Final integration phase — add Split Shipment navigation to the sidebar, polish the UI across all new pages, ensure consistent styling, and perform end-to-end testing.

---

## Modified Files

| File | Changes |
|------|---------|
| `components/partials/sidebar.templ` | Add Split Shipment nav items under Delivery Challans |
| `components/pages/dashboard/dashboard.templ` | Add Transfer DC stats to dashboard |
| `internal/handlers/dashboard.go` | Query Transfer DC stats for dashboard |
| Various templ files | Consistent styling, responsive design fixes |

---

## Tests to Write First

- [ ] `TestSidebar_TransferDCLinks` — Verify nav items render for Split Shipments
- [ ] `TestSidebar_ActiveState_TransferDC` — Active highlighting works for transfer DC pages
- [ ] `TestDashboard_TransferDCStats` — Dashboard shows Transfer DC counts
- [ ] `TestEndToEnd_CreateTransferDC` — Full wizard flow (integration test)
- [ ] `TestEndToEnd_SplitTransferDC` — Create → Issue → Split flow
- [ ] `TestEndToEnd_UndoSplit` — Split → Undo → Re-split flow

---

## Implementation Steps

### 1. Update sidebar navigation — `components/partials/sidebar.templ`

Under the "Delivery Challans" expandable section, add new items:

```templ
// Existing items:
// - Shipments (list)
// - New Shipment
// - All DCs
// - Official DCs
// - Serial Search

// NEW items to add:
<li>
    <a href={ templ.SafeURL(fmt.Sprintf("/projects/%d/transfer-dcs", project.ID)) }
       class={ templ.KV("active", hasPrefix(currentPath, "/transfer-dcs")) }>
        <svg><!-- truck icon --></svg>
        <span>Split Shipments</span>
    </a>
</li>
<li>
    <a href={ templ.SafeURL(fmt.Sprintf("/projects/%d/transfer-dcs/new", project.ID)) }
       class={ templ.KV("active", currentPath == fmt.Sprintf("/projects/%d/transfer-dcs/new", project.ID)) }>
        <svg><!-- plus icon --></svg>
        <span>New Split Shipment</span>
    </a>
</li>
```

### 2. Update dashboard — `components/pages/dashboard/dashboard.templ`

Add Transfer DC stats card to the DC breakdown row:

```templ
// Existing cards: Transit DCs, Official DCs, Total DCs, Serial Numbers
// Add new card:
<div class="stat-card">
    <h4>Transfer DCs</h4>
    <div class="stat-value">{ stats.TransferDCTotal }</div>
    <div class="stat-detail">
        <span class="text-blue-500">{ stats.TransferIssued } issued</span>
        <span class="text-orange-500">{ stats.TransferSplitting } splitting</span>
        <span class="text-green-500">{ stats.TransferSplit } split</span>
        <span class="text-gray-400">{ stats.TransferDraft } draft</span>
    </div>
</div>
```

### 3. Update dashboard handler — `internal/handlers/dashboard.go`

Add Transfer DC queries:
```go
type DashboardStats struct {
    // ... existing fields ...
    TransferDCTotal    int
    TransferDraft      int
    TransferIssued     int
    TransferSplitting  int
    TransferSplit      int
}
```

### 4. Add Quick Actions — `components/pages/dashboard/dashboard.templ`

Add to Quick Actions section:
```templ
<a href={newTransferDCURL} class="quick-action-btn">
    New Split Shipment
</a>
```

### 5. UI Polish Checklist

- [ ] **Consistent status badges** across all pages:
  - `draft` → gray badge
  - `issued` → blue badge
  - `splitting` → orange badge (animated pulse optional)
  - `split` → green badge

- [ ] **Responsive design**:
  - Transfer DC detail page works on mobile (stack layout)
  - Quantity grid horizontally scrollable on small screens
  - Destination table responsive
  - Split wizard works on tablet/mobile

- [ ] **Loading states**:
  - HTMX indicators on form submissions
  - Disable submit buttons during processing

- [ ] **Confirmation dialogs**:
  - Issue Transfer DC: "This will lock the Transfer DC. Continue?"
  - Delete Transfer DC: "This will permanently delete the Transfer DC and all its data."
  - Undo Split: "This will delete the child shipment group and return destinations to the pool."

- [ ] **Breadcrumbs** on Transfer DC pages:
  - List: `Projects > ProjectName > Transfer DCs`
  - Detail: `Projects > ProjectName > Transfer DCs > STDC-001`
  - Wizard: `Projects > ProjectName > Transfer DCs > New`
  - Split: `Projects > ProjectName > Transfer DCs > STDC-001 > Split`

- [ ] **Empty states**:
  - No Transfer DCs: "No split shipments yet. Create one to get started."
  - No splits: "This Transfer DC has not been split yet. Issue it to begin splitting."
  - No un-split destinations: "All destinations have been split."

- [ ] **Tooltips/help text**:
  - Hub location field: "The intermediate location where the large truck delivers. Material will be split into smaller vehicles here."
  - Split button: "Create a new vehicle group from remaining un-split destinations"

### 6. CSS additions — `static/css/design-system.css`

```css
/* Transfer DC specific styles */
.badge-splitting {
    background-color: #f97316;  /* orange-500 */
    color: white;
}

.badge-split {
    background-color: #22c55e;  /* green-500 */
    color: white;
}

.badge-transfer {
    background-color: #8b5cf6;  /* violet-500 */
    color: white;
}

/* Split progress bar */
.split-progress-bar {
    height: 8px;
    border-radius: 4px;
    background: #e5e7eb;
}
.split-progress-bar .fill {
    height: 100%;
    border-radius: 4px;
    background: #22c55e;
    transition: width 0.3s ease;
}
```

### 7. Run full test suite and manual smoke tests

```bash
task test          # All automated tests
task templ:gen     # Regenerate templ
task build         # Verify build
```

Manual smoke test scenarios:
1. Create Transfer DC → verify all 5 steps → verify detail page
2. Issue Transfer DC → verify status change → verify Edit locked
3. Split: select 5 destinations → enter vehicle → enter serials → confirm
4. Verify child shipment group created (1 TDC + 5 ODCs)
5. Second split: select 5 more → verify progress bar
6. Undo last split → verify destinations return to pool
7. PDF export → verify all sections
8. Excel export → verify data
9. Reports → verify Transfer DC counts
10. Dashboard → verify stats
11. Sidebar → verify navigation and active states

---

## Acceptance Criteria

- [ ] Sidebar shows "Split Shipments" and "New Split Shipment" links under Delivery Challans
- [ ] Active state highlighting works for all Transfer DC pages
- [ ] Dashboard shows Transfer DC statistics with status breakdown
- [ ] Quick Actions includes "New Split Shipment" link
- [ ] All status badges are consistent across every page
- [ ] Responsive design works on mobile/tablet
- [ ] Breadcrumbs correct on all Transfer DC pages
- [ ] Empty states shown when appropriate
- [ ] Loading indicators on form submissions
- [ ] Confirmation dialogs for destructive actions
- [ ] All automated tests pass
- [ ] Manual smoke tests pass
- [ ] `task templ:gen` clean
- [ ] `go vet ./...` clean
- [ ] `go build ./...` clean
- [ ] No regressions in existing features
