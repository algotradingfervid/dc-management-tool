# Phase 16: Official DC View & Print Layout

## Overview
Implement the official Delivery Challan view and print layout matching the mockup `15-official-dc-detail.html`. This view displays a formal DC format with company header, DC details, product information without pricing, and dual signature blocks for FSSPL representative and department official. The layout is optimized for A4 printing with clean print CSS.

**Tech Stack:**
- Go + Gin backend
- HTMX + Tailwind frontend
- SQLite database
- Print CSS for browser printing

## Prerequisites
- Phase 14 (Official DC Creation) completed
- Phase 15 (Transit DC View) completed
- Delivery Challans table with all required fields
- Products and Serial Numbers tables
- Addresses table with Ship To information
- Template selection system in place
- Understanding of print CSS and A4 layout

## Goals
1. Create official DC detail view matching the mockup design
2. Display company header with full legal information
3. Show DC details including mandal/ULB extracted from Ship To address
4. Display product table without pricing information
5. Implement acknowledgement section with dual signature blocks
6. Create print-optimized CSS for A4 output
7. Add browser print functionality
8. Support company signature image upload/display

## Detailed Implementation Steps

### Step 1: Database Schema Review
Verify the delivery_challans table has all required fields for official DC:
- `template_type` (for purpose field)
- `acknowledgement_text`
- `company_signature_path`
- `company_rep_name`
- `company_rep_designation`
- `company_rep_mobile`
- `receipt_date`
- `official_name`
- `official_designation`
- `official_mobile`

### Step 2: Backend - Official DC Detail Handler
Create handler to fetch official DC with all related data:
- DC basic information
- Project/Tender/PO reference details
- Ship To address (to extract Mandal/ULB and Mandal Code)
- Products with serial numbers
- Signature information

### Step 3: Mandal/ULB Code Extraction Logic
Implement logic to parse Ship To address and extract:
- District name
- Mandal or ULB name
- Mandal Code (if stored in address or DC)

### Step 4: Frontend - Official DC View Template
Create HTML template with sections:
- Company header block
- DC title and basic details
- Project reference block
- Purpose field
- Issued To section
- Product table (no pricing)
- Acknowledgement statement
- Receipt date line
- Dual signature block

### Step 5: Print CSS Implementation
Create print-specific styles:
- A4 page size and margins
- Page break controls
- Hide print button and navigation
- Optimize font sizes and spacing
- Ensure signature blocks fit properly
- Header on first page only

### Step 6: Company Signature Management
Implement company signature upload/storage:
- Image upload endpoint
- Store signature image path in database
- Display signature in left signature block
- Fallback if no signature uploaded

### Step 7: Print Functionality
Add browser print button with JavaScript:
- Trigger `window.print()`
- Ensure print CSS is applied
- Test across browsers

### Step 8: Testing & Refinement
Test print output:
- Verify A4 layout
- Check all fields render correctly
- Test with various data lengths
- Ensure signature blocks align properly
- Test browser print functionality

## Files to Create/Modify

### Backend Files

**handlers/dc_handler.go** (modify)
```go
// Add official DC detail handler
func (h *DCHandler) GetOfficialDCDetail(c *gin.Context) {
    dcID := c.Param("id")

    // Fetch DC with all related data
    dc, err := h.dcService.GetDCWithDetails(dcID)
    if err != nil {
        c.HTML(http.StatusNotFound, "404.html", nil)
        return
    }

    // Verify this is an official DC
    if dc.Type != "official" {
        c.Redirect(http.StatusFound, "/dcs/"+dcID+"/transit")
        return
    }

    // Extract mandal/ULB info from Ship To address
    mandalInfo := extractMandalInfo(dc.ShipToAddress)

    c.HTML(http.StatusOK, "official-dc-detail.html", gin.H{
        "DC":         dc,
        "MandalInfo": mandalInfo,
        "Company":    getCompanyInfo(),
    })
}

// Helper to extract mandal information from address
func extractMandalInfo(address Address) map[string]string {
    // Parse address.Line2 or address.Notes for mandal code
    // Extract district from address.District
    // Extract mandal/ULB from address.Line1 or custom field
    return map[string]string{
        "District":    address.District,
        "MandalName":  parseMandalName(address),
        "MandalCode":  parseMandalCode(address),
    }
}

// Company information (hardcoded as per requirements)
func getCompanyInfo() map[string]string {
    return map[string]string{
        "Name":    "FERVID SMART SOLUTIONS PRIVATE LIMITED",
        "Address": "Plot No 14/2, Dwaraka Park View, 1st Floor, Sector-1, HUDA Techno Enclave, Madhapur, Hyderabad, Telangana 500081",
        "Email":   "odishaprojects@fervidsmart.com",
        "GSTIN":   "36AACCF9742K1Z8",
        "CIN":     "U45100TG2016PTC113752",
    }
}
```

