# Phase 11: Transit DC Creation (Draft)

## Overview

This phase implements the Transit DC creation functionality, which allows users to create delivery challans with complete pricing and tax information from DC Templates. Transit DCs include detailed tax calculations (CGST+SGST or IGST), serial number tracking, and auto-calculated quantities based on scanned serial numbers.

**Key Features:**
- Create Transit DC from DC Template via "Issue Transit DC" button
- Auto-generate DC number using Phase 10 logic
- Complete tax and pricing information
- Serial number scanning with barcode scanner support
- Real-time quantity calculation from serial numbers
- Dual tax type support (CGST+SGST or IGST)
- Searchable dropdowns for addresses
- Draft status allowing edits

## Prerequisites

- Phase 10 (DC Templates) completed
- Phase 9 (DC number generation logic) implemented
- Database schema for delivery_challans, dc_transit_details, dc_line_items tables
- HTMX and Tailwind CSS configured
- SQLite database connection established
- Mockup file: 12-create-dc.html

## Goals

1. Enable Transit DC creation from DC Templates
2. Implement complete tax calculation system
3. Support barcode scanner input for serial numbers
4. Auto-calculate quantities from serial number counts
5. Provide searchable address dropdowns
6. Calculate totals in real-time (taxable value, GST, round-off)
7. Convert final amount to words (Indian format)
8. Save Transit DC in Draft status
9. Match mockup 12-create-dc.html layout and functionality

## Detailed Implementation Steps

### Step 1: Database Schema Setup

Create/verify tables for Transit DC storage:

**delivery_challans table:**
- id (PRIMARY KEY)
- project_id (FOREIGN KEY)
- template_id (FOREIGN KEY - nullable)
- dc_number (UNIQUE)
- dc_type ('transit' or 'official')
- dc_date
- purpose
- bill_to_id (FOREIGN KEY to project_addresses)
- ship_to_id (FOREIGN KEY to project_addresses)
- notes
- status ('draft' or 'issued')
- created_at
- updated_at
- issued_at (nullable)

**dc_transit_details table:**
- id (PRIMARY KEY)
- dc_id (FOREIGN KEY to delivery_challans)
- mode_of_transport
- driver_name
- vehicle_number
- docket_number (nullable)
- eway_bill_number (nullable)
- reverse_charge (BOOLEAN, default false)
- tax_type ('cgst_sgst' or 'igst')
- taxable_value (DECIMAL)
- cgst_amount (DECIMAL, nullable)
- sgst_amount (DECIMAL, nullable)
- igst_amount (DECIMAL, nullable)
- round_off (DECIMAL)
- total_value (DECIMAL)

**dc_line_items table:**
- id (PRIMARY KEY)
- dc_id (FOREIGN KEY to delivery_challans)
- product_id (FOREIGN KEY to products)
- line_number (INTEGER)
- item_name
- description
- brand_model
- uom
- hsn_code
- price (DECIMAL)
- gst_percentage (DECIMAL)
- quantity (INTEGER)
- taxable_value (DECIMAL)
- gst_amount (DECIMAL)
- total_amount (DECIMAL)
- remarks

### Step 2: Backend Route Setup

Create Gin routes in `routes/dc_routes.go`:

```go
// Transit DC Routes
dcRoutes := r.Group("/projects/:project_id/dc")
{
    dcRoutes.GET("/templates/:template_id/transit/new", handlers.NewTransitDCFromTemplate)
    dcRoutes.POST("/transit", handlers.CreateTransitDC)
    dcRoutes.GET("/transit/:dc_id/edit", handlers.EditTransitDC)
    dcRoutes.PUT("/transit/:dc_id", handlers.UpdateTransitDC)
}

// HTMX partial endpoints
apiRoutes := r.Group("/api")
{
    apiRoutes.POST("/dc/calculate-line", handlers.CalculateDCLineItem)
    apiRoutes.POST("/dc/calculate-totals", handlers.CalculateDCTotals)
    apiRoutes.GET("/projects/:project_id/addresses/search", handlers.SearchProjectAddresses)
}
```

### Step 3: Handler Implementation

Create `handlers/transit_dc_handler.go`:

**NewTransitDCFromTemplate:**
- Fetch DC Template by template_id
- Fetch project details (for default bill-to, ship-to)
- Fetch template products with pricing
- Generate new DC number using Phase 10 logic
- Pre-fill form with template data
- Render Transit DC creation form

**CreateTransitDC:**
- Validate form inputs
- Parse serial numbers (newline-separated) for each line item
- Count serial numbers to set quantity
- Calculate line item totals (taxable_value = price * quantity)
- Calculate GST amounts based on tax_type
- Calculate grand totals
- Apply round-off logic
- Convert total to words (Indian format)
- Insert into delivery_challans (dc_type='transit', status='draft')
- Insert into dc_transit_details
- Insert into dc_line_items (multiple rows)
- Return success response or validation errors

