package models

import (
	"testing"

	"github.com/narendhupati/dc-management-tool/internal/helpers"
)

func TestTransporterValidate(t *testing.T) {
	tests := []struct {
		name      string
		transport Transporter
		wantErr   bool
		errField  string
	}{
		{
			name:      "valid transporter",
			transport: Transporter{CompanyName: "ABC Transport"},
			wantErr:   false,
		},
		{
			name:      "missing company name",
			transport: Transporter{CompanyName: ""},
			wantErr:   true,
			errField:  "company_name",
		},
		{
			name: "valid with all fields",
			transport: Transporter{
				CompanyName:   "ABC Transport",
				ContactPerson: "John",
				Phone:         "9876543210",
				GSTNumber:     "27AABCU9603R1ZM",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := helpers.ValidateStruct(&tt.transport)
			if tt.wantErr {
				if len(errors) == 0 {
					t.Error("expected validation errors, got none")
				}
				if tt.errField != "" {
					if _, ok := errors[tt.errField]; !ok {
						t.Errorf("expected error for field %q, got errors: %v", tt.errField, errors)
					}
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("expected no errors, got: %v", errors)
				}
			}
		})
	}
}

func TestTransporterVehicleValidate(t *testing.T) {
	tests := []struct {
		name     string
		vehicle  TransporterVehicle
		wantErr  bool
		errField string
	}{
		{
			name:    "valid vehicle",
			vehicle: TransporterVehicle{VehicleNumber: "MH12AB1234", VehicleType: "truck", DriverName: "Raju", DriverPhone1: "9876543210"},
			wantErr: false,
		},
		{
			name:     "missing vehicle number",
			vehicle:  TransporterVehicle{VehicleNumber: "", VehicleType: "truck", DriverName: "Raju", DriverPhone1: "9876543210"},
			wantErr:  true,
			errField: "vehicle_number",
		},
		{
			name:     "missing vehicle type",
			vehicle:  TransporterVehicle{VehicleNumber: "MH12AB1234", VehicleType: "", DriverName: "Raju", DriverPhone1: "9876543210"},
			wantErr:  true,
			errField: "vehicle_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := helpers.ValidateStruct(&tt.vehicle)
			if tt.wantErr {
				if len(errors) == 0 {
					t.Error("expected validation errors, got none")
				}
				if tt.errField != "" {
					if _, ok := errors[tt.errField]; !ok {
						t.Errorf("expected error for field %q, got errors: %v", tt.errField, errors)
					}
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("expected no errors, got: %v", errors)
				}
			}
		})
	}
}
