# Phase 14: DC Lifecycle (Issue & Lock)

## Overview

This phase implements the complete lifecycle management for both Transit and Official DCs. DCs start as editable Drafts and can be issued to lock all fields. Issued DCs become read-only but can still be deleted with double confirmation. This phase ensures data integrity by preventing accidental modifications while allowing authorized deletion when necessary.

**Key Features:**
- Two states: Draft and Issued
- Draft DCs are fully editable and deletable
- "Issue DC" button locks all fields and changes status
- Issued DCs are read-only (cannot be edited)
- Issued DCs CAN be deleted with double confirmation
- Visual status indicators (badges)
- Different action buttons based on status
- issued_at timestamp tracking
- Hard delete removes DC and frees serial numbers
- DC numbers are NOT reused after deletion

## Prerequisites

- Phase 11 (Transit DC Creation) completed
- Phase 12 (Official DC Creation) completed
- Phase 13 (Serial Number Management) completed
- Database schema with status and issued_at columns
- Serial number deletion cascade configured

## Goals

1. Implement Draft and Issued status workflow
2. Create "Issue DC" action to lock DCs
3. Prevent editing of Issued DCs
4. Allow deletion of Issued DCs with double confirmation
5. Free serial numbers on DC deletion
6. Show appropriate action buttons based on status
7. Display visual status indicators
8. Track issued_at timestamp
9. Ensure DC numbers are not reused
10. Support both Transit and Official DC types

## Detailed Implementation Steps

### Step 1: Verify Database Schema

Ensure `delivery_challans` table has necessary columns:

```sql
-- Verify columns exist
SELECT status, issued_at FROM delivery_challans LIMIT 1;

-- If not, add columns
ALTER TABLE delivery_challans
ADD COLUMN status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'issued'));

ALTER TABLE delivery_challans
ADD COLUMN issued_at DATETIME NULL;

-- Create index for status queries
CREATE INDEX idx_dc_status ON delivery_challans(status);

-- Create index for issued date
CREATE INDEX idx_dc_issued_at ON delivery_challans(issued_at);
```

### Step 2: Backend Route Setup

Add routes in `routes/dc_routes.go`:

```go
// DC Lifecycle Routes
dcRoutes := r.Group("/projects/:project_id/dc")
{
    // Issue DC (change status to issued)
    dcRoutes.POST("/transit/:dc_id/issue", handlers.IssueTransitDC)
    dcRoutes.POST("/official/:dc_id/issue", handlers.IssueOfficialDC)

    // Delete DC (with confirmation)
    dcRoutes.DELETE("/transit/:dc_id", handlers.DeleteTransitDC)
    dcRoutes.DELETE("/official/:dc_id", handlers.DeleteOfficialDC)

    // Unified delete endpoint (works for both types)
    dcRoutes.DELETE("/:dc_id", handlers.DeleteDC)

    // Get DC status
    dcRoutes.GET("/:dc_id/status", handlers.GetDCStatus)
}
```

### Step 3: Handler Implementation

Create `handlers/dc_lifecycle_handler.go`:

**IssueTransitDC / IssueOfficialDC:**
- Verify DC is currently in 'draft' status
- Update status to 'issued'
- Set issued_at timestamp to current time
- Prevent editing after issuance
- Return success response

**DeleteDC:**
- Verify DC exists
- Delete in transaction:
  1. Delete serial numbers (frees them)
  2. Delete line items
  3. Delete transit details (if exists)
  4. Delete DC record
- Return success response
- Note: DC number is NOT reused