**EditTransitDC:**
- Fetch Transit DC by dc_id (only if status='draft')
- Fetch associated transit_details and line_items
- Fetch serial numbers for each line item (Phase 13)
- Render edit form with populated data

**UpdateTransitDC:**
- Validate DC is still in 'draft' status
- Update delivery_challans record
- Update dc_transit_details record
- Delete existing dc_line_items
- Re-insert updated dc_line_items
- Recalculate all totals
- Return success response

### Step 4: Models and Database Queries

Create `models/transit_dc.go`:

```go
type TransitDC struct {
    ID              int64
    ProjectID       int64
    TemplateID      *int64
    DCNumber        string
    DCDate          time.Time
    Purpose         string
    BillToID        int64
    ShipToID        int64
    Notes           string
    Status          string
    CreatedAt       time.Time
    UpdatedAt       time.Time
    IssuedAt        *time.Time
}

type TransitDetails struct {
    DCID            int64
    ModeOfTransport string
    DriverName      string
    VehicleNumber   string
    DocketNumber    *string
    EWayBillNumber  *string
    ReverseCharge   bool
    TaxType         string
    TaxableValue    float64
    CGSTAmount      *float64
    SGSTAmount      *float64
    IGSTAmount      *float64
    RoundOff        float64
    TotalValue      float64
}

type DCLineItem struct {
    ID             int64
    DCID           int64
    ProductID      int64
    LineNumber     int
    ItemName       string
    Description    string
    BrandModel     string
    UOM            string
    HSNCode        string
    Price          float64
    GSTPercentage  float64
    Quantity       int
    TaxableValue   float64
    GSTAmount      float64
    TotalAmount    float64
    Remarks        string
    SerialNumbers  []string // For form binding
}
```

**Key Queries:**

```sql
-- Insert Transit DC
INSERT INTO delivery_challans (
    project_id, template_id, dc_number, dc_type, dc_date,
    purpose, bill_to_id, ship_to_id, notes, status, created_at, updated_at
) VALUES (?, ?, ?, 'transit', ?, ?, ?, ?, ?, 'draft', ?, ?);

-- Insert Transit Details
INSERT INTO dc_transit_details (
    dc_id, mode_of_transport, driver_name, vehicle_number,
    docket_number, eway_bill_number, reverse_charge, tax_type,
    taxable_value, cgst_amount, sgst_amount, igst_amount,
    round_off, total_value
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- Insert Line Items
INSERT INTO dc_line_items (
    dc_id, product_id, line_number, item_name, description,
    brand_model, uom, hsn_code, price, gst_percentage,
    quantity, taxable_value, gst_amount, total_amount, remarks
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- Fetch Template with Products
SELECT
    dt.id, dt.project_id, dt.template_number, dt.purpose,
    dt.default_bill_to_id, dt.default_ship_to_id,
    dtp.id as product_id, dtp.product_id, dtp.line_number,
    p.name, p.description, p.brand_model, p.uom, p.hsn_code,
    p.price, p.gst_percentage
FROM dc_templates dt
LEFT JOIN dc_template_products dtp ON dt.id = dtp.template_id
LEFT JOIN products p ON dtp.product_id = p.id
WHERE dt.id = ?
ORDER BY dtp.line_number;
```

### Step 5: Frontend Form Implementation

Create `views/dc/transit_create.html`:

