# Phase 12: Official DC Creation (Draft)

## Overview

This phase implements the Official DC creation functionality for non-commercial delivery challans. Official DCs are simpler than Transit DCs—they contain NO pricing information and NO tax calculations. They are used for internal transfers, demos, service calls, or returns where invoicing is not required.

**Key Features:**
- Create Official DC from DC Template via "Issue Official DC" button
- Auto-generate DC number with /D/ suffix
- Simplified form (no pricing, no tax)
- Serial number tracking with barcode scanner support
- Auto-calculated quantities from serial numbers
- Purpose pre-filled from template
- Shares delivery_challans table with dc_type='official'
- Draft status allowing edits
- No dc_transit_details record needed

## Prerequisites

- Phase 11 (Transit DC Creation) completed
- Phase 10 (DC Templates) completed
- Phase 9 (DC number generation logic) implemented
- Serial number scanning component from Phase 11 reusable
- Database schema for delivery_challans and dc_line_items tables
- HTMX and Tailwind CSS configured

## Goals

1. Enable Official DC creation from DC Templates
2. Implement simplified form without pricing/tax
3. Reuse serial number scanning component from Phase 11
4. Auto-calculate quantities from serial number counts
5. Support searchable address dropdowns
6. Save Official DC in Draft status
7. Differentiate Official DCs using dc_type='official'
8. Share database infrastructure with Transit DCs

## Detailed Implementation Steps

### Step 1: Database Schema Verification

Verify existing tables support Official DCs:

**delivery_challans table:**
- dc_type column must support 'official' value
- All columns from Phase 11 are reused
- template_id links to original DC Template

**dc_line_items table:**
- Existing columns used, but price/GST fields will be NULL or 0 for Official DCs
- Add remarks column if not present (for per-line notes)

**No dc_transit_details record:**
- Official DCs do NOT create records in dc_transit_details
- Transport details not applicable to Official DCs

### Step 2: Backend Route Setup

Add routes in `routes/dc_routes.go`:

```go
// Official DC Routes
dcRoutes := r.Group("/projects/:project_id/dc")
{
    dcRoutes.GET("/templates/:template_id/official/new", handlers.NewOfficialDCFromTemplate)
    dcRoutes.POST("/official", handlers.CreateOfficialDC)
    dcRoutes.GET("/official/:dc_id/edit", handlers.EditOfficialDC)
    dcRoutes.PUT("/official/:dc_id", handlers.UpdateOfficialDC)
}

// Reuse HTMX endpoints from Phase 11
apiRoutes := r.Group("/api")
{
    apiRoutes.POST("/dc/calculate-quantity", handlers.CalculateQuantityFromSerials)
    apiRoutes.GET("/projects/:project_id/addresses/search", handlers.SearchProjectAddresses)
}
```

### Step 3: Handler Implementation

Create `handlers/official_dc_handler.go`:

**NewOfficialDCFromTemplate:**
- Fetch DC Template by template_id
- Fetch project details for default addresses
- Fetch template products (name, description, brand/model only)
- Generate DC number with /D/ suffix (e.g., DC/2024-25/00042/D/)
- Pre-fill purpose from template
- Render Official DC creation form

**CreateOfficialDC:**
- Validate form inputs (dates, addresses, purpose)
- Parse serial numbers (newline-separated) for each line item
- Count serial numbers to set quantity
- Insert into delivery_challans (dc_type='official', status='draft')
- Insert into dc_line_items (price=0, gst=0, totals=0)
- Skip dc_transit_details table
- Return success response or validation errors

**EditOfficialDC:**
- Fetch Official DC by dc_id (only if status='draft')
- Fetch associated line_items
- Fetch serial numbers for each line item (Phase 13)
- Render edit form with populated data

**UpdateOfficialDC:**
- Validate DC is still in 'draft' status
- Update delivery_challans record
- Delete existing dc_line_items
- Re-insert updated dc_line_items with new serial numbers
- Return success response

### Step 4: DC Number Generation with /D/ Suffix

Modify `utils/dc_number_generator.go`:

