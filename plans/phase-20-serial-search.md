# Phase 20: Serial Number Search

## Overview
Implement a global serial number search feature matching mockup `17-serial-search.html`. This powerful search tool allows users to track specific items across all delivery challans by searching for serial numbers. Users can search globally across all projects or filter within a specific project. Results show which DC contains each serial number, along with product details, project information, and shipment location.

**Tech Stack:**
- Go + Gin backend
- HTMX + Tailwind frontend
- SQLite database

## Prerequisites
- Phase 1-19 completed
- Serial Numbers table fully populated
- Products table linked to DCs
- Delivery Challans table with project associations
- Understanding of SQL JOINs across multiple tables

## Goals
1. Create serial number search page at GET /serial-search
2. Implement global search across all projects
3. Support filtering search within a specific project
4. Display results: DC Number, DC Type, Project Name, Product Name, DC Date, Ship To location, Status
5. Enable real-time search as user types (HTMX)
6. Support partial matches (LIKE query)
7. Support exact matches
8. Show "No results found" state
9. Link from results to specific DC detail page
10. Optimize search queries for performance

## Detailed Implementation Steps

### Step 1: Database Query Design
Design search query to JOIN:
- serial_numbers table
- products table (to get product details)
- delivery_challans table (to get DC details)
- projects table (to get project name)
- addresses table (to get ship-to location)

### Step 2: Backend - Serial Search Service
Create service methods:
- Search serial numbers globally
- Search serial numbers within project
- Support partial matching
- Support exact matching
- Optimize query performance

### Step 3: Backend - Serial Search Handler
Create handler to:
- Parse search query parameter
- Parse optional project filter
- Call service to fetch results
- Return full page or partial (HTMX)

### Step 4: Frontend - Serial Search Page
Create template with sections:
- Page header with title and description
- Search bar with project filter
- Results table
- Empty state (before search)
- No results state

### Step 5: Search Input Component
Implement search UI:
- Large search input field
- Placeholder text
- Optional project dropdown filter
- Real-time search with debounce
- Clear search button

### Step 6: HTMX Integration
Add HTMX for real-time search:
- Trigger on keyup with delay (300ms)
- Update results section only
- Show loading state
- Handle empty query

### Step 7: Results Table Component
Design results table:
- Columns: Serial Number, DC Number, DC Type, Project, Product, DC Date, Ship To, Status
- Link DC Number to detail page
- Badge styling for type and status
- Responsive design

### Step 8: Empty States
Implement multiple states:
- Initial state: "Enter a serial number to search"
- Searching state: Loading spinner
- No results state: "No serial numbers found"
- Results state: Table with data

### Step 9: Performance Optimization
Optimize search:
- Index on serial_numbers.serial_number
- LIMIT results (max 100)
- Efficient JOIN query
- Cache common searches (optional)

### Step 10: Search Highlighting
Highlight matched portion of serial number:
- Use text highlighting
- Show context around match

### Step 11: Export Results
Optional: Add export functionality:
- Download results as Excel
- Include all result columns

## Files to Create/Modify

### Backend Files