**Form Structure:**
```html
<form hx-post="/projects/{project_id}/dc/transit"
      hx-target="#form-container"
      class="space-y-6">

    <!-- DC Header -->
    <div class="grid grid-cols-3 gap-4">
        <div>
            <label>DC Number</label>
            <input type="text" name="dc_number" value="{{.DCNumber}}" readonly>
        </div>
        <div>
            <label>DC Date</label>
            <input type="date" name="dc_date" required>
        </div>
        <div>
            <label>Purpose</label>
            <input type="text" name="purpose" value="{{.Purpose}}" readonly>
        </div>
    </div>

    <!-- Address Section -->
    <div class="grid grid-cols-2 gap-4">
        <div>
            <label>Bill To</label>
            <select name="bill_to_id"
                    hx-get="/api/projects/{{.ProjectID}}/addresses/search"
                    hx-trigger="search"
                    class="searchable-dropdown" required>
                {{range .BillToAddresses}}
                <option value="{{.ID}}" {{if eq .ID $.DefaultBillToID}}selected{{end}}>
                    {{.CompanyName}} - {{.City}}
                </option>
                {{end}}
            </select>
        </div>
        <div>
            <label>Ship To</label>
            <select name="ship_to_id" class="searchable-dropdown" required>
                {{range .ShipToAddresses}}
                <option value="{{.ID}}" {{if eq .ID $.DefaultShipToID}}selected{{end}}>
                    {{.CompanyName}} - {{.City}}
                </option>
                {{end}}
            </select>
        </div>
    </div>

    <!-- Transport Details -->
    <div class="grid grid-cols-4 gap-4">
        <div>
            <label>Mode of Transport</label>
            <select name="mode_of_transport" required>
                <option value="Road">Road</option>
                <option value="Rail">Rail</option>
                <option value="Air">Air</option>
                <option value="Ship">Ship</option>
            </select>
        </div>
        <div>
            <label>Driver/Transporter Name</label>
            <input type="text" name="driver_name" required>
        </div>
        <div>
            <label>Vehicle Number</label>
            <input type="text" name="vehicle_number" required>
        </div>
        <div>
            <label>Docket Number</label>
            <input type="text" name="docket_number">
        </div>
    </div>

    <div class="grid grid-cols-3 gap-4">
        <div>
            <label>E-Way Bill Number</label>
            <input type="text" name="eway_bill_number">
        </div>
        <div>
            <label>Reverse Charge</label>
            <select name="reverse_charge">
                <option value="N" selected>No</option>
                <option value="Y">Yes</option>
            </select>
        </div>
        <div>
            <label>Tax Type</label>
            <div class="flex gap-4">
                <label class="inline-flex items-center">
                    <input type="radio" name="tax_type" value="cgst_sgst" checked
                           hx-trigger="change"
                           hx-post="/api/dc/calculate-totals"
                           hx-target="#tax-summary">
                    <span class="ml-2">CGST + SGST</span>
                </label>
                <label class="inline-flex items-center">
                    <input type="radio" name="tax_type" value="igst"
                           hx-trigger="change"
                           hx-post="/api/dc/calculate-totals"
                           hx-target="#tax-summary">
                    <span class="ml-2">IGST</span>
                </label>
            </div>
        </div>
    </div>

    <!-- Product Lines Table -->
    <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
            <thead>
                <tr>
                    <th>S.No</th>
                    <th>Item Name</th>
                    <th>Description</th>
                    <th>UoM</th>
                    <th>HSN</th>
                    <th>Price</th>
                    <th>GST%</th>
                    <th>Serial Numbers</th>
                    <th>Qty</th>
                    <th>Taxable Value</th>
                    <th>GST Amount</th>
                    <th>Total</th>
                </tr>
            </thead>
            <tbody id="product-lines">
                {{range $index, $product := .Products}}
                <tr class="product-line" data-line-index="{{$index}}">
                    <td>{{add $index 1}}</td>
                    <td>
                        <input type="hidden" name="line_items[{{$index}}].product_id" value="{{$product.ID}}">
                        <span>{{$product.Name}}</span>
                    </td>
                    <td><span>{{$product.Description}}</span></td>
                    <td><span>{{$product.UOM}}</span></td>
                    <td><span>{{$product.HSNCode}}</span></td>
                    <td>
                        <input type="hidden" name="line_items[{{$index}}].price" value="{{$product.Price}}">
                        <span>₹{{printf "%.2f" $product.Price}}</span>
                    </td>
                    <td>
                        <input type="hidden" name="line_items[{{$index}}].gst_percentage" value="{{$product.GSTPercentage}}">
                        <span>{{$product.GSTPercentage}}%</span>
                    </td>
                    <td>
                        <textarea
                            name="line_items[{{$index}}].serial_numbers"
                            class="serial-input font-mono text-sm"
                            rows="3"
                            placeholder="Scan or enter serial numbers (one per line)"
                            hx-post="/api/dc/calculate-line"
                            hx-trigger="input changed delay:300ms"
                            hx-target="closest tr"
                            hx-include="closest tr"
                            hx-vals='{"line_index": "{{$index}}"}'
                        ></textarea>
                    </td>
                    <td>
                        <input type="number"
                               name="line_items[{{$index}}].quantity"
                               class="quantity-display"
                               value="0"
                               readonly>
                    </td>
                    <td class="taxable-value">₹0.00</td>
                    <td class="gst-amount">₹0.00</td>
                    <td class="line-total">₹0.00</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>

    <!-- Tax Summary Section -->
    <div id="tax-summary" class="bg-gray-50 p-4 rounded-lg">
        <div class="grid grid-cols-2 gap-4 max-w-md ml-auto">
            <div class="font-semibold">Taxable Value:</div>
            <div class="text-right" id="summary-taxable">₹0.00</div>

            <div class="font-semibold tax-cgst">CGST:</div>
            <div class="text-right tax-cgst" id="summary-cgst">₹0.00</div>

            <div class="font-semibold tax-sgst">SGST:</div>
            <div class="text-right tax-sgst" id="summary-sgst">₹0.00</div>

            <div class="font-semibold tax-igst hidden">IGST:</div>
            <div class="text-right tax-igst hidden" id="summary-igst">₹0.00</div>

            <div class="font-semibold">Round Off:</div>
            <div class="text-right" id="summary-roundoff">₹0.00</div>

            <div class="font-bold text-lg border-t pt-2">Invoice Value:</div>
            <div class="text-right font-bold text-lg border-t pt-2" id="summary-total">₹0.00</div>

            <div class="col-span-2 text-sm text-gray-600 mt-2">
                <strong>Amount in Words:</strong>
                <div id="amount-in-words" class="mt-1 italic">Zero Rupees Only</div>
            </div>
        </div>
    </div>

    <!-- Notes -->
    <div>
        <label>Notes</label>
        <textarea name="notes" rows="3" class="w-full"></textarea>
    </div>

    <!-- Action Buttons -->
    <div class="flex justify-end gap-3">
        <a href="/projects/{{.ProjectID}}/dc" class="btn-secondary">Cancel</a>
        <button type="submit" class="btn-primary">Save as Draft</button>
    </div>
</form>
```