```go
func GenerateDCNumber(db *sql.DB, projectID int64, dcType string) (string, error) {
    // Get financial year (Apr-Mar)
    now := time.Now()
    var fyYear string
    if now.Month() >= time.April {
        fyYear = fmt.Sprintf("%d-%d", now.Year(), now.Year()+1)
    } else {
        fyYear = fmt.Sprintf("%d-%d", now.Year()-1, now.Year())
    }

    // Get next sequence number
    var maxNumber int
    query := `
        SELECT COALESCE(MAX(CAST(SUBSTR(dc_number, -10, 5) AS INTEGER)), 0)
        FROM delivery_challans
        WHERE project_id = ?
          AND dc_number LIKE ?
    `
    pattern := "DC/" + fyYear + "/%"
    err := db.QueryRow(query, projectID, pattern).Scan(&maxNumber)
    if err != nil {
        return "", err
    }

    nextNumber := maxNumber + 1

    // Format: DC/2024-25/00042 or DC/2024-25/00042/D/
    baseNumber := fmt.Sprintf("DC/%s/%05d", fyYear, nextNumber)

    if dcType == "official" {
        return baseNumber + "/D/", nil
    }

    return baseNumber, nil
}
```

**Logic:**
- Same sequence counter for both Transit and Official DCs
- Official DCs append /D/ suffix to base number
- Example sequence:
  - DC/2024-25/00041 (Transit)
  - DC/2024-25/00042/D/ (Official)
  - DC/2024-25/00043 (Transit)
  - DC/2024-25/00044/D/ (Official)

### Step 5: Models and Database Queries

Add to `models/official_dc.go`:

```go
type OfficialDC struct {
    ID           int64
    ProjectID    int64
    TemplateID   *int64
    DCNumber     string
    DCDate       time.Time
    Purpose      string
    BillToID     int64
    ShipToID     int64
    Notes        string
    Status       string // 'draft' or 'issued'
    CreatedAt    time.Time
    UpdatedAt    time.Time
    IssuedAt     *time.Time
}

type OfficialDCLineItem struct {
    ID            int64
    DCID          int64
    ProductID     int64
    LineNumber    int
    ItemName      string
    Description   string
    BrandModel    string
    Quantity      int
    SerialNumbers []string // For form binding
    Remarks       string
}
```

**Key Queries:**

```sql
-- Insert Official DC
INSERT INTO delivery_challans (
    project_id, template_id, dc_number, dc_type, dc_date,
    purpose, bill_to_id, ship_to_id, notes, status,
    created_at, updated_at
) VALUES (
    ?, ?, ?, 'official', ?,
    ?, ?, ?, ?, 'draft',
    CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);

-- Insert Official DC Line Items (no pricing)
INSERT INTO dc_line_items (
    dc_id, product_id, line_number, item_name, description,
    brand_model, uom, hsn_code, price, gst_percentage,
    quantity, taxable_value, gst_amount, total_amount, remarks
) VALUES (
    ?, ?, ?, ?, ?,
    ?, '', '', 0, 0,
    ?, 0, 0, 0, ?
);

-- Fetch Official DC
SELECT
    dc.id, dc.project_id, dc.template_id, dc.dc_number,
    dc.dc_date, dc.purpose, dc.bill_to_id, dc.ship_to_id,
    dc.notes, dc.status, dc.created_at, dc.updated_at,
    bt.company_name as bill_to_name, bt.address as bill_to_address,
    st.company_name as ship_to_name, st.address as ship_to_address
FROM delivery_challans dc
LEFT JOIN project_addresses bt ON dc.bill_to_id = bt.id
LEFT JOIN project_addresses st ON dc.ship_to_id = st.id
WHERE dc.id = ? AND dc.dc_type = 'official';

-- Fetch Line Items (Official)
SELECT
    li.id, li.dc_id, li.product_id, li.line_number,
    li.item_name, li.description, li.brand_model,
    li.quantity, li.remarks
FROM dc_line_items li
WHERE li.dc_id = ?
ORDER BY li.line_number;
```

### Step 6: Frontend Form Implementation

