package database

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func setupTemplateTestDB(t *testing.T) func() {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	db.Exec("PRAGMA foreign_keys = ON")

	db.Exec(`CREATE TABLE projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		created_by INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`INSERT INTO projects (id, name) VALUES (1, 'Test Project')`)
	db.Exec(`INSERT INTO projects (id, name) VALUES (2, 'Other Project')`)

	db.Exec(`CREATE TABLE products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		item_name TEXT NOT NULL,
		item_description TEXT DEFAULT '',
		hsn_code TEXT DEFAULT '',
		uom TEXT DEFAULT 'Nos',
		brand_model TEXT DEFAULT '',
		per_unit_price DECIMAL(10,2) DEFAULT 0,
		gst_percentage DECIMAL(5,2) DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	)`)
	db.Exec(`INSERT INTO products (id, project_id, item_name, uom, per_unit_price) VALUES (1, 1, 'Product A', 'Nos', 100.00)`)
	db.Exec(`INSERT INTO products (id, project_id, item_name, uom, per_unit_price) VALUES (2, 1, 'Product B', 'Kg', 200.00)`)
	db.Exec(`INSERT INTO products (id, project_id, item_name, uom, per_unit_price) VALUES (3, 1, 'Product C', 'Nos', 300.00)`)

	db.Exec(`CREATE TABLE dc_templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		purpose TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	)`)

	db.Exec(`CREATE TABLE dc_template_products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		template_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		default_quantity INTEGER DEFAULT 1,
		sort_order INTEGER DEFAULT 0,
		FOREIGN KEY (template_id) REFERENCES dc_templates(id) ON DELETE CASCADE,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
		UNIQUE(template_id, product_id)
	)`)

	db.Exec(`CREATE TABLE delivery_challans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		dc_number TEXT,
		dc_type TEXT DEFAULT 'transit',
		status TEXT DEFAULT 'draft',
		template_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (template_id) REFERENCES dc_templates(id)
	)`)

	DB = db
	return func() { db.Close() }
}

func TestCreateTemplate(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "Testing"}
	products := []TemplateProductInput{
		{ProductID: 1, DefaultQuantity: 5, SortOrder: 0},
		{ProductID: 2, DefaultQuantity: 10, SortOrder: 1},
	}

	err := CreateTemplate(tmpl, products)
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}
	if tmpl.ID == 0 {
		t.Error("Expected template ID to be set")
	}
}

func TestGetTemplateByID(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "Testing"}
	CreateTemplate(tmpl, []TemplateProductInput{
		{ProductID: 1, DefaultQuantity: 5},
	})

	got, err := GetTemplateByID(tmpl.ID)
	if err != nil {
		t.Fatalf("GetTemplateByID failed: %v", err)
	}
	if got.Name != "Kit A" {
		t.Errorf("Expected name 'Kit A', got '%s'", got.Name)
	}
	if got.ProductCount != 1 {
		t.Errorf("Expected product count 1, got %d", got.ProductCount)
	}
}

func TestGetTemplatesByProjectID(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	CreateTemplate(&models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "A"}, []TemplateProductInput{{ProductID: 1, DefaultQuantity: 1}})
	CreateTemplate(&models.DCTemplate{ProjectID: 1, Name: "Kit B", Purpose: "B"}, []TemplateProductInput{{ProductID: 2, DefaultQuantity: 2}})
	CreateTemplate(&models.DCTemplate{ProjectID: 2, Name: "Other Kit", Purpose: "C"}, []TemplateProductInput{})

	templates, err := GetTemplatesByProjectID(1)
	if err != nil {
		t.Fatalf("GetTemplatesByProjectID failed: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("Expected 2 templates for project 1, got %d", len(templates))
	}
}

func TestGetTemplateProducts(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "Testing"}
	CreateTemplate(tmpl, []TemplateProductInput{
		{ProductID: 1, DefaultQuantity: 5, SortOrder: 1},
		{ProductID: 2, DefaultQuantity: 10, SortOrder: 0},
	})

	products, err := GetTemplateProducts(tmpl.ID)
	if err != nil {
		t.Fatalf("GetTemplateProducts failed: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("Expected 2 products, got %d", len(products))
	}
	// Should be ordered by sort_order (Product B first at sort_order=0)
	if products[0].ItemName != "Product B" {
		t.Errorf("Expected first product to be 'Product B' (sort_order=0), got '%s'", products[0].ItemName)
	}
}

func TestUpdateTemplate(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "Testing"}
	CreateTemplate(tmpl, []TemplateProductInput{{ProductID: 1, DefaultQuantity: 5}})

	tmpl.Name = "Kit A Updated"
	tmpl.Purpose = "Updated purpose"
	err := UpdateTemplate(tmpl, []TemplateProductInput{
		{ProductID: 2, DefaultQuantity: 20},
		{ProductID: 3, DefaultQuantity: 30},
	})
	if err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
	}

	got, _ := GetTemplateByID(tmpl.ID)
	if got.Name != "Kit A Updated" {
		t.Errorf("Expected updated name, got '%s'", got.Name)
	}
	if got.ProductCount != 2 {
		t.Errorf("Expected 2 products after update, got %d", got.ProductCount)
	}
}

func TestDeleteTemplate(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "Testing"}
	CreateTemplate(tmpl, []TemplateProductInput{{ProductID: 1, DefaultQuantity: 5}})

	err := DeleteTemplate(tmpl.ID, 1)
	if err != nil {
		t.Fatalf("DeleteTemplate failed: %v", err)
	}

	_, err = GetTemplateByID(tmpl.ID)
	if err == nil {
		t.Error("Expected error getting deleted template")
	}
}

func TestDeleteTemplate_WithDCs(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "Testing"}
	CreateTemplate(tmpl, []TemplateProductInput{{ProductID: 1, DefaultQuantity: 5}})

	// Create a DC linked to this template
	DB.Exec("INSERT INTO delivery_challans (project_id, template_id, dc_type) VALUES (1, ?, 'transit')", tmpl.ID)

	err := DeleteTemplate(tmpl.ID, 1)
	if err == nil {
		t.Error("Expected error when deleting template with DCs")
	}
}

func TestCheckTemplateNameUnique(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	CreateTemplate(&models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "A"}, []TemplateProductInput{{ProductID: 1, DefaultQuantity: 1}})

	unique, err := CheckTemplateNameUnique(1, "Kit A", 0)
	if err != nil {
		t.Fatalf("CheckTemplateNameUnique failed: %v", err)
	}
	if unique {
		t.Error("Expected name not to be unique")
	}

	unique, _ = CheckTemplateNameUnique(1, "Kit B", 0)
	if !unique {
		t.Error("Expected 'Kit B' to be unique")
	}

	// Different project should be unique
	unique, _ = CheckTemplateNameUnique(2, "Kit A", 0)
	if !unique {
		t.Error("Expected 'Kit A' in project 2 to be unique")
	}
}

func TestDuplicateTemplate(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "Original purpose"}
	CreateTemplate(tmpl, []TemplateProductInput{
		{ProductID: 1, DefaultQuantity: 5, SortOrder: 0},
		{ProductID: 2, DefaultQuantity: 10, SortOrder: 1},
	})

	dup, err := DuplicateTemplate(tmpl.ID, 1)
	if err != nil {
		t.Fatalf("DuplicateTemplate failed: %v", err)
	}

	if dup.Name != "Copy of Kit A" {
		t.Errorf("Expected name 'Copy of Kit A', got '%s'", dup.Name)
	}
	if dup.ID == tmpl.ID {
		t.Error("Duplicate should have a different ID")
	}

	// Check products were copied
	products, _ := GetTemplateProducts(dup.ID)
	if len(products) != 2 {
		t.Errorf("Expected 2 products in duplicate, got %d", len(products))
	}
}

func TestDuplicateTemplate_WrongProject(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "A"}
	CreateTemplate(tmpl, []TemplateProductInput{{ProductID: 1, DefaultQuantity: 1}})

	_, err := DuplicateTemplate(tmpl.ID, 2)
	if err == nil {
		t.Error("Expected error duplicating template from wrong project")
	}
}

func TestGetTemplateProductIDs(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "A"}
	CreateTemplate(tmpl, []TemplateProductInput{
		{ProductID: 1, DefaultQuantity: 5},
		{ProductID: 2, DefaultQuantity: 10},
	})

	ids, err := GetTemplateProductIDs(tmpl.ID)
	if err != nil {
		t.Fatalf("GetTemplateProductIDs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("Expected 2 product IDs, got %d", len(ids))
	}
	if ids[1] != 5 {
		t.Errorf("Expected quantity 5 for product 1, got %d", ids[1])
	}
	if ids[2] != 10 {
		t.Errorf("Expected quantity 10 for product 2, got %d", ids[2])
	}
}

func TestUsageCount(t *testing.T) {
	cleanup := setupTemplateTestDB(t)
	defer cleanup()

	tmpl := &models.DCTemplate{ProjectID: 1, Name: "Kit A", Purpose: "A"}
	CreateTemplate(tmpl, []TemplateProductInput{{ProductID: 1, DefaultQuantity: 1}})

	// No DCs initially
	got, _ := GetTemplateByID(tmpl.ID)
	if got.UsageCount != 0 {
		t.Errorf("Expected usage count 0, got %d", got.UsageCount)
	}

	// Add DCs
	DB.Exec("INSERT INTO delivery_challans (project_id, template_id, dc_type) VALUES (1, ?, 'transit')", tmpl.ID)
	DB.Exec("INSERT INTO delivery_challans (project_id, template_id, dc_type) VALUES (1, ?, 'official')", tmpl.ID)

	got, _ = GetTemplateByID(tmpl.ID)
	if got.UsageCount != 2 {
		t.Errorf("Expected usage count 2, got %d", got.UsageCount)
	}
	if got.TransitDCCount != 1 {
		t.Errorf("Expected transit count 1, got %d", got.TransitDCCount)
	}
	if got.OfficialDCCount != 1 {
		t.Errorf("Expected official count 1, got %d", got.OfficialDCCount)
	}
}