### Step 6: JavaScript for Real-time Calculations

Create `static/js/dc_calculations.js`:

```javascript
// Auto-calculate quantity from serial numbers
document.addEventListener('input', function(e) {
    if (e.target.classList.contains('serial-input')) {
        const textarea = e.target;
        const row = textarea.closest('tr');
        const quantityInput = row.querySelector('.quantity-display');

        // Count non-empty lines
        const serialNumbers = textarea.value
            .split('\n')
            .map(s => s.trim())
            .filter(s => s.length > 0);

        quantityInput.value = serialNumbers.length;

        // Trigger calculation
        calculateLineTotal(row);
    }
});

function calculateLineTotal(row) {
    const price = parseFloat(row.querySelector('input[name*=".price"]').value) || 0;
    const gstPercentage = parseFloat(row.querySelector('input[name*=".gst_percentage"]').value) || 0;
    const quantity = parseInt(row.querySelector('.quantity-display').value) || 0;

    const taxableValue = price * quantity;
    const gstAmount = (taxableValue * gstPercentage) / 100;
    const total = taxableValue + gstAmount;

    row.querySelector('.taxable-value').textContent = `₹${taxableValue.toFixed(2)}`;
    row.querySelector('.gst-amount').textContent = `₹${gstAmount.toFixed(2)}`;
    row.querySelector('.line-total').textContent = `₹${total.toFixed(2)}`;

    // Recalculate grand totals
    calculateGrandTotals();
}

function calculateGrandTotals() {
    const taxType = document.querySelector('input[name="tax_type"]:checked').value;

    let totalTaxable = 0;
    let totalGST = 0;

    document.querySelectorAll('.product-line').forEach(row => {
        const taxableValue = parseFloat(row.querySelector('.taxable-value').textContent.replace('₹', '')) || 0;
        const gstAmount = parseFloat(row.querySelector('.gst-amount').textContent.replace('₹', '')) || 0;

        totalTaxable += taxableValue;
        totalGST += gstAmount;
    });

    // Calculate round-off
    const subtotal = totalTaxable + totalGST;
    const roundedTotal = Math.round(subtotal);
    const roundOff = roundedTotal - subtotal;

    // Update summary
    document.getElementById('summary-taxable').textContent = `₹${totalTaxable.toFixed(2)}`;

    if (taxType === 'cgst_sgst') {
        const cgst = totalGST / 2;
        const sgst = totalGST / 2;

        document.querySelectorAll('.tax-cgst').forEach(el => el.classList.remove('hidden'));
        document.querySelectorAll('.tax-sgst').forEach(el => el.classList.remove('hidden'));
        document.querySelectorAll('.tax-igst').forEach(el => el.classList.add('hidden'));

        document.getElementById('summary-cgst').textContent = `₹${cgst.toFixed(2)}`;
        document.getElementById('summary-sgst').textContent = `₹${sgst.toFixed(2)}`;
    } else {
        document.querySelectorAll('.tax-cgst').forEach(el => el.classList.add('hidden'));
        document.querySelectorAll('.tax-sgst').forEach(el => el.classList.add('hidden'));
        document.querySelectorAll('.tax-igst').forEach(el => el.classList.remove('hidden'));

        document.getElementById('summary-igst').textContent = `₹${totalGST.toFixed(2)}`;
    }

    document.getElementById('summary-roundoff').textContent = `₹${roundOff.toFixed(2)}`;
    document.getElementById('summary-total').textContent = `₹${roundedTotal.toFixed(2)}`;

    // Convert to words
    document.getElementById('amount-in-words').textContent = numberToWords(roundedTotal);
}

// Indian number to words conversion
function numberToWords(num) {
    if (num === 0) return 'Zero Rupees Only';

    const ones = ['', 'One', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine'];
    const tens = ['', '', 'Twenty', 'Thirty', 'Forty', 'Fifty', 'Sixty', 'Seventy', 'Eighty', 'Ninety'];
    const teens = ['Ten', 'Eleven', 'Twelve', 'Thirteen', 'Fourteen', 'Fifteen', 'Sixteen', 'Seventeen', 'Eighteen', 'Nineteen'];

    function convertTwoDigits(n) {
        if (n < 10) return ones[n];
        if (n >= 10 && n < 20) return teens[n - 10];
        return tens[Math.floor(n / 10)] + (n % 10 !== 0 ? ' ' + ones[n % 10] : '');
    }

    function convertThreeDigits(n) {
        if (n < 100) return convertTwoDigits(n);
        return ones[Math.floor(n / 100)] + ' Hundred' + (n % 100 !== 0 ? ' ' + convertTwoDigits(n % 100) : '');
    }

    const crore = Math.floor(num / 10000000);
    const lakh = Math.floor((num % 10000000) / 100000);
    const thousand = Math.floor((num % 100000) / 1000);
    const remainder = num % 1000;

    let words = '';

    if (crore > 0) words += convertTwoDigits(crore) + ' Crore ';
    if (lakh > 0) words += convertTwoDigits(lakh) + ' Lakh ';
    if (thousand > 0) words += convertTwoDigits(thousand) + ' Thousand ';
    if (remainder > 0) words += convertThreeDigits(remainder);

    return words.trim() + ' Rupees Only';
}

// Initialize calculations on page load
document.addEventListener('DOMContentLoaded', function() {
    calculateGrandTotals();
});
```

