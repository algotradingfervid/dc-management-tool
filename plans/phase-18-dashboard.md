# Phase 18: Dashboard & Statistics

## Overview
Implement a comprehensive dashboard as the home page after login, matching the mockup `02-dashboard.html`. The dashboard provides an overview of the DC Management Tool with summary statistics, recent activity, and quick action buttons. Features include total projects, total DCs issued, monthly DC count, draft DC count, type breakdowns, recent DCs list, and date range filters for statistics.

**Tech Stack:**
- Go + Gin backend
- HTMX + Tailwind frontend
- SQLite database
- Optional: Chart.js for visualizations

## Prerequisites
- Phase 1-17 completed
- Projects table populated
- Delivery Challans table with status and type fields
- Understanding of aggregate SQL queries
- HTMX for partial page updates

## Goals
1. Create dashboard as home page route (GET /dashboard)
2. Display summary cards: Total Projects, Total DCs, DCs This Month, Draft DCs
3. Show breakdown: Transit vs Official DC counts
4. List recent DCs (last 10-15) with key details
5. Implement quick action buttons
6. Add date range filter for statistics
7. Implement HTMX partial refresh for filter changes
8. Optional: Add simple charts/visualizations
9. Optimize dashboard queries for performance

## Detailed Implementation Steps

### Step 1: Database Queries for Statistics
Create service methods to calculate:
- Total count of projects
- Total count of DCs (all statuses)
- Count of DCs issued this month
- Count of draft DCs (status = 'draft')
- Count of Transit DCs
- Count of Official DCs
- Recent DCs with project name, ship-to summary

### Step 2: Date Range Filter Logic
Implement date range filtering:
- Default: current month statistics
- Custom ranges: last 7 days, last 30 days, last 3 months, all time
- Update statistics based on selected range
- Apply filter to all relevant counts

### Step 3: Backend - Dashboard Handler
Create handler to:
- Fetch all statistics
- Get recent DCs list
- Handle date range query parameters
- Return data for template rendering

### Step 4: Frontend - Dashboard Layout
Create dashboard template with sections:
- Header with title and date range filter
- Summary cards grid (4 cards)
- Stats breakdown section (Transit vs Official)
- Recent DCs table
- Quick action buttons

### Step 5: Summary Cards Component
Design and implement 4 summary cards:
- Total Projects (with icon)
- Total DCs Issued (with icon)
- DCs This Month (with icon)
- Draft DCs (with icon, warning color if > 0)

### Step 6: Recent DCs List Component
Create table/list showing:
- DC Number (link to detail page)
- Type (badge: Transit/Official)
- Project Name
- Date
- Status (badge: Draft/Issued)

### Step 7: Date Range Filter Component
Implement filter UI:
- Dropdown or button group for preset ranges
- Custom date picker for specific dates
- Apply button
- HTMX integration for partial updates

### Step 8: HTMX Integration
Add HTMX for dynamic updates:
- Filter change triggers partial reload
- Update only statistics and recent DCs sections
- Preserve URL query parameters
- Loading states during refresh

### Step 9: Quick Actions Section
Create quick action buttons:
- "Create New Project" -> /projects/new
- "View All DCs" -> /dcs
- "View All Projects" -> /projects

### Step 10: Optional Charts
If implementing charts:
- Use Chart.js (lightweight)
- Bar chart: DCs per month (last 6 months)
- Pie chart: Transit vs Official breakdown
- Line chart: DC trend over time

### Step 11: Performance Optimization
Optimize dashboard queries:
- Use COUNT queries with indexes
- Combine queries where possible
- Cache statistics (refresh every 5 minutes)
- Pagination for recent DCs if needed

### Step 12: Responsive Design
Ensure dashboard is responsive:
- Stack cards on mobile
- Responsive table for recent DCs
- Mobile-friendly filters

## Files to Create/Modify

### Backend Files

