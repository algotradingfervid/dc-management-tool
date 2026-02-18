package models

import "time"

type Transporter struct {
	ID            int                   `json:"id"`
	ProjectID     int                   `json:"project_id"`
	CompanyName   string                `json:"company_name" validate:"required,max=255"`
	ContactPerson string                `json:"contact_person" validate:"max=255"`
	Phone         string                `json:"phone" validate:"max=15"`
	GSTNumber     string                `json:"gst_number" validate:"omitempty,len=15"`
	IsActive      bool                  `json:"is_active"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	Vehicles      []*TransporterVehicle `json:"vehicles,omitempty"`
}

type TransporterVehicle struct {
	ID            int       `json:"id"`
	TransporterID int       `json:"transporter_id"`
	VehicleNumber string    `json:"vehicle_number" validate:"required,max=50"`
	VehicleType   string    `json:"vehicle_type" validate:"required,max=50"`
	DriverName    string    `json:"driver_name" validate:"required,max=255"`
	DriverPhone1  string    `json:"driver_phone1" validate:"required,max=15"`
	DriverPhone2  string    `json:"driver_phone2" validate:"omitempty,max=15"`
	CreatedAt     time.Time `json:"created_at"`
}

type TransporterPage struct {
	Transporters []*Transporter `json:"transporters"`
	CurrentPage  int            `json:"current_page"`
	PerPage      int            `json:"per_page"`
	TotalCount   int            `json:"total_count"`
	TotalPages   int            `json:"total_pages"`
	Search       string         `json:"search"`
}
