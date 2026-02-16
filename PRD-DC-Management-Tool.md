# Product Requirements Document (PRD)
# Fervid Smart Solutions — DC Management Tool

**Version:** 1.0  
**Date:** February 16, 2026  
**Status:** Draft  
**Company:** Fervid Smart Solutions Private Limited  

---

## 1. Executive Summary

An internal web application for creating and managing Delivery Challans (DCs) across projects. The tool supports two DC types — **Transit DC** (issued to transporters with full pricing and tax details) and **Official DC** (for statutory documentation without pricing). Both DC types are generated from a shared template system built on a project → template → issued DC hierarchy.

---

## 2. Tech Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go + Gin framework |
| **Frontend** | HTMX + Tailwind CSS (minimal JavaScript) |
| **Database** | SQLite (file-based) |
| **Mobile** | Responsive / mobile-compatible design |
| **Output Formats** | HTML preview, PDF download, Excel (.xlsx) download, Browser print |

---

## 3. User Authentication

- **Multi-user** system with simple username/password login
- **No role distinction** — all users have equal access to all features
- No approval workflows; any user can create, issue, and delete DCs

---

## 4. Core Data Hierarchy

```
Company (Fervid Smart Solutions — fixed)
  └── Project
        ├── Products Master (all products for this project)
        ├── Bill To Address List (uploaded CSV, user-defined columns)
        ├── Ship To Address List (uploaded CSV, user-defined columns)
        ├── Company Signature Image
        └── DC Templates (subsets of project products)
              └── Issued DCs (Transit DC + Official DC from same template)
```

---

## 5. Entity Definitions & Field Specifications

### 5.1 Project

A project represents a procurement/delivery engagement with a client.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **Project Name** | Text | Yes | Short identifying name |
| **Project Description** | Long Text | Yes | Detailed project description (e.g., "GSWS Department — Procurement of IT Hardware items to Village/Ward Secretariats for supply of AVO 1000 1 KVA Line Interactive UPS Systems — Reg.") |
| **DC Number Prefix** | Text | Yes | User-defined prefix for DC numbering (e.g., `FS/GS`). System auto-appends financial year + suffix (`/T/` for Transit, `/D/` for Official) + sequential number. Example: `FS/GS/25-26/T/001` |
| **PO Number** | Text | Yes | Purchase Order number (e.g., "Purchase Order E. File No 3059446 : / GSWS/2025") |
| **PO Date** | Date | Yes | Purchase Order date |
| **Tender Reference Number** | Text | Yes | Tender reference (e.g., "ITC51-15021/20/2025-PROC-APTS Dt:30-10-2025") |
| **Tender Reference Details** | Long Text | Yes | Full tender description |
| **Purchase Order Reference** | Long Text | Yes | PO reference description (e.g., "Procurement of IT Hardware items to Village/Ward Secretariats — issue of Purchase Order (P.O) to M/s Fervid Smart Solutions Pvt Ltd, for supply of AVO 1000 1 KVA Line Interactive UPS Systems — Reg.") |
| **Bill From Address** | Long Text | Yes | Company dispatch address (configurable per project, e.g., "FERVID SMART SOLUTIONS PVT. LTD, Vijayawada") |
| **Company Signature** | Image Upload | Yes | Authorized signatory signature image for Fervid, used on all DCs in this project |
| **Company GSTIN** | Text | Yes | Pre-filled: `36AAACCF9742K1Z8` (editable) |

#### 5.1.1 Bill To Address List (per project)

- Uploaded via **CSV/Excel** file
- **User defines the column structure** per project (e.g., Project A may have `District, Office Name, Address, Pin Code` while Project B has `State, City, Ward, Office`)
- When uploading, user maps/names columns; system stores them dynamically
- Used as a **dropdown/searchable list** when issuing a DC
- **Separate list** from Ship To addresses

#### 5.1.2 Ship To Address List (per project)

- Same mechanism as Bill To — uploaded via CSV/Excel with user-defined columns
- **Separate upload** from Bill To
- Each DC has **exactly one Ship To** address selected from this list
- User-defined column structure (e.g., `District, SRO, Location, Location ID, Mandal/ULB, Secretariat Name, Secretariat Code`)

