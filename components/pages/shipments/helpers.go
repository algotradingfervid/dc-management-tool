package shipments

import (
	"encoding/json"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// vehiclesJSON serializes a slice of TransporterVehicle pointers to a JSON
// string suitable for embedding in a data-* HTML attribute.
func vehiclesJSON(vehicles []*models.TransporterVehicle) string {
	if len(vehicles) == 0 {
		return "[]"
	}
	type vehicleDTO struct {
		VehicleNumber string `json:"vehicle_number"`
		VehicleType   string `json:"vehicle_type"`
		DriverName    string `json:"driver_name"`
		DriverPhone1  string `json:"driver_phone1"`
	}
	dtos := make([]vehicleDTO, len(vehicles))
	for i, v := range vehicles {
		dtos[i] = vehicleDTO{
			VehicleNumber: v.VehicleNumber,
			VehicleType:   v.VehicleType,
			DriverName:    v.DriverName,
			DriverPhone1:  v.DriverPhone1,
		}
	}
	b, err := json.Marshal(dtos)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// shipToAddressesJSON serializes []*models.Address into a JS-safe JSON array
// of {id, name} objects for embedding inside a <script> tag.
func shipToAddressesJSON(addresses []*models.Address) string { //nolint:unused
	if len(addresses) == 0 {
		return "[]"
	}
	type addrDTO struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	dtos := make([]addrDTO, len(addresses))
	for i, a := range addresses {
		dtos[i] = addrDTO{ID: a.ID, Name: a.DisplayName()}
	}
	b, err := json.Marshal(dtos)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// joinStrings joins a slice of strings with a separator — mirrors the
// Go template "join" helper used in wizard_step4.
func joinStrings(items []string, sep string) string {
	return strings.Join(items, sep)
}

// WizardSerialData holds serial numbers for one product gathered in wizard step 3.
// This mirrors the local productSerialData struct in the shipment wizard handler.
type WizardSerialData struct {
	ProductID   int
	AllSerials  []string
	Assignments map[int][]string // shipToAddressID -> serials
}