**services/serial_search_service.go** (create new)
```go
package services

import (
    "database/sql"
    "strings"
    "time"
)

type SerialSearchService struct {
    db *sql.DB
}

type SerialSearchResult struct {
    SerialNumber  string
    DCNumber      string
    DCID          string
    DCType        string
    ProjectName   string
    ProductName   string
    DCDate        time.Time
    ShipToSummary string
    Status        string
}

func NewSerialSearchService(db *sql.DB) *SerialSearchService {
    return &SerialSearchService{db: db}
}

// Search serial numbers globally or within a project
func (s *SerialSearchService) SearchSerialNumbers(query string, projectID string) ([]SerialSearchResult, error) {
    if query == "" {
        return []SerialSearchResult{}, nil
    }

    // Build WHERE clause
    whereClauses := []string{"sn.serial_number LIKE ?"}
    args := []interface{}{"%" + query + "%"}

    if projectID != "" && projectID != "all" {
        whereClauses = append(whereClauses, "dc.project_id = ?")
        args = append(args, projectID)
    }

    whereClause := strings.Join(whereClauses, " AND ")

    // Search query with JOINs
    sqlQuery := `
        SELECT
            sn.serial_number,
            dc.dc_number,
            dc.id as dc_id,
            dc.type as dc_type,
            p.name as project_name,
            pr.item_name as product_name,
            dc.dc_date,
            COALESCE(sa.city || ', ' || sa.state, 'N/A') as ship_to_summary,
            dc.status
        FROM serial_numbers sn
        INNER JOIN products pr ON sn.product_id = pr.id
        INNER JOIN delivery_challans dc ON pr.delivery_challan_id = dc.id
        LEFT JOIN projects p ON dc.project_id = p.id
        LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
        WHERE ` + whereClause + `
        ORDER BY dc.dc_date DESC
        LIMIT 100
    `

    rows, err := s.db.Query(sqlQuery, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []SerialSearchResult
    for rows.Next() {
        var result SerialSearchResult
        err := rows.Scan(
            &result.SerialNumber,
            &result.DCNumber,
            &result.DCID,
            &result.DCType,
            &result.ProjectName,
            &result.ProductName,
            &result.DCDate,
            &result.ShipToSummary,
            &result.Status,
        )
        if err != nil {
            return nil, err
        }
        results = append(results, result)
    }

    return results, nil
}

// Search for exact match (useful for verification)
func (s *SerialSearchService) SearchExact(serialNumber string, projectID string) (*SerialSearchResult, error) {
    whereClauses := []string{"sn.serial_number = ?"}
    args := []interface{}{serialNumber}

    if projectID != "" && projectID != "all" {
        whereClauses = append(whereClauses, "dc.project_id = ?")
        args = append(args, projectID)
    }

    whereClause := strings.Join(whereClauses, " AND ")

    query := `
        SELECT
            sn.serial_number,
            dc.dc_number,
            dc.id as dc_id,
            dc.type as dc_type,
            p.name as project_name,
            pr.item_name as product_name,
            dc.dc_date,
            COALESCE(sa.city || ', ' || sa.state, 'N/A') as ship_to_summary,
            dc.status
        FROM serial_numbers sn
        INNER JOIN products pr ON sn.product_id = pr.id
        INNER JOIN delivery_challans dc ON pr.delivery_challan_id = dc.id
        LEFT JOIN projects p ON dc.project_id = p.id
        LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
        WHERE ` + whereClause + `
        LIMIT 1
    `

    var result SerialSearchResult
    err := s.db.QueryRow(query, args...).Scan(
        &result.SerialNumber,
        &result.DCNumber,
        &result.DCID,
        &result.DCType,
        &result.ProjectName,
        &result.ProductName,
        &result.DCDate,
        &result.ShipToSummary,
        &result.Status,
    )

    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    return &result, nil
}

// Get statistics about serial number usage
func (s *SerialSearchService) GetSerialNumberStats() (map[string]int, error) {
    query := `
        SELECT
            COUNT(DISTINCT sn.serial_number) as total_serials,
            COUNT(DISTINCT dc.id) as total_dcs,
            COUNT(DISTINCT p.id) as total_projects
        FROM serial_numbers sn
        INNER JOIN products pr ON sn.product_id = pr.id
        INNER JOIN delivery_challans dc ON pr.delivery_challan_id = dc.id
        LEFT JOIN projects p ON dc.project_id = p.id
    `

    var totalSerials, totalDCs, totalProjects int
    err := s.db.QueryRow(query).Scan(&totalSerials, &totalDCs, &totalProjects)
    if err != nil {
        return nil, err
    }

    return map[string]int{
        "total_serials":  totalSerials,
        "total_dcs":      totalDCs,
        "total_projects": totalProjects,
    }, nil
}
```