---

### 5.2 Product (Project-Level Master)

Products are defined at the project level with **all details fixed**. DC Templates select subsets of these products.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **Item Name** | Text | Yes | Short name (e.g., "UPS") |
| **Item Description** | Long Text | Yes | Detailed description |
| **HSN Code** | Text | Yes | Harmonized System of Nomenclature code |
| **UoM** | Text | Yes | Unit of Measurement (e.g., "Nos", "Kg", "Mtr") |
| **Brand / Model Number** | Text | Yes | Brand and model info |
| **Per Unit Price** | Decimal | Yes | Unit price in INR (fixed for the project) |
| **GST Percentage** | Decimal | Yes | GST rate (e.g., 18, 12, 5) — fixed for the project |

---

### 5.3 DC Template

A DC Template is a **subset of project products** used as a reusable base for issuing DCs. One project can have multiple templates (e.g., Template A with 4 products, Template B with 3 products, Template C with remaining 3 products).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **Template Name** | Text | Yes | Identifying name (e.g., "UPS + Battery Kit") |
| **Selected Products** | Multi-select | Yes | Subset of project products included in this template |
| **Purpose** | Text | Yes | Purpose text for Official DC (e.g., "DELIVERED AS PART OF PROJECT EXECUTION") |

**Key behavior:** One template produces **both** Transit DC and Official DC. The user picks which type to issue from the template.

---

### 5.4 Issued DC (Transit DC)

Created from a DC Template. Contains transport, pricing, and tax details.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **DC Number** | Auto-generated | System | Format: `{prefix}/{FY}/T/{seq}` (e.g., `FS/GS/25-26/T/001`) |
| **DC Date** | Date | Yes | Date of issue |
| **Status** | Enum | System | `Draft` or `Issued` |
| **Bill To** | Selection | Yes | Selected from project's Bill To address list (default from project, overridable) |
| **Ship To** | Selection | Yes | Selected from project's Ship To address list |
| **Mode of Transport** | Text | Yes | e.g., "By Road", "By Rail" |
| **Name of Driver / Transporter** | Text | Yes | Driver or transporter name |
| **Vehicle Number** | Text | Yes | Vehicle registration number |
| **Docket Number** | Text | No | Docket/consignment number |
| **E-Way Bill Number** | Text | No | E-Way bill reference |
| **Reverse Charge (Y/N)** | Boolean | Yes | Default: N |
| **Tax Type** | Enum | Yes | `CGST+SGST` or `IGST` — user manually selects |
| **Notes** | Long Text | No | Free text notes |

**Per Product Line (for each product in the template):**

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| **S.No** | Auto | System | Sequential number |
| **Item Name** | Text | From project product | Pre-filled, read-only |
| **Item Description** | Text | From project product | Pre-filled, read-only |
| **Serial Numbers** | Text (scan field) | User input | Scanned via barcode scanner (newline-separated). Each scan appends to the field. |
| **Quantity** | Integer | Auto-calculated | = count of scanned serial numbers |
| **UoM** | Text | From project product | Pre-filled, read-only |
| **HSN Code** | Text | From project product | Pre-filled, read-only |
| **Per Unit Price** | Decimal | From project product | Pre-filled, read-only |
| **Taxable Value** | Decimal | Auto-calculated | = Quantity × Per Unit Price |
| **GST %** | Decimal | From project product | Pre-filled, read-only |
| **GST Amount** | Decimal | Auto-calculated | = Taxable Value × GST % |
| **Total Value** | Decimal | Auto-calculated | = Taxable Value + GST Amount |

**Tax Summary Section (auto-calculated):**

| Field | Calculation |
|-------|------------|
| **Taxable Value** | Sum of all line Taxable Values |
| **CGST Value** | Sum of GST Amounts ÷ 2 *(only if CGST+SGST selected)* |
| **SGST Value** | Sum of GST Amounts ÷ 2 *(only if CGST+SGST selected)* |
| **IGST Value** | Sum of GST Amounts *(only if IGST selected)* |
| **Total GST** | CGST + SGST or IGST |
| **Round Off** | Rounding adjustment to nearest rupee |
| **Invoice/DC Value** | Taxable Value + Total GST + Round Off |
| **Amount in Words** | Auto-generated (e.g., "Rupees One Lakh Twenty Thousand Only") — Indian numbering system |

