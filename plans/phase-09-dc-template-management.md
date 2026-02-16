# Phase 9: DC Template Management

## Overview
This phase implements DC (Delivery Challan) template management, which allows users to create reusable templates from project products. Templates define a subset of products that are commonly shipped together, streamlining the DC creation process. Each template can be used to issue both Transit and Official DCs, with tracking of all issued DCs.

## Prerequisites
- Phase 1-5: Project management functionality complete
- Phase 6: Product Management (templates reference products)
- Database: products table with project products
- Understanding of Transit DC vs Official DC distinction

## Goals
- Create DC templates with name, purpose, and selected products
- Display templates list in card layout matching mockup 09-templates-list.html
- Show template creation form matching mockup 10-template-form.html
- Display template detail view matching mockup 11-template-detail.html
- Track Transit DC and Official DC counts per template
- Support template editing and deletion
- Prevent deletion of templates with issued DCs
- Provide quick access to issue DCs from template detail page
- Implement multi-select product picker for template creation

## Detailed Implementation Steps

### 1. Database Schema Setup
1.1. Create dc_templates table
   - Store template metadata (name, purpose)
   - Link to project
   - Timestamps for audit trail

1.2. Create dc_template_products junction table
   - Many-to-many relationship between templates and products
   - No additional fields needed (just linking)

1.3. Add indexes
   - Index on project_id for template listing
   - Index on template_id for product lookups

1.4. Create migration with rollback

### 2. Backend Model Implementation
2.1. Create models/dc_template.go
   - DCTemplate struct with fields matching database
   - JSON tags for API responses
   - Validation tags

2.2. Implement CRUD methods
   - GetTemplatesByProjectID(projectID int) ([]DCTemplate, error)
   - GetTemplateByID(id int) (*DCTemplate, error)
   - CreateTemplate(template *DCTemplate, productIDs []int) error
   - UpdateTemplate(template *DCTemplate, productIDs []int) error
   - DeleteTemplate(id int) error
   - CheckTemplateHasDCs(templateID int) (bool, int, error)

2.3. Add helper methods
   - GetTemplateProducts(templateID int) ([]Product, error)
   - GetTemplateDCCounts(templateID int) (transitCount, officialCount int, error)
   - GetTemplateWithStats(templateID int) (*TemplateStats, error)

2.4. Create TemplateStats struct
   - Template details
   - Product count
   - Transit DC count
   - Official DC count
   - Last used date

### 3. API Routes and Handlers
3.1. Define routes in routes/routes.go
   - Template CRUD routes under project group
   - Authentication and authorization middleware

3.2. Create handlers/dc_templates.go
   - ListTemplates(c *gin.Context) - GET /projects/:id/templates
   - ShowCreateTemplateForm(c *gin.Context) - GET /projects/:id/templates/new
   - CreateTemplate(c *gin.Context) - POST /projects/:id/templates
   - ShowTemplateDetail(c *gin.Context) - GET /projects/:id/templates/:tid
   - ShowEditTemplateForm(c *gin.Context) - GET /projects/:id/templates/:tid/edit
   - UpdateTemplate(c *gin.Context) - PUT /projects/:id/templates/:tid
   - DeleteTemplate(c *gin.Context) - DELETE /projects/:id/templates/:tid
   - GetTemplateProducts(c *gin.Context) - GET /projects/:id/templates/:tid/products

3.3. Implement validation
   - Template name required and unique within project
   - At least one product must be selected
   - Purpose is optional but recommended
   - Products must belong to the same project

3.4. Error handling
   - Duplicate template name
   - Invalid product IDs
   - Template not found
   - Cannot delete template with issued DCs

### 4. Frontend Templates
4.1. Create templates/templates/list.html (mockup 09)
   - Page header with "Create Template" button
   - Grid/card layout for templates
   - Each card shows: template name, product count, DC counts
   - Empty state when no templates
   - Responsive grid (2-3 columns)

4.2. Create templates/templates/card.html
   - Template name as heading
   - Purpose description (truncated)
   - Product count badge
   - DC counts (Transit: X, Official: Y)
   - Actions: View, Edit, Delete
   - Click anywhere to open detail view