**handlers/serial_search_handler.go** (create new)
```go
package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

type SerialSearchHandler struct {
    serialSearchService *SerialSearchService
    projectService      *ProjectService
}

func NewSerialSearchHandler(serialSearchService *SerialSearchService, projectService *ProjectService) *SerialSearchHandler {
    return &SerialSearchHandler{
        serialSearchService: serialSearchService,
        projectService:      projectService,
    }
}

// Show serial search page
func (h *SerialSearchHandler) ShowSerialSearch(c *gin.Context) {
    query := c.Query("q")
    projectID := c.DefaultQuery("project_id", "all")

    // Get all projects for filter
    projects, err := h.projectService.GetAllProjects()
    if err != nil {
        c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
        return
    }

    // If no query, show initial state
    if query == "" {
        // If HTMX request, return only results section
        if c.GetHeader("HX-Request") == "true" {
            c.HTML(http.StatusOK, "serial-search-results-partial.html", gin.H{
                "Query":   query,
                "Results": []SerialSearchResult{},
                "Initial": true,
            })
            return
        }

        c.HTML(http.StatusOK, "serial-search.html", gin.H{
            "Projects":  projects,
            "Query":     query,
            "ProjectID": projectID,
            "Results":   []SerialSearchResult{},
            "Initial":   true,
        })
        return
    }

    // Perform search
    results, err := h.serialSearchService.SearchSerialNumbers(query, projectID)
    if err != nil {
        c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
        return
    }

    // Get stats
    stats, _ := h.serialSearchService.GetSerialNumberStats()

    // If HTMX request, return only results section
    if c.GetHeader("HX-Request") == "true" {
        c.HTML(http.StatusOK, "serial-search-results-partial.html", gin.H{
            "Query":   query,
            "Results": results,
            "Initial": false,
        })
        return
    }

    // Full page render
    c.HTML(http.StatusOK, "serial-search.html", gin.H{
        "Projects":  projects,
        "Query":     query,
        "ProjectID": projectID,
        "Results":   results,
        "Stats":     stats,
        "Initial":   false,
    })
}
```

### Frontend Files

**templates/serial-search.html** (create new)
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Serial Number Search - DC Management Tool</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@2.0.8"></script>
</head>
<body class="bg-gray-50">
    <!-- Navigation -->
    {{ template "nav.html" . }}

    <!-- Main Container -->
    <div class="max-w-6xl mx-auto px-4 py-8">
        <!-- Header -->
        <div class="text-center mb-8">
            <h1 class="text-4xl font-bold text-gray-900 mb-4">Serial Number Search</h1>
            <p class="text-lg text-gray-600 max-w-2xl mx-auto">
                Track any item across all delivery challans by searching for its serial number
            </p>
        </div>

        <!-- Stats Cards (if available) -->
        {{ if .Stats }}
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
            <div class="bg-white rounded-lg shadow p-4 text-center">
                <p class="text-2xl font-bold text-blue-600">{{ .Stats.total_serials }}</p>
                <p class="text-sm text-gray-600">Total Serial Numbers</p>
            </div>
            <div class="bg-white rounded-lg shadow p-4 text-center">
                <p class="text-2xl font-bold text-green-600">{{ .Stats.total_dcs }}</p>
                <p class="text-sm text-gray-600">DCs with Serials</p>
            </div>
            <div class="bg-white rounded-lg shadow p-4 text-center">
                <p class="text-2xl font-bold text-purple-600">{{ .Stats.total_projects }}</p>
                <p class="text-sm text-gray-600">Projects Tracked</p>
            </div>
        </div>
        {{ end }}

        <!-- Search Form -->
        <div class="bg-white rounded-lg shadow p-6 mb-8">
            <form id="search-form" hx-get="/serial-search" hx-target="#search-results" hx-trigger="submit">
                <div class="flex flex-col md:flex-row gap-4">
                    <!-- Search Input -->
                    <div class="flex-1">
                        <label class="block text-sm font-medium text-gray-700 mb-2">Serial Number</label>
                        <input
                            type="text"
                            name="q"
                            value="{{ .Query }}"
                            placeholder="Enter serial number to search..."
                            class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 text-lg"
                            hx-get="/serial-search"
                            hx-target="#search-results"
                            hx-trigger="keyup changed delay:300ms"
                            hx-include="#search-form"
                            autofocus
                        >
                    </div>

                    <!-- Project Filter -->
                    <div class="md:w-64">
                        <label class="block text-sm font-medium text-gray-700 mb-2">Filter by Project</label>
                        <select
                            name="project_id"
                            class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                            hx-get="/serial-search"
                            hx-target="#search-results"
                            hx-trigger="change"
                            hx-include="#search-form"
                        >
                            <option value="all" {{ if eq .ProjectID "all" }}selected{{ end }}>All Projects</option>
                            {{ range .Projects }}
                            <option value="{{ .ID }}" {{ if eq $.ProjectID .ID }}selected{{ end }}>{{ .Name }}</option>
                            {{ end }}
                        </select>
                    </div>

                    <!-- Search Button -->
                    <div class="flex items-end">
                        <button type="submit" class="bg-blue-600 text-white px-8 py-3 rounded-lg hover:bg-blue-700 font-medium whitespace-nowrap">
                            Search
                        </button>
                    </div>
                </div>
            </form>
        </div>

        <!-- Results Container (HTMX Target) -->
        <div id="search-results">
            {{ template "serial-search-results-partial.html" . }}
        </div>
    </div>
