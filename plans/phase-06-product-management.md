# Phase 6: Product Management

## Overview
This phase implements product management functionality at the project level. Products are defined with fixed details including item information, pricing, and GST rates. The system provides a comprehensive interface for adding, editing, viewing, and deleting products within each project.

## Prerequisites
- Phase 1: Project Setup (complete)
- Phase 2-3: Database schema and project model established
- Phase 4-5: Basic project management functionality (list and detail views)

## Goals
- Display products list in tabular format matching mockup 06-products.html
- Enable adding products via slide-over panel form
- Support inline editing of product details
- Implement product deletion with safety checks
- Validate product data (HSN code format, price ranges, GST percentages)
- Prevent deletion of products used in DC templates
- Store product data in normalized SQLite database

## Detailed Implementation Steps

### 1. Database Schema Setup
1.1. Create products table migration file
   - Define table with all required columns
   - Add foreign key constraint to projects table
   - Add indexes on project_id and item_name for query performance
   - Add created_at and updated_at timestamps

1.2. Add validation constraints
   - Ensure per_unit_price is positive
   - Ensure gst_percentage is between 0 and 100
   - Ensure HSN code follows valid format (6-8 digits)

1.3. Create migration rollback logic

### 2. Backend Model Implementation
2.1. Create Product struct in models/product.go
   - Define struct fields matching database columns
   - Add JSON tags for API responses
   - Add validation tags using validator package

2.2. Implement CRUD methods
   - GetProductsByProjectID(projectID int) ([]Product, error)
   - GetProductByID(id int) (*Product, error)
   - CreateProduct(product *Product) error
   - UpdateProduct(product *Product) error
   - DeleteProduct(id int) error
   - CheckProductUsageInTemplates(productID int) (bool, error)

2.3. Add helper methods
   - ValidateHSNCode(code string) bool
   - CalculatePriceWithGST() float64

### 3. API Routes and Handlers
3.1. Define routes in routes/routes.go
   - Add product routes under project group
   - Apply authentication middleware

3.2. Create handlers/products.go
   - ListProducts(c *gin.Context) - GET /projects/:id/products
   - ShowAddProductForm(c *gin.Context) - GET /projects/:id/products/new
   - CreateProduct(c *gin.Context) - POST /projects/:id/products
   - ShowEditProductForm(c *gin.Context) - GET /projects/:id/products/:pid/edit
   - UpdateProduct(c *gin.Context) - PUT /projects/:id/products/:pid
   - DeleteProduct(c *gin.Context) - DELETE /projects/:id/products/:pid

3.3. Implement validation logic
   - Validate all required fields are present
   - Check price and GST percentage ranges
   - Validate HSN code format
   - Ensure project ownership before operations

3.4. Add error handling
   - Return appropriate HTTP status codes
   - Provide user-friendly error messages
   - Handle database constraint violations

### 4. Frontend Templates
4.1. Create templates/products/list.html
   - Header with project name and "Add Product" button
   - Table with columns: Item Name, HSN, UoM, Brand/Model, Price, GST%, Actions
   - Empty state message when no products exist
   - Edit and delete action buttons for each row
   - Use HTMX for dynamic interactions

4.2. Create templates/products/form.html (slide-over panel)
   - Form fields: Item Name, Item Description, HSN Code, UoM, Brand/Model, Price, GST%
   - Input validation (client-side)
   - Cancel and Submit buttons
   - Use Alpine.js for slide-over animation
   - HTMX form submission

4.3. Create templates/products/row.html (table row partial)
   - Reusable template for single product row
   - Used for HTMX swaps after create/update

4.4. Create templates/products/edit-inline.html
   - Inline editing form (optional alternative to slide-over)
   - Same fields as add form
   - Save/Cancel actions

4.5. Style with Tailwind CSS
   - Responsive table design
   - Slide-over panel styling
   - Form input styling
   - Action button styling
   - Hover states and transitions

### 5. HTMX Integration
5.1. Add Product Flow
   - Click "Add Product" triggers slide-over open
   - Form loads via HTMX: hx-get="/projects/:id/products/new"
   - Form submit: hx-post="/projects/:id/products"
   - On success: append new row to table, close slide-over
   - On error: display validation errors in form

5.2. Edit Product Flow
   - Click edit icon: hx-get="/projects/:id/products/:pid/edit"
   - Load edit form in slide-over or inline
   - Submit: hx-put="/projects/:id/products/:pid"
   - On success: swap updated row, close form
   - On error: show validation errors

5.3. Delete Product Flow
   - Click delete icon: show confirmation modal
   - Confirm: hx-delete="/projects/:id/products/:pid"
   - On success: remove row from table with fade animation
   - On error: show error message (e.g., "Product is used in templates")

### 6. Validation and Business Logic
6.1. Server-side validation
   - Required fields: item_name, hsn_code, uom, per_unit_price, gst_percentage
   - HSN code: 6-8 digits
   - Price: positive number, max 2 decimal places
   - GST: 0-100 range, common values: 0, 5, 12, 18, 28