**handlers/signature_handler.go** (create new)
```go
package handlers

import (
    "net/http"
    "path/filepath"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type SignatureHandler struct {
    uploadDir string
}

func NewSignatureHandler(uploadDir string) *SignatureHandler {
    return &SignatureHandler{uploadDir: uploadDir}
}

// Upload company signature
func (h *SignatureHandler) UploadCompanySignature(c *gin.Context) {
    file, err := c.FormFile("signature")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
        return
    }

    // Validate file type
    ext := filepath.Ext(file.Filename)
    if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Only PNG/JPG images allowed"})
        return
    }

    // Generate unique filename
    filename := uuid.New().String() + ext
    filepath := filepath.Join(h.uploadDir, "signatures", filename)

    // Save file
    if err := c.SaveUploadedFile(file, filepath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "path": "/uploads/signatures/" + filename,
    })
}
```

**services/dc_service.go** (modify)
```go
// Add method to get DC with all details for official view
func (s *DCService) GetDCWithDetails(dcID string) (*DCDetail, error) {
    var dc DCDetail

    query := `
        SELECT
            dc.*,
            p.name as project_name,
            p.tender_reference,
            p.po_number,
            p.po_date,
            sa.line1 as ship_line1,
            sa.line2 as ship_line2,
            sa.city as ship_city,
            sa.state as ship_state,
            sa.pincode as ship_pincode,
            sa.district as ship_district
        FROM delivery_challans dc
        LEFT JOIN projects p ON dc.project_id = p.id
        LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
        WHERE dc.id = ?
    `

    err := s.db.QueryRow(query, dcID).Scan(/* scan all fields */)
    if err != nil {
        return nil, err
    }

    // Fetch products
    dc.Products, err = s.getProductsForDC(dcID)
    if err != nil {
        return nil, err
    }

    return &dc, nil
}

func (s *DCService) getProductsForDC(dcID string) ([]DCProduct, error) {
    query := `
        SELECT
            p.id,
            p.item_name,
            p.description,
            p.brand_model,
            p.quantity,
            p.remarks,
            GROUP_CONCAT(sn.serial_number, ', ') as serial_numbers
        FROM products p
        LEFT JOIN serial_numbers sn ON p.id = sn.product_id
        WHERE p.delivery_challan_id = ?
        GROUP BY p.id
        ORDER BY p.id
    `

    rows, err := s.db.Query(query, dcID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var products []DCProduct
    for rows.Next() {
        var product DCProduct
        err := rows.Scan(/* scan fields */)
        if err != nil {
            return nil, err
        }
        products = append(products, product)
    }

    return products, nil
}
```

### Frontend Files