```go
package handlers

import (
    "database/sql"
    "net/http"
    "time"
    "github.com/gin-gonic/gin"
)

// IssueTransitDC changes DC status from draft to issued
func IssueTransitDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    issueDC(c, dcID, "transit")
}

// IssueOfficialDC changes DC status from draft to issued
func IssueOfficialDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    issueDC(c, dcID, "official")
}

// Generic issue DC function
func issueDC(c *gin.Context, dcID string, dcType string) {
    db := c.MustGet("db").(*sql.DB)

    // Verify DC exists and is in draft status
    var currentStatus string
    var currentType string
    err := db.QueryRow(`
        SELECT status, dc_type
        FROM delivery_challans
        WHERE id = ?
    `, dcID).Scan(&currentStatus, &currentType)

    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Verify DC type matches
    if currentType != dcType {
        c.JSON(http.StatusBadRequest, gin.H{"error": "DC type mismatch"})
        return
    }

    // Check if already issued
    if currentStatus == "issued" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "DC is already issued"})
        return
    }

    // Update status to issued
    _, err = db.Exec(`
        UPDATE delivery_challans
        SET status = 'issued',
            issued_at = ?,
            updated_at = ?
        WHERE id = ?
    `, time.Now(), time.Now(), dcID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue DC"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "DC issued successfully",
        "status": "issued",
        "issued_at": time.Now().Format("2006-01-02 15:04:05"),
    })
}

// DeleteTransitDC soft check then hard delete
func DeleteTransitDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    deleteDC(c, dcID, "transit")
}

// DeleteOfficialDC soft check then hard delete
func DeleteOfficialDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    deleteDC(c, dcID, "official")
}

// Unified DeleteDC (works for both types)
func DeleteDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    deleteDC(c, dcID, "")
}

// Generic delete DC function
func deleteDC(c *gin.Context, dcID string, dcType string) {
    db := c.MustGet("db").(*sql.DB)

    // Verify DC exists
    var currentType string
    var dcNumber string
    err := db.QueryRow(`
        SELECT dc_type, dc_number
        FROM delivery_challans
        WHERE id = ?
    `, dcID).Scan(&currentType, &dcNumber)

    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Verify DC type if specified
    if dcType != "" && currentType != dcType {
        c.JSON(http.StatusBadRequest, gin.H{"error": "DC type mismatch"})
        return
    }

    // Begin transaction for safe deletion
    tx, err := db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer tx.Rollback()

    // 1. Delete serial numbers (CASCADE DELETE should handle this, but explicit is safer)
    _, err = tx.Exec("DELETE FROM serial_numbers WHERE dc_id = ?", dcID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete serial numbers"})
        return
    }

    // 2. Delete line items
    _, err = tx.Exec("DELETE FROM dc_line_items WHERE dc_id = ?", dcID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete line items"})
        return
    }

    // 3. Delete transit details (if exists - only for Transit DCs)
    _, err = tx.Exec("DELETE FROM dc_transit_details WHERE dc_id = ?", dcID)
    // Ignore error if no transit details (Official DC)

    // 4. Delete main DC record
    result, err := tx.Exec("DELETE FROM delivery_challans WHERE id = ?", dcID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete DC"})
        return
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }

    // Commit transaction
    if err = tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "DC deleted successfully. Serial numbers have been freed for reuse.",
        "dc_number": dcNumber,
    })
}

// GetDCStatus returns current status of a DC
func GetDCStatus(c *gin.Context) {
    dcID := c.Param("dc_id")
    db := c.MustGet("db").(*sql.DB)

    var status string
    var dcType string
    var issuedAt sql.NullTime

    err := db.QueryRow(`
        SELECT status, dc_type, issued_at
        FROM delivery_challans
        WHERE id = ?
    `, dcID).Scan(&status, &dcType, &issuedAt)

    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    response := gin.H{
        "status": status,
        "dc_type": dcType,
        "is_draft": status == "draft",
        "is_issued": status == "issued",
    }

    if issuedAt.Valid {
        response["issued_at"] = issuedAt.Time.Format("2006-01-02 15:04:05")
    }

    c.JSON(http.StatusOK, response)
}
```

### Step 4: Update Edit Handlers

Modify `handlers/transit_dc_handler.go` and `handlers/official_dc_handler.go`:

**EditTransitDC / EditOfficialDC:**

```go
func EditTransitDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    db := c.MustGet("db").(*sql.DB)

    // Verify DC is in draft status
    var status string
    err := db.QueryRow("SELECT status FROM delivery_challans WHERE id = ?", dcID).Scan(&status)
    if err == sql.ErrNoRows {
        c.HTML(http.StatusNotFound, "404.html", gin.H{"error": "DC not found"})
        return
    }
    if err != nil {
        c.HTML(http.StatusInternalServerError, "500.html", gin.H{"error": err.Error()})
        return
    }

    // Prevent editing if issued
    if status != "draft" {
        c.HTML(http.StatusForbidden, "error.html", gin.H{
            "error": "Cannot edit issued DC. Only draft DCs can be edited.",
            "status": status,
        })
        return
    }

    // Proceed with fetching DC data and rendering edit form
    // ... rest of edit logic ...
}
```

