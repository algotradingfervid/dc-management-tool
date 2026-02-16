# Phase 19: Global DC Listing & Filters

## Overview
Implement a comprehensive global DC listing page matching mockup `16-all-dcs.html`. This page displays ALL delivery challans across all projects in a single table with advanced filtering, searching, and sorting capabilities. Users can filter by project, DC type, status, and date range, search by DC number, sort by columns, and navigate to individual DC details.

**Tech Stack:**
- Go + Gin backend
- HTMX + Tailwind frontend
- SQLite database

## Prerequisites
- Phase 1-18 completed
- Delivery Challans table fully populated
- Projects and Addresses tables available
- Understanding of SQL filtering and pagination
- HTMX for dynamic filtering

## Goals
1. Create global DC listing page at GET /dcs
2. Display table with all DCs across all projects
3. Implement columns: DC Number, Type, Date, Project Name, Ship To, Status, Total Value
4. Add filters: By Project, By DC Type, By Status, By Date Range
5. Implement search by DC number
6. Enable column sorting (date, DC number)
7. Add pagination for large datasets
8. Use HTMX for filter changes without page reload
9. Make table rows clickable to navigate to DC detail
10. Optimize queries with JOINs

## Detailed Implementation Steps

### Step 1: Database Query Planning
Design efficient query to fetch:
- All DCs with pagination
- JOIN projects for project name
- JOIN addresses for ship-to summary
- Support multiple WHERE clauses for filters
- Support ORDER BY for sorting
- Support LIMIT/OFFSET for pagination

### Step 2: Backend - DC Listing Service
Create service methods:
- Get all DCs with filters
- Get total count for pagination
- Get list of all projects (for filter dropdown)
- Handle search query
- Handle sorting

### Step 3: Backend - DC Listing Handler
Create handler to:
- Parse query parameters (filters, search, sort, page)
- Call service to fetch filtered DCs
- Calculate pagination metadata
- Return full page or partial (HTMX)

### Step 4: Frontend - DC Listing Page
Create template with sections:
- Page header with title
- Filter bar (project, type, status, date range, search)
- DC table with sortable columns
- Pagination controls
- Empty state

### Step 5: Filter Components
Implement filter UI:
- Project dropdown (with "All Projects" option)
- DC Type dropdown (All/Transit/Official)
- Status dropdown (All/Draft/Issued)
- Date range inputs (from/to)
- Search input for DC number
- Apply/Clear buttons

### Step 6: HTMX Integration
Add HTMX attributes:
- Filter changes trigger partial table reload
- Search input with debounce
- Sort column clicks
- Pagination links
- Preserve other filters when changing one filter

### Step 7: Sorting Implementation
Implement column sorting:
- Default: sort by date descending (newest first)
- Click column header to sort
- Show sort indicators (▲▼)
- Support ascending/descending toggle

### Step 8: Pagination
Implement pagination:
- Page size: 25 DCs per page
- Show total count and page numbers
- Previous/Next buttons
- Jump to page input
- Show "Showing X-Y of Z results"

### Step 9: Table Design
Create responsive table:
- Fixed header on scroll
- Hover effects on rows
- Badge styling for Type and Status
- Clickable rows
- Responsive design (stack on mobile)

### Step 10: Empty States
Handle edge cases:
- No DCs exist
- No DCs match filters
- Different messages for each state
- "Clear filters" button

### Step 11: URL State Management
Preserve filter state in URL:
- Query parameters for all filters
- Shareable URLs
- Browser back/forward support
- Bookmark-friendly

### Step 12: Performance Optimization
Optimize for performance:
- Index on commonly filtered columns
- Efficient COUNT query
- Limit JOIN depth
- Paginate to reduce data transfer

## Files to Create/Modify

### Backend Files