Create `views/dc/official_create.html`:

```html
<div class="container mx-auto px-4 py-6">
    <div class="bg-white shadow rounded-lg p-6">
        <h1 class="text-2xl font-bold mb-6">Create Official Delivery Challan</h1>

        <form hx-post="/projects/{{.ProjectID}}/dc/official"
              hx-target="#form-container"
              class="space-y-6">

            <!-- DC Header Section -->
            <div class="grid grid-cols-3 gap-4">
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">
                        DC Number
                    </label>
                    <input type="text"
                           name="dc_number"
                           value="{{.DCNumber}}"
                           readonly
                           class="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50">
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">
                        DC Date <span class="text-red-500">*</span>
                    </label>
                    <input type="date"
                           name="dc_date"
                           value="{{.Today}}"
                           required
                           class="w-full px-3 py-2 border border-gray-300 rounded-md">
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">
                        Purpose
                    </label>
                    <input type="text"
                           name="purpose"
                           value="{{.Purpose}}"
                           readonly
                           class="w-full px-3 py-2 border border-gray-300 rounded-md bg-gray-50">
                </div>
            </div>

            <!-- Address Section -->
            <div class="grid grid-cols-2 gap-4">
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">
                        Bill To <span class="text-red-500">*</span>
                    </label>
                    <div class="relative">
                        <input type="text"
                               placeholder="Type to search..."
                               class="address-search-input w-full px-3 py-2 border border-gray-300 rounded-md mb-1">
                        <select name="bill_to_id"
                                class="searchable-dropdown w-full px-3 py-2 border border-gray-300 rounded-md"
                                required>
                            <option value="">Select Bill To Address</option>
                            {{range .BillToAddresses}}
                            <option value="{{.ID}}"
                                    {{if eq .ID $.DefaultBillToID}}selected{{end}}
                                    data-search="{{.CompanyName}} {{.City}} {{.State}}">
                                {{.CompanyName}} - {{.City}}, {{.State}}
                            </option>
                            {{end}}
                        </select>
                    </div>
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">
                        Ship To <span class="text-red-500">*</span>
                    </label>
                    <div class="relative">
                        <input type="text"
                               placeholder="Type to search..."
                               class="address-search-input w-full px-3 py-2 border border-gray-300 rounded-md mb-1">
                        <select name="ship_to_id"
                                class="searchable-dropdown w-full px-3 py-2 border border-gray-300 rounded-md"
                                required>
                            <option value="">Select Ship To Address</option>
                            {{range .ShipToAddresses}}
                            <option value="{{.ID}}"
                                    {{if eq .ID $.DefaultShipToID}}selected{{end}}
                                    data-search="{{.CompanyName}} {{.City}} {{.State}}">
                                {{.CompanyName}} - {{.City}}, {{.State}}
                            </option>
                            {{end}}
                        </select>
                    </div>
                </div>
            </div>

            <!-- Product Lines Table -->
            <div class="overflow-x-auto">
                <table class="min-w-full divide-y divide-gray-200 border">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-16">
                                S.No
                            </th>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Item Name
                            </th>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Description
                            </th>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Brand/Model No
                            </th>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">
                                Quantity
                            </th>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Serial Numbers
                            </th>
                            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Remarks
                            </th>
                        </tr>
                    </thead>
                    <tbody class="bg-white divide-y divide-gray-200">
                        {{range $index, $product := .Products}}
                        <tr class="product-line hover:bg-gray-50">
                            <td class="px-4 py-3 text-sm text-gray-900">
                                {{add $index 1}}
                            </td>
                            <td class="px-4 py-3 text-sm text-gray-900">
                                <input type="hidden"
                                       name="line_items[{{$index}}].product_id"
                                       value="{{$product.ID}}">
                                {{$product.Name}}
                            </td>
                            <td class="px-4 py-3 text-sm text-gray-500">
                                {{$product.Description}}
                            </td>
                            <td class="px-4 py-3 text-sm text-gray-500">
                                {{$product.BrandModel}}
                            </td>
                            <td class="px-4 py-3 text-sm">
                                <input type="number"
                                       name="line_items[{{$index}}].quantity"
                                       class="quantity-display w-20 px-2 py-1 border border-gray-300 rounded bg-gray-50 text-center"
                                       value="0"
                                       readonly>
                            </td>
                            <td class="px-4 py-3">
                                <textarea
                                    name="line_items[{{$index}}].serial_numbers"
                                    class="serial-input font-mono text-xs w-full px-2 py-1 border border-gray-300 rounded"
                                    rows="3"
                                    placeholder="Scan serial numbers (one per line)"
                                    data-line-index="{{$index}}"
                                ></textarea>
                            </td>
                            <td class="px-4 py-3">
                                <input type="text"
                                       name="line_items[{{$index}}].remarks"
                                       class="w-full px-2 py-1 border border-gray-300 rounded text-sm"
                                       placeholder="Optional notes">
                            </td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>

            <!-- Notes Section -->
            <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                    Additional Notes
                </label>
                <textarea name="notes"
                          rows="3"
                          class="w-full px-3 py-2 border border-gray-300 rounded-md"
                          placeholder="Enter any additional information or instructions..."></textarea>
            </div>

            <!-- Action Buttons -->
            <div class="flex justify-end gap-3 pt-4 border-t">
                <a href="/projects/{{.ProjectID}}/dc"
                   class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50">
                    Cancel
                </a>
                <button type="submit"
                        class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700">
                    Save as Draft
                </button>
            </div>
        </form>
    </div>
</div>
```