**UpdateTransitDC / UpdateOfficialDC:**

```go
func UpdateTransitDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    db := c.MustGet("db").(*sql.DB)

    // CRITICAL: Check status before allowing update
    var status string
    err := db.QueryRow("SELECT status FROM delivery_challans WHERE id = ?", dcID).Scan(&status)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if status != "draft" {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Cannot update issued DC",
            "status": status,
        })
        return
    }

    // Proceed with update logic
    // ... rest of update logic ...
}
```

### Step 5: Frontend - DC Detail/View Page

Create `views/dc/detail.html` (works for both Transit and Official):

```html
<div class="container mx-auto px-4 py-6">
    <!-- DC Header with Status Badge -->
    <div class="bg-white shadow rounded-lg p-6">
        <div class="flex justify-between items-start mb-4">
            <div>
                <h1 class="text-2xl font-bold">{{.DC.DCNumber}}</h1>
                <p class="text-gray-600 mt-1">
                    {{if eq .DC.DCType "transit"}}Transit Delivery Challan{{else}}Official Delivery Challan{{end}}
                </p>
            </div>

            <!-- Status Badge -->
            <div>
                {{if eq .DC.Status "draft"}}
                <span class="px-3 py-1 bg-yellow-100 text-yellow-800 rounded-full text-sm font-semibold">
                    Draft
                </span>
                {{else}}
                <span class="px-3 py-1 bg-green-100 text-green-800 rounded-full text-sm font-semibold">
                    Issued
                </span>
                {{end}}
            </div>
        </div>

        <!-- DC Details -->
        <div class="grid grid-cols-3 gap-4 mb-6">
            <div>
                <label class="text-sm text-gray-600">DC Date</label>
                <p class="font-medium">{{.DC.DCDate.Format "02-Jan-2006"}}</p>
            </div>
            <div>
                <label class="text-sm text-gray-600">Created At</label>
                <p class="font-medium">{{.DC.CreatedAt.Format "02-Jan-2006 15:04"}}</p>
            </div>
            {{if .DC.IssuedAt}}
            <div>
                <label class="text-sm text-gray-600">Issued At</label>
                <p class="font-medium">{{.DC.IssuedAt.Format "02-Jan-2006 15:04"}}</p>
            </div>
            {{end}}
        </div>

        <!-- Action Buttons Based on Status -->
        <div class="flex gap-3 pt-4 border-t">
            {{if eq .DC.Status "draft"}}
                <!-- Draft Actions -->
                <a href="/projects/{{.DC.ProjectID}}/dc/{{.DC.DCType}}/{{.DC.ID}}/edit"
                   class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700">
                    Edit
                </a>
                <button type="button"
                        onclick="issueDC({{.DC.ID}}, '{{.DC.DCType}}')"
                        class="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700">
                    Issue DC
                </button>
                <button type="button"
                        onclick="deleteDC({{.DC.ID}}, '{{.DC.DCType}}', false)"
                        class="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700">
                    Delete
                </button>
            {{else}}
                <!-- Issued Actions -->
                <a href="/projects/{{.DC.ProjectID}}/dc/{{.DC.DCType}}/{{.DC.ID}}/view"
                   class="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700">
                    View
                </a>
                <a href="/projects/{{.DC.ProjectID}}/dc/{{.DC.DCType}}/{{.DC.ID}}/print"
                   class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700">
                    Print
                </a>
                <a href="/projects/{{.DC.ProjectID}}/dc/{{.DC.DCType}}/{{.DC.ID}}/download/pdf"
                   class="px-4 py-2 bg-purple-600 text-white rounded-md hover:bg-purple-700">
                    Download PDF
                </a>
                <a href="/projects/{{.DC.ProjectID}}/dc/{{.DC.DCType}}/{{.DC.ID}}/download/excel"
                   class="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700">
                    Download Excel
                </a>
                <button type="button"
                        onclick="deleteDC({{.DC.ID}}, '{{.DC.DCType}}', true)"
                        class="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700">
                    Delete
                </button>
            {{end}}

            <a href="/projects/{{.DC.ProjectID}}/dc"
               class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50">
                Back to List
            </a>
        </div>
    </div>
</div>

<!-- Double Confirmation Modal for Deletion -->
<div id="deleteModal" class="fixed inset-0 bg-black bg-opacity-50 hidden items-center justify-center z-50">
    <div class="bg-white rounded-lg p-6 max-w-md mx-4">
        <h3 class="text-lg font-bold mb-4">Confirm Deletion</h3>
        <p class="text-gray-600 mb-6" id="deleteMessage">
            Are you sure you want to delete this DC? This action cannot be undone.
            All serial numbers will be freed for reuse.
        </p>
        <div class="flex justify-end gap-3">
            <button type="button"
                    onclick="closeDeleteModal()"
                    class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50">
                Cancel
            </button>
            <button type="button"
                    onclick="confirmDelete()"
                    class="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700">
                Confirm Delete
            </button>
        </div>
    </div>
</div>
```

