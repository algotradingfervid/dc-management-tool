package models

import "time"

// ShipmentGroup represents a group of delivery challans shipped together.
type ShipmentGroup struct {
	ID            int       `json:"id"`
	ProjectID     int       `json:"project_id" validate:"required,gt=0"`
	TemplateID    *int      `json:"template_id"`
	NumSets       int       `json:"num_sets" validate:"required,gte=1"`
	TaxType       string    `json:"tax_type" validate:"required,oneof=cgst_sgst igst"` // "cgst_sgst" or "igst"
	ReverseCharge string    `json:"reverse_charge" validate:"required,oneof=Y N"`      // "Y" or "N"
	Status        string    `json:"status"`                                            // "draft", "issued"
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
