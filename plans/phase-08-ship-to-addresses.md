# Phase 8: Ship To Address Management

## Overview
This phase implements ship-to address management with the same flexible architecture as bill-to addresses but with separate configuration and data. The system provides two view modes (table and card/grid) as shown in mockup 08-ship-to-addresses.html, along with enhanced search, filtering, and pagination capabilities for managing potentially large address lists.

## Prerequisites
- Phase 1: Project Setup (complete)
- Phase 7: Bill To Address Management (provides reusable components and patterns)
- Database tables: address_list_configs, addresses (already created in Phase 7)
- Go libraries: encoding/csv, excelize (already added in Phase 7)

## Goals
- Implement ship-to address management using existing infrastructure
- Support different column structure from bill-to addresses
- Provide table view and card/grid view modes
- Implement robust search across all fields
- Support advanced filtering by column values
- Enable pagination for large datasets
- Reuse address management components from Phase 7
- Match mockup 08-ship-to-addresses.html UI/UX
- Optimize for handling 1000+ addresses efficiently

## Detailed Implementation Steps

### 1. Database Setup (Reuse Existing)
1.1. Verify address_list_configs table supports ship_to type
   - Already created in Phase 7
   - list_type CHECK constraint includes 'ship_to'

1.2. No new migrations needed
   - Reuse existing tables
   - Data isolation through list_type and config_id

1.3. Create initial ship-to configuration for projects
   - Default columns based on mockup: District, SRO, Location, Location ID, Mandal/ULB, Secretariat Name, Secretariat Code

### 2. Backend Implementation (Extend Existing)
2.1. Create handlers/ship_to_addresses.go
   - Similar structure to bill_to_addresses.go
   - Different list_type parameter
   - Shared validation and parsing logic

2.2. Reuse existing utilities
   - utils/csv_parser.go (no changes needed)
   - utils/excel_parser.go (no changes needed)
   - utils/address_validator.go (no changes needed)

2.3. Add view mode handling
   - Support table and grid view modes
   - Store user preference (session/cookie)
   - Return appropriate template based on mode

2.4. Implement advanced search
   - Search across all JSON fields
   - Support partial matching
   - Case-insensitive search
   - Highlight matching terms (optional)

2.5. Implement column-based filtering
   - Filter by exact match on specific columns
   - Support multiple filters simultaneously
   - Clear filter options

### 3. API Routes and Handlers
3.1. Define routes in routes/routes.go
   - Mirror bill-to routes structure
   - Add view mode parameter support

3.2. Create handler methods
   - ShowShipToPage(c *gin.Context) - GET /projects/:id/ship-to
   - GetShipToConfig(c *gin.Context) - GET /projects/:id/ship-to/config
   - UpdateShipToConfig(c *gin.Context) - POST /projects/:id/ship-to/config
   - UploadShipToAddresses(c *gin.Context) - POST /projects/:id/ship-to/upload
   - ListShipToAddresses(c *gin.Context) - GET /projects/:id/ship-to/addresses
   - DeleteShipToAddress(c *gin.Context) - DELETE /projects/:id/ship-to/:aid
   - ToggleViewMode(c *gin.Context) - POST /projects/:id/ship-to/view-mode

3.3. Search and filter handling
   - Accept search query parameter
   - Accept multiple filter parameters
   - Build dynamic SQL/JSON queries
   - Return filtered, paginated results

3.4. Error handling
   - Same patterns as bill-to addresses
   - Consistent error messages

### 4. Frontend Templates
4.1. Create templates/addresses/ship-to.html
   - Page header with view mode toggle
   - Column configuration section
   - Upload interface
   - Search bar with advanced options
   - Filter chips/tags for active filters
   - View container (switches between table/grid)

4.2. Reuse templates/addresses/config-form.html
   - Same component, different list_type
   - Pre-populate with ship-to defaults

4.3. Reuse templates/addresses/upload-form.html
   - Same component, different endpoint

4.4. Create templates/addresses/ship-to-table.html
   - Similar to bill-to table
   - Dynamic columns from config
   - Responsive design

4.5. Create templates/addresses/ship-to-grid.html
   - Card-based layout
   - Display key fields per card
   - Actions menu on each card
   - Responsive grid (2-4 columns based on screen)

4.6. Create templates/addresses/ship-to-card.html
   - Individual address card component
   - Prominent display of key fields
   - Expandable details (optional)
   - Delete action button