### Step 7: Searchable Dropdown Implementation

Add Select2 or custom HTMX-based searchable dropdown:

```html
<!-- Include Select2 CSS/JS or use HTMX approach -->
<script>
document.addEventListener('DOMContentLoaded', function() {
    // Initialize searchable dropdowns
    const searchableSelects = document.querySelectorAll('.searchable-dropdown');

    searchableSelects.forEach(select => {
        // Add search input above select
        const wrapper = document.createElement('div');
        wrapper.className = 'relative';
        select.parentNode.insertBefore(wrapper, select);
        wrapper.appendChild(select);

        const searchInput = document.createElement('input');
        searchInput.type = 'text';
        searchInput.placeholder = 'Type to search...';
        searchInput.className = 'w-full px-3 py-2 border border-gray-300 rounded-md mb-1';
        wrapper.insertBefore(searchInput, select);

        searchInput.addEventListener('input', function() {
            const filter = this.value.toLowerCase();
            const options = select.options;

            for (let i = 0; i < options.length; i++) {
                const text = options[i].text.toLowerCase();
                options[i].style.display = text.includes(filter) ? '' : 'none';
            }
        });
    });
});
</script>
```

### Step 8: Validation and Error Handling

Add validation in backend handler:

```go
func validateTransitDC(dc *TransitDC, details *TransitDetails, lineItems []DCLineItem) []string {
    errors := []string{}

    // Validate dates
    if dc.DCDate.IsZero() {
        errors = append(errors, "DC date is required")
    }

    // Validate addresses
    if dc.BillToID == 0 {
        errors = append(errors, "Bill To address is required")
    }
    if dc.ShipToID == 0 {
        errors = append(errors, "Ship To address is required")
    }

    // Validate transport details
    if details.ModeOfTransport == "" {
        errors = append(errors, "Mode of transport is required")
    }
    if details.DriverName == "" {
        errors = append(errors, "Driver/Transporter name is required")
    }
    if details.VehicleNumber == "" {
        errors = append(errors, "Vehicle number is required")
    }

    // Validate tax type
    if details.TaxType != "cgst_sgst" && details.TaxType != "igst" {
        errors = append(errors, "Invalid tax type")
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
            errors = append(errors, fmt.Sprintf("Line %d: Serial number count must match quantity", i+1))
        }
    }

    return errors
}
```

## Files to Create/Modify

### New Files

1. **handlers/transit_dc_handler.go**
   - NewTransitDCFromTemplate
   - CreateTransitDC
   - EditTransitDC
   - UpdateTransitDC
   - CalculateDCLineItem (HTMX endpoint)
   - CalculateDCTotals (HTMX endpoint)

2. **models/transit_dc.go**
   - TransitDC struct
   - TransitDetails struct
   - DCLineItem struct
   - CreateTransitDC method
   - UpdateTransitDC method
   - GetTransitDCByID method

3. **views/dc/transit_create.html**
   - Transit DC creation form
   - Product lines table
   - Serial number input areas
   - Tax summary section
   - HTMX attributes for real-time calculation

4. **views/dc/transit_edit.html**
   - Same as create but pre-populated with existing data
   - Only accessible for Draft status DCs

5. **static/js/dc_calculations.js**
   - calculateLineTotal function
   - calculateGrandTotals function
   - numberToWords function (Indian format)
   - Serial number input handler

6. **static/js/searchable_dropdown.js**
   - Initialize searchable dropdowns
   - Filter options based on search input

### Modified Files

1. **routes/dc_routes.go**
   - Add Transit DC routes
   - Add HTMX calculation endpoints

2. **database/migrations/003_create_dc_tables.sql**
   - Ensure delivery_challans table exists
   - Ensure dc_transit_details table exists
   - Ensure dc_line_items table exists
   - Add indexes on foreign keys

