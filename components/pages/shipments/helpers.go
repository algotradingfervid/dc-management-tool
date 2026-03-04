package shipments

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// ShipStep1Prefill holds pre-populated values for the edit wizard's Step 1 form.
// All fields map directly to form inputs. Zero values mean "no pre-fill".
type ShipStep1Prefill struct {
	TemplateID      *int   // nil = no selection
	NumLocations    int
	ChallanDate     string // "YYYY-MM-DD" or ""
	TransporterID   int    // 0 = no pre-selection
	TransporterName string
	VehicleNumber   string
	EwayBillNumber  string
	DocketNumber    string
	TaxType         string // "igst" | "cgst_sgst" | ""
	ReverseCharge   string // "Y" | "N" | ""
}

// prefillStr returns fn(p) if p is non-nil, or "" otherwise.
func prefillStr(p *ShipStep1Prefill, fn func(*ShipStep1Prefill) string) string {
	if p == nil {
		return ""
	}
	return fn(p)
}

// prefillNumLocations returns p.NumLocations as a string, or "" when p is nil or NumLocations is 0.
func prefillNumLocations(p *ShipStep1Prefill) string {
	if p == nil || p.NumLocations == 0 {
		return ""
	}
	return strconv.Itoa(p.NumLocations)
}

// preselectedIDsJSON serializes a slice of address IDs to a JSON array string
// for embedding in a data-* attribute. Returns "[]" for nil input.
func preselectedIDsJSON(ids []int) string {
	if ids == nil {
		return "[]"
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

// prefillAssignmentsForProduct returns newline-joined serials assigned to
// a specific (product, shipTo) pair, or "" if the map is nil/missing.
func prefillAssignmentsForProduct(m map[string][]string, productID, shipToID int) string {
	if m == nil {
		return ""
	}
	key := fmt.Sprintf("%d_%d", productID, shipToID)
	return strings.Join(m[key], "\n")
}

// serialTextareaClass returns the appropriate Tailwind border class for a
// serial textarea depending on whether that product has a server-side error.
func serialTextareaClass(serialErrors map[int]string, productID int) string {
	if serialErrors[productID] != "" {
		return "mt-1 block w-full rounded-md border-red-500 shadow-sm font-mono text-sm"
	}
	return "mt-1 block w-full rounded-md border-gray-300 shadow-sm font-mono text-sm"
}

// prefillSerialsForProduct returns the newline-joined serial numbers for productID
// from the prefill map, or "" if the map is nil or has no entry for that product.
func prefillSerialsForProduct(m map[int][]string, productID int) string {
	if m == nil {
		return ""
	}
	return strings.Join(m[productID], "\n")
}

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

// QuantityAddress pairs an address ID with a display name for the quantity grid columns.
type QuantityAddress struct {
	ID   int
	Name string
}

// QuantityHiddenField holds a name/value pair for carrying quantity data through form steps.
type QuantityHiddenField struct {
	Name  string
	Value string
}

// qtyInputValue returns the value string for a quantity input field.
// If prefillQuantities has a value for this product+address, use it.
// Otherwise fall back to the product's default quantity.
func qtyInputValue(prefillQuantities map[int]map[int]int, productID, addrID, defaultQty int) string {
	if prefillQuantities != nil {
		if qMap, ok := prefillQuantities[productID]; ok {
			if qty, ok := qMap[addrID]; ok {
				return strconv.Itoa(qty)
			}
		}
	}
	return strconv.Itoa(defaultQty)
}

// productExpectedTotal returns the total expected serial count for a product
// by summing its quantities across all locations. Falls back to defaultQty * numLocations
// if the map is nil or missing.
func productExpectedTotal(quantities map[int]int, productID, defaultQty, numLocations int) int {
	if quantities != nil {
		if total, ok := quantities[productID]; ok {
			return total
		}
	}
	return defaultQty * numLocations
}

// WizardSerialData holds serial numbers for one product gathered in wizard step 3.
// This mirrors the local productSerialData struct in the shipment wizard handler.
type WizardSerialData struct {
	ProductID   int
	AllSerials  []string
	Assignments map[int][]string // shipToAddressID -> serials
}
