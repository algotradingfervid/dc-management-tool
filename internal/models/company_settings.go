package models

// CompanySettings holds the company information for DC print views.
type CompanySettings struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	City           string `json:"city"`
	State          string `json:"state"`
	StateCode      string `json:"state_code"`
	Pincode        string `json:"pincode"`
	GSTIN          string `json:"gstin"`
	SignatureImage string `json:"signature_image"`
	Email          string `json:"email"`
	CIN            string `json:"cin"`
}
