# Phase 13: Serial Number Management & Validation

## Overview

This phase implements comprehensive serial number tracking and validation for both Transit and Official DCs. It ensures serial number uniqueness within a project, prevents duplicate usage, and provides real-time validation during DC creation. Serial numbers can be reused only after the associated DC is deleted.

**Key Features:**
- Dedicated serial_numbers table with indexed lookups
- UNIQUE constraint: (project_id, serial_number) within same project
- Real-time validation during serial number input
- Visual feedback (red highlight, error messages) for duplicates
- Bulk insert/delete operations on DC save/delete
- Duplicate detection within same DC
- Barcode scanner optimized with debouncing
- Serial number freeing on DC deletion
- Whitespace trimming and empty line filtering

## Prerequisites

- Phase 11 (Transit DC Creation) completed
- Phase 12 (Official DC Creation) completed
- Database schema for delivery_challans and dc_line_items tables
- HTMX configured for real-time validation
- JavaScript for client-side duplicate detection

## Goals

1. Create serial_numbers table with proper indexing
2. Implement UNIQUE constraint per project
3. Provide real-time serial number validation
4. Show inline validation errors during input
5. Prevent saving DCs with duplicate serial numbers
6. Auto-insert serial numbers on DC save
7. Auto-delete serial numbers on DC deletion
8. Handle barcode scanner input efficiently
9. Support serial number reuse after DC deletion
10. Validate serial numbers across both Transit and Official DCs

## Detailed Implementation Steps

### Step 1: Database Schema for Serial Numbers

Create `database/migrations/004_create_serial_numbers_table.sql`:

```sql
-- Serial Numbers Table
CREATE TABLE IF NOT EXISTS serial_numbers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    dc_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    serial_number VARCHAR(255) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (dc_id) REFERENCES delivery_challans(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT
);

-- UNIQUE constraint: Same serial cannot exist twice in same project
CREATE UNIQUE INDEX idx_serial_project_unique
ON serial_numbers(project_id, serial_number);

-- Index for fast DC lookup
CREATE INDEX idx_serial_dc_id
ON serial_numbers(dc_id);

-- Index for fast product lookup
CREATE INDEX idx_serial_product_id
ON serial_numbers(product_id);

-- Composite index for validation queries
CREATE INDEX idx_serial_project_product
ON serial_numbers(project_id, product_id, serial_number);
```

**Key Design Decisions:**
- **UNIQUE constraint on (project_id, serial_number):** Same serial can exist in different projects but NOT within the same project
- **CASCADE DELETE on dc_id:** When a DC is deleted, all serial numbers are freed automatically
- **RESTRICT DELETE on product_id:** Cannot delete a product if it has serial numbers (data integrity)
- **Indexes optimized for:**
  - Validation queries (check if serial exists in project)
  - DC deletion (bulk delete all serials for a DC)
  - Product reporting (all serials for a product)

### Step 2: Backend Route Setup

Add routes in `routes/dc_routes.go`:

```go
// Serial Number Validation Routes
apiRoutes := r.Group("/api")
{
    // Real-time validation endpoint
    apiRoutes.POST("/dc/validate-serials", handlers.ValidateSerialNumbers)

    // Batch validation for all line items
    apiRoutes.POST("/dc/validate-all-serials", handlers.ValidateAllSerials)

    // Check serial number history/usage
    apiRoutes.GET("/projects/:project_id/serials/:serial_number/history", handlers.GetSerialHistory)
}
```

### Step 3: Handler Implementation

Create `handlers/serial_number_handler.go`:

**ValidateSerialNumbers (Real-time HTMX endpoint):**
- Accepts: project_id, product_id, serial_numbers (newline-separated string)
- Parse and sanitize serial numbers
- Check each serial against serial_numbers table for duplicates within project
- Check for duplicates within the submitted list itself
- Return JSON with validation results
- Include which DC currently uses each duplicate serial

**ValidateAllSerials (Pre-save batch validation):**
- Accepts all line items from form
- Validate ALL serial numbers across ALL line items
- Return comprehensive validation report
- Used before final DC save

**GetSerialHistory:**
- Fetch all DC usage history for a specific serial number
- Show when it was used, in which DC, and current status
- Useful for troubleshooting