6.2. Business rules
   - Product names must be unique within a project
   - Cannot delete products used in DC templates
   - All monetary values stored as DECIMAL(10,2)

6.3. Client-side validation
   - Required field indicators
   - Format validation on input
   - Immediate feedback on invalid entries

### 7. Testing Implementation
7.1. Unit tests for models
   - Test Product CRUD operations
   - Test validation methods
   - Test CheckProductUsageInTemplates

7.2. Integration tests for handlers
   - Test each API endpoint
   - Test authentication requirements
   - Test validation error responses
   - Test database constraints

7.3. Frontend interaction tests
   - Test HTMX form submissions
   - Test slide-over open/close
   - Test table row updates
   - Test delete confirmation

## Files to Create/Modify

### New Files
```
/migrations/006_create_products_table.sql
/models/product.go
/handlers/products.go
/templates/products/list.html
/templates/products/form.html
/templates/products/row.html
/templates/products/edit-inline.html
/templates/products/delete-confirm.html
/static/js/products.js (if needed for additional interactivity)
```

### Modified Files
```
/routes/routes.go (add product routes)
/templates/projects/detail.html (add link to products page)
/main.go (run new migration)
```

## API Routes / Endpoints

### Product Management Routes
```
GET    /projects/:id/products           - List all products for a project
GET    /projects/:id/products/new       - Show add product form (HTMX)
POST   /projects/:id/products           - Create new product
GET    /projects/:id/products/:pid      - Show single product (optional)
GET    /projects/:id/products/:pid/edit - Show edit product form (HTMX)
PUT    /projects/:id/products/:pid      - Update product
DELETE /projects/:id/products/:pid      - Delete product
```

### Request/Response Examples

#### POST /projects/:id/products
Request Body:
```json
{
  "item_name": "LED Street Light",
  "item_description": "60W LED street light with photocell",
  "hsn_code": "94054090",
  "uom": "Nos",
  "brand_model": "Philips BRP132 LED100",
  "per_unit_price": 4500.00,
  "gst_percentage": 18.00
}
```

Response (Success):
```json
{
  "success": true,
  "message": "Product added successfully",
  "product": {
    "id": 15,
    "project_id": 3,
    "item_name": "LED Street Light",
    "item_description": "60W LED street light with photocell",
    "hsn_code": "94054090",
    "uom": "Nos",
    "brand_model": "Philips BRP132 LED100",
    "per_unit_price": 4500.00,
    "gst_percentage": 18.00,
    "created_at": "2026-02-16T10:30:00Z"
  }
}
```

Response (Error):
```json
{
  "success": false,
  "errors": {
    "hsn_code": "Invalid HSN code format",
    "per_unit_price": "Price must be greater than 0"
  }
}
```

#### DELETE /projects/:id/products/:pid
Response (Cannot Delete):
```json
{
  "success": false,
  "message": "Cannot delete product: it is used in 2 DC templates"
}
```

## Database Queries

### Table Creation
```sql
CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    item_name TEXT NOT NULL,
    item_description TEXT,
    hsn_code TEXT NOT NULL,
    uom TEXT NOT NULL,
    brand_model TEXT,
    per_unit_price DECIMAL(10,2) NOT NULL,
    gst_percentage DECIMAL(5,2) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CHECK (per_unit_price > 0),
    CHECK (gst_percentage >= 0 AND gst_percentage <= 100)
);

CREATE INDEX idx_products_project_id ON products(project_id);
CREATE INDEX idx_products_item_name ON products(item_name);
CREATE UNIQUE INDEX idx_products_project_item ON products(project_id, item_name);
```

### Key Queries

#### Get all products for a project
```sql
SELECT id, project_id, item_name, item_description, hsn_code, uom,
       brand_model, per_unit_price, gst_percentage, created_at
FROM products
WHERE project_id = ?
ORDER BY item_name ASC;
```

#### Check if product is used in templates
```sql
SELECT COUNT(*)
FROM dc_template_products
WHERE product_id = ?;
```

#### Create product
```sql
INSERT INTO products (
    project_id, item_name, item_description, hsn_code, uom,
    brand_model, per_unit_price, gst_percentage
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);
```

#### Update product
```sql
UPDATE products
SET item_name = ?,
    item_description = ?,
    hsn_code = ?,
    uom = ?,
    brand_model = ?,
    per_unit_price = ?,
    gst_percentage = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND project_id = ?;
```

#### Delete product
```sql
DELETE FROM products
WHERE id = ? AND project_id = ?;
```

#### Get product with calculated total price
```sql
SELECT id, item_name, per_unit_price, gst_percentage,
       ROUND(per_unit_price * (1 + gst_percentage/100), 2) as price_with_gst
FROM products
WHERE id = ?;
```