**Signature Section:**
- Fervid signature: company signature image (from project)
- Receiver's Signature: placeholder lines (Name, blank line)

---

### 5.5 Issued DC (Official DC)

Created from the same DC Template as Transit DC but **without pricing or tax**.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **DC Number** | Auto-generated | System | Format: `{prefix}/{FY}/D/{seq}` (e.g., `FS/GS/25-26/D/001`) |
| **DC Date** | Date | Yes | Date of issue |
| **Status** | Enum | System | `Draft` or `Issued` |
| **Purpose** | Text | From template | Pre-filled from template's Purpose field |
| **Bill To** | Selection | Yes | Selected from project's Bill To address list |
| **Ship To** | Selection | Yes | Selected from project's Ship To address list |
| **Notes** | Long Text | No | Free text notes |

**Per Product Line (for each product in the template):**

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| **S.No** | Auto | System | Sequential number |
| **Item Name** | Text | From project product | Pre-filled |
| **Item Description** | Text | From project product | Pre-filled |
| **Brand / Model No** | Text | From project product | Pre-filled |
| **Quantity** | Integer | Auto-calculated | = count of scanned serial numbers |
| **Serial Numbers** | Text (scan field) | User input | Scanned via barcode scanner (newline-separated) |
| **Remarks** | Text | No | Optional per-line remarks |

**No pricing columns. No tax summary.**

**Signature Section:**
- Fervid (FSSPL Representative): company signature image (from project) + placeholder for Name, Designation, Mobile Number
- Department Official: placeholder lines for Signature with Seal and Date, Name, Designation, Mobile Number

**Acknowledgement Section:**
- "It is certified that the material is received in good condition."
- "Date of Receipt: _______________"

---

## 6. DC Numbering Logic

### Format
```
{Project DC Prefix}/{Financial Year}/{DC Type Suffix}/{Sequential Number}
```

### Rules
- **Financial Year:** Indian financial year (April to March). E.g., April 2025 to March 2026 = `25-26`
- **DC Type Suffix:** `T` for Transit DC, `D` for Official DC
- **Sequential Number:** Auto-incremented per project, per DC type, per financial year. Padded to 3 digits (001, 002, ..., 999)
- **Example:** Project prefix = `FS/GS` → Transit DC #5 in FY 25-26 = `FS/GS/25-26/T/005`
- Numbering resets each financial year

---

## 7. Serial Number Management

### Input Method
- Text area field per product line on the DC
- Optimized for **barcode scanner** input (scanner typically appends newline after each scan)
- Each line in the text area = one serial number
- Manual typing also supported

### Quantity Auto-Calculation
- **Quantity = number of non-empty lines** in the serial number field
- Updates in real-time as serial numbers are scanned/entered
- All pricing calculations cascade from this quantity

### Validation Rules
- **Unique within project:** A serial number cannot appear in more than one issued DC within the same project
- **Cross-project:** Same serial number CAN exist in different projects
- System shows real-time validation error if a duplicate is detected
- On DC deletion, all associated serial numbers are **freed** and can be reused

---

## 8. DC Lifecycle

```
┌─────────┐      User clicks       ┌─────────┐
│  DRAFT  │  ─── "Issue DC" ───►   │ ISSUED  │
└─────────┘                        └─────────┘
     │                                   │
     │  Can be edited freely             │  LOCKED — cannot be edited
     │  Can be deleted                   │  Can be DELETED (double confirmation)
     │                                   │  Serial numbers freed on deletion
     └───────────────────────────────────┘
```

### Deletion Rules
- **Double confirmation required:** First click shows confirmation dialog → Second click confirms permanent deletion
- When a DC is deleted, all serial numbers associated with it are removed from the used-serial-numbers pool
- Deleted DCs are **permanently removed** (hard delete)
- DC numbers of deleted DCs are **not reused** (sequence continues)