```go
package handlers

import (
    "database/sql"
    "net/http"
    "strings"
    "github.com/gin-gonic/gin"
)

type SerialValidationRequest struct {
    ProjectID     int64    `json:"project_id"`
    ProductID     int64    `json:"product_id"`
    SerialNumbers string   `json:"serial_numbers"` // Newline-separated
    ExcludeDCID   *int64   `json:"exclude_dc_id"`  // For edit mode
}

type SerialValidationResult struct {
    Valid            bool                    `json:"valid"`
    DuplicateInDB    []SerialConflict        `json:"duplicate_in_db"`
    DuplicateInInput []string                `json:"duplicate_in_input"`
    TotalCount       int                     `json:"total_count"`
}

type SerialConflict struct {
    SerialNumber string `json:"serial_number"`
    ExistingDCID int64  `json:"existing_dc_id"`
    DCNumber     string `json:"dc_number"`
    DCStatus     string `json:"dc_status"`
    ProductName  string `json:"product_name"`
}

func ValidateSerialNumbers(c *gin.Context) {
    var req SerialValidationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    db := c.MustGet("db").(*sql.DB)

    // Parse and sanitize serial numbers
    serials := parseSerialNumbers(req.SerialNumbers)

    // Check for duplicates within input
    duplicatesInInput := findDuplicatesInList(serials)

    // Check for duplicates in database
    duplicatesInDB, err := checkSerialsInDatabase(db, req.ProjectID, req.ProductID, serials, req.ExcludeDCID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    result := SerialValidationResult{
        Valid:            len(duplicatesInDB) == 0 && len(duplicatesInInput) == 0,
        DuplicateInDB:    duplicatesInDB,
        DuplicateInInput: duplicatesInInput,
        TotalCount:       len(serials),
    }

    c.JSON(http.StatusOK, result)
}

func parseSerialNumbers(input string) []string {
    lines := strings.Split(input, "\n")
    serials := make([]string, 0, len(lines))

    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" {
            serials = append(serials, trimmed)
        }
    }

    return serials
}

func findDuplicatesInList(serials []string) []string {
    seen := make(map[string]bool)
    duplicates := make(map[string]bool)

    for _, serial := range serials {
        if seen[serial] {
            duplicates[serial] = true
        } else {
            seen[serial] = true
        }
    }

    result := make([]string, 0, len(duplicates))
    for serial := range duplicates {
        result = append(result, serial)
    }

    return result
}

func checkSerialsInDatabase(db *sql.DB, projectID, productID int64, serials []string, excludeDCID *int64) ([]SerialConflict, error) {
    if len(serials) == 0 {
        return []SerialConflict{}, nil
    }

    // Build placeholders for IN clause
    placeholders := make([]string, len(serials))
    args := make([]interface{}, 0, len(serials)+2)

    args = append(args, projectID)
    for i, serial := range serials {
        placeholders[i] = "?"
        args = append(args, serial)
    }

    query := `
        SELECT
            sn.serial_number,
            sn.dc_id,
            dc.dc_number,
            dc.status,
            p.name as product_name
        FROM serial_numbers sn
        INNER JOIN delivery_challans dc ON sn.dc_id = dc.id
        INNER JOIN products p ON sn.product_id = p.id
        WHERE sn.project_id = ?
          AND sn.serial_number IN (` + strings.Join(placeholders, ",") + `)`

    // Exclude current DC if editing
    if excludeDCID != nil {
        query += " AND sn.dc_id != ?"
        args = append(args, *excludeDCID)
    }

    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    conflicts := []SerialConflict{}
    for rows.Next() {
        var conflict SerialConflict
        err := rows.Scan(
            &conflict.SerialNumber,
            &conflict.ExistingDCID,
            &conflict.DCNumber,
            &conflict.DCStatus,
            &conflict.ProductName,
        )
        if err != nil {
            return nil, err
        }
        conflicts = append(conflicts, conflict)
    }

    return conflicts, nil
}

func GetSerialHistory(c *gin.Context) {
    projectID := c.Param("project_id")
    serialNumber := c.Param("serial_number")

    db := c.MustGet("db").(*sql.DB)

    query := `
        SELECT
            sn.id,
            sn.dc_id,
            dc.dc_number,
            dc.dc_type,
            dc.dc_date,
            dc.status,
            p.name as product_name,
            sn.created_at
        FROM serial_numbers sn
        INNER JOIN delivery_challans dc ON sn.dc_id = dc.id
        INNER JOIN products p ON sn.product_id = p.id
        WHERE sn.project_id = ? AND sn.serial_number = ?
        ORDER BY sn.created_at DESC
    `

    rows, err := db.Query(query, projectID, serialNumber)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    history := []map[string]interface{}{}
    for rows.Next() {
        var record map[string]interface{}
        // Scan and append to history
        history = append(history, record)
    }

    c.JSON(http.StatusOK, gin.H{
        "serial_number": serialNumber,
        "history": history,
    })
}
```

### Step 4: Database Operations for Serial Numbers

Add methods to `models/serial_number.go`:

```go
package models

import (
    "database/sql"
    "time"
)

type SerialNumber struct {
    ID           int64
    ProjectID    int64
    DCID         int64
    ProductID    int64
    SerialNumber string
    CreatedAt    time.Time
}

// BulkInsertSerialNumbers inserts all serial numbers for a DC in a transaction
func BulkInsertSerialNumbers(db *sql.DB, dcID, projectID int64, lineItems []DCLineItem) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.Prepare(`
        INSERT INTO serial_numbers (project_id, dc_id, product_id, serial_number, created_at)
        VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, lineItem := range lineItems {
        for _, serial := range lineItem.SerialNumbers {
            trimmed := strings.TrimSpace(serial)
            if trimmed == "" {
                continue
            }

            _, err = stmt.Exec(projectID, dcID, lineItem.ProductID, trimmed)
            if err != nil {
                // Check if it's a UNIQUE constraint violation
                if strings.Contains(err.Error(), "UNIQUE constraint failed") {
                    return fmt.Errorf("serial number '%s' already exists in this project", trimmed)
                }
                return err
            }
        }
    }

    return tx.Commit()
}

// DeleteSerialNumbersByDC removes all serial numbers associated with a DC
func DeleteSerialNumbersByDC(db *sql.DB, dcID int64) error {
    query := `DELETE FROM serial_numbers WHERE dc_id = ?`
    _, err := db.Exec(query, dcID)
    return err
}

// GetSerialNumbersByDC fetches all serial numbers for a DC (for edit mode)
func GetSerialNumbersByDC(db *sql.DB, dcID int64) (map[int64][]string, error) {
    query := `
        SELECT product_id, serial_number
        FROM serial_numbers
        WHERE dc_id = ?
        ORDER BY id
    `

    rows, err := db.Query(query, dcID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Map: product_id -> []serial_numbers
    serialsByProduct := make(map[int64][]string)

    for rows.Next() {
        var productID int64
        var serialNumber string

        err := rows.Scan(&productID, &serialNumber)
        if err != nil {
            return nil, err
        }

        serialsByProduct[productID] = append(serialsByProduct[productID], serialNumber)
    }

    return serialsByProduct, nil
}

// CheckSerialExists checks if a serial number exists in a project
func CheckSerialExists(db *sql.DB, projectID int64, serialNumber string, excludeDCID *int64) (bool, error) {
    query := `
        SELECT COUNT(*)
        FROM serial_numbers
        WHERE project_id = ? AND serial_number = ?
    `
    args := []interface{}{projectID, serialNumber}

    if excludeDCID != nil {
        query += " AND dc_id != ?"
        args = append(args, *excludeDCID)
    }

    var count int
    err := db.QueryRow(query, args...).Scan(&count)
    if err != nil {
        return false, err
    }

    return count > 0, nil
}
```

### Step 5: Update DC Creation Handlers

Modify `handlers/transit_dc_handler.go` and `handlers/official_dc_handler.go`:

**CreateTransitDC / CreateOfficialDC:**

```go
func CreateTransitDC(c *gin.Context) {
    var req TransitDCRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    db := c.MustGet("db").(*sql.DB)

    // Validate DC data
    errors := validateTransitDC(&req.DC, &req.TransitDetails, req.LineItems)
    if len(errors) > 0 {
        c.JSON(http.StatusBadRequest, gin.H{"errors": errors})
        return
    }

    // CRITICAL: Validate serial numbers BEFORE creating DC
    serialErrors := validateAllSerialNumbers(db, req.DC.ProjectID, req.LineItems, nil)
    if len(serialErrors) > 0 {
        c.JSON(http.StatusBadRequest, gin.H{"errors": serialErrors})
        return
    }

    // Begin transaction
    tx, err := db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer tx.Rollback()

    // Insert DC record
    dcID, err := insertDeliveryChallan(tx, &req.DC)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Insert transit details
    err = insertTransitDetails(tx, dcID, &req.TransitDetails)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Insert line items
    err = insertLineItems(tx, dcID, req.LineItems)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // CRITICAL: Insert serial numbers
    err = insertSerialNumbers(tx, dcID, req.DC.ProjectID, req.LineItems)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Commit transaction
    if err = tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "dc_id": dcID,
        "dc_number": req.DC.DCNumber,
        "redirect": fmt.Sprintf("/projects/%d/dc/transit/%d", req.DC.ProjectID, dcID),
    })
}

