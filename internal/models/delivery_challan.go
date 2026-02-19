package models

import (
	"time"
)

// DeliveryChallan represents a delivery challan (transit or official).
type DeliveryChallan struct {
	ID              int        `json:"id"`
	ProjectID       int        `json:"project_id" validate:"required,gt=0"`
	DCNumber        string     `json:"dc_number"`
	DCType          string     `json:"dc_type" validate:"required,oneof=transit official"` // "transit" or "official"
	Status          string     `json:"status"`                                             // "draft" or "issued"
	TemplateID      *int       `json:"template_id"`
	BillToAddressID *int       `json:"bill_to_address_id"`
	ShipToAddressID int        `json:"ship_to_address_id" validate:"required,gt=0"`
	ChallanDate     *string    `json:"challan_date" validate:"required"`
	IssuedAt        *time.Time `json:"issued_at"`
	IssuedBy        *int       `json:"issued_by"`
	CreatedBy       int        `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Optional bundle reference
	BundleID *int `json:"bundle_id"`

	// Shipment group fields
	ShipmentGroupID       *int `json:"shipment_group_id"`
	BillFromAddressID     *int `json:"bill_from_address_id"`
	DispatchFromAddressID *int `json:"dispatch_from_address_id"`

	// Computed/joined fields
	ProjectName   string `json:"project_name"`
	TemplateName  string `json:"template_name"`
	LineItemCount int    `json:"line_item_count"`
	TotalQuantity int    `json:"total_quantity"`
}

// DCTransitDetails stores transit-specific details for a delivery challan.
type DCTransitDetails struct {
	ID              int    `json:"id"`
	DCID            int    `json:"dc_id"`
	TransporterName string `json:"transporter_name"`
	VehicleNumber   string `json:"vehicle_number"`
	EwayBillNumber  string `json:"eway_bill_number"`
	Notes           string `json:"notes"`
}

// DCLineItem represents a product line in a delivery challan.
type DCLineItem struct {
	ID            int       `json:"id"`
	DCID          int       `json:"dc_id"`
	ProductID     int       `json:"product_id"`
	Quantity      int       `json:"quantity"`
	Rate          float64   `json:"rate"`
	TaxPercentage float64   `json:"tax_percentage"`
	TaxableAmount float64   `json:"taxable_amount"`
	TaxAmount     float64   `json:"tax_amount"`
	TotalAmount   float64   `json:"total_amount"`
	LineOrder     int       `json:"line_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Joined/computed fields (not stored in dc_line_items)
	ItemName        string   `json:"item_name"`
	ItemDescription string   `json:"item_description"`
	HSNCode         string   `json:"hsn_code"`
	UoM             string   `json:"uom"`
	BrandModel      string   `json:"brand_model"`
	GSTPercentage   float64  `json:"gst_percentage"`
	SerialNumbers   []string `json:"serial_numbers"`
}

// SerialNumber represents a serial number tracked per line item.
type SerialNumber struct {
	ID           int       `json:"id"`
	ProjectID    int       `json:"project_id"`
	LineItemID   int       `json:"line_item_id"`
	SerialNumber string    `json:"serial_number"`
	CreatedAt    time.Time `json:"created_at"`
}

// TransitDCFormData holds all data needed to render the Transit DC creation form.
type TransitDCFormData struct {
	Project         *Project
	Template        *DCTemplate
	Products        []*TemplateProductRow
	ShipToAddresses []*Address
	BillToAddresses []*Address
	DCNumber        string
	ChallanDate     string
	Purpose         string
}

// TransitDCSubmission holds form submission data for creating a Transit DC.
type TransitDCSubmission struct {
	// DC fields
	TemplateID      int    `json:"template_id"`
	ChallanDate     string `json:"challan_date"`
	ShipToAddressID int    `json:"ship_to_address_id"`
	BillToAddressID int    `json:"bill_to_address_id"`

	// Transit details
	TransporterName string `json:"transporter_name"`
	VehicleNumber   string `json:"vehicle_number"`
	EwayBillNumber  string `json:"eway_bill_number"`
	Notes           string `json:"notes"`

	// Tax info
	TaxType       string `json:"tax_type"`       // "cgst_sgst" or "igst"
	ReverseCharge string `json:"reverse_charge"` // "Y" or "N"

	// Line items
	LineItems []TransitDCLineItemInput `json:"line_items"`
}

// TransitDCLineItemInput holds input for a single line item.
type TransitDCLineItemInput struct {
	ProductID     int      `json:"product_id"`
	Rate          float64  `json:"rate"`
	TaxPercentage float64  `json:"tax_percentage"`
	SerialNumbers []string `json:"serial_numbers"`
	Remarks       string   `json:"remarks"`
}