**services/dc_listing_service.go** (create new)
```go
package services

import (
    "database/sql"
    "fmt"
    "strings"
    "time"
)

type DCListingService struct {
    db *sql.DB
}

type DCListItem struct {
    ID            string
    DCNumber      string
    Type          string
    DCDate        time.Time
    ProjectName   string
    ShipToSummary string
    Status        string
    TotalValue    *float64
}

type DCListFilters struct {
    ProjectID  string
    DCType     string // "all", "transit", "official"
    Status     string // "all", "draft", "issued"
    DateFrom   *time.Time
    DateTo     *time.Time
    Search     string
    SortBy     string // "dc_number", "dc_date"
    SortOrder  string // "asc", "desc"
    Page       int
    PageSize   int
}

type DCListResult struct {
    DCs        []DCListItem
    TotalCount int
    Page       int
    PageSize   int
    TotalPages int
}

func NewDCListingService(db *sql.DB) *DCListingService {
    return &DCListingService{db: db}
}

// Get all DCs with filters and pagination
func (s *DCListingService) GetDCs(filters DCListFilters) (*DCListResult, error) {
    // Build WHERE clause
    whereClauses := []string{}
    args := []interface{}{}

    if filters.ProjectID != "" && filters.ProjectID != "all" {
        whereClauses = append(whereClauses, "dc.project_id = ?")
        args = append(args, filters.ProjectID)
    }

    if filters.DCType != "" && filters.DCType != "all" {
        whereClauses = append(whereClauses, "dc.type = ?")
        args = append(args, filters.DCType)
    }

    if filters.Status != "" && filters.Status != "all" {
        whereClauses = append(whereClauses, "dc.status = ?")
        args = append(args, filters.Status)
    }

    if filters.DateFrom != nil {
        whereClauses = append(whereClauses, "dc.dc_date >= ?")
        args = append(args, filters.DateFrom)
    }

    if filters.DateTo != nil {
        whereClauses = append(whereClauses, "dc.dc_date <= ?")
        args = append(args, filters.DateTo)
    }

    if filters.Search != "" {
        whereClauses = append(whereClauses, "dc.dc_number LIKE ?")
        args = append(args, "%"+filters.Search+"%")
    }

    whereClause := ""
    if len(whereClauses) > 0 {
        whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
    }

    // Get total count
    countQuery := fmt.Sprintf(`
        SELECT COUNT(*)
        FROM delivery_challans dc
        %s
    `, whereClause)

    var totalCount int
    err := s.db.QueryRow(countQuery, args...).Scan(&totalCount)
    if err != nil {
        return nil, err
    }

    // Build ORDER BY clause
    orderBy := "dc.dc_date DESC" // Default: newest first
    if filters.SortBy != "" {
        sortColumn := "dc.dc_date"
        if filters.SortBy == "dc_number" {
            sortColumn = "dc.dc_number"
        }
        sortOrder := "DESC"
        if filters.SortOrder == "asc" {
            sortOrder = "ASC"
        }
        orderBy = fmt.Sprintf("%s %s", sortColumn, sortOrder)
    }

    // Calculate pagination
    if filters.PageSize <= 0 {
        filters.PageSize = 25
    }
    if filters.Page <= 0 {
        filters.Page = 1
    }
    offset := (filters.Page - 1) * filters.PageSize

    // Get DCs
    query := fmt.Sprintf(`
        SELECT
            dc.id,
            dc.dc_number,
            dc.type,
            dc.dc_date,
            p.name as project_name,
            COALESCE(sa.city || ', ' || sa.state, 'N/A') as ship_to_summary,
            dc.status,
            dc.total_value
        FROM delivery_challans dc
        LEFT JOIN projects p ON dc.project_id = p.id
        LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
        %s
        ORDER BY %s
        LIMIT ? OFFSET ?
    `, whereClause, orderBy)

    args = append(args, filters.PageSize, offset)

    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var dcs []DCListItem
    for rows.Next() {
        var dc DCListItem
        err := rows.Scan(
            &dc.ID,
            &dc.DCNumber,
            &dc.Type,
            &dc.DCDate,
            &dc.ProjectName,
            &dc.ShipToSummary,
            &dc.Status,
            &dc.TotalValue,
        )
        if err != nil {
            return nil, err
        }
        dcs = append(dcs, dc)
    }

    totalPages := (totalCount + filters.PageSize - 1) / filters.PageSize

    return &DCListResult{
        DCs:        dcs,
        TotalCount: totalCount,
        Page:       filters.Page,
        PageSize:   filters.PageSize,
        TotalPages: totalPages,
    }, nil
}

// Get all projects for filter dropdown
func (s *DCListingService) GetAllProjects() ([]Project, error) {
    query := "SELECT id, name FROM projects ORDER BY name"
    rows, err := s.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var projects []Project
    for rows.Next() {
        var project Project
        err := rows.Scan(&project.ID, &project.Name)
        if err != nil {
            return nil, err
        }
        projects = append(projects, project)
    }

    return projects, nil
}

type Project struct {
    ID   string
    Name string
}
```