4.3. Create templates/templates/form.html (mockup 10)
   - Form fields: Template Name, Purpose
   - Product multi-select component
   - Available products list with checkboxes
   - Selected count indicator
   - Select All / Deselect All buttons
   - Submit and Cancel buttons

4.4. Create templates/templates/detail.html (mockup 11)
   - Template header with name and purpose
   - Edit and Delete buttons
   - Products section: table of selected products
   - Issued DCs section: tabs for Transit and Official
   - "Issue Transit DC" and "Issue Official DC" buttons
   - DC list tables (will be populated in Phase 11-12)

4.5. Create templates/templates/product-selector.html
   - Searchable product list
   - Checkbox for each product
   - Product details: name, HSN, price
   - Visual feedback for selection
   - Counter: "X of Y products selected"

4.6. Create templates/templates/delete-confirm.html
   - Confirmation modal
   - Warning if template has issued DCs
   - Prevent deletion with clear message
   - Confirm/Cancel buttons

### 5. Product Multi-Select Component
5.1. Design product selector UI
   - List all project products
   - Checkbox next to each product
   - Show product name, HSN, UoM
   - Search/filter products
   - Visual indication of selected products

5.2. Implement selection logic
   - Track selected product IDs
   - Submit as array to backend
   - Validate at least one selected

5.3. Add helper controls
   - Select All button
   - Deselect All button
   - Search box to filter products
   - Selected count display

5.4. Accessibility
   - Keyboard navigation
   - ARIA labels
   - Focus management

### 6. Template Detail View
6.1. Products section
   - Table showing all products in template
   - Columns: Item Name, HSN, UoM, Brand/Model, Price, GST%
   - Read-only view (edit via Edit Template)

6.2. Issued DCs section
   - Two tabs: Transit DCs | Official DCs
   - Each tab shows list of DCs issued from this template
   - DC list columns: DC Number, Date, Destination, Status
   - Empty state: "No Transit DCs issued yet"
   - Click DC to view details (Phase 11-12)

6.3. Action buttons
   - "Issue Transit DC" - navigates to Transit DC form (Phase 11)
   - "Issue Official DC" - navigates to Official DC form (Phase 12)
   - "Edit Template" - opens edit form
   - "Delete Template" - confirmation required

6.4. Template statistics
   - Product count
   - Total Transit DCs issued
   - Total Official DCs issued
   - Last used date

### 7. HTMX Integration
7.1. Create Template Flow
   - Click "Create Template" opens form
   - Load: hx-get="/projects/:id/templates/new"
   - Product selector loaded dynamically
   - Submit: hx-post="/projects/:id/templates"
   - On success: redirect to template detail or list
   - On error: show validation errors

7.2. Edit Template Flow
   - Click Edit from detail page
   - Load: hx-get="/projects/:id/templates/:tid/edit"
   - Pre-select products in selector
   - Submit: hx-put="/projects/:id/templates/:tid"
   - On success: update detail view
   - On error: show validation errors

7.3. Delete Template Flow
   - Click Delete button
   - Load confirmation modal
   - Check if template has DCs
   - If has DCs: show error, prevent deletion
   - If no DCs: confirm deletion
   - Delete: hx-delete="/projects/:id/templates/:tid"
   - On success: redirect to templates list

7.4. Template List Interaction
   - Cards are clickable
   - Click navigates to detail view
   - Edit/Delete icons trigger respective actions
   - HTMX boosts for smooth navigation

### 8. Validation and Business Logic
8.1. Template validation
   - Name required, max 100 characters
   - Name unique within project
   - Purpose optional, max 500 characters
   - At least one product required
   - Maximum 100 products per template (configurable)

8.2. Business rules
   - Cannot delete template if any DCs issued from it
   - Can edit template even with issued DCs (careful)
   - Editing template products doesn't affect already issued DCs
   - Products must exist and belong to same project

8.3. Data integrity
   - Use database transactions for create/update
   - Cascade delete template_products on template delete
   - Foreign key constraints enforced