4.7. Create templates/addresses/search-bar.html
   - Search input with icon
   - Filter dropdown/menu
   - Active filters display
   - Clear filters button

4.8. Create templates/addresses/filter-panel.html
   - Slide-out panel for advanced filters
   - Checkbox/select per column
   - Apply/reset buttons

### 5. View Mode Implementation
5.1. Table View
   - Horizontal scrolling for many columns
   - Fixed header on scroll
   - Sticky first column (optional)
   - Row hover effects
   - Bulk selection (future)

5.2. Card/Grid View
   - Responsive grid layout (CSS Grid or Flexbox)
   - Card shows 4-6 key fields
   - "Show more" to expand
   - Compact delete action
   - Better for mobile devices

5.3. View mode toggle
   - Button/toggle in header
   - Icon indicators (table/grid icons)
   - Store preference in session
   - Smooth transition between views

### 6. Search and Filter Implementation
6.1. Search functionality
   - Search box in header
   - Debounced input (300ms delay)
   - HTMX request to filter addresses
   - Update view without page reload
   - Show search term and result count

6.2. Column-based filters
   - Filter panel or dropdowns
   - Multi-select for enum columns
   - Text input for text columns
   - Range inputs for number columns (future)
   - Apply filters button

6.3. Active filter display
   - Filter chips/badges below search bar
   - Show: "District: Krishna × | SRO: Vijayawada ×"
   - Click × to remove individual filter
   - "Clear all" option

6.4. Combined search and filter
   - Search within filtered results
   - Filters narrow down, search highlights
   - Clear indication of applied filters

### 7. Pagination Enhancement
7.1. Smart pagination
   - Show page numbers: « 1 2 3 ... 10 »
   - Jump to page input
   - Per-page selector (25, 50, 100)
   - Show range: "1-50 of 1,234"

7.2. HTMX pagination
   - Page links use hx-get
   - Replace table/grid content only
   - Preserve search and filter state
   - Update URL parameters

7.3. Infinite scroll (optional)
   - Alternative to pagination
   - Load more on scroll
   - Good for card view

### 8. HTMX Integration
8.1. View mode toggle
   - Click table/grid icon
   - POST /projects/:id/ship-to/view-mode with mode parameter
   - Swap entire view container
   - Maintain search/filter state

8.2. Search interaction
   - Input: hx-get="/projects/:id/ship-to/addresses"
   - Trigger: keyup changed delay:300ms
   - Target: view container
   - Include pagination reset

8.3. Filter interaction
   - Filter panel submit
   - POST filter values
   - Update view and active filters
   - Update result count

8.4. Delete in grid view
   - Click delete on card
   - Confirmation modal
   - DELETE request
   - Remove card with animation

### 9. Performance Optimization
9.1. Database query optimization
   - Use indexes for JSON queries
   - Limit result sets
   - Optimize search queries

9.2. Frontend optimization
   - Lazy load cards in grid view
   - Virtual scrolling for large lists (optional)
   - Minimize DOM updates

9.3. Caching
   - Cache column configuration
   - Cache filter options
   - Session-based result caching

### 10. Reusable Components
10.1. Identify shared code with bill-to
   - Abstract common handler logic
   - Create base address handler
   - Shared validation functions

10.2. Create utils/address_manager.go
   - ManageAddresses(listType string) functions
   - Reduce code duplication
   - Consistent behavior

10.3. Shared templates
   - config-form.html
   - upload-form.html
   - upload-result.html
   - Use partials with parameters

## Files to Create/Modify

### New Files
```
/handlers/ship_to_addresses.go
/templates/addresses/ship-to.html
/templates/addresses/ship-to-table.html
/templates/addresses/ship-to-grid.html
/templates/addresses/ship-to-card.html
/templates/addresses/search-bar.html
/templates/addresses/filter-panel.html
/templates/addresses/view-toggle.html
/static/js/address-search.js
/static/js/view-toggle.js
/utils/address_manager.go (shared utilities)
```

### Modified Files
```
/routes/routes.go (add ship-to routes)
/templates/projects/detail.html (add link to ship-to addresses)
/handlers/bill_to_addresses.go (refactor to use shared utilities)
/templates/addresses/config-form.html (make reusable with list_type param)
/templates/addresses/upload-form.html (make reusable with endpoint param)
```

### No New Migrations
- Reuse existing address_list_configs and addresses tables from Phase 7

## API Routes / Endpoints