func validateAllSerialNumbers(db *sql.DB, projectID int64, lineItems []DCLineItem, excludeDCID *int64) []string {
    errors := []string{}

    for i, item := range lineItems {
        // Check for duplicates within same line
        seen := make(map[string]bool)
        for _, serial := range item.SerialNumbers {
            trimmed := strings.TrimSpace(serial)
            if trimmed == "" {
                continue
            }

            if seen[trimmed] {
                errors = append(errors, fmt.Sprintf("Line %d: Serial '%s' appears multiple times in same line", i+1, trimmed))
                continue
            }
            seen[trimmed] = true

            // Check against database
            exists, err := models.CheckSerialExists(db, projectID, trimmed, excludeDCID)
            if err != nil {
                errors = append(errors, fmt.Sprintf("Error validating serial '%s': %v", trimmed, err))
                continue
            }

            if exists {
                errors = append(errors, fmt.Sprintf("Line %d: Serial '%s' is already used in another DC", i+1, trimmed))
            }
        }
    }

    return errors
}
```

**UpdateTransitDC / UpdateOfficialDC:**

```go
func UpdateTransitDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    var req TransitDCRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    db := c.MustGet("db").(*sql.DB)

    // Verify DC is in draft status
    var status string
    err := db.QueryRow("SELECT status FROM delivery_challans WHERE id = ?", dcID).Scan(&status)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
        return
    }
    if status != "draft" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Cannot edit issued DC"})
        return
    }

    // Validate serial numbers (excluding current DC)
    dcIDInt, _ := strconv.ParseInt(dcID, 10, 64)
    serialErrors := validateAllSerialNumbers(db, req.DC.ProjectID, req.LineItems, &dcIDInt)
    if len(serialErrors) > 0 {
        c.JSON(http.StatusBadRequest, gin.H{"errors": serialErrors})
        return
    }

    // Begin transaction
    tx, err := db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer tx.Rollback()

    // Update DC record
    err = updateDeliveryChallan(tx, dcIDInt, &req.DC)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Update transit details
    err = updateTransitDetails(tx, dcIDInt, &req.TransitDetails)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Delete old line items and serial numbers
    _, err = tx.Exec("DELETE FROM dc_line_items WHERE dc_id = ?", dcIDInt)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    _, err = tx.Exec("DELETE FROM serial_numbers WHERE dc_id = ?", dcIDInt)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Re-insert line items and serial numbers
    err = insertLineItems(tx, dcIDInt, req.LineItems)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    err = insertSerialNumbers(tx, dcIDInt, req.DC.ProjectID, req.LineItems)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Commit transaction
    if err = tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "DC updated successfully",
    })
}
```

### Step 6: Frontend Real-time Validation

Update `static/js/dc_calculations.js` to add real-time serial validation:

```javascript
// Real-time serial number validation
document.addEventListener('DOMContentLoaded', function() {
    const serialInputs = document.querySelectorAll('.serial-input');

    serialInputs.forEach(textarea => {
        // Debounce validation to avoid excessive API calls during rapid scanning
        let validationTimeout;

        textarea.addEventListener('input', function() {
            const row = this.closest('tr');
            const lineIndex = this.getAttribute('data-line-index');
            const productId = row.querySelector('input[name*=".product_id"]').value;
            const projectId = document.querySelector('input[name="project_id"]').value;
            const serialNumbers = this.value;

            // Clear previous validation state
            clearValidationUI(row);

            // Update quantity
            updateQuantity(row, serialNumbers);

            // Debounce validation
            clearTimeout(validationTimeout);
            validationTimeout = setTimeout(() => {
                validateSerialNumbers(projectId, productId, serialNumbers, row, lineIndex);
            }, 500); // Wait 500ms after user stops typing/scanning
        });
    });
});

function updateQuantity(row, serialNumbers) {
    const quantityInput = row.querySelector('.quantity-display');
    const serials = serialNumbers
        .split('\n')
        .map(s => s.trim())
        .filter(s => s.length > 0);

    quantityInput.value = serials.length;
}

function validateSerialNumbers(projectId, productId, serialNumbers, row, lineIndex) {
    // Get exclude_dc_id if in edit mode
    const excludeDCID = document.querySelector('input[name="dc_id"]')?.value || null;

    fetch('/api/dc/validate-serials', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            project_id: parseInt(projectId),
            product_id: parseInt(productId),
            serial_numbers: serialNumbers,
            exclude_dc_id: excludeDCID ? parseInt(excludeDCID) : null,
        })
    })
    .then(response => response.json())
    .then(result => {
        if (!result.valid) {
            showValidationErrors(row, lineIndex, result);
        } else {
            showValidationSuccess(row);
        }
    })
    .catch(error => {
        console.error('Validation error:', error);
    });
}

function showValidationErrors(row, lineIndex, result) {
    const textarea = row.querySelector('.serial-input');
    const errorContainer = getOrCreateErrorContainer(row);

    // Highlight textarea
    textarea.classList.add('border-red-500', 'bg-red-50');
    textarea.classList.remove('border-green-500', 'bg-green-50');

    // Build error message
    let errorMessage = '';

    if (result.duplicate_in_input && result.duplicate_in_input.length > 0) {
        errorMessage += `<div class="text-red-600 text-sm mb-1">
            <strong>Duplicates in this line:</strong> ${result.duplicate_in_input.join(', ')}
        </div>`;
    }

    if (result.duplicate_in_db && result.duplicate_in_db.length > 0) {
        errorMessage += `<div class="text-red-600 text-sm">
            <strong>Already used in other DCs:</strong>
            <ul class="list-disc list-inside">`;

        result.duplicate_in_db.forEach(conflict => {
            errorMessage += `<li>${conflict.serial_number} - Used in ${conflict.dc_number} (${conflict.dc_status})</li>`;
        });

        errorMessage += `</ul></div>`;
    }

    errorContainer.innerHTML = errorMessage;
    errorContainer.classList.remove('hidden');
}