**services/dashboard_service.go** (create new)
```go
package services

import (
    "database/sql"
    "time"
)

type DashboardService struct {
    db *sql.DB
}

type DashboardStats struct {
    TotalProjects    int
    TotalDCs         int
    DCsThisMonth     int
    DraftDCs         int
    TransitDCs       int
    OfficialDCs      int
}

type RecentDC struct {
    ID            string
    DCNumber      string
    Type          string
    ProjectName   string
    DCDate        time.Time
    Status        string
    ShipToSummary string
    TotalValue    *float64 // Nullable, only for transit DCs
}

func NewDashboardService(db *sql.DB) *DashboardService {
    return &DashboardService{db: db}
}

// Get dashboard statistics with optional date range filter
func (s *DashboardService) GetStatistics(startDate, endDate *time.Time) (*DashboardStats, error) {
    stats := &DashboardStats{}

    // Total Projects
    err := s.db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&stats.TotalProjects)
    if err != nil {
        return nil, err
    }

    // Total DCs (with optional date filter)
    query := "SELECT COUNT(*) FROM delivery_challans WHERE 1=1"
    args := []interface{}{}

    if startDate != nil {
        query += " AND dc_date >= ?"
        args = append(args, startDate)
    }
    if endDate != nil {
        query += " AND dc_date <= ?"
        args = append(args, endDate)
    }

    err = s.db.QueryRow(query, args...).Scan(&stats.TotalDCs)
    if err != nil {
        return nil, err
    }

    // DCs This Month (current month only, ignoring date filter)
    now := time.Now()
    firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
    lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

    err = s.db.QueryRow(`
        SELECT COUNT(*) FROM delivery_challans
        WHERE dc_date >= ? AND dc_date <= ?
    `, firstDayOfMonth, lastDayOfMonth).Scan(&stats.DCsThisMonth)
    if err != nil {
        return nil, err
    }

    // Draft DCs (no date filter)
    err = s.db.QueryRow(`
        SELECT COUNT(*) FROM delivery_challans WHERE status = 'draft'
    `).Scan(&stats.DraftDCs)
    if err != nil {
        return nil, err
    }

    // Transit DCs (with optional date filter)
    query = "SELECT COUNT(*) FROM delivery_challans WHERE type = 'transit'"
    args = []interface{}{}

    if startDate != nil {
        query += " AND dc_date >= ?"
        args = append(args, startDate)
    }
    if endDate != nil {
        query += " AND dc_date <= ?"
        args = append(args, endDate)
    }

    err = s.db.QueryRow(query, args...).Scan(&stats.TransitDCs)
    if err != nil {
        return nil, err
    }

    // Official DCs (with optional date filter)
    query = "SELECT COUNT(*) FROM delivery_challans WHERE type = 'official'"
    args = []interface{}{}

    if startDate != nil {
        query += " AND dc_date >= ?"
        args = append(args, startDate)
    }
    if endDate != nil {
        query += " AND dc_date <= ?"
        args = append(args, endDate)
    }

    err = s.db.QueryRow(query, args...).Scan(&stats.OfficialDCs)
    if err != nil {
        return nil, err
    }

    return stats, nil
}

// Get recent DCs
func (s *DashboardService) GetRecentDCs(limit int) ([]RecentDC, error) {
    query := `
        SELECT
            dc.id,
            dc.dc_number,
            dc.type,
            p.name as project_name,
            dc.dc_date,
            dc.status,
            COALESCE(sa.city || ', ' || sa.state, 'N/A') as ship_to_summary,
            dc.total_value
        FROM delivery_challans dc
        LEFT JOIN projects p ON dc.project_id = p.id
        LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
        ORDER BY dc.created_at DESC
        LIMIT ?
    `

    rows, err := s.db.Query(query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var recentDCs []RecentDC
    for rows.Next() {
        var dc RecentDC
        err := rows.Scan(
            &dc.ID,
            &dc.DCNumber,
            &dc.Type,
            &dc.ProjectName,
            &dc.DCDate,
            &dc.Status,
            &dc.ShipToSummary,
            &dc.TotalValue,
        )
        if err != nil {
            return nil, err
        }
        recentDCs = append(recentDCs, dc)
    }

    return recentDCs, nil
}

// Get DCs per month for charting (last N months)
func (s *DashboardService) GetDCsPerMonth(months int) ([]MonthlyCount, error) {
    query := `
        SELECT
            strftime('%Y-%m', dc_date) as month,
            COUNT(*) as count
        FROM delivery_challans
        WHERE dc_date >= date('now', '-' || ? || ' months')
        GROUP BY month
        ORDER BY month DESC
    `

    rows, err := s.db.Query(query, months)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var monthlyCounts []MonthlyCount
    for rows.Next() {
        var mc MonthlyCount
        err := rows.Scan(&mc.Month, &mc.Count)
        if err != nil {
            return nil, err
        }
        monthlyCounts = append(monthlyCounts, mc)
    }

    return monthlyCounts, nil
}

type MonthlyCount struct {
    Month string
    Count int
}
```