### Ship To Address Management Routes
```
GET    /projects/:id/ship-to                - Show ship-to addresses page
GET    /projects/:id/ship-to/config         - Get column configuration
POST   /projects/:id/ship-to/config         - Update column configuration
POST   /projects/:id/ship-to/upload         - Upload addresses (CSV/Excel)
GET    /projects/:id/ship-to/addresses      - List addresses (with search/filter/pagination)
DELETE /projects/:id/ship-to/:aid           - Delete single address
DELETE /projects/:id/ship-to/all            - Delete all addresses
POST   /projects/:id/ship-to/view-mode      - Toggle view mode (table/grid)
GET    /projects/:id/ship-to/export         - Export addresses to CSV
```

### Request/Response Examples

#### POST /projects/:id/ship-to/config
Request Body:
```json
{
  "columns": [
    {"name": "District", "type": "text", "required": true},
    {"name": "SRO", "type": "text", "required": false},
    {"name": "Location", "type": "text", "required": true},
    {"name": "Location ID", "type": "text", "required": true},
    {"name": "Mandal/ULB", "type": "text", "required": false},
    {"name": "Secretariat Name", "type": "text", "required": true},
    {"name": "Secretariat Code", "type": "text", "required": true}
  ]
}
```

Response:
```json
{
  "success": true,
  "message": "Ship-to configuration updated",
  "config_id": 6
}
```

#### GET /projects/:id/ship-to/addresses
Query Parameters:
```
view=grid
page=2
per_page=50
search=vijayawada
filter[District]=Krishna
filter[SRO]=Vijayawada
```

Response:
```json
{
  "success": true,
  "data": {
    "addresses": [
      {
        "id": 201,
        "data": {
          "District": "Krishna",
          "SRO": "Vijayawada",
          "Location": "Governorpet",
          "Location ID": "LOC-001",
          "Mandal/ULB": "Vijayawada Municipal Corporation",
          "Secretariat Name": "Governorpet Secretariat",
          "Secretariat Code": "VJA-001"
        },
        "created_at": "2026-02-12T09:15:00Z"
      }
    ],
    "pagination": {
      "current_page": 2,
      "per_page": 50,
      "total_records": 1234,
      "total_pages": 25
    },
    "active_filters": {
      "search": "vijayawada",
      "District": "Krishna",
      "SRO": "Vijayawada"
    },
    "view_mode": "grid"
  }
}
```

#### POST /projects/:id/ship-to/view-mode
Request Body:
```json
{
  "mode": "grid"
}
```

Response:
```json
{
  "success": true,
  "mode": "grid"
}
```

## Database Queries

### Key Queries (Extending Phase 7)

#### Get ship-to configuration
```sql
SELECT id, column_definitions, updated_at
FROM address_list_configs
WHERE project_id = ? AND list_type = 'ship_to';
```

#### Search addresses across all fields (SQLite JSON)
```sql
SELECT id, data, created_at
FROM addresses
WHERE config_id = ?
  AND deleted_at IS NULL
  AND (
    json_extract(data, '$.District') LIKE '%' || ? || '%'
    OR json_extract(data, '$.SRO') LIKE '%' || ? || '%'
    OR json_extract(data, '$.Location') LIKE '%' || ? || '%'
    OR json_extract(data, '$.Secretariat Name') LIKE '%' || ? || '%'
  )
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
```

#### Filter by specific column values
```sql
SELECT id, data, created_at
FROM addresses
WHERE config_id = ?
  AND deleted_at IS NULL
  AND json_extract(data, '$.District') = ?
  AND json_extract(data, '$.SRO') = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
```

#### Combined search and filter
```sql
SELECT id, data, created_at
FROM addresses
WHERE config_id = ?
  AND deleted_at IS NULL
  AND json_extract(data, '$.District') = ?  -- filter
  AND (  -- search within filtered
    json_extract(data, '$.Location') LIKE '%' || ? || '%'
    OR json_extract(data, '$.Secretariat Name') LIKE '%' || ? || '%'
  )
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
```

#### Get distinct values for filter dropdowns
```sql
SELECT DISTINCT json_extract(data, '$.District') as district_value
FROM addresses
WHERE config_id = ?
  AND deleted_at IS NULL
  AND json_extract(data, '$.District') IS NOT NULL
ORDER BY district_value;
```

#### Count filtered results
```sql
SELECT COUNT(*)
FROM addresses
WHERE config_id = ?
  AND deleted_at IS NULL
  AND json_extract(data, '$.District') = ?;
```