### Step 7: Reuse Serial Number Component from Phase 11

Include `static/js/dc_calculations.js` (modified for Official DCs):

```javascript
// Official DC Serial Number Handler (no pricing calculations)
document.addEventListener('DOMContentLoaded', function() {
    // Handle serial number input for quantity calculation
    const serialInputs = document.querySelectorAll('.serial-input');

    serialInputs.forEach(textarea => {
        textarea.addEventListener('input', function() {
            const row = this.closest('tr');
            const quantityInput = row.querySelector('.quantity-display');

            // Count non-empty lines
            const serialNumbers = this.value
                .split('\n')
                .map(s => s.trim())
                .filter(s => s.length > 0);

            quantityInput.value = serialNumbers.length;
        });
    });

    // Initialize searchable dropdowns
    initializeSearchableDropdowns();
});

function initializeSearchableDropdowns() {
    const searchInputs = document.querySelectorAll('.address-search-input');

    searchInputs.forEach(searchInput => {
        const selectElement = searchInput.nextElementSibling;

        searchInput.addEventListener('input', function() {
            const filter = this.value.toLowerCase();
            const options = selectElement.options;

            for (let i = 0; i < options.length; i++) {
                const searchText = options[i].getAttribute('data-search') || '';
                options[i].style.display = searchText.toLowerCase().includes(filter) ? '' : 'none';
            }
        });
    });
}
```

**Key Differences from Transit DC:**
- No price/tax calculations needed
- Only quantity calculation from serial numbers
- Simpler JavaScript (no tax type toggle, no totals)
- Same barcode scanner optimization

### Step 8: Template Detail Page Integration

Modify `views/dc_templates/detail.html` to add "Issue Official DC" button:

```html
<div class="flex gap-3">
    <!-- Existing "Issue Transit DC" button -->
    <a href="/projects/{{.ProjectID}}/dc/templates/{{.Template.ID}}/transit/new"
       class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700">
        Issue Transit DC
    </a>

    <!-- New "Issue Official DC" button -->
    <a href="/projects/{{.ProjectID}}/dc/templates/{{.Template.ID}}/official/new"
       class="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700">
        Issue Official DC
    </a>
</div>
```

### Step 9: Validation and Error Handling

Add validation in `handlers/official_dc_handler.go`:

```go
func validateOfficialDC(dc *OfficialDC, lineItems []OfficialDCLineItem) []string {
    errors := []string{}

    // Validate dates
    if dc.DCDate.IsZero() {
        errors = append(errors, "DC date is required")
    }

    // Validate purpose
    if strings.TrimSpace(dc.Purpose) == "" {
        errors = append(errors, "Purpose is required")
    }

    // Validate addresses
    if dc.BillToID == 0 {
        errors = append(errors, "Bill To address is required")
    }
    if dc.ShipToID == 0 {
        errors = append(errors, "Ship To address is required")
    }

    // Validate line items
    if len(lineItems) == 0 {
        errors = append(errors, "At least one line item is required")
    }

    for i, item := range lineItems {
        if item.Quantity == 0 {
            errors = append(errors, fmt.Sprintf("Line %d: Quantity must be greater than 0", i+1))
        }
        if len(item.SerialNumbers) != item.Quantity {
            errors = append(errors, fmt.Sprintf("Line %d: Serial number count (%d) must match quantity (%d)",
                i+1, len(item.SerialNumbers), item.Quantity))
        }
    }

    return errors
}
```

## Files to Create/Modify

### New Files

1. **handlers/official_dc_handler.go**
   - NewOfficialDCFromTemplate
   - CreateOfficialDC
   - EditOfficialDC
   - UpdateOfficialDC
   - validateOfficialDC

2. **models/official_dc.go**
   - OfficialDC struct
   - OfficialDCLineItem struct
   - CreateOfficialDC method
   - UpdateOfficialDC method
   - GetOfficialDCByID method

3. **views/dc/official_create.html**
   - Official DC creation form
   - Simplified product table (no pricing)
   - Serial number scanning
   - Address dropdowns

4. **views/dc/official_edit.html**
   - Same as create but pre-populated
   - Only accessible for Draft status

### Modified Files

1. **routes/dc_routes.go**
   - Add Official DC routes
   - Reuse existing HTMX endpoints

2. **utils/dc_number_generator.go**
   - Add /D/ suffix logic for Official DCs
   - Maintain shared sequence counter

3. **views/dc_templates/detail.html**
   - Add "Issue Official DC" button
   - Separate from "Issue Transit DC" button

4. **static/js/dc_calculations.js**
   - Add Official DC serial number handler
   - Reuse searchable dropdown logic
   - Remove pricing calculation logic for Official DCs

5. **static/css/styles.css**
   - Ensure serial number textarea styles apply
   - Add any Official DC-specific styles

## API Routes/Endpoints

### Main Routes

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects/:project_id/dc/templates/:template_id/official/new` | Show Official DC creation form |
| POST | `/projects/:project_id/dc/official` | Create new Official DC |
| GET | `/projects/:project_id/dc/official/:dc_id/edit` | Show Official DC edit form (Draft only) |
| PUT | `/projects/:project_id/dc/official/:dc_id` | Update Official DC (Draft only) |

### Reused HTMX Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/dc/calculate-quantity` | Calculate quantity from serial numbers |
| GET | `/api/projects/:project_id/addresses/search` | Search project addresses |

### Request/Response Examples

**POST /projects/1/dc/official**

Request Body:
```json
{
  "dc_date": "2026-02-16",
  "purpose": "Demo Equipment - Client XYZ",
  "bill_to_id": 5,
  "ship_to_id": 6,
  "notes": "Return by 2026-02-20",
  "line_items": [
    {
      "product_id": 10,
      "item_name": "Smart Lock Pro",
      "description": "WiFi enabled smart door lock",
      "brand_model": "SL-2000X",
      "serial_numbers": "DEMO001\nDEMO002",
      "quantity": 2,
      "remarks": "Demo units"
    }
  ]
}
```

Response:
```json
{
  "success": true,
  "dc_id": 43,
  "dc_number": "DC/2024-25/00043/D/",
  "redirect": "/projects/1/dc/official/43"
}
```

## Database Queries

### Insert Official DC

```sql
-- Insert main DC record (dc_type='official')
INSERT INTO delivery_challans (
    project_id, template_id, dc_number, dc_type, dc_date,
    purpose, bill_to_id, ship_to_id, notes, status,
    created_at, updated_at
) VALUES (
    ?, ?, ?, 'official', ?,
    ?, ?, ?, ?, 'draft',
    CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);

-- Insert line items (pricing fields = 0 or NULL)
INSERT INTO dc_line_items (
    dc_id, product_id, line_number, item_name, description,
    brand_model, uom, hsn_code, price, gst_percentage,
    quantity, taxable_value, gst_amount, total_amount, remarks
) VALUES (
    ?, ?, ?, ?, ?,
    ?, '', '', 0, 0,
    ?, 0, 0, 0, ?
);
```