---

## 9. Address List Management

### Upload Flow
1. User navigates to Project → "Manage Bill To Addresses" or "Manage Ship To Addresses"
2. **First time:** User defines column names (e.g., "District", "Mandal", "Secretariat", "Code")
3. User uploads CSV/Excel file matching those columns
4. System parses and stores the addresses
5. Subsequent uploads can append or replace the list

### Column Structure
- **User-defined per project** — different projects can have completely different column structures
- Bill To and Ship To can have **different column structures** within the same project
- All columns are stored and displayed when selecting an address during DC creation

### Selection During DC Issuance
- **Searchable dropdown** showing all columns for easy identification
- Bill To: defaults to project-level default, but can be overridden per DC
- Ship To: must be selected per DC (one Ship To per DC)

---

## 10. Output Formats

All four output formats for both DC types:

| Format | Description |
|--------|-------------|
| **HTML Preview** | On-screen preview matching the final DC layout |
| **PDF Download** | Downloadable PDF matching the Excel template layouts |
| **Excel Download** | .xlsx file matching the original template format |
| **Browser Print** | Print-optimized CSS for direct printing from browser |

### Transit DC Layout
Matches the `FSS-Transit-DC` sheet layout:
- Company header (Fervid Smart Solutions Pvt. Ltd with address and GSTIN)
- "DELIVERY CHALLAN" title
- DC number, date, transport details (left side), location details (right side)
- PO details, project description
- Bill From / Bill To / Dispatch From / Ship To addresses
- Product table with pricing columns (S.No, Item Description, Serial Nos, UoM, HSN Code, Quantity, Per Unit, Taxable Value, GST %, GST Amount, Total Value)
- Totals row
- Tax summary (Taxable Value, CGST, SGST or IGST, Round Off, Invoice Value)
- Amount in words
- Notes
- Signature section (Receiver's Signature | For Fervid Smart Solutions Pvt. Ltd — Authorised Signatory with uploaded image)

### Official DC Layout
Matches the `Fervid-DC-V1` sheet layout:
- Company header with full address, email, GSTIN, CIN
- "DELIVERY CHALLAN" title
- DC number, date, Mandal/ULB Name, Mandal Code
- Project/Tender/PO reference details
- Purpose field
- Issued To: District and Mandal/ULB name
- Product table (S.No, Item Name, Description, Brand/Model No, Quantity, Serial Number, Remarks) — **no pricing**
- Acknowledgement: "It is certified that the material is received in good condition."
- Date of Receipt line
- Dual signature block (FSSPL Representative | Department Official) with Name, Designation, Mobile Number fields

---

## 11. Dashboard & Navigation

### 11.1 Home Dashboard
- **Summary statistics:** Total projects, total DCs issued (Transit + Official), DCs by status, DCs by date range
- **Quick actions:** Create new project, recent DCs list

### 11.2 Projects List
- All projects with: Project Name, PO Number, number of DC Templates, number of issued DCs (Transit count + Official count)
- Click into a project for full details

### 11.3 Project Detail View
- Project info (all fields from Section 5.1)
- Tabs or sections for:
  - **Products:** List of all project products
  - **DC Templates:** List of templates with their product subsets
  - **Bill To Addresses:** View/upload/manage
  - **Ship To Addresses:** View/upload/manage
  - **Issued DCs:** All DCs issued under this project (filterable by type, status, date)

### 11.4 DC Listing (Global)
- All DCs across all projects
- **Filters:** By project, by DC type (Transit/Official), by status (Draft/Issued), by date range, by Ship To location
- **Search by serial number:** Enter a serial number → find which DC it belongs to
- Columns: DC Number, Type, Date, Project, Ship To (summary), Status, Total Value (Transit only)

### 11.5 Serial Number Search
- Global search across all projects or within a specific project
- Input a serial number → returns: DC Number, DC Type, Project, Product, Date, Ship To location

---

## 12. Workflow Summary

### Creating a Project
1. Enter project details (name, description, PO, tender ref, etc.)
2. Define Bill From address
3. Upload company signature image
4. Add products (Item Name, Description, HSN, UoM, Brand, Price, GST%)
5. Define Bill To address columns → Upload Bill To address list (CSV/Excel)
6. Define Ship To address columns → Upload Ship To address list (CSV/Excel)

### Creating a DC Template
1. Navigate to a project
2. Click "Create DC Template"
3. Enter template name and purpose
4. Select products from the project's product list (subset)
5. Save template

### Issuing a DC
1. Navigate to a DC Template
2. Click "Issue Transit DC" or "Issue Official DC"
3. System creates a Draft DC with auto-generated DC number
4. Fill in:
   - **Transit DC:** DC date, Bill To (default pre-filled, overridable), Ship To (select from list), transport details (mode, driver, vehicle, docket, e-way bill), reverse charge, tax type (CGST+SGST or IGST), notes
   - **Official DC:** DC date, Bill To, Ship To, notes
5. For each product line: scan serial numbers via barcode scanner → quantity auto-calculates → pricing auto-calculates (Transit only)
6. Preview the DC (HTML)
7. Click "Issue" → status changes to Issued (locked)
8. Download as PDF / Excel or Print

---

## 13. Company Information (Hardcoded/Default)

```
FERVID SMART SOLUTIONS PRIVATE LIMITED
Plot No 14/2, Dwaraka Park View, 1st Floor, Sector-1
HUDA Techno Enclave, Madhapur, Hyderabad, Telangana 500081
Email: odishaprojects@fervidsmart.com
GSTIN: 36AACCF9742K1Z8
CIN No: U45100TG2016PTC113752
```

These values are used as defaults in the DC headers. GSTIN is editable per project.

---

## 14. Database Schema (High-Level)

### Core Tables
- **users** — id, username, password_hash, created_at
- **projects** — id, name, description, dc_prefix, po_number, po_date, tender_ref_number, tender_ref_details, po_reference, bill_from_address, gstin, signature_image_path, created_at, updated_at
- **products** — id, project_id, item_name, item_description, hsn_code, uom, brand_model, per_unit_price, gst_percentage, created_at
- **dc_templates** — id, project_id, template_name, purpose, created_at
- **dc_template_products** — id, template_id, product_id (maps which products are in each template)

### Address Tables
- **address_list_configs** — id, project_id, list_type (bill_to/ship_to), column_definitions (JSON array of column names)
- **addresses** — id, config_id, data (JSON object with column_name: value pairs), created_at

### DC Tables
- **delivery_challans** — id, project_id, template_id, dc_type (transit/official), dc_number, dc_date, status (draft/issued), bill_to_address_id, ship_to_address_id, notes, issued_at, created_at, updated_at
- **dc_transit_details** — id, dc_id, mode_of_transport, driver_name, vehicle_number, docket_number, eway_bill_number, reverse_charge, tax_type (cgst_sgst/igst)
- **dc_line_items** — id, dc_id, product_id, serial_numbers (text, newline-separated), quantity (auto-calculated), remarks
- **serial_numbers** — id, project_id, dc_id, product_id, serial_number (indexed, unique within project scope)

---

## 15. Non-Functional Requirements

| Requirement | Specification |
|-------------|--------------|
| **Mobile Compatibility** | Fully responsive; usable on mobile browsers for barcode scanning |
| **Barcode Scanner Support** | Text input fields that accept rapid scanner input (newline-delimited) |
| **Performance** | SQLite is sufficient for expected load; no concurrent write-heavy scenarios expected |
| **Data Backup** | SQLite file can be backed up manually; no automated backup in v1 |
| **Browser Support** | Modern browsers (Chrome, Firefox, Safari, Edge) |
| **Offline** | Not required in v1 |

---

## 16. Out of Scope (v1)

- Installation Reports (IR) — may be added in v2
- Role-based access control / approval workflows
- Custom fields on DCs (may be added later)
- Automated email/notification on DC issuance
- Integration with e-Way Bill portal
- Inventory/stock tracking
- Offline mode
- Automated backup