### Step 6: JavaScript for Actions

Create `static/js/dc_lifecycle.js`:

```javascript
// Issue DC
function issueDC(dcID, dcType) {
    if (!confirm('Issue this DC? Once issued, it cannot be edited.')) {
        return;
    }

    const url = `/projects/${getProjectID()}/dc/${dcType}/${dcID}/issue`;

    fetch(url, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        }
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            alert('DC issued successfully!');
            location.reload(); // Refresh to show new status and actions
        } else {
            alert('Error: ' + (data.error || 'Failed to issue DC'));
        }
    })
    .catch(error => {
        console.error('Error:', error);
        alert('Failed to issue DC. Please try again.');
    });
}

// Delete DC with double confirmation
let pendingDelete = null;

function deleteDC(dcID, dcType, isIssued) {
    const modal = document.getElementById('deleteModal');
    const message = document.getElementById('deleteMessage');

    // Show different message for issued DCs
    if (isIssued) {
        message.textContent = 'This is an ISSUED DC. Are you sure you want to delete it? ' +
                             'This action cannot be undone. All serial numbers will be freed for reuse.';
    } else {
        message.textContent = 'Are you sure you want to delete this DC? ' +
                             'This action cannot be undone. All serial numbers will be freed for reuse.';
    }

    // Store deletion info
    pendingDelete = { dcID, dcType };

    // Show modal
    modal.classList.remove('hidden');
    modal.classList.add('flex');
}

function closeDeleteModal() {
    const modal = document.getElementById('deleteModal');
    modal.classList.add('hidden');
    modal.classList.remove('flex');
    pendingDelete = null;
}

function confirmDelete() {
    if (!pendingDelete) return;

    const { dcID, dcType } = pendingDelete;
    const url = `/projects/${getProjectID()}/dc/${dcType}/${dcID}`;

    fetch(url, {
        method: 'DELETE',
        headers: {
            'Content-Type': 'application/json',
        }
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            alert('DC deleted successfully. Serial numbers have been freed.');
            window.location.href = `/projects/${getProjectID()}/dc`;
        } else {
            alert('Error: ' + (data.error || 'Failed to delete DC'));
            closeDeleteModal();
        }
    })
    .catch(error => {
        console.error('Error:', error);
        alert('Failed to delete DC. Please try again.');
        closeDeleteModal();
    });
}

function getProjectID() {
    // Extract project ID from URL or data attribute
    const match = window.location.pathname.match(/\/projects\/(\d+)/);
    return match ? match[1] : null;
}

// Close modal on Escape key
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        closeDeleteModal();
    }
});

// Close modal on outside click
document.getElementById('deleteModal')?.addEventListener('click', function(e) {
    if (e.target === this) {
        closeDeleteModal();
    }
});
```

### Step 7: DC List View with Status Indicators

Update `views/dc/list.html`:

```html
<div class="container mx-auto px-4 py-6">
    <h1 class="text-2xl font-bold mb-6">Delivery Challans</h1>

    <div class="bg-white shadow rounded-lg overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
                <tr>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">DC Number</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Type</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Date</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Purpose</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
                {{range .DCs}}
                <tr class="hover:bg-gray-50">
                    <td class="px-6 py-4 whitespace-nowrap">
                        <a href="/projects/{{.ProjectID}}/dc/{{.DCType}}/{{.ID}}"
                           class="text-blue-600 hover:underline font-medium">
                            {{.DCNumber}}
                        </a>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        {{if eq .DCType "transit"}}
                        <span class="text-sm text-gray-600">Transit</span>
                        {{else}}
                        <span class="text-sm text-gray-600">Official</span>
                        {{end}}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        {{.DCDate.Format "02-Jan-2006"}}
                    </td>
                    <td class="px-6 py-4 text-sm text-gray-600">
                        {{.Purpose}}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                        {{if eq .Status "draft"}}
                        <span class="px-2 py-1 bg-yellow-100 text-yellow-800 rounded-full text-xs font-semibold">
                            Draft
                        </span>
                        {{else}}
                        <span class="px-2 py-1 bg-green-100 text-green-800 rounded-full text-xs font-semibold">
                            Issued
                        </span>
                        {{end}}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm">
                        {{if eq .Status "draft"}}
                        <a href="/projects/{{.ProjectID}}/dc/{{.DCType}}/{{.ID}}/edit"
                           class="text-blue-600 hover:underline mr-3">Edit</a>
                        <button onclick="issueDC({{.ID}}, '{{.DCType}}')"
                                class="text-green-600 hover:underline mr-3">Issue</button>
                        <button onclick="deleteDC({{.ID}}, '{{.DCType}}', false)"
                                class="text-red-600 hover:underline">Delete</button>
                        {{else}}
                        <a href="/projects/{{.ProjectID}}/dc/{{.DCType}}/{{.ID}}/view"
                           class="text-blue-600 hover:underline mr-3">View</a>
                        <a href="/projects/{{.ProjectID}}/dc/{{.DCType}}/{{.ID}}/print"
                           class="text-blue-600 hover:underline mr-3">Print</a>
                        <button onclick="deleteDC({{.ID}}, '{{.DCType}}', true)"
                                class="text-red-600 hover:underline">Delete</button>
                        {{end}}
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</div>
```

### Step 8: Prevent Direct Edit Access

Add middleware or route guard:

```go
// Middleware to check DC status before allowing edit
func CheckDCStatusMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        dcID := c.Param("dc_id")
        db := c.MustGet("db").(*sql.DB)

        var status string
        err := db.QueryRow("SELECT status FROM delivery_challans WHERE id = ?", dcID).Scan(&status)

        if err == sql.ErrNoRows {
            c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "DC not found"})
            return
        }

        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        if status != "draft" {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "Cannot edit issued DC",
                "status": status,
            })
            return
        }

        c.Next()
    }
}

// Apply middleware to edit routes
dcRoutes.GET("/transit/:dc_id/edit", CheckDCStatusMiddleware(), handlers.EditTransitDC)
dcRoutes.PUT("/transit/:dc_id", CheckDCStatusMiddleware(), handlers.UpdateTransitDC)
dcRoutes.GET("/official/:dc_id/edit", CheckDCStatusMiddleware(), handlers.EditOfficialDC)
dcRoutes.PUT("/official/:dc_id", CheckDCStatusMiddleware(), handlers.UpdateOfficialDC)
```

## Files to Create/Modify

### New Files

1. **handlers/dc_lifecycle_handler.go**
   - IssueTransitDC
   - IssueOfficialDC
   - DeleteDC (unified)
   - DeleteTransitDC
   - DeleteOfficialDC
   - GetDCStatus

2. **middleware/dc_status_check.go**
   - CheckDCStatusMiddleware

3. **views/dc/detail.html**
   - DC detail page with status badge
   - Action buttons based on status
   - Delete confirmation modal

4. **static/js/dc_lifecycle.js**
   - issueDC function
   - deleteDC function with double confirmation
   - Modal handling

### Modified Files

1. **routes/dc_routes.go**
   - Add issue DC routes
   - Add delete DC routes
   - Apply status check middleware to edit routes

2. **handlers/transit_dc_handler.go**
   - Add status check in EditTransitDC
   - Add status check in UpdateTransitDC

3. **handlers/official_dc_handler.go**
   - Add status check in EditOfficialDC
   - Add status check in UpdateOfficialDC