**templates/official-dc-detail.html** (create new)
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Official DC - {{ .DC.DCNumber }}</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="stylesheet" href="/static/css/print.css">
</head>
<body class="bg-gray-50">
    <!-- Navigation (hidden on print) -->
    <nav class="bg-white shadow-sm print:hidden">
        <div class="max-w-7xl mx-auto px-4 py-4">
            <a href="/projects/{{ .DC.ProjectID }}/dcs" class="text-blue-600 hover:text-blue-800">
                ← Back to DCs
            </a>
        </div>
    </nav>

    <!-- Print Button (hidden on print) -->
    <div class="max-w-4xl mx-auto px-4 py-4 print:hidden">
        <button onclick="window.print()" class="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700">
            🖨️ Print DC
        </button>
    </div>

    <!-- Official DC Document -->
    <div class="max-w-4xl mx-auto bg-white shadow-lg print:shadow-none" id="official-dc">
        <!-- Company Header -->
        <div class="border-b-2 border-gray-800 pb-4 mb-6">
            <h1 class="text-2xl font-bold text-center text-gray-900">
                {{ .Company.Name }}
            </h1>
            <p class="text-center text-sm text-gray-700 mt-2">
                {{ .Company.Address }}
            </p>
            <div class="flex justify-center gap-8 text-xs text-gray-600 mt-2">
                <span><strong>Email:</strong> {{ .Company.Email }}</span>
                <span><strong>GSTIN:</strong> {{ .Company.GSTIN }}</span>
                <span><strong>CIN:</strong> {{ .Company.CIN }}</span>
            </div>
        </div>

        <!-- DC Title -->
        <h2 class="text-xl font-bold text-center text-gray-900 mb-6 uppercase tracking-wide">
            DELIVERY CHALLAN
        </h2>

        <!-- DC Details Grid -->
        <div class="grid grid-cols-2 gap-4 mb-6 text-sm">
            <div>
                <p><strong>DC Number:</strong> {{ .DC.DCNumber }}</p>
                <p><strong>DC Date:</strong> {{ formatDate .DC.DCDate }}</p>
            </div>
            <div>
                <p><strong>Mandal/ULB Name:</strong> {{ .MandalInfo.MandalName }}</p>
                <p><strong>Mandal Code:</strong> {{ .MandalInfo.MandalCode }}</p>
            </div>
        </div>

        <!-- Project Reference Block -->
        <div class="bg-gray-50 p-4 rounded-lg mb-6 text-sm">
            <h3 class="font-semibold mb-2">Project Details:</h3>
            <div class="grid grid-cols-2 gap-2">
                <p><strong>Project Name:</strong> {{ .DC.ProjectName }}</p>
                <p><strong>Tender Reference:</strong> {{ .DC.TenderReference }}</p>
                <p><strong>PO Number:</strong> {{ .DC.PONumber }}</p>
                <p><strong>PO Date:</strong> {{ formatDate .DC.PODate }}</p>
            </div>
        </div>

        <!-- Purpose -->
        <div class="mb-6 text-sm">
            <p><strong>Purpose:</strong> {{ .DC.TemplateType }}</p>
        </div>

        <!-- Issued To -->
        <div class="mb-6 text-sm">
            <h3 class="font-semibold mb-2">Issued To:</h3>
            <p><strong>District:</strong> {{ .MandalInfo.District }}</p>
            <p><strong>Mandal/ULB:</strong> {{ .MandalInfo.MandalName }}</p>
            <p class="mt-2">{{ .DC.ShipToAddress }}</p>
        </div>

        <!-- Product Table -->
        <table class="w-full border-collapse border border-gray-300 mb-6 text-sm">
            <thead>
                <tr class="bg-gray-100">
                    <th class="border border-gray-300 px-2 py-2">S.No</th>
                    <th class="border border-gray-300 px-2 py-2">Item Name</th>
                    <th class="border border-gray-300 px-2 py-2">Description</th>
                    <th class="border border-gray-300 px-2 py-2">Brand/Model No</th>
                    <th class="border border-gray-300 px-2 py-2">Quantity</th>
                    <th class="border border-gray-300 px-2 py-2">Serial Number</th>
                    <th class="border border-gray-300 px-2 py-2">Remarks</th>
                </tr>
            </thead>
            <tbody>
                {{ range $index, $product := .DC.Products }}
                <tr>
                    <td class="border border-gray-300 px-2 py-2 text-center">{{ add $index 1 }}</td>
                    <td class="border border-gray-300 px-2 py-2">{{ $product.ItemName }}</td>
                    <td class="border border-gray-300 px-2 py-2">{{ $product.Description }}</td>
                    <td class="border border-gray-300 px-2 py-2">{{ $product.BrandModel }}</td>
                    <td class="border border-gray-300 px-2 py-2 text-center">{{ $product.Quantity }}</td>
                    <td class="border border-gray-300 px-2 py-2">{{ $product.SerialNumbers }}</td>
                    <td class="border border-gray-300 px-2 py-2">{{ $product.Remarks }}</td>
                </tr>
                {{ end }}
            </tbody>
        </table>

        <!-- Acknowledgement -->
        <div class="mb-6 text-sm">
            <p class="italic">
                It is certified that the material is received in good condition.
            </p>
        </div>

        <!-- Receipt Date -->
        <div class="mb-8 text-sm">
            <p><strong>Date of Receipt:</strong> _______________________</p>
        </div>

        <!-- Dual Signature Block -->
        <div class="grid grid-cols-2 gap-8 mb-8">
            <!-- FSSPL Representative -->
            <div class="border border-gray-300 p-4">
                <h4 class="font-semibold text-center mb-4">FSSPL Representative</h4>
                <div class="text-center mb-4">
                    {{ if .DC.CompanySignaturePath }}
                    <img src="{{ .DC.CompanySignaturePath }}" alt="Company Signature" class="mx-auto h-16">
                    {{ else }}
                    <div class="h-16 border-b border-gray-400 mb-2"></div>
                    {{ end }}
                </div>
                <p class="text-sm"><strong>Name:</strong> {{ .DC.CompanyRepName }}</p>
                <p class="text-sm"><strong>Designation:</strong> {{ .DC.CompanyRepDesignation }}</p>
                <p class="text-sm"><strong>Mobile:</strong> {{ .DC.CompanyRepMobile }}</p>
            </div>

            <!-- Department Official -->
            <div class="border border-gray-300 p-4">
                <h4 class="font-semibold text-center mb-4">Department Official</h4>
                <div class="text-center mb-4">
                    <div class="h-16 border-b border-gray-400 mb-2"></div>
                    <p class="text-xs text-gray-600">Signature with Seal and Date</p>
                </div>
                <p class="text-sm"><strong>Name:</strong> _______________________</p>
                <p class="text-sm"><strong>Designation:</strong> _______________________</p>
                <p class="text-sm"><strong>Mobile:</strong> _______________________</p>
            </div>
        </div>
    </div>

    <script>
        // Helper function for template
        function add(a, b) {
            return a + b;
        }
    </script>