function showValidationSuccess(row) {
    const textarea = row.querySelector('.serial-input');
    const errorContainer = getOrCreateErrorContainer(row);

    // Highlight textarea as valid
    textarea.classList.remove('border-red-500', 'bg-red-50');
    textarea.classList.add('border-green-500', 'bg-green-50');

    // Hide error messages
    errorContainer.classList.add('hidden');
}

function clearValidationUI(row) {
    const textarea = row.querySelector('.serial-input');
    const errorContainer = getOrCreateErrorContainer(row);

    textarea.classList.remove('border-red-500', 'bg-red-50', 'border-green-500', 'bg-green-50');
    errorContainer.classList.add('hidden');
}

function getOrCreateErrorContainer(row) {
    let errorContainer = row.querySelector('.serial-validation-errors');

    if (!errorContainer) {
        const serialCell = row.querySelector('.serial-input').closest('td');
        errorContainer = document.createElement('div');
        errorContainer.className = 'serial-validation-errors mt-2 hidden';
        serialCell.appendChild(errorContainer);
    }

    return errorContainer;
}

// Prevent form submission if there are validation errors
document.addEventListener('submit', function(e) {
    const form = e.target;
    if (form.querySelector('.serial-input.border-red-500')) {
        e.preventDefault();
        alert('Please fix serial number validation errors before submitting.');
        return false;
    }
});
```

### Step 7: Update DC Deletion Handler

Modify deletion handlers to free serial numbers:

```go
func DeleteDC(c *gin.Context) {
    dcID := c.Param("dc_id")
    db := c.MustGet("db").(*sql.DB)

    // Begin transaction
    tx, err := db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer tx.Rollback()

    // Delete serial numbers (frees them for reuse)
    _, err = tx.Exec("DELETE FROM serial_numbers WHERE dc_id = ?", dcID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete serial numbers"})
        return
    }

    // Delete line items
    _, err = tx.Exec("DELETE FROM dc_line_items WHERE dc_id = ?", dcID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete line items"})
        return
    }

    // Delete transit details (if exists)
    _, err = tx.Exec("DELETE FROM dc_transit_details WHERE dc_id = ?", dcID)
    // Ignore error if no transit details (Official DC)

    // Delete DC
    _, err = tx.Exec("DELETE FROM delivery_challans WHERE id = ?", dcID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete DC"})
        return
    }

    // Commit transaction
    if err = tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "DC and all associated serial numbers deleted successfully",
    })
}
```

### Step 8: Barcode Scanner Optimization

Add barcode scanner detection and optimization:

```javascript
// Barcode scanner detection
let barcodeBuffer = '';
let barcodeTimeout;
const BARCODE_SCAN_THRESHOLD = 100; // ms between keystrokes for scanner detection

document.addEventListener('keydown', function(e) {
    // Detect if input is coming from a barcode scanner (rapid keystrokes)
    if (document.activeElement.classList.contains('serial-input')) {
        const now = Date.now();

        if (barcodeTimeout) {
            clearTimeout(barcodeTimeout);
        }

        if (e.key === 'Enter') {
            // Scanner typically sends Enter at the end
            e.preventDefault();

            // Move to next line in textarea
            const textarea = document.activeElement;
            const cursorPos = textarea.selectionStart;
            const textBefore = textarea.value.substring(0, cursorPos);
            const textAfter = textarea.value.substring(cursorPos);
            textarea.value = textBefore + '\n' + textAfter;
            textarea.selectionStart = textarea.selectionEnd = cursorPos + 1;

            // Trigger validation after short delay
            textarea.dispatchEvent(new Event('input'));
        }

        barcodeTimeout = setTimeout(() => {
            barcodeBuffer = '';
        }, BARCODE_SCAN_THRESHOLD);
    }
});
```

## Files to Create/Modify

### New Files

1. **database/migrations/004_create_serial_numbers_table.sql**
   - CREATE TABLE serial_numbers
   - UNIQUE INDEX on (project_id, serial_number)
   - Indexes for performance

2. **handlers/serial_number_handler.go**
   - ValidateSerialNumbers (HTMX endpoint)
   - ValidateAllSerials (batch validation)
   - GetSerialHistory
   - Helper functions

3. **models/serial_number.go**
   - SerialNumber struct
   - BulkInsertSerialNumbers
   - DeleteSerialNumbersByDC
   - GetSerialNumbersByDC
   - CheckSerialExists

4. **static/js/serial_validation.js**
   - Real-time validation logic
   - UI feedback (red/green highlighting)
   - Error message display
   - Barcode scanner optimization

### Modified Files

1. **handlers/transit_dc_handler.go**
   - Add serial number validation in CreateTransitDC
   - Add serial number validation in UpdateTransitDC
   - Call BulkInsertSerialNumbers after DC creation
   - Call DeleteSerialNumbersByDC before update

2. **handlers/official_dc_handler.go**
   - Same serial number validation as Transit DC
   - Reuse serial number functions

3. **routes/dc_routes.go**
   - Add /api/dc/validate-serials endpoint
   - Add /api/projects/:project_id/serials/:serial_number/history endpoint

4. **views/dc/transit_create.html**
   - Add error container div for validation messages
   - Include serial_validation.js script

5. **views/dc/official_create.html**
   - Same validation UI as Transit DC

6. **static/css/styles.css**
   - Add validation state styles (red/green borders)
   - Add error message styles

## API Routes/Endpoints

### Serial Number Validation

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/dc/validate-serials` | Real-time validation of serial numbers |
| POST | `/api/dc/validate-all-serials` | Batch validation before save |
| GET | `/api/projects/:project_id/serials/:serial_number/history` | Get usage history of a serial number |