### 9. Testing Implementation
9.1. Unit tests for models
   - Test DCTemplate CRUD operations
   - Test product association
   - Test DC count queries
   - Test validation methods

9.2. Integration tests for handlers
   - Test each API endpoint
   - Test authentication
   - Test validation errors
   - Test business rule enforcement

9.3. Frontend tests
   - Test template creation workflow
   - Test product selection
   - Test template editing
   - Test template deletion (with and without DCs)
   - Test template detail view

## Files to Create/Modify

### New Files
```
/migrations/009_create_dc_templates_table.sql
/migrations/010_create_dc_template_products_table.sql
/models/dc_template.go
/handlers/dc_templates.go
/templates/templates/list.html
/templates/templates/card.html
/templates/templates/form.html
/templates/templates/detail.html
/templates/templates/product-selector.html
/templates/templates/delete-confirm.html
/templates/templates/dc-list-tab.html
/static/js/product-selector.js
/static/js/template-tabs.js
```

### Modified Files
```
/routes/routes.go (add template routes)
/templates/projects/detail.html (add link to templates)
/main.go (run new migrations)
```

## API Routes / Endpoints

### DC Template Management Routes
```
GET    /projects/:id/templates              - List all templates for project
GET    /projects/:id/templates/new          - Show create template form
POST   /projects/:id/templates              - Create new template
GET    /projects/:id/templates/:tid         - Show template detail
GET    /projects/:id/templates/:tid/edit    - Show edit template form
PUT    /projects/:id/templates/:tid         - Update template
DELETE /projects/:id/templates/:tid         - Delete template
GET    /projects/:id/templates/:tid/products - Get template products (JSON)
```

### Request/Response Examples

#### POST /projects/:id/templates
Request Body:
```json
{
  "template_name": "Standard Secretariat Kit",
  "purpose": "Standard equipment package for secretariat office setup",
  "product_ids": [12, 15, 18, 23, 45]
}
```

Response (Success):
```json
{
  "success": true,
  "message": "Template created successfully",
  "template": {
    "id": 5,
    "project_id": 3,
    "template_name": "Standard Secretariat Kit",
    "purpose": "Standard equipment package for secretariat office setup",
    "created_at": "2026-02-16T11:00:00Z"
  },
  "product_count": 5
}
```

Response (Error):
```json
{
  "success": false,
  "errors": {
    "template_name": "Template name already exists",
    "product_ids": "At least one product must be selected"
  }
}
```

#### GET /projects/:id/templates/:tid
Response:
```json
{
  "success": true,
  "template": {
    "id": 5,
    "project_id": 3,
    "template_name": "Standard Secretariat Kit",
    "purpose": "Standard equipment package for secretariat office setup",
    "created_at": "2026-02-16T11:00:00Z"
  },
  "products": [
    {
      "id": 12,
      "item_name": "LED Street Light",
      "hsn_code": "94054090",
      "uom": "Nos",
      "brand_model": "Philips BRP132",
      "per_unit_price": 4500.00,
      "gst_percentage": 18.00
    }
  ],
  "stats": {
    "product_count": 5,
    "transit_dc_count": 12,
    "official_dc_count": 8,
    "last_used": "2026-02-15T14:30:00Z"
  }
}
```

#### DELETE /projects/:id/templates/:tid
Response (Cannot Delete):
```json
{
  "success": false,
  "message": "Cannot delete template: 20 DCs have been issued using this template",
  "dc_count": 20
}
```

Response (Success):
```json
{
  "success": true,
  "message": "Template deleted successfully"
}
```

## Database Queries

### Table Creation

#### dc_templates table
```sql
CREATE TABLE dc_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    template_name TEXT NOT NULL,
    purpose TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE (project_id, template_name)
);

CREATE INDEX idx_dc_templates_project ON dc_templates(project_id);
```

#### dc_template_products junction table
```sql
CREATE TABLE dc_template_products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES dc_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE (template_id, product_id)
);

CREATE INDEX idx_template_products_template ON dc_template_products(template_id);
CREATE INDEX idx_template_products_product ON dc_template_products(product_id);
```