3. **views/dc_templates/detail.html**
   - Add "Issue Transit DC" button
   - Link to Transit DC creation route

4. **static/css/styles.css**
   - Add styles for serial number textarea
   - Add styles for tax summary section
   - Add styles for searchable dropdown

## API Routes/Endpoints

### Main Routes

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects/:project_id/dc/templates/:template_id/transit/new` | Show Transit DC creation form from template |
| POST | `/projects/:project_id/dc/transit` | Create new Transit DC |
| GET | `/projects/:project_id/dc/transit/:dc_id/edit` | Show Transit DC edit form (Draft only) |
| PUT | `/projects/:project_id/dc/transit/:dc_id` | Update Transit DC (Draft only) |

### HTMX Partial Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/dc/calculate-line` | Calculate single line item totals |
| POST | `/api/dc/calculate-totals` | Recalculate grand totals when tax type changes |
| GET | `/api/projects/:project_id/addresses/search` | Search project addresses (autocomplete) |

### Request/Response Examples

**POST /projects/1/dc/transit**

Request Body:
```json
{
  "dc_date": "2026-02-16",
  "bill_to_id": 5,
  "ship_to_id": 6,
  "mode_of_transport": "Road",
  "driver_name": "Rajesh Kumar",
  "vehicle_number": "MH12AB1234",
  "docket_number": "DOC123456",
  "eway_bill_number": "381234567890",
  "reverse_charge": "N",
  "tax_type": "cgst_sgst",
  "notes": "Handle with care",
  "line_items": [
    {
      "product_id": 10,
      "serial_numbers": "SN001\nSN002\nSN003",
      "quantity": 3,
      "price": 15000.00,
      "gst_percentage": 18.0
    }
  ]
}
```

Response:
```json
{
  "success": true,
  "dc_id": 42,
  "dc_number": "DC/2024-25/00042",
  "redirect": "/projects/1/dc/transit/42"
}
```

## Database Queries

### Create Transit DC

```sql
-- Insert main DC record
INSERT INTO delivery_challans (
    project_id, template_id, dc_number, dc_type, dc_date,
    purpose, bill_to_id, ship_to_id, notes, status,
    created_at, updated_at
) VALUES (
    ?, ?, ?, 'transit', ?,
    ?, ?, ?, ?, 'draft',
    CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);

-- Insert transit-specific details
INSERT INTO dc_transit_details (
    dc_id, mode_of_transport, driver_name, vehicle_number,
    docket_number, eway_bill_number, reverse_charge, tax_type,
    taxable_value, cgst_amount, sgst_amount, igst_amount,
    round_off, total_value
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?
);

-- Insert line items (loop)
INSERT INTO dc_line_items (
    dc_id, product_id, line_number, item_name, description,
    brand_model, uom, hsn_code, price, gst_percentage,
    quantity, taxable_value, gst_amount, total_amount, remarks
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?
);
```

### Fetch Transit DC with Details

```sql
SELECT
    dc.id, dc.project_id, dc.template_id, dc.dc_number, dc.dc_type,
    dc.dc_date, dc.purpose, dc.bill_to_id, dc.ship_to_id, dc.notes,
    dc.status, dc.created_at, dc.updated_at, dc.issued_at,

    td.mode_of_transport, td.driver_name, td.vehicle_number,
    td.docket_number, td.eway_bill_number, td.reverse_charge,
    td.tax_type, td.taxable_value, td.cgst_amount, td.sgst_amount,
    td.igst_amount, td.round_off, td.total_value,

    bt.company_name as bill_to_name, bt.address as bill_to_address,
    bt.city as bill_to_city, bt.state as bill_to_state,
    bt.pincode as bill_to_pincode, bt.gstin as bill_to_gstin,

    st.company_name as ship_to_name, st.address as ship_to_address,
    st.city as ship_to_city, st.state as ship_to_state,
    st.pincode as ship_to_pincode, st.gstin as ship_to_gstin

FROM delivery_challans dc
INNER JOIN dc_transit_details td ON dc.id = td.dc_id
LEFT JOIN project_addresses bt ON dc.bill_to_id = bt.id
LEFT JOIN project_addresses st ON dc.ship_to_id = st.id
WHERE dc.id = ? AND dc.dc_type = 'transit';
```

### Fetch Line Items

```sql
SELECT
    li.id, li.dc_id, li.product_id, li.line_number,
    li.item_name, li.description, li.brand_model,
    li.uom, li.hsn_code, li.price, li.gst_percentage,
    li.quantity, li.taxable_value, li.gst_amount,
    li.total_amount, li.remarks
FROM dc_line_items li
WHERE li.dc_id = ?
ORDER BY li.line_number;
```

### Search Project Addresses

```sql
SELECT
    id, company_name, address, city, state, pincode, gstin
FROM project_addresses
WHERE project_id = ?
  AND (
    company_name LIKE ? OR
    city LIKE ? OR
    state LIKE ?
  )
ORDER BY company_name
LIMIT 20;
```