</body>
</html>
```

**static/css/print.css** (create new)
```css
@media print {
    /* Page setup */
    @page {
        size: A4;
        margin: 1cm;
    }

    /* Hide non-printable elements */
    .print\:hidden,
    nav,
    button {
        display: none !important;
    }

    /* Reset page styles */
    body {
        margin: 0;
        padding: 0;
        background: white !important;
    }

    /* Main document */
    #official-dc {
        box-shadow: none !important;
        padding: 1cm;
        width: 100%;
        max-width: none;
    }

    /* Typography */
    body {
        font-size: 10pt;
        line-height: 1.4;
        color: black;
    }

    h1 {
        font-size: 18pt;
        margin-bottom: 0.5cm;
    }

    h2 {
        font-size: 14pt;
        margin-bottom: 0.5cm;
    }

    h3 {
        font-size: 11pt;
    }

    /* Tables */
    table {
        page-break-inside: avoid;
        border-collapse: collapse;
        width: 100%;
    }

    th, td {
        border: 1px solid #000;
        padding: 4pt;
    }

    th {
        background-color: #f0f0f0 !important;
        -webkit-print-color-adjust: exact;
        print-color-adjust: exact;
    }

    /* Signature blocks */
    .grid-cols-2 {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 1cm;
    }

    /* Prevent page breaks */
    .border-b-2,
    .bg-gray-50 {
        page-break-inside: avoid;
    }

    /* Ensure borders print */
    .border,
    .border-gray-300,
    .border-gray-800 {
        border-color: #000 !important;
    }

    /* Ensure backgrounds print */
    .bg-gray-50,
    .bg-gray-100 {
        -webkit-print-color-adjust: exact;
        print-color-adjust: exact;
    }

    /* Signature image sizing */
    img {
        max-width: 100%;
        height: auto;
    }
}
```

### Database Migration

**migrations/016_add_official_dc_fields.sql** (create new)
```sql
-- Add fields for official DC signature and acknowledgement

ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS acknowledgement_text TEXT;
ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS company_signature_path VARCHAR(500);
ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS company_rep_name VARCHAR(255);
ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS company_rep_designation VARCHAR(255);
ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS company_rep_mobile VARCHAR(20);
ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS receipt_date DATE;
ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS official_name VARCHAR(255);
ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS official_designation VARCHAR(255);
ALTER TABLE delivery_challans ADD COLUMN IF NOT EXISTS official_mobile VARCHAR(20);