## UI Components

### Ship To Addresses Page Structure (Table View)
```
┌──────────────────────────────────────────────────────────────────┐
│ Project Name > Ship To Addresses            [≡ Table] [⊞ Grid]  │
├──────────────────────────────────────────────────────────────────┤
│ Column Configuration: [Edit Columns]                             │
│ Current: District, SRO, Location, Location ID, Mandal/ULB, ...   │
├──────────────────────────────────────────────────────────────────┤
│ Upload Addresses: [Choose File] ○ Replace ○ Append [Upload]     │
├──────────────────────────────────────────────────────────────────┤
│ 🔍 [Search addresses...________]  [Advanced Filters ▼]          │
│ Active: District: Krishna × | SRO: Vijayawada × | Clear all     │
├──────────────────────────────────────────────────────────────────┤
│ Showing 51-100 of 1,234 results                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ District│ SRO    │Location│Loc ID│Mandal/ULB│Secret.│  × │   │
│ ├────────────────────────────────────────────────────────────┤   │
│ │ Krishna │ Vjwada │Govpet  │LOC001│VMC       │VJA-001│  × │   │
│ │ Krishna │ Vjwada │Patamata│LOC002│VMC       │VJA-002│  × │   │
│ └────────────────────────────────────────────────────────────┘   │
│                « 1 2 [3] 4 5 ... 25 »     [100 per page ▼]      │
└──────────────────────────────────────────────────────────────────┘
```

### Ship To Addresses Page Structure (Grid View)
```
┌──────────────────────────────────────────────────────────────────┐
│ Project Name > Ship To Addresses            [≡ Table] [⊞ Grid]  │
├──────────────────────────────────────────────────────────────────┤
│ 🔍 [Search addresses...________]  [Advanced Filters ▼]          │
│ Active: District: Krishna × | Clear all                          │
├──────────────────────────────────────────────────────────────────┤
│ Showing 1-50 of 1,234 results                                    │
│ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐    │
│ │ Krishna         │ │ Krishna         │ │ Guntur          │    │
│ │ SRO: Vijayawada │ │ SRO: Vijayawada │ │ SRO: Guntur     │    │
│ │ Location:       │ │ Location:       │ │ Location:       │    │
│ │ Governorpet     │ │ Patamata        │ │ Arundalpet      │    │
│ │ ID: LOC-001     │ │ ID: LOC-002     │ │ ID: LOC-050     │    │
│ │ Code: VJA-001   │ │ Code: VJA-002   │ │ Code: GNT-001   │    │
│ │          [× Del]│ │          [× Del]│ │          [× Del]│    │
│ └─────────────────┘ └─────────────────┘ └─────────────────┘    │
│                « 1 [2] 3 4 ... 25 »                              │
└──────────────────────────────────────────────────────────────────┘
```

### Advanced Filter Panel
```
┌────────────────────────────────┐
│ Advanced Filters          [×]  │
├────────────────────────────────┤
│ District:                      │
│ ☐ Krishna                      │
│ ☐ Guntur                       │
│ ☐ West Godavari                │
│                                │
│ SRO:                           │
│ ☐ Vijayawada                   │
│ ☐ Guntur                       │
│ ☐ Eluru                        │
│                                │
│ Mandal/ULB:                    │
│ [Type to search...___]         │
│                                │
│      [Reset]  [Apply Filters]  │
└────────────────────────────────┘
```

### Address Card Component (Grid View)
```
┌─────────────────────────────┐
│ District: Krishna           │
│ SRO: Vijayawada             │
│ ─────────────────────────── │
│ Location: Governorpet       │
│ Location ID: LOC-001        │
│ Mandal/ULB: VMC             │
│ Secretariat: Governorpet    │
│ Code: VJA-001               │
│                             │
│                  [× Delete] │
└─────────────────────────────┘
```

### Template Details

#### ship-to.html
- Same structure as bill-to.html
- View mode toggle in header
- Search bar with filter panel trigger
- Active filters display
- View container (swapped by HTMX)

#### ship-to-table.html
- Dynamic table based on config
- Horizontal scroll wrapper
- Fixed header
- Responsive design

#### ship-to-grid.html
- CSS Grid layout (responsive columns)
- Wraps ship-to-card.html components
- Loading skeleton for pagination

#### ship-to-card.html
- Card container with shadow
- Key fields displayed
- Delete action button
- Hover effects

