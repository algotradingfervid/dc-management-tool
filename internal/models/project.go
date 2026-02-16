package models

import (
	"strings"
	"time"
)

type Project struct {
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	DCPrefix             string    `json:"dc_prefix"`
	TenderRefNumber      string    `json:"tender_ref_number"`
	TenderRefDetails     string    `json:"tender_ref_details"`
	POReference          string    `json:"po_reference"`
	PODate               *string   `json:"po_date"`
	BillFromAddress      string    `json:"bill_from_address"`
	CompanyGSTIN         string    `json:"company_gstin"`
	CompanySignaturePath string    `json:"company_signature_path"`
	LastTransitDCNumber  int       `json:"last_transit_dc_number"`
	LastOfficialDCNumber int       `json:"last_official_dc_number"`
	CreatedBy            int       `json:"created_by"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`

	// Computed fields
	TransitDCCount  int `json:"transit_dc_count"`
	OfficialDCCount int `json:"official_dc_count"`
	TemplateCount   int `json:"template_count"`
	ProductCount    int `json:"product_count"`
}

func (p *Project) Validate() map[string]string {
	errors := make(map[string]string)

	if strings.TrimSpace(p.Name) == "" {
		errors["name"] = "Project name is required"
	}

	if strings.TrimSpace(p.DCPrefix) == "" {
		errors["dc_prefix"] = "DC prefix is required"
	} else if len(p.DCPrefix) > 10 {
		errors["dc_prefix"] = "DC prefix must be 10 characters or less"
	}

	if p.CompanyGSTIN != "" && len(p.CompanyGSTIN) != 15 {
		errors["company_gstin"] = "GSTIN must be exactly 15 characters"
	}

	return errors
}