**handlers/dashboard_handler.go** (create new)
```go
package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

type DashboardHandler struct {
    dashboardService *DashboardService
}

func NewDashboardHandler(dashboardService *DashboardService) *DashboardHandler {
    return &DashboardHandler{dashboardService: dashboardService}
}

// Show dashboard (home page)
func (h *DashboardHandler) ShowDashboard(c *gin.Context) {
    // Parse date range filter
    dateRange := c.DefaultQuery("range", "current_month")
    var startDate, endDate *time.Time

    switch dateRange {
    case "last_7_days":
        start := time.Now().AddDate(0, 0, -7)
        startDate = &start
    case "last_30_days":
        start := time.Now().AddDate(0, 0, -30)
        startDate = &start
    case "last_3_months":
        start := time.Now().AddDate(0, -3, 0)
        startDate = &start
    case "current_month":
        now := time.Now()
        start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
        end := start.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
        startDate = &start
        endDate = &end
    case "all_time":
        // No date filter
    }

    // Get statistics
    stats, err := h.dashboardService.GetStatistics(startDate, endDate)
    if err != nil {
        c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
        return
    }

    // Get recent DCs
    recentDCs, err := h.dashboardService.GetRecentDCs(15)
    if err != nil {
        c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
        return
    }

    // If HTMX request, return only the stats section
    if c.GetHeader("HX-Request") == "true" {
        c.HTML(http.StatusOK, "dashboard-stats-partial.html", gin.H{
            "Stats":     stats,
            "RecentDCs": recentDCs,
            "Range":     dateRange,
        })
        return
    }

    // Full page render
    c.HTML(http.StatusOK, "dashboard.html", gin.H{
        "Stats":     stats,
        "RecentDCs": recentDCs,
        "Range":     dateRange,
    })
}
```

### Frontend Files