### Request/Response Examples

**POST /api/dc/validate-serials**

Request:
```json
{
  "project_id": 1,
  "product_id": 10,
  "serial_numbers": "SN001\nSN002\nSN001\nSN003",
  "exclude_dc_id": null
}
```

Response (with duplicates):
```json
{
  "valid": false,
  "duplicate_in_input": ["SN001"],
  "duplicate_in_db": [
    {
      "serial_number": "SN003",
      "existing_dc_id": 42,
      "dc_number": "DC/2024-25/00042",
      "dc_status": "issued",
      "product_name": "Smart Lock Pro"
    }
  ],
  "total_count": 4
}
```

Response (all valid):
```json
{
  "valid": true,
  "duplicate_in_input": [],
  "duplicate_in_db": [],
  "total_count": 3
}
```

## Database Queries

### Insert Serial Numbers (Bulk)

```sql
INSERT INTO serial_numbers (project_id, dc_id, product_id, serial_number, created_at)
VALUES
    (1, 42, 10, 'SN001', CURRENT_TIMESTAMP),
    (1, 42, 10, 'SN002', CURRENT_TIMESTAMP),
    (1, 42, 10, 'SN003', CURRENT_TIMESTAMP);
```

### Check Serial Uniqueness

```sql
SELECT COUNT(*)
FROM serial_numbers
WHERE project_id = ?
  AND serial_number = ?
  AND dc_id != ?; -- Optional: exclude current DC in edit mode
```

### Find Duplicate Serials

```sql
SELECT
    sn.serial_number,
    sn.dc_id,
    dc.dc_number,
    dc.status,
    p.name as product_name
FROM serial_numbers sn
INNER JOIN delivery_challans dc ON sn.dc_id = dc.id
INNER JOIN products p ON sn.product_id = p.id
WHERE sn.project_id = ?
  AND sn.serial_number IN (?, ?, ?)
  AND sn.dc_id != ?; -- Optional: exclude current DC
```

### Delete Serial Numbers on DC Deletion

```sql
DELETE FROM serial_numbers
WHERE dc_id = ?;
```

### Get Serial Numbers for DC (Edit Mode)

```sql
SELECT product_id, serial_number
FROM serial_numbers
WHERE dc_id = ?
ORDER BY id;
```

### Serial Number Usage History

```sql
SELECT
    sn.id,
    sn.dc_id,
    dc.dc_number,
    dc.dc_type,
    dc.dc_date,
    dc.status,
    p.name as product_name,
    sn.created_at
FROM serial_numbers sn
INNER JOIN delivery_challans dc ON sn.dc_id = dc.id
INNER JOIN products p ON sn.product_id = p.id
WHERE sn.project_id = ?
  AND sn.serial_number = ?
ORDER BY sn.created_at DESC;
```

## UI Components

### 1. Serial Number Textarea with Validation

```html
<textarea
    class="serial-input font-mono text-xs w-full px-2 py-1 border border-gray-300 rounded"
    rows="3"
    placeholder="Scan serial numbers (one per line)"
    data-line-index="{{$index}}"
></textarea>
<div class="serial-validation-errors mt-2 hidden">
    <!-- Validation errors appear here -->
</div>
```

**Validation States:**
- **Default:** Gray border
- **Valid:** Green border, light green background
- **Invalid:** Red border, light red background
- **Error messages:** Red text below textarea

### 2. Validation Error Display