-- Add mandal code to addresses if needed
ALTER TABLE addresses ADD COLUMN IF NOT EXISTS mandal_code VARCHAR(50);
```

## API Routes/Endpoints

### Route Definitions

**main.go** (modify)
```go
// Official DC routes
dcGroup := r.Group("/dcs")
{
    dcGroup.GET("/:id/official", dcHandler.GetOfficialDCDetail)
}

// Signature upload
r.POST("/api/signatures/upload", signatureHandler.UploadCompanySignature)
```

### Endpoint Details

| Method | Endpoint | Description | Response |
|--------|----------|-------------|----------|
| GET | `/dcs/:id/official` | Get official DC detail view | HTML page |
| POST | `/api/signatures/upload` | Upload company signature image | JSON with path |

## Database Queries

### Get Official DC with All Details
```sql
SELECT
    dc.id,
    dc.dc_number,
    dc.dc_date,
    dc.type,
    dc.template_type,
    dc.status,
    dc.acknowledgement_text,
    dc.company_signature_path,
    dc.company_rep_name,
    dc.company_rep_designation,
    dc.company_rep_mobile,
    dc.receipt_date,
    dc.official_name,
    dc.official_designation,
    dc.official_mobile,
    p.id as project_id,
    p.name as project_name,
    p.tender_reference,
    p.po_number,
    p.po_date,
    sa.line1 as ship_line1,
    sa.line2 as ship_line2,
    sa.city as ship_city,
    sa.state as ship_state,
    sa.pincode as ship_pincode,
    sa.district as ship_district,
    sa.mandal_code
FROM delivery_challans dc
LEFT JOIN projects p ON dc.project_id = p.id
LEFT JOIN addresses sa ON dc.ship_to_id = sa.id
WHERE dc.id = ? AND dc.type = 'official';
```

### Get Products for Official DC (No Pricing)
```sql
SELECT
    p.id,
    p.item_name,
    p.description,
    p.brand_model,
    p.quantity,
    p.remarks,
    GROUP_CONCAT(sn.serial_number, ', ') as serial_numbers
FROM products p
LEFT JOIN serial_numbers sn ON p.id = sn.product_id
WHERE p.delivery_challan_id = ?
GROUP BY p.id
ORDER BY p.id;
```

### Update Company Representative Info
```sql
UPDATE delivery_challans
SET
    company_signature_path = ?,
    company_rep_name = ?,
    company_rep_designation = ?,
    company_rep_mobile = ?