**templates/dashboard.html** (create new)
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard - DC Management Tool</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@2.0.8"></script>
</head>
<body class="bg-gray-50">
    <!-- Navigation -->
    {{ template "nav.html" . }}

    <!-- Dashboard Container -->
    <div class="max-w-7xl mx-auto px-4 py-8">
        <!-- Header -->
        <div class="flex justify-between items-center mb-8">
            <h1 class="text-3xl font-bold text-gray-900">Dashboard</h1>

            <!-- Date Range Filter -->
            <div class="flex gap-2">
                <select
                    name="range"
                    hx-get="/dashboard"
                    hx-target="#dashboard-content"
                    hx-trigger="change"
                    class="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                >
                    <option value="current_month" {{ if eq .Range "current_month" }}selected{{ end }}>Current Month</option>
                    <option value="last_7_days" {{ if eq .Range "last_7_days" }}selected{{ end }}>Last 7 Days</option>
                    <option value="last_30_days" {{ if eq .Range "last_30_days" }}selected{{ end }}>Last 30 Days</option>
                    <option value="last_3_months" {{ if eq .Range "last_3_months" }}selected{{ end }}>Last 3 Months</option>
                    <option value="all_time" {{ if eq .Range "all_time" }}selected{{ end }}>All Time</option>
                </select>
            </div>
        </div>

        <!-- Dashboard Content (HTMX Target) -->
        <div id="dashboard-content">
            {{ template "dashboard-stats-partial.html" . }}
        </div>

        <!-- Quick Actions -->
        <div class="mt-8">
            <h2 class="text-xl font-semibold text-gray-900 mb-4">Quick Actions</h2>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                <a href="/projects/new" class="bg-blue-600 text-white px-6 py-4 rounded-lg hover:bg-blue-700 text-center font-semibold transition">
                    + Create New Project
                </a>
                <a href="/projects" class="bg-indigo-600 text-white px-6 py-4 rounded-lg hover:bg-indigo-700 text-center font-semibold transition">
                    View All Projects
                </a>
                <a href="/dcs" class="bg-purple-600 text-white px-6 py-4 rounded-lg hover:bg-purple-700 text-center font-semibold transition">
                    View All DCs
                </a>
            </div>
        </div>
    </div>
</body>
</html>
```

**templates/dashboard-stats-partial.html** (create new)
```html
<!-- Summary Cards Grid -->
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
    <!-- Total Projects Card -->
    <div class="bg-white rounded-lg shadow p-6">
        <div class="flex items-center justify-between">
            <div>
                <p class="text-sm font-medium text-gray-600">Total Projects</p>
                <p class="text-3xl font-bold text-gray-900 mt-2">{{ .Stats.TotalProjects }}</p>
            </div>
            <div class="bg-blue-100 rounded-full p-3">
                <svg class="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"></path>
                </svg>
            </div>
        </div>
    </div>

    <!-- Total DCs Card -->
    <div class="bg-white rounded-lg shadow p-6">
        <div class="flex items-center justify-between">
            <div>
                <p class="text-sm font-medium text-gray-600">Total DCs Issued</p>
                <p class="text-3xl font-bold text-gray-900 mt-2">{{ .Stats.TotalDCs }}</p>
            </div>
            <div class="bg-green-100 rounded-full p-3">
                <svg class="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                </svg>
            </div>
        </div>
    </div>

    <!-- DCs This Month Card -->
    <div class="bg-white rounded-lg shadow p-6">
        <div class="flex items-center justify-between">
            <div>
                <p class="text-sm font-medium text-gray-600">DCs This Month</p>
                <p class="text-3xl font-bold text-gray-900 mt-2">{{ .Stats.DCsThisMonth }}</p>
            </div>
            <div class="bg-purple-100 rounded-full p-3">
                <svg class="w-8 h-8 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
                </svg>
            </div>
        </div>
    </div>

    <!-- Draft DCs Card -->
    <div class="bg-white rounded-lg shadow p-6">
        <div class="flex items-center justify-between">
            <div>
                <p class="text-sm font-medium text-gray-600">Draft DCs</p>
                <p class="text-3xl font-bold {{ if gt .Stats.DraftDCs 0 }}text-orange-600{{ else }}text-gray-900{{ end }} mt-2">
                    {{ .Stats.DraftDCs }}
                </p>
            </div>
            <div class="bg-orange-100 rounded-full p-3">
                <svg class="w-8 h-8 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path>
                </svg>
            </div>
        </div>
    </div>
</div>