</body>
</html>
```

**templates/serial-search-results-partial.html** (create new)
```html
{{ if .Initial }}
<!-- Initial State: Before Search -->
<div class="bg-white rounded-lg shadow p-12 text-center">
    <svg class="w-24 h-24 text-gray-300 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
    </svg>
    <h3 class="text-xl font-semibold text-gray-900 mb-2">Enter a serial number to search</h3>
    <p class="text-gray-600">Start typing to see results from all delivery challans</p>
</div>

{{ else if eq (len .Results) 0 }}
<!-- No Results State -->
<div class="bg-white rounded-lg shadow p-12 text-center">
    <svg class="w-24 h-24 text-gray-300 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
    </svg>
    <h3 class="text-xl font-semibold text-gray-900 mb-2">No serial numbers found</h3>
    <p class="text-gray-600 mb-4">No items match "{{ .Query }}" in any delivery challan</p>
    <p class="text-sm text-gray-500">Try a different search term or check your spelling</p>
</div>

{{ else }}
<!-- Results Table -->
<div class="bg-white rounded-lg shadow overflow-hidden">
    <!-- Results Header -->
    <div class="px-6 py-4 bg-gray-50 border-b border-gray-200">
        <h3 class="text-lg font-semibold text-gray-900">
            Found {{ len .Results }} result{{ if ne (len .Results) 1 }}s{{ end }} for "{{ .Query }}"
        </h3>
    </div>

    <!-- Results Table -->
    <div class="overflow-x-auto">
        <table class="w-full">
            <thead class="bg-gray-50 border-b border-gray-200">
                <tr>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Serial Number</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">DC Number</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">DC Type</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Project</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Product</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">DC Date</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Ship To</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
                {{ range .Results }}
                <tr class="hover:bg-gray-50 transition">
                    <!-- Serial Number -->
                    <td class="px-6 py-4 whitespace-nowrap">
                        <span class="font-mono text-sm font-medium text-gray-900">{{ .SerialNumber }}</span>
                    </td>

                    <!-- DC Number -->
                    <td class="px-6 py-4 whitespace-nowrap">
                        <a href="/dcs/{{ .DCID }}/{{ if eq .DCType "transit" }}transit{{ else }}official{{ end }}"
                           class="text-blue-600 hover:text-blue-800 font-medium">
                            {{ .DCNumber }}
                        </a>
                    </td>

                    <!-- DC Type -->
                    <td class="px-6 py-4 whitespace-nowrap">
                        {{ if eq .DCType "transit" }}
                        <span class="px-2 py-1 text-xs font-semibold rounded-full bg-blue-100 text-blue-800">Transit</span>
                        {{ else }}
                        <span class="px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">Official</span>
                        {{ end }}
                    </td>

                    <!-- Project Name -->
                    <td class="px-6 py-4">
                        <div class="text-sm text-gray-900">{{ .ProjectName }}</div>
                    </td>

                    <!-- Product Name -->
                    <td class="px-6 py-4">
                        <div class="text-sm text-gray-900">{{ .ProductName }}</div>
                    </td>

                    <!-- DC Date -->
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        {{ .DCDate.Format "02/01/2006" }}
                    </td>

                    <!-- Ship To -->
                    <td class="px-6 py-4">
                        <div class="text-sm text-gray-600">{{ .ShipToSummary }}</div>
                    </td>

                    <!-- Status -->
                    <td class="px-6 py-4 whitespace-nowrap">
                        {{ if eq .Status "draft" }}
                        <span class="px-2 py-1 text-xs font-semibold rounded-full bg-orange-100 text-orange-800">Draft</span>
                        {{ else }}
                        <span class="px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">Issued</span>
                        {{ end }}
                    </td>
                </tr>
                {{ end }}
            </tbody>
        </table>
    </div>

    <!-- Result Limit Notice -->
    {{ if eq (len .Results) 100 }}
    <div class="px-6 py-3 bg-yellow-50 border-t border-yellow-200">
        <p class="text-sm text-yellow-800">
            Showing first 100 results. Refine your search for more specific results.
        </p>
    </div>
    {{ end }}