## UI Components

### Products List Page Structure
```
┌─────────────────────────────────────────────────────────┐
│ Project Name > Products                    [Add Product]│
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Item Name   │ HSN    │ UoM │ Brand/Model │ Price │  │ │
│ ├─────────────────────────────────────────────────────┤ │
│ │ LED Light   │ 940540 │ Nos │ Philips     │ 4500  │  │ │
│ │ Cable 4-Core│ 854442 │ Mtr │ Polycab     │ 85    │  │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘

Slide-over Panel (Add/Edit Product):
┌───────────────────────────┐
│ Add Product          [X]  │
├───────────────────────────┤
│ Item Name:      [____]    │
│ Description:    [____]    │
│ HSN Code:       [____]    │
│ UoM:            [____]    │
│ Brand/Model:    [____]    │
│ Per Unit Price: [____]    │
│ GST %:          [____]    │
│                           │
│        [Cancel] [Submit]  │
└───────────────────────────┘
```

### Template Details

#### list.html
- Header section with breadcrumb navigation
- "Add Product" button (triggers slide-over)
- Responsive table container
- Table headers with proper alignment
- Product rows with edit/delete icons
- Empty state when no products exist
- Loading states for HTMX requests

#### form.html (Slide-over)
- Overlay backdrop (semi-transparent)
- Slide-in panel from right
- Form header with title and close button
- Form fields with labels and validation
- Input fields with proper types (text, number)
- Dropdown for common GST rates
- Cancel and Submit buttons
- Error message display area
- HTMX attributes for form submission

#### row.html (Partial)
- Single table row template
- Data cells with proper formatting
- Price formatting with currency symbol
- GST percentage with % symbol
- Action buttons (edit/delete) with icons
- Hover effects
- Used for HTMX swaps

## Testing Checklist

### Backend Tests
- [ ] Create product with valid data
- [ ] Create product with missing required fields (should fail)
- [ ] Create product with invalid HSN code (should fail)
- [ ] Create product with negative price (should fail)
- [ ] Create product with GST > 100 (should fail)
- [ ] Update existing product successfully
- [ ] Update product with invalid data (should fail)
- [ ] Delete product not used in templates (should succeed)
- [ ] Delete product used in templates (should fail)
- [ ] List products for a project
- [ ] Verify product uniqueness within project
- [ ] Check database constraints enforcement

### Frontend Tests
- [ ] Products list page loads correctly
- [ ] Empty state displays when no products
- [ ] Click "Add Product" opens slide-over
- [ ] Add product form displays all fields
- [ ] Form validation works (client-side)
- [ ] Submit valid product (should succeed)
- [ ] Submit invalid product (should show errors)
- [ ] New product appears in table after creation
- [ ] Click edit icon loads edit form
- [ ] Update product and verify changes
- [ ] Click delete shows confirmation
- [ ] Confirm delete removes product from table
- [ ] Delete product in use shows error message
- [ ] Slide-over closes on cancel
- [ ] Slide-over closes after successful submission
- [ ] Table remains responsive on various screen sizes

### Integration Tests
- [ ] End-to-end product creation workflow
- [ ] End-to-end product editing workflow
- [ ] End-to-end product deletion workflow
- [ ] Authentication required for all operations
- [ ] User can only manage products in their projects
- [ ] Concurrent product creation handling
- [ ] Database transaction rollback on errors

### Performance Tests
- [ ] List page loads quickly with 100+ products
- [ ] Search/filter performs well with large dataset
- [ ] Form submission responds within acceptable time

## Acceptance Criteria

### Must Have
1. Users can view all products for a project in a table format
2. Table displays: Item Name, HSN Code, UoM, Brand/Model, Per Unit Price, GST%, and Actions
3. Users can add new products via slide-over panel form
4. All required fields must be filled before submission
5. HSN code must be validated (6-8 digits)
6. Price must be positive and support 2 decimal places
7. GST percentage must be between 0 and 100
8. Users can edit existing products
9. Users can delete products not used in templates
10. System prevents deletion of products used in DC templates with clear error message
11. Product names are unique within a project
12. All operations use HTMX for seamless UX (no full page reloads)
13. Slide-over panel animates smoothly (open/close)
14. Form validation errors display clearly
15. Success/error messages show after operations

### Should Have
16. Products list is sortable by column headers
17. Search/filter functionality for large product lists
18. Inline editing option (alternative to slide-over)
19. Keyboard shortcuts for common actions
20. Confirmation dialog for destructive actions (delete)
21. Loading indicators during HTMX requests
22. Optimistic UI updates where appropriate

### Nice to Have
23. Bulk product import via CSV/Excel
24. Bulk product operations (delete multiple)
25. Product templates or quick-add for common items
26. HSN code lookup/autocomplete
27. Price history tracking
28. Product usage analytics
29. Export products list to CSV/Excel
30. Duplicate product functionality