#### search-bar.html
- Search input with debounce
- Filter trigger button
- Active filter chips
- Clear all button

#### filter-panel.html
- Slide-out panel
- Dynamic filter controls per column
- Checkbox lists for enums
- Apply/reset buttons

## Testing Checklist

### Backend Tests
- [ ] Create ship-to configuration with default columns
- [ ] Update ship-to configuration
- [ ] Upload ship-to addresses via CSV
- [ ] Upload ship-to addresses via Excel
- [ ] List ship-to addresses (table view)
- [ ] List ship-to addresses (grid view)
- [ ] Search addresses across all fields
- [ ] Filter addresses by single column
- [ ] Filter addresses by multiple columns
- [ ] Combined search and filter
- [ ] Pagination with search/filter
- [ ] Delete single ship-to address
- [ ] Toggle view mode preference
- [ ] Export ship-to addresses to CSV
- [ ] Verify ship-to and bill-to data isolation

### Frontend Tests
- [ ] Page loads in default view (table or grid)
- [ ] View mode toggle switches between table and grid
- [ ] Table view displays all columns correctly
- [ ] Grid view displays cards correctly
- [ ] Cards show 4-6 key fields
- [ ] Search input triggers filtered results
- [ ] Search highlights matching terms (optional)
- [ ] Filter panel opens/closes
- [ ] Apply filters updates results
- [ ] Active filters display as chips
- [ ] Remove individual filter chip works
- [ ] Clear all filters works
- [ ] Pagination works in table view
- [ ] Pagination works in grid view
- [ ] Delete address in table view
- [ ] Delete address in grid view
- [ ] Upload addresses via CSV
- [ ] Upload addresses via Excel
- [ ] Edit column configuration
- [ ] Empty state displays correctly
- [ ] Mobile responsive design works

### Integration Tests
- [ ] End-to-end ship-to address workflow
- [ ] Search and filter performance with 1000+ addresses
- [ ] View mode preference persists across sessions
- [ ] Concurrent address operations
- [ ] Reusable components work for both bill-to and ship-to
- [ ] Column configurations independent between list types
- [ ] Data isolation between bill-to and ship-to

### Performance Tests
- [ ] Page load with 1000+ addresses
- [ ] Search performance with large dataset
- [ ] Filter performance with multiple filters
- [ ] View mode switching speed
- [ ] Grid rendering with many cards
- [ ] Table rendering with many rows

## Acceptance Criteria

### Must Have
1. Ship-to addresses use same infrastructure as bill-to (reuse tables)
2. Separate column configuration for ship-to addresses
3. Default columns match mockup: District, SRO, Location, Location ID, Mandal/ULB, Secretariat Name, Secretariat Code
4. Support CSV and Excel upload for ship-to addresses
5. Table view displays addresses in responsive table
6. Grid/card view displays addresses in responsive grid layout
7. View mode toggle switches between table and grid views
8. Search functionality works across all address fields
9. Search results update without page reload (HTMX)
10. Filter by individual columns (District, SRO, etc.)
11. Multiple filters can be applied simultaneously
12. Active filters display as removable chips
13. Pagination works in both view modes
14. Delete individual addresses in both views
15. Reuse config-form and upload-form templates from Phase 7
16. Ship-to and bill-to data completely isolated

### Should Have
17. View mode preference saved in session/cookie
18. Advanced filter panel with multi-select options
19. Filter dropdowns populated from actual data (distinct values)
20. Search with debounce (300ms delay)
21. Loading indicators for search/filter operations
22. Result count displayed (e.g., "Showing 1-50 of 1,234")
23. Per-page selector (25, 50, 100, 200)
24. Jump to page input
25. Empty state with helpful instructions
26. Export filtered/searched results to CSV
27. Keyboard navigation support
28. Confirmation before bulk operations
29. Responsive design optimized for mobile
30. Grid view shows 2-4 columns based on screen size

### Nice to Have
31. Infinite scroll option for grid view
32. Saved search/filter presets
33. Recent searches dropdown
34. Bulk select and delete in both views
35. Inline edit for individual addresses
36. Address detail modal/slide-over
37. Map view integration (if location coordinates available)
38. Duplicate detection across ship-to addresses
39. Address validation against postal database
40. Column visibility toggle (hide/show columns in table)
41. Column reordering in table view
42. Expand/collapse card details in grid view
43. Print-friendly view
44. Share filter URL (encode filters in URL)