## UI Components

### 1. Serial Number Textarea Component

**Purpose:** Optimized for barcode scanner input with auto-calculation

```html
<textarea
    class="serial-input font-mono text-sm w-full p-2 border rounded"
    rows="4"
    placeholder="Scan serial numbers (one per line)"
    data-line-index="{{$index}}"
></textarea>
```

**Features:**
- Monospace font for easy reading
- Auto-expands based on content
- Real-time quantity calculation
- Whitespace trimming
- Empty line filtering

### 2. Tax Type Radio Buttons

```html
<div class="flex gap-4">
    <label class="inline-flex items-center">
        <input type="radio" name="tax_type" value="cgst_sgst" checked>
        <span class="ml-2">CGST + SGST</span>
    </label>
    <label class="inline-flex items-center">
        <input type="radio" name="tax_type" value="igst">
        <span class="ml-2">IGST</span>
    </label>
</div>
```

**Behavior:**
- Toggle tax summary display
- Recalculate totals on change

### 3. Tax Summary Box

```html
<div class="bg-gray-50 p-4 rounded-lg border">
    <div class="grid grid-cols-2 gap-2 max-w-md ml-auto text-sm">
        <div class="font-semibold">Taxable Value:</div>
        <div class="text-right">₹<span id="summary-taxable">0.00</span></div>
        <!-- Additional rows -->
    </div>
</div>
```

**Features:**
- Right-aligned for readability
- Conditional display of CGST/SGST vs IGST
- Amount in words below totals

### 4. Searchable Dropdown for Addresses

```html
<div class="relative">
    <input type="text" placeholder="Type to search..." class="search-input">
    <select name="bill_to_id" class="searchable-dropdown">
        <option value="5">ABC Corp - Mumbai</option>
        <option value="6">XYZ Ltd - Delhi</option>
    </select>
</div>
```

**Features:**
- Filter-as-you-type
- Display company name + city
- Maintain selected value

### 5. Product Lines Table

```html
<table class="min-w-full divide-y divide-gray-200">
    <thead class="bg-gray-50">
        <tr>
            <th class="px-2 py-3 text-left text-xs font-medium text-gray-500 uppercase">S.No</th>
            <!-- More headers -->
        </tr>
    </thead>
    <tbody class="bg-white divide-y divide-gray-200">
        <!-- Product rows -->
    </tbody>
    <tfoot class="bg-gray-50">
        <tr class="font-bold">
            <td colspan="9" class="text-right px-2 py-3">Total:</td>
            <td id="footer-taxable">₹0.00</td>
            <td id="footer-gst">₹0.00</td>
            <td id="footer-total">₹0.00</td>
        </tr>
    </tfoot>
</table>
```

## Testing Checklist

### Functional Testing

- [ ] Click "Issue Transit DC" from DC Template detail page
- [ ] Verify DC number is auto-generated correctly
- [ ] Verify purpose is pre-filled from template
- [ ] Verify default Bill To and Ship To are selected
- [ ] Verify product lines are populated from template with pricing
- [ ] Test barcode scanner input in serial number field
- [ ] Verify quantity auto-calculates from serial number count
- [ ] Test manual serial number entry (newline-separated)
- [ ] Verify whitespace trimming in serial numbers
- [ ] Verify empty lines are ignored in serial count
- [ ] Test line item calculations (taxable value, GST amount, total)
- [ ] Test CGST+SGST calculation mode
- [ ] Test IGST calculation mode
- [ ] Verify tax type toggle updates summary correctly
- [ ] Test round-off calculation
- [ ] Verify amount in words conversion (Indian format: Lakhs, Crores)
- [ ] Test searchable dropdown for Bill To addresses
- [ ] Test searchable dropdown for Ship To addresses
- [ ] Test Mode of Transport dropdown
- [ ] Test optional fields (Docket Number, E-Way Bill)
- [ ] Test Reverse Charge toggle
- [ ] Test Notes field
- [ ] Submit form and verify DC is created with status='draft'
- [ ] Verify all data is saved correctly in database
- [ ] Test edit functionality for Draft DC
- [ ] Verify cannot edit Issued DC

### Edge Cases

- [ ] Test with 0 products in template (should prevent creation)
- [ ] Test with very long serial number list (100+ items)
- [ ] Test with special characters in serial numbers
- [ ] Test with duplicate serial numbers within same line
- [ ] Test with decimal quantities (should be integer only)
- [ ] Test with negative quantities
- [ ] Test form validation with missing required fields
- [ ] Test with invalid DC date (future date)
- [ ] Test with invalid vehicle number format
- [ ] Test concurrent DC creation (number generation)
- [ ] Test with very large amounts (overflow checking)
- [ ] Test round-off edge cases (e.g., ₹0.50, ₹0.49)
- [ ] Test amount in words for edge values (0, 1, 99, 100, 1000, 1 lakh, 1 crore)