4. **views/dc/list.html**
   - Add status column with badges
   - Add conditional action buttons based on status

5. **database/migrations/002_update_dc_status.sql**
   - Add status column if not exists
   - Add issued_at column if not exists
   - Add indexes

## API Routes/Endpoints

### DC Lifecycle Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/projects/:project_id/dc/transit/:dc_id/issue` | Issue Transit DC (lock) |
| POST | `/projects/:project_id/dc/official/:dc_id/issue` | Issue Official DC (lock) |
| DELETE | `/projects/:project_id/dc/:dc_id` | Delete DC (unified) |
| DELETE | `/projects/:project_id/dc/transit/:dc_id` | Delete Transit DC |
| DELETE | `/projects/:project_id/dc/official/:dc_id` | Delete Official DC |
| GET | `/projects/:project_id/dc/:dc_id/status` | Get DC status |

### Request/Response Examples

**POST /projects/1/dc/transit/42/issue**

Response:
```json
{
  "success": true,
  "message": "DC issued successfully",
  "status": "issued",
  "issued_at": "2026-02-16 14:30:45"
}
```

**DELETE /projects/1/dc/42**

Response:
```json
{
  "success": true,
  "message": "DC deleted successfully. Serial numbers have been freed for reuse.",
  "dc_number": "DC/2024-25/00042"
}
```

**GET /projects/1/dc/42/status**

Response:
```json
{
  "status": "issued",
  "dc_type": "transit",
  "is_draft": false,
  "is_issued": true,
  "issued_at": "2026-02-16 14:30:45"
}
```

## Database Queries

### Issue DC

```sql
UPDATE delivery_challans
SET status = 'issued',
    issued_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'draft';
```

### Check DC Status

```sql
SELECT status, dc_type, issued_at
FROM delivery_challans
WHERE id = ?;
```

### Delete DC (Transaction)

```sql
-- 1. Delete serial numbers
DELETE FROM serial_numbers WHERE dc_id = ?;

-- 2. Delete line items
DELETE FROM dc_line_items WHERE dc_id = ?;

-- 3. Delete transit details (if exists)
DELETE FROM dc_transit_details WHERE dc_id = ?;

-- 4. Delete main DC
DELETE FROM delivery_challans WHERE id = ?;
```

### List DCs with Status

```sql
SELECT
    id, project_id, dc_number, dc_type, dc_date,
    purpose, status, created_at, issued_at
FROM delivery_challans
WHERE project_id = ?
ORDER BY created_at DESC;
```

## UI Components

### 1. Status Badge

```html
<!-- Draft Badge -->
<span class="px-3 py-1 bg-yellow-100 text-yellow-800 rounded-full text-sm font-semibold">
    Draft
</span>

<!-- Issued Badge -->
<span class="px-3 py-1 bg-green-100 text-green-800 rounded-full text-sm font-semibold">
    Issued
</span>
```

### 2. Delete Confirmation Modal

```html
<div id="deleteModal" class="fixed inset-0 bg-black bg-opacity-50 hidden items-center justify-center z-50">
    <div class="bg-white rounded-lg p-6 max-w-md mx-4">
        <h3 class="text-lg font-bold mb-4">Confirm Deletion</h3>
        <p class="text-gray-600 mb-6" id="deleteMessage"></p>
        <div class="flex justify-end gap-3">
            <button onclick="closeDeleteModal()" class="btn-secondary">Cancel</button>
            <button onclick="confirmDelete()" class="btn-danger">Confirm Delete</button>
        </div>
    </div>
</div>
```

### 3. Action Buttons (Draft)

```html
<button onclick="issueDC(42, 'transit')" class="btn-success">Issue DC</button>
<a href="/projects/1/dc/transit/42/edit" class="btn-primary">Edit</a>
<button onclick="deleteDC(42, 'transit', false)" class="btn-danger">Delete</button>
```

### 4. Action Buttons (Issued)

```html
<a href="/projects/1/dc/transit/42/view" class="btn-secondary">View</a>
<a href="/projects/1/dc/transit/42/print" class="btn-primary">Print</a>
<button onclick="deleteDC(42, 'transit', true)" class="btn-danger">Delete</button>
```

## Testing Checklist

### Functional Testing