### Key Queries

#### Get all templates for a project with stats
```sql
SELECT
    t.id,
    t.template_name,
    t.purpose,
    t.created_at,
    COUNT(DISTINCT tp.product_id) as product_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'transit' THEN dc.id END) as transit_count,
    COUNT(DISTINCT CASE WHEN dc.dc_type = 'official' THEN dc.id END) as official_count
FROM dc_templates t
LEFT JOIN dc_template_products tp ON t.id = tp.template_id
LEFT JOIN delivery_challans dc ON t.id = dc.template_id
WHERE t.project_id = ?
GROUP BY t.id
ORDER BY t.created_at DESC;
```

#### Get template by ID with products
```sql
-- Get template
SELECT id, project_id, template_name, purpose, created_at
FROM dc_templates
WHERE id = ?;

-- Get template products
SELECT p.*
FROM products p
INNER JOIN dc_template_products tp ON p.id = tp.product_id
WHERE tp.template_id = ?
ORDER BY p.item_name;
```

#### Create template with products (transaction)
```sql
BEGIN TRANSACTION;

-- Insert template
INSERT INTO dc_templates (project_id, template_name, purpose)
VALUES (?, ?, ?);

-- Get last inserted ID
SELECT last_insert_rowid() as template_id;

-- Insert template products
INSERT INTO dc_template_products (template_id, product_id)
VALUES (?, ?), (?, ?), (?, ?); -- repeat for each product

COMMIT;
```

#### Update template (transaction)
```sql
BEGIN TRANSACTION;

-- Update template
UPDATE dc_templates
SET template_name = ?,
    purpose = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND project_id = ?;

-- Delete existing product associations
DELETE FROM dc_template_products
WHERE template_id = ?;

-- Insert new product associations
INSERT INTO dc_template_products (template_id, product_id)
VALUES (?, ?), (?, ?), (?, ?); -- repeat for each product

COMMIT;
```

#### Check if template has issued DCs
```sql
SELECT COUNT(*) as dc_count
FROM delivery_challans
WHERE template_id = ?;
```

#### Delete template (only if no DCs)
```sql
-- First check DC count
SELECT COUNT(*) FROM delivery_challans WHERE template_id = ?;

-- If count is 0, proceed with delete
DELETE FROM dc_templates
WHERE id = ? AND project_id = ?;
-- This will cascade delete dc_template_products due to foreign key
```

#### Get template DC counts by type
```sql
SELECT
    dc_type,
    COUNT(*) as count
FROM delivery_challans
WHERE template_id = ?
GROUP BY dc_type;
```

#### Get template usage statistics
```sql
SELECT
    t.id,
    t.template_name,
    COUNT(DISTINCT tp.product_id) as product_count,
    COUNT(DISTINCT dc.id) as total_dc_count,
    MAX(dc.created_at) as last_used
FROM dc_templates t
LEFT JOIN dc_template_products tp ON t.id = tp.template_id
LEFT JOIN delivery_challans dc ON t.id = dc.template_id
WHERE t.id = ?
GROUP BY t.id;
```

## UI Components

