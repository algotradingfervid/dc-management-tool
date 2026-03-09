package transfer_dcs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// TransferStep1Prefill holds pre-populated values for the wizard's Step 1 form.
type TransferStep1Prefill struct {
	TemplateID      *int
	ChallanDate     string
	HubAddressID    int
	TransporterID   int
	TransporterName string
	VehicleNumber   string
	EwayBillNumber  string
	DocketNumber    string
	TaxType         string
	ReverseCharge   string
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

// TransferSerialData holds serial numbers for one product in the transfer wizard.
// Unlike the shipment wizard, there are NO per-destination assignments.
type TransferSerialData struct {
	ProductID  int
	AllSerials []string
}

// prefillStr returns fn(p) if p is non-nil, or "" otherwise.
func prefillStr(p *TransferStep1Prefill, fn func(*TransferStep1Prefill) string) string {
	if p == nil {
		return ""
	}
	return fn(p)
}

// hubAddressIDStr returns the hub address ID as a string for form prefill, or "" for 0.
func hubAddressIDStr(prefill *TransferStep1Prefill) string {
	if prefill == nil || prefill.HubAddressID == 0 {
		return ""
	}
	return strconv.Itoa(prefill.HubAddressID)
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

// hubAddressesJSON serializes []*models.Address into a JS-safe JSON array
// of {id, name} objects for embedding in a data-* attribute.
func hubAddressesJSON(addresses []*models.Address) string {
	return shipToAddressesJSON(addresses)
}

// hubPreselectedJSON returns the prefilled hub address ID as a string
// for embedding in a data-* attribute, or "" if no prefill.
func hubPreselectedJSON(prefill *TransferStep1Prefill) string {
	if prefill == nil || prefill.HubAddressID == 0 {
		return ""
	}
	return strconv.Itoa(prefill.HubAddressID)
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
// Go template "join" helper used in wizard steps.
func joinStrings(items []string, sep string) string {
	return strings.Join(items, sep)
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

// SplitProductInfo holds minimal product information for split wizard display.
type SplitProductInfo struct {
	ID   int
	Name string
}

// splitSerialTextareaClass returns the appropriate Tailwind border class for a
// serial textarea in the split wizard depending on whether that product has a server-side error.
func splitSerialTextareaClass(serialErrors map[int]string, productID int) string {
	if serialErrors != nil && serialErrors[productID] != "" {
		return "mt-1 block w-full rounded-md border-red-500 shadow-sm font-mono text-sm"
	}
	return "mt-1 block w-full rounded-md border-gray-300 shadow-sm font-mono text-sm"
}

// transferWizardURL returns the correct URL for a wizard step in create or edit mode.
func transferWizardURL(projectID, tdcID int, path string) string {
	if tdcID > 0 {
		return fmt.Sprintf("/projects/%d/transfer-dcs/%d/edit%s", projectID, tdcID, path)
	}
	return fmt.Sprintf("/projects/%d/transfer-dcs/new%s", projectID, path)
}

// transferWizardSubmitURL returns the final submit URL for create or edit mode.
func transferWizardSubmitURL(projectID, tdcID int) string {
	if tdcID > 0 {
		return fmt.Sprintf("/projects/%d/transfer-dcs/%d/edit", projectID, tdcID)
	}
	return fmt.Sprintf("/projects/%d/transfer-dcs", projectID)
}

// Ensure fmt is used (referenced in other helpers via Sprintf patterns).
var _ = fmt.Sprintf