- [ ] Create Draft DC (Transit and Official)
- [ ] Verify Draft status badge appears
- [ ] Click "Issue DC" button
- [ ] Verify status changes to Issued
- [ ] Verify issued_at timestamp is set
- [ ] Verify Edit button disappears after issue
- [ ] Attempt to access edit URL for Issued DC (should be blocked)
- [ ] Verify View/Print buttons appear for Issued DC
- [ ] Click Delete on Draft DC (single confirmation)
- [ ] Click Delete on Issued DC (double confirmation modal appears)
- [ ] Confirm deletion and verify DC is deleted
- [ ] Verify serial numbers are freed after deletion
- [ ] Verify DC number is NOT reused after deletion
- [ ] Test status API endpoint

### Edge Cases

- [ ] Attempt to issue already issued DC (should fail)
- [ ] Attempt to edit issued DC via API (should return 403)
- [ ] Attempt to delete non-existent DC (should return 404)
- [ ] Test transaction rollback on delete failure
- [ ] Test concurrent issue attempts
- [ ] Test deleting DC with 1000+ serial numbers
- [ ] Verify cascade delete works correctly

### UI/UX Testing

- [ ] Verify status badge colors (yellow for draft, green for issued)
- [ ] Verify action buttons change based on status
- [ ] Test delete modal open/close
- [ ] Test modal close on Escape key
- [ ] Test modal close on outside click
- [ ] Verify confirmation message changes for issued vs draft
- [ ] Test responsive design of modal
- [ ] Verify loading states during issue/delete

### Integration Testing

- [ ] Issue DC and verify cannot edit (Phase 11/12 integration)
- [ ] Delete DC and verify serials freed (Phase 13 integration)
- [ ] Verify DC list shows correct status
- [ ] Test with both Transit and Official DCs
- [ ] Verify middleware blocks edit access for issued DCs

## Acceptance Criteria

### Must Have

1. **Status Management**
   - ✅ DCs start in 'draft' status
   - ✅ "Issue DC" button changes status to 'issued'
   - ✅ issued_at timestamp recorded accurately
   - ✅ Status persists in database

2. **Edit Protection**
   - ✅ Draft DCs are fully editable
   - ✅ Issued DCs cannot be edited
   - ✅ Edit routes blocked for issued DCs (403 error)
   - ✅ Edit button hidden for issued DCs

3. **Deletion**
   - ✅ Draft DCs can be deleted with single confirmation
   - ✅ Issued DCs can be deleted with double confirmation
   - ✅ Serial numbers freed on deletion
   - ✅ All related records deleted in transaction
   - ✅ DC number NOT reused

4. **UI Indicators**
   - ✅ Draft badge (yellow)
   - ✅ Issued badge (green)
   - ✅ Conditional action buttons
   - ✅ Delete confirmation modal

5. **Data Integrity**
   - ✅ Transaction safety for deletions
   - ✅ Status validation before operations
   - ✅ Proper error handling

### Should Have

1. **User Experience**
   - ✅ Clear confirmation messages
   - ✅ Loading states during operations
   - ✅ Success/error notifications
   - ✅ Keyboard shortcuts (Escape to close modal)

2. **Performance**
   - ✅ Status checks are fast (indexed queries)
   - ✅ Deletion handles large DC efficiently

### Nice to Have

1. **Advanced Features**
   - ⭕ Bulk issue multiple DCs
   - ⭕ Bulk delete multiple DCs
   - ⭕ Audit log for status changes
   - ⭕ Email notification on DC issue

---

## Notes

- Status is a simple two-state system: draft and issued
- No "void" or "cancelled" status (delete instead)
- Serial numbers are immediately freed on deletion
- DC numbers follow continuous sequence despite deletions
- Both Transit and Official DCs share same lifecycle
- issued_at is important for reporting and auditing

## Dependencies

- **Phase 11:** Transit DC creation
- **Phase 12:** Official DC creation
- **Phase 13:** Serial number management (for freeing serials)
- **Database:** status and issued_at columns in delivery_challans table

## Next Steps

After Phase 14 completion:
1. Proceed to Phase 15 (Transit DC View & Print Layout)
2. Implement PDF generation for issued DCs
3. Implement Excel export for issued DCs
4. Add email functionality to send DCs to customers
