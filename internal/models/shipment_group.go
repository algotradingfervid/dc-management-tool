package models

import "time"

// ShipmentGroup represents a group of delivery challans shipped together.
type ShipmentGroup struct {
	ID            int       `json:"id"`
	ProjectID     int       `json:"project_id"`
	TemplateID    *int      `json:"template_id"`
	NumSets       int       `json:"num_sets"`
	TaxType       string    `json:"tax_type"`       // "cgst_sgst" or "igst"
	ReverseCharge string    `json:"reverse_charge"` // "Y" or "N"
	Status        string    `json:"status"`         // "draft", "issued"
	CreatedBy     int       `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Computed/joined fields
	TemplateName    string `json:"template_name"`
	TransitDCID     *int   `json:"transit_dc_id"`
	TransitDCNumber string `json:"transit_dc_number"`
	OfficialDCCount int    `json:"official_dc_count"`
	ProjectName     string `json:"project_name"`
}

func (sg *ShipmentGroup) Validate() map[string]string {
	errors := make(map[string]string)

	if sg.ProjectID == 0 {
		errors["project_id"] = "Project is required"
	}
	if sg.NumSets < 1 {
		errors["num_sets"] = "Number of sets must be at least 1"
	}
	if sg.TaxType != "cgst_sgst" && sg.TaxType != "igst" {
		errors["tax_type"] = "Tax type must be 'cgst_sgst' or 'igst'"
	}
	if sg.ReverseCharge != "Y" && sg.ReverseCharge != "N" {
		errors["reverse_charge"] = "Reverse charge must be 'Y' or 'N'"
	}

	return errors
}