**handlers/dc_listing_handler.go** (create new)
```go
package handlers

import (
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
)

type DCListingHandler struct {
    dcListingService *DCListingService
}

func NewDCListingHandler(dcListingService *DCListingService) *DCListingHandler {
    return &DCListingHandler{dcListingService: dcListingService}
}

// Show all DCs with filters
func (h *DCListingHandler) ListAllDCs(c *gin.Context) {
    // Parse filters from query params
    filters := DCListFilters{
        ProjectID: c.DefaultQuery("project", "all"),
        DCType:    c.DefaultQuery("type", "all"),
        Status:    c.DefaultQuery("status", "all"),
        Search:    c.Query("search"),
        SortBy:    c.DefaultQuery("sort_by", "dc_date"),
        SortOrder: c.DefaultQuery("sort_order", "desc"),
    }

    // Parse date filters
    if dateFrom := c.Query("date_from"); dateFrom != "" {
        if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
            filters.DateFrom = &t
        }
    }
    if dateTo := c.Query("date_to"); dateTo != "" {
        if t, err := time.Parse("2006-01-02", dateTo); err == nil {
            filters.DateTo = &t
        }
    }

    // Parse pagination
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    filters.Page = page
    filters.PageSize = 25

    // Get DCs
    result, err := h.dcListingService.GetDCs(filters)
    if err != nil {
        c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
        return
    }

    // Get all projects for filter dropdown
    projects, err := h.dcListingService.GetAllProjects()
    if err != nil {
        c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
        return
    }

    // If HTMX request, return only the table section
    if c.GetHeader("HX-Request") == "true" {
        c.HTML(http.StatusOK, "dc-listing-table-partial.html", gin.H{
            "Result":  result,
            "Filters": filters,
        })
        return
    }

    // Full page render
    c.HTML(http.StatusOK, "dc-listing.html", gin.H{
        "Result":   result,
        "Filters":  filters,
        "Projects": projects,
    })
}
```

### Frontend Files

**templates/dc-listing.html** (create new)
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>All DCs - DC Management Tool</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@2.0.8"></script>
</head>
<body class="bg-gray-50">
    <!-- Navigation -->
    {{ template "nav.html" . }}

    <!-- Main Container -->
    <div class="max-w-7xl mx-auto px-4 py-8">
        <!-- Header -->
        <div class="mb-8">
            <h1 class="text-3xl font-bold text-gray-900">All Delivery Challans</h1>
            <p class="text-gray-600 mt-2">View and manage all DCs across all projects</p>
        </div>

        <!-- Filter Bar -->
        <div class="bg-white rounded-lg shadow p-6 mb-6">
            <form id="filter-form" hx-get="/dcs" hx-target="#dc-table-container" hx-trigger="submit">
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4 mb-4">
                    <!-- Project Filter -->
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">Project</label>
                        <select name="project" class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                            <option value="all" {{ if eq .Filters.ProjectID "all" }}selected{{ end }}>All Projects</option>
                            {{ range .Projects }}
                            <option value="{{ .ID }}" {{ if eq $.Filters.ProjectID .ID }}selected{{ end }}>{{ .Name }}</option>
                            {{ end }}
                        </select>
                    </div>

                    <!-- DC Type Filter -->
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">DC Type</label>
                        <select name="type" class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                            <option value="all" {{ if eq .Filters.DCType "all" }}selected{{ end }}>All Types</option>
                            <option value="transit" {{ if eq .Filters.DCType "transit" }}selected{{ end }}>Transit</option>
                            <option value="official" {{ if eq .Filters.DCType "official" }}selected{{ end }}>Official</option>
                        </select>
                    </div>

                    <!-- Status Filter -->
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">Status</label>
                        <select name="status" class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                            <option value="all" {{ if eq .Filters.Status "all" }}selected{{ end }}>All Status</option>
                            <option value="draft" {{ if eq .Filters.Status "draft" }}selected{{ end }}>Draft</option>
                            <option value="issued" {{ if eq .Filters.Status "issued" }}selected{{ end }}>Issued</option>
                        </select>
                    </div>

                    <!-- Date From -->
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">From Date</label>
                        <input type="date" name="date_from" value="{{ if .Filters.DateFrom }}{{ .Filters.DateFrom.Format "2006-01-02" }}{{ end }}"
                               class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                    </div>

                    <!-- Date To -->
                    <div>
                        <label class="block text-sm font-medium text-gray-700 mb-1">To Date</label>
                        <input type="date" name="date_to" value="{{ if .Filters.DateTo }}{{ .Filters.DateTo.Format "2006-01-02" }}{{ end }}"
                               class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500">
                    </div>
                </div>

                <!-- Search and Buttons -->
                <div class="flex flex-col md:flex-row gap-4">
                    <!-- Search -->
                    <div class="flex-1">
                        <label class="block text-sm font-medium text-gray-700 mb-1">Search DC Number</label>
                        <input type="text" name="search" value="{{ .Filters.Search }}" placeholder="Enter DC number..."
                               class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                               hx-get="/dcs" hx-target="#dc-table-container" hx-trigger="keyup changed delay:500ms">
                    </div>

                    <!-- Buttons -->
                    <div class="flex gap-2 items-end">
                        <button type="submit" class="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 font-medium">
                            Apply Filters
                        </button>
                        <a href="/dcs" class="bg-gray-200 text-gray-700 px-6 py-2 rounded-lg hover:bg-gray-300 font-medium">
                            Clear
                        </a>
                    </div>
                </div>
            </form>
        </div>

        <!-- DC Table Container (HTMX Target) -->
        <div id="dc-table-container">
            {{ template "dc-listing-table-partial.html" . }}
        </div>
    </div>