<!-- Stats Breakdown -->
<div class="bg-white rounded-lg shadow p-6 mb-8">
    <h2 class="text-xl font-semibold text-gray-900 mb-4">DC Type Breakdown</h2>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- Transit DCs -->
        <div class="flex items-center justify-between p-4 bg-blue-50 rounded-lg">
            <div>
                <p class="text-sm font-medium text-gray-600">Transit DCs</p>
                <p class="text-2xl font-bold text-blue-600 mt-1">{{ .Stats.TransitDCs }}</p>
            </div>
            <div class="text-right text-sm text-gray-600">
                {{ if gt .Stats.TotalDCs 0 }}
                    {{ printf "%.1f" (div (mul .Stats.TransitDCs 100.0) .Stats.TotalDCs) }}%
                {{ else }}
                    0%
                {{ end }}
            </div>
        </div>

        <!-- Official DCs -->
        <div class="flex items-center justify-between p-4 bg-green-50 rounded-lg">
            <div>
                <p class="text-sm font-medium text-gray-600">Official DCs</p>
                <p class="text-2xl font-bold text-green-600 mt-1">{{ .Stats.OfficialDCs }}</p>
            </div>
            <div class="text-right text-sm text-gray-600">
                {{ if gt .Stats.TotalDCs 0 }}
                    {{ printf "%.1f" (div (mul .Stats.OfficialDCs 100.0) .Stats.TotalDCs) }}%
                {{ else }}
                    0%
                {{ end }}
            </div>
        </div>
    </div>
</div>

<!-- Recent DCs -->
<div class="bg-white rounded-lg shadow overflow-hidden">
    <div class="px-6 py-4 border-b border-gray-200">
        <h2 class="text-xl font-semibold text-gray-900">Recent DCs</h2>
    </div>
    <div class="overflow-x-auto">
        <table class="w-full">
            <thead class="bg-gray-50">
                <tr>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">DC Number</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Project</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Date</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Ship To</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Total Value</th>
                </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
                {{ range .RecentDCs }}
                <tr class="hover:bg-gray-50 cursor-pointer" onclick="window.location='/dcs/{{ .ID }}/{{ if eq .Type "transit" }}transit{{ else }}official{{ end }}'">
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
                    <td class="px-6 py-4">
                        <div class="text-sm text-gray-900">{{ .ProjectName }}</div>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        {{ .DCDate.Format "02/01/2006" }}
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
                    <td colspan="7" class="px-6 py-8 text-center text-gray-500">
                        No DCs found. Create your first DC to get started.
                    </td>
                </tr>
                {{ end }}
            </tbody>
        </table>
    </div>
</div>
```

### Route Configuration

**main.go** (modify)
```go
// Dashboard route (home page)
dashboardHandler := handlers.NewDashboardHandler(dashboardService)

r.GET("/", func(c *gin.Context) {
    c.Redirect(http.StatusFound, "/dashboard")
})
r.GET("/dashboard", dashboardHandler.ShowDashboard)
```

## API Routes/Endpoints

### Route Definitions

| Method | Endpoint | Description | Response |
|--------|----------|-------------|----------|
| GET | `/` | Redirect to dashboard | Redirect |
| GET | `/dashboard` | Show dashboard (home page) | HTML page |
| GET | `/dashboard?range=<range>` | Dashboard with date filter | HTML page or partial |

### Query Parameters

| Parameter | Values | Description |
|-----------|--------|-------------|
| `range` | `current_month` (default) | Current month statistics |
| `range` | `last_7_days` | Last 7 days |
| `range` | `last_30_days` | Last 30 days |
| `range` | `last_3_months` | Last 3 months |
| `range` | `all_time` | All time statistics |

### HTMX Behavior
- When `HX-Request: true` header present, return only `dashboard-stats-partial.html`
- Otherwise, return full `dashboard.html` page

## Database Queries

### Total Projects Count
```sql
SELECT COUNT(*) FROM projects;
```

### Total DCs Count (with optional date filter)
```sql
SELECT COUNT(*) FROM delivery_challans
WHERE dc_date >= ? AND dc_date <= ?;
```

### DCs This Month
```sql
SELECT COUNT(*) FROM delivery_challans
WHERE dc_date >= date('now', 'start of month')
AND dc_date <= date('now', 'start of month', '+1 month', '-1 day');
```

### Draft DCs Count
```sql
SELECT COUNT(*) FROM delivery_challans
WHERE status = 'draft';
```

### Transit DCs Count (with optional date filter)
```sql
SELECT COUNT(*) FROM delivery_challans
WHERE type = 'transit'
AND dc_date >= ? AND dc_date <= ?;
```

### Official DCs Count (with optional date filter)
```sql
SELECT COUNT(*) FROM delivery_challans
WHERE type = 'official'
AND dc_date >= ? AND dc_date <= ?;
```

### Recent DCs List
```sql
SELECT
    dc.id,
    dc.dc_number,
    dc.type,
    p.name as project_name,
    dc.dc_date,
    dc.status,
    COALESCE(sa.city || ', ' || sa.state, 'N/A') as ship_to_summary,
    dc.total_value