### Fetch Official DC

```sql
SELECT
    dc.id, dc.project_id, dc.template_id, dc.dc_number, dc.dc_type,
    dc.dc_date, dc.purpose, dc.bill_to_id, dc.ship_to_id, dc.notes,
    dc.status, dc.created_at, dc.updated_at, dc.issued_at,

    bt.company_name as bill_to_name, bt.address as bill_to_address,
    bt.city as bill_to_city, bt.state as bill_to_state,
    bt.pincode as bill_to_pincode, bt.gstin as bill_to_gstin,

    st.company_name as ship_to_name, st.address as ship_to_address,
    st.city as ship_to_city, st.state as ship_to_state,
    st.pincode as ship_to_pincode, st.gstin as ship_to_gstin

FROM delivery_challans dc
LEFT JOIN project_addresses bt ON dc.bill_to_id = bt.id
LEFT JOIN project_addresses st ON dc.ship_to_id = st.id
WHERE dc.id = ? AND dc.dc_type = 'official';
```

### Fetch Official DC Line Items

```sql
SELECT
    li.id, li.dc_id, li.product_id, li.line_number,
    li.item_name, li.description, li.brand_model,
    li.quantity, li.remarks
FROM dc_line_items li
WHERE li.dc_id = ?
ORDER BY li.line_number;
```

### Update Official DC

```sql
-- Update main record
UPDATE delivery_challans
SET dc_date = ?,
    bill_to_id = ?,
    ship_to_id = ?,
    notes = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = 'draft' AND dc_type = 'official';

-- Delete old line items
DELETE FROM dc_line_items WHERE dc_id = ?;

-- Re-insert updated line items (same INSERT as above)
```

## UI Components

### 1. Simplified Product Table (No Pricing)

```html
<table class="min-w-full divide-y divide-gray-200">
    <thead class="bg-gray-50">
        <tr>
            <th>S.No</th>
            <th>Item Name</th>
            <th>Description</th>
            <th>Brand/Model No</th>
            <th>Quantity</th>
            <th>Serial Numbers</th>
            <th>Remarks</th>
        </tr>
    </thead>
    <tbody>
        <!-- Product rows without price/tax columns -->
    </tbody>
</table>
```

**Features:**
- No Price, GST%, Taxable Value, or Total columns
- Focus on item identification and serial numbers
- Remarks column for per-line notes

### 2. Serial Number Textarea (Reused from Phase 11)

```html
<textarea
    class="serial-input font-mono text-xs"
    rows="3"
    placeholder="Scan serial numbers (one per line)"
    data-line-index="{{$index}}"
></textarea>
```

**Features:**
- Same barcode scanner optimization
- Auto-calculates quantity
- Monospace font for readability

### 3. Simplified Header Form

```html
<div class="grid grid-cols-3 gap-4">
    <div>
        <label>DC Number</label>
        <input type="text" value="DC/2024-25/00042/D/" readonly>
    </div>
    <div>
        <label>DC Date</label>
        <input type="date" required>
    </div>
    <div>
        <label>Purpose</label>
        <input type="text" value="Demo Equipment" readonly>
    </div>
</div>
```

**Features:**
- DC number with /D/ suffix
- Purpose pre-filled from template
- No transport details section

### 4. Address Dropdowns (Reused from Phase 11)

```html
<div class="relative">
    <input type="text" placeholder="Type to search..." class="address-search-input">
    <select name="bill_to_id" class="searchable-dropdown" required>
        <option value="">Select Address</option>
        <!-- Options -->
    </select>
</div>
```

**Features:**
- Same searchable functionality
- Filter by company name, city, state

## Testing Checklist

### Functional Testing