```html
<div class="serial-validation-errors mt-2">
    <div class="text-red-600 text-sm mb-1">
        <strong>Duplicates in this line:</strong> SN001, SN002
    </div>
    <div class="text-red-600 text-sm">
        <strong>Already used in other DCs:</strong>
        <ul class="list-disc list-inside">
            <li>SN003 - Used in DC/2024-25/00042 (issued)</li>
            <li>SN004 - Used in DC/2024-25/00039 (draft)</li>
        </ul>
    </div>
</div>
```

### 3. Barcode Scanner Visual Feedback

```html
<!-- Optional: Show scanning indicator -->
<div class="scanning-indicator hidden">
    <svg class="animate-spin h-4 w-4 text-blue-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
    </svg>
    <span class="ml-2 text-sm text-gray-600">Scanning...</span>
</div>
```

## Testing Checklist

### Functional Testing

- [ ] Create serial_numbers table with migration
- [ ] Verify UNIQUE constraint on (project_id, serial_number)
- [ ] Test serial number insertion on DC creation
- [ ] Test duplicate serial within same line (should show error)
- [ ] Test duplicate serial in same project (should show error with DC number)
- [ ] Test duplicate serial in different project (should be allowed)
- [ ] Test real-time validation during typing
- [ ] Test real-time validation with barcode scanner
- [ ] Verify validation debouncing (doesn't fire on every keystroke)
- [ ] Test edit mode (should exclude current DC from duplicate check)
- [ ] Test serial number deletion on DC deletion
- [ ] Verify serial numbers are freed for reuse after deletion
- [ ] Test form submission prevention with invalid serials
- [ ] Test whitespace trimming
- [ ] Test empty line filtering
- [ ] Test validation error messages display correctly
- [ ] Test validation success (green border)

### Edge Cases

- [ ] Test with 0 serial numbers (should fail quantity validation)
- [ ] Test with 1000+ serial numbers in single line
- [ ] Test with very long serial numbers (255+ characters)
- [ ] Test with special characters in serial numbers
- [ ] Test with Unicode characters (Chinese, Arabic, etc.)
- [ ] Test concurrent DC creation with same serials
- [ ] Test database UNIQUE constraint violation handling
- [ ] Test transaction rollback on serial insert failure
- [ ] Test barcode scanner rapid input (10+ scans/second)
- [ ] Test manual typing vs scanner input
- [ ] Test copy-paste of serial numbers
- [ ] Test validation with network delay/timeout

### Performance Testing

- [ ] Test validation API response time with 100 serials
- [ ] Test bulk insert performance with 500 serials
- [ ] Test database query performance with 10,000+ existing serials
- [ ] Test UNIQUE index lookup speed
- [ ] Test debounce delay (should be ~500ms)
- [ ] Verify no UI lag during rapid scanning

### Integration Testing

- [ ] Test Transit DC creation with serial validation
- [ ] Test Official DC creation with serial validation
- [ ] Test DC edit with serial validation
- [ ] Test DC deletion frees serial numbers
- [ ] Verify serial numbers can be reused after DC deletion
- [ ] Test serial history API endpoint
- [ ] Verify serial numbers persist across DC status changes (draft → issued)
- [ ] Test Phase 14 integration (Issue DC does not affect serials)

### UI/UX Testing

- [ ] Verify red border appears immediately on duplicate
- [ ] Verify green border appears on valid input
- [ ] Verify error messages are clear and actionable
- [ ] Test error message positioning (below textarea)
- [ ] Verify form submission is blocked with validation errors
- [ ] Test validation UI on mobile/tablet devices
- [ ] Verify barcode scanner experience is smooth
- [ ] Test keyboard navigation with validation errors

## Acceptance Criteria

### Must Have

1. **Database Schema**
   - ✅ serial_numbers table created with proper structure
   - ✅ UNIQUE constraint on (project_id, serial_number)
   - ✅ Indexes for fast lookups
   - ✅ CASCADE DELETE on dc_id

2. **Serial Number Validation**
   - ✅ Real-time validation during input
   - ✅ Duplicate detection within same line
   - ✅ Duplicate detection across project DCs
   - ✅ Exclude current DC in edit mode
   - ✅ Show which DC is using duplicate serial
   - ✅ Prevent form submission with duplicates

3. **Data Operations**
   - ✅ Bulk insert serial numbers on DC save
   - ✅ Delete serial numbers on DC deletion
   - ✅ Update serial numbers on DC edit
   - ✅ Transaction safety (all-or-nothing)

4. **UI Feedback**
   - ✅ Red border/background for invalid serials
   - ✅ Green border/background for valid serials
   - ✅ Clear error messages with DC numbers
   - ✅ Validation debouncing (no excessive API calls)

5. **Barcode Scanner Support**
   - ✅ Handle rapid input without lag
   - ✅ Proper newline handling
   - ✅ Auto-trigger validation after scan

6. **Serial Reuse**
   - ✅ Serial numbers freed on DC deletion
   - ✅ Freed serials can be reused immediately
   - ✅ UNIQUE constraint prevents accidental reuse

### Should Have

1. **Performance**
   - ✅ Validation response < 500ms for 100 serials
   - ✅ Bulk insert < 1 second for 500 serials
   - ✅ Database indexes optimize queries

2. **Error Handling**
   - ✅ Handle database constraint violations gracefully
   - ✅ Transaction rollback on any error
   - ✅ Clear error messages for users

3. **Edit Mode**
   - ✅ Pre-populate serial numbers in edit form
   - ✅ Exclude current DC from duplicate checks
   - ✅ Allow saving with unchanged serials

### Nice to Have

1. **Advanced Features**
   - ⭕ Serial number history view (all DCs that used a serial)
   - ⭕ Bulk serial number upload (CSV/Excel)
   - ⭕ Serial number format validation (regex patterns)
   - ⭕ Warning for similar serials (typo detection)

2. **UX Enhancements**
   - ⭕ Auto-focus next line after successful scan
   - ⭕ Visual indicator for scanning mode
   - ⭕ Sound feedback on duplicate detection
   - ⭕ Keyboard shortcut to clear validation errors

---

## Notes

- Serial numbers are case-sensitive by default (can be configured)
- UNIQUE constraint is at database level for data integrity
- Same serial can exist in different projects (project_id is part of UNIQUE constraint)
- Deleted DC numbers are NOT reused, but serial numbers ARE reused
- Validation is both client-side (UX) and server-side (security)
- Barcode scanners typically send Enter key at end of scan
- Debouncing prevents excessive API calls during manual typing

## Dependencies

- **Phase 11:** Transit DC creation (serial number input UI)
- **Phase 12:** Official DC creation (serial number input UI)
- **Database:** delivery_challans and dc_line_items tables
- **HTMX:** For real-time validation
- **JavaScript:** For client-side duplicate detection and UI feedback

## Implementation Summary (Completed 2026-02-16)

### What Was Implemented

1. **Database Layer** (`internal/database/delivery_challans.go`):
   - `CheckSerialsInProject()` - Queries serial_numbers table joined with DCs and products to find conflicts, with optional DC exclusion for edit mode
   - `DeleteDC()` - Transactional deletion of DC + line items + serial numbers

2. **Validation API** (`internal/handlers/serial_validation.go`):
   - `POST /api/serial-numbers/validate` - Accepts project_id, serial_numbers (newline-separated), optional exclude_dc_id. Returns JSON with duplicate_in_db (with DC number, status, product name), duplicate_in_input, valid flag, total_count
   - `DELETE /projects/:id/dcs/:dcid` - DC deletion handler (draft only)

3. **Frontend Validation** (`static/js/serial_validation.js`):
   - 500ms debounced validation on all `.serial-textarea` inputs
   - Red border + error messages for duplicates (within-input and cross-DC)
   - Green border for valid serials
   - Form submission prevention when validation errors exist
   - Barcode scanner Enter key handling
   - CSRF token included in validation requests

4. **Template Integration**:
   - `serial_validation.js` included in both `create.html` (Transit) and `official_create.html` (Official)
   - Validation errors appear inline below each serial textarea

5. **Route Registration** (`cmd/server/main.go`):
   - `POST /api/serial-numbers/validate` - Serial validation endpoint
   - `DELETE /projects/:id/dcs/:dcid` - DC deletion endpoint

### Pre-existing (from Phase 11/12)

- `serial_numbers` table with UNIQUE(project_id, serial_number) constraint
- Serial number insertion in `CreateDeliveryChallan()` transaction
- Serial number retrieval by line item ID
- UNIQUE constraint error handling in DC creation handlers

### Test Results (Playwright Browser Tests)

- [x] Login and navigate to Transit DC creation form
- [x] Fill serial numbers - quantity counter updates correctly (3 serials = Qty 3)
- [x] Duplicate within same textarea detected ("Duplicates in this list: SN001")
- [x] Red border applied to textarea with duplicates
- [x] Save DC with valid serials - redirects to detail page with serials displayed
- [x] Cross-DC duplicate detection works ("Already used in other DCs: SN001 — SCP-TDC-2526-014 (draft)")
- [x] Official DC form also validates serials against same project scope
- [x] Green border shown for valid serial inputs
- [x] Debouncing works (500ms delay before API call)

### Screenshots

- `serial-validation-duplicate-detected.png` - Within-input duplicate detection
- `serial-validation-cross-dc-duplicate.png` - Cross-DC duplicate detection

## Next Steps

After Phase 13 completion:
1. Proceed to Phase 14 (DC Lifecycle - Issue & Lock)
2. Ensure issued DCs prevent serial number modification
3. Implement serial number reporting (which serials are in which DCs)
4. Consider serial number export functionality