### Templates List Page Structure (Mockup 09)
```
┌─────────────────────────────────────────────────────────┐
│ Project Name > DC Templates            [+ Create Template]│
├─────────────────────────────────────────────────────────┤
│ ┌──────────────────┐ ┌──────────────────┐ ┌────────────┐ │
│ │ Standard         │ │ Street Light Kit │ │ Cable Kit  │ │
│ │ Secretariat Kit  │ │                  │ │            │ │
│ │                  │ │ Standard street  │ │ Cabling    │ │
│ │ Equipment pkg... │ │ light deployment │ │ materials  │ │
│ │                  │ │                  │ │            │ │
│ │ Products: 5      │ │ Products: 3      │ │ Products: 8│ │
│ │ Transit: 12      │ │ Transit: 25      │ │ Transit: 6 │ │
│ │ Official: 8      │ │ Official: 18     │ │ Official: 4│ │
│ │                  │ │                  │ │            │ │
│ │ [View] [Edit] [×]│ │ [View] [Edit] [×]│ │[View][Edit]│ │
│ └──────────────────┘ └──────────────────┘ └────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Template Form (Mockup 10)
```
┌─────────────────────────────────────────────────────────┐
│ Create DC Template                                      │
├─────────────────────────────────────────────────────────┤
│ Template Name: *                                        │
│ [Standard Secretariat Kit________________________]      │
│                                                         │
│ Purpose:                                                │
│ [Standard equipment package for secretariat offices_]   │
│                                                         │
│ Select Products: * (5 selected)                         │
│ [Search products...___________] [Select All] [Clear]   │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ ☑ LED Street Light - HSN: 94054090 - ₹4,500       │ │
│ │ ☐ Cable 4-Core - HSN: 854442 - ₹85                │ │
│ │ ☑ Distribution Board - HSN: 853720 - ₹2,300       │ │
│ │ ☑ MCB 32A - HSN: 853630 - ₹450                    │ │
│ │ ☐ Conduit Pipe - HSN: 391723 - ₹125               │ │
│ │ ☑ Junction Box - HSN: 853690 - ₹180               │ │
│ │ ☑ Earth Electrode - HSN: 854459 - ₹850            │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│                             [Cancel]  [Create Template] │
└─────────────────────────────────────────────────────────┘
```

### Template Detail Page (Mockup 11)
```
┌──────────────────────────────────────────────────────────┐
│ Standard Secretariat Kit                  [Edit] [Delete]│
│ Purpose: Standard equipment package for secretariat...   │
├──────────────────────────────────────────────────────────┤
│ Products in Template (5)                                 │
│ ┌────────────────────────────────────────────────────┐   │
│ │ Item Name      │ HSN    │ UoM │ Brand  │ Price    │   │
│ ├────────────────────────────────────────────────────┤   │
│ │ LED Street Lt  │ 940540 │ Nos │ Philips│ ₹4,500   │   │
│ │ Distrib. Board │ 853720 │ Nos │ Havells│ ₹2,300   │   │
│ │ MCB 32A        │ 853630 │ Nos │ Siemens│ ₹450     │   │
│ │ Junction Box   │ 853690 │ Nos │ Generic│ ₹180     │   │
│ │ Earth Electrode│ 854459 │ Nos │ Generic│ ₹850     │   │
│ └────────────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────────────┤
│ Issued DCs                                               │
│ [Transit DCs (12)] [Official DCs (8)]                    │
│ ┌────────────────────────────────────────────────────┐   │
│ │ DC Number      │ Date       │ Destination│ Status  │   │
│ ├────────────────────────────────────────────────────┤   │
│ │ FS/GS/25-26/T/│ 2026-02-10│ Warehouse B│ Delivered│   │
│ │ 001            │            │            │         │   │
│ │ FS/GS/25-26/T/│ 2026-02-12│ Secretariat│ In Transit│  │
│ │ 002            │            │ VJA-001    │         │   │
│ └────────────────────────────────────────────────────┘   │
│                                                          │
│ [Issue Transit DC]  [Issue Official DC]                 │
└──────────────────────────────────────────────────────────┘
```

### Template Details

#### list.html
- Grid container (2-3 columns)
- Template cards
- Create button in header
- Empty state with illustration
- Loading skeleton

#### card.html
- Card container with border and shadow
- Template name (truncated if long)
- Purpose description (2 lines max)
- Stats section (products, DCs)
- Action buttons
- Hover effects

#### form.html
- Form fields for name and purpose
- Product selector component
- Selected count indicator
- Validation error display
- Submit and cancel buttons

#### detail.html
- Template header section
- Products table
- DC tabs section
- Issue DC buttons
- Edit/Delete actions

#### product-selector.html
- Search bar
- Select All/Clear buttons
- Scrollable product list
- Checkboxes for each product
- Product details per row
- Selected count

#### dc-list-tab.html
- Tab navigation
- DC table per tab
- Empty state per tab
- Link to DC detail pages

## Testing Checklist

### Backend Tests
- [ ] Create template with valid data
- [ ] Create template with missing name (should fail)
- [ ] Create template with duplicate name (should fail)
- [ ] Create template with no products (should fail)
- [ ] Create template with invalid product IDs (should fail)
- [ ] Update template successfully
- [ ] Update template name to duplicate (should fail)
- [ ] Delete template without DCs (should succeed)
- [ ] Delete template with DCs (should fail)
- [ ] Get template by ID
- [ ] Get template with products
- [ ] Get template with DC counts
- [ ] List templates for project
- [ ] Verify template-product associations
- [ ] Transaction rollback on error

### Frontend Tests
- [ ] Templates list page loads
- [ ] Templates display in card layout
- [ ] Card shows correct stats
- [ ] Click card opens detail view
- [ ] Click "Create Template" opens form
- [ ] Template form displays all fields
- [ ] Product selector loads products
- [ ] Select products via checkboxes
- [ ] Select All button works
- [ ] Clear/Deselect All button works
- [ ] Selected count updates correctly
- [ ] Search products filters list
- [ ] Submit valid template (should succeed)
- [ ] Submit without name (should show error)
- [ ] Submit without products (should show error)
- [ ] Template detail page loads
- [ ] Products table displays correctly
- [ ] DC tabs switch correctly
- [ ] Edit template loads pre-filled form
- [ ] Update template (should succeed)
- [ ] Delete template without DCs (should succeed)
- [ ] Delete template with DCs (should fail with message)
- [ ] Empty states display correctly

### Integration Tests
- [ ] End-to-end template creation workflow
- [ ] End-to-end template editing workflow
- [ ] End-to-end template deletion workflow
- [ ] Template products persist correctly
- [ ] DC counts display correctly (after creating DCs in Phase 11-12)
- [ ] Authentication required for all operations
- [ ] User can only manage templates in their projects

### UI/UX Tests
- [ ] Responsive design on various screen sizes
- [ ] Card layout adapts to screen width
- [ ] Long template names truncate properly
- [ ] Long purpose text truncates properly
- [ ] Product selector scrollable with many products
- [ ] Search debounce works smoothly
- [ ] Loading states display during operations
- [ ] Success messages display after operations
- [ ] Error messages display clearly

## Acceptance Criteria

### Must Have
1. Users can create DC templates with name and purpose
2. Users can select multiple products for a template
3. At least one product must be selected
4. Template name must be unique within a project
5. Templates list displays in card layout (mockup 09)
6. Each card shows: name, purpose (truncated), product count, DC counts
7. Template creation form displays product selector (mockup 10)
8. Product selector shows all project products with checkboxes
9. Selected product count displayed during selection
10. Template detail page shows all template info (mockup 11)
11. Detail page displays products in a table
12. Detail page has tabs for Transit DCs and Official DCs
13. Buttons to "Issue Transit DC" and "Issue Official DC" present
14. Users can edit templates (name, purpose, products)
15. Users can delete templates without issued DCs
16. System prevents deletion of templates with issued DCs
17. Clear error message when deletion prevented
18. All operations use HTMX for smooth UX

### Should Have
19. Empty state when no templates exist
20. Loading indicators during operations
21. Search functionality in product selector
22. Select All / Deselect All buttons in product selector
23. Template cards clickable (navigate to detail)
24. Confirmation dialog before deletion
25. Success messages after create/update/delete
26. Validation errors display inline
27. Product selector shows HSN, UoM, price
28. Responsive card grid (2-3 columns)
29. Template statistics accurate (product count, DC counts)
30. Last used date displayed (if available)

### Nice to Have
31. Duplicate template functionality
32. Template templates (pre-defined common templates)
33. Bulk template operations
34. Template usage analytics
35. Sort templates by name, date, usage
36. Filter templates by product
37. Export template details to PDF
38. Template version history
39. Inactive/archive templates
40. Template sharing across projects (optional)
41. Quick add common products feature
42. Product recommendations based on template type
43. Template preview before creation
44. Drag-and-drop product selection
