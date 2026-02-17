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
	DispatchFromAddress  string    `json:"dispatch_from_address"`
	CompanyGSTIN         string    `json:"company_gstin"`
	CompanyEmail         string    `json:"company_email"`
	CompanyCIN           string    `json:"company_cin"`
	CompanySignaturePath string    `json:"company_signature_path"`
	CompanySealPath      string    `json:"company_seal_path"`
	DCNumberFormat       string    `json:"dc_number_format"`
	DCNumberSeparator    string    `json:"dc_number_separator"`
	PurposeText          string    `json:"purpose_text"`
	SeqPadding           int       `json:"seq_padding"`
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

// DefaultDCNumberFormat is the default DC number format pattern.
const DefaultDCNumberFormat = "{PREFIX}-{TYPE}-{FY}-{SEQ}"

// DCFormatTokens lists available tokens for DC number formatting.
var DCFormatTokens = []string{"{PREFIX}", "{PROJECT_CODE}", "{FY}", "{SEQ}", "{TYPE}"}

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

	if p.CompanyEmail != "" && !strings.Contains(p.CompanyEmail, "@") {
		errors["company_email"] = "Invalid email address"
	}

	if p.SeqPadding != 0 && (p.SeqPadding < 2 || p.SeqPadding > 6) {
		errors["seq_padding"] = "Sequence padding must be between 2 and 6"
	}

	return errors
}