### UI/UX Testing

- [ ] Verify form layout matches mockup 12-create-dc.html
- [ ] Test responsive design on different screen sizes
- [ ] Verify serial number textarea is easily scannable
- [ ] Test keyboard navigation through form
- [ ] Verify loading states during HTMX requests
- [ ] Test error message display
- [ ] Verify success message after creation
- [ ] Test Cancel button returns to correct page
- [ ] Verify real-time calculation happens smoothly
- [ ] Test dropdown search performance with 100+ addresses
- [ ] Verify all currency values display with 2 decimals
- [ ] Verify Indian Rupee symbol (₹) displays correctly

### Performance Testing

- [ ] Test form load time with 50 products in template
- [ ] Test calculation speed with 20 line items
- [ ] Test serial number validation with 500 serials
- [ ] Verify HTMX requests don't cause UI lag
- [ ] Test database query performance for address search

### Integration Testing

- [ ] Verify DC number sequence increments correctly
- [ ] Verify template data is accurately copied
- [ ] Verify project addresses are fetched correctly
- [ ] Test with multiple users creating DCs simultaneously
- [ ] Verify serial numbers link to Phase 13 table (when implemented)

## Acceptance Criteria

### Must Have

1. **Template Integration**
   - ✅ Transit DC can be created from any DC Template
   - ✅ DC number is auto-generated using Phase 10 logic
   - ✅ Purpose is pre-filled from template
   - ✅ Product lines are auto-populated from template with pricing

2. **Form Functionality**
   - ✅ All required fields are present and validated
   - ✅ Bill To and Ship To dropdowns are searchable
   - ✅ Default Bill To and Ship To are pre-selected
   - ✅ Mode of Transport dropdown works correctly
   - ✅ Optional fields (Docket, E-Way Bill) are optional
   - ✅ Tax type radio buttons toggle correctly

3. **Serial Number Management**
   - ✅ Serial number textarea accepts barcode scanner input
   - ✅ Serial numbers are newline-separated
   - ✅ Whitespace is trimmed, empty lines ignored
   - ✅ Quantity auto-calculates from serial count
   - ✅ Quantity display is read-only

4. **Calculations**
   - ✅ Line item taxable value = price × quantity
   - ✅ Line item GST amount = taxable value × GST%
   - ✅ Line item total = taxable value + GST amount
   - ✅ Grand totals calculate correctly
   - ✅ CGST+SGST mode splits GST 50/50
   - ✅ IGST mode shows single IGST value
   - ✅ Round-off calculated correctly
   - ✅ Amount in words displays in Indian format

5. **Data Persistence**
   - ✅ DC is saved with status='draft'
   - ✅ All form data is saved to delivery_challans table
   - ✅ Transit details saved to dc_transit_details table
   - ✅ Line items saved to dc_line_items table
   - ✅ Serial numbers prepared for Phase 13 integration

6. **UI/UX**
   - ✅ Form matches mockup 12-create-dc.html
   - ✅ Real-time calculations work smoothly
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
   - ✅ Form loads in < 2 seconds with 50 products
   - ✅ Calculations update in < 500ms
   - ✅ Address search returns results in < 1 second

3. **Accessibility**
   - ✅ Form is keyboard navigable
   - ✅ Labels are associated with inputs
   - ✅ Error messages are screen-reader friendly

### Nice to Have

1. **Advanced Features**
   - ⭕ Auto-save draft every 60 seconds
   - ⭕ Duplicate DC from existing DC
   - ⭕ Bulk serial number validation (check format)
   - ⭕ Export serial numbers to clipboard
   - ⭕ Recently used addresses quick select

2. **UX Enhancements**
   - ⭕ Toast notifications for successful save
   - ⭕ Undo last serial number scan
   - ⭕ Keyboard shortcuts (e.g., Ctrl+S to save)
   - ⭕ Progress indicator during save

---

## Notes

- This phase focuses on Transit DC creation in Draft status
- Phase 14 will handle the "Issue DC" workflow
- Phase 13 will implement full serial number validation
- Phase 15 will implement the print layout
- Serial number uniqueness validation is deferred to Phase 13
- Amount in words should handle up to 10 crores (₹100,000,000)
- Tax calculation assumes prices are exclusive of GST
- Round-off uses standard rounding (0.5 rounds up)

## Dependencies

- **Phase 10:** DC Templates must be implemented
- **Phase 9:** DC number generation logic
- **Database:** delivery_challans, dc_transit_details, dc_line_items tables must exist
- **Frontend:** HTMX and Tailwind CSS must be configured

## Next Steps

After Phase 11 completion:
1. Proceed to Phase 12 (Official DC Creation)
2. Implement Phase 13 (Serial Number Validation)
3. Implement Phase 14 (DC Lifecycle - Issue & Lock)
4. Implement Phase 15 (Transit DC View & Print)