- [ ] Click "Issue Official DC" from DC Template detail page
- [ ] Verify DC number is auto-generated with /D/ suffix
- [ ] Verify purpose is pre-filled from template
- [ ] Verify default Bill To and Ship To are selected
- [ ] Verify product lines are populated without pricing columns
- [ ] Test barcode scanner input in serial number field
- [ ] Verify quantity auto-calculates from serial number count
- [ ] Test manual serial number entry (newline-separated)
- [ ] Verify whitespace trimming and empty line filtering
- [ ] Test searchable dropdown for Bill To addresses
- [ ] Test searchable dropdown for Ship To addresses
- [ ] Test Remarks field per line item
- [ ] Test Notes field
- [ ] Submit form and verify DC is created with status='draft' and dc_type='official'
- [ ] Verify NO dc_transit_details record is created
- [ ] Verify pricing fields in dc_line_items are 0 or NULL
- [ ] Test edit functionality for Draft Official DC
- [ ] Verify cannot edit Issued Official DC

### Edge Cases

- [ ] Test with 0 products in template
- [ ] Test with very long serial number list (100+ items)
- [ ] Test with special characters in serial numbers
- [ ] Test with duplicate serial numbers within same line
- [ ] Test form validation with missing required fields
- [ ] Test with invalid DC date (future date)
- [ ] Test concurrent Official DC creation (number generation)
- [ ] Test DC number sequence shared between Transit and Official DCs
- [ ] Test /D/ suffix appears correctly in all displays

### UI/UX Testing

- [ ] Verify form layout is clean without pricing columns
- [ ] Verify serial number textarea is easily scannable
- [ ] Test responsive design on different screen sizes
- [ ] Test keyboard navigation through form
- [ ] Verify loading states during form submission
- [ ] Test error message display
- [ ] Verify success message after creation
- [ ] Test Cancel button returns to correct page
- [ ] Verify dropdown search performance

### Integration Testing

- [ ] Verify DC number sequence works correctly with Transit DCs
- [ ] Verify template data is accurately copied
- [ ] Verify project addresses are fetched correctly
- [ ] Test with multiple users creating DCs simultaneously
- [ ] Verify serial numbers prepare for Phase 13 integration
- [ ] Test that Official and Transit DCs can coexist in same project

## Acceptance Criteria

### Must Have

1. **Template Integration**
   - ✅ Official DC can be created from any DC Template
   - ✅ DC number is auto-generated with /D/ suffix
   - ✅ Purpose is pre-filled from template
   - ✅ Product lines are auto-populated (no pricing)

2. **DC Number Generation**
   - ✅ DC number format: DC/YYYY-YY/NNNNN/D/
   - ✅ Sequence counter shared with Transit DCs
   - ✅ /D/ suffix distinguishes Official DCs

3. **Form Functionality**
   - ✅ All required fields are present and validated
   - ✅ Bill To and Ship To dropdowns are searchable
   - ✅ Default addresses pre-selected
   - ✅ No pricing or tax fields displayed
   - ✅ Remarks field available per line item

4. **Serial Number Management**
   - ✅ Serial number textarea accepts barcode scanner input
   - ✅ Serial numbers are newline-separated
   - ✅ Whitespace trimmed, empty lines ignored
   - ✅ Quantity auto-calculates from serial count
   - ✅ Quantity display is read-only

5. **Data Persistence**
   - ✅ DC saved with status='draft' and dc_type='official'
   - ✅ All form data saved to delivery_challans table
   - ✅ Line items saved to dc_line_items table
   - ✅ NO dc_transit_details record created
   - ✅ Pricing fields in line items are 0 or NULL

6. **UI/UX**
   - ✅ Form is clean and simple (no pricing clutter)
   - ✅ Search dropdowns filter correctly
   - ✅ Error messages display clearly
   - ✅ Success redirect works

### Should Have

1. **Validation**
   - ✅ Client-side validation prevents empty submissions
   - ✅ Server-side validation checks all required fields
   - ✅ Serial number count must match quantity
   - ✅ Meaningful error messages displayed

