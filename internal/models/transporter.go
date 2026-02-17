package models

import (
	"strings"
	"time"
)

type Transporter struct {
	ID            int                  `json:"id"`
	ProjectID     int                  `json:"project_id"`
	CompanyName   string               `json:"company_name"`
	ContactPerson string               `json:"contact_person"`
	Phone         string               `json:"phone"`
	GSTNumber     string               `json:"gst_number"`
	IsActive      bool                 `json:"is_active"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Vehicles      []*TransporterVehicle `json:"vehicles,omitempty"`
}

func (t *Transporter) Validate() map[string]string {
	errors := make(map[string]string)

	if strings.TrimSpace(t.CompanyName) == "" {
		errors["company_name"] = "Company name is required"
	}

	return errors
}

type TransporterVehicle struct {
	ID            int       `json:"id"`
	TransporterID int       `json:"transporter_id"`
	VehicleNumber string    `json:"vehicle_number"`
	VehicleType   string    `json:"vehicle_type"`
	DriverName    string    `json:"driver_name"`
	DriverPhone1  string    `json:"driver_phone1"`
	DriverPhone2  string    `json:"driver_phone2"`
	CreatedAt     time.Time `json:"created_at"`
}

func (v *TransporterVehicle) Validate() map[string]string {
	errors := make(map[string]string)

	if strings.TrimSpace(v.VehicleNumber) == "" {
		errors["vehicle_number"] = "Vehicle number is required"
	}

	if strings.TrimSpace(v.VehicleType) == "" {
		errors["vehicle_type"] = "Vehicle type is required"
	}

	if strings.TrimSpace(v.DriverName) == "" {
		errors["driver_name"] = "Driver name is required"
	}

	if strings.TrimSpace(v.DriverPhone1) == "" {
		errors["driver_phone1"] = "Driver phone number is required"
	}

	return errors
}

type TransporterPage struct {
	Transporters []*Transporter `json:"transporters"`
	CurrentPage  int            `json:"current_page"`
	PerPage      int            `json:"per_page"`
	TotalCount   int            `json:"total_count"`
	TotalPages   int            `json:"total_pages"`
	Search       string         `json:"search"`
}