FROM delivery_challans dc
LEFT JOIN projects p ON dc.project_id = p.id
LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
ORDER BY dc.created_at DESC
LIMIT 15;
```

### DCs Per Month (for charts)
```sql
SELECT
    strftime('%Y-%m', dc_date) as month,
    COUNT(*) as count
FROM delivery_challans
WHERE dc_date >= date('now', '-6 months')
GROUP BY month
ORDER BY month DESC;
```

## UI Components

### Component Breakdown

1. **Summary Card Component**
   - White background, rounded corners, shadow
   - Icon in colored circle (blue, green, purple, orange)
   - Label text (small, gray)
   - Large number (3xl font)
   - Responsive grid layout

2. **Stats Breakdown Component**
   - Transit DCs: Blue background, percentage
   - Official DCs: Green background, percentage
   - 2-column grid on desktop, stack on mobile

3. **Recent DCs Table Component**
   - Header with gray background
   - Columns: DC Number, Type, Project, Date, Ship To, Status, Total Value
   - Type badges (Transit/Official)
   - Status badges (Draft/Issued)
   - Clickable rows
   - Hover effects
   - Empty state

4. **Date Range Filter Component**
   - Dropdown select
   - HTMX integration
   - Auto-submit on change
   - Preset options

5. **Quick Actions Component**
   - 3-column grid
   - Large buttons with different colors
   - Icons or text
   - Hover effects

### Tailwind Classes Used
- Layout: `max-w-7xl`, `grid`, `grid-cols-1`, `md:grid-cols-2`, `lg:grid-cols-4`
- Cards: `bg-white`, `rounded-lg`, `shadow`, `p-6`
- Typography: `text-3xl`, `font-bold`, `text-gray-900`
- Colors: `bg-blue-100`, `text-blue-600`
- Spacing: `gap-6`, `mb-8`, `px-4`, `py-8`
- Badges: `px-2`, `py-1`, `rounded-full`

## Testing Checklist

### Functional Testing
- [ ] Dashboard loads as home page (/)
- [ ] Dashboard accessible at /dashboard
- [ ] Total Projects count is accurate
- [ ] Total DCs count is accurate
- [ ] DCs This Month count is accurate
- [ ] Draft DCs count is accurate
- [ ] Transit DCs count is accurate
- [ ] Official DCs count is accurate
- [ ] Recent DCs list shows last 15 DCs
- [ ] DC type badges display correctly (Transit/Official)
- [ ] Status badges display correctly (Draft/Issued)
- [ ] Total value shows for Transit DCs
- [ ] Total value shows "—" for Official DCs
- [ ] Percentages calculate correctly for type breakdown

### Date Range Filter Testing
- [ ] Current Month filter works
- [ ] Last 7 Days filter works
- [ ] Last 30 Days filter works
- [ ] Last 3 Months filter works
- [ ] All Time filter works
- [ ] Filter changes update statistics correctly
- [ ] DCs This Month card always shows current month (not affected by filter)
- [ ] Draft DCs count not affected by filter

### HTMX Testing
- [ ] Filter change triggers HTMX request
- [ ] Only stats section refreshes (not full page)
- [ ] HX-Request header detected correctly
- [ ] Partial template renders correctly
- [ ] No page flicker during refresh
- [ ] Loading state visible (optional)

### Navigation Testing
- [ ] Click DC number navigates to correct detail page
- [ ] Click table row navigates to DC detail
- [ ] "Create New Project" button navigates correctly
- [ ] "View All Projects" button navigates correctly
- [ ] "View All DCs" button navigates correctly

### Responsive Testing
- [ ] 4-column grid on desktop (summary cards)
- [ ] 2-column grid on tablet
- [ ] 1-column stack on mobile
- [ ] Table scrolls horizontally on mobile
- [ ] Filter dropdown usable on mobile
- [ ] Quick actions stack on mobile

### Performance Testing
- [ ] Dashboard loads in < 1 second
- [ ] Statistics queries optimized (use COUNT)
- [ ] Recent DCs query uses LIMIT
- [ ] Filter change responds quickly
- [ ] No N+1 query issues

### Edge Cases
- [ ] Zero projects handled gracefully
- [ ] Zero DCs handled gracefully
- [ ] Empty recent DCs shows message
- [ ] Long project names truncate or wrap
- [ ] Long DC numbers display properly
- [ ] Null total values handled (Official DCs)

## Acceptance Criteria

### Must Have
1. ✅ Dashboard accessible at GET /dashboard (home page)
2. ✅ Root path (/) redirects to /dashboard
3. ✅ Summary cards display: Total Projects, Total DCs Issued, DCs This Month, Draft DCs
4. ✅ Each summary card has icon and large number
5. ✅ Draft DCs card has warning color (orange) if count > 0
6. ✅ Stats breakdown shows Transit DCs count and percentage
7. ✅ Stats breakdown shows Official DCs count and percentage
8. ✅ Recent DCs list shows last 15 DCs
9. ✅ Recent DCs table columns: DC Number, Type, Project, Date, Ship To, Status, Total Value
10. ✅ DC Type displayed as badge (Transit = blue, Official = green)
11. ✅ Status displayed as badge (Draft = orange, Issued = green)
12. ✅ Total Value shows amount for Transit DCs, "—" for Official DCs
13. ✅ Date range filter with options: Current Month, Last 7 Days, Last 30 Days, Last 3 Months, All Time
14. ✅ Filter changes trigger HTMX partial refresh
15. ✅ Quick action buttons: Create New Project, View All Projects, View All DCs
16. ✅ Clicking DC number navigates to DC detail page
17. ✅ Responsive design (cards stack on mobile)

### Nice to Have
1. ⭐ Charts: Bar chart for DCs per month
2. ⭐ Charts: Pie chart for Transit vs Official breakdown
3. ⭐ Charts: Line chart for DC trend
4. ⭐ Loading spinner during HTMX refresh
5. ⭐ Animated counters for summary cards
6. ⭐ Recent projects list
7. ⭐ Activity feed/timeline
8. ⭐ Export dashboard as PDF report
9. ⭐ Customizable dashboard widgets
10. ⭐ Dark mode support

### Performance Criteria
- Dashboard load time < 1 second
- Statistics calculation < 200ms
- HTMX partial refresh < 500ms
- Handle 1000+ DCs without performance degradation
- Efficient SQL queries with proper indexes

### Accessibility Criteria
- Semantic HTML structure
- Proper heading hierarchy (h1, h2)
- ARIA labels for icons
- Keyboard navigation
- Screen reader friendly tables

---

## Notes
- Dashboard is the landing page after login
- Statistics refresh on every page load (or implement 5-minute cache)
- DCs This Month always shows current month, regardless of filter
- Draft DCs count is global (not filtered by date)
- Consider pagination if recent DCs list grows beyond 15
- Charts are optional but highly recommended for visualization
- Use Chart.js for lightweight charting (CDN available)
- Future: allow users to customize dashboard layout
- Future: add widgets for recent projects, activity timeline