2. **Performance**
   - ✅ Form loads in < 2 seconds
   - ✅ Address search returns results in < 1 second
   - ✅ Serial number input is responsive

3. **Consistency**
   - ✅ Reuses components from Phase 11
   - ✅ Consistent styling with Transit DC forms
   - ✅ Same keyboard shortcuts and navigation

### Nice to Have

1. **Advanced Features**
   - ⭕ Auto-save draft every 60 seconds
   - ⭕ Duplicate DC from existing Official DC
   - ⭕ Quick toggle between Transit and Official DC creation

2. **UX Enhancements**
   - ⭕ Toast notifications for successful save
   - ⭕ Keyboard shortcuts (Ctrl+S to save)
   - ⭕ Recently used addresses quick select

---

## Notes

- Official DCs share the same database tables as Transit DCs
- dc_type column differentiates between 'transit' and 'official'
- DC number sequence is shared but Official DCs have /D/ suffix
- NO dc_transit_details records for Official DCs
- Pricing fields in dc_line_items are set to 0 for Official DCs
- Serial number scanning component is reused from Phase 11
- Phase 14 will handle Issue/Lock workflow for both DC types
- Phase 13 will implement serial number validation for both DC types

## Dependencies

- **Phase 11:** Transit DC creation must be completed (for shared components)
- **Phase 10:** DC Templates must be implemented
- **Phase 9:** DC number generation logic
- **Database:** delivery_challans and dc_line_items tables must exist

## Implementation Summary (Completed 2026-02-16)

### What Was Built

**New Files Created:**
1. `internal/handlers/official_dc.go` - ShowCreateOfficialDC, CreateOfficialDC, ShowOfficialDCDetail handlers
2. `templates/pages/delivery_challans/official_create.html` - Simplified creation form (no pricing/tax)
3. `templates/pages/delivery_challans/official_detail.html` - Detail view for Official DCs

**Modified Files:**
1. `cmd/server/main.go` - Added Official DC routes (`GET /dcs/official/new`, `POST /dcs/official`) and unified DC detail route
2. `internal/handlers/transit_dc.go` - Added `ShowDCDetail` dispatcher that routes to transit or official detail based on `dc_type`
3. `templates/pages/dc_templates/detail.html` - Enabled "Issue Official DC" button (was disabled placeholder)

### Key Design Decisions
- **Reused existing database layer** - `CreateDeliveryChallan()` already supports `nil` transit details, so no DB changes needed
- **DC Number Format**: Uses existing `PREFIX-ODC-YYYYYY-NNN` format (e.g., `SCP-ODC-2526-002`) with separate sequence from Transit DCs
- **Unified detail route**: `/projects/:id/dcs/:dcid` dispatches to correct template based on `dc_type`
- **No pricing fields**: Official DC form shows only product info, serial numbers, and quantity
- **Green theme**: Official DC uses green badge/button to visually distinguish from Transit DC (indigo)

### Test Results (Playwright Browser Tests)
- ✅ Login and navigate to project templates
- ✅ "Issue Official DC" button visible and clickable on template detail page
- ✅ Official DC form loads with ODC number auto-generated
- ✅ "OFFICIAL DC" badge displayed
- ✅ Template and Purpose pre-filled from template
- ✅ Bill To and Ship To address dropdowns work
- ✅ Product lines show without pricing columns (no Price, GST%, Taxable, Total)
- ✅ "No pricing for Official DCs" label visible
- ✅ Serial number entry works with quantity auto-calculation
- ✅ Duplicate serial number validation works (shows error, re-renders form)
- ✅ Form submission creates DC with status='draft', dc_type='official'
- ✅ Redirect to detail page after successful creation
- ✅ Detail page shows "Official DC" label, serial numbers, quantities, no pricing
- ✅ Ship To address details displayed correctly

## Next Steps

After Phase 12 completion:
1. Proceed to Phase 13 (Serial Number Management & Validation)
2. Implement Phase 14 (DC Lifecycle - Issue & Lock for both types)
3. Implement Phase 15 (Transit DC View & Print)
4. Implement Official DC View & Print (similar to Phase 15)
