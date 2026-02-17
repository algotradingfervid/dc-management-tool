package database

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	db.Exec("PRAGMA foreign_keys = ON")

	// Create tables
	db.Exec(`CREATE TABLE projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		created_by INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`INSERT INTO projects (id, name) VALUES (1, 'Test Project')`)

	db.Exec(`CREATE TABLE transporters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		company_name TEXT NOT NULL,
		contact_person TEXT DEFAULT '',
		phone TEXT DEFAULT '',
		gst_number TEXT DEFAULT '',
		is_active BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	)`)

	_, err2 := db.Exec(`CREATE TABLE transporter_vehicles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		transporter_id INTEGER NOT NULL,
		vehicle_number TEXT NOT NULL,
		vehicle_type TEXT DEFAULT 'truck',
		driver_name TEXT DEFAULT '',
		driver_phone1 TEXT DEFAULT '',
		driver_phone2 TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (transporter_id) REFERENCES transporters(id) ON DELETE CASCADE
	)`)
	if err2 != nil {
		t.Fatalf("Failed to create transporter_vehicles: %v", err2)
	}

	DB = db
	return func() { db.Close() }
}

func TestCreateTransporter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{
		ProjectID:     1,
		CompanyName:   "ABC Transport",
		ContactPerson: "John",
		Phone:         "9876543210",
		GSTNumber:     "27AABCU9603R1ZM",
		IsActive:      true,
	}

	if err := CreateTransporter(tr); err != nil {
		t.Fatalf("CreateTransporter failed: %v", err)
	}

	if tr.ID == 0 {
		t.Error("expected ID to be set after creation")
	}
}

func TestGetTransporterByID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{ProjectID: 1, CompanyName: "Test Co", IsActive: true}
	CreateTransporter(tr)

	got, err := GetTransporterByID(tr.ID)
	if err != nil {
		t.Fatalf("GetTransporterByID failed: %v", err)
	}
	if got.CompanyName != "Test Co" {
		t.Errorf("expected company name 'Test Co', got %q", got.CompanyName)
	}
}

func TestUpdateTransporter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{ProjectID: 1, CompanyName: "Old Name", IsActive: true}
	CreateTransporter(tr)

	tr.CompanyName = "New Name"
	tr.ContactPerson = "Jane"
	if err := UpdateTransporter(tr); err != nil {
		t.Fatalf("UpdateTransporter failed: %v", err)
	}

	got, _ := GetTransporterByID(tr.ID)
	if got.CompanyName != "New Name" {
		t.Errorf("expected 'New Name', got %q", got.CompanyName)
	}
	if got.ContactPerson != "Jane" {
		t.Errorf("expected 'Jane', got %q", got.ContactPerson)
	}
}

func TestDeactivateTransporter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{ProjectID: 1, CompanyName: "Active Co", IsActive: true}
	CreateTransporter(tr)

	if err := DeactivateTransporter(tr.ID, 1); err != nil {
		t.Fatalf("DeactivateTransporter failed: %v", err)
	}

	got, _ := GetTransporterByID(tr.ID)
	if got.IsActive {
		t.Error("expected transporter to be deactivated")
	}
}

func TestActivateTransporter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{ProjectID: 1, CompanyName: "Inactive Co", IsActive: false}
	CreateTransporter(tr)
	DeactivateTransporter(tr.ID, 1)

	if err := ActivateTransporter(tr.ID, 1); err != nil {
		t.Fatalf("ActivateTransporter failed: %v", err)
	}

	got, _ := GetTransporterByID(tr.ID)
	if !got.IsActive {
		t.Error("expected transporter to be activated")
	}
}

func TestGetTransportersByProjectID_ActiveOnly(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	CreateTransporter(&models.Transporter{ProjectID: 1, CompanyName: "Active1", IsActive: true})
	CreateTransporter(&models.Transporter{ProjectID: 1, CompanyName: "Active2", IsActive: true})
	tr3 := &models.Transporter{ProjectID: 1, CompanyName: "Inactive1", IsActive: true}
	CreateTransporter(tr3)
	DeactivateTransporter(tr3.ID, 1)

	// Active only
	active, err := GetTransportersByProjectID(1, true)
	if err != nil {
		t.Fatalf("GetTransportersByProjectID failed: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active transporters, got %d", len(active))
	}

	// All
	all, err := GetTransportersByProjectID(1, false)
	if err != nil {
		t.Fatalf("GetTransportersByProjectID failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 total transporters, got %d", len(all))
	}
}

func TestSearchTransporters(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	CreateTransporter(&models.Transporter{ProjectID: 1, CompanyName: "Alpha Transport", IsActive: true})
	CreateTransporter(&models.Transporter{ProjectID: 1, CompanyName: "Beta Logistics", IsActive: true})
	CreateTransporter(&models.Transporter{ProjectID: 1, CompanyName: "Gamma Transport", IsActive: true})

	// Search by name
	page, err := SearchTransporters(1, "Transport", 1, 20)
	if err != nil {
		t.Fatalf("SearchTransporters failed: %v", err)
	}
	if page.TotalCount != 2 {
		t.Errorf("expected 2 results for 'Transport', got %d", page.TotalCount)
	}

	// Search all
	page, err = SearchTransporters(1, "", 1, 20)
	if err != nil {
		t.Fatalf("SearchTransporters failed: %v", err)
	}
	if page.TotalCount != 3 {
		t.Errorf("expected 3 total, got %d", page.TotalCount)
	}
}

func TestSearchTransporters_Pagination(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		CreateTransporter(&models.Transporter{ProjectID: 1, CompanyName: "Company " + string(rune('A'+i)), IsActive: true})
	}

	page, err := SearchTransporters(1, "", 1, 2)
	if err != nil {
		t.Fatalf("SearchTransporters failed: %v", err)
	}
	if page.TotalCount != 5 {
		t.Errorf("expected total 5, got %d", page.TotalCount)
	}
	if page.TotalPages != 3 {
		t.Errorf("expected 3 pages, got %d", page.TotalPages)
	}
	if len(page.Transporters) != 2 {
		t.Errorf("expected 2 on page 1, got %d", len(page.Transporters))
	}
}

func TestGetTransporterCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	CreateTransporter(&models.Transporter{ProjectID: 1, CompanyName: "A", IsActive: true})
	CreateTransporter(&models.Transporter{ProjectID: 1, CompanyName: "B", IsActive: true})
	tr := &models.Transporter{ProjectID: 1, CompanyName: "C", IsActive: true}
	CreateTransporter(tr)
	DeactivateTransporter(tr.ID, 1)

	count, err := GetTransporterCount(1)
	if err != nil {
		t.Fatalf("GetTransporterCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2 (active only), got %d", count)
	}
}

func TestCreateVehicle(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{ProjectID: 1, CompanyName: "Test Co", IsActive: true}
	CreateTransporter(tr)

	v := &models.TransporterVehicle{
		TransporterID: tr.ID,
		VehicleNumber: "MH12AB1234",
		VehicleType:   "truck",
	}

	if err := CreateVehicle(v); err != nil {
		t.Fatalf("CreateVehicle failed: %v", err)
	}
	if v.ID == 0 {
		t.Error("expected vehicle ID to be set")
	}
}

func TestGetVehiclesByTransporterID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{ProjectID: 1, CompanyName: "Test Co", IsActive: true}
	CreateTransporter(tr)

	CreateVehicle(&models.TransporterVehicle{TransporterID: tr.ID, VehicleNumber: "MH12AB1234", VehicleType: "truck"})
	CreateVehicle(&models.TransporterVehicle{TransporterID: tr.ID, VehicleNumber: "MH14CD5678", VehicleType: "mini-truck"})

	vehicles, err := GetVehiclesByTransporterID(tr.ID)
	if err != nil {
		t.Fatalf("GetVehiclesByTransporterID failed: %v", err)
	}
	if len(vehicles) != 2 {
		t.Errorf("expected 2 vehicles, got %d", len(vehicles))
	}
}

func TestDeleteVehicle(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{ProjectID: 1, CompanyName: "Test Co", IsActive: true}
	CreateTransporter(tr)

	v := &models.TransporterVehicle{TransporterID: tr.ID, VehicleNumber: "MH12AB1234", VehicleType: "truck"}
	CreateVehicle(v)

	if err := DeleteVehicle(v.ID, tr.ID); err != nil {
		t.Fatalf("DeleteVehicle failed: %v", err)
	}

	vehicles, _ := GetVehiclesByTransporterID(tr.ID)
	if len(vehicles) != 0 {
		t.Errorf("expected 0 vehicles after delete, got %d", len(vehicles))
	}
}

func TestGetTransporterByID_WithVehicles(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tr := &models.Transporter{ProjectID: 1, CompanyName: "Test Co", IsActive: true}
	CreateTransporter(tr)

	CreateVehicle(&models.TransporterVehicle{TransporterID: tr.ID, VehicleNumber: "MH12AB1234", VehicleType: "truck"})
	CreateVehicle(&models.TransporterVehicle{TransporterID: tr.ID, VehicleNumber: "MH14CD5678", VehicleType: "mini-truck"})

	got, err := GetTransporterByID(tr.ID)
	if err != nil {
		t.Fatalf("GetTransporterByID failed: %v", err)
	}
	if len(got.Vehicles) != 2 {
		t.Errorf("expected 2 vehicles loaded, got %d", len(got.Vehicles))
	}
}