</div>
{{ end }}
```

### Navigation Component

**templates/nav.html** (modify - add serial search link)
```html
<!-- Add to navigation menu -->
<a href="/serial-search" class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium">
    Serial Search
</a>
```

## API Routes/Endpoints

### Route Definitions

**main.go** (modify)
```go
// Serial search route
serialSearchHandler := handlers.NewSerialSearchHandler(serialSearchService, projectService)
r.GET("/serial-search", serialSearchHandler.ShowSerialSearch)
```

### Endpoint Details

| Method | Endpoint | Description | Response |
|--------|----------|-------------|----------|
| GET | `/serial-search` | Serial number search page | HTML page or partial |

### Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `q` | string | Serial number to search (partial match) | `q=ABC123` |
| `project_id` | string | Filter by project ID or "all" | `project_id=123` |

### Example URLs
```
/serial-search
/serial-search?q=ABC123
/serial-search?q=ABC&project_id=456
```

## Database Queries

### Search Serial Numbers (Partial Match)
```sql
SELECT
    sn.serial_number,
    dc.dc_number,
    dc.id as dc_id,
    dc.type as dc_type,
    p.name as project_name,
    pr.item_name as product_name,
    dc.dc_date,
    COALESCE(sa.city || ', ' || sa.state, 'N/A') as ship_to_summary,
    dc.status
FROM serial_numbers sn
INNER JOIN products pr ON sn.product_id = pr.id
INNER JOIN delivery_challans dc ON pr.delivery_challan_id = dc.id
LEFT JOIN projects p ON dc.project_id = p.id
LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
WHERE sn.serial_number LIKE '%ABC123%'
AND dc.project_id = '456'
ORDER BY dc.dc_date DESC
LIMIT 100;
```

### Search Serial Number (Exact Match)
```sql
SELECT
    sn.serial_number,
    dc.dc_number,
    dc.id as dc_id,
    dc.type as dc_type,
    p.name as project_name,
    pr.item_name as product_name,
    dc.dc_date,
    COALESCE(sa.city || ', ' || sa.state, 'N/A') as ship_to_summary,
    dc.status
FROM serial_numbers sn
INNER JOIN products pr ON sn.product_id = pr.id
INNER JOIN delivery_challans dc ON pr.delivery_challan_id = dc.id
LEFT JOIN projects p ON dc.project_id = p.id
LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
WHERE sn.serial_number = 'ABC123456'
LIMIT 1;
```

### Get Serial Number Statistics
```sql
SELECT
    COUNT(DISTINCT sn.serial_number) as total_serials,
    COUNT(DISTINCT dc.id) as total_dcs,
    COUNT(DISTINCT p.id) as total_projects
