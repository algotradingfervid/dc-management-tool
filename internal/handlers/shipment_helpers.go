package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	pageshipments "github.com/narendhupati/dc-management-tool/components/pages/shipments"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// parseStep2Form parses the Step 1 form values that are carried forward as hidden
// fields into Step 2 and beyond.
func parseStep2Form(c echo.Context) (templateID, numLocations int, challanDate, transporterName, vehicleNumber, ewayBillNumber, docketNumber, taxType, reverseCharge string) {
	templateID, _ = strconv.Atoi(c.FormValue("template_id"))
	numLocations, _ = strconv.Atoi(c.FormValue("num_locations"))
	challanDate = c.FormValue("challan_date")
	transporterName = c.FormValue("transporter_name")
	vehicleNumber = c.FormValue("vehicle_number")
	ewayBillNumber = c.FormValue("eway_bill_number")
	docketNumber = c.FormValue("docket_number")
	taxType = c.FormValue("tax_type")
	reverseCharge = c.FormValue("reverse_charge")
	return
}

// parseStep3Form parses the Step 2 address selections (carried into Step 3 and beyond).
// It also calls ParseMultipartForm so that PostForm["ship_to_address_ids"] is populated.
func parseStep3Form(c echo.Context) (billFromAddrID, dispatchFromAddrID, billToAddrID, transitShipToAddrID int, shipToAddressIDs []int) {
	billFromAddrID, _ = strconv.Atoi(c.FormValue("bill_from_address_id"))
	dispatchFromAddrID, _ = strconv.Atoi(c.FormValue("dispatch_from_address_id"))
	billToAddrID, _ = strconv.Atoi(c.FormValue("bill_to_address_id"))
	transitShipToAddrID, _ = strconv.Atoi(c.FormValue("transit_ship_to_address_id"))
	if parseErr := c.Request().ParseMultipartForm(32 << 20); parseErr != nil {
		_ = c.Request().ParseForm()
	}
	for _, s := range c.Request().PostForm["ship_to_address_ids"] {
		id, idErr := strconv.Atoi(s)
		if idErr == nil && id > 0 {
			shipToAddressIDs = append(shipToAddressIDs, id)
		}
	}
	return
}

// parseQuantityForm parses qty_{productID}_{shipToAddrID} form fields into a
// nested map[productID]map[shipToAddrID]qty.
func parseQuantityForm(c echo.Context, products []*models.TemplateProductRow, shipToAddressIDs []int) map[int]map[int]int {
	quantities := make(map[int]map[int]int)
	for _, p := range products {
		quantities[p.ID] = make(map[int]int)
		for _, addrID := range shipToAddressIDs {
			key := fmt.Sprintf("qty_%d_%d", p.ID, addrID)
			qty, _ := strconv.Atoi(c.FormValue(key))
			quantities[p.ID][addrID] = qty
		}
	}
	return quantities
}

// validateQuantities validates the parsed quantity grid. Returns a map of
// error keys → messages. Keys are "qty_{productID}_{addrID}" for per-cell
// errors, "product_{productID}" for per-product errors, and "global" for
// form-wide errors. An empty map means validation passed.
func validateQuantities(quantities map[int]map[int]int, products []*models.TemplateProductRow, shipToAddressIDs []int) map[string]string {
	errors := make(map[string]string)

	grandTotal := 0
	for _, p := range products {
		productTotal := 0
		for _, addrID := range shipToAddressIDs {
			qty := quantities[p.ID][addrID]
			if qty < 0 {
				errors[fmt.Sprintf("qty_%d_%d", p.ID, addrID)] = "Quantity cannot be negative"
			}
			productTotal += qty
		}
		if productTotal == 0 {
			errors[fmt.Sprintf("product_%d", p.ID)] = fmt.Sprintf("%s has zero total quantity across all locations", p.ItemName)
		}
		grandTotal += productTotal
	}

	if grandTotal == 0 {
		errors["global"] = "At least one product must have a non-zero quantity"
	}

	return errors
}

// computeProductQuantityTotals sums per-location quantities into per-product totals.
func computeProductQuantityTotals(quantities map[int]map[int]int) map[int]int {
	totals := make(map[int]int)
	for productID, locQtys := range quantities {
		total := 0
		for _, qty := range locQtys {
			total += qty
		}
		totals[productID] = total
	}
	return totals
}

// parseStep4Form parses per-product serial numbers and per-address assignments
// from Step 3 form data.
func parseStep4Form(c echo.Context, products []*models.TemplateProductRow, shipToAddressIDs []int) []pageshipments.WizardSerialData {
	var result []pageshipments.WizardSerialData
	for _, p := range products {
		pd := pageshipments.WizardSerialData{
			ProductID:   p.ID,
			Assignments: make(map[int][]string),
		}
		serialsRaw := c.FormValue(fmt.Sprintf("serials_%d", p.ID))
		if serialsRaw != "" {
			for _, sn := range strings.Split(serialsRaw, "\n") {
				sn = strings.TrimSpace(sn)
				if sn != "" {
					pd.AllSerials = append(pd.AllSerials, sn)
				}
			}
		}
		for _, shipToID := range shipToAddressIDs {
			assignRaw := c.FormValue(fmt.Sprintf("assign_%d_%d", p.ID, shipToID))
			if assignRaw != "" {
				for _, sn := range strings.Split(assignRaw, "\n") {
					sn = strings.TrimSpace(sn)
					if sn != "" {
						pd.Assignments[shipToID] = append(pd.Assignments[shipToID], sn)
					}
				}
			}
		}
		result = append(result, pd)
	}
	return result
}