</body>
</html>
```

**templates/dc-listing-table-partial.html** (create new)
```html
<!-- Results Count -->
<div class="bg-white rounded-lg shadow mb-4 px-6 py-3 flex justify-between items-center">
    <p class="text-sm text-gray-600">
        Showing {{ if gt .Result.TotalCount 0 }}{{ add (mul (sub .Result.Page 1) .Result.PageSize) 1 }}-{{ min (mul .Result.Page .Result.PageSize) .Result.TotalCount }}{{ else }}0{{ end }} of {{ .Result.TotalCount }} results
    </p>
    <p class="text-sm text-gray-600">
        Page {{ .Result.Page }} of {{ .Result.TotalPages }}
    </p>
</div>

<!-- DC Table -->
<div class="bg-white rounded-lg shadow overflow-hidden">
    <div class="overflow-x-auto">
        <table class="w-full">
            <thead class="bg-gray-50 border-b border-gray-200">
                <tr>
                    <!-- DC Number (sortable) -->
                    <th class="px-6 py-3 text-left">
                        <a href="#" hx-get="/dcs?sort_by=dc_number&sort_order={{ if and (eq .Filters.SortBy "dc_number") (eq .Filters.SortOrder "asc") }}desc{{ else }}asc{{ end }}"
                           hx-target="#dc-table-container" hx-include="#filter-form"
                           class="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center gap-1 hover:text-gray-700">
                            DC Number
                            {{ if eq .Filters.SortBy "dc_number" }}
                                {{ if eq .Filters.SortOrder "asc" }}▲{{ else }}▼{{ end }}
                            {{ end }}
                        </a>
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                    <!-- DC Date (sortable) -->
                    <th class="px-6 py-3 text-left">
                        <a href="#" hx-get="/dcs?sort_by=dc_date&sort_order={{ if and (eq .Filters.SortBy "dc_date") (eq .Filters.SortOrder "asc") }}desc{{ else }}asc{{ end }}"
                           hx-target="#dc-table-container" hx-include="#filter-form"
                           class="text-xs font-medium text-gray-500 uppercase tracking-wider flex items-center gap-1 hover:text-gray-700">
                            Date
                            {{ if eq .Filters.SortBy "dc_date" }}
                                {{ if eq .Filters.SortOrder "asc" }}▲{{ else }}▼{{ end }}
                            {{ end }}
                        </a>
                    </th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Project Name</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Ship To</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Total Value</th>
                </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
                {{ range .Result.DCs }}
                <tr class="hover:bg-gray-50 cursor-pointer transition" onclick="window.location='/dcs/{{ .ID }}/{{ if eq .Type "transit" }}transit{{ else }}official{{ end }}'">
                    <td class="px-6 py-4 whitespace-nowrap">
                        <a href="/dcs/{{ .ID }}/{{ if eq .Type "transit" }}transit{{ else }}official{{ end }}" class="text-blue-600 hover:text-blue-800 font-medium">
                            {{ .DCNumber }}
                        </a>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        {{ if eq .Type "transit" }}
                        <span class="px-2 py-1 text-xs font-semibold rounded-full bg-blue-100 text-blue-800">Transit</span>
                        {{ else }}
                        <span class="px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">Official</span>
                        {{ end }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        {{ .DCDate.Format "02/01/2006" }}
                    </td>
                    <td class="px-6 py-4">
                        <div class="text-sm text-gray-900">{{ .ProjectName }}</div>
                    </td>
                    <td class="px-6 py-4">
                        <div class="text-sm text-gray-600">{{ .ShipToSummary }}</div>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        {{ if eq .Status "draft" }}
                        <span class="px-2 py-1 text-xs font-semibold rounded-full bg-orange-100 text-orange-800">Draft</span>
                        {{ else }}
                        <span class="px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">Issued</span>
                        {{ end }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {{ if .TotalValue }}
                            ₹{{ printf "%.2f" .TotalValue }}
                        {{ else }}
                            —
                        {{ end }}
                    </td>
                </tr>
                {{ else }}
                <tr>
                    <td colspan="7" class="px-6 py-12 text-center">
                        <div class="flex flex-col items-center">
                            <svg class="w-16 h-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                            </svg>
                            <p class="text-gray-600 text-lg font-medium mb-2">No DCs found</p>
                            <p class="text-gray-500">Try adjusting your filters or create a new DC</p>
                            <a href="/dcs" class="mt-4 text-blue-600 hover:text-blue-800 font-medium">Clear all filters</a>
                        </div>
                    </td>
                </tr>
                {{ end }}
            </tbody>
        </table>
    </div>
</div>

<!-- Pagination -->
{{ if gt .Result.TotalPages 1 }}
<div class="bg-white rounded-lg shadow mt-4 px-6 py-4">
    <div class="flex flex-col md:flex-row justify-between items-center gap-4">
        <!-- Previous Button -->
        <div>
            {{ if gt .Result.Page 1 }}
            <a href="#" hx-get="/dcs?page={{ sub .Result.Page 1 }}" hx-target="#dc-table-container" hx-include="#filter-form"
               class="bg-gray-200 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-300 font-medium">
                ← Previous
            </a>
            {{ else }}
            <span class="bg-gray-100 text-gray-400 px-4 py-2 rounded-lg cursor-not-allowed">
                ← Previous
            </span>
            {{ end }}
        </div>

        <!-- Page Numbers -->
        <div class="flex gap-2">
            {{ range $i := iterate .Result.TotalPages }}
            {{ if eq (add $i 1) $.Result.Page }}
            <span class="bg-blue-600 text-white px-4 py-2 rounded-lg font-medium">
                {{ add $i 1 }}
            </span>
            {{ else }}
            <a href="#" hx-get="/dcs?page={{ add $i 1 }}" hx-target="#dc-table-container" hx-include="#filter-form"
               class="bg-gray-200 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-300 font-medium">
                {{ add $i 1 }}
            </a>
            {{ end }}
            {{ end }}
        </div>

        <!-- Next Button -->
        <div>
            {{ if lt .Result.Page .Result.TotalPages }}
            <a href="#" hx-get="/dcs?page={{ add .Result.Page 1 }}" hx-target="#dc-table-container" hx-include="#filter-form"
               class="bg-gray-200 text-gray-700 px-4 py-2 rounded-lg hover:bg-gray-300 font-medium">
                Next →
            </a>
            {{ else }}
            <span class="bg-gray-100 text-gray-400 px-4 py-2 rounded-lg cursor-not-allowed">
                Next →
            </span>
            {{ end }}
        </div>
    </div>
</div>
{{ end }}
```

## API Routes/Endpoints

### Route Definitions

**main.go** (modify)
```go
// DC listing route
dcListingHandler := handlers.NewDCListingHandler(dcListingService)
r.GET("/dcs", dcListingHandler.ListAllDCs)
```

### Endpoint Details

| Method | Endpoint | Description | Response |
|--------|----------|-------------|----------|
| GET | `/dcs` | List all DCs with filters | HTML page or partial |

### Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `project` | string | Filter by project ID or "all" | `project=123` |
| `type` | string | Filter by DC type: "all", "transit", "official" | `type=transit` |
| `status` | string | Filter by status: "all", "draft", "issued" | `status=draft` |
| `date_from` | date | Filter from date (YYYY-MM-DD) | `date_from=2024-01-01` |
| `date_to` | date | Filter to date (YYYY-MM-DD) | `date_to=2024-12-31` |
| `search` | string | Search by DC number (partial match) | `search=FSS-24` |
| `sort_by` | string | Sort column: "dc_number", "dc_date" | `sort_by=dc_date` |
| `sort_order` | string | Sort order: "asc", "desc" | `sort_order=desc` |
| `page` | int | Page number (default: 1) | `page=2` |

### Example URLs
```
/dcs
/dcs?type=transit
/dcs?project=123&status=draft
/dcs?date_from=2024-01-01&date_to=2024-12-31
/dcs?search=FSS-24&page=2
/dcs?sort_by=dc_number&sort_order=asc
```

## Database Queries

### Get All DCs with Filters
```sql
SELECT
    dc.id,
    dc.dc_number,
    dc.type,
    dc.dc_date,
    p.name as project_name,
    COALESCE(sa.city || ', ' || sa.state, 'N/A') as ship_to_summary,
    dc.status,
    dc.total_value
FROM delivery_challans dc
LEFT JOIN projects p ON dc.project_id = p.id
LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
WHERE dc.project_id = ?
AND dc.type = ?
AND dc.status = ?
AND dc.dc_date >= ?
AND dc.dc_date <= ?
AND dc.dc_number LIKE ?
ORDER BY dc.dc_date DESC
LIMIT 25 OFFSET 0;
```

### Count Total DCs with Filters
```sql
SELECT COUNT(*)
FROM delivery_challans dc
WHERE dc.project_id = ?
AND dc.type = ?
AND dc.status = ?
AND dc.dc_date >= ?
AND dc.dc_date <= ?
AND dc.dc_number LIKE ?;
```

### Get All Projects for Filter
```sql
SELECT id, name
FROM projects
ORDER BY name;
```

### Indexes for Performance
```sql
CREATE INDEX idx_dc_project_id ON delivery_challans(project_id);
CREATE INDEX idx_dc_type ON delivery_challans(type);
CREATE INDEX idx_dc_status ON delivery_challans(status);
CREATE INDEX idx_dc_date ON delivery_challans(dc_date);
CREATE INDEX idx_dc_number ON delivery_challans(dc_number);
```

## UI Components

### Component Breakdown

1. **Filter Bar Component**
   - White background card
   - Grid layout (responsive)
   - 5 filter inputs: Project, Type, Status, Date From, Date To
   - Search input with debounce
   - Apply and Clear buttons

2. **DC Table Component**
   - Fixed header
   - 7 columns
   - Sortable column headers (DC Number, Date)
   - Row hover effects
   - Clickable rows
   - Badge styling

3. **Pagination Component**
   - Results count
   - Page numbers
   - Previous/Next buttons
   - Current page highlighted
   - Disabled state for first/last page

4. **Empty State Component**
   - Icon
   - Message
   - "Clear filters" link

### Tailwind Classes Used
- Layout: `grid`, `grid-cols-1`, `md:grid-cols-2`, `lg:grid-cols-3`
- Table: `table`, `w-full`, `divide-y`
- Badges: `px-2`, `py-1`, `rounded-full`
- Buttons: `bg-blue-600`, `hover:bg-blue-700`, `rounded-lg`
- Filters: `border`, `focus:ring-2`, `focus:ring-blue-500`

## Testing Checklist

### Functional Testing
- [ ] DC listing page loads at /dcs
- [ ] All DCs display in table
- [ ] Columns display correctly: DC Number, Type, Date, Project, Ship To, Status, Total Value
- [ ] Project filter works (all projects, specific project)
- [ ] DC Type filter works (all, transit, official)
- [ ] Status filter works (all, draft, issued)
- [ ] Date From filter works
- [ ] Date To filter works
- [ ] Search by DC number works (partial match)
- [ ] Sort by DC Number works (asc/desc)
- [ ] Sort by Date works (asc/desc)
- [ ] Pagination works (next/previous)
- [ ] Page numbers clickable
- [ ] Pagination disabled at boundaries
- [ ] Results count accurate
- [ ] Clear filters button resets all filters

### HTMX Testing
- [ ] Filter changes trigger partial reload
- [ ] Search input debounces (500ms delay)
- [ ] Sort links trigger partial reload
- [ ] Pagination links trigger partial reload
- [ ] Only table section refreshes (not full page)
- [ ] Filter values preserved across HTMX requests
- [ ] HX-Request header detected

### Navigation Testing
- [ ] Click DC number navigates to detail page
- [ ] Click table row navigates to detail page
- [ ] Correct detail page loaded (transit vs official)
- [ ] Back button works (preserves filters in URL)

### Badge Testing
- [ ] Transit badge shows blue
- [ ] Official badge shows green
- [ ] Draft badge shows orange
- [ ] Issued badge shows green

### Responsive Testing
- [ ] Filters stack on mobile
- [ ] Table scrolls horizontally on mobile
- [ ] Pagination works on mobile
- [ ] Touch-friendly controls

### Performance Testing
- [ ] Page loads in < 1 second
- [ ] Filter application < 500ms
- [ ] Handles 1000+ DCs without slowdown
- [ ] Efficient queries with JOINs
- [ ] Indexes utilized

### Edge Cases
- [ ] No DCs exist (empty state)
- [ ] No DCs match filters (empty state with clear link)
- [ ] Only 1 DC exists
- [ ] 1000+ DCs (pagination works)
- [ ] Long project names truncate
- [ ] Long DC numbers display properly
- [ ] Null ship-to addresses show "N/A"
- [ ] Null total values show "—"

## Acceptance Criteria

### Must Have
1. ✅ DC listing page accessible at GET /dcs
2. ✅ Table displays all DCs across all projects
3. ✅ Columns: DC Number (link), Type (badge), Date, Project Name, Ship To (summary), Status (badge), Total Value
4. ✅ Filter by Project (dropdown with all projects)
5. ✅ Filter by DC Type (All/Transit/Official)
6. ✅ Filter by Status (All/Draft/Issued)
7. ✅ Filter by Date Range (From/To)
8. ✅ Search by DC number (partial match, real-time)
9. ✅ Sort by DC Number (asc/desc)
10. ✅ Sort by Date (asc/desc, default: desc)
11. ✅ Pagination (25 per page)
12. ✅ Results count: "Showing X-Y of Z results"
13. ✅ Page numbers clickable
14. ✅ Previous/Next buttons
15. ✅ HTMX partial table refresh on filter changes
16. ✅ Clickable rows navigate to DC detail
17. ✅ Empty state when no DCs found
18. ✅ Clear filters button
19. ✅ URL preserves filter state (shareable)

### Nice to Have
1. ⭐ Export filtered results as Excel/CSV
2. ⭐ Bulk actions (delete, mark as issued)
3. ⭐ Save filter presets
4. ⭐ Column visibility toggle
5. ⭐ Advanced filters (ship-to location, product name)
6. ⭐ Infinite scroll instead of pagination
7. ⭐ Loading spinner during HTMX refresh
8. ⭐ Filter chips showing active filters
9. ⭐ Recent searches dropdown

### Performance Criteria
- Page load < 1 second
- Filter application < 500ms
- Handle 10,000+ DCs
- Efficient SQL queries with indexes
- Pagination reduces data transfer

### Accessibility Criteria
- Keyboard navigation for filters
- Screen reader friendly table
- Focus states on interactive elements
- Semantic HTML

---

## Notes
- Default sort: Date descending (newest first)
- Page size fixed at 25 (can make configurable later)
- Search is real-time with 500ms debounce
- Filters are AND logic (all must match)
- URL state enables bookmarking and sharing
- Consider adding filter chips to show active filters
- Future: save user filter presets
- Future: export filtered results to Excel