FROM serial_numbers sn
INNER JOIN products pr ON sn.product_id = pr.id
INNER JOIN delivery_challans dc ON pr.delivery_challan_id = dc.id
LEFT JOIN projects p ON dc.project_id = p.id;
```

### Index for Performance
```sql
CREATE INDEX idx_serial_numbers_serial ON serial_numbers(serial_number);
CREATE INDEX idx_serial_numbers_product ON serial_numbers(product_id);
CREATE INDEX idx_products_dc ON products(delivery_challan_id);
```

## UI Components

### Component Breakdown

1. **Search Header Component**
   - Large title
   - Descriptive subtitle
   - Centered layout

2. **Stats Cards Component** (optional)
   - 3 cards: Total Serials, DCs with Serials, Projects Tracked
   - Colored numbers
   - Grid layout

3. **Search Form Component**
   - Large search input
   - Project filter dropdown
   - Search button
   - HTMX integration for real-time search

4. **Results Table Component**
   - 8 columns
   - Clickable DC numbers
   - Badge styling
   - Row hover effects

5. **Initial State Component**
   - Search icon
   - Instructional text
   - Centered layout

6. **No Results State Component**
   - Sad face icon
   - "No results" message
   - Search term display
   - Helpful suggestions

7. **Result Limit Notice**
   - Yellow banner
   - "Showing first 100 results" message

### Tailwind Classes Used
- Layout: `max-w-6xl`, `mx-auto`, `grid`
- Forms: `px-4`, `py-3`, `border`, `focus:ring-2`
- Table: `table`, `w-full`, `divide-y`
- Badges: `px-2`, `py-1`, `rounded-full`
- Icons: `w-24`, `h-24`, `text-gray-300`

## Testing Checklist

### Functional Testing
- [ ] Serial search page loads at /serial-search
- [ ] Initial state displays before search
- [ ] Search input triggers real-time search
- [ ] Partial match search works (e.g., "ABC" finds "ABC123")
- [ ] Exact match search works
- [ ] Case-insensitive search works
- [ ] Project filter works (all projects, specific project)
- [ ] Results display all columns correctly
- [ ] DC number links to correct detail page
- [ ] DC type badge displays correctly
- [ ] Status badge displays correctly
- [ ] No results state displays when appropriate
- [ ] Result limit notice shows when 100+ results

### HTMX Testing
- [ ] Search input debounces (300ms delay)
- [ ] Project filter change triggers search
- [ ] Only results section refreshes (not full page)
- [ ] HX-Request header detected
- [ ] Loading state visible during search

### Navigation Testing
- [ ] Click DC number navigates to detail page
- [ ] Correct detail page loaded (transit vs official)
- [ ] Back button works
- [ ] Serial search link in navigation works

### Performance Testing
- [ ] Search completes in < 500ms
- [ ] Handles 10,000+ serial numbers
- [ ] Index on serial_number improves performance
- [ ] LIMIT 100 prevents slowdown

### Edge Cases
- [ ] Empty search query shows initial state
- [ ] Single character search works
- [ ] Very long serial numbers handled
- [ ] Special characters in serial numbers
- [ ] No serial numbers exist (empty state)
- [ ] 100+ results (limit notice)

### Responsive Testing
- [ ] Search form stacks on mobile
- [ ] Table scrolls horizontally on mobile
- [ ] Stats cards stack on mobile

## Acceptance Criteria

### Must Have
1. ✅ Serial search page accessible at GET /serial-search
2. ✅ Search input field with placeholder
3. ✅ Project filter dropdown (All Projects + specific projects)
4. ✅ Real-time search as user types (300ms debounce)
5. ✅ Partial match search (LIKE query)
6. ✅ Results table columns: Serial Number, DC Number, DC Type, Project, Product, DC Date, Ship To, Status
7. ✅ DC Number is clickable link to detail page
8. ✅ DC Type displayed as badge (Transit/Official)
9. ✅ Status displayed as badge (Draft/Issued)
10. ✅ Initial state: "Enter a serial number to search"
11. ✅ No results state: "No serial numbers found"
12. ✅ Result limit: max 100 results
13. ✅ Result limit notice when 100 results shown
14. ✅ HTMX partial refresh for search results
15. ✅ Search works globally across all projects
16. ✅ Search works within specific project when filtered

### Nice to Have
1. ⭐ Statistics cards: Total Serials, DCs with Serials, Projects Tracked
2. ⭐ Highlight matched portion of serial number
3. ⭐ Export results as Excel
4. ⭐ Recent searches dropdown
5. ⭐ Advanced search: search by product name
6. ⭐ Barcode scanner integration (mobile)
7. ⭐ Search history
8. ⭐ Save search queries
9. ⭐ QR code generation for serial numbers

### Performance Criteria
- Search query execution < 500ms
- Handle 100,000+ serial numbers
- Efficient JOIN query with indexes
- LIMIT 100 prevents memory issues

### Accessibility Criteria
- Search input is keyboard accessible
- Autofocus on search input
- Screen reader friendly table
- Clear focus states

---

## Notes
- Limit results to 100 to prevent performance issues
- Index on serial_numbers.serial_number is critical for performance
- Real-time search improves UX significantly
- Consider adding barcode scanner for mobile (future enhancement)
- Future: track serial number movements across multiple DCs
- Future: serial number verification API for external systems
- Serial numbers are case-insensitive for search