WHERE id = ?;
```

## UI Components

### Component Breakdown

1. **Company Header Component**
   - Company name (large, bold, centered)
   - Full address (centered, smaller text)
   - Email, GSTIN, CIN in row

2. **DC Title Component**
   - "DELIVERY CHALLAN" in uppercase
   - Centered, bold, large font
   - Border separation from header

3. **DC Details Grid**
   - 2-column layout
   - DC Number, DC Date, Mandal Name, Mandal Code
   - Clean typography

4. **Project Reference Block**
   - Light background box
   - Project name, tender ref, PO number, PO date
   - Grid layout

5. **Purpose Field**
   - Simple text line
   - Template type value

6. **Issued To Section**
   - District and Mandal/ULB
   - Full Ship To address

7. **Product Table Component**
   - 7 columns (S.No, Item, Description, Brand, Qty, Serial, Remarks)
   - NO pricing columns
   - Clean borders, alternating row colors

8. **Acknowledgement Component**
   - Italicized certification text
   - Receipt date line with blank

9. **Dual Signature Block**
   - 2-column grid
   - Left: FSSPL rep with signature image
   - Right: Department official (blank for manual)
   - Border around each block
   - Name, Designation, Mobile fields

10. **Print Button**
    - Positioned outside document
    - Triggers browser print
    - Hidden on print

### Tailwind Classes Used
- Layout: `max-w-4xl`, `mx-auto`, `grid`, `grid-cols-2`
- Spacing: `p-4`, `mb-6`, `gap-4`
- Typography: `text-xl`, `font-bold`, `text-center`
- Borders: `border`, `border-gray-300`
- Backgrounds: `bg-gray-50`, `bg-white`
- Print utilities: `print:hidden`, `print:shadow-none`

## Testing Checklist

### Functional Testing
- [ ] Official DC detail page loads successfully
- [ ] Company header displays all information correctly
- [ ] DC number and date display correctly
- [ ] Mandal/ULB name and code extracted from Ship To address
- [ ] Project reference block shows all details
- [ ] Purpose field displays template type
- [ ] Issued To section shows district and mandal
- [ ] Product table displays all products with correct columns
- [ ] NO pricing information is shown
- [ ] Serial numbers display correctly (comma-separated)
- [ ] Acknowledgement text displays
- [ ] Receipt date line is present
- [ ] Dual signature blocks render correctly
- [ ] Company signature image displays if uploaded
- [ ] Company rep info displays (name, designation, mobile)
- [ ] Department official section has blank fields for manual entry

### Print Testing
- [ ] Print button triggers browser print dialog
- [ ] Print button hidden in print preview
- [ ] Navigation hidden in print preview
- [ ] A4 page size is correct
- [ ] Margins are appropriate (1cm)
- [ ] All content fits on page (or breaks appropriately)
- [ ] Tables don't break awkwardly across pages
- [ ] Borders print correctly
- [ ] Background colors print correctly
- [ ] Font sizes are readable when printed
- [ ] Signature blocks align properly
- [ ] Company signature image prints at correct size

### Cross-Browser Testing
- [ ] Chrome print preview renders correctly
- [ ] Firefox print preview renders correctly
- [ ] Safari print preview renders correctly
- [ ] Edge print preview renders correctly

### Data Validation Testing
- [ ] Handle missing Ship To address gracefully
- [ ] Handle missing mandal code gracefully
- [ ] Handle DCs with no products
- [ ] Handle DCs with many products (pagination/page breaks)
- [ ] Handle long product descriptions
- [ ] Handle multiple serial numbers per product
- [ ] Handle missing company signature
- [ ] Handle missing company rep information

### Edge Cases
- [ ] DC with only 1 product
- [ ] DC with 20+ products (multiple pages)
- [ ] Product with no serial numbers
- [ ] Product with many serial numbers (line wrapping)
- [ ] Very long project names
- [ ] Very long addresses
- [ ] Missing project reference details

## Acceptance Criteria

### Must Have
1. ✅ Official DC detail view matches mockup design
2. ✅ Company header displays: FERVID SMART SOLUTIONS PRIVATE LIMITED, full address, email, GSTIN, CIN
3. ✅ "DELIVERY CHALLAN" title is prominent and centered
4. ✅ DC Number and DC Date display correctly
5. ✅ Mandal/ULB Name and Mandal Code extracted from Ship To address
6. ✅ Project reference block shows: Project Name, Tender Reference, PO Number, PO Date
7. ✅ Purpose field displays template type
8. ✅ "Issued To" section shows District and Mandal/ULB from Ship To
9. ✅ Product table has columns: S.No, Item Name, Description, Brand/Model No, Quantity, Serial Number, Remarks
10. ✅ Product table does NOT show any pricing information
11. ✅ Acknowledgement statement: "It is certified that the material is received in good condition."
12. ✅ Date of Receipt line is present
13. ✅ Dual signature block with two columns:
    - Left: FSSPL Representative with signature image, name, designation, mobile
    - Right: Department Official with blank signature area, name, designation, mobile fields
14. ✅ Print CSS creates clean A4 output
15. ✅ Browser print button is functional
16. ✅ Print preview hides navigation and print button
17. ✅ All borders and styling print correctly

### Nice to Have
1. ⭐ Company signature upload interface (admin panel)
2. ⭐ Signature management (replace/delete signature)
3. ⭐ Template customization for acknowledgement text
4. ⭐ Print preview before printing
5. ⭐ Save as PDF client-side option
6. ⭐ Email DC functionality
7. ⭐ QR code with DC verification link
8. ⭐ Watermark for draft DCs

### Performance Criteria
- Page load time < 1 second
- Print generation < 2 seconds
- Responsive on different screen sizes (before print)
- Image optimization for signature

### Security Criteria
- Only authorized users can view official DCs
- Signature upload restricted to admin users
- File upload validation (type, size)
- Prevent directory traversal in signature paths

### Accessibility Criteria
- Semantic HTML structure
- Proper heading hierarchy
- Print-friendly contrast ratios
- Keyboard navigation for print button

---

## Notes
- Company information is hardcoded as specified
- Mandal code extraction logic may need customization based on address format
- Consider adding a settings page for company signature management
- Future enhancement: allow template customization per project
- Consider adding a "Download as PDF" option in Phase 17
